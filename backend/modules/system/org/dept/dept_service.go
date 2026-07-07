package org

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"
)

type DeptService struct {
	db *gorm.DB
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

	var dept SystemDept
	if err := s.db.First(&dept, deptID).Error; err != nil {
		return err
	}
	if dept.IsRoot == common.StatusFlagYes {
		return common.NewForbidden("dept.root.delete_forbidden")
	}

	var childCount int64
	if err := s.db.Model(&SystemDept{}).Where("parent_id = ?", deptID).Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return common.NewInternal("dept.delete.error.has_children")
	}

	var postCount int64
	if err := s.db.Table("system_post").Where("dept_id = ? AND deleted_at IS NULL", deptID).Count(&postCount).Error; err != nil {
		return err
	}
	if postCount > 0 {
		return common.NewInternal("dept.delete.error.has_posts")
	}

	var userCount int64
	if err := s.db.Table("system_user").Where("dept_id = ? AND deleted_at IS NULL", deptID).Count(&userCount).Error; err != nil {
		return err
	}
	if userCount > 0 {
		return common.NewInternal("dept.delete.error.has_users")
	}

	return s.db.Delete(&SystemDept{}, deptID).Error
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
	if err := s.db.Where("id IN ?", normalizedIDs).Find(&depts).Error; err != nil {
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
		Where("id IN ?", normalizedIDs).
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
		return 0, common.NewNotFound("dept.batch.not_found")
	}
	updates := make([]struct {
		DeptID       uint64
		Leader       string
		LeaderUserID uint64
	}, 0, len(depts))
	for _, dept := range depts {
		if dept.IsRoot == common.StatusFlagYes {
			return 0, common.NewForbidden("dept.root.update_forbidden")
		}
		item := deptToLeader[dept.ID]
		if item.LeaderUserID == 0 {
			return 0, common.NewBadRequest("dept.leader.required")
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
		var root SystemDept
		err := tx.Where("is_root = ?", common.StatusFlagYes).Order("id asc").First(&root).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
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
				return err
			}
		} else {
			root.ParentID = 0
			root.Ancestors = ""
			root.IsRoot = common.StatusFlagYes
			root.Status = common.StatusEnabled
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
			dept.IsRoot = common.StatusFlagNo
			if err := tx.Save(&dept).Error; err != nil {
				return err
			}
			if err := s.refreshChildAncestors(tx, dept.ID); err != nil {
				return err
			}
		}

		return tx.Model(&SystemDept{}).
			Where("id <> ? AND is_root = ?", root.ID, common.StatusFlagYes).
			Update("is_root", common.StatusFlagNo).Error
	})
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
