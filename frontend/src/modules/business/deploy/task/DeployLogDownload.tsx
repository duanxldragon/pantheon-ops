import { useEffect, useRef, useState } from 'react';
import { Button, Message } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { usePermission } from '../../../../hooks/usePermission';
import { downloadDeployTaskLogs, type DeployTaskLogFilter } from '../api';

export default function DeployLogDownload({ taskId, filter, forbidden }: Readonly<{
  taskId: number; filter: DeployTaskLogFilter; forbidden: boolean;
}>) {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const allowed = !forbidden && hasPerm('business:deploy:task:export') && hasPerm('business:deploy:task:detail');
  const [busySource, setBusySource] = useState<string | null>(null);
  const source = JSON.stringify([allowed, taskId, filter.hostId, filter.stepId, filter.attemptNo, filter.firstError]);
  const busy = busySource === source;
  const request = useRef<AbortController | null>(null);
  useEffect(() => () => { request.current?.abort(); request.current = null; }, [allowed, taskId, filter.hostId, filter.stepId, filter.attemptNo, filter.firstError]);
  const download = async () => {
    if (!allowed || busy) return;
    const controller = new AbortController();
    request.current = controller;
    setBusySource(source);
    try {
      const snapshot = await downloadDeployTaskLogs(taskId, controller.signal, filter);
      if (controller.signal.aborted) return;
      const header = `${t('business.deploy.logs.knownRedaction')}\n${snapshot.truncated ? t('business.deploy.logs.downloadLimited') : ''}`;
      const text = `${header}\n${snapshot.items.map((row) => `#${row.sequence} ${row.createdAt} [${row.stream}] ${row.content}`).join('\n')}`;
      const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }));
      const link = document.createElement('a');
      link.href = url;
      link.download = `task-${taskId}-logs${snapshot.truncated ? '-partial' : ''}.txt`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 0);
      if (snapshot.truncated) Message.warning(t('business.deploy.logs.downloadLimited'));
    } catch {
      if (!controller.signal.aborted) Message.error(t('business.deploy.logs.downloadFailed'));
    } finally {
      if (request.current === controller) { request.current = null; setBusySource(null); }
    }
  };
  if (!allowed) return null;
  return <Button loading={busy} onClick={() => { void download(); }}>{t('business.deploy.logs.download')}</Button>;
}
