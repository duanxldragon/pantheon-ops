Independent CODE review, read-only source-only. Tools disabled. Report only proved P0/P1/P2 defects, concrete source line and minimal fix. Max 450 words. End with verdict APPROVE or CHANGES_REQUESTED for supplied source only, plus explicit scope limits. No production, complete project, or release acceptance. Generic shared source belongs to Base, business semantics to Ops. Unknown runtime dependencies must be flagged conditional rather than assumed defective.

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
