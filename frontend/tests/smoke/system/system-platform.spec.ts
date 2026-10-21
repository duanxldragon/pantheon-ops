import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import { registerSystemWorkspaceTaskDepthSmokeTests } from './system-workspace-task-depth';
import {
  adminCredentials,
  compactMainAreaPages,
  expectNoPageError,
  expectPageIdentityReady,
  expectVisiblePageTitle,
  fetchSettingGroupItems,
  formItem,
  openUserMenu,
  signInAsAdmin,
  signInWithUi,
  updateSettingGroup,
  withListViewOnlyViewer,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

registerSystemWorkspaceTaskDepthSmokeTests({
  expectVisiblePageTitle,
  expectPageIdentityReady,
  formItem,
});

test('governance and audit pages remove hero-heavy main-area blocks', async ({ page }) => {
  test.setTimeout(60000);
  for (const path of compactMainAreaPages) {
    await page.goto(path, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.system-page-hero')).toHaveCount(0);
    await expect(page.locator('.system-list__hero')).toHaveCount(0);
    await expect(
      page
        .locator(
          '.system-list__table-card, .permission-workbench__tabs, .org-structure, .dict-workbench',
        )
        .first(),
    ).toBeVisible();
  }
});

test('owning domain label smoke: post dict and module manager surface the correct eyebrow', async ({
  page,
}) => {
  test.setTimeout(60000);
  const pageEyebrows = [
    { path: '/system/post', title: '岗位管理', eyebrow: '系统域 / 组织治理', locale: 'zh-CN' },
    { path: '/system/dict', title: '字典管理', eyebrow: '系统域 / 配置治理', locale: 'zh-CN' },
    { path: '/system/modules', title: '模块注册表', eyebrow: '系统域 / 低代码', locale: 'zh-CN' },
    {
      path: '/system/post',
      title: 'Close post status, in-use blockers, and remediation actions in one workbench',
      eyebrow: 'System Domain / Organization',
      locale: 'en-US',
    },
  ] as const;

  for (const pageMeta of pageEyebrows) {
    if (pageMeta.locale === 'en-US') {
      await page.addInitScript((locale) => {
        localStorage.setItem('pantheon_lang', locale);
        localStorage.setItem('pantheon_lang_explicit', '1');
      }, pageMeta.locale);
    }
    await page.goto(pageMeta.path, { waitUntil: 'domcontentloaded' });
    await expectVisiblePageTitle(page, pageMeta.title);
    await expect(page.locator('.governance-summary-bar__eyebrow')).toContainText(pageMeta.eyebrow);
    for (const metricLabel of pageMeta.metricLabels ?? []) {
      await expect(
        page.locator('.governance-summary-bar__metric-label').getByText(metricLabel, {
          exact: true,
        }),
      ).toBeVisible();
    }
  }
});

test('module generator smoke: governance summary shows low-code ownership and progress metrics', async ({
  page,
}) => {
  await page.goto('/system/generator', { waitUntil: 'domcontentloaded' });

  await expectVisiblePageTitle(page, /模块生成(?:器|向导)/);
  await expect(page.locator('.governance-summary-bar__eyebrow')).toContainText('系统域 / 低代码');
  await expect(page.locator('.governance-summary-bar__title')).toContainText(
    '在受控治理约束内生成业务模块脚手架',
  );
  await expect(page.locator('.governance-summary-bar__metric-label')).toContainText([
    '步骤',
    '字段',
    '文件',
  ]);
});

test('platform smoke: lock screen keeps current route and opened tabs', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const originalItems = await fetchSettingGroupItems(page, accessToken, 'ui');
  const nextItems = originalItems.map((item) => ({
    settingKey: item.settingKey,
    settingValue: item.settingKey === 'ui.enable_tab_bar' ? 'true' : item.settingValue,
  }));

  try {
    const updateResponse = await updateSettingGroup(page, accessToken, 'ui', nextItems);
    expect(updateResponse.ok()).toBeTruthy();

    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await expectPageIdentityReady(page, '用户管理');
    await expect(page.locator('.app-shell__tabs [role="tab"]')).toHaveCount(2);

    await openUserMenu(page);
    await page.getByRole('menuitem', { name: /锁定屏幕|Lock Screen/i }).click();
    await expect(page.getByRole('dialog')).toContainText(/会话已锁定|Session Locked/);

    await page
      .getByPlaceholder(/请输入当前账号密码以解锁|Enter the current account password to unlock/)
      .fill('123456');
    await page.getByRole('button', { name: /解锁|Unlock/ }).click();

    await expect(page.getByRole('dialog')).toHaveCount(0);
    await expect(page).toHaveURL(/\/system\/user$/);
    await expectPageIdentityReady(page, '用户管理');
    await expect(page.locator('.app-shell__tabs [role="tab"]')).toHaveCount(2);
  } finally {
    await updateSettingGroup(page, accessToken, 'ui', originalItems);
  }
});

test('platform smoke: lock screen refreshes activity timestamp and blocks command palette while locked', async ({
  page,
}) => {
  await signInAsAdmin(page);

  await page.goto('/auth/security', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '安全中心');

  await page.goto('/system/profile', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '个人中心');

  const beforeLockActivity = await page.evaluate(() =>
    sessionStorage.getItem('pantheon_shell_last_activity_at'),
  );
  expect(beforeLockActivity).toBeTruthy();

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: /锁定屏幕|Lock Screen/i }).click();
  const lockDialog = page.getByRole('dialog');
  await expect(lockDialog).toContainText(/会话已锁定|Session Locked/);

  await page.keyboard.press(`${process.platform === 'darwin' ? 'Meta' : 'Control'}+K`);
  await expect(page.locator('.app-command')).toHaveCount(0);

  await page
    .getByPlaceholder(/请输入当前账号密码以解锁|Enter the current account password to unlock/)
    .fill('123456');
  await page.getByRole('button', { name: /解锁|Unlock/ }).click();
  await expect(lockDialog).toHaveCount(0);
  await expect(page).toHaveURL(/\/system\/profile$/);

  const afterUnlockActivity = await page.evaluate(() =>
    sessionStorage.getItem('pantheon_shell_last_activity_at'),
  );
  expect(afterUnlockActivity).toBeTruthy();
  expect(Number(afterUnlockActivity)).toBeGreaterThan(Number(beforeLockActivity));
});

test('platform + system/auth smoke: locked session times out, relogin notice appears, and security center shows session context', async ({
  page,
}) => {
  test.setTimeout(60_000);

  await page.goto('/system/profile', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '个人中心');

  await openUserMenu(page);
  await page.getByRole('menuitem', { name: /锁定屏幕|Lock Screen/i }).click();
  const lockDialog = page.getByRole('dialog');
  await expect(lockDialog).toContainText(/会话已锁定|Session Locked/);

  await page.evaluate(() => {
    sessionStorage.setItem('pantheon_shell_last_activity_at', String(Date.now() - 31 * 60 * 1000));
  });

  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('dialog')).toContainText(/会话已锁定|Session Locked/);
  await expect(page).toHaveURL(/\/login$/, { timeout: 25_000 });
  await expect(
    page.getByText(
      /当前账号因超过会话空闲时长被自动退出，请重新登录继续操作。|This account was signed out automatically after being idle for too long\. Sign in again to continue\./,
    ),
  ).toBeVisible();

  await signInWithUi(page, adminCredentials);
  await expect(page).toHaveURL(/\/dashboard$/);

  await page.goto('/auth/security', { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, '安全中心');
  await expect(page.getByText('在线会话', { exact: true })).toBeVisible();
  await expect(page.getByText('最近登录', { exact: true })).toBeVisible();
  await expect(page.getByText('当前设备', { exact: true }).first()).toBeVisible();
  await expect(page.locator('.arco-table')).toHaveCount(2);
  await expect(page.getByText('成功', { exact: true }).first()).toBeVisible();
});

test('module permission smoke: list-only role can view registry but cannot register or unregister modules', async ({
  page,
}) => {
  await withListViewOnlyViewer(
    page,
    {
      slug: 'module',
      cleanupTag: 'system-pages:module-viewer',
      roleName: '模块注册表只读烟测角色',
      nickname: '模块注册表只读烟测用户',
      listPermission: 'system:module:list',
      apiPermissions: [['/api/v1/lowcode/dynamic-modules', 'GET']],
      route: '/system/modules',
      identityLabel: '模块注册表',
    },
    async (viewerPage) => {
      await expect(viewerPage.getByRole('button', { name: '前往生成器' })).toHaveCount(0);
      await expect(viewerPage.getByRole('button', { name: '卸载' })).toHaveCount(0);
    },
  );
});

test('module manager smoke: auto-recycle module shows explicit lifecycle and purge guidance', async ({
  page,
}) => {
  await signInAsAdmin(page);
  await page.route(/\/api\/v1\/lowcode\/dynamic-modules(?:\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: [
          {
            id: 4101,
            name: 'business.tempautoqa',
            displayName: '临时 QA 模块',
            scope: 'business',
            source: 'generated',
            owner: 'codex',
            boundedContext: 'qa',
            summary: '用于回收交互验收',
            tableName: 'biz_temp_auto_qa',
            autoRecycle: true,
            status: 1,
            installedAt: '2026-05-20T10:00:00+08:00',
            builtIn: false,
          },
        ],
      }),
    });
  });

  await page.goto('/system/modules', { waitUntil: 'domcontentloaded' });
  await expectPageIdentityReady(page, '模块注册表');
  await expectNoPageError(page);

  const row = page.locator('.arco-table-tr').filter({ hasText: 'business.tempautoqa' }).first();
  await expect(row).toBeVisible();

  await row.getByRole('button', { name: '卸载', exact: true }).click();
  await expect(
    page.getByText('确认卸载该临时模块吗？卸载时会自动回收业务表 biz_temp_auto_qa。'),
  ).toBeVisible();

  const popconfirmCancel = page.getByRole('button', { name: '取消' }).last();
  await popconfirmCancel.click();

  await row.getByRole('button', { name: '彻底删除', exact: true }).click();
  const purgeDialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('彻底删除模块', { exact: true }) });
  await expect(purgeDialog).toBeVisible();
  await expect(
    purgeDialog.getByText('该模块已标记为临时模块，彻底删除时会自动回收业务表 biz_temp_auto_qa。'),
  ).toBeVisible();
  await expect(
    purgeDialog.getByText('临时模块的业务表会随彻底删除一起自动回收，这个动作不需要额外勾选。'),
  ).toBeVisible();
  await expect(purgeDialog.getByText('同时删除业务数据表', { exact: false })).toHaveCount(0);
});
