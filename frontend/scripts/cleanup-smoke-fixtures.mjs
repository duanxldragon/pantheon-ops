import process from 'node:process';
import {
  buildAuthHeaders,
  buildVerifiedHeaders,
  getOperationToken,
  loginWithOptionalMfa,
} from './smoke-auth.mjs';

const apiOrigin = process.env.PANTHEON_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? `${apiOrigin}/api/v1`;
const adminUsername = process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin';
const adminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456';

const entityPatterns = {
  roles: {
    prefixes: [
      'setting_view_only_',
      'dict_view_only_',
      'i18n_view_only_',
      'login_log_view_only_',
      'session_view_only_',
      'operation_log_view_only_',
      'module_view_only_',
      'setting_view_only_route_',
      'user_view_only_route_',
      'qa_role_auth_',
      'smoke_batch_delete_role_',
    ],
    exacts: ['smoke_impexp_role'],
  },
  users: {
    prefixes: [
      'setting_viewer_',
      'dict_viewer_',
      'i18n_viewer_',
      'login_log_viewer_',
      'session_viewer_',
      'operation_log_viewer_',
      'module_viewer_',
      'setting_viewer_route_',
      'user_viewer_route_',
      'smoke_user_',
      'smoke_user_batch_',
      'smoke_post_user_',
      'smoke_batch_delete_user_',
    ],
    exacts: ['smoke_impexp_user'],
  },
  depts: {
    prefixes: ['烟测用户部门-', '烟测部门-', '烟测岗位部门-', '视觉巡检部门-', 'smoke_batch_delete_dept_'],
    exacts: ['烟测研发中心'],
  },
  posts: {
    prefixes: ['SMOKE_DEPT_POST_', 'SMOKE_POST_', 'VISUAL_POST_', 'smoke_batch_delete_post_'],
    exacts: ['smoke_developer'],
  },
  dictCodes: {
    prefixes: ['enter_dict_', 'system_sync_', 'smoke_batch_delete_dict_'],
    exacts: ['smoke_biz_status'],
  },
  i18nKeys: {
    prefixes: ['i18n.enter.', 'i18n.smoke.', 'i18n.import.'],
    exacts: [],
  },
  permissionPaths: {
    prefixes: ['/api/v1/system/permission/enter-smoke-', '/api/v1/system/smoke-batch-delete/'],
    exacts: [],
  },
};

function readArg(flag, fallback = 'all') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return process.argv[index + 1];
}

function matchesPattern(value, pattern) {
  return pattern.exacts.includes(value) || pattern.prefixes.some((prefix) => value.startsWith(prefix));
}

function authHeaders(session) {
  return buildAuthHeaders(session);
}

function verifiedHeaders(session, operationToken) {
  return buildVerifiedHeaders(session, operationToken);
}

async function apiGet(session, path, params = {}) {
  const url = new URL(`${apiBaseUrl}${path}`);
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, String(value));
  }
  const response = await fetch(url, {
    headers: authHeaders(session),
  });
  if (!response.ok) {
    throw new Error(`GET ${path} failed: HTTP ${response.status}`);
  }
  const payload = await response.json();
  if (payload.code !== 200) {
    throw new Error(`GET ${path} failed: code ${payload.code}`);
  }
  return payload.data;
}

async function apiDelete(session, operationToken, path) {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: 'DELETE',
    headers: verifiedHeaders(session, operationToken),
  });
  if ([200, 404, 500].includes(response.status)) {
    return;
  }
  if (!response.ok) {
    throw new Error(`DELETE ${path} failed: HTTP ${response.status}`);
  }
  const payload = await response.json().catch(() => null);
  if (payload && payload.code !== 200) {
    throw new Error(`DELETE ${path} failed: code ${payload.code}`);
  }
}

function flattenDepts(nodes) {
  return nodes.flatMap((node) => [node, ...flattenDepts(Array.isArray(node.children) ? node.children : [])]);
}

async function cleanupUsers(session, operationToken) {
  const data = await apiGet(session, '/system/user/list', { page: 1, pageSize: 200 });
  const items = Array.isArray(data.items) ? data.items : [];
  let deleted = 0;
  for (const item of items) {
    if (matchesPattern(String(item.username || ''), entityPatterns.users)) {
      await apiDelete(session, operationToken, `/system/user/${item.id}`);
      deleted += 1;
    }
  }
  return deleted;
}

async function cleanupRoles(session, operationToken) {
  const data = await apiGet(session, '/system/role/list', { page: 1, pageSize: 200 });
  const items = Array.isArray(data.items) ? data.items : [];
  let deleted = 0;
  for (const item of items) {
    const roleKey = String(item.roleKey || '');
    if (roleKey !== 'admin' && matchesPattern(roleKey, entityPatterns.roles)) {
      await apiDelete(session, operationToken, `/system/role/${item.id}`);
      deleted += 1;
    }
  }
  return deleted;
}

async function cleanupPosts(session, operationToken) {
  const data = await apiGet(session, '/system/post/list', { page: 1, pageSize: 500 });
  const items = Array.isArray(data.items) ? data.items : [];
  let deleted = 0;
  for (const item of items) {
    if (matchesPattern(String(item.postCode || ''), entityPatterns.posts)) {
      await apiDelete(session, operationToken, `/system/post/${item.id}`);
      deleted += 1;
    }
  }
  return deleted;
}

async function cleanupDepts(session, operationToken) {
  const tree = await apiGet(session, '/system/dept/tree', { sortField: 'sort', sortOrder: 'asc' });
  const items = flattenDepts(Array.isArray(tree) ? tree : [])
    .filter((item) => !item.isRoot && matchesPattern(String(item.deptName || ''), entityPatterns.depts))
    .reverse();
  let deleted = 0;
  for (const item of items) {
    await apiDelete(session, operationToken, `/system/dept/${item.id}`);
    deleted += 1;
  }
  return deleted;
}

async function cleanupDictTypes(session, operationToken) {
  const data = await apiGet(session, '/system/dict/type/list', {});
  const items = Array.isArray(data) ? data : [];
  let deleted = 0;
  for (const item of items) {
    if (matchesPattern(String(item.dictCode || ''), entityPatterns.dictCodes)) {
      await apiDelete(session, operationToken, `/system/dict/type/${item.id}`);
      deleted += 1;
    }
  }
  return deleted;
}

async function cleanupDictItems(session, operationToken) {
  const typeData = await apiGet(session, '/system/dict/type/list', {});
  const types = Array.isArray(typeData) ? typeData : [];
  let deleted = 0;
  for (const type of types) {
    const dictCode = String(type.dictCode || '');
    if (!matchesPattern(dictCode, entityPatterns.dictCodes)) {
      continue;
    }
    const itemData = await apiGet(session, '/system/dict/item/list', { dictCode, page: 1, pageSize: 500 });
    const items = Array.isArray(itemData.items) ? itemData.items : [];
    for (const item of items) {
      await apiDelete(session, operationToken, `/system/dict/item/${item.id}`);
      deleted += 1;
    }
  }
  return deleted;
}

async function cleanupI18n(session, operationToken) {
  let deleted = 0;
  for (const prefix of entityPatterns.i18nKeys.prefixes) {
    const data = await apiGet(session, '/system/i18n/list', { key: prefix, page: 1, pageSize: 500 });
    const items = Array.isArray(data.items) ? data.items : [];
    for (const item of items) {
      if (matchesPattern(String(item.key || ''), entityPatterns.i18nKeys)) {
        await apiDelete(session, operationToken, `/system/i18n/${item.id}`);
        deleted += 1;
      }
    }
  }
  return deleted;
}

async function cleanupPermissions(session, operationToken) {
  const roleData = await apiGet(session, '/system/role/list', { page: 1, pageSize: 200 });
  const roles = Array.isArray(roleData.items) ? roleData.items : [];
  let deleted = 0;

  for (const role of roles) {
    const roleKey = String(role.roleKey || '');
    const shouldScanRole = matchesPattern(roleKey, entityPatterns.roles) || roleKey === 'admin';
    if (!shouldScanRole) {
      continue;
    }
    const permissionData = await apiGet(session, '/system/permission/list', {
      roleKey,
      page: 1,
      pageSize: 500,
    });
    const items = Array.isArray(permissionData.items) ? permissionData.items : [];
    for (const item of items) {
      const path = String(item.path || '');
      if (
        matchesPattern(roleKey, entityPatterns.roles) ||
        matchesPattern(path, entityPatterns.permissionPaths)
      ) {
        await apiDelete(session, operationToken, `/system/permission/${item.id}`);
        deleted += 1;
      }
    }
  }

  return deleted;
}

async function main() {
  const scope = readArg('--scope', 'all');
  const session = await loginWithOptionalMfa(apiBaseUrl, {
    username: adminUsername,
    password: adminPassword,
  });
  const operationToken = await getOperationToken(apiBaseUrl, session, adminPassword);
  const summary = {
    users: 0,
    roles: 0,
    posts: 0,
    depts: 0,
    dictItems: 0,
    dictTypes: 0,
    i18n: 0,
    permissions: 0,
  };

  if (scope === 'all' || scope === 'iam') {
    summary.users = await cleanupUsers(session, operationToken);
    summary.permissions = await cleanupPermissions(session, operationToken);
    summary.roles = await cleanupRoles(session, operationToken);
  }
  if (scope === 'all' || scope === 'org') {
    summary.posts = await cleanupPosts(session, operationToken);
    summary.depts = await cleanupDepts(session, operationToken);
  }
  if (scope === 'all' || scope === 'config') {
    summary.dictItems = await cleanupDictItems(session, operationToken);
    summary.dictTypes = await cleanupDictTypes(session, operationToken);
    summary.i18n = await cleanupI18n(session, operationToken);
  }

  console.info('[smoke-fixtures] cleanup complete');
  console.info(JSON.stringify({ scope, ...summary }, null, 2));
}

await main();
