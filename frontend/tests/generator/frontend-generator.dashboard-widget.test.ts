import assert from 'node:assert/strict';

import { FrontendGenerator } from '../../src/modules/system/generator/frontend-generator';
import type { ModuleSchema } from '../../src/modules/system/generator/schema';

function createSchema(overrides: Partial<ModuleSchema> = {}): ModuleSchema {
  const name = overrides.name ?? 'cmdb/asset';
  const scope = overrides.scope ?? 'business';

  return {
    name,
    displayName: 'Asset',
    displayNameEn: 'Asset',
    scope,
    templateLevel: 'enterprise',
    pageActions: ['view', 'create', 'update', 'delete'],
    metadata: {
      businessContext: scope === 'business' ? 'cmdb' : undefined,
      tableRole: 'main',
      ...overrides.metadata,
    },
    model: {
      tableName: scope === 'business' ? 'biz_cmdb_asset' : 'system_asset',
      modelName: 'Asset',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '名称',
          labelEn: 'Name',
          required: true,
          searchable: true,
          visibleInList: true,
          visibleInForm: true,
        },
      ],
      ...overrides.model,
    },
    menus: [],
    permissions: [],
    i18n: {
      namespace: scope === 'business' ? 'business.cmdb.asset' : 'system.asset',
      translations: {
        zh: {},
        en: {},
      },
    },
    enableAudit: true,
    enableDataScope: scope === 'business',
    includeDashboardWidget: true,
    ...overrides,
  };
}

function generateIndex(schema: ModuleSchema): string {
  return new FrontendGenerator(schema).generateModuleIndex();
}

function assertHasDashboardWidget(source: string) {
  assert.match(source, /dashboardWidgets:\s*\[/);
  assert.match(source, /slot: 'quick-action'/);
  assert.match(source, /sourceDomain: 'business\/cmdb'/);
  assert.match(source, /descriptionKey: 'business\.cmdb\.asset\.dashboard\.quickAction'/);
  assert.match(source, /cleanupPolicy: 'remove_with_source_module'/);
  assert.match(source, /registrationOwner: 'business\.cmdb\.asset'/);
}

function assertOmitsDashboardWidget(source: string) {
  assert.doesNotMatch(source, /dashboardWidgets:\s*\[/);
  assert.doesNotMatch(source, /slot: 'quick-action'/);
  assert.doesNotMatch(source, /registrationOwner:/);
}

assertHasDashboardWidget(generateIndex(createSchema()));

assertOmitsDashboardWidget(generateIndex(createSchema({ includeDashboardWidget: false })));

assertOmitsDashboardWidget(generateIndex(createSchema({
  includeDashboardWidget: true,
  metadata: {
    businessContext: 'cmdb',
    tableRole: 'relation',
  },
})));

console.log('generator dashboard widget contract tests passed');
