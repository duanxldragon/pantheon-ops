import { expect, test, type Page } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { signInAsAdmin } from '../helpers/auth';
import { expectPagePathname } from '../helpers/url-pattern';

const artifactDir = join(process.cwd(), 'test-results', 'backoffice-ui');

const pageErrorTexts = [
  '加载失败',
  '网络异常',
  '请求超时',
  'Load failed',
  'Network error',
  'Request timed out',
];

type SecondaryRoute = {
  path: string;
  title: string;
  screenshot: string;
  assertReady: (page: Page) => Promise<void>;
};

const settingGroupRoutes: SecondaryRoute[] = [
  {
    path: '/system/setting/basic',
    title: '基础信息',
    screenshot: 'system-setting-basic-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('基础信息', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
  {
    path: '/system/setting/platform',
    title: '平台能力',
    screenshot: 'system-setting-platform-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('平台能力', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
  {
    path: '/system/setting/security',
    title: '安全策略',
    screenshot: 'system-setting-security-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('安全策略', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
  {
    path: '/system/setting/login',
    title: '登录策略',
    screenshot: 'system-setting-login-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('登录策略', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
  {
    path: '/system/setting/audit',
    title: '日志治理',
    screenshot: 'system-setting-audit-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.locator('.setting-page__audit-card')).toBeVisible();
    },
  },
  {
    path: '/system/setting/upload',
    title: '上传配置',
    screenshot: 'system-setting-upload-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('上传配置', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
  {
    path: '/system/setting/i18n',
    title: '国际化',
    screenshot: 'system-setting-i18n-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('国际化', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
  {
    path: '/system/setting/ui',
    title: '界面偏好',
    screenshot: 'system-setting-ui-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.setting-group-page')).toBeVisible();
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.getByText('界面偏好', { exact: true }).filter({ visible: true }).first()).toBeVisible();
    },
  },
];

const secondaryRoutes: SecondaryRoute[] = [
  {
    path: '/system/profile',
    title: '个人中心',
    screenshot: 'system-profile-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.locator('.submit-bar')).toBeVisible();
      await expect(page.locator('.arco-form')).toBeVisible();
      await expect(page.getByText('资料维护', { exact: true })).toBeVisible();
    },
  },
  {
    path: '/system/user/1',
    title: '用户详情',
    screenshot: 'system-user-detail-route-desktop.png',
    assertReady: async (page) => {
      await expect(page.getByRole('button', { name: /返回|Back/ })).toBeVisible();
      await expect(page.getByText('基础信息', { exact: true })).toBeVisible();
      await expect(page.getByText('账号摘要', { exact: true })).toBeVisible();
    },
  },
  ...settingGroupRoutes,
];

const viewportCases = [
  { width: 1440, height: 900, suffix: '-desktop.png' },
  { width: 1280, height: 900, suffix: '-desktop-1280.png' },
] as const;

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

function expectOnlyAllowedRuntimeErrors(runtimeErrors: string[], allowedPatterns: RegExp[] = []) {
  const unexpectedErrors = runtimeErrors.filter(
    (message) => !allowedPatterns.some((pattern) => pattern.test(message)),
  );
  expect(unexpectedErrors).toEqual([]);
}

async function expectNoPageError(page: Page) {
  for (const text of pageErrorTexts) {
    await expect(page.getByText(text, { exact: false })).toHaveCount(0);
  }
}

async function expectProfessionalBackofficeSurface(page: Page) {
  await expect(page.locator('.app-shell__header')).toBeVisible();
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(
    page
      .locator(
        '.page-panel, .arco-card, .system-list__work-actions, .setting-overview-page, .setting-group-page',
      )
      .first(),
  ).toBeVisible();
  await expect(page.locator('.arco-layout-sider-dark')).toHaveCount(0);
  const usesHorizontalShell = (await page.locator('.app-shell--horizontal').count()) > 0;
  if (usesHorizontalShell) {
    await expect(page.locator('.app-shell__header-brand')).toBeVisible();
    await expect(page.locator('.app-shell__top-menu')).toBeVisible();
    return;
  }
  await expect(page.locator('.app-shell__sider')).toBeVisible();
}

async function expectPageIdentity(page: Page, title: string) {
  const activeTab = page.locator('[role="tab"][aria-selected="true"]').first();
  await expect(activeTab).toContainText(title);

  const breadcrumbs = page.locator('.app-shell__header').getByRole('listitem');
  await expect(breadcrumbs.last()).toContainText(title);
}

async function expectNoViewportOverflow(page: Page) {
  await expect
    .poll(async () =>
      page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth),
    )
    .toBeLessThanOrEqual(1);
}

test.beforeAll(async () => {
  await ensureArtifactDir();
});

test.describe('system secondary route visual acceptance', () => {
  for (const viewport of viewportCases) {
    for (const route of secondaryRoutes) {
      test(`${route.path} keeps stable visual evidence at ${viewport.width}w`, async ({ page }) => {
        const runtimeErrors = collectRuntimeErrors(page);
        await signInAsAdmin(page);

        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await page.goto(route.path, { waitUntil: 'networkidle' });

        expectPagePathname(page, route.path);
        await expectPageIdentity(page, route.title);
        await expectNoPageError(page);
        await expectProfessionalBackofficeSurface(page);
        await route.assertReady(page);
        await expectNoViewportOverflow(page);
        await page.screenshot({
          path: join(artifactDir, route.screenshot.replace('-desktop.png', viewport.suffix)),
          fullPage: true,
        });

        expectOnlyAllowedRuntimeErrors(runtimeErrors);
      });
    }
  }
});
