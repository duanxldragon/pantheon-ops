Independent CODE review, read-only source-only. Tools disabled. Report only proved P0/P1/P2 defects, concrete source line and minimal fix. Max 450 words. End with verdict APPROVE or CHANGES_REQUESTED for supplied source only, plus explicit scope limits. No production, complete project, or release acceptance. Generic shared source belongs to Base, business semantics to Ops. Unknown runtime dependencies must be flagged conditional rather than assumed defective.

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
