/**
 * System Import/Export - 导入导出关键路径
 *
 * 覆盖范围:
 * - 用户数据导出 (CSV blob 下载)
 * - 导入模板下载
 * - 角色管理导出入口
 *
 * 优先级: P1 (数据批量操作)
 * 预估耗时: ~2分钟
 *
 * 行为基准 (与 modules/system/user/UserList.tsx、src/api/file.ts 对齐):
 * - 导出/下载模板/导入均通过 downloadFile 走浏览器 blob 下载,
 *   不弹对话框、不产生可见页面导航 → 断言 Playwright download 事件
 * - 用户导出文件名: system-user-export.csv; 模板: system-user-import-template.csv
 * - 入口按钮位于列表工具条: 导出 / 下载模板 / 导入
 */

import { test, expect } from '@playwright/test';
import { signInAsAdmin } from '../smoke/helpers/auth';

test.describe('System Import/Export @priority:high @smoke:core', () => {
  test('can export user data as csv download', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    const exportButton = page.getByRole('button', { name: '导出', exact: true }).first();
    await expect(exportButton).toBeVisible();
    await expect(exportButton).toBeEnabled();

    // 导出走 blob 下载, 通过 download 事件断言 (不等待网络响应)
    const downloadPromise = page.waitForEvent('download', { timeout: 15000 });
    await exportButton.click();
    const download = await downloadPromise;

    const fileName = download.suggestedFilename();
    expect(fileName).toMatch(/user|用户|export/i);
    expect(fileName).toMatch(/\.(csv|xlsx|xls)$/i);
  });

  test('can download user import template', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    const templateButton = page.getByRole('button', { name: '下载模板', exact: true }).first();
    await expect(templateButton).toBeVisible();
    await expect(templateButton).toBeEnabled();

    const downloadPromise = page.waitForEvent('download', { timeout: 15000 });
    await templateButton.click();
    const download = await downloadPromise;

    const fileName = download.suggestedFilename();
    expect(fileName).toMatch(/template|模板/i);
    expect(fileName).toMatch(/\.(csv|xlsx|xls)$/i);
  });

  test('export button is available on role management', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/system/role', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    // 验证导出按钮存在且可用
    const exportButton = page.getByRole('button', { name: '导出', exact: true }).first();
    await expect(exportButton).toBeVisible();
    await expect(exportButton).toBeEnabled();
  });
});
