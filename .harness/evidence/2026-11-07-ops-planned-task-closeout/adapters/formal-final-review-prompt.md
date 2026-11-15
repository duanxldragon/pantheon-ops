Read-only independent final static review for formal Pantheon Ops planned task closeout. Do not use tools, modify files, or run commands. Review supplied code only. Report concrete P0/P1/P2 defects with source lines and minimal fixes. Distinguish assumptions from proved defects. Scope: CMDB/Deploy target selection + preview/start cancellation/auth/version/redaction; Base permission registry grant/revoke; optional Dashboard provider lifecycle and budgets; consumer archive registry projection. Base Group is a global conditions directory with no business scope ownership; host reads are always DataScope/BizScope scoped. Existing taskVisible loads authorized targets before update. Background execution must continue after a successfully committed HTTP request. Base changes are un-released; Ops remains locked to official v0.18.6. External and production evidence are explicitly missing. Passing static review does not approve production or close full task scope.

FILE pantheon-ops/backend/modules/business/cmdb/target_selection.go
1: package cmdb
2:
3: import (
4: "context"
5: "crypto/sha256"
6: "encoding/hex"
7: "encoding/json"
8: "errors"
9: "net/netip"
10: "sort"
11: "strconv"
12: "strings"
13: "time"
14:
15: bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
16: cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
17: cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
18: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
19: "github.com/duanxldragon/pantheon-base/backend/pkg/database"
20: "gorm.io/gorm"
21: "gorm.io/gorm/clause"
22: )
23:
24: const TargetSelectionBudget = 10000
25:
26: var ErrTargetSelectionInvalid = errors.New("business.cmdb.selection.invalid")
27: var ErrTargetSelectionConflict = errors.New("business.cmdb.selection.conflict")
28:
29: // TargetSelection is a provider-owned selection intent. Unavailable manual entries
30: // combine unknown and unauthorized assets, so resolution cannot enumerate hidden assets.
31: type TargetSelection struct {
32: BusinessScopeID uint64 `json:"businessScopeId"`
33: HostIDs []uint64 `json:"hostIds"`
34: GroupIDs []uint64 `json:"groupIds"`
35: IncludeScope bool `json:"includeScope"`
36: Manual []string `json:"manual"`
37: ExcludedIDs []uint64 `json:"excludedIds"`
38: }
39: type SelectionHost struct {
40: ID uint64 `json:"id"`
41: Hostname string `json:"hostname"`
42: IP string `json:"ip"`
43: Status string `json:"status"`
44: StateVersion uint64 `json:"stateVersion"`
45: BusinessScopeID uint64 `json:"businessScopeId"`
46: Source string `json:"source"`
47: }
48: type TargetSelectionResult struct {
49: Selection TargetSelection `json:"selection"`
50: ScopeVersion string `json:"scopeVersion"`
51: ResolvedAt time.Time `json:"resolvedAt"`
52: Hosts []SelectionHost `json:"hosts"`
53: DuplicateCount int `json:"duplicateCount"`
54: UnavailableCount int `json:"unavailableCount"`
55: IneligibleCount int `json:"ineligibleCount"`
56: ExcludedCount int `json:"excludedCount"`
57: Sources map[string]int `json:"sources"`
58: }
59: type TargetSelectionCapability interface {
60: ResolveTargetSelection(context.Context, TargetSelection, *common.DataScopeReq, bool) (*TargetSelectionResult, error)
61: WithTargetTransaction(*gorm.DB) TargetSelectionCapability
62: }
63:
64: func (c *deployCMDBCapability) WithTargetTransaction(tx *gorm.DB) TargetSelectionCapability {
65: reader := c.bizScopeReader
66: if owner, ok := reader.(interface {
67: WithBizScopeTransaction(*gorm.DB) bizcap.BizScopeReader
68: }); ok {
69: reader = owner.WithBizScopeTransaction(tx)
70: }
71: return &deployCMDBCapability{db: tx, bizScopeReader: reader, selectionLock: true}
72: }
73:
74: func (c *deployCMDBCapability) ResolveTargetSelection(ctx context.Context, req TargetSelection, scope *common.DataScopeReq, lock bool) (*TargetSelectionResult, error) {
75: if c.db == nil {
76: return nil, errors.New("database.not_initialized")
77: }
78: if req.BusinessScopeID == 0 || len(req.HostIDs)+len(req.GroupIDs)+len(req.Manual)+len(req.ExcludedIDs) > TargetSelectionBudget {
79: return nil, ErrTargetSelectionInvalid
80: }
81: if c.bizScopeReader == nil {
82: return nil, errors.New("business.bizscope.readerNotConfigured")
83: }
84: scopeRef, err := c.bizScopeReader.GetActive(ctx, req.BusinessScopeID, scope)
85: if err != nil {
86: return nil, ErrTargetSelectionInvalid
87: }
88: out := &TargetSelectionResult{Selection: req, ResolvedAt: time.Now().UTC(), Hosts: []SelectionHost{}, Sources: map[string]int{}}
89: req.HostIDs = normalizeSelectionIDs(req.HostIDs, &out.DuplicateCount)
90: req.GroupIDs = normalizeSelectionIDs(req.GroupIDs, &out.DuplicateCount)
91: req.ExcludedIDs = normalizeSelectionIDs(req.ExcludedIDs, nil)
92: out.Selection = req
93: // A bounded provider scan also discovers group/topology changes. Only authorized
94: // assets in the selected business scope enter memory or revision material.
95: query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(scope)).Where(businessScopeIDWhereClause, req.BusinessScopeID).Order(idAscOrder).Limit(TargetSelectionBudget + 1)
96: if lock {
97: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
98: }
99: var hosts []cmdbhost.Host
100: if err := query.Find(&hosts).Error; err != nil {
101: return nil, err
102: }
103: if len(hosts) > TargetSelectionBudget {
104: return nil, ErrTargetSelectionInvalid
105: }
106: byID := make(map[uint64]cmdbhost.Host, len(hosts))
107: byIP := map[string]cmdbhost.Host{}
108: ambiguousIP := map[string]bool{}
109: for _, host := range hosts {
110: byID[host.ID] = host
111: if address, err := netip.ParseAddr(strings.TrimSpace(host.IP)); err == nil {
112: ip := address.Unmap().String()
113: if _, duplicate := byIP[ip]; duplicate {
114: ambiguousIP[ip] = true
115: }
116: byIP[ip] = host
117: }
118: }
119: selected := map[uint64]string{}
120: add := func(id uint64, source string) {
121: if _, ok := selected[id]; ok {
122: out.DuplicateCount++
123: return
124: }
125: selected[id] = source
126: }
127: for _, id := range req.HostIDs {
128: if _, ok := byID[id]; !ok {
129: out.UnavailableCount++
130: continue
131: }
132: add(id, "host")
133: }
134: if req.IncludeScope {
135: for _, host := range hosts {
136: add(host.ID, "scope")
137: }
138: }
139: // Group metadata remains provider-owned; inherited conditions are included in
140: // the revision, so editing a group after preview cannot silently change targets.
141: groups, groupMap, err := c.selectionGroups(ctx, req.GroupIDs, lock)
142: if err != nil {
143: return nil, err
144: }
145: for _, id := range req.GroupIDs {
146: group, ok := groupMap[id]
147: valid := ok
148: visited := map[uint64]bool{}
149: for node := group; valid && node.ID != 0; {
150: if visited[node.ID] {
151: valid = false
152: break
153: }
154: visited[node.ID] = true
155: if node.ParentID == 0 {
156: break
157: }
158: node, valid = groupMap[node.ParentID]
159: }
160: if !valid {
161: out.UnavailableCount++
162: continue
163: }
164: chain := conditionChainFromMap(group, groupMap)
165: for _, host := range hosts {
166: if groupConditionChainMatchesHost(chain, host.LabelValues) {
167: add(host.ID, "group")
168: }
169: }
170: }
171: manualSeen := map[string]bool{}
172: for _, raw := range req.Manual {
173: value := strings.TrimSpace(raw)
174: if len(value) > 128 || value == "" {
175: out.UnavailableCount++
176: continue
177: }
178: if manualSeen[value] {
179: out.DuplicateCount++
180: continue
181: }
182: manualSeen[value] = true
183: var host cmdbhost.Host
184: var found bool
185: if addr, err := netip.ParseAddr(value); err == nil {
186: ip := addr.Unmap().String()
187: host, found = byIP[ip]
188: found = found && !ambiguousIP[ip]
189: } else if id, err := strconv.ParseUint(value, 10, 64); err == nil {
190: host, found = byID[id]
191: }
192: if !found {
193: out.UnavailableCount++
194: continue
195: }
196: add(host.ID, "manual")
197: }
198: exclude := map[uint64]bool{}
199: for _, id := range req.ExcludedIDs {
200: exclude[id] = true
201: }
202: for _, host := range hosts {
203: source, ok := selected[host.ID]
204: if !ok {
205: continue
206: }
207: if exclude[host.ID] {
208: out.ExcludedCount++
209: continue
210: }
211: if host.Status == "retired" || host.Status == "pending" || host.Status == "unassigned" {
212: out.IneligibleCount++
213: continue
214: }
215: out.Hosts = append(out.Hosts, SelectionHost{ID: host.ID, Hostname: host.Hostname, IP: host.IP, Status: host.Status, StateVersion: host.StateVersion, BusinessScopeID: host.BusinessScopeID, Source: source})
216: out.Sources[source]++
217: }
218: // Include full authorized scoped host facts, not just stateVersion: existing
219: // metadata edits may legitimately keep a lifecycle version unchanged.
220: material := struct {
221: Selection TargetSelection
222: Scope bizcap.BizScopeRef
223: Hosts []cmdbhost.Host
224: Groups []cmdbgroup.Group
225: }{req, scopeRef, hosts, groups}
226: bytes, err := json.Marshal(material)
227: if err != nil {
228: return nil, err
229: }
230: digest := sha256.Sum256(bytes)
231: out.ScopeVersion = hex.EncodeToString(digest[:])
232: return out, nil
233: }
234: func normalizeSelectionIDs(ids []uint64, duplicates *int) []uint64 {
235: seen := map[uint64]bool{}
236: result := make([]uint64, 0, len(ids))
237: for _, id := range ids {
238: if id == 0 {
239: continue
240: }
241: if seen[id] {
242: if duplicates != nil {
243: *duplicates++
244: }
245: continue
246: }
247: seen[id] = true
248: result = append(result, id)
249: }
250: sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
251: return result
252: }
253:
254: // Groups are a global condition catalog in the existing CMDB contract (no
255: // ownership columns). Hosts remain scoped. Read only referenced ancestor chains
256: // so unrelated catalog edits cannot invalidate a selection or acquire its locks.
257: func (c *deployCMDBCapability) selectionGroups(ctx context.Context, ids []uint64, lock bool) ([]cmdbgroup.Group, map[uint64]cmdbgroup.Group, error) {
258: groups := []cmdbgroup.Group{}
259: byID := map[uint64]cmdbgroup.Group{}
260: seen := map[uint64]bool{}
261: pending := ids
262: for depth := 0; len(pending) > 0; depth++ {
263: if depth >= 128 {
264: return nil, nil, ErrTargetSelectionInvalid
265: }
266: queryIDs := []uint64{}
267: for _, id := range pending {
268: if id != 0 && !seen[id] {
269: seen[id] = true
270: queryIDs = append(queryIDs, id)
271: }
272: }
273: if len(queryIDs) == 0 {
274: break
275: }
276: if len(seen) > TargetSelectionBudget {
277: return nil, nil, ErrTargetSelectionInvalid
278: }
279: query := c.db.WithContext(ctx).Where(idInWhereClause, queryIDs).Order(idAscOrder)
280: if lock {
281: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
282: }
283: var rows []cmdbgroup.Group
284: if err := query.Find(&rows).Error; err != nil {
285: return nil, nil, err
286: }
287: pending = nil
288: for _, group := range rows {
289: byID[group.ID] = group
290: groups = append(groups, group)
291: if group.ParentID != 0 {
292: pending = append(pending, group.ParentID)
293: }
294: }
295: }
296: sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
297: return groups, byID, nil
298: }

FILE pantheon-ops/backend/modules/business/cmdb/capability.go
1: package cmdb
2:
3: import (
4: "context"
5: "encoding/json"
6: "errors"
7: "fmt"
8: "strings"
9: "time"
10:
11: bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
12: cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
13: cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
14: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
15: "github.com/duanxldragon/pantheon-base/backend/pkg/database"
16:
17: "gorm.io/datatypes"
18: "gorm.io/gorm"
19: "gorm.io/gorm/clause"
20: )
21:
22: const (
23: idInWhereClause = "id IN ?"
24: businessScopeIDWhereClause = "business_scope_id = ?"
25: )
26:
27: type DeployHostTarget struct {
28: StateVersion uint64
29: UpdatedAt time.Time
30: InstalledComponents datatypes.JSON
31: ID uint64
32: Hostname string
33: IP string
34: SSHPort int
35: OS string
36: Status string
37: BusinessScopeID uint64
38: BusinessScopeName string
39: LabelValues datatypes.JSON
40: DeptID uint64
41: }
42:
43: type InstalledComponentUpsert struct {
44: Name string
45: Version string
46: DeployedAt time.Time
47: DeployTaskID uint64
48: DeployTaskName string
49: ExecutorType string
50: }
51:
52: type DeployHostResolveRequest struct {
53: BusinessScopeID uint64
54: TargetType string
55: TargetIDs []uint64
56: DataScope *common.DataScopeReq
57: }
58:
59: type DeployHostWritebackRequest struct {
60: HostID uint64
61: Status string
62: Actor string
63: InstalledComponents []InstalledComponentUpsert
64: RemovedComponentNames []string
65: }
66:
67: type DeployCMDBCapability interface {
68: bizcap.CMDBHostReader
69: bizcap.CMDBOwnershipCommand
70: ResolveDeployTargets(req DeployHostResolveRequest) ([]DeployHostTarget, error)
71: WriteDeployHostResult(req DeployHostWritebackRequest) error
72: }
73:
74: type deployCMDBCapability struct {
75: db *gorm.DB
76: bizScopeReader bizcap.BizScopeReader
77: selectionLock bool
78: }
79:
80: // NewDeployCMDBCapability creates the CMDB capability consumed by deploy and bizscope.
81: func NewDeployCMDBCapability(db *gorm.DB, readers ...bizcap.BizScopeReader) *deployCMDBCapability {
82: capability := &deployCMDBCapability{db: db}
83: if len(readers) > 0 {
84: capability.bizScopeReader = readers[0]
85: }
86: return capability
87: }
88:
89: func (c *deployCMDBCapability) SetBizScopeReader(reader bizcap.BizScopeReader) {
90: c.bizScopeReader = reader
91: }
92:
93: var _ bizcap.CMDBHostReader = (*deployCMDBCapability)(nil)
94: var _ bizcap.CMDBOwnershipCommand = (*deployCMDBCapability)(nil)
95:
96: func (c *deployCMDBCapability) GetByIDs(ctx context.Context, req bizcap.HostIDsQuery) (bizcap.HostPage, error) {
97: if c.db == nil {
98: return bizcap.HostPage{}, errors.New("database.not_initialized")
99: }
100: ids := common.NormalizeUint64IDs(req.HostIDs)
101: if len(ids) == 0 {
102: return bizcap.HostPage{Items: []bizcap.HostRef{}}, nil
103: }
104: query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).
105: Scopes(database.WithDataScope(req.DataScope)).
106: Where(idInWhereClause, ids)
107: var rows []cmdbhost.Host
108: if err := query.Order("id ASC").Find(&rows).Error; err != nil {
109: return bizcap.HostPage{}, err
110: }
111: return bizcap.HostPage{Items: mapCapabilityHostRefs(rows), Total: int64(len(rows))}, nil
112: }
113:
114: func (c *deployCMDBCapability) ResolveDeployTargets(req DeployHostResolveRequest) ([]DeployHostTarget, error) {
115: if c.db == nil {
116: return nil, errors.New("database.not_initialized")
117: }
118: targetIDs := common.NormalizeUint64IDs(req.TargetIDs)
119: if len(targetIDs) == 0 {
120: return []DeployHostTarget{}, nil
121: }
122: switch strings.TrimSpace(req.TargetType) {
123: case "host":
124: return c.resolveHostTargets(req.BusinessScopeID, targetIDs, req.DataScope)
125: case "group":
126: return c.resolveGroupTargets(targetIDs, req.DataScope)
127: default:
128: return nil, errors.New("business.deploy.task.invalidTargetType")
129: }
130: }
131:
132: func (c *deployCMDBCapability) WriteDeployHostResult(req DeployHostWritebackRequest) error {
133: if c.db == nil {
134: return errors.New("database.not_initialized")
135: }
136: now := time.Now()
137: status := strings.TrimSpace(req.Status)
138:
139: return c.db.Transaction(func(tx *gorm.DB) error {
140: if len(req.InstalledComponents) == 0 && len(req.RemovedComponentNames) == 0 {
141: updates := map[string]any{
142: "updated_by": req.Actor,
143: "updated_at": now,
144: }
145: if status != "" {
146: updates["status"] = status
147: }
148: return tx.Table("biz_cmdb_host").Where(idWhereClause, req.HostID).Updates(updates).Error
149: }
150: components, err := loadInstalledComponents(tx, req.HostID)
151: if err != nil {
152: return err
153: }
154: components = removeComponentsByName(components, req.RemovedComponentNames)
155: components = upsertComponents(components, req.InstalledComponents)
156: status = reconcileHostStatus(status, len(components))
157: payload, _ := json.Marshal(components)
158: updates := map[string]any{
159: "installed_components": datatypes.JSON(payload),
160: "updated_by": req.Actor,
161: "updated_at": now,
162: }
163: if status != "" {
164: updates["status"] = status
165: }
166: return tx.Table("biz_cmdb_host").Where(idWhereClause, req.HostID).Updates(updates).Error
167: })
168: }
169:
170: func (c *deployCMDBCapability) ListByBusinessScope(ctx context.Context, req bizcap.HostScopeQuery) (bizcap.HostPage, error) {
171: if c.db == nil {
172: return bizcap.HostPage{}, errors.New("database.not_initialized")
173: }
174: query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).
175: Scopes(database.WithDataScope(req.DataScope)).
176: Where(businessScopeIDWhereClause, req.BusinessScopeID)
177: var total int64
178: if err := query.Count(&total).Error; err != nil {
179: return bizcap.HostPage{}, err
180: }
181: var rows []cmdbhost.Host
182: if err := query.Order("id DESC").Find(&rows).Error; err != nil {
183: return bizcap.HostPage{}, err
184: }
185: return bizcap.HostPage{Items: mapCapabilityHostRefs(rows), Total: total}, nil
186: }
187:
188: func (c *deployCMDBCapability) ListAvailable(ctx context.Context, req bizcap.AvailableHostQuery) (bizcap.HostPage, error) {
189: if c.db == nil {
190: return bizcap.HostPage{}, errors.New("database.not_initialized")
191: }
192: query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).
193: Scopes(database.WithDataScope(req.DataScope)).
194: Where("(business_scope_id = 0 OR business_scope_id IS NULL)")
195: var total int64
196: if err := query.Count(&total).Error; err != nil {
197: return bizcap.HostPage{}, err
198: }
199: var rows []cmdbhost.Host
200: if err := query.Order("id DESC").Find(&rows).Error; err != nil {
201: return bizcap.HostPage{}, err
202: }
203: return bizcap.HostPage{Items: mapCapabilityHostRefs(rows), Total: total}, nil
204: }
205:
206: func (c *deployCMDBCapability) HasBusinessScopeReferences(ctx context.Context, businessScopeID uint64) (bool, error) {
207: if c.db == nil {
208: return false, errors.New("database.not_initialized")
209: }
210: var count int64
211: if err := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).Where(businessScopeIDWhereClause, businessScopeID).Count(&count).Error; err != nil {
212: return false, err
213: }
214: return count > 0, nil
215: }
216:
217: func (c *deployCMDBCapability) Bind(ctx context.Context, req bizcap.BindOwnershipRequest) error {
218: if c.db == nil {
219: return errors.New("database.not_initialized")
220: }
221: hostIDs := common.NormalizeUint64IDs(req.HostIDs)
222: if len(hostIDs) == 0 {
223: return errors.New("param.invalid")
224: }
225: if req.BusinessScopeID == 0 {
226: return errors.New("business.bizscope.notFound")
227: }
228: return c.WithBusinessScopeOwnershipLock(ctx, req.BusinessScopeID, func() error {
229: if c.bizScopeReader == nil {
230: return errors.New("business.bizscope.readerNotConfigured")
231: }
232: if _, err := c.bizScopeReader.GetActive(ctx, req.BusinessScopeID, req.DataScope); err != nil {
233: return err
234: }
235: return c.bindUnlocked(ctx, req, hostIDs)
236: })
237: }
238:
239: func (c *deployCMDBCapability) bindUnlocked(ctx context.Context, req bizcap.BindOwnershipRequest, hostIDs []uint64) error {
240: return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
241: var found int64
242: if err := tx.Model(&cmdbhost.Host{}).
243: Scopes(database.WithDataScope(req.DataScope)).
244: Where(idInWhereClause, hostIDs).
245: Count(&found).Error; err != nil {
246: return err
247: }
248: if found != int64(len(hostIDs)) {
249: return errors.New("cmdbhost.not_found")
250: }
251: var owned int64
252: if err := tx.Model(&cmdbhost.Host{}).
253: Scopes(database.WithDataScope(req.DataScope)).
254: Where(idInWhereClause, hostIDs).
255: Where("business_scope_id <> 0").
256: Count(&owned).Error; err != nil {
257: return err
258: }
259: if owned > 0 {
260: return errors.New("cmdbhost.ownership_conflict")
261: }
262: updates := map[string]any{
263: "business_scope_id": req.BusinessScopeID,
264: "business_scope_code": strings.TrimSpace(req.BusinessScopeCode),
265: "business_scope_name": strings.TrimSpace(req.BusinessScopeName),
266: "status": "assigned",
267: "updated_by": strings.TrimSpace(req.Actor),
268: "updated_at": time.Now(),
269: }
270: result := tx.Model(&cmdbhost.Host{}).
271: Scopes(database.WithDataScope(req.DataScope)).
272: Where(idInWhereClause, hostIDs).
273: Where("business_scope_id = 0").
274: Updates(updates)
275: if result.Error != nil {
276: return result.Error
277: }
278: if result.RowsAffected == int64(len(hostIDs)) {
279: return nil
280: }
281: return classifyOwnershipBindFailure(tx, req.DataScope, hostIDs)
282: })
283: }
284:
285: func (c *deployCMDBCapability) WithBusinessScopeOwnershipLock(ctx context.Context, businessScopeID uint64, action func() error) error {
286: if c.db == nil {
287: return errors.New("database.not_initialized")
288: }
289: if businessScopeID == 0 {
290: return errors.New("business.bizscope.notFound")
291: }
292: if c.db.Name() != "mysql" {
293: return action()
294: }
295: lockName := fmt.Sprintf("pantheon:bizscope:%d", businessScopeID)
296: return c.db.WithContext(ctx).Connection(func(conn *gorm.DB) error {
297: var acquired int
298: if err := conn.Raw("SELECT GET_LOCK(?, 10)", lockName).Scan(&acquired).Error; err != nil {
299: return err
300: }
301: if acquired != 1 {
302: return errors.New("cmdbhost.ownership_lock_timeout")
303: }
304: defer func() {
305: _ = conn.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
306: }()
307: return action()
308: })
309: }
310:
311: func (c *deployCMDBCapability) Unbind(ctx context.Context, req bizcap.UnbindOwnershipRequest) error {
312: if c.db == nil {
313: return errors.New("database.not_initialized")
314: }
315: return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
316: updates := map[string]any{
317: "business_scope_id": uint64(0),
318: "business_scope_code": "",
319: "business_scope_name": "",
320: "status": gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", "assigned", "pending"),
321: "updated_by": strings.TrimSpace(req.Actor),
322: "updated_at": time.Now(),
323: }
324: result := tx.Model(&cmdbhost.Host{}).
325: Scopes(database.WithDataScope(req.DataScope)).
326: Where("id = ? AND business_scope_id = ?", req.HostID, req.BusinessScopeID).
327: Updates(updates)
328: if result.Error != nil {
329: return result.Error
330: }
331: if result.RowsAffected == 0 {
332: return classifyOwnershipUnbindFailure(tx, req)
333: }
334: return nil
335: })
336: }
337:
338: func classifyOwnershipBindFailure(tx *gorm.DB, dataScope *common.DataScopeReq, hostIDs []uint64) error {
339: var found int64
340: if err := tx.Model(&cmdbhost.Host{}).
341: Scopes(database.WithDataScope(dataScope)).
342: Where(idInWhereClause, hostIDs).
343: Count(&found).Error; err != nil {
344: return err
345: }
346: if found != int64(len(hostIDs)) {
347: return errors.New("cmdbhost.not_found")
348: }
349: return errors.New("cmdbhost.ownership_conflict")
350: }
351:
352: func classifyOwnershipUnbindFailure(tx *gorm.DB, req bizcap.UnbindOwnershipRequest) error {
353: var found int64
354: if err := tx.Model(&cmdbhost.Host{}).
355: Scopes(database.WithDataScope(req.DataScope)).
356: Where("id = ?", req.HostID).
357: Count(&found).Error; err != nil {
358: return err
359: }
360: if found == 0 {
361: return errors.New("cmdbhost.not_found")
362: }
363: return errors.New("cmdbhost.ownership_conflict")
364: }
365:
366: func loadInstalledComponents(tx *gorm.DB, hostID uint64) ([]cmdbhost.ComponentEntry, error) {
367: var snapshot struct {
368: InstalledComponents datatypes.JSON `gorm:"column:installed_components"`
369: }
370: if err := tx.Table("biz_cmdb_host").Select("installed_components").Where(idWhereClause, hostID).Take(&snapshot).Error; err != nil {
371: return nil, err
372: }
373: var components []cmdbhost.ComponentEntry
374: if len(snapshot.InstalledComponents) > 0 {
375: _ = json.Unmarshal(snapshot.InstalledComponents, &components)
376: }
377: if components == nil {
378: components = []cmdbhost.ComponentEntry{}
379: }
380: return components, nil
381: }
382:
383: func mapCapabilityHostRefs(rows []cmdbhost.Host) []bizcap.HostRef {
384: items := make([]bizcap.HostRef, 0, len(rows))
385: for _, row := range rows {
386: items = append(items, bizcap.HostRef{
387: ID: row.ID,
388: Hostname: row.Hostname,
389: IP: row.IP,
390: SSHPort: row.SSHPort,
391: OS: row.OS,
392: Status: row.Status,
393: BusinessScopeID: row.BusinessScopeID,
394: BusinessScopeCode: row.BusinessScopeCode,
395: BusinessScopeName: row.BusinessScopeName,
396: LabelValues: row.LabelValues,
397: DeptID: row.DeptID,
398: })
399: }
400: return items
401: }
402:
403: func removeComponentsByName(components []cmdbhost.ComponentEntry, names []string) []cmdbhost.ComponentEntry {
404: if len(names) == 0 {
405: return components
406: }
407: removed := make(map[string]struct{}, len(names))
408: for _, name := range names {
409: if trimmed := strings.TrimSpace(name); trimmed != "" {
410: removed[strings.ToLower(trimmed)] = struct{}{}
411: }
412: }
413: filtered := make([]cmdbhost.ComponentEntry, 0, len(components))
414: for _, component := range components {
415: if _, ok := removed[strings.ToLower(strings.TrimSpace(component.Name))]; ok {
416: continue
417: }
418: filtered = append(filtered, component)
419: }
420: return filtered
421: }
422:
423: func upsertComponents(components []cmdbhost.ComponentEntry, items []InstalledComponentUpsert) []cmdbhost.ComponentEntry {
424: for _, item := range items {
425: updated := false
426: for index := range components {
427: if strings.EqualFold(strings.TrimSpace(components[index].Name), strings.TrimSpace(item.Name)) {
428: components[index].Version = item.Version
429: components[index].DeployedAt = item.DeployedAt.Format(time.RFC3339)
430: components[index].DeployTaskID = item.DeployTaskID
431: components[index].DeployTaskName = item.DeployTaskName
432: components[index].ExecutorType = item.ExecutorType
433: updated = true
434: break
435: }
436: }
437: if updated {
438: continue
439: }
440: components = append(components, cmdbhost.ComponentEntry{
441: Name: item.Name,
442: Version: item.Version,
443: DeployedAt: item.DeployedAt.Format(time.RFC3339),
444: DeployTaskID: item.DeployTaskID,
445: DeployTaskName: item.DeployTaskName,
446: ExecutorType: item.ExecutorType,
447: })
448: }
449: return components
450: }
451:
452: func reconcileHostStatus(status string, componentCount int) string {
453: if status == "assigned" && componentCount > 0 {
454: return "online"
455: }
456: if status == "online" && componentCount == 0 {
457: return "assigned"
458: }
459: return status
460: }
461:
462: func (c *deployCMDBCapability) resolveHostTargets(businessScopeID uint64, targetIDs []uint64, dataScope *common.DataScopeReq) ([]DeployHostTarget, error) {
463: query := c.db.Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(dataScope)).Where(idInWhereClause, targetIDs)
464: if businessScopeID > 0 {
465: query = query.Where(businessScopeIDWhereClause, businessScopeID)
466: }
467: if c.selectionLock {
468: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
469: }
470: var rows []cmdbhost.Host
471: if err := query.Order(idAscOrder).Find(&rows).Error; err != nil {
472: return nil, err
473: }
474: items := make([]DeployHostTarget, 0, len(rows))
475: for _, row := range rows {
476: items = append(items, DeployHostTarget{
477: StateVersion: row.StateVersion, UpdatedAt: row.UpdatedAt, InstalledComponents: row.InstalledComponents,
478: ID: row.ID,
479: Hostname: row.Hostname,
480: IP: row.IP,
481: SSHPort: row.SSHPort,
482: OS: row.OS,
483: Status: row.Status,
484: BusinessScopeID: row.BusinessScopeID,
485: BusinessScopeName: row.BusinessScopeName,
486: LabelValues: row.LabelValues,
487: DeptID: row.DeptID,
488: })
489: }
490: return items, nil
491: }
492:
493: func (c *deployCMDBCapability) resolveGroupTargets(targetIDs []uint64, dataScope *common.DataScopeReq) ([]DeployHostTarget, error) {
494: var selectedGroups []cmdbgroup.Group
495: if err := c.db.Where(idInWhereClause, targetIDs).Find(&selectedGroups).Error; err != nil {
496: return nil, err
497: }
498: var allGroups []cmdbgroup.Group
499: if err := c.db.Order(idAscOrder).Find(&allGroups).Error; err != nil {
500: return nil, err
501: }
502: groupsByID := make(map[uint64]cmdbgroup.Group, len(allGroups))
503: for _, group := range allGroups {
504: groupsByID[group.ID] = group
505: }
506: var hosts []cmdbhost.Host
507: hostQuery := c.db.Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(dataScope)).Order(idAscOrder)
508: if c.selectionLock {
509: hostQuery = hostQuery.Clauses(clause.Locking{Strength: "UPDATE"})
510: }
511: if err := hostQuery.Find(&hosts).Error; err != nil {
512: return nil, err
513: }
514: result := make([]DeployHostTarget, 0)
515: seen := make(map[uint64]struct{})
516: for _, host := range hosts {
517: for _, group := range selectedGroups {
518: if groupConditionChainMatchesHost(conditionChainFromMap(group, groupsByID), host.LabelValues) {
519: if _, ok := seen[host.ID]; ok {
520: break
521: }
522: seen[host.ID] = struct{}{}
523: result = append(result, DeployHostTarget{
524: StateVersion: host.StateVersion, UpdatedAt: host.UpdatedAt, InstalledComponents: host.InstalledComponents,
525: ID: host.ID,
526: Hostname: host.Hostname,
527: IP: host.IP,
528: SSHPort: host.SSHPort,
529: OS: host.OS,
530: Status: host.Status,
531: BusinessScopeID: host.BusinessScopeID,
532: BusinessScopeName: host.BusinessScopeName,
533: LabelValues: host.LabelValues,
534: DeptID: host.DeptID,
535: })
536: break
537: }
538: }
539: }
540: return result, nil
541: }
542:
543: func conditionChainFromMap(group cmdbgroup.Group, groupsByID map[uint64]cmdbgroup.Group) []datatypes.JSON {
544: chain := []datatypes.JSON{group.Conditions}
545: visited := map[uint64]struct{}{group.ID: {}}
546: parentID := group.ParentID
547: for parentID != 0 {
548: if _, ok := visited[parentID]; ok {
549: break
550: }
551: parent, ok := groupsByID[parentID]
552: if !ok {
553: break
554: }
555: visited[parent.ID] = struct{}{}
556: chain = append([]datatypes.JSON{parent.Conditions}, chain...)
557: parentID = parent.ParentID
558: }
559: return chain
560: }
561:
562: func groupConditionChainMatchesHost(conditionChain []datatypes.JSON, labelJSON datatypes.JSON) bool {
563: for _, conditionJSON := range conditionChain {
564: if !groupMatchesHost(conditionJSON, labelJSON) {
565: return false
566: }
567: }
568: return len(conditionChain) > 0
569: }
570:
571: func groupMatchesHost(conditionJSON, labelJSON datatypes.JSON) bool {
572: var condition struct {
573: Operator string `json:"operator"`
574: Rules []struct {
575: Key string `json:"key"`
576: Op string `json:"op"`
577: Val string `json:"val"`
578: } `json:"rules"`
579: }
580: if err := json.Unmarshal(conditionJSON, &condition); err != nil || len(condition.Rules) == 0 {
581: return false
582: }
583: var labels []cmdbhost.LabelEntry
584: _ = json.Unmarshal(labelJSON, &labels)
585: labelMap := make(map[string]string, len(labels))
586: for _, label := range labels {
587: labelMap[label.Key] = label.Val
588: }
589: operator := strings.ToUpper(strings.TrimSpace(condition.Operator))
590: if operator == "" {
591: operator = "AND"
592: }
593: matched := operator == "AND"
594: for _, rule := range condition.Rules {
595: actual, ok := labelMap[rule.Key]
596: ruleMatched := false
597: if ok {
598: ruleMatched = labelRuleMatches(actual, rule.Op, rule.Val)
599: }
600: if operator == "OR" && ruleMatched {
601: return true
602: }
603: if operator == "AND" && !ruleMatched {
604: return false
605: }
606: matched = ruleMatched
607: }
608: return matched
609: }
610:
611: func labelRuleMatches(actual, op, raw string) bool {
612: switch strings.TrimSpace(op) {
613: case "eq":
614: return actual == raw
615: case "neq":
616: return actual != raw
617: case "in":
618: for _, item := range strings.Split(raw, ",") {
619: if actual == strings.TrimSpace(item) {
620: return true
621: }
622: }
623: return false
624: case "notIn":
625: for _, item := range strings.Split(raw, ",") {
626: if actual == strings.TrimSpace(item) {
627: return false
628: }
629: }
630: return true
631: default:
632: return false
633: }
634: }

FILE pantheon-ops/backend/modules/business/deploy/target_selection.go
1: package deploy
2:
3: import (
4: "context"
5: "encoding/json"
6: "errors"
7: bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
8: "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
9: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
10: "gorm.io/datatypes"
11: "gorm.io/gorm"
12: "gorm.io/gorm/clause"
13: "time"
14: )
15:
16: func (s *DeployService) ResolveTargetSelection(ctx context.Context, req cmdb.TargetSelection, scope *common.DataScopeReq) (*cmdb.TargetSelectionResult, error) {
17: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
18: if !ok {
19: return nil, errors.New("business.cmdb.selection.notConfigured")
20: }
21: return provider.ResolveTargetSelection(ctx, req, scope, false)
22: }
23: func (s *DeployService) CreateTask(req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
24: return s.CreateTaskContext(context.Background(), req, actor, scope)
25: }
26: func (s *DeployService) CreateTaskContext(ctx context.Context, req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
27: if err := ctx.Err(); err != nil {
28: return nil, err
29: }
30: return s.targetRequestService(ctx).createSelectedTask(req, actor, scope)
31: }
32: func (s *DeployService) createSelectedTask(req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
33: if req.TargetSelection == nil {
34: return s.createTask(req, actor, scope)
35: }
36: if s.db == nil {
37: return nil, errors.New("database.not_initialized")
38: }
39: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
40: if !ok {
41: return nil, errors.New("business.cmdb.selection.notConfigured")
42: }
43: var taskID uint64
44: err := s.db.Transaction(func(tx *gorm.DB) error {
45: owner := provider.WithTargetTransaction(tx)
46: selection, err := owner.ResolveTargetSelection(tx.Statement.Context, *req.TargetSelection, scope, true)
47: if err != nil {
48: return err
49: }
50: if req.ScopeVersion == "" || selection.ScopeVersion != req.ScopeVersion {
51: return cmdb.ErrTargetSelectionConflict
52: }
53: if selection.UnavailableCount > 0 || selection.IneligibleCount > 0 || len(selection.Hosts) == 0 {
54: return cmdb.ErrTargetSelectionInvalid
55: }
56: req.BusinessScopeID = selection.Selection.BusinessScopeID
57: req.TargetType = TargetTypeHost
58: req.TargetIDs = make([]uint64, 0, len(selection.Hosts))
59: for _, host := range selection.Hosts {
60: req.TargetIDs = append(req.TargetIDs, host.ID)
61: }
62: capability, ok := owner.(cmdb.DeployCMDBCapability)
63: if !ok {
64: return errors.New("business.cmdb.selection.notConfigured")
65: }
66: inner := s.targetTransactionService(tx, capability)
67: task, err := inner.createTask(req, actor, scope)
68: if err != nil {
69: return err
70: }
71: taskID = task.ID
72: snapshot := deployTargetSnapshot{Selection: &selection.Selection, ScopeVersion: selection.ScopeVersion, Sources: selection.Sources, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano), TargetType: TargetTypeHost}
73: bytes, err := json.Marshal(snapshot)
74: if err != nil {
75: return err
76: }
77: return tx.Model(&DeployTask{}).Where(idWhereClause, taskID).UpdateColumn("target_snapshot", datatypes.JSON(bytes)).Error
78: })
79: if err != nil {
80: return nil, err
81: }
82: return s.GetTask(taskID, scope)
83: }
84:
85: func (s *DeployService) UpdateTask(id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
86: return s.UpdateTaskContext(context.Background(), id, req, actor, scope)
87: }
88: func (s *DeployService) UpdateTaskContext(ctx context.Context, id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
89: if err := ctx.Err(); err != nil {
90: return nil, err
91: }
92: return s.targetRequestService(ctx).updateSelectedTask(id, req, actor, scope)
93: }
94: func (s *DeployService) updateSelectedTask(id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
95: if req.TargetSelection == nil {
96: current, err := s.loadVisibleTask(id, scope)
97: if err != nil {
98: return nil, err
99: }
100: var snapshot deployTargetSnapshot
101: if len(current.TargetSnapshot) > 0 && json.Unmarshal(current.TargetSnapshot, &snapshot) != nil {
102: return nil, cmdb.ErrTargetSelectionConflict
103: }
104: if snapshot.Selection != nil && (req.TargetIDs != nil || req.TargetType != nil || req.BusinessScopeID != nil) {
105: return nil, cmdb.ErrTargetSelectionInvalid
106: }
107: return s.updateTask(id, req, actor, scope)
108: }
109: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
110: if !ok {
111: return nil, cmdb.ErrTargetSelectionInvalid
112: }
113: err := s.db.Transaction(func(tx *gorm.DB) error {
114: var current DeployTask
115: if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, id).Error; err != nil {
116: return err
117: }
118: owner := provider.WithTargetTransaction(tx)
119: result, err := owner.ResolveTargetSelection(tx.Statement.Context, *req.TargetSelection, scope, true)
120: if err != nil {
121: return err
122: }
123: if req.ScopeVersion == "" || req.ScopeVersion != result.ScopeVersion {
124: return cmdb.ErrTargetSelectionConflict
125: }
126: if result.UnavailableCount > 0 || result.IneligibleCount > 0 || len(result.Hosts) == 0 {
127: return cmdb.ErrTargetSelectionInvalid
128: }
129: req.BusinessScopeID = &result.Selection.BusinessScopeID
130: targetType := TargetTypeHost
131: req.TargetType = &targetType
132: req.TargetIDs = make([]uint64, 0, len(result.Hosts))
133: for _, host := range result.Hosts {
134: req.TargetIDs = append(req.TargetIDs, host.ID)
135: }
136: capability, ok := owner.(cmdb.DeployCMDBCapability)
137: if !ok {
138: return cmdb.ErrTargetSelectionInvalid
139: }
140: inner := s.targetTransactionService(tx, capability)
141: if _, err := inner.updateTask(id, req, actor, scope); err != nil {
142: return err
143: }
144: snapshot := deployTargetSnapshot{Selection: &result.Selection, ScopeVersion: result.ScopeVersion, Sources: result.Sources, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano), TargetType: TargetTypeHost}
145: raw, err := json.Marshal(snapshot)
146: if err != nil {
147: return err
148: }
149: return tx.Model(&DeployTask{}).Where(idWhereClause, id).UpdateColumn("target_snapshot", datatypes.JSON(raw)).Error
150: })
151: if err != nil {
152: return nil, err
153: }
154: return s.GetTask(id, scope)
155: }
156:
157: // Preserve provider ownership and transaction-bound scope reads in both paths.
158: func (s *DeployService) targetTransactionService(tx *gorm.DB, provider cmdb.DeployCMDBCapability) *DeployService {
159: reader := s.bizScopeReader
160: if owner, ok := reader.(interface {
161: WithBizScopeTransaction(*gorm.DB) bizcap.BizScopeReader
162: }); ok {
163: reader = owner.WithBizScopeTransaction(tx)
164: }
165: inner := NewDeployService(tx, provider, reader)
166: inner.serviceState = s.serviceState
167: return inner
168: }
169:
170: // Request wrappers do not copy worker maps or replace the long-lived dispatcher.
171: func (s *DeployService) targetRequestService(ctx context.Context) *DeployService {
172: if s.db == nil {
173: return s
174: }
175: db := s.db.WithContext(ctx)
176: provider := s.cmdbCapability
177: if owner, ok := provider.(cmdb.TargetSelectionCapability); ok {
178: if bound, ok := owner.WithTargetTransaction(db).(cmdb.DeployCMDBCapability); ok {
179: provider = bound
180: }
181: }
182: return s.targetTransactionService(db, provider)
183: }
184: func (s *DeployService) requestContext() context.Context {
185: if s.db != nil && s.db.Statement != nil && s.db.Statement.Context != nil {
186: return s.db.Statement.Context
187: }
188: return context.Background()
189: }

FILE pantheon-ops/backend/modules/business/deploy/task_preview.go
1: package deploy
2:
3: import (
4: "bytes"
5: "context"
6: "crypto/hmac"
7: "crypto/sha256"
8: "encoding/base64"
9: "encoding/hex"
10: "encoding/json"
11: "errors"
12: "fmt"
13: "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
14: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
15: "gorm.io/gorm"
16: "gorm.io/gorm/clause"
17: "strings"
18: "time"
19: )
20:
21: var ErrTaskPreviewRequired = errors.New("business.deploy.preview.required")
22: var ErrTaskPreviewConflict = errors.New("business.deploy.preview.conflict")
23:
24: // Derive a separate signing key from the existing deployment secret, shared by
25: // all replicas. Rotation invalidates outstanding reviews and requires refresh.
26: func taskPreviewSigningKey() ([]byte, error) {
27: root, err := deployCredentialKey()
28: if err != nil {
29: return nil, err
30: }
31: mac := hmac.New(sha256.New, root)
32: _, _ = mac.Write([]byte("pantheon-ops/deploy-preview/v1"))
33: return mac.Sum(nil), nil
34: }
35:
36: type TaskPreviewChange struct {
37: ID string `json:"id"`
38: Field string `json:"field"`
39: Kind string `json:"kind"`
40: Before string `json:"before,omitempty"`
41: After string `json:"after,omitempty"`
42: }
43: type TaskPreviewResponse struct {
44: IntentID string `json:"intentId"`
45: IntentVersion string `json:"intentVersion"`
46: CurrentVersion string `json:"currentVersion"`
47: TargetVersion string `json:"targetVersion"`
48: TargetCount int `json:"targetCount"`
49: BusinessScopeName string `json:"businessScopeName"`
50: Changes []TaskPreviewChange `json:"changes"`
51: RiskKeys []string `json:"riskKeys"`
52: RollbackAvailable bool `json:"rollbackAvailable"`
53: ExpiresAt time.Time `json:"expiresAt"`
54: }
55: type taskPreviewTicket struct {
56: TaskID uint64 `json:"taskId"`
57: Actor string `json:"actor"`
58: Revision string `json:"revision"`
59: Expires int64 `json:"expires"`
60: }
61:
62: func signPreviewTicket(ticket taskPreviewTicket) (string, error) {
63: payload, err := json.Marshal(ticket)
64: if err != nil {
65: return "", err
66: }
67: key, err := taskPreviewSigningKey()
68: if err != nil {
69: return "", err
70: }
71: mac := hmac.New(sha256.New, key)
72: _, _ = mac.Write(payload)
73: return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
74: }
75: func parsePreviewTicket(token string) (taskPreviewTicket, error) {
76: var ticket taskPreviewTicket
77: if len(token) > 1024 {
78: return ticket, ErrTaskPreviewConflict
79: }
80: parts := strings.Split(token, ".")
81: if len(parts) != 2 {
82: return ticket, ErrTaskPreviewConflict
83: }
84: payload, err := base64.RawURLEncoding.DecodeString(parts[0])
85: if err != nil {
86: return ticket, ErrTaskPreviewConflict
87: }
88: signature, err := base64.RawURLEncoding.DecodeString(parts[1])
89: if err != nil {
90: return ticket, ErrTaskPreviewConflict
91: }
92: key, err := taskPreviewSigningKey()
93: if err != nil {
94: return ticket, err
95: }
96: mac := hmac.New(sha256.New, key)
97: _, _ = mac.Write(payload)
98: if !hmac.Equal(signature, mac.Sum(nil)) {
99: return ticket, ErrTaskPreviewConflict
100: }
101: decoder := json.NewDecoder(bytes.NewReader(payload))
102: decoder.DisallowUnknownFields()
103: if err := decoder.Decode(&ticket); err != nil {
104: return ticket, ErrTaskPreviewConflict
105: }
106: if ticket.Expires <= time.Now().Unix() {
107: return ticket, ErrTaskPreviewConflict
108: }
109: return ticket, nil
110: }
111: func taskPreviewRevision(db *gorm.DB, task *DeployTask, hosts []cmdbHostSnapshot, req StartTaskRequest, credentialVersion uint64, lock bool) (string, error) {
112: var secrets []struct {
113: SecretKey string
114: SecretEncrypted string
115: }
116: query := db.Model(&DeployTaskSecret{}).Where(taskIDWhereClause, task.ID).Order("secret_key ASC")
117: if lock {
118: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
119: }
120: if err := query.Find(&secrets).Error; err != nil {
121: return "", err
122: }
123: request := struct {
124: CredentialRefID uint64
125: TimeoutSeconds int
126: HostFingerprint string
127: }{req.CredentialRefID, int(normalizeDeployExecutionTimeout(req.TimeoutSeconds).Seconds()), strings.TrimSpace(req.HostFingerprint)}
128: material := struct {
129: Task *DeployTask
130: Hosts []cmdbHostSnapshot
131: Secrets any
132: Request any
133: CredentialVersion uint64
134: }{task, hosts, secrets, request, credentialVersion}
135: payload, err := json.Marshal(material)
136: if err != nil {
137: return "", err
138: }
139: hash := sha256.Sum256(payload)
140: return hex.EncodeToString(hash[:]), nil
141: }
142: func (s *DeployService) PreviewTask(id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
143: return s.PreviewTaskContext(context.Background(), id, req, actor, scope)
144: }
145: func (s *DeployService) PreviewTaskContext(ctx context.Context, id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
146: if err := ctx.Err(); err != nil {
147: return nil, err
148: }
149: return s.targetRequestService(ctx).previewTask(id, req, actor, scope)
150: }
151: func (s *DeployService) previewTask(id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
152: task, err := s.loadVisibleTask(id, scope)
153: if err != nil {
154: return nil, err
155: }
156: if task.Status != TaskStatusDraft && task.Status != TaskStatusPending {
157: return nil, ErrTaskPreviewConflict
158: }
159: credentialVersion := uint64(0)
160: if task.ExecutorType == ExecutorTypeSSH {
161: resolved, version, err := s.prepareStartCredential(req)
162: if err != nil {
163: return nil, err
164: }
165: req = resolved
166: credentialVersion = version
167: }
168: hosts, err := s.prepareStartTargets(task, req, scope)
169: if err != nil {
170: return nil, err
171: }
172: for _, host := range hosts {
173: if !hostStatusAllowedForAction(host.Status, task.Action) {
174: return nil, ErrTaskPreviewConflict
175: }
176: }
177: if err := s.validateSelectionForPreview(s.requestContext(), task, scope, false); err != nil {
178: return nil, err
179: }
180: revision, err := taskPreviewRevision(s.db, task, hosts, req, credentialVersion, false)
181: if err != nil {
182: return nil, err
183: }
184: expires := time.Now().UTC().Add(10 * time.Minute)
185: token, err := signPreviewTicket(taskPreviewTicket{TaskID: id, Actor: actor, Revision: revision, Expires: expires.Unix()})
186: if err != nil {
187: return nil, err
188: }
189: targetHash := sha256.Sum256(task.ExecutionSnapshot)
190: result := &TaskPreviewResponse{IntentID: fmt.Sprintf("deploy:%d", id), IntentVersion: token, CurrentVersion: revision, TargetVersion: hex.EncodeToString(targetHash[:]), TargetCount: len(hosts), BusinessScopeName: task.BusinessScopeName, Changes: []TaskPreviewChange{}, RiskKeys: []string{"business.deploy.preview.healthUnknown", "business.deploy.preview.rollbackManual"}, RollbackAvailable: false, ExpiresAt: expires}
191: for _, host := range hosts {
192: current := ""
193: known := true
194: var installed []struct {
195: Name string `json:"name"`
196: Version string `json:"version"`
197: }
198: if len(host.InstalledComponents) > 0 && json.Unmarshal(host.InstalledComponents, &installed) != nil {
199: known = false
200: }
201: for _, component := range installed {
202: if component.Name == task.PackageName {
203: current = component.Version
204: }
205: }
206: kind := "modified"
207: target := task.PackageVersion
208: if !known {
209: kind = "conflict"
210: } else if current == "" {
211: kind = "added"
212: } else if current == target {
213: kind = "unchanged"
214: }
215: if task.Action == TaskActionUninstall || task.Action == TaskActionRetire {
216: kind = "removed"
217: target = ""
218: }
219: result.Changes = append(result.Changes, TaskPreviewChange{ID: fmt.Sprintf("host:%d", host.ID), Field: host.Hostname + " / " + task.PackageName, Kind: kind, Before: current, After: target})
220: }
221: if len(hosts) > 100 {
222: result.RiskKeys = append(result.RiskKeys, "business.deploy.preview.largeScope")
223: }
224: if task.Action == TaskActionUninstall || task.Action == TaskActionRetire || task.Action == TaskActionStop {
225: result.RiskKeys = append(result.RiskKeys, "business.deploy.preview.disruptive")
226: }
227: return result, nil
228: }
229: func (s *DeployService) validateSelectionForPreview(ctx context.Context, task *DeployTask, scope *common.DataScopeReq, lock bool) error {
230: var snapshot deployTargetSnapshot
231: if len(task.TargetSnapshot) == 0 {
232: return nil
233: }
234: if err := json.Unmarshal(task.TargetSnapshot, &snapshot); err != nil {
235: return ErrTaskPreviewConflict
236: }
237: if snapshot.Selection == nil {
238: return nil
239: }
240: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
241: if !ok {
242: return cmdb.ErrTargetSelectionInvalid
243: }
244: resolved, err := provider.ResolveTargetSelection(ctx, *snapshot.Selection, scope, lock)
245: if err != nil {
246: return err
247: }
248: if resolved.ScopeVersion != snapshot.ScopeVersion || resolved.UnavailableCount > 0 || resolved.IneligibleCount > 0 {
249: return cmdb.ErrTargetSelectionConflict
250: }
251: return nil
252: }
253: func (c *startTaskClaim) validatePreview(tx *gorm.DB) error {
254: var selection deployTargetSnapshot
255: if len(c.task.TargetSnapshot) > 0 && json.Unmarshal(c.task.TargetSnapshot, &selection) != nil {
256: return ErrTaskPreviewConflict
257: }
258: if !c.req.RequirePreview && c.req.IntentVersion == "" && selection.Selection == nil {
259: return nil
260: }
261: if c.req.IntentVersion == "" {
262: return ErrTaskPreviewRequired
263: }
264: ticket, err := parsePreviewTicket(c.req.IntentVersion)
265: if err != nil {
266: return err
267: }
268: if ticket.TaskID != c.task.ID || ticket.Actor != c.actor {
269: return ErrTaskPreviewConflict
270: }
271: var current DeployTask
272: if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, c.task.ID).Error; err != nil {
273: return err
274: }
275: if current.Status != TaskStatusDraft && current.Status != TaskStatusPending {
276: return errors.New(errDeployTaskAlreadyRunning)
277: }
278: provider := c.service.cmdbCapability
279: if owner, ok := provider.(cmdb.TargetSelectionCapability); ok {
280: transactionProvider, valid := owner.WithTargetTransaction(tx).(cmdb.DeployCMDBCapability)
281: if !valid {
282: return cmdb.ErrTargetSelectionInvalid
283: }
284: provider = transactionProvider
285: }
286: inner := c.service.targetTransactionService(tx, provider)
287: if err := inner.validateSelectionForPreview(tx.Statement.Context, &current, c.dataScope, true); err != nil {
288: return err
289: }
290: hosts, err := inner.prepareStartTargets(&current, c.req, c.dataScope)
291: if err != nil {
292: return err
293: }
294: if current.ExecutorType == ExecutorTypeSSH {
295: var credential DeployCredentialRef
296: if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&credential, c.req.CredentialRefID).Error; err != nil {
297: return err
298: }
299: if credential.Version != c.credentialVersion || credential.Status != "active" {
300: return ErrTaskPreviewConflict
301: }
302: }
303: revision, err := taskPreviewRevision(tx, &current, hosts, c.req, c.credentialVersion, true)
304: if err != nil {
305: return err
306: }
307: if ticket.Revision != revision {
308: return ErrTaskPreviewConflict
309: }
310: c.task = &current
311: c.hosts = hosts
312: return nil
313: }

FILE pantheon-ops/backend/modules/business/deploy/task_preview_test.go
1: package deploy
2:
3: import (
4: "context"
5: "encoding/json"
6: "errors"
7: "github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
8: "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
9: cmdbgroup "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/group"
10: cmdbhost "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
11: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
12: "gorm.io/gorm"
13: "strings"
14: "testing"
15: "time"
16: )
17:
18: func previewFixture(t *testing.T, selected bool) (*gorm.DB, *DeployService, *TaskResponse, uint64) {
19: t.Helper()
20: t.Setenv(deployCredentialKeyEnv, strings.Repeat("12", 32))
21: db := setupDeployTestDB(t)
22: if err := db.AutoMigrate(&cmdbhost.Host{}, &cmdbgroup.Group{}); err != nil {
23: t.Fatal(err)
24: }
25: reader := bizscope.NewService(db)
26: svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db, reader), reader)
27: scopeID, hostID := seedDeployReliabilityFixture(t, db, "preview-host", "10.40.9.1")
28: pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "preview-package", Version: "2.0.0", InstallCommand: "echo install", Status: PackageStatusEnabled}, "1")
29: if err != nil {
30: t.Fatal(err)
31: }
32: req := CreateTaskRequest{Name: "preview-task", PackageID: pkg.ID, BusinessScopeID: scopeID, TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}, ExecutorType: ExecutorTypeManual}
33: if selected {
34: selection := cmdb.TargetSelection{BusinessScopeID: scopeID, HostIDs: []uint64{hostID}}
35: resolved, err := svc.ResolveTargetSelection(context.Background(), selection, nil)
36: if err != nil {
37: t.Fatal(err)
38: }
39: req.TargetSelection = &selection
40: req.ScopeVersion = resolved.ScopeVersion
41: }
42: task, err := svc.CreateTask(req, "1", nil)
43: if err != nil {
44: t.Fatal(err)
45: }
46: return db, svc, task, hostID
47: }
48: func TestPreviewTicketRejectsForgeryExpiryAndBudget(t *testing.T) {
49: t.Setenv(deployCredentialKeyEnv, strings.Repeat("12", 32))
50: valid := taskPreviewTicket{TaskID: 7, Actor: "1", Revision: "facts", Expires: time.Now().Add(time.Minute).Unix()}
51: token, err := signPreviewTicket(valid)
52: if err != nil {
53: t.Fatal(err)
54: }
55: parsed, err := parsePreviewTicket(token)
56: if err != nil || parsed != valid {
57: t.Fatalf("valid ticket: %v", err)
58: }
59: expired := valid
60: expired.Expires = time.Now().Add(-time.Minute).Unix()
61: expiredToken, _ := signPreviewTicket(expired)
62: for _, bad := range []string{"", token + "x", strings.Repeat("x", 1025), expiredToken, "forged.signature"} {
63: if _, err := parsePreviewTicket(bad); !errors.Is(err, ErrTaskPreviewConflict) {
64: t.Fatalf("accepted invalid ticket: %v", err)
65: }
66: }
67: }
68: func TestPreviewRequiredValidStartAndReplay(t *testing.T) {
69: db, svc, task, _ := previewFixture(t, true)
70: if _, err := svc.StartTask(task.ID, StartTaskRequest{RequirePreview: true}, "1", nil); !errors.Is(err, ErrTaskPreviewRequired) {
71: t.Fatalf("missing preview accepted: %v", err)
72: }
73: preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
74: if err != nil {
75: t.Fatal(err)
76: }
77: if preview.TargetCount != 1 || len(preview.Changes) != 1 || preview.Changes[0].Kind != "added" {
78: t.Fatalf("wrong preview: %+v", preview)
79: }
80: req := StartTaskRequest{IntentVersion: preview.IntentVersion, RequirePreview: true, IdempotencyKey: "preview-start"}
81: started, err := svc.StartTask(task.ID, req, "1", nil)
82: if err != nil {
83: t.Fatal(err)
84: }
85: if started.Status != TaskStatusRunning || len(started.Hosts) != 1 {
86: t.Fatalf("start did not claim: %+v", started)
87: }
88: replay, err := svc.StartTask(task.ID, req, "1", nil)
89: if err != nil || !replay.StartRequestReused {
90: t.Fatalf("replay: %v", err)
91: }
92: var count int64
93: db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&count)
94: if count != 1 {
95: t.Fatalf("duplicate host: %d", count)
96: }
97: }
98: func TestPreviewRejectsActorRequestAndPersistedDrift(t *testing.T) {
99: for _, kind := range []string{"actor", "request", "task", "host", "secret", "scope", "permission"} {
100: t.Run(kind, func(t *testing.T) {
101: db, svc, task, hostID := previewFixture(t, true)
102: preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
103: if err != nil {
104: t.Fatal(err)
105: }
106: req := StartTaskRequest{RequirePreview: true, IntentVersion: preview.IntentVersion}
107: actor := "1"
108: var scope *common.DataScopeReq
109: switch kind {
110: case "actor":
111: actor = "2"
112: case "request":
113: req.TimeoutSeconds = 42
114: case "task":
115: err = db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("remark", "changed").Error
116: case "host":
117: err = db.Model(&cmdbhost.Host{}).Where("id = ?", hostID).Update("ip", "10.40.9.2").Error
118: case "secret":
119: err = db.Create(&DeployTaskSecret{TaskID: task.ID, SecretKey: "password", SecretEncrypted: "ciphertext-change"}).Error
120: case "scope":
121: err = db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("status", "disabled").Error
122: case "permission":
123: scope = &common.DataScopeReq{Mode: common.DataScopeModeSelf, UserID: 999}
124: }
125: if err != nil {
126: t.Fatal(err)
127: }
128: if _, err = svc.StartTask(task.ID, req, actor, scope); err == nil {
129: t.Fatal("stale preview started")
130: }
131: var current DeployTask
132: if err = db.First(&current, task.ID).Error; err != nil {
133: t.Fatal(err)
134: }
135: if current.Status == TaskStatusRunning {
136: t.Fatal("failed validation mutated task")
137: }
138: var count int64
139: db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&count)
140: if count != 0 {
141: t.Fatal("failed validation persisted host")
142: }
143: })
144: }
145: }
146: func TestSelectedCreateRejectsStaleResolution(t *testing.T) {
147: db, svc, task, hostID := previewFixture(t, true)
148: selected := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, HostIDs: []uint64{hostID}}
149: resolved, err := svc.ResolveTargetSelection(context.Background(), selected, nil)
150: if err != nil {
151: t.Fatal(err)
152: }
153: if err = db.Model(&cmdbhost.Host{}).Where("id = ?", hostID).Update("hostname", "changed-name").Error; err != nil {
154: t.Fatal(err)
155: }
156: _, err = svc.CreateTask(CreateTaskRequest{Name: "stale-create", PackageID: task.PackageID, TargetSelection: &selected, ScopeVersion: resolved.ScopeVersion, ExecutorType: ExecutorTypeManual}, "1", nil)
157: if !errors.Is(err, cmdb.ErrTargetSelectionConflict) {
158: t.Fatalf("accepted stale resolution: %v", err)
159: }
160: var count int64
161: db.Model(&DeployTask{}).Where("name = ?", "stale-create").Count(&count)
162: if count != 0 {
163: t.Fatal("stale create persisted")
164: }
165: }
166: func TestPreviewSecretMaterialIsNeverReturned(t *testing.T) {
167: db, svc, task, _ := previewFixture(t, false)
168: if err := db.Create(&DeployTaskSecret{TaskID: task.ID, SecretKey: "password", SecretEncrypted: "never-expose-ciphertext"}).Error; err != nil {
169: t.Fatal(err)
170: }
171: preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
172: if err != nil {
173: t.Fatal(err)
174: }
175: raw, _ := json.Marshal(preview)
176: if strings.Contains(string(raw), "never-expose") || strings.Contains(string(raw), "echo install") {
177: t.Fatal("preview leaked sensitive execution material")
178: }
179: }
180:
181: func TestSelectedUpdateKeepsNewProvenanceAndInvalidatesPreview(t *testing.T) {
182: db, svc, task, hostID := previewFixture(t, true)
183: preview, err := svc.PreviewTask(task.ID, StartTaskRequest{}, "1", nil)
184: if err != nil {
185: t.Fatal(err)
186: }
187: selection := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, Manual: []string{"10.40.9.1"}}
188: result, err := svc.ResolveTargetSelection(context.Background(), selection, nil)
189: if err != nil {
190: t.Fatal(err)
191: }
192: updated, err := svc.UpdateTask(task.ID, UpdateTaskRequest{TargetSelection: &selection, ScopeVersion: result.ScopeVersion}, "1", nil)
193: if err != nil {
194: t.Fatal(err)
195: }
196: if updated.TargetSelection == nil || len(updated.TargetSelection.Manual) != 1 || updated.TargetIDs[0] != hostID {
197: t.Fatal("update lost provenance")
198: }
199: if _, err := svc.StartTask(task.ID, StartTaskRequest{IntentVersion: preview.IntentVersion, RequirePreview: true}, "1", nil); err == nil {
200: t.Fatal("old preview started changed selection")
201: }
202: var count int64
203: db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&count)
204: if count != 0 {
205: t.Fatal("failed start leaked host")
206: }
207: if _, err := svc.UpdateTask(task.ID, UpdateTaskRequest{TargetIDs: []uint64{hostID}}, "1", nil); !errors.Is(err, cmdb.ErrTargetSelectionInvalid) {
208: t.Fatalf("legacy target edit bypassed selection: %v", err)
209: }
210: }
211: func TestPreviewCredentialVersionChangesInvalidate(t *testing.T) {
212: db, svc, task, _ := previewFixture(t, false)
213: if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Update("executor_type", ExecutorTypeSSH).Error; err != nil {
214: t.Fatal(err)
215: }
216: credential := setupDeployTestCredential(t, svc)
217: req := StartTaskRequest{CredentialRefID: credential, HostFingerprint: "SHA256:preview-fixture", RequirePreview: true}
218: preview, err := svc.PreviewTask(task.ID, req, "1", nil)
219: if err != nil {
220: t.Fatal(err)
221: }
222: if err := db.Model(&DeployCredentialRef{}).Where("id = ?", credential).Update("version", 2).Error; err != nil {
223: t.Fatal(err)
224: }
225: req.IntentVersion = preview.IntentVersion
226: if _, err := svc.StartTask(task.ID, req, "1", nil); !errors.Is(err, ErrTaskPreviewConflict) {
227: t.Fatalf("changed credential accepted: %v", err)
228: }
229: }
230:
231: func TestPreviewSigningConfigurationAndRotation(t *testing.T) {
232: t.Setenv(deployCredentialKeyEnv, strings.Repeat("12", 32))
233: ticket := taskPreviewTicket{TaskID: 1, Actor: "1", Revision: "revision", Expires: time.Now().Add(time.Minute).Unix()}
234: first, err := signPreviewTicket(ticket)
235: if err != nil {
236: t.Fatal(err)
237: }
238: second, err := signPreviewTicket(ticket)
239: if err != nil || first != second {
240: t.Fatal("shared secret did not yield a stable signature")
241: }
242: t.Setenv(deployCredentialKeyEnv, strings.Repeat("34", 32))
243: if _, err := parsePreviewTicket(first); !errors.Is(err, ErrTaskPreviewConflict) {
244: t.Fatalf("rotation accepted old signature: %v", err)
245: }
246: for _, key := range []string{"", "malformed"} {
247: t.Setenv(deployCredentialKeyEnv, key)
248: if _, err := signPreviewTicket(ticket); err == nil {
249: t.Fatal("missing/invalid config accepted")
250: }
251: }
252: }
253: func TestPreviewCorruptSnapshotCannotDowngradeEnforcement(t *testing.T) {
254: db, svc, task, _ := previewFixture(t, false)
255: if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).UpdateColumn("target_snapshot", "[]").Error; err != nil {
256: t.Fatal(err)
257: }
258: if _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil); !errors.Is(err, ErrTaskPreviewConflict) {
259: t.Fatalf("corrupt snapshot bypassed gate: %v", err)
260: }
261: }
262:
263: func TestSelectedUpdateCannotMoveInvisibleTaskIntoVisibleScope(t *testing.T) {
264: db, svc, task, hostID := previewFixture(t, true)
265: // The replacement target is visible, while the existing task target is not.
266: visible := cmdbhost.Host{Hostname: "visible-replacement", IP: "10.40.9.3", BusinessScopeID: task.BusinessScopeID, Status: "assigned", StateVersion: 1, DeptID: 9}
267: visible.CreatedBy = "2"
268: if err := db.Create(&visible).Error; err != nil {
269: t.Fatal(err)
270: }
271: scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 9}
272: selection := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, HostIDs: []uint64{visible.ID}}
273: // Scope itself must be visible so the regression reaches task visibility.
274: if err := db.Model(&deployTestBizScope{}).Where("id = ?", task.BusinessScopeID).Update("dept_id", 9).Error; err != nil {
275: t.Fatal(err)
276: }
277: resolved, err := svc.ResolveTargetSelection(context.Background(), selection, scope)
278: if err != nil {
279: t.Fatal(err)
280: }
281: before := DeployTask{}
282: if err := db.First(&before, task.ID).Error; err != nil {
283: t.Fatal(err)
284: }
285: if _, err := svc.UpdateTask(task.ID, UpdateTaskRequest{TargetSelection: &selection, ScopeVersion: resolved.ScopeVersion}, "2", scope); err == nil {
286: t.Fatal("invisible task became editable through visible replacement")
287: }
288: after := DeployTask{}
289: if err := db.First(&after, task.ID).Error; err != nil {
290: t.Fatal(err)
291: }
292: if string(before.TargetSnapshot) != string(after.TargetSnapshot) || string(before.TargetIDs) != string(after.TargetIDs) {
293: t.Fatal("denied update mutated provenance/targets")
294: }
295: if after.TargetIDs == nil || hostID == visible.ID {
296: t.Fatal("invalid fixture")
297: }
298: }
299: func TestCanceledSelectionRequestsDoNotMutate(t *testing.T) {
300: db, svc, task, hostID := previewFixture(t, true)
301: ctx, cancel := context.WithCancel(context.Background())
302: cancel()
303: selection := cmdb.TargetSelection{BusinessScopeID: task.BusinessScopeID, HostIDs: []uint64{hostID}}
304: resolved, err := svc.ResolveTargetSelection(context.Background(), selection, nil)
305: if err != nil {
306: t.Fatal(err)
307: }
308: cases := []struct {
309: name string
310: call func() error
311: }{
312: {"create", func() error {
313: _, err := svc.CreateTaskContext(ctx, CreateTaskRequest{Name: "canceled-create", PackageID: task.PackageID, TargetSelection: &selection, ScopeVersion: resolved.ScopeVersion}, "1", nil)
314: return err
315: }},
316: {"update", func() error {
317: _, err := svc.UpdateTaskContext(ctx, task.ID, UpdateTaskRequest{TargetSelection: &selection, ScopeVersion: resolved.ScopeVersion}, "1", nil)
318: return err
319: }},
320: {"preview", func() error { _, err := svc.PreviewTaskContext(ctx, task.ID, StartTaskRequest{}, "1", nil); return err }},
321: {"start", func() error {
322: _, err := svc.StartTaskContext(ctx, task.ID, StartTaskRequest{RequirePreview: true}, "1", nil)
323: return err
324: }},
325: }
326: for _, c := range cases {
327: if err := c.call(); !errors.Is(err, context.Canceled) {
328: t.Fatalf("%s ignored cancellation: %v", c.name, err)
329: }
330: }
331: var count int64
332: if err := db.Model(&DeployTask{}).Where("name = ?", "canceled-create").Count(&count).Error; err != nil || count != 0 {
333: t.Fatal("canceled create persisted")
334: }
335: var current DeployTask
336: if err := db.First(&current, task.ID).Error; err != nil || current.Status != TaskStatusDraft {
337: t.Fatal("canceled start changed draft")
338: }
339: }

FILE pantheon-ops/frontend/src/modules/business/deploy/task/DeployTargetSelector.tsx
1: import { useEffect, useMemo, useRef, useState } from 'react';
2: import { useTranslation } from 'react-i18next';
3: import { Alert, Button, Checkbox, Input, Pagination, Select, Space, Typography } from '@arco-design/web-react';
4: import { ContextSelector, type ContextSelectorOption } from '../../../../components';
5: import { isForbiddenRequestError } from '../../../../api/request';
6: import { getHostList, type HostRow } from '../../cmdb/host/api';
7: import { resolveDeployTargets, type DeployTargetResolution, type DeployTargetSelection } from '../api';
8: interface Props {
9: scopeId: number;
10: groups: Array<{ id: number; label: string }>;
11: initial?: DeployTargetSelection;
12: initialHostIds?: number[];
13: onChange: (resolution: DeployTargetResolution | null) => void;
14: }
15: export default function DeployTargetSelector({ scopeId, groups, initial, initialHostIds, onChange }: Readonly<Props>) {
16: const { t } = useTranslation();
17: const [intent, setIntent] = useState<DeployTargetSelection>(() => (initial?.businessScopeId === scopeId ? initial : undefined) || { businessScopeId: scopeId, hostIds: initialHostIds || [], groupIds: [], includeScope: false, manual: [], excludedIds: [] });
18: const [hosts, setHosts] = useState<HostRow[]>([]);
19: const [total, setTotal] = useState(0);
20: const [page, setPage] = useState(1);
21: const [keyword, setKeyword] = useState('');
22: const [loading, setLoading] = useState(true);
23: const [lookupFailed, setLookupFailed] = useState<'error' | 'forbidden' | null>(null);
24: const [retry, setRetry] = useState(0);
25: const [resolving, setResolving] = useState(false);
26: const [resolution, setResolution] = useState<DeployTargetResolution | null>(null);
27: const [failure, setFailure] = useState<'error' | 'forbidden' | null>(null);
28: const generation = useRef(0);
29: const resolver = useRef<AbortController | null>(null);
30: useEffect(() => {
31: const controller = new AbortController();
32: const timer = setTimeout(() => {
33: setLoading(true);
34: setLookupFailed(null);
35: void getHostList({ page, pageSize: 25, keyword, businessScopeId: scopeId }, controller.signal).then((response) => {
36: if (controller.signal.aborted) return;
37: setHosts(response.items); setTotal(response.total); setLoading(false);
38: }).catch((error: unknown) => { if (!controller.signal.aborted) { setHosts([]); setLookupFailed(isForbiddenRequestError(error) ? 'forbidden' : 'error'); setLoading(false); } });
39: }, 250);
40: return () => { clearTimeout(timer); controller.abort(); };
41: }, [scopeId, page, keyword, retry]);
42: useEffect(() => () => { generation.current += 1; resolver.current?.abort(); }, []);
43: const change = (next: DeployTargetSelection) => {
44: generation.current += 1; resolver.current?.abort();
45: setIntent(next); setResolution(null); setFailure(null); setResolving(false); onChange(null);
46: };
47: const options = useMemo(() => hosts.map((host) => ({ id: String(host.id), source: 'host', label: `${host.hostname} · ${host.ip}`, disabled: !['assigned', 'online'].includes(host.status) })), [hosts]);
48: const selected = intent.hostIds.map((id) => options.find((option) => option.id === String(id)) || { id: String(id), source: 'host', label: `#${id}` });
49: const toggle = (option: ContextSelectorOption, remove: boolean) => change({ ...intent, hostIds: remove ? intent.hostIds.filter((id) => String(id) !== option.id) : [...intent.hostIds, Number(option.id)] });
50: const resolve = async () => {
51: const current = ++generation.current;
52: resolver.current?.abort(); const controller = new AbortController(); resolver.current = controller;
53: setResolving(true); setFailure(null); setResolution(null); onChange(null);
54: try {
55: const result = await resolveDeployTargets({ ...intent, businessScopeId: scopeId }, controller.signal);
56: if (current !== generation.current || controller.signal.aborted) return;
57: setResolution(result); onChange(result);
58: } catch (error) { if (current === generation.current && !controller.signal.aborted) setFailure(isForbiddenRequestError(error) ? 'forbidden' : 'error'); }
59: finally { if (current === generation.current) setResolving(false); }
60: };
61: const labels = {
62: searchPlaceholder: t('business.deploy.selection.filterPage'), available: t('business.deploy.selection.available'), selected: t('business.deploy.selection.selected'),
63: add: t('business.deploy.selection.add'), remove: t('business.deploy.selection.remove'), excluded: t('business.deploy.selection.excluded'), staleItem: t('business.deploy.selection.stale'),
64: loading: t('common.loading'), empty: t('common.noData'), error: t('business.deploy.selection.loadFailed'), forbidden: t('business.deploy.preview.forbidden'), stale: t('business.deploy.selection.stale'), partial: t('business.deploy.preview.partial'), ready: t('business.deploy.preview.ready'),
65: };
66: const budget = intent.hostIds.length + intent.groupIds.length + intent.manual.length + intent.excludedIds.length;
67: return <Space className="deploy-target-selector" direction="vertical" size={12} style={{ width: '100%' }}>
68: <Input allowClear value={keyword} aria-label={t('business.deploy.selection.search')} placeholder={t('business.deploy.selection.search')} onChange={(value) => { setHosts([]); setLoading(true); setKeyword(value); setPage(1); }} />
69: <ContextSelector title={t('business.deploy.selection.title')} options={options} selected={selected} labels={labels} state={loading ? 'loading' : lookupFailed || (hosts.length === 0 ? 'empty' : 'ready')} maxVisibleOptions={25} onSelect={(option) => toggle(option, false)} onRemove={(option) => toggle(option, true)} />
70: {lookupFailed && <Button onClick={() => setRetry((value) => value + 1)}>{t('common.retry')}</Button>}
71: <Pagination simple current={page} pageSize={25} total={total} onChange={(value) => { setHosts([]); setLoading(true); setPage(value); }} />
72: <Checkbox checked={intent.includeScope} onChange={(value) => change({ ...intent, includeScope: value })}>{t('business.deploy.selection.includeScope')}</Checkbox>
73: <Select mode="multiple" allowClear showSearch value={intent.groupIds} placeholder={t('business.deploy.selection.groups')} aria-label={t('business.deploy.selection.groups')} onChange={(ids: number[]) => change({ ...intent, groupIds: ids })}>{groups.map((group) => <Select.Option key={group.id} value={group.id}>{group.label}</Select.Option>)}</Select>
74: <Input.TextArea value={intent.manual.join('\n')} aria-label={t('business.deploy.selection.manual')} placeholder={t('business.deploy.selection.manual')} maxLength={1280000} autoSize={{ minRows: 2, maxRows: 5 }} onChange={(value) => change({ ...intent, manual: value.split(/\n/).filter((item) => item.trim()) })} />
75: <Select mode="multiple" allowClear value={intent.excludedIds} placeholder={t('business.deploy.selection.exclude')} aria-label={t('business.deploy.selection.exclude')} onChange={(ids: number[]) => change({ ...intent, excludedIds: ids })}>{(resolution?.hosts || hosts).map((host) => <Select.Option key={host.id} value={host.id}>{host.hostname} · {host.ip}</Select.Option>)}</Select>
76: <Button loading={resolving} disabled={budget > 10000} onClick={() => void resolve()}>{t('business.deploy.selection.resolve')}</Button>
77: {budget > 10000 && <Alert type="error" content={t('business.deploy.selection.budget')} />}
78: {failure && <Alert type="error" content={t(failure === 'forbidden' ? 'business.deploy.preview.forbidden' : 'business.deploy.selection.loadFailed')} />}
79: {resolution && <div role="status"><Typography.Text>{t('business.deploy.selection.summary', { count: resolution.hosts.length, duplicate: resolution.duplicateCount, unavailable: resolution.unavailableCount, ineligible: resolution.ineligibleCount, excluded: resolution.excludedCount })}</Typography.Text>{(resolution.unavailableCount > 0 || resolution.ineligibleCount > 0) && <Alert type="warning" content={t('business.deploy.selection.invalid')} />}</div>}
80: </Space>;
81: }

FILE pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskPreview.tsx
1: import { useEffect, useRef, useState } from 'react';
2: import { useTranslation } from 'react-i18next';
3: import { Alert, Button, Input, Pagination, Space, Typography } from '@arco-design/web-react';
4: import { AppModal, ChangeDiff } from '../../../../components';
5: import { isForbiddenRequestError } from '../../../../api/request';
6: import { previewDeployTask, startDeployTask, type DeployTaskPreviewResponse, type StartDeployTaskPayload } from '../api';
7:
8: interface Props {
9: taskId: number;
10: payload: StartDeployTaskPayload;
11: onClose: () => void;
12: onStarted: () => void | Promise<void>;
13: }
14: export default function DeployTaskPreview({ taskId, payload, onClose, onStarted }: Readonly<Props>) {
15: const { t } = useTranslation();
16: const [preview, setPreview] = useState<DeployTaskPreviewResponse | null>(null);
17: const [state, setState] = useState<'loading' | 'ready' | 'error' | 'forbidden' | 'stale'>('loading');
18: const [generation, setGeneration] = useState(0);
19: const [keyword, setKeyword] = useState('');
20: const [page, setPage] = useState(1);
21: const [submitting, setSubmitting] = useState(false);
22: const submittingRef = useRef(false);
23: const requestKey = useRef(crypto.randomUUID());
24: useEffect(() => {
25: const controller = new AbortController();
26: let expiry: ReturnType<typeof setTimeout> | undefined;
27: void previewDeployTask(taskId, payload, controller.signal).then((result) => {
28: if (controller.signal.aborted) return;
29: const remaining = new Date(result.expiresAt).getTime() - Date.now();
30: if (!Number.isFinite(remaining) || remaining <= 0) { setState('stale'); return; }
31: setPreview(result);
32: setState('ready');
33: expiry = setTimeout(() => { setPreview(null); setState('stale'); }, remaining);
34: }).catch((error: unknown) => {
35: if (!controller.signal.aborted) setState(isForbiddenRequestError(error) ? 'forbidden' : 'error');
36: });
37: return () => { controller.abort(); if (expiry) clearTimeout(expiry); };
38: }, [taskId, payload, generation]);
39: const refresh = () => { setPreview(null); setState('loading'); setPage(1); requestKey.current = crypto.randomUUID(); setGeneration((value) => value + 1); };
40: const confirm = async () => {
41: if (!preview || submittingRef.current || new Date(preview.expiresAt).getTime() <= Date.now()) return;
42: submittingRef.current = true;
43: setSubmitting(true);
44: try {
45: await startDeployTask(taskId, { ...payload, intentVersion: preview.intentVersion }, requestKey.current);
46: await onStarted();
47: } catch (error) {
48: setPreview(null);
49: setState(isForbiddenRequestError(error) ? 'forbidden' : 'stale');
50: } finally { submittingRef.current = false; setSubmitting(false); }
51: };
52: const filteredChanges = (preview?.changes || []).filter((line) => `${line.field} ${line.before || ''} ${line.after || ''}`.toLocaleLowerCase().includes(keyword.toLocaleLowerCase()));
53: const labels = {
54: loading: t('common.loading'), empty: t('common.noData'), error: t('business.deploy.preview.loadFailed'),
55: forbidden: t('business.deploy.preview.forbidden'), stale: t('business.deploy.preview.stale'),
56: partial: t('business.deploy.preview.partial'), ready: t('business.deploy.preview.ready'),
57: kindLabels: { added: t('business.deploy.preview.added'), removed: t('business.deploy.preview.removed'), modified: t('business.deploy.preview.modified'), unchanged: t('business.deploy.preview.unchanged'), conflict: t('business.deploy.preview.conflict') },
58: before: t('business.deploy.preview.before'), after: t('business.deploy.preview.after'), sensitive: t('business.deploy.preview.sensitive'),
59: unchangedHidden: t('business.deploy.preview.unchangedHidden'), expandUnchanged: t('business.deploy.preview.expandUnchanged'), collapseUnchanged: t('business.deploy.preview.collapseUnchanged'),
60: };
61: return <AppModal title={t('business.deploy.preview.title')} visible size="detail" onCancel={submitting ? undefined : onClose} footer={null}>
62: <Space direction="vertical" size={16} style={{ width: '100%' }}>
63: {preview && <Typography.Paragraph>{t('business.deploy.preview.scopeSummary', { scope: preview.businessScopeName, count: preview.targetCount })}</Typography.Paragraph>}
64: {preview && <Input.Search value={keyword} aria-label={t('business.deploy.preview.search')} placeholder={t('business.deploy.preview.search')} allowClear onChange={(value) => { setKeyword(value); setPage(1); }} />}
65: <ChangeDiff title={t('business.deploy.preview.changes')} state={state} labels={labels}
66: lines={filteredChanges.slice((page - 1) * 100, page * 100).map((line) => ({ key: line.id, label: line.field, kind: line.kind, before: line.before || '—', after: line.after || '—' }))}
67: riskSummary={<Space direction="vertical" size={8}>{preview?.riskKeys.map((key) => <Alert key={key} type="warning" content={t(key)} />)}</Space>} />
68: {filteredChanges.length > 100 && <Pagination simple current={page} pageSize={100} total={filteredChanges.length} onChange={setPage} />}
69: <Space wrap>
70: <Button disabled={submitting} onClick={onClose}>{t('common.cancel')}</Button>
71: <Button disabled={submitting || state === 'loading'} onClick={refresh}>{t('business.deploy.preview.refresh')}</Button>
72: <Button type="primary" loading={submitting} disabled={state !== 'ready' || !preview} onClick={() => void confirm()}>{t('business.deploy.preview.confirm')}</Button>
73: </Space>
74: </Space>
75: </AppModal>;
76: }

FILE pantheon-base/backend/pkg/permissionroutes/registry.go
1: // Package permissionroutes owns the process-scoped, provider-owned API binding
2: // registry. Providers register immutable business route bindings during startup;
3: // IAM consumes copies for role synchronization and workbench coverage.
4: package permissionroutes
5:
6: import (
7: "errors"
8: "regexp"
9: "sort"
10: "strings"
11: "sync"
12: )
13:
14: type Route struct {
15: Method string
16: Path string
17: }
18: type Binding struct {
19: Permission string
20: Routes []Route
21: }
22:
23: var ErrInvalid = errors.New("permission_route.invalid")
24: var ErrConflict = errors.New("permission_route.conflict")
25: var modulePattern = regexp.MustCompile(`^business:[a-z][a-z0-9-]*$`)
26: var permissionPattern = regexp.MustCompile(`^business:[a-z][a-z0-9-]*:[a-z][a-z0-9:-]*$`)
27: var segmentPattern = regexp.MustCompile(`^(?:[a-zA-Z0-9_-]+|:[a-zA-Z_][a-zA-Z0-9_]*)$`)
28: var registry = struct {
29: sync.RWMutex
30: bindings map[string][]Route
31: }{bindings: make(map[string][]Route)}
32:
33: // Register adds a complete provider declaration. Invalid/conflicting batches
34: // leave the registry unchanged. Repeating exactly the same binding is harmless.
35: // This does not grant a role: only assigned keys create desired API policies.
36: func Register(module string, bindings []Binding) error {
37: if !modulePattern.MatchString(module) || len(bindings) == 0 || len(bindings) > 10000 {
38: return ErrInvalid
39: }
40: domain := strings.TrimPrefix(module, "business:")
41: prefix := "/api/v1/business/" + domain
42: batch := make(map[string][]Route, len(bindings))
43: for _, binding := range bindings {
44: if !permissionPattern.MatchString(binding.Permission) || !strings.HasPrefix(binding.Permission, module+":") || len(binding.Routes) == 0 || len(binding.Routes) > 1000 {
45: return ErrInvalid
46: }
47: if _, duplicate := batch[binding.Permission]; duplicate {
48: return ErrInvalid
49: }
50: routes := make([]Route, 0, len(binding.Routes))
51: seen := make(map[Route]bool)
52: for _, route := range binding.Routes {
53: if !(route.Path == prefix || strings.HasPrefix(route.Path, prefix+"/")) || !validPath(route.Path) {
54: return ErrInvalid
55: }
56: switch route.Method {
57: case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
58: default:
59: return ErrInvalid
60: }
61: if !seen[route] {
62: routes = append(routes, route)
63: seen[route] = true
64: }
65: }
66: sort.Slice(routes, func(i, j int) bool {
67: if routes[i].Path == routes[j].Path {
68: return routes[i].Method < routes[j].Method
69: }
70: return routes[i].Path < routes[j].Path
71: })
72: batch[binding.Permission] = routes
73: }
74: registry.Lock()
75: defer registry.Unlock()
76: for key, routes := range batch {
77: if existing, ok := registry.bindings[key]; ok {
78: if len(existing) != len(routes) {
79: return ErrConflict
80: }
81: for i, route := range routes {
82: if existing[i] != route {
83: return ErrConflict
84: }
85: }
86: }
87: }
88: for key, routes := range batch {
89: registry.bindings[key] = routes
90: }
91: return nil
92: }
93:
94: // Snapshot returns a deep copy; callers cannot mutate the authoritative map.
95: func Snapshot() map[string][]Route {
96: registry.RLock()
97: defer registry.RUnlock()
98: out := make(map[string][]Route, len(registry.bindings))
99: for key, routes := range registry.bindings {
100: out[key] = append([]Route(nil), routes...)
101: }
102: return out
103: }
104:
105: // Lookup copies only one key for coverage checks of individual permissions.
106: func Lookup(key string) []Route {
107: registry.RLock()
108: defer registry.RUnlock()
109: return append([]Route(nil), registry.bindings[key]...)
110: }
111:
112: func validPath(path string) bool {
113: if !strings.HasPrefix(path, "/api/v1/business/") {
114: return false
115: }
116: for _, segment := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
117: if !segmentPattern.MatchString(segment) {
118: return false
119: }
120: }
121: return true
122: }

FILE pantheon-base/backend/modules/system/iam/role/role_policy_sync.go
1: package role
2:
3: import (
4: "strings"
5:
6: "github.com/casbin/casbin/v2/util"
7: "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
8: "github.com/duanxldragon/pantheon-base/backend/pkg/database"
9:
10: "gorm.io/gorm"
11: "gorm.io/gorm/clause"
12: )
13:
14: type roleAPIRoute struct {
15: path string
16: method string
17: }
18:
19: // syncRoleAPIPolicies keeps the Casbin policies of one role in step with the
20: // permission keys the role currently holds. Only routes declared by the shared
21: // permission→route binding table are managed here: policies outside that table
22: // (wildcards written by tenant provisioning, manual grants from the permission
23: // page) are never touched.
24: // syncRoleAPIPolicies reconciles one role's Casbin rows with the routes its
25: // permission keys declare. The three steps are separate functions because each
26: // one is a different question - what the binding table manages, which rows are
27: // stale, which grants are missing - and the policy set is only correct when all
28: // three read the same route maps.
29: func syncRoleAPIPolicies(tx *gorm.DB, subject string, permissionKeys []string) error {
30: subject = strings.TrimSpace(subject)
31: if subject == "" {
32: return nil
33: }
34:
35: managed, desired := rolePolicyRouteSets(permissionKeys)
36: if len(managed) == 0 {
37: return nil
38: }
39:
40: var existing []database.CasbinRule
41: if err := tx.Where("ptype = ? AND v0 = ?", "p", subject).Find(&existing).Error; err != nil {
42: return err
43: }
44:
45: if err := deleteRetiredRolePolicies(tx, existing, managed, desired); err != nil {
46: return err
47: }
48:
49: created := missingRolePolicies(subject, desired, existing)
50: if len(created) == 0 {
51: return nil
52: }
53: // Remediation and role saves can race on the same natural key; the unique
54: // index keeps one row and the loser simply observes it as already present.
55: return tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(created, 100).Error
56: }
57:
58: // rolePolicyRouteSets returns every route the shared binding table declares and
59: // the subset of those routes the given permission keys grant.
60: func rolePolicyRouteSets(permissionKeys []string) (managed, desired map[roleAPIRoute]struct{}) {
61: heldKeys := make(map[string]struct{}, len(permissionKeys))
62: for _, key := range normalizePermissionKeys(permissionKeys) {
63: heldKeys[key] = struct{}{}
64: }
65:
66: managed = make(map[roleAPIRoute]struct{})
67: desired = make(map[roleAPIRoute]struct{})
68: for permissionKey, entries := range permission.RequiredAPIRoutesByPermission() {
69: for _, entry := range entries {
70: route := roleAPIRoute{
71: path: strings.TrimSpace(entry.Path),
72: method: normalizeRolePolicyMethod(entry.Method),
73: }
74: if route.path == "" || route.method == "" {
75: continue
76: }
77: managed[route] = struct{}{}
78: if _, ok := heldKeys[permissionKey]; ok {
79: desired[route] = struct{}{}
80: }
81: }
82: }
83: return managed, desired
84: }
85:
86: // deleteRetiredRolePolicies drops the stored rows that the binding table
87: // manages but the role no longer grants. Rows outside `managed` are never
88: // touched, so wildcard and manual grants survive a role save.
89: func deleteRetiredRolePolicies(
90: tx *gorm.DB,
91: existing []database.CasbinRule,
92: managed, desired map[roleAPIRoute]struct{},
93: ) error {
94: removedIDs := make([]uint64, 0)
95: for _, policy := range existing {
96: route := roleAPIRoute{
97: path: strings.TrimSpace(policy.V1),
98: method: normalizeRolePolicyMethod(policy.V2),
99: }
100: if _, ok := managed[route]; !ok {
101: continue
102: }
103: if _, ok := desired[route]; !ok {
104: removedIDs = append(removedIDs, policy.ID)
105: }
106: }
107: if len(removedIDs) == 0 {
108: return nil
109: }
110: return tx.Where("id IN ?", removedIDs).Delete(&database.CasbinRule{}).Error
111: }
112:
113: // missingRolePolicies builds the rows to insert for granted routes the role has
114: // no policy for yet.
115: func missingRolePolicies(
116: subject string,
117: desired map[roleAPIRoute]struct{},
118: existing []database.CasbinRule,
119: ) []database.CasbinRule {
120: present := make(map[roleAPIRoute]struct{}, len(existing))
121: for _, policy := range existing {
122: route := roleAPIRoute{
123: path: strings.TrimSpace(policy.V1),
124: method: normalizeRolePolicyMethod(policy.V2),
125: }
126: present[route] = struct{}{}
127: }
128:
129: created := make([]database.CasbinRule, 0, len(desired))
130: for route := range desired {
131: if _, ok := present[route]; ok {
132: continue
133: }
134: // A wildcard grant (`/api/v1/*`) already enforces this route; adding an
135: // exact duplicate would only bloat the policy table.
136: if rolePolicyCoversRoute(existing, route) {
137: continue
138: }
139: created = append(created, database.CasbinRule{
140: PType: "p",
141: V0: subject,
142: V1: route.path,
143: V2: route.method,
144: })
145: }
146: return created
147: }
148:
149: func normalizeRolePolicyMethod(method string) string {
150: return strings.ToUpper(strings.TrimSpace(method))
151: }
152:
153: // rolePolicyCoversRoute reports whether an existing policy already authorises
154: // the route under the Casbin keyMatch2 matcher used by the enforcement model.
155: func rolePolicyCoversRoute(policies []database.CasbinRule, route roleAPIRoute) bool {
156: for _, policy := range policies {
157: if normalizeRolePolicyMethod(policy.V2) != route.method {
158: continue
159: }
160: if util.KeyMatch2(route.path, strings.TrimSpace(policy.V1)) {
161: return true
162: }
163: }
164: return false
165: }

FILE pantheon-base/frontend/src/modules/platform/OperationalDashboardWidgets.tsx
1: import { useEffect, useRef, useState } from 'react';
2: import { Button, Card, Typography } from '@arco-design/web-react';
3: import { useTranslation } from 'react-i18next';
4: import { useNavigate } from 'react-router-dom';
5: import { DateTimeMeta, PageEmpty, PageLoading, PageRequestError } from '../../components';
6: import { isForbiddenRequestError } from '../../api/request';
7: import type { DashboardOperationalSnapshot, DashboardOperationalWidget } from '../../core/router/types';
8: import './operational-dashboard.css';
9:
10: function validateSnapshot(snapshot: DashboardOperationalSnapshot, maxItems: number) {
11: if (!snapshot || !Number.isFinite(Date.parse(snapshot.updatedAt)) ||
12: Date.parse(snapshot.updatedAt) > Date.now() + 60000 || !Array.isArray(snapshot.items) ||
13: snapshot.items.length > maxItems || (snapshot.timeRangeLabel?.length ?? 0) > 240) {
14: throw new Error('Invalid dashboard provider snapshot');
15: }
16: const ids = new Set<string>();
17: for (const item of snapshot.items) {
18: if (!item || typeof item.id !== 'string' || !item.id || ids.has(item.id) ||
19: typeof item.label !== 'string' || item.label.length > 240 || (item.detail?.length ?? 0) > 480 ||
20: (item.value !== undefined && typeof item.value !== 'string' && typeof item.value !== 'number') ||
21: (typeof item.value === 'number' && !Number.isFinite(item.value)) ||
22: (typeof item.value === 'string' && item.value.length > 240) ||
23: (item.query && (Object.keys(item.query).length > 12 || Object.entries(item.query).some(([key, value]) =>
24: key.length > 80 || typeof value !== 'string' || value.length > 240)))) {
25: throw new Error('Invalid dashboard provider item');
26: }
27: ids.add(item.id);
28: }
29: return snapshot;
30: }
31:
32: function OperationalWidget({ widget }: { widget: DashboardOperationalWidget }) {
33: const { t } = useTranslation();
34: const navigate = useNavigate();
35: const [snapshot, setSnapshot] = useState<DashboardOperationalSnapshot | null>(null);
36: const [error, setError] = useState<unknown>(null);
37: const [loading, setLoading] = useState(true);
38: const [denied, setDenied] = useState(false);
39: const [clock, setClock] = useState(Date.now);
40: const [retryKey, setRetryKey] = useState(0);
41: const nextAllowedAt = useRef(0);
42: const metadata = widget.metadata!;
43: useEffect(() => {
44: let active = true;
45: let controller: AbortController | undefined;
46: let retry: ReturnType<typeof setTimeout> | undefined;
47: const source = widget.dataSource;
48: if (!source) return;
49: const refresh = Math.max(30000, 60000 / metadata.queryBudget.maxRequestsPerMinute,
50: metadata.freshness.refreshIntervalMs ?? 30000);
51: const load = async () => {
52: const now = Date.now();
53: if (now < nextAllowedAt.current) { retry = setTimeout(load, nextAllowedAt.current - now); return; }
54: nextAllowedAt.current = now + refresh;
55: let stopPolling = false;
56: controller = new AbortController();
57: const signal = controller.signal;
58: let deadline: ReturnType<typeof setTimeout> | undefined;
59: try {
60: const timeout = new Promise<never>((_, reject) => {
61: deadline = setTimeout(() => { controller?.abort(); reject(new Error('Dashboard provider timeout')); }, source.timeoutMs);
62: });
63: const data = await Promise.race([source.load({ signal, maxItems: metadata.queryBudget.maxItems }), timeout]);
64: if (!active) return;
65: setSnapshot(validateSnapshot(data, metadata.queryBudget.maxItems));
66: setError(null); setClock(Date.now());
67: } catch (reason) {
68: if (!active) return;
69: if (isForbiddenRequestError(reason)) { setSnapshot(null); setDenied(true); stopPolling = true; controller?.abort(); return; }
70: setError(reason);
71: } finally {
72: if (deadline) clearTimeout(deadline);
73: if (active) {
74: setLoading(false);
75: if (!stopPolling && metadata.freshness.refreshIntervalMs !== undefined) retry = setTimeout(load, refresh);
76: }
77: }
78: };
79: // StrictMode replays setup/cleanup before this microtask. A canceled setup
80: // must not spend the initial request budget or defer first data by a minute.
81: void Promise.resolve().then(() => { if (active) void load(); });
82: return () => { active = false; controller?.abort(); if (retry) clearTimeout(retry); };
83: }, [widget, metadata, retryKey]);
84: useEffect(() => {
85: if (!snapshot || !metadata.freshness.maxAgeMs) return;
86: const remaining = Date.parse(snapshot.updatedAt) + metadata.freshness.maxAgeMs - Date.now();
87: if (remaining <= 0) return;
88: const timer = setTimeout(() => setClock(Date.now()), Math.min(remaining + 1, 2147483647));
89: return () => clearTimeout(timer);
90: }, [snapshot, metadata.freshness.maxAgeMs]);
91: if (denied) return null;
92: const stale = snapshot && metadata.freshness.maxAgeMs !== undefined && clock - Date.parse(snapshot.updatedAt) > metadata.freshness.maxAgeMs;
93: return <Card className="operational-dashboard-widget" title={t(widget.titleKey)} data-slot={widget.slot}>
94: {loading && !snapshot ? <PageLoading /> : null}
95: {error ? <PageRequestError error={error} description={t(metadata.errorStateKey)} onRetry={() => setRetryKey(value => value + 1)} /> : null}
96: {snapshot ? <>
97: <div className="operational-dashboard-widget__freshness">
98: <DateTimeMeta value={snapshot.updatedAt} showRelative={false} />
99: {snapshot.timeRangeLabel ? <Typography.Text type="secondary">{snapshot.timeRangeLabel}</Typography.Text> : null}
100: {stale ? <Typography.Text>{t('dashboard.operational.stale')}</Typography.Text> : null}
101: {snapshot.partial ? <Typography.Text>{t('dashboard.operational.partial')}</Typography.Text> : null}
102: </div>
103: {snapshot.items.length === 0 ? <PageEmpty description={t(metadata.emptyStateKey)} /> : <ul className="operational-dashboard-widget__items">
104: {snapshot.items.slice(0, metadata.queryBudget.maxRenderItems ?? metadata.queryBudget.maxItems).map(item => <li key={item.id}>
105: <Button type="text" onClick={() => navigate(widget.path + (item.query ? '?' + new URLSearchParams(item.query).toString() : ''))}>
106: <span>{item.label}</span>{item.value !== undefined ? <strong>{item.value}</strong> : null}
107: </Button>
108: {item.detail ? <Typography.Text type="secondary">{item.detail}</Typography.Text> : null}
109: </li>)}
110: </ul>}
111: </> : null}
112: </Card>;
113: }
114:
115: const priority = { 'attention-queue': 0, 'status-summary': 1, 'trend-snapshot': 2, 'recent-activity': 3 };
116: export default function OperationalDashboardWidgets({ widgets }: { widgets: readonly DashboardOperationalWidget[] }) {
117: const supported = widgets.filter(widget => widget.dataSource).slice().sort((a,b) => priority[a.slot] - priority[b.slot]).slice(0, 8);
118: if (!supported.length) return null;
119: return <section className="operational-dashboard-widgets">{supported.map(widget => <OperationalWidget key={widget.key} widget={widget} />)}</section>;
120: }

FILE pantheon-base/frontend/src/modules/platform/widgets.tsx
1: import type { MenuNode } from '../../api/menu';
2: import type { PlanFeatureKey } from '../../core/entitlements/plans';
3: import { registeredModules } from '../../core/router/modules';
4: import type {
5: DashboardDomainOverviewWidget,
6: DashboardOperationalWidget,
7: DashboardWidgetCleanupPolicy,
8: DashboardQuickActionWidget,
9: DashboardWidgetDefinition,
10: DashboardWidgetOperationalMetadata,
11: DashboardWidgetSlot,
12: } from '../../core/router/types';
13:
14: interface DashboardWidgetModuleLike {
15: name: string;
16: dashboardWidgets?: DashboardWidgetDefinition[];
17: }
18:
19: interface DashboardWidgetVisibilityContext {
20: menuTree: MenuNode[];
21: hasPerm: (permission: string) => boolean;
22: isAdmin: boolean;
23: /** Omitted by callers outside the authenticated shell; then no plan gate applies. */
24: isFeatureEnabled?: (feature: PlanFeatureKey) => boolean;
25: }
26:
27: function findMenuNodeByPath(nodes: MenuNode[], path: string): MenuNode | undefined {
28: for (const item of nodes) {
29: if (item.path === path || item.activeMenu === path) {
30: return item;
31: }
32: if (item.children?.length) {
33: const child = findMenuNodeByPath(item.children, path);
34: if (child) {
35: return child;
36: }
37: }
38: }
39: return undefined;
40: }
41:
42: const LEGACY_DASHBOARD_SLOTS = new Set<DashboardWidgetSlot>(['quick-action', 'domain-overview']);
43: const OPERATIONAL_DASHBOARD_SLOTS = new Set<DashboardWidgetSlot>([
44: 'status-summary',
45: 'attention-queue',
46: 'trend-snapshot',
47: 'recent-activity',
48: ]);
49: const MIN_REFRESH_INTERVAL_MS = 30_000;
50: const MAX_REQUESTS_PER_MINUTE = 12;
51: const MAX_WIDGET_ITEMS = 100;
52: const MAX_RENDER_ITEMS = 24;
53:
54: function isOperationalDashboardSlot(slot: DashboardWidgetSlot) {
55: return OPERATIONAL_DASHBOARD_SLOTS.has(slot);
56: }
57:
58: function assertTrimmed(value: string | undefined, message: string) {
59: if (!value?.trim()) {
60: throw new Error(message);
61: }
62: }
63:
64: function assertCleanupPolicy(
65: widget: DashboardWidgetDefinition,
66: cleanupPolicy: DashboardWidgetCleanupPolicy,
67: ) {
68: if (widget.sourceDomain.startsWith('business/')) {
69: if (cleanupPolicy !== 'remove_with_source_module') {
70: throw new Error(
71: `Business dashboard widget "${widget.key}" must declare remove_with_source_module cleanup.`,
72: );
73: }
74: return;
75: }
76:
77: if (cleanupPolicy === 'remove_with_source_module') {
78: throw new Error(
79: `Platform/system dashboard widget "${widget.key}" must not use business cleanup policy.`,
80: );
81: }
82: }
83:
84: function assertOperationalMetadata(
85: widget: DashboardWidgetDefinition,
86: ): asserts widget is DashboardOperationalWidget {
87: if (!isOperationalDashboardSlot(widget.slot)) {
88: return;
89: }
90: if (!widget.permission) {
91: throw new Error(`Operational dashboard widget "${widget.key}" must declare a permission.`);
92: }
93: if ('dataSource' in widget && widget.dataSource && (typeof widget.dataSource.load !== 'function' ||
94: !Number.isFinite(widget.dataSource.timeoutMs) || widget.dataSource.timeoutMs < 1000 || widget.dataSource.timeoutMs > 10000)) {
95: throw new Error(`Operational dashboard widget "${widget.key}" must use a bounded data source.`);
96: }
97: const metadata = widget.metadata;
98: if (!metadata) {
99: throw new Error(`Operational dashboard widget "${widget.key}" must declare metadata.`);
100: }
101:
102: assertTrimmed(metadata.owner, `Operational dashboard widget "${widget.key}" must declare owner.`);
103: assertCleanupPolicy(widget, metadata.cleanupPolicy);
104: assertTrimmed(
105: metadata.emptyStateKey,
106: `Operational dashboard widget "${widget.key}" must declare emptyStateKey.`,
107: );
108: assertTrimmed(
109: metadata.errorStateKey,
110: `Operational dashboard widget "${widget.key}" must declare errorStateKey.`,
111: );
112:
113: if (metadata.errorIsolation !== 'widget') {
114: throw new Error(
115: `Operational dashboard widget "${widget.key}" must isolate errors at widget level.`,
116: );
117: }
118: if (metadata.cleanupPolicy !== widget.cleanupPolicy) {
119: throw new Error(
120: `Operational dashboard widget "${widget.key}" cleanup policy must match metadata.`,
121: );
122: }
123: if (metadata.freshness.maxAgeMs !== undefined &&
124: (!Number.isFinite(metadata.freshness.maxAgeMs) || metadata.freshness.maxAgeMs <= 0)) {
125: throw new Error(`Operational dashboard widget "${widget.key}" must declare a positive freshness age.`);
126: }
127: if (metadata.freshness.refreshIntervalMs !== undefined) {
128: if (
129: !Number.isFinite(metadata.freshness.refreshIntervalMs) ||
130: metadata.freshness.refreshIntervalMs < MIN_REFRESH_INTERVAL_MS
131: ) {
132: throw new Error(
133: `Operational dashboard widget "${widget.key}" refresh interval is below query budget floor.`,
134: );
135: }
136: }
137: if (
138: !Number.isFinite(metadata.queryBudget.maxRequestsPerMinute) ||
139: metadata.queryBudget.maxRequestsPerMinute <= 0 ||
140: metadata.queryBudget.maxRequestsPerMinute > MAX_REQUESTS_PER_MINUTE
141: ) {
142: throw new Error(`Operational dashboard widget "${widget.key}" exceeds request budget.`);
143: }
144: if (
145: !Number.isFinite(metadata.queryBudget.maxItems) ||
146: metadata.queryBudget.maxItems <= 0 ||
147: metadata.queryBudget.maxItems > MAX_WIDGET_ITEMS
148: ) {
149: throw new Error(`Operational dashboard widget "${widget.key}" exceeds item budget.`);
150: }
151: const maxRenderItems = metadata.queryBudget.maxRenderItems ?? metadata.queryBudget.maxItems;
152: if (!Number.isFinite(maxRenderItems) || maxRenderItems <= 0 || maxRenderItems > MAX_RENDER_ITEMS) {
153: throw new Error(`Operational dashboard widget "${widget.key}" exceeds render budget.`);
154: }
155: }
156:
157: function assertDashboardWidgetDefinition(widget: DashboardWidgetDefinition) {
158: if (!widget.key.trim()) {
159: throw new Error('Dashboard widget key is required.');
160: }
161: if (!widget.path.startsWith('/')) {
162: throw new Error(`Dashboard widget "${widget.key}" must use an absolute route path.`);
163: }
164: if (!LEGACY_DASHBOARD_SLOTS.has(widget.slot) && !OPERATIONAL_DASHBOARD_SLOTS.has(widget.slot)) {
165: throw new Error(`Dashboard widget "${widget.key}" uses an unknown slot.`);
166: }
167: assertCleanupPolicy(widget, widget.cleanupPolicy);
168: if (widget.sourceDomain.startsWith('business/')) {
169: if (!widget.permission) {
170: throw new Error(`Business dashboard widget "${widget.key}" must declare a permission.`);
171: }
172: if (!widget.registrationOwner?.trim()) {
173: throw new Error(
174: `Business dashboard widget "${widget.key}" must declare a registration owner.`,
175: );
176: }
177: }
178: assertOperationalMetadata(widget);
179: }
180:
181: export function buildDashboardWidgetRegistry(
182: modules: DashboardWidgetModuleLike[],
183: ): DashboardWidgetDefinition[] {
184: const widgets: DashboardWidgetDefinition[] = [];
185: const keys = new Set<string>();
186:
187: modules.forEach((module) => {
188: module.dashboardWidgets?.forEach((widget) => {
189: assertDashboardWidgetDefinition(widget);
190: if (keys.has(widget.key)) {
191: throw new Error(
192: `Duplicate dashboard widget key "${widget.key}" declared by module "${module.name}".`,
193: );
194: }
195: keys.add(widget.key);
196: widgets.push(widget);
197: });
198: });
199:
200: return widgets;
201: }
202:
203: export function isDashboardWidgetVisible(
204: widget: DashboardWidgetDefinition,
205: context: DashboardWidgetVisibilityContext,
206: ) {
207: if (
208: widget.feature &&
209: context.isFeatureEnabled &&
210: !context.isFeatureEnabled(widget.feature)
211: ) {
212: return false;
213: }
214: const hasAccess = !widget.permission || context.isAdmin || context.hasPerm(widget.permission);
215: if (!hasAccess) {
216: return false;
217: }
218: if (widget.navigationSource === 'direct') {
219: return true;
220: }
221: return Boolean(findMenuNodeByPath(context.menuTree, widget.path));
222: }
223:
224: export function getVisibleDashboardWidgets<T extends DashboardWidgetDefinition>(
225: widgets: readonly T[],
226: context: DashboardWidgetVisibilityContext,
227: ) {
228: return widgets.filter((widget) => isDashboardWidgetVisible(widget, context));
229: }
230:
231: export function getDashboardWidgetMetadata(widget: DashboardOperationalWidget) {
232: return widget.metadata as DashboardWidgetOperationalMetadata;
233: }
234:
235: export const dashboardWidgetRegistry = buildDashboardWidgetRegistry(registeredModules);
236:
237: export const dashboardQuickActionWidgets = dashboardWidgetRegistry.filter(
238: (widget): widget is DashboardQuickActionWidget => widget.slot === 'quick-action',
239: );
240:
241: export const dashboardDomainOverviewWidgets = dashboardWidgetRegistry.filter(
242: (widget): widget is DashboardDomainOverviewWidget => widget.slot === 'domain-overview',
243: );
244:
245: export const dashboardStatusSummaryWidgets = dashboardWidgetRegistry.filter(
246: (widget): widget is DashboardOperationalWidget => widget.slot === 'status-summary',
247: );
248:
249: export const dashboardAttentionQueueWidgets = dashboardWidgetRegistry.filter(
250: (widget): widget is DashboardOperationalWidget => widget.slot === 'attention-queue',
251: );
252:
253: export const dashboardTrendSnapshotWidgets = dashboardWidgetRegistry.filter(
254: (widget): widget is DashboardOperationalWidget => widget.slot === 'trend-snapshot',
255: );
256:
257: export const dashboardRecentActivityWidgets = dashboardWidgetRegistry.filter(
258: (widget): widget is DashboardOperationalWidget => widget.slot === 'recent-activity',
259: );

FILE pantheon-base/frontend/src/core/router/types.ts
1: import type { LazyExoticComponent, ComponentType } from 'react';
2: import type { RegisteredComponentKey } from './componentRegistry';
3: import type { MenuIconKey } from '../menu/icon';
4: import type { PlanFeatureKey } from '../entitlements/plans';
5: import type { TFunction } from 'i18next';
6:
7: /** 消费方业务组件覆盖注册表：字符串 key 映射到懒加载组件。 _/
8: export type ComponentRegistryMap = Record<string, LazyExoticComponent<ComponentType>>;
9:
10: export type ModuleScope = 'platform' | 'system' | 'business' | 'lowcode';
11:
12: export type DashboardWidgetSourceDomain =
13: | 'platform'
14: | 'system/auth'
15: | 'system/iam'
16: | 'system/org'
17: | 'system/config'
18: | 'system/lowcode'
19: | 'system/audit'
20: | `business/${string}`;
21:
22: export type DashboardWidgetCleanupPolicy =
23: 'platform_owned' | 'hide_when_forbidden' | 'remove_with_source_module';
24:
25: export type DashboardWidgetSlot =
26: | 'quick-action'
27: | 'domain-overview'
28: | 'status-summary'
29: | 'attention-queue'
30: | 'trend-snapshot'
31: | 'recent-activity';
32: export type DashboardWidgetNavigationSource = 'menu' | 'direct';
33: export type DashboardWidgetFreshnessPolicy = 'static' | 'snapshot' | 'near_realtime';
34:
35: export interface DashboardWidgetQueryBudget {
36: maxRequestsPerMinute: number;
37: maxItems: number;
38: maxRenderItems?: number;
39: }
40:
41: export interface DashboardWidgetFreshness {
42: policy: DashboardWidgetFreshnessPolicy;
43: maxAgeMs?: number;
44: refreshIntervalMs?: number;
45: timeRangeKey?: string;
46: }
47:
48: export interface DashboardWidgetOperationalMetadata {
49: owner: string;
50: cleanupPolicy: DashboardWidgetCleanupPolicy;
51: freshness: DashboardWidgetFreshness;
52: queryBudget: DashboardWidgetQueryBudget;
53: emptyStateKey: string;
54: errorStateKey: string;
55: errorIsolation: 'widget';
56: }
57:
58: export interface DashboardSummarySnapshot {
59: totalUsers?: number;
60: totalRoles?: number;
61: totalDepts?: number;
62: totalPosts?: number;
63: totalDictTypes?: number;
64: totalSettings?: number;
65: totalI18nEntries?: number;
66: activeModuleCount?: number;
67: pendingSecurityEventCount?: number;
68: totalSecurityEventCount?: number;
69: todayOperationCount?: number;
70: loginFailureCount?: number;
71: }
72:
73: /_* Safe, bounded provider DTO; credentials and raw query/config data stay in the provider. _/
74: export interface DashboardOperationalItem {
75: id: string;
76: label: string;
77: value?: string | number;
78: detail?: string;
79: query?: Record<string, string>;
80: }
81: export interface DashboardOperationalSnapshot {
82: updatedAt: string;
83: timeRangeLabel?: string;
84: partial?: boolean;
85: items: DashboardOperationalItem[];
86: }
87: export interface DashboardOperationalDataSource {
88: timeoutMs: number;
89: load: (request: { signal: AbortSignal; maxItems: number }) => Promise<DashboardOperationalSnapshot>;
90: }
91:
92: interface DashboardWidgetBase {
93: key: string;
94: slot: DashboardWidgetSlot;
95: sourceDomain: DashboardWidgetSourceDomain;
96: titleKey: string;
97: descriptionKey: string;
98: path: string;
99: permission?: string;
100: /_* Minimum tenant plan that exposes this widget. Omit for every plan. _/
101: feature?: PlanFeatureKey;
102: cleanupPolicy: DashboardWidgetCleanupPolicy;
103: navigationSource?: DashboardWidgetNavigationSource;
104: registrationOwner?: string;
105: metadata?: DashboardWidgetOperationalMetadata;
106: }
107:
108: export interface DashboardQuickActionWidget extends DashboardWidgetBase {
109: slot: 'quick-action';
110: icon: string;
111: }
112:
113: export interface DashboardDomainOverviewWidget extends DashboardWidgetBase {
114: slot: 'domain-overview';
115: summary: (summary: DashboardSummarySnapshot | null, t: TFunction) => string;
116: }
117:
118: export interface DashboardStatusSummaryWidget extends DashboardWidgetBase {
119: dataSource?: DashboardOperationalDataSource;
120: slot: 'status-summary';
121: }
122:
123: export interface DashboardAttentionQueueWidget extends DashboardWidgetBase {
124: dataSource?: DashboardOperationalDataSource;
125: slot: 'attention-queue';
126: }
127:
128: export interface DashboardTrendSnapshotWidget extends DashboardWidgetBase {
129: dataSource?: DashboardOperationalDataSource;
130: slot: 'trend-snapshot';
131: }
132:
133: export interface DashboardRecentActivityWidget extends DashboardWidgetBase {
134: dataSource?: DashboardOperationalDataSource;
135: slot: 'recent-activity';
136: }
137:
138: export type DashboardOperationalWidget =
139: | DashboardStatusSummaryWidget
140: | DashboardAttentionQueueWidget
141: | DashboardTrendSnapshotWidget
142: | DashboardRecentActivityWidget;
143:
144: export type DashboardWidgetDefinition =
145: DashboardQuickActionWidget | DashboardDomainOverviewWidget | DashboardOperationalWidget;
146:
147: export interface RouteDataWarmer {
148: path: string;
149: key: string;
150: load: () => Promise<unknown>;
151: ttlMs?: number;
152: }
153:
154: interface ModuleRouteConfigBase {
155: path: string;
156: routeName?: string;
157: titleKey: string;
158: resolveTitleKey?: (path: string) => string | undefined;
159: icon?: MenuIconKey;
160: isCache?: boolean;
161: activeMenu?: string;
162: pagePermission?: string;
163: /_* Minimum tenant plan that exposes this page. Omit for every plan. */
164: feature?: PlanFeatureKey;
165: }
166:
167: export type ModuleRouteConfig = ModuleRouteConfigBase &
168: (
169: | {
170: component: LazyExoticComponent<ComponentType>;
171: componentKey?: RegisteredComponentKey;
172: }
173: | {
174: component?: undefined;
175: componentKey: RegisteredComponentKey;
176: }
177: );
178:
179: export interface ModuleMenuMeta {
180: titleKey: string;
181: path: string;
182: icon?: MenuIconKey;
183: routeName?: string;
184: module?: string;
185: isCache?: boolean;
186: isExternal?: boolean;
187: activeMenu?: string;
188: }
189:
190: export interface ModuleConfig {
191: name: string;
192: scope: ModuleScope;
193: routes: ModuleRouteConfig[];
194: menus?: ModuleMenuMeta[];
195: routeDataWarmers?: RouteDataWarmer[];
196: dashboardWidgets?: DashboardWidgetDefinition[];
197: permissions?: string[];
198: i18nNamespaces?: string[];
199: featureFlags?: string[];
200: }
201:
202: export function defineModule(config: ModuleConfig): ModuleConfig {
203: return config;
204: }

FILE pantheon-base/frontend/src/components/operational/ConditionBuilder.tsx
1: import React from 'react';
2: import { Button, Input, Select, Space, Tag } from '@arco-design/web-react';
3: import { validateConditionAst } from './conditionAst';
4: import type { ConditionAstNode, ConditionFieldOption, ConditionRule, OperationalDataState } from './types';
5: import './operational.css';
6:
7: export interface ConditionBuilderProps {
8: title: React.ReactNode;
9: value: ConditionAstNode;
10: fields: ConditionFieldOption[];
11: labels: {
12: and: React.ReactNode;
13: or: React.ReactNode;
14: operatorLabels: Record<string, React.ReactNode> & { unknown: React.ReactNode };
15: addRule: React.ReactNode;
16: invalidField: React.ReactNode;
17: unserializableValue: React.ReactNode;
18: } & Record<OperationalDataState, React.ReactNode>;
19: state?: OperationalDataState;
20: onChange?: (value: ConditionAstNode) => void;
21: onValidationChange?: (invalidIds: string[]) => void;
22: editorLabels?: { field: string; operator: string; value: string; remove: React.ReactNode; addGroup: React.ReactNode; combinator: string };
23: renderValueEditor?: (rule: ConditionRule, onValueChange: (value: unknown) => void) => React.ReactNode;
24: }
25:
26: function formatConditionValue(value: unknown, unserializableValue: React.ReactNode): React.ReactNode {
27: if (value === null || value === undefined) {
28: return '';
29: }
30: if (typeof value === 'string') {
31: return value;
32: }
33: if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
34: return value.toString();
35: }
36: if (typeof value === 'symbol') {
37: return value.description ?? unserializableValue;
38: }
39: if (typeof value === 'function') {
40: return value.name || unserializableValue;
41: }
42: try {
43: return JSON.stringify(value) ?? unserializableValue;
44: } catch {
45: return unserializableValue;
46: }
47: }
48:
49: function renderNode(
50: node: ConditionAstNode,
51: fields: ConditionFieldOption[],
52: invalidIds: Set<string>,
53: labels: ConditionBuilderProps['labels'],
54: ): React.ReactNode {
55: if (node.type === 'group')
56: return (
57: <div className="condition-builder__node" key={node.id}>
58: <Tag>{node.combinator === 'and' ? labels.and : labels.or}</Tag>
59: <div className="condition-builder__tree">
60: {node.children.map((child) => renderNode(child, fields, invalidIds, labels))}
61: </div>
62: </div>
63: );
64: return (
65: <div
66: className={
67: invalidIds.has(node.id)
68: ? 'condition-builder__node condition-builder__node--invalid'
69: : 'condition-builder__node'
70: }
71: key={node.id}
72: aria-invalid={invalidIds.has(node.id)}
73: >
74: <Select
75: value={node.field}
76: disabled
77: options={fields.map((field) => ({ label: field.label, value: field.key }))}
78: />
79: <Tag>{labels.operatorLabels[node.operator] ?? labels.operatorLabels.unknown}</Tag>
80: <span>{formatConditionValue(node.value, labels.unserializableValue)}</span>
81: {invalidIds.has(node.id) ? <span>{labels.invalidField}</span> : null}
82: </div>
83: );
84: }
85: export default function ConditionBuilder({
86: title,
87: value,
88: fields,
89: labels,
90: onChange,
91: onValidationChange,
92: editorLabels,
93: renderValueEditor,
94: state = 'ready',
95: }: Readonly<ConditionBuilderProps>) {
96: const invalidIds = new Set(validateConditionAst(value, fields));
97: const editable = Boolean(onChange && editorLabels && state === 'ready');
98: const changeNode = (id: string, transform: (node: ConditionAstNode) => ConditionAstNode | null) => {
99: const visit = (node: ConditionAstNode): ConditionAstNode | null => {
100: if (node.id === id) return transform(node);
101: return node.type === 'rule' ? node : { ...node, children: node.children.map(visit).filter((child): child is ConditionAstNode => child !== null) };
102: };
103: const next = visit(value);
104: if (next) { onChange?.(next); onValidationChange?.(validateConditionAst(next, fields)); }
105: };
106: const newRule = (): ConditionRule => {
107: const field = fields.find((item) => !item.disabled && item.operators.length > 0);
108: return { type: 'rule', id: crypto.randomUUID(), field: field?.key || '', operator: field?.operators[0] || '', value: '' };
109: };
110: const addRule = (id: string) => changeNode(id, (node) => node.type === 'group' ? { ...node, children: [...node.children, newRule()] } : { type: 'group', id: crypto.randomUUID(), combinator: 'and', children: [node, newRule()] });
111: const renderEditable = (node: ConditionAstNode, root = false): React.ReactNode => {
112: if (!editorLabels) return null;
113: const remove = !root && <Button onClick={() => changeNode(node.id, () => null)}>{editorLabels.remove}</Button>;
114: if (node.type === 'group') return <div className="condition-builder__node" key={node.id}>
115: <Space wrap><Select value={node.combinator} aria-label={editorLabels.combinator} onChange={(combinator: 'and' | 'or') => changeNode(node.id, (current) => current.type === 'group' ? { ...current, combinator } : current)} options={[{ value: 'and', label: labels.and }, { value: 'or', label: labels.or }]} />
116: <Button onClick={() => addRule(node.id)}>{labels.addRule}</Button>
117: <Button onClick={() => changeNode(node.id, (current) => current.type === 'group' ? { ...current, children: [...current.children, { type: 'group', id: crypto.randomUUID(), combinator: 'and', children: [newRule()] }] } : current)}>{editorLabels.addGroup}</Button>{remove}</Space>
118: <div className="condition-builder__tree">{node.children.map((child) => renderEditable(child))}</div>
119: </div>;
120: const field = fields.find((item) => item.key === node.field);
121: const changeValue = (next: unknown) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, value: next } : current);
122: return <div className={`condition-builder__node${invalidIds.has(node.id) ? ' condition-builder__node--invalid' : ''}`} key={node.id} aria-invalid={invalidIds.has(node.id)}>
123: <Select value={node.field} aria-label={editorLabels.field} options={fields.map((item) => ({ value: item.key, label: item.label, disabled: item.disabled }))} onChange={(key: string) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, field: key, operator: fields.find((item) => item.key === key)?.operators[0] || '' } : current)} />
124: <Select value={node.operator} aria-label={editorLabels.operator} options={(field?.operators || []).map((operator) => ({ value: operator, label: labels.operatorLabels[operator] ?? labels.operatorLabels.unknown }))} onChange={(operator: string) => changeNode(node.id, (current) => current.type === 'rule' ? { ...current, operator } : current)} />
125: {renderValueEditor ? renderValueEditor(node, changeValue) : <Input value={String(formatConditionValue(node.value, labels.unserializableValue))} aria-label={editorLabels.value} onChange={changeValue} />}
126: {invalidIds.has(node.id) && <span>{labels.invalidField}</span>}{remove}
127: </div>;
128: };
129: return (
130: <section className="operational-primitive condition-builder" aria-label={String(title)}>
131: <div className="operational-primitive__header">
132: <div className="operational-primitive__title">{title}</div>
133: <Button disabled={!editable} onClick={() => addRule(value.id)}>{labels.addRule}</Button>
134: </div>
135: {state !== 'ready' ? (
136: <div className="operational-primitive__state">{labels[state]}</div>
137: ) : (
138: <div className="operational-primitive__body condition-builder__tree">
139: {editable ? renderEditable(value, true) : renderNode(value, fields, invalidIds, labels)}
140: </div>
141: )}
142: </section>
143: );
144: }

EXCERPT backend/modules/business/deploy/deploy_service.go
31: type DeployService struct {
32: db *gorm.DB
33: cmdbCapability cmdb.DeployCMDBCapability
34: bizScopeReader bizcap.BizScopeReader
35: hostReader bizcap.CMDBHostReader
36: serviceState bizcap.ServiceInstanceStateApplier
37: sshRunnerFactory func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error)
38: executor DeployExecutor
39: asyncExecution bool
40: taskCancels sync.Map
41: taskWorkers sync.Map
42: }
43:
44: const deployAttemptMax = 3

EXCERPT backend/modules/business/deploy/deploy_service.go
150: func NewDeployService(db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, readers ...bizcap.BizScopeReader) *DeployService {
151: var bizScopeReader bizcap.BizScopeReader
152: if len(readers) > 0 {
153: bizScopeReader = readers[0]
154: }
155: if bizScopeReader == nil {
156: bizScopeReader = bizscope.NewService(db)
157: }
158: hostReader, _ := cmdbCapability.(bizcap.CMDBHostReader)
159: service := &DeployService{
160: db: db,
161: cmdbCapability: cmdbCapability,
162: bizScopeReader: bizScopeReader,
163: hostReader: hostReader,
164: sshRunnerFactory: newDeploySSHRunner,
165: }
166: service.executor = &sshDeployExecutor{service: service}
167: return service
168: }
169:
170: // SetAsyncExecution enables the same-process durable worker used by production
171: // module wiring. Tests can leave it disabled to exercise execution inline.
172: func (s *DeployService) SetAsyncExecution(enabled bool) { s.asyncExecution = enabled }
173:
174: // SetExecutor replaces the remote execution provider. A nil provider restores
175: // the SSH provider used by this module.
176: func (s *DeployService) SetExecutor(executor DeployExecutor) {

EXCERPT backend/modules/business/deploy/deploy_service.go
755: func (s *DeployService) updateTask(id uint64, req UpdateTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
756: task, err := s.loadVisibleTask(id, dataScope)
757: if err != nil {
758: return nil, err
759: }
760: if task.Status != TaskStatusPending && task.Status != TaskStatusDraft {
761: return nil, errors.New(errDeployTaskInvalidUpdateState)
762: }
763: // FZ-02: task updates may carry write-only secret params while the task is
764: // still draft/pending; a started task is rejected above.
765: secretParams := map[string]string{}
766: if req.SecretParams != nil {
767: if err := validateDeploySecretParams(*req.SecretParams); err != nil {
768: return nil, err
769: }
770: secretParams = *req.SecretParams
771: }
772: if req.TemplateParams != nil {
773: stripped, _ := splitDeploySecretParams(*req.TemplateParams)
774: req.TemplateParams = &stripped
775: }
776: values, err := s.buildTaskUpdateValues(task, req)
777: if err != nil {
778: return nil, err
779: }
780: template, err := s.applyTemplateToTaskUpdate(&values, req)
781: if err != nil {
782: return nil, err
783: }
784: pkg, err := s.loadTaskPackage(values.packageID, template, values.templateParams)
785: if err != nil {
786: return nil, err
787: }
788: values.targetIDs = common.NormalizeUint64IDs(values.targetIDs)
789: if len(values.targetIDs) == 0 {
790: return nil, errors.New("business.deploy.task.targetRequired")
791: }
792: scopeName, err := s.resolveTaskUpdateScopeName(&values, dataScope)
793: if err != nil {
794: return nil, err
795: }
796: if err := s.persistTaskUpdate(task, &values, pkg, template, scopeName, actor); err != nil {
797: return nil, err
798: }
799: if err := s.persistDeployTaskSecrets(task.ID, secretParams); err != nil {
800: return nil, err
801: }
802: return s.GetTask(id, dataScope)
803: }
804:
805: func (s *DeployService) buildTaskUpdateValues(task *DeployTask, req UpdateTaskRequest) (taskUpdateValues, error) {

EXCERPT backend/modules/business/deploy/deploy_service.go
1060: return errDeployTaskNotFound
1061: }
1062: return nil
1063: })
1064: }
1065:
1066: func (s *DeployService) StartTask(id uint64, req StartTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
1067: return s.StartTaskContext(context.Background(), id, req, actor, dataScope)
1068: }
1069: func (s *DeployService) StartTaskContext(ctx context.Context, id uint64, req StartTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
1070: if err := ctx.Err(); err != nil {
1071: return nil, err
1072: }
1073: reads := s.targetRequestService(ctx)
1074: task, err := reads.loadVisibleTask(id, dataScope)
1075: if err != nil {
1076: return nil, err
1077: }
1078: credentialVersion := uint64(0)
1079: key := strings.TrimSpace(req.IdempotencyKey)
1080:
1081: // Reject or replay non-draft/pending states before attempting any claim.
1082: switch task.Status {
1083: case TaskStatusRunning:
1084: if key != "" && key == task.StartRequestKey {
1085: return s.getTaskReused(id, dataScope)
1086: }
1087: return nil, errors.New(errDeployTaskAlreadyRunning)
1088: case TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled,
1089: TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested:
1090: return nil, errors.New("business.deploy.task.invalidStartState")
1091: }
1092: if task.ExecutorType == ExecutorTypeSSH {
1093: resolvedReq, version, err := reads.prepareStartCredential(req)
1094: if err != nil {
1095: return nil, err
1096: }
1097: req, credentialVersion = resolvedReq, version
1098: }
1099: hosts, err := reads.prepareStartTargets(task, req, dataScope)
1100: if err != nil {
1101: return nil, err
1102: }
1103:
1104: now := time.Now()
1105: claim := startTaskClaim{
1106: service: s, dataScope: dataScope,
1107: task: task,
1108: req: req,
1109: hosts: hosts,
1110: actor: actor,
1111: key: key,
1112: now: now,
1113: timeout: normalizeDeployExecutionTimeout(req.TimeoutSeconds),
1114: leaseOwner: deployHostLeaseOwner(task.ID),
1115: credentialVersion: credentialVersion,
1116: }
1117: claimErr := s.db.WithContext(ctx).Transaction(claim.claim)
1118: if claimErr != nil {
1119: return s.rejectFailedStartClaim(id, claimErr, key, dataScope)
1120: }
1121: task, hosts = claim.task, claim.hosts
1122: s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.started", fromStatus: task.Status, toStatus: TaskStatusRunning, actor: actor, requestID: key})
1123: if err := s.beginServiceInstanceState(*task, actor, dataScope); err != nil {
1124: s.failStartedTask(task, hosts, actor, err)
1125: return nil, err
1126: }
1127: if err := s.dispatchStartedTask(task, hosts, req, actor); err != nil {
1128: return nil, err
1129: }
1130: return s.GetTask(id, dataScope)
1131: }
1132:
1133: // prepareStartCredential resolves the SSH credential reference of a start
1134: // request into the values persisted on the task.
1135: func (s *DeployService) prepareStartCredential(req StartTaskRequest) (StartTaskRequest, uint64, error) {
1136: if err := validateCredentialReferenceRequest(req); err != nil {
1137: return req, 0, err
1138: }
1139: resolvedReq, version, err := s.resolveStartCredential(req)
1140: if err != nil {
1141: return req, 0, err
1142: }
1143: return resolvedReq, version, nil
1144: }
1145:
1146: // prepareStartTargets validates the frozen execution snapshot and resolves the
1147: // hosts the start request targets.
1148: func (s *DeployService) prepareStartTargets(task *DeployTask, req StartTaskRequest, dataScope *common.DataScopeReq) ([]cmdbHostSnapshot, error) {
1149: // Validate the frozen execution snapshot before mutating anything.
1150: if _, err := s.resolveTaskExecutionPlan(*task); err != nil {
1151: return nil, mapDeployTaskExecutionPlanError(err)
1152: }
1153: hosts, err := s.resolveTaskTargets(task, dataScope)
1154: if err != nil {
1155: return nil, err
1156: }
1157: if len(hosts) == 0 {
1158: return nil, errors.New("business.deploy.task.emptyResolvedTargets")
1159: }
1160: if task.ExecutorType == ExecutorTypeSSH {
1161: if err := validateDeploySSHStartRequest(req); err != nil {
1162: return nil, err
1163: }
1164: if err := s.validateTaskExecutionPlan(*task, hosts); err != nil {
1165: return nil, mapDeployTaskExecutionPlanError(err)
1166: }
1167: }
1168: return hosts, nil
1169: }
1170:
1171: // startTaskClaim carries the resolved start request of the claiming
1172: // transaction so the transaction keeps its inputs in one place.
1173: type startTaskClaim struct {
1174: service *DeployService
1175: dataScope *common.DataScopeReq
1176: task *DeployTask
1177: req StartTaskRequest
1178: hosts []cmdbHostSnapshot
1179: actor string
1180: key string
1181: now time.Time
1182: timeout time.Duration
1183: leaseOwner string
1184: credentialVersion uint64
1185: }
1186:
1187: // claim flips the task to running, freezes the target snapshot and creates the
1188: // per-host rows together with their leases.
1189: func (c *startTaskClaim) claim(tx *gorm.DB) error {
1190: if err := c.validatePreview(tx); err != nil {
1191: return err
1192: }
1193: result := tx.Model(&DeployTask{}).
1194: Where(idWhereClause, c.task.ID).
1195: Where("status IN ?", []string{TaskStatusDraft, TaskStatusPending}).
1196: Updates(map[string]interface{}{
1197: "status": TaskStatusRunning,
1198: "started_at": &c.now,
1199: "start_request_key": c.key,
1200: "credential_ref_id": c.req.CredentialRefID,
1201: "credential_ref_version": c.credentialVersion,
1202: "ssh_host_fingerprint": strings.TrimSpace(c.req.HostFingerprint),
1203: "execution_timeout_seconds": int(c.timeout.Seconds()),
1204: "updated_by": c.actor,
1205: "updated_at": c.now,
1206: })
1207: if result.Error != nil {
1208: return result.Error
1209: }
1210: if result.RowsAffected == 0 {
1211: return errors.New(errDeployTaskAlreadyRunning)
1212: }
1213:
1214: targetSnapshot := buildDeployTargetSnapshot(c.task, c.hosts, c.now)
1215: var selectionSnapshot deployTargetSnapshot
1216: if err := json.Unmarshal(c.task.TargetSnapshot, &selectionSnapshot); err == nil {
1217: targetSnapshot.Selection = selectionSnapshot.Selection
1218: targetSnapshot.ScopeVersion = selectionSnapshot.ScopeVersion
1219: targetSnapshot.Sources = selectionSnapshot.Sources
1220: }
1221: targetJSON, _ := json.Marshal(targetSnapshot)
1222: if err := tx.Model(&DeployTask{}).Where(idWhereClause, c.task.ID).Update("target_snapshot", datatypes.JSON(targetJSON)).Error; err != nil {
1223: return err
1224: }
1225:
1226: for _, host := range c.hosts {
1227: if err := c.createTaskHostRow(tx, host); err != nil {
1228: return err
1229: }
1230: }
1231: return nil
1232: }
1233:
1234: // createTaskHostRow acquires the host lease and persists the running host row
1235: // with the execution steps needed for checkpoint recovery.
1236: func (c startTaskClaim) createTaskHostRow(tx *gorm.DB, host cmdbHostSnapshot) error {
1237: if err := acquireHostLease(tx, host.ID, c.task.ID, c.leaseOwner, c.now); err != nil {
1238: return err
1239: }
1240: initialTrace, _ := json.Marshal([]map[string]any{
1241: {"at": c.now.Format(time.RFC3339), "phase": "start", "message": "Task host queued"},
1242: })
1243: row := DeployTaskHost{
1244: TaskID: c.task.ID,
1245: HostID: host.ID,
1246: Hostname: host.Hostname,
1247: HostIP: host.IP,
1248: SSHPort: host.SSHPort,
1249: OS: host.OS,
1250: BusinessScopeID: host.BusinessScopeID,
1251: Status: TaskHostStatusRunning,
1252: TraceSteps: datatypes.JSON(initialTrace),
1253: StartedAt: &c.now,
1254: ResolvedAt: &c.now,
1255: UpdatedBy: c.actor,
1256: }
1257: if err := tx.Create(&row).Error; err != nil {
1258: return err
1259: }
1260: // FZ-03: persist execution steps for checkpoint recovery.
1261: var executionSnapshot deployExecutionSnapshot
1262: if err := json.Unmarshal(c.task.ExecutionSnapshot, &executionSnapshot); err == nil && len(executionSnapshot.Steps) > 0 {
1263: return persistTaskSteps(tx, c.task.ID, row.ID, executionSnapshot.Steps)
1264: }
1265: return nil
1266: }
1267:
1268: // rejectFailedStartClaim replays an idempotent start that lost the claim race
1269: // and otherwise reports the claim failure.
1270: func (s *DeployService) rejectFailedStartClaim(id uint64, claimErr error, key string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
1271: if strings.TrimSpace(errorText(claimErr)) != errDeployTaskAlreadyRunning {
1272: return nil, claimErr
1273: }
1274: current, loadErr := s.loadVisibleTask(id, dataScope)
1275: if loadErr == nil && current.Status == TaskStatusRunning && key != "" && current.StartRequestKey == key {
1276: return s.getTaskReused(id, dataScope)
1277: }
1278: return nil, errors.New(errDeployTaskAlreadyRunning)
1279: }
1280:
1281: // failStartedTask marks the claimed task and its hosts failed when the service
1282: // instance could not be opened, and releases their leases.
1283: func (s *DeployService) failStartedTask(task *DeployTask, hosts []cmdbHostSnapshot, actor string, reason error) {
1284: now := time.Now()
1285: _ = s.db.Model(&DeployTask{}).Where(idWhereClause, task.ID).Updates(map[string]interface{}{
1286: "status": TaskStatusFailed, "finished_at": &now, "updated_by": actor, "updated_at": now,
1287: }).Error
1288: _ = s.db.Model(&DeployTaskHost{}).Where(taskIDAndStatusInWhereClause, task.ID, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
1289: "status": TaskHostStatusFailed, "error_message": reason.Error(), "finished_at": &now, "updated_by": actor, "updated_at": now,
1290: }).Error

EXCERPT backend/modules/business/deploy/deploy_service.go
2480: if err == nil {
2481: return ""
2482: }
2483: return err.Error()
2484: }
2485:
2486: func (s *DeployService) filterVisibleTasks(rows []DeployTask, dataScope *common.DataScopeReq) ([]DeployTask, error) {
2487: if !requiresScopedTaskVisibility(dataScope) {
2488: return rows, nil
2489: }
2490: filtered := make([]DeployTask, 0, len(rows))
2491: for i := range rows {
2492: visible, err := s.taskVisible(&rows[i], dataScope)
2493: if err != nil {
2494: return nil, err
2495: }
2496: if visible {
2497: filtered = append(filtered, rows[i])
2498: }
2499: }
2500: return filtered, nil
2501: }
2502:
2503: func (s *DeployService) taskVisible(task *DeployTask, dataScope *common.DataScopeReq) (bool, error) {
2504: if task == nil {
2505: return false, nil
2506: }
2507: if !requiresScopedTaskVisibility(dataScope) {
2508: return true, nil
2509: }
2510: hasVisibleTaskHost, hasTaskHosts, err := s.taskHasVisibleTaskHost(task.ID, dataScope)
2511: if err != nil {
2512: return false, err
2513: }
2514: if hasTaskHosts {
2515: return hasVisibleTaskHost, nil
2516: }
2517: hosts, err := s.resolveTaskTargets(task, dataScope)
2518: if err != nil {
2519: return false, err
2520: }
2521: return len(hosts) > 0, nil
2522: }
2523:
2524: func (s *DeployService) taskHasVisibleTaskHost(taskID uint64, dataScope *common.DataScopeReq) (bool, bool, error) {
2525: var taskHostCount int64
2526: if err := s.db.Model(&DeployTaskHost{}).Where(taskIDWhereClause, taskID).Count(&taskHostCount).Error; err != nil {
2527: return false, false, err
2528: }
2529: if taskHostCount == 0 {
2530: return false, false, nil
2531: }
2532: if s.hostReader == nil {
2533: return false, true, errors.New("business.cmdb.readerNotConfigured")
2534: }
2535: var taskHosts []DeployTaskHost
2536: if err := s.db.Where(taskIDWhereClause, taskID).Find(&taskHosts).Error; err != nil {
2537: return false, true, err
2538: }
2539: ids := make([]uint64, 0, len(taskHosts))
2540: for _, taskHost := range taskHosts {
2541: ids = append(ids, taskHost.HostID)
2542: }
2543: page, err := s.hostReader.GetByIDs(s.requestContext(), bizcap.HostIDsQuery{HostIDs: ids, DataScope: dataScope})
2544: if err != nil {
2545: return false, true, err
2546: }
2547: return page.Total > 0, true, nil
2548: }
2549:
2550: func requiresScopedTaskVisibility(dataScope *common.DataScopeReq) bool {
2551: if dataScope == nil || dataScope.IsAdmin {
2552: return false
2553: }
2554: mode := strings.TrimSpace(dataScope.Mode)
2555: return mode != "" && mode != common.DataScopeModeAll
2556: }
2557:
2558: func (s *DeployService) recomputeTaskStatus(taskID uint64, actor string) error {
2559: var hosts []DeployTaskHost
2560: if err := s.db.Where(taskIDWhereClause, taskID).Find(&hosts).Error; err != nil {
2561: return err
2562: }
2563: if len(hosts) == 0 {
2564: return nil
2565: }
2566: status := terminalTaskStatus(hosts)
2567: if status == "" {
2568: return nil
2569: }
2570: activeAttempt, err := s.hasActiveTaskAttempt(taskID, 0, time.Now())

EXCERPT scripts/business-overlay/rebuild-from-base.mjs
568: export function projectConsumerReviewRegistry(baseRoot, targetRoot, manifest, baseCommit) {
569: const descriptorPath = path.join(baseRoot, 'config/foundation-consumer-artifacts.json');
570: if (!fs.existsSync(descriptorPath)) return [];
571: const descriptor = JSON.parse(fs.readFileSync(descriptorPath, 'utf8'));
572: const contract = descriptor.reviewLegacyRegistry;
573: if (descriptor.schemaVersion !== 1 || descriptor.sourceOwner !== 'pantheon-base' ||
574: contract?.path !== 'config/review-loop-legacy-artifacts.json' ||
575: contract?.artifactRoot !== '.harness/archive') {
576: throw new Error('Invalid Base consumer review packaging contract');
577: }
578: const sourcePath = path.join(baseRoot, contract.path);
579: const sourceBytes = fs.readFileSync(sourcePath);
580: const source = JSON.parse(sourceBytes.toString('utf8'));
581: if (!Array.isArray(source.entries)) throw new Error('Base legacy review entries must be an array');
582: const excluded = [...(manifest.replaceRoots ?? []), ...(manifest.baseExcludedPaths ?? [])].map(repoPath);
583: const retained = [];
584: const external = [];
585: for (const entry of source.entries) {
586: if (typeof entry !== 'string' || !entry.startsWith('.harness/archive/') ||
587: entry.includes('..') || entry.includes('\\') || !entry.endsWith('/review.md')) {
588: throw new Error('Base legacy entry must be a canonical archive review');
589: }
590: if (!fs.existsSync(path.join(baseRoot, entry))) throw new Error(`Stale Base legacy entry: ${entry}`);
591: if (excluded.some((root) => isWithin(entry, root))) external.push(entry);
592: else {
593: if (!fs.existsSync(path.join(targetRoot, entry))) throw new Error(`Missing retained Base review: ${entry}`);
594: retained.push(entry);
595: }
596: }
597: if (external.length === 0) return [];
598: const targetPath = path.join(targetRoot, contract.path);
599: const projection = {
600: ...source,
601: description: 'Consumer projection of the Base archive registry. Excluded Base artifacts stay recorded below; all current consumer reviews remain subject to strict review validation.',
602: entries: retained,
603: projection: { sourceOwner: descriptor.sourceOwner, baseCommit,
604: sourceConfigSha256: crypto.createHash('sha256').update(sourceBytes).digest('hex'),
605: externalBaseArtifacts: external },
606: };
607: fs.mkdirSync(path.dirname(targetPath), { recursive: true });
608: fs.writeFileSync(targetPath, `${JSON.stringify(projection, null, 2)}\n`, 'utf8');
609: return [contract.path];
610: }
611:
612: export function assembleConsumerTree({ opsRoot = defaultOpsRoot, snapshot, targetRoot, manifestPath } = {}) {
613: const resolvedManifest = manifestPath ?? path.join(opsRoot, 'business-overlay.json');
614: const manifest = JSON.parse(fs.readFileSync(resolvedManifest, 'utf8'));
615: assertSafeManifest(manifest);
616:
617: const resolvedBase = snapshot?.baseRoot;
