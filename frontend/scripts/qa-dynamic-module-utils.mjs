import { buildVerifiedHeaders } from './smoke-auth.mjs';

export function createSmokeConfig() {
  const apiOrigin = process.env.PANTHEON_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
  return {
    apiBaseUrl: process.env.PANTHEON_API_BASE_URL ?? `${apiOrigin}/api/v1`,
    adminUsername: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
    adminPassword: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
  };
}

export function buildModuleNamespace(scope, name) {
  return `${scope}.${name}`;
}

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
      key: buildModuleNamespace(schema.scope, schema.name),
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

export function finalizeSchema(schema) {
  schema.menus = generateDefaultMenus(schema);
  schema.permissions = generateDefaultPermissions(schema);
  schema.i18n = buildI18n(schema);
  return schema;
}

export async function requestJson(apiBaseUrl, path, options = {}) {
  const response = await fetch(`${apiBaseUrl}${path}`, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok || payload.code !== 200) {
    throw new Error(`${path} failed: ${response.status} ${payload.message || payload.msg || ''}`);
  }
  return payload.data;
}

export async function generateModule(apiBaseUrl, schema, accessToken, csrfToken, operationToken) {
  return requestJson(apiBaseUrl, '/system/dynamic-modules/generate', {
    method: 'POST',
    headers: {
      ...buildVerifiedHeaders({ accessToken, csrfToken }, operationToken),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ schema, overwrite: true }),
  });
}

export async function purgeModule(
  apiBaseUrl,
  moduleKey,
  accessToken,
  csrfToken,
  operationToken,
) {
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
