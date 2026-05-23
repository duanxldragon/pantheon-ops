import { expect, test, type Page } from '@playwright/test';
import {
  apiBaseUrl,
  authHeaders,
  requestHeaders,
  signInAsAdmin,
  verifiedHeaders,
} from '../../helpers/auth';

type MenuNode = {
  id: number;
  parentId: number;
  titleKey: string;
  path: string;
  pagePerm: string;
  perms: string;
  type: string;
  module: string;
  children?: MenuNode[];
};

type WorkbenchRole = {
  roleKey: string;
  governanceStatus: 'pending' | 'remediated' | 'clean';
  hasApiGap: boolean;
  missingApiPolicyCount: number;
  apiPolicies: Array<{ path: string; method: string }>;
  missingApiPolicies: Array<{ path: string; method: string }>;
};

async function deleteRoleByKey(page: Page, accessToken: string, roleKey: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(accessToken),
    params: { roleKey, page: 1, pageSize: 20 },
    timeout: 10000,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const item of items) {
    if (item.roleKey === roleKey && item.roleKey !== 'admin') {
      await page.request.delete(`${apiBaseUrl}/system/role/${item.id}`, {
        headers: await verifiedHeaders(page, accessToken),
      });
    }
  }
}

async function fetchManageMenuTree(page: Page, accessToken: string): Promise<MenuNode[]> {
  const response = await page.request.get(`${apiBaseUrl}/system/menu/tree`, {
    headers: authHeaders(accessToken),
    params: { scope: 'manage' },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? payload.data as MenuNode[] : [];
}

function flattenMenus(nodes: MenuNode[]): MenuNode[] {
  return nodes.flatMap((node) => [node, ...flattenMenus(node.children || [])]);
}

async function ensureGeneratePermissionMenu(page: Page, accessToken: string): Promise<number | null> {
  const menus = flattenMenus(await fetchManageMenuTree(page, accessToken));
  const existing = menus.find((item) => item.perms === 'system:module:generate');
  if (existing) {
    return null;
  }

  const generatorPage = menus.find((item) => item.path === '/system/generator' && item.pagePerm === 'system:generator:use');
  expect(generatorPage).toBeTruthy();

  const response = await page.request.post(`${apiBaseUrl}/system/menu`, {
    headers: await requestHeaders(page, accessToken),
    data: {
      parentId: generatorPage?.id,
      titleKey: 'system.permission.module.generate',
      path: '',
      component: '',
      pagePerm: '',
      perms: 'system:module:generate',
      type: 'F',
      icon: '',
      routeName: '',
      module: 'system.lowcode',
      sort: 99,
      isVisible: 1,
      isCache: 0,
      isExternal: 0,
      activeMenu: '',
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data.id as number;
}

async function deleteMenuById(page: Page, accessToken: string, menuId: number | null) {
  if (!menuId) {
    return;
  }
  await page.request.delete(`${apiBaseUrl}/system/menu/${menuId}`, {
    headers: authHeaders(accessToken),
  });
}

async function createRole(page: Page, accessToken: string, roleKey: string, roleName: string) {
  const response = await page.request.post(`${apiBaseUrl}/system/role`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      roleName,
      roleKey,
      sort: 998,
      status: 1,
      menuIds: [],
      permissionKeys: ['system:generator:use', 'system:module:generate'],
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
}

async function fetchWorkbenchRole(page: Page, accessToken: string, roleKey: string): Promise<WorkbenchRole> {
  const response = await page.request.get(`${apiBaseUrl}/system/permission/workbench`, {
    headers: authHeaders(accessToken),
    params: { roleKey },
    timeout: 10000,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  const roles = Array.isArray(payload.data?.roles) ? payload.data.roles : [];
  const role = roles.find((item: WorkbenchRole) => item.roleKey === roleKey);
  expect(role).toBeTruthy();
  return role as WorkbenchRole;
}

test('permission workbench can remediate recommended generator policy against real backend', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const roleKey = `qa_perm_real_${Date.now()}`;
  const roleName = `权限工作台整改回归_${Date.now()}`;
  let createdMenuId: number | null = null;

  await deleteRoleByKey(page, accessToken, roleKey);

  try {
    createdMenuId = await ensureGeneratePermissionMenu(page, accessToken);
    await createRole(page, accessToken, roleKey, roleName);

    const before = await fetchWorkbenchRole(page, accessToken, roleKey);
    expect(before.governanceStatus).toBe('pending');
    expect(before.hasApiGap).toBeTruthy();
    expect(before.missingApiPolicyCount).toBe(1);
    expect(before.missingApiPolicies).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ path: '/api/v1/system/dynamic-modules/generate', method: 'POST' }),
      ]),
    );

    await page.goto('/system/permission', { waitUntil: 'networkidle' });
    await expect(page.getByText('权限管理', { exact: false }).filter({ visible: true }).first()).toBeVisible();
    await expect(page.locator('.governance-summary-bar, .permission-workbench__tabs').first()).toBeVisible();
    await page.getByRole('tab', { name: '权限工作台', exact: true }).click();
    await expect(page.getByText('整改任务台', { exact: false })).toBeVisible();
    await expect(page.getByText('待整改角色', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: '仅看待整改', exact: true })).toBeVisible();

    const remediateResponse = await page.request.post(`${apiBaseUrl}/system/permission/workbench/remediate`, {
      headers: await verifiedHeaders(page, accessToken),
      data: { roleKey },
    });
    expect(remediateResponse.ok()).toBeTruthy();
    const remediatePayload = await remediateResponse.json();
    expect(remediatePayload.code).toBe(200);
    expect(remediatePayload.data?.createdCount).toBe(1);

    const after = await fetchWorkbenchRole(page, accessToken, roleKey);
    expect(after.governanceStatus).toBe('remediated');
    expect(after.hasApiGap).toBeFalsy();
    expect(after.missingApiPolicyCount).toBe(0);
    expect(after.apiPolicies).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ path: '/api/v1/system/dynamic-modules/generate', method: 'POST' }),
      ]),
    );
  } finally {
    await deleteRoleByKey(page, accessToken, roleKey).catch(() => undefined);
    await deleteMenuById(page, accessToken, createdMenuId).catch(() => undefined);
  }
});
