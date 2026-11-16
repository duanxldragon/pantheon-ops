import i18n from 'i18next';
import { apiRequest } from '@duanxldragon/pantheon-base-ui';
import { formatDateTime, isValidDateValue } from '../../../core/format/dateTime';
import type {
  DashboardOperationalSnapshot,
  DashboardOperationalWidget,
} from '../../../core/router/types';

interface SummaryRow {
  id: number;
  name: string;
  status: string;
  businessScopeId?: number;
  kind?: string;
  observedAt?: string;
}
interface SummarySnapshot {
  items: SummaryRow[];
  partial: boolean;
  sampledAt: string;
  rangeStart?: string;
  rangeEnd?: string;
}
interface WidgetOptions {
  key: string;
  domain: 'deploy' | 'k8s' | 'observability';
  slot: 'status-summary' | 'attention-queue' | 'recent-activity';
  path: string;
  url: string;
  permission: string;
  status?: 'failed' | 'error';
  filters?: boolean;
  timeRange?: boolean;
}

const MAX_ITEMS = 8;
const statuses = new Set([
  'draft', 'pending', 'running', 'success', 'failed', 'canceled', 'pause_requested',
  'paused', 'cancel_requested', 'healthy', 'unreachable', 'unknown', 'degraded', 'unhealthy',
  'active', 'error', 'firing', 'resolved',
]);

function loadSummary(options: WidgetOptions) {
  return async ({ signal, maxItems }: { signal: AbortSignal; maxItems: number }): Promise<DashboardOperationalSnapshot> => {
    signal.throwIfAborted();
    const limit = Number.isInteger(maxItems) && maxItems > 0 ? Math.min(MAX_ITEMS, maxItems) : MAX_ITEMS;
    const result = await apiRequest<SummarySnapshot>({
      url: options.url,
      method: 'GET',
      params: { dashboard: 'true', limit, ...(options.status ? { status: options.status } : {}) },
      signal,
      timeout: 5000,
      skipErrorMessage: true,
    });
    signal.throwIfAborted();
    if (!result || !Array.isArray(result.items) || result.items.length > limit ||
        !Number.isFinite(Date.parse(result.sampledAt)) || typeof result.partial !== 'boolean') {
      throw new Error('Invalid business dashboard summary');
    }
    if (options.timeRange && (!isValidDateValue(result.rangeStart) || !isValidDateValue(result.rangeEnd) ||
        Date.parse(result.rangeEnd!) - Date.parse(result.rangeStart!) !== 86400000)) {
      throw new Error('Invalid business dashboard time range');
    }
    const ids = new Set<number>();
    const items = result.items.map((row) => {
      if (!row || !Number.isSafeInteger(row.id) || row.id <= 0 || ids.has(row.id) ||
          typeof row.name !== 'string' || typeof row.status !== 'string' ||
          (row.observedAt != null && (!isValidDateValue(row.observedAt) || Date.parse(row.observedAt) > Date.parse(result.sampledAt) + 60000)) ||
          (row.businessScopeId !== undefined && (!Number.isSafeInteger(row.businessScopeId) || row.businessScopeId < 0))) {
        throw new Error('Invalid business dashboard row');
      }
      ids.add(row.id);
      const status = statuses.has(row.status) ? row.status : 'other';
      const query = options.filters ? {
        keyword: row.name.slice(0, 128),
        ...(status !== 'other' ? { status } : {}),
        ...(row.businessScopeId ? { businessScopeId: String(row.businessScopeId) } : {}),
        ...(options.timeRange ? { createdFrom: result.rangeStart!, createdTo: result.rangeEnd! } : {}),
      } : undefined;
      const details = [];
      if (row.kind) {
        const kind = ['prometheus', 'victoriametrics', 'loki', 'tempo'].includes(row.kind) ? row.kind : 'other';
        details.push(i18n.t(`business.dashboard.kind.${kind}`));
      }
      details.push(row.observedAt ?
        i18n.t('business.dashboard.observedAt', { time: formatDateTime(row.observedAt) }) :
        i18n.t('business.dashboard.observationUnavailable'));
      return {
        id: String(row.id),
        label: row.name.trim().slice(0, 160) || i18n.t('business.dashboard.record', { id: row.id }),
        value: i18n.t(`business.dashboard.status.${status}`),
        detail: details.join(' · '),
        ...(query ? { query } : {}),
      };
    });
    return {
      updatedAt: result.sampledAt,
      timeRangeLabel: options.timeRange ? i18n.t('business.dashboard.last24Hours', { count: limit }) :
        i18n.t('business.dashboard.recentSample', { count: limit }),
      partial: result.partial,
      items,
    };
  };
}

function widget(options: WidgetOptions): DashboardOperationalWidget {
  const owner = `business/${options.domain}` as const;
  return {
    key: `business.${options.key}`,
    slot: options.slot,
    sourceDomain: owner,
    titleKey: `business.dashboard.${options.key}.title`,
    descriptionKey: `business.dashboard.${options.key}.description`,
    path: options.path,
    permission: options.permission,
    cleanupPolicy: 'remove_with_source_module',
    navigationSource: 'menu',
    registrationOwner: owner,
    metadata: {
      owner,
      cleanupPolicy: 'remove_with_source_module',
      freshness: { policy: 'snapshot', maxAgeMs: 180000, refreshIntervalMs: 60000 },
      queryBudget: { maxRequestsPerMinute: 1, maxItems: MAX_ITEMS, maxRenderItems: MAX_ITEMS },
      emptyStateKey: 'business.dashboard.empty',
      errorStateKey: 'business.dashboard.error',
      errorIsolation: 'widget',
    },
    dataSource: { timeoutMs: 5000, load: loadSummary(options) },
  };
}

export const deployDashboardWidgets = [
  widget({ key: 'deployFailed', domain: 'deploy', slot: 'attention-queue', path: '/business/deploy/task',
    url: '/business/deploy/tasks', permission: 'business:deploy:task:view', status: 'failed', filters: true, timeRange: true }),
  widget({ key: 'deployRecent', domain: 'deploy', slot: 'recent-activity', path: '/business/deploy/task',
    url: '/business/deploy/tasks', permission: 'business:deploy:task:view', filters: true, timeRange: true }),
];
export const k8sDashboardWidgets = [
  { ...widget({ key: 'workloads', domain: 'k8s', slot: 'attention-queue', path: '/business/k8s/workload',
    url: '/business/k8s/workloads', permission: 'business:k8s:workload:list' }),
    dataSource: { timeoutMs: 5000, load: loadWorkloadSummary } },
  widget({ key: 'clusters', domain: 'k8s', slot: 'status-summary', path: '/business/k8s/cluster',
    url: '/business/k8s/clusters', permission: 'business:k8s:cluster:list', filters: true }),
];

async function loadWorkloadSummary({ signal, maxItems }: { signal: AbortSignal; maxItems: number }): Promise<DashboardOperationalSnapshot> {
  signal.throwIfAborted();
  const limit = Number.isInteger(maxItems) && maxItems > 0 ? Math.min(MAX_ITEMS, maxItems) : MAX_ITEMS;
  const result = await apiRequest<{ sampledAt: string; partial: boolean; clusterId?: number; businessScopeId?: number;
    items: { id: string; name: string; namespace: string; status: string }[] }>({
    url: '/business/k8s/workloads', method: 'GET', params: { dashboard: 'true', limit }, signal, timeout: 5000, skipErrorMessage: true,
  });
  signal.throwIfAborted();
  if (!result || !isValidDateValue(result.sampledAt) || !Array.isArray(result.items) || result.items.length > limit ||
      result.partial !== true || (result.items.length > 0 && (!Number.isSafeInteger(result.clusterId) || result.clusterId! <= 0))) {
    throw new Error('Invalid workload dashboard summary');
  }
  const ids = new Set<string>();
  const items = result.items.map((row) => {
    if (!row || typeof row.name !== 'string' || !row.name || row.name.length > 253 ||
        typeof row.namespace !== 'string' || !row.namespace || row.namespace.length > 63 ||
        row.id !== row.namespace + '/' + row.name || ids.has(row.id)) throw new Error('Invalid workload dashboard row');
    ids.add(row.id);
    const status = ['ready', 'progressing', 'scaled_down'].includes(row.status) ? row.status : 'unknown';
    return { id: row.id, label: row.name.slice(0, 160), value: i18n.t(`business.dashboard.status.${status}`),
      detail: i18n.t('business.dashboard.workloadContext', { cluster: result.clusterId, namespace: row.namespace }),
      query: { clusterId: String(result.clusterId), namespace: row.namespace, kind: 'deployment', keyword: row.name } };
  });
  return { updatedAt: result.sampledAt, partial: true,
    timeRangeLabel: i18n.t('business.dashboard.workloadSample', { count: limit }), items };
}
export const observabilityDashboardWidgets = [
  widget({ key: 'alertRules', domain: 'observability', slot: 'attention-queue', path: '/business/observability/alerts',
    url: '/business/observability/alert-rules', permission: 'business:observability:alert_rule:view', status: 'error', filters: true }),
  widget({ key: 'sources', domain: 'observability', slot: 'status-summary', path: '/business/observability/monitoring',
    url: '/business/observability/datasources', permission: 'business:observability:metric_source:view' }),
];
