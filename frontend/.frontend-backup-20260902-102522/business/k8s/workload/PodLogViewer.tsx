import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Space, Tag } from '@arco-design/web-react';
import { IconRefresh } from '@arco-design/web-react/icon';

const MAX_LOG_LINES = 1000;

function buildLogWebSocketUrl(clusterId: number, namespace: string, podName: string, container?: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const base = `${proto}//${window.location.host}/api/v1/business/k8s/clusters/${clusterId}/pods/${namespace}/${podName}/logs`;
  return container ? `${base}?container=${encodeURIComponent(container)}` : base;
}

export interface PodLogViewerProps {
  readonly clusterId: number;
  readonly namespace: string;
  readonly podName: string;
  readonly container?: string;
}

type LogStatus = 'connecting' | 'connected' | 'closed' | 'error';

export default function PodLogViewer({ clusterId, namespace, podName, container }: PodLogViewerProps) {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<string[]>([]);
  const [status, setStatus] = useState<LogStatus>('connecting');
  const [reconnectKey, setReconnectKey] = useState(0);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let disposed = false;
    const ws = new WebSocket(buildLogWebSocketUrl(clusterId, namespace, podName, container));

    ws.onopen = () => {
      if (!disposed) setStatus('connected');
    };
    ws.onmessage = (event) => {
      if (disposed) return;
      const line = typeof event.data === 'string' ? event.data : '';
      setLogs((prev) => {
        const next = [...prev, line];
        return next.length > MAX_LOG_LINES ? next.slice(next.length - MAX_LOG_LINES) : next;
      });
    };
    ws.onclose = () => {
      if (!disposed) setStatus('closed');
    };
    ws.onerror = () => {
      if (!disposed) setStatus('error');
    };

    return () => {
      disposed = true;
      ws.close();
    };
  }, [clusterId, container, namespace, podName, reconnectKey]);

  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logs]);

  const handleReconnect = () => {
    setLogs([]);
    setStatus('connecting');
    setReconnectKey((key) => key + 1);
  };

  const statusMeta: Record<LogStatus, { color: string; text: string }> = {
    connecting: { color: 'orange', text: t('business.k8s.workload.log.connecting') },
    connected: { color: 'green', text: t('business.k8s.workload.log.connected') },
    closed: { color: 'gray', text: t('business.k8s.workload.log.closed') },
    error: { color: 'red', text: t('business.k8s.workload.log.error') },
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Space style={{ marginBottom: 8 }} size={12}>
        <Tag color={statusMeta[status].color}>{statusMeta[status].text}</Tag>
        <Button type="text" size="small" icon={<IconRefresh />} onClick={handleReconnect}>
          {t('business.k8s.workload.log.reconnect')}
        </Button>
        <Button type="text" size="small" onClick={() => setLogs([])}>
          {t('business.k8s.workload.log.clear')}
        </Button>
      </Space>
      <div
        ref={logRef}
        style={{
          flex: 1,
          overflow: 'auto',
          background: '#1e1e1e',
          color: '#d4d4d4',
          fontFamily: "'JetBrains Mono', 'Consolas', monospace",
          fontSize: 12,
          lineHeight: 1.6,
          padding: 12,
          borderRadius: 6,
          minHeight: 320,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all',
        }}
      >
        {logs.length === 0 && status === 'connecting' ? t('business.k8s.workload.log.loading') : logs.join('\n')}
      </div>
    </div>
  );
}
