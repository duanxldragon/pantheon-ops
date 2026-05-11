import { expect, request as playwrightRequest, test, type APIRequestContext, type APIResponse } from '@playwright/test';
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

type ListPage<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};

type BatchDeleteResp = {
  deletedCount: number;
  failedCount: number;
  failures: Array<{ id: number; reason: string }>;
};

type RoleItem = {
  id: number;
  roleName: string;
  roleKey: string;
};

type UserItem = {
  id: number;
  username: string;
};

type DeptNode = {
  id: number;
  deptName: string;
  isRoot: boolean;
  children?: DeptNode[];
};

type PostItem = {
  id: number;
  postCode: string;
};

type DictTypeItem = {
  id: number;
  dictCode: string;
};

type DictItem = {
  id: number;
  dictCode: string;
  itemValue: string;
};

type PermissionPolicyItem = {
  id: number;
  roleKey: string;
  path: string;
  method: string;
};

const smokePrefix = 'smoke_batch_delete';

test.describe.serial('system batch delete api smoke', () => {
  let apiContext: APIRequestContext;
  let login: BrowserLoginResult;
  let operationHeaders: Record<string, string>;

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
    operationHeaders = await verifiedApiHeaders(apiContext, login);
    await cleanupBatchDeleteFixtures(apiContext, operationHeaders);
    await loginContext.dispose();
  });

  test.afterAll(async () => {
    await cleanupBatchDeleteFixtures(apiContext, operationHeaders);
    await apiContext.dispose();
  });

  test('batch delete removes system domain records and reports blocked records', async () => {
    const suffix = Date.now();
    const roleKey = `${smokePrefix}_role_${suffix}`;
    const username = `${smokePrefix}_user_${suffix}`;
    const deptName = `${smokePrefix}_dept_${suffix}`;
    const postCode = `${smokePrefix}_post_${suffix}`;
    const dictCode = `${smokePrefix}_dict_${suffix}`;
    const policyPath = `/api/v1/system/smoke-batch-delete/${suffix}`;

    const role = await createRole(apiContext, roleKey);
    const user = await createUser(apiContext, username, role.id);
    const dept = await createDept(apiContext, deptName);
    const post = await createPost(apiContext, postCode, dept.id);
    const dictType = await createDictType(apiContext, dictCode);
    const dictItem = await createDictItem(apiContext, dictCode);
    const policy = await createPermissionPolicy(apiContext, roleKey, policyPath);

    const protectedUserResponse = await apiContext.post(`${apiBaseUrl}/system/user/batch-delete`, {
      headers: operationHeaders,
      data: { ids: [1, user.id] },
    });
    const protectedUserResult = await expectSuccess<BatchDeleteResp>(protectedUserResponse);
    expect(protectedUserResult.deletedCount).toBe(1);
    expect(protectedUserResult.failedCount).toBe(1);
    expect(protectedUserResult.failures.some((item) => item.id === 1)).toBeTruthy();

    await expectUserMissing(apiContext, username);

    await expectBatchDelete(apiContext, '/system/permission/batch-delete', [policy.id]);
    await expectBatchDelete(apiContext, '/system/post/batch-delete', [post.id]);
    await expectBatchDelete(apiContext, '/system/dept/batch-delete', [dept.id]);
    await expectBatchDelete(apiContext, '/system/dict/item/batch-delete', [dictItem.id]);
    await expectBatchDelete(apiContext, '/system/dict/type/batch-delete', [dictType.id]);
    await expectBatchDelete(apiContext, '/system/role/batch-delete', [role.id]);

    expect(await getRoleByKey(apiContext, roleKey)).toBeNull();
    expect(await getDeptByName(apiContext, deptName)).toBeNull();
    expect(await getPostByCode(apiContext, postCode)).toBeNull();
    expect(await getDictTypeByCode(apiContext, dictCode)).toBeNull();
    expect(await getPermissionByPath(apiContext, roleKey, policyPath)).toBeNull();
  });

  async function expectBatchDelete(context: APIRequestContext, path: string, ids: number[]) {
    const response = await context.post(`${apiBaseUrl}${path}`, {
      headers: operationHeaders,
      data: { ids },
    });
    const result = await expectSuccess<BatchDeleteResp>(response);
    expect(result.deletedCount).toBe(ids.length);
    expect(result.failedCount).toBe(0);
  }
});

async function createRole(context: APIRequestContext, roleKey: string) {
  const response = await context.post(`${apiBaseUrl}/system/role`, {
    data: {
      roleName: `Smoke Batch Delete Role ${roleKey}`,
      roleKey,
      sort: 99,
      status: 1,
      menuIds: [],
      permissionKeys: [],
    },
  });
  return expectSuccess<RoleItem>(response);
}

async function createUser(context: APIRequestContext, username: string, roleId: number) {
  const response = await context.post(`${apiBaseUrl}/system/user`, {
    data: {
      username,
      password: 'ChangeMe123',
      nickname: 'Smoke Batch Delete User',
      email: `${username}@example.com`,
      phone: '13800138009',
      deptId: 0,
      postId: 0,
      status: 1,
      roleIds: [roleId],
    },
  });
  return expectSuccess<UserItem>(response);
}

async function createDept(context: APIRequestContext, deptName: string) {
  const rootDept = await getRootDept(context);
  const response = await context.post(`${apiBaseUrl}/system/dept`, {
    data: {
      parentId: rootDept.id,
      deptName,
      sort: 99,
      leader: '',
      phone: '',
      email: '',
      status: 1,
    },
  });
  return expectSuccess<DeptNode>(response);
}

async function createPost(context: APIRequestContext, postCode: string, deptId: number) {
  const response = await context.post(`${apiBaseUrl}/system/post`, {
    data: {
      deptId,
      postCode,
      postName: `Smoke Batch Delete Post ${postCode}`,
      sort: 99,
      status: 1,
      remark: 'smoke batch delete',
    },
  });
  return expectSuccess<PostItem>(response);
}

async function createDictType(context: APIRequestContext, dictCode: string) {
  const response = await context.post(`${apiBaseUrl}/system/dict/type`, {
    data: {
      dictCode,
      dictName: `dict.${dictCode}`,
      module: 'system.smoke',
      status: 1,
      remark: 'smoke batch delete',
    },
  });
  return expectSuccess<DictTypeItem>(response);
}

async function createDictItem(context: APIRequestContext, dictCode: string) {
  const response = await context.post(`${apiBaseUrl}/system/dict/item`, {
    data: {
      dictCode,
      itemLabelKey: `dict.${dictCode}.enabled`,
      itemValue: 'enabled',
      itemColor: 'green',
      sort: 1,
      status: 1,
      remark: 'smoke batch delete',
    },
  });
  return expectSuccess<DictItem>(response);
}

async function createPermissionPolicy(context: APIRequestContext, roleKey: string, path: string) {
  const response = await context.post(`${apiBaseUrl}/system/permission`, {
    data: {
      roleKey,
      path,
      method: 'GET',
    },
  });
  return expectSuccess<PermissionPolicyItem>(response);
}

async function cleanupBatchDeleteFixtures(
  context: APIRequestContext,
  headers: Record<string, string>,
) {
  const users = await listUsersByPrefix(context);
  if (users.length > 0) {
    await context.post(`${apiBaseUrl}/system/user/batch-delete`, {
      headers,
      data: { ids: users.map((item) => item.id) },
    }).catch(() => undefined);
  }

  const policies = await listPoliciesByPrefix(context);
  if (policies.length > 0) {
    await context.post(`${apiBaseUrl}/system/permission/batch-delete`, {
      headers,
      data: { ids: policies.map((item) => item.id) },
    }).catch(() => undefined);
  }

  const posts = await listPostsByPrefix(context);
  if (posts.length > 0) {
    await context.post(`${apiBaseUrl}/system/post/batch-delete`, {
      headers,
      data: { ids: posts.map((item) => item.id) },
    }).catch(() => undefined);
  }

  const depts = await listDeptsByPrefix(context);
  if (depts.length > 0) {
    await context.post(`${apiBaseUrl}/system/dept/batch-delete`, {
      headers,
      data: { ids: depts.map((item) => item.id) },
    }).catch(() => undefined);
  }

  const dictItems = await listDictItemsByPrefix(context);
  if (dictItems.length > 0) {
    await context.post(`${apiBaseUrl}/system/dict/item/batch-delete`, {
      headers,
      data: { ids: dictItems.map((item) => item.id) },
    }).catch(() => undefined);
  }

  const dictTypes = await listDictTypesByPrefix(context);
  if (dictTypes.length > 0) {
    await context.post(`${apiBaseUrl}/system/dict/type/batch-delete`, {
      headers,
      data: { ids: dictTypes.map((item) => item.id) },
    }).catch(() => undefined);
  }

  const roles = await listRolesByPrefix(context);
  if (roles.length > 0) {
    await context.post(`${apiBaseUrl}/system/role/batch-delete`, {
      headers,
      data: { ids: roles.map((item) => item.id) },
    }).catch(() => undefined);
  }
}

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ResponseEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function getRoleByKey(context: APIRequestContext, roleKey: string) {
  const response = await context.get(`${apiBaseUrl}/system/role/list`, {
    params: { roleKey, page: '1', pageSize: '20' },
  });
  const list = await expectSuccess<ListPage<RoleItem>>(response);
  return list.items.find((item) => item.roleKey === roleKey) ?? null;
}

async function getRootDept(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/dept/tree`);
  const tree = await expectSuccess<DeptNode[]>(response);
  const root = tree.find((item) => item.isRoot) || tree[0];
  expect(root).toBeTruthy();
  return root;
}

async function getDeptByName(context: APIRequestContext, deptName: string) {
  const response = await context.get(`${apiBaseUrl}/system/dept/tree`, {
    params: { deptName },
  });
  const tree = await expectSuccess<DeptNode[]>(response);
  return findDeptByName(tree, deptName);
}

async function getPostByCode(context: APIRequestContext, postCode: string) {
  const response = await context.get(`${apiBaseUrl}/system/post/list`, {
    params: { postCode, page: '1', pageSize: '20' },
  });
  const list = await expectSuccess<ListPage<PostItem>>(response);
  return list.items.find((item) => item.postCode === postCode) ?? null;
}

async function getDictTypeByCode(context: APIRequestContext, dictCode: string) {
  const response = await context.get(`${apiBaseUrl}/system/dict/type/list`, {
    params: { dictCode },
  });
  const items = await expectSuccess<DictTypeItem[]>(response);
  return items.find((item) => item.dictCode === dictCode) ?? null;
}

async function getPermissionByPath(context: APIRequestContext, roleKey: string, path: string) {
  const response = await context.get(`${apiBaseUrl}/system/permission/list`, {
    params: { roleKey, path, method: 'GET', page: '1', pageSize: '20' },
  });
  const list = await expectSuccess<ListPage<PermissionPolicyItem>>(response);
  return list.items.find((item) => item.roleKey === roleKey && item.path === path) ?? null;
}

async function expectUserMissing(context: APIRequestContext, username: string) {
  const response = await context.get(`${apiBaseUrl}/system/user/list`, {
    params: { username, page: '1', pageSize: '20' },
  });
  const list = await expectSuccess<ListPage<UserItem>>(response);
  expect(list.items.some((item) => item.username === username)).toBeFalsy();
}

async function listRolesByPrefix(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/role/list`, {
    params: { roleKey: smokePrefix, page: '1', pageSize: '100' },
  });
  if (!response.ok()) return [];
  const list = await expectSuccess<ListPage<RoleItem>>(response);
  return list.items.filter((item) => item.roleKey.startsWith(smokePrefix));
}

async function listUsersByPrefix(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/user/list`, {
    params: { username: smokePrefix, page: '1', pageSize: '100' },
  });
  if (!response.ok()) return [];
  const list = await expectSuccess<ListPage<UserItem>>(response);
  return list.items.filter((item) => item.username.startsWith(smokePrefix));
}

async function listPostsByPrefix(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/post/list`, {
    params: { postCode: smokePrefix, page: '1', pageSize: '100' },
  });
  if (!response.ok()) return [];
  const list = await expectSuccess<ListPage<PostItem>>(response);
  return list.items.filter((item) => item.postCode.startsWith(smokePrefix));
}

async function listDeptsByPrefix(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/dept/tree`, {
    params: { deptName: smokePrefix },
  });
  if (!response.ok()) return [];
  const tree = await expectSuccess<DeptNode[]>(response);
  return flattenDepts(tree).filter((item) => !item.isRoot && item.deptName.startsWith(smokePrefix));
}

async function listDictTypesByPrefix(context: APIRequestContext) {
  const response = await context.get(`${apiBaseUrl}/system/dict/type/list`, {
    params: { dictCode: smokePrefix },
  });
  if (!response.ok()) return [];
  const items = await expectSuccess<DictTypeItem[]>(response);
  return items.filter((item) => item.dictCode.startsWith(smokePrefix));
}

async function listDictItemsByPrefix(context: APIRequestContext) {
  const dictTypes = await listDictTypesByPrefix(context);
  const items: DictItem[] = [];
  for (const dictType of dictTypes) {
    const response = await context.get(`${apiBaseUrl}/system/dict/item/list`, {
      params: { dictCode: dictType.dictCode, page: '1', pageSize: '100' },
    });
    if (!response.ok()) continue;
    const list = await expectSuccess<ListPage<DictItem>>(response);
    items.push(...list.items);
  }
  return items;
}

async function listPoliciesByPrefix(context: APIRequestContext) {
  const roles = await listRolesByPrefix(context);
  const items: PermissionPolicyItem[] = [];
  for (const role of roles) {
    const response = await context.get(`${apiBaseUrl}/system/permission/list`, {
      params: { roleKey: role.roleKey, page: '1', pageSize: '100' },
    });
    if (!response.ok()) continue;
    const list = await expectSuccess<ListPage<PermissionPolicyItem>>(response);
    items.push(...list.items);
  }
  return items;
}

function findDeptByName(nodes: DeptNode[], deptName: string): DeptNode | null {
  for (const node of nodes) {
    if (node.deptName === deptName) return node;
    const found = findDeptByName(node.children || [], deptName);
    if (found) return found;
  }
  return null;
}

function flattenDepts(nodes: DeptNode[]): DeptNode[] {
  return nodes.flatMap((item) => [item, ...flattenDepts(item.children || [])]);
}
