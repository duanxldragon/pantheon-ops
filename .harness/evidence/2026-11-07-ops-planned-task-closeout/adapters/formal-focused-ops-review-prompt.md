Independent read-only code review. Tools disabled. Inspect only supplied source. Return concrete P0/P1/P2 defects with path, lines, trigger, minimal fix; max 650 words. If no proved defects say so. Separate assumptions. No production acceptance. Group is a global conditions catalog without BizScope/dept ownership; host queries are BizScope+DataScope scoped. Exclusions remove targets before eligibility intentionally. Dynamic scope/group/IP resolution has a deliberate 10k candidate budget; explicit IDs are narrowed. Request cancellation must propagate until transaction commit; committed async execution must continue. Reviewer sees selected source only, not full runtime.

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
93: // Explicit IDs only need their scoped rows. Dynamic group/scope selections and
94: // canonical IP ambiguity checks still require the bounded candidate scan.
95: // Every path keeps both DataScope and BizScope predicates before row locking.
96: query := c.db.WithContext(ctx).Model(&cmdbhost.Host{}).Scopes(database.WithDataScope(scope)).Where(businessScopeIDWhereClause, req.BusinessScopeID).Order(idAscOrder).Limit(TargetSelectionBudget + 1)
97: if ids, bounded := selectionCandidateIDs(req); bounded {
98: query = query.Where("id IN ?", ids)
99: }
100: if lock {
101: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
102: }
103: var hosts []cmdbhost.Host
104: if err := query.Find(&hosts).Error; err != nil {
105: return nil, err
106: }
107: if len(hosts) > TargetSelectionBudget {
108: return nil, ErrTargetSelectionInvalid
109: }
110: byID := make(map[uint64]cmdbhost.Host, len(hosts))
111: byIP := map[string]cmdbhost.Host{}
112: ambiguousIP := map[string]bool{}
113: for _, host := range hosts {
114: byID[host.ID] = host
115: if address, err := netip.ParseAddr(strings.TrimSpace(host.IP)); err == nil {
116: ip := address.Unmap().String()
117: if _, duplicate := byIP[ip]; duplicate {
118: ambiguousIP[ip] = true
119: }
120: byIP[ip] = host
121: }
122: }
123: selected := map[uint64]string{}
124: add := func(id uint64, source string) {
125: if _, ok := selected[id]; ok {
126: out.DuplicateCount++
127: return
128: }
129: selected[id] = source
130: }
131: for _, id := range req.HostIDs {
132: if _, ok := byID[id]; !ok {
133: out.UnavailableCount++
134: continue
135: }
136: add(id, "host")
137: }
138: if req.IncludeScope {
139: for _, host := range hosts {
140: add(host.ID, "scope")
141: }
142: }
143: // Group metadata remains provider-owned; inherited conditions are included in
144: // the revision, so editing a group after preview cannot silently change targets.
145: groups, groupMap, err := c.selectionGroups(ctx, req.GroupIDs, lock)
146: if err != nil {
147: return nil, err
148: }
149: for _, id := range req.GroupIDs {
150: group, ok := groupMap[id]
151: valid := ok
152: visited := map[uint64]bool{}
153: for node := group; valid && node.ID != 0; {
154: if visited[node.ID] {
155: valid = false
156: break
157: }
158: visited[node.ID] = true
159: if node.ParentID == 0 {
160: break
161: }
162: node, valid = groupMap[node.ParentID]
163: }
164: if !valid {
165: out.UnavailableCount++
166: continue
167: }
168: chain := conditionChainFromMap(group, groupMap)
169: for _, host := range hosts {
170: if groupConditionChainMatchesHost(chain, host.LabelValues) {
171: add(host.ID, "group")
172: }
173: }
174: }
175: manualSeen := map[string]bool{}
176: for _, raw := range req.Manual {
177: value := strings.TrimSpace(raw)
178: if len(value) > 128 || value == "" {
179: out.UnavailableCount++
180: continue
181: }
182: if manualSeen[value] {
183: out.DuplicateCount++
184: continue
185: }
186: manualSeen[value] = true
187: var host cmdbhost.Host
188: var found bool
189: if addr, err := netip.ParseAddr(value); err == nil {
190: ip := addr.Unmap().String()
191: host, found = byIP[ip]
192: found = found && !ambiguousIP[ip]
193: } else if id, err := strconv.ParseUint(value, 10, 64); err == nil {
194: host, found = byID[id]
195: }
196: if !found {
197: out.UnavailableCount++
198: continue
199: }
200: add(host.ID, "manual")
201: }
202: exclude := map[uint64]bool{}
203: for _, id := range req.ExcludedIDs {
204: exclude[id] = true
205: }
206: for _, host := range hosts {
207: source, ok := selected[host.ID]
208: if !ok {
209: continue
210: }
211: if exclude[host.ID] {
212: out.ExcludedCount++
213: continue
214: }
215: if host.Status == "retired" || host.Status == "pending" || host.Status == "unassigned" {
216: out.IneligibleCount++
217: continue
218: }
219: out.Hosts = append(out.Hosts, SelectionHost{ID: host.ID, Hostname: host.Hostname, IP: host.IP, Status: host.Status, StateVersion: host.StateVersion, BusinessScopeID: host.BusinessScopeID, Source: source})
220: out.Sources[source]++
221: }
222: // Include full authorized candidate facts, not just stateVersion: existing
223: // metadata edits may legitimately keep a lifecycle version unchanged.
224: material := struct {
225: Selection TargetSelection
226: Scope bizcap.BizScopeRef
227: Hosts []cmdbhost.Host
228: Groups []cmdbgroup.Group
229: }{req, scopeRef, hosts, groups}
230: bytes, err := json.Marshal(material)
231: if err != nil {
232: return nil, err
233: }
234: digest := sha256.Sum256(bytes)
235: out.ScopeVersion = hex.EncodeToString(digest[:])
236: return out, nil
237: }
238:
239: // selectionCandidateIDs narrows explicit-ID selections without changing IP
240: // canonicalization or dynamic group membership semantics.
241: func selectionCandidateIDs(req TargetSelection) ([]uint64, bool) {
242: if req.IncludeScope || len(req.GroupIDs) > 0 {
243: return nil, false
244: }
245: ids := append([]uint64(nil), req.HostIDs...)
246: for _, raw := range req.Manual {
247: value := strings.TrimSpace(raw)
248: if value == "" || len(value) > 128 {
249: continue
250: }
251: if _, err := netip.ParseAddr(value); err == nil {
252: return nil, false
253: }
254: if id, err := strconv.ParseUint(value, 10, 64); err == nil {
255: ids = append(ids, id)
256: }
257: }
258: return normalizeSelectionIDs(ids, nil), true
259: }
260:
261: func normalizeSelectionIDs(ids []uint64, duplicates *int) []uint64 {
262: seen := map[uint64]bool{}
263: result := make([]uint64, 0, len(ids))
264: for _, id := range ids {
265: if id == 0 {
266: continue
267: }
268: if seen[id] {
269: if duplicates != nil {
270: *duplicates++
271: }
272: continue
273: }
274: seen[id] = true
275: result = append(result, id)
276: }
277: sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
278: return result
279: }
280:
281: // Groups are a global condition catalog in the existing CMDB contract (no
282: // ownership columns). Hosts remain scoped. Read only referenced ancestor chains
283: // so unrelated catalog edits cannot invalidate a selection or acquire its locks.
284: func (c *deployCMDBCapability) selectionGroups(ctx context.Context, ids []uint64, lock bool) ([]cmdbgroup.Group, map[uint64]cmdbgroup.Group, error) {
285: groups := []cmdbgroup.Group{}
286: byID := map[uint64]cmdbgroup.Group{}
287: seen := map[uint64]bool{}
288: pending := ids
289: for depth := 0; len(pending) > 0; depth++ {
290: if depth >= 128 {
291: return nil, nil, ErrTargetSelectionInvalid
292: }
293: queryIDs := []uint64{}
294: for _, id := range pending {
295: if id != 0 && !seen[id] {
296: seen[id] = true
297: queryIDs = append(queryIDs, id)
298: }
299: }
300: if len(queryIDs) == 0 {
301: break
302: }
303: if len(seen) > TargetSelectionBudget {
304: return nil, nil, ErrTargetSelectionInvalid
305: }
306: query := c.db.WithContext(ctx).Where(idInWhereClause, queryIDs).Order(idAscOrder)
307: if lock {
308: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
309: }
310: var rows []cmdbgroup.Group
311: if err := query.Find(&rows).Error; err != nil {
312: return nil, nil, err
313: }
314: pending = nil
315: for _, group := range rows {
316: byID[group.ID] = group
317: groups = append(groups, group)
318: if group.ParentID != 0 {
319: pending = append(pending, group.ParentID)
320: }
321: }
322: }
323: sort.Slice(groups, func(i, j int) bool { return groups[i].ID < groups[j].ID })
324: return groups, byID, nil
325: }

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

FILE pantheon-ops/backend/modules/business/deploy/deploy_service.go
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
