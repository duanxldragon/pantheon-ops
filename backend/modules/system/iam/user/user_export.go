package iam

import (
	"context"
	"fmt"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common/security"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxUserExportRows 对齐日志导出的行数上限，防止大库全表导出占满内存并
// 长期占用连接（var 便于测试降低阈值）。
var maxUserExportRows = 10000

// ExportUsers 导出用户 CSV（受 maxUserExportRows 上限与请求上下文取消约束）。
func (s *UserService) ExportUsers(ctx context.Context, query *UserListQuery, dataScope *common.DataScopeReq) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	users, err := s.listUsersForExport(ctx, query, dataScope)
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

// importRow represents a single parsed CSV row that will be applied to the
// database, either as a create or an update operation.
type importRow struct {
	Username string
	Password string
	Create   *UserCreateReq
	Update   *UserUpdateReq
	Existing *SystemUser
}

// userImportLookups holds the pre-loaded reference data required to resolve
// CSV field values into concrete database identifiers.
type userImportLookups struct {
	deptPathToID       map[string]uint64
	postIDByCode       map[string]uint64
	roleIDByKey        map[string]uint64
	existingByUsername map[string]*SystemUser
}

func (s *UserService) ImportUsers(records [][]string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{
		Applied: false,
		Errors:  []impexp.ImportError{},
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}

	headerIndex := s.buildUserImportHeaderIndex(records, result)
	if result.Failed > 0 {
		return result, nil
	}

	lookups, err := s.loadUserImportLookups()
	if err != nil {
		return nil, err
	}

	rows := s.parseUserImportRows(records, headerIndex, lookups, result)
	if result.Failed > 0 {
		return result, nil
	}

	if err := s.applyUserImportRows(rows, result); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

// buildUserImportHeaderIndex maps each header name to its column index and
// records an error for any required header that is missing from the file.
func (s *UserService) buildUserImportHeaderIndex(records [][]string, result *impexp.ImportResult) map[string]int {
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
	return headerIndex
}

// loadUserImportLookups pre-loads the reference data needed to resolve CSV
// values into database identifiers.
func (s *UserService) loadUserImportLookups() (*userImportLookups, error) {
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
	return &userImportLookups{
		deptPathToID:       deptPathToID,
		postIDByCode:       postIDByCode,
		roleIDByKey:        roleIDByKey,
		existingByUsername: existingByUsername,
	}, nil
}

// parseUserImportRows iterates over the data records and builds the list of
// import rows, appending validation errors to the result as needed.
func (s *UserService) parseUserImportRows(records [][]string, headerIndex map[string]int, lookups *userImportLookups, result *impexp.ImportResult) []importRow {
	rows := make([]importRow, 0, len(records)-1)
	seenUsernames := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		row := s.buildUserImportRow(record, rowNumber, headerIndex, lookups, seenUsernames, result)
		rows = append(rows, row)
	}
	return rows
}

// buildUserImportRow parses a single data record into an importRow, performing
// all field-level validations and appending errors to the result.
func (s *UserService) buildUserImportRow(record []string, rowNumber int, headerIndex map[string]int, lookups *userImportLookups, seenUsernames map[string]int, result *impexp.ImportResult) importRow {
	username := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "username"))
	password := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "password"))
	nickname := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "nickname"))
	email := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "email"))
	phone := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "phone"))
	deptPath := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "deptPath"))
	postCode := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "postCode"))
	roleKeys := impexp.SplitPipeValues(impexp.ReadCSVField(record, headerIndex, "roleKeys"))

	s.validateUserImportIdentity(username, rowNumber, seenUsernames, result)
	if err := validateOptionalEmail(email); err != nil {
		impexp.AppendImportError(result, rowNumber, "email", err.Error())
	}

	deptID := s.resolveUserImportDeptID(deptPath, lookups.deptPathToID, rowNumber, result)
	postID := s.resolveUserImportPostID(postCode, lookups.postIDByCode, rowNumber, result)
	roleIDs := s.resolveUserImportRoleIDs(roleKeys, lookups.roleIDByKey, rowNumber, result)

	status := impexp.ParseEnabledStatus(impexp.ReadCSVField(record, headerIndex, "status"))
	existing := lookups.existingByUsername[username]
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
		return importRow{
			Username: username,
			Password: password,
			Update:   updateReq,
			Existing: existing,
		}
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
	return importRow{
		Username: username,
		Password: password,
		Create:   createReq,
	}
}

// validateUserImportIdentity checks that the username is present and not
// duplicated within the file, recording errors on failure.
func (s *UserService) validateUserImportIdentity(username string, rowNumber int, seenUsernames map[string]int, result *impexp.ImportResult) {
	if username == "" {
		impexp.AppendImportError(result, rowNumber, "username", "user.username.required")
	}
	if firstRow, ok := seenUsernames[username]; ok && username != "" {
		impexp.AppendImportError(result, rowNumber, "username", fmt.Sprintf("import.duplicate.row.%d", firstRow))
	} else if username != "" {
		seenUsernames[username] = rowNumber
	}
}

// resolveUserImportDeptID resolves the department path to its ID, recording an
// error when the path is provided but not found.
func (s *UserService) resolveUserImportDeptID(deptPath string, deptPathToID map[string]uint64, rowNumber int, result *impexp.ImportResult) uint64 {
	if deptPath == "" {
		return 0
	}
	deptID := deptPathToID[deptPath]
	if deptID == 0 {
		impexp.AppendImportError(result, rowNumber, "deptPath", "user.dept.invalid")
	}
	return deptID
}

// resolveUserImportPostID resolves the post code to its ID, recording an error
// when the code is provided but not found.
func (s *UserService) resolveUserImportPostID(postCode string, postIDByCode map[string]uint64, rowNumber int, result *impexp.ImportResult) uint64 {
	if postCode == "" {
		return 0
	}
	postID := postIDByCode[postCode]
	if postID == 0 {
		impexp.AppendImportError(result, rowNumber, "postCode", "user.post.invalid")
	}
	return postID
}

// resolveUserImportRoleIDs resolves the list of role keys to role IDs,
// recording an error for every unknown role key.
func (s *UserService) resolveUserImportRoleIDs(roleKeys []string, roleIDByKey map[string]uint64, rowNumber int, result *impexp.ImportResult) []uint64 {
	roleIDs := make([]uint64, 0, len(roleKeys))
	for _, roleKey := range roleKeys {
		roleID := roleIDByKey[roleKey]
		if roleID == 0 {
			impexp.AppendImportError(result, rowNumber, "roleKeys", "user.role.invalid")
			continue
		}
		roleIDs = append(roleIDs, roleID)
	}
	return roleIDs
}

// applyUserImportRows applies all parsed rows inside a single database
// transaction.
func (s *UserService) applyUserImportRows(rows []importRow, result *impexp.ImportResult) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.applyUserImportRowsInTx(tx, rows, result)
	})
}

// applyUserImportRowsInTx iterates over the parsed rows and applies each one.
func (s *UserService) applyUserImportRowsInTx(tx *gorm.DB, rows []importRow, result *impexp.ImportResult) error {
	for _, row := range rows {
		if err := s.applyUserImportRow(tx, row, result); err != nil {
			return err
		}
	}
	return nil
}

// applyUserImportRow applies a single parsed row as an update or a create.
func (s *UserService) applyUserImportRow(tx *gorm.DB, row importRow, result *impexp.ImportResult) error {
	if row.Update != nil && row.Existing != nil {
		return s.applyUserImportUpdate(tx, row, result)
	}
	return s.applyUserImportCreate(tx, row, result)
}

// applyUserImportUpdate persists the update for an existing user.
func (s *UserService) applyUserImportUpdate(tx *gorm.DB, row importRow, result *impexp.ImportResult) error {
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
	return nil
}

// applyUserImportCreate persists the creation of a new user.
func (s *UserService) applyUserImportCreate(tx *gorm.DB, row importRow, result *impexp.ImportResult) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(row.Create.Password), security.GetBcryptCost())
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
	return nil
}

func (s *UserService) listUsersForExport(ctx context.Context, query *UserListQuery, dataScope *common.DataScopeReq) ([]SystemUser, error) {
	var users []SystemUser
	db := s.db.WithContext(ctx).Model(&SystemUser{}).Scopes(database.WithDataScope(dataScope))
	db = applyUserListFilters(db, query)

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
		Limit(maxUserExportRows).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// applyUserListFilters applies common filter conditions for user queries.
// Shared by ListUsers (paginated) and listUsersForExport (full dump).
func applyUserListFilters(db *gorm.DB, query *UserListQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if strings.TrimSpace(query.Keyword) != "" {
		keyword := fmt.Sprintf("%%%s%%", common.EscapeLikePattern(strings.TrimSpace(query.Keyword)))
		db = db.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?", keyword, keyword, keyword)
	}
	if strings.TrimSpace(query.Username) != "" {
		db = db.Where("username LIKE ?", fmt.Sprintf("%%%s%%", common.EscapeLikePattern(strings.TrimSpace(query.Username))))
	}
	if strings.TrimSpace(query.Nickname) != "" {
		db = db.Where("nickname LIKE ?", fmt.Sprintf("%%%s%%", common.EscapeLikePattern(strings.TrimSpace(query.Nickname))))
	}
	if query.DeptID > 0 {
		db = db.Where("dept_id = ?", query.DeptID)
	}
	if query.PostID > 0 {
		db = db.Where("post_id = ?", query.PostID)
	}
	if query.Status != nil && common.IsEnabledStatus(*query.Status) {
		db = db.Where("status = ?", *query.Status)
	}
	return db
}
