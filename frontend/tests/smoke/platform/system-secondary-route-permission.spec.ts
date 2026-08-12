import { test } from '../../fixtures/coverage';
import { expect, type Page } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  installClientSession,
  loginByApi,
  type BrowserLoginResult,
  verifiedApiHeaders,
} from '../helpers/auth';
import { runOptionalSmokeCleanup } from '../helpers/fixture-policy';
import { expectOnlyAllowedRuntimeErrors } from '../helpers/runtime-errors';

const artifactDir = join(process.cwd(), 'test-results', 'backoffice-ui');

const pageErrorTexts = [
  '加载失败',
  '网络异常',
  '请求超时',
  'Load failed',
  'Network error',
  'Request timed out',
];

type ViewportCase = {
  width: number;
  height: number;
  suffix: '-desktop.png' | '-desktop-1280.png';
};

type ProvisionedViewer = {
  username: string;
  roleKey: string;
  password: string;
};

type ViewerSession = {
  viewerPage: Page;
  close: () => Promise<void>;
};

const viewportCases: ViewportCase[] = [
  { width: 1440, height: 900, suffix: '-desktop.png' },
  { width: 1280, height: 900, suffix: '-desktop-1280.png' },
];

async function ensureArtifactDir() {
  if (!existsSync(artifactDir)) {
    mkdirSync(artifactDir, { recursive: true });
  }
}

function collectRuntimeErrors(page: Page) {
  const runtimeErrors: string[] = [];

  page.on('console', (message) => {
    if (message.type() === 'error') {
      runtimeErrors.push(message.text());
    }
  });

  page.on('pageerror', (error) => {
    runtimeErrors.push(error.message);
  });

  return runtimeErrors;
}

async function expectNoPageError(page: Page) {
  for (const text of pageErrorTexts) {
    await expect(page.getByText(text, { exact: false })).toHaveCount(0);
  }
}

async function expectRouteShell(page: Page) {
  await expect(page.locator('.app-shell__header')).toBeVisible();
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(
    page
      .locator(
        '.page-panel, .arco-card, .system-list__work-actions, .setting-overview-page, .setting-group-page',
      )
      .first(),
  ).toBeVisible();
}

async function expectPageIdentity(page: Page, title: string) {
  await expect(page.locator('[role="tab"][aria-selected="true"]').first()).toContainText(title);
  await expect(page.locator('.app-shell__header').getByRole('listitem').last()).toContainText(title);
}

async function expectNoViewportOverflow(page: Page) {
  await expect
    .poll(async () =>
      page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    )
    .toBeLessThanOrEqual(1);
}

async function deleteUserByUsername(page: Page, login: BrowserLoginResult, username: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/user/list`, {
    headers: authHeaders(login.accessToken),
    params: { username, page: 1, pageSize: 20 },
  });
  if (!response.ok()) {
    return;
  }
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const item of items) {
    if (item.username === username) {
      await page.request.delete(`${apiBaseUrl}/system/user/${item.id}`, {
        headers: await verifiedApiHeaders(page.request, login),
      }).catch(() => undefined);
    }
  }
}

async function deleteRoleByKey(page: Page, login: BrowserLoginResult, roleKey: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(login.accessToken),
    params: { roleKey, page: 1, pageSize: 20 },
  });
  if (!response.ok()) {
    return;
  }
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const item of items) {
    if (item.roleKey === roleKey && item.roleKey !== 'admin') {
      await page.request.delete(`${apiBaseUrl}/system/role/${item.id}`, {
        headers: await verifiedApiHeaders(page.request, login),
      }).catch(() => undefined);
    }
  }
}

async function getRoleByKey(page: Page, login: BrowserLoginResult, roleKey: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(login.accessToken),
    params: { roleKey, page: 1, pageSize: 20 },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
  return items.find((item: { roleKey: string }) => item.roleKey === roleKey) as
    | { id: number; roleKey: string }
    | undefined;
}

async function createApiPermission(
  page: Page,
  login: BrowserLoginResult,
  roleKey: string,
  path: string,
  method: string,
) {
  const response = await page.request.post(`${apiBaseUrl}/system/permission`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: { roleKey, path, method },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
}

async function provisionViewer(
  page: Page,
  login: BrowserLoginResult,
  options: {
    roleName: string;
    roleKey: string;
    username: string;
    permissionKeys: string[];
    apiPermissions: Array<{ path: string; method: string }>;
  },
): Promise<ProvisionedViewer> {
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, login, options.username);
  await deleteRoleByKey(page, login, options.roleKey);

  const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: {
      roleName: options.roleName,
      roleKey: options.roleKey,
      sort: 10,
      status: 1,
      menuIds: [],
      permissionKeys: options.permissionKeys,
    },
  });
  expect(createRoleResponse.ok()).toBeTruthy();

  const role = await getRoleByKey(page, login, options.roleKey);
  expect(role).toBeTruthy();

  for (const permission of options.apiPermissions) {
    await createApiPermission(page, login, options.roleKey, permission.path, permission.method);
  }

  const createUserResponse = await page.request.post(`${apiBaseUrl}/system/user`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: {
      username: options.username,
      password,
      nickname: options.roleName,
      status: 1,
      roleIds: [role!.id],
    },
  });
  expect(createUserResponse.ok()).toBeTruthy();
  const payload = await createUserResponse.json();
  expect(payload.code).toBe(200);

  return {
    username: options.username,
    roleKey: options.roleKey,
    password,
  };
}

async function openViewerPage(page: Page, viewer: ProvisionedViewer): Promise<ViewerSession> {
  const viewerTokens = await loginByApi(page.request, {
    username: viewer.username,
    password: viewer.password,
  });
  const browser = page.context().browser();
  expect(browser).toBeTruthy();
  const viewerContext = await browser!.newContext();
  const viewerPage = await viewerContext.newPage();
  await installClientSession(viewerPage, viewerTokens);
  return {
    viewerPage,
    close: async () => {
      await viewerContext.close();
    },
  };
}

test.beforeAll(async () => {
  await ensureArtifactDir();
});

test.describe('system secondary route permission and readonly states', () => {
  for (const viewport of viewportCases) {
    test(`/system/setting/basic keeps readonly permission state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const adminLogin = await loginByApi(page.request, adminCredentials);
      const roleKey = `setting_view_only_route_${viewport.width}_${Date.now()}`;
      const username = `setting_viewer_route_${viewport.width}_${Date.now()}`;
      const viewer = await provisionViewer(page, adminLogin, {
        roleName: '系统设置次级路由只读烟测角色',
        roleKey,
        username,
        permissionKeys: ['system:setting:list'],
        apiPermissions: [
          { path: '/api/v1/system/setting/list', method: 'GET' },
          { path: '/api/v1/system/setting/overview', method: 'GET' },
          { path: '/api/v1/system/menu/tree', method: 'GET' },
        ],
      });
      const viewerSession = await openViewerPage(page, viewer);
      const { viewerPage } = viewerSession;
      const runtimeErrors = collectRuntimeErrors(viewerPage);

      try {
        await viewerPage.setViewportSize({ width: viewport.width, height: viewport.height });
        await viewerPage.goto('/system/setting/basic', { waitUntil: 'networkidle' });

        await expect(viewerPage).toHaveURL(/\/system\/setting\/basic$/);
        await expectRouteShell(viewerPage);
        await expectPageIdentity(viewerPage, '基础信息');
        await expectNoPageError(viewerPage);

        const settingPanel = viewerPage.locator('.setting-page__config-card');
        const saveButton = settingPanel.locator('.submit-bar').getByRole('button', { name: '保存' });
        const cancelButton = settingPanel.locator('.submit-bar').getByRole('button', { name: '取消' });
        const refreshButton = settingPanel.getByRole('button', { name: '刷新设置缓存' });

        await expect(settingPanel).toBeVisible();
        await expect(saveButton).toBeDisabled();
        await expect(cancelButton).toBeEnabled();
        await expect(refreshButton).toBeDisabled();
        await expectNoViewportOverflow(viewerPage);
        await viewerPage.screenshot({
          path: join(artifactDir, `system-setting-readonly-permission${viewport.suffix}`),
          fullPage: true,
        });

        expectOnlyAllowedRuntimeErrors(runtimeErrors);
      } finally {
        await viewerSession.close();
        await runOptionalSmokeCleanup('secondary-route-permission:setting-viewer', async () => {
          await deleteUserByUsername(page, adminLogin, username);
          await deleteRoleByKey(page, adminLogin, roleKey);
        });
      }
    });

    test(`/system/user/1 keeps readonly permission state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const adminLogin = await loginByApi(page.request, adminCredentials);
      const roleKey = `user_view_only_route_${viewport.width}_${Date.now()}`;
      const username = `user_viewer_route_${viewport.width}_${Date.now()}`;
      const viewer = await provisionViewer(page, adminLogin, {
        roleName: '用户详情次级路由只读烟测角色',
        roleKey,
        username,
        permissionKeys: ['system:user:list', 'system:user:view'],
        apiPermissions: [
          { path: '/api/v1/system/user/1', method: 'GET' },
          { path: '/api/v1/system/user/list', method: 'GET' },
          { path: '/api/v1/system/dept/tree', method: 'GET' },
          { path: '/api/v1/system/post/list', method: 'GET' },
          { path: '/api/v1/system/role/list', method: 'GET' },
          { path: '/api/v1/system/menu/tree', method: 'GET' },
        ],
      });
      const viewerSession = await openViewerPage(page, viewer);
      const { viewerPage } = viewerSession;
      const runtimeErrors = collectRuntimeErrors(viewerPage);

      try {
        await viewerPage.setViewportSize({ width: viewport.width, height: viewport.height });
        await viewerPage.goto('/system/user/1', { waitUntil: 'networkidle' });

        await expect(viewerPage).toHaveURL(/\/system\/user\/1$/);
        await expectRouteShell(viewerPage);
        await expectPageIdentity(viewerPage, '用户详情');
        await expectNoPageError(viewerPage);

        const workActions = viewerPage.locator('.system-list__work-actions');
        const backButton = workActions.getByRole('button', { name: /返回|Back/ });

        await expect(workActions).toBeVisible();
        await expect(backButton).toBeVisible();
        await expect(workActions.getByRole('button')).toHaveCount(1);
        await expect(viewerPage.getByText('基础信息', { exact: true })).toBeVisible();
        await expect(viewerPage.getByText('账号摘要', { exact: true })).toBeVisible();
        await expect(viewerPage.getByRole('button', { name: '保存', exact: true })).toHaveCount(0);
        await expect(viewerPage.getByRole('button', { name: '编辑用户', exact: true })).toHaveCount(0);
        await expect(viewerPage.getByRole('button', { name: '重置密码', exact: true })).toHaveCount(0);
        await expect(viewerPage.getByRole('button', { name: '删除', exact: true })).toHaveCount(0);
        await expectNoViewportOverflow(viewerPage);
        await viewerPage.screenshot({
          path: join(artifactDir, `system-user-detail-readonly-permission${viewport.suffix}`),
          fullPage: true,
        });

        await backButton.click();
        await expect(viewerPage).toHaveURL(/\/system\/user$/);
        await expect(viewerPage.locator('[role="tab"][aria-selected="true"]').first()).toContainText('用户管理');
        await expect(viewerPage.locator('.filter-panel')).toBeVisible();
        await expect(viewerPage.locator('.app-table')).toBeVisible();

        expectOnlyAllowedRuntimeErrors(runtimeErrors);
      } finally {
        await viewerSession.close();
        await runOptionalSmokeCleanup('secondary-route-permission:user-detail-viewer', async () => {
          await deleteUserByUsername(page, adminLogin, username);
          await deleteRoleByKey(page, adminLogin, roleKey);
        });
      }
    });
  }
});
