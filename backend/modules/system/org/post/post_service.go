package org

import (
	"errors"
	"fmt"
	"pantheon-base/pkg/common"
	"strings"
	"time"

	"pantheon-base/pkg/impexp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const condIDIn = "id IN ?"

type PostService struct {
	db *gorm.DB
}

const deletedPostCodePrefix = "__deleted_post_"

const errPostHasUsers = "post.status.error.has_users"

func NewPostService(db *gorm.DB) *PostService {
	return &PostService{db: db}
}

func (s *PostService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.db.AutoMigrate(&SystemPost{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *PostService) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.releaseDeletedPostCodes()
}

func (s *PostService) ListPosts(query *PostListQuery) (*PostListPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var posts []SystemPost
	db := s.db.Model(&SystemPost{})
	page, pageSize := normalizePostPageQuery(query)
	db = applyPostListFilters(db, query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	sortColumn, sortDesc := normalizePostSort(query)
	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error; err != nil {
		return nil, err
	}

	deptNames, err := s.loadPostDeptNames(posts)
	if err != nil {
		return nil, err
	}
	userCountByPost, err := s.loadPostUserCounts()
	if err != nil {
		return nil, err
	}
	items := make([]PostListResp, 0, len(posts))
	for _, post := range posts {
		items = append(items, toPostListResp(post, deptNames[post.DeptID], userCountByPost[post.ID]))
	}

	return &PostListPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// applyPostListFilters 将查询条件等价应用到 *gorm.DB，保持筛选口径与原 ListPosts 完全一致：
// 关键字同时匹配 post_code/post_name（LIKE），其余字段各自独立 LIKE/等值匹配；nil 查询不附加任何条件。
func applyPostListFilters(db *gorm.DB, query *PostListQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if strings.TrimSpace(query.Keyword) != "" {
		keyword := "%" + common.EscapeLikePattern(strings.TrimSpace(query.Keyword)) + "%"
		db = db.Where("post_code LIKE ? OR post_name LIKE ?", keyword, keyword)
	}
	if strings.TrimSpace(query.PostCode) != "" {
		db = db.Where("post_code LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.PostCode))+"%")
	}
	if strings.TrimSpace(query.PostName) != "" {
		db = db.Where("post_name LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.PostName))+"%")
	}
	if query.DeptID > 0 {
		db = db.Where("dept_id = ?", query.DeptID)
	}
	if query.Status != nil && common.IsEnabledStatus(*query.Status) {
		db = db.Where("status = ?", *query.Status)
	}
	return db
}

func (s *PostService) CreatePost(req *PostCreateReq) (*PostListResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if err := s.validatePostCreate(0, req.PostCode, req.DeptID); err != nil {
		return nil, err
	}

	post := SystemPost{
		DeptID:   req.DeptID,
		PostCode: strings.TrimSpace(req.PostCode),
		PostName: strings.TrimSpace(req.PostName),
		Sort:     req.Sort,
		Status:   normalizePostStatus(req.Status),
		Remark:   strings.TrimSpace(req.Remark),
	}
	if err := s.db.Create(&post).Error; err != nil {
		return nil, err
	}

	deptNames, err := s.loadPostDeptNames([]SystemPost{post})
	if err != nil {
		return nil, err
	}
	resp := toPostListResp(post, deptNames[post.DeptID], 0)
	return &resp, nil
}

func (s *PostService) UpdatePost(postID uint64, req *PostUpdateReq) (*PostListResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var post SystemPost
	if err := s.db.First(&post, postID).Error; err != nil {
		return nil, err
	}
	if err := s.validatePostCreate(postID, req.PostCode, req.DeptID); err != nil {
		return nil, err
	}
	if post.Status != common.StatusDisabled && normalizePostStatus(req.Status) == common.StatusDisabled {
		if err := s.ensurePostsNotAssignedToUsers(s.db, []uint64{postID}); err != nil {
			return nil, err
		}
	}

	post.DeptID = req.DeptID
	post.PostCode = strings.TrimSpace(req.PostCode)
	post.PostName = strings.TrimSpace(req.PostName)
	post.Sort = req.Sort
	post.Status = normalizePostStatus(req.Status)
	post.Remark = strings.TrimSpace(req.Remark)

	if err := s.db.Save(&post).Error; err != nil {
		return nil, err
	}
	deptNames, err := s.loadPostDeptNames([]SystemPost{post})
	if err != nil {
		return nil, err
	}
	resp := toPostListResp(post, deptNames[post.DeptID], 0)
	return &resp, nil
}

func (s *PostService) DeletePost(postID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var post SystemPost
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, postID).Error; err != nil {
			return err
		}
		// 用户占用检查放在事务内，收窄检查与删除之间的并发窗口。
		if err := s.ensurePostsNotAssignedToUsers(tx, []uint64{postID}); err != nil {
			if common.ErrMessage(err) == errPostHasUsers {
				return common.NewInternal("post.delete.error.has_users")
			}
			return err
		}
		deletedCode, err := s.allocateDeletedPostCode(tx, post.ID)
		if err != nil {
			return err
		}
		if err := tx.Model(&post).Update("post_code", deletedCode).Error; err != nil {
			return err
		}
		return tx.Delete(&post).Error
	})
}

func (s *PostService) BatchUpdatePostStatus(postIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	normalizedIDs := normalizePostIDs(postIDs)
	if len(normalizedIDs) == 0 {
		return 0, common.NewBadRequest("post.batch.empty")
	}
	if !common.IsEnabledStatus(status) {
		return 0, common.NewBadRequest("param.invalid")
	}

	var posts []SystemPost
	if err := s.db.Where(condIDIn, normalizedIDs).Find(&posts).Error; err != nil {
		return 0, err
	}
	if len(posts) != len(normalizedIDs) {
		return 0, common.NewNotFound("post.batch.not_found")
	}
	if normalizePostStatus(status) == common.StatusDisabled {
		activeIDs := make([]uint64, 0, len(posts))
		for _, post := range posts {
			if post.Status != common.StatusDisabled {
				activeIDs = append(activeIDs, post.ID)
			}
		}
		if err := s.ensurePostsNotAssignedToUsers(s.db, activeIDs); err != nil {
			return 0, err
		}
	}

	if err := s.db.Model(&SystemPost{}).
		Where(condIDIn, normalizedIDs).
		Updates(map[string]any{
			"status":     normalizePostStatus(status),
			"updated_at": time.Now(),
		}).Error; err != nil {
		return 0, err
	}

	return len(normalizedIDs), nil
}

func (s *PostService) ExportPosts(query *PostListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	rows, err := s.listPostsForExport(query)
	if err != nil {
		return nil, err
	}
	userCountByPost, err := s.loadPostUserCounts()
	if err != nil {
		return nil, err
	}
	deptPathByID, _, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		assignedUserCount := userCountByPost[row.ID]
		governanceTags := buildPostGovernanceTags(row.Status, assignedUserCount)
		governanceBlockedBy := buildPostGovernanceBlockers(assignedUserCount)
		governanceActions := buildPostGovernanceActions(row.Status, assignedUserCount)
		result = append(result, []string{
			deptPathByID[row.DeptID],
			row.PostCode,
			row.PostName,
			fmt.Sprintf("%d", row.Sort),
			fmt.Sprintf("%d", row.Status),
			row.Remark,
			fmt.Sprintf("%d", assignedUserCount),
			"post",
			strings.Join(governanceTags, "|"),
			fmt.Sprintf("%d", impexp.CountGovernanceProblems(governanceTags, map[string]struct{}{"in-use": {}})),
			strings.Join(governanceBlockedBy, "|"),
			strings.Join(governanceActions, "|"),
			impexp.GovernanceScopeLabel("post"),
			impexp.GovernanceTagLabels(governanceTags),
			impexp.GovernanceBlockedByLabels(governanceBlockedBy),
			impexp.GovernanceActionLabels(governanceActions),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-post-export.csv",
		Headers:  append([]string{"deptPath", "postCode", "postName", "sort", "status", "remark", "assignedUserCount"}, impexp.GovernanceExportHeaders...),
		Rows:     result,
	}, nil
}

func (s *PostService) BuildPostImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-post-import-template.csv",
		Headers:  []string{"deptPath", "postCode", "postName", "sort", "status", "remark"},
		Rows: [][]string{
			{"#说明：保留第一行表头；deptPath 使用部门导出的完整路径；postCode 是稳定唯一编码；status 使用 1=启用、2=禁用。", "", "", "", "", ""},
			{"#Pantheon Base/研发中心", "developer", "研发工程师", "10", "1", "负责研发交付"},
		},
	}
}

// postImportRow 表示一次导入解析后的单行数据。
type postImportRow struct {
	DeptID   uint64
	PostCode string
	PostName string
	Sort     int
	Status   int
	Remark   string
}

// appendPostImportDeptError records domain validation errors as i18n keys and
// returns unexpected infrastructure errors to the caller.
func appendPostImportDeptError(result *impexp.ImportResult, rowNumber int, err error) error {
	if !errors.Is(err, common.ErrBadRequest) && !errors.Is(err, common.ErrForbidden) {
		return err
	}
	impexp.AppendImportError(result, rowNumber, "deptPath", common.ErrMessage(err))
	return nil
}

// validateImportHeaders 校验导入表头是否包含必填列，缺失时追加错误到 result。
func validateImportHeaders(headerIndex map[string]int, result *impexp.ImportResult) {
	requiredHeaders := []string{"deptPath", "postCode", "postName", "sort", "status", "remark"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
}

// readImportRow 解析并校验单行导入记录：
//   - 空/注释行（IsCSVRecordEmpty）返回 nil，调用方跳过且不计入 rows；
//   - 校验失败仅追加到 result.Errors，返回的非 nil 行仍应被追加到 rows；
//   - 部门校验与重复码检测的顺序与原循环一致。
//   - 非领域校验错误立即上抛，避免将基础设施错误暴露给导入结果。
func (s *PostService) readImportRow(
	record []string,
	rowNumber int,
	headerIndex map[string]int,
	deptPathToID map[string]uint64,
	seenCodes map[string]int,
	result *impexp.ImportResult,
) (*postImportRow, error) {
	if impexp.IsCSVRecordEmpty(record) {
		return nil, nil
	}

	postCode := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "postCode"))
	postName := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "postName"))
	deptPath := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "deptPath"))
	sortValue, sortErr := impexp.ParseCSVInt(impexp.ReadCSVField(record, headerIndex, "sort"))
	status := impexp.ParseEnabledStatus(impexp.ReadCSVField(record, headerIndex, "status"))
	remark := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "remark"))

	if postCode == "" {
		impexp.AppendImportError(result, rowNumber, "postCode", "post.code.required")
	}
	if postName == "" {
		impexp.AppendImportError(result, rowNumber, "postName", "post.name.required")
	}
	deptID := deptPathToID[deptPath]
	if deptPath == "" {
		impexp.AppendImportError(result, rowNumber, "deptPath", "post.dept.required")
	} else if deptID == 0 {
		impexp.AppendImportError(result, rowNumber, "deptPath", "post.dept.invalid")
	} else if err := s.ensurePostDeptID(deptID); err != nil {
		if err := appendPostImportDeptError(result, rowNumber, err); err != nil {
			return nil, err
		}
	}
	if sortErr != nil {
		impexp.AppendImportError(result, rowNumber, "sort", "import.field.invalid_integer")
	}
	if postCode != "" {
		if firstRow, ok := seenCodes[postCode]; ok {
			impexp.AppendImportError(result, rowNumber, "postCode", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else {
			seenCodes[postCode] = rowNumber
		}
	}

	return &postImportRow{
		DeptID:   deptID,
		PostCode: postCode,
		PostName: postName,
		Sort:     sortValue,
		Status:   status,
		Remark:   remark,
	}, nil
}

// applyImportRows 在事务内对每行执行新增或更新并累加导入统计，与原事务语义完全一致：
// 已存在的 post_code 执行更新（Updated++），否则新增（Created++）；任何写入错误立即回滚事务。
func applyImportRows(
	tx *gorm.DB,
	rows []postImportRow,
	existingByCode map[string]SystemPost,
	result *impexp.ImportResult,
) error {
	for _, row := range rows {
		existing, ok := existingByCode[row.PostCode]
		if ok {
			existing.DeptID = row.DeptID
			existing.PostName = row.PostName
			existing.Sort = row.Sort
			existing.Status = row.Status
			existing.Remark = row.Remark
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			result.Updated++
			continue
		}

		post := SystemPost{
			DeptID:   row.DeptID,
			PostCode: row.PostCode,
			PostName: row.PostName,
			Sort:     row.Sort,
			Status:   row.Status,
			Remark:   row.Remark,
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		result.Created++
	}
	return nil
}

func (s *PostService) ImportPosts(records [][]string) (*impexp.ImportResult, error) {
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
	validateImportHeaders(headerIndex, result)
	if result.Failed > 0 {
		return result, nil
	}

	rows := make([]postImportRow, 0, len(records)-1)
	seenCodes := make(map[string]int, len(records)-1)
	_, deptPathToID, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		row, err := s.readImportRow(records[rowIndex], rowIndex+1, headerIndex, deptPathToID, seenCodes, result)
		if err != nil {
			return nil, err
		}
		if row == nil {
			continue
		}
		rows = append(rows, *row)
	}

	if result.Failed > 0 {
		return result, nil
	}

	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, row.PostCode)
	}

	var existingRows []SystemPost
	if err := s.db.Where("post_code IN ?", codes).Find(&existingRows).Error; err != nil {
		return nil, err
	}
	existingByCode := make(map[string]SystemPost, len(existingRows))
	for _, row := range existingRows {
		existingByCode[row.PostCode] = row
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return applyImportRows(tx, rows, existingByCode, result)
	}); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func (s *PostService) listPostsForExport(query *PostListQuery) ([]SystemPost, error) {
	var posts []SystemPost
	db := s.db.Model(&SystemPost{})
	if query != nil {
		if strings.TrimSpace(query.PostCode) != "" {
			db = db.Where("post_code LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.PostCode))+"%")
		}
		if strings.TrimSpace(query.PostName) != "" {
			db = db.Where("post_name LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.PostName))+"%")
		}
		if query.DeptID > 0 {
			db = db.Where("dept_id = ?", query.DeptID)
		}
		if query.Status != nil && common.IsEnabledStatus(*query.Status) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	sortColumn, sortDesc := normalizePostSort(query)
	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&posts).Error; err != nil {
		return nil, err
	}
	return posts, nil
}

func (s *PostService) validatePostCreate(postID uint64, postCode string, deptID uint64) error {
	trimmedCode := strings.TrimSpace(postCode)
	if trimmedCode == "" {
		return common.NewBadRequest("param.invalid")
	}
	if err := s.ensurePostDeptID(deptID); err != nil {
		return err
	}

	var count int64
	db := s.db.Model(&SystemPost{}).Where("post_code = ?", trimmedCode)
	if postID > 0 {
		db = db.Where("id <> ?", postID)
	}
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return common.NewConflict("post.code.exists")
	}
	return nil
}

func (s *PostService) ensurePostDeptID(deptID uint64) error {
	if deptID == 0 {
		return common.NewBadRequest("post.dept.required")
	}
	type row struct {
		ID     uint64 `gorm:"column:id"`
		IsRoot int    `gorm:"column:is_root"`
	}
	var dept row
	if err := s.db.Table("system_dept").Select("id, is_root").Where("id = ?", deptID).First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NewBadRequest("post.dept.invalid")
		}
		return err
	}
	if dept.IsRoot == common.StatusFlagYes {
		return common.NewForbidden("post.dept.root_forbidden")
	}
	return nil
}

func (s *PostService) loadPostDeptNames(posts []SystemPost) (map[uint64]string, error) {
	result := make(map[uint64]string)
	deptIDs := make([]uint64, 0, len(posts))
	seen := make(map[uint64]struct{}, len(posts))
	for _, post := range posts {
		if post.DeptID == 0 {
			continue
		}
		if _, ok := seen[post.DeptID]; ok {
			continue
		}
		seen[post.DeptID] = struct{}{}
		deptIDs = append(deptIDs, post.DeptID)
	}
	if len(deptIDs) == 0 {
		return result, nil
	}
	type row struct {
		ID       uint64 `gorm:"column:id"`
		DeptName string `gorm:"column:dept_name"`
	}
	var rows []row
	if err := s.db.Table("system_dept").Select("id, dept_name").Where(condIDIn, deptIDs).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ID] = row.DeptName
	}
	return result, nil
}

func toPostListResp(post SystemPost, deptName string, assignedUserCount int) PostListResp {
	governanceTags := buildPostGovernanceTags(post.Status, assignedUserCount)
	governanceBlockedBy := buildPostGovernanceBlockers(assignedUserCount)
	governanceActions := buildPostGovernanceActions(post.Status, assignedUserCount)
	return PostListResp{
		ID:                    post.ID,
		DeptID:                post.DeptID,
		DeptName:              deptName,
		PostCode:              post.PostCode,
		PostName:              post.PostName,
		Sort:                  post.Sort,
		Status:                post.Status,
		Remark:                post.Remark,
		AssignedUserCount:     assignedUserCount,
		GovernanceTags:        governanceTags,
		GovernanceTagLabels:   splitGovernanceLabels(impexp.GovernanceTagLabels(governanceTags)),
		GovernanceBlockedBy:   governanceBlockedBy,
		GovernanceBlockedDesc: splitGovernanceLabels(impexp.GovernanceBlockedByLabels(governanceBlockedBy)),
		GovernanceActions:     governanceActions,
		GovernanceActionLabel: splitGovernanceLabels(impexp.GovernanceActionLabels(governanceActions)),
		CreatedAt:             post.CreatedAt.Format(time.RFC3339),
	}
}

func splitGovernanceLabels(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func normalizePostPageQuery(query *PostListQuery) (int, int) {
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

func normalizePostSort(query *PostListQuery) (string, bool) {
	if query == nil {
		return "id", true
	}

	sortWhitelist := map[string]string{
		"id":        "id",
		"postCode":  "post_code",
		"post_code": "post_code",
		"postName":  "post_name",
		"post_name": "post_name",
		"sort":      "sort",
		"status":    "status",
		"createdAt": "created_at",
	}
	column, ok := sortWhitelist[strings.TrimSpace(query.SortField)]
	if !ok {
		column = "id"
	}
	return column, strings.ToLower(strings.TrimSpace(query.SortOrder)) == "desc"
}

func normalizePostStatus(status int) int {
	return common.NormalizeEnabledStatus(status)
}

func normalizePostIDs(ids []uint64) []uint64 {
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

func (s *PostService) loadPostUserCounts() (map[uint64]int, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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

func buildPostGovernanceTags(status, assignedUserCount int) []string {
	tags := make([]string, 0, 2)
	if assignedUserCount > 0 {
		tags = append(tags, "in-use")
	}
	if normalizePostStatus(status) == common.StatusDisabled {
		tags = append(tags, "disabled")
	}
	if len(tags) == 0 {
		return []string{"clean"}
	}
	return tags
}

func buildPostGovernanceBlockers(assignedUserCount int) []string {
	if assignedUserCount > 0 {
		return []string{"users"}
	}
	return []string{"none"}
}

func buildPostGovernanceActions(status, assignedUserCount int) []string {
	if assignedUserCount > 0 {
		if normalizePostStatus(status) == common.StatusDisabled {
			return []string{"reassign-users", "review-status"}
		}
		return []string{"reassign-users"}
	}
	if normalizePostStatus(status) == common.StatusDisabled {
		return []string{"delete-or-keep-disabled"}
	}
	return []string{"keep-observing"}
}

func (s *PostService) ensurePostsNotAssignedToUsers(db *gorm.DB, postIDs []uint64) error {
	if len(postIDs) == 0 {
		return nil
	}
	var userCount int64
	if err := db.Table("system_user").Where("post_id IN ? AND deleted_at IS NULL", postIDs).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return common.NewInternal(errPostHasUsers)
	}
	return nil
}

func (s *PostService) releaseDeletedPostCodes() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var deletedPosts []SystemPost
		if err := tx.Unscoped().Where("deleted_at IS NOT NULL").Find(&deletedPosts).Error; err != nil {
			return err
		}
		for _, post := range deletedPosts {
			if strings.HasPrefix(post.PostCode, deletedPostCodePrefix) {
				continue
			}
			deletedCode, err := s.allocateDeletedPostCode(tx, post.ID)
			if err != nil {
				return err
			}
			if err := tx.Unscoped().Model(&SystemPost{}).Where("id = ?", post.ID).Update("post_code", deletedCode).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PostService) allocateDeletedPostCode(tx *gorm.DB, postID uint64) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate := fmt.Sprintf("%s%d", deletedPostCodePrefix, postID)
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d_%d", deletedPostCodePrefix, postID, time.Now().UnixNano())
		}

		var count int64
		if err := tx.Unscoped().Model(&SystemPost{}).Where("post_code = ? AND id <> ?", candidate, postID).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", common.NewConflict("post.delete.error.archive_code_conflict")
}
