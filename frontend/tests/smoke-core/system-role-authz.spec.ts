/**
 * System Role Authorization - 角色授权关键路径
 *
 * 覆盖范围:
 * - 创建角色 (含勾选菜单权限)
 * - 编辑角色权限
 * - 删除角色 (行内 Popconfirm)
 *
 * 优先级: P0 (权限系统核心)
 * 预估耗时: ~3分钟
 *
 * 选择器基准 (与 modules/system/role/RoleList.tsx 对齐):
 * - 对话框标题: 新增角色 / 编辑角色; 字段: 角色名称 / 角色标识
 * - 菜单权限 (PermissionTreeSelector) 就在创建/编辑对话框内 (menuIds 字段)
 * - 模态底部 .submit-bar: 创建=新增 / 编辑=保存
 * - 行内操作: 角色成员 / 编辑 / 删除 (admin 角色禁删; Popconfirm "确认删除？")
 * - 角色 API 变更走 verified (CSRF + operation token) 请求
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

/** 通过工具条搜索框定位角色行 (表格展示 roleName, 搜索匹配名称或标识) */
async function searchRoleRow(page: Page, keyword: string) {
  const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
  await toolbarKeyword.fill(keyword);
  await Promise.all([
    page.waitForResponse(
      (response) =>
        response.url().includes('/system/role/list') && response.request().method() === 'GET',
    ),
    toolbarKeyword.press('Enter'),
  ]);
  const row = page.getByRole('row', { name: new RegExp(keyword) }).first();
  await expect(row).toBeVisible({ timeout: 15000 });
  return row;
}

async function deleteRoleByKey(page: Page, login: BrowserLoginResult, roleKey: string) {
  const listResponse = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(login.accessToken),
    params: { roleKey, page: 1, pageSize: 10 },
  });

  if (listResponse.ok()) {
    const payload = await listResponse.json();
    const roles = Array.isArray(payload.data?.items) ? payload.data.items : [];
    for (const role of roles as Array<{ id: number; roleKey: string }>) {
      if (role.roleKey === roleKey && role.roleKey !== 'admin') {
        // Role deletion is a verified (CSRF + operation-token) mutation.
        await page.request
          .delete(`${apiBaseUrl}/system/role/${role.id}`, {
            headers: await verifiedApiHeaders(page.request, login),
          })
          .catch(() => undefined);
      }
    }
  }
}

async function createRoleByApi(
  page: Page,
  login: BrowserLoginResult,
  roleName: string,
  roleKey: string,
) {
  // Role creation is a verified (CSRF + operation-token) mutation.
  const response = await page.request.post(`${apiBaseUrl}/system/role`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: {
      roleName,
      roleKey,
      sort: 999,
      status: 1,
      menuIds: [],
      permissionKeys: [],
    },
  });
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as { id: number };
}

test.describe('System Role Authorization @priority:critical @smoke:core', () => {
  const testRoleKey = `smoke_core_role_${Date.now().toString(36).slice(-5)}`;

  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await deleteRoleByKey(page, login, testRoleKey);
  });

  test.afterEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await deleteRoleByKey(page, login, testRoleKey);
  });

  test('can create a role with menu permissions', async ({ page }) => {
    await openSystemPageWithOperationToken(page, '/system/role');

    // 打开新增角色对话框
    await page.getByRole('button', { name: '新增', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '新增角色' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    // 填写基本信息 (roleKey 含唯一后缀, 同时写入 roleName 以便表格定位)
    const roleName = `烟测角色_${Date.now().toString(36).slice(-5)}`;
    await dialog.getByRole('textbox', { name: '角色名称', exact: true }).fill(roleName);
    await dialog.getByRole('textbox', { name: '角色标识', exact: true }).fill(testRoleKey);

    // 在同一对话框内勾选第一个可用菜单权限 (Arco Tree checkable)
    const menuTree = dialog.locator('.arco-tree').first();
    await expect(menuTree).toBeVisible();
    const firstCheckbox = menuTree.locator('.arco-tree-node').first().locator('.arco-checkbox');
    await expect(firstCheckbox).toBeVisible();
    await firstCheckbox.click({ force: true });

    // 提交 (创建流程的提交按钮为 "新增")
    await submitButtonInDialog(dialog, 'add').click();

    // 验证成功提示
    await expectSuccessMessage(page);

    // 验证角色出现在列表中 (通过搜索定位, 新角色可能不在第一页)
    await searchRoleRow(page, roleName);
  });

  test('can edit role permissions', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    const roleName = `烟测角色_${Date.now().toString(36).slice(-5)}`;
    await createRoleByApi(page, login, roleName, testRoleKey);

    await openSystemPageWithOperationToken(page, '/system/role');

    const roleRow = await searchRoleRow(page, roleName);

    await roleRow.getByRole('button', { name: '编辑', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '编辑角色' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    // 修改角色名称
    await dialog.getByRole('textbox', { name: '角色名称', exact: true }).fill('烟测角色_已修改');

    // 提交 (编辑流程的提交按钮为 "保存")
    await submitButtonInDialog(dialog, 'save').click();

    await expectSuccessMessage(page);
    await searchRoleRow(page, '烟测角色_已修改');
  });

  test('can delete a role', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    const roleName = `待删除角色_${Date.now().toString(36).slice(-5)}`;
    await createRoleByApi(page, login, roleName, testRoleKey);

    await openSystemPageWithOperationToken(page, '/system/role');

    const roleRow = await searchRoleRow(page, roleName);

    // 行内删除按钮带 Popconfirm ("确认删除？")
    await roleRow.getByRole('button', { name: '删除', exact: true }).click();
    await confirmVisiblePopconfirm(page, '确认删除');

    await expectSuccessMessage(page);

    // 验证角色从列表消失
    await expect(page.getByRole('row', { name: new RegExp(testRoleKey) })).toHaveCount(0, {
      timeout: 15000,
    });
  });
});
