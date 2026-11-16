import { useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Message, Select, Space, Tag } from '@arco-design/web-react';
import { AppModal } from '../../../components';
import { filterTaskLogChunks, mergeTaskLogChunks, TaskLogViewer, type OperationalDataState, type TaskLogChunk } from '../../../components/operational';
import './operational-log.css';

export interface OperationalLogPanelProps {
  title: string;
  chunks: TaskLogChunk[];
  state?: OperationalDataState;
  truncated?: boolean;
  stream?: string;
  onStreamChange?: (stream: string | undefined) => void;
  actions?: ReactNode;
  redactionNotice?: string;
}

// Pausing freezes the visible snapshot; the adapter keeps receiving and owns
// the bounded buffer. Resuming always shows the latest retained window.
export default function OperationalLogPanel({ title, chunks, state = 'ready', truncated = false, stream, onStreamChange, actions, redactionNotice }: Readonly<OperationalLogPanelProps>) {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<TaskLogChunk[] | null>(null);
  const [wrap, setWrap] = useState(true);
  const [expanded, setExpanded] = useState(false);
  const [localStream, setLocalStream] = useState<string>();
  const selectedStream = onStreamChange ? stream : localStream;
  const [keyword, setKeyword] = useState('');
  const rows = state === 'forbidden' ? [] : snapshot ?? chunks;
  const streams = [...new Set(rows.flatMap((chunk) => chunk.stream ? [chunk.stream] : []))];
  const visibleRows = selectedStream ? rows.filter((chunk) => chunk.stream === selectedStream) : rows;
  const copyRows = filterTaskLogChunks(mergeTaskLogChunks([], visibleRows, 500), { keyword });
  const labels = {
    loading: t('business.deploy.logs.loading'), empty: t('business.deploy.logs.empty'),
    error: t('business.deploy.logs.error'), forbidden: t('business.deploy.logs.forbidden'),
    stale: t('business.deploy.logs.stale'), partial: t('business.deploy.logs.partial'),
    ready: t('business.deploy.logs.ready'), searchPlaceholder: t('business.deploy.logs.search'),
    pause: t('business.deploy.logs.pause'), resume: t('business.deploy.logs.resume'),
    wrap: t('business.deploy.logs.wrap'), showingRows: t('business.deploy.logs.rows'),
  };
  const content = (
    <>
      <Space wrap className="business-log-panel__toolbar">
        {state !== 'ready' ? <Tag>{labels[state]}</Tag> : null}
        {truncated ? <Tag color="orange">{t('business.deploy.logs.truncated')}</Tag> : null}
        <Button aria-pressed={expanded} onClick={() => setExpanded((value) => !value)}>
          {t(expanded ? 'business.deploy.logs.exitFullscreen' : 'business.deploy.logs.fullscreen')}
        </Button>
        <Select aria-label={t('business.deploy.logs.stream')} placeholder={t('business.deploy.logs.stream')}
          value={selectedStream} onChange={onStreamChange ?? setLocalStream} allowClear options={streams.map((value) => ({ value, label: value }))} />
        <Button disabled={state === 'forbidden' || copyRows.length === 0} onClick={() => {
          const text = copyRows.map((row) => row.content).join('\n');
          void Promise.resolve().then(() => navigator.clipboard.writeText(text)).then(() => Message.success(t('business.deploy.logs.copied'))).catch(() => Message.error(t('business.deploy.logs.copyFailed')));
        }}>{t('business.deploy.logs.copy')}</Button>
        {actions}
      </Space>
      {redactionNotice ? <p role="status">{redactionNotice}</p> : null}
      <TaskLogViewer title={title} chunks={visibleRows} labels={labels}
        state={state === 'forbidden' ? 'forbidden' : visibleRows.length > 0 ? 'ready' : state === 'ready' ? 'empty' : state}
        keyword={keyword} onKeywordChange={setKeyword}
        windowSize={500} paused={snapshot !== null}
        onPausedChange={(paused) => setSnapshot(paused ? [...chunks] : null)}
        wrap={wrap} onWrapChange={setWrap} />
    </>
  );
  return <div className="business-log-panel">
    {expanded ? <AppModal title={title} visible escToExit focusLock autoFocus footer={null} onCancel={() => setExpanded(false)}
      className="business-log-dialog" style={{ width: 'calc(100vw - 32px)' }}>{content}</AppModal> : content}
  </div>;
}
