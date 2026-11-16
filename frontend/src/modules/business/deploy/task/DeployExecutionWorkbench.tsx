import type { ReactNode } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button, Select, Space, Tag } from '@arco-design/web-react';
import { ExecutionStepRail, type ExecutionStep } from '../../../../components/operational';
import OperationalLogPanel from '../../shared/OperationalLogPanel';
import type { DeployTaskHostRow, DeployTaskRow, DeployTaskStepRow } from '../api';
import { useDeployTaskLogs } from './useDeployTaskLogs';
import DeployLogDownload from './DeployLogDownload';

export default function DeployExecutionWorkbench({ task, stale, renderHost }: Readonly<{
  task: DeployTaskRow;
  stale: boolean;
  renderHost: (host: DeployTaskHostRow) => ReactNode;
}>) {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  const hosts = task.hosts ?? [];
  const selectedHost = hosts.find((host) => String(host.id) === params.get('host'))
    ?? hosts.find((host) => host.status === 'failed') ?? hosts[0];
  const hostSteps = (task.steps ?? []).filter((step) => step.taskHostId === selectedHost?.id);
  const historySteps = (task.stepHistory ?? []).filter((step) => step.taskHostId === selectedHost?.id);
  const recordedSteps = new Map<string, DeployTaskStepRow & { capturedAt?: string }>();
  for (const step of historySteps) recordedSteps.set(`${step.id}:${step.attemptNo}`, step);
  for (const step of hostSteps) {
    const key = `${step.id}:${step.attemptNo}`;
    if (!recordedSteps.has(key) || recordedSteps.get(key)!.stateVersion <= step.stateVersion) recordedSteps.set(key, step);
  }
  const attemptSteps = [...recordedSteps.values()];
  const attempts = [...new Set([...attemptSteps.map((step) => step.attemptNo), ...(task.logAttempts ?? []).filter((entry) => entry.taskHostId === selectedHost?.id).map((entry) => entry.attemptNo)]
    .filter((attempt) => Number.isSafeInteger(attempt) && attempt > 0))].sort((a, b) => a - b);
  const requestedAttempt = params.get('attempt');
  const linkedStep = hostSteps.find((step) => String(step.id) === params.get('step'))
    ?? historySteps.find((step) => String(step.id) === params.get('step'));
  const explicitAttempt = requestedAttempt && /^[1-9]\d{0,9}$/.test(requestedAttempt) && Number(requestedAttempt) <= 2147483647
    ? Number(requestedAttempt) : undefined;
  const failedCheckpoint = hostSteps.filter((step) => step.status === 'failed')
    .sort((a, b) => a.stepOrder - b.stepOrder || a.id - b.id)[0];
  const currentAttempts = new Set(hostSteps.map((step) => step.attemptNo).filter((attempt) => attempt > 0));
  const defaultAttempt = linkedStep && linkedStep.attemptNo > 0 ? linkedStep.attemptNo
    : failedCheckpoint && failedCheckpoint.attemptNo > 0 ? failedCheckpoint.attemptNo
    : currentAttempts.size > 1 ? undefined : attempts.at(-1);
  const selectedAttempt = requestedAttempt === 'all' ? undefined : explicitAttempt ?? defaultAttempt;
  const displayAttempts = selectedAttempt !== undefined && !attempts.includes(selectedAttempt)
    ? [...attempts, selectedAttempt].sort((a, b) => a - b) : attempts;
  const steps = (selectedAttempt === undefined ? hostSteps : attemptSteps.filter((step) => step.attemptNo === selectedAttempt))
    .sort((a, b) => a.stepOrder - b.stepOrder || a.id - b.id);
  const selectedStep = steps.find((step) => String(step.id) === params.get('step'));
  const capturedTimes = steps.map((step) => 'capturedAt' in step ? Date.parse(step.capturedAt as string) : Number.NaN).filter(Number.isFinite);
  const latestCapture = capturedTimes.length > 0 ? new Date(Math.max(...capturedTimes)).toISOString() : undefined;
  const active = ['pending', 'running', 'pause_requested', 'cancel_requested'].includes(task.status);
  const { rows, state, truncated, reload } = useDeployTaskLogs(task.id, active, { hostId: selectedHost?.id, attemptNo: selectedAttempt, stepId: selectedStep?.id, firstError: params.get('firstError') === '1' });
  const setSelection = (values: Record<string, string | undefined>) => {
    const next = new URLSearchParams(params);
    next.delete('firstError');
    for (const [key, value] of Object.entries(values)) {
      if (value === undefined) next.delete(key); else next.set(key, value);
    }
    setParams(next, { replace: true });
  };
  const chooseHost = (id: number | string) => setSelection({ host: String(id), attempt: undefined, step: undefined, stream: undefined });
  const chunks = rows.filter((row) => row.taskHostId === selectedHost?.id
    && (selectedAttempt === undefined || row.attemptNo === selectedAttempt)
    && (!selectedStep || row.taskStepId === selectedStep.id))
    .map((row) => ({ sequence: row.sequence, timestamp: row.createdAt, stream: row.stream, source: selectedHost?.hostname,
      content: row.content }));
  const stepStatuses = new Set(['pending', 'running', 'success', 'warning', 'failed', 'skipped', 'canceled']);
  const rail: ExecutionStep[] = steps.map((step) => ({ id: String(step.id), title: step.stepKey,
    attempts: step.attemptNo, status: (stepStatuses.has(step.status) ? step.status : 'warning') as ExecutionStep['status'] }));
  const labels = {
    loading: t('business.deploy.logs.loading'), empty: t('business.deploy.logs.empty'),
    error: t('business.deploy.logs.error'), forbidden: t('business.deploy.logs.forbidden'),
    stale: t('business.deploy.logs.stale'), partial: t('business.deploy.logs.partial'), ready: t('business.deploy.logs.ready'),
    attempts: t('business.deploy.logs.attempts'), duration: t('business.deploy.task.duration'), jumpToStep: t('business.deploy.logs.jumpStep'),
  };
  if (!selectedHost) return null;
  return (
    <section className="deploy-execution-workbench" aria-label={t('business.deploy.logs.context')}>
      <Space wrap>
        <Select virtualListProps={{ height: 240 }} value={selectedHost.id} onChange={chooseHost} aria-label={t('business.deploy.logs.host')}
          options={hosts.map((host) => ({ value: host.id, label: `${host.hostname} / ${host.hostIp}` }))} />
        {displayAttempts.length > 0 ? <Select value={selectedAttempt === undefined ? 'all' : String(selectedAttempt)}
          aria-label={t('business.deploy.logs.attempts')}
          options={[{ value: 'all', label: t('business.deploy.logs.allAttempts') }, ...displayAttempts.map((attempt) => ({ value: String(attempt), label: attempts.includes(attempt) ? `#${attempt}` : t('business.deploy.logs.unrecordedAttempt', { attempt }) }))]}
          onChange={(attempt) => setSelection({ host: String(selectedHost.id), attempt, step: undefined, stream: undefined })} /> : null}
        <Button disabled={!hosts.some((host) => host.status === 'failed')} onClick={() => {
          const host = hosts.find((item) => item.status === 'failed');
          if (!host) return;
          const failedHostSteps = (task.steps ?? []).filter((step) => step.taskHostId === host.id);
          const failedStep = failedHostSteps.filter((step) => step.status === 'failed')
            .sort((a, b) => a.stepOrder - b.stepOrder || a.id - b.id)[0];
          setSelection({ host: String(host.id), attempt: failedStep && failedStep.attemptNo > 0 ? String(failedStep.attemptNo) : 'all',
            step: failedStep ? String(failedStep.id) : undefined, stream: 'stderr', firstError: '1' });
        }}>{t('business.deploy.logs.jumpFailure')}</Button>
        <Button onClick={reload}>{t('business.deploy.logs.reconnect')}</Button>
        <Tag>{selectedHost.hostname} / {selectedHost.hostIp}</Tag>
      </Space>
      {task.stepHistoryLimited ? <p role="status">{t('business.deploy.logs.historyLimited')}</p> : null}
      {selectedAttempt === undefined && attempts.length > 0 ? <p>{t('business.deploy.logs.allAttemptsCheckpoint')}</p> : null}
      {latestCapture ? <div role="status">
        <Tag>{t('business.deploy.logs.historySnapshot')}</Tag>
        <p>{t('business.deploy.logs.historyCapturedAt', { time: latestCapture })}</p>
      </div> : null}
      {selectedAttempt !== undefined && steps.length === 0 ? <Tag>{t('business.deploy.logs.historicalAttempt')}</Tag> : null}
      {steps.length > 0 ? <ExecutionStepRail title={t('business.deploy.logs.steps')} steps={rail} labels={labels}
        activeStepId={selectedStep ? String(selectedStep.id) : undefined}
        onStepClick={(step) => setSelection({ host: String(selectedHost.id), attempt: selectedAttempt === undefined ? 'all' : String(selectedAttempt), step: step.id, stream: undefined })} /> : null}
      {params.get('firstError') === '1' ? <Button onClick={() => setSelection({ firstError: undefined })}>{t('business.deploy.logs.showStepLogs')}</Button> : null}
      {selectedStep ? <Button onClick={() => setSelection({ step: undefined, stream: undefined })}>{t('business.deploy.logs.allSteps')}</Button> : null}
      <OperationalLogPanel key={`${task.id}:${selectedHost.id}:${selectedAttempt ?? 'all'}:${selectedStep?.id ?? 'all'}:${params.get('firstError') ?? ''}:${state === 'forbidden'}`}
        title={`${task.name} / ${selectedHost.hostname}${selectedAttempt !== undefined ? ` / #${selectedAttempt}` : ''}${selectedStep ? ` / ${selectedStep.stepKey}` : ''}`}
        chunks={chunks} state={stale && state === 'ready' ? 'stale' : state} truncated={truncated}
        redactionNotice={t('business.deploy.logs.knownRedaction')}
        actions={<DeployLogDownload taskId={task.id} forbidden={state === 'forbidden'} filter={{ hostId: selectedHost.id, stepId: selectedStep?.id, attemptNo: selectedAttempt, firstError: params.get('firstError') === '1' }} />}
        stream={params.get('stream') || undefined} onStreamChange={(stream) => setSelection({ stream })} />
      {renderHost(selectedHost)}
    </section>
  );
}
