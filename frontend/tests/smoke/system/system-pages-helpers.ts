// Shared scaffolding extracted from the former system-pages.spec.ts monolith.
// Non-spec module consumed by the per-module system smoke specs.
import syncFs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { expect, type BrowserContext, type Locator, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  type BrowserLoginResult,
  installClientSession,
  loginByApi,
  requestHeaders,
  signInAsAdmin,
  verifiedHeaders,
} from '../helpers/auth';
import { runOptionalSmokeCleanup } from '../helpers/fixture-policy';
import { installSharedPageReadCache, type CachedReadResponse } from '../helpers/shared-read-cache';
import { listRegisteredComponentKeys } from '../../../src/core/router/componentRegistry';
export { expectPagePathname } from '../helpers/url-pattern';
export { installOperationToken, signInWithUi, verifiedApiHeaders } from '../helpers/auth';
export {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  installClientSession,
  loginByApi,
  path,
  requestHeaders,
  runOptionalSmokeCleanup,
  signInAsAdmin,
  verifiedHeaders,
};

export const pageErrorTitles = ['加载失败', '网络异常', '请求超时'];
export const pageEmptyTexts = [
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
export const reactElementRefWarningPattern = /Accessing element\.ref was removed in React 19/i;
export const menuComponentKey =
  listRegisteredComponentKeys().find((key) => key.endsWith('/menu/MenuList')) ??
  'system/menu/MenuList';
export type SettingItem = { settingKey: string; settingValue: string };
export type UserPlatformPreferences = {
  theme?: string;
  language?: string;
  layoutMode?: string;
  densityMode?: string;
};
export type ManageMenuNode = {
  id: number;
  parentId: number;
  titleKey: string;
  path: string;
  type: string;
  children?: ManageMenuNode[];
};

export const __filename = fileURLToPath(import.meta.url);
export const __dirname = path.dirname(__filename);
export const admissionConfig = JSON.parse(
  syncFs.readFileSync(
    path.resolve(__dirname, '../../../config/system-page-admission.json'),
    'utf8',
  ),
) as Array<{
  path: string;
  hero: 'allowed' | 'forbidden';
  governanceDrawer: 'allowed' | 'forbidden';
}>;

export const compactMainAreaPages = admissionConfig
  .filter((item) => item.hero === 'forbidden' && item.governanceDrawer === 'allowed')
  .map((item) => item.path)
  .filter((item) => !item.includes(':'));

export const pageIdentitySelectors = [
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

export const reloadStableReadPaths = new Set([
  '/api/v1/system/setting/public',
  '/api/v1/system/menu/tree',
  '/api/v1/system/dept/tree',
  '/api/v1/system/dict/type/list',
  '/api/v1/system/post/list',
]);

export async function installReloadStableReadCache(page: Page) {
  return installSharedPageReadCache(page, new Map<string, CachedReadResponse>(), {
    shouldHandleRequest: ({ method, url }) =>
      method === 'GET' && reloadStableReadPaths.has(url.pathname),
  });
}

export const systemPages = [
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

export const workspacePages = [
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

export async function updateSettingGroup(
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

export async function waitForRefreshBootstrap(page: Page) {
  await page.waitForResponse(
    (response) =>
      response.url().includes('/system/refresh/state') &&
      response.request().method() === 'GET' &&
      response.ok(),
    { timeout: 15000 },
  );
}

export async function closeExtraBrowserContext(context: BrowserContext) {
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

export function isIgnorableConsoleError(text: string) {
  return (
    text.includes('Failed to load resource: the server responded with a status of 404') ||
    reactElementRefWarningPattern.test(text)
  );
}

export async function createSharedAdminLogin(page: Page): Promise<BrowserLoginResult> {
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  return login;
}

export async function openUserMenu(page: Page) {
  await page.getByRole('button').and(page.locator('[aria-haspopup="menu"]')).click();
}

export function waitForOkApiResponse(
  page: Page,
  matcher: (
    response: Parameters<Page['waitForResponse']>[0] extends (arg: infer T) => boolean ? T : never,
  ) => boolean,
) {
  return page.waitForResponse((response) => matcher(response) && response.ok(), { timeout: 15000 });
}

export async function clickVisibleConfirmButton(page: Page, titleText?: string) {
  const confirmDialog = getVisibleConfirmDialog(page, titleText);
  await expect(confirmDialog).toBeVisible();
  await confirmDialog.getByRole('button', { name: '确定', exact: true }).click();
}

export function getVisibleConfirmDialog(page: Page, titleText?: string) {
  const allVisibleConfirmDialogs = page.locator(
    '.app-dialog:visible, .arco-modal:visible, .arco-modal-confirm:visible, .arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-popover:visible, [role="dialog"]:visible, [role="alertdialog"]:visible, [role="tooltip"]:visible',
  );
  return titleText
    ? allVisibleConfirmDialogs.filter({ hasText: titleText }).last()
    : allVisibleConfirmDialogs.last();
}

export async function waitForVisibleConfirmDialog(page: Page, titleText?: string, timeout = 1000) {
  try {
    await expect(getVisibleConfirmDialog(page, titleText)).toBeVisible({ timeout });
    return true;
  } catch {
    return false;
  }
}

export async function findVisibleRowByText(container: Locator, text: string, actionName?: string) {
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

export async function clickVisibleRowAction(
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

export async function waitForVisibleRowByText(
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

export async function dismissVisibleSuccessDialog(page: Page) {
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

export async function deleteUserByUsername(page: Page, accessToken: string, username: string) {
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
      await page.request // NOSONAR — sequential cleanup deletes keep fixture teardown deterministic
        .delete(`${apiBaseUrl}/system/user/${item.id}`, {
          headers: await verifiedHeaders(page, accessToken), // NOSONAR — headers fetched per delete by design
        })
        .catch(() => undefined);
    }
  }
}

export async function findUserByUsername(page: Page, accessToken: string, username: string) {
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

export async function deleteRoleByKey(page: Page, accessToken: string, roleKey: string) {
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
      await page.request // NOSONAR — sequential cleanup deletes keep fixture teardown deterministic
        .delete(`${apiBaseUrl}/system/role/${item.id}`, {
          headers: await verifiedHeaders(page, accessToken), // NOSONAR — headers fetched per delete by design
        })
        .catch(() => undefined);
    }
  }
}

export async function deleteUserByIdWithHeaders(
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

export async function deleteRoleByIdWithHeaders(
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

export async function cleanupViewerIdentity(
  page: Page,
  accessToken: string,
  username: string,
  roleKey: string,
) {
  await deleteUserByUsername(page, accessToken, username);
  await deleteRoleByKey(page, accessToken, roleKey);
}

export type RoleListItem = {
  id: number;
  roleKey: string;
  roleName: string;
  status?: number;
};

export type DeptListItem = {
  id: number;
  parentId: number;
  deptName: string;
  isRoot?: boolean;
  children?: DeptListItem[];
};

export type PostListItem = {
  id: number;
  deptId: number;
  deptName: string;
  postCode: string;
  postName: string;
};

export async function getRoleList(page: Page, accessToken: string, params?: Record<string, unknown>) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(accessToken),
    params: { page: 1, pageSize: 100, ...params },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items as RoleListItem[];
}

export async function getFirstActiveRole(page: Page, accessToken: string) {
  const items = await getRoleList(page, accessToken, {
    status: 1,
    sortField: 'sort',
    sortOrder: 'asc',
  });
  const role = items.find((item) => item.roleKey !== 'guest');
  expect(role).toBeTruthy();
  return role!;
}

export function flattenDeptTreeNodes(nodes: DeptListItem[]): DeptListItem[] {
  return nodes.flatMap((node) => [node, ...flattenDeptTreeNodes(node.children || [])]);
}

export async function getDeptTree(page: Page, accessToken: string, params?: Record<string, unknown>) {
  const response = await page.request.get(`${apiBaseUrl}/system/dept/tree`, {
    headers: authHeaders(accessToken),
    params,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? (payload.data as DeptListItem[]) : [];
}

export async function deleteDeptByName(page: Page, accessToken: string, deptName: string) {
  const rows = flattenDeptTreeNodes(
    await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' }),
  );
  const targets = rows.filter((item) => item.deptName === deptName && !item.isRoot);
  targets.reverse(); // children before parents so nested depts delete without FK failures
  for (const item of targets) {
    await page.request // NOSONAR — children-first deletion order must stay sequential
      .delete(`${apiBaseUrl}/system/dept/${item.id}`, {
        headers: await verifiedHeaders(page, accessToken), // NOSONAR — headers fetched per delete by design
      })
      .catch(() => undefined);
  }
}

export async function createDeptByApi(
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

export async function getPostListItems(page: Page, accessToken: string, params?: Record<string, unknown>) {
  const response = await page.request.get(`${apiBaseUrl}/system/post/list`, {
    headers: authHeaders(accessToken),
    params: { page: 1, pageSize: 100, ...params },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items as PostListItem[];
}

export async function findPostByCode(page: Page, accessToken: string, postCode: string) {
  const items = await getPostListItems(page, accessToken, { postCode });
  return items.find((item) => item.postCode === postCode);
}

export async function deletePostByCode(page: Page, accessToken: string, postCode: string) {
  const items = await getPostListItems(page, accessToken, { postCode });
  for (const item of items) {
    if (item.postCode === postCode) {
      await page.request // NOSONAR — sequential cleanup deletes keep fixture teardown deterministic
        .delete(`${apiBaseUrl}/system/post/${item.id}`, {
          headers: await verifiedHeaders(page, accessToken), // NOSONAR — headers fetched per delete by design
        })
        .catch(() => undefined);
    }
  }
}

export async function createPostByApi(
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

export async function createUserByApi(
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

export async function fetchManageMenuTree(page: Page, accessToken: string): Promise<ManageMenuNode[]> {
  const response = await page.request.get(`${apiBaseUrl}/system/menu/tree`, {
    headers: authHeaders(accessToken),
    params: { scope: 'manage' },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? (payload.data as ManageMenuNode[]) : [];
}

export function flattenManageMenus(nodes: ManageMenuNode[]): ManageMenuNode[] {
  return nodes.flatMap((node) => [node, ...flattenManageMenus(node.children || [])]);
}

export async function deleteMenuById(page: Page, accessToken: string, menuId: number | null) {
  if (!menuId) {
    return;
  }
  await page.request.delete(`${apiBaseUrl}/system/menu/${menuId}`, {
    headers: await verifiedHeaders(page, accessToken),
  });
}

export async function getRoleByKey(page: Page, accessToken: string, roleKey: string) {
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

export async function createApiPermission(
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
  // Saving a role already syncs the API routes its permission keys declare, so
  // the route this fixture asks for is usually in place before the POST. The
  // backend's duplicate check covers exactly (subject, path, method), so that
  // conflict reports the grant as present rather than a failed grant.
  if (payload.code !== 200) {
    expect(payload.message).toBe('permission.policy.exists');
  }
}

/**
 * Reads one setting group through the API. The returned items are also the
 * restore payload handed to updateSettingGroup in each test's finally block.
 */
export async function fetchSettingGroupItems(
  page: Page,
  accessToken: string,
  group: string,
): Promise<SettingItem[]> {
  const response = await page.request.get(`${apiBaseUrl}/system/setting/group/${group}`, {
    headers: authHeaders(accessToken),
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data.items as SettingItem[];
}

/** Reads a single setting value from a group; used with expect.poll(). */
export async function readSettingValueFromApi(
  page: Page,
  accessToken: string,
  group: string,
  settingKey: string,
): Promise<string | undefined> {
  const items = await fetchSettingGroupItems(page, accessToken, group);
  return items.find((item) => item.settingKey === settingKey)?.settingValue;
}

export async function getCurrentUserPreferences(
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

export async function updateCurrentUserPreferences(
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

export async function expectNoPageError(page: Page) {
  for (const title of pageErrorTitles) {
    const matches = page.getByText(title, { exact: false });
    const count = await matches.count(); // NOSONAR — per-title count must resolve before inner assertions
    for (let i = 0; i < count; i += 1) {
      await expect(matches.nth(i)).not.toBeVisible(); // NOSONAR — auto-retrying assertions run sequentially by design
    }
  }
}

export async function expectPageBodyReady(page: Page) {
  const readySelector = [
    '.arco-table',
    '.arco-empty',
    '.setting-page__group-nav-grid',
    '.setting-page__config-card',
    '.generator-wizard__steps',
  ].join(', '); // keep in sync with the original per-indicator locators

  // Body readiness must retry while the page settles: one-shot count()
  // snapshots race table/empty mounting on slow environments and turn the
  // system-pages smoke loop flaky.
  await expect(page.locator(readySelector).filter({ visible: true }).first()).toBeVisible();

  // When an empty state is on screen, its message must be a known per-page
  // empty text. Sample visibility and text inside toPass so an empty that
  // resolves into a table between sampling and reading text cannot stall the
  // locator on a detached element.
  await expect(async () => {
    const emptyNode = page.locator('.arco-empty').first();
    if (!(await emptyNode.isVisible())) {
      return;
    }
    const emptyText = await emptyNode.innerText();
    expect(pageEmptyTexts.some((text) => emptyText.includes(text))).toBeTruthy();
  }).toPass();
}

export async function expectVisiblePageTitle(page: Page, title: string | RegExp) {
  const visibleMatches = page.getByText(title, { exact: false }).filter({ visible: true });
  await expect(visibleMatches.first()).toBeVisible();
}

export function shellBrandTextLocator(page: Page) {
  return page.locator('.app-shell__brand-title, .app-shell__header-brand-text').first();
}

export async function expectPageIdentityReady(page: Page, title: string | RegExp) {
  await expectVisiblePageTitle(page, title);
  await expect(page.locator(pageIdentitySelectors.join(', ')).first()).toBeVisible();
}

export function formItem(scope: Page | ReturnType<Page['locator']>, label: string) {
  return scope
    .locator('.arco-form-item')
    .filter({ has: scope.getByText(label, { exact: true }) })
    .first();
}

export type ListViewOnlyViewerOptions = {
  /** Prefix for the throwaway role key and username. */
  slug: string;
  /** Cleanup label reported by runOptionalSmokeCleanup. */
  cleanupTag: string;
  roleName: string;
  nickname: string;
  /** The single list permission the role is granted. */
  listPermission: string;
  /** Extra GET endpoints the page needs, granted to the role. */
  apiPermissions: Array<[path: string, method: string]>;
  route: string;
  identityLabel: string;
};

/**
 * Shared workflow for the "list-only role" permission smoke tests: provision a
 * role holding only the list permission, provision a user bound to that role,
 * sign that user in and open the target page. `assertViewer` receives the
 * viewer page and performs the page-specific capability assertions. The viewer
 * page and the provisioned identity are always cleaned up.
 */
export async function withListViewOnlyViewer(
  page: Page,
  options: ListViewOnlyViewerOptions,
  assertViewer: (viewerPage: Page) => Promise<void>,
): Promise<void> {
  const adminAccessToken = await signInAsAdmin(page);
  const roleKey = `${options.slug}_view_only_${Date.now()}`;
  const username = `${options.slug}_viewer_${Date.now()}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, adminAccessToken, username);
  await deleteRoleByKey(page, adminAccessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        roleName: options.roleName,
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: [options.listPermission],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();

    const role = await getRoleByKey(page, adminAccessToken, roleKey);
    expect(role).toBeTruthy();

    for (const [permissionPath, method] of options.apiPermissions) {
      await createApiPermission(page, adminAccessToken, roleKey, permissionPath, method); // NOSONAR — ordered permission seeding; parallel POSTs could race server-side
    }
    await createApiPermission(page, adminAccessToken, roleKey, '/api/v1/system/menu/tree', 'GET');

    const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
      headers: await verifiedHeaders(page, adminAccessToken),
      data: {
        username,
        password,
        nickname: options.nickname,
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
      await viewerPage.goto(options.route, { waitUntil: 'domcontentloaded' });
      await expectPageIdentityReady(viewerPage, options.identityLabel);
      await expectNoPageError(viewerPage);
      await assertViewer(viewerPage);
    } finally {
      await viewerPage.close();
    }
  } finally {
    await runOptionalSmokeCleanup(options.cleanupTag, async () => {
      await cleanupViewerIdentity(page, adminAccessToken, username, roleKey);
    });
  }
}
