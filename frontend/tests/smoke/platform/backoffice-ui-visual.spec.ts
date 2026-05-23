import { expect, test, type Page } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { apiBaseUrl, authHeaders, requestHeaders, signInAsAdmin } from '../helpers/auth';
const artifactDir = join(process.cwd(), 'test-results', 'backoffice-ui');

const pageErrorTexts = ['加载失败', '网络异常', '请求超时', 'Load failed', 'Network error', 'Request timed out'];

interface ShellPreferenceOptions {
  language?: string;
  layoutMode?: 'vertical' | 'horizontal';
  densityMode?: 'comfortable' | 'compact';
}

async function installExplicitZhCNPreference(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('pantheon_lang', 'zh-CN');
    localStorage.setItem('pantheon_lang_explicit', '1');
  });
}

async function installShellPreferences(page: Page, options: ShellPreferenceOptions = {}) {
  const {
    language = 'zh-CN',
    layoutMode = 'vertical',
    densityMode = 'comfortable',
  } = options;
  await page.addInitScript(
    ({ nextLanguage, nextLayoutMode, nextDensityMode }) => {
      localStorage.setItem('pantheon_lang', nextLanguage);
      localStorage.setItem('pantheon_lang_explicit', '1');
      localStorage.setItem('pantheon_shell_layout_mode', nextLayoutMode);
      localStorage.setItem('pantheon_shell_density_mode', nextDensityMode);
    },
    {
      nextLanguage: language,
      nextLayoutMode: layoutMode,
      nextDensityMode: densityMode,
    },
  );
}

async function getCurrentUserPreferences(page: Page, accessToken: string) {
  const response = await page.request.get(`${apiBaseUrl}/auth/me`, {
    headers: authHeaders(accessToken),
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return (payload.data?.preferences || {}) as ShellPreferenceOptions;
}

async function updateCurrentUserPreferences(
  page: Page,
  accessToken: string,
  preferences: ShellPreferenceOptions,
) {
  const response = await page.request.put(`${apiBaseUrl}/auth/me/preferences`, {
    headers: await requestHeaders(page, accessToken),
    data: preferences,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
}

const authenticatedPages = [
  { path: '/dashboard', title: '工作台', screenshot: 'dashboard-desktop.png' },
  { path: '/system/user', title: '用户管理', screenshot: 'system-user-desktop.png' },
  { path: '/system/role', title: '角色管理', screenshot: 'system-role-desktop.png' },
  { path: '/system/permission', title: '权限管理', screenshot: 'system-permission-desktop.png' },
  { path: '/system/menu', title: '菜单管理', screenshot: 'system-menu-desktop.png' },
  { path: '/system/dept', title: '部门管理', screenshot: 'system-dept-desktop.png' },
  { path: '/system/post', title: '岗位管理', screenshot: 'system-post-desktop.png' },
  { path: '/system/setting', title: '系统设置', screenshot: 'system-setting-desktop.png' },
  { path: '/auth/security', title: '安全中心', screenshot: 'auth-security-desktop.png' },
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
  await expect(page.locator('.app-shell__sider')).toBeVisible();
  await expect(page.locator('.app-shell__header')).toBeVisible();
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(page.locator('.page-panel').first()).toBeVisible();
  await expect(page.locator('.arco-layout-sider-dark')).toHaveCount(0);
}

async function expectPageIdentity(page: Page, title: string) {
  const activeTab = page.locator('[role="tab"][aria-selected="true"]').first();
  await expect(activeTab).toContainText(title);

  const breadcrumbs = page.locator('.app-shell__header').getByRole('listitem');
  await expect(breadcrumbs.last()).toContainText(title);
}

test.beforeAll(async () => {
  await ensureArtifactDir();
});

test.describe('backoffice UI visual acceptance', () => {
  test('login page keeps a professional authentication console on desktop and mobile', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);

    await installExplicitZhCNPreference(page);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/login', { waitUntil: 'networkidle' });

    await expect(page.locator('.auth-login-page')).toBeVisible();
    await expect(page.locator('.auth-login-card')).toBeVisible();
    await expect(page.locator('.auth-login-page__brand-pane')).toBeVisible();
    await expect(page.getByRole('button', { name: '登录' })).toBeVisible();
    await expect(page.locator('.arco-carousel')).toHaveCount(0);
    await expect(page.getByText('记住我', { exact: false })).toHaveCount(0);
    await expect(page.getByText('忘记密码', { exact: false })).toHaveCount(0);
    await expect(page.getByText('AI', { exact: true })).toHaveCount(0);
    await page.locator('.auth-login-card .arco-input-inner-wrapper .arco-input').first().focus();
    const loginInputContract = await page.evaluate(() => {
      const controls = Array.from(
        document.querySelectorAll<HTMLElement>(
          '.auth-login-card .arco-input-inner-wrapper, .auth-login-card .arco-input-password',
        ),
      ).filter((control) => {
        const passwordControl = control.closest('.arco-input-password');
        return !passwordControl || passwordControl === control;
      });

      return controls.slice(0, 2).map((wrapper) => {
        const input = wrapper.querySelector<HTMLElement>('.arco-input');
        const wrapperStyle = window.getComputedStyle(wrapper);
        const inputStyle = input ? window.getComputedStyle(input) : null;

        return {
          wrapperBorderWidth: wrapperStyle.borderTopWidth,
          wrapperBackground: wrapperStyle.backgroundColor,
          wrapperBoxShadow: wrapperStyle.boxShadow,
          inputBorderWidth: inputStyle?.borderTopWidth || null,
          inputBackground: inputStyle?.backgroundColor || null,
          inputBoxShadow: inputStyle?.boxShadow || null,
          inputOutlineStyle: inputStyle?.outlineStyle || null,
          inputOutlineWidth: inputStyle?.outlineWidth || null,
        };
      });
    });
    expect(loginInputContract.length).toBeGreaterThanOrEqual(2);
    for (const control of loginInputContract) {
      expect(control.wrapperBorderWidth).toBe('1px');
      expect(control.wrapperBackground).toBe('rgb(255, 255, 255)');
      expect(control.inputBorderWidth).toBe('0px');
      expect(control.inputBackground).toBe('rgba(0, 0, 0, 0)');
      expect(control.inputBoxShadow).toBe('none');
      expect(control.inputOutlineStyle).toBe('none');
      expect(control.inputOutlineWidth).toBe('0px');
    }
    expect(loginInputContract.some((control) => control.wrapperBoxShadow !== 'none')).toBe(true);
    await page.screenshot({ path: join(artifactDir, 'login-desktop.png'), fullPage: true });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/login', { waitUntil: 'networkidle' });
    await expect(page.locator('.auth-login-card')).toBeVisible();
    await expect(page.getByRole('button', { name: '登录' })).toBeVisible();
    await page.screenshot({ path: join(artifactDir, 'login-mobile.png'), fullPage: true });

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  for (const pageMeta of authenticatedPages) {
    test(`${pageMeta.path} has unified shell and no runtime UI regression`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto(pageMeta.path, { waitUntil: 'networkidle' });

      await expect(page).toHaveURL(new RegExp(`${pageMeta.path.replace(/\//g, '\\/')}$`));
      await expectPageIdentity(page, pageMeta.title);
      await expectNoPageError(page);
      await expectProfessionalBackofficeSurface(page);
      await page.screenshot({ path: join(artifactDir, pageMeta.screenshot), fullPage: true });

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });
  }

  test('platform dashboard does not hard-code business module cards', async ({ page }) => {
    await signInAsAdmin(page);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/dashboard', { waitUntil: 'networkidle' });

    const dashboardContent = page.locator('.dashboard-page');
    await expect(dashboardContent).toBeVisible();
    await expect(dashboardContent.getByText('业务资产', { exact: false })).toHaveCount(0);
    await expect(dashboardContent.getByText('CMDB', { exact: false })).toHaveCount(0);
  });

  test('platform shell keeps horizontal compact preference baseline stable', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    const accessToken = await signInAsAdmin(page);
    const originalPreferences = await getCurrentUserPreferences(page, accessToken);

    try {
      await updateCurrentUserPreferences(page, accessToken, {
        ...originalPreferences,
        language: 'zh-CN',
        layoutMode: 'horizontal',
        densityMode: 'compact',
      });
      await installShellPreferences(page, { layoutMode: 'horizontal', densityMode: 'compact' });

      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto('/dashboard', { waitUntil: 'networkidle' });
      await expect(page.locator('html')).toHaveAttribute('data-pantheon-density', 'compact');
      await expect(page.locator('.app-shell--horizontal')).toBeVisible();
      await expect(page.locator('.app-shell__top-menu')).toBeVisible();
      await page.screenshot({ path: join(artifactDir, 'dashboard-horizontal-compact.png'), fullPage: true });

      await page.goto('/system/user', { waitUntil: 'networkidle' });
      await expect(page.locator('html')).toHaveAttribute('data-pantheon-density', 'compact');
      await expect(page.locator('.app-shell--horizontal')).toBeVisible();
      await expect(page.locator('.app-table')).toBeVisible();
      await page.screenshot({ path: join(artifactDir, 'system-user-horizontal-compact.png'), fullPage: true });
    } finally {
      await updateCurrentUserPreferences(page, accessToken, originalPreferences);
    }

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('system user page keeps filter and table patterns readable on phone and tablet', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.app-table')).toBeVisible();
    await expect(page.locator('.app-table__mobile-hint')).toBeVisible();
    await expectNoPageError(page);
    await page.screenshot({ path: join(artifactDir, 'system-user-phone.png'), fullPage: true });

    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.app-table')).toBeVisible();
    await page.screenshot({ path: join(artifactDir, 'system-user-tablet-portrait.png'), fullPage: true });

    await page.setViewportSize({ width: 1024, height: 768 });
    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.app-table')).toBeVisible();
    await page.screenshot({ path: join(artifactDir, 'system-user-tablet-landscape.png'), fullPage: true });

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('secondary verify modal uses natural localized copy', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/system/setting/basic', { waitUntil: 'networkidle' });
    await page.locator('.submit-bar').getByRole('button', { name: '保存' }).click();

    const verifyDialog = page.getByRole('dialog').filter({ has: page.getByText('敏感操作验证', { exact: true }) });
    await expect(verifyDialog).toBeVisible();
    await expect(verifyDialog.getByText('为了您的账号安全，请重新输入登录密码以继续操作。')).toBeVisible();

    const passwordInput = verifyDialog.locator('input').first();
    await expect(passwordInput).toHaveAttribute('placeholder', '请输入密码');
    await verifyDialog.getByRole('button', { name: '确定' }).click();
    await expect(verifyDialog.getByText('请输入密码')).toBeVisible();

    await page.screenshot({ path: join(artifactDir, 'secondary-verify-dialog-validation.png'), fullPage: true });
    expectOnlyAllowedRuntimeErrors(runtimeErrors, [/403 \(Forbidden\)/]);
  });

  test('system i18n create dialog shows localized required prompts', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/system/i18n', { waitUntil: 'networkidle' });
    await page.getByRole('button', { name: '新增' }).click();

    const createDialog = page.getByRole('dialog').filter({ has: page.getByText('新增翻译', { exact: true }) });
    await expect(createDialog).toBeVisible();
    await createDialog.getByRole('button', { name: '确定' }).click();

    await expect(createDialog.getByText('请输入所属模块')).toBeVisible();
    await expect(createDialog.getByText('请输入翻译键')).toBeVisible();
    await expect(createDialog.getByText('请输入内容')).toBeVisible();

    await page.screenshot({ path: join(artifactDir, 'system-i18n-create-validation.png'), fullPage: true });
    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });
});
