package tenant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
