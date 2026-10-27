/**
 * System User CRUD - 用户管理关键路径
 *
 * 覆盖范围:
 * - 创建用户
 * - 编辑用户
 * - 批量禁用/启用 (UI 批量操作)
 * - 批量删除
 *
 * 优先级: P0 (数据破坏性操作)
 * 预估耗时: ~3分钟
 *
 * 选择器基准 (与 components/patterns/SystemRowActions、UserFormModal 对齐):
 * - 表单控件无 name 属性, 通过可见 label (用户名/昵称/密码) 定位
 * - 模态底部为 .submit-bar, 创建=新增 / 编辑=保存
 * - 行内操作: 详情/编辑/重置密码/禁用|启用 (Popconfirm 确认)
 * - 删除为批量操作: 勾选后点击 删除所选 → Popconfirm 确认
 * - UI 变更操作需要 operation token (openSystemPageWithOperationToken 安装)
 */

import { test, expect, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  loginByApi,
  verifiedApiHeaders,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';
import {
  confirmVisiblePopconfirm,
  expectSuccessMessage,
  openSystemPageWithOperationToken,
  submitButtonInDialog,
} from './smoke-core-fixtures';

async function deleteTestUser(page: Page, login: BrowserLoginResult, username: string) {
  const listResponse = await page.request.get(`${apiBaseUrl}/system/user/list`, {
    headers: authHeaders(login.accessToken),
    params: { username, page: 1, pageSize: 10 },
  });

  if (listResponse.ok()) {
    const payload = await listResponse.json();
    const users = Array.isArray(payload.data?.items) ? payload.data.items : [];
    for (const user of users as Array<{ id: number; username: string }>) {
      if (user.username === username) {
        // User deletion is a verified (CSRF + operation-token) mutation.
        await page.request
          .delete(`${apiBaseUrl}/system/user/${user.id}`, {
            headers: await verifiedApiHeaders(page.request, login),
          })
          .catch(() => undefined);
      }
    }
  }
}

async function createUserByApi(
  page: Page,
  login: BrowserLoginResult,
  username: string,
  nickname: string,
) {
  // User creation is a verified (CSRF + operation-token) mutation.
  const response = await page.request.post(`${apiBaseUrl}/system/user`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: {
      username,
      password: 'Test@123456',
      nickname,
      email: '',
      phone: '',
      status: 1,
    },
  });
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as { id: number };
}

test.describe('System User CRUD @priority:critical @smoke:core', () => {
  const testUsername = `smoke_core_user_${Date.now().toString(36).slice(-5)}`;

  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await deleteTestUser(page, login, testUsername);
  });

  test.afterEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await deleteTestUser(page, login, testUsername);
  });

  test('can create a new user', async ({ page }) => {
    await openSystemPageWithOperationToken(page, '/system/user');

    // 打开新增用户对话框
    await page.getByRole('button', { name: '新增', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '新增用户' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    // 填写表单 (Arco 表单无 name 属性, 通过可见 label 定位)
    await dialog.getByRole('textbox', { name: '用户名', exact: true }).fill(testUsername);
    await dialog.getByRole('textbox', { name: '密码', exact: true }).fill('Test@123456');
    await dialog.getByRole('textbox', { name: '昵称', exact: true }).fill('烟测用户');

    // 提交 (创建流程的提交按钮为 "新增")
    await submitButtonInDialog(dialog, 'add').click();

    // 验证成功提示
    await expectSuccessMessage(page);

    // 验证用户出现在列表中
    await expect(page.getByRole('row', { name: new RegExp(testUsername) }).first()).toBeVisible({
      timeout: 15000,
    });
  });

  test('can edit an existing user', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await createUserByApi(page, login, testUsername, '烟测用户');

    await openSystemPageWithOperationToken(page, '/system/user');

    const userRow = page.getByRole('row', { name: new RegExp(testUsername) }).first();
    await expect(userRow).toBeVisible({ timeout: 15000 });

    await userRow.getByRole('button', { name: '编辑', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '编辑用户' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    await dialog.getByRole('textbox', { name: '昵称', exact: true }).fill('烟测用户_已修改');

    // 提交 (编辑流程的提交按钮为 "保存")
    await submitButtonInDialog(dialog, 'save').click();

    await expectSuccessMessage(page);
    await expect(page.getByRole('row', { name: /烟测用户_已修改/ }).first()).toBeVisible({
      timeout: 15000,
    });
  });

  test('can batch disable, enable and delete a user', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await createUserByApi(page, login, testUsername, '烟测批量用户');

    await openSystemPageWithOperationToken(page, '/system/user');

    // 用搜索定位测试用户 (等待过滤后的列表响应, 避免与初始加载竞态)
    const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
    await toolbarKeyword.fill(testUsername);
    await Promise.all([
      page.waitForResponse(
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${testUsername}`) &&
          response.request().method() === 'GET',
      ),
      toolbarKeyword.press('Enter'),
    ]);

    const userRow = page.getByRole('row', { name: new RegExp(testUsername) }).first();
    await expect(userRow).toBeVisible({ timeout: 15000 });

    // 勾选测试用户 (行内复选框; 增删改后的 refresh-topic 重渲染会清空选择,
    // 点击重试直到选中态生效)
    const selectRow = async () => {
      const batchMeta = page.locator('.table-batch-action-bar__meta');
      for (let attempt = 0; attempt < 8; attempt += 1) {
        const checkbox = page
          .getByRole('row', { name: new RegExp(testUsername) })
          .first()
          .locator('.arco-checkbox')
          .first();
        await expect(checkbox).toBeVisible();
        await checkbox.click({ force: true });
        const selected = await batchMeta
          .textContent()
          .then((text) => (text ?? '').includes('已选 1 条'))
          .catch(() => false);
        if (selected) {
          return;
        }
      }
      await expect(batchMeta).toContainText('已选 1 条', { timeout: 5000 });
    };

    // 批量禁用 (Popconfirm: 确认批量禁用所选用户？)
    await selectRow();
    await page.getByRole('button', { name: '批量禁用', exact: true }).click();
    await confirmVisiblePopconfirm(page, '确认批量禁用所选用户');
    await expectSuccessMessage(page);

    // 批量启用 (Popconfirm: 确认批量启用所选用户？)
    await selectRow();
    await page.getByRole('button', { name: '批量启用', exact: true }).click();
    await confirmVisiblePopconfirm(page, '确认批量启用所选用户');
    await expectSuccessMessage(page);

    // 批量删除 (Popconfirm: 确认批量删除所选用户？)
    await selectRow();
    await page.getByRole('button', { name: '删除所选', exact: true }).click();
    await confirmVisiblePopconfirm(page, '确认批量删除所选用户');
    await expectSuccessMessage(page);

    // 验证用户从列表消失
    await expect(page.getByRole('row', { name: new RegExp(testUsername) })).toHaveCount(0, {
      timeout: 15000,
    });
  });
});
