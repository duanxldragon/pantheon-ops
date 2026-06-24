package iam

import (
	"errors"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserService struct {
	db *gorm.DB
}

// NewUserService 构造函数
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
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
		Where("system_user_role.user_id = ? AND system_role.status = ?", userID, 1).
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
	page, pageSize := normalizeUserPageQuery(query)

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

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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
	if req.Status == 1 || req.Status == 2 {
		updates["status"] = req.Status
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Updates(updates).Error; err != nil {
			return err
		}
		if req.ProfileExt != nil {
			if err := upsertUserProfileExt(tx, userID, profileExtJSON); err != nil {
				return err
			}
		}
		return replaceUserRoles(tx, userID, roleIDs)
	}); err != nil {
		return nil, err
	}
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
		return 0, errors.New("user.update.error.password_too_short")
	}

	var user SystemUser
	if err := s.db.First(&user, userID).Error; err != nil {
		return 0, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(trimmedPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	var revokedSessionCount int64
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&user).Update("password", string(passwordHash)).Error; err != nil {
			return err
		}

		now := time.Now()
		result := tx.Table("system_user_session").
			Where("user_id = ? AND revoked_at IS NULL", userID).
			Updates(map[string]interface{}{"revoked_at": &now})
		if result.Error != nil {
			return result.Error
		}
		revokedSessionCount = result.RowsAffected
		return nil
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
		return 0, errors.New("user.batch.empty")
	}
	if status != 1 && status != 2 {
		return 0, errors.New("param.invalid")
	}

	var users []SystemUser
	if err := s.db.Where("id IN ?", normalizedIDs).Find(&users).Error; err != nil {
		return 0, err
	}
	if len(users) != len(normalizedIDs) {
		return 0, errors.New("user.batch.not_found")
	}
	if status == 2 {
		for _, user := range users {
			if user.ID == 1 {
				return 0, errors.New("user.update.error.protected")
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

	return len(normalizedIDs), nil
}

// DeleteUser 删除用户。
func (s *UserService) DeleteUser(userID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if userID == 1 {
		return errors.New("user.delete.error.protected")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var user SystemUser
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM system_user_session WHERE user_id = ?", userID).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM system_user_role WHERE user_id = ?", userID).Error; err != nil {
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
	})
}
