/**
 * Auth Critical Path - 认证关键路径
 *
 * 覆盖范围:
 * - 用户登录/登出
 * - Token 刷新机制
 * - 会话状态管理
 *
 * 优先级: P0 (影响所有用户)
 * 预估耗时: ~2分钟
 */

import { test, expect } from '@playwright/test';
import { adminCredentials } from '../smoke/helpers/auth';

test.describe('Auth Critical Path @priority:critical @smoke:core', () => {
  test('user can login with valid credentials and logout', async ({ page }) => {
    // 访问登录页
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // 填写凭据
    await page.getByPlaceholder(/请输入用户名|username/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/请输入密码|password/i).fill(adminCredentials.password);

    // 提交登录（支持 Enter 键）
    await page.locator('input[type="password"]').press('Enter');

    // 验证登录成功：跳转到 dashboard
    await expect(page).toHaveURL(/\/dashboard$/, { timeout: 10000 });
    await expect(page.locator('.app-shell')).toBeVisible();

    // 验证用户信息显示
    const userDropdown = page.locator('.app-shell__user-trigger');
    await expect(userDropdown).toBeVisible();

    // 执行登出
    await userDropdown.click();
    const logoutButton = page.getByText(/退出登录|logout/i);
    await expect(logoutButton).toBeVisible();
    await logoutButton.click();

    // 验证登出成功：返回登录页
    await expect(page).toHaveURL(/\/login/, { timeout: 5000 });
  });

  test('login fails with invalid credentials', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // 使用错误的密码
    await page.getByPlaceholder(/请输入用户名|username/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/请输入密码|password/i).fill('wrong-password');
    await page.locator('input[type="password"]').press('Enter');

    // 验证错误提示
    await expect(page.locator('.arco-message-error, .arco-notification-error')).toBeVisible({
      timeout: 5000,
    });

    // 验证仍在登录页
    await expect(page).toHaveURL(/\/login/);
  });

  test('expired token redirects to login page', async ({ page, context }) => {
    // 1. 正常登录
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.getByPlaceholder(/请输入用户名|username/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/请输入密码|password/i).fill(adminCredentials.password);
    await page.locator('input[type="password"]').press('Enter');
    await expect(page).toHaveURL(/\/dashboard$/, { timeout: 10000 });

    // 2. 手动清除 token（模拟过期）
    await context.clearCookies();
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    // 3. 尝试访问需要认证的页面
    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });

    // 4. 验证被重定向到登录页
    await expect(page).toHaveURL(/\/login/, { timeout: 5000 });
  });

  test('session persists across page reload', async ({ page }) => {
    // 登录
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.getByPlaceholder(/请输入用户名|username/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/请输入密码|password/i).fill(adminCredentials.password);
    await page.locator('input[type="password"]').press('Enter');
    await expect(page).toHaveURL(/\/dashboard$/, { timeout: 10000 });

    // 刷新页面
    await page.reload({ waitUntil: 'domcontentloaded' });

    // 验证仍在 dashboard（未被踢回登录页）
    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.locator('.app-shell')).toBeVisible();
  });
});
