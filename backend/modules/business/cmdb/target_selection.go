package cmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
	cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TargetSelectionBudget caps the number of explicit IDs a single selection may resolve.
const TargetSelectionBudget = 10000

// ErrTargetSelectionInvalid reports a malformed or unbounded target selection.
var ErrTargetSelectionInvalid = errors.New("business.cmdb.selection.invalid")

// ErrTargetSelectionConflict reports mutually inconsistent selection inputs.
var ErrTargetSelectionConflict = errors.New("business.cmdb.selection.conflict")

// TargetSelection is a provider-owned selection intent. Unavailable manual entries
// combine unknown and unauthorized assets, so resolution cannot enumerate hidden assets.
type TargetSelection struct {
	BusinessScopeID uint64   `json:"businessScopeId"`
	HostIDs         []uint64 `json:"hostIds"`
	GroupIDs        []uint64 `json:"groupIds"`
	IncludeScope    bool     `json:"includeScope"`
	Manual          []string `json:"manual"`
	ExcludedIDs     []uint64 `json:"excludedIds"`
}

// SelectionHost is the resolved, authorized target host returned by a selection.
type SelectionHost struct {
	ID              uint64 `json:"id"`
	Hostname        string `json:"hostname"`
	IP              string `json:"ip"`
	Status          string `json:"status"`
	StateVersion    uint64 `json:"stateVersion"`
	BusinessScopeID uint64 `json:"businessScopeId"`
	Source          string `json:"source"`
}

// TargetSelectionResult is the outcome of resolving a target selection.
type TargetSelectionResult struct {
	Selection        TargetSelection `json:"selection"`
	ScopeVersion     string          `json:"scopeVersion"`
	ResolvedAt       time.Time       `json:"resolvedAt"`
	Hosts            []SelectionHost `json:"hosts"`
	DuplicateCount   int             `json:"duplicateCount"`
	UnavailableCount int             `json:"unavailableCount"`
	IneligibleCount  int             `json:"ineligibleCount"`
	ExcludedCount    int             `json:"excludedCount"`
	Sources          map[string]int  `json:"sources"`
}

// TargetSelectionCapability resolves and locks target selections within a scope.
type TargetSelectionCapability interface {
	ResolveTargetSelection(context.Context, TargetSelection, *common.DataScopeReq, bool) (*TargetSelectionResult, error)
	WithTargetTransaction(*gorm.DB) TargetSelectionCapability
}

func (c *deployCMDBCapability) WithTargetTransaction(tx *gorm.DB) TargetSelectionCapability {
	reader := c.bizScopeReader
	if owner, ok := reader.(interface {
		WithBizScopeTransaction(*gorm.DB) bizcap.BizScopeReader
	}); ok {
		reader = owner.WithBizScopeTransaction(tx)
	}
	return &deployCMDBCapability{db: tx, bizScopeReader: reader, selectionLock: true}
}

func (c *deployCMDBCapability) ResolveTargetSelection(ctx context.Context, req TargetSelection, scope *common.DataScopeReq, lock bool) (*TargetSelectionResult, error) {
	if c.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if req.BusinessScopeID == 0 || len(req.HostIDs)+len(req.GroupIDs)+len(req.Manual)+len(req.ExcludedIDs) > TargetSelectionBudget {
		return nil, ErrTargetSelectionInvalid
	}
	if c.bizScopeReader == nil {
		return nil, errors.New("business.bizscope.readerNotConfigured")
	}
	scopeRef, err := c.bizScopeReader.GetActive(ctx, req.BusinessScopeID, scope)
	if err != nil {
		return nil, ErrTargetSelectionInvalid
	}
	out := &TargetSelectionResult{Selection: req, ResolvedAt: time.Now().UTC(), Hosts: []SelectionHost{}, Sources: map[string]int{}}
	req.HostIDs = normalizeSelectionIDs(req.HostIDs, &out.DuplicateCount)
	req.GroupIDs = normalizeSelectionIDs(req.GroupIDs, &out.DuplicateCount)
	req.ExcludedIDs = normalizeSelectionIDs(req.ExcludedIDs, nil)
	out.Selection = req
	// Explicit IDs only need their scoped rows. Dynamic group/scope selections and
	// canonical IP ambiguity checks still require the bounded candidate scan.
	// Every path keeps both DataScope and BizScope predicates before row locking.
	query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(scope)).Where(businessScopeIDWhereClause, req.BusinessScopeID).Order(idAscOrder).Limit(TargetSelectionBudget + 1)
	if ids, bounded := selectionCandidateIDs(req); bounded {
		query = query.Where("id IN ?", ids)
	}
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var hosts []cmdbhost.Host
	if err := query.Find(&hosts).Error; err != nil {
		return nil, err
	}
	if len(hosts) > TargetSelectionBudget {
		return nil, ErrTargetSelectionInvalid
	}
	byID := make(map[uint64]cmdbhost.Host, len(hosts))
	byIP := map[string]cmdbhost.Host{}
	ambiguousIP := map[string]bool{}
	for _, host := range hosts {
		byID[host.ID] = host
		if address, err := netip.ParseAddr(strings.TrimSpace(host.IP)); err == nil {
			ip := address.Unmap().String()
			if _, duplicate := byIP[ip]; duplicate {
				ambiguousIP[ip] = true
			}
			byIP[ip] = host
		}
	}
	selected := map[uint64]string{}
	add := func(id uint64, source string) {
		if _, ok := selected[id]; ok {
			out.DuplicateCount++
			return
		}
		selected[id] = source
	}
	for _, id := range req.HostIDs {
		if _, ok := byID[id]; !ok {
			out.UnavailableCount++
			continue
		}
		add(id, "host")
	}
	if req.IncludeScope {
		for _, host := range hosts {
			add(host.ID, "scope")
		}
	}
	// Group metadata remains provider-owned; inherited conditions are included in
	// the revision, so editing a group after preview cannot silently change targets.
	groups, groupMap, err := c.selectionGroups(ctx, req.GroupIDs, lock)
	if err != nil {
		return nil, err
	}
	for _, id := range req.GroupIDs {
		group, ok := groupMap[id]
		valid := ok
		visited := map[uint64]bool{}
		for node := group; valid && node.ID != 0; {
			if visited[node.ID] {
				valid = false
				break
			}
			visited[node.ID] = true
			if node.ParentID == 0 {
				break
			}
			node, valid = groupMap[node.ParentID]
		}
		if !valid {
			out.UnavailableCount++
			continue
		}
		chain := conditionChainFromMap(group, groupMap)
		for _, host := range hosts {
			if groupConditionChainMatchesHost(chain, host.LabelValues) {
				add(host.ID, "group")
			}
		}
	}
	manualSeen := map[string]bool{}
	for _, raw := range req.Manual {
		value := strings.TrimSpace(raw)
		if len(value) > 128 || value == "" {
			out.UnavailableCount++
			continue
		}
		if manualSeen[value] {
			out.DuplicateCount++
			continue
		}
		manualSeen[value] = true
		var host cmdbhost.Host
		var found bool
		if addr, err := netip.ParseAddr(value); err == nil {
			ip := addr.Unmap().String()
			host, found = byIP[ip]
			found = found && !ambiguousIP[ip]
		} else if id, err := strconv.ParseUint(value, 10, 64); err == nil {
			host, found = byID[id]
		}
		if !found {
			out.UnavailableCount++
			continue
		}
		add(host.ID, "manual")
	}
	exclude := map[uint64]bool{}
	for _, id := range req.ExcludedIDs {
		exclude[id] = true
	}
	for _, host := range hosts {
		source, ok := selected[host.ID]
		if !ok {
			continue
		}
		if exclude[host.ID] {
			out.ExcludedCount++
			continue
		}
		if host.Status == "retired" || host.Status == "pending" || host.Status == "unassigned" {
			out.IneligibleCount++
			continue
		}
		out.Hosts = append(out.Hosts, SelectionHost{ID: host.ID, Hostname: host.Hostname, IP: host.IP, Status: host.Status, StateVersion: host.StateVersion, BusinessScopeID: host.BusinessScopeID, Source: source})
		out.Sources[source]++
	}
	// Include full authorized candidate facts, not just stateVersion: existing
	// metadata edits may legitimately keep a lifecycle version unchanged.
	material := struct {
		Selection TargetSelection
		Scope     bizcap.BizScopeRef
		Hosts     []cmdbhost.Host
		Groups    []cmdbgroup.Group
	}{req, scopeRef, hosts, groups}
	bytes, err := json.Marshal(material)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(bytes)
	out.ScopeVersion = hex.EncodeToString(digest[:])
	return out, nil
}

// selectionCandidateIDs narrows explicit-ID selections without changing IP
// canonicalization or dynamic group membership semantics.
func selectionCandidateIDs(req TargetSelection) ([]uint64, bool) {
	if req.IncludeScope || len(req.GroupIDs) > 0 {
		return nil, false
	}
	ids := append([]uint64(nil), req.HostIDs...)
	for _, raw := range req.Manual {
		value := strings.TrimSpace(raw)
		if value == "" || len(value) > 128 {
			continue
		}
		if _, err := netip.ParseAddr(value); err == nil {
			return nil, false
		}
		if id, err := strconv.ParseUint(value, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return normalizeSelectionIDs(ids, nil), true
}

func normalizeSelectionIDs(ids []uint64, duplicates *int) []uint64 {
	seen := map[uint64]bool{}
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if seen[id] {
			if duplicates != nil {
				*duplicates++
			}
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// Groups are a global condition catalog in the existing CMDB contract (no
// ownership columns). Hosts remain scoped. Read only referenced ancestor chains
// so unrelated catalog edits cannot invalidate a selection or acquire its locks.
func (c *deployCMDBCapability) selectionGroups(ctx context.Context, ids []uint64, lock bool) ([]cmdbgroup.Group, map[uint64]cmdbgroup.Group, error) {
	groups := []cmdbgroup.Group{}
	byID := map[uint64]cmdbgroup.Group{}
	seen := map[uint64]bool{}
	pending := ids
	for depth := 0; len(pending) > 0; depth++ {
		if depth >= 128 {
			return nil, nil, ErrTargetSelectionInvalid
		}
		queryIDs := []uint64{}
		for _, id := range pending {
			if id != 0 && !seen[id] {
				seen[id] = true
				queryIDs = append(queryIDs, id)
			}
		}
		if len(queryIDs) == 0 {
			break
		}
		if len(seen) > TargetSelectionBudget {
			return nil, nil, ErrTargetSelectionInvalid
		}
		query := c.db.WithContext(ctx).Where(idInWhereClause, queryIDs).Order(idAscOrder)
		if lock {
			query = query.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var rows []cmdbgroup.Group
		if err := query.Find(&rows).Error; err != nil {
			return nil, nil, err
		}
		pending = nil
		for _, group := range rows {
			byID[group.ID] = group
			groups = append(groups, group)
			if group.ParentID != 0 {
				pending = append(pending, group.ParentID)
			}
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
	return groups, byID, nil
}
