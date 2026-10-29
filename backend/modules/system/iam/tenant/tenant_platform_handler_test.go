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

// newPlatformTenantEngine registers the production PlatformHandler on the same
// path shape the system module mounts (system_modules.go registers it under
// /api/v1/system with platform middlewares in front). The middleware chain is
// out of scope here: these tests pin the envelope, pagination and
// sentinel-mapping contract of the handler that actually ships. They moved
// here from the deleted legacy Handler test file (W8).
func newPlatformTenantEngine(service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewPlatformHandler(service)
	engine := gin.New()
	system := engine.Group("/api/v1/system")
	{
		system.GET("/tenant/list", handler.ListTenants)
		system.GET("/tenant/:id", handler.GetTenant)
		system.POST("/tenant", handler.CreateTenant)
		system.PUT("/tenant/:id", handler.UpdateTenant)
		system.DELETE("/tenant/:id", handler.DeleteTenant)
		system.GET("/tenant/:id/members", handler.ListTenantMembers)
		system.POST("/tenant/:id/members", handler.AddTenantMember)
		system.DELETE("/tenant/:id/members/:userId", handler.RemoveTenantMember)
	}
	return engine
}

func decodePlatformEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result))
	return result
}

func TestPlatformTenantHandlerReturnsPaginationContract(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "platform-page", Name: "Platform Page"})
	require.NoError(t, err)
	mustInsertTenantUsers(t, db, created.ID, 101, 3)
	for userID := uint64(101); userID <= 103; userID++ {
		require.NoError(t, service.AddMember(created.ID, userID, "member"))
	}

	engine := newPlatformTenantEngine(service)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/system/tenant/%d/members?page=2&pageSize=2", created.ID), nil))

	result := decodePlatformEnvelope(t, recorder)
	require.Equal(t, float64(200), result["code"])
	require.Equal(t, "success", result["message"])
	data, ok := result["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(3), data["total"])
	require.Equal(t, float64(2), data["page"])
	require.Equal(t, float64(2), data["pageSize"])
	items, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(103), first["user_id"])

	for _, tc := range []struct {
		query    string
		page     int
		pageSize int
		items    int
	}{
		{"?page=0&pageSize=0", 1, defaultTenantMemberPageSize, 3},
		{"?page=invalid&pageSize=1000", 1, maxTenantMemberPageSize, 3},
		{"?page=999999999999999999999&pageSize=-1", 1, defaultTenantMemberPageSize, 3},
	} {
		recorder = httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/api/v1/system/tenant/%d/members%s", created.ID, tc.query), nil))
		result = decodePlatformEnvelope(t, recorder)
		data, ok = result["data"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, float64(tc.page), data["page"])
		require.Equal(t, float64(tc.pageSize), data["pageSize"])
		require.Equal(t, float64(3), data["total"])
		require.Len(t, data["items"], tc.items)
	}
}

func TestPlatformTenantHandlerUsesBoundedContract(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "platform-bounded", Name: "Platform Bounded"})
	require.NoError(t, err)
	mustInsertTenantUsers(t, db, created.ID, 1, 105)
	for userID := uint64(1); userID <= 105; userID++ {
		require.NoError(t, service.AddMember(created.ID, userID, "member"))
	}
	// An inactive membership must not count toward the active listing. The
	// user row is real (W2): no phantom ids in fixtures.
	mustInsertTenantUser(t, db, created.ID, 106)
	require.NoError(t, db.Create(&Membership{TenantID: created.ID, UserID: 106, Role: "member", Status: "inactive"}).Error)

	engine := newPlatformTenantEngine(service)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/system/tenant/%d/members?page=2&pageSize=1", created.ID), nil))

	result := decodePlatformEnvelope(t, recorder)
	data, ok := result["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(105), data["total"])
	require.Equal(t, float64(2), data["page"])
	require.Equal(t, float64(1), data["pageSize"])
	items, ok := data["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(2), first["user_id"])

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/v1/system/tenant/%d/members", created.ID), nil))
	result = decodePlatformEnvelope(t, recorder)
	data, ok = result["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(105), data["total"])
	require.Equal(t, float64(defaultTenantMemberPageSize), data["pageSize"])
	require.Len(t, data["items"], defaultTenantMemberPageSize)

	// An unknown tenant id is an empty page, not an error (contract §5: the
	// list path does not leak existence).
	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/system/tenant/999999/members", nil))
	result = decodePlatformEnvelope(t, recorder)
	require.Equal(t, float64(200), result["code"])
	data, ok = result["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(0), data["total"])
	require.Equal(t, float64(1), data["page"])
	require.Equal(t, float64(defaultTenantMemberPageSize), data["pageSize"])
	require.Len(t, data["items"], 0)
}

// requireUnifiedErrorEnvelope asserts the platform error envelope shape written
// by common.Fail: HTTP 200 with {code, data, message}. These pin the unified
// failure contract of the mounted PlatformHandler.
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

// TestPlatformTenantHandlerErrorEnvelopeIsUnifiedAndLeakFree covers the error
// paths that fail before the service is touched, so it runs without a database
// (the service pointer is never dereferenced on these paths).
func TestPlatformTenantHandlerErrorEnvelopeIsUnifiedAndLeakFree(t *testing.T) {
	engine := newPlatformTenantEngine(NewService(nil))

	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		wantCode    int
		wantMessage string
	}{
		{"get bad id", http.MethodGet, "/api/v1/system/tenant/abc", "", 400, "param.invalid"},
		{"list bad filter", http.MethodGet, "/api/v1/system/tenant/list?status=bogus", "", 400, "param.invalid"},
		{"create bad json", http.MethodPost, "/api/v1/system/tenant", `{"code":`, 400, "param.invalid"},
		{"update bad id", http.MethodPut, "/api/v1/system/tenant/abc", `{"name":"x"}`, 400, "param.invalid"},
		{"update bad json", http.MethodPut, "/api/v1/system/tenant/1", `not-json`, 400, "param.invalid"},
		{"delete bad id", http.MethodDelete, "/api/v1/system/tenant/-1", "", 400, "param.invalid"},
		{"add member bad id", http.MethodPost, "/api/v1/system/tenant/x/members", `{"user_id":1,"role":"member"}`, 400, "param.invalid"},
		{"add member bad json", http.MethodPost, "/api/v1/system/tenant/1/members", `{"user_id":`, 400, "param.invalid"},
		{"remove member bad user id", http.MethodDelete, "/api/v1/system/tenant/1/members/abc", "", 400, "param.invalid"},
		{"members bad id", http.MethodGet, "/api/v1/system/tenant/abc/members", "", 400, "param.invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			// common.Fail answers with HTTP 200 and the business code inside
			// the envelope, matching every other system module.
			requireUnifiedErrorEnvelope(t, recorder.Body.String(), tc.wantCode, tc.wantMessage)
		})
	}
}

// TestPlatformTenantHandlerSentinelMappingRequiresNoInternalLeak drives the
// service-error branches with a real database and asserts the sentinel texts
// ("tenant not found", "tenant code already exists") are replaced by i18n
// keys instead of being echoed to the client. Skips without MySQL.
func TestPlatformTenantHandlerSentinelMappingRequiresNoInternalLeak(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "platform-envelope", Name: "Platform Envelope"})
	require.NoError(t, err)

	engine := newPlatformTenantEngine(service)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/system/tenant/999999", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	requireUnifiedErrorEnvelope(t, recorder.Body.String(), 404, "tenant.not_found")
	require.NotContains(t, recorder.Body.String(), "tenant not found")

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/system/tenant",
		strings.NewReader(fmt.Sprintf(`{"code":%q,"name":"dup"}`, created.Code))))
	require.Equal(t, http.StatusOK, recorder.Code)
	requireUnifiedErrorEnvelope(t, recorder.Body.String(), 400, "tenant.code_taken")
	require.NotContains(t, recorder.Body.String(), "tenant code already exists")

	mustInsertTenantUser(t, db, created.ID, 88001)
	require.NoError(t, service.AddMember(created.ID, 88001, "member"))

	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/system/tenant/%d", created.ID), nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	// The member row alone no longer decides the refusal: an active tenant is
	// refused because it is not archived (contract §2.1).
	requireUnifiedErrorEnvelope(t, recorder.Body.String(), 400, "tenant.not_archived")
	require.NotContains(t, recorder.Body.String(), "tenant has active users")
	require.NotContains(t, recorder.Body.String(), "must be archived")
}
