import { test } from '../../fixtures/coverage';
import { expect, type Locator, type Page } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { apiBaseUrl, authHeaders, requestHeaders, signInAsAdmin } from '../helpers/auth';
import { runOptionalSmokeCleanup } from '../helpers/fixture-policy';
import { expectOnlyAllowedRuntimeErrors } from '../helpers/runtime-errors';
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

interface ShellPreferenceOptions {
  theme?: 'indigo' | 'emerald' | 'violet' | 'slate';
  language?: string;
  layoutMode?: 'vertical' | 'horizontal';
  densityMode?: 'comfortable' | 'compact';
}

interface FieldChrome {
  borderWidth: string;
  borderStyle: string;
  borderColor: string;
  borderRadius: string;
  backgroundColor: string;
  boxShadow: string;
}

interface TreeSelectLayoutContract {
  selectorDisplay: string;
  selectorPosition: string;
  selectorOverflow: string;
  inputPosition: string;
  mirrorOpacity: string;
  suffixDisplay: string;
}

async function installEmeraldThemePreference(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('pantheon_theme', 'emerald');
  });
}

async function installExplicitZhCNPreference(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('pantheon_lang', 'zh-CN');
    localStorage.setItem('pantheon_lang_explicit', '1');
  });
}

async function installShellPreferences(page: Page, options: ShellPreferenceOptions = {}) {
  const { language = 'zh-CN', layoutMode = 'vertical', densityMode = 'comfortable' } = options;
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

async function getDeptTreeItems(page: Page, accessToken: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/dept/tree`, {
    headers: authHeaders(accessToken),
    params: { sortField: 'sort', sortOrder: 'asc' },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? (payload.data as Array<Record<string, unknown>>) : [];
}

function flattenDeptTree(nodes: Array<Record<string, unknown>>): Array<Record<string, unknown>> {
  return nodes.flatMap((node) => [
    node,
    ...flattenDeptTree(
      Array.isArray(node.children) ? (node.children as Array<Record<string, unknown>>) : [],
    ),
  ]);
}

async function createDeptSeed(page: Page, accessToken: string) {
  const deptTree = flattenDeptTree(await getDeptTreeItems(page, accessToken));
  const rootDept = deptTree.find((item) => Boolean(item.isRoot));
  expect(rootDept).toBeTruthy();
  const deptName = `视觉巡检部门-${Date.now()}`;
  const response = await page.request.post(`${apiBaseUrl}/system/dept`, {
    headers: await requestHeaders(page, accessToken),
    data: {
      parentId: rootDept!.id,
      deptName,
      sort: 10,
      phone: '',
      email: '',
      status: 1,
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return {
    id: payload.data.id as number,
    deptName,
  };
}

async function deleteDeptSeed(page: Page, accessToken: string, deptId: number) {
  await page.request
    .delete(`${apiBaseUrl}/system/dept/${deptId}`, {
      headers: await requestHeaders(page, accessToken),
    })
    .catch(() => undefined);
}

async function createPostSeed(page: Page, accessToken: string) {
  const postCode = `VISUAL_POST_${Date.now()}`;
  const response = await page.request.post(`${apiBaseUrl}/system/post`, {
    headers: await requestHeaders(page, accessToken),
    data: {
      deptId: (await createDeptSeed(page, accessToken)).id,
      postCode,
      postName: `视觉巡检岗位-${postCode}`,
      sort: 10,
      status: 1,
      remark: 'visual smoke seed',
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return {
    id: payload.data.id as number,
    postCode,
    deptId: payload.data.deptId as number,
  };
}

async function deletePostSeed(page: Page, accessToken: string, postId: number) {
  await page.request
    .delete(`${apiBaseUrl}/system/post/${postId}`, {
      headers: await requestHeaders(page, accessToken),
    })
    .catch(() => undefined);
}

const authenticatedPages = [
  { path: '/dashboard', title: '工作台', screenshot: 'dashboard-desktop.png' },
  { path: '/system/user', title: '用户管理', screenshot: 'system-user-desktop.png' },
  { path: '/system/role', title: '角色管理', screenshot: 'system-role-desktop.png' },
  { path: '/system/permission', title: '权限管理', screenshot: 'system-permission-desktop.png' },
  { path: '/system/menu', title: '菜单管理', screenshot: 'system-menu-desktop.png' },
  { path: '/system/dept', title: '部门管理', screenshot: 'system-dept-desktop.png' },
  { path: '/system/post', title: '岗位管理', screenshot: 'system-post-desktop.png' },
  { path: '/system/dict', title: '字典管理', screenshot: 'system-dict-desktop.png' },
  { path: '/system/setting', title: '系统设置', screenshot: 'system-setting-desktop.png' },
  { path: '/system/i18n', title: '国际化管理', screenshot: 'system-i18n-desktop.png' },
  { path: '/system/login-log', title: '登录日志', screenshot: 'system-login-log-desktop.png' },
  { path: '/system/session', title: '会话管理', screenshot: 'system-session-desktop.png' },
  {
    path: '/system/security-event',
    title: '安全事件',
    screenshot: 'system-security-event-desktop.png',
  },
  {
    path: '/system/operation-log',
    title: '操作日志',
    screenshot: 'system-operation-log-desktop.png',
  },
  {
    path: '/system/modules',
    title: '模块注册表',
    identityTitle: '模块管理',
    screenshot: 'system-modules-desktop.png',
  },
  { path: '/system/generator', title: '模块生成器', screenshot: 'system-generator-desktop.png' },
  { path: '/auth/security', title: '安全中心', screenshot: 'auth-security-desktop.png' },
] as const;

const narrowSystemPages = authenticatedPages.filter((page) => page.path.startsWith('/system/'));

const sharedControlPages = [
  '/system/user',
  '/system/role',
  '/system/menu',
  '/system/post',
  '/system/dept',
  '/system/i18n',
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

async function expectNoPageError(page: Page) {
  for (const text of pageErrorTexts) {
    await expect(page.getByText(text, { exact: false })).toHaveCount(0);
  }
}

async function readFieldChrome(input: Locator) {
  await input.scrollIntoViewIfNeeded();
  await input.focus();
  return input.evaluate((node) => {
    const passwordSurface = node.closest('.arco-input-password');
    const fieldSurface =
      passwordSurface ||
      node.closest('.arco-input-inner-wrapper, .arco-textarea-wrapper') ||
      node.closest('.arco-select-view, .arco-tree-select-view, .arco-picker') ||
      node;
    const style = globalThis.getComputedStyle(fieldSurface as Element);
    return {
      borderWidth: style.borderTopWidth,
      borderStyle: style.borderTopStyle,
      borderColor: style.borderTopColor,
      borderRadius: style.borderTopLeftRadius,
      backgroundColor: style.backgroundColor,
      boxShadow: style.boxShadow,
    } satisfies FieldChrome;
  });
}

async function readSurfaceChrome(surface: Locator) {
  await surface.scrollIntoViewIfNeeded();
  return surface.evaluate((node) => {
    const style = globalThis.getComputedStyle(node as Element);
    return {
      borderWidth: style.borderTopWidth,
      borderStyle: style.borderTopStyle,
      borderColor: style.borderTopColor,
      borderRadius: style.borderTopLeftRadius,
      backgroundColor: style.backgroundColor,
      boxShadow: style.boxShadow,
    } satisfies FieldChrome;
  });
}

async function readTreeSelectLayoutContract(root: Locator) {
  await root.scrollIntoViewIfNeeded();
  return root.evaluate((node) => {
    const selector = node.querySelector('.arco-tree-select-view-selector');
    const input = node.querySelector('.arco-tree-select-view-input');
    const mirror = node.querySelector('.arco-tree-select-view-value-mirror');
    const suffix = node.querySelector('.arco-tree-select-suffix');
    const readStyle = (element: Element | null) => {
      if (!element) {
        return null;
      }
      const style = globalThis.getComputedStyle(element);
      return {
        display: style.display,
        position: style.position,
        overflow: style.overflow,
        opacity: style.opacity,
      };
    };
    return {
      selectorDisplay: readStyle(selector)?.display || '',
      selectorPosition: readStyle(selector)?.position || '',
      selectorOverflow: readStyle(selector)?.overflow || '',
      inputPosition: readStyle(input)?.position || '',
      mirrorOpacity: readStyle(mirror)?.opacity || '',
      suffixDisplay: readStyle(suffix)?.display || '',
    } satisfies TreeSelectLayoutContract;
  });
}

async function readVisibleSurfaceChromes(root: Locator) {
  return root.evaluate((node) => {
    const outerSelector = [
      '.arco-input-password',
      '.arco-input-number',
      '.arco-select-view',
      '.arco-tree-select-view',
      '.arco-picker',
      '.arco-textarea-wrapper',
      'textarea.arco-textarea',
      '.arco-input-inner-wrapper',
      'input.arco-input',
    ].join(',');
    const read = (element: HTMLElement) => {
      const style = globalThis.getComputedStyle(element);
      return {
        borderWidth: style.borderTopWidth,
        borderStyle: style.borderTopStyle,
        borderColor: style.borderTopColor,
        borderRadius: style.borderTopLeftRadius,
        backgroundColor: style.backgroundColor,
        boxShadow: style.boxShadow,
      };
    };
    const isOuterControl = (element: HTMLElement) => {
      if (
        element.matches('input.arco-input, textarea.arco-textarea') &&
        element.closest(
          '.arco-input-inner-wrapper, .arco-input-password, .arco-input-number, .arco-textarea-wrapper',
        )
      ) {
        return false;
      }
      if (element.classList.contains('arco-input-inner-wrapper')) {
        return !element.closest('.arco-input-password, .arco-input-number');
      }
      return true;
    };

    return Array.from(node.querySelectorAll<HTMLElement>(outerSelector))
      .filter((element) => {
        const rect = element.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0 && isOuterControl(element);
      })
      .map((element) => read(element));
  });
}

function expectMatchingFieldChrome(actual: FieldChrome, expected: FieldChrome) {
  expect(actual.borderWidth).toBe(expected.borderWidth);
  expect(actual.borderStyle).toBe(expected.borderStyle);
  expect(actual.borderColor).not.toBe(actual.backgroundColor);
  expect(actual.borderColor).not.toBe('rgba(0, 0, 0, 0)');
  expect(actual.borderRadius).toBe(expected.borderRadius);
  expect(actual.backgroundColor).toBe(expected.backgroundColor);
  expect(actual.boxShadow).not.toBe('none');
}

function expectMatchingSurfaceChrome(actual: FieldChrome, expected: FieldChrome) {
  expect(actual.borderWidth).toBe(expected.borderWidth);
  expect(actual.borderStyle).toBe(expected.borderStyle);
  expect(actual.borderRadius).toBe(expected.borderRadius);
  expect(actual.backgroundColor).toBe(expected.backgroundColor);
}

function expectTreeSelectLayoutContract(contract: TreeSelectLayoutContract) {
  expect(contract.selectorDisplay).toBe('flex');
  expect(contract.selectorPosition).toBe('relative');
  expect(contract.selectorOverflow).toBe('hidden');
  expect(contract.inputPosition).toBe('absolute');
  expect(contract.mirrorOpacity).toBe('0');
  expect(contract.suffixDisplay).toBe('flex');
}

async function expectProfessionalBackofficeSurface(page: Page) {
  await expect(page.locator('.app-shell__header')).toBeVisible();
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(page.locator('.page-panel').first()).toBeVisible();
  await expect(page.locator('.arco-layout-sider-dark')).toHaveCount(0);
  const usesHorizontalShell = (await page.locator('.app-shell--horizontal').count()) > 0;
  if (usesHorizontalShell) {
    await expect(page.locator('.app-shell__header-brand')).toBeVisible();
    await expect(page.locator('.app-shell__top-menu')).toBeVisible();
    await expect(page.locator('.app-shell__sider')).toHaveCount(0);
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

async function openSystemListPage(page: Page, path: string) {
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.app-shell__content')).toBeVisible();
  await expect(page.getByRole('button', { name: '新增', exact: true }).first()).toBeVisible({
    timeout: 20000,
  });
}

test.beforeAll(async () => {
  await ensureArtifactDir();
});

test.describe('backoffice UI visual acceptance', () => {
  test('login page keeps a professional authentication console on desktop and mobile', async ({
    page,
  }) => {
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
    const loginUsernameChrome = await readFieldChrome(
      page.getByPlaceholder(/请输入用户名|username/i),
    );
    const loginPasswordChrome = await readFieldChrome(
      page.getByPlaceholder(/请输入密码|password/i),
    );
    expect(loginUsernameChrome.borderWidth).toBe('1px');
    expect(loginUsernameChrome.borderStyle).toBe('solid');
    expect(loginUsernameChrome.backgroundColor).toBe('rgb(255, 255, 255)');
    expectMatchingFieldChrome(loginPasswordChrome, loginUsernameChrome);
    await page.screenshot({ path: join(artifactDir, 'login-desktop.png'), fullPage: true });

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/login', { waitUntil: 'networkidle' });
    await expect(page.locator('.auth-login-card')).toBeVisible();
    await expect(page.getByRole('button', { name: '登录' })).toBeVisible();
    await page.screenshot({ path: join(artifactDir, 'login-mobile.png'), fullPage: true });

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('reduced-motion preference suppresses login transitions and animations', async ({
    page,
  }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await installExplicitZhCNPreference(page);
    await page.goto('/login', { waitUntil: 'networkidle' });

    const motion = await page.getByRole('button', { name: '登录' }).evaluate((node) => {
      const style = globalThis.getComputedStyle(node);
      return {
        animationDuration: style.animationDuration,
        animationIterationCount: style.animationIterationCount,
        transitionDuration: style.transitionDuration,
      };
    });

    const durationsAreReduced = (value: string) =>
      value.split(',').every((duration) => Number.parseFloat(duration) <= 0.00001);

    expect(durationsAreReduced(motion.animationDuration)).toBe(true);
    expect(motion.animationIterationCount).toBe('1');
    expect(durationsAreReduced(motion.transitionDuration)).toBe(true);
  });

  test('login and user-management forms keep one shared input border contract', async ({
    page,
  }) => {
    const runtimeErrors = collectRuntimeErrors(page);

    await installExplicitZhCNPreference(page);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/login', { waitUntil: 'networkidle' });

    const baselineChrome = await readFieldChrome(page.getByPlaceholder(/请输入用户名|username/i));
    const loginPasswordChrome = await readFieldChrome(
      page.getByPlaceholder(/请输入密码|password/i),
    );
    expect(baselineChrome.borderWidth).toBe('1px');
    expect(baselineChrome.borderStyle).toBe('solid');
    expect(baselineChrome.backgroundColor).toBe('rgb(255, 255, 255)');
    expectMatchingFieldChrome(loginPasswordChrome, baselineChrome);

    await signInAsAdmin(page);
    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await page.getByRole('button', { name: '新增', exact: true }).click();

    const createDialog = page
      .getByRole('dialog')
      .filter({ has: page.getByText('新增用户', { exact: true }) });
    await expect(createDialog).toBeVisible();

    const createUsernameChrome = await readFieldChrome(
      createDialog
        .locator('.arco-form-item')
        .filter({ hasText: '用户名' })
        .locator('input')
        .first(),
    );
    const createPasswordChrome = await readFieldChrome(
      createDialog.locator('.arco-form-item').filter({ hasText: '密码' }).locator('input').first(),
    );
    expectMatchingFieldChrome(createUsernameChrome, baselineChrome);
    expectMatchingFieldChrome(createPasswordChrome, baselineChrome);
    await createDialog.getByRole('button', { name: '取消', exact: true }).click();
    await expect(createDialog).toBeHidden();

    await page
      .locator('.system-user-list__table-card')
      .getByRole('button', { name: '重置密码' })
      .first()
      .click();
    const resetDialog = page
      .getByRole('dialog')
      .filter({ has: page.getByText('重置用户密码', { exact: true }) });
    await expect(resetDialog).toBeVisible();

    const resetPasswordChrome = await readFieldChrome(
      resetDialog.locator('.arco-form-item').filter({ hasText: '新密码' }).locator('input').first(),
    );
    const resetConfirmPasswordChrome = await readFieldChrome(
      resetDialog
        .locator('.arco-form-item')
        .filter({ hasText: '确认新密码' })
        .locator('input')
        .first(),
    );
    expectMatchingFieldChrome(resetPasswordChrome, baselineChrome);
    expectMatchingFieldChrome(resetConfirmPasswordChrome, baselineChrome);
    await resetDialog.getByRole('button', { name: '取消', exact: true }).click();

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('add dialogs keep one shared control contract under the emerald theme', async ({ page }) => {
    test.setTimeout(60000);
    const runtimeErrors = collectRuntimeErrors(page);

    await installExplicitZhCNPreference(page);
    await installEmeraldThemePreference(page);
    const accessToken = await signInAsAdmin(page);
    const originalPreferences = await getCurrentUserPreferences(page, accessToken);
    await updateCurrentUserPreferences(page, accessToken, {
      ...originalPreferences,
      theme: 'emerald',
    });
    await page.setViewportSize({ width: 1440, height: 900 });
    try {
      await page.goto('/system/user', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      const userDialog = page
        .getByRole('dialog')
        .filter({ has: page.getByText('新增用户', { exact: true }) });
      await expect(userDialog).toBeVisible();

      const userNameInput = userDialog
        .locator('.arco-form-item')
        .filter({ hasText: '用户名' })
        .locator('input')
        .first();
      const userBaseline = await readFieldChrome(userNameInput);
      const userPassword = await readFieldChrome(
        userDialog.locator('.arco-form-item').filter({ hasText: '密码' }).locator('input').first(),
      );
      const userDept = await readSurfaceChrome(
        userDialog
          .locator('.arco-form-item')
          .filter({ hasText: '部门' })
          .locator('.arco-select-view')
          .first(),
      );
      const userStatus = await readSurfaceChrome(
        userDialog
          .locator('.arco-form-item')
          .filter({ hasText: '状态' })
          .locator('.arco-select-view')
          .first(),
      );
      expect(userBaseline.backgroundColor).toBe('rgb(255, 255, 255)');
      expectMatchingFieldChrome(userPassword, userBaseline);
      expect(userDept.borderWidth).toBe(userBaseline.borderWidth);
      expect(userDept.borderStyle).toBe(userBaseline.borderStyle);
      expect(userDept.borderRadius).toBe(userBaseline.borderRadius);
      expect(userDept.backgroundColor).toBe(userBaseline.backgroundColor);
      expect(userStatus.borderWidth).toBe(userBaseline.borderWidth);
      expect(userStatus.borderStyle).toBe(userBaseline.borderStyle);
      expect(userStatus.borderRadius).toBe(userBaseline.borderRadius);
      expect(userStatus.backgroundColor).toBe(userBaseline.backgroundColor);
      await userDialog.getByRole('button', { name: '取消', exact: true }).click();

      await openSystemListPage(page, '/system/menu');
      await page.getByRole('button', { name: '新增', exact: true }).click();
      const menuDialog = page
        .getByRole('dialog')
        .filter({ has: page.getByText('新增菜单', { exact: true }) });
      await expect(menuDialog).toBeVisible();
      const parentMenu = await readSurfaceChrome(
        menuDialog.locator('.arco-tree-select-view').first(),
      );
      const parentMenuLayout = await readTreeSelectLayoutContract(
        menuDialog.locator('.arco-tree-select').first(),
      );
      expect(parentMenu.borderWidth).toBe(userBaseline.borderWidth);
      expect(parentMenu.borderStyle).toBe(userBaseline.borderStyle);
      expect(parentMenu.borderRadius).toBe(userBaseline.borderRadius);
      expect(parentMenu.backgroundColor).toBe(userBaseline.backgroundColor);
      expectTreeSelectLayoutContract(parentMenuLayout);
      await menuDialog.getByRole('button', { name: '取消', exact: true }).click();
      await expect(menuDialog).toHaveCount(0);

      await openSystemListPage(page, '/system/role');
      await page.getByRole('button', { name: '新增', exact: true }).click();
      const roleDialog = page
        .getByRole('dialog')
        .filter({ has: page.getByText('新增角色', { exact: true }) });
      await expect(roleDialog).toBeVisible();

      const roleStatus = roleDialog
        .locator('.arco-form-item')
        .filter({ hasText: '状态' })
        .locator('.arco-select-view')
        .first();
      await roleStatus.click();
      const roleStatusFocused = await readSurfaceChrome(roleStatus);
      expect(roleStatusFocused.borderColor).not.toBe('rgb(22, 93, 255)');
      expect(roleStatusFocused.boxShadow).not.toContain('190, 218, 255');

      const authorizationTitleMetrics = await roleDialog.evaluate((root) => {
        return Array.from(
          root.querySelectorAll('.dialog-grid-card .arco-card-header-title .arco-typography'),
        )
          .slice(0, 3)
          .map((node) => {
            const style = globalThis.getComputedStyle(node);
            const lineHeight = Number.parseFloat(style.lineHeight || '0');
            const height = node.getBoundingClientRect().height;
            return {
              text: node.textContent?.trim() || '',
              lines: lineHeight > 0 ? Math.round(height / lineHeight) : 0,
            };
          });
      });
      expect(authorizationTitleMetrics).toHaveLength(3);
      for (const title of authorizationTitleMetrics) {
        expect(title.lines).toBeLessThanOrEqual(1);
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    } finally {
      await updateCurrentUserPreferences(page, accessToken, originalPreferences);
    }
  });

  test('edit dialogs and filter panels keep one shared control thickness contract', async ({
    page,
  }) => {
    test.setTimeout(120000);
    const runtimeErrors = collectRuntimeErrors(page);

    await installExplicitZhCNPreference(page);
    await installEmeraldThemePreference(page);
    const accessToken = await signInAsAdmin(page);
    const originalPreferences = await getCurrentUserPreferences(page, accessToken);
    await updateCurrentUserPreferences(page, accessToken, {
      ...originalPreferences,
      theme: 'emerald',
    });
    await page.setViewportSize({ width: 1440, height: 900 });
    let seededPost: { id: number; postCode: string } | null = null;
    try {
      await page.goto('/system/user', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: '新增', exact: true }).click();
      const createDialog = page
        .getByRole('dialog')
        .filter({ has: page.getByText('新增用户', { exact: true }) });
      await expect(createDialog).toBeVisible();
      const baselineChrome = await readFieldChrome(
        createDialog
          .locator('.arco-form-item')
          .filter({ hasText: '用户名' })
          .locator('input')
          .first(),
      );
      await createDialog.getByRole('button', { name: '取消', exact: true }).click();
      await expect(createDialog).toBeHidden();

      for (const path of sharedControlPages) {
        if (path === '/system/post') {
          seededPost = await createPostSeed(page, accessToken);
        }

        await openSystemListPage(page, path);
        const filterPanel = page.locator('.filter-panel').first();
        await expect(filterPanel).toBeVisible();
        const filterChromes = await readVisibleSurfaceChromes(filterPanel);
        expect(filterChromes.length, `${path} filter controls`).toBeGreaterThan(0);
        for (const controlChrome of filterChromes) {
          expectMatchingSurfaceChrome(controlChrome, baselineChrome);
        }

        await page.getByRole('button', { name: '新增', exact: true }).first().click();
        const createModuleDialog = page.getByRole('dialog').first();
        await expect(createModuleDialog).toBeVisible();
        const createChromes = await readVisibleSurfaceChromes(createModuleDialog);
        expect(createChromes.length, `${path} create dialog controls`).toBeGreaterThan(0);
        for (const controlChrome of createChromes) {
          expectMatchingSurfaceChrome(controlChrome, baselineChrome);
        }
        await createModuleDialog.getByRole('button', { name: '取消', exact: true }).click();
        await expect(createModuleDialog).toBeHidden();

        const editButton = page.getByRole('button', { name: '编辑', exact: true }).first();
        await expect(editButton, `${path} edit button`).toBeVisible({ timeout: 20000 });
        await editButton.click();
        const editDialog = page.getByRole('dialog').first();
        await expect(editDialog).toBeVisible();
        if (path === '/system/menu') {
          const editTreeSelectLayout = await readTreeSelectLayoutContract(
            editDialog.locator('.arco-tree-select').first(),
          );
          expectTreeSelectLayoutContract(editTreeSelectLayout);
        }
        const editChromes = await readVisibleSurfaceChromes(editDialog);
        expect(editChromes.length, `${path} edit dialog controls`).toBeGreaterThan(0);
        for (const controlChrome of editChromes) {
          expectMatchingSurfaceChrome(controlChrome, baselineChrome);
        }
        await editDialog.getByRole('button', { name: '取消', exact: true }).click();
        await expect(editDialog).toBeHidden();

        if (seededPost) {
          const currentSeed = seededPost;
          await runOptionalSmokeCleanup(`backoffice-ui:${path}:seeded-post`, async () => {
            await deletePostSeed(page, accessToken, currentSeed.id);
            await deleteDeptSeed(page, accessToken, currentSeed.deptId);
          });
          seededPost = null;
        }
      }

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    } finally {
      if (seededPost) {
        const currentSeed = seededPost;
        await runOptionalSmokeCleanup('backoffice-ui:final-seeded-post', async () => {
          await deletePostSeed(page, accessToken, currentSeed.id);
          await deleteDeptSeed(page, accessToken, currentSeed.deptId);
        });
      }
      await updateCurrentUserPreferences(page, accessToken, originalPreferences);
    }
  });

  for (const pageMeta of authenticatedPages) {
    test(`${pageMeta.path} has unified shell and no runtime UI regression`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto(pageMeta.path, { waitUntil: 'networkidle' });

      expectPagePathname(page, pageMeta.path);
      await expectPageIdentity(page, pageMeta.identityTitle ?? pageMeta.title);
      await expectNoPageError(page);
      await expectProfessionalBackofficeSurface(page);
      await page.screenshot({ path: join(artifactDir, pageMeta.screenshot), fullPage: true });

      expectOnlyAllowedRuntimeErrors(runtimeErrors);
    });
  }

  for (const pageMeta of narrowSystemPages) {
    test(`${pageMeta.path} keeps screenshot evidence stable at 1280w`, async ({ page }) => {
      const runtimeErrors = collectRuntimeErrors(page);
      await signInAsAdmin(page);

      await page.setViewportSize({ width: 1280, height: 900 });
      await page.goto(pageMeta.path, { waitUntil: 'networkidle' });

      expectPagePathname(page, pageMeta.path);
      await expectPageIdentity(page, pageMeta.identityTitle ?? pageMeta.title);
      await expectNoPageError(page);
      await expectProfessionalBackofficeSurface(page);
      await page.screenshot({
        path: join(artifactDir, pageMeta.screenshot.replace('-desktop.png', '-desktop-1280.png')),
        fullPage: true,
      });

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
      await expect(page.locator('.app-shell__tabs--horizontal')).toBeVisible();
      await expect(page.locator('.app-shell__top-menu')).toBeVisible();
      const horizontalShellOrder = await page.evaluate(() => {
        const tabs = document.querySelector<HTMLElement>('.app-shell__tabs--horizontal');
        const menu = document.querySelector<HTMLElement>('.app-shell__top-nav');
        return {
          tabsTop: tabs?.getBoundingClientRect().top ?? null,
          menuTop: menu?.getBoundingClientRect().top ?? null,
          menuBottom: menu?.getBoundingClientRect().bottom ?? null,
        };
      });
      expect(horizontalShellOrder.tabsTop).not.toBeNull();
      expect(horizontalShellOrder.menuTop).not.toBeNull();
      expect(horizontalShellOrder.menuBottom).not.toBeNull();
      expect(horizontalShellOrder.menuTop!).toBeLessThan(horizontalShellOrder.tabsTop!);
      expect(horizontalShellOrder.menuBottom!).toBeLessThanOrEqual(horizontalShellOrder.tabsTop!);
      await page.screenshot({
        path: join(artifactDir, 'dashboard-horizontal-compact.png'),
        fullPage: true,
      });

      await page.goto('/system/user', { waitUntil: 'networkidle' });
      await expect(page.locator('html')).toHaveAttribute('data-pantheon-density', 'compact');
      await expect(page.locator('.app-shell--horizontal')).toBeVisible();
      await expect(page.locator('.app-table')).toBeVisible();
      await page.screenshot({
        path: join(artifactDir, 'system-user-horizontal-compact.png'),
        fullPage: true,
      });
    } finally {
      await updateCurrentUserPreferences(page, accessToken, originalPreferences);
    }

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('system user page keeps filter and table patterns readable on phone and tablet', async ({
    page,
  }) => {
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
    await page.screenshot({
      path: join(artifactDir, 'system-user-tablet-portrait.png'),
      fullPage: true,
    });

    await page.setViewportSize({ width: 1024, height: 768 });
    await page.goto('/system/user', { waitUntil: 'networkidle' });
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.app-table')).toBeVisible();
    await page.screenshot({
      path: join(artifactDir, 'system-user-tablet-landscape.png'),
      fullPage: true,
    });

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('system setting overview and group pages keep navigation and forms readable on phone', async ({
    page,
  }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/system/setting', { waitUntil: 'networkidle' });
    await expect(page.locator('.setting-overview-page')).toBeVisible();
    await expect(page.locator('.setting-overview-page__anchor-strip')).toBeVisible();
    await expect(page.locator('.setting-group-workspace').first()).toBeVisible();
    await expect(page.locator('.setting-overview-page .side-rail-panel').first()).toBeVisible();
    await expectNoPageError(page);

    const overviewContract = await page.evaluate(() => {
      const nav = document.querySelector<HTMLElement>('.setting-overview-page__anchor-strip');
      const firstItem = document.querySelector<HTMLElement>('.setting-overview-page__anchor-item');
      const viewportWidth = globalThis.innerWidth;
      const bodyScrollWidth = document.documentElement.scrollWidth;
      const navStyle = nav ? globalThis.getComputedStyle(nav) : null;
      const firstRect = firstItem?.getBoundingClientRect();
      return {
        viewportWidth,
        bodyScrollWidth,
        navOverflowX: navStyle?.overflowX || '',
        navAutoFlow: navStyle?.gridAutoFlow || '',
        firstWidth: firstRect ? Math.round(firstRect.width) : 0,
        workspaceCount: document.querySelectorAll('.setting-group-workspace').length,
      };
    });

    expect(overviewContract.bodyScrollWidth).toBeLessThanOrEqual(
      overviewContract.viewportWidth + 1,
    );
    expect(overviewContract.navOverflowX).toBe('auto');
    expect(overviewContract.navAutoFlow).toBe('column');
    expect(overviewContract.firstWidth).toBeGreaterThanOrEqual(160);
    expect(overviewContract.workspaceCount).toBe(1);
    await page.screenshot({
      path: join(artifactDir, 'system-setting-overview-phone.png'),
      fullPage: true,
    });

    await page.goto('/system/setting/basic', { waitUntil: 'networkidle' });
    await expect(page.locator('.setting-group-page')).toBeVisible();
    await expect(page.locator('.setting-page__group-nav-grid')).toBeVisible();
    await expect(page.locator('.setting-page__config-card')).toBeVisible();
    await expectNoPageError(page);

    const groupContract = await page.evaluate(() => {
      const viewportWidth = globalThis.innerWidth;
      const bodyScrollWidth = document.documentElement.scrollWidth;
      const nav = document.querySelector<HTMLElement>('.setting-page__group-nav-grid');
      const navStyle = nav ? globalThis.getComputedStyle(nav) : null;
      const configCard = document.querySelector<HTMLElement>('.setting-page__config-card');
      const saveButton = Array.from(document.querySelectorAll<HTMLButtonElement>('button')).find(
        (button) => button.textContent?.includes('保存'),
      );
      const configRect = configCard?.getBoundingClientRect();
      const saveRect = saveButton?.getBoundingClientRect();
      return {
        viewportWidth,
        bodyScrollWidth,
        navOverflowX: navStyle?.overflowX || '',
        navAutoFlow: navStyle?.gridAutoFlow || '',
        configRight: configRect ? Math.round(configRect.right) : 0,
        saveRight: saveRect ? Math.round(saveRect.right) : 0,
      };
    });

    expect(groupContract.bodyScrollWidth).toBeLessThanOrEqual(groupContract.viewportWidth + 1);
    expect(groupContract.navOverflowX).toBe('auto');
    expect(groupContract.navAutoFlow).toBe('column');
    expect(groupContract.configRight).toBeLessThanOrEqual(groupContract.viewportWidth + 1);
    expect(groupContract.saveRight).toBeLessThanOrEqual(groupContract.viewportWidth + 1);
    await page.screenshot({
      path: join(artifactDir, 'system-setting-group-phone.png'),
      fullPage: true,
    });

    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('module generator keeps step density readable on narrow screens', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/system/generator', { waitUntil: 'networkidle' });

    await expect(page.locator('.generator-wizard-card')).toBeVisible();
    await expect(page.locator('.generator-wizard__steps')).toBeVisible();
    await expect(page.locator('.generator-wizard__step-card')).toHaveCount(4);
    await expect(page.locator('.generator-wizard-card .arco-form')).toBeVisible();
    await expectNoPageError(page);

    const mobileStepLayout = await page.evaluate(() => {
      return Array.from(document.querySelectorAll<HTMLElement>('.generator-wizard__step-card')).map(
        (card) => {
          const rect = card.getBoundingClientRect();
          return {
            top: rect.top,
            left: rect.left,
          };
        },
      );
    });
    expect(mobileStepLayout).toHaveLength(4);
    for (let index = 1; index < mobileStepLayout.length; index += 1) {
      expect(mobileStepLayout[index].top).toBeGreaterThan(mobileStepLayout[index - 1].top);
      expect(Math.abs(mobileStepLayout[index].left - mobileStepLayout[0].left)).toBeLessThanOrEqual(
        1,
      );
    }
    await expect
      .poll(async () =>
        page.evaluate(
          () => document.documentElement.scrollWidth - document.documentElement.clientWidth,
        ),
      )
      .toBeLessThanOrEqual(1);

    await page.screenshot({
      path: join(artifactDir, 'system-generator-phone.png'),
      fullPage: true,
    });
    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });

  test('secondary verify modal uses natural localized copy', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/system/setting/basic', { waitUntil: 'networkidle' });
    await page.locator('.submit-bar').getByRole('button', { name: '保存' }).click();

    const verifyDialog = page
      .getByRole('dialog')
      .filter({ has: page.getByText('敏感操作验证', { exact: true }) });
    await expect(verifyDialog).toBeVisible();
    await expect(
      verifyDialog.getByText('为了您的账号安全，请重新输入登录密码以继续操作。'),
    ).toBeVisible();

    const passwordInput = verifyDialog.locator('input').first();
    await expect(passwordInput).toHaveAttribute('placeholder', '请输入密码');
    await verifyDialog.getByRole('button', { name: '确定' }).click();
    await expect(verifyDialog.getByText('请输入密码')).toBeVisible();

    await page.screenshot({
      path: join(artifactDir, 'secondary-verify-dialog-validation.png'),
      fullPage: true,
    });
    expectOnlyAllowedRuntimeErrors(runtimeErrors, [/403 \(Forbidden\)/]);
  });

  test('system i18n create dialog shows localized required prompts', async ({ page }) => {
    const runtimeErrors = collectRuntimeErrors(page);
    await signInAsAdmin(page);

    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/system/i18n', { waitUntil: 'networkidle' });
    await page.getByRole('button', { name: '新增' }).click();

    const createDialog = page
      .getByRole('dialog')
      .filter({ has: page.getByText('新增翻译', { exact: true }) });
    await expect(createDialog).toBeVisible();
    await createDialog.getByRole('button', { name: '确定' }).click();

    await expect(createDialog.getByText('请输入所属模块')).toBeVisible();
    await expect(createDialog.getByText('请输入翻译键')).toBeVisible();
    await expect(createDialog.getByText('请输入内容')).toBeVisible();

    await page.screenshot({
      path: join(artifactDir, 'system-i18n-create-validation.png'),
      fullPage: true,
    });
    expectOnlyAllowedRuntimeErrors(runtimeErrors);
  });
});
