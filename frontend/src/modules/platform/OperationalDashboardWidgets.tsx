import { useEffect, useRef, useState } from 'react';
import { Button, Card, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { DateTimeMeta, PageEmpty, PageLoading, PageRequestError } from '../../components';
import { isForbiddenRequestError } from '../../api/request';
import type {
  DashboardOperationalSnapshot,
  DashboardOperationalWidget,
} from '../../core/router/types';
import './operational-dashboard.css';

function validateSnapshot(snapshot: DashboardOperationalSnapshot, maxItems: number) {
  if (
    !snapshot ||
    !Number.isFinite(Date.parse(snapshot.updatedAt)) ||
    Date.parse(snapshot.updatedAt) > Date.now() + 60000 ||
    !Array.isArray(snapshot.items) ||
    snapshot.items.length > maxItems ||
    (snapshot.timeRangeLabel?.length ?? 0) > 240
  ) {
    throw new Error('Invalid dashboard provider snapshot');
  }
  const ids = new Set<string>();
  for (const item of snapshot.items) {
    if (
      !item ||
      typeof item.id !== 'string' ||
      !item.id ||
      ids.has(item.id) ||
      typeof item.label !== 'string' ||
      item.label.length > 240 ||
      (item.detail?.length ?? 0) > 480 ||
      (item.value !== undefined &&
        typeof item.value !== 'string' &&
        typeof item.value !== 'number') ||
      (typeof item.value === 'number' && !Number.isFinite(item.value)) ||
      (typeof item.value === 'string' && item.value.length > 240) ||
      (item.query &&
        (Object.keys(item.query).length > 12 ||
          Object.entries(item.query).some(
            ([key, value]) => key.length > 80 || typeof value !== 'string' || value.length > 240,
          )))
    ) {
      throw new Error('Invalid dashboard provider item');
    }
    ids.add(item.id);
  }
  return snapshot;
}

function FreshnessFlags({
  snapshot,
  loading,
  stale,
}: Readonly<{ snapshot: DashboardOperationalSnapshot; loading: boolean; stale: boolean }>) {
  const { t } = useTranslation();
  return (
    <div className="operational-dashboard-widget__freshness">
      <DateTimeMeta value={snapshot.updatedAt} showRelative={false} />
      {loading ? <Typography.Text>{t('dashboard.operational.refreshing')}</Typography.Text> : null}
      {snapshot.timeRangeLabel ? (
        <Typography.Text type="secondary">{snapshot.timeRangeLabel}</Typography.Text>
      ) : null}
      {stale ? <Typography.Text>{t('dashboard.operational.stale')}</Typography.Text> : null}
      {snapshot.partial ? (
        <Typography.Text>{t('dashboard.operational.partial')}</Typography.Text>
      ) : null}
    </div>
  );
}

function OperationalWidget({ widget }: Readonly<{ widget: DashboardOperationalWidget }>) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [snapshot, setSnapshot] = useState<DashboardOperationalSnapshot | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [loading, setLoading] = useState(true);
  const [denied, setDenied] = useState(false);
  const [clock, setClock] = useState(Date.now);
  const [retryKey, setRetryKey] = useState(0);
  const nextAllowedAt = useRef(0);
  const metadata = widget.metadata!;
  useEffect(() => {
    let active = true;
    let controller: AbortController | undefined;
    let retry: ReturnType<typeof setTimeout> | undefined;
    const source = widget.dataSource;
    if (!source) return;
    const refresh = Math.max(
      30000,
      60000 / metadata.queryBudget.maxRequestsPerMinute,
      metadata.freshness.refreshIntervalMs ?? 30000,
    );
    const load = async () => {
      const now = Date.now();
      if (now < nextAllowedAt.current) {
        retry = setTimeout(load, nextAllowedAt.current - now);
        return;
      }
      nextAllowedAt.current = now + refresh;
      let stopPolling = false;
      controller = new AbortController();
      const signal = controller.signal;
      let deadline: ReturnType<typeof setTimeout> | undefined;
      try {
        const timeout = new Promise<never>((_, reject) => {
          deadline = setTimeout(() => {
            controller?.abort();
            reject(new Error('Dashboard provider timeout'));
          }, source.timeoutMs);
        });
        const data = await Promise.race([
          source.load({ signal, maxItems: metadata.queryBudget.maxItems }),
          timeout,
        ]);
        if (!active) return;
        setSnapshot(validateSnapshot(data, metadata.queryBudget.maxItems));
        setError(null);
        setClock(Date.now());
      } catch (reason) {
        if (!active) return;
        if (isForbiddenRequestError(reason)) {
          setSnapshot(null);
          setDenied(true);
          stopPolling = true;
          controller?.abort();
          return;
        }
        setError(reason);
      } finally {
        if (deadline) clearTimeout(deadline);
        if (active) {
          setLoading(false);
          if (!stopPolling && metadata.freshness.refreshIntervalMs !== undefined)
            retry = setTimeout(load, refresh);
        }
      }
    };
    // StrictMode replays setup/cleanup before this microtask. A canceled setup
    // must not spend the initial request budget or defer first data by a minute.
    void Promise.resolve().then(() => {
      if (active) void load();
    });
    return () => {
      active = false;
      controller?.abort();
      if (retry) clearTimeout(retry);
    };
  }, [widget, metadata, retryKey]);
  useEffect(() => {
    if (!snapshot || !metadata.freshness.maxAgeMs) return;
    const remaining = Date.parse(snapshot.updatedAt) + metadata.freshness.maxAgeMs - Date.now();
    if (remaining <= 0) return;
    const timer = setTimeout(() => setClock(Date.now()), Math.min(remaining + 1, 2147483647));
    return () => clearTimeout(timer);
  }, [snapshot, metadata.freshness.maxAgeMs]);
  if (denied) return null;
  const stale =
    snapshot &&
    metadata.freshness.maxAgeMs !== undefined &&
    clock - Date.parse(snapshot.updatedAt) > metadata.freshness.maxAgeMs;
  return (
    <Card
      className="operational-dashboard-widget"
      title={t(widget.titleKey)}
      data-slot={widget.slot}
    >
      {loading && !snapshot ? <PageLoading /> : null}
      {error ? (
        <PageRequestError
          error={error}
          description={t(metadata.errorStateKey)}
          onRetry={() => {
            setLoading(true);
            setError(null);
            setRetryKey((value) => value + 1);
          }}
        />
      ) : null}
      {snapshot ? (
        <>
          <FreshnessFlags snapshot={snapshot} loading={loading} stale={Boolean(stale)} />
          {snapshot.items.length === 0 ? (
            <PageEmpty description={t(metadata.emptyStateKey)} />
          ) : (
            <ul className="operational-dashboard-widget__items">
              {snapshot.items
                .slice(0, metadata.queryBudget.maxRenderItems ?? metadata.queryBudget.maxItems)
                .map((item) => (
                  <li key={item.id}>
                    <Button
                      type="text"
                      onClick={() =>
                        navigate(
                          widget.path +
                            (item.query ? '?' + new URLSearchParams(item.query).toString() : ''),
                        )
                      }
                    >
                      <span>{item.label}</span>
                      {item.value !== undefined ? <strong>{item.value}</strong> : null}
                    </Button>
                    {item.detail ? (
                      <Typography.Text type="secondary">{item.detail}</Typography.Text>
                    ) : null}
                  </li>
                ))}
            </ul>
          )}
        </>
      ) : null}
    </Card>
  );
}

const priority = {
  'attention-queue': 0,
  'status-summary': 1,
  'trend-snapshot': 2,
  'recent-activity': 3,
};
export default function OperationalDashboardWidgets({
  widgets,
}: Readonly<{ widgets: readonly DashboardOperationalWidget[] }>) {
  const supported = widgets
    .filter((widget) => widget.dataSource)
    .slice()
    .sort((a, b) => priority[a.slot] - priority[b.slot])
    .slice(0, 8);
  if (!supported.length) return null;
  return (
    <section className="operational-dashboard-widgets">
      {supported.map((widget) => (
        <OperationalWidget key={widget.key} widget={widget} />
      ))}
    </section>
  );
}
