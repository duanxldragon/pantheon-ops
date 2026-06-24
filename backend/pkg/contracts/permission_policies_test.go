package contracts

import (
	"reflect"
	"testing"
)

func TestRequiredAPIPoliciesByPermissionKeyBizScope(t *testing.T) {
	tests := []struct {
		name           string
		permissionKey  string
		expectedPolicy []PermissionAPIPolicy
	}{
		{
			name:          "list",
			permissionKey: "business:bizscope:list",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/bizscope/list", Method: "GET"},
				{Path: "/api/v1/business/bizscope/options", Method: "GET"},
			},
		},
		{
			name:          "view",
			permissionKey: "business:bizscope:view",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/bizscope/:id", Method: "GET"},
				{Path: "/api/v1/business/bizscope/:id/hosts", Method: "GET"},
				{Path: "/api/v1/business/bizscope/:id/available-hosts", Method: "GET"},
			},
		},
		{
			name:          "create",
			permissionKey: "business:bizscope:create",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/bizscope", Method: "POST"},
			},
		},
		{
			name:          "update",
			permissionKey: "business:bizscope:update",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/bizscope/:id", Method: "PUT"},
				{Path: "/api/v1/business/bizscope/:id/hosts/bind", Method: "POST"},
				{Path: "/api/v1/business/bizscope/:id/hosts/:hostId", Method: "DELETE"},
			},
		},
		{
			name:          "delete",
			permissionKey: "business:bizscope:delete",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/bizscope/:id", Method: "DELETE"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policies := RequiredAPIPoliciesByPermissionKey(tt.permissionKey)
			if !reflect.DeepEqual(policies, tt.expectedPolicy) {
				t.Fatalf("unexpected policies for %s: got %+v want %+v", tt.permissionKey, policies, tt.expectedPolicy)
			}
		})
	}
}

func TestRequiredAPIPoliciesByPermissionKeyDeployTask(t *testing.T) {
	tests := []struct {
		name           string
		permissionKey  string
		expectedPolicy []PermissionAPIPolicy
	}{
		{
			name:          "detail",
			permissionKey: "business:deploy:task:detail",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/deploy/tasks/:id", Method: "GET"},
			},
		},
		{
			name:          "update",
			permissionKey: "business:deploy:task:update",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/deploy/tasks/:id", Method: "PUT"},
			},
		},
		{
			name:          "delete",
			permissionKey: "business:deploy:task:delete",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/deploy/tasks/:id", Method: "DELETE"},
			},
		},
		{
			name:          "cancel",
			permissionKey: "business:deploy:task:cancel",
			expectedPolicy: []PermissionAPIPolicy{
				{Path: "/api/v1/business/deploy/tasks/:id/cancel", Method: "POST"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policies := RequiredAPIPoliciesByPermissionKey(tt.permissionKey)
			if !reflect.DeepEqual(policies, tt.expectedPolicy) {
				t.Fatalf("unexpected policies for %s: got %+v want %+v", tt.permissionKey, policies, tt.expectedPolicy)
			}
		})
	}
}
