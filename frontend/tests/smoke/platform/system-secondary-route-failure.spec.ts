import { test } from '../../fixtures/coverage';
import { expect, type Page, type Route } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { signInAsAdmin } from '../helpers/auth';
import { expectOnlyAllowedRuntimeErrors } from '../helpers/runtime-errors';

const artifactDir = join(process.cwd(), 'test-results', 'backoffice-ui');

type ViewportCase = {
  width: number;
  height: number;
  suffix: '-desktop.png' | '-desktop-1280.png';
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

async function expectRouteShell(page: Page) {
  await expect(page.locator('.app-shell__header')).toBeVisible();
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(page.locator('[role="tab"][aria-selected="true"]').first()).toBeVisible();
}

async function expectNoViewportOverflow(page: Page) {
  await expect
    .poll(async () =>
      page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    )
    .toBeLessThanOrEqual(1);
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

test.describe('system secondary route failure and empty states', () => {
  for (const viewport of viewportCases) {
    test(`/system/user/0 keeps invalid-id empty state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.setViewportSize({ width: viewport.width, height: viewport.height });
      await page.goto('/system/user/0', { waitUntil: 'networkidle' });

      await expectRouteShell(page);
      await expect(page.locator('[role="tab"][aria-selected="true"]').first()).toContainText('用户详情');
      await expect(page.getByText('用户标识无效', { exact: true })).toBeVisible();
      await expect(page.locator('.page-empty')).toBeVisible();
      await expectNoViewportOverflow(page);
      await page.screenshot({
        path: join(artifactDir, `system-user-detail-invalid-empty${viewport.suffix}`),
        fullPage: true,
      });

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });

    test(`/system/user/1 keeps load-failed state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.route(/\/api\/v1\/system\/user\/1$/, async (route) => {
        if (route.request().method() !== 'GET') {
          await route.fallback();
          return;
        }
        await fulfillJson(route, 500, {
          code: 500,
          message: 'request.failed',
        });
      });

      try {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/system/user/1', { waitUntil: 'networkidle' });

        await expectRouteShell(page);
        await expect(page.getByText('加载失败', { exact: true })).toBeVisible();
        await expect(page.getByRole('button', { name: '重试', exact: true })).toBeVisible();
        await expect(page.locator('.page-result')).toBeVisible();
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(artifactDir, `system-user-detail-load-error${viewport.suffix}`),
          fullPage: true,
        });
      } finally {
        await page.unroute(/\/api\/v1\/system\/user\/1$/);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors, [
        /Failed to load resource: the server responded with a status of 500/i,
        /^request\.failed$/,
      ]);
    });

    test(`/system/user/1 keeps no-data empty state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.route(/\/api\/v1\/system\/user\/1$/, async (route) => {
        if (route.request().method() !== 'GET') {
          await route.fallback();
          return;
        }
        await fulfillJson(route, 200, {
          code: 200,
          data: null,
        });
      });

      try {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/system/user/1', { waitUntil: 'networkidle' });

        await expectRouteShell(page);
        await expect(page.getByText('暂无数据', { exact: true })).toBeVisible();
        await expect(page.locator('.page-empty')).toBeVisible();
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(artifactDir, `system-user-detail-no-data${viewport.suffix}`),
          fullPage: true,
        });
      } finally {
        await page.unroute(/\/api\/v1\/system\/user\/1$/);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });

    test(`/system/setting/basic keeps server-error state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.route(/\/api\/v1\/system\/setting\/list(?:\?.*)?$/, async (route) => {
        if (route.request().method() !== 'GET') {
          await route.fallback();
          return;
        }
        await fulfillJson(route, 500, {
          code: 500,
          message: 'request.failed',
        });
      });

      try {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/system/setting/basic', { waitUntil: 'networkidle' });

        await expectRouteShell(page);
        await expect(page.getByText('服务异常', { exact: true })).toBeVisible();
        await expect(page.getByRole('button', { name: '重试', exact: true })).toBeVisible();
        await expect(page.locator('.page-result')).toBeVisible();
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(artifactDir, `system-setting-server-error${viewport.suffix}`),
          fullPage: true,
        });
      } finally {
        await page.unroute(/\/api\/v1\/system\/setting\/list(?:\?.*)?$/);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors, [
        /Failed to load resource: the server responded with a status of 500/i,
        /^request\.failed$/,
      ]);
    });

    test(`/system/setting/basic keeps empty state stable at ${viewport.width}w`, async ({
      page,
    }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.route(/\/api\/v1\/system\/setting\/list(?:\?.*)?$/, async (route) => {
        if (route.request().method() !== 'GET') {
          await route.fallback();
          return;
        }
        await fulfillJson(route, 200, {
          code: 200,
          data: [],
        });
      });
      await page.route(/\/api\/v1\/system\/setting\/overview$/, async (route) => {
        if (route.request().method() !== 'GET') {
          await route.fallback();
          return;
        }
        await fulfillJson(route, 200, {
          code: 200,
          data: {
            totalSettingCount: 0,
            publicSettingCount: 0,
            encryptedSettingCount: 0,
            requiredMissingCount: 0,
            riskCount: 0,
            storageDriver: 'local',
            defaultLanguage: 'zh-CN',
            defaultTheme: 'emerald',
            issues: [],
          },
        });
      });

      try {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto('/system/setting/basic', { waitUntil: 'networkidle' });

        await expectRouteShell(page);
        await expect(page.getByText('暂无系统设置', { exact: true })).toBeVisible();
        await expect(page.locator('.page-empty')).toBeVisible();
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(artifactDir, `system-setting-empty${viewport.suffix}`),
          fullPage: true,
        });
      } finally {
        await page.unroute(/\/api\/v1\/system\/setting\/list(?:\?.*)?$/);
        await page.unroute(/\/api\/v1\/system\/setting\/overview$/);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });
  }
});
