import assert from 'node:assert/strict';
import test from 'node:test';

import { ModuleExporter } from '../../src/modules/lowcode/generator/exporter';
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
