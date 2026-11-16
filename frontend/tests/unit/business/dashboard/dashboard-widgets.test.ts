import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  deployDashboardWidgets,
  k8sDashboardWidgets,
  observabilityDashboardWidgets,
} from '../../../../src/modules/business/shared/dashboardWidgets';

const client = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock('@duanxldragon/pantheon-base-ui', () => ({ apiRequest: client.request }));
vi.mock('i18next', () => ({ default: { t: (key: string) => key } }));
const response = () => ({ sampledAt: '2026-11-14T10:00:00Z', partial: true,
  rangeStart: '2026-11-13T10:00:00Z', rangeEnd: '2026-11-14T10:00:00Z',
  items: [{ id: 42, name: 'release task', status: 'failed', businessScopeId: 7 }] });
afterEach(() => client.request.mockReset());

describe('business dashboard provider boundaries', () => {
  it('spends no request for canceled work and rejects a late canceled reply', async () => {
    const controller = new AbortController();
    controller.abort();
    await expect(deployDashboardWidgets[0].dataSource!.load({ signal: controller.signal, maxItems: 8 })).rejects.toThrow();
    expect(client.request).not.toHaveBeenCalled();
    const current = new AbortController();
    client.request.mockImplementation(async () => { current.abort(); return response(); });
    await expect(deployDashboardWidgets[0].dataSource!.load({ signal: current.signal, maxItems: 8 })).rejects.toThrow();
  });

  it('uses the smaller renderer budget and preserves the authorized list filters', async () => {
    client.request.mockResolvedValue(response());
    const signal = new AbortController().signal;
    const snapshot = await deployDashboardWidgets[0].dataSource!.load({ signal, maxItems: 2 });
    expect(client.request).toHaveBeenCalledWith(expect.objectContaining({
      url: '/business/deploy/tasks', signal, timeout: 5000, skipErrorMessage: true,
      params: { dashboard: 'true', limit: 2, status: 'failed' },
    }));
    expect(snapshot).toMatchObject({ partial: true, updatedAt: response().sampledAt,
      items: [{ id: '42', query: { keyword: 'release task', status: 'failed', businessScopeId: '7', createdFrom: response().rangeStart, createdTo: response().rangeEnd } }] });
  });

  it('rejects oversized and malformed provider replies rather than silently certifying a full sample', async () => {
    const load = deployDashboardWidgets[1].dataSource!.load;
    const args = { signal: new AbortController().signal, maxItems: 2 };
    client.request.mockResolvedValue({ ...response(), items: [1, 2, 3].map((id) => ({ ...response().items[0], id })) });
    await expect(load(args)).rejects.toThrow('Invalid business dashboard summary');
    client.request.mockResolvedValue({ ...response(), items: [response().items[0], response().items[0]] });
    await expect(load(args)).rejects.toThrow('Invalid business dashboard row');
    client.request.mockResolvedValue({ ...response(), sampledAt: 'invalid-time' });
    await expect(load(args)).rejects.toThrow('Invalid business dashboard summary');
  });

  it('projects only bounded safe display fields and never carries secrets or endpoint data', async () => {
    client.request.mockResolvedValue({ ...response(), items: [{ ...response().items[0],
      name: 'X'.repeat(1000), status: 'unexpected-server-state', password: 'secret-marker', url: 'secret-endpoint-marker',
    }] });
    const snapshot = await k8sDashboardWidgets.find((widget) => widget.key === 'business.clusters')!.dataSource!.load({ signal: new AbortController().signal, maxItems: 1000 });
    expect(client.request.mock.calls[0][0].params.limit).toBe(8);
    expect(snapshot.items[0].label).toHaveLength(160);
    expect(snapshot.items[0].query?.keyword).toHaveLength(128);
    expect(snapshot.items[0].value).toBe('business.dashboard.status.other');
    expect(JSON.stringify(snapshot)).not.toMatch(/secret-marker|secret-endpoint-marker|unexpected-server-state/);
  });

  it('defines permission, owner, cleanup, freshness and one-request-per-minute budgets for every provider', () => {
    for (const widget of [...deployDashboardWidgets, ...k8sDashboardWidgets, ...observabilityDashboardWidgets]) {
      expect(widget.permission).toMatch(/^business:/);
      expect(widget.registrationOwner).toBe(widget.sourceDomain);
      expect(widget.cleanupPolicy).toBe('remove_with_source_module');
      expect(widget.metadata).toMatchObject({ owner: widget.sourceDomain, cleanupPolicy: widget.cleanupPolicy,
        errorIsolation: 'widget', freshness: { refreshIntervalMs: 60000 },
        queryBudget: { maxRequestsPerMinute: 1, maxItems: 8 } });
    }
  });

  it('does not invent datasource filters unsupported by the monitoring page', async () => {
    client.request.mockResolvedValue(response());
    const widget = observabilityDashboardWidgets.find((item) => item.key === 'business.sources')!;
    const result = await widget.dataSource!.load({ signal: new AbortController().signal, maxItems: 8 });
    expect(result.items[0].query).toBeUndefined();
    expect(widget.path).toBe('/business/observability/monitoring');
  });

  it('requires a precise 24h deployment range and valid observation metadata', async () => {
    const load = deployDashboardWidgets[0].dataSource!.load;
    const args = { signal: new AbortController().signal, maxItems: 8 };
    client.request.mockResolvedValue({ ...response(), rangeStart: undefined });
    await expect(load(args)).rejects.toThrow('Invalid business dashboard time range');
    client.request.mockResolvedValue({ ...response(), rangeStart: '2026-11-12T10:00:00Z' });
    await expect(load(args)).rejects.toThrow('Invalid business dashboard time range');
    client.request.mockResolvedValue({ ...response(), items: [{ ...response().items[0], observedAt: '2026-11-15T10:00:00Z' }] });
    await expect(load(args)).rejects.toThrow('Invalid business dashboard row');
  });

  it('projects a bounded Deployment sample with cluster, namespace, kind and exact-name navigation', async () => {
    const widget = k8sDashboardWidgets.find((item) => item.key === 'business.workloads')!;
    const args = { signal: new AbortController().signal, maxItems: 2 };
    client.request.mockResolvedValue({ sampledAt: response().sampledAt, partial: true, clusterId: 71,
      items: [{ id: 'payments/api', namespace: 'payments', name: 'api', status: 'ready', images: ['secret-marker'] }] });
    const snapshot = await widget.dataSource!.load(args);
    expect(client.request).toHaveBeenCalledTimes(1);
    expect(client.request).toHaveBeenCalledWith(expect.objectContaining({ params: { dashboard: 'true', limit: 2 }, signal: args.signal }));
    expect(snapshot).toMatchObject({ partial: true, items: [{ id: 'payments/api', value: 'business.dashboard.status.ready',
      query: { clusterId: '71', namespace: 'payments', kind: 'deployment', keyword: 'api' } }] });
    expect(JSON.stringify(snapshot)).not.toContain('secret-marker');
    client.request.mockResolvedValue({ sampledAt: response().sampledAt, partial: false, items: [] });
    await expect(widget.dataSource!.load(args)).rejects.toThrow('Invalid workload dashboard summary');
    client.request.mockResolvedValue({ sampledAt: response().sampledAt, partial: true, clusterId: 71,
      items: [{ id: 'other/api', namespace: 'payments', name: 'api', status: 'ready' }] });
    await expect(widget.dataSource!.load(args)).rejects.toThrow('Invalid workload dashboard row');
  });

  it('navigates recorded alert evaluation errors with supported list filters', async () => {
    const widget = observabilityDashboardWidgets.find((item) => item.key === 'business.alertRules')!;
    client.request.mockResolvedValue({ ...response(), items: [{ id: 9, name: 'evaluation', status: 'error' }] });
    const snapshot = await widget.dataSource!.load({ signal: new AbortController().signal, maxItems: 8 });
    expect(widget.path).toBe('/business/observability/alerts');
    expect(snapshot.items[0].query).toEqual({ keyword: 'evaluation', status: 'error' });
  });
});
