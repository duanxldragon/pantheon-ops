package iam

import (
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common/security"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SessionLifecycle interface {
	RevokeUserSessions(userID uint64, now time.Time) (int64, error)
	DeleteUserSessions(userID uint64) error
	PurgeUserAuthArtifacts(userID uint64) error
	RevokeUserTokens(userID uint64) error
}

type SessionLifecycleFactory func(db *gorm.DB) SessionLifecycle

type UserServiceOption func(*UserService)

type UserService struct {
	db                      *gorm.DB
	sessionLifecycleFactory SessionLifecycleFactory
}

func WithSessionLifecycle(factory SessionLifecycleFactory) UserServiceOption {
	return func(s *UserService) {
		s.sessionLifecycleFactory = factory
	}
}

// NewUserService 构造函数
func NewUserService(db *gorm.DB, options ...UserServiceOption) *UserService {
	s := &UserService{
		db: db,
	}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *UserService) withSessionLifecycle(db *gorm.DB, fn func(SessionLifecycle) error) error {
	if s == nil || s.sessionLifecycleFactory == nil {
		return nil
	}
	lifecycle := s.sessionLifecycleFactory(db)
	if lifecycle == nil {
		return nil
	}
	return fn(lifecycle)
}

// Migrate 初始化表结构和种子数据
func (s *UserService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	if err := s.db.AutoMigrate(&SystemUser{}, &SystemUserRole{}, &SystemUserProfileExt{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *UserService) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.normalizeUserPreferenceJSON(); err != nil {
		return err
	}
	if err := s.ensureAdminUserSeed(); err != nil {
		return err
	}
	return s.ensureAdminRoleBinding()
}

// CleanupDeletedUsernames releases soft-deleted usernames by prefixing them.
// This is separated from Migrate as it is a data cleanup task, not a schema migration.
func (s *UserService) CleanupDeletedUsernames() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.releaseDeletedUsernames()
}

// GetUserRoles 获取用户角色标识。
func (s *UserService) GetUserRoles(userID uint64) ([]string, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var roles []string
	err := s.db.Table("system_role").
		Select("system_role.role_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role.id").
		Where("system_user_role.user_id = ? AND system_role.status = ?", userID, common.StatusEnabled).
		Pluck("system_role.role_key", &roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserPerms 获取用户按钮/接口权限标识。
func (s *UserService) GetUserPerms(userID uint64) ([]string, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var permissionKeys []string
	err := s.db.Table("system_role_permission").
		Select("DISTINCT system_role_permission.permission_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role_permission.role_id").
		Where("system_user_role.user_id = ? AND system_role_permission.permission_key <> ''", userID).
		Pluck("system_role_permission.permission_key", &permissionKeys).Error
	if err != nil {
		return nil, err
	}
	return mergeUserPermissionKeys(permissionKeys), nil
}

// GetProfile 获取当前登录用户个人中心信息。
func (s *UserService) GetProfile(userID uint64) (*UserProfileResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	roles, err := s.GetUserRoles(user.ID)
	if err != nil {
		return nil, err
	}
	perms, err := s.GetUserPerms(user.ID)
	if err != nil {
		return nil, err
	}
	profileExt, err := s.loadUserProfileExt(user.ID)
	if err != nil {
		return nil, err
	}

	return &UserProfileResp{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Email:       user.Email,
		Phone:       user.Phone,
		Preferences: ParseUserPlatformPreferences(user.PreferenceJSON),
		DeptID:      user.DeptID,
		PostID:      user.PostID,
		Status:      user.Status,
		Roles:       roles,
		Perms:       perms,
		ProfileExt:  profileExt,
		CreatedAt:   formatUserTime(user.CreatedAt),
	}, nil
}

// ListUsers 获取用户列表。
func (s *UserService) ListUsers(query *UserListQuery, dataScope *common.DataScopeReq) (*UserListPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var users []SystemUser
	db := s.db.Model(&SystemUser{}).Scopes(database.WithDataScope(dataScope))
	db = applyUserListFilters(db, query)
	page, pageSize, err := normalizeUserPageQuery(query)
	if err != nil {
		return nil, err
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	sortColumn, sortDesc := normalizeUserSort(query)
	if err := db.
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortColumn},
			Desc:   sortDesc,
		}).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uint64, 0, len(users))
	for _, item := range users {
		userIDs = append(userIDs, item.ID)
	}

	userRoleIDs, userRoleKeys, userRoleNames, err := s.loadUserRoles(userIDs)
	if err != nil {
		return nil, err
	}

	list := make([]UserListResp, 0, len(users))
	deptNames, err := s.loadDeptNames(users)
	if err != nil {
		return nil, err
	}
	postNames, err := s.loadPostNames(users)
	if err != nil {
		return nil, err
	}
	for _, item := range users {
		list = append(list, UserListResp{
			ID:        item.ID,
			Username:  item.Username,
			Nickname:  item.Nickname,
			Email:     item.Email,
			Phone:     item.Phone,
			DeptID:    item.DeptID,
			DeptName:  deptNames[item.DeptID],
			PostID:    item.PostID,
			PostName:  postNames[item.PostID],
			Status:    item.Status,
			CreatedAt: formatUserTime(item.CreatedAt),
			RoleIDs:   userRoleIDs[item.ID],
			RoleKeys:  userRoleKeys[item.ID],
			RoleNames: userRoleNames[item.ID],
		})
	}
	return &UserListPageResp{
		Items:    list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetUserDetail 获取用户详情。
func (s *UserService) GetUserDetail(userID uint64) (*UserDetailResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	roleIDsMap, roleKeysMap, roleNamesMap, err := s.loadUserRoles([]uint64{user.ID})
	if err != nil {
		return nil, err
	}
	deptNames, err := s.loadDeptNames([]SystemUser{user})
	if err != nil {
		return nil, err
	}
	postNames, err := s.loadPostNames([]SystemUser{user})
	if err != nil {
		return nil, err
	}
	profileExt, err := s.loadUserProfileExt(user.ID)
	if err != nil {
		return nil, err
	}
	lastLoginAt, err := s.loadUserLastLoginAt(user.Username)
	if err != nil {
		return nil, err
	}

	return &UserDetailResp{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Email:       user.Email,
		Phone:       user.Phone,
		DeptID:      user.DeptID,
		DeptName:    deptNames[user.DeptID],
		PostID:      user.PostID,
		PostName:    postNames[user.PostID],
		Status:      user.Status,
		CreatedAt:   formatUserTime(user.CreatedAt),
		UpdatedAt:   formatUserTime(user.UpdatedAt),
		LastLoginAt: lastLoginAt,
		RoleIDs:     roleIDsMap[user.ID],
		RoleKeys:    roleKeysMap[user.ID],
		RoleNames:   roleNamesMap[user.ID],
		ProfileExt:  profileExt,
	}, nil
}

// CreateUser 创建用户。
func (s *UserService) CreateUser(req *UserCreateReq) (*UserListResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if err := s.validateUserCreate(req); err != nil {
		return nil, err
	}
	profileExtJSON, err := marshalUserProfileExt(req.ProfileExt)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), security.GetBcryptCost())
	if err != nil {
		return nil, err
	}

	user := SystemUser{
		Username: req.Username,
		Password: string(passwordHash),
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Email:    req.Email,
		Phone:    req.Phone,
		DeptID:   req.DeptID,
		PostID:   req.PostID,
		Status:   normalizeStatus(req.Status),
	}
	roleIDs := normalizeUint64IDs(req.RoleIDs)

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if profileExtJSON != "" {
			if err := upsertUserProfileExt(tx, user.ID, profileExtJSON); err != nil {
				return err
			}
		}
		return replaceUserRoles(tx, user.ID, roleIDs)
	})
	if err != nil {
		return nil, err
	}

	return &UserListResp{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Phone:     user.Phone,
		DeptID:    user.DeptID,
		DeptName:  "",
		PostID:    user.PostID,
		PostName:  "",
		Status:    user.Status,
		CreatedAt: formatUserTime(user.CreatedAt),
		RoleIDs:   roleIDs,
		RoleKeys:  []string{},
	}, nil
}

// UpdateUser 更新用户。
func (s *UserService) UpdateUser(userID uint64, req *UserUpdateReq) (*UserListResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	if err := s.validateUserUpdate(&user, req); err != nil {
		return nil, err
	}
	profileExtJSON, err := marshalUserProfileExt(req.ProfileExt)
	if err != nil {
		return nil, err
	}
	roleIDs := normalizeUint64IDs(req.RoleIDs)

	updates := map[string]interface{}{
		"nickname": req.Nickname,
		"avatar":   req.Avatar,
		"email":    req.Email,
		"phone":    req.Phone,
		"dept_id":  req.DeptID,
		"post_id":  req.PostID,
	}
	if common.IsEnabledStatus(req.Status) {
		updates["status"] = req.Status
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return updateUserInTransaction(tx, &user, userID, updates, req.ProfileExt, profileExtJSON, roleIDs)
	}); err != nil {
		return nil, err
	}
	// 单用户禁用同样即时吊销令牌与会话。
	if shouldRevokeUpdatedUserAccess(user.Status, req.Status) {
		if err := s.revokeUsersAccess([]uint64{userID}); err != nil {
			return nil, err
		}
	}
	return s.loadUpdatedUserResponse(userID)
}

func updateUserInTransaction(tx *gorm.DB, user *SystemUser, userID uint64, updates map[string]interface{}, profileExt map[string]interface{}, profileExtJSON string, roleIDs []uint64) error {
	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return err
	}
	if profileExt != nil {
		if err := upsertUserProfileExt(tx, userID, profileExtJSON); err != nil {
			return err
		}
	}
	return replaceUserRoles(tx, userID, roleIDs)
}

func shouldRevokeUpdatedUserAccess(previousStatus, requestedStatus int) bool {
	return common.IsEnabledStatus(requestedStatus) &&
		requestedStatus == common.StatusDisabled &&
		previousStatus != common.StatusDisabled
}

func (s *UserService) loadUpdatedUserResponse(userID uint64) (*UserListResp, error) {
	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	roleIDMap, roleKeysMap, roleNamesMap, err := s.loadUserRoles([]uint64{user.ID})
	if err != nil {
		return nil, err
	}
	deptNames, err := s.loadDeptNames([]SystemUser{user})
	if err != nil {
		return nil, err
	}
	postNames, err := s.loadPostNames([]SystemUser{user})
	if err != nil {
		return nil, err
	}

	return &UserListResp{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Phone:     user.Phone,
		DeptID:    user.DeptID,
		DeptName:  deptNames[user.DeptID],
		PostID:    user.PostID,
		PostName:  postNames[user.PostID],
		Status:    user.Status,
		CreatedAt: formatUserTime(user.CreatedAt),
		RoleIDs:   roleIDMap[user.ID],
		RoleKeys:  roleKeysMap[user.ID],
		RoleNames: roleNamesMap[user.ID],
	}, nil
}

// UpdateProfile 更新当前登录用户个人资料。
func (s *UserService) UpdateProfile(userID uint64, req *UserProfileUpdateReq) (*UserProfileResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if err := validateOptionalEmail(req.Email); err != nil {
		return nil, err
	}
	profileExtJSON, err := marshalUserProfileExt(req.ProfileExt)
	if err != nil {
		return nil, err
	}

	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"nickname": strings.TrimSpace(req.Nickname),
		"avatar":   strings.TrimSpace(req.Avatar),
		"email":    strings.TrimSpace(req.Email),
		"phone":    strings.TrimSpace(req.Phone),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		if req.ProfileExt != nil {
			return upsertUserProfileExt(tx, userID, profileExtJSON)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return s.GetProfile(userID)
}

// ResetPassword 重置指定用户密码，并吊销该用户全部活跃会话。
func (s *UserService) ResetPassword(userID uint64, newPassword string) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}

	trimmedPassword := strings.TrimSpace(newPassword)
	if len(trimmedPassword) < s.getConfiguredPasswordMinLength() {
		return 0, common.NewBadRequest("user.update.error.password_too_short")
	}

	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return 0, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(trimmedPassword), security.GetBcryptCost())
	if err != nil {
		return 0, err
	}

	var revokedSessionCount int64
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password", string(passwordHash)).Error; err != nil {
			return err
		}

		return s.withSessionLifecycle(tx, func(lifecycle SessionLifecycle) error {
			var err error
			// 令牌吊销要在 DB 会话标记 revoked 之前拉取 session_id 列表。
			if err = lifecycle.RevokeUserTokens(userID); err != nil {
				return err
			}
			revokedSessionCount, err = lifecycle.RevokeUserSessions(userID, time.Now())
			return err
		})
	}); err != nil {
		return 0, err
	}

	return revokedSessionCount, nil
}

func (s *UserService) BatchUpdateUserStatus(userIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	normalizedIDs := normalizeUint64IDs(userIDs)
	if len(normalizedIDs) == 0 {
		return 0, common.NewBadRequest("user.batch.empty")
	}
	if !common.IsEnabledStatus(status) {
		return 0, common.NewBadRequest("param.invalid")
	}

	var users []SystemUser
	if err := s.db.Where("id IN ?", normalizedIDs).Find(&users).Error; err != nil {
		return 0, err
	}
	if len(users) != len(normalizedIDs) {
		return 0, common.NewNotFound("user.batch.not_found")
	}
	if status == common.StatusDisabled {
		for _, user := range users {
			if user.ID == common.BuiltinAdminUserID {
				return 0, common.NewForbidden("user.update.error.protected")
			}
		}
	}

	if err := s.db.Model(&SystemUser{}).
		Where("id IN ?", normalizedIDs).
		Updates(map[string]any{
			"status":     normalizeStatus(status),
			"updated_at": time.Now(),
		}).Error; err != nil {
		return 0, err
	}

	// 禁用即吊销：黑名单 + 会话，令牌不再等 TTL 自然过期。
	if normalizeStatus(status) == common.StatusDisabled {
		if err := s.revokeUsersAccess(normalizedIDs); err != nil {
			return 0, err
		}
	}

	return len(normalizedIDs), nil
}

// revokeUsersAccess 吊销一批用户的 Redis 令牌并撤销其 DB 会话。
func (s *UserService) revokeUsersAccess(userIDs []uint64) error {
	now := time.Now()
	return s.withSessionLifecycle(s.db, func(lifecycle SessionLifecycle) error {
		for _, id := range userIDs {
			if err := lifecycle.RevokeUserTokens(id); err != nil {
				return err
			}
			if _, err := lifecycle.RevokeUserSessions(id, now); err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteUser 删除用户。
func (s *UserService) DeleteUser(userID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if userID == common.BuiltinAdminUserID {
		return common.NewForbidden("user.delete.error.protected")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.deleteUserInTransaction(tx, userID)
	})
}

func (s *UserService) deleteUserInTransaction(tx *gorm.DB, userID uint64) error {
	var user SystemUser
	if err := tx.First(&user, userID).Error; err != nil {
		return err
	}
	if err := s.deleteUserAuthArtifacts(tx, userID); err != nil {
		return err
	}
	if err := deleteUserRelations(tx, userID); err != nil {
		return err
	}
	deletedUsername, err := s.allocateDeletedUsername(tx, user.ID)
	if err != nil {
		return err
	}
	if err := tx.Model(&user).Update("username", deletedUsername).Error; err != nil {
		return err
	}
	return tx.Delete(&user).Error
}

func (s *UserService) deleteUserAuthArtifacts(tx *gorm.DB, userID uint64) error {
	return s.withSessionLifecycle(tx, func(lifecycle SessionLifecycle) error {
		// 先吊销 Redis 令牌（需要还能查到 session_id），再删 DB 会话行。
		if err := lifecycle.RevokeUserTokens(userID); err != nil {
			return err
		}
		if err := lifecycle.DeleteUserSessions(userID); err != nil {
			return err
		}
		return lifecycle.PurgeUserAuthArtifacts(userID)
	})
}

func deleteUserRelations(tx *gorm.DB, userID uint64) error {
	if err := tx.Exec("DELETE FROM system_user_role WHERE user_id = ?", userID).Error; err != nil {
		return err
	}
	// 用户档案扩展随用户一并清理，避免孤儿行。
	return tx.Where("user_id = ?", userID).Delete(&SystemUserProfileExt{}).Error
}
