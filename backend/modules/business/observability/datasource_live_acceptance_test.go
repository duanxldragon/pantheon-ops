package observability

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
)

func successfulProviderEnvelope(payload []byte) bool {
	var envelope struct {
		Code int `json:"code"`
	}
	return json.Unmarshal(payload, &envelope) == nil && envelope.Code == common.CodeSuccess
}

func TestDatasourceLiveEnvelopeMatchesNativeSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	common.Success(c, map[string]string{"status": "success"})
	if recorder.Code != http.StatusOK || !successfulProviderEnvelope(recorder.Body.Bytes()) {
		t.Fatal("acceptance oracle rejects the native successful envelope")
	}
	for _, payload := range []string{`{}`, `{"code":0}`, `{"code":403}`, `{"code":500}`, `not-json`} {
		if successfulProviderEnvelope([]byte(payload)) {
			t.Fatal("acceptance oracle accepts a non-successful envelope")
		}
	}
}

// Root supplies public-API-created sources and real narrow-grant/denied sessions
// for a separately built, pinned product. No identity, scope or policy is mocked.
func TestDatasourceLiveAcceptance(t *testing.T) {
	if os.Getenv("PANTHEON_ACCEPTANCE_HOST") == "" {
		t.Skip("real composed-product/provider acceptance not requested")
	}
	if os.Getenv("PANTHEON_ACCEPTANCE_HOST") != "115.190.251.143" {
		t.Fatal("provider host outside recorded acceptance scope")
	}
	endpoint, err := url.Parse(os.Getenv("PANTHEON_O4_ACCEPTANCE_API_URL"))
	if err != nil || endpoint.User != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") ||
		(endpoint.Hostname() != "127.0.0.1" && endpoint.Hostname() != "115.190.251.143") ||
		strings.TrimRight(endpoint.Path, "/") != observabilityAPIPrefix || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		t.Fatal("exact public composed-product API endpoint required")
	}
	token, deniedToken := os.Getenv("PANTHEON_O4_ACCEPTANCE_TOKEN"), os.Getenv("PANTHEON_O4_ACCEPTANCE_DENIED_TOKEN")
	if token == "" || deniedToken == "" || token == deniedToken {
		t.Fatal("separate real narrow-grant and authenticated denied sessions required")
	}
	var ids map[string]uint64
	if json.Unmarshal([]byte(os.Getenv("PANTHEON_O4_ACCEPTANCE_SOURCE_IDS")), &ids) != nil || len(ids) != 4 {
		t.Fatal("four public-API-created source IDs required")
	}
	client := &http.Client{Timeout: 35 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	request := func(t *testing.T, method, path, session string, body any) (int, []byte) {
		t.Helper()
		var encoded []byte
		if body != nil {
			var encodeErr error
			encoded, encodeErr = json.Marshal(body)
			if encodeErr != nil {
				t.Fatal("fixture encoding failed")
			}
		}
		req, err := http.NewRequest(method, strings.TrimRight(endpoint.String(), "/")+path, bytes.NewReader(encoded))
		if err != nil {
			t.Fatal("fixture request invalid")
		}
		req.Header.Set("Authorization", "Bearer "+session)
		req.Header.Set("Content-Type", "application/json")
		response, err := client.Do(req)
		if err != nil {
			t.Fatal("composed product unavailable")
		}
		defer response.Body.Close()
		payload, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
		if err != nil || len(payload) > 8<<20 {
			t.Fatal("response outside accepted budget")
		}
		return response.StatusCode, payload
	}
	for _, fixture := range []struct{ kind, route, query, expected string }{
		{"prometheus", "query", "vector(1)", `"1"`},
		{"victoriametrics", "query", "pantheon_p0_acceptance", `"1"`},
		{"loki", "logs", `{app="pantheon-p0-acceptance"}`, "pantheon-p0-synthetic-log"},
		{"tempo", "traces", "", "f0f0110920264a55a0a0110920260001"},
	} {
		t.Run(fixture.kind, func(t *testing.T) {
			id := ids[fixture.kind]
			if id == 0 {
				t.Fatal("source kind ID missing")
			}
			path := "/datasources/" + strconv.FormatUint(id, 10) + "/" + fixture.route
			body := map[string]any{"rawQuery": fixture.query, "time": QueryTime{Mode: "relative", Duration: "15m"}, "limit": 100, "resultMode": "table"}
			if fixture.kind == "tempo" {
				body = map[string]any{"recipe": QueryRecipe{Version: QueryCatalogVersion, SourceKind: "tempo", Rules: []QueryRule{{Field: "service", Op: "exists"}}, Time: QueryTime{Mode: "relative", Duration: "15m"}, Limit: 100, ResultMode: "table"}}
			}
			t.Run("public_POST_provider_success", func(t *testing.T) {
				status, payload := request(t, http.MethodPost, path, token, body)
				if status != http.StatusOK || !bytes.Contains(payload, []byte(fixture.expected)) {
					t.Fatal("real provider synthetic result missing")
				}
				if !successfulProviderEnvelope(payload) {
					t.Fatal("provider result not successful")
				}
			})
			t.Run("current_authenticated_denial", func(t *testing.T) {
				status, payload := request(t, http.MethodPost, path, deniedToken, body)
				if status != http.StatusForbidden || bytes.Contains(payload, []byte(fixture.expected)) {
					t.Fatal("current narrow source/action denial failed")
				}
			})
			if fixture.kind != "tempo" {
				t.Run("provider_failure", func(t *testing.T) {
					invalid := map[string]any{"rawQuery": "invalid(((", "time": QueryTime{Mode: "relative", Duration: "15m"}, "limit": 100, "resultMode": "table"}
					status, _ := request(t, http.MethodPost, path, token, invalid)
					if status != http.StatusBadGateway {
						t.Fatal("provider failure did not use safe upstream outcome")
					}
				})
			} else {
				t.Run("unknown_intent_denied", func(t *testing.T) {
					status, _ := request(t, http.MethodPost, path, token, map[string]any{"recipe": map[string]any{"secret": "synthetic-intent-sentinel"}})
					if status != http.StatusBadRequest {
						t.Fatal("unreviewed trace intent accepted")
					}
				})
				t.Run("public_trace_detail", func(t *testing.T) {
					status, payload := request(t, http.MethodGet, "/datasources/"+strconv.FormatUint(id, 10)+"/traces/"+fixture.expected, token, nil)
					var envelope struct {
						Data map[string]any `json:"data"`
					}
					if status != http.StatusOK || json.Unmarshal(payload, &envelope) != nil || len(envelope.Data) == 0 {
						t.Fatal("real trace detail missing")
					}
				})
			}
		})
	}
}
