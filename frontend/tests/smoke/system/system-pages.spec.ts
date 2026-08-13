import fs from 'node:fs/promises';
import syncFs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { test } from '../../fixtures/coverage';
import { expect, type BrowserContext, type Locator, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  type BrowserLoginResult,
  installClientSession,
  installOperationToken,
  loginByApi,
  requestHeaders,
  signInAsAdmin,
  signInWithUi,
  verifiedApiHeaders,
  verifiedHeaders,
} from '../helpers/auth';
import { runOptionalSmokeCleanup } from '../helpers/fixture-policy';
import { installSharedPageReadCache, type CachedReadResponse } from '../helpers/shared-read-cache';
import { expectPagePathname } from '../helpers/url-pattern';
import { registerSystemWorkspaceTaskDepthSmokeTests } from './system-workspace-task-depth';
import { listRegisteredComponentKeys } from '../../../src/core/router/componentRegistry';
const pageErrorTitles = ['加载失败', '网络异常', '请求超时'];
const pageEmptyTexts = [
  '暂无数据',
  '当前筛选范围内没有可展示的数据',
  '当前筛选下暂无岗位',
  '暂无系统设置',
  '请选择左侧字典类型后维护字典项',
  '暂无字典类型',
  '暂无字典项',
  '暂无登录日志',
  '暂无会话数据',
];
const reactElementRefWarningPattern = /Accessing element\.ref was removed in React 19/i;
const menuComponentKey =
  listRegisteredComponentKeys().find((key) => key.endsWith('/menu/MenuList')) ??
  'system/menu/MenuList';
type SettingItem = { settingKey: string; settingValue: string };
type UserPlatformPreferences = {
  theme?: string;
  language?: string;
  layoutMode?: string;
  densityMode?: string;
};
type ManageMenuNode = {
  id: number;
  parentId: number;
  titleKey: string;
  path: string;
  type: string;
  children?: ManageMenuNode[];
};

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const admissionConfig = JSON.parse(
  syncFs.readFileSync(
    path.resolve(__dirname, '../../../config/system-page-admission.json'),
    'utf8',
  ),
) as Array<{
  path: string;
  hero: 'allowed' | 'forbidden';
  governanceDrawer: 'allowed' | 'forbidden';
}>;

const compactMainAreaPages = admissionConfig
  .filter((item) => item.hero === 'forbidden' && item.governanceDrawer === 'allowed')
  .map((item) => item.path)
  .filter((item) => !item.includes(':'));

const pageIdentitySelectors = [
  '.governance-summary-bar',
  '.system-list__table-card',
  '.permission-workbench__tabs',
  '.dict-workbench',
  '.setting-overview-page',
  '.setting-group-page',
  '.module-manager-page',
  '.generator-wizard-card',
  '.dashboard-hero-card',
  '.auth-security-page',
];

const reloadStableReadPaths = new Set([
  '/api/v1/system/setting/public',
  '/api/v1/system/menu/tree',
  '/api/v1/system/dept/tree',
  '/api/v1/system/dict/type/list',
  '/api/v1/system/post/list',
]);

async function installReloadStableReadCache(page: Page) {
  return installSharedPageReadCache(page, new Map<string, CachedReadResponse>(), {
    shouldHandleRequest: ({ method, url }) =>
      method === 'GET' && reloadStableReadPaths.has(url.pathname),
  });
}

const systemPages = [
  { path: '/system/user', title: '用户管理' },
  { path: '/system/role', title: '角色管理' },
  { path: '/system/menu', title: '菜单管理' },
  { path: '/system/dept', title: '部门管理' },
  { path: '/system/post', title: '岗位管理' },
  { path: '/system/permission', title: '权限管理' },
  { path: '/system/dict', title: '字典管理' },
  { path: '/system/setting', title: '系统设置' },
  { path: '/system/i18n', title: '国际化管理' },
  { path: '/system/generator', title: /模块生成(?:器|向导)/ },
  { path: '/system/login-log', title: '登录日志' },
  { path: '/system/session', title: '会话管理' },
  { path: '/system/operation-log', title: '操作日志' },
  { path: '/system/modules', title: '模块注册表' },
] as const;

const workspacePages = [
  {
    path: '/dashboard',
    title: '工作台',
    assertReady: async (page: Page) => {
      await expect(page.locator('.dashboard-hero-card')).toBeVisible();
      await expect(page.locator('.dashboard-stat-card').first()).toBeVisible();
    },
  },
  {
    path: '/auth/security',
    title: '安全中心',
    assertReady: async (page: Page) => {
      await expect(page.locator('.page-split-layout--with-rail')).toBeVisible();
      await expect(
        page
          .locator('.page-main-column .arco-card')
          .filter({ hasText: /在线会话|Active Sessions/ })
          .first(),
      ).toBeVisible();
      await expect(
        page
          .locator('.page-main-column .arco-card')
          .filter({ hasText: /最近登录|Recent Logins/ })
          .first(),
      ).toBeVisible();
    },
  },
  {
    path: '/system/profile',
    title: '个人中心',
    assertReady: async (page: Page) => {
      await expect(page.locator('.arco-form')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
    },
  },
  {
    path: '/system/user/1',
    assertReady: async (page: Page) => {
      await expect(page.getByRole('button', { name: /返回|Back/ })).toBeVisible();
      await expect(page.getByText('基础信息', { exact: true })).toBeVisible();
      await expect(page.getByText('账号摘要', { exact: true })).toBeVisible();
    },
  },
] as const;

async function updateSettingGroup(
  page: Page,
  accessToken: string,
  groupKey: string,
  items: SettingItem[],
) {
  return page.request.put(`${apiBaseUrl}/system/setting/group/${groupKey}`, {
    headers: await verifiedHeaders(page, accessToken),
    data: { items },
  });
}

async function waitForRefreshBootstrap(page: Page) {
  await page.waitForResponse(
    (response) =>
      response.url().includes('/system/refresh/state') &&
      response.request().method() === 'GET' &&
      response.ok(),
    { timeout: 15000 },
  );
}

async function closeExtraBrowserContext(context: BrowserContext) {
  try {
    await context.close();
  } catch (error) {
    if (
      error instanceof Error &&
      error.message.includes('apiRequestContext._wrapApiCall: ENOENT') &&
      error.message.includes('.playwright-artifacts-0')
    ) {
      return;
    }
    throw error;
  }
}

function isIgnorableConsoleError(text: string) {
  return (
    text.includes('Failed to load resource: the server responded with a status of 404') ||
    reactElementRefWarningPattern.test(text)
  );
}

async function createSharedAdminLogin(page: Page): Promise<BrowserLoginResult> {
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  return login;
}

async function openUserMenu(page: Page) {
  await page.getByRole('button').and(page.locator('[aria-haspopup="menu"]')).click();
}

async function waitForOkApiResponse(
  page: Page,
  matcher: (
    response: Parameters<Page['waitForResponse']>[0] extends (arg: infer T) => boolean ? T : never,
  ) => boolean,
) {
  return page.waitForResponse((response) => matcher(response) && response.ok(), { timeout: 15000 });
}

async function clickVisibleConfirmButton(page: Page, titleText?: string) {
  const confirmDialog = getVisibleConfirmDialog(page, titleText);
  await expect(confirmDialog).toBeVisible();
  await confirmDialog.getByRole('button', { name: '确定', exact: true }).click();
}

function getVisibleConfirmDialog(page: Page, titleText?: string) {
  const allVisibleConfirmDialogs = page.locator(
    '.app-dialog:visible, .arco-modal:visible, .arco-modal-confirm:visible, .arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-popover:visible, [role="dialog"]:visible, [role="alertdialog"]:visible, [role="tooltip"]:visible',
  );
  return titleText
    ? allVisibleConfirmDialogs.filter({ hasText: titleText }).last()
    : allVisibleConfirmDialogs.last();
}

async function waitForVisibleConfirmDialog(page: Page, titleText?: string, timeout = 1000) {
  try {
    await expect(getVisibleConfirmDialog(page, titleText)).toBeVisible({ timeout });
    return true;
  } catch {
    return false;
  }
}

async function findVisibleRowByText(container: Locator, text: string, actionName?: string) {
  const rows = container.getByRole('row').filter({ hasText: text });
  const rowCount = await rows.count();
  for (let index = rowCount - 1; index >= 0; index -= 1) {
    const row = rows.nth(index);
    if (!(await row.isVisible())) {
      continue;
    }
    const rowText = (await row.textContent()) || '';
    if (!rowText.includes(text)) {
      continue;
    }
    if (actionName) {
      const actionButton = row.getByRole('button', { name: actionName, exact: true }).last();
      if (!(await actionButton.count())) {
        continue;
      }
      if (!(await actionButton.isVisible())) {
        continue;
      }
    }
    return row;
  }
  throw new Error(`Failed to find visible table row containing "${text}"`);
}

async function clickVisibleRowAction(
  page: Page,
  row: Locator,
  actionName: string,
  confirmTitleText?: string,
) {
  const actionButtons = row.getByRole('button', { name: actionName, exact: true });
  const actionCount = await actionButtons.count();
  expect(actionCount).toBeGreaterThan(0);
  for (let index = actionCount - 1; index >= 0; index -= 1) {
    const actionButton = actionButtons.nth(index);
    if (!(await actionButton.isVisible())) {
      continue;
    }
    await actionButton.click();
    if (!confirmTitleText || (await waitForVisibleConfirmDialog(page, confirmTitleText))) {
      return;
    }
  }
  throw new Error(`Failed to trigger "${actionName}" row action dialog`);
}

async function waitForVisibleRowByText(
  container: Locator,
  text: string,
  actionName?: string,
  timeout = 15000,
) {
  await expect(container.getByRole('row', { name: new RegExp(text) }).last()).toBeVisible({
    timeout,
  });
  return findVisibleRowByText(container, text, actionName);
}

async function dismissVisibleSuccessDialog(page: Page) {
  const successDialog = page
    .locator('.app-dialog:visible, [role="dialog"]:visible')
    .filter({
      has: page.getByRole('button', { name: '确定', exact: true }),
    })
    .last();
  if (await successDialog.count()) {
    await successDialog.getByRole('button', { name: '确定', exact: true }).click();
  }
}

async function deleteUserByUsername(page: Page, accessToken: string, username: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/user/list`, {
    headers: authHeaders(accessToken),
    params: { username, page: 1, pageSize: 20 },
  });
  if (!response.ok()) {
    return;
  }
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const item of items) {
    if (item.username === username) {
      await page.request
        .delete(`${apiBaseUrl}/system/user/${item.id}`, {
          headers: await verifiedHeaders(page, accessToken),
        })
        .catch(() => undefined);
    }
  }
}

async function findUserByUsername(page: Page, accessToken: string, username: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/user/list`, {
    headers: authHeaders(accessToken),
    params: { username, page: 1, pageSize: 20 },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items.find((item: { username: string }) => item.username === username) as
    | {
        id: number;
        username: string;
        nickname: string;
        email?: string;
        status: number;
      }
    | undefined;
}

async function deleteRoleByKey(page: Page, accessToken: string, roleKey: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(accessToken),
    params: { roleKey, page: 1, pageSize: 20 },
  });
  if (!response.ok()) {
    return;
  }
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const item of items) {
    if (item.roleKey === roleKey && item.roleKey !== 'admin') {
      await page.request
        .delete(`${apiBaseUrl}/system/role/${item.id}`, {
          headers: await verifiedHeaders(page, accessToken),
        })
        .catch(() => undefined);
    }
  }
}

async function deleteUserByIdWithHeaders(
  page: Page,
  headers: Record<string, string>,
  userId: number | null | undefined,
) {
  if (!userId) {
    return;
  }
  await page.request
    .delete(`${apiBaseUrl}/system/user/${userId}`, {
      headers,
    })
    .catch(() => undefined);
}

async function deleteRoleByIdWithHeaders(
  page: Page,
  headers: Record<string, string>,
  roleId: number | null | undefined,
) {
  if (!roleId) {
    return;
  }
  await page.request
    .delete(`${apiBaseUrl}/system/role/${roleId}`, {
      headers,
    })
    .catch(() => undefined);
}

async function cleanupViewerIdentity(
  page: Page,
  accessToken: string,
  username: string,
  roleKey: string,
) {
  await deleteUserByUsername(page, accessToken, username);
  await deleteRoleByKey(page, accessToken, roleKey);
}

type RoleListItem = {
  id: number;
  roleKey: string;
  roleName: string;
  status?: number;
};

type DeptListItem = {
  id: number;
  parentId: number;
  deptName: string;
  isRoot?: boolean;
  children?: DeptListItem[];
};

type PostListItem = {
  id: number;
  deptId: number;
  deptName: string;
  postCode: string;
  postName: string;
};

async function getRoleList(page: Page, accessToken: string, params?: Record<string, unknown>) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(accessToken),
    params: { page: 1, pageSize: 100, ...params },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items as RoleListItem[];
}

async function getFirstActiveRole(page: Page, accessToken: string) {
  const items = await getRoleList(page, accessToken, {
    status: 1,
    sortField: 'sort',
    sortOrder: 'asc',
  });
  const role = items.find((item) => item.roleKey !== 'guest');
  expect(role).toBeTruthy();
  return role!;
}

function flattenDeptTreeNodes(nodes: DeptListItem[]): DeptListItem[] {
  return nodes.flatMap((node) => [node, ...flattenDeptTreeNodes(node.children || [])]);
}

async function getDeptTree(page: Page, accessToken: string, params?: Record<string, unknown>) {
  const response = await page.request.get(`${apiBaseUrl}/system/dept/tree`, {
    headers: authHeaders(accessToken),
    params,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? (payload.data as DeptListItem[]) : [];
}

async function deleteDeptByName(page: Page, accessToken: string, deptName: string) {
  const rows = flattenDeptTreeNodes(
    await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' }),
  );
  const targets = rows.filter((item) => item.deptName === deptName && !item.isRoot);
  for (const item of targets.reverse()) {
    await page.request
      .delete(`${apiBaseUrl}/system/dept/${item.id}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  }
}

async function createDeptByApi(
  page: Page,
  accessToken: string,
  data: {
    parentId: number;
    deptName: string;
    sort?: number;
    phone?: string;
    email?: string;
    status?: number;
  },
) {
  const response = await page.request.post(`${apiBaseUrl}/system/dept`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      sort: 10,
      phone: '',
      email: '',
      status: 1,
      ...data,
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as DeptListItem;
}

async function getPostListItems(page: Page, accessToken: string, params?: Record<string, unknown>) {
  const response = await page.request.get(`${apiBaseUrl}/system/post/list`, {
    headers: authHeaders(accessToken),
    params: { page: 1, pageSize: 100, ...params },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items as PostListItem[];
}

async function findPostByCode(page: Page, accessToken: string, postCode: string) {
  const items = await getPostListItems(page, accessToken, { postCode });
  return items.find((item) => item.postCode === postCode);
}

async function deletePostByCode(page: Page, accessToken: string, postCode: string) {
  const items = await getPostListItems(page, accessToken, { postCode });
  for (const item of items) {
    if (item.postCode === postCode) {
      await page.request
        .delete(`${apiBaseUrl}/system/post/${item.id}`, {
          headers: await verifiedHeaders(page, accessToken),
        })
        .catch(() => undefined);
    }
  }
}

async function createPostByApi(
  page: Page,
  accessToken: string,
  data: {
    deptId: number;
    postCode: string;
    postName: string;
    sort?: number;
    remark?: string;
    status?: number;
  },
) {
  const response = await page.request.post(`${apiBaseUrl}/system/post`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      sort: 10,
      remark: '',
      status: 1,
      ...data,
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as PostListItem;
}

async function createUserByApi(
  page: Page,
  accessToken: string,
  data: {
    username: string;
    password: string;
    nickname: string;
    roleIds?: number[];
    deptId?: number;
    postId?: number;
    email?: string;
    phone?: string;
    status?: number;
  },
) {
  const response = await page.request.post(`${apiBaseUrl}/system/user`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      email: '',
      phone: '',
      status: 1,
      ...data,
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as { id: number };
}

async function fetchManageMenuTree(page: Page, accessToken: string): Promise<ManageMenuNode[]> {
  const response = await page.request.get(`${apiBaseUrl}/system/menu/tree`, {
    headers: authHeaders(accessToken),
    params: { scope: 'manage' },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? (payload.data as ManageMenuNode[]) : [];
}

function flattenManageMenus(nodes: ManageMenuNode[]): ManageMenuNode[] {
  return nodes.flatMap((node) => [node, ...flattenManageMenus(node.children || [])]);
}

async function deleteMenuById(page: Page, accessToken: string, menuId: number | null) {
  if (!menuId) {
    return;
  }
  await page.request.delete(`${apiBaseUrl}/system/menu/${menuId}`, {
    headers: await verifiedHeaders(page, accessToken),
  });
}

async function getRoleByKey(page: Page, accessToken: string, roleKey: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(accessToken),
    params: { roleKey, page: 1, pageSize: 20 },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items.find((item: { roleKey: string }) => item.roleKey === roleKey) as
    { id: number; roleKey: string } | undefined;
}

async function createApiPermission(
  page: Page,
  accessToken: string,
  roleKey: string,
  path: string,
  method: string,
) {
  const response = await page.request.post(`${apiBaseUrl}/system/permission`, {
    headers: await verifiedHeaders(page, accessToken),
    data: { roleKey, path, method },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
}

async function getCurrentUserPreferences(
  page: Page,
  accessToken: string,
): Promise<UserPlatformPreferences> {
  const response = await page.request.get(`${apiBaseUrl}/auth/me`, {
    headers: authHeaders(accessToken),
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return (payload.data?.preferences || {}) as UserPlatformPreferences;
}

async function updateCurrentUserPreferences(
  page: Page,
  accessToken: string,
  preferences: UserPlatformPreferences,
) {
  const response = await page.request.put(`${apiBaseUrl}/auth/me/preferences`, {
    headers: await requestHeaders(page, accessToken),
    data: preferences,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
}

async function expectNoPageError(page: Page) {
  for (const title of pageErrorTitles) {
    const matches = page.getByText(title, { exact: false });
    const count = await matches.count();
    for (let i = 0; i < count; i += 1) {
      await expect(matches.nth(i)).not.toBeVisible();
    }
  }
}

async function expectPageBodyReady(page: Page) {
  const table = page.locator('.arco-table');
  const empty = page.locator('.arco-empty');
  const settingGroupNav = page.locator('.setting-page__group-nav-grid');
  const settingConfigCard = page.locator('.setting-page__config-card');
  const generatorSteps = page.locator('.generator-wizard__steps');

  const hasTable = (await table.count()) > 0;
  const hasEmpty = (await empty.count()) > 0;
  const hasSettingGroupNav = (await settingGroupNav.count()) > 0;
  const hasSettingConfigCard = (await settingConfigCard.count()) > 0;
  const hasGeneratorSteps = (await generatorSteps.count()) > 0;

  expect(
    hasTable || hasEmpty || hasSettingGroupNav || hasSettingConfigCard || hasGeneratorSteps,
  ).toBeTruthy();

  if (hasEmpty) {
    const emptyText = await empty.first().innerText();
    expect(pageEmptyTexts.some((text) => emptyText.includes(text))).toBeTruthy();
  }
}

async function expectVisiblePageTitle(page: Page, title: string | RegExp) {
  const visibleMatches = page.getByText(title, { exact: false }).filter({ visible: true });
  await expect(visibleMatches.first()).toBeVisible();
}

function shellBrandTextLocator(page: Page) {
  return page.locator('.app-shell__brand-title, .app-shell__header-brand-text').first();
}

async function expectPageIdentityReady(page: Page, title: string | RegExp) {
  await expectVisiblePageTitle(page, title);
  await expect(page.locator(pageIdentitySelectors.join(', ')).first()).toBeVisible();
}

function formItem(scope: Page | ReturnType<Page['locator']>, label: string) {
  return scope
    .locator('.arco-form-item')
    .filter({ has: scope.getByText(label, { exact: true }) })
    .first();
}

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

registerSystemWorkspaceTaskDepthSmokeTests({
  expectVisiblePageTitle,
  expectPageIdentityReady,
  formItem,
});

for (const pageMeta of systemPages) {
  test(`system smoke: ${pageMeta.path}`, async ({ page }) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error' && !isIgnorableConsoleError(message.text())) {
        consoleErrors.push(message.text());
      }
    });

    await page.goto(pageMeta.path, { waitUntil: 'networkidle' });
    expectPagePathname(page, pageMeta.path);
    await expectVisiblePageTitle(page, pageMeta.title);
    await expectNoPageError(page);
    await expectPageBodyReady(page);
    expect(consoleErrors).toEqual([]);
  });
}

for (const pageMeta of workspacePages) {
  test(`workspace smoke: ${pageMeta.path} is reachable`, async ({ page }) => {
    await page.goto(pageMeta.path, { waitUntil: 'networkidle' });
    expectPagePathname(page, pageMeta.path);
    if ('title' in pageMeta && pageMeta.title) {
      await expectVisiblePageTitle(page, pageMeta.title);
    }
    await expectNoPageError(page);
    await pageMeta.assertReady(page);
  });
}

test('user page keeps list workflow primary without governance drawer entry', async ({ page }) => {
  await page.goto('/system/user', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '用户管理');
  await expect(page.locator('.system-user-list__hero')).toHaveCount(0);
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.getByRole('button', { name: '治理摘要' })).toHaveCount(1);
  await expect(page.locator('.governance-insight-drawer')).toHaveCount(0);
  await expect(page.locator('.system-list__table-card')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions')).toBeVisible();
  await expect(
    page.locator('.table-batch-action-bar__prefix-actions').getByText('新增'),
  ).toBeVisible();
  await expect(
    page.locator('.table-batch-action-bar__prefix-actions').getByText('导入'),
  ).toBeVisible();
  await expect(
    page.locator('.table-batch-action-bar__prefix-actions').getByText('导出'),
  ).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '状态' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '部门' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '岗位' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '角色' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '邮箱' })).toBeVisible();

  const firstRowActions = page.locator('.system-user-list__row-actions').first();
  await expect(firstRowActions.getByRole('button', { name: '详情' })).toBeVisible();
  await expect(firstRowActions.getByRole('button', { name: '编辑' })).toBeVisible();
  await expect(firstRowActions.getByRole('button', { name: '重置密码' })).toBeVisible();
  await expect(firstRowActions.getByRole('button', { name: '删除' })).toHaveCount(0);
});

test('setting page shows audit table only in audit group and removes governance drawer entry', async ({
  page,
}) => {
  await page.goto('/system/setting', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '系统设置');
  await expect(page.getByRole('button', { name: '治理摘要' })).toHaveCount(0);
  await expect(page.locator('.setting-page__audit-card')).toHaveCount(0);
  await expect(page.locator('.setting-overview-page')).toBeVisible();
  await expect(page.locator('.setting-overview-page__anchor-strip')).toBeVisible();
  await expect(page.locator('.setting-group-workspace')).toHaveCount(1);

  await page
    .getByRole('tab', { name: /日志治理/ })
    .first()
    .click();
  await expect(page).toHaveURL(/\/system\/setting\?group=audit$/);
  await expect(page.locator('#setting-group-section-audit')).toBeVisible();
  await expect(page.locator('.setting-page__audit-card')).toBeVisible();
});

test('setting route lands in a single visible group workspace', async ({ page }) => {
  await page.goto('/system/setting', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '系统设置');
  await expect(page.locator('.setting-overview-page')).toBeVisible();
  await expect(page.locator('.setting-overview-page__anchor-strip')).toBeVisible();
  await expect(page.locator('.setting-group-workspace')).toHaveCount(1);
  await expect(page).toHaveURL(/\/system\/setting$/);
  await expect(page.getByRole('tab', { name: '系统设置' })).toBeVisible();
  await expect(page.locator('#setting-group-section-basic')).toBeVisible();
  await expect(page.getByRole('button', { name: '保存' }).first()).toBeVisible();
});

test('setting group route isolates one group context per route', async ({ page }) => {
  await page.goto('/system/setting/security', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '系统设置');
  await expect(page.locator('.setting-group-page')).toBeVisible();
  await expect(page.locator('.setting-page__group-nav-grid')).toBeVisible();
  await expect(page.getByRole('tab', { name: '安全策略' })).toBeVisible();
  await expect(page.locator('.setting-page__audit-card')).toHaveCount(0);
  await expect(
    page.locator('.form-section__title').getByText('安全策略', { exact: true }),
  ).toBeVisible();

  await page.getByRole('button', { name: /日志治理/ }).click();
  await expect(page).toHaveURL(/\/system\/setting\/audit$/);
  await expect(page.locator('.setting-page__audit-card')).toBeVisible();
});

test('governance and audit pages remove hero-heavy main-area blocks', async ({ page }) => {
  test.setTimeout(60000);
  for (const path of compactMainAreaPages) {
    await page.goto(path, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.system-page-hero')).toHaveCount(0);
    await expect(page.locator('.system-list__hero')).toHaveCount(0);
    await expect(
      page
        .locator(
          '.system-list__table-card, .permission-workbench__tabs, .org-structure, .dict-workbench',
        )
        .first(),
    ).toBeVisible();
  }
});

test('menu smoke: create dialog uses tree parent selector', async ({ page }) => {
  await signInAsAdmin(page);
  await page.goto('/system/menu', { waitUntil: 'networkidle' });

  await page.getByRole('button', { name: '新增', exact: true }).click();
  const dialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('新增菜单', { exact: true }) });
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('.arco-tree-select').first()).toBeVisible();
});

test('menu smoke: create child action preselects clicked parent', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const menus = flattenManageMenus(await fetchManageMenuTree(page, accessToken));
  const parentMenu =
    menus.find((item) => item.path === '/system/access') ??
    menus.find((item) => item.path && item.type !== 'F');
  expect(parentMenu).toBeTruthy();

  let createdMenuId: number | null = null;
  try {
    await page.goto('/system/menu', { waitUntil: 'networkidle' });

    const parentRow = page
      .locator('.arco-table-tr')
      .filter({ hasText: parentMenu?.path || '' })
      .first();
    await expect(parentRow).toBeVisible();
    await parentRow.getByRole('button', { name: '新增子菜单', exact: true }).click();

    const dialog = page
      .getByRole('dialog')
      .filter({ has: page.getByText('新增菜单', { exact: true }) });
    await expect(dialog).toBeVisible();
    await expect(dialog.locator('.arco-tree-select').first()).toContainText('/system/access');

    const uniqueSuffix = Date.now();
    await dialog
      .getByPlaceholder('例如：system.menu.example')
      .fill(`system.menu.child.${uniqueSuffix}`);
    await dialog
      .getByPlaceholder('例如：/system/example')
      .fill(`/system/menu-child-${uniqueSuffix}`);
    await dialog.getByPlaceholder('例如：business/cmdb/CMDBTypeList').fill(menuComponentKey);
    await dialog.getByPlaceholder('例如：system-example').fill(`system-menu-child-${uniqueSuffix}`);
    await dialog
      .getByPlaceholder('例如：system.iam / system.auth / platform / business.order')
      .fill('system.iam');
    await dialog
      .getByPlaceholder('例如：system:example:list')
      .nth(0)
      .fill(`system:menu:child:${uniqueSuffix}`);

    const createResponsePromise = page.waitForResponse(
      (response) =>
        response.url().includes('/api/v1/system/menu') && response.request().method() === 'POST',
    );
    await dialog.locator('.submit-bar').getByRole('button', { name: '新增', exact: true }).click();
    const createResponse = await createResponsePromise;
    expect(createResponse.ok()).toBeTruthy();
    const requestBody = createResponse.request().postDataJSON() as { parentId: number };
    expect(requestBody.parentId).toBe(parentMenu?.id);
    const payload = await createResponse.json();
    expect(payload.code).toBe(200);
    createdMenuId = Number(payload.data?.id || 0) || null;
  } finally {
    await deleteMenuById(page, accessToken, createdMenuId);
  }
});

test('config high-sensitivity pages keep one summary container and no hero wall', async ({
  page,
}) => {
  test.setTimeout(60000);
  await page.goto('/system/modules', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '模块注册表');
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.module-manager-page__intro')).toHaveCount(0);
  await expect(page.locator('.module-manager-page__stats')).toHaveCount(0);
  await expect(
    page.locator('.module-manager-page__header-actions .arco-btn-primary'),
  ).toBeVisible();
  await expect(page.getByText('临时模块', { exact: true }).first()).toBeVisible();

  await page.goto('/system/generator', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, /模块生成(?:器|向导)/);
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.system-list__work-actions .arco-btn')).toBeVisible();
  await expect(page.locator('.generator-wizard__steps')).toBeVisible();
  await expect(page.locator('.generator-wizard__lifecycle-card')).toBeVisible();

  await page.goto('/system/i18n', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '国际化管理');
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.system-list__table-card')).toBeVisible();
});

test('owning domain label smoke: post dict and module manager surface the correct eyebrow', async ({
  page,
}) => {
  test.setTimeout(60000);
  const pageEyebrows = [
    { path: '/system/post', title: '岗位管理', eyebrow: '系统域 / 组织治理', locale: 'zh-CN' },
    { path: '/system/dict', title: '字典管理', eyebrow: '系统域 / 配置治理', locale: 'zh-CN' },
    { path: '/system/modules', title: '模块注册表', eyebrow: '系统域 / 低代码', locale: 'zh-CN' },
    {
      path: '/system/post',
      title: 'Close post status, in-use blockers, and remediation actions in one workbench',
      eyebrow: 'System Domain / Organization',
      locale: 'en-US',
    },
  ] as const;

  for (const pageMeta of pageEyebrows) {
    if (pageMeta.locale === 'en-US') {
      await page.addInitScript((locale) => {
        localStorage.setItem('pantheon_lang', locale);
        localStorage.setItem('pantheon_lang_explicit', '1');
      }, pageMeta.locale);
    }
    await page.goto(pageMeta.path, { waitUntil: 'networkidle' });
    await expectVisiblePageTitle(page, pageMeta.title);
    await expect(page.locator('.governance-summary-bar__eyebrow')).toContainText(pageMeta.eyebrow);
    for (const metricLabel of pageMeta.metricLabels ?? []) {
      await expect(
        page.locator('.governance-summary-bar__metric-label').getByText(metricLabel, {
          exact: true,
        }),
      ).toBeVisible();
    }
  }
});

test('module generator smoke: governance summary shows low-code ownership and progress metrics', async ({
  page,
}) => {
  await page.goto('/system/generator', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, /模块生成(?:器|向导)/);
  await expect(page.locator('.governance-summary-bar__eyebrow')).toContainText('系统域 / 低代码');
  await expect(page.locator('.governance-summary-bar__title')).toContainText(
    '在受控治理约束内生成业务模块脚手架',
  );
  await expect(page.locator('.governance-summary-bar__metric-label')).toContainText([
    '步骤',
    '字段',
    '文件',
  ]);
});

test('dict workspace keeps the governance summary outside one tabbed task surface', async ({
  page,
}) => {
  await page.goto('/system/dict', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '字典管理');
  const governanceBar = page.locator('.dict-page__governance-bar');
  const workbench = page.locator('.dict-page__table-card');

  await expect(governanceBar).toHaveCount(1);
  await expect(governanceBar).toBeVisible();
  await expect(workbench).toBeVisible();
  await expect(
    workbench.locator('.governance-summary-bar, .dict-page__governance-bar'),
  ).toHaveCount(0);
  await expect(workbench.getByRole('tab')).toHaveCount(2);
  await expect(workbench.locator('[role="tab"][aria-selected="true"]')).toHaveCount(1);
  await expect(workbench.locator('.dict-page__actions').first()).toBeVisible();

  const initialHierarchy = await page.evaluate(() => {
    const summary = document.querySelector<HTMLElement>('.dict-page__governance-bar');
    const taskSurface = document.querySelector<HTMLElement>('.dict-page__table-card');
    return summary && taskSurface
      ? {
          summaryHeight: summary.getBoundingClientRect().height,
          summaryTop: summary.getBoundingClientRect().top,
          surfaceHeight: taskSurface.getBoundingClientRect().height,
          surfaceTop: taskSurface.getBoundingClientRect().top,
        }
      : null;
  });
  expect(initialHierarchy).not.toBeNull();
  expect(initialHierarchy!.summaryTop).toBeLessThan(initialHierarchy!.surfaceTop);
  expect(initialHierarchy!.surfaceHeight).toBeGreaterThan(initialHierarchy!.summaryHeight);

  const itemTab = workbench.getByRole('tab').nth(1);
  await itemTab.click();
  await expect(itemTab).toHaveAttribute('aria-selected', 'true');
  await expect(
    workbench.locator('.governance-summary-bar, .dict-page__governance-bar'),
  ).toHaveCount(0);
  await expect(workbench.locator('.dict-page__actions').first()).toBeVisible();
});

test('i18n workspace keeps translation work primary and utility actions subordinate', async ({
  page,
}) => {
  await page.goto('/system/i18n', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '国际化管理');
  const summaryBar = page.locator('.governance-summary-bar');
  const tableCard = page.locator('.i18n-list-page__table-card');
  const batchBar = tableCard.locator('.table-batch-action-bar');
  const tableBody = tableCard.locator('.system-list__table, .arco-empty').first();

  await expect(summaryBar).toBeVisible();
  await expect(tableCard).toBeVisible();
  await expect(batchBar).toBeVisible();
  await expect(tableBody).toBeVisible();
  await expect(tableCard.getByRole('button', { name: '导出' })).toBeVisible();
  await expect(tableCard.getByRole('button', { name: '导入' })).toBeVisible();

  const tableHierarchy = await page.evaluate(() => {
    const card = document.querySelector<HTMLElement>('.i18n-list-page__table-card');
    const actionBar = document.querySelector<HTMLElement>(
      '.i18n-list-page__table-card .table-batch-action-bar',
    );
    const table = document.querySelector<HTMLElement>(
      '.i18n-list-page__table-card .system-list__table, .i18n-list-page__table-card .arco-empty',
    );
    return card && actionBar && table
      ? {
          actionBottom: actionBar.getBoundingClientRect().bottom,
          actionHeight: actionBar.getBoundingClientRect().height,
          cardTop: card.getBoundingClientRect().top,
          cardHeight: card.getBoundingClientRect().height,
          controlRegionHeight: table.getBoundingClientRect().top - card.getBoundingClientRect().top,
          tableHeight: table.getBoundingClientRect().height,
          tableTop: table.getBoundingClientRect().top,
        }
      : null;
  });
  expect(tableHierarchy).not.toBeNull();
  expect(tableHierarchy!.actionHeight).toBeGreaterThan(0);
  expect(tableHierarchy!.cardTop).toBeLessThan(tableHierarchy!.tableTop);
  expect(tableHierarchy!.cardHeight).toBeGreaterThan(tableHierarchy!.actionHeight);
  expect(tableHierarchy!.tableHeight).toBeGreaterThan(tableHierarchy!.controlRegionHeight);
  expect(tableHierarchy!.tableTop).toBeGreaterThanOrEqual(tableHierarchy!.actionBottom - 1);
});

test('setting workspace keeps group navigation and config primary while audit stays secondary', async ({
  page,
}) => {
  await page.goto('/system/setting/audit', { waitUntil: 'networkidle' });

  await expectVisiblePageTitle(page, '系统设置');
  const groupNav = page.locator('.setting-page__group-nav-grid');
  const configCard = page.locator('.setting-page__config-card');
  const auditCard = page.locator('.setting-page__audit-card');

  await expect(groupNav).toBeVisible();
  await expect(configCard).toBeVisible();
  await expect(auditCard).toBeVisible();
  await expect(configCard.locator('.submit-bar')).toBeVisible();

  const panelHierarchy = await page.evaluate(() => {
    const nav = document.querySelector<HTMLElement>('.setting-page__group-nav-grid');
    const config = document.querySelector<HTMLElement>('.setting-page__config-card');
    const audit = document.querySelector<HTMLElement>('.setting-page__audit-card');
    return nav && config && audit
      ? {
          navBottom: nav.getBoundingClientRect().bottom,
          navTop: nav.getBoundingClientRect().top,
          configTop: config.getBoundingClientRect().top,
          configBottom: config.getBoundingClientRect().bottom,
          auditTop: audit.getBoundingClientRect().top,
        }
      : null;
  });
  expect(panelHierarchy).not.toBeNull();
  expect(panelHierarchy!.navTop).toBeLessThan(panelHierarchy!.configTop);
  expect(panelHierarchy!.navBottom).toBeLessThanOrEqual(panelHierarchy!.configTop + 1);
  expect(panelHierarchy!.configTop).toBeLessThan(panelHierarchy!.auditTop);
  expect(panelHierarchy!.configBottom).toBeLessThanOrEqual(panelHierarchy!.auditTop + 1);
});

test('setting smoke: site name updates public brand display', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/basic`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextSiteName = `Pantheon QA ${Date.now()}`;
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'site.name' ? nextSiteName : item.settingValue,
  }));

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'basic', nextItems);
    expect(updateResponse.ok()).toBeTruthy();
    const updatePayload = await updateResponse.json();
    expect(updatePayload.code).toBe(200);

    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await expect(shellBrandTextLocator(page)).toHaveText(nextSiteName);
    await expect(page).toHaveTitle(nextSiteName);
  } finally {
    await updateSettingGroup(page, accessToken, 'basic', originalItems);
  }
});

test('i18n smoke: detail edit create and delete dialogs work', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const seedKey = `i18n.smoke.${Date.now()}`;
  const createResponse = await page.request.post(`${apiBaseUrl}/system/i18n`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      module: 'system.config',
      group: 'messages',
      key: seedKey,
      locale: 'zh-CN',
      value: '初始值',
      remark: 'smoke',
    },
  });
  expect(createResponse.ok()).toBeTruthy();
  const createPayload = await createResponse.json();
  expect(createPayload.code).toBe(200);

  await page.goto('/system/i18n', { waitUntil: 'networkidle' });
  const i18nToolbar = page.locator('.search-toolbar');
  await i18nToolbar.getByPlaceholder(/搜索/).fill(seedKey);
  await i18nToolbar.getByPlaceholder(/搜索/).press('Enter');

  const seededListResponse = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
    headers: authHeaders(accessToken),
    params: { key: seedKey, page: '1', pageSize: '10' },
  });
  const seededListPayload = await seededListResponse.json();
  const rawCreatedAt = seededListPayload.data.items[0]?.createdAt as string | undefined;
  const rawUpdatedAt = seededListPayload.data.items[0]?.updatedAt as string | undefined;

  const targetRow = page.getByRole('row', { name: new RegExp(seedKey) }).first();
  await expect(targetRow).toBeVisible();
  if (rawCreatedAt) {
    await expect(targetRow).not.toContainText(rawCreatedAt);
  }
  if (rawUpdatedAt) {
    await expect(targetRow).not.toContainText(rawUpdatedAt);
  }

  await targetRow.getByRole('button', { name: '详情' }).click();
  const detailDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('翻译详情', { exact: true }) });
  await expect(detailDialog).toBeVisible();
  await expect(detailDialog.getByText(seedKey)).toBeVisible();
  const detailText = (await detailDialog.textContent()) || '';
  if (rawCreatedAt) {
    expect(detailText).not.toContain(rawCreatedAt);
  }
  if (rawUpdatedAt) {
    expect(detailText).not.toContain(rawUpdatedAt);
  }
  expect(detailText).toMatch(/\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}(?!:\d{2})/);
  await detailDialog.getByRole('button', { name: '关闭' }).click();

  await targetRow.getByRole('button', { name: '编辑' }).click();
  const editDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('编辑翻译', { exact: true }) });
  await expect(editDialog).toBeVisible();
  const editTextarea = editDialog.locator('textarea').first();
  await editTextarea.fill('更新值');
  await editDialog.getByRole('button', { name: '确定' }).click();
  await expect(editDialog).toHaveCount(0);

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: seedKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items[0]?.value;
    })
    .toBe('更新值');

  await page.getByRole('button', { name: '新增' }).click();
  const createDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('新增翻译', { exact: true }) });
  await expect(createDialog).toBeVisible();
  const createKey = `${seedKey}.created`;
  await createDialog.locator('input').nth(0).fill('system.config');
  await createDialog.locator('input').nth(1).fill('messages');
  await createDialog.locator('input').nth(2).fill(createKey);
  await createDialog.getByRole('combobox').first().click();
  await createDialog.getByRole('option', { name: 'en-US' }).click();
  await createDialog.locator('textarea').first().fill('Created Value');
  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n') && response.request().method() === 'POST',
    ),
    createDialog.getByRole('button', { name: '确定' }).click(),
  ]);
  await expect(createDialog).toHaveCount(0);

  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n/list') &&
        decodeURIComponent(response.url()).includes(`key=${createKey}`) &&
        response.request().method() === 'GET',
    ),
    (async () => {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(createKey);
      await toolbarKeyword.press('Enter');
    })(),
  ]);

  const createdListResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
    headers: authHeaders(accessToken),
    params: { key: createKey, page: '1', pageSize: '10' },
  });
  const createdListPayload = await createdListResp.json();
  const createdRowId = createdListPayload.data.items[0]?.id as number | undefined;
  expect(createdListPayload.data.items[0]?.key).toBe(createKey);

  {
    const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
    await toolbarKeyword.fill(seedKey);
    await toolbarKeyword.press('Enter');
  }
  const deleteRow = page.getByRole('row', { name: new RegExp(seedKey) }).first();
  await deleteRow.getByRole('button', { name: '删除' }).click();
  const deleteConfirmPopup = page
    .locator(
      '.arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-popover:visible, [role="tooltip"]:visible, [role="dialog"]:visible',
    )
    .filter({ has: page.getByRole('button', { name: '确定', exact: true }) })
    .last();
  await expect(deleteConfirmPopup).toBeVisible();
  await deleteConfirmPopup.getByRole('button', { name: '确定', exact: true }).click();

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: seedKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items.length;
    })
    .toBe(1);

  if (createdRowId) {
    await page.request
      .delete(`${apiBaseUrl}/system/i18n/${createdRowId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  }
});

test('i18n smoke: import csv creates updates and downloads error file', async ({
  page,
}, testInfo) => {
  const accessToken = await signInAsAdmin(page);
  const seedBase = `i18n.import.${Date.now()}`;
  const updateKey = `${seedBase}.update`;
  const createKey = `${seedBase}.create`;

  const seedResponse = await page.request.post(`${apiBaseUrl}/system/i18n`, {
    headers: await verifiedHeaders(page, accessToken),
    data: {
      module: 'system.config',
      group: 'messages',
      key: updateKey,
      locale: 'zh-CN',
      value: '导入前旧值',
      remark: 'seed',
    },
  });
  expect(seedResponse.ok()).toBeTruthy();
  const seedPayload = await seedResponse.json();
  expect(seedPayload.code).toBe(200);

  const successCsv = [
    'module,group,key,locale,value,remark',
    `system.config,messages,${updateKey},zh-CN,导入后新值,updated by smoke`,
    `system.config,messages,${createKey},zh-CN,批量新增值,created by smoke`,
  ].join('\n');

  await page.goto('/system/i18n', { waitUntil: 'networkidle' });
  const importButton = page.getByRole('button', { name: '导入', exact: true });
  await expect(importButton).toBeVisible();
  const successFileChooserPromise = page.waitForEvent('filechooser');
  await importButton.click();
  const successFileChooser = await successFileChooserPromise;
  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n/import') && response.request().method() === 'POST',
    ),
    successFileChooser.setFiles({
      name: 'system-i18n-import.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(`\uFEFF${successCsv}`, 'utf8'),
    }),
  ]);
  await dismissVisibleSuccessDialog(page);

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: updateKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items[0]?.value;
    })
    .toBe('导入后新值');

  await expect
    .poll(async () => {
      const listResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
        headers: authHeaders(accessToken),
        params: { key: createKey, page: '1', pageSize: '10' },
      });
      const listPayload = await listResp.json();
      return listPayload.data.items[0]?.value;
    })
    .toBe('批量新增值');

  const invalidKey = `${seedBase}.invalid`;
  const invalidCsv = [
    'module,group,key,locale,value,remark',
    `system.config,messages,${invalidKey},zh-CN,,missing value`,
    `system.config,messages,${invalidKey},zh-CN,重复值,duplicate row`,
  ].join('\n');

  const downloadPromise = page.waitForEvent('download');
  const invalidFileChooserPromise = page.waitForEvent('filechooser');
  await importButton.click();
  const invalidFileChooser = await invalidFileChooserPromise;
  await Promise.all([
    waitForOkApiResponse(
      page,
      (response) =>
        response.url().includes('/system/i18n/import') && response.request().method() === 'POST',
    ),
    invalidFileChooser.setFiles({
      name: 'system-i18n-import-invalid.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(`\uFEFF${invalidCsv}`, 'utf8'),
    }),
  ]);

  const errorDownload = await downloadPromise;
  expect(errorDownload.suggestedFilename()).toBe('system-i18n-import-errors.csv');
  const downloadPath = testInfo.outputPath('system-i18n-import-errors.csv');
  await errorDownload.saveAs(downloadPath);
  const errorCsv = await fs.readFile(downloadPath, 'utf8');
  expect(errorCsv).toContain('i18n.value.required');
  expect(errorCsv).toContain('import.duplicate.row.2');

  await expect(page.locator('.search-toolbar')).toBeVisible();

  const createdListResp = await page.request.get(`${apiBaseUrl}/system/i18n/list`, {
    headers: authHeaders(accessToken),
    params: { key: createKey, page: '1', pageSize: '10' },
  });
  const createdListPayload = await createdListResp.json();
  const createdRowId = createdListPayload.data.items[0]?.id as number | undefined;

  if (createdRowId) {
    await page.request
      .delete(`${apiBaseUrl}/system/i18n/${createdRowId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  }
  await page.request
    .delete(`${apiBaseUrl}/system/i18n/${seedPayload.data.id}`, {
      headers: await verifiedHeaders(page, accessToken),
    })
    .catch(() => undefined);
});

test('setting smoke: security policy saves through setting page UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/security`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const originalValue =
    originalItems.find((item) => item.settingKey === 'security.password_min_length')
      ?.settingValue ?? '6';
  const nextValue = originalValue === '6' ? '7' : '6';

  try {
    await page.goto('/system/setting/security', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await page.locator('input[role="spinbutton"]').first().fill(nextValue);
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(async () => {
        const verifyResponse = await page.request.get(
          `${apiBaseUrl}/system/setting/group/security`,
          {
            headers: authHeaders(accessToken),
          },
        );
        const verifyPayload = await verifyResponse.json();
        return (
          verifyPayload.data.items as Array<{ settingKey: string; settingValue: string }>
        ).find((item) => item.settingKey === 'security.password_min_length')?.settingValue;
      })
      .toBe(nextValue);
  } finally {
    await updateSettingGroup(page, accessToken, 'security', originalItems);
  }
});

test('setting smoke: setting audit row opens unified operation log detail', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/audit`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const originalValue =
    originalItems.find((item) => item.settingKey === 'audit.login_log_retention_days')
      ?.settingValue ?? '90';
  const nextValue = originalValue === '90' ? '91' : '90';
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue:
      item.settingKey === 'audit.login_log_retention_days' ? nextValue : item.settingValue,
  }));

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'audit', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/system/setting/audit', { waitUntil: 'networkidle' });
    const auditCard = page.locator('.setting-page__audit-card');
    await expect(auditCard).toBeVisible();
    const viewAuditButton = auditCard.getByRole('button', { name: '查看统一审计' }).first();
    await expect(viewAuditButton).toBeVisible();
    await viewAuditButton.click();

    await expect(page).toHaveURL(/\/system\/operation-log\?detailId=\d+/);
    const detailDialog = page.getByRole('dialog');
    await expect(detailDialog).toBeVisible();
    await expect(detailDialog.getByText('请求摘要')).toBeVisible();
    await expect(detailDialog.getByText('日志治理')).toBeVisible();
  } finally {
    await updateSettingGroup(page, accessToken, 'audit', originalItems);
  }
});

test('setting smoke: login policy saves through setting page UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/login`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const originalValue =
    originalItems.find((item) => item.settingKey === 'login.max_failed_attempts')?.settingValue ??
    '5';
  const nextValue = originalValue === '5' ? '6' : '5';

  try {
    await page.goto('/system/setting/login', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await page.locator('input[role="spinbutton"]').first().fill(nextValue);
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(async () => {
        const verifyResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/login`, {
          headers: authHeaders(accessToken),
        });
        const verifyPayload = await verifyResponse.json();
        return (
          verifyPayload.data.items as Array<{ settingKey: string; settingValue: string }>
        ).find((item) => item.settingKey === 'login.max_failed_attempts')?.settingValue;
      })
      .toBe(nextValue);
  } finally {
    await updateSettingGroup(page, accessToken, 'login', originalItems);
  }
});

test('setting smoke: upload storage driver can be selected through setting page UI', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/upload`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const currentValue =
    originalItems.find((item) => item.settingKey === 'upload.storage_driver')?.settingValue ??
    'local';
  const nextValue = currentValue === 'local' ? 's3' : 'local';
  const nextLabel = nextValue === 's3' ? 'S3 兼容对象存储' : '本地存储';

  try {
    await page.goto('/system/setting/upload', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await page.locator('.arco-select-view').first().click();
    await page.locator('.arco-select-option').filter({ hasText: nextLabel }).first().click();
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(async () => {
        const verifyResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/upload`, {
          headers: authHeaders(accessToken),
        });
        const verifyPayload = await verifyResponse.json();
        return (
          verifyPayload.data.items as Array<{ settingKey: string; settingValue: string }>
        ).find((item) => item.settingKey === 'upload.storage_driver')?.settingValue;
      })
      .toBe(nextValue);
  } finally {
    await updateSettingGroup(page, accessToken, 'upload', originalItems);
  }
});

test('setting smoke: default language applies when there is no explicit choice', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  await installReloadStableReadCache(page);
  const originalPreferences = await getCurrentUserPreferences(page, accessToken);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/i18n`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'i18n.default_language' ? 'en-US' : item.settingValue,
  }));

  try {
    await updateCurrentUserPreferences(page, accessToken, {
      theme: originalPreferences.theme,
      language: '',
      layoutMode: originalPreferences.layoutMode,
      densityMode: originalPreferences.densityMode,
    });
    const updateResponse = await updateSettingGroup(page, accessToken, 'i18n', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.addInitScript(() => {
      localStorage.removeItem('pantheon_lang');
      localStorage.removeItem('pantheon_lang_explicit');
    });
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await page.reload({ waitUntil: 'networkidle' });

    await expect(page.getByRole('heading', { name: 'Workbench' }).first()).toBeVisible();
  } finally {
    await updateCurrentUserPreferences(page, accessToken, originalPreferences);
    await updateSettingGroup(page, accessToken, 'i18n', originalItems);
  }
});

test('setting smoke: default language can be selected through setting page UI', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/i18n`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const currentValue =
    originalItems.find((item) => item.settingKey === 'i18n.default_language')?.settingValue ??
    'zh-CN';
  const nextValue = currentValue === 'zh-CN' ? 'en-US' : 'zh-CN';
  const nextLabel = nextValue === 'en-US' ? 'English' : '中文';

  try {
    await page.goto('/system/setting/i18n', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await page.locator('.arco-select-view').first().click();
    await page.locator('.arco-select-option').filter({ hasText: nextLabel }).first().click();
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(async () => {
        const verifyResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/i18n`, {
          headers: authHeaders(accessToken),
        });
        const verifyPayload = await verifyResponse.json();
        return (
          verifyPayload.data.items as Array<{ settingKey: string; settingValue: string }>
        ).find((item) => item.settingKey === 'i18n.default_language')?.settingValue;
      })
      .toBe(nextValue);
  } finally {
    await updateSettingGroup(page, accessToken, 'i18n', originalItems);
  }
});

test('setting smoke: login page language choice overrides saved preference and system default for the session', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  await installReloadStableReadCache(page);
  const originalPreferences = await getCurrentUserPreferences(page, accessToken);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/i18n`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'i18n.default_language' ? 'ja-JP' : item.settingValue,
  }));

  try {
    await updateCurrentUserPreferences(page, accessToken, {
      theme: originalPreferences.theme,
      language: 'zh-CN',
      layoutMode: originalPreferences.layoutMode,
      densityMode: originalPreferences.densityMode,
    });
    const updateResponse = await updateSettingGroup(page, accessToken, 'i18n', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.context().clearCookies();
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
    await page.reload({ waitUntil: 'networkidle' });

    await page.locator('.auth-login-page__tools .arco-select-view').click();
    await page.locator('.arco-select-option').filter({ hasText: 'English' }).first().click();
    await page.getByPlaceholder(/username/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/password/i).fill(adminCredentials.password);
    await page.getByRole('button', { name: /sign in/i }).click();

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole('heading', { name: 'Workbench' }).first()).toBeVisible();
    await expect
      .poll(async () => {
        return page.evaluate(() => ({
          language: localStorage.getItem('pantheon_lang'),
          explicit: localStorage.getItem('pantheon_lang_explicit'),
        }));
      })
      .toEqual({
        language: 'en-US',
        explicit: '1',
      });
  } finally {
    const restoreToken = await signInAsAdmin(page);
    await updateCurrentUserPreferences(page, restoreToken, originalPreferences);
    await updateSettingGroup(page, restoreToken, 'i18n', originalItems);
  }
});

test('setting smoke: logout clears explicit language and falls back to default language', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  await installReloadStableReadCache(page);
  const originalPreferences = await getCurrentUserPreferences(page, accessToken);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/i18n`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'i18n.default_language' ? 'en-US' : item.settingValue,
  }));

  try {
    await updateCurrentUserPreferences(page, accessToken, {
      theme: originalPreferences.theme,
      language: '',
      layoutMode: originalPreferences.layoutMode,
      densityMode: originalPreferences.densityMode,
    });
    const updateResponse = await updateSettingGroup(page, accessToken, 'i18n', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await openUserMenu(page);
    await page.getByRole('menuitem', { name: '退出登录' }).click();

    await expect(page).toHaveURL(/\/login$/);
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible();

    await page.getByRole('textbox', { name: 'Username' }).fill('admin');
    await page.getByLabel('Password').fill('123456');
    await page.getByRole('button', { name: 'Sign in' }).click();

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole('heading', { name: 'Workbench' }).first()).toBeVisible();
  } finally {
    const restoreToken = await signInAsAdmin(page);
    await updateCurrentUserPreferences(page, restoreToken, originalPreferences);
    await updateSettingGroup(page, restoreToken, 'i18n', originalItems);
  }
});

test('auth smoke: logout sends revoke request without stale invalid-session prompt', async ({
  page,
}) => {
  const tokens = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, tokens);

  let captureAuthFailures = false;
  const authFailures: string[] = [];

  page.on('response', async (response) => {
    if (
      !captureAuthFailures ||
      !response.url().includes('/api/v1/') ||
      response.url().endsWith('/auth/logout')
    ) {
      return;
    }
    try {
      const payload = await response.json();
      if (!payload || typeof payload !== 'object') {
        return;
      }
      const code = 'code' in payload ? payload.code : undefined;
      const message = 'message' in payload ? payload.message : undefined;
      const isAuthFailure =
        code === 401 ||
        message === 'session.invalid' ||
        (typeof message === 'string' && message.startsWith('token.'));
      if (isAuthFailure) {
        authFailures.push(
          `${response.request().method()} ${response.url()} -> ${String(message || code)}`,
        );
      }
    } catch {
      // ignore non-json responses
    }
  });

  await page.goto('/dashboard', { waitUntil: 'networkidle' });
  captureAuthFailures = true;
  const logoutResponsePromise = page.waitForResponse((response) =>
    response.url().includes('/api/v1/auth/logout'),
  );
  await openUserMenu(page);
  await page.getByRole('menuitem', { name: /退出登录|Sign out|Logout/ }).click();
  const logoutPayload = await (await logoutResponsePromise).json();

  await expect(page).toHaveURL(/\/login$/);
  // Wait for the login page to settle (all in-flight requests done) so any
  // stray auth-failure message would have been rendered before the negative
  // assertions below — replaces a fixed 1s delay.
  await page.waitForLoadState('networkidle').catch(() => {});
  expect(logoutPayload.code).toBe(200);
  await expect(
    page.locator('.arco-message').filter({ hasText: /无效会话|session\.invalid|token\./i }),
  ).toHaveCount(0);
  expect(authFailures).toEqual([]);
});

test('platform smoke: lock screen keeps current route and opened tabs', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/ui`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'ui.enable_tab_bar' ? 'true' : item.settingValue,
  }));

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'ui', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await expectPageIdentityReady(page, '用户管理');
    await expect(page.locator('.app-shell__tabs [role="tab"]')).toHaveCount(2);

    await openUserMenu(page);
    await page.getByRole('menuitem', { name: /锁定屏幕|Lock Screen/i }).click();
    await expect(page.getByRole('dialog')).toContainText(/会话已锁定|Session Locked/);

    await page
      .getByPlaceholder(/请输入当前账号密码以解锁|Enter the current account password to unlock/)
      .fill('123456');
    await page.getByRole('button', { name: /解锁|Unlock/ }).click();

    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page).toHaveURL(/\/system\/user$/);
    await expectPageIdentityReady(page, '用户管理');
    await expect(page.locator('.app-shell__tabs [role="tab"]')).toHaveCount(2);
  } finally {
    await updateSettingGroup(page, accessToken, 'ui', originalItems);
  }
});

test('platform smoke: lock screen refreshes activity timestamp and blocks command palette while locked', async ({
  page,
}) => {
  await signInAsAdmin(page);

  await page.goto('/auth/security', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '安全中心');

  await page.goto('/system/profile', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '个人中心');

  const beforeLockActivity = await page.evaluate(() =>
    sessionStorage.getItem('pantheon_shell_last_activity_at'),
  );
  expect(beforeLockActivity).toBeTruthy();

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: /锁定屏幕|Lock Screen/i }).click();
  const lockDialog = page.getByRole('dialog');
  await expect(lockDialog).toContainText(/会话已锁定|Session Locked/);

  await page.keyboard.press(`${process.platform === 'darwin' ? 'Meta' : 'Control'}+K`);
  await expect(page.locator('.app-command')).toHaveCount(0);

  await page
    .getByPlaceholder(/请输入当前账号密码以解锁|Enter the current account password to unlock/)
    .fill('123456');
  await page.getByRole('button', { name: /解锁|Unlock/ }).click();
  await expect(lockDialog).toHaveCount(0);
  await expect(page).toHaveURL(/\/system\/profile$/);

  const afterUnlockActivity = await page.evaluate(() =>
    sessionStorage.getItem('pantheon_shell_last_activity_at'),
  );
  expect(afterUnlockActivity).toBeTruthy();
  expect(Number(afterUnlockActivity)).toBeGreaterThan(Number(beforeLockActivity));
});

test('auth smoke: login page shows idle-timeout notice once', async ({ page }) => {
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.evaluate(() => {
    sessionStorage.setItem('pantheon_login_notice', 'session.idle_timeout');
  });

  await page.reload({ waitUntil: 'networkidle' });
  await expect(
    page.getByText('当前账号因超过会话空闲时长被自动退出，请重新登录继续操作。', { exact: true }),
  ).toBeVisible();

  await page.reload({ waitUntil: 'networkidle' });
  await expect(
    page.getByText('当前账号因超过会话空闲时长被自动退出，请重新登录继续操作。', { exact: true }),
  ).toHaveCount(0);
});

test('platform + system/auth smoke: locked session times out, relogin notice appears, and security center shows session context', async ({
  page,
}) => {
  test.setTimeout(60_000);

  await page.goto('/system/profile', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '个人中心');

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: /锁定屏幕|Lock Screen/i }).click();
  const lockDialog = page.getByRole('dialog');
  await expect(lockDialog).toContainText(/会话已锁定|Session Locked/);

  await page.evaluate(() => {
    sessionStorage.setItem('pantheon_shell_last_activity_at', String(Date.now() - 31 * 60 * 1000));
  });

  await page.reload({ waitUntil: 'networkidle' });
  await expect(page.getByRole('dialog')).toContainText(/会话已锁定|Session Locked/);
  await expect(page).toHaveURL(/\/login$/, { timeout: 25_000 });
  await expect(
    page.getByText(
      /当前账号因超过会话空闲时长被自动退出，请重新登录继续操作。|This account was signed out automatically after being idle for too long\. Sign in again to continue\./,
    ),
  ).toBeVisible();

  await signInWithUi(page, adminCredentials);
  await expect(page).toHaveURL(/\/dashboard$/);

  await page.goto('/auth/security', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '安全中心');
  await expect(page.getByText('在线会话', { exact: true })).toBeVisible();
  await expect(page.getByText('最近登录', { exact: true })).toBeVisible();
  await expect(page.getByText('当前设备', { exact: true }).first()).toBeVisible();
  await expect(page.locator('.arco-table')).toHaveCount(2);
  await expect(page.getByText('成功', { exact: true }).first()).toBeVisible();
});

test('setting smoke: logout clears explicit theme and falls back to default theme', async ({
  page,
}) => {
  test.setTimeout(45000);
  const accessToken = await signInAsAdmin(page);
  const originalPreferences = await getCurrentUserPreferences(page, accessToken);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/ui`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextTheme = 'emerald';
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'ui.default_theme' ? nextTheme : item.settingValue,
  }));

  try {
    await updateCurrentUserPreferences(page, accessToken, {
      theme: '',
      language: originalPreferences.language,
      layoutMode: originalPreferences.layoutMode,
      densityMode: originalPreferences.densityMode,
    });
    const updateResponse = await updateSettingGroup(page, accessToken, 'ui', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await page.evaluate(() => {
      localStorage.setItem('pantheon_theme', 'slate');
      document.documentElement.dataset.pantheonTheme = 'slate';
    });

    await expect
      .poll(async () => page.evaluate(() => document.documentElement.dataset.pantheonTheme))
      .toBe('slate');

    await openUserMenu(page);
    await page.getByRole('menuitem', { name: '退出登录' }).click();

    await expect(page).toHaveURL(/\/login$/);
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.dataset.pantheonTheme))
      .toBe(nextTheme);

    await page.getByRole('textbox', { name: /用户名|Username/ }).fill('admin');
    await page.getByLabel(/密码|Password/).fill('123456');
    await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.dataset.pantheonTheme))
      .toBe(nextTheme);
  } finally {
    const restoreToken = await page.evaluate(() => localStorage.getItem('pantheon_access_token'));
    const effectiveToken = restoreToken || (await signInAsAdmin(page));
    await updateCurrentUserPreferences(page, effectiveToken, originalPreferences);
    await updateSettingGroup(page, effectiveToken, 'ui', originalItems);
  }
});

test('setting smoke: tab bar visibility follows ui preference', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/ui`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'ui.enable_tab_bar' ? 'false' : item.settingValue,
  }));

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'ui', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await expect(page.locator('.app-shell__tabs')).toHaveCount(0);
  } finally {
    await updateSettingGroup(page, accessToken, 'ui', originalItems);
  }
});

test('setting smoke: default theme applies when explicit theme preference is cleared', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  await installReloadStableReadCache(page);
  const originalPreferences = await getCurrentUserPreferences(page, accessToken);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/ui`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextTheme = 'emerald';
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'ui.default_theme' ? nextTheme : item.settingValue,
  }));

  try {
    await updateCurrentUserPreferences(page, accessToken, {
      theme: '',
      language: originalPreferences.language,
      layoutMode: originalPreferences.layoutMode,
      densityMode: originalPreferences.densityMode,
    });
    const updateResponse = await updateSettingGroup(page, accessToken, 'ui', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/dashboard', { waitUntil: 'networkidle' });
    await page.evaluate(() => {
      localStorage.removeItem('pantheon_theme');
    });
    await page.reload({ waitUntil: 'networkidle' });

    await expect
      .poll(async () => page.evaluate(() => document.documentElement.dataset.pantheonTheme))
      .toBe(nextTheme);
  } finally {
    await updateCurrentUserPreferences(page, accessToken, originalPreferences);
    await updateSettingGroup(page, accessToken, 'ui', originalItems);
  }
});

test('setting smoke: upload config affects runtime upload endpoint', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/upload`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const nextItems = originalItems.map((item) => {
    switch (item.settingKey) {
      case 'upload.storage_driver':
        return { settingKey: item.settingKey, settingValue: 'local' };
      case 'upload.max_file_size':
        return { settingKey: item.settingKey, settingValue: '1' };
      case 'upload.allowed_types':
        return { settingKey: item.settingKey, settingValue: '["png"]' };
      case 'upload.public_base_url':
        return { settingKey: item.settingKey, settingValue: '' };
      default:
        return item;
    }
  });

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'upload', nextItems);
    expect(updateResponse.ok()).toBeTruthy();
    const uploadPayload = await page.request.post(
      `${apiBaseUrl}/system/upload?scope=profile/avatar`,
      {
        headers: await requestHeaders(page, accessToken),
        multipart: {
          file: {
            name: 'avatar.png',
            mimeType: 'image/png',
            buffer: Buffer.concat([
              Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
              Buffer.from('pantheon-upload-smoke', 'utf8'),
            ]),
          },
        },
      },
    );
    expect(uploadPayload.ok()).toBeTruthy();
    const uploadResult = await uploadPayload.json();
    expect(uploadResult.code).toBe(200);
    expect(uploadResult.data.url).toContain('/api/v1/system/upload/files/profile/avatar/');

    const fileResponse = await page.request.get(uploadResult.data.url);
    expect(fileResponse.ok()).toBeTruthy();
    expect((await fileResponse.body()).toString('utf8')).toContain('pantheon-upload-smoke');

    const blockedPayload = await page.request.post(
      `${apiBaseUrl}/system/upload?scope=profile/avatar`,
      {
        headers: await requestHeaders(page, accessToken),
        multipart: {
          file: {
            name: 'avatar.txt',
            mimeType: 'text/plain',
            buffer: Buffer.from('blocked', 'utf8'),
          },
        },
      },
    );
    expect(blockedPayload.ok()).toBeTruthy();
    const blockedResult = await blockedPayload.json();
    expect(blockedResult.code).not.toBe(200);
    expect(blockedResult.message).toBe('upload.file.type_not_allowed');
  } finally {
    await updateSettingGroup(page, accessToken, 'upload', originalItems);
  }
});

test('operation log smoke: failed reason and detail summary are visible', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const failedUploadResponse = await page.request.post(
    `${apiBaseUrl}/system/upload?scope=profile/avatar`,
    {
      headers: await requestHeaders(page, accessToken),
      multipart: {
        file: {
          name: 'audit-failure.txt',
          mimeType: 'text/plain',
          buffer: Buffer.from('audit-failure', 'utf8'),
        },
      },
    },
  );
  expect(failedUploadResponse.ok()).toBeTruthy();
  const failedUploadPayload = await failedUploadResponse.json();
  expect(failedUploadPayload.code).not.toBe(200);
  expect(failedUploadPayload.message).toBe('upload.file.type_not_allowed');

  await page.goto('/system/operation-log', { waitUntil: 'networkidle' });
  // SearchToolbar：关键词 + 行内下拉即时触发；低频筛选在“筛选”弹层内。
  // keyword 按存储值匹配（title 列存 i18n key、oper_name、request_id），
  // 所以这里用操作人 admin 覆盖关键词交互，行定位仍用渲染后的中文标题。
  const auditToolbar = page.locator('.search-toolbar');
  await auditToolbar.getByPlaceholder(/搜索/).fill('admin');
  await auditToolbar.getByPlaceholder(/搜索/).press('Enter');
  await auditToolbar.locator('.arco-select-view').first().click();
  await page.locator('.arco-select-option').filter({ hasText: '失败' }).first().click();
  await auditToolbar.locator('.arco-select-view').nth(1).click();
  await page.locator('.arco-select-option').filter({ hasText: '系统配置' }).first().click();
  await auditToolbar.locator('.search-toolbar__advanced-trigger').click();
  const auditPopover = page.locator('.search-toolbar__popover');
  await expect(auditPopover).toBeVisible();
  await auditPopover.locator('.arco-select-view').first().click();
  await page.locator('.arco-select-option').filter({ hasText: '参数/校验失败' }).first().click();
  await auditPopover.locator('.arco-select-view').nth(1).click();
  await page.locator('.arco-select-option').filter({ hasText: '上传配置' }).first().click();
  await page.keyboard.press('Escape');

  const firstRow = page.getByRole('row', { name: /上传文件/ }).first();
  await expect(firstRow).toBeVisible();
  await expect(firstRow).toContainText('上传文件');
  await expect(firstRow).toContainText('系统配置');
  await expect(firstRow).toContainText('当前文件类型不在允许范围内');
  await firstRow.getByRole('button', { name: '详情' }).click();

  const detailDialog = page.getByRole('dialog');
  await expect(detailDialog.getByText('失败原因')).toBeVisible();
  await expect(detailDialog.getByText('参数/校验失败').first()).toBeVisible();
  await expect(detailDialog.getByText('系统配置').first()).toBeVisible();
  await expect(detailDialog.getByText('上传配置').first()).toBeVisible();
  await expect(detailDialog.getByText('当前文件类型不在允许范围内').first()).toBeVisible();
  await expect(detailDialog.getByText('upload.file.type_not_allowed').first()).toBeVisible();
});

test('setting permission smoke: list-only role can view page but cannot save or refresh', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `setting_view_only_${Date.now()}`;
  const username = `setting_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '系统设置只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:setting:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/setting/list',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '系统设置只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();
    const createUserPayload = await createUserResponse.json();
    expect(createUserPayload.code).toBe(200);

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/setting/basic', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '系统设置');
      await expectNoPageError(viewerPage);

      const settingPanel = viewerPage.locator('.setting-page__config-card');
      await expect(settingPanel.getByRole('button', { name: '刷新设置缓存' })).toBeDisabled();
      await expect(
        settingPanel.locator('.submit-bar').getByRole('button', { name: '保存' }),
      ).toBeDisabled();
      await expect(
        settingPanel.locator('.submit-bar').getByRole('button', { name: '取消' }),
      ).toBeEnabled();
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:setting-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('dict permission smoke: list-only role can view page but cannot mutate config', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `dict_view_only_${Date.now()}`;
  const username = `dict_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '字典只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:dict:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/dict/type/list',
      'GET',
    );
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/dict/item/list',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '字典只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/dict', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '字典管理');
      await expectNoPageError(viewerPage);

      const typePanel = viewerPage.locator('.dict-page__actions').first();
      await expect(typePanel.getByRole('button', { name: '导出' })).toBeDisabled();
      await expect(typePanel.getByRole('button', { name: '下载模板' })).toBeDisabled();
      await expect(typePanel.getByRole('button', { name: '导入' })).toBeDisabled();
      await expect(typePanel.getByRole('button', { name: '新增' })).toBeDisabled();

      await viewerPage.getByRole('tab', { name: '字典项' }).click();
      const itemPanel = viewerPage.locator('.dict-page__actions').nth(1);
      await expect(itemPanel.getByRole('button', { name: '刷新缓存' })).toBeDisabled();
      await expect(itemPanel.getByRole('button', { name: '导出' })).toBeDisabled();
      await expect(itemPanel.getByRole('button', { name: '下载模板' })).toBeDisabled();
      await expect(itemPanel.getByRole('button', { name: '导入' })).toBeDisabled();
      await expect(itemPanel.getByRole('button', { name: '新增字典项' })).toBeDisabled();
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:dict-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('i18n permission smoke: list-only role can view page but cannot mutate translations', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `i18n_view_only_${Date.now()}`;
  const username = `i18n_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '国际化只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:i18n:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/i18n/list', 'GET');
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/i18n/overview',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '国际化只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/i18n', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '国际化管理');
      await expectNoPageError(viewerPage);

      const headerActions = viewerPage.locator('.system-list__work-actions').first();
      await expect(headerActions.getByRole('button', { name: '新增' })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '刷新缓存' })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '刷新', exact: true })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '导出' })).toHaveCount(0);
      await expect(headerActions.getByRole('button', { name: '导入' })).toHaveCount(0);
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:i18n-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('login-log permission smoke: list-only role can view page but cannot clear, export, or batch delete', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `login_log_view_only_${Date.now()}`;
  const username = `login_log_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '登录日志只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:login-log:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/login-log/list',
      'GET',
    );
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/setting/group/audit',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '登录日志只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/login-log', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '登录日志');
      await expectNoPageError(viewerPage);

      await expect(viewerPage.getByRole('button', { name: '导出' })).toBeDisabled();
      await expect(viewerPage.getByRole('button', { name: '清理日志' })).toHaveCount(0);
      await expect(viewerPage.getByRole('button', { name: '删除所选' })).toHaveCount(0);
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:login-log-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('session permission smoke: list-only role can view page but cannot revoke or clear sessions', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `session_view_only_${Date.now()}`;
  const username = `session_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '会话只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:session:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/session/list',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '会话只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/session', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '会话管理');
      await expectNoPageError(viewerPage);

      await expect(viewerPage.getByRole('button', { name: '清理历史会话' })).toHaveCount(0);
      const revokeButtons = viewerPage.getByRole('button', { name: '下线会话' });
      await expect(revokeButtons.first()).toBeDisabled();
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:session-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('operation-log permission smoke: list-only role can view page but cannot clear, export, or batch delete', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `operation_log_view_only_${Date.now()}`;
  const username = `operation_log_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '操作日志只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:operation-log:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/operation-log/list',
      'GET',
    );
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/system/setting/group/audit',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '操作日志只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/operation-log', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '操作日志');
      await expectNoPageError(viewerPage);

      await expect(viewerPage.getByRole('button', { name: '导出' })).toBeDisabled();
      await expect(viewerPage.getByRole('button', { name: '清理日志' })).toHaveCount(0);
      await expect(viewerPage.getByRole('button', { name: '删除所选' })).toHaveCount(0);
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:operation-log-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('module permission smoke: list-only role can view registry but cannot register or unregister modules', async ({
  page,
}) => {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `module_view_only_${Date.now()}`;
  const username = `module_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: '模块注册表只读烟测角色',
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: ['system:module:list'],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    await createApiPermission(
      page,
      adminAccessToken,
      roleKey,
      '/api/v1/lowcode/dynamic-modules',
      'GET',
    );
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: '模块注册表只读烟测用户',
        status: 1,
        roleIds: [role!.id],
      },
    });
    expect(createUserResponse.ok()).toBeTruthy();

    const viewerTokens = await loginByApi(page.request, { username, password });
    const viewerPage = await page.context().newPage();

    try {
      await installClientSession(viewerPage, viewerTokens);
      await viewerPage.goto('/system/modules', { waitUntil: 'networkidle' });
      await expectPageIdentityReady(viewerPage, '模块注册表');
      await expectNoPageError(viewerPage);

      await expect(viewerPage.getByRole('button', { name: '前往生成器' })).toHaveCount(0);
      await expect(viewerPage.getByRole('button', { name: '卸载' })).toHaveCount(0);
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup('system-pages:module-viewer', async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
});

test('module manager smoke: auto-recycle module shows explicit lifecycle and purge guidance', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/lowcode\/dynamic-modules(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: [
          {
            id: 4101,
            name: 'business.tempautoqa',
            displayName: '临时 QA 模块',
            scope: 'business',
            source: 'generated',
            owner: 'codex',
            boundedContext: 'qa',
            summary: '用于回收交互验收',
            tableName: 'biz_temp_auto_qa',
            autoRecycle: true,
            status: 1,
            installedAt: '2026-05-20T10:00:00+08:00',
            builtIn: false,
          },
        ],
      }),
    });
  });

  await page.goto('/system/modules', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '模块注册表');
  await expectNoPageError(page);

  const row = page.locator('.arco-table-tr').filter({ hasText: 'business.tempautoqa' }).first();
  await expect(row).toBeVisible();

  await row.getByRole('button', { name: '卸载', exact: true }).click();
  await expect(
    page.getByText('确认卸载该临时模块吗？卸载时会自动回收业务表 biz_temp_auto_qa。'),
  ).toBeVisible();

  const popconfirmCancel = page.getByRole('button', { name: '取消' }).last();
  await popconfirmCancel.click();

  await row.getByRole('button', { name: '彻底删除', exact: true }).click();
  const purgeDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('彻底删除模块', { exact: true }) });
  await expect(purgeDialog).toBeVisible();
  await expect(
    purgeDialog.getByText('该模块已标记为临时模块，彻底删除时会自动回收业务表 biz_temp_auto_qa。'),
  ).toBeVisible();
  await expect(
    purgeDialog.getByText('临时模块的业务表会随彻底删除一起自动回收，这个动作不需要额外勾选。'),
  ).toBeVisible();
  await expect(purgeDialog.getByText('同时删除业务数据表', { exact: false })).toHaveCount(0);
});

test('login-log governance smoke: selecting rows enables batch delete affordance', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.goto('/system/login-log', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '登录日志');
  await expectNoPageError(page);

  const batchDeleteButton = page.getByRole('button', { name: '删除所选' });
  await expect(batchDeleteButton).toBeDisabled();
  await page.locator('.system-list__table .arco-checkbox').nth(1).click({ force: true });
  await expect(batchDeleteButton).toBeEnabled();
});

test('login-log governance smoke: pager exposes first and last page controls', async ({ page }) => {
  const pageSize = 10;
  const total = pageSize + 1;
  const totalPages = Math.ceil(total / pageSize);
  const loginLogItems = Array.from({ length: total }, (_, index) => ({
    id: 910000 + index,
    username: `${adminCredentials.username}-${index + 1}`,
    ipaddr: '127.0.0.1',
    loginLocation: '本地',
    browser: 'Chrome',
    os: 'Windows',
    status: 1,
    msg: '',
    loginTime: `2026-05-${String((index % 28) + 1).padStart(2, '0')} 09:00:00`,
  }));

  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    const requestUrl = new URL(route.request().url());
    const currentPage = Number(requestUrl.searchParams.get('page') || '1');
    const currentPageSize = Number(requestUrl.searchParams.get('pageSize') || String(pageSize));
    const startIndex = (currentPage - 1) * currentPageSize;
    const items = loginLogItems.slice(startIndex, startIndex + currentPageSize);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items,
          total,
          page: currentPage,
          pageSize: currentPageSize,
        },
      }),
    });
  });

  await page.goto('/system/login-log', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '登录日志');
  await expectNoPageError(page);

  const pager = page.locator('.system-list__table').first();
  const firstPageButton = pager.getByRole('button', { name: /^(首页|First page)$/ });
  const lastPageButton = pager.getByRole('button', { name: /^(末页|Last page)$/ });
  const prevPageButton = pager.locator('.arco-pagination-item-prev');
  const nextPageButton = pager.locator('.arco-pagination-item-next');
  await expect(prevPageButton.getByRole('button', { name: /^(首页|First page)$/ })).toBeVisible();
  await expect(nextPageButton.getByRole('button', { name: /^(末页|Last page)$/ })).toBeVisible();
  await expect(pager.locator('.arco-pagination-item-active')).toContainText('1');
  await expect(firstPageButton).toBeVisible();
  await expect(lastPageButton).toBeVisible();
  await expect(firstPageButton).toBeDisabled();
  await expect(prevPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeEnabled();

  await lastPageButton.click();
  await expect
    .poll(async () => pager.locator('.arco-pagination-item-active').innerText())
    .toBe(String(totalPages));
  await expect(firstPageButton).toBeEnabled();
  await expect(prevPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeDisabled();

  await firstPageButton.click();
  await expect
    .poll(async () => pager.locator('.arco-pagination-item-active').innerText())
    .toBe('1');
  await expect(firstPageButton).toBeDisabled();
  await expect(prevPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeEnabled();
});

test('login-log governance smoke: single-page data keeps boundary pager controls disabled', async ({
  page,
}) => {
  const pageSize = 10;
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/login-log\/list(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 920001,
              username: adminCredentials.username,
              ipaddr: '127.0.0.1',
              loginLocation: '本地',
              browser: 'Chrome',
              os: 'Windows',
              status: 1,
              msg: '',
              loginTime: '2026-05-20 09:00:00',
            },
          ],
          total: 1,
          page: 1,
          pageSize,
        },
      }),
    });
  });

  await page.goto('/system/login-log', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '登录日志');
  await expectNoPageError(page);

  const pager = page.locator('.system-list__table').first();
  await expect(pager.locator('.arco-pagination')).toBeVisible();
  await expect(pager.getByRole('button', { name: /^(首页|First page)$/ })).toHaveCount(0);
  await expect(pager.getByRole('button', { name: /^(末页|Last page)$/ })).toHaveCount(0);
});

test('security-center smoke: client-side AppTable pagination exposes shared boundary controls', async ({
  page,
}) => {
  const sessions = Array.from({ length: 6 }, (_, index) => ({
    sessionId: `client-session-${index + 1}`,
    isCurrent: index === 0,
    lastIp: `127.0.0.${index + 1}`,
    browser: 'Chrome',
    os: 'Windows',
    device: `Device ${index + 1}`,
    userAgent: 'Mozilla/5.0',
    refreshExpiresAt: '2026-05-21 09:00:00',
    lastRefreshAt: '2026-05-20 09:00:00',
    lastActivityAt: '2026-05-20 09:00:00',
    createdAt: `2026-05-${String(index + 1).padStart(2, '0')} 08:00:00`,
  }));

  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/auth\/security$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          user: {
            id: 1,
            username: adminCredentials.username,
            nickname: '管理员',
          },
          currentSession: sessions[0],
          activeSessionCount: sessions.length,
          lastLoginAt: '2026-05-20 09:00:00',
          passwordExpired: false,
          recentSecurityEvents: [],
          policy: {
            passwordMinLength: 8,
            passwordRequireDigit: true,
            passwordRequireUpper: false,
            passwordHistoryLimit: 3,
            passwordExpireDays: 90,
            maxFailedAttempts: 5,
            lockMinutes: 30,
            sourceMaxFailedAttempts: 5,
            sourceWindowMinutes: 15,
            sourceLockMinutes: 30,
            sessionIdleMinutes: 30,
            maxActiveSessions: 10,
            sessionRetentionDays: 30,
            captchaEnabled: true,
            mfaEnabled: false,
            ssoEnabled: false,
          },
        },
      }),
    });
  });
  await page.route(/\/api\/v1\/auth\/sessions$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: sessions,
      }),
    });
  });
  await page.route(/\/api\/v1\/auth\/login-logs(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [],
          total: 0,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });

  await page.goto('/auth/security', { waitUntil: 'networkidle' });
  await expectVisiblePageTitle(page, '安全中心');
  await expectNoPageError(page);
  await expect(page.locator('.page-split-layout--with-rail')).toBeVisible();
  await expect(
    page
      .locator('.page-main-column .arco-card')
      .filter({ hasText: /在线会话|Active Sessions/ })
      .first(),
  ).toBeVisible();

  const sessionCard = page
    .locator('.page-panel')
    .filter({ hasText: /在线会话|Active Sessions/ })
    .first();
  const firstPageButton = sessionCard.getByRole('button', { name: /^(首页|First page)$/ });
  const lastPageButton = sessionCard.getByRole('button', { name: /^(末页|Last page)$/ });
  const prevPageButton = sessionCard.locator('.arco-pagination-item-prev');
  const nextPageButton = sessionCard.locator('.arco-pagination-item-next');
  await expect(prevPageButton.getByRole('button', { name: /^(首页|First page)$/ })).toBeVisible();
  await expect(nextPageButton.getByRole('button', { name: /^(末页|Last page)$/ })).toBeVisible();
  await expect(firstPageButton).toBeVisible();
  await expect(lastPageButton).toBeVisible();
  await expect(firstPageButton).toBeDisabled();
  await expect(prevPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(nextPageButton).not.toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeEnabled();

  await lastPageButton.click();
  await expect
    .poll(async () => sessionCard.locator('.arco-pagination-item-active').innerText())
    .toBe('2');
  await expect(nextPageButton).toHaveClass(/arco-pagination-item-disabled/);
  await expect(lastPageButton).toBeDisabled();
  await expect(firstPageButton).toBeEnabled();
});

test('operation-log governance smoke: selecting rows enables batch delete affordance', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.goto('/system/operation-log', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '操作日志');
  await expectNoPageError(page);

  const batchDeleteButton = page.getByRole('button', { name: '删除所选' });
  await expect(
    page.locator('.table-batch-action-bar').getByRole('button', { name: '导出' }),
  ).toBeVisible();
  await expect(batchDeleteButton).toBeDisabled();
  await page.locator('.system-list__table .arco-checkbox').nth(1).click({ force: true });
  await expect(batchDeleteButton).toBeEnabled();
});

test('user governance smoke: cross-page selection keeps the full selected set', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const userPrefix = `smoke_cross_page_${now}`;
  const password = 'ChangeMe123';
  const roleId = (await getFirstActiveRole(page, accessToken)).id;
  const usernames = Array.from(
    { length: 11 },
    (_, index) => `${userPrefix}_${String(index + 1).padStart(2, '0')}`,
  );

  try {
    for (const username of usernames) {
      await deleteUserByUsername(page, accessToken, username);
      await createUserByApi(page, accessToken, {
        username,
        password,
        nickname: username,
        email: `${username}@example.com`,
        roleIds: [roleId],
      });
    }

    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await expectPageIdentityReady(page, '用户管理');
    await expectNoPageError(page);

    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(userPrefix);
      await Promise.all([
        waitForOkApiResponse(
          page,
          (response) =>
            response.url().includes('/system/user/list') &&
            decodeURIComponent(response.url()).includes(`keyword=${userPrefix}`) &&
            response.request().method() === 'GET',
        ),
        toolbarKeyword.press('Enter'),
      ]);
    }

    const pager = page.locator('.system-user-list__table');
    const selectedText = page.locator('.table-batch-action-bar__meta');
    await expect(
      pager.locator('.arco-pagination-item').filter({ hasText: /^2$/ }).first(),
    ).toBeVisible();

    const firstPageCheckbox = pager.locator('.arco-checkbox').nth(1);
    await firstPageCheckbox.click({ force: true });
    await expect(selectedText).toContainText('已选 1 条');

    const secondPageButton = pager
      .locator('.arco-pagination-item')
      .filter({ hasText: /^2$/ })
      .first();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${userPrefix}`) &&
          decodeURIComponent(response.url()).includes('page=2') &&
          response.request().method() === 'GET',
      ),
      secondPageButton.click(),
    ]);
    await expect
      .poll(async () => {
        return pager.locator('.arco-pagination-item-active').innerText();
      })
      .toBe('2');
    await expect(selectedText).toContainText('已选 1 条');

    const secondPageCheckbox = pager.locator('.arco-checkbox').nth(1);
    await secondPageCheckbox.click({ force: true });
    await expect(selectedText).toContainText('已选 2 条');

    const firstPageButton = pager
      .locator('.arco-pagination-item')
      .filter({ hasText: /^1$/ })
      .first();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${userPrefix}`) &&
          decodeURIComponent(response.url()).includes('page=1') &&
          response.request().method() === 'GET',
      ),
      firstPageButton.click(),
    ]);
    await expect
      .poll(async () => {
        return pager.locator('.arco-pagination-item-active').innerText();
      })
      .toBe('1');
    await expect(selectedText).toContainText('已选 2 条');
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-cross-page-selection', async () => {
      for (const username of usernames) {
        await deleteUserByUsername(page, accessToken, username);
      }
    });
  }
});

test('user smoke: edit and detail work through the UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const deptName = `烟测用户部门-${now}`;
  const username = `smoke_user_${now}`;
  const nickname = `烟测用户${now}`;
  const nextNickname = `${nickname}-已编辑`;
  const email = `smoke-user-${now}@example.com`;
  const nextEmail = `smoke-user-${now}-updated@example.com`;
  const password = 'ChangeMe123';
  const deptTree = await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' });
  const rootDept = flattenDeptTreeNodes(deptTree).find(
    (item) => item.isRoot || item.parentId === 0,
  );
  expect(rootDept).toBeTruthy();

  await deleteUserByUsername(page, accessToken, username);
  await deleteDeptByName(page, accessToken, deptName);

  try {
    const dept = await createDeptByApi(page, accessToken, {
      parentId: rootDept!.id,
      deptName,
      sort: 12,
      email: `user-dept-${now}@example.com`,
      phone: '13800000003',
    });
    await createUserByApi(page, accessToken, {
      username,
      password,
      nickname,
      email,
      deptId: dept.id,
      roleIds: [(await getFirstActiveRole(page, accessToken)).id],
    });

    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '用户管理');
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(username);
      await toolbarKeyword.press('Enter');
    }
    const userRow = page.getByRole('row', { name: new RegExp(username) }).first();
    await expect(userRow).toBeVisible();
    await expect(userRow).toContainText(nickname);

    await userRow.getByRole('button', { name: '编辑' }).click();
    const editDialog = page.getByRole('dialog').filter({ hasText: '编辑用户' });
    await expect(editDialog).toBeVisible();
    await editDialog.getByRole('textbox').nth(1).fill(nextNickname);
    await editDialog.getByRole('textbox').nth(2).fill(nextEmail);
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/') && response.request().method() === 'PUT',
      ),
      editDialog.locator('.submit-bar').getByRole('button', { name: '保存' }).click(),
    ]);
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.nickname)
      .toBe(nextNickname);
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.email)
      .toBe(nextEmail);
    await expect(userRow).toContainText(nextNickname);
    await expect(userRow).toContainText(nextEmail);

    await userRow.getByRole('button', { name: '详情' }).click();
    const detailDialog = page.getByRole('dialog').filter({ hasText: username });
    await expect(detailDialog).toBeVisible();
    await expect(detailDialog.getByText(nextNickname).first()).toBeVisible();
    await expect(detailDialog.getByText(nextEmail).first()).toBeVisible();
    await page.keyboard.press('Escape');
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-smoke-edit-detail', async () => {
      await deleteUserByUsername(page, accessToken, username);
      await deleteDeptByName(page, accessToken, deptName);
    });
  }
});

test('user and role smoke: role binding can be deferred to role management and removed there', async ({
  page,
}) => {
  test.setTimeout(120000);
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  const accessToken = login.accessToken;
  const now = Date.now();
  const username = `smoke_roleless_${now}`;
  const nickname = `待分配角色用户${now}`;
  const password = 'ChangeMe123';
  const roleKey = `role_member_smoke_${now}`;
  const roleName = `角色成员烟测${now}`;
  let createdUserId: number | null = null;
  let createdRoleId: number | null = null;
  const verifiedAuthHeaders = await verifiedApiHeaders(page.request, login);

  await deleteUserByUsername(page, accessToken, username);
  await deleteRoleByKey(page, accessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: verifiedAuthHeaders,
      data: {
        roleName,
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: [],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    const role = await getRoleByKey(page, accessToken, roleKey);
    expect(role).toBeTruthy();
    createdRoleId = role?.id ?? null;

    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '用户管理');
    await page.getByRole('button', { name: '新增' }).click();
    const createDialog = page.getByRole('dialog').filter({ hasText: '新增用户' });
    await expect(createDialog).toBeVisible();
    await expect(
      createDialog.locator('.system-user-list__role-field').first().locator('strong'),
    ).toHaveCount(0);
    await createDialog.locator('.submit-bar').getByRole('button', { name: '取消' }).click();
    await expect(createDialog).toHaveCount(0);

    createdUserId = (
      await createUserByApi(page, accessToken, {
        username,
        password,
        nickname,
        roleIds: [],
      })
    ).id;
    expect(createdUserId).toBeTruthy();

    const userDetailBeforeResponse = await page.request.get(
      `${apiBaseUrl}/system/user/${createdUserId}`,
      {
        headers: authHeaders(accessToken),
      },
    );
    expect(userDetailBeforeResponse.ok()).toBeTruthy();
    const userDetailBeforePayload = await userDetailBeforeResponse.json();
    expect(userDetailBeforePayload.data.roleIds).toEqual([]);

    await page.goto('/system/role', { waitUntil: 'networkidle' });
    await expectVisiblePageTitle(page, '角色管理');
    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(roleKey);
      await toolbarKeyword.press('Enter');
    }
    const roleRow = page.getByRole('row', { name: new RegExp(roleName) }).first();
    await expect(roleRow).toBeVisible();
    await roleRow.getByRole('button', { name: '角色成员' }).click();

    const memberDrawer = page.locator('.role-member-drawer');
    await expect(memberDrawer).toBeVisible();
    const candidateResponse = await page.request.get(
      `${apiBaseUrl}/system/role/${role!.id}/user-candidates`,
      {
        headers: authHeaders(accessToken),
        params: { keyword: username, page: '1', pageSize: '20' },
      },
    );
    expect(candidateResponse.ok()).toBeTruthy();
    const candidatePayload = await candidateResponse.json();
    expect(
      Array.isArray(candidatePayload.data?.items) &&
        candidatePayload.data.items.some(
          (item: { username: string }) => item.username === username,
        ),
    ).toBeTruthy();
    await memberDrawer.getByPlaceholder('搜索并选择待加入该角色的用户').fill(username);
    const candidateChip = memberDrawer
      .locator('.role-member-drawer__candidate-pill')
      .filter({ hasText: new RegExp(`${username}|${nickname}`) })
      .first();
    await expect(candidateChip).toBeVisible();
    await candidateChip.click({ force: true });
    await expect(memberDrawer.getByRole('button', { name: '新增' })).toBeEnabled();
    await memberDrawer.getByRole('button', { name: '新增' }).click();
    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/user/${createdUserId}`, {
          headers: authHeaders(accessToken),
        });
        const payload = await response.json();
        return payload.data.roleIds;
      })
      .toEqual([role!.id]);
    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/role/${role!.id}/users`, {
          headers: authHeaders(accessToken),
          params: { keyword: username, page: 1, pageSize: 10 },
        });
        const payload = await response.json();
        const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
        return items.some((item: { username: string }) => item.username === username);
      })
      .toBeTruthy();
    await memberDrawer.getByPlaceholder('按用户名或昵称搜索当前成员').fill(username);
    await memberDrawer.getByRole('button', { name: '搜索' }).click();
    const memberRow = await waitForVisibleRowByText(memberDrawer, username, '删除');
    await clickVisibleRowAction(page, memberRow, '删除', '确认移除该成员的当前角色绑定？');
    await clickVisibleConfirmButton(page, '确认移除该成员的当前角色绑定？');
    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/user/${createdUserId}`, {
          headers: authHeaders(accessToken),
        });
        const payload = await response.json();
        return payload.data.roleIds;
      })
      .toEqual([]);
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-role-membership-governance', async () => {
      await deleteUserByIdWithHeaders(page, verifiedAuthHeaders, createdUserId);
      await deleteRoleByIdWithHeaders(page, verifiedAuthHeaders, createdRoleId);
    });
  }
});

test('user smoke: batch disable enable and delete stay stable through the UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const username = `smoke_user_batch_${now}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, accessToken, username);

  try {
    await createUserByApi(page, accessToken, {
      username,
      password,
      nickname: `批量烟测用户${now}`,
      email: `smoke-user-batch-${now}@example.com`,
      roleIds: [(await getFirstActiveRole(page, accessToken)).id],
    });

    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '用户管理');
    const tableCard = page.locator('.system-list__table-card');
    const userTable = page.locator('.system-user-list__table');
    const selectedText = page.locator('.table-batch-action-bar__meta');
    await expect(tableCard).toBeVisible({ timeout: 30000 });

    const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
    await toolbarKeyword.fill(username);
    const waitForFilteredUserList = () =>
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${username}`) &&
          response.request().method() === 'GET',
      );
    const selectBatchUser = async () => {
      const rowCheckbox = userTable.locator('.arco-checkbox').nth(1);
      await expect(rowCheckbox).toBeVisible();
      await rowCheckbox.click({ force: true });
      await expect(selectedText).toContainText('已选 1 条');
    };

    await Promise.all([waitForFilteredUserList(), toolbarKeyword.press('Enter')]);
    await selectBatchUser();

    await page.getByRole('button', { name: '批量禁用' }).click();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/batch-status') &&
          response.request().method() === 'POST',
      ),
      clickVisibleConfirmButton(page),
    ]);
    await waitForFilteredUserList();
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.status)
      .toBe(2);
    await expect(selectedText).toContainText('已选 0 条');

    await selectBatchUser();
    await page.getByRole('button', { name: '批量启用' }).click();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/batch-status') &&
          response.request().method() === 'POST',
      ),
      clickVisibleConfirmButton(page),
    ]);
    await waitForFilteredUserList();
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.status)
      .toBe(1);
    await expect(selectedText).toContainText('已选 0 条');

    await selectBatchUser();
    await page.getByRole('button', { name: '删除所选' }).click();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/batch-delete') &&
          response.request().method() === 'POST',
      ),
      clickVisibleConfirmButton(page),
    ]);
    await waitForFilteredUserList();
    await expect
      .poll(async () => await findUserByUsername(page, accessToken, username))
      .toBeUndefined();
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-smoke-batch', async () => {
      await deleteUserByUsername(page, accessToken, username);
    });
  }
});

test('dept smoke: create dialog and root edit are reachable through the UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  await page.goto('/system/dept', { waitUntil: 'networkidle' });
  await installOperationToken(page, accessToken);
  await expectVisiblePageTitle(page, '部门管理');
  await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

  await page.getByRole('button', { name: '新增' }).click();
  const createDialog = page.getByRole('dialog').filter({ hasText: '新增部门' });
  await expect(createDialog).toBeVisible();
  await expect(createDialog.getByText('Pantheon Base').first()).toBeVisible();
  await expect(createDialog.getByRole('textbox').first()).toBeVisible();
  await createDialog.locator('.submit-bar').getByRole('button', { name: '取消' }).click();
  await expect(createDialog).not.toBeVisible();

  const rootRow = page.getByRole('row', { name: /Pantheon Base/ }).first();
  await rootRow.getByRole('button', { name: '编辑' }).click();
  const editDialog = page.getByRole('dialog').filter({ hasText: '编辑部门' });
  await expect(editDialog).toBeVisible();
  await expect(editDialog.getByRole('textbox').first()).toHaveValue('Pantheon Base');
  await expect(editDialog.getByText('启用').first()).toBeVisible();
  await page.keyboard.press('Escape');
});

test('dept smoke: blocked delete through API is covered', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const deptName = `烟测部门-${now}`;
  const postCode = `SMOKE_DEPT_POST_${now}`;
  const postName = `烟测部门岗位-${now}`;
  const deptTree = await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' });
  const rootDept = flattenDeptTreeNodes(deptTree).find(
    (item) => item.isRoot || item.parentId === 0,
  );
  expect(rootDept).toBeTruthy();

  await deletePostByCode(page, accessToken, postCode);
  await deleteDeptByName(page, accessToken, deptName);

  try {
    const createdDept = await createDeptByApi(page, accessToken, {
      parentId: rootDept!.id,
      deptName,
      sort: 11,
      email: `dept-${now}@example.com`,
      phone: '021-12345678',
    });

    await createPostByApi(page, accessToken, {
      deptId: createdDept.id,
      postCode,
      postName,
      sort: 10,
      remark: 'dept smoke',
    });

    const deleteResponse = await page.request.delete(
      `${apiBaseUrl}/system/dept/${createdDept.id}`,
      {
        headers: await verifiedHeaders(page, accessToken),
      },
    );
    expect(deleteResponse.ok()).toBeTruthy();
    const deletePayload = await deleteResponse.json();
    expect(deletePayload.code).not.toBe(200);
    expect(deletePayload.message).toBe('dept.delete.error.has_posts');
  } finally {
    await runOptionalSmokeCleanup('system-pages:dept-smoke-blocked-delete', async () => {
      await deletePostByCode(page, accessToken, postCode);
      await deleteDeptByName(page, accessToken, deptName);
    });
  }
});

test('post smoke: edit through UI and blocked delete through API are covered', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const deptName = `烟测岗位部门-${now}`;
  const postCode = `SMOKE_POST_${now}`;
  const postName = `烟测岗位-${now}`;
  const username = `smoke_post_user_${now}`;
  const password = 'ChangeMe123';
  const role = await getFirstActiveRole(page, accessToken);
  const deptTree = await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' });
  const rootDept = flattenDeptTreeNodes(deptTree).find(
    (item) => item.isRoot || item.parentId === 0,
  );
  expect(rootDept).toBeTruthy();

  await deleteUserByUsername(page, accessToken, username);
  await deletePostByCode(page, accessToken, postCode);
  await deleteDeptByName(page, accessToken, deptName);

  try {
    const dept = await createDeptByApi(page, accessToken, {
      parentId: rootDept!.id,
      deptName,
      sort: 10,
      email: `post-dept-${now}@example.com`,
      phone: '13800000002',
    });
    const createdPost = await createPostByApi(page, accessToken, {
      deptId: dept.id,
      postCode,
      postName,
      sort: 21,
      remark: 'post smoke',
    });

    await page.goto('/system/post', { waitUntil: 'networkidle' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '岗位管理');
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(postCode);
      await toolbarKeyword.press('Enter');
    }
    const postRow = page.getByRole('row', { name: new RegExp(postCode) }).first();
    await expect(postRow).toBeVisible();
    await expect(postRow).toContainText(postName);

    await postRow.getByRole('button', { name: '编辑' }).click();
    const editDialog = page.getByRole('dialog').filter({ hasText: '编辑岗位' });
    await expect(editDialog).toBeVisible();
    await expect(editDialog.getByRole('textbox').first()).toHaveValue(postCode);
    await expect(editDialog.getByRole('textbox').nth(1)).toHaveValue(postName);
    await page.keyboard.press('Escape');

    const post = await findPostByCode(page, accessToken, postCode);
    expect(post).toBeTruthy();
    await createUserByApi(page, accessToken, {
      username,
      password,
      nickname: `岗位占用用户${now}`,
      roleIds: [role.id],
      deptId: dept.id,
      postId: post!.id,
      email: `post-user-${now}@example.com`,
    });

    const deleteResponse = await page.request.delete(
      `${apiBaseUrl}/system/post/${createdPost.id}`,
      {
        headers: await verifiedHeaders(page, accessToken),
      },
    );
    expect(deleteResponse.ok()).toBeTruthy();
    const deletePayload = await deleteResponse.json();
    expect(deletePayload.code).not.toBe(200);
    expect(deletePayload.message).toBe('post.delete.error.has_users');
  } finally {
    await runOptionalSmokeCleanup('system-pages:post-smoke-edit-blocked-delete', async () => {
      await deleteUserByUsername(page, accessToken, username);
      await deletePostByCode(page, accessToken, postCode);
      await deleteDeptByName(page, accessToken, deptName);
    });
  }
});

test('session governance smoke: revocation uses the shared table batch bar', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/session\/list(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              sessionId: 'smoke-self-session',
              userId: 1,
              username: adminCredentials.username,
              nickname: 'Admin',
              lastIp: '127.0.0.1',
              browser: 'Chrome',
              os: 'Windows',
              device: 'Desktop',
              userAgent: 'Chrome/136',
              refreshExpiresAt: '2026-05-21T10:00:00+08:00',
              lastRefreshAt: '2026-05-20T09:30:00+08:00',
              lastActivityAt: '2026-05-20T09:30:00+08:00',
              revokedAt: null,
              createdAt: '2026-05-20T08:00:00+08:00',
            },
            {
              sessionId: 'smoke-target-session',
              userId: 2,
              username: 'other-admin',
              nickname: 'Other Admin',
              lastIp: '10.0.0.8',
              browser: 'Edge',
              os: 'Windows',
              device: 'Desktop',
              userAgent: 'Edge/136',
              refreshExpiresAt: '2026-05-21T10:00:00+08:00',
              lastRefreshAt: '2026-05-20T08:30:00+08:00',
              lastActivityAt: '2026-05-20T08:35:00+08:00',
              revokedAt: null,
              createdAt: '2026-05-20T07:00:00+08:00',
            },
          ],
          total: 2,
          activeCount: 2,
          revokedCount: 0,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });
  await page.goto('/system/session', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '会话管理');
  await expectNoPageError(page);

  // 治理摘要：迁移到 GovernanceSummaryBar（eyebrow + title 两行）。
  // 与基线 selector 保持一致（shell-visual-contract / governance-insight-drawer），
  // 避免 Arco <Space> 包裹打断 `.system-page-template >` 的直系选择器。
  const summaryBar = page.locator('.governance-summary-bar').first();
  await expect(summaryBar).toBeVisible();
  await expect(summaryBar.locator('.governance-summary-bar__title-row')).toBeVisible();

  // 批量下线 + 手动清理共用 GovernanceCleanupBar（--governance 修饰；
  // 2026-07-20 维护者决策恢复受控手动清理入口，自动保留策略仍为主）
  const batchBar = page.locator('.page-panel.system-list__table-card .table-batch-action-bar').first();
  await expect(batchBar).toBeVisible();
  await expect(batchBar).toHaveClass(/table-batch-action-bar--governance/);
  const revokeSelectedButton = batchBar.getByRole('button', { name: '下线所选' });
  await expect(revokeSelectedButton).toBeDisabled();

  // 手动清理入口存在且受权限保护（admin 会话可见）
  await expect(batchBar.getByRole('button', { name: '清理历史会话', exact: true })).toBeVisible();

  // 选中一行后，批量下线按钮启用
  await page.locator('.app-table tbody .arco-checkbox').nth(1).click({ force: true });
  await expect(revokeSelectedButton).toBeEnabled();
});

test('security-event governance smoke: pending event can be acknowledged with a note', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/system\/security-event\/list(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          items: [
            {
              id: 9001,
              userId: 7,
              username: 'risk-user',
              eventType: 'source_blocked',
              severity: 'high',
              sourceKey: 'ip:10.0.0.9',
              ip: '10.0.0.9',
              userAgent: 'Chrome',
              messageKey: 'auth.security.event.source_blocked',
              metadata: '{}',
              createdAt: '2026-05-20T10:00:00+08:00',
              acknowledgedAt: null,
              acknowledgedBy: 0,
              acknowledgedByUser: '',
              acknowledgementNote: '',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 10,
        },
      }),
    });
  });

  let acknowledgePayload: Record<string, unknown> | null = null;
  await page.route(/\/api\/v1\/system\/security-event\/9001\/acknowledge$/, async (route) => {
    acknowledgePayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 200, data: { acknowledged: true } }),
    });
  });

  await page.goto('/system/security-event', { waitUntil: 'networkidle' });
  await expectPageIdentityReady(page, '安全事件');
  await expectNoPageError(page);

  await page.getByRole('button', { name: '确认事件' }).click();
  const dialog = page.getByRole('dialog');
  await dialog.getByRole('textbox').fill('来源已复核，风险已处置');
  await dialog.locator('.arco-btn-primary').last().click();

  await expect.poll(() => acknowledgePayload).not.toBeNull();
  expect(acknowledgePayload).toMatchObject({
    acknowledgementNote: '来源已复核，风险已处置',
  });
});

test('refresh sync smoke: setting page auto-updates across isolated contexts', async ({
  browser,
  page,
}) => {
  test.setTimeout(45000);
  const adminLogin = await createSharedAdminLogin(page);
  const accessToken = adminLogin.accessToken;
  const groupResponse = await page.request.get(`${apiBaseUrl}/system/setting/group/basic`, {
    headers: authHeaders(accessToken),
  });
  expect(groupResponse.ok()).toBeTruthy();
  const groupPayload = await groupResponse.json();
  expect(groupPayload.code).toBe(200);
  const originalItems = groupPayload.data.items as SettingItem[];
  const originalSiteName =
    originalItems.find((item) => item.settingKey === 'site.name')?.settingValue || 'Pantheon Base';
  const nextSiteName = `Pantheon Sync ${Date.now()}`;
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'site.name' ? nextSiteName : item.settingValue,
  }));

  const syncContext = await browser.newContext();
  const syncPage = await syncContext.newPage();

  try {
    await installClientSession(syncPage, adminLogin);
    const refreshBootstrap = waitForRefreshBootstrap(syncPage);
    await syncPage.goto('/system/setting/basic', { waitUntil: 'networkidle' });
    const siteNameInput = formItem(syncPage, '站点名称').locator('input').first();
    await expect(siteNameInput).toHaveValue(originalSiteName);
    await refreshBootstrap;

    const updateResponse = await updateSettingGroup(page, accessToken, 'basic', nextItems);
    expect(updateResponse.ok()).toBeTruthy();
    const updatePayload = await updateResponse.json();
    expect(updatePayload.code).toBe(200);

    await expect(siteNameInput).toHaveValue(nextSiteName, { timeout: 15000 });
  } finally {
    await updateSettingGroup(page, accessToken, 'basic', originalItems);
    await closeExtraBrowserContext(syncContext);
  }
});

test('refresh sync smoke: dict page auto-updates across isolated contexts', async ({
  browser,
  page,
}) => {
  test.setTimeout(45000);
  const adminLogin = await createSharedAdminLogin(page);
  const accessToken = adminLogin.accessToken;
  const dictCode = `system_sync_${Date.now()}`;
  const dictName = `system.dict.sync.${Date.now()}`;

  const syncContext = await browser.newContext();
  const syncPage = await syncContext.newPage();

  try {
    await installClientSession(syncPage, adminLogin);
    const refreshBootstrap = waitForRefreshBootstrap(syncPage);
    await syncPage.goto('/system/dict', { waitUntil: 'networkidle' });
    {
      const toolbarKeyword = syncPage.locator('.search-toolbar').getByPlaceholder(/搜索/).first();
      await toolbarKeyword.fill(dictCode);
      await toolbarKeyword.press('Enter');
    }
    await expect(syncPage.getByText(dictCode, { exact: false })).toHaveCount(0);
    await refreshBootstrap;

    const createResponse = await page.request.post(`${apiBaseUrl}/system/dict/type`, {
      headers: await verifiedHeaders(page, accessToken),
      data: {
        dictCode,
        dictName,
        module: 'system',
        status: 1,
        remark: 'sync smoke',
      },
    });
    expect(createResponse.ok()).toBeTruthy();
    const createPayload = await createResponse.json();
    expect(createPayload.code).toBe(200);
    const createdId = createPayload.data.id as number;

    await expect(syncPage.getByText(dictCode, { exact: false }).first()).toBeVisible({
      timeout: 15000,
    });

    await page.request
      .delete(`${apiBaseUrl}/system/dict/type/${createdId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  } finally {
    await closeExtraBrowserContext(syncContext);
  }
});

test('refresh sync smoke: i18n page auto-updates across isolated contexts', async ({
  browser,
  page,
}) => {
  test.setTimeout(45000);
  const adminLogin = await createSharedAdminLogin(page);
  const accessToken = adminLogin.accessToken;
  const i18nKey = `i18n.sync.${Date.now()}`;

  const syncContext = await browser.newContext();
  const syncPage = await syncContext.newPage();

  try {
    await installClientSession(syncPage, adminLogin);
    await syncPage.goto('/system/i18n', { waitUntil: 'networkidle' });
    {
      const toolbarKeyword = syncPage.locator('.search-toolbar').getByPlaceholder(/搜索/).first();
      await toolbarKeyword.fill(i18nKey);
      await toolbarKeyword.press('Enter');
    }
    await expect(syncPage.getByText(i18nKey, { exact: false })).toHaveCount(0);

    const createResponse = await page.request.post(`${apiBaseUrl}/system/i18n`, {
      headers: await verifiedHeaders(page, accessToken),
      data: {
        module: 'system.config',
        group: 'messages',
        key: i18nKey,
        locale: 'zh-CN',
        value: '跨上下文同步',
        remark: 'sync smoke',
      },
    });
    expect(createResponse.ok()).toBeTruthy();
    const createPayload = await createResponse.json();
    expect(createPayload.code).toBe(200);
    const createdId = createPayload.data.id as number;

    await expect(syncPage.getByText(i18nKey, { exact: false }).first()).toBeVisible({
      timeout: 15000,
    });

    await page.request
      .delete(`${apiBaseUrl}/system/i18n/${createdId}`, {
        headers: await verifiedHeaders(page, accessToken),
      })
      .catch(() => undefined);
  } finally {
    await closeExtraBrowserContext(syncContext);
  }
});
