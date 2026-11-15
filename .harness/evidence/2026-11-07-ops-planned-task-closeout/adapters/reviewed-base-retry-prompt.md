Read-only independent CODE follow-up. No tools. Max 150 words. Report proved P1/P2, otherwise APPROVE for this bounded change only. No release/project approval.
Follow-up to APPROVE with P2 silently deferred retry. Keep request budget: nextAllowedAt remains armed, load waits until allowed time. Do not use proposed reset-to-zero, because it bypasses maxRequestsPerMinute. Handler sets loading=true and error=null, hiding retry button during wait. Retained snapshots get refreshing label; empty ones show PageLoading. Fake-timer tests prove immediate feedback, retained data and 1/min bound. Review only the supplied final behavior; the remaining load code is unchanged.
if (denied) return null;
const stale = snapshot && metadata.freshness.maxAgeMs !== undefined && clock - Date.parse(snapshot.updatedAt) > metadata.freshness.maxAgeMs;
return <Card className="operational-dashboard-widget" title={t(widget.titleKey)} data-slot={widget.slot}>
{loading && !snapshot ? <PageLoading /> : null}
{error ? <PageRequestError error={error} description={t(metadata.errorStateKey)} onRetry={() => { setLoading(true); setError(null); setRetryKey(value => value + 1); }} /> : null}
{snapshot ? <>
<div className="operational-dashboard-widget__freshness">
<DateTimeMeta value={snapshot.updatedAt} showRelative={false} />
{loading ? <Typography.Text>{t('dashboard.operational.refreshing')}</Typography.Text> : null}
{snapshot.timeRangeLabel ? <Typography.Text type="secondary">{snapshot.timeRangeLabel}</Typography.Text> : null}
{stale ? <Typography.Text>{t('dashboard.operational.stale')}</Typography.Text> : null}
{snapshot.partial ? <Typography.Text>{t('dashboard.operational.partial')}</Typography.Text> : null}
</div>
