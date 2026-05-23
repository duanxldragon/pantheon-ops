import {
  buildVerifiedHeaders,
  executeMysql,
  getOperationToken,
  loginWithOptionalMfa,
} from './smoke-auth.mjs';

const apiOrigin = process.env.PANTHEON_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? `${apiOrigin}/api/v1`;
const adminUsername = process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin';
const adminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456';

const moduleName = 'autorecycleqa';
const moduleKey = `business.${moduleName}`;
const tableName = 'biz_auto_recycle_qa';

function buildTitleKey(scope, name) {
  return `${scope}.${name}.title`;
}

function buildFieldLabelKey(scope, name, fieldName) {
  return `${scope}.${name}.field.${fieldName}.label`;
}

function buildFieldPlaceholderKey(scope, name, fieldName) {
  return `${scope}.${name}.field.${fieldName}.placeholder`;
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

function buildSchema() {
  return finalizeSchema({
    name: moduleName,
    templateVersion: 'v1',
    displayName: '自动回收 QA 模块',
    displayNameEn: 'Auto Recycle QA Module',
    description: 'QA module for lifecycle purge verification',
    scope: 'business',
    parentMenu: '',
    templateLevel: 'enterprise',
    pageActionTemplate: 'masterData',
    pageActions: ['view', 'create', 'update', 'delete', 'detail'],
    metadata: {
      businessContext: 'qa',
      businessContextTitle: 'QA',
      businessContextTitleEn: 'QA',
      tableRole: 'main',
      autoRecycle: true,
      boundedContext: 'qa',
      owner: 'codex',
      summary: '验证临时 QA 表自动回收',
      sourceMode: 'manual',
      sourceTable: '',
    },
    model: {
      tableName,
      modelName: 'Autorecycleqa',
      fields: [
        {
          name: 'name',
          type: 'string',
          label: '名称',
          labelEn: 'Name',
          required: true,
          searchable: true,
          sortable: true,
          visibleInList: true,
          visibleInForm: true,
          placeholder: '请输入名称',
          placeholderEn: 'Enter name',
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
    dependencies: [],
    relations: [],
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

async function purgeModule(accessToken, csrfToken, operationToken) {
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

async function ensureManagedTable() {
  await executeMysql(`
CREATE TABLE IF NOT EXISTS ${tableName} (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  KEY idx_${tableName}_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`);
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
    await purgeModule(accessToken, csrfToken, operationToken);
    return;
  }

  await purgeModule(accessToken, csrfToken, operationToken);
  await generateModule(buildSchema(), accessToken, csrfToken, operationToken);
  await ensureManagedTable();
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
