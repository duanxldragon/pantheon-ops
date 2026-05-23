import {
  buildVerifiedHeaders,
  getOperationToken,
  loginWithOptionalMfa,
} from './smoke-auth.mjs';

const apiOrigin = process.env.PANTHEON_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? `${apiOrigin}/api/v1`;
const adminUsername = process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin';
const adminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456';

const ownerModuleName = 'm2mqaasset';
const targetModuleName = 'm2mqatag';
const ownerModuleKey = `business.${ownerModuleName}`;
const targetModuleKey = `business.${targetModuleName}`;
const ownerTableName = 'biz_m2mqa_asset';
const targetTableName = 'biz_m2mqa_tag';

function buildTitleKey(scope, name) {
  return `${scope}.${name}.title`;
}

function buildFieldLabelKey(scope, name, fieldName) {
  return `${scope}.${name}.field.${fieldName}.label`;
}

function buildFieldPlaceholderKey(scope, name, fieldName) {
  return `${scope}.${name}.field.${fieldName}.placeholder`;
}

function buildEnumOptionKey(scope, name, fieldName, value) {
  return `${scope}.${name}.field.${fieldName}.option.${value}`;
}

function buildPermissionTitleKey(scope, name, action) {
  return `${scope}.${name}.permission.${action}`;
}

function buildModuleNamespace(scope, name) {
  return `${scope}.${name}`;
}

function buildI18n(schema) {
  const titleKey = buildTitleKey(schema.scope, schema.name);
  const zh = { [titleKey]: schema.displayName };
  const en = { [titleKey]: schema.displayNameEn || schema.displayName };

  for (const field of schema.model.fields) {
    zh[buildFieldLabelKey(schema.scope, schema.name, field.name)] = field.label;
    en[buildFieldLabelKey(schema.scope, schema.name, field.name)] = field.labelEn || field.label;
    if (field.placeholder || field.placeholderEn) {
      zh[buildFieldPlaceholderKey(schema.scope, schema.name, field.name)] = field.placeholder || '';
      en[buildFieldPlaceholderKey(schema.scope, schema.name, field.name)] =
        field.placeholderEn || field.placeholder || '';
    }
    for (const item of field.enumOptions || []) {
      zh[buildEnumOptionKey(schema.scope, schema.name, field.name, item.value)] = item.label;
      en[buildEnumOptionKey(schema.scope, schema.name, field.name, item.value)] = item.labelEn || item.label;
    }
  }

  for (const action of ['view', 'create', 'update', 'delete', 'detail']) {
    zh[buildPermissionTitleKey(schema.scope, schema.name, action)] = `${action}${schema.displayName}`;
    en[buildPermissionTitleKey(schema.scope, schema.name, action)] =
      `${action} ${schema.displayNameEn || schema.displayName}`;
  }

  return {
    namespace: buildModuleNamespace(schema.scope, schema.name),
    translations: { zh, en },
  };
}

function generateDefaultMenus(schema) {
  const path = `/${schema.scope}/${schema.name}`;
  return [
    {
      key: `${schema.scope}.${schema.name}`,
      titleKey: buildTitleKey(schema.scope, schema.name),
      path,
      component: `${schema.scope}/${schema.name}/${schema.model.modelName}List`,
      pagePermission: `${schema.scope}:${schema.name}:list`,
      type: 'C',
      icon: 'apps',
      routeName: `${schema.scope}-${schema.name}`,
      module: buildModuleNamespace(schema.scope, schema.name),
    },
  ];
}

function generateDefaultPermissions(schema) {
  return ['list', 'view', 'create', 'update', 'delete', 'detail'].map((action) => ({
    key: `${schema.scope}:${schema.name}:${action}`,
    name: buildPermissionTitleKey(schema.scope, schema.name, action),
    type: action === 'list' ? 'menu' : 'button',
    module: buildModuleNamespace(schema.scope, schema.name),
  }));
}

function finalizeSchema(schema) {
  schema.menus = generateDefaultMenus(schema);
  schema.permissions = generateDefaultPermissions(schema);
  schema.i18n = buildI18n(schema);
  return schema;
}

function buildOwnerSchema() {
  return finalizeSchema({
    name: ownerModuleName,
    templateVersion: 'v1',
    displayName: '多对多资产',
    displayNameEn: 'Many To Many Asset',
    description: 'QA owner module for many-to-many runtime flow',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    dependencies: [{ module: targetModuleName, required: true, reason: '绑定标签' }],
    relations: [
      {
        name: 'assetTags',
        type: 'manyToMany',
        targetModule: targetModuleName,
        localField: 'id',
        targetField: 'id',
        targetLabelField: 'name',
        lookupApi: `/business/${targetModuleName}/options`,
        lookupValueField: 'id',
        junctionTable: 'biz_m2mqa_asset_tag_rel',
      },
    ],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'main',
      boundedContext: 'qa',
      owner: 'codex',
      summary: '多对多详情绑定链路验证',
      sourceMode: 'manual',
      sourceTable: '',
      autoRecycle: true,
    },
    model: {
      tableName: ownerTableName,
      modelName: 'M2mqaasset',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '资产名称',
          labelEn: 'Asset Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入资产名称',
          placeholderEn: 'Enter asset name',
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
          placeholder: '请选择状态',
          placeholderEn: 'Select status',
          enumOptions: [
            { value: 'draft', label: '草稿', labelEn: 'Draft' },
            { value: 'active', label: '启用', labelEn: 'Active' },
          ],
        },
      ],
    },
    menus: [],
    permissions: [],
    i18n: { namespace: '', translations: { zh: {}, en: {} } },
  });
}

function buildTargetSchema() {
  return finalizeSchema({
    name: targetModuleName,
    templateVersion: 'v1',
    displayName: '多对多标签',
    displayNameEn: 'Many To Many Tag',
    description: 'QA target module for many-to-many runtime flow',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    dependencies: [],
    relations: [],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'dictionary',
      boundedContext: 'qa',
      owner: 'codex',
      summary: '多对多标签字典',
      sourceMode: 'manual',
      sourceTable: '',
      autoRecycle: true,
    },
    model: {
      tableName: targetTableName,
      modelName: 'M2mqatag',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '标签名称',
          labelEn: 'Tag Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入标签名称',
          placeholderEn: 'Enter tag name',
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
          placeholder: '请选择状态',
          placeholderEn: 'Select status',
          enumOptions: [
            { value: 'active', label: '启用', labelEn: 'Active' },
            { value: 'disabled', label: '停用', labelEn: 'Disabled' },
          ],
        },
      ],
    },
    menus: [],
    permissions: [],
    i18n: { namespace: '', translations: { zh: {}, en: {} } },
  });
}

async function request(path, options = {}) {
  const response = await fetch(`${apiBaseUrl}${path}`, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok || payload.code !== 200) {
    throw new Error(`${path} failed: ${response.status} ${payload.message || payload.msg || ''}`);
  }
  return payload.data;
}

async function generateModule(schema, accessToken, csrfToken, operationToken) {
  return request('/system/dynamic-modules/generate', {
    method: 'POST',
    headers: {
      ...buildVerifiedHeaders({ accessToken, csrfToken }, operationToken),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ schema, overwrite: true }),
  });
}

async function purgeModule(moduleKey, accessToken, csrfToken, operationToken) {
  const response = await fetch(
    `${apiBaseUrl}/system/dynamic-modules/${moduleKey}/purge?dropTable=false&purgeSource=true`,
    {
      method: 'DELETE',
      headers: buildVerifiedHeaders({ accessToken, csrfToken }, operationToken),
    },
  );
  if (response.status === 404) {
    return;
  }
  const payload = await response.json().catch(() => ({}));
  if (payload.message === 'module.not_found' || payload.msg === 'module.not_found') {
    return;
  }
  if (!response.ok || payload.code !== 200) {
    throw new Error(`purge ${moduleKey} failed: ${response.status} ${payload.message || payload.msg || ''}`);
  }
}

async function main() {
  const action = process.argv[2] || 'up';
  const loginData = await loginWithOptionalMfa(apiBaseUrl, {
    username: adminUsername,
    password: adminPassword,
  });
  const accessToken = loginData.accessToken;
  const csrfToken = loginData.csrfToken;
  const operationToken = await getOperationToken(apiBaseUrl, loginData, adminPassword);

  if (action === 'down') {
    await purgeModule(ownerModuleKey, accessToken, csrfToken, operationToken);
    await purgeModule(targetModuleKey, accessToken, csrfToken, operationToken);
    return;
  }

  await purgeModule(ownerModuleKey, accessToken, csrfToken, operationToken);
  await purgeModule(targetModuleKey, accessToken, csrfToken, operationToken);
  await generateModule(buildTargetSchema(), accessToken, csrfToken, operationToken);
  await generateModule(buildOwnerSchema(), accessToken, csrfToken, operationToken);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
