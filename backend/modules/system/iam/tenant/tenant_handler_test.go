package tenant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTenantMemberPagination(t *testing.T) {
	page, pageSize := normalizeTenantMemberPagination(-1, 1000)
	require.Equal(t, 1, page)
	require.Equal(t, maxTenantMemberPageSize, pageSize)

	page, pageSize = normalizeTenantMemberPagination(2, 0)
	require.Equal(t, 2, page)
	require.Equal(t, defaultTenantMemberPageSize, pageSize)

	page, pageSize = normalizeTenantMemberPagination(int(^uint(0)>>1), 100)
	require.LessOrEqual(t, (page-1)*pageSize, int(^uint(0)>>1))
}

func TestListTenantMembersHandlerReturnsPaginationContract(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	tenant, err := service.CreateTenant(CreateTenantDTO{Code: "handler-page", Name: "Handler Page"})
	require.NoError(t, err)
	for userID := uint64(101); userID <= 103; userID++ {
		require.NoError(t, service.AddMember(tenant.ID, userID, "member"))
	}

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine.Group(""), NewHandler(service))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenants/%d/members/page?page=2&pageSize=2", tenant.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var result struct {
		Items    []Membership `json:"items"`
		Total    int64        `json:"total"`
		Page     int          `json:"page"`
		PageSize int          `json:"pageSize"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Equal(t, int64(3), result.Total)
	require.Equal(t, 2, result.Page)
	require.Equal(t, 2, result.PageSize)
	require.Len(t, result.Items, 1)
	require.Equal(t, uint64(103), result.Items[0].UserID)

	for _, tc := range []struct {
		query    string
		page     int
		pageSize int
	}{
		{"?page=0&pageSize=0", 1, defaultTenantMemberPageSize},
		{"?page=invalid&pageSize=1000", 1, maxTenantMemberPageSize},
		{"?page=999999999999999999999&pageSize=-1", 1, defaultTenantMemberPageSize},
	} {
		recorder = httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenants/%d/members/page%s", tenant.ID, tc.query), nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
		require.Equal(t, tc.page, result.Page)
		require.Equal(t, tc.pageSize, result.PageSize)
		require.Equal(t, int64(3), result.Total)
		require.Len(t, result.Items, 3)
	}
}

func TestListTenantMembersHandlerUsesBoundedContract(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	tenant, err := service.CreateTenant(CreateTenantDTO{Code: "handler-legacy", Name: "Handler Legacy"})
	require.NoError(t, err)
	for userID := uint64(1); userID <= 105; userID++ {
		require.NoError(t, service.AddMember(tenant.ID, userID, "member"))
	}
	require.NoError(t, db.Create(&Membership{TenantID: tenant.ID, UserID: 106, Role: "member", Status: "inactive"}).Error)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine.Group(""), NewHandler(service))
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenants/%d/members?page=2&pageSize=1", tenant.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)

	var result map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Equal(t, float64(105), result["total"])
	require.Equal(t, float64(2), result["page"])
	require.Equal(t, float64(1), result["pageSize"])
	items, ok := result["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	require.Equal(t, float64(2), items[0].(map[string]any)["user_id"])

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenants/%d/members", tenant.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Equal(t, float64(105), result["total"])
	require.Equal(t, float64(defaultTenantMemberPageSize), result["pageSize"])
	require.Len(t, result["items"], defaultTenantMemberPageSize)

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/tenants/%d/members/page?page=2&pageSize=1", tenant.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	require.Equal(t, float64(105), result["total"])
	require.Equal(t, float64(2), result["page"])

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tenants/999999/members", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, "{\"items\":[],\"total\":0,\"page\":1,\"pageSize\":20}", recorder.Body.String())
}

// requireUnifiedErrorEnvelope asserts the platform error envelope shape written
// by common.Fail: HTTP 200 with {code, data, message}. The legacy raw success
// shape is asserted by the tests above; these pin the unified failure contract.
func requireUnifiedErrorEnvelope(t *testing.T, body string, wantCode int, wantMessage string) {
	t.Helper()

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &parsed), "error body must be a JSON envelope")
	require.Equal(t, float64(wantCode), parsed["code"], body)
	require.Equal(t, wantMessage, parsed["message"], body)
	require.Contains(t, parsed, "data", body)
	require.NotContains(t, parsed, "error", "legacy raw {\"error\": ...} shape must be gone: "+body)
	// Internal detail must never be echoed: bind/parse/driver strings are the
	// realistic leak vectors on these paths.
	for _, leak := range []string{"json:", "strconv:", "Error:", "Key:", "Field validation", "gorm", "sql"} {
		require.NotContains(t, body, leak, body)
	}
}

// TestLegacyTenantHandlerErrorEnvelopeIsUnifiedAndLeakFree covers the error
// paths that fail before the service is touched, so it runs without a database
// (the service pointer is never dereferenced on these paths).
func TestLegacyTenantHandlerErrorEnvelopeIsUnifiedAndLeakFree(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine.Group(""), NewHandler(NewService(nil)))

	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		wantCode    int
		wantMessage string
	}{
		{"get bad id", http.MethodGet, "/tenants/abc", "", 400, "param.invalid"},
		{"list bad filter", http.MethodGet, "/tenants?status=bogus", "", 400, "param.invalid"},
		{"create bad json", http.MethodPost, "/tenants", `{"code":`, 400, "param.invalid"},
		{"update bad id", http.MethodPut, "/tenants/abc", `{"name":"x"}`, 400, "param.invalid"},
		{"update bad json", http.MethodPut, "/tenants/1", `not-json`, 400, "param.invalid"},
		{"delete bad id", http.MethodDelete, "/tenants/-1", "", 400, "param.invalid"},
		{"add member bad id", http.MethodPost, "/tenants/x/members", `{"user_id":1,"role":"member"}`, 400, "param.invalid"},
		{"add member bad json", http.MethodPost, "/tenants/1/members", `{"user_id":`, 400, "param.invalid"},
		{"remove member bad user id", http.MethodDelete, "/tenants/1/members/abc", "", 400, "param.invalid"},
		{"members bad id", http.MethodGet, "/tenants/abc/members", "", 400, "param.invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			// common.Fail answers with HTTP 200 and the business code inside
			// the envelope, matching every other system module.
			require.Equal(t, http.StatusOK, recorder.Code)
			requireUnifiedErrorEnvelope(t, recorder.Body.String(), tc.wantCode, tc.wantMessage)
		})
	}

	t.Run("my tenants without auth context", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tenants/my", nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		requireUnifiedErrorEnvelope(t, recorder.Body.String(), 401, "token.invalid")
	})

	t.Run("my tenants with malformed auth context", func(t *testing.T) {
		bad := gin.New()
		bad.Use(func(c *gin.Context) {
			c.Set("user_id", "7")
			c.Next()
		})
		RegisterRoutes(bad.Group(""), NewHandler(NewService(nil)))
		recorder := httptest.NewRecorder()
		bad.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tenants/my", nil))
		require.Equal(t, http.StatusOK, recorder.Code)
		requireUnifiedErrorEnvelope(t, recorder.Body.String(), 500, "request.failed")
		require.NotContains(t, recorder.Body.String(), "invalid user id type")
	})
}

// TestLegacyTenantHandlerSentinelMappingRequiresNoInternalLeak drives the
// service-error branches with a real database and asserts the sentinel texts
// ("tenant not found", "tenant code already exists") are replaced by i18n
// keys instead of being echoed to the client. Skips without MySQL.
func TestLegacyTenantHandlerSentinelMappingRequiresNoInternalLeak(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "legacy-envelope", Name: "Legacy Envelope"})
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine.Group(""), NewHandler(service))

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/tenants/999999", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	requireUnifiedErrorEnvelope(t, recorder.Body.String(), 404, "tenant.not_found")
	require.NotContains(t, recorder.Body.String(), "tenant not found")

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/tenants",
		strings.NewReader(fmt.Sprintf(`{"code":%q,"name":"dup"}`, created.Code))))
	require.Equal(t, http.StatusOK, recorder.Code)
	requireUnifiedErrorEnvelope(t, recorder.Body.String(), 400, "tenant.code_taken")
	require.NotContains(t, recorder.Body.String(), "tenant code already exists")

	require.NoError(t, service.AddMember(created.ID, 88001, "member"))

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/tenants/%d", created.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	requireUnifiedErrorEnvelope(t, recorder.Body.String(), 400, "tenant.has_active_members")
	require.NotContains(t, recorder.Body.String(), "tenant has active users")
}
