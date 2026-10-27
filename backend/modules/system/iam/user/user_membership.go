package user

import (
	"fmt"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"gorm.io/gorm"
)

// 租户成员关系（tenant_memberships）由用户服务维护。
//
// 为什么放在这里：登录门禁（pkg/tenant.GateSessionIssuance）在 multi 模式下要求
// 「active membership + 租户主数据 active」。创建用户时只写 system_user 会得到一个
// 「列表里看得见、却登不进去」的半成品——实测返回 403 tenant.forbidden（验收 F1）。
// 唯一的绕过办法是平台管理员再调一次 POST /system/tenant/:id/members，
// 这恰好废掉了租户自助开通的意义。
//
// 用裸表写入而不是 import modules/system/iam/tenant，是为了不引入
// user -> tenant 的模块依赖（tenant 已经在 provision 里用同样方式写 system_user）。

const tenantMembershipTable = "tenant_memberships"

const (
	membershipRoleMember     = tenant.MembershipRoleMember
	membershipStatusActive   = "active"
	membershipStatusDisabled = "disabled"
)

// ensureTenantMembership 在创建用户的同一个事务里补齐成员关系。
//
// tenantID 为 0（平台/全局作用域）时什么都不做：平台主体没有成员关系，
// GateSessionIssuance 对 tenant_id = 0 直接放行（contract §2.3/§5）。
func ensureTenantMembership(tx *gorm.DB, tenantID, userID uint64) error {
	if tx == nil || tenantID == tenant.PlatformGlobalTenantID || userID == 0 {
		return nil
	}

	var existing int64
	if err := tx.Table(tenantMembershipTable).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Count(&existing).Error; err != nil {
		return fmt.Errorf("check tenant membership: %w", err)
	}
	if existing > 0 {
		// 已有行（例如用户名被复用后同 id 再次启用）只需确保它是 active，
		// 不要插第二行——唯一键是 (tenant_id, user_id)。
		return tx.Table(tenantMembershipTable).
			Where("tenant_id = ? AND user_id = ?", tenantID, userID).
			Update("status", membershipStatusActive).Error
	}

	row := struct {
		TenantID uint64
		UserID   uint64
		Role     string
		Status   string
	}{
		TenantID: tenantID,
		UserID:   userID,
		Role:     membershipRoleMember,
		Status:   membershipStatusActive,
	}
	if err := tx.Table(tenantMembershipTable).Create(&row).Error; err != nil {
		return fmt.Errorf("create tenant membership: %w", err)
	}
	return nil
}

// deactivateTenantMemberships 在删除用户时把成员关系置为 disabled。
//
// 不物理删除：成员关系是「曾经属于这个租户」的审计记录，登录门禁已经把
// 非 active 视为拒绝，因此禁用就足以让被删用户彻底登不进去（验收 F4：
// 物理删除会让成员列表堆满 __deleted_user_* 幽灵行）。
//
// 作用域：租户作用域只动本租户的行（§2.3.1 第 2 条：租户只能写自己那份）；
// 平台作用域才清理该用户的全部成员关系。
func deactivateTenantMemberships(tx *gorm.DB, tenantID, userID uint64) error {
	if tx == nil || userID == 0 {
		return nil
	}
	db := tx.Table(tenantMembershipTable).
		Where("user_id = ? AND status = ?", userID, membershipStatusActive)
	if tenantID != tenant.PlatformGlobalTenantID {
		db = db.Where("tenant_id = ?", tenantID)
	}
	return db.Update("status", membershipStatusDisabled).Error
}
