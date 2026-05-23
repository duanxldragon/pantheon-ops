package iam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/impexp"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserService struct {
	db *gorm.DB
}

const deletedUsernamePrefix = "__deleted_user_"

// NewUserService 构造函数
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// Migrate 初始化表结构和种子数据
func (s *UserService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}

	if err := s.db.AutoMigrate(&SystemUser{}, &SystemUserRole{}, &SystemUserProfileExt{}); err != nil {
		return err
	}
	if err := s.normalizeUserPreferenceJSON(); err != nil {
		return err
	}
	if err := s.releaseDeletedUsernames(); err != nil {
		return err
	}
	if err := s.ensureAdminUserSeed(); err != nil {
		return err
	}
	return s.ensureAdminRoleBinding()
}

func (s *UserService) normalizeUserPreferenceJSON() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}

	type userPreferenceRow struct {
		ID             uint64 `gorm:"column:id"`
		PreferenceJSON string `gorm:"column:preference_json"`
	}

	var rows []userPreferenceRow
	if err := s.db.Unscoped().
		Model(&SystemUser{}).
		Select("id", "preference_json").
		Where("preference_json <> ''").
		Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		normalized, err := MarshalUserPlatformPreferences(ParseUserPlatformPreferences(row.PreferenceJSON))
		if err != nil {
			return err
		}
		if normalized == row.PreferenceJSON {
			continue
		}
		if err := s.db.Unscoped().
			Model(&SystemUser{}).
			Where("id = ?", row.ID).
			Update("preference_json", normalized).Error; err != nil {
			return err
		}
	}

	return nil
}

const (
	defaultConfiguredPasswordMinLength = 6
	defaultDevInitialAdminPassword     = "123456"
	productionInitialAdminMinLength    = 12
)

func (s *UserService) ensureAdminUserSeed() error {
	var count int64
	if err := s.db.Model(&SystemUser{}).Where("id = ?", 1).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initialPassword, err := resolveInitialAdminPassword()
	if err != nil {
		return err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	admin := SystemUser{
		Username: "admin",
		Password: string(passwordHash),
		Nickname: "Administrator",
		Status:   1,
	}
	admin.ID = 1
	return s.db.Create(&admin).Error
}

func resolveInitialAdminPassword() (string, error) {
	password := strings.TrimSpace(os.Getenv("PANTHEON_INITIAL_ADMIN_PASSWORD"))
	if !common.IsProductionEnv() {
		if password != "" {
			return password, nil
		}
		return defaultDevInitialAdminPassword, nil
	}
	if password == "" {
		return "", errors.New("admin.initial_password_required")
	}
	if len(password) < productionInitialAdminMinLength {
		return "", errors.New("admin.initial_password_too_short")
	}
	return password, nil
}

func (s *UserService) ensureAdminRoleBinding() error {
	if !s.db.Migrator().HasTable("system_user_role") || !s.db.Migrator().HasTable("system_role") {
		return nil
	}

	var adminRoleID uint64
	if err := s.db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &adminRoleID).Error; err != nil {
		return err
	}
	if adminRoleID == 0 {
		return nil
	}

	var count int64
	if err := s.db.Table("system_user_role").Where("user_id = ? AND role_id = ?", 1, adminRoleID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)", 1, adminRoleID).Error
}

// GetUserRoles 获取用户角色标识。
func (s *UserService) GetUserRoles(userID uint64) ([]string, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
	}

	var users []SystemUser
	db := s.db.Model(&SystemUser{}).Scopes(database.WithDataScope(dataScope))
	page, pageSize := normalizeUserPageQuery(query)
	if query != nil {
		if strings.TrimSpace(query.Username) != "" {
			db = db.Where("username LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Username)))
		}
		if strings.TrimSpace(query.Nickname) != "" {
			db = db.Where("nickname LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Nickname)))
		}
		if query.DeptID > 0 {
			db = db.Where("dept_id = ?", query.DeptID)
		}
		if query.PostID > 0 {
			db = db.Where("post_id = ?", query.PostID)
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
			db = db.Where("status = ?", *query.Status)
		}
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
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
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
		return nil, errors.New("database.not_initialized")
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
		return 0, errors.New("database.not_initialized")
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
		return 0, errors.New("database.not_initialized")
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
		return errors.New("database.not_initialized")
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

func (s *UserService) ExportUsers(query *UserListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	users, err := s.listUsersForExport(query)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uint64, 0, len(users))
	for _, item := range users {
		userIDs = append(userIDs, item.ID)
	}
	_, roleKeysMap, _, err := s.loadUserRoles(userIDs)
	if err != nil {
		return nil, err
	}
	deptPathByID, _, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}
	postCodeByID, err := s.loadPostCodes(users)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, []string{
			user.Username,
			"",
			user.Nickname,
			user.Email,
			user.Phone,
			deptPathByID[user.DeptID],
			postCodeByID[user.PostID],
			fmt.Sprintf("%d", user.Status),
			impexp.JoinStringSlice(roleKeysMap[user.ID], "|"),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-user-export.csv",
		Headers:  []string{"username", "password", "nickname", "email", "phone", "deptPath", "postCode", "status", "roleKeys"},
		Rows:     rows,
	}, nil
}

func (s *UserService) BuildUserImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-user-import-template.csv",
		Headers:  []string{"username", "password", "nickname", "email", "phone", "deptPath", "postCode", "status", "roleKeys"},
		Rows: [][]string{
			{"#说明：保留第一行表头；status 使用 1=启用、2=禁用；roleKeys 多角色用 | 分隔；deptPath 使用部门导出的完整路径；postCode 使用岗位编码；导入新用户 password 必填。", "", "", "", "", "", "", "", ""},
			{"#sample_user", "ChangeMe123", "示例用户", "sample@example.com", "13800138000", "", "", "1", "admin"},
		},
	}
}

func (s *UserService) ImportUsers(records [][]string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{
		Applied: false,
		Errors:  []impexp.ImportError{},
	}
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	requiredHeaders := []string{"username", "password", "nickname", "email", "phone", "deptPath", "postCode", "status", "roleKeys"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	_, deptPathToID, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}
	postIDByCode, err := s.loadPostIDsByCode()
	if err != nil {
		return nil, err
	}
	roleIDByKey, err := s.loadRoleIDsByKey()
	if err != nil {
		return nil, err
	}
	existingByUsername, err := s.loadUsersByUsername()
	if err != nil {
		return nil, err
	}

	type importRow struct {
		Username string
		Password string
		Create   *UserCreateReq
		Update   *UserUpdateReq
		Existing *SystemUser
	}

	rows := make([]importRow, 0, len(records)-1)
	seenUsernames := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		username := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "username"))
		password := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "password"))
		nickname := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "nickname"))
		email := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "email"))
		phone := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "phone"))
		deptPath := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "deptPath"))
		postCode := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "postCode"))
		roleKeys := impexp.SplitPipeValues(impexp.ReadCSVField(record, headerIndex, "roleKeys"))

		if username == "" {
			impexp.AppendImportError(result, rowNumber, "username", "user.username.required")
		}
		if firstRow, ok := seenUsernames[username]; ok && username != "" {
			impexp.AppendImportError(result, rowNumber, "username", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else if username != "" {
			seenUsernames[username] = rowNumber
		}
		if err := validateOptionalEmail(email); err != nil {
			impexp.AppendImportError(result, rowNumber, "email", err.Error())
		}

		var deptID uint64
		if deptPath != "" {
			deptID = deptPathToID[deptPath]
			if deptID == 0 {
				impexp.AppendImportError(result, rowNumber, "deptPath", "user.dept.invalid")
			}
		}
		var postID uint64
		if postCode != "" {
			postID = postIDByCode[postCode]
			if postID == 0 {
				impexp.AppendImportError(result, rowNumber, "postCode", "user.post.invalid")
			}
		}
		roleIDs := make([]uint64, 0, len(roleKeys))
		for _, roleKey := range roleKeys {
			roleID := roleIDByKey[roleKey]
			if roleID == 0 {
				impexp.AppendImportError(result, rowNumber, "roleKeys", "user.role.invalid")
				continue
			}
			roleIDs = append(roleIDs, roleID)
		}

		status := impexp.ParseEnabledStatus(impexp.ReadCSVField(record, headerIndex, "status"))
		existing := existingByUsername[username]
		if existing != nil {
			updateReq := &UserUpdateReq{
				Nickname: nickname,
				Email:    email,
				Phone:    phone,
				DeptID:   deptID,
				PostID:   postID,
				Status:   status,
				RoleIDs:  roleIDs,
			}
			if err := s.validateUserUpdate(existing, updateReq); err != nil {
				impexp.AppendImportError(result, rowNumber, "username", err.Error())
			}
			rows = append(rows, importRow{
				Username: username,
				Password: password,
				Update:   updateReq,
				Existing: existing,
			})
			continue
		}

		createReq := &UserCreateReq{
			Username: username,
			Password: password,
			Nickname: nickname,
			Email:    email,
			Phone:    phone,
			DeptID:   deptID,
			PostID:   postID,
			Status:   status,
			RoleIDs:  roleIDs,
		}
		if strings.TrimSpace(password) == "" {
			impexp.AppendImportError(result, rowNumber, "password", "user.password.required")
		}
		if err := s.validateUserCreate(createReq); err != nil {
			impexp.AppendImportError(result, rowNumber, "username", err.Error())
		}
		rows = append(rows, importRow{
			Username: username,
			Password: password,
			Create:   createReq,
		})
	}

	if result.Failed > 0 {
		return result, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			if row.Update != nil && row.Existing != nil {
				updates := map[string]interface{}{
					"nickname": row.Update.Nickname,
					"email":    row.Update.Email,
					"phone":    row.Update.Phone,
					"dept_id":  row.Update.DeptID,
					"post_id":  row.Update.PostID,
					"status":   normalizeStatus(row.Update.Status),
				}
				if err := tx.Model(row.Existing).Updates(updates).Error; err != nil {
					return err
				}
				if err := replaceUserRoles(tx, row.Existing.ID, normalizeUint64IDs(row.Update.RoleIDs)); err != nil {
					return err
				}
				result.Updated++
				continue
			}

			passwordHash, err := bcrypt.GenerateFromPassword([]byte(row.Create.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			user := SystemUser{
				Username: row.Create.Username,
				Password: string(passwordHash),
				Nickname: row.Create.Nickname,
				Email:    row.Create.Email,
				Phone:    row.Create.Phone,
				DeptID:   row.Create.DeptID,
				PostID:   row.Create.PostID,
				Status:   normalizeStatus(row.Create.Status),
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			if err := replaceUserRoles(tx, user.ID, normalizeUint64IDs(row.Create.RoleIDs)); err != nil {
				return err
			}
			result.Created++
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func normalizeStatus(status int) int {
	if status == 2 {
		return 2
	}
	return 1
}

func normalizeUserPageQuery(query *UserListQuery) (int, int) {
	page := 1
	pageSize := 10
	if query == nil {
		return page, pageSize
	}
	if query.Page > 0 {
		page = query.Page
	}
	if query.PageSize > 0 {
		pageSize = query.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeUserSort(query *UserListQuery) (string, bool) {
	if query == nil {
		return "id", true
	}

	sortWhitelist := map[string]string{
		"id":         "id",
		"username":   "username",
		"nickname":   "nickname",
		"email":      "email",
		"phone":      "phone",
		"status":     "status",
		"createdAt":  "created_at",
		"created_at": "created_at",
	}

	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		column = "id"
	}

	order := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if order == "asc" {
		return column, false
	}
	return column, true
}

func replaceUserRoles(tx *gorm.DB, userID uint64, roleIDs []uint64) error {
	if err := tx.Exec("DELETE FROM system_user_role WHERE user_id = ?", userID).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if err := tx.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (?, ?)", userID, roleID).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeUint64IDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return []uint64{}
	}
	result := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func (s *UserService) validateUserCreate(req *UserCreateReq) error {
	if len(strings.TrimSpace(req.Password)) < s.getConfiguredPasswordMinLength() {
		return errors.New("user.update.error.password_too_short")
	}
	if err := validateOptionalEmail(req.Email); err != nil {
		return err
	}
	if err := s.ensureDeptID(req.DeptID); err != nil {
		return err
	}
	if err := s.ensurePostForDept(req.DeptID, req.PostID); err != nil {
		return err
	}
	if err := s.ensureUserRoleIDs(req.RoleIDs); err != nil {
		return err
	}

	var count int64
	if err := s.db.Model(&SystemUser{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("user.create.error.username_exists")
	}
	return nil
}

func (s *UserService) validateUserUpdate(user *SystemUser, req *UserUpdateReq) error {
	if err := validateOptionalEmail(req.Email); err != nil {
		return err
	}
	if err := s.ensureDeptID(req.DeptID); err != nil {
		return err
	}
	if err := s.ensurePostForDept(req.DeptID, req.PostID); err != nil {
		return err
	}
	if err := s.ensureUserRoleIDs(req.RoleIDs); err != nil {
		return err
	}

	if user.ID == 1 && req.Status == 2 {
		return errors.New("user.update.error.protected")
	}
	if user.ID == 1 {
		adminRoleID, err := s.getAdminRoleID()
		if err != nil {
			return err
		}
		if adminRoleID > 0 {
			hasAdmin := false
			for _, roleID := range normalizeUint64IDs(req.RoleIDs) {
				if roleID == adminRoleID {
					hasAdmin = true
					break
				}
			}
			if !hasAdmin {
				return errors.New("user.update.error.protected")
			}
		}
	}
	return nil
}

func (s *UserService) ensureUserRoleIDs(roleIDs []uint64) error {
	normalized := normalizeUint64IDs(roleIDs)
	if len(normalized) == 0 {
		return errors.New("user.role.required")
	}

	var count int64
	if err := s.db.Table("system_role").Where("id IN ? AND status = ?", normalized, 1).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(normalized)) {
		return errors.New("user.role.invalid")
	}
	return nil
}

func (s *UserService) getAdminRoleID() (uint64, error) {
	var roleID uint64
	if err := s.db.Table("system_role").Select("id").Where("role_key = ?", "admin").Limit(1).Pluck("id", &roleID).Error; err != nil {
		return 0, err
	}
	return roleID, nil
}

func (s *UserService) getConfiguredPasswordMinLength() int {
	if s.db == nil {
		return defaultConfiguredPasswordMinLength
	}

	var rawValue string
	err := s.db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ?", "security.password_min_length").
		Limit(1).
		Pluck("setting_value", &rawValue).Error
	if err != nil {
		lowerError := strings.ToLower(err.Error())
		if strings.Contains(lowerError, "no such table") || strings.Contains(lowerError, "doesn't exist") {
			return defaultConfiguredPasswordMinLength
		}
		return defaultConfiguredPasswordMinLength
	}

	value, err := strconv.Atoi(strings.TrimSpace(rawValue))
	if err != nil || value <= 0 {
		return defaultConfiguredPasswordMinLength
	}
	return value
}

func (s *UserService) ensureDeptID(deptID uint64) error {
	if deptID == 0 {
		return nil
	}
	var count int64
	if err := s.db.Table("system_dept").Where("id = ?", deptID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user.dept.invalid")
	}
	return nil
}

func (s *UserService) ensurePostForDept(deptID uint64, postID uint64) error {
	if postID == 0 {
		return nil
	}
	if deptID == 0 {
		return errors.New("user.post.dept_required")
	}
	type postRow struct {
		ID     uint64 `gorm:"column:id"`
		DeptID uint64 `gorm:"column:dept_id"`
	}
	var post postRow
	if err := s.db.Table("system_post").Select("id, dept_id").Where("id = ?", postID).First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user.post.invalid")
		}
		return err
	}
	if post.ID == 0 {
		return errors.New("user.post.invalid")
	}
	if post.DeptID > 0 && post.DeptID != deptID {
		return errors.New("user.post.dept_mismatch")
	}
	return nil
}

func (s *UserService) loadUserRoles(userIDs []uint64) (map[uint64][]uint64, map[uint64][]string, map[uint64][]string, error) {
	roleIDMap := make(map[uint64][]uint64, len(userIDs))
	roleKeyMap := make(map[uint64][]string, len(userIDs))
	roleNameMap := make(map[uint64][]string, len(userIDs))
	if len(userIDs) == 0 {
		return roleIDMap, roleKeyMap, roleNameMap, nil
	}

	type userRolePair struct {
		UserID   uint64 `gorm:"column:user_id"`
		RoleID   uint64 `gorm:"column:role_id"`
		RoleKey  string `gorm:"column:role_key"`
		RoleName string `gorm:"column:role_name"`
	}

	var pairs []userRolePair
	if err := s.db.Table("system_user_role").
		Select("system_user_role.user_id, system_user_role.role_id, system_role.role_key, system_role.role_name").
		Joins("JOIN system_role ON system_role.id = system_user_role.role_id").
		Where("system_user_role.user_id IN ?", userIDs).
		Order("system_user_role.role_id asc").
		Scan(&pairs).Error; err != nil {
		return nil, nil, nil, err
	}

	for _, pair := range pairs {
		roleIDMap[pair.UserID] = append(roleIDMap[pair.UserID], pair.RoleID)
		roleKeyMap[pair.UserID] = append(roleKeyMap[pair.UserID], pair.RoleKey)
		roleNameMap[pair.UserID] = append(roleNameMap[pair.UserID], pair.RoleName)
	}
	return roleIDMap, roleKeyMap, roleNameMap, nil
}

func (s *UserService) loadUserLastLoginAt(username string) (*string, error) {
	trimmedUsername := strings.TrimSpace(username)
	if trimmedUsername == "" {
		return nil, nil
	}
	if !s.db.Migrator().HasTable("system_log_login") {
		return nil, nil
	}

	type loginRow struct {
		LoginTime time.Time `gorm:"column:login_time"`
	}

	var lastLogin loginRow
	err := s.db.Table("system_log_login").
		Select("login_time").
		Where("username = ? AND status = ?", trimmedUsername, 1).
		Order("login_time desc, id desc").
		Limit(1).
		Scan(&lastLogin).Error
	if err != nil {
		return nil, err
	}
	if lastLogin.LoginTime.IsZero() {
		return nil, nil
	}
	formatted := lastLogin.LoginTime.Format(time.RFC3339)
	return &formatted, nil
}

func (s *UserService) loadDeptNames(users []SystemUser) (map[uint64]string, error) {
	result := make(map[uint64]string)
	deptIDs := make([]uint64, 0, len(users))
	seen := make(map[uint64]struct{})
	for _, user := range users {
		if user.DeptID == 0 {
			continue
		}
		if _, ok := seen[user.DeptID]; ok {
			continue
		}
		seen[user.DeptID] = struct{}{}
		deptIDs = append(deptIDs, user.DeptID)
	}
	if len(deptIDs) == 0 {
		return result, nil
	}

	type deptNameRow struct {
		ID       uint64 `gorm:"column:id"`
		DeptName string `gorm:"column:dept_name"`
	}
	var rows []deptNameRow
	if err := s.db.Table("system_dept").Select("id, dept_name").Where("id IN ?", deptIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.DeptName
	}
	return result, nil
}

func (s *UserService) loadPostNames(users []SystemUser) (map[uint64]string, error) {
	result := make(map[uint64]string)
	postIDs := make([]uint64, 0, len(users))
	seen := make(map[uint64]struct{})
	for _, user := range users {
		if user.PostID == 0 {
			continue
		}
		if _, ok := seen[user.PostID]; ok {
			continue
		}
		seen[user.PostID] = struct{}{}
		postIDs = append(postIDs, user.PostID)
	}
	if len(postIDs) == 0 {
		return result, nil
	}

	type postNameRow struct {
		ID       uint64 `gorm:"column:id"`
		PostName string `gorm:"column:post_name"`
	}
	var rows []postNameRow
	if err := s.db.Table("system_post").Select("id, post_name").Where("id IN ?", postIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.PostName
	}
	return result, nil
}

func (s *UserService) loadPostCodes(users []SystemUser) (map[uint64]string, error) {
	result := make(map[uint64]string)
	postIDs := make([]uint64, 0, len(users))
	seen := make(map[uint64]struct{})
	for _, user := range users {
		if user.PostID == 0 {
			continue
		}
		if _, ok := seen[user.PostID]; ok {
			continue
		}
		seen[user.PostID] = struct{}{}
		postIDs = append(postIDs, user.PostID)
	}
	if len(postIDs) == 0 {
		return result, nil
	}

	type postCodeRow struct {
		ID       uint64 `gorm:"column:id"`
		PostCode string `gorm:"column:post_code"`
	}
	var rows []postCodeRow
	if err := s.db.Table("system_post").Select("id, post_code").Where("id IN ?", postIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.PostCode
	}
	return result, nil
}

func (s *UserService) loadPostIDsByCode() (map[string]uint64, error) {
	type row struct {
		ID       uint64 `gorm:"column:id"`
		PostCode string `gorm:"column:post_code"`
	}
	var rows []row
	if err := s.db.Table("system_post").Select("id, post_code").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]uint64, len(rows))
	for _, row := range rows {
		result[row.PostCode] = row.ID
	}
	return result, nil
}

func (s *UserService) loadRoleIDsByKey() (map[string]uint64, error) {
	type row struct {
		ID      uint64 `gorm:"column:id"`
		RoleKey string `gorm:"column:role_key"`
		Status  int    `gorm:"column:status"`
	}
	var rows []row
	if err := s.db.Table("system_role").Select("id, role_key, status").Where("deleted_at IS NULL").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]uint64, len(rows))
	for _, row := range rows {
		if row.Status == 1 {
			result[row.RoleKey] = row.ID
		}
	}
	return result, nil
}

func (s *UserService) loadUsersByUsername() (map[string]*SystemUser, error) {
	var rows []SystemUser
	if err := s.db.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]*SystemUser, len(rows))
	for index := range rows {
		result[rows[index].Username] = &rows[index]
	}
	return result, nil
}

func (s *UserService) listUsersForExport(query *UserListQuery) ([]SystemUser, error) {
	var users []SystemUser
	db := s.db.Model(&SystemUser{})
	if query != nil {
		if strings.TrimSpace(query.Username) != "" {
			db = db.Where("username LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Username)))
		}
		if strings.TrimSpace(query.Nickname) != "" {
			db = db.Where("nickname LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Nickname)))
		}
		if query.DeptID > 0 {
			db = db.Where("dept_id = ?", query.DeptID)
		}
		if query.PostID > 0 {
			db = db.Where("post_id = ?", query.PostID)
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	sortColumn, sortDesc := normalizeUserSort(query)
	if err := db.
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: sortColumn},
			Desc:   sortDesc,
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   false,
		}).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (s *UserService) releaseDeletedUsernames() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var deletedUsers []SystemUser
		if err := tx.Unscoped().
			Where("deleted_at IS NOT NULL").
			Find(&deletedUsers).Error; err != nil {
			return err
		}

		for _, user := range deletedUsers {
			if strings.HasPrefix(user.Username, deletedUsernamePrefix) {
				continue
			}
			deletedUsername, err := s.allocateDeletedUsername(tx, user.ID)
			if err != nil {
				return err
			}
			if err := tx.Unscoped().
				Model(&SystemUser{}).
				Where("id = ?", user.ID).
				Update("username", deletedUsername).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *UserService) allocateDeletedUsername(tx *gorm.DB, userID uint64) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate := fmt.Sprintf("%s%d", deletedUsernamePrefix, userID)
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d_%d", deletedUsernamePrefix, userID, time.Now().UnixNano())
		}

		var count int64
		if err := tx.Unscoped().
			Model(&SystemUser{}).
			Where("username = ? AND id <> ?", candidate, userID).
			Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}

	return "", errors.New("user.delete.error.archive_username_conflict")
}

func validateOptionalEmail(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return errors.New("user.email.invalid")
	}
	return nil
}

func mergeUserPermissionKeys(groups ...[]string) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, item := range group {
			key := strings.TrimSpace(item)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, key)
		}
	}
	return result
}

const maxUserProfileExtBytes = 16 * 1024

func marshalUserProfileExt(profileExt map[string]interface{}) (string, error) {
	if profileExt == nil {
		return "", nil
	}
	data, err := json.Marshal(profileExt)
	if err != nil {
		return "", errors.New("user.profile_ext.invalid")
	}
	if len(data) > maxUserProfileExtBytes {
		return "", errors.New("user.profile_ext.too_large")
	}
	return string(data), nil
}

func unmarshalUserProfileExt(raw string) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var profileExt map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &profileExt); err != nil {
		return nil, errors.New("user.profile_ext.invalid")
	}
	if profileExt == nil {
		return map[string]interface{}{}, nil
	}
	return profileExt, nil
}

func (s *UserService) loadUserProfileExt(userID uint64) (map[string]interface{}, error) {
	if userID == 0 || !s.db.Migrator().HasTable(&SystemUserProfileExt{}) {
		return nil, nil
	}
	var ext SystemUserProfileExt
	if err := s.db.First(&ext, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return unmarshalUserProfileExt(ext.ProfileJSON)
}

func upsertUserProfileExt(tx *gorm.DB, userID uint64, profileExtJSON string) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"profile_json", "updated_at"}),
	}).Create(&SystemUserProfileExt{
		UserID:      userID,
		ProfileJSON: profileExtJSON,
	}).Error
}

func formatUserTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
