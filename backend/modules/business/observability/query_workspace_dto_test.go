package observability

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRunEnvelopeStrictAndRawMemoryOnly(t *testing.T) {
	raw := "SECRET_RAW_EDITOR_LITERAL"
	request := `{"rawQuery":"` + raw + `","time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table"}`
	decoded, compiled, intent, limit, mode, err := decodeRunQuery(strings.NewReader(request), "loki", time.Now().UTC())
	if err != nil || compiled.Expression != raw || compiled.Canonical != nil || decoded.Recipe != nil || limit != 100 || mode != "table" || intent.Duration != "15m" {
		t.Fatalf("raw request contract failed: %v", err)
	}
	recipe := safeMetricRecipe()
	bytes, _ := json.Marshal(RunQueryRequest{Recipe: &recipe})
	if _, _, _, _, _, err := decodeRunQuery(strings.NewReader(string(bytes)), "prometheus", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	for _, forged := range []string{
		strings.Replace(request, `"rawQuery":`, `"rawQuery":"x","rawQuery":`, 1),
		strings.Replace(request, `"limit":100`, `"limit":100,"tenantId":11`, 1),
		strings.Replace(request, `"limit":100`, `"limit":100,"actorId":31`, 1),
		strings.Replace(request, `"limit":100`, `"limit":100,"name":"SECRET_NAME"`, 1),
		strings.Replace(request, `"limit":100`, `"limit":5001`, 1),
		strings.Replace(request, `"rawQuery":"`+raw+`"`, `"rawQuery":"`+raw+`","recipe":{}`, 1),
		string(bytes[:len(bytes)-1]) + `,"time":{"mode":"relative","duration":"5m"}}`,
		strings.Repeat(" ", 16385) + request,
	} {
		if _, _, _, _, _, err := decodeRunQuery(strings.NewReader(forged), "loki", time.Now().UTC()); err == nil {
			t.Fatal("forged envelope accepted")
		} else if strings.Contains(err.Error(), "SECRET") {
			t.Fatal("forged literal entered error")
		}
	}
	if _, _, _, _, _, err := decodeRunQuery(strings.NewReader(request), "tempo", time.Now().UTC()); err == nil {
		t.Fatal("raw Tempo query accepted")
	}
}
