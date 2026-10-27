import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  ChangeDiff,
  ConditionBuilder,
  ContextSelector,
  ExecutionStepRail,
  TaskLogViewer,
  collectConditionNodeIds,
  createContextSelectorFixture,
  createExecutionStepFixture,
  createTaskLogFixture,
  filterTaskLogChunks,
  mergeTaskLogChunks,
  validateConditionAst,
} from '../../../../src/components/operational';

const conditionBuilderLabels = {
  loading: 'loading',
  empty: 'empty',
  error: 'error',
  forbidden: 'forbidden',
  stale: 'stale',
  partial: 'partial',
  ready: 'ready',
  and: 'and',
  or: 'or',
  operatorLabels: { eq: 'equals', unknown: 'unknown' },
  addRule: 'add',
  invalidField: 'invalid',
  unserializableValue: 'unavailable',
};

describe('operational primitive helpers', () => {
  it('deduplicates logs by sequence, sorts them, and caps the rendered window', () => {
    const merged = mergeTaskLogChunks(
      [
        { sequence: 2, content: 'two' },
        { sequence: 1, content: 'one' },
      ],
      [
        { sequence: 2, content: 'two-new' },
        { sequence: 3, content: 'three' },
      ],
      2,
    );

    expect(merged.map((item) => item.sequence)).toEqual([2, 3]);
    expect(merged[0]?.content).toBe('two-new');
  });

  it('filters logs without mutating source order', () => {
    const filtered = filterTaskLogChunks(createTaskLogFixture(30), {
      keyword: 'payload',
      levels: ['error'],
    });

    expect(filtered.every((item) => item.level === 'error')).toBe(true);
    expect(filtered.map((item) => item.sequence)).toEqual([1, 18]);
  });

  it('round-trips condition AST node ids and flags removed fields', () => {
    const ast = {
      type: 'group' as const,
      id: 'root',
      combinator: 'and' as const,
      children: [
        {
          type: 'rule' as const,
          id: 'stale',
          field: 'missing',
          operator: 'eq' as const,
          value: '1',
        },
      ],
    };

    expect(collectConditionNodeIds(ast)).toEqual(['root', 'stale']);
    expect(
      validateConditionAst(ast, [{ key: 'status', label: 'status', operators: ['eq'] }]),
    ).toEqual(['stale']);
  });
});

describe('operational primitive components', () => {
  it('bounds large log DOM output', () => {
    render(
      <TaskLogViewer
        title="logs"
        chunks={createTaskLogFixture(10_000)}
        labels={{
          loading: 'loading',
          empty: 'empty',
          error: 'error',
          forbidden: 'forbidden',
          stale: 'stale',
          partial: 'partial',
          search: 'search',
          searchPlaceholder: 'search',
          pause: 'pause',
          resume: 'resume',
          wrap: 'wrap',
          showingRows: '{count}',
        }}
        windowSize={120}
      />,
    );

    expect(document.querySelectorAll('.task-log-viewer__row')).toHaveLength(120);
  });

  it('guards sensitive diff keys even when callers forget to mark them', () => {
    render(
      <ChangeDiff
        title="diff"
        lines={[
          { key: 'apiToken', kind: 'modified', before: 'plain-before', after: 'plain-after' },
        ]}
        labels={{
          loading: 'loading',
          empty: 'empty',
          error: 'error',
          forbidden: 'forbidden',
          stale: 'stale',
          partial: 'partial',
          kindLabels: {
            added: 'added',
            removed: 'removed',
            modified: 'modified',
            unchanged: 'unchanged',
            conflict: 'conflict',
          },
          before: 'before',
          after: 'after',
          sensitive: 'masked',
          unchangedHidden: 'hidden',
          expandUnchanged: 'expand',
          collapseUnchanged: 'collapse',
        }}
      />,
    );

    expect(screen.getAllByText('masked')).toHaveLength(2);
    expect(screen.queryByText('plain-before')).toBeNull();
  });

  it('limits context candidates and records stale choices', () => {
    render(
      <ContextSelector
        title="context"
        options={createContextSelectorFixture(500)}
        selected={[]}
        labels={{
          loading: 'loading',
          empty: 'empty',
          error: 'error',
          forbidden: 'forbidden',
          stale: 'stale',
          partial: 'partial',
          operatorLabels: { eq: 'equals', neq: 'not equals', unknown: 'unknown' },
          searchPlaceholder: 'search',
          available: 'available',
          selected: 'selected',
          excluded: 'excluded',
          add: 'add',
          remove: 'remove',
          staleItem: 'stale-item',
        }}
        maxVisibleOptions={40}
      />,
    );

    expect(document.querySelectorAll('.context-selector__option')).toHaveLength(40);
    expect(screen.getAllByText('stale-item').length).toBeGreaterThan(0);
  });

  it('limits long execution rails without inferring retry legality', () => {
    render(
      <ExecutionStepRail
        title="steps"
        steps={createExecutionStepFixture(120)}
        labels={{
          loading: 'loading',
          empty: 'empty',
          error: 'error',
          forbidden: 'forbidden',
          stale: 'stale',
          partial: 'partial',
          attempts: 'attempts',
          duration: 'duration',
          jumpToStep: 'jump',
        }}
        maxVisibleSteps={50}
      />,
    );

    expect(document.querySelectorAll('.execution-step-rail__step')).toHaveLength(50);
    expect(screen.getByRole('list')).toBeTruthy();
    expect(screen.getAllByRole('listitem')).toHaveLength(50);
  });

  it('renders structured condition values without object stringification', () => {
    render(
      <ConditionBuilder
        title="conditions"
        value={{
          type: 'group',
          id: 'root',
          combinator: 'and',
          children: [
            { type: 'rule', id: 'metadata', field: 'status', operator: 'eq', value: { active: true } },
          ],
        }}
        fields={[{ key: 'status', label: 'status', operators: ['eq'] }]}
        labels={conditionBuilderLabels}
      />,
    );

    expect(screen.getByText('{"active":true}')).toBeTruthy();
  });

  it('shows a supplied fallback when a structured condition value cannot be serialized', () => {
    const circularValue: { self?: unknown } = {};
    circularValue.self = circularValue;

    render(
      <ConditionBuilder
        title="conditions"
        value={{
          type: 'group',
          id: 'root',
          combinator: 'and',
          children: [
            { type: 'rule', id: 'metadata', field: 'status', operator: 'eq', value: circularValue },
          ],
        }}
        fields={[{ key: 'status', label: 'status', operators: ['eq'] }]}
        labels={conditionBuilderLabels}
      />,
    );

    expect(screen.getByText('unavailable')).toBeTruthy();
  });

  it('shows the fallback when custom serialization has no result', () => {
    render(
      <ConditionBuilder
        title="conditions"
        value={{
          type: 'group',
          id: 'root',
          combinator: 'and',
          children: [
            {
              type: 'rule',
              id: 'metadata',
              field: 'status',
              operator: 'eq',
              value: { toJSON: () => undefined },
            },
          ],
        }}
        fields={[{ key: 'status', label: 'status', operators: ['eq'] }]}
        labels={conditionBuilderLabels}
      />,
    );

    expect(screen.getByText('unavailable')).toBeTruthy();
  });
});
