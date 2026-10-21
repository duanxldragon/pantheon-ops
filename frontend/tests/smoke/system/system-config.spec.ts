import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  closeExtraBrowserContext,
  createSharedAdminLogin,
  expectVisiblePageTitle,
  fetchSettingGroupItems,
  formItem,
  getCurrentUserPreferences,
  installClientSession,
  installOperationToken,
  installReloadStableReadCache,
  openUserMenu,
  readSettingValueFromApi,
  requestHeaders,
  shellBrandTextLocator,
  signInAsAdmin,
  updateCurrentUserPreferences,
  updateSettingGroup,
  verifiedHeaders,
  waitForRefreshBootstrap,
  withListViewOnlyViewer,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

test('setting page shows audit table only in audit group and removes governance drawer entry', async ({
  page,
}) => {
  await page.goto('/system/setting', { waitUntil: 'domcontentloaded' });

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
  await page.goto('/system/setting', { waitUntil: 'domcontentloaded' });

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
  await page.goto('/system/setting/security', { waitUntil: 'domcontentloaded' });

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

test('config high-sensitivity pages keep one summary container and no hero wall', async ({
  page,
}) => {
  test.setTimeout(60000);
  await page.goto('/system/modules', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '模块注册表');
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.module-manager-page__intro')).toHaveCount(0);
  await expect(page.locator('.module-manager-page__stats')).toHaveCount(0);
  await expect(
    page.locator('.module-manager-page__header-actions .arco-btn-primary'),
  ).toBeVisible();
  await expect(page.getByText('临时模块', { exact: true }).first()).toBeVisible();

  await page.goto('/system/generator', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, /模块生成(?:器|向导)/);
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.system-list__work-actions .arco-btn')).toBeVisible();
  await expect(page.locator('.generator-wizard__steps')).toBeVisible();
  await expect(page.locator('.generator-wizard__lifecycle-card')).toBeVisible();

  await page.goto('/system/i18n', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '国际化管理');
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.locator('.system-page-hero')).toHaveCount(0);
  await expect(page.locator('.system-list__hero')).toHaveCount(0);
  await expect(page.locator('.system-list__table-card')).toBeVisible();
});

test('dict workspace keeps the governance summary outside one tabbed task surface', async ({
  page,
}) => {
  await page.goto('/system/dict', { waitUntil: 'domcontentloaded' });

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

test('setting workspace keeps group navigation and config primary while audit stays secondary', async ({
  page,
}) => {
  await page.goto('/system/setting/audit', { waitUntil: 'domcontentloaded' });

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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'basic');
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

    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
    await expect(shellBrandTextLocator(page)).toHaveText(nextSiteName);
    await expect(page).toHaveTitle(nextSiteName);
  } finally {
    await updateSettingGroup(page, accessToken, 'basic', originalItems);
  }
});

test('setting smoke: security policy saves through setting page UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'security');
  const originalValue =
    originalItems.find((item) => item.settingKey === 'security.password_min_length')
      ?.settingValue ?? '6';
  const nextValue = originalValue === '6' ? '7' : '6';

  try {
    await page.goto('/system/setting/security', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await page.locator('input[role="spinbutton"]').first().fill(nextValue);
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(() => readSettingValueFromApi(page, accessToken, 'security', 'security.password_min_length'))
      .toBe(nextValue);
  } finally {
    await updateSettingGroup(page, accessToken, 'security', originalItems);
  }
});

test('setting smoke: setting audit row opens unified operation log detail', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'audit');
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

    await page.goto('/system/setting/audit', { waitUntil: 'domcontentloaded' });
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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'login');
  const originalValue =
    originalItems.find((item) => item.settingKey === 'login.max_failed_attempts')?.settingValue ??
    '5';
  const nextValue = originalValue === '5' ? '6' : '5';

  try {
    await page.goto('/system/setting/login', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await page.locator('input[role="spinbutton"]').first().fill(nextValue);
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(() => readSettingValueFromApi(page, accessToken, 'login', 'login.max_failed_attempts'))
      .toBe(nextValue);
  } finally {
    await updateSettingGroup(page, accessToken, 'login', originalItems);
  }
});

test('setting smoke: upload storage driver can be selected through setting page UI', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'upload');
  const currentValue =
    originalItems.find((item) => item.settingKey === 'upload.storage_driver')?.settingValue ??
    'local';
  const nextValue = currentValue === 'local' ? 's3' : 'local';
  const nextLabel = nextValue === 's3' ? 'S3 兼容对象存储' : '本地存储';

  try {
    await page.goto('/system/setting/upload', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await page.locator('.arco-select-view').first().click();
    await page.locator('.arco-select-option').filter({ hasText: nextLabel }).first().click();
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(() => readSettingValueFromApi(page, accessToken, 'upload', 'upload.storage_driver'))
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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'i18n');
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
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
    await page.reload({ waitUntil: 'domcontentloaded' });

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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'i18n');
  const currentValue =
    originalItems.find((item) => item.settingKey === 'i18n.default_language')?.settingValue ??
    'zh-CN';
  const nextValue = currentValue === 'zh-CN' ? 'en-US' : 'zh-CN';
  const nextLabel = nextValue === 'en-US' ? 'English' : '中文';

  try {
    await page.goto('/system/setting/i18n', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await page.locator('.arco-select-view').first().click();
    await page.locator('.arco-select-option').filter({ hasText: nextLabel }).first().click();
    await page.locator('.submit-bar button').last().click();
    await expect
      .poll(() => readSettingValueFromApi(page, accessToken, 'i18n', 'i18n.default_language'))
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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'i18n');
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
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });
    await page.reload({ waitUntil: 'domcontentloaded' });

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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'i18n');
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

    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
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

test('setting smoke: logout clears explicit theme and falls back to default theme', async ({
  page,
}) => {
  test.setTimeout(45000);
  const accessToken = await signInAsAdmin(page);
  const originalPreferences = await getCurrentUserPreferences(page, accessToken);
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'ui');
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

    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'ui');
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'ui.enable_tab_bar' ? 'false' : item.settingValue,
  }));

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'ui', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'ui');
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

    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      localStorage.removeItem('pantheon_theme');
    });
    await page.reload({ waitUntil: 'domcontentloaded' });

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
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'upload');
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

test('setting permission smoke: list-only role can view page but cannot save or refresh', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'setting',
      cleanupTag: 'system-pages:setting-viewer',
      roleName: '系统设置只读烟测角色',
      nickname: '系统设置只读烟测用户',
      listPermission: 'system:setting:list',
      apiPermissions: [['/api/v1/system/setting/list', 'GET']],
      route: '/system/setting/basic',
      identityLabel: '系统设置',
    },
    async (viewerPage) => {
      const settingPanel = viewerPage.locator('.setting-page__config-card');
      await expect(settingPanel.getByRole('button', { name: '刷新设置缓存' })).toBeDisabled();
      await expect(
        settingPanel.locator('.submit-bar').getByRole('button', { name: '保存' }),
      ).toBeDisabled();
      await expect(
        settingPanel.locator('.submit-bar').getByRole('button', { name: '取消' }),
      ).toBeEnabled();
    },
  );
});

test('dict permission smoke: list-only role can view page but cannot mutate config', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'dict',
      cleanupTag: 'system-pages:dict-viewer',
      roleName: '字典只读烟测角色',
      nickname: '字典只读烟测用户',
      listPermission: 'system:dict:list',
      apiPermissions: [
        ['/api/v1/system/dict/type/list', 'GET'],
        ['/api/v1/system/dict/item/list', 'GET'],
      ],
      route: '/system/dict',
      identityLabel: '字典管理',
    },
    async (viewerPage) => {
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
    },
  );
});

test('refresh sync smoke: setting page auto-updates across isolated contexts', async ({
  browser,
  page,
}) => {
  test.setTimeout(45000);
  const adminLogin = await createSharedAdminLogin(page);
  const accessToken = adminLogin.accessToken;
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'basic');
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
    await syncPage.goto('/system/setting/basic', { waitUntil: 'domcontentloaded' });
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
    await syncPage.goto('/system/dict', { waitUntil: 'domcontentloaded' });
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
