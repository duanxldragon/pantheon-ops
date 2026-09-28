/**
 * System Menu Permission - 菜单权限联动关键路径
 *
 * 覆盖范围:
 * - 创建菜单
 * - 编辑菜单
 * - 删除菜单 (行内 Popconfirm)
 *
 * 优先级: P1 (权限系统基础)
 * 预估耗时: ~2分钟
 *
 * 选择器基准 (与 modules/system/menu/MenuList.tsx 对齐):
 * - 菜单节点以 titleKey 标识; 表格默认展示顶层树, 新节点需用搜索框定位
 * - 表单: 父级 TreeSelect / 标题键 (必填) / 路径 / 组件键 / 模块
 *   - 服务端校验: type=C 必须携带已注册组件键, module 需在注册域内
 *   - (与完整冒烟套件 create-child 用例一致: system.iam + system/menu/MenuList)
 * - 模态底部 .submit-bar: 创建=新增 / 编辑=保存
 * - 行内操作: 新增子菜单 / 编辑 / 删除 (Popconfirm "确认删除？")
 * - 菜单 API 变更走 verified (CSRF + operation token) 请求
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
import { listRegisteredComponentKeys } from '../../src/core/router/componentRegistry';
import {
  confirmVisiblePopconfirm,
  expectSuccessMessage,
  openSystemPageWithOperationToken,
  revealTreeRow,
  submitButtonInDialog,
} from './smoke-core-fixtures';

const menuComponentKey =
  listRegisteredComponentKeys().find((key) => key.endsWith('/menu/MenuList')) ??
  'system/menu/MenuList';

type MenuTreeNode = {
  id: number;
  titleKey: string;
  path: string;
  type: string;
  children?: MenuTreeNode[];
};

async function findAccessGroupMenuId(page: Page, accessToken: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/menu/tree`, {
    headers: authHeaders(accessToken),
    params: { scope: 'manage' },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const menus = (Array.isArray(payload.data) ? payload.data : []) as MenuTreeNode[];
  const accessGroup = menus.find((item) => item.path === '/system/access') ?? menus[0];
  expect(accessGroup).toBeTruthy();
  return accessGroup!.id;
}

async function deleteTestMenu(page: Page, login: BrowserLoginResult, titleKey: string) {
  const listResponse = await page.request.get(`${apiBaseUrl}/system/menu/tree`, {
    headers: authHeaders(login.accessToken),
    params: { scope: 'manage' },
  });

  if (listResponse.ok()) {
    const payload = await listResponse.json();
    const menus = Array.isArray(payload.data) ? payload.data : [];

    const findAndDelete = async (items: MenuTreeNode[]) => {
      for (const menu of items) {
        if (menu.titleKey === titleKey) {
          // Menu deletion is a verified (CSRF + operation-token) mutation.
          await page.request
            .delete(`${apiBaseUrl}/system/menu/${menu.id}`, {
              headers: await verifiedApiHeaders(page.request, login),
            })
            .catch(() => undefined);
        }
        if (Array.isArray(menu.children)) {
          await findAndDelete(menu.children);
        }
      }
    };

    await findAndDelete(menus);
  }
}

async function createMenuByApi(page: Page, login: BrowserLoginResult, parentMenuId: number, titleKey: string) {
  const suffix = Date.now().toString(36);
  // Menu creation is a verified (CSRF + operation-token) mutation.
  // 服务端校验 (menu_service.go validateMenuMeta): type=C 必须提供
  // component (注册键) + routeName + pagePerm。
  const response = await page.request.post(`${apiBaseUrl}/system/menu`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: {
      parentId: parentMenuId,
      titleKey,
      path: `/system/menu-smoke-${suffix}`,
      component: menuComponentKey,
      pagePerm: `system:menu-smoke-${suffix}:list`,
      perms: '',
      type: 'C',
      icon: 'menu',
      routeName: `system-menu-smoke-${suffix}`,
      module: 'system.iam',
      sort: 999,
      isVisible: 1,
      isCache: 0,
      isExternal: 0,
      activeMenu: '',
    },
  });
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as { id: number };
}

/** 通过搜索 + 展开折叠树定位菜单行 (新菜单可能是分组的子节点) */
function searchMenuRow(page: Page, titleKey: string) {
  // 标题列是 Typography.Text ellipsis, DOM 文本会被截断为 "前缀...",
  // 搜索请求本身按完整 titleKey 服务端过滤, 可见性断言用稳定前缀即可。
  return revealTreeRow(page, titleKey, 'smoke_core');
}

test.describe('System Menu Permission @priority:high @smoke:core', () => {
  const testMenuTitleKey = `system.menu.smoke_core_${Date.now().toString(36).slice(-5)}`;
  // 编辑用例会把 titleKey 改为 *_edited, 清理时需要覆盖两个键。
  const cleanupKeys = [testMenuTitleKey, `${testMenuTitleKey}_edited`];

  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    for (const key of cleanupKeys) {
      await deleteTestMenu(page, login, key);
    }
  });

  test.afterEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    for (const key of cleanupKeys) {
      await deleteTestMenu(page, login, key);
    }
  });

  test('can create a menu item', async ({ page }) => {
    await openSystemPageWithOperationToken(page, '/system/menu');

    // 打开新增菜单对话框
    await page.getByRole('button', { name: '新增', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '新增菜单' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    // 服务端要求 type=C: 标题键 + 模块 + 组件键 + 路由名 + 页面权限
    // (menu_service.go validateMenuMeta; 与完整套件 create-child 用例一致)
    const uniqueSuffix = Date.now().toString(36);
    await dialog.getByPlaceholder('例如：system.menu.example').fill(testMenuTitleKey);
    await dialog.getByPlaceholder('例如：system.iam / system.auth / platform / business.order').fill('system.iam');
    await dialog.getByPlaceholder('例如：business/cmdb/CMDBTypeList').fill(menuComponentKey);
    await dialog.getByPlaceholder('例如：system-example').fill(`system-menu-smoke-${uniqueSuffix}`);
    await dialog.getByPlaceholder('例如：system:example:list').first().fill(`system:menu:smoke:${uniqueSuffix}`);

    // 提交 (创建流程的提交按钮为 "新增")
    await submitButtonInDialog(dialog, 'add').click();

    // 验证成功提示
    await expectSuccessMessage(page);

    // 表格默认折叠, 通过搜索定位新菜单
    await searchMenuRow(page, testMenuTitleKey);
  });

  test('can edit a menu item', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    const parentMenuId = await findAccessGroupMenuId(page, login.accessToken);
    await createMenuByApi(page, login, parentMenuId, testMenuTitleKey);

    await openSystemPageWithOperationToken(page, '/system/menu');

    const menuRow = await searchMenuRow(page, testMenuTitleKey);

    await menuRow.getByRole('button', { name: '编辑', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '编辑菜单' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    const nextTitleKey = `${testMenuTitleKey}_edited`;
    await dialog.getByPlaceholder('例如：system.menu.example').fill(nextTitleKey);

    // 提交 (编辑流程的提交按钮为 "保存")
    await submitButtonInDialog(dialog, 'save').click();

    await expectSuccessMessage(page);
    await searchMenuRow(page, nextTitleKey);
  });

  test('can delete a menu item', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    const parentMenuId = await findAccessGroupMenuId(page, login.accessToken);
    await createMenuByApi(page, login, parentMenuId, testMenuTitleKey);

    await openSystemPageWithOperationToken(page, '/system/menu');

    const menuRow = await searchMenuRow(page, testMenuTitleKey);

    // 行内删除按钮带 Popconfirm ("确认删除？")
    await menuRow.getByRole('button', { name: '删除', exact: true }).click();
    await confirmVisiblePopconfirm(page, '确认删除');

    await expectSuccessMessage(page);

    // 验证菜单从树中消失 (清空搜索后重新查询)
    const searchInput = page.getByPlaceholder('按菜单标题键或路径搜索…');
    await searchInput.fill('');
    await searchInput.press('Enter');
    await expect(page.locator('.arco-table-tr').filter({ hasText: testMenuTitleKey })).toHaveCount(0, {
      timeout: 15000,
    });
  });
});
