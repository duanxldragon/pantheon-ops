package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"pantheon-ops/backend/modules/business/bizscope"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"golang.org/x/crypto/ssh"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func hostKeyCallback(expectedFingerprint string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if strings.TrimSpace(ssh.FingerprintSHA256(key)) != strings.TrimSpace(expectedFingerprint) {
			return errors.New("cmdbhost.ssh_host_key_mismatch")
		}
		return nil
	}
}

type HostService struct {
	db *gorm.DB
}

func NewHostService(db *gorm.DB) *HostService {
	return &HostService{db: db}
}

func (s *HostService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&Host{})
}

func (s *HostService) hostQuery(dataScope *common.DataScopeReq) *gorm.DB {
	return s.db.Model(&Host{}).Scopes(database.WithDataScope(dataScope))
}

func (s *HostService) List(query HostListQuery, dataScope *common.DataScopeReq) (*HostListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}

	db := s.hostQuery(dataScope)
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		db = db.Where("hostname LIKE ? OR ip LIKE ?", like, like)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OS != "" {
		db = db.Where("os = ?", query.OS)
	}
	if query.BusinessScopeID > 0 {
		db = db.Where("business_scope_id = ?", query.BusinessScopeID)
	}
	if query.DeptID > 0 {
		db = db.Where("dept_id = ?", query.DeptID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var hosts []Host
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("id DESC").Offset(offset).Limit(query.PageSize).Find(&hosts).Error; err != nil {
		return nil, err
	}

	groupIndex, err := s.loadGroupIndex()
	if err != nil {
		return nil, err
	}
	items := make([]HostResponse, len(hosts))
	for i, h := range hosts {
		items[i] = hostToResponse(&h, groupIndex)
	}

	return &HostListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *HostService) GetByID(id uint64, dataScope *common.DataScopeReq) (*HostResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var host Host
	if err := s.hostQuery(dataScope).First(&host, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cmdbhost.not_found")
		}
		return nil, err
	}
	groupIndex, err := s.loadGroupIndex()
	if err != nil {
		return nil, err
	}
	resp := hostToResponse(&host, groupIndex)
	return &resp, nil
}

func (s *HostService) Create(req CreateHostRequest, createdBy string) (*HostResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if s.ipExists(req.IP, 0) {
		return nil, errors.New("cmdbhost.ip_exists")
	}

	labelsJSON, _ := json.Marshal(req.Labels)
	businessScopeID := req.BusinessScopeID
	businessScopeCode := ""
	businessScopeName := ""
	status := "pending"
	if businessScopeID > 0 {
		scopeRecord, err := s.getBusinessScope(businessScopeID)
		if err != nil {
			return nil, err
		}
		businessScopeCode = scopeRecord.Code
		businessScopeName = scopeRecord.Name
		status = "assigned"
	}
	host := Host{
		Hostname:          req.Hostname,
		IP:                req.IP,
		SSHPort:           req.SSHPort,
		OS:                req.OS,
		OSVersion:         req.OSVersion,
		CPUCores:          req.CPUCores,
		MemoryGB:          req.MemoryGB,
		DiskGB:            req.DiskGB,
		LabelValues:       datatypes.JSON(labelsJSON),
		BusinessScopeID:   businessScopeID,
		BusinessScopeCode: businessScopeCode,
		BusinessScopeName: businessScopeName,
		DeptID:            req.DeptID,
		Owner:             req.Owner,
		Remark:            req.Remark,
		Status:            status,
		CreatedBy:         createdBy,
		UpdatedBy:         createdBy,
	}
	if host.SSHPort == 0 {
		host.SSHPort = 22
	}

	if err := s.db.Create(&host).Error; err != nil {
		return nil, err
	}
	groupIndex, err := s.loadGroupIndex()
	if err != nil {
		return nil, err
	}
	resp := hostToResponse(&host, groupIndex)
	return &resp, nil
}

func (s *HostService) Update(id uint64, req UpdateHostRequest, updatedBy string, dataScope *common.DataScopeReq) (*HostResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var host Host
	if err := s.hostQuery(dataScope).First(&host, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cmdbhost.not_found")
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if req.Hostname != nil {
		updates["hostname"] = *req.Hostname
	}
	if req.IP != nil {
		if s.ipExists(*req.IP, id) {
			return nil, errors.New("cmdbhost.ip_exists")
		}
		updates["ip"] = *req.IP
	}
	if req.SSHPort != nil {
		updates["ssh_port"] = *req.SSHPort
	}
	if req.OS != nil {
		updates["os"] = *req.OS
	}
	if req.OSVersion != nil {
		updates["os_version"] = *req.OSVersion
	}
	if req.CPUCores != nil {
		updates["cpu_cores"] = *req.CPUCores
	}
	if req.MemoryGB != nil {
		updates["memory_gb"] = *req.MemoryGB
	}
	if req.DiskGB != nil {
		updates["disk_gb"] = *req.DiskGB
	}
	if req.Labels != nil {
		labelsJSON, _ := json.Marshal(*req.Labels)
		updates["label_values"] = datatypes.JSON(labelsJSON)
	}
	if req.BusinessScopeID != nil {
		if *req.BusinessScopeID == 0 {
			updates["business_scope_id"] = uint64(0)
			updates["business_scope_code"] = ""
			updates["business_scope_name"] = ""
			if host.Status == "assigned" {
				updates["status"] = "pending"
			}
		} else {
			scopeRecord, err := s.getBusinessScope(*req.BusinessScopeID)
			if err != nil {
				return nil, err
			}
			updates["business_scope_id"] = scopeRecord.ID
			updates["business_scope_code"] = scopeRecord.Code
			updates["business_scope_name"] = scopeRecord.Name
			if host.Status == "pending" || host.Status == "" {
				updates["status"] = "assigned"
			}
		}
	}
	if req.DeptID != nil {
		updates["dept_id"] = *req.DeptID
	}
	if req.Owner != nil {
		updates["owner"] = *req.Owner
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	updates["updated_by"] = updatedBy
	updates["updated_at"] = time.Now()

	if err := s.db.Model(&host).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.hostQuery(dataScope).First(&host, id).Error; err != nil {
		return nil, err
	}
	groupIndex, err := s.loadGroupIndex()
	if err != nil {
		return nil, err
	}
	resp := hostToResponse(&host, groupIndex)
	return &resp, nil
}

func (s *HostService) Delete(id uint64, dataScope *common.DataScopeReq) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	result := s.hostQuery(dataScope).Where("id = ?", id).Delete(&Host{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("cmdbhost.not_found")
	}
	return nil
}

func (s *HostService) Collect(id uint64, req CollectRequest, dataScope *common.DataScopeReq) (*HostResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var host Host
	if err := s.hostQuery(dataScope).First(&host, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cmdbhost.not_found")
		}
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", host.IP, host.SSHPort)
	fingerprint := strings.TrimSpace(req.HostFingerprint)
	if fingerprint == "" {
		return nil, errors.New("cmdbhost.ssh_host_key_required")
	}
	config := &ssh.ClientConfig{
		User:            req.SSHUser,
		HostKeyCallback: hostKeyCallback(fingerprint),
		Timeout:         10 * time.Second,
	}
	if req.AuthMode == "private_key" {
		signer, err := ssh.ParsePrivateKey([]byte(req.SSHPrivateKey))
		if err != nil {
			return nil, errors.New("cmdbhost.ssh_auth_failed")
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else {
		config.Auth = []ssh.AuthMethod{ssh.Password(req.SSHPassword)}
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, errors.New("cmdbhost.ssh_connect_failed")
	}
	defer client.Close()

	info, err := collectSystemInfo(client)
	if err != nil {
		return nil, errors.New("cmdbhost.collect_failed")
	}

	updates := map[string]interface{}{
		"os":         info.OS,
		"os_version": info.OSVersion,
		"cpu_cores":  info.CPUCores,
		"memory_gb":  info.MemoryGB,
		"disk_gb":    info.DiskGB,
		"updated_at": time.Now(),
	}
	if err := s.db.Model(&host).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.hostQuery(dataScope).First(&host, id).Error; err != nil {
		return nil, err
	}
	groupIndex, err := s.loadGroupIndex()
	if err != nil {
		return nil, err
	}
	resp := hostToResponse(&host, groupIndex)
	return &resp, nil
}

func (s *HostService) UpdateStatus(id uint64, status string, dataScope *common.DataScopeReq) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	result := s.hostQuery(dataScope).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("cmdbhost.not_found")
	}
	return nil
}

func (s *HostService) ipExists(ip string, excludeID uint64) bool {
	var count int64
	q := s.db.Model(&Host{}).Where("ip = ?", ip)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

type systemInfo struct {
	OS        string
	OSVersion string
	CPUCores  int
	MemoryGB  float64
	DiskGB    float64
}

const collectSystemInfoCommand = `os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
if [ "$os" = "linux" ] && [ -r /etc/os-release ]; then
  . /etc/os-release
  os_version="${PRETTY_NAME:-${NAME:-}}"
fi
if [ -z "$os_version" ]; then
  os_version="$(uname -sr 2>/dev/null)"
fi
cpu_cores="$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN)"
memory_gb="$(awk '/MemTotal/ {printf "%.1f", $2/1024/1024}' /proc/meminfo)"
disk_gb="$(lsblk -b -d -n -o SIZE,TYPE 2>/dev/null | awk '$2=="disk"{sum+=$1} END {if (sum > 0) printf "%.0f", sum/1024/1024/1024}')"
if [ -z "$disk_gb" ]; then
  disk_gb="$(df -BG -x tmpfs -x devtmpfs -x overlay --total 2>/dev/null | awk 'END {gsub(/G/, "", $2); print $2 + 0}')"
fi
printf 'os=%s\n' "$os"
printf 'os_version=%s\n' "$os_version"
printf 'cpu_cores=%s\n' "$cpu_cores"
printf 'memory_gb=%s\n' "$memory_gb"
printf 'disk_gb=%s\n' "$disk_gb"`

func collectSystemInfo(client *ssh.Client) (*systemInfo, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(collectSystemInfoCommand)
	if err != nil {
		return nil, err
	}
	return parseSystemInfoOutput(output)
}

func parseSystemInfoOutput(output []byte) (*systemInfo, error) {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	fields := map[string]string{}
	for _, line := range lines {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	if len(fields) > 0 {
		return parseSystemInfoFields(fields)
	}

	info := &systemInfo{}
	if len(lines) < 5 {
		return nil, fmt.Errorf("incomplete system info output: got %d lines", len(lines))
	}

	info.OS = strings.ToLower(strings.TrimSpace(lines[0]))
	info.OSVersion = strings.TrimSpace(lines[1])
	cpuCores, err := strconv.Atoi(strings.TrimSpace(lines[2]))
	if err != nil {
		return nil, err
	}
	memoryGB, err := strconv.ParseFloat(strings.TrimSpace(lines[3]), 64)
	if err != nil {
		return nil, err
	}
	diskGB, err := strconv.ParseFloat(strings.TrimSpace(lines[4]), 64)
	if err != nil {
		return nil, err
	}
	if info.OS == "" || info.OSVersion == "" || cpuCores <= 0 || memoryGB <= 0 || diskGB <= 0 {
		return nil, errors.New("incomplete system info values")
	}

	info.CPUCores = cpuCores
	info.MemoryGB = memoryGB
	info.DiskGB = diskGB
	return info, nil
}

func parseSystemInfoFields(fields map[string]string) (*systemInfo, error) {
	info := &systemInfo{
		OS:        strings.ToLower(strings.TrimSpace(fields["os"])),
		OSVersion: strings.TrimSpace(fields["os_version"]),
	}
	cpuCores, err := strconv.Atoi(strings.TrimSpace(fields["cpu_cores"]))
	if err != nil {
		return nil, err
	}
	memoryGB, err := strconv.ParseFloat(strings.TrimSpace(fields["memory_gb"]), 64)
	if err != nil {
		return nil, err
	}
	diskGB, err := strconv.ParseFloat(strings.TrimSpace(fields["disk_gb"]), 64)
	if err != nil {
		return nil, err
	}
	if info.OS == "" || info.OSVersion == "" || cpuCores <= 0 || memoryGB <= 0 || diskGB <= 0 {
		return nil, errors.New("incomplete system info values")
	}

	info.CPUCores = cpuCores
	info.MemoryGB = memoryGB
	info.DiskGB = diskGB
	return info, nil
}

type hostGroupSnapshot struct {
	ID         uint64         `gorm:"column:id"`
	ParentID   uint64         `gorm:"column:parent_id"`
	Name       string         `gorm:"column:name"`
	Conditions datatypes.JSON `gorm:"column:conditions"`
}

type hostGroupIndex struct {
	groups     []hostGroupSnapshot
	groupsByID map[uint64]hostGroupSnapshot
}

func hostToResponse(h *Host, groupIndex hostGroupIndex) HostResponse {
	var labels []LabelEntry
	if len(h.LabelValues) > 0 {
		json.Unmarshal(h.LabelValues, &labels)
	}
	if labels == nil {
		labels = []LabelEntry{}
	}
	var components []ComponentEntry
	if len(h.InstalledComponents) > 0 {
		json.Unmarshal(h.InstalledComponents, &components)
	}
	if components == nil {
		components = []ComponentEntry{}
	}
	matchedGroups := resolveMatchedGroups(h.LabelValues, groupIndex)
	return HostResponse{
		ID:                  h.ID,
		Hostname:            h.Hostname,
		IP:                  h.IP,
		SSHPort:             h.SSHPort,
		OS:                  h.OS,
		OSVersion:           h.OSVersion,
		CPUCores:            h.CPUCores,
		MemoryGB:            h.MemoryGB,
		DiskGB:              h.DiskGB,
		LabelValues:         labels,
		InstalledComponents: components,
		MatchedGroups:       matchedGroups,
		MatchedGroupCount:   len(matchedGroups),
		Status:              h.Status,
		BusinessScopeID:     h.BusinessScopeID,
		BusinessScopeCode:   h.BusinessScopeCode,
		BusinessScopeName:   h.BusinessScopeName,
		DeptID:              h.DeptID,
		Owner:               h.Owner,
		Remark:              h.Remark,
		CreatedAt:           h.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           h.UpdatedAt.Format(time.RFC3339),
		CreatedBy:           h.CreatedBy,
		UpdatedBy:           h.UpdatedBy,
	}
}

func (s *HostService) loadGroupIndex() (hostGroupIndex, error) {
	index := hostGroupIndex{
		groups:     []hostGroupSnapshot{},
		groupsByID: map[uint64]hostGroupSnapshot{},
	}
	if s.db == nil {
		return index, errors.New("database.not_initialized")
	}
	var groups []hostGroupSnapshot
	if err := s.db.Table("biz_cmdb_group").Where("deleted_at IS NULL").Order("parent_id ASC, id ASC").Find(&groups).Error; err != nil {
		return index, err
	}
	index.groups = groups
	index.groupsByID = make(map[uint64]hostGroupSnapshot, len(groups))
	for _, group := range groups {
		index.groupsByID[group.ID] = group
	}
	return index, nil
}

func resolveMatchedGroups(labelJSON datatypes.JSON, groupIndex hostGroupIndex) []MatchedGroupEntry {
	if len(groupIndex.groups) == 0 {
		return []MatchedGroupEntry{}
	}
	matched := make([]MatchedGroupEntry, 0)
	for _, group := range groupIndex.groups {
		if !hostGroupConditionChainMatchesLabel(group, groupIndex.groupsByID, labelJSON) {
			continue
		}
		matched = append(matched, MatchedGroupEntry{
			ID:       group.ID,
			ParentID: group.ParentID,
			Name:     group.Name,
			FullPath: hostGroupPath(group, groupIndex.groupsByID),
		})
	}
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].FullPath == matched[j].FullPath {
			return matched[i].ID < matched[j].ID
		}
		return matched[i].FullPath < matched[j].FullPath
	})
	return matched
}

func hostGroupConditionChainMatchesLabel(group hostGroupSnapshot, groupsByID map[uint64]hostGroupSnapshot, labelJSON datatypes.JSON) bool {
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
	for _, conditionJSON := range chain {
		if !hostGroupMatchesLabel(conditionJSON, labelJSON) {
			return false
		}
	}
	return len(chain) > 0
}

func hostGroupMatchesLabel(conditionJSON datatypes.JSON, labelJSON datatypes.JSON) bool {
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
	var labels []LabelEntry
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
			ruleMatched = hostLabelRuleMatches(actual, rule.Op, rule.Val)
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

func hostLabelRuleMatches(actual string, op string, raw string) bool {
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

func hostGroupPath(group hostGroupSnapshot, groupsByID map[uint64]hostGroupSnapshot) string {
	names := []string{group.Name}
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
		names = append([]string{parent.Name}, names...)
		parentID = parent.ParentID
	}
	return strings.Join(names, " / ")
}

func (s *HostService) getBusinessScope(id uint64) (*bizscope.BizScope, error) {
	if id == 0 {
		return nil, errors.New("bizscope.not_found")
	}
	var item bizscope.BizScope
	if err := s.db.Where("id = ?", id).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bizscope.not_found")
		}
		return nil, err
	}
	return &item, nil
}
