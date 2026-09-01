import React, { useMemo, useState } from 'react';
import { Button, Input, Switch } from '@arco-design/web-react';
import { filterTaskLogChunks, mergeTaskLogChunks } from './logBuffer';
import type { OperationalDataState, TaskLogChunk, TaskLogViewerLabels } from './types';
import './operational.css';

export interface TaskLogViewerProps {
  title: React.ReactNode;
  chunks: TaskLogChunk[];
  labels: TaskLogViewerLabels;
  state?: OperationalDataState;
  windowSize?: number;
  keyword?: string;
  onKeywordChange?: (keyword: string) => void;
  paused?: boolean;
  onPausedChange?: (paused: boolean) => void;
  wrap?: boolean;
  onWrapChange?: (wrap: boolean) => void;
}

export default function TaskLogViewer({
  title,
  chunks,
  labels,
  state = 'ready',
  windowSize,
  keyword,
  onKeywordChange,
  paused = false,
  onPausedChange,
  wrap = false,
  onWrapChange,
}: Readonly<TaskLogViewerProps>) {
  const [localKeyword, setLocalKeyword] = useState('');
  const effectiveKeyword = keyword ?? localKeyword;
  const visibleChunks = useMemo(
    () =>
      filterTaskLogChunks(mergeTaskLogChunks([], chunks, windowSize), {
        keyword: effectiveKeyword,
      }),
    [chunks, effectiveKeyword, windowSize],
  );
  const updateKeyword = (value: string) => {
    setLocalKeyword(value);
    onKeywordChange?.(value);
  };
  return (
    <section className="operational-primitive task-log-viewer" aria-label={String(title)}>
      <div className="operational-primitive__header">
        <div className="operational-primitive__title">{title}</div>
        <div className="operational-primitive__controls">
          <Input.Search
            value={effectiveKeyword}
            placeholder={labels.searchPlaceholder}
            onChange={updateKeyword}
            allowClear
          />
          <Button type={paused ? 'primary' : 'secondary'} onClick={() => onPausedChange?.(!paused)}>
            {paused ? labels.resume : labels.pause}
          </Button>
          <Switch checked={wrap} onChange={onWrapChange} aria-label={labels.wrap} />
        </div>
      </div>
      {state !== 'ready' ? (
        <div className="operational-primitive__state">{labels[state]}</div>
      ) : (
        <div className="operational-primitive__body">
          <div
            className={
              wrap ? 'task-log-viewer__rows task-log-viewer__rows--wrap' : 'task-log-viewer__rows'
            }
          >
            {visibleChunks.map((chunk) => (
              <div className="task-log-viewer__row" key={chunk.sequence}>
                <span className="task-log-viewer__meta">#{chunk.sequence}</span>
                <span className="task-log-viewer__meta">{chunk.level ?? chunk.stream ?? '-'}</span>
                <span className="task-log-viewer__content">{chunk.content}</span>
              </div>
            ))}
          </div>
          <div className="operational-primitive__state" aria-live="polite">
            {labels.showingRows.replace('{count}', String(visibleChunks.length))}
          </div>
        </div>
      )}
    </section>
  );
}
