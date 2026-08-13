import { test } from '../../fixtures/coverage';
import { expect, type Page, type Route } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import {
  apiBaseUrl,
  authHeaders,
  installOperationToken,
  signInAsAdmin,
} from '../helpers/auth';
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

type Deferred<T = void> = {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
};

type ViewportCase = {
  width: number;
  height: number;
  suffix: '-desktop.png' | '-desktop-1280.png';
};

const viewportCases: ViewportCase[] = [
  { width: 1440, height: 900, suffix: '-desktop.png' },
  { width: 1280, height: 900, suffix: '-desktop-1280.png' },
];

function createDeferred<T = void>(): Deferred<T> {
  let resolve!: (value: T | PromiseLike<T>) => void;
  const promise = new Promise<T>((nextResolve) => {
    resolve = nextResolve;
  });
  return { promise, resolve };
}

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

async function expectNoViewportOverflow(page: Page) {
  await expect
    .poll(async () =>
      page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    )
    .toBeLessThanOrEqual(1);
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

async function fulfillJson(route: Route, status: number, body: Record<string, unknown>) {
  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

test.beforeAll(async () => {
  await ensureArtifactDir();
});

test.describe('system secondary route interaction states', () => {
  for (const viewport of viewportCases) {
    test(`/system/profile keeps save loading stable at ${viewport.width}w`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      const accessToken = await signInAsAdmin(page);
      const profileResponse = await page.request.get(`${apiBaseUrl}/system/profile`, {
        headers: authHeaders(accessToken),
      });
      expect(profileResponse.ok()).toBeTruthy();
      const profilePayload = await profileResponse.json();
      const profile = profilePayload.data as Record<string, unknown>;
      const nextNickname = `${String(profile.nickname || 'Administrator')} 巡检`;
      const submitGate = createDeferred<void>();
      let intercepted = false;

      await page.route(/\/api\/v1\/system\/profile$/, async (route) => {
        if (route.request().method() !== 'PUT') {
          await route.fallback();
          return;
        }
        intercepted = true;
        const body = route.request().postDataJSON() as Record<string, unknown>;
        await submitGate.promise;
        await fulfillJson(route, 200, {
          code: 200,
          data: {
            ...profile,
            nickname: body.nickname,
            avatar: body.avatar ?? profile.avatar ?? '',
            email: body.email ?? profile.email ?? '',
            phone: body.phone ?? profile.phone ?? '',
          },
        });
      });

      try {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/system/profile', { waitUntil: 'networkidle' });

        await expectRouteShell(page);
        await expectNoPageError(page);

        const nicknameInput = page.locator('input').nth(1);
        const saveButton = page.getByRole('button', { name: '保存资料', exact: true });
        await nicknameInput.fill(nextNickname);
        await saveButton.click({ noWaitAfter: true });

        await expect.poll(() => intercepted).toBeTruthy();
        await expect(saveButton).toHaveClass(/arco-btn-loading/);
        await expect(page.locator('.submit-bar')).toBeVisible();
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(
            artifactDir,
            `system-profile-save-interaction${viewport.suffix}`,
          ),
          fullPage: true,
        });

        submitGate.resolve();
        await expect(saveButton).not.toHaveClass(/arco-btn-loading/);
        await expect(nicknameInput).toHaveValue(nextNickname);
      } finally {
        await page.unroute(/\/api\/v1\/system\/profile$/);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });

    test(`/system/user/1 keeps back navigation stable at ${viewport.width}w`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('/system/user/1', { waitUntil: 'networkidle' });

      await expectRouteShell(page);
      await expectNoPageError(page);
      await page.getByRole('button', { name: /返回|Back/ }).click();
      await expect(page).toHaveURL(/\/system\/user$/);
      await expect(page.locator('[role="tab"][aria-selected="true"]').first()).toContainText('用户管理');
      await expect(page.locator('.filter-panel')).toBeVisible();
      await expect(page.locator('.app-table')).toBeVisible();
      await expectNoViewportOverflow(page);
      await page.screenshot({
        path: join(
          artifactDir,
          `system-user-detail-back-interaction${viewport.suffix}`,
        ),
        fullPage: true,
      });

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });

    test(`/system/setting overview keeps single-tab group switching stable at ${viewport.width}w`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('/system/setting', { waitUntil: 'networkidle' });

      await expectRouteShell(page);
      await expectNoPageError(page);
      await expect(page).toHaveURL(/\/system\/setting$/);
      await expect(page.locator('.setting-overview-page')).toBeVisible();

      const nav = page.locator('.setting-overview-page__anchor-strip');
      await expect(nav).toBeVisible();

      const securityItem = nav.locator('.setting-overview-page__anchor-item').filter({ hasText: '安全策略' }).first();
      await securityItem.click();
      await expect(page).toHaveURL(/\/system\/setting\?group=security$/);
      await expect(page.locator('#setting-group-section-security')).toBeVisible();
      await expect(page.locator('.setting-overview-page__anchor-item--active')).toContainText('安全策略');
      await expect(page.locator('.setting-group-workspace')).toHaveCount(1);

      const auditItem = nav.locator('.setting-overview-page__anchor-item').filter({ hasText: '日志治理' }).first();
      await auditItem.click();
      await expect(page).toHaveURL(/\/system\/setting\?group=audit$/);
      await expect(page.locator('#setting-group-section-audit')).toBeVisible();
      await expect(page.locator('.setting-overview-page__anchor-item--active')).toContainText('日志治理');
      await expect(page.locator('.setting-page__audit-card')).toBeVisible();
      await expectNoViewportOverflow(page);
      await page.screenshot({
        path: join(
          artifactDir,
          `system-setting-route-switch-interaction${viewport.suffix}`,
        ),
        fullPage: true,
      });

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });

    test(`/system/setting/security keeps save interaction stable at ${viewport.width}w`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      const accessToken = await signInAsAdmin(page);
      const submitGate = createDeferred<void>();
      let intercepted = false;

      await page.route(/\/api\/v1\/system\/setting\/group\/security$/, async (route) => {
        if (route.request().method() !== 'PUT') {
          await route.fallback();
          return;
        }
        intercepted = true;
        await submitGate.promise;
        await fulfillJson(route, 200, {
          code: 200,
          data: {
            groupKey: 'security',
          },
        });
      });

      try {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await installOperationToken(page, accessToken);
        await page.goto('/system/setting/security', { waitUntil: 'networkidle' });

        await expectRouteShell(page);
        await expectNoPageError(page);

        const firstSpinButton = page.getByRole('spinbutton').first();
        const saveButton = page.getByRole('button', { name: '保存', exact: true });
        const refreshButton = page.getByRole('button', { name: '刷新设置缓存', exact: true });
        await firstSpinButton.fill('9');
        await saveButton.click({ noWaitAfter: true });

        await expect.poll(() => intercepted).toBeTruthy();
        await expect(saveButton).toHaveClass(/arco-btn-loading/);
        await expect(page.locator('.submit-bar')).toBeVisible();
        await expect(refreshButton).toBeVisible();
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(
            artifactDir,
            `system-setting-security-save-interaction${viewport.suffix}`,
          ),
          fullPage: true,
        });

        submitGate.resolve();
        await expect(saveButton).not.toHaveClass(/arco-btn-loading/);
      } finally {
        await page.unroute(/\/api\/v1\/system\/setting\/group\/security$/);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });
  }
});
