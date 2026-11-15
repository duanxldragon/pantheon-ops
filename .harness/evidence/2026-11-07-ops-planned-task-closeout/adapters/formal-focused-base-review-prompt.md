Independent read-only code review. Tools disabled. Inspect only supplied source. Return concrete P0/P1/P2 defects with path, lines, trigger, minimal fix; max 650 words. If no proved defects say so. Separate assumptions. No production acceptance. Group is a global conditions catalog without BizScope/dept ownership; host queries are BizScope+DataScope scoped. Exclusions remove targets before eligibility intentionally. Dynamic scope/group/IP resolution has a deliberate 10k candidate budget; explicit IDs are narrowed. Request cancellation must propagate until transaction commit; committed async execution must continue. Reviewer sees selected source only, not full runtime.

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

FILE pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs
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
618: if (!resolvedBase || !snapshot.baseCommit) throw new Error('Verified release snapshot is required');
619: const resolvedTarget = path.resolve(opsRoot, targetRoot ?? '.tmp/business-overlay-rebuild');
620: const targetContainsSource = [path.resolve(opsRoot), path.resolve(resolvedBase)].some((source) => source === resolvedTarget || source.startsWith(resolvedTarget + path.sep));
