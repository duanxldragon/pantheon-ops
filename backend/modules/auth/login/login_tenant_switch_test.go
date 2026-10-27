package login

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

func TestTenantSwitchErrorCodeClassifiesClientRejections(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"missing session", gorm.ErrRecordNotFound, common.CodeUnauthorized},
		{"revoked session", common.ErrUnauthorized, common.CodeUnauthorized},
		{"same tenant", common.NewBadRequest("tenant.switch.same"), common.CodeParamInvalid},
		{"forbidden tenant", tenantctx.ErrTenantForbidden, common.CodeForbidden},
		{"suspended tenant", tenantctx.ErrTenantSuspended, common.CodeForbidden},
		{"archived tenant", tenantctx.ErrTenantArchived, common.CodeForbidden},
		{"unexpected database failure", errors.New("connection reset"), common.CodeError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tenantSwitchErrorCode(tc.err); got != tc.want {
				t.Fatalf("tenantSwitchErrorCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
