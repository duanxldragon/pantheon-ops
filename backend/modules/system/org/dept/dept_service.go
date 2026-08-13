package org

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-base/pkg/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const condIDIn = "id IN ?"

type DeptService struct {
	db *gorm.DB
}

type deptLeaderUpdate struct {
	deptID       uint64
	leader       string
	leaderUserID uint64
}

const defaultRootDeptName = "Pantheon Base"

func NewDeptService(db *gorm.DB) *DeptService {
	return &DeptService{db: db}
}

// Migrate runs database migration
func (s *DeptService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.db.AutoMigrate(&SystemDept{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

// Bootstrap initializes default data
func (s *DeptService) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.ensureRootDept()
}

// CreateDept creates a new department
func (s *DeptService) CreateDept(req *DeptCreateReq) (*DeptTreeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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

// UpdateDept updates an existing department
func (s *DeptService) UpdateDept(deptID uint64, req *DeptUpdateReq) (*DeptTreeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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

// DeleteDept deletes a department
func (s *DeptService) DeleteDept(deptID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	// 检查与删除同事务并对部门行加锁，避免检查通过后、删除前
	// 出现新增子部门/岗位/用户的并发窗口。
	return s.db.Transaction(func(tx *gorm.DB) error {
		return deleteDeptInTransaction(tx, deptID)
	})
}

func deleteDeptInTransaction(tx *gorm.DB, deptID uint64) error {
	var dept SystemDept
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&dept, deptID).Error; err != nil {
		return err
	}
	if dept.IsRoot == common.StatusFlagYes {
		return common.NewForbidden("dept.root.delete_forbidden")
	}
	if err := ensureDeptHasNoChildren(tx, deptID); err != nil {
		return err
	}
	if err := ensureDeptHasNoPosts(tx, deptID); err != nil {
		return err
	}
	if err := ensureDeptHasNoUsers(tx, deptID); err != nil {
		return err
	}
	return tx.Delete(&SystemDept{}, deptID).Error
}

func ensureDeptHasNoChildren(tx *gorm.DB, deptID uint64) error {
	return ensureDeptReferenceCountZero(
		tx.Model(&SystemDept{}).Where("parent_id = ?", deptID),
		"dept.delete.error.has_children",
	)
}

func ensureDeptHasNoPosts(tx *gorm.DB, deptID uint64) error {
	return ensureDeptReferenceCountZero(
		tx.Table("system_post").Where("dept_id = ? AND deleted_at IS NULL", deptID),
		"dept.delete.error.has_posts",
	)
}

func ensureDeptHasNoUsers(tx *gorm.DB, deptID uint64) error {
	return ensureDeptReferenceCountZero(
		tx.Table("system_user").Where("dept_id = ? AND deleted_at IS NULL", deptID),
		"dept.delete.error.has_users",
	)
}

func ensureDeptReferenceCountZero(query *gorm.DB, errorKey string) error {
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return common.NewInternal(errorKey)
	}
	return nil
}

// BatchUpdateDeptStatus updates multiple departments status
func (s *DeptService) BatchUpdateDeptStatus(deptIDs []uint64, status int) (int, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	normalizedIDs := normalizeDeptIDs(deptIDs)
	if len(normalizedIDs) == 0 {
		return 0, common.NewBadRequest("dept.batch.empty")
	}
	if !common.IsEnabledStatus(status) {
		return 0, common.NewBadRequest("param.invalid")
	}

	var depts []SystemDept
	if err := s.db.Where(condIDIn, normalizedIDs).Find(&depts).Error; err != nil {
		return 0, err
	}
	if len(depts) != len(normalizedIDs) {
		return 0, common.NewNotFound("dept.batch.not_found")
	}
	for _, dept := range depts {
		if dept.IsRoot == common.StatusFlagYes {
			return 0, common.NewBadRequest("dept.root.status_fixed")
		}
	}

	if err := s.db.Model(&SystemDept{}).
		Where(condIDIn, normalizedIDs).
		Updates(map[string]any{
			"status":     normalizeSystemStatus(status),
			"updated_at": time.Now(),
		}).Error; err != nil {
		return 0, err
	}

	return len(normalizedIDs), nil
}

// BatchUpdateDeptLeader updates multiple departments leader
func (s *DeptService) BatchUpdateDeptLeader(items []DeptBatchLeaderItem) (int, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	normalizedItems := normalizeDeptLeaderItems(items)
	if len(normalizedItems) == 0 {
		return 0, common.NewBadRequest("dept.batch.empty")
	}

	updates, err := s.prepareDeptLeaderUpdates(normalizedItems)
	if err != nil {
		return 0, err
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return applyDeptLeaderUpdates(tx, updates)
	}); err != nil {
		return 0, err
	}

	return len(normalizedItems), nil
}

func (s *DeptService) prepareDeptLeaderUpdates(items []DeptBatchLeaderItem) ([]deptLeaderUpdate, error) {
	deptIDs := make([]uint64, 0, len(items))
	deptToLeader := make(map[uint64]DeptBatchLeaderItem, len(items))
	for _, item := range items {
		deptIDs = append(deptIDs, item.DeptID)
		deptToLeader[item.DeptID] = item
	}
	var depts []SystemDept
	if err := s.db.Where(condIDIn, deptIDs).Find(&depts).Error; err != nil {
		return nil, err
	}
	if len(depts) != len(deptIDs) {
		return nil, common.NewNotFound("dept.batch.not_found")
	}
	updates := make([]deptLeaderUpdate, 0, len(depts))
	for _, dept := range depts {
		update, err := s.resolveDeptLeaderUpdate(dept, deptToLeader[dept.ID])
		if err != nil {
			return nil, err
		}
		updates = append(updates, update)
	}
	return updates, nil
}

func (s *DeptService) resolveDeptLeaderUpdate(dept SystemDept, item DeptBatchLeaderItem) (deptLeaderUpdate, error) {
	if dept.IsRoot == common.StatusFlagYes {
		return deptLeaderUpdate{}, common.NewForbidden("dept.root.update_forbidden")
	}
	if item.LeaderUserID == 0 {
		return deptLeaderUpdate{}, common.NewBadRequest("dept.leader.required")
	}
	leader, leaderUserID, err := s.resolveDeptLeaderFields(dept.ID, "", item.LeaderUserID)
	if err != nil {
		return deptLeaderUpdate{}, err
	}
	return deptLeaderUpdate{
		deptID:       dept.ID,
		leader:       leader,
		leaderUserID: leaderUserID,
	}, nil
}

func applyDeptLeaderUpdates(tx *gorm.DB, updates []deptLeaderUpdate) error {
	for _, item := range updates {
		if err := tx.Model(&SystemDept{}).
			Where("id = ?", item.deptID).
			Updates(map[string]any{
				"leader_user_id": item.leaderUserID,
				"leader":         item.leader,
				"updated_at":     time.Now(),
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

// Validation and helper functions

func (s *DeptService) validateDeptCreate(req *DeptCreateReq) error {
	if req.ParentID == 0 {
		return common.NewBadRequest("dept.parent.required")
	}
	if req.LeaderUserID > 0 {
		return common.NewBadRequest("dept.leader.bind_after_create")
	}
	if err := validateDeptOptionalEmail(req.Email); err != nil {
		return err
	}
	return s.ensureDeptParentExists(req.ParentID)
}

func (s *DeptService) validateDeptUpdate(dept *SystemDept, req *DeptUpdateReq) error {
	if dept == nil {
		return common.NewNotFound("dept.not_found")
	}
	if req.ParentID == dept.ID {
		return common.NewInternal("dept.update.error.parent_self")
	}
	if dept.IsRoot == common.StatusFlagYes {
		if req.ParentID != 0 {
			return common.NewBadRequest("dept.root.parent_fixed")
		}
		if normalizeSystemStatus(req.Status) != common.StatusEnabled {
			return common.NewBadRequest("dept.root.status_fixed")
		}
	} else if req.ParentID == 0 {
		return common.NewBadRequest("dept.parent.required")
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
		return "", 0, common.NewBadRequest("dept.leader.bind_after_create")
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
			return "", 0, common.NewBadRequest("dept.leader.user_invalid")
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
		return common.NewNotFound("dept.parent.not_found")
	}
	return nil
}

func (s *DeptService) ensureDeptParentNotDescendant(deptID, parentID uint64) error {
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
			return common.NewInternal("dept.update.error.parent_descendant")
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
		root, err := ensureRootDeptRecord(tx)
		if err != nil {
			return err
		}
		if err := s.attachTopLevelDeptsToRoot(tx, root.ID); err != nil {
			return err
		}
		return clearAdditionalRootFlags(tx, root.ID)
	})
}

func ensureRootDeptRecord(tx *gorm.DB) (SystemDept, error) {
	var root SystemDept
	err := tx.Where("is_root = ?", common.StatusFlagYes).Order("id asc").First(&root).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return root, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		root = SystemDept{
			ParentID:  0,
			Ancestors: "",
			IsRoot:    common.StatusFlagYes,
			DeptName:  defaultRootDeptName,
			Sort:      0,
			Status:    common.StatusEnabled,
		}
		if err := tx.Create(&root).Error; err != nil {
			return root, err
		}
		return root, nil
	}
	root.ParentID = 0
	root.Ancestors = ""
	root.IsRoot = common.StatusFlagYes
	root.Status = common.StatusEnabled
	return root, tx.Save(&root).Error
}

func (s *DeptService) attachTopLevelDeptsToRoot(tx *gorm.DB, rootID uint64) error {
	var topLevelDepts []SystemDept
	if err := tx.Where("parent_id = ? AND id <> ?", 0, rootID).Find(&topLevelDepts).Error; err != nil {
		return err
	}
	for _, dept := range topLevelDepts {
		dept.ParentID = rootID
		dept.Ancestors = fmt.Sprintf("%d", rootID)
		dept.IsRoot = common.StatusFlagNo
		if err := tx.Save(&dept).Error; err != nil {
			return err
		}
		if err := s.refreshChildAncestors(tx, dept.ID); err != nil {
			return err
		}
	}
	return nil
}

func clearAdditionalRootFlags(tx *gorm.DB, rootID uint64) error {
	return tx.Model(&SystemDept{}).
		Where("id <> ? AND is_root = ?", rootID, common.StatusFlagYes).
		Update("is_root", common.StatusFlagNo).Error
}

// Utility functions

func normalizeSystemStatus(status int) int {
	return common.NormalizeEnabledStatus(status)
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
	if value == common.StatusFlagYes {
		return common.StatusFlagYes
	}
	return common.StatusFlagNo
}
