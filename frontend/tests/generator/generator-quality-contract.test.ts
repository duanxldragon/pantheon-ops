import assert from 'node:assert/strict';
import { existsSync, mkdtempSync, readFileSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { basename, dirname, join, resolve } from 'node:path';
import test from 'node:test';

import { GENERATED_MARKER, ModuleExporter } from '../../src/modules/lowcode/generator/exporter';
import { GO_MODULE_PATH } from '../../src/modules/lowcode/generator/backendGenerator';
import {
  buildAuditActionKey,
  buildDashboardQuickActionDescriptionKey,
  buildEnumOptionKey,
  buildFieldLabelKey,
  buildFieldPlaceholderKey,
  buildMenuGroupTitleKey,
  buildPageRoutePath,
  buildPermissionTitleKey,
  buildTitleKey,
  generateDefaultMenus,
  generateDefaultPermissions,
  validateGeneratorCompleteness,
  type ModuleField,
  type ModuleSchema,
} from '../../src/modules/lowcode/generator/schema';

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

function createRelationFields(): ModuleField[] {
  return [
    ...createFields(),
    {
      name: 'vendorId',
      type: 'relation',
      label: '供应商',
      labelEn: 'Vendor',
      required: true,
      searchable: false,
      sortable: false,
      visibleInList: true,
      visibleInForm: true,
      placeholder: '请选择供应商',
      placeholderEn: 'Select vendor',
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
        targetLabelField: 'name',
        lookupApi: '/business/cmdb/vendor/options',
        lookupValueField: 'id',
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
      fields: createRelationFields(),
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
      en[buildFieldPlaceholderKey(schema.scope, schema.name, field.name)] =
        field.placeholderEn || field.placeholder || '';
    }
    for (const item of field.enumOptions ?? []) {
      zh[buildEnumOptionKey(schema.scope, schema.name, field.name, item.value)] = item.label;
      en[buildEnumOptionKey(schema.scope, schema.name, field.name, item.value)] =
        item.labelEn || item.label;
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

test('every exported file carries the generated marker on its first line', () => {
  const files = new ModuleExporter(createSchema()).generateAll();

  assert.ok(files.length > 0);
  for (const file of files) {
    assert.equal(
      file.content.split('\n', 1)[0],
      `// ${GENERATED_MARKER}`,
      `${file.path} must start with the generated marker`,
    );
  }
});

test('the generated marker matches the sentence the check:generated gate looks for', () => {
  // scripts/harness/check-generated.mjs hardcodes the same sentence; changing one
  // side alone makes every generated artifact fail that gate.
  assert.equal(GENERATED_MARKER, 'Code generated by pantheon low-code generator. DO NOT EDIT.');
});

test('the generated marker is applied exactly once per file', () => {
  const files = new ModuleExporter(createSchema()).generateAll();
  const markerLine = `// ${GENERATED_MARKER}`;

  for (const file of files) {
    assert.equal(
      file.content.split(markerLine).length - 1,
      1,
      `${file.path} must carry the marker exactly once`,
    );
  }
});

test('generated file paths only use extensions where a line comment is valid', () => {
  const files = new ModuleExporter(createSchema()).generateAll();

  for (const file of files) {
    assert.match(
      file.path,
      /\.(go|ts|tsx)$/,
      `${file.path} must use a comment-capable extension so the marker stays valid`,
    );
  }
});

function assertCompletenessPasses(schema: ModuleSchema) {
  const blockingIssues = validateGeneratorCompleteness(schema).filter(
    (issue) => issue.level === 'error',
  );
  assert.deepEqual(blockingIssues, []);
}

test('exports a complete business module contract for a main table schema', () => {
  const schema = createSchema();
  const files = generatedFiles(schema);

  assertCompletenessPasses(schema);
  assert.deepEqual(
    [...files.keys()],
    [
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
    ],
  );

  const backendModule = files.get('backend/modules/business/cmdb/asset/module.go') || '';
  assert.match(backendModule, /ModuleName:\s+"business\.cmdb\.asset"/);
  assert.match(backendModule, /PagePerm:\s+"business:cmdb:asset:list"/);
  assert.match(backendModule, /Perms:\s+"business:cmdb:asset:create"/);
  assert.match(backendModule, /Key:\s+"business\.cmdb\.asset\.field\.assetCode\.label"/);
  assert.match(backendModule, /Key:\s+"business\.cmdb\.asset\.audit\.create"/);
  assert.match(backendModule, /protected\.GET\("\/options", handler\.GetCmdbAssetOptions\)/);

  const backendService = files.get('backend/modules/business/cmdb/asset/asset_service.go') || '';
  assert.match(backendService, /database\.WithDataScope\(dataScope\)/);
  assert.doesNotMatch(backendService, /backend\/modules\/system\//);
  assert.match(backendService, /ListCmdbAssetOptions/);
  assert.match(backendService, /CmdbAssetOptionItem/);
  assert.match(
    backendService,
    /db = db\.Where\("asset_code LIKE \?", "%"\+query\.AssetCode\+"%"\)/,
  );
  assert.match(backendService, /db = db\.Where\("status LIKE \?", "%"\+query\.Status\+"%"\)/);
  assert.match(backendService, /"assetCode": "asset_code"/);
  assert.doesNotMatch(backendService, /a_et_code/);

  const frontendIndex = files.get('frontend/src/modules/business/cmdb/asset/index.ts') || '';
  assert.match(frontendIndex, /routes:\s*\[/);
  assert.match(frontendIndex, /menus:\s*\[/);
  assert.match(frontendIndex, /permissions:\s*\[/);
  assert.match(frontendIndex, /i18nNamespaces:\s*\['business\.cmdb\.asset'\]/);
  assert.match(frontendIndex, /dashboardWidgets:\s*\[/);
  assert.match(frontendIndex, /componentKey: 'business\/cmdb\/asset\/CmdbAssetList'/);
  assert.match(frontendIndex, /componentKey: 'business\/cmdb\/asset\/CmdbAssetDetail'/);
  assert.equal(buildPageRoutePath('business', 'cmdb/asset'), '/business/cmdb/asset');
  assert.match(frontendIndex, /path: 'business\/cmdb\/asset\/:id'/);
  assert.match(frontendIndex, /activeMenu: '\/business\/cmdb\/asset'/);
  assert.match(frontendIndex, /pagePermission: 'business:cmdb:asset:view'/);

  const frontendList =
    files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetList.tsx') || '';
  assert.match(frontendList, /t\('business\.cmdb\.asset\.field\.assetCode\.label'\)/);
  assert.match(frontendList, /<TableBatchActionBar/);
  assert.doesNotMatch(frontendList, /system-list__table-head/);
  assert.doesNotMatch(frontendList, /generator\.wizard\.result\.relatedModules/);
  assert.doesNotMatch(frontendList, /generator\.wizard\.result\.relatedModuleAction/);
  assert.doesNotMatch(frontendList, />资产管理</);

  const frontendApi = files.get('frontend/src/modules/business/cmdb/asset/api.ts') || '';
  assert.match(frontendApi, /export interface CmdbAssetDetail/);
  assert.match(frontendApi, /export function getCmdbAssetDetail\(id: number\)/);
  assert.match(frontendApi, /VendorIdRelationLookupContract/);
  assert.match(frontendApi, /lookupApi: '\/business\/cmdb\/vendor\/options'/);
  assert.match(frontendApi, /getVendorIdRelationOptions/);
  assert.match(frontendApi, /normalizeRelationOptionRows/);
  assert.match(frontendApi, /apiRequest<unknown>/);
  assert.match(frontendApi, /item\['name'\] \?\? item\.label \?\? item\.name/);

  const frontendForm =
    files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetForm.tsx') || '';
  assert.match(frontendForm, /export interface CmdbAssetFormValues/);
  assert.match(
    frontendForm,
    /FormItem label=\{t\('business\.cmdb\.asset\.field\.assetCode\.label'\)\}/,
  );
  assert.match(frontendForm, /field="vendorId"/);
  assert.match(frontendForm, /loading=\{Boolean\(relationLoading\['vendorId'\]\)\}/);
  assert.match(frontendForm, /generator\.fieldEditor\.type\.relation/);
  assert.match(frontendForm, /business\.cmdb\.asset\.field\.vendorId\.placeholder/);
  assert.doesNotMatch(frontendForm, /generator\.wizard\.result\.relatedModules/);
  assert.doesNotMatch(frontendForm, /generator\.wizard\.result\.scaffoldOnly/);
  assert.match(frontendForm, /relationLookupContractMap/);
  assert.match(frontendForm, /\/business\/cmdb\/vendor\/options/);
  assert.match(frontendForm, /targetLabelField/);
  assert.match(frontendForm, /getVendorIdRelationOptions/);
  assert.match(frontendForm, /const \[relationOptions, setRelationOptions\]/);
  assert.match(frontendForm, /options=\{relationOptions\['vendorId'\] \|\| \[\]\}/);
  assert.match(frontendForm, /disabled=\{!relationLookupContractMap\['vendorId'\]\?\.lookupApi\}/);

  const frontendDetail =
    files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetDetail.tsx') || '';
  assert.match(frontendDetail, /getCmdbAssetDetail/);
  assert.match(frontendDetail, /PageContainer/);
  assert.match(frontendDetail, /Descriptions/);
  assert.match(frontendDetail, /generator\.wizard\.result\.contractTitle/);
  assert.match(frontendDetail, /generator\.wizard\.result\.dependencies/);
  assert.doesNotMatch(frontendDetail, /generator\.wizard\.result\.relatedModules/);
  assert.match(frontendDetail, /generator\.wizard\.result\.relatedData/);
  assert.match(frontendDetail, /loadModuleRelationRows/);
  assert.match(frontendDetail, /loadLookupRelationRows/);
  assert.match(frontendDetail, /buildRelationRuntimePath/);
  assert.match(frontendDetail, /generator\.wizard\.result\.openRelatedModule/);
  assert.match(frontendDetail, /generator\.wizard\.result\.relatedDataUnsupported/);
  assert.match(frontendDetail, /apiRequest<\{ items\?: RelationRuntimeRow\[\] \}>/);
  assert.match(frontendDetail, /url: `\/\$\{scope\}\/\$\{moduleName\}\/list`/);
  assert.doesNotMatch(frontendDetail, /generator\.wizard\.result\.primaryTableContext/);
  assert.match(frontendDetail, /cmdb\/vendor/);
  assert.match(frontendDetail, /assetVendor/);
  assert.match(frontendDetail, /const governanceTableRole = "main"/);
});

test('omits routes, menus, permissions, and widgets for relation tables', () => {
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
    listLayout: {
      governance: true,
    },
    model: {
      tableName: 'biz_cmdb_asset_vendor',
      modelName: 'CmdbAssetVendor',
      fields: createFields(),
    },
    includeDashboardWidget: true,
  });
  const relationFiles = generatedFiles(relationSchema);
  const relationIndex =
    relationFiles.get('frontend/src/modules/business/cmdb/asset_vendor/index.ts') || '';
  const relationModule =
    relationFiles.get('backend/modules/business/cmdb/asset_vendor/module.go') || '';
  const relationDetail =
    relationFiles.get(
      'frontend/src/modules/business/cmdb/asset_vendor/CmdbAssetVendorDetail.tsx',
    ) || '';
  const relationList =
    relationFiles.get('frontend/src/modules/business/cmdb/asset_vendor/CmdbAssetVendorList.tsx') ||
    '';

  assertCompletenessPasses(relationSchema);
  assert.match(relationIndex, /routes:\s*\[\]/);
  assert.match(relationIndex, /menus:\s*\[\]/);
  assert.match(relationIndex, /permissions:\s*\[\]/);
  assert.doesNotMatch(relationIndex, /dashboardWidgets:/);
  assert.doesNotMatch(relationModule, /PagePerm:/);
  assert.doesNotMatch(relationModule, /Perms:/);
  assert.ok(
    relationFiles.has('frontend/src/modules/business/cmdb/asset_vendor/CmdbAssetVendorForm.tsx'),
  );
  assert.ok(
    relationFiles.has('frontend/src/modules/business/cmdb/asset_vendor/CmdbAssetVendorDetail.tsx'),
  );
  assert.match(relationDetail, /const governanceTableRole = "relation"/);
  assert.match(relationDetail, /biz_cmdb_asset/);
  assert.match(relationDetail, /asset_id/);
  assert.match(relationDetail, /vendor_id/);
  assert.match(relationDetail, /generator\.wizard\.result\.relatedData/);
  assert.match(relationDetail, /generator\.wizard\.result\.openRelatedModule/);
  assert.match(relationList, /generator\.wizard\.relationFields/);
  assert.match(relationList, /generator\.wizard\.relationFromField/);
  assert.match(relationList, /generator\.wizard\.relationToField/);
  assert.doesNotMatch(relationList, /generator\.wizard\.result\.relatedModules/);
  assert.doesNotMatch(relationList, /generator\.wizard\.result\.relatedModuleAction/);
  assert.match(relationList, /asset_id/);
  assert.match(relationList, /vendor_id/);
});

test('serializes and rejects unsafe relation governance field metadata', () => {
  const unsafeField = "asset_id';\nalert(1)//";
  const schema = createSchema({
    name: 'cmdb/unsafe_relation',
    displayName: '不安全关系',
    displayNameEn: 'Unsafe Relation',
    pageActions: [],
    metadata: {
      tableRole: 'relation',
      primaryTable: 'biz_cmdb_asset',
      relationFromField: unsafeField,
      relationToField: 'vendor_id',
    },
    listLayout: {
      governance: true,
    },
    model: {
      tableName: 'biz_cmdb_unsafe_relation',
      modelName: 'CmdbUnsafeRelation',
      fields: createFields(),
    },
  });

  const issues = validateGeneratorCompleteness(schema);
  assert.ok(issues.some((issue) => issue.code === 'relation_from_field_invalid'));

  const files = generatedFiles(schema);
  const relationList =
    files.get('frontend/src/modules/business/cmdb/unsafe_relation/CmdbUnsafeRelationList.tsx') ||
    '';
  assert.ok(
    relationList.includes(`const governanceRelationFromField = ${JSON.stringify(unsafeField)};`),
  );
  assert.doesNotMatch(relationList, /const governanceRelationFromField = 'asset_id'/);

  for (const invalidField of [' asset_id ', '\t']) {
    schema.metadata!.relationFromField = invalidField;
    assert.ok(
      validateGeneratorCompleteness(schema).some(
        (issue) => issue.code === 'relation_from_field_invalid',
      ),
    );
  }
});

test('emits many-to-many and master-detail runtime contracts', () => {
  const manyToManySchema = createSchema({
    name: 'cmdb/asset_tag',
    displayName: '资产标签',
    displayNameEn: 'Asset Tag',
    pageActions: ['view', 'create', 'update', 'delete'],
    dependencies: [],
    relations: [
      {
        name: 'assetTags',
        type: 'manyToMany',
        targetModule: 'cmdb/tag',
        localField: 'id',
        targetField: 'id',
        targetLabelField: 'name',
        lookupApi: '/business/cmdb/tag/options',
        lookupValueField: 'id',
        junctionTable: 'biz_cmdb_asset_tag_rel',
      },
    ],
    metadata: {
      businessContext: 'cmdb',
      businessContextTitle: 'CMDB',
      businessContextTitleEn: 'CMDB',
      tableRole: 'main',
      boundedContext: 'asset-tag',
      owner: 'platform',
      summary: '资产标签绑定',
    },
    model: {
      tableName: 'biz_cmdb_asset',
      modelName: 'CmdbAssetTag',
      fields: createFields(),
    },
    includeDashboardWidget: true,
  });
  const manyToManyFiles = generatedFiles(manyToManySchema);
  const manyToManyApi =
    manyToManyFiles.get('frontend/src/modules/business/cmdb/asset_tag/api.ts') || '';
  const manyToManyDetail =
    manyToManyFiles.get('frontend/src/modules/business/cmdb/asset_tag/CmdbAssetTagDetail.tsx') ||
    '';
  const manyToManyService =
    manyToManyFiles.get('backend/modules/business/cmdb/asset_tag/asset_tag_service.go') || '';
  const manyToManyHandler =
    manyToManyFiles.get('backend/modules/business/cmdb/asset_tag/asset_tag_handler.go') || '';
  const manyToManyModule =
    manyToManyFiles.get('backend/modules/business/cmdb/asset_tag/module.go') || '';

  assertCompletenessPasses(manyToManySchema);
  assert.match(manyToManyApi, /bindAssetTagsRelation/);
  assert.match(manyToManyApi, /unbindAssetTagsRelation/);
  assert.match(manyToManyApi, /\/relations\/assetTags/);
  assert.match(manyToManyApi, /targetIds: number\[\] \| string\[\]/);
  assert.match(manyToManyDetail, /relation\.type === 'manyToMany'/);
  assert.match(manyToManyDetail, /generator\.wizard\.result\.relatedDataBind/);
  assert.match(manyToManyDetail, /generator\.wizard\.result\.relatedDataUnbind/);
  assert.match(manyToManyDetail, /openRelationBindModal/);
  assert.match(manyToManyDetail, /submitManyToManyBinding/);
  assert.match(manyToManyDetail, /unbindManyToManyRelation/);
  assert.match(manyToManyDetail, /multiple/);
  assert.equal((manyToManyDetail.match(/const \{ id \} = useParams/g) || []).length, 1);
  assert.equal((manyToManyDetail.match(/const scope = 'business';/g) || []).length, 1);
  assert.equal((manyToManyDetail.match(/const navigate = useNavigate\(\);/g) || []).length, 1);
  assert.match(manyToManyService, /BindAssetTagsRelation/);
  assert.match(manyToManyService, /UnbindAssetTagsRelation/);
  assert.match(manyToManyService, /biz_cmdb_asset_tag_rel/);
  assert.match(manyToManyService, /asset_id/);
  assert.match(manyToManyService, /tag_id/);
  assert.match(manyToManyService, /Create\(map\[string\]interface\{\}/);
  assert.match(manyToManyService, /Delete\(nil\)/);
  assert.match(manyToManyHandler, /BindAssetTagsRelation/);
  assert.match(manyToManyHandler, /UnbindAssetTagsRelation/);
  assert.match(manyToManyHandler, /ShouldBindJSON/);
  assert.match(manyToManyModule, /protected\.POST\("\/:id\/relations\/assetTags"/);
  assert.match(manyToManyModule, /protected\.DELETE\("\/:id\/relations\/assetTags\/:targetId"/);

  const masterDetailSchema = createSchema({
    name: 'cmdb/order',
    displayName: '工单主表',
    displayNameEn: 'Order Master',
    relations: [
      {
        name: 'orderItems',
        type: 'oneToMany',
        targetModule: 'cmdb/order_item',
        localField: 'id',
        targetField: 'orderId',
        targetLabelField: 'itemName',
      },
    ],
    metadata: {
      businessContext: 'cmdb',
      businessContextTitle: 'CMDB',
      businessContextTitleEn: 'CMDB',
      tableRole: 'main',
      boundedContext: 'order',
      owner: 'platform',
      summary: '工单主表',
    },
    model: {
      tableName: 'biz_cmdb_order',
      modelName: 'CmdbOrder',
      fields: createFields(),
    },
  });
  const masterDetailFiles = generatedFiles(masterDetailSchema);
  const masterDetailPage =
    masterDetailFiles.get('frontend/src/modules/business/cmdb/order/CmdbOrderDetail.tsx') || '';

  assert.match(masterDetailPage, /getGeneratedModuleSchema/);
  assert.match(masterDetailPage, /childSchemas/);
  assert.match(masterDetailPage, /editingChildRelation/);
  assert.match(masterDetailPage, /openChildCreateModal/);
  assert.match(masterDetailPage, /openChildEditModal/);
  assert.match(masterDetailPage, /submitChildRelationForm/);
  assert.match(masterDetailPage, /buildModuleCrudBasePath/);
  assert.match(masterDetailPage, /resolveEditableChildRelation/);
  assert.match(masterDetailPage, /renderChildFieldInput/);
  assert.match(masterDetailPage, /relation\.targetField]: detail\[relation\.localField/);
  assert.match(
    masterDetailPage,
    /url: `\$\{buildModuleCrudBasePath\(scope, relation\.targetModule\)\}\/\$\{recordId\}`/,
  );
  assert.match(masterDetailPage, /generator\.wizard\.result\.childTableActions/);
  assert.match(masterDetailPage, /generator\.wizard\.result\.childTableCreate/);
  assert.match(masterDetailPage, /generator\.wizard\.result\.childTableEdit/);
  assert.match(masterDetailPage, /generator\.wizard\.result\.childTableDialogCreate/);
  assert.match(masterDetailPage, /generator\.wizard\.result\.childTableDialogEdit/);
  assert.match(masterDetailPage, /generator\.wizard\.result\.childTableSchemaLoadFailed/);
  assert.equal((masterDetailPage.match(/const \{ id \} = useParams/g) || []).length, 1);
  assert.equal((masterDetailPage.match(/const scope = 'business';/g) || []).length, 1);
  assert.equal((masterDetailPage.match(/const navigate = useNavigate\(\);/g) || []).length, 1);
});

// AUD-021 lock #14 (artifact side): the generator must emit SearchToolbar and never
// FilterPanel. The source side of the same rule is lock #10 in
// tests/scripts/check-search-toolbar-contract.test.mjs; dropping either one leaves a
// gap — source-only misses a generator wiring FilterPanel back in, artifact-only
// lets a dirty template go unnoticed. Negative-verified by pushing FilterPanel into
// frontendGenerator.ts's imports array: not ok 14.
test('the generated List page uses SearchToolbar and never emits FilterPanel', () => {
  const files = generatedFiles(createSchema());
  const listPage = files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetList.tsx') || '';

  assert.ok(listPage.length > 0, 'expected a generated List page');
  assert.match(listPage, /SearchToolbar/);
  assert.doesNotMatch(listPage, /FilterPanel/);
});

// ── AUD-001：生成物必须可编译、可建表，导入前缀单一来源 ──────────────────

function readGoModuleLine(): string {
  // 该测试被 transpile 到临时目录执行，相对路径不可用；npm run 的 cwd 是 frontend/，
  // 从仓库根起跑时也能命中，两个候选覆盖两种启动目录。
  const candidates = [
    join(process.cwd(), '..', 'backend', 'go.mod'),
    join(process.cwd(), 'backend', 'go.mod'),
  ];
  for (const candidate of candidates) {
    try {
      const match = readFileSync(candidate, 'utf8').match(/^module\s+(\S+)\s*$/m);
      if (match) {
        return match[1];
      }
    } catch {
      // try next candidate
    }
  }
  throw new Error('backend/go.mod not found from cwd: ' + candidates.join(', '));
}

test('GO_MODULE_PATH stays in sync with backend/go.mod', () => {
  // 棘轮：模板曾手写 pantheon-base/... 而注册器写真实模块路径，两处不同源且长期无人发现。
  // 改模块名但忘改常量时，这条先红。
  assert.equal(GO_MODULE_PATH, readGoModuleLine());
});

test('generated Go imports are all derived from GO_MODULE_PATH', () => {
  const files = generatedFiles(createSchema());
  for (const [path, content] of files) {
    if (!path.startsWith('backend/') || !path.endsWith('.go')) {
      continue;
    }
    assert.doesNotMatch(
      content,
      /"pantheon-base\//,
      `${path} must not carry a hand-written pantheon-base/ import`,
    );
  }
  const moduleSrc = files.get('backend/modules/business/cmdb/asset/module.go') || '';
  const serviceSrc = files.get('backend/modules/business/cmdb/asset/asset_service.go') || '';
  const handlerSrc = files.get('backend/modules/business/cmdb/asset/asset_handler.go') || '';
  assert.ok(
    moduleSrc.includes('"' + GO_MODULE_PATH + '/pkg/contracts"'),
    'module.go must import contracts via GO_MODULE_PATH',
  );
  assert.ok(
    serviceSrc.includes('"' + GO_MODULE_PATH + '/pkg/common"'),
    'service.go must import common via GO_MODULE_PATH',
  );
  assert.ok(
    serviceSrc.includes('"' + GO_MODULE_PATH + '/pkg/database"'),
    'service.go must import database via GO_MODULE_PATH',
  );
  assert.ok(
    handlerSrc.includes('"' + GO_MODULE_PATH + '/pkg/common"'),
    'handler.go must import common via GO_MODULE_PATH',
  );
});

test('generated module wires BootstrapFunc for the default versioned-migration mode', () => {
  const files = generatedFiles(createSchema());
  const moduleSrc = files.get('backend/modules/business/cmdb/asset/module.go') || '';
  // RegisterBackendModules 默认模式只跑 Bootstrap；缺 BootstrapFunc 时生成的表永远不会建出来。
  // 列间距用 \s+ 匹配：AUD-001b 之后 `MigrateFunc:` 按 gofmt 规则补到
  // `BootstrapFunc:` 的列宽，这里断言的是接线本身，不是对齐空格。
  assert.match(
    moduleSrc,
    /BootstrapFunc:\s+func\(_ \*gorm\.DB\) error { return service\.Migrate\(\) },/,
  );
  // dev 的 PANTHEON_AUTO_MIGRATE=true 路径仍走 MigrateFunc。
  assert.match(
    moduleSrc,
    /MigrateFunc:\s+func\(_ \*gorm\.DB\) error { return service\.Migrate\(\) },/,
  );
});

test('generated routes use contracts route-group helpers instead of assembling middleware chains', () => {
  const scopedModule =
    generatedFiles(createSchema()).get('backend/modules/business/cmdb/asset/module.go') || '';
  const plainModule =
    generatedFiles(createSchema({ enableDataScope: false, templateLevel: 'basic' })).get(
      'backend/modules/business/cmdb/asset/module.go',
    ) || '';

  for (const [label, moduleSrc] of [
    ['data-scope', scopedModule],
    ['plain', plainModule],
  ] as const) {
    assert.doesNotMatch(
      moduleSrc,
      /middleware\./,
      `${label} module must not call middleware directly`,
    );
    assert.doesNotMatch(
      moduleSrc,
      /\.Group\([^)]*\)\.Use\(/,
      `${label} module must not hand-assemble the chain`,
    );
  }
  assert.match(
    scopedModule,
    /protected := contracts\.DataScopedGroup\(r, "\/business\/cmdb\/asset", db\)/,
  );
  assert.match(
    plainModule,
    /protected := contracts\.ProtectedGroup\(r, "\/business\/cmdb\/asset"\)/,
  );
});

test('generated routes stay outside tenant isolation', () => {
  // W2 / TENANT_CONTRACT_V1 §2.6：业务域不参与租户隔离，生成物不得挂 TenantContextMiddleware，
  // 否则 multi 模式下所有业务请求都会变成 TENANT_CONTEXT_MISSING。
  const plainModule =
    generatedFiles(createSchema({ enableDataScope: false, templateLevel: 'basic' })).get(
      'backend/modules/business/cmdb/asset/module.go',
    ) || '';
  assert.doesNotMatch(plainModule, /TenantContextMiddleware/);
  assert.match(
    plainModule,
    /protected := contracts\.ProtectedGroup\(r, "\/business\/cmdb\/asset"\)/,
  );
});

test('generated list connects real create/update forms and preserves recoverable submission state', () => {
  const schema = createSchema();
  const files = generatedFiles(schema);
  const list = files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetList.tsx') || '';
  const form = files.get('frontend/src/modules/business/cmdb/asset/CmdbAssetForm.tsx') || '';
  assert.doesNotMatch(list, /comingSoon/);
  assert.match(list, /await createCmdbAsset\(\{/);
  assert.match(list, /await getCmdbAssetDetail\(id\)/);
  assert.match(list, /await updateCmdbAsset\(editor.id, \{/);
  assert.match(list, /values.assetCode == null/);
  assert.match(list, /values.vendorId == null/);
  assert.match(list, /hasPerm\('business:cmdb:asset:create'\)/);
  assert.match(list, /hasPerm\('business:cmdb:asset:update'\)/);
  assert.match(list, /editorBusy.current = true/);
  assert.match(list, /setEditorFailed\(true\)/);
  assert.match(list, /submitError=\{editorFailed\}/);
  assert.match(form, /disabled=\{submitting\}/);
  assert.match(form, /errorSummary=\{submitError/);
});

test('list editor respects action/layout switches and row-only delete imports', () => {
  for (const [pageActions, listLayout, expected] of [
    [['create'], {}, 'create'],
    [['update'], {}, 'update'],
    [['view', 'create', 'update'], { headerActions: false, rowActions: false }, 'none'],
    [['delete'], { batchActions: false }, 'delete'],
  ] as const) {
    const schema = createSchema({
      pageActions: [...pageActions],
      listLayout,
      relations: [],
      dependencies: [],
    });
    const list =
      generatedFiles(schema).get('frontend/src/modules/business/cmdb/asset/CmdbAssetList.tsx') ||
      '';
    if (expected === 'none' || expected === 'delete') {
      assert.doesNotMatch(list, /CmdbAssetForm|usePermission|editorBusy/);
    } else {
      assert.match(list, /CmdbAssetForm/);
      assert.equal(list.includes('openCreateEditor'), expected === 'create');
      assert.equal(list.includes('openUpdateEditor'), expected === 'update');
    }
    if (expected === 'delete') {
      assert.match(list, /deleteCmdbAsset/);
    }
  }
});

test('generated required boolean create DTO preserves presence and both boolean values', (t) => {
  const schema = createSchema({ templateLevel: 'basic', enableDataScope: false, relations: [], dependencies: [] });
  schema.model.fields = [{
    name: 'enabled', type: 'bool', label: '启用', labelEn: 'Enabled',
    required: true, visibleInList: true, visibleInForm: true,
  }];
  const files = generatedFiles(schema);
  const backendRoot = [resolve(process.cwd(), '../backend'), resolve(process.cwd(), 'backend')]
    .find((root) => existsSync(join(root, 'go.mod')));
  assert.ok(backendRoot, 'backend/go.mod is required for generated Go behavior validation');
  const tempDir = mkdtempSync(join(backendRoot, '.generator-required-bool-'));
  t.after(() => {
    assert.equal(dirname(realpathSync(tempDir)), realpathSync(backendRoot));
    rmSync(tempDir, { recursive: true, force: true });
  });
  for (const [file, content] of files) {
    if (file.startsWith('backend/') && /_(model|dto|service)\.go$/.test(file)) {
      writeFileSync(join(tempDir, basename(file)), content);
    }
  }
  writeFileSync(join(tempDir, 'required_bool_test.go'), String.raw`package asset

import (
    "testing"
    "github.com/gin-gonic/gin/binding"
)

func TestRequiredBooleanCreate(t *testing.T) {
    for _, tc := range []struct {
        name, body string
        valid, want bool
    }{
        {"missing", "{}", false, false},
        {"null", "{\"enabled\":null}", false, false},
        {"false", "{\"enabled\":false}", true, false},
        {"true", "{\"enabled\":true}", true, true},
    } {
        t.Run(tc.name, func(t *testing.T) {
            var req CmdbAssetCreateReq
            err := binding.JSON.BindBody([]byte(tc.body), &req)
            if !tc.valid {
                if err == nil { t.Fatal("missing/null required boolean was accepted") }
                return
            }
            if err != nil { t.Fatalf("explicit boolean rejected: %v", err) }
            item := (&CmdbAssetService{}).fromCreateReq(&req)
            if item.Enabled != tc.want { t.Fatalf("created boolean=%t, want %t", item.Enabled, tc.want) }
        })
    }
}
`);
  const result = spawnSync('go', ['test', '-mod=readonly', '.'], {
    cwd: tempDir, encoding: 'utf8', timeout: 120_000,
  });
  assert.equal(result.status, 0, `${result.error?.message ?? ''}\n${result.stdout}\n${result.stderr}`);
});
