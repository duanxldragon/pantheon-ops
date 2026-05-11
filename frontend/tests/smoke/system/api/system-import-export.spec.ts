import { expect, request as playwrightRequest, test, type APIRequestContext, type APIResponse } from '@playwright/test';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import {
  adminCredentials,
  apiBaseUrl,
  loginByApi,
  verifiedApiHeaders,
  type BrowserLoginResult,
} from '../../helpers/auth';

type ResponseEnvelope<T> = {
  code: number;
  data: T;
  message: string;
};

type ImportError = {
  row: number;
  field: string;
  message: string;
};

type ImportResult = {
  applied: boolean;
  created: number;
  updated: number;
  failed: number;
  errors: ImportError[];
};

type ListPage<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};

type RoleItem = {
  id: number;
  roleName: string;
  roleKey: string;
  status: number;
};

type UserItem = {
  id: number;
  username: string;
  nickname: string;
  roleKeys: string[];
};

type PostItem = {
  id: number;
  deptName: string;
  postCode: string;
  postName: string;
};

type DictTypeItem = {
  id: number;
  dictCode: string;
  dictName: string;
};

type DictItem = {
  id: number;
  dictCode: string;
  itemValue: string;
  itemLabelKey: string;
};

type LoginLogItem = {
  id: number;
  username: string;
  status: number;
};

type OperationLogItem = {
  id: number;
  title: string;
  sourcePage: string;
};

type DeptNode = {
  id: number;
  deptName: string;
  isRoot: boolean;
  children?: DeptNode[];
};

const smokeRoleKey = 'smoke_impexp_role';
const smokeRoleName = '导入导出烟测角色';
const smokeUserName = 'smoke_impexp_user';
const smokeDeptName = '烟测研发中心';
const smokePostCode = 'smoke_developer';
const smokePostName = '烟测研发工程师';
const smokeDictCode = 'smoke_biz_status';
const smokeItemValue = 'enabled';
const smokeItemLabelKey = 'dict.smoke_biz_status.enabled';
const smokePermissionPath = '/api/v1/system/user/list';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const fixtureDir = path.resolve(__dirname, '../../../../../tests/fixtures/system-import-export');

test.describe.serial('system import/export api smoke', () => {
  let apiContext: APIRequestContext;
  let login: BrowserLoginResult;

  test.beforeAll(async () => {
    const loginContext = await playwrightRequest.newContext();
    login = await loginByApi(loginContext, adminCredentials);

    apiContext = await playwrightRequest.newContext({
      extraHTTPHeaders: {
        Authorization: `Bearer ${login.accessToken}`,
        'X-CSRF-Token': login.csrfToken,
        Cookie: `pantheon_csrf_token=${login.csrfToken}`,
      },
    });

    await cleanupSmokeFixtures(apiContext, login);
    await ensureSmokeRole(apiContext);
    await loginContext.dispose();
  });

  test.afterAll(async () => {
    await cleanupSmokeFixtures(apiContext, login);
    await apiContext.dispose();
  });

  test('import templates are downloadable and include comment rows', async () => {
    const templates = [
      {
        path: '/system/user/import-template',
        filename: 'system-user-import-template.csv',
        header: 'username,password,nickname,email,phone,deptPath,postCode,status,roleKeys',
      },
      {
        path: '/system/dept/import-template',
        filename: 'system-dept-import-template.csv',
        header: 'parentDeptPath,deptName,sort,leader,phone,email,status',
      },
      {
        path: '/system/post/import-template',
        filename: 'system-post-import-template.csv',
        header: 'deptPath,postCode,postName,sort,status,remark',
      },
      {
        path: '/system/permission/import-template',
        filename: 'system-permission-import-template.csv',
        header: 'roleKey,path,method',
      },
      {
        path: '/system/dict/type/import-template',
        filename: 'system-dict-type-import-template.csv',
        header: 'dictCode,dictName,module,status,remark',
      },
      {
        path: '/system/dict/item/import-template',
        filename: 'system-dict-item-import-template.csv',
        header: 'dictCode,itemLabelKey,itemValue,itemColor,sort,status,remark',
      },
    ] as const;

    for (const template of templates) {
      const response = await apiContext.get(`${apiBaseUrl}${template.path}`);
      const csv = await expectCsv(response, template.filename);
      const lines = parseCsvLines(csv);

      expect(lines[0]).toBe(template.header);
      expect(lines[1]?.startsWith('#')).toBeTruthy();
      expect(lines.length).toBeGreaterThanOrEqual(3);
    }
  });

  test('dept import and export are usable', async () => {
    const rootDeptName = await getRootDeptName(apiContext);
    const csv = buildCsv(
      ['parentDeptPath', 'deptName', 'sort', 'leader', 'phone', 'email', 'status'],
      '#说明：保留第一行表头；导入器会自动忽略以 # 开头的注释行。',
      [rootDeptName, smokeDeptName, '10', '张三', '13800138000', 'rd-smoke@example.com', '1'],
    );

    const importResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/dept/import`, 'system-dept-import.csv', csv);
    const result = await expectSuccess<ImportResult>(importResponse);
    expect(result.applied).toBeTruthy();
    expect(result.failed).toBe(0);
    expect(result.created + result.updated).toBeGreaterThan(0);

    const treeResponse = await apiContext.get(`${apiBaseUrl}/system/dept/tree`, {
      params: { deptName: smokeDeptName },
    });
    const tree = await expectSuccess<DeptNode[]>(treeResponse);
    expect(findDeptByName(tree, smokeDeptName)).toBeTruthy();

    const exportResponse = await apiContext.post(`${apiBaseUrl}/system/dept/export`, {
      data: { deptName: smokeDeptName },
    });
    const exported = await expectCsv(exportResponse, 'system-dept-export.csv');
    expect(exported).toContain(smokeDeptName);
    expect(exported).toContain(rootDeptName);
  });

  test('post import and export are usable', async () => {
    const rootDeptName = await getRootDeptName(apiContext);
    const deptPath = `${rootDeptName}/${smokeDeptName}`;
    const deptCsv = buildCsv(
      ['parentDeptPath', 'deptName', 'sort', 'leader', 'phone', 'email', 'status'],
      '#说明：保留第一行表头；确保岗位所属部门存在。',
      [rootDeptName, smokeDeptName, '10', '张三', '13800138000', 'rd-smoke@example.com', '1'],
    );
    const deptImportResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/dept/import`, 'system-dept-import.csv', deptCsv);
    const deptImportResult = await expectSuccess<ImportResult>(deptImportResponse);
    expect(deptImportResult.failed).toBe(0);

    const csv = buildCsv(
      ['deptPath', 'postCode', 'postName', 'sort', 'status', 'remark'],
      '#说明：保留第一行表头；postCode 是稳定唯一编码。',
      [deptPath, smokePostCode, smokePostName, '10', '1', '负责烟测验证'],
    );

    const importResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/post/import`, 'system-post-import.csv', csv);
    const result = await expectSuccess<ImportResult>(importResponse);
    expect(result.applied).toBeTruthy();
    expect(result.failed).toBe(0);
    expect(result.created + result.updated).toBeGreaterThan(0);

    const listResponse = await apiContext.get(`${apiBaseUrl}/system/post/list`, {
      params: { postCode: smokePostCode, page: '1', pageSize: '10' },
    });
    const list = await expectSuccess<ListPage<PostItem>>(listResponse);
    expect(list.items.some((item) => item.postCode === smokePostCode && item.deptName === smokeDeptName)).toBeTruthy();

    const exportResponse = await apiContext.post(`${apiBaseUrl}/system/post/export`, {
      data: { postCode: smokePostCode },
    });
    const exported = await expectCsv(exportResponse, 'system-post-export.csv');
    expect(exported).toContain(smokePostCode);
    expect(exported).toContain(smokePostName);
    expect(exported).toContain(deptPath);
  });

  test('permission import and export are usable', async () => {
    const csv = buildCsv(
      ['roleKey', 'path', 'method'],
      '#说明：保留第一行表头；只导入 Casbin 接口策略。',
      [smokeRoleKey, smokePermissionPath, 'GET'],
    );

    const importResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/permission/import`, 'system-permission-import.csv', csv);
    const result = await expectSuccess<ImportResult>(importResponse);
    expect(result.applied).toBeTruthy();
    expect(result.failed).toBe(0);
    expect(result.created + result.updated).toBeGreaterThan(0);

    const listResponse = await apiContext.get(`${apiBaseUrl}/system/permission/list`, {
      params: { roleKey: smokeRoleKey, page: '1', pageSize: '20' },
    });
    const list = await expectSuccess<ListPage<{ roleKey: string; path: string; method: string }>>(listResponse);
    expect(list.items.some((item) => item.roleKey === smokeRoleKey && item.path === smokePermissionPath && item.method === 'GET')).toBeTruthy();

    const exportResponse = await apiContext.post(`${apiBaseUrl}/system/permission/export`, {
      data: { roleKey: smokeRoleKey },
    });
    const exported = await expectCsv(exportResponse, 'system-permission-export.csv');
    expect(exported).toContain(smokeRoleKey);
    expect(exported).toContain(smokePermissionPath);
  });

  test('dict type and item import/export are usable', async () => {
    const typeCsv = buildCsv(
      ['dictCode', 'dictName', 'module', 'status', 'remark'],
      '#说明：保留第一行表头；dictCode 是稳定唯一编码。',
      [smokeDictCode, '烟测业务状态', 'business.smoke', '1', '烟测状态字典'],
    );

    const typeImportResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/dict/type/import`, 'system-dict-type-import.csv', typeCsv);
    const typeResult = await expectSuccess<ImportResult>(typeImportResponse);
    expect(typeResult.applied).toBeTruthy();
    expect(typeResult.failed).toBe(0);
    expect(typeResult.created + typeResult.updated).toBeGreaterThan(0);

    const typeListResponse = await apiContext.get(`${apiBaseUrl}/system/dict/type/list`, {
      params: { dictCode: smokeDictCode },
    });
    const typeList = await expectSuccess<DictTypeItem[]>(typeListResponse);
    expect(typeList.some((item) => item.dictCode === smokeDictCode)).toBeTruthy();

    const typeExportResponse = await apiContext.post(`${apiBaseUrl}/system/dict/type/export`, {
      data: { dictCode: smokeDictCode },
    });
    const typeExported = await expectCsv(typeExportResponse, 'system-dict-type-export.csv');
    expect(typeExported).toContain(smokeDictCode);

    const itemCsv = buildCsv(
      ['dictCode', 'itemLabelKey', 'itemValue', 'itemColor', 'sort', 'status', 'remark'],
      '#说明：保留第一行表头；dictCode 必须已存在。',
      [smokeDictCode, smokeItemLabelKey, smokeItemValue, 'green', '10', '1', '烟测启用态'],
    );

    const itemImportResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/dict/item/import`, 'system-dict-item-import.csv', itemCsv);
    const itemResult = await expectSuccess<ImportResult>(itemImportResponse);
    expect(itemResult.applied).toBeTruthy();
    expect(itemResult.failed).toBe(0);
    expect(itemResult.created + itemResult.updated).toBeGreaterThan(0);

    const itemListResponse = await apiContext.get(`${apiBaseUrl}/system/dict/item/list`, {
      params: { dictCode: smokeDictCode },
    });
    const itemList = await expectSuccess<ListPage<DictItem>>(itemListResponse);
    expect(itemList.items.some((item) => item.dictCode === smokeDictCode && item.itemValue === smokeItemValue)).toBeTruthy();

    const itemExportResponse = await apiContext.post(`${apiBaseUrl}/system/dict/item/export`, {
      data: { dictCode: smokeDictCode },
    });
    const itemExported = await expectCsv(itemExportResponse, 'system-dict-item-export.csv');
    expect(itemExported).toContain(smokeDictCode);
    expect(itemExported).toContain(smokeItemLabelKey);
  });

  test('user import and export are usable', async () => {
    const csv = buildCsv(
      ['username', 'password', 'nickname', 'email', 'phone', 'deptPath', 'postCode', 'status', 'roleKeys'],
      '#说明：保留第一行表头；roleKeys 多角色用 | 分隔；空部门和岗位表示不绑定组织字段。',
      [smokeUserName, 'ChangeMe123', '导入导出烟测用户', 'smoke-user@example.com', '13800138001', '', '', '1', smokeRoleKey],
    );

    const importResponse = await uploadCsv(apiContext, `${apiBaseUrl}/system/user/import`, 'system-user-import.csv', csv);
    const result = await expectSuccess<ImportResult>(importResponse);
    expect(result.applied).toBeTruthy();
    expect(result.failed).toBe(0);
    expect(result.created + result.updated).toBeGreaterThan(0);

    const listResponse = await apiContext.get(`${apiBaseUrl}/system/user/list`, {
      params: { username: smokeUserName, page: '1', pageSize: '10' },
    });
    const list = await expectSuccess<ListPage<UserItem>>(listResponse);
    const user = list.items.find((item) => item.username === smokeUserName);
    expect(user).toBeTruthy();
    expect(user?.roleKeys.includes(smokeRoleKey)).toBeTruthy();

    const exportResponse = await apiContext.post(`${apiBaseUrl}/system/user/export`, {
      data: { username: smokeUserName },
    });
    const exported = await expectCsv(exportResponse, 'system-user-export.csv');
    expect(exported).toContain(smokeUserName);
    expect(exported).toContain(smokeRoleKey);
  });

  test('role export and batch status are usable', async () => {
    const role = await ensureSmokeRole(apiContext);

    const exportResponse = await apiContext.post(`${apiBaseUrl}/system/role/export`, {
      data: { roleKey: smokeRoleKey },
    });
    const exported = await expectCsv(exportResponse, 'system-role-export.csv');
    expect(exported).toContain(smokeRoleKey);

    const disableResponse = await apiContext.post(`${apiBaseUrl}/system/role/batch-status`, {
      data: { roleIds: [role.id], status: 2 },
    });
    const disableResult = await expectSuccess<{ updatedCount: number }>(disableResponse);
    expect(disableResult.updatedCount).toBeGreaterThan(0);

    const disabledRole = await getRoleByKey(apiContext, smokeRoleKey);
    expect(disabledRole?.status).toBe(2);

    const enableResponse = await apiContext.post(`${apiBaseUrl}/system/role/batch-status`, {
      data: { roleIds: [role.id], status: 1 },
    });
    const enableResult = await expectSuccess<{ updatedCount: number }>(enableResponse);
    expect(enableResult.updatedCount).toBeGreaterThan(0);

    const enabledRole = await getRoleByKey(apiContext, smokeRoleKey);
    expect(enabledRole?.status).toBe(1);
  });

  test('login log and operation log export are usable', async () => {
    const loginLogResponse = await apiContext.post(`${apiBaseUrl}/system/login-log/export`, {
      data: { username: adminCredentials.username },
    });
    const loginLogCsv = await expectCsv(loginLogResponse, 'system-login-log-export.csv');
    expect(parseCsvLines(loginLogCsv)[0]).toBe('username,ipaddr,loginLocation,browser,os,status,msg,loginTime');

    const operationLogResponse = await apiContext.post(`${apiBaseUrl}/system/operation-log/export`, {
      data: {},
    });
    const operationLogCsv = await expectCsv(operationLogResponse, 'system-operation-log-export.csv');
    expect(parseCsvLines(operationLogCsv)[0]).toBe(
      'requestId,title,businessType,sourceDomain,sourcePage,method,operName,operUrl,operIp,status,failureCategory,errorMsg,operTime,costTime',
    );
  });

  test('login log and operation log cleanup / batch delete are usable', async () => {
    const operationHeaders = await verifiedApiHeaders(apiContext, login);

    const loginLogListResponse = await apiContext.get(`${apiBaseUrl}/system/login-log/list`, {
      params: { username: adminCredentials.username, page: '1', pageSize: '10' },
    });
    const loginLogList = await expectSuccess<ListPage<LoginLogItem>>(loginLogListResponse);
    expect(loginLogList.items.length).toBeGreaterThan(0);

    const loginCleanupResponse = await apiContext.post(`${apiBaseUrl}/system/login-log/cleanup`, {
      headers: operationHeaders,
      data: { retentionDays: 30 },
    });
    const loginCleanupResult = await expectSuccess<{ clearedCount: number }>(loginCleanupResponse);
    expect(loginCleanupResult.clearedCount).toBeGreaterThanOrEqual(0);

    const loginDeleteCandidate = loginLogList.items[0];
    const loginBatchDeleteResponse = await apiContext.post(`${apiBaseUrl}/system/login-log/batch-delete`, {
      headers: operationHeaders,
      data: { ids: [loginDeleteCandidate.id] },
    });
    const loginBatchDeleteResult = await expectSuccess<{ deletedCount: number }>(loginBatchDeleteResponse);
    expect(loginBatchDeleteResult.deletedCount).toBeGreaterThanOrEqual(1);

    const loginVerifyResponse = await apiContext.get(`${apiBaseUrl}/system/login-log/list`, {
      params: { username: adminCredentials.username, page: '1', pageSize: '10' },
    });
    const loginVerifyList = await expectSuccess<ListPage<LoginLogItem>>(loginVerifyResponse);
    expect(loginVerifyList.items.some((item) => item.id === loginDeleteCandidate.id)).toBeFalsy();

    await expect.poll(async () => {
      const operationListResponse = await apiContext.get(`${apiBaseUrl}/system/operation-log/list`, {
        params: { page: '1', pageSize: '10' },
      });
      const payload = await expectSuccess<ListPage<OperationLogItem>>(operationListResponse);
      return payload.items.length;
    }).toBeGreaterThan(0);

    const operationListResponse = await apiContext.get(`${apiBaseUrl}/system/operation-log/list`, {
      params: { page: '1', pageSize: '10' },
    });
    const operationList = await expectSuccess<ListPage<OperationLogItem>>(operationListResponse);

    const operationCleanupResponse = await apiContext.post(`${apiBaseUrl}/system/operation-log/cleanup`, {
      headers: operationHeaders,
      data: { retentionDays: 30 },
    });
    const operationCleanupResult = await expectSuccess<{ clearedCount: number }>(operationCleanupResponse);
    expect(operationCleanupResult.clearedCount).toBeGreaterThanOrEqual(0);

    const operationDeleteCandidate = operationList.items[0];
    const operationBatchDeleteResponse = await apiContext.post(`${apiBaseUrl}/system/operation-log/batch-delete`, {
      headers: operationHeaders,
      data: { ids: [operationDeleteCandidate.id] },
    });
    const operationBatchDeleteResult = await expectSuccess<{ deletedCount: number }>(operationBatchDeleteResponse);
    expect(operationBatchDeleteResult.deletedCount).toBeGreaterThanOrEqual(1);

    const operationVerifyResponse = await apiContext.get(`${apiBaseUrl}/system/operation-log/list`, {
      params: { page: '1', pageSize: '10' },
    });
    const operationVerifyList = await expectSuccess<ListPage<OperationLogItem>>(operationVerifyResponse);
    expect(operationVerifyList.items.some((item) => item.id === operationDeleteCandidate.id)).toBeFalsy();
  });

  test('sample fixture files are present for manual smoke', async () => {
    const fixtureFiles = [
      'user-import.csv',
      'dept-import.csv',
      'post-import.csv',
      'permission-import.csv',
      'dict-type-import.csv',
      'dict-item-import.csv',
    ];

    for (const file of fixtureFiles) {
      const fullPath = path.join(fixtureDir, file);
      const content = await readFixtureCsv(fullPath);
      const lines = parseCsvLines(content);
      expect(lines.length).toBeGreaterThanOrEqual(3);
      expect(lines[1]?.startsWith('#')).toBeTruthy();
    }
  });
});

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ResponseEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function expectCsv(response: APIResponse, expectedFilename: string): Promise<string> {
  expect(response.ok()).toBeTruthy();
  expect(response.headers()['content-type']).toContain('text/csv');
  expect(response.headers()['content-disposition']).toContain(expectedFilename);
  return stripBom(await response.text());
}

async function uploadCsv(context: APIRequestContext, url: string, filename: string, content: string) {
  return context.post(url, {
    multipart: {
      file: {
        name: filename,
        mimeType: 'text/csv',
        buffer: Buffer.from(content, 'utf8'),
      },
    },
  });
}

function buildCsv(headers: string[], comment: string, row: string[]) {
  return [headers.join(','), `${comment}${','.repeat(Math.max(headers.length - 1, 0))}`, row.join(',')].join('\n');
}

function parseCsvLines(csv: string) {
  return stripBom(csv)
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line.length > 0);
}

function stripBom(value: string) {
  return value.replace(/^\uFEFF/, '');
}

async function ensureSmokeRole(context: APIRequestContext) {
  const existing = await getRoleByKey(context, smokeRoleKey);
  if (existing) {
    if (existing.status !== 1) {
      const enableResponse = await context.post(`${apiBaseUrl}/system/role/batch-status`, {
        data: { roleIds: [existing.id], status: 1 },
      });
      await expectSuccess<{ updatedCount: number }>(enableResponse);
    }
    return existing;
  }

  const createResponse = await context.post(`${apiBaseUrl}/system/role`, {
    data: {
      roleName: smokeRoleName,
      roleKey: smokeRoleKey,
      sort: 90,
      status: 1,
      menuIds: [],
      permissionKeys: [],
    },
  });
  return expectSuccess<RoleItem>(createResponse);
}

async function cleanupSmokeFixtures(context: APIRequestContext, login: BrowserLoginResult) {
  const operationHeaders = await verifiedApiHeaders(context, login);

  await deleteUsersByUsername(context, operationHeaders, smokeUserName);
  await deletePostsByCode(context, operationHeaders, smokePostCode);
  await deleteDeptByName(context, operationHeaders, smokeDeptName);
  await deleteDictItems(context, operationHeaders, smokeDictCode, smokeItemValue);
  await deleteDictTypes(context, operationHeaders, smokeDictCode);
  await deletePolicies(context, operationHeaders, smokeRoleKey, smokePermissionPath, 'GET');
  await deleteRolesByKey(context, operationHeaders, smokeRoleKey);
}

async function deleteUsersByUsername(context: APIRequestContext, headers: Record<string, string>, username: string) {
  const listResponse = await context.get(`${apiBaseUrl}/system/user/list`, {
    params: { username, page: '1', pageSize: '20' },
  });
  if (!listResponse.ok()) {
    return;
  }
  const list = await expectSuccess<ListPage<UserItem>>(listResponse);
  for (const item of list.items) {
    if (item.username === username) {
      await context.delete(`${apiBaseUrl}/system/user/${item.id}`, { headers }).catch(() => undefined);
    }
  }
}

async function deletePostsByCode(context: APIRequestContext, headers: Record<string, string>, postCode: string) {
  const listResponse = await context.get(`${apiBaseUrl}/system/post/list`, {
    params: { postCode, page: '1', pageSize: '20' },
  });
  if (!listResponse.ok()) {
    return;
  }
  const list = await expectSuccess<ListPage<PostItem>>(listResponse);
  for (const item of list.items) {
    if (item.postCode === postCode) {
      await context.delete(`${apiBaseUrl}/system/post/${item.id}`, { headers }).catch(() => undefined);
    }
  }
}

async function deleteDeptByName(context: APIRequestContext, headers: Record<string, string>, deptName: string) {
  const treeResponse = await context.get(`${apiBaseUrl}/system/dept/tree`, {
    params: { deptName },
  });
  if (!treeResponse.ok()) {
    return;
  }
  const tree = await expectSuccess<DeptNode[]>(treeResponse);
  const dept = findDeptByName(tree, deptName);
  if (dept?.id) {
    await context.delete(`${apiBaseUrl}/system/dept/${dept.id}`, { headers }).catch(() => undefined);
  }
}

async function deleteDictItems(
  context: APIRequestContext,
  headers: Record<string, string>,
  dictCode: string,
  itemValue: string,
) {
  const listResponse = await context.get(`${apiBaseUrl}/system/dict/item/list`, {
    params: { dictCode, page: '1', pageSize: '50' },
  });
  if (!listResponse.ok()) {
    return;
  }
  const list = await expectSuccess<ListPage<DictItem>>(listResponse);
  for (const item of list.items) {
    if (item.dictCode === dictCode && item.itemValue === itemValue) {
      await context.delete(`${apiBaseUrl}/system/dict/item/${item.id}`, { headers }).catch(() => undefined);
    }
  }
}

async function deleteDictTypes(context: APIRequestContext, headers: Record<string, string>, dictCode: string) {
  const listResponse = await context.get(`${apiBaseUrl}/system/dict/type/list`, {
    params: { dictCode },
  });
  if (!listResponse.ok()) {
    return;
  }
  const items = await expectSuccess<DictTypeItem[]>(listResponse);
  for (const item of items) {
    if (item.dictCode === dictCode) {
      await context.delete(`${apiBaseUrl}/system/dict/type/${item.id}`, { headers }).catch(() => undefined);
    }
  }
}

async function deletePolicies(
  context: APIRequestContext,
  headers: Record<string, string>,
  roleKey: string,
  path: string,
  method: string,
) {
  const listResponse = await context.get(`${apiBaseUrl}/system/permission/list`, {
    params: { roleKey, path, method, page: '1', pageSize: '50' },
  });
  if (!listResponse.ok()) {
    return;
  }
  const list = await expectSuccess<ListPage<{ id: number; roleKey: string; path: string; method: string }>>(listResponse);
  for (const item of list.items) {
    if (item.roleKey === roleKey && item.path === path && item.method === method) {
      await context.delete(`${apiBaseUrl}/system/permission/${item.id}`, { headers }).catch(() => undefined);
    }
  }
}

async function deleteRolesByKey(context: APIRequestContext, headers: Record<string, string>, roleKey: string) {
  const role = await getRoleByKey(context, roleKey);
  if (role?.id && role.roleKey !== 'admin') {
    await context.delete(`${apiBaseUrl}/system/role/${role.id}`, { headers }).catch(() => undefined);
  }
}

async function getRoleByKey(context: APIRequestContext, roleKey: string) {
  const listResponse = await context.get(`${apiBaseUrl}/system/role/list`, {
    params: { roleKey, page: '1', pageSize: '10' },
  });
  const list = await expectSuccess<ListPage<RoleItem>>(listResponse);
  return list.items.find((item) => item.roleKey === roleKey) ?? null;
}

async function getRootDeptName(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/dept/tree`);
  const tree = await expectSuccess<DeptNode[]>(response);
  const rootNode = tree.find((node) => node.isRoot) ?? tree[0];
  expect(rootNode).toBeTruthy();
  return rootNode.deptName;
}

function findDeptByName(nodes: DeptNode[], deptName: string): DeptNode | null {
  for (const node of nodes) {
    if (node.deptName === deptName) {
      return node;
    }
    if (node.children?.length) {
      const found = findDeptByName(node.children, deptName);
      if (found) {
        return found;
      }
    }
  }
  return null;
}

async function readFixtureCsv(fullPath: string) {
  const fs = await import('node:fs/promises');
  return fs.readFile(fullPath, 'utf8');
}
