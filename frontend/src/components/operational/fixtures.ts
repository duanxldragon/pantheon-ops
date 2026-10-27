import type {
  ConditionAstNode,
  ConditionFieldOption,
  ContextSelectorOption,
  ExecutionStep,
  TaskLogChunk,
} from './types';

export function createTaskLogFixture(count = 1000): TaskLogChunk[] {
  return Array.from({ length: count }, (_, index) => ({
    sequence: index + 1,
    level: index % 17 === 0 ? 'error' : 'info',
      content: `log-${index + 1}-payload`,
  }));
}

export const conditionFieldFixture: ConditionFieldOption[] = [
  { key: 'status', label: 'status', operators: ['eq', 'neq'] },
];
export const conditionAstFixture: ConditionAstNode = {
  type: 'group',
  id: 'root',
  combinator: 'and',
  children: [{ type: 'rule', id: 'status-rule', field: 'status', operator: 'eq', value: 'active' }],
};

export function createContextSelectorFixture(count = 500): ContextSelectorOption[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `candidate-${index + 1}`,
    source: 'fixture',
    label: `candidate ${index + 1}`,
    stale: index % 19 === 0,
  }));
}

export function createExecutionStepFixture(count = 60): ExecutionStep[] {
  const resolveStatus = (index: number): ExecutionStep['status'] => {
    if (index === 4) {
      return 'failed';
    }
    if (index < 4) {
      return 'success';
    }
    return 'pending';
  };

  return Array.from({ length: count }, (_, index) => ({
    id: `step-${index + 1}`,
    title: `step ${index + 1}`,
    status: resolveStatus(index),
    attempts: 1,
  }));
}
