import React from 'react';
import { Tag } from '@arco-design/web-react';
import type { ExecutionStep, OperationalDataState } from './types';
import './operational.css';

export interface ExecutionStepRailProps {
  title: React.ReactNode;
  steps: ExecutionStep[];
  labels: { attempts: React.ReactNode; duration: React.ReactNode; jumpToStep: string } & Record<
    OperationalDataState,
    React.ReactNode
  >;
  state?: OperationalDataState;
  maxVisibleSteps?: number;
  activeStepId?: string;
  onStepClick?: (step: ExecutionStep) => void;
}
export default function ExecutionStepRail({
  title,
  steps,
  labels,
  state = 'ready',
  maxVisibleSteps = 80,
  activeStepId,
  onStepClick,
}: Readonly<ExecutionStepRailProps>) {
  const visibleSteps = steps.slice(0, Math.max(1, maxVisibleSteps));
  return (
    <section className="operational-primitive execution-step-rail" aria-label={String(title)}>
      <div className="operational-primitive__header">
        <div className="operational-primitive__title">{title}</div>
      </div>
      {state !== 'ready' ? (
        <div className="operational-primitive__state">{labels[state]}</div>
      ) : (
        <ol className="execution-step-rail__list">
          {visibleSteps.map((step, index) => (
            <li key={step.id}>
              <button
                type="button"
                className={`execution-step-rail__step execution-step-rail__step--${step.status}`}
                aria-current={activeStepId === step.id ? 'step' : undefined}
                aria-label={`${labels.jumpToStep} ${index + 1}`}
                onClick={() => onStepClick?.(step)}
              >
                <span className="execution-step-rail__marker" aria-hidden="true" />
                <span>{step.title}</span>
                <span>
                  <Tag>{step.status}</Tag>
                  {step.attempts ? ` ${labels.attempts}: ${step.attempts}` : ''}
                  {step.durationMs ? ` ${labels.duration}: ${step.durationMs}ms` : ''}
                </span>
              </button>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
