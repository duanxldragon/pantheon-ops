package group

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrGroupHasChildren = errors.New("cmdbgroup.has_children")
	ErrGroupCycle       = errors.New("cmdbgroup.parent_cycle")
)

type GroupService struct {
	db *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{db: db}
}

func (s *GroupService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&Group{})
}

func (s *GroupService) List(dataScope *common.DataScopeReq) ([]GroupResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var groups []Group
	if err := s.db.Order("parent_id ASC, id DESC").Find(&groups).Error; err != nil {
		return nil, err
	}
	hosts, err := s.scopedHosts(dataScope)
	if err != nil {
		return nil, err
	}
	groupsByID := make(map[uint64]Group, len(groups))
	for _, g := range groups {
		groupsByID[g.ID] = g
	}
	items := make([]GroupResponse, 0, len(groups))
	indexByID := make(map[uint64]int, len(groups))
	for _, g := range groups {
		resp := s.toResponse(&g, hosts, conditionChainFromMap(g, groupsByID))
		indexByID[resp.ID] = len(items)
		items = append(items, resp)
	}
	return buildGroupTree(items, indexByID), nil
}

func (s *GroupService) GetByID(id uint64, dataScope *common.DataScopeReq) (*GroupResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var group Group
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cmdbgroup.not_found")
		}
		return nil, err
	}
	hosts, err := s.scopedHosts(dataScope)
	if err != nil {
		return nil, err
	}
	chain, err := s.conditionChainForGroup(group)
	if err != nil {
		return nil, err
	}
	resp := s.toResponse(&group, hosts, chain)
	return &resp, nil
}

func (s *GroupService) Create(req CreateGroupRequest, dataScope *common.DataScopeReq) (*GroupResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if err := validateConditions(req.Conditions); err != nil {
		return nil, err
	}
	if err := s.validateParent(0, req.ParentID); err != nil {
		return nil, err
	}
	condJSON, _ := json.Marshal(req.Conditions)
	group := Group{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Conditions:  datatypes.JSON(condJSON),
		Description: req.Description,
	}
	if err := s.db.Create(&group).Error; err != nil {
		return nil, err
	}
	hosts, err := s.scopedHosts(dataScope)
	if err != nil {
		return nil, err
	}
	chain, err := s.conditionChainForGroup(group)
	if err != nil {
		return nil, err
	}
	resp := s.toResponse(&group, hosts, chain)
	return &resp, nil
}

func (s *GroupService) Update(id uint64, req UpdateGroupRequest, dataScope *common.DataScopeReq) (*GroupResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var group Group
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cmdbgroup.not_found")
		}
		return nil, err
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ParentID != nil {
		if err := s.validateParent(id, *req.ParentID); err != nil {
			return nil, err
		}
		updates["parent_id"] = *req.ParentID
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Conditions != nil {
		if err := validateConditions(*req.Conditions); err != nil {
			return nil, err
		}
		condJSON, _ := json.Marshal(*req.Conditions)
		updates["conditions"] = datatypes.JSON(condJSON)
	}
	updates["updated_at"] = time.Now()
	if err := s.db.Model(&group).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&group, id).Error; err != nil {
		return nil, err
	}
	hosts, err := s.scopedHosts(dataScope)
	if err != nil {
		return nil, err
	}
	chain, err := s.conditionChainForGroup(group)
	if err != nil {
		return nil, err
	}
	resp := s.toResponse(&group, hosts, chain)
	return &resp, nil
}

func (s *GroupService) Delete(id uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	var childCount int64
	if err := s.db.Model(&Group{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		return err
	}
	if childCount > 0 {
		return ErrGroupHasChildren
	}
	result := s.db.Delete(&Group{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("cmdbgroup.not_found")
	}
	return nil
}

func (s *GroupService) GetMembers(id uint64, dataScope *common.DataScopeReq) ([]Host, *Group, error) {
	if s.db == nil {
		return nil, nil, errors.New("database.not_initialized")
	}
	var group Group
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("cmdbgroup.not_found")
		}
		return nil, nil, err
	}
	hosts, err := s.scopedHosts(dataScope)
	if err != nil {
		return nil, nil, err
	}
	chain, err := s.conditionChainForGroup(group)
	if err != nil {
		return nil, nil, err
	}
	members := filterHostsByConditionChain(hosts, chain)
	return members, &group, nil
}

func (s *GroupService) toResponse(g *Group, hosts []Host, conditionChain []datatypes.JSON) GroupResponse {
	var conds ConditionExpression
	if len(g.Conditions) > 0 {
		json.Unmarshal(g.Conditions, &conds)
	}
	members := filterHostsByConditionChain(hosts, conditionChain)
	memberIDs := make(map[uint64]struct{}, len(members))
	for _, member := range members {
		memberIDs[member.ID] = struct{}{}
	}

	return GroupResponse{
		ID:                   g.ID,
		ParentID:             g.ParentID,
		Name:                 g.Name,
		Description:          g.Description,
		Conditions:           conds,
		MemberCount:          len(members),
		AggregateMemberCount: len(members),
		CreatedAt:            g.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            g.UpdatedAt.Format(time.RFC3339),
		memberIDs:            memberIDs,
	}
}

func (s *GroupService) validateParent(currentID uint64, parentID uint64) error {
	if parentID == 0 {
		return nil
	}
	if currentID > 0 && parentID == currentID {
		return ErrGroupCycle
	}
	var parent Group
	if err := s.db.First(&parent, parentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("cmdbgroup.parent_not_found")
		}
		return err
	}
	if currentID == 0 {
		return nil
	}
	for parent.ParentID != 0 {
		if parent.ParentID == currentID {
			return ErrGroupCycle
		}
		nextID := parent.ParentID
		if err := s.db.First(&parent, nextID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (s *GroupService) conditionChainForGroup(group Group) ([]datatypes.JSON, error) {
	chain := []datatypes.JSON{group.Conditions}
	visited := map[uint64]struct{}{group.ID: {}}
	parentID := group.ParentID
	for parentID != 0 {
		if _, ok := visited[parentID]; ok {
			return nil, ErrGroupCycle
		}
		var parent Group
		if err := s.db.First(&parent, parentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return nil, err
		}
		visited[parent.ID] = struct{}{}
		chain = append([]datatypes.JSON{parent.Conditions}, chain...)
		parentID = parent.ParentID
	}
	return chain, nil
}

func conditionChainFromMap(group Group, groupsByID map[uint64]Group) []datatypes.JSON {
	chain := []datatypes.JSON{group.Conditions}
	visited := map[uint64]struct{}{group.ID: {}}
	parentID := group.ParentID
	for parentID != 0 {
		if _, ok := visited[parentID]; ok {
			break
		}
		parent, ok := groupsByID[parentID]
		if !ok {
			break
		}
		visited[parent.ID] = struct{}{}
		chain = append([]datatypes.JSON{parent.Conditions}, chain...)
		parentID = parent.ParentID
	}
	return chain
}

func buildGroupTree(items []GroupResponse, indexByID map[uint64]int) []GroupResponse {
	childrenByParent := make(map[uint64][]GroupResponse)
	for _, item := range items {
		item.Children = nil
		childrenByParent[item.ParentID] = append(childrenByParent[item.ParentID], item)
	}
	var attach func(GroupResponse) GroupResponse
	attach = func(item GroupResponse) GroupResponse {
		children := childrenByParent[item.ID]
		if len(children) > 0 {
			item.Children = make([]GroupResponse, 0, len(children))
			aggregateMemberIDs := make(map[uint64]struct{}, len(item.memberIDs))
			for id := range item.memberIDs {
				aggregateMemberIDs[id] = struct{}{}
			}
			for _, child := range children {
				attachedChild := attach(child)
				item.Children = append(item.Children, attachedChild)
				item.DescendantGroupCount += 1 + attachedChild.DescendantGroupCount
				for id := range attachedChild.memberIDs {
					aggregateMemberIDs[id] = struct{}{}
				}
			}
			item.ChildCount = len(item.Children)
			item.AggregateMemberCount = len(aggregateMemberIDs)
			item.memberIDs = aggregateMemberIDs
		}
		return item
	}
	roots := make([]GroupResponse, 0)
	for _, item := range items {
		if item.ParentID == 0 {
			roots = append(roots, attach(item))
			continue
		}
		if _, ok := indexByID[item.ParentID]; !ok {
			roots = append(roots, attach(item))
		}
	}
	return roots
}

func (s *GroupService) scopedHosts(dataScope *common.DataScopeReq) ([]Host, error) {
	var hosts []Host
	if err := s.db.Model(&Host{}).Scopes(database.WithDataScope(dataScope)).Find(&hosts).Error; err != nil {
		return nil, err
	}
	return hosts, nil
}

func filterHostsByConditions(hosts []Host, conditions datatypes.JSON) []Host {
	var expr ConditionExpression
	if len(conditions) == 0 {
		return hosts
	}
	if err := json.Unmarshal(conditions, &expr); err != nil {
		return hosts
	}
	var result []Host
	for _, h := range hosts {
		if matchHost(h, expr) {
			result = append(result, h)
		}
	}
	return result
}

func filterHostsByConditionChain(hosts []Host, conditionChain []datatypes.JSON) []Host {
	result := hosts
	for _, conditions := range conditionChain {
		result = filterHostsByConditions(result, conditions)
	}
	return result
}

func validateConditions(expr ConditionExpression) error {
	operator := strings.TrimSpace(expr.Operator)
	if operator == "" {
		operator = "AND"
	}
	if operator != "AND" && operator != "OR" {
		return errors.New("cmdbgroup.invalid_conditions")
	}
	if len(expr.Rules) == 0 {
		return errors.New("cmdbgroup.invalid_conditions")
	}
	for _, rule := range expr.Rules {
		if strings.TrimSpace(rule.Key) == "" || strings.TrimSpace(rule.Val) == "" {
			return errors.New("cmdbgroup.invalid_conditions")
		}
		switch strings.TrimSpace(rule.Op) {
		case "eq", "neq", "in", "notIn":
		default:
			return errors.New("cmdbgroup.invalid_conditions")
		}
	}
	return nil
}

func matchHost(h Host, expr ConditionExpression) bool {
	var labels []LabelEntry
	if len(h.LabelValues) > 0 {
		json.Unmarshal(h.LabelValues, &labels)
	}
	labelMap := make(map[string]string)
	for _, l := range labels {
		labelMap[l.Key] = l.Val
	}
	if expr.Operator == "OR" {
		for _, rule := range expr.Rules {
			if matchRule(labelMap, rule) {
				return true
			}
		}
		return false
	}
	for _, rule := range expr.Rules {
		if !matchRule(labelMap, rule) {
			return false
		}
	}
	return true
}

func matchRule(labelMap map[string]string, rule ConditionRule) bool {
	val, ok := labelMap[rule.Key]
	if !ok {
		return false
	}
	switch rule.Op {
	case "eq":
		return val == rule.Val
	case "neq":
		return val != rule.Val
	case "in":
		for _, v := range strings.Split(rule.Val, ",") {
			if val == strings.TrimSpace(v) {
				return true
			}
		}
		return false
	case "notIn":
		for _, v := range strings.Split(rule.Val, ",") {
			if val == strings.TrimSpace(v) {
				return false
			}
		}
		return true
	default:
		return val == rule.Val
	}
}
