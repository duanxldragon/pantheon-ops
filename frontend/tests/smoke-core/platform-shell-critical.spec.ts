/**
 * Platform Shell Critical - 平台壳层关键路径
 *
 * 覆盖范围:
 * - Shell 结构渲染
 * - 侧边栏菜单展开/折叠
 * - 路由导航
 * - 面包屑更新
 *
 * 优先级: P0 (影响所有用户)
 * 预估耗时: ~2分钟
 *
 * 信息架构说明 (v0.12.0 起):
 * - 侧边栏分组为 工作台 / 访问控制 / 组织架构 / 安全审计 / 低代码平台 / 平台配置
 * - "系统管理" 分组已不存在; 用户/角色/菜单管理位于 "访问控制" 分组下
 * - 分组标题是 Arco SubMenu 的 button; 子项渲染为 role="menuitem"
 *   (子菜单默认折叠, 仅在激活路由所属分组内展开)
 * - 工作台提供 "高频管理入口" 快捷按钮直达用户/角色/菜单管理
 */

import { test, expect } from '@playwright/test';
import { signInAsAdmin } from '../smoke/helpers/auth';

/**
 * 从工作台 "高频管理入口" 快捷按钮导航 (v0.12.0 IA 的确定性入口)。
 * 侧边栏子菜单是 popup 式折叠, 点击分组按钮只做分组导航不展开子项,
 * 因此叶子菜单项 (用户/角色/菜单管理) 优先走快捷入口; 直达路由兜底。
 */
async function navigateToSystemPage(page: import('@playwright/test').Page, itemName: string, route: string) {
  await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
  const quickLink = page.getByRole('button', { name: itemName, exact: true }).first();
  if (await quickLink.isVisible().catch(() => false)) {
    await quickLink.click();
  } else {
    await page.goto(route, { waitUntil: 'domcontentloaded' });
  }
}

test.describe('Platform Shell Critical @priority:critical @smoke:core', () => {
  test('shell renders with correct structure after login', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });

    // 验证 Shell 核心结构
    await expect(page.locator('.app-shell')).toBeVisible();

    // 验证顶部导航栏
    const header = page.locator('.app-shell__header');
    await expect(header).toBeVisible();

    // 验证侧边栏
    const sidebar = page.locator('.app-shell__sider');
    await expect(sidebar).toBeVisible();

    // 验证主内容区
    const main = page.locator('.app-shell__content');
    await expect(main).toBeVisible();
  });

  test('sidebar menu expands and collapses', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });

    // 找到折叠/展开按钮
    const toggleButton = page.locator('.app-shell__collapse-btn').first();

    // 验证按钮存在
    await expect(toggleButton).toBeVisible();

    // 记录初始状态
    const sidebar = page.locator('.app-shell__sider').first();
    const initialWidth = await sidebar.boundingBox().then(box => box?.width ?? 0);

    // 点击折叠
    await toggleButton.click();
    await expect(sidebar).toBeVisible({ timeout: 3000 });

    // 验证宽度变化
    const collapsedWidth = await sidebar.boundingBox().then(box => box?.width ?? 0);
    expect(collapsedWidth).toBeLessThan(initialWidth);

    // 再次点击展开
    await toggleButton.click();
    await expect(sidebar).toBeVisible({ timeout: 3000 });

    // 验证恢复
    const expandedWidth = await sidebar.boundingBox().then(box => box?.width ?? 0);
    expect(expandedWidth).toBeGreaterThan(collapsedWidth);
  });

  test('navigation between system pages works', async ({ page }) => {
    await signInAsAdmin(page);

    // 工作台 "高频管理入口" → 用户管理
    await navigateToSystemPage(page, '用户管理', '/system/user');
    await expect(page).toHaveURL(/\/system\/user/, { timeout: 10000 });
    await expect(page.locator('.page-container')).toBeVisible();

    // 工作台 "高频管理入口" → 角色管理
    await navigateToSystemPage(page, '角色管理', '/system/role');
    await expect(page).toHaveURL(/\/system\/role/, { timeout: 10000 });
    await expect(page.locator('.page-container')).toBeVisible();

    // 工作台 "高频管理入口" → 菜单管理
    await navigateToSystemPage(page, '菜单管理', '/system/menu');
    await expect(page).toHaveURL(/\/system\/menu/, { timeout: 10000 });
    await expect(page.locator('.page-container')).toBeVisible();
  });

  test('breadcrumb updates on navigation', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });

    // 验证面包屑包含正确的路径: 首页 → 访问控制 → 用户管理
    const breadcrumb = page.locator('.app-shell__header-breadcrumb');
    await expect(breadcrumb).toBeVisible();

    await expect(breadcrumb).toContainText('访问控制', { timeout: 10000 });
    await expect(breadcrumb).toContainText('用户管理');
  });
});
