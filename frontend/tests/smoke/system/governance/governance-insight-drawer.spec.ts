import { expect, test, type Page } from '@playwright/test';
import { signInAsAdmin } from '../../helpers/auth';

async function navigateInShell(page: Page, path: string) {
  if (page.url() === 'about:blank') {
    await page.goto('/dashboard', { waitUntil: 'networkidle' });
  }
  await page.evaluate((nextPath) => {
    window.history.pushState({}, '', nextPath);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }, path);
  await expect(page).toHaveURL(new RegExp(`${path.replace(/\//g, '\\/')}$`));
}

const governancePages = [
  { path: '/system/user', title: '用户管理', button: '治理摘要', drawerTitle: '治理摘要' },
  { path: '/system/role', title: '角色管理', button: '授权摘要', drawerTitle: '授权摘要' },
  { path: '/system/menu', title: '菜单管理', button: '元数据摘要', drawerTitle: '元数据摘要' },
  { path: '/system/permission', title: '权限管理', button: '治理摘要', drawerTitle: '治理摘要' },
  { path: '/system/dept', title: '部门管理', button: '治理视角', drawerTitle: '治理视角' },
  { path: '/system/post', title: '岗位管理', button: '治理摘要', drawerTitle: '治理摘要' },
  { path: '/system/setting', title: '系统设置', button: '治理摘要', drawerTitle: '治理摘要' },
  { path: '/system/dict', title: '字典管理', button: '治理摘要', drawerTitle: '治理摘要' },
  { path: '/system/i18n', title: '国际化管理', button: '治理摘要', drawerTitle: '治理摘要' },
  { path: '/system/login-log', title: '登录日志', button: '审计摘要', drawerTitle: '审计摘要' },
  { path: '/system/session', title: '会话管理', button: '会话摘要', drawerTitle: '会话摘要' },
  { path: '/system/operation-log', title: '操作日志', button: '审计摘要', drawerTitle: '审计摘要' },
] as const;

test('governance insight opens as drawer without compressing main pages', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await signInAsAdmin(page);

  for (const item of governancePages) {
    await navigateInShell(page, item.path);
    await expect(page.getByRole('heading', { name: item.title })).toBeVisible();
    await expect(page.locator('.page-split-layout')).toHaveCount(0);
    const mainWidthBefore = await page.evaluate(() => {
      const target = document.querySelector<HTMLElement>(
        '.system-list__table-card, .filter-panel, .permission-workbench__tabs',
      );
      return Math.round(target?.getBoundingClientRect().width || 0);
    });
    expect(mainWidthBefore, item.path).toBeGreaterThan(700);

    await page.getByRole('button', { name: item.button }).first().click();

    const drawer = page.locator('.governance-insight-drawer');
    await expect(drawer).toBeVisible();
    await expect(drawer.getByText(item.drawerTitle, { exact: true }).first()).toBeVisible();
    await expect(page.locator('.page-split-layout')).toHaveCount(0);
    const mainWidthAfter = await page.evaluate(() => {
      const target = document.querySelector<HTMLElement>(
        '.system-list__table-card, .filter-panel, .permission-workbench__tabs',
      );
      return Math.round(target?.getBoundingClientRect().width || 0);
    });
    expect(mainWidthAfter, item.path).toBeGreaterThanOrEqual(mainWidthBefore - 4);
  }
});
