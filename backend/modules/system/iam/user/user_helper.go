package iam

import (
	"encoding/json"
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
	if len(roleIDs) == 0 {
		return nil
	}
	// Batch insert using a single query
	type userRole struct {
		UserID uint64 `gorm:"column:user_id"`
		RoleID uint64 `gorm:"column:role_id"`
	}
	roles := make([]userRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		roles = append(roles, userRole{UserID: userID, RoleID: roleID})
	}
	return tx.Table("system_user_role").CreateInBatches(roles, 100).Error
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
		return nil
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

func (s *UserService) ensurePostForDept(deptID, postID uint64) error {
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
	for _, userID := range userIDs {
		roleIDMap[userID] = []uint64{}
		roleKeyMap[userID] = []string{}
		roleNameMap[userID] = []string{}
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
