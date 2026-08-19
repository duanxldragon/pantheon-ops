package cmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	cmdbgroup "pantheon-base/modules/business/cmdb/group"
	cmdbhost "pantheon-base/modules/business/cmdb/host"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DeployHostTarget struct {
	ID                uint64
	Hostname          string
	IP                string
	SSHPort           int
	OS                string
	Status            string
	BusinessScopeID   uint64
	BusinessScopeName string
	LabelValues       datatypes.JSON
	DeptID            uint64
}

type InstalledComponentUpsert struct {
	Name           string
	Version        string
	DeployedAt     time.Time
	DeployTaskID   uint64
	DeployTaskName string
	ExecutorType   string
}

type DeployHostResolveRequest struct {
	BusinessScopeID uint64
	TargetType      string
	TargetIDs       []uint64
	DataScope       *common.DataScopeReq
}

type DeployHostWritebackRequest struct {
	HostID                uint64
	Status                string
	Actor                 string
	InstalledComponents   []InstalledComponentUpsert
	RemovedComponentNames []string
}

type DeployCMDBCapability interface {
	bizcap.CMDBHostReader
	bizcap.CMDBOwnershipCommand
	ResolveDeployTargets(req DeployHostResolveRequest) ([]DeployHostTarget, error)
	WriteDeployHostResult(req DeployHostWritebackRequest) error
}

type deployCMDBCapability struct {
	db             *gorm.DB
	bizScopeReader bizcap.BizScopeReader
}

// NewDeployCMDBCapability creates the CMDB capability consumed by deploy and bizscope.
func NewDeployCMDBCapability(db *gorm.DB, readers ...bizcap.BizScopeReader) *deployCMDBCapability {
	capability := &deployCMDBCapability{db: db}
	if len(readers) > 0 {
		capability.bizScopeReader = readers[0]
	}
	return capability
}

func (c *deployCMDBCapability) SetBizScopeReader(reader bizcap.BizScopeReader) {
	c.bizScopeReader = reader
}

var _ bizcap.CMDBHostReader = (*deployCMDBCapability)(nil)
var _ bizcap.CMDBOwnershipCommand = (*deployCMDBCapability)(nil)

func (c *deployCMDBCapability) GetByIDs(ctx context.Context, req bizcap.HostIDsQuery) (bizcap.HostPage, error) {
	if c.db == nil {
		return bizcap.HostPage{}, errors.New("database.not_initialized")
	}
	ids := common.NormalizeUint64IDs(req.HostIDs)
	if len(ids) == 0 {
		return bizcap.HostPage{Items: []bizcap.HostRef{}}, nil
	}
	query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("id IN ?", ids)
	var rows []cmdbhost.Host
	if err := query.Order("id ASC").Find(&rows).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	return bizcap.HostPage{Items: mapCapabilityHostRefs(rows), Total: int64(len(rows))}, nil
}

func (c *deployCMDBCapability) ResolveDeployTargets(req DeployHostResolveRequest) ([]DeployHostTarget, error) {
	if c.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	targetIDs := common.NormalizeUint64IDs(req.TargetIDs)
	if len(targetIDs) == 0 {
		return []DeployHostTarget{}, nil
	}
	switch strings.TrimSpace(req.TargetType) {
	case "host":
		return c.resolveHostTargets(req.BusinessScopeID, targetIDs, req.DataScope)
	case "group":
		return c.resolveGroupTargets(targetIDs, req.DataScope)
	default:
		return nil, errors.New("business.deploy.task.invalidTargetType")
	}
}

func (c *deployCMDBCapability) WriteDeployHostResult(req DeployHostWritebackRequest) error {
	if c.db == nil {
		return errors.New("database.not_initialized")
	}
	now := time.Now()
	status := strings.TrimSpace(req.Status)

	return c.db.Transaction(func(tx *gorm.DB) error {
		if len(req.InstalledComponents) == 0 && len(req.RemovedComponentNames) == 0 {
			updates := map[string]any{
				"updated_by": req.Actor,
				"updated_at": now,
			}
			if status != "" {
				updates["status"] = status
			}
			return tx.Table("biz_cmdb_host").Where(idWhereClause, req.HostID).Updates(updates).Error
		}
		components, err := loadInstalledComponents(tx, req.HostID)
		if err != nil {
			return err
		}
		components = removeComponentsByName(components, req.RemovedComponentNames)
		components = upsertComponents(components, req.InstalledComponents)
		status = reconcileHostStatus(status, len(components))
		payload, _ := json.Marshal(components)
		updates := map[string]any{
			"installed_components": datatypes.JSON(payload),
			"updated_by":           req.Actor,
			"updated_at":           now,
		}
		if status != "" {
			updates["status"] = status
		}
		return tx.Table("biz_cmdb_host").Where(idWhereClause, req.HostID).Updates(updates).Error
	})
}

func (c *deployCMDBCapability) ListByBusinessScope(ctx context.Context, req bizcap.HostScopeQuery) (bizcap.HostPage, error) {
	if c.db == nil {
		return bizcap.HostPage{}, errors.New("database.not_initialized")
	}
	query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("business_scope_id = ?", req.BusinessScopeID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	var rows []cmdbhost.Host
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	return bizcap.HostPage{Items: mapCapabilityHostRefs(rows), Total: total}, nil
}

func (c *deployCMDBCapability) ListAvailable(ctx context.Context, req bizcap.AvailableHostQuery) (bizcap.HostPage, error) {
	if c.db == nil {
		return bizcap.HostPage{}, errors.New("database.not_initialized")
	}
	query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("(business_scope_id = 0 OR business_scope_id IS NULL)")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	var rows []cmdbhost.Host
	if err := query.Order("id DESC").Find(&rows).Error; err != nil {
		return bizcap.HostPage{}, err
	}
	return bizcap.HostPage{Items: mapCapabilityHostRefs(rows), Total: total}, nil
}

func (c *deployCMDBCapability) HasBusinessScopeReferences(ctx context.Context, businessScopeID uint64) (bool, error) {
	if c.db == nil {
		return false, errors.New("database.not_initialized")
	}
	var count int64
	if err := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).Where("business_scope_id = ?", businessScopeID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *deployCMDBCapability) Bind(ctx context.Context, req bizcap.BindOwnershipRequest) error {
	if c.db == nil {
		return errors.New("database.not_initialized")
	}
	hostIDs := common.NormalizeUint64IDs(req.HostIDs)
	if len(hostIDs) == 0 {
		return errors.New("param.invalid")
	}
	if req.BusinessScopeID == 0 {
		return errors.New("business.bizscope.notFound")
	}
	return c.WithBusinessScopeOwnershipLock(ctx, req.BusinessScopeID, func() error {
		if c.bizScopeReader == nil {
			return errors.New("business.bizscope.readerNotConfigured")
		}
		if _, err := c.bizScopeReader.GetActive(ctx, req.BusinessScopeID, req.DataScope); err != nil {
			return err
		}
		return c.bindUnlocked(ctx, req, hostIDs)
	})
}

func (c *deployCMDBCapability) bindUnlocked(ctx context.Context, req bizcap.BindOwnershipRequest, hostIDs []uint64) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found int64
		if err := tx.Model(&cmdbhost.Host{}).
			Scopes(database.WithDataScope(req.DataScope)).
			Where("id IN ?", hostIDs).
			Count(&found).Error; err != nil {
			return err
		}
		if found != int64(len(hostIDs)) {
			return errors.New("cmdbhost.not_found")
		}
		var owned int64
		if err := tx.Model(&cmdbhost.Host{}).
			Scopes(database.WithDataScope(req.DataScope)).
			Where("id IN ?", hostIDs).
			Where("business_scope_id <> 0").
			Count(&owned).Error; err != nil {
			return err
		}
		if owned > 0 {
			return errors.New("cmdbhost.ownership_conflict")
		}
		updates := map[string]any{
			"business_scope_id":   req.BusinessScopeID,
			"business_scope_code": strings.TrimSpace(req.BusinessScopeCode),
			"business_scope_name": strings.TrimSpace(req.BusinessScopeName),
			"status":              "assigned",
			"updated_by":          strings.TrimSpace(req.Actor),
			"updated_at":          time.Now(),
		}
		result := tx.Model(&cmdbhost.Host{}).
			Scopes(database.WithDataScope(req.DataScope)).
			Where("id IN ?", hostIDs).
			Where("business_scope_id = 0").
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == int64(len(hostIDs)) {
			return nil
		}
		return classifyOwnershipBindFailure(tx, req.DataScope, hostIDs)
	})
}

func (c *deployCMDBCapability) WithBusinessScopeOwnershipLock(ctx context.Context, businessScopeID uint64, action func() error) error {
	if c.db == nil {
		return errors.New("database.not_initialized")
	}
	if businessScopeID == 0 {
		return errors.New("business.bizscope.notFound")
	}
	if c.db.Name() != "mysql" {
		return action()
	}
	lockName := fmt.Sprintf("pantheon:bizscope:%d", businessScopeID)
	return c.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
		var acquired int
		if err := conn.Raw("SELECT GET_LOCK(?, 10)", lockName).Scan(&acquired).Error; err != nil {
			return err
		}
		if acquired != 1 {
			return errors.New("cmdbhost.ownership_lock_timeout")
		}
		defer func() {
			_ = conn.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
		}()
		return action()
	})
}

func (c *deployCMDBCapability) Unbind(ctx context.Context, req bizcap.UnbindOwnershipRequest) error {
	if c.db == nil {
		return errors.New("database.not_initialized")
	}
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"business_scope_id":   uint64(0),
			"business_scope_code": "",
			"business_scope_name": "",
			"status":              gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", "assigned", "pending"),
			"updated_by":          strings.TrimSpace(req.Actor),
			"updated_at":          time.Now(),
		}
		result := tx.Model(&cmdbhost.Host{}).
			Scopes(database.WithDataScope(req.DataScope)).
			Where("id = ? AND business_scope_id = ?", req.HostID, req.BusinessScopeID).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return classifyOwnershipUnbindFailure(tx, req)
		}
		return nil
	})
}

func classifyOwnershipBindFailure(tx *gorm.DB, dataScope *common.DataScopeReq, hostIDs []uint64) error {
	var found int64
	if err := tx.Model(&cmdbhost.Host{}).
		Scopes(database.WithDataScope(dataScope)).
		Where("id IN ?", hostIDs).
		Count(&found).Error; err != nil {
		return err
	}
	if found != int64(len(hostIDs)) {
		return errors.New("cmdbhost.not_found")
	}
	return errors.New("cmdbhost.ownership_conflict")
}

func classifyOwnershipUnbindFailure(tx *gorm.DB, req bizcap.UnbindOwnershipRequest) error {
	var found int64
	if err := tx.Model(&cmdbhost.Host{}).
		Scopes(database.WithDataScope(req.DataScope)).
		Where("id = ?", req.HostID).
		Count(&found).Error; err != nil {
		return err
	}
	if found == 0 {
		return errors.New("cmdbhost.not_found")
	}
	return errors.New("cmdbhost.ownership_conflict")
}

func loadInstalledComponents(tx *gorm.DB, hostID uint64) ([]cmdbhost.ComponentEntry, error) {
	var snapshot struct {
		InstalledComponents datatypes.JSON `gorm:"column:installed_components"`
	}
	if err := tx.Table("biz_cmdb_host").Select("installed_components").Where(idWhereClause, hostID).Take(&snapshot).Error; err != nil {
		return nil, err
	}
	var components []cmdbhost.ComponentEntry
	if len(snapshot.InstalledComponents) > 0 {
		_ = json.Unmarshal(snapshot.InstalledComponents, &components)
	}
	if components == nil {
		components = []cmdbhost.ComponentEntry{}
	}
	return components, nil
}

func mapCapabilityHostRefs(rows []cmdbhost.Host) []bizcap.HostRef {
	items := make([]bizcap.HostRef, 0, len(rows))
	for _, row := range rows {
		items = append(items, bizcap.HostRef{
			ID:                row.ID,
			Hostname:          row.Hostname,
			IP:                row.IP,
			SSHPort:           row.SSHPort,
			OS:                row.OS,
			Status:            row.Status,
			BusinessScopeID:   row.BusinessScopeID,
			BusinessScopeCode: row.BusinessScopeCode,
			BusinessScopeName: row.BusinessScopeName,
			LabelValues:       row.LabelValues,
			DeptID:            row.DeptID,
		})
	}
	return items
}

func removeComponentsByName(components []cmdbhost.ComponentEntry, names []string) []cmdbhost.ComponentEntry {
	if len(names) == 0 {
		return components
	}
	removed := make(map[string]struct{}, len(names))
	for _, name := range names {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			removed[strings.ToLower(trimmed)] = struct{}{}
		}
	}
	filtered := make([]cmdbhost.ComponentEntry, 0, len(components))
	for _, component := range components {
		if _, ok := removed[strings.ToLower(strings.TrimSpace(component.Name))]; ok {
			continue
		}
		filtered = append(filtered, component)
	}
	return filtered
}

func upsertComponents(components []cmdbhost.ComponentEntry, items []InstalledComponentUpsert) []cmdbhost.ComponentEntry {
	for _, item := range items {
		updated := false
		for index := range components {
			if strings.EqualFold(strings.TrimSpace(components[index].Name), strings.TrimSpace(item.Name)) {
				components[index].Version = item.Version
				components[index].DeployedAt = item.DeployedAt.Format(time.RFC3339)
				components[index].DeployTaskID = item.DeployTaskID
				components[index].DeployTaskName = item.DeployTaskName
				components[index].ExecutorType = item.ExecutorType
				updated = true
				break
			}
		}
		if updated {
			continue
		}
		components = append(components, cmdbhost.ComponentEntry{
			Name:           item.Name,
			Version:        item.Version,
			DeployedAt:     item.DeployedAt.Format(time.RFC3339),
			DeployTaskID:   item.DeployTaskID,
			DeployTaskName: item.DeployTaskName,
			ExecutorType:   item.ExecutorType,
		})
	}
	return components
}

func reconcileHostStatus(status string, componentCount int) string {
	if status == "assigned" && componentCount > 0 {
		return "online"
	}
	if status == "online" && componentCount == 0 {
		return "assigned"
	}
	return status
}

func (c *deployCMDBCapability) resolveHostTargets(businessScopeID uint64, targetIDs []uint64, dataScope *common.DataScopeReq) ([]DeployHostTarget, error) {
	query := c.db.Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(dataScope)).Where("id IN ?", targetIDs)
	if businessScopeID > 0 {
		query = query.Where("business_scope_id = ?", businessScopeID)
	}
	var rows []cmdbhost.Host
	if err := query.Order(idAscOrder).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]DeployHostTarget, 0, len(rows))
	for _, row := range rows {
		items = append(items, DeployHostTarget{
			ID:                row.ID,
			Hostname:          row.Hostname,
			IP:                row.IP,
			SSHPort:           row.SSHPort,
			OS:                row.OS,
			Status:            row.Status,
			BusinessScopeID:   row.BusinessScopeID,
			BusinessScopeName: row.BusinessScopeName,
			LabelValues:       row.LabelValues,
			DeptID:            row.DeptID,
		})
	}
	return items, nil
}

func (c *deployCMDBCapability) resolveGroupTargets(targetIDs []uint64, dataScope *common.DataScopeReq) ([]DeployHostTarget, error) {
	var selectedGroups []cmdbgroup.Group
	if err := c.db.Where("id IN ?", targetIDs).Find(&selectedGroups).Error; err != nil {
		return nil, err
	}
	var allGroups []cmdbgroup.Group
	if err := c.db.Order(idAscOrder).Find(&allGroups).Error; err != nil {
		return nil, err
	}
	groupsByID := make(map[uint64]cmdbgroup.Group, len(allGroups))
	for _, group := range allGroups {
		groupsByID[group.ID] = group
	}
	var hosts []cmdbhost.Host
	if err := c.db.Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(dataScope)).Order(idAscOrder).Find(&hosts).Error; err != nil {
		return nil, err
	}
	result := make([]DeployHostTarget, 0)
	seen := make(map[uint64]struct{})
	for _, host := range hosts {
		for _, group := range selectedGroups {
			if groupConditionChainMatchesHost(conditionChainFromMap(group, groupsByID), host.LabelValues) {
				if _, ok := seen[host.ID]; ok {
					break
				}
				seen[host.ID] = struct{}{}
				result = append(result, DeployHostTarget{
					ID:                host.ID,
					Hostname:          host.Hostname,
					IP:                host.IP,
					SSHPort:           host.SSHPort,
					OS:                host.OS,
					Status:            host.Status,
					BusinessScopeID:   host.BusinessScopeID,
					BusinessScopeName: host.BusinessScopeName,
					LabelValues:       host.LabelValues,
					DeptID:            host.DeptID,
				})
				break
			}
		}
	}
	return result, nil
}

func conditionChainFromMap(group cmdbgroup.Group, groupsByID map[uint64]cmdbgroup.Group) []datatypes.JSON {
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

func groupConditionChainMatchesHost(conditionChain []datatypes.JSON, labelJSON datatypes.JSON) bool {
	for _, conditionJSON := range conditionChain {
		if !groupMatchesHost(conditionJSON, labelJSON) {
			return false
		}
	}
	return len(conditionChain) > 0
}

func groupMatchesHost(conditionJSON, labelJSON datatypes.JSON) bool {
	var condition struct {
		Operator string `json:"operator"`
		Rules    []struct {
			Key string `json:"key"`
			Op  string `json:"op"`
			Val string `json:"val"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(conditionJSON, &condition); err != nil || len(condition.Rules) == 0 {
		return false
	}
	var labels []cmdbhost.LabelEntry
	_ = json.Unmarshal(labelJSON, &labels)
	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		labelMap[label.Key] = label.Val
	}
	operator := strings.ToUpper(strings.TrimSpace(condition.Operator))
	if operator == "" {
		operator = "AND"
	}
	matched := operator == "AND"
	for _, rule := range condition.Rules {
		actual, ok := labelMap[rule.Key]
		ruleMatched := false
		if ok {
			ruleMatched = labelRuleMatches(actual, rule.Op, rule.Val)
		}
		if operator == "OR" && ruleMatched {
			return true
		}
		if operator == "AND" && !ruleMatched {
			return false
		}
		matched = ruleMatched
	}
	return matched
}

func labelRuleMatches(actual, op, raw string) bool {
	switch strings.TrimSpace(op) {
	case "eq":
		return actual == raw
	case "neq":
		return actual != raw
	case "in":
		for _, item := range strings.Split(raw, ",") {
			if actual == strings.TrimSpace(item) {
				return true
			}
		}
		return false
	case "notIn":
		for _, item := range strings.Split(raw, ",") {
			if actual == strings.TrimSpace(item) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
