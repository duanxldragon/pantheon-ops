Read-only independent review. User task: complete formal Pantheon Ops planning tasks. Scope is current O1 CMDB target selection and O3 Deploy immutable preview/start. Find concrete high-impact bugs in authorization, TOCTOU, consistency, malformed input, secret redaction, compatibility. Do not modify files or invoke tools. Review only provided source. Findings-first with line references; give explicit limits (no execution proof). Existing middleware authorizes API routes; business routes need forthcoming Base provider mapping extension. No credentials included.

FILE backend/modules/business/cmdb/target_selection.go
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
108: for _, host := range hosts {
109: byID[host.ID] = host
110: byIP[host.IP] = host
111: }
112: selected := map[uint64]string{}
113: add := func(id uint64, source string) {
114: if _, ok := selected[id]; ok {
115: out.DuplicateCount++
116: return
117: }
118: selected[id] = source
119: }
120: for _, id := range req.HostIDs {
121: if _, ok := byID[id]; !ok {
122: out.UnavailableCount++
123: continue
124: }
125: add(id, "host")
126: }
127: if req.IncludeScope {
128: for _, host := range hosts {
129: add(host.ID, "scope")
130: }
131: }
132: // Group metadata remains provider-owned; inherited conditions are included in
133: // the revision, so editing a group after preview cannot silently change targets.
134: var groups []cmdbgroup.Group
135: if len(req.GroupIDs) > 0 {
136: gq := c.db.WithContext(ctx).Order(idAscOrder).Limit(TargetSelectionBudget + 1)
137: if lock {
138: gq = gq.Clauses(clause.Locking{Strength: "UPDATE"})
139: }
140: if err := gq.Find(&groups).Error; err != nil {
141: return nil, err
142: }
143: if len(groups) > TargetSelectionBudget {
144: return nil, ErrTargetSelectionInvalid
145: }
146: groupMap := make(map[uint64]cmdbgroup.Group, len(groups))
147: for _, group := range groups {
148: groupMap[group.ID] = group
149: }
150: for _, id := range req.GroupIDs {
151: group, ok := groupMap[id]
152: if !ok {
153: out.UnavailableCount++
154: continue
155: }
156: chain := conditionChainFromMap(group, groupMap)
157: for _, host := range hosts {
158: if groupConditionChainMatchesHost(chain, host.LabelValues) {
159: add(host.ID, "group")
160: }
161: }
162: }
163: }
164: manualSeen := map[string]bool{}
165: for _, raw := range req.Manual {
166: value := strings.TrimSpace(raw)
167: if len(value) > 128 || value == "" {
168: out.UnavailableCount++
169: continue
170: }
171: if manualSeen[value] {
172: out.DuplicateCount++
173: continue
174: }
175: manualSeen[value] = true
176: var host cmdbhost.Host
177: var found bool
178: if addr, err := netip.ParseAddr(value); err == nil {
179: host, found = byIP[addr.Unmap().String()]
180: } else if id, err := strconv.ParseUint(value, 10, 64); err == nil {
181: host, found = byID[id]
182: }
183: if !found {
184: out.UnavailableCount++
185: continue
186: }
187: add(host.ID, "manual")
188: }
189: exclude := map[uint64]bool{}
190: for _, id := range req.ExcludedIDs {
191: exclude[id] = true
192: }
193: for _, host := range hosts {
194: source, ok := selected[host.ID]
195: if !ok {
196: continue
197: }
198: if exclude[host.ID] {
199: out.ExcludedCount++
200: continue
201: }
202: if host.Status == "retired" || host.Status == "pending" || host.Status == "unassigned" {
203: out.IneligibleCount++
204: continue
205: }
206: out.Hosts = append(out.Hosts, SelectionHost{ID: host.ID, Hostname: host.Hostname, IP: host.IP, Status: host.Status, StateVersion: host.StateVersion, BusinessScopeID: host.BusinessScopeID, Source: source})
207: out.Sources[source]++
208: }
209: // Include full authorized scoped host facts, not just stateVersion: existing
210: // metadata edits may legitimately keep a lifecycle version unchanged.
211: material := struct {
212: Selection TargetSelection
213: Scope bizcap.BizScopeRef
214: Hosts []cmdbhost.Host
215: Groups []cmdbgroup.Group
216: }{req, scopeRef, hosts, groups}
217: bytes, err := json.Marshal(material)
218: if err != nil {
219: return nil, err
220: }
221: digest := sha256.Sum256(bytes)
222: out.ScopeVersion = hex.EncodeToString(digest[:])
223: return out, nil
224: }
225: func normalizeSelectionIDs(ids []uint64, duplicates *int) []uint64 {
226: seen := map[uint64]bool{}
227: result := make([]uint64, 0, len(ids))
228: for _, id := range ids {
229: if id == 0 {
230: continue
231: }
232: if seen[id] {
233: if duplicates != nil {
234: *duplicates++
235: }
236: continue
237: }
238: seen[id] = true
239: result = append(result, id)
240: }
241: sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
242: return result
243: }

FILE backend/modules/business/deploy/target_selection.go
1: package deploy
2:
3: import (
4: "context"
5: "encoding/json"
6: "errors"
7: "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
8: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
9: "gorm.io/datatypes"
10: "gorm.io/gorm"
11: "gorm.io/gorm/clause"
12: "time"
13: )
14:
15: func (s *DeployService) ResolveTargetSelection(ctx context.Context, req cmdb.TargetSelection, scope *common.DataScopeReq) (*cmdb.TargetSelectionResult, error) {
16: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
17: if !ok {
18: return nil, errors.New("business.cmdb.selection.notConfigured")
19: }
20: return provider.ResolveTargetSelection(ctx, req, scope, false)
21: }
22: func (s *DeployService) CreateTask(req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
23: if req.TargetSelection == nil {
24: return s.createTask(req, actor, scope)
25: }
26: if s.db == nil {
27: return nil, errors.New("database.not_initialized")
28: }
29: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
30: if !ok {
31: return nil, errors.New("business.cmdb.selection.notConfigured")
32: }
33: var taskID uint64
34: err := s.db.Transaction(func(tx *gorm.DB) error {
35: owner := provider.WithTargetTransaction(tx)
36: selection, err := owner.ResolveTargetSelection(context.Background(), *req.TargetSelection, scope, true)
37: if err != nil {
38: return err
39: }
40: if req.ScopeVersion == "" || selection.ScopeVersion != req.ScopeVersion {
41: return cmdb.ErrTargetSelectionConflict
42: }
43: if selection.UnavailableCount > 0 || selection.IneligibleCount > 0 || len(selection.Hosts) == 0 {
44: return cmdb.ErrTargetSelectionInvalid
45: }
46: req.BusinessScopeID = selection.Selection.BusinessScopeID
47: req.TargetType = TargetTypeHost
48: req.TargetIDs = make([]uint64, 0, len(selection.Hosts))
49: for _, host := range selection.Hosts {
50: req.TargetIDs = append(req.TargetIDs, host.ID)
51: }
52: capability, ok := owner.(cmdb.DeployCMDBCapability)
53: if !ok {
54: return errors.New("business.cmdb.selection.notConfigured")
55: }
56: inner := &DeployService{db: tx, cmdbCapability: capability, bizScopeReader: s.bizScopeReader, hostReader: capability, serviceState: s.serviceState}
57: task, err := inner.createTask(req, actor, scope)
58: if err != nil {
59: return err
60: }
61: taskID = task.ID
62: snapshot := deployTargetSnapshot{Selection: &selection.Selection, ScopeVersion: selection.ScopeVersion, Sources: selection.Sources, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano), TargetType: TargetTypeHost}
63: bytes, err := json.Marshal(snapshot)
64: if err != nil {
65: return err
66: }
67: return tx.Model(&DeployTask{}).Where(idWhereClause, taskID).UpdateColumn("target_snapshot", datatypes.JSON(bytes)).Error
68: })
69: if err != nil {
70: return nil, err
71: }
72: return s.GetTask(taskID, scope)
73: }
74:
75: func (s *DeployService) UpdateTask(id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
76: if req.TargetSelection == nil {
77: current, err := s.loadVisibleTask(id, scope)
78: if err != nil {
79: return nil, err
80: }
81: var snapshot deployTargetSnapshot
82: if len(current.TargetSnapshot) > 0 && json.Unmarshal(current.TargetSnapshot, &snapshot) != nil {
83: return nil, cmdb.ErrTargetSelectionConflict
84: }
85: if snapshot.Selection != nil && (req.TargetIDs != nil || req.TargetType != nil || req.BusinessScopeID != nil) {
86: return nil, cmdb.ErrTargetSelectionInvalid
87: }
88: return s.updateTask(id, req, actor, scope)
89: }
90: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
91: if !ok {
92: return nil, cmdb.ErrTargetSelectionInvalid
93: }
94: err := s.db.Transaction(func(tx *gorm.DB) error {
95: var current DeployTask
96: if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, id).Error; err != nil {
97: return err
98: }
99: owner := provider.WithTargetTransaction(tx)
100: result, err := owner.ResolveTargetSelection(context.Background(), *req.TargetSelection, scope, true)
101: if err != nil {
102: return err
103: }
104: if req.ScopeVersion == "" || req.ScopeVersion != result.ScopeVersion {
105: return cmdb.ErrTargetSelectionConflict
106: }
107: if result.UnavailableCount > 0 || result.IneligibleCount > 0 || len(result.Hosts) == 0 {
108: return cmdb.ErrTargetSelectionInvalid
109: }
110: req.BusinessScopeID = &result.Selection.BusinessScopeID
111: targetType := TargetTypeHost
112: req.TargetType = &targetType
113: req.TargetIDs = make([]uint64, 0, len(result.Hosts))
114: for _, host := range result.Hosts {
115: req.TargetIDs = append(req.TargetIDs, host.ID)
116: }
117: capability, ok := owner.(cmdb.DeployCMDBCapability)
118: if !ok {
119: return cmdb.ErrTargetSelectionInvalid
120: }
121: inner := NewDeployService(tx, capability)
122: inner.serviceState = s.serviceState
123: if _, err := inner.updateTask(id, req, actor, scope); err != nil {
124: return err
125: }
126: snapshot := deployTargetSnapshot{Selection: &result.Selection, ScopeVersion: result.ScopeVersion, Sources: result.Sources, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano), TargetType: TargetTypeHost}
127: raw, err := json.Marshal(snapshot)
128: if err != nil {
129: return err
130: }
131: return tx.Model(&DeployTask{}).Where(idWhereClause, id).UpdateColumn("target_snapshot", datatypes.JSON(raw)).Error
132: })
133: if err != nil {
134: return nil, err
135: }
136: return s.GetTask(id, scope)
137: }

FILE backend/modules/business/deploy/task_preview.go
1: package deploy
2:
3: import (
4: "bytes"
5: "context"
6: "crypto/hmac"
7: "crypto/rand"
8: "crypto/sha256"
9: "encoding/base64"
10: "encoding/hex"
11: "encoding/json"
12: "errors"
13: "fmt"
14: "github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
15: "github.com/duanxldragon/pantheon-base/backend/pkg/common"
16: "gorm.io/gorm"
17: "gorm.io/gorm/clause"
18: "strings"
19: "time"
20: )
21:
22: var ErrTaskPreviewRequired = errors.New("business.deploy.preview.required")
23: var ErrTaskPreviewConflict = errors.New("business.deploy.preview.conflict")
24: var previewSigningKey = func() []byte {
25: key := make([]byte, 32)
26: if _, err := rand.Read(key); err != nil {
27: panic(err)
28: }
29: return key
30: }()
31:
32: type TaskPreviewChange struct {
33: ID string `json:"id"`
34: Field string `json:"field"`
35: Kind string `json:"kind"`
36: Before string `json:"before,omitempty"`
37: After string `json:"after,omitempty"`
38: }
39: type TaskPreviewResponse struct {
40: IntentID string `json:"intentId"`
41: IntentVersion string `json:"intentVersion"`
42: CurrentVersion string `json:"currentVersion"`
43: TargetVersion string `json:"targetVersion"`
44: TargetCount int `json:"targetCount"`
45: BusinessScopeName string `json:"businessScopeName"`
46: Changes []TaskPreviewChange `json:"changes"`
47: RiskKeys []string `json:"riskKeys"`
48: RollbackAvailable bool `json:"rollbackAvailable"`
49: ExpiresAt time.Time `json:"expiresAt"`
50: }
51: type taskPreviewTicket struct {
52: TaskID uint64 `json:"taskId"`
53: Actor string `json:"actor"`
54: Revision string `json:"revision"`
55: Expires int64 `json:"expires"`
56: }
57:
58: func signPreviewTicket(ticket taskPreviewTicket) (string, error) {
59: payload, err := json.Marshal(ticket)
60: if err != nil {
61: return "", err
62: }
63: mac := hmac.New(sha256.New, previewSigningKey)
64: _, _ = mac.Write(payload)
65: return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
66: }
67: func parsePreviewTicket(token string) (taskPreviewTicket, error) {
68: var ticket taskPreviewTicket
69: if len(token) > 1024 {
70: return ticket, ErrTaskPreviewConflict
71: }
72: parts := strings.Split(token, ".")
73: if len(parts) != 2 {
74: return ticket, ErrTaskPreviewConflict
75: }
76: payload, err := base64.RawURLEncoding.DecodeString(parts[0])
77: if err != nil {
78: return ticket, ErrTaskPreviewConflict
79: }
80: signature, err := base64.RawURLEncoding.DecodeString(parts[1])
81: if err != nil {
82: return ticket, ErrTaskPreviewConflict
83: }
84: mac := hmac.New(sha256.New, previewSigningKey)
85: _, _ = mac.Write(payload)
86: if !hmac.Equal(signature, mac.Sum(nil)) {
87: return ticket, ErrTaskPreviewConflict
88: }
89: decoder := json.NewDecoder(bytes.NewReader(payload))
90: decoder.DisallowUnknownFields()
91: if err := decoder.Decode(&ticket); err != nil {
92: return ticket, ErrTaskPreviewConflict
93: }
94: if ticket.Expires <= time.Now().Unix() {
95: return ticket, ErrTaskPreviewConflict
96: }
97: return ticket, nil
98: }
99: func taskPreviewRevision(db *gorm.DB, task *DeployTask, hosts []cmdbHostSnapshot, req StartTaskRequest, credentialVersion uint64, lock bool) (string, error) {
100: var secrets []struct {
101: SecretKey string
102: SecretEncrypted string
103: }
104: query := db.Model(&DeployTaskSecret{}).Where(taskIDWhereClause, task.ID).Order("secret_key ASC")
105: if lock {
106: query = query.Clauses(clause.Locking{Strength: "UPDATE"})
107: }
108: if err := query.Find(&secrets).Error; err != nil {
109: return "", err
110: }
111: request := struct {
112: CredentialRefID uint64
113: TimeoutSeconds int
114: HostFingerprint string
115: }{req.CredentialRefID, int(normalizeDeployExecutionTimeout(req.TimeoutSeconds).Seconds()), strings.TrimSpace(req.HostFingerprint)}
116: material := struct {
117: Task *DeployTask
118: Hosts []cmdbHostSnapshot
119: Secrets any
120: Request any
121: CredentialVersion uint64
122: }{task, hosts, secrets, request, credentialVersion}
123: payload, err := json.Marshal(material)
124: if err != nil {
125: return "", err
126: }
127: hash := sha256.Sum256(payload)
128: return hex.EncodeToString(hash[:]), nil
129: }
130: func (s *DeployService) PreviewTask(id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
131: task, err := s.loadVisibleTask(id, scope)
132: if err != nil {
133: return nil, err
134: }
135: if task.Status != TaskStatusDraft && task.Status != TaskStatusPending {
136: return nil, ErrTaskPreviewConflict
137: }
138: credentialVersion := uint64(0)
139: if task.ExecutorType == ExecutorTypeSSH {
140: resolved, version, err := s.prepareStartCredential(req)
141: if err != nil {
142: return nil, err
143: }
144: req = resolved
145: credentialVersion = version
146: }
147: hosts, err := s.prepareStartTargets(task, req, scope)
148: if err != nil {
149: return nil, err
150: }
151: for _, host := range hosts {
152: if !hostStatusAllowedForAction(host.Status, task.Action) {
153: return nil, ErrTaskPreviewConflict
154: }
155: }
156: if err := s.validateSelectionForPreview(context.Background(), task, scope, false); err != nil {
157: return nil, err
158: }
159: revision, err := taskPreviewRevision(s.db, task, hosts, req, credentialVersion, false)
160: if err != nil {
161: return nil, err
162: }
163: expires := time.Now().UTC().Add(10 * time.Minute)
164: token, err := signPreviewTicket(taskPreviewTicket{TaskID: id, Actor: actor, Revision: revision, Expires: expires.Unix()})
165: if err != nil {
166: return nil, err
167: }
168: targetHash := sha256.Sum256(task.ExecutionSnapshot)
169: result := &TaskPreviewResponse{IntentID: fmt.Sprintf("deploy:%d", id), IntentVersion: token, CurrentVersion: revision, TargetVersion: hex.EncodeToString(targetHash[:]), TargetCount: len(hosts), BusinessScopeName: task.BusinessScopeName, Changes: []TaskPreviewChange{}, RiskKeys: []string{"business.deploy.preview.healthUnknown", "business.deploy.preview.rollbackManual"}, RollbackAvailable: false, ExpiresAt: expires}
170: for _, host := range hosts {
171: current := ""
172: known := true
173: var installed []struct {
174: Name string `json:"name"`
175: Version string `json:"version"`
176: }
177: if len(host.InstalledComponents) > 0 && json.Unmarshal(host.InstalledComponents, &installed) != nil {
178: known = false
179: }
180: for _, component := range installed {
181: if component.Name == task.PackageName {
182: current = component.Version
183: }
184: }
185: kind := "modified"
186: target := task.PackageVersion
187: if !known {
188: kind = "conflict"
189: } else if current == "" {
190: kind = "added"
191: } else if current == target {
192: kind = "unchanged"
193: }
194: if task.Action == TaskActionUninstall || task.Action == TaskActionRetire {
195: kind = "removed"
196: target = ""
197: }
198: result.Changes = append(result.Changes, TaskPreviewChange{ID: fmt.Sprintf("host:%d", host.ID), Field: host.Hostname + " / " + task.PackageName, Kind: kind, Before: current, After: target})
199: }
200: if len(hosts) > 100 {
201: result.RiskKeys = append(result.RiskKeys, "business.deploy.preview.largeScope")
202: }
203: if task.Action == TaskActionUninstall || task.Action == TaskActionRetire || task.Action == TaskActionStop {
204: result.RiskKeys = append(result.RiskKeys, "business.deploy.preview.disruptive")
205: }
206: return result, nil
207: }
208: func (s *DeployService) validateSelectionForPreview(ctx context.Context, task *DeployTask, scope *common.DataScopeReq, lock bool) error {
209: var snapshot deployTargetSnapshot
210: if len(task.TargetSnapshot) == 0 {
211: return nil
212: }
213: if err := json.Unmarshal(task.TargetSnapshot, &snapshot); err != nil {
214: return ErrTaskPreviewConflict
215: }
216: if snapshot.Selection == nil {
217: return nil
218: }
219: provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
220: if !ok {
221: return cmdb.ErrTargetSelectionInvalid
222: }
223: resolved, err := provider.ResolveTargetSelection(ctx, *snapshot.Selection, scope, lock)
224: if err != nil {
225: return err
226: }
227: if resolved.ScopeVersion != snapshot.ScopeVersion || resolved.UnavailableCount > 0 || resolved.IneligibleCount > 0 {
228: return cmdb.ErrTargetSelectionConflict
229: }
230: return nil
231: }
232: func (c *startTaskClaim) validatePreview(tx *gorm.DB) error {
233: var selection deployTargetSnapshot
234: _ = json.Unmarshal(c.task.TargetSnapshot, &selection)
235: if !c.req.RequirePreview && c.req.IntentVersion == "" && selection.Selection == nil {
236: return nil
237: }
238: if c.req.IntentVersion == "" {
239: return ErrTaskPreviewRequired
240: }
241: ticket, err := parsePreviewTicket(c.req.IntentVersion)
242: if err != nil {
243: return err
244: }
245: if ticket.TaskID != c.task.ID || ticket.Actor != c.actor {
246: return ErrTaskPreviewConflict
247: }
248: var current DeployTask
249: if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, c.task.ID).Error; err != nil {
250: return err
251: }
252: if current.Status != TaskStatusDraft && current.Status != TaskStatusPending {
253: return errors.New(errDeployTaskAlreadyRunning)
254: }
255: provider := c.service.cmdbCapability
256: if owner, ok := provider.(cmdb.TargetSelectionCapability); ok {
257: transactionProvider, valid := owner.WithTargetTransaction(tx).(cmdb.DeployCMDBCapability)
258: if !valid {
259: return cmdb.ErrTargetSelectionInvalid
260: }
261: provider = transactionProvider
262: }
263: inner := &DeployService{db: tx, cmdbCapability: provider, bizScopeReader: c.service.bizScopeReader, hostReader: c.service.hostReader}
264: if err := inner.validateSelectionForPreview(context.Background(), &current, c.dataScope, true); err != nil {
265: return err
266: }
267: hosts, err := inner.prepareStartTargets(&current, c.req, c.dataScope)
268: if err != nil {
269: return err
270: }
271: if current.ExecutorType == ExecutorTypeSSH {
272: var credential DeployCredentialRef
273: if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&credential, c.req.CredentialRefID).Error; err != nil {
274: return err
275: }
276: if credential.Version != c.credentialVersion || credential.Status != "active" {
277: return ErrTaskPreviewConflict
278: }
279: }
280: revision, err := taskPreviewRevision(tx, &current, hosts, c.req, c.credentialVersion, true)
281: if err != nil {
282: return err
283: }
284: if ticket.Revision != revision {
285: return ErrTaskPreviewConflict
286: }
287: c.task = &current
288: c.hosts = hosts
289: return nil
290: }

START/CLAIM SOURCE
func (s *DeployService) StartTask(id uint64, req StartTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
task, err := s.loadVisibleTask(id, dataScope)
if err != nil {
return nil, err
}
credentialVersion := uint64(0)
key := strings.TrimSpace(req.IdempotencyKey)

    // Reject or replay non-draft/pending states before attempting any claim.
    switch task.Status {
    case TaskStatusRunning:
    	if key != "" && key == task.StartRequestKey {
    		return s.getTaskReused(id, dataScope)
    	}
    	return nil, errors.New(errDeployTaskAlreadyRunning)
    case TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled,
    	TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested:
    	return nil, errors.New("business.deploy.task.invalidStartState")
    }
    if task.ExecutorType == ExecutorTypeSSH {
    	resolvedReq, version, err := s.prepareStartCredential(req)
    	if err != nil {
    		return nil, err
    	}
    	req, credentialVersion = resolvedReq, version
    }
    hosts, err := s.prepareStartTargets(task, req, dataScope)
    if err != nil {
    	return nil, err
    }

    now := time.Now()
    claim := startTaskClaim{
    	service: s, dataScope: dataScope,
    	task:              task,
    	req:               req,
    	hosts:             hosts,
    	actor:             actor,
    	key:               key,
    	now:               now,
    	timeout:           normalizeDeployExecutionTimeout(req.TimeoutSeconds),
    	leaseOwner:        deployHostLeaseOwner(task.ID),
    	credentialVersion: credentialVersion,
    }
    claimErr := s.db.Transaction(claim.claim)
    if claimErr != nil {
    	return s.rejectFailedStartClaim(id, claimErr, key, dataScope)
    }
    task, hosts = claim.task, claim.hosts
    s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.started", fromStatus: task.Status, toStatus: TaskStatusRunning, actor: actor, requestID: key})
    if err := s.beginServiceInstanceState(*task, actor, dataScope); err != nil {
    	s.failStartedTask(task, hosts, actor, err)
    	return nil, err
    }
    if err := s.dispatchStartedTask(task, hosts, req, actor); err != nil {
    	return nil, err
    }
    return s.GetTask(id, dataScope)

}

// prepareStartCredential resolves the SSH credential reference of a start
// request into the values persisted on the task.
func (s *DeployService) prepareStartCredential(req StartTaskRequest) (StartTaskRequest, uint64, error) {
if err := validateCredentialReferenceRequest(req); err != nil {
return req, 0, err
}
resolvedReq, version, err := s.resolveStartCredential(req)
if err != nil {
return req, 0, err
}
return resolvedReq, version, nil
}

// prepareStartTargets validates the frozen execution snapshot and resolves the
// hosts the start request targets.
func (s *DeployService) prepareStartTargets(task *DeployTask, req StartTaskRequest, dataScope *common.DataScopeReq) ([]cmdbHostSnapshot, error) {
// Validate the frozen execution snapshot before mutating anything.
if _, err := s.resolveTaskExecutionPlan(*task); err != nil {
return nil, mapDeployTaskExecutionPlanError(err)
}
hosts, err := s.resolveTaskTargets(task, dataScope)
if err != nil {
return nil, err
}
if len(hosts) == 0 {
return nil, errors.New("business.deploy.task.emptyResolvedTargets")
}
if task.ExecutorType == ExecutorTypeSSH {
if err := validateDeploySSHStartRequest(req); err != nil {
return nil, err
}
if err := s.validateTaskExecutionPlan(*task, hosts); err != nil {
return nil, mapDeployTaskExecutionPlanError(err)
}
}
return hosts, nil
}

// startTaskClaim carries the resolved start request of the claiming
// transaction so the transaction keeps its inputs in one place.
type startTaskClaim struct {
service *DeployService
dataScope *common.DataScopeReq
task *DeployTask
req StartTaskRequest
hosts []cmdbHostSnapshot
actor string
key string
now time.Time
timeout time.Duration
leaseOwner string
credentialVersion uint64
}

// claim flips the task to running, freezes the target snapshot and creates the
// per-host rows together with their leases.
func (c *startTaskClaim) claim(tx *gorm.DB) error {
if err := c.validatePreview(tx); err != nil {
return err
}
result := tx.Model(&DeployTask{}).
Where(idWhereClause, c.task.ID).
Where("status IN ?", []string{TaskStatusDraft, TaskStatusPending}).
Updates(map[string]interface{}{
"status": TaskStatusRunning,
"started_at": &c.now,
"start_request_key": c.key,
"credential_ref_id": c.req.CredentialRefID,
"credential_ref_version": c.credentialVersion,
"ssh_host_fingerprint": strings.TrimSpace(c.req.HostFingerprint),
"execution_timeout_seconds": int(c.timeout.Seconds()),
"updated_by": c.actor,
"updated_at": c.now,
})
if result.Error != nil {
return result.Error
}
if result.RowsAffected == 0 {
return errors.New(errDeployTaskAlreadyRunning)
}

    targetSnapshot := buildDeployTargetSnapshot(c.task, c.hosts, c.now)
    var selectionSnapshot deployTargetSnapshot
    if err := json.Unmarshal(c.task.TargetSnapshot, &selectionSnapshot); err == nil {
    	targetSnapshot.Selection = selectionSnapshot.Selection
    	targetSnapshot.ScopeVersion = selectionSnapshot.ScopeVersion
    	targetSnapshot.Sources = selectionSnapshot.Sources
    }
    targetJSON, _ := json.Marshal(targetSnapshot)
    if err := tx.Model(&DeployTask{}).Where(idWhereClause, c.task.ID).Update("target_snapshot", datatypes.JSON(targetJSON)).Error; err != nil {
    	return err
    }

    for _, host := range c.hosts {
    	if err := c.createTaskHostRow(tx, host); err != nil {
    		return err
    	}
    }
    return nil

}

// createTaskHostRow acquires the host lease and persists the running host row
// with the execution steps needed for checkpoint recovery.
func (c startTaskClaim) createTaskHostRow(tx *gorm.DB, host cmdbHostSnapshot) error {
if err := acquireHostLease(tx, host.ID, c.task.ID, c.leaseOwner, c.now); err != nil {
return err
}
initialTrace, _ := json.Marshal([]map[string]any{
{"at": c.now.Format(time.RFC3339), "phase": "start", "message": "Task host queued"},
})
row := DeployTaskHost{
TaskID: c.task.ID,
HostID: host.ID,
Hostname: host.Hostname,
HostIP: host.IP,
SSHPort: host.SSHPort,
OS: host.OS,
BusinessScopeID: host.BusinessScopeID,
Status: TaskHostStatusRunning,
TraceSteps: datatypes.JSON(initialTrace),
StartedAt: &c.now,
ResolvedAt: &c.now,
UpdatedBy: c.actor,
}
if err := tx.Create(&row).Error; err != nil {
return err
}
// FZ-03: persist execution steps for checkpoint recovery.
var executionSnapshot deployExecutionSnapshot
if err := json.Unmarshal(c.task.ExecutionSnapshot, &executionSnapshot); err == nil && len(executionSnapshot.Steps) > 0 {
return persistTaskSteps(tx, c.task.ID, row.ID, executionSnapshot.Steps)
}
return nil
}

// rejectFailedStartClaim replays an idempotent start that lost the claim race
// and otherwise reports the claim failure.
func (s *DeployService) rejectFailedStartClaim(id uint64, claimErr error, key string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
if strings.TrimSpace(errorText(claimErr)) != errDeployTaskAlreadyRunning {
return nil, claimErr
}
current, loadErr := s.loadVisibleTask(id, dataScope)
if loadErr == nil && current.Status == TaskStatusRunning && key != "" && current.StartRequestKey == key {
return s.getTaskReused(id, dataScope)
}
return nil, errors.New(errDeployTaskAlreadyRunning)
}

// failStartedTask marks the claimed task and its hosts failed when the service
// instance could not be opened, and releases their leases.
func (s *DeployService) failStartedTask(task *DeployTask, hosts []cmdbHostSnapshot, actor string, reason error) {
now := time.Now()
_ = s.db.Model(&DeployTask{}).Where(idWhereClause, task.ID).Updates(map[string]interface{}{
"status": TaskStatusFailed, "finished_at": &now, "updated_by": actor, "updated_at": now,
}).Error
_ = s.db.Model(&DeployTaskHost{}).Where(taskIDAndStatusInWhereClause, task.ID, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
"status": TaskHostStatusFailed, "error_message": reason.Error(), "finished_at": &now, "updated_by": actor, "updated_at": now,
}).Error
for _, host := range hosts {
_ = releaseHostLease(s.db, host.ID, deployHostLeaseOwner(task.ID))
}
}

// dispatchStartedTask hands the claimed task to the async worker or runs it
// inline for the current request.
func (s *DeployService) dispatchStartedTask(task *DeployTask, hosts []cmdbHostSnapshot, req StartTaskRequest, actor string) error {
if task.ExecutorType != ExecutorTypeSSH {
return nil
}
if s.asyncExecution {
// 记录任务开始 metrics
recordTaskStart(task)
go s.runAsyncTask(task.ID, actor)
return nil
}
return s.executeSSHTask(*task, hosts, req, actor)
}
