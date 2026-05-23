import { test, expect } from '@playwright/test';

const BASE_URL = 'http://localhost:5173';

test.describe('Admin Smoke Test', () => {
  test('Login and navigate to dashboard', async ({ page }) => {
    // 1. 访问登录页
    await page.goto(`${BASE_URL}/login`);
    await expect(page).toHaveTitle(/Pantheon/);

    // 2. 输入账号密码
    await page.fill('input[placeholder*="username"]', 'admin');
    await page.fill('input[placeholder*="password"]', '123456');

    // 3. 点击登录
    await page.click('button[type="submit"]');

    // 4. 验证跳转到 Dashboard
    await expect(page).toHaveURL(`${BASE_URL}/dashboard`);
    await expect(page.locator('h2')).toContainText(/Dashboard/);

    // 5. 检查侧边栏核心菜单是否存在
    const sidebar = page.locator('aside');
    await expect(sidebar).toBeVisible();
    await expect(sidebar).toContainText(/Access & Permissions/);
    await expect(sidebar).toContainText(/System Configuration/);
  });

  test('Navigate to user management', async ({ page }) => {
    // 登录预处理
    await page.goto(`${BASE_URL}/login`);
    await page.fill('input[placeholder*="username"]', 'admin');
    await page.fill('input[placeholder*="password"]', '123456');
    await page.click('button[type="submit"]');

    // 点击访问控制 -> 用户管理
    await page.click('text=Access & Permissions');
    await page.click('text=Users');

    // 验证进入用户列表页
    await expect(page).toHaveURL(/.*system\/user/);
    const table = page.locator('table');
    await expect(table).toBeVisible();
  });
});
