package iam

import (
	"fmt"
	"strings"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/impexp"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *UserService) ExportUsers(query *UserListQuery, dataScope *common.DataScopeReq) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	users, err := s.listUsersForExport(query, dataScope)
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
		return nil, common.ErrDatabaseNotInitialized
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

func (s *UserService) listUsersForExport(query *UserListQuery, dataScope *common.DataScopeReq) ([]SystemUser, error) {
	var users []SystemUser
	db := s.db.Model(&SystemUser{}).Scopes(database.WithDataScope(dataScope))
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
