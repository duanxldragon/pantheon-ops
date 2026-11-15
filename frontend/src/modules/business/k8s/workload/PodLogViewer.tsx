import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Select, Space, Switch, Tag } from '@arco-design/web-react';
import { mergeTaskLogChunks, type TaskLogChunk } from '../../../../components/operational';
import OperationalLogPanel from '../../shared/OperationalLogPanel';

export interface PodLogViewerProps {
  readonly clusterId: number;
  readonly namespace: string;
  readonly podName: string;
  readonly container?: string;
}

interface PodLogOptions { follow: boolean; previous: boolean; timestamps: boolean; sinceSeconds?: number }

function buildLogWebSocketUrl({ clusterId, namespace, podName, container }: PodLogViewerProps, options: PodLogOptions): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const base = `${proto}//${window.location.host}/api/v1/business/k8s/clusters/${clusterId}/pods/${encodeURIComponent(namespace)}/${encodeURIComponent(podName)}/logs`;
  const query = new URLSearchParams({ follow: String(options.follow), previous: String(options.previous), timestamps: String(options.timestamps) });
  if (container) query.set('container', container);
  if (options.sinceSeconds) query.set('sinceSeconds', String(options.sinceSeconds));
  return `${base}?${query}`;
}

type LogStatus = 'connecting' | 'connected' | 'closed' | 'error' | 'forbidden';

function PodLogSession(props: Readonly<PodLogViewerProps & { options: PodLogOptions }>) {
  const { t } = useTranslation();
  const [chunks, setChunks] = useState<TaskLogChunk[]>([]);
  const [status, setStatus] = useState<LogStatus>('connecting');
  const [reconnectKey, setReconnectKey] = useState(0);
  const [truncated, setTruncated] = useState(false);
  const sequence = useRef(0);
  const { clusterId, namespace, podName, container, options } = props;
  const source = `${clusterId} / ${namespace} / ${podName}${container ? ` / ${container}` : ''}`;

  useEffect(() => {
    let disposed = false;
    const ws = new WebSocket(buildLogWebSocketUrl({ clusterId, namespace, podName, container }, options));
    ws.onopen = () => { if (!disposed) setStatus('connected'); };
    ws.onmessage = (event) => {
      if (disposed || typeof event.data !== 'string') return;
      const content = event.data.slice(0, 8192);
      const nextSequence = ++sequence.current;
      if (nextSequence > 1000 || event.data.length > 8192) setTruncated(true);
      setChunks((previous) => mergeTaskLogChunks(previous, [{ sequence: nextSequence, source, stream: 'pod', content }], 1000));
    };
    ws.onclose = (event) => {
      if (disposed) return;
      if (event.code === 4403 || event.code === 1008) {
        disposed = true;
        setChunks([]); setTruncated(false); sequence.current = 0; setStatus('forbidden');
      } else setStatus('closed');
    };
    ws.onerror = () => { if (!disposed) setStatus('error'); };
    return () => { disposed = true; ws.close(); };
  }, [clusterId, namespace, podName, container, options, reconnectKey, source]);

  return (
    <div>
      <Space wrap>
        <Tag>{t(status === 'forbidden' ? 'business.deploy.logs.forbidden' : `business.k8s.workload.log.${status === 'closed' ? 'closed' : status}`)}</Tag>
        <Button onClick={() => { setStatus('connecting'); setTruncated(true); setReconnectKey((value) => value + 1); }}>
          {t('business.k8s.workload.log.reconnect')}
        </Button>
      </Space>
      <OperationalLogPanel key={status === 'forbidden' ? 'denied' : 'source'} title={source} chunks={chunks} truncated={truncated}
        state={status === 'forbidden' ? 'forbidden' : status === 'connecting' ? 'loading' : status === 'closed' ? 'stale' : status === 'error' ? 'error' : 'ready'} />
    </div>
  );
}

function PodLogOptionsViewer(props: Readonly<PodLogViewerProps>) {
  const { t } = useTranslation();
  const [options, setOptions] = useState<PodLogOptions>({ follow: true, previous: false, timestamps: false });
  return <>
    <Space wrap>
      {(['follow', 'previous', 'timestamps'] as const).map((key) => <label key={key}>
        {t(`business.k8s.workload.log.${key}`)}{' '}
        <Switch aria-label={t(`business.k8s.workload.log.${key}`)} checked={options[key]} onChange={(value) => setOptions((current) => ({ ...current, [key]: value }))} />
      </label>)}
      <Select aria-label={t('business.k8s.workload.log.since')} placeholder={t('business.k8s.workload.log.since')}
        value={options.sinceSeconds} allowClear onChange={(value: number | undefined) => setOptions((current) => ({ ...current, sinceSeconds: value }))}
        options={[300, 3600, 86400].map((value) => ({ value, label: t(`business.k8s.workload.log.since${value}`) }))} />
    </Space>
    <PodLogSession key={JSON.stringify(options)} {...props} options={options} />
  </>;
}

export default function PodLogViewer(props: Readonly<PodLogViewerProps>) {
  // Source and query switches get fresh receive buffers and pause/search state.
  return <PodLogOptionsViewer key={`${props.clusterId}:${props.namespace}:${props.podName}:${props.container ?? ''}`} {...props} />;
}
