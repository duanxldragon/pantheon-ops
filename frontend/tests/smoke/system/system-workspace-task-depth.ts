import { expect, test, type Locator, type Page, type Response } from '@playwright/test';
import { apiBaseUrl, authHeaders, requestHeaders, signInAsAdmin } from '../helpers/auth';

type ApiEnvelope<T> = {
  code: number;
  data: T;
};

type DashboardSummary = {
  recentLogins: Array<{ id: number }>;
  orgGovernanceTasks: Array<{ taskKey: string }>;
};

type SecurityOverview = {
  policy: {
    passwordMinLength: number;
  };
};

type SecuritySession = {
  sessionId: string;
  isCurrent: boolean;
  lastIp: string;
};

type LoginLogPage = {
  items: Array<{ id: number; status: number }>;
};

type UserProfile = {
  nickname: string;
  username: string;
  avatar?: string;
  email?: string;
  phone?: string;
  profileExt?: Record<string, unknown>;
};

type UserDetail = {
  username: string;
  nickname: string;
  roleKeys: string[];
  roleNames?: string[];
};

type WorkspaceTaskDepthDeps = {
  expectVisiblePageTitle: (page: Page, title: string | RegExp) => Promise<void>;
  expectPageIdentityReady: (page: Page, title: string | RegExp) => Promise<void>;
  formItem: (page: Page, label: string) => Locator;
};

function cardByTitle(page: Page, title: RegExp) {
  return page.locator('.arco-card').filter({
    has: page.locator('.arco-card-header-title').filter({ hasText: title }),
  }).first();
}

async function expectOkJson<T>(responsePromise: Promise<Response>) {
  const response = await responsePromise;
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as ApiEnvelope<T>;
}

export function registerSystemWorkspaceTaskDepthSmokeTests({
  expectVisiblePageTitle,
  expectPageIdentityReady,
  formItem,
}: WorkspaceTaskDepthDeps) {
  test.describe('workspace task-depth smoke', () => {
    test('dashboard keeps narrow reflow stable and task widgets ready', async ({ page }) => {
      await signInAsAdmin(page);
      await page.setViewportSize({ width: 520, height: 960 });

      const summaryPayloadPromise = expectOkJson<DashboardSummary>(
        page.waitForResponse(
          (response) =>
            response.url().includes('/api/v1/platform/dashboard/summary') &&
            response.request().method() === 'GET',
        ),
      );

      await page.goto('/dashboard', { waitUntil: 'networkidle' });
      await expectVisiblePageTitle(page, '工作台');
      const summaryPayload = await summaryPayloadPromise;
      expect(summaryPayload.code).toBe(200);

      await expect(page.locator('.dashboard-grid')).toBeVisible();
      await expect(page.locator('.dashboard-hero-card')).toBeVisible();
      await expect(page.locator('.dashboard-stat-card').first()).toBeVisible();
      await expect(page.locator('.dashboard-panel-card--attention .dashboard-focus-item').first()).toBeVisible();
      await expect
        .poll(async () => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth))
        .toBeLessThanOrEqual(1);

      await expect(
        page.locator(
          '.dashboard-panel-card--actions .dashboard-quick-action, .dashboard-panel-card--actions .arco-empty',
        ).first(),
      ).toBeVisible();
      await expect(page.locator('.dashboard-login-table .arco-table, .dashboard-login-table .arco-empty').first()).toBeVisible();

      const todoCard = cardByTitle(page, /统一待办|Unified Todo/);
      if (summaryPayload.data.orgGovernanceTasks.length > 0) {
        await expect(todoCard.locator('.dashboard-task-card').first()).toBeVisible();
      } else {
        await expect(todoCard.locator('.arco-empty')).toBeVisible();
      }

      if (summaryPayload.data.recentLogins.length > 0) {
        await expect(page.locator('.dashboard-login-table tbody tr').first()).toBeVisible();
      } else {
        await expect(page.locator('.dashboard-login-table .arco-empty')).toBeVisible();
      }
    });

    test('security center keeps split layout, policy rail, sessions, and login logs ready', async ({
      page,
    }) => {
      await signInAsAdmin(page);

      const overviewPayloadPromise = expectOkJson<SecurityOverview>(
        page.waitForResponse(
          (response) =>
            response.url().includes('/api/v1/auth/security') &&
            response.request().method() === 'GET',
        ),
      );
      const sessionsPayloadPromise = expectOkJson<SecuritySession[]>(
        page.waitForResponse(
          (response) =>
            response.url().includes('/api/v1/auth/sessions') &&
            response.request().method() === 'GET',
        ),
      );
      const loginLogsPayloadPromise = expectOkJson<LoginLogPage>(
        page.waitForResponse(
          (response) =>
            response.url().includes('/api/v1/auth/login-logs') &&
            response.request().method() === 'GET',
        ),
      );

      await page.goto('/auth/security', { waitUntil: 'networkidle' });
      await expectVisiblePageTitle(page, '安全中心');

      const [overviewPayload, sessionsPayload, loginLogsPayload] = await Promise.all([
        overviewPayloadPromise,
        sessionsPayloadPromise,
        loginLogsPayloadPromise,
      ]);
      expect(overviewPayload.code).toBe(200);
      expect(sessionsPayload.code).toBe(200);
      expect(loginLogsPayload.code).toBe(200);
      expect(overviewPayload.data.policy.passwordMinLength).toBeGreaterThan(0);

      await expect(page.locator('.page-split-layout--with-rail')).toBeVisible();
      await expect
        .poll(async () => page.locator('.page-side-column .side-rail-panel').count())
        .toBeGreaterThanOrEqual(3);
      await expect(page.locator('.page-side-column').getByText(/当前安全策略|Current Security Policy/)).toBeVisible();
      await expect(page.locator('.page-side-column .side-rail-item').first()).toBeVisible();

      const sessionsCard = cardByTitle(page, /在线会话|Active Sessions/);
      await expect(sessionsCard).toBeVisible();
      await expect(sessionsCard.getByText(/当前设备|Current Device/).first()).toBeVisible();
      await expect(sessionsCard.locator('.arco-table, .arco-empty').first()).toBeVisible();
      if (sessionsPayload.data.length > 0) {
        await expect(sessionsCard.locator('.arco-table tbody tr').first()).toBeVisible();
      } else {
        await expect(sessionsCard.locator('.arco-empty')).toBeVisible();
      }

      const loginLogsCard = cardByTitle(page, /最近登录|Recent Logins/);
      await expect(loginLogsCard).toBeVisible();
      await expect(loginLogsCard.locator('.arco-table, .arco-empty').first()).toBeVisible();
      if (loginLogsPayload.data.items.length > 0) {
        await expect(loginLogsCard.getByText(/成功|Success|失败|Failed/).first()).toBeVisible();
        await expect(loginLogsCard.locator('.arco-table tbody tr').first()).toBeVisible();
      } else {
        await expect(loginLogsCard.locator('.arco-empty')).toBeVisible();
      }
    });

    test('profile center loads editable fields and echoes saved nickname changes', async ({ page }) => {
      const accessToken = await signInAsAdmin(page);
      const profileLoadPayloadPromise = expectOkJson<UserProfile>(
        page.waitForResponse(
          (response) =>
            response.url().includes('/api/v1/system/profile') &&
            response.request().method() === 'GET',
        ),
      );

      await page.goto('/system/profile', { waitUntil: 'networkidle' });
      await expectVisiblePageTitle(page, '个人中心');
      const profileLoadPayload = await profileLoadPayloadPromise;
      expect(profileLoadPayload.code).toBe(200);
      const originalProfile = profileLoadPayload.data;

      const usernameInput = formItem(page, '用户名').locator('input').first();
      const nicknameInput = formItem(page, '昵称').locator('input').first();
      const emailInput = formItem(page, '邮箱').locator('input').first();
      const phoneInput = formItem(page, '手机号').locator('input').first();

      await expect(usernameInput).toHaveValue(originalProfile.username);
      await expect(nicknameInput).toHaveValue(originalProfile.nickname || '');
      await expect(emailInput).toBeVisible();
      await expect(phoneInput).toBeVisible();
      await expect(page.locator('.submit-bar button').last()).toBeVisible();

      const nextNickname = `${(originalProfile.nickname || originalProfile.username || 'admin').slice(0, 12)}-${Date.now().toString().slice(-4)}`;

      try {
        const savePayloadPromise = expectOkJson<UserProfile>(
          page.waitForResponse(
            (response) =>
              response.url().includes('/api/v1/system/profile') &&
              response.request().method() === 'PUT',
          ),
        );

        await nicknameInput.fill(nextNickname);
        await page.locator('.submit-bar button').last().click();
        const savePayload = await savePayloadPromise;
        expect(savePayload.code).toBe(200);
        expect(savePayload.data.nickname).toBe(nextNickname);

        await expect(nicknameInput).toHaveValue(nextNickname);
        await expect(page.locator('.page-panel--soft')).toContainText(nextNickname);
      } finally {
        const restoreResponse = await page.request.put(`${apiBaseUrl}/system/profile`, {
          headers: await requestHeaders(page, accessToken),
          data: {
            nickname: originalProfile.nickname,
            avatar: originalProfile.avatar,
            email: originalProfile.email,
            phone: originalProfile.phone,
            profileExt: originalProfile.profileExt,
          },
        });
        expect(restoreResponse.ok()).toBeTruthy();
        const restorePayload = (await restoreResponse.json()) as ApiEnvelope<UserProfile>;
        expect(restorePayload.code).toBe(200);
      }
    });

    test('user detail loads content and returns to list context', async ({ page }) => {
      const accessToken = await signInAsAdmin(page);
      const detailResponse = await page.request.get(`${apiBaseUrl}/system/user/1`, {
        headers: authHeaders(accessToken),
      });
      expect(detailResponse.ok()).toBeTruthy();
      const detailPayload = (await detailResponse.json()) as ApiEnvelope<UserDetail>;
      expect(detailPayload.code).toBe(200);
      expect(detailPayload.data.username).toBeTruthy();
      expect(detailPayload.data.roleKeys.length).toBeGreaterThan(0);
      const expectedRoleLabel =
        detailPayload.data.roleNames?.find(Boolean) || detailPayload.data.roleKeys[0];

      const detailLoadPayloadPromise = expectOkJson<UserDetail>(
        page.waitForResponse(
          (response) =>
            response.url().includes('/api/v1/system/user/1') &&
            response.request().method() === 'GET',
        ),
      );

      await page.goto('/system/user/1', { waitUntil: 'networkidle' });
      const detailLoadPayload = await detailLoadPayloadPromise;
      expect(detailLoadPayload.code).toBe(200);

      await expect(page.getByRole('button', { name: /返回|Back/ })).toBeVisible();
      await expect(
        page.getByRole('heading', {
          name: detailPayload.data.nickname || detailPayload.data.username,
        }),
      ).toBeVisible();
      await expect(page.getByText(detailPayload.data.username, { exact: true }).first()).toBeVisible();
      await expect(
        page.getByRole('row', {
          name: new RegExp(`用户名\\s+${detailPayload.data.username}`),
        }).first(),
      ).toBeVisible();
      await expect(page.getByRole('row', { name: new RegExp(`角色\\s+${expectedRoleLabel}`) }).first()).toBeVisible();
      await expect(page.getByText('基础信息', { exact: true })).toBeVisible();
      await expect(page.getByText('账号摘要', { exact: true })).toBeVisible();
      await expect(page.locator('.arco-card').first()).toContainText(
        detailPayload.data.nickname || detailPayload.data.username,
      );

      await Promise.all([
        page.waitForURL(/\/system\/user$/),
        page.getByRole('button', { name: /返回|Back/ }).click(),
      ]);
      await expectPageIdentityReady(page, '用户管理');
    });
  });
}
