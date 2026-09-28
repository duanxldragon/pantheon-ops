import type React from 'react';

export type OperationalDataState =
  'loading' | 'empty' | 'error' | 'forbidden' | 'stale' | 'partial' | 'ready';

export type OperationalStateLabels = Record<OperationalDataState, React.ReactNode>;

export interface TaskLogChunk {
  sequence: number;
  timestamp?: string;
  source?: string;
  stream?: string;
  level?: string;
  content: string;
}

export interface TaskLogViewerLabels extends OperationalStateLabels {
  searchPlaceholder: string;
  pause: React.ReactNode;
  resume: React.ReactNode;
  wrap: string;
  showingRows: string;
}

export interface ChangeDiffLine {
  key: string;
  kind: 'added' | 'removed' | 'modified' | 'unchanged' | 'conflict';
  before?: React.ReactNode;
  after?: React.ReactNode;
  label?: React.ReactNode;
  sensitive?: boolean;
}

export interface ChangeDiffLabels extends OperationalStateLabels {
  kindLabels: Record<ChangeDiffLine['kind'], React.ReactNode>;
  before: string;
  after: string;
  sensitive: React.ReactNode;
  unchangedHidden: React.ReactNode;
  expandUnchanged: React.ReactNode;
  collapseUnchanged: React.ReactNode;
}

export type ConditionAstNode = ConditionGroup | ConditionRule;

export interface ConditionRule {
  type: 'rule';
  id: string;
  field: string;
  operator: string;
  value: unknown;
}

export interface ConditionGroup {
  type: 'group';
  id: string;
  combinator: 'and' | 'or';
  children: ConditionAstNode[];
}

export interface ConditionFieldOption {
  key: string;
  label: React.ReactNode;
  operators: string[];
  disabled?: boolean;
}

export interface ContextSelectorOption {
  id: string;
  source: string;
  label: React.ReactNode;
  snapshot?: Record<string, unknown>;
  disabled?: boolean;
  stale?: boolean;
}

export interface ExecutionStep {
  id: string;
  title: React.ReactNode;
  status: 'pending' | 'running' | 'success' | 'warning' | 'failed' | 'skipped' | 'canceled';
  durationMs?: number;
  attempts?: number;
}
