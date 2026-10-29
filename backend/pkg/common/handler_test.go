package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newParseUintParamEngine(result *uint64, ok *bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/things/:id", func(c *gin.Context) {
		*result, *ok = ParseUintParam(c, "id")
		if *ok {
			c.Status(http.StatusNoContent)
		}
	})
	return engine
}

func TestParseUintParamAcceptsValidAndZero(t *testing.T) {
	var got uint64
	var ok bool
	engine := newParseUintParamEngine(&got, &ok)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/things/42", nil))
	require.True(t, ok)
	require.Equal(t, uint64(42), got)
	// The success path must not have written any response.
	require.Equal(t, http.StatusNoContent, recorder.Code)

	// Zero parses through: the call sites that treat 0 as invalid keep that
	// check local (e.g. tenant), so the shared helper preserves the majority
	// shape — parse error is the only failure here.
	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/things/0", nil))
	require.True(t, ok)
	require.Equal(t, uint64(0), got)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestParseUintParamRejectsInvalidInputs(t *testing.T) {
	var got uint64
	var ok bool
	engine := newParseUintParamEngine(&got, &ok)

	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"garbage", "abc"},
		{"negative", "-1"},
		{"float", "1.5"},
		{"oversized overflow", "99999999999999999999999"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/things/"+tc.raw, nil))
			require.False(t, ok)
			require.Zero(t, got)
			// The failure path writes the exact envelope the inline blocks
			// write today: HTTP 200 carrying business code 400 inside.
			require.Equal(t, http.StatusOK, recorder.Code)
			var envelope struct {
				Code    int             `json:"code"`
				Data    json.RawMessage `json:"data"`
				Message string          `json:"message"`
			}
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &envelope))
			require.Equal(t, 400, envelope.Code)
			require.Equal(t, ErrParamInvalid, envelope.Message)
			require.Equal(t, ErrParamInvalid, "param.invalid")
			require.Contains(t, string(envelope.Data), "null")
		})
	}
}

// A handler can name a parameter its route does not declare; gin then yields
// an empty Param value. That path is unreachable through the router above
// ("/things/" 404s before the handler runs), so it is driven directly.
func TestParseUintParamRejectsMissingParamValue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/things", nil)

	got, ok := ParseUintParam(ctx, "id")
	require.False(t, ok)
	require.Zero(t, got)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"code":400`)
	require.Contains(t, recorder.Body.String(), `"message":"param.invalid"`)
}

func TestSharedMessageKeyConstantsMatchInlineValues(t *testing.T) {
	require.Equal(t, "param.invalid", ErrParamInvalid)
	require.Equal(t, "request.failed", ErrRequestFailed)
}
