import { describe, expect, it } from 'vitest';
import type { DashboardWidgetDefinition } from '../../../../src/core/router/types';
import {
  buildDashboardWidgetRegistry,
  getVisibleDashboardWidgets,
} from '../../../../src/modules/platform/widgets';

function operationalWidget(
  overrides: Partial<DashboardWidgetDefinition> = {},
): DashboardWidgetDefinition {
  return {
    key: 'ops.status',
    slot: 'status-summary',
    sourceDomain: 'business/cmdb',
    titleKey: 'business.cmdb.dashboard.status',
    descriptionKey: 'business.cmdb.dashboard.statusDesc',
    path: '/business/cmdb',
    permission: 'business:cmdb:list',
    cleanupPolicy: 'remove_with_source_module',
    registrationOwner: 'cmdb',
    metadata: {
      owner: 'cmdb',
      cleanupPolicy: 'remove_with_source_module',
      freshness: {
        policy: 'snapshot',
        maxAgeMs: 300_000,
        refreshIntervalMs: 60_000,
        timeRangeKey: 'business.cmdb.dashboard.timeRange',
      },
      queryBudget: {
        maxRequestsPerMinute: 4,
        maxItems: 80,
        maxRenderItems: 12,
      },
      emptyStateKey: 'business.cmdb.dashboard.empty',
      errorStateKey: 'business.cmdb.dashboard.error',
      errorIsolation: 'widget',
    },
    ...overrides,
  } as DashboardWidgetDefinition;
}

describe('dashboard widget registry', () => {
  it('keeps legacy quick-action widgets compatible', () => {
    const widgets = buildDashboardWidgetRegistry([
      {
        name: 'legacy',
        dashboardWidgets: [
          {
            key: 'platform.legacy',
            slot: 'quick-action',
            sourceDomain: 'system/iam',
            titleKey: 'system.menu.user',
            descriptionKey: 'dashboard.quickAction.user',
            path: '/system/user',
            permission: 'system:user:list',
            icon: 'user',
            cleanupPolicy: 'hide_when_forbidden',
          },
        ],
      },
    ]);

    expect(widgets).toHaveLength(1);
  });

  it('requires operational metadata and bounded query budgets', () => {
    expect(() =>
      buildDashboardWidgetRegistry([{ name: 'ops', dashboardWidgets: [operationalWidget()] }]),
    ).not.toThrow();

    expect(() =>
      buildDashboardWidgetRegistry([
        {
          name: 'ops',
          dashboardWidgets: [
            operationalWidget({
              metadata: {
                ...operationalWidget().metadata!,
                queryBudget: { maxRequestsPerMinute: 60, maxItems: 80 },
              },
            }),
          ],
        },
      ]),
    ).toThrow(/request budget/);

    expect(() =>
      buildDashboardWidgetRegistry([
        {
          name: 'ops',
          dashboardWidgets: [
            operationalWidget({
              metadata: {
                ...operationalWidget().metadata!,
                queryBudget: { maxRequestsPerMinute: 0, maxItems: 80 },
              },
            }),
          ],
        },
      ]),
    ).toThrow(/request budget/);
  });

  it('filters forbidden widgets before callers request or render them', () => {
    const [widget] = buildDashboardWidgetRegistry([
      { name: 'ops', dashboardWidgets: [operationalWidget()] },
    ]);

    expect(
      getVisibleDashboardWidgets([widget], {
        menuTree: [{ id: 1, title: 'hidden', path: '/business/cmdb', children: [] }],
        isAdmin: false,
        hasPerm: () => false,
      }),
    ).toEqual([]);
    expect(
      getVisibleDashboardWidgets([widget], {
        menuTree: [{ id: 1, title: 'visible', path: '/business/cmdb', children: [] }],
        isAdmin: false,
        hasPerm: (permission) => permission === 'business:cmdb:list',
      }),
    ).toEqual([widget]);
  });
});
