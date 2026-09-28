/**
 * System Department Operations - 部门树操作关键路径
 *
 * 覆盖范围:
 * - 添加根部门
 * - 编辑部门
 * - 删除部门 (行内 Popconfirm)
 *
 * 优先级: P1 (树形结构操作)
 * 预估耗时: ~2分钟
 *
 * 选择器基准 (与 modules/system/dept/DeptList.tsx 对齐):
 * - 对话框标题: 新增部门 / 编辑部门; 字段: 上级部门(TreeSelect, 默认 Pantheon Base) / 部门名称
 * - 模态底部 .submit-bar: 创建=新增 / 编辑=保存
 * - 行内操作: 编辑 / 删除 (Popconfirm "确认删除该部门？...")
 * - 部门树 API 节点含 deptName; 创建/删除走 verified (CSRF + operation token) 请求
 * - 服务端会为 parentId<=0 归位真实根节点; 表格默认折叠树, 用搜索框定位新行
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
  expandTreeRow,
  expectSuccessMessage,
  openSystemPageWithOperationToken,
  revealTreeRow,
  submitButtonInDialog,
} from './smoke-core-fixtures';

type DeptTreeNode = {
  id: number;
  deptName: string;
  parentId?: number;
  isRoot?: boolean;
  children?: DeptTreeNode[];
};

async function findRootDeptId(page: Page, accessToken: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/dept/tree`, {
    headers: authHeaders(accessToken),
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  const depts = (Array.isArray(payload.data) ? payload.data : []) as DeptTreeNode[];
  const root = depts.find((item) => item.isRoot || item.parentId === 0) ?? depts[0];
  expect(root).toBeTruthy();
  return root!.id;
}

async function deleteTestDept(page: Page, login: BrowserLoginResult, deptName: string) {
  const listResponse = await page.request.get(`${apiBaseUrl}/system/dept/tree`, {
    headers: authHeaders(login.accessToken),
  });

  if (listResponse.ok()) {
    const payload = await listResponse.json();
    const depts = Array.isArray(payload.data) ? payload.data : [];

    const findAndDelete = async (items: DeptTreeNode[]) => {
      for (const dept of items) {
        if (dept.deptName === deptName) {
          // Dept deletion is a verified (CSRF + operation-token) mutation.
          await page.request
            .delete(`${apiBaseUrl}/system/dept/${dept.id}`, {
              headers: await verifiedApiHeaders(page.request, login),
            })
            .catch(() => undefined);
        }
        if (Array.isArray(dept.children)) {
          await findAndDelete(dept.children);
        }
      }
    };

    await findAndDelete(depts);
  }
}

async function createDeptByApi(
  page: Page,
  login: BrowserLoginResult,
  rootDeptId: number,
  deptName: string,
) {
  // Dept creation is a verified (CSRF + operation-token) mutation.
  const response = await page.request.post(`${apiBaseUrl}/system/dept`, {
    headers: await verifiedApiHeaders(page.request, login),
    data: {
      parentId: rootDeptId,
      deptName,
      sort: 999,
      phone: '',
      email: '',
      status: 1,
    },
  });
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as { id: number };
}

/** 通过搜索 + 展开折叠树定位部门行 (新部门是根节点的子节点) */
function searchDeptRow(page: Page, deptName: string) {
  return revealTreeRow(page, deptName);
}

/**
 * 创建流程专用: 提交会同时触发列表刷新与 refresh-topic 失效重拉,
 * 关键字过滤后的结果可能立刻被未过滤重拉覆盖。直接在未过滤树里
 * 展开根节点定位新行, 并容忍刷新竞态 (行消失后等待重新出现)。
 */
async function revealCreatedDeptRow(page: Page, deptName: string) {
  const deptRow = page.locator('.arco-table-tr').filter({ hasText: deptName }).first();
  // 展开竞态兜底: refresh-topic 重拉没有可观测信号, 显式 sleep 会触发
  // SonarCloud typescript:S3516。展开钮是翻转式点击——已展开时再点会折叠回去,
  // 所以每次尝试都先查行可见性再决定是否展开, 点击后要求行在短窗口内出现;
  // 整个 "确保展开 → 找行" 流程交给 toPass 重试, 等刷新沉淀后自然收敛。
  await expect(async () => {
    if (await deptRow.isVisible().catch(() => false)) {
      return;
    }
    await expandTreeRow(page, 'Pantheon Base');
    await expect(deptRow).toBeVisible({ timeout: 2_000 });
  }).toPass({ timeout: 15_000 });
  return deptRow;
}

test.describe('System Department Operations @priority:high @smoke:core', () => {
  const testDeptName = `烟测部门_Core_${Date.now().toString(36).slice(-5)}`;

  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await deleteTestDept(page, login, testDeptName);
    await deleteTestDept(page, login, `${testDeptName}_已修改`);
  });

  test.afterEach(async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    await deleteTestDept(page, login, testDeptName);
    await deleteTestDept(page, login, `${testDeptName}_已修改`);
  });

  test('can create a root department', async ({ page }) => {
    await openSystemPageWithOperationToken(page, '/system/dept');

    // 打开新增部门对话框
    await page.getByRole('button', { name: '新增', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '新增部门' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    // 填写部门名称 (上级部门默认 Pantheon Base)
    await dialog.getByRole('textbox', { name: '部门名称', exact: true }).fill(testDeptName);

    // 提交 (创建流程的提交按钮为 "新增")
    await submitButtonInDialog(dialog, 'add').click();

    // 验证成功提示
    await expectSuccessMessage(page);

    // 表格树默认折叠, 展开根节点定位新部门 (容忍创建后的刷新竞态)
    await revealCreatedDeptRow(page, testDeptName);
  });

  test('can edit a department', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    const rootDeptId = await findRootDeptId(page, login.accessToken);
    await createDeptByApi(page, login, rootDeptId, testDeptName);

    await openSystemPageWithOperationToken(page, '/system/dept');

    const deptRow = await searchDeptRow(page, testDeptName);

    await deptRow.getByRole('button', { name: '编辑', exact: true }).click();
    const dialog = page.getByRole('dialog').filter({ hasText: '编辑部门' });
    await expect(dialog).toBeVisible({ timeout: 10000 });

    const nextName = `${testDeptName}_已修改`;
    await dialog.getByRole('textbox', { name: '部门名称', exact: true }).fill(nextName);

    // 提交 (编辑流程的提交按钮为 "保存")
    await submitButtonInDialog(dialog, 'save').click();

    await expectSuccessMessage(page);
    await searchDeptRow(page, nextName);
  });

  test('can delete a department', async ({ page }) => {
    const login = await loginByApi(page, adminCredentials);
    const rootDeptId = await findRootDeptId(page, login.accessToken);
    await createDeptByApi(page, login, rootDeptId, testDeptName);

    await openSystemPageWithOperationToken(page, '/system/dept');

    const deptRow = await searchDeptRow(page, testDeptName);

    // 行内删除按钮带 Popconfirm ("确认删除该部门？...")
    await deptRow.getByRole('button', { name: '删除', exact: true }).click();
    await confirmVisiblePopconfirm(page, '确认删除该部门');

    await expectSuccessMessage(page);

    // 验证部门从树中消失 (清空搜索后重新查询)
    const searchInput = page.getByPlaceholder('按部门名称搜索…');
    await searchInput.fill('');
    await searchInput.press('Enter');
    await expect(page.locator('.arco-table-tr').filter({ hasText: testDeptName })).toHaveCount(0, {
      timeout: 15000,
    });
  });
});
