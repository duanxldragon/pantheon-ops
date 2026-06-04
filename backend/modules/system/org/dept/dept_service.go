package org

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"pantheon-platform/backend/pkg/impexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeptService struct {
	db *gorm.DB
}

const defaultRootDeptName = "Pantheon Base"

func NewDeptService(db *gorm.DB) *DeptService {
	return &DeptService{db: db}
}

func (s *DeptService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&SystemDept{}); err != nil {
		return err
	}
	return s.ensureRootDept()
}

func (s *DeptService) GetDeptTree(query *DeptListQuery) ([]*DeptTreeResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var depts []SystemDept
	sortColumn, sortDesc := normalizeDeptSort(query)
	if err := s.db.Model(&SystemDept{}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&depts).Error; err != nil {
		return nil, err
	}

	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	depts = filterDeptTreeNodes(depts, query, postCountByDept)
	return buildDeptTree(depts, 0, postCountByDept), nil
}

func (s *DeptService) GetOverview() (*DeptOverviewResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var depts []SystemDept
	if err := s.db.Order("id asc").Find(&depts).Error; err != nil {
		return nil, err
	}

	type orgPostRow struct {
		DeptID uint64
		Status int
	}
	posts := make([]orgPostRow, 0)
	if s.db.Migrator().HasTable("system_post") {
		if err := s.db.Table("system_post").Select("dept_id, status").Order("id asc").Find(&posts).Error; err != nil {
			return nil, err
		}
	}

	childCountByDept := make(map[uint64]int, len(depts))
	for _, dept := range depts {
		childCountByDept[dept.ParentID]++
	}
	postCountByDept := make(map[uint64]int, len(posts))
	enabledPostCountByDept := make(map[uint64]int, len(posts))
	for _, post := range posts {
		postCountByDept[post.DeptID]++
		if post.Status == 1 {
			enabledPostCountByDept[post.DeptID]++
		}
	}

	resp := &DeptOverviewResp{}
	var rootID uint64
	for _, dept := range depts {
		resp.TotalDeptCount++
		if dept.Status == 1 {
			resp.EnabledDeptCount++
		} else {
			resp.DisabledDeptCount++
		}
		if dept.IsRoot == 1 {
			resp.RootDeptCount++
			rootID = dept.ID
			continue
		}
		if strings.TrimSpace(dept.Leader) == "" {
			resp.LeaderlessDeptCount++
		}
		if postCountByDept[dept.ID] == 0 {
			resp.NoPostDeptCount++
		}
		if childCountByDept[dept.ID] == 0 && postCountByDept[dept.ID] == 0 {
			resp.EmptyDeptCount++
		}
	}

	for _, post := range posts {
		resp.TotalPostCount++
		if post.Status == 1 {
			resp.EnabledPostCount++
		}
	}

	if rootID > 0 {
		resp.DirectChildDeptCount = childCountByDept[rootID]
	}
	resp.HealthIssueCount = resp.LeaderlessDeptCount + resp.NoPostDeptCount + resp.DisabledDeptCount
	return resp, nil
}

func (s *DeptService) ListGovernanceTasks(query *DeptGovernanceTaskQuery) ([]DeptGovernanceTaskResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var depts []SystemDept
	if err := s.db.Order("sort asc").Order("id asc").Find(&depts).Error; err != nil {
		return nil, err
	}
	childCountByDept, err := s.loadDeptChildCounts()
	if err != nil {
		return nil, err
	}
	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	userCountByDept, err := s.loadDeptUserCounts()
	if err != nil {
		return nil, err
	}
	postUserCountByPost, err := s.loadPostUserCounts()
	if err != nil {
		return nil, err
	}
	pathByID, _, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}

	items := make([]DeptGovernanceTaskResp, 0)
	for _, dept := range depts {
		if dept.IsRoot == 1 {
			continue
		}
		deptPath := pathByID[dept.ID]
		governanceTags := buildDeptGovernanceTags(dept, childCountByDept[dept.ID], postCountByDept[dept.ID])
		governanceBlockedBy := buildDeptDeleteBlockers(childCountByDept[dept.ID], postCountByDept[dept.ID], userCountByDept[dept.ID])
		governanceActions := buildDeptGovernanceActions(governanceTags, governanceBlockedBy)
		for index, action := range governanceActions {
			if action == "keep-observing" {
				continue
			}
			task := DeptGovernanceTaskResp{
				TaskKey:               fmt.Sprintf("dept:%d:%s", dept.ID, action),
				GovernanceScope:       "dept",
				GovernanceScopeLabel:  impexp.GovernanceScopeLabel("dept"),
				GovernanceTag:         pickDeptTaskTag(governanceTags, action),
				GovernanceBlockedBy:   pickDeptTaskBlockedBy(governanceBlockedBy, action),
				GovernanceAction:      action,
				GovernanceActionLabel: impexp.GovernanceActionLabels([]string{action}),
				DeptID:                dept.ID,
				DeptName:              dept.DeptName,
				DeptPath:              deptPath,
				RelatedUserCount:      userCountByDept[dept.ID],
				ResourceStatus:        dept.Status,
			}
			if task.GovernanceTag == "" && len(governanceTags) > 0 {
				task.GovernanceTag = governanceTags[min(index, len(governanceTags)-1)]
			}
			if task.GovernanceBlockedBy == "" {
				task.GovernanceBlockedBy = "none"
			}
			task.GovernanceTagLabel = impexp.GovernanceTagLabels([]string{task.GovernanceTag})
			task.GovernanceBlockedByLabel = impexp.GovernanceBlockedByLabels([]string{task.GovernanceBlockedBy})
			if matchGovernanceTaskQuery(task, query) {
				items = append(items, task)
			}
		}
	}

	if s.db.Migrator().HasTable("system_post") {
		type postTaskRow struct {
			ID       uint64
			DeptID   uint64
			PostCode string
			PostName string
			Status   int
		}
		posts := make([]postTaskRow, 0)
		if err := s.db.Table("system_post").Select("id, dept_id, post_code, post_name, status").Order("sort asc").Order("id asc").Scan(&posts).Error; err != nil {
			return nil, err
		}
		for _, post := range posts {
			governanceTags := buildLocalPostGovernanceTags(post.Status, postUserCountByPost[post.ID])
			governanceBlockedBy := buildLocalPostGovernanceBlockers(postUserCountByPost[post.ID])
			governanceActions := buildLocalPostGovernanceActions(post.Status, postUserCountByPost[post.ID])
			for _, action := range governanceActions {
				if action == "keep-observing" {
					continue
				}
				task := DeptGovernanceTaskResp{
					TaskKey:               fmt.Sprintf("post:%d:%s", post.ID, action),
					GovernanceScope:       "post",
					GovernanceScopeLabel:  impexp.GovernanceScopeLabel("post"),
					GovernanceTag:         pickPostTaskTag(governanceTags, action),
					GovernanceBlockedBy:   pickPostTaskBlockedBy(governanceBlockedBy, action),
					GovernanceAction:      action,
					GovernanceActionLabel: impexp.GovernanceActionLabels([]string{action}),
					DeptID:                post.DeptID,
					DeptName:              deptNameByID(depts, post.DeptID),
					DeptPath:              pathByID[post.DeptID],
					PostID:                post.ID,
					PostName:              post.PostName,
					RelatedUserCount:      postUserCountByPost[post.ID],
					ResourceStatus:        post.Status,
				}
				if task.GovernanceTag == "" && len(governanceTags) > 0 {
					task.GovernanceTag = governanceTags[0]
				}
				if task.GovernanceBlockedBy == "" {
					task.GovernanceBlockedBy = "none"
				}
				task.GovernanceTagLabel = impexp.GovernanceTagLabels([]string{task.GovernanceTag})
				task.GovernanceBlockedByLabel = impexp.GovernanceBlockedByLabels([]string{task.GovernanceBlockedBy})
				if matchGovernanceTaskQuery(task, query) {
					items = append(items, task)
				}
			}
		}
	}

	return items, nil
}

func (s *DeptService) ExportGovernanceTasks(query *DeptGovernanceTaskQuery) (*impexp.CSVFile, error) {
	items, err := s.ListGovernanceTasks(query)
	if err != nil {
		return nil, err
	}
	snapshotAt := time.Now().Format(time.RFC3339)
	querySummary := buildGovernanceTaskQuerySummary(query)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			snapshotAt,
			querySummary,
			item.TaskKey,
			item.GovernanceScope,
			item.GovernanceScopeLabel,
			item.GovernanceTag,
			item.GovernanceTagLabel,
			item.GovernanceBlockedBy,
			item.GovernanceBlockedByLabel,
			item.GovernanceAction,
			item.GovernanceActionLabel,
			fmt.Sprintf("%d", item.DeptID),
			item.DeptName,
			item.DeptPath,
			fmt.Sprintf("%d", item.PostID),
			item.PostName,
			fmt.Sprintf("%d", item.RelatedUserCount),
			fmt.Sprintf("%d", item.ResourceStatus),
		})
	}
	return &impexp.CSVFile{
		Filename: "system-org-governance-tasks.csv",
		Headers: []string{
			"governanceSnapshotAt", "governanceQuerySummary", "taskKey",
			"governanceScope", "governanceScopeLabel",
			"governanceTag", "governanceTagLabel",
			"governanceBlockedBy", "governanceBlockedByLabel",
			"governanceAction", "governanceActionLabel",
			"deptId", "deptName", "deptPath",
			"postId", "postName",
			"relatedUserCount", "resourceStatus",
		},
		Rows: rows,
	}, nil
}

func (s *DeptService) ListLeaderCandidates(deptID uint64) ([]DeptLeaderCandidateResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if deptID == 0 {
		return nil, errors.New("dept.not_found")
	}

	var dept SystemDept
	if err := s.db.First(&dept, deptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("dept.not_found")
		}
		return nil, err
	}
	if dept.IsRoot == 1 {
		return []DeptLeaderCandidateResp{}, nil
	}

	type leaderCandidateRow struct {
		UserID   uint64 `gorm:"column:user_id"`
		Username string `gorm:"column:username"`
		Nickname string `gorm:"column:nickname"`
		DeptID   uint64 `gorm:"column:dept_id"`
		DeptName string `gorm:"column:dept_name"`
		PostID   uint64 `gorm:"column:post_id"`
		PostName string `gorm:"column:post_name"`
	}

	rows := make([]leaderCandidateRow, 0)
	if err := s.db.Table("system_user AS u").
		Select("u.id AS user_id, u.username, u.nickname, u.dept_id, d.dept_name, u.post_id, p.post_name").
		Joins("JOIN system_dept AS d ON d.id = u.dept_id").
		Joins("JOIN system_post AS p ON p.id = u.post_id AND p.dept_id = u.dept_id").
		Where("u.deleted_at IS NULL AND u.status = ? AND u.dept_id = ? AND u.post_id > 0", 1, deptID).
		Order("u.nickname asc").
		Order("u.username asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]DeptLeaderCandidateResp, 0, len(rows))
	for _, row := range rows {
		displayName := strings.TrimSpace(row.Nickname)
		if displayName == "" {
			displayName = row.Username
		}
		result = append(result, DeptLeaderCandidateResp{
			UserID:      row.UserID,
			Username:    row.Username,
			Nickname:    row.Nickname,
			DisplayName: displayName,
			DeptID:      row.DeptID,
			DeptName:    row.DeptName,
			PostID:      row.PostID,
			PostName:    row.PostName,
		})
	}
	return result, nil
}

func (s *DeptService) CreateDept(req *DeptCreateReq) (*DeptTreeResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if err := s.validateDeptCreate(req); err != nil {
		return nil, err
	}

	ancestors, err := s.buildAncestors(req.ParentID)
	if err != nil {
		return nil, err
	}

	dept := SystemDept{
		ParentID:     req.ParentID,
		Ancestors:    ancestors,
		DeptName:     strings.TrimSpace(req.DeptName),
		Sort:         req.Sort,
		LeaderUserID: 0,
		Leader:       strings.TrimSpace(req.Leader),
		Phone:        strings.TrimSpace(req.Phone),
		Email:        strings.TrimSpace(req.Email),
		Status:       normalizeSystemStatus(req.Status),
	}
	if err := s.db.Create(&dept).Error; err != nil {
		return nil, err
	}
	return toDeptTreeResp(dept, 0, 0), nil
}

func (s *DeptService) UpdateDept(deptID uint64, req *DeptUpdateReq) (*DeptTreeResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var dept SystemDept
	if err := s.db.First(&dept, deptID).Error; err != nil {
		return nil, err
	}
	if err := s.validateDeptUpdate(&dept, req); err != nil {
		return nil, err
	}
	resolvedLeader, resolvedLeaderUserID, err := s.resolveDeptLeaderFields(dept.ID, req.Leader, req.LeaderUserID)
	if err != nil {
		return nil, err
	}

	ancestors, err := s.buildAncestors(req.ParentID)
	if err != nil {
		return nil, err
	}

	dept.ParentID = req.ParentID
	dept.Ancestors = ancestors
	dept.IsRoot = normalizeDeptRootFlag(dept.IsRoot)
	dept.DeptName = strings.TrimSpace(req.DeptName)
	dept.Sort = req.Sort
	dept.LeaderUserID = resolvedLeaderUserID
	dept.Leader = resolvedLeader
	dept.Phone = strings.TrimSpace(req.Phone)
	dept.Email = strings.TrimSpace(req.Email)
	dept.Status = normalizeSystemStatus(req.Status)

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&dept).Error; err != nil {
			return err
		}
		return s.refreshChildAncestors(tx, dept.ID)
	}); err != nil {
		return nil, err
	}

	return toDeptTreeResp(dept, 0, 0), nil
}

func (s *DeptService) DeleteDept(deptID uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}

	var dept SystemDept
	if err := s.db.First(&dept, deptID).Error; err != nil {
		return err
	}
	if dept.IsRoot == 1 {
		return errors.New("dept.root.delete_forbidden")
	}

	var childCount int64
	if err := s.db.Model(&SystemDept{}).Where("parent_id = ?", deptID).Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return errors.New("dept.delete.error.has_children")
	}

	var postCount int64
	if err := s.db.Table("system_post").Where("dept_id = ? AND deleted_at IS NULL", deptID).Count(&postCount).Error; err != nil {
		return err
	}
	if postCount > 0 {
		return errors.New("dept.delete.error.has_posts")
	}

	var userCount int64
	if err := s.db.Table("system_user").Where("dept_id = ? AND deleted_at IS NULL", deptID).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return errors.New("dept.delete.error.has_users")
	}

	return s.db.Delete(&SystemDept{}, deptID).Error
}

func (s *DeptService) BatchUpdateDeptStatus(deptIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	normalizedIDs := normalizeDeptIDs(deptIDs)
	if len(normalizedIDs) == 0 {
		return 0, errors.New("dept.batch.empty")
	}
	if status != 1 && status != 2 {
		return 0, errors.New("param.invalid")
	}

	var depts []SystemDept
	if err := s.db.Where("id IN ?", normalizedIDs).Find(&depts).Error; err != nil {
		return 0, err
	}
	if len(depts) != len(normalizedIDs) {
		return 0, errors.New("dept.batch.not_found")
	}
	for _, dept := range depts {
		if dept.IsRoot == 1 {
			return 0, errors.New("dept.root.status_fixed")
		}
	}

	if err := s.db.Model(&SystemDept{}).
		Where("id IN ?", normalizedIDs).
		Updates(map[string]any{
			"status":     normalizeSystemStatus(status),
			"updated_at": time.Now(),
		}).Error; err != nil {
		return 0, err
	}

	return len(normalizedIDs), nil
}

func (s *DeptService) BatchUpdateDeptLeader(items []DeptBatchLeaderItem) (int, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	normalizedItems := normalizeDeptLeaderItems(items)
	if len(normalizedItems) == 0 {
		return 0, errors.New("dept.batch.empty")
	}

	deptIDs := make([]uint64, 0, len(normalizedItems))
	deptToLeader := make(map[uint64]DeptBatchLeaderItem, len(normalizedItems))
	for _, item := range normalizedItems {
		deptIDs = append(deptIDs, item.DeptID)
		deptToLeader[item.DeptID] = item
	}
	var depts []SystemDept
	if err := s.db.Where("id IN ?", deptIDs).Find(&depts).Error; err != nil {
		return 0, err
	}
	if len(depts) != len(deptIDs) {
		return 0, errors.New("dept.batch.not_found")
	}
	updates := make([]struct {
		DeptID       uint64
		Leader       string
		LeaderUserID uint64
	}, 0, len(depts))
	for _, dept := range depts {
		if dept.IsRoot == 1 {
			return 0, errors.New("dept.root.update_forbidden")
		}
		item := deptToLeader[dept.ID]
		if item.LeaderUserID == 0 {
			return 0, errors.New("dept.leader.required")
		}
		leader, leaderUserID, err := s.resolveDeptLeaderFields(dept.ID, "", item.LeaderUserID)
		if err != nil {
			return 0, err
		}
		updates = append(updates, struct {
			DeptID       uint64
			Leader       string
			LeaderUserID uint64
		}{
			DeptID:       dept.ID,
			Leader:       leader,
			LeaderUserID: leaderUserID,
		})
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range updates {
			if err := tx.Model(&SystemDept{}).
				Where("id = ?", item.DeptID).
				Updates(map[string]any{
					"leader_user_id": item.LeaderUserID,
					"leader":         item.Leader,
					"updated_at":     time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return 0, err
	}

	return len(normalizedItems), nil
}

func (s *DeptService) ExportDepts(query *DeptListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	depts, err := s.listDeptsForExport(query)
	if err != nil {
		return nil, err
	}
	childCountByDept, err := s.loadDeptChildCounts()
	if err != nil {
		return nil, err
	}
	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	userCountByDept, err := s.loadDeptUserCounts()
	if err != nil {
		return nil, err
	}
	pathByID, _, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(depts))
	for _, dept := range depts {
		if dept.IsRoot == 1 {
			continue
		}
		governanceTags := buildDeptGovernanceTags(dept, childCountByDept[dept.ID], postCountByDept[dept.ID])
		governanceBlockedBy := buildDeptDeleteBlockers(childCountByDept[dept.ID], postCountByDept[dept.ID], userCountByDept[dept.ID])
		governanceActions := buildDeptGovernanceActions(governanceTags, governanceBlockedBy)
		rows = append(rows, []string{
			pathByID[dept.ParentID],
			dept.DeptName,
			fmt.Sprintf("%d", dept.Sort),
			dept.Leader,
			dept.Phone,
			dept.Email,
			fmt.Sprintf("%d", dept.Status),
			pathByID[dept.ID],
			fmt.Sprintf("%d", childCountByDept[dept.ID]),
			fmt.Sprintf("%d", postCountByDept[dept.ID]),
			fmt.Sprintf("%d", userCountByDept[dept.ID]),
			"dept",
			strings.Join(governanceTags, "|"),
			fmt.Sprintf("%d", impexp.CountGovernanceProblems(governanceTags, map[string]struct{}{"leaderless": {}, "no-post": {}, "empty": {}})),
			strings.Join(governanceBlockedBy, "|"),
			strings.Join(governanceActions, "|"),
			impexp.GovernanceScopeLabel("dept"),
			impexp.GovernanceTagLabels(governanceTags),
			impexp.GovernanceBlockedByLabels(governanceBlockedBy),
			impexp.GovernanceActionLabels(governanceActions),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-dept-export.csv",
		Headers:  append([]string{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status", "deptPath", "childDeptCount", "postCount", "userCount"}, impexp.GovernanceExportHeaders...),
		Rows:     rows,
	}, nil
}

func (s *DeptService) BuildDeptImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-dept-import-template.csv",
		Headers:  []string{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status"},
		Rows: [][]string{
			{"#说明：保留第一行表头；parentDeptPath 使用部门导出的完整路径；根节点下创建部门时通常填写 Pantheon Base；status 使用 1=启用、2=禁用。", "", "", "", "", "", ""},
			{"#Pantheon Base", "研发中心", "10", "张三", "13800138000", "rd@example.com", "1"},
		},
	}
}

func (s *DeptService) ImportDepts(records [][]string) (*impexp.ImportResult, error) {
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
	requiredHeaders := []string{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	type importRow struct {
		RowNumber      int
		ParentDeptPath string
		DeptName       string
		Sort           int
		Leader         string
		Phone          string
		Email          string
		Status         int
	}

	rows := make([]importRow, 0, len(records)-1)
	seenPaths := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		parentPath := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "parentDeptPath"))
		deptName := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "deptName"))
		sortValue, sortErr := impexp.ParseCSVInt(impexp.ReadCSVField(record, headerIndex, "sort"))
		email := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "email"))
		if parentPath == "" {
			impexp.AppendImportError(result, rowNumber, "parentDeptPath", "dept.parent.required")
		}
		if deptName == "" {
			impexp.AppendImportError(result, rowNumber, "deptName", "dept.name.required")
		}
		if sortErr != nil {
			impexp.AppendImportError(result, rowNumber, "sort", "import.field.invalid_integer")
		}
		if err := validateDeptOptionalEmail(email); err != nil {
			impexp.AppendImportError(result, rowNumber, "email", err.Error())
		}
		fullPath := parentPath + "/" + deptName
		if firstRow, ok := seenPaths[fullPath]; ok {
			impexp.AppendImportError(result, rowNumber, "deptName", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else {
			seenPaths[fullPath] = rowNumber
		}
		rows = append(rows, importRow{
			RowNumber:      rowNumber,
			ParentDeptPath: parentPath,
			DeptName:       deptName,
			Sort:           sortValue,
			Leader:         strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "leader")),
			Phone:          strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "phone")),
			Email:          email,
			Status:         impexp.ParseEnabledStatus(impexp.ReadCSVField(record, headerIndex, "status")),
		})
	}

	if result.Failed > 0 {
		return result, nil
	}

	_, pathToID, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}
	rollbackValidation := errors.New("dept.import.validation_failed")
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			parentID := pathToID[row.ParentDeptPath]
			if parentID == 0 {
				impexp.AppendImportError(result, row.RowNumber, "parentDeptPath", "dept.parent.not_found")
				return rollbackValidation
			}

			var dept SystemDept
			err := tx.Where("parent_id = ? AND dept_name = ?", parentID, row.DeptName).First(&dept).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				ancestors, buildErr := s.buildAncestorsWithDB(tx, parentID)
				if buildErr != nil {
					return buildErr
				}
				dept = SystemDept{
					ParentID:  parentID,
					Ancestors: ancestors,
					IsRoot:    0,
					DeptName:  row.DeptName,
					Sort:      row.Sort,
					Leader:    row.Leader,
					Phone:     row.Phone,
					Email:     row.Email,
					Status:    normalizeSystemStatus(row.Status),
				}
				if err := tx.Create(&dept).Error; err != nil {
					return err
				}
				result.Created++
			} else {
				if dept.IsRoot == 1 {
					impexp.AppendImportError(result, row.RowNumber, "deptName", "dept.root.update_forbidden")
					return rollbackValidation
				}
				dept.Sort = row.Sort
				dept.Leader = row.Leader
				dept.Phone = row.Phone
				dept.Email = row.Email
				dept.Status = normalizeSystemStatus(row.Status)
				if err := tx.Save(&dept).Error; err != nil {
					return err
				}
				result.Updated++
			}
			pathToID[row.ParentDeptPath+"/"+row.DeptName] = dept.ID
		}
		return nil
	}); err != nil {
		if errors.Is(err, rollbackValidation) {
			return result, nil
		}
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func (s *DeptService) listDeptsForExport(query *DeptListQuery) ([]SystemDept, error) {
	var depts []SystemDept
	sortColumn, sortDesc := normalizeDeptSort(query)
	if err := s.db.Model(&SystemDept{}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&depts).Error; err != nil {
		return nil, err
	}
	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	return filterDeptTreeNodes(depts, query, postCountByDept), nil
}

func (s *DeptService) loadDeptPostCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if !s.db.Migrator().HasTable("system_post") {
		return map[uint64]int{}, nil
	}

	type deptPostCountRow struct {
		DeptID uint64
		Count  int
	}

	rows := make([]deptPostCountRow, 0)
	if err := s.db.Table("system_post").
		Select("dept_id, COUNT(1) AS count").
		Group("dept_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(rows))
	for _, row := range rows {
		result[row.DeptID] = row.Count
	}
	return result, nil
}

func (s *DeptService) loadDeptChildCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var depts []SystemDept
	if err := s.db.Select("id, parent_id").Find(&depts).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(depts))
	for _, dept := range depts {
		result[dept.ParentID]++
	}
	return result, nil
}

func (s *DeptService) loadDeptUserCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if !s.db.Migrator().HasTable("system_user") {
		return map[uint64]int{}, nil
	}

	type deptUserCountRow struct {
		DeptID uint64
		Count  int
	}

	rows := make([]deptUserCountRow, 0)
	if err := s.db.Table("system_user").
		Select("dept_id, COUNT(1) AS count").
		Where("deleted_at IS NULL").
		Group("dept_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(rows))
	for _, row := range rows {
		result[row.DeptID] = row.Count
	}
	return result, nil
}

func (s *DeptService) loadPostUserCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if !s.db.Migrator().HasTable("system_user") {
		return map[uint64]int{}, nil
	}

	type postUserCountRow struct {
		PostID uint64
		Count  int
	}

	rows := make([]postUserCountRow, 0)
	if err := s.db.Table("system_user").
		Select("post_id, COUNT(1) AS count").
		Where("deleted_at IS NULL AND post_id > 0").
		Group("post_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[uint64]int, len(rows))
	for _, row := range rows {
		result[row.PostID] = row.Count
	}
	return result, nil
}

func (s *DeptService) validateDeptCreate(req *DeptCreateReq) error {
	if req.ParentID == 0 {
		return errors.New("dept.parent.required")
	}
	if req.LeaderUserID > 0 {
		return errors.New("dept.leader.bind_after_create")
	}
	if err := validateDeptOptionalEmail(req.Email); err != nil {
		return err
	}
	return s.ensureDeptParentExists(req.ParentID)
}

func (s *DeptService) validateDeptUpdate(dept *SystemDept, req *DeptUpdateReq) error {
	if dept == nil {
		return errors.New("dept.not_found")
	}
	if req.ParentID == dept.ID {
		return errors.New("dept.update.error.parent_self")
	}
	if dept.IsRoot == 1 {
		if req.ParentID != 0 {
			return errors.New("dept.root.parent_fixed")
		}
		if normalizeSystemStatus(req.Status) != 1 {
			return errors.New("dept.root.status_fixed")
		}
	} else if req.ParentID == 0 {
		return errors.New("dept.parent.required")
	}
	if err := validateDeptOptionalEmail(req.Email); err != nil {
		return err
	}
	if err := s.ensureDeptParentExists(req.ParentID); err != nil {
		return err
	}
	if _, _, err := s.resolveDeptLeaderFields(dept.ID, req.Leader, req.LeaderUserID); err != nil {
		return err
	}
	return s.ensureDeptParentNotDescendant(dept.ID, req.ParentID)
}

func (s *DeptService) resolveDeptLeaderFields(deptID uint64, leader string, leaderUserID uint64) (string, uint64, error) {
	if leaderUserID == 0 {
		return strings.TrimSpace(leader), 0, nil
	}
	if deptID == 0 {
		return "", 0, errors.New("dept.leader.bind_after_create")
	}

	type leaderUserRow struct {
		UserID   uint64 `gorm:"column:user_id"`
		Username string `gorm:"column:username"`
		Nickname string `gorm:"column:nickname"`
	}
	var row leaderUserRow
	if err := s.db.Table("system_user AS u").
		Select("u.id AS user_id, u.username, u.nickname").
		Joins("JOIN system_post AS p ON p.id = u.post_id AND p.dept_id = u.dept_id").
		Where("u.deleted_at IS NULL AND u.status = ? AND u.id = ? AND u.dept_id = ? AND u.post_id > 0", 1, leaderUserID, deptID).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", 0, errors.New("dept.leader.user_invalid")
		}
		return "", 0, err
	}

	displayName := strings.TrimSpace(row.Nickname)
	if displayName == "" {
		displayName = row.Username
	}
	return displayName, row.UserID, nil
}

func (s *DeptService) ensureDeptParentExists(parentID uint64) error {
	if parentID == 0 {
		return nil
	}

	var count int64
	if err := s.db.Model(&SystemDept{}).Where("id = ?", parentID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("dept.parent.not_found")
	}
	return nil
}

func (s *DeptService) ensureDeptParentNotDescendant(deptID uint64, parentID uint64) error {
	if parentID == 0 {
		return nil
	}

	var parent SystemDept
	if err := s.db.First(&parent, parentID).Error; err != nil {
		return err
	}
	ancestors := splitAncestors(parent.Ancestors)
	for _, ancestorID := range ancestors {
		if ancestorID == deptID {
			return errors.New("dept.update.error.parent_descendant")
		}
	}
	return nil
}

func (s *DeptService) buildAncestors(parentID uint64) (string, error) {
	return s.buildAncestorsWithDB(s.db, parentID)
}

func (s *DeptService) buildAncestorsWithDB(db *gorm.DB, parentID uint64) (string, error) {
	if parentID == 0 {
		return "", nil
	}

	var parent SystemDept
	if err := db.First(&parent, parentID).Error; err != nil {
		return "", err
	}
	if parent.Ancestors == "" {
		return fmt.Sprintf("%d", parent.ID), nil
	}
	return fmt.Sprintf("%s,%d", parent.Ancestors, parent.ID), nil
}

func (s *DeptService) refreshChildAncestors(tx *gorm.DB, deptID uint64) error {
	var children []SystemDept
	if err := tx.Where("parent_id = ?", deptID).Find(&children).Error; err != nil {
		return err
	}
	if len(children) == 0 {
		return nil
	}

	var parent SystemDept
	if err := tx.First(&parent, deptID).Error; err != nil {
		return err
	}
	for _, child := range children {
		if parent.Ancestors == "" {
			child.Ancestors = fmt.Sprintf("%d", parent.ID)
		} else {
			child.Ancestors = fmt.Sprintf("%s,%d", parent.Ancestors, parent.ID)
		}
		if err := tx.Model(&child).Update("ancestors", child.Ancestors).Error; err != nil {
			return err
		}
		if err := s.refreshChildAncestors(tx, child.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeptService) ensureRootDept() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var root SystemDept
		err := tx.Where("is_root = ?", 1).Order("id asc").First(&root).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			root = SystemDept{
				ParentID:  0,
				Ancestors: "",
				IsRoot:    1,
				DeptName:  defaultRootDeptName,
				Sort:      0,
				Status:    1,
			}
			if err := tx.Create(&root).Error; err != nil {
				return err
			}
		} else {
			root.ParentID = 0
			root.Ancestors = ""
			root.IsRoot = 1
			root.Status = 1
			if err := tx.Save(&root).Error; err != nil {
				return err
			}
		}

		var topLevelDepts []SystemDept
		if err := tx.Where("parent_id = ? AND id <> ?", 0, root.ID).Find(&topLevelDepts).Error; err != nil {
			return err
		}
		for _, dept := range topLevelDepts {
			dept.ParentID = root.ID
			dept.Ancestors = fmt.Sprintf("%d", root.ID)
			dept.IsRoot = 0
			if err := tx.Save(&dept).Error; err != nil {
				return err
			}
			if err := s.refreshChildAncestors(tx, dept.ID); err != nil {
				return err
			}
		}

		return tx.Model(&SystemDept{}).
			Where("id <> ? AND is_root = ?", root.ID, 1).
			Update("is_root", 0).Error
	})
}

func normalizeDeptSort(query *DeptListQuery) (string, bool) {
	if query == nil {
		return "sort", false
	}

	sortWhitelist := map[string]string{
		"id":        "id",
		"deptName":  "dept_name",
		"dept_name": "dept_name",
		"sort":      "sort",
		"leader":    "leader",
		"status":    "status",
	}
	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		column = "sort"
	}
	return column, strings.ToLower(strings.TrimSpace(query.SortOrder)) == "desc"
}

func filterDeptTreeNodes(depts []SystemDept, query *DeptListQuery, postCountByDept map[uint64]int) []SystemDept {
	if query == nil {
		return depts
	}

	nameFilter := strings.TrimSpace(query.DeptName)
	governanceFilter := strings.TrimSpace(query.Governance)
	statusFilterEnabled := query.Status != nil && (*query.Status == 1 || *query.Status == 2)
	if nameFilter == "" && !statusFilterEnabled && governanceFilter == "" {
		return depts
	}

	byID := make(map[uint64]SystemDept, len(depts))
	childCountByDept := make(map[uint64]int, len(depts))
	included := make(map[uint64]struct{}, len(depts))
	for _, dept := range depts {
		byID[dept.ID] = dept
		childCountByDept[dept.ParentID]++
	}
	for _, dept := range depts {
		if matchesDeptQuery(dept, nameFilter, query.Status, governanceFilter, childCountByDept, postCountByDept) {
			included[dept.ID] = struct{}{}
			for _, ancestorID := range splitAncestors(dept.Ancestors) {
				if _, ok := byID[ancestorID]; ok {
					included[ancestorID] = struct{}{}
				}
			}
		}
	}

	filtered := make([]SystemDept, 0, len(included))
	for _, dept := range depts {
		if _, ok := included[dept.ID]; ok {
			filtered = append(filtered, dept)
		}
	}
	return filtered
}

func matchesDeptQuery(
	dept SystemDept,
	nameFilter string,
	status *int,
	governance string,
	childCountByDept map[uint64]int,
	postCountByDept map[uint64]int,
) bool {
	if nameFilter != "" && !strings.Contains(strings.ToLower(dept.DeptName), strings.ToLower(nameFilter)) {
		return false
	}
	if status != nil && (*status == 1 || *status == 2) && dept.Status != *status {
		return false
	}
	switch strings.ToLower(governance) {
	case "", "all":
	case "leaderless":
		return dept.IsRoot != 1 && strings.TrimSpace(dept.Leader) == ""
	case "no-post":
		return dept.IsRoot != 1 && postCountByDept[dept.ID] == 0
	case "empty":
		return dept.IsRoot != 1 && childCountByDept[dept.ID] == 0 && postCountByDept[dept.ID] == 0
	default:
		return false
	}
	return true
}

func buildDeptTree(depts []SystemDept, parentID uint64, postCountByDept map[uint64]int) []*DeptTreeResp {
	childCountByDept := make(map[uint64]int, len(depts))
	for _, dept := range depts {
		childCountByDept[dept.ParentID]++
	}
	if postCountByDept == nil {
		postCountByDept = make(map[uint64]int)
	}
	tree := make([]*DeptTreeResp, 0)
	for _, dept := range depts {
		if dept.ParentID != parentID {
			continue
		}
		node := toDeptTreeResp(dept, childCountByDept[dept.ID], postCountByDept[dept.ID])
		node.Children = buildDeptTree(depts, dept.ID, postCountByDept)
		tree = append(tree, node)
	}
	return tree
}

func toDeptTreeResp(dept SystemDept, childDeptCount int, postCount int) *DeptTreeResp {
	isLeaderless := dept.IsRoot != 1 && strings.TrimSpace(dept.Leader) == ""
	isNoPost := dept.IsRoot != 1 && postCount == 0
	isEmpty := dept.IsRoot != 1 && childDeptCount == 0 && postCount == 0
	return &DeptTreeResp{
		ID:             dept.ID,
		ParentID:       dept.ParentID,
		Ancestors:      dept.Ancestors,
		IsRoot:         dept.IsRoot == 1,
		DeptName:       dept.DeptName,
		Sort:           dept.Sort,
		LeaderUserID:   dept.LeaderUserID,
		Leader:         dept.Leader,
		Phone:          dept.Phone,
		Email:          dept.Email,
		Status:         dept.Status,
		ChildDeptCount: childDeptCount,
		PostCount:      postCount,
		IsLeaderless:   isLeaderless,
		IsNoPost:       isNoPost,
		IsEmpty:        isEmpty,
	}
}

func buildDeptGovernanceTags(dept SystemDept, childDeptCount int, postCount int) []string {
	if dept.IsRoot == 1 {
		return []string{"root"}
	}

	tags := make([]string, 0, 3)
	if strings.TrimSpace(dept.Leader) == "" {
		tags = append(tags, "leaderless")
	}
	if postCount == 0 {
		tags = append(tags, "no-post")
	}
	if childDeptCount == 0 && postCount == 0 {
		tags = append(tags, "empty")
	}
	if len(tags) == 0 {
		tags = append(tags, "clean")
	}
	return tags
}

func buildDeptDeleteBlockers(childDeptCount int, postCount int, userCount int) []string {
	blockers := make([]string, 0, 3)
	if childDeptCount > 0 {
		blockers = append(blockers, "children")
	}
	if postCount > 0 {
		blockers = append(blockers, "posts")
	}
	if userCount > 0 {
		blockers = append(blockers, "users")
	}
	if len(blockers) == 0 {
		return []string{"none"}
	}
	return blockers
}

func buildDeptGovernanceActions(tags []string, deleteBlockedBy []string) []string {
	actions := make([]string, 0, 6)
	seen := make(map[string]struct{}, 6)
	appendAction := func(value string) {
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		actions = append(actions, value)
	}

	for _, tag := range tags {
		switch tag {
		case "leaderless":
			appendAction("assign-leader")
		case "no-post":
			appendAction("create-post")
		case "empty":
			appendAction("review-merge-or-delete")
		}
	}
	for _, blocker := range deleteBlockedBy {
		switch blocker {
		case "children":
			appendAction("clear-child-depts")
		case "posts":
			appendAction("clear-posts")
		case "users":
			appendAction("clear-users")
		}
	}
	if len(actions) == 0 {
		return []string{"keep-observing"}
	}
	return actions
}

func buildLocalPostGovernanceTags(status int, assignedUserCount int) []string {
	tags := make([]string, 0, 2)
	if assignedUserCount > 0 {
		tags = append(tags, "in-use")
	}
	if normalizeSystemStatus(status) == 2 {
		tags = append(tags, "disabled")
	}
	if len(tags) == 0 {
		return []string{"clean"}
	}
	return tags
}

func buildLocalPostGovernanceBlockers(assignedUserCount int) []string {
	if assignedUserCount > 0 {
		return []string{"users"}
	}
	return []string{"none"}
}

func buildLocalPostGovernanceActions(status int, assignedUserCount int) []string {
	if assignedUserCount > 0 {
		if normalizeSystemStatus(status) == 2 {
			return []string{"reassign-users", "review-status"}
		}
		return []string{"reassign-users"}
	}
	if normalizeSystemStatus(status) == 2 {
		return []string{"delete-or-keep-disabled"}
	}
	return []string{"keep-observing"}
}

func pickDeptTaskTag(tags []string, action string) string {
	switch action {
	case "assign-leader":
		return "leaderless"
	case "create-post":
		return "no-post"
	case "review-merge-or-delete":
		return "empty"
	case "clear-child-depts", "clear-posts", "clear-users":
		return "clean"
	default:
		if len(tags) > 0 {
			return tags[0]
		}
		return ""
	}
}

func pickDeptTaskBlockedBy(blockedBy []string, action string) string {
	switch action {
	case "clear-child-depts":
		return "children"
	case "clear-posts":
		return "posts"
	case "clear-users":
		return "users"
	default:
		if len(blockedBy) > 0 {
			return blockedBy[0]
		}
		return "none"
	}
}

func pickPostTaskTag(tags []string, action string) string {
	switch action {
	case "reassign-users":
		return "in-use"
	case "review-status", "delete-or-keep-disabled":
		return "disabled"
	default:
		if len(tags) > 0 {
			return tags[0]
		}
		return ""
	}
}

func pickPostTaskBlockedBy(blockedBy []string, action string) string {
	switch action {
	case "reassign-users":
		return "users"
	default:
		if len(blockedBy) > 0 {
			return blockedBy[0]
		}
		return "none"
	}
}

func matchGovernanceTaskQuery(task DeptGovernanceTaskResp, query *DeptGovernanceTaskQuery) bool {
	if query == nil {
		return true
	}
	scope := strings.ToLower(strings.TrimSpace(query.Scope))
	if scope != "" && scope != "all" && task.GovernanceScope != scope {
		return false
	}
	governance := strings.ToLower(strings.TrimSpace(query.Governance))
	if governance != "" && task.GovernanceTag != governance {
		return false
	}
	blockedBy := strings.ToLower(strings.TrimSpace(query.BlockedBy))
	if blockedBy != "" && task.GovernanceBlockedBy != blockedBy {
		return false
	}
	action := strings.ToLower(strings.TrimSpace(query.Action))
	if action != "" && task.GovernanceAction != action {
		return false
	}
	keyword := strings.ToLower(strings.TrimSpace(query.Keyword))
	if keyword == "" {
		return true
	}
	return strings.Contains(strings.ToLower(task.DeptName), keyword) ||
		strings.Contains(strings.ToLower(task.DeptPath), keyword) ||
		strings.Contains(strings.ToLower(task.PostName), keyword)
}

func buildGovernanceTaskQuerySummary(query *DeptGovernanceTaskQuery) string {
	if query == nil {
		return "scope=all"
	}
	parts := []string{}
	if strings.TrimSpace(query.Scope) != "" {
		parts = append(parts, "scope="+strings.TrimSpace(query.Scope))
	}
	if strings.TrimSpace(query.Keyword) != "" {
		parts = append(parts, "keyword="+strings.TrimSpace(query.Keyword))
	}
	if strings.TrimSpace(query.Governance) != "" {
		parts = append(parts, "governance="+strings.TrimSpace(query.Governance))
	}
	if strings.TrimSpace(query.BlockedBy) != "" {
		parts = append(parts, "blockedBy="+strings.TrimSpace(query.BlockedBy))
	}
	if strings.TrimSpace(query.Action) != "" {
		parts = append(parts, "action="+strings.TrimSpace(query.Action))
	}
	if len(parts) == 0 {
		return "scope=all"
	}
	return strings.Join(parts, "; ")
}

func deptNameByID(depts []SystemDept, deptID uint64) string {
	for _, dept := range depts {
		if dept.ID == deptID {
			return dept.DeptName
		}
	}
	return ""
}

func splitAncestors(value string) []uint64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]uint64, 0, len(parts))
	for _, part := range parts {
		var id uint64
		_, _ = fmt.Sscanf(strings.TrimSpace(part), "%d", &id)
		if id > 0 {
			result = append(result, id)
		}
	}
	return result
}

func normalizeSystemStatus(status int) int {
	if status == 2 {
		return 2
	}
	return 1
}

func normalizeDeptIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
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

func normalizeDeptLeaderItems(items []DeptBatchLeaderItem) []DeptBatchLeaderItem {
	seen := make(map[uint64]struct{}, len(items))
	result := make([]DeptBatchLeaderItem, 0, len(items))
	for _, item := range items {
		if item.DeptID == 0 {
			continue
		}
		if _, ok := seen[item.DeptID]; ok {
			continue
		}
		seen[item.DeptID] = struct{}{}
		result = append(result, item)
	}
	return result
}

func normalizeDeptRootFlag(value int) int {
	if value == 1 {
		return 1
	}
	return 0
}

func validateDeptOptionalEmail(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return errors.New("dept.email.invalid")
	}
	return nil
}
