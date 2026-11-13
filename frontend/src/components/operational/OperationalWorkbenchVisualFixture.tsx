import { useMemo, useState } from 'react';
import { Card, Typography } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import AppTable from '../data-display/AppTable';
import SubmitBar from '../patterns/actions/SubmitBar';
import ChangeDiff from './ChangeDiff';
import ConditionBuilder from './ConditionBuilder';
import ContextSelector from './ContextSelector';
import ExecutionStepRail from './ExecutionStepRail';
import TaskLogViewer from './TaskLogViewer';
import type { ConditionAstNode } from './types';
import type { DashboardOperationalWidget } from '../../core/router/types';
import OperationalDashboardWidgets from '../../modules/platform/OperationalDashboardWidgets';
import {
  conditionAstFixture,
  conditionFieldFixture,
  createContextSelectorFixture,
  createExecutionStepFixture,
  createTaskLogFixture,
} from './fixtures';

const stateKeys = ['loading', 'empty', 'error', 'forbidden', 'stale', 'partial'] as const;

function fixtureSlot(index: number): DashboardOperationalWidget['slot'] {
  if (index === 0) return 'attention-queue';
  if (index === 3) return 'recent-activity';
  return 'status-summary';
}

function fixtureTitleKey(state: string): string {
  if (state === 'empty') return 'common.noData';
  if (state === 'error') return 'common.loadFailed';
  return `operational.fixture.${state}`;
}

function useOperationalLabels() {
  const { t } = useTranslation();
  return {
    loading: t('common.loading'),
    empty: t('common.empty'),
    error: t('common.error'),
    forbidden: t('common.forbidden'),
    stale: t('operational.fixture.stale'),
    partial: t('operational.fixture.partial'),
    ready: t('operational.fixture.ready'),
  };
}

export default function OperationalWorkbenchVisualFixture() {
  const { t } = useTranslation();
  const stateLabels = useOperationalLabels();
  const [submitState, setSubmitState] = useState<'dirty' | 'submitting' | 'error'>('dirty');
  const [logPaused, setLogPaused] = useState(false);
  const [condition, setCondition] = useState<ConditionAstNode>(conditionAstFixture);
  const editorMode = new URLSearchParams(globalThis.location.search).get('editor') === '1';
  const [selectedContext, setSelectedContext] = useState(
    [] as ReturnType<typeof createContextSelectorFixture>,
  );
  const contextOptions = useMemo(() => createContextSelectorFixture(80), []);
  const selectContext = (option: (typeof contextOptions)[number]) =>
    setSelectedContext((current) => [...current, option]);

  const dashboardFixtures = useMemo<DashboardOperationalWidget[]>(
    () =>
      ['partial', 'empty', 'error', 'ready'].map((state, index) => ({
        key: `fixture-${state}`,
        slot: fixtureSlot(index),
        sourceDomain: 'platform',
        titleKey: fixtureTitleKey(state),
        descriptionKey: 'operational.fixture.description',
        path: '/__visual/operational-workbench',
        permission: 'visual:view',
        cleanupPolicy: 'platform_owned',
        navigationSource: 'direct',
        metadata: {
          owner: 'platform',
          cleanupPolicy: 'platform_owned',
          freshness: { policy: 'snapshot', maxAgeMs: 60000 },
          queryBudget: { maxRequestsPerMinute: 2, maxItems: 4, maxRenderItems: 4 },
          emptyStateKey: 'common.noDataDesc',
          errorStateKey: 'common.loadFailedDesc',
          errorIsolation: 'widget',
        },
        dataSource: {
          timeoutMs: 1000,
          load: () => {
            if (state === 'error') return Promise.reject(new Error('fixture failure'));
            return Promise.resolve({
              updatedAt: new Date(Date.now() - (state === 'partial' ? 120000 : 0)).toISOString(),
              partial: state === 'partial',
              items:
                state === 'empty'
                  ? []
                  : [
                      { id: '1', label: t('operational.fixture.rowAlpha'), value: 12 },
                      {
                        id: '2',
                        label: t('operational.fixture.rowBeta'),
                        value: 8,
                        detail: t('operational.fixture.description'),
                      },
                    ],
            });
          },
        },
      })),
    [t],
  );
  if (new URLSearchParams(globalThis.location.search).get('dashboard') === '1') {
    return (
      <main className="operational-visual-fixture">
        <OperationalDashboardWidgets widgets={dashboardFixtures} />
      </main>
    );
  }
  return (
    <main className="operational-visual-fixture" data-testid="operational-workbench-fixture">
      <Typography.Title heading={3}>{t('operational.fixture.title')}</Typography.Title>
      <Typography.Paragraph>{t('operational.fixture.description')}</Typography.Paragraph>
      <section className="operational-visual-fixture__grid">
        <Card title={t('operational.fixture.b1')}>
          <SubmitBar
            sticky
            status={submitState}
            statusText={t(`operational.fixture.submit.${submitState}`)}
            errorSummary={
              submitState === 'error' ? t('operational.fixture.submit.errorSummary') : undefined
            }
            onSubmit={() => {
              setSubmitState('submitting');
              globalThis.setTimeout(() => setSubmitState('error'), 250);
            }}
            onCancel={() => setSubmitState('dirty')}
            secondaryActions={[
              { key: 'draft', label: t('operational.fixture.saveDraft'), onClick: () => undefined },
            ]}
          />
        </Card>
        <Card title={t('operational.fixture.b2')}>
          <AppTable
            viewKey="visual-fixture-workbench"
            rowKey="id"
            pagination={false}
            data={[
              {
                id: '1',
                name: t('operational.fixture.rowAlpha'),
                state: t('operational.fixture.ready'),
              },
              {
                id: '2',
                name: t('operational.fixture.rowBeta'),
                state: t('operational.fixture.stale'),
              },
            ]}
            columns={[
              {
                title: t('operational.fixture.name'),
                dataIndex: 'name',
                columnKey: 'name',
                width: 180,
              },
              { title: t('operational.fixture.status'), dataIndex: 'state', columnKey: 'state' },
            ]}
          />
        </Card>
        <TaskLogViewer
          title={t('operational.fixture.logs')}
          chunks={createTaskLogFixture(600)}
          windowSize={120}
          paused={logPaused}
          onPausedChange={setLogPaused}
          labels={{
            ...stateLabels,
            searchPlaceholder: t('common.search'),
            pause: t('operational.fixture.pause'),
            resume: t('operational.fixture.resume'),
            wrap: t('operational.fixture.wrap'),
            showingRows: t('operational.fixture.showingRows'),
          }}
        />
        <ChangeDiff
          title={t('operational.fixture.diff')}
          lines={[
            {
              key: 'apiToken',
              kind: 'modified',
              before: t('operational.fixture.hidden'),
              after: t('operational.fixture.hidden'),
            },
            { key: 'replicas', kind: 'modified', before: '2', after: '3' },
            { key: 'region', kind: 'unchanged', before: 'cn', after: 'cn' },
          ]}
          labels={{
            ...stateLabels,
            kindLabels: {
              added: t('operational.fixture.diff.added'),
              removed: t('operational.fixture.diff.removed'),
              modified: t('operational.fixture.diff.modified'),
              unchanged: t('operational.fixture.diff.unchanged'),
              conflict: t('operational.fixture.diff.conflict'),
            },
            before: t('operational.fixture.before'),
            after: t('operational.fixture.after'),
            sensitive: t('operational.fixture.masked'),
            unchangedHidden: t('operational.fixture.unchangedHidden'),
            expandUnchanged: t('operational.fixture.expand'),
            collapseUnchanged: t('operational.fixture.collapse'),
          }}
        />
        <ConditionBuilder
          title={t('operational.fixture.conditions')}
          value={condition}
          onChange={editorMode ? setCondition : undefined}
          editorLabels={
            editorMode
              ? {
                  field: t('operational.fixture.editor.field'),
                  operator: t('operational.fixture.editor.operator'),
                  value: t('operational.fixture.editor.value'),
                  remove: t('operational.fixture.editor.remove'),
                  addGroup: t('operational.fixture.editor.addGroup'),
                  combinator: t('operational.fixture.editor.combinator'),
                }
              : undefined
          }
          fields={conditionFieldFixture}
          labels={{
            ...stateLabels,
            and: t('operational.fixture.and'),
            or: t('operational.fixture.or'),
            operatorLabels: {
              eq: t('operational.fixture.operator.eq'),
              neq: t('operational.fixture.operator.neq'),
              unknown: t('operational.fixture.operator.unknown'),
            },
            addRule: t('operational.fixture.addRule'),
            invalidField: t('operational.fixture.invalidField'),
            unserializableValue: t('operational.fixture.unserializableValue'),
          }}
        />
        <ContextSelector
          title={t('operational.fixture.context')}
          options={contextOptions}
          selected={selectedContext}
          onSelect={selectContext}
          onRemove={(option) =>
            setSelectedContext((current) => current.filter((item) => item.id !== option.id))
          }
          labels={{
            ...stateLabels,
            searchPlaceholder: t('common.search'),
            available: t('operational.fixture.available'),
            selected: t('common.selected'),
            add: t('common.add'),
            remove: t('operational.fixture.remove'),
            excluded: t('operational.fixture.excluded'),
            staleItem: t('operational.fixture.stale'),
          }}
        />
        <ExecutionStepRail
          title={t('operational.fixture.steps')}
          steps={createExecutionStepFixture(60)}
          maxVisibleSteps={20}
          labels={{
            ...stateLabels,
            attempts: t('operational.fixture.attempts'),
            duration: t('operational.fixture.duration'),
            jumpToStep: t('operational.fixture.jumpToStep'),
          }}
        />
        <Card title={t('operational.fixture.b4')} className="operational-visual-fixture__registry">
          {stateKeys.map((state) => (
            <span key={state} data-dashboard-slot={state}>
              {t(`operational.fixture.slot.${state}`)}
            </span>
          ))}
        </Card>
      </section>
    </main>
  );
}
