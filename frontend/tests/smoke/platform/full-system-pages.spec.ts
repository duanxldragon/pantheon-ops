import { expect, test, type Page } from '@playwright/test';
import {
  adminCredentials,
  installClientSession,
  loginByApi,
} from '../helpers/auth';

type ViewportCase = {
  key: string;
  width: number;
  height: number;
};

type SmokePage = {
  path: string;
  title: string;
  layer: 'platform' | 'system/auth' | 'system/iam' | 'system/org' | 'system/config';
};

const viewportCases: ViewportCase[] = [
  { key: 'pc', width: 1440, height: 900 },
  { key: 'pad', width: 1024, height: 768 },
  { key: 'phone', width: 390, height: 844 },
];

const smokePages: SmokePage[] = [
  { layer: 'platform', path: '/dashboard', title: '工作台' },
  { layer: 'system/auth', path: '/auth/security', title: '安全中心' },
  { layer: 'system/auth', path: '/system/login-log', title: '登录日志' },
  { layer: 'system/auth', path: '/system/session', title: '会话管理' },
  { layer: 'system/iam', path: '/system/profile', title: '个人中心' },
  { layer: 'system/iam', path: '/system/user', title: '用户管理' },
  { layer: 'system/iam', path: '/system/user/1', title: '用户详情' },
  { layer: 'system/iam', path: '/system/role', title: '角色管理' },
  { layer: 'system/iam', path: '/system/menu', title: '菜单管理' },
  { layer: 'system/iam', path: '/system/permission', title: '权限管理' },
  { layer: 'system/iam', path: '/system/operation-log', title: '操作日志' },
  { layer: 'system/org', path: '/system/dept', title: '部门管理' },
  { layer: 'system/org', path: '/system/post', title: '岗位管理' },
  { layer: 'system/config', path: '/system/dict', title: '字典管理' },
  { layer: 'system/config', path: '/system/setting', title: '系统设置' },
  { layer: 'system/config', path: '/system/i18n', title: '国际化管理' },
  { layer: 'system/config', path: '/system/modules', title: '模块注册表' },
  { layer: 'system/config', path: '/system/generator', title: '模块生成器' },
];

function collectRuntimeErrors(page: Page) {
  const errors: string[] = [];

  page.on('console', (message) => {
    const text = message.text();
    if (
      message.type() === 'error'
      && !/Failed to load resource: the server responded with a status of 404/i.test(text)
      && !/Failed to load resource: net::ERR_CONNECTION_CLOSED/i.test(text)
      && !/favicon/i.test(text)
    ) {
      errors.push(text);
    }
  });

  page.on('pageerror', (error) => {
    errors.push(error.message);
  });

  page.on('response', (response) => {
    const status = response.status();
    const url = response.url();
    if (status >= 500 && url.includes('/api/')) {
      errors.push(`${status} ${url}`);
    }
  });

  return errors;
}

async function expectNoBrokenState(page: Page) {
  const forbidden = page.getByText('无权访问', { exact: false });
  const notFound = page.getByText('页面不存在', { exact: false });
  const loadFailed = page.getByText('加载失败', { exact: false });
  const networkError = page.getByText('网络异常', { exact: false });

  await expect(forbidden).toHaveCount(0);
  await expect(notFound).toHaveCount(0);
  await expect(loadFailed).toHaveCount(0);
  await expect(networkError).toHaveCount(0);
}

async function expectVisiblePageTitle(page: Page, title: string) {
  const visibleMatches = page.getByText(title, { exact: false }).filter({ visible: true });
  await expect(visibleMatches.first()).toBeVisible();
}

test.describe('full system page smoke', () => {
  for (const viewport of viewportCases) {
    test.describe(`${viewport.key} ${viewport.width}x${viewport.height}`, () => {
      test.beforeEach(async ({ page }) => {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
      });

      test(`login page renders on ${viewport.key}`, async ({ page }) => {
        const errors = collectRuntimeErrors(page);
        await page.goto('/login', { waitUntil: 'domcontentloaded' });
        await expect(page.getByRole('button', { name: /登录|Sign In/i })).toBeVisible();
        await expect(page.locator('input[type="password"]')).toBeVisible();
        expect(errors).toEqual([]);
      });

      for (const smokePage of smokePages) {
        test(`${smokePage.layer}: ${smokePage.path}`, async ({ page }) => {
          const tokens = await loginByApi(page.request, adminCredentials);
          await installClientSession(page, tokens);
          const errors = collectRuntimeErrors(page);

          await page.goto(smokePage.path, { waitUntil: 'domcontentloaded' });

          await expect(page).toHaveURL(new RegExp(`${smokePage.path.replace(/\//g, '\\/')}$`));
          await expectVisiblePageTitle(page, smokePage.title);
          await expect(page.locator('#root')).not.toBeEmpty();
          await expectNoBrokenState(page);
          expect(errors).toEqual([]);
        });
      }
    });
  }
});
