import React, { useState } from 'react';
import { Button, Tag } from '@arco-design/web-react';
import type { ChangeDiffLabels, ChangeDiffLine, OperationalDataState } from './types';
import './operational.css';

export interface ChangeDiffProps {
  title: React.ReactNode;
  lines: ChangeDiffLine[];
  labels: ChangeDiffLabels;
  state?: OperationalDataState;
  riskSummary?: React.ReactNode;
  sensitiveKeyPattern?: RegExp;
}

export default function ChangeDiff({
  title,
  lines,
  labels,
  state = 'ready',
  riskSummary,
  sensitiveKeyPattern = /secret|token|password|credential/i,
}: Readonly<ChangeDiffProps>) {
  const [showUnchanged, setShowUnchanged] = useState(false);
  const visibleLines = showUnchanged ? lines : lines.filter((line) => line.kind !== 'unchanged');
  return (
    <section className="operational-primitive change-diff" aria-label={String(title)}>
      <div className="operational-primitive__header">
        <div className="operational-primitive__title">{title}</div>
        <Button onClick={() => setShowUnchanged((value) => !value)}>
          {showUnchanged ? labels.collapseUnchanged : labels.expandUnchanged}
        </Button>
      </div>
      {state !== 'ready' ? (
        <div className="operational-primitive__state">{labels[state]}</div>
      ) : (
        <div className="operational-primitive__body change-diff__grid">
          {riskSummary}
          {visibleLines.length === 0 ? (
            <div className="operational-primitive__state">{labels.unchangedHidden}</div>
          ) : null}
          {visibleLines.map((line) => {
            const guarded = line.sensitive || sensitiveKeyPattern.test(line.key);
            return (
              <div className={`change-diff__row change-diff__row--${line.kind}`} key={line.key}>
                <div className="change-diff__cell">
                  <Tag>{labels.kindLabels[line.kind]}</Tag> {line.label ?? line.key}
                </div>
                <div className="change-diff__cell" aria-label={labels.before}>
                  {guarded ? labels.sensitive : line.before}
                </div>
                <div className="change-diff__cell" aria-label={labels.after}>
                  {guarded ? labels.sensitive : line.after}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
