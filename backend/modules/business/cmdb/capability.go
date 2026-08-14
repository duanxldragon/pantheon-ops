package cmdb

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

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
	ResolveDeployTargets(req DeployHostResolveRequest) ([]DeployHostTarget, error)
	WriteDeployHostResult(req DeployHostWritebackRequest) error
}

type deployCMDBCapability struct {
	db *gorm.DB
}

func NewDeployCMDBCapability(db *gorm.DB) DeployCMDBCapability {
	return &deployCMDBCapability{db: db}
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
