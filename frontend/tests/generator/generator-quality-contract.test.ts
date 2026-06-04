import assert from 'node:assert/strict';
import test from 'node:test';

import { ModuleExporter } from '../../src/modules/system/generator/exporter';
import {
  buildAuditActionKey,
  buildDashboardQuickActionDescriptionKey,
  buildEnumOptionKey,
  buildFieldLabelKey,
  buildFieldPlaceholderKey,
  buildMenuGroupTitleKey,
  buildPermissionTitleKey,
  buildTitleKey,
  generateDefaultMenus,
  generateDefaultPermissions,
  validateGeneratorCompleteness,
  type ModuleField,
  type ModuleSchema,
} from '../../src/modules/system/generator/schema';

function createFields(): ModuleField[] {
  return [
    {
      name: 'assetCode',
      type: 'string',
      label: '资产编码',
      labelEn: 'Asset Code',
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      placeholder: '请输入资产编码',
      placeholderEn: 'Enter asset code',
      validation: { unique: true, maxLength: 64 },
    },
    {
      name: 'status',
      type: 'enum',
      label: '状态',
      labelEn: 'Status',
      required: true,
      searchable: true,
      sortable: true,
      visibleInList: true,
      visibleInForm: true,
      enumOptions: [
        { value: 'active', label: '启用', labelEn: 'Active', color: 'green' },
        { value: 'inactive', label: '停用', labelEn: 'Inactive', color: 'gray' },
      ],
      validation: { enum: ['active', 'inactive'] },
    },
  ];
}

function createSchema(overrides: Partial<ModuleSchema> = {}): ModuleSchema {
  const schema: ModuleSchema = {
    name: 'cmdb/asset',
    templateVersion: 'v1',
    displayName: '资产管理',
    displayNameEn: 'Asset Management',
    description: '资产台账',
    scope: 'business',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'export', 'import'],
    dependencies: [{ module: 'cmdb/vendor', required: true, reason: '资产需要选择供应商' }],
    relations: [
      {
        name: 'assetVendor',
        type: 'lookup',
        targetModule: 'cmdb/vendor',
        localField: 'vendorId',
        targetField: 'id',
      },
    ],
    dataScopeMode: 'dept',
    metadata: {
      businessContext: 'cmdb',
      businessContextTitle: 'CMDB',
      businessContextTitleEn: 'CMDB',
      tableRole: 'main',
      boundedContext: 'asset',
      owner: 'platform',
      summary: '资产台账',
    },
    model: {
      tableName: 'biz_cmdb_asset',
      modelName: 'CmdbAsset',
      fields: createFields(),
    },
    menus: [],
    permissions: [],
    i18n: {
      namespace: 'business.cmdb.asset',
      translations: { zh: {}, en: {} },
    },
    enableExport: true,
    enableImport: true,
    enableAudit: true,
    enableDataScope: true,
    includeDashboardWidget: true,
    ...overrides,
  };

  schema.menus = generateDefaultMenus(schema);
  schema.permissions = generateDefaultPermissions(schema);
  schema.i18n = buildI18n(schema);
  return schema;
}

function buildI18n(schema: ModuleSchema): ModuleSchema['i18n'] {
  const titleKey = buildTitleKey(schema.scope, schema.name);
  const zh: Record<string, string> = {
    [titleKey]: schema.displayName,
    [buildMenuGroupTitleKey(schema.scope, ['cmdb'])]: 'CMDB',
    [buildDashboardQuickActionDescriptionKey(schema.scope, schema.name)]: '进入资产管理',
    [buildAuditActionKey(schema.scope, schema.name, 'create')]: '新增资产管理',
    [buildAuditActionKey(schema.scope, schema.name, 'update')]: '编辑资产管理',
    [buildAuditActionKey(schema.scope, schema.name, 'delete')]: '删除资产管理',
  };
  const en: Record<string, string> = {
    [titleKey]: schema.displayNameEn || schema.displayName,
    [buildMenuGroupTitleKey(schema.scope, ['cmdb'])]: 'CMDB',
    [buildDashboardQuickActionDescriptionKey(schema.scope, schema.name)]: 'Open Asset Management',
    [buildAuditActionKey(schema.scope, schema.name, 'create')]: 'Create Asset Management',
    [buildAuditActionKey(schema.scope, schema.name, 'update')]: 'Update Asset Management',
    [buildAuditActionKey(schema.scope, schema.name, 'delete')]: 'Delete Asset Management',
  };

  for (const field of schema.model.fields) {
    zh[buildFieldLabelKey(schema.scope, schema.name, field.name)] = field.label;
    en[buildFieldLabelKey(schema.scope, schema.name, field.name)] = field.labelEn || field.label;
    if (field.placeholder || field.placeholderEn) {
      zh[buildFieldPlaceholderKey(schema.scope, schema.name, field.name)] = field.placeholder || '';
      en[buildFieldPlaceholderKey(schema.scope, schema.name, field.name)] = field.placeholderEn || field.placeholder || '';
    }
    for (const item of field.enumOptions ?? []) {
      zh[buildEnumOptionKey(schema.scope, schema.name, field.name, item.value)] = item.label;
      en[buildEnumOptionKey(schema.scope, schema.name, field.name, item.value)] = item.labelEn || item.label;
    }
  }

  for (const action of ['view', 'create', 'update', 'delete', 'export', 'import'] as const) {
    zh[buildPermissionTitleKey(schema.scope, schema.name, action)] = `${action}资产管理`;
    en[buildPermissionTitleKey(schema.scope, schema.name, action)] = `${action} Asset Management`;
  }

  return {
    namespace: 'business.cmdb.asset',
    translations: { zh, en },
  };
}

function generatedFiles(schema: ModuleSchema) {
  const files = new ModuleExporter(schema).generateAll();
  return new Map(files.map((file) => [file.path, file.content]));
}

function assertCompletenessPasses(schema: ModuleSchema) {
  const blockingIssues = validateGeneratorCompleteness(schema).filter((issue) => issue.level === 'error');
  assert.deepEqual(blockingIssues, []);
}

test('generator emits expected business main-table contract', () => {
  const schema = createSchema();
  const files = generatedFiles(schema);

  assertCompletenessPasses(schema);
  assert.deepEqual([...files.keys()], [
    'backend/modules/business/cmdb/asset/asset_model.go',
    'backend/modules/business/cmdb/asset/asset_dto.go',
    'backend/modules/business/cmdb/asset/asset_service.go',
    'backend/modules/business/cmdb/asset/asset_handler.go',
    'backend/modules/business/cmdb/asset/module.go',
    'frontend/src/modules/business/cmdb/asset/index.ts',
    'frontend/src/modules/business/cmdb/asset/api.ts',
    'frontend/src/modules/business/cmdb/asset/CmdbAssetList.tsx',
    'frontend/src/modules/business/cmdb/asset/CmdbAssetForm.tsx',
    'frontend/src/modules/business/cmdb/asset/CmdbAssetDetail.tsx',
  ]);

  const backendModule = files.get('backend/modules/business/cmdb/asset/module.go') || '';
  assert.match(backendModule, /ModuleName:\s+"business\.cmdb\.asset"/);
  assert.match(backendModule, /PagePerm:\s+"business:cmdb:asset:list"/);
  assert.match(backendModule, /Perms:\s+"business:cmdb:asset:create"/);
  assert.match(backendModule, /Key:\s+"business\.cmdb\.asset\.field\.assetCode\.label"/);
  assert.match(backendModule, /Key:\s+"business\.cmdb\.asset\.audit\.create"/);

  const backendService = files.get('backend/modules/business/cmdb/asset/asset_service.go') || '';
  assert.match(backendService, /database\.WithDataScope\(dataScope\)/);
  assert.doesNotMatch(backendService, /backend\/modules\/system\//);

  const frontendIndex = files.get('frontend/src/modules/business/cmdb/asset/index.ts') || '';
  assert.match(frontendIndex, /routes:\s*\[/);
  assert.match(frontendIndex, /menus:\s*\[/);
  assert.match(frontendIndex, /permissions:\s*\[/);
  assert.match(frontendIndex, /i18nNamespaces:\s*\['business\.cmdb\.asset'\]/);
  assert.match(frontendIndex, /dashboardWidgets:\s*\[/);

  const frontendList = files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetList.tsx') || '';
  assert.match(frontendList, /t\('business\.cmdb\.asset\.field\.assetCode\.label'\)/);
  assert.doesNotMatch(frontendList, />资产管理</);
});

test('relation table contract omits navigation and permission bindings', () => {
  const relationSchema = createSchema({
    name: 'cmdb/asset_vendor',
    displayName: '资产供应商关系',
    displayNameEn: 'Asset Vendor Relation',
    pageActions: [],
    dependencies: [],
    relations: [],
    metadata: {
      businessContext: 'cmdb',
      businessContextTitle: 'CMDB',
      businessContextTitleEn: 'CMDB',
      tableRole: 'relation',
      primaryTable: 'biz_cmdb_asset',
      relationFromField: 'asset_id',
      relationToField: 'vendor_id',
    },
    model: {
      tableName: 'biz_cmdb_asset_vendor',
      modelName: 'CmdbAssetVendor',
      fields: createFields(),
    },
    includeDashboardWidget: true,
  });
  const relationFiles = generatedFiles(relationSchema);
  const relationIndex = relationFiles.get('frontend/src/modules/business/cmdb/asset_vendor/index.ts') || '';
  const relationModule = relationFiles.get('backend/modules/business/cmdb/asset_vendor/module.go') || '';

  assertCompletenessPasses(relationSchema);
  assert.match(relationIndex, /routes:\s*\[\]/);
  assert.match(relationIndex, /menus:\s*\[\]/);
  assert.match(relationIndex, /permissions:\s*\[\]/);
  assert.doesNotMatch(relationIndex, /dashboardWidgets:/);
  assert.doesNotMatch(relationModule, /PagePerm:/);
  assert.doesNotMatch(relationModule, /Perms:/);
});
