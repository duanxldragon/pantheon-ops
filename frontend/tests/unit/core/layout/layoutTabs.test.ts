import { describe, expect, it } from 'vitest';
import {
  buildOpenedPageTab,
  selectAccessibleTabs,
} from '../../../../src/core/layout/layoutTabs';

describe('opened page tabs', () => {
  it('drops tabs that are not plan-entitled or lack the page permission', () => {
    const tabs = [
      buildOpenedPageTab('/dashboard', 'Dashboard', { titleKey: 'dashboard.title' }),
      buildOpenedPageTab('/system/modules', 'Modules'),
      buildOpenedPageTab('/system/generator', 'Generator'),
      buildOpenedPageTab('/system/tenant', 'Tenants'),
    ];

    const accessible = selectAccessibleTabs(
      tabs,
      (path) => path !== '/system/generator' && path !== '/system/tenant',
    );

    expect(accessible.map((tab) => tab.path)).toEqual(['/dashboard', '/system/modules']);
  });

  it('keeps every tab when the session is plan-gated and permitted', () => {
    const tabs = [
      buildOpenedPageTab('/dashboard', 'Dashboard'),
      buildOpenedPageTab('/system/generator', 'Generator'),
    ];

    expect(selectAccessibleTabs(tabs, () => true)).toEqual(tabs);
  });
});
