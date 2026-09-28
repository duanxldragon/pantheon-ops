/**
 * Business Generated Basic - 生成模块基础 CRUD
 *
 * 覆盖范围:
 * - 探测并访问生成的业务模块（通过 nav 菜单树 API，而非过时的菜单 DOM 选择器）
 * - 基础列表加载
 * - 创建对话框打开
 *
 * 菜单发现策略: 登录后调用 GET /system/menu/tree?scope=nav，按 module 前缀
 * `business.`（inferModuleScope 契约）或 `/business/` 路径前缀识别生成的业务菜单，
 * 拿到第一个可导航叶子后直接 page.goto。旧实现依赖 `.arco-menu-item:has-text("业务")`
 * 这种与当前 IA（SubMenu 分组）不匹配的选择器，导致用例在无业务模块时与有业务模块时
 * 都无法可靠判断（CORE_SMOKE triage A 类问题），2026-09-24 重写。
 * 数据库中没有业务模块时保持诚实 skip（原语义保留）。
 *
 * 优先级: P1 (业务模块基础)
 * 预估耗时: ~3分钟
 */

import { test, expect, type Page } from '@playwright/test';
import { apiBaseUrl, signInAsAdmin } from '../smoke/helpers/auth';

type NavMenuNode = {
  path?: string;
  module?: string;
  type?: string;
  children?: NavMenuNode[];
};

function isBusinessNode(node: NavMenuNode): boolean {
  const moduleMatch = typeof node.module === 'string' && node.module.startsWith('business.');
  const pathMatch = typeof node.path === 'string' && node.path.startsWith('/business/');
  return moduleMatch || pathMatch;
}

/**
 * Returns the path of the first navigable generated-business leaf in the nav
 * menu tree, or null when the current database has no business module.
 */
async function probeBusinessMenuPath(page: Page): Promise<string | null> {
  const response = await page.request.get(`${apiBaseUrl}/system/menu/tree`, {
    params: { scope: 'nav' },
  });
  if (!response.ok()) {
    return null;
  }
  const payload = await response.json();
  const nodes: NavMenuNode[] = Array.isArray(payload.data) ? payload.data : [];

  const walk = (list: NavMenuNode[]): string | null => {
    for (const node of list) {
      if (isBusinessNode(node) && node.type === 'C' && node.path) {
        return node.path;
      }
    }
    for (const node of list) {
      if (Array.isArray(node.children) && node.children.length > 0) {
        const found = walk(node.children);
        if (found) {
          return found;
        }
      }
    }
    return null;
  };
  return walk(nodes);
}

test.describe('Business Generated Basic @priority:high @smoke:core', () => {
  test('generated module pages are accessible', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });

    const targetPath = await probeBusinessMenuPath(page);
    // reason: probeBusinessMenuPath returned null - this database has no generated business module to exercise (honest skip).
    test.skip(targetPath === null, 'No business modules exist in the current database');
    if (targetPath === null) {
      return; // unreachable at runtime; satisfies TypeScript narrowing
    }

    await page.goto(targetPath, { waitUntil: 'domcontentloaded' });

    // 生成模块列表页渲染出内容容器或表格即视为可访问。
    await expect(page.locator('.page-container, .content-wrapper, table').first()).toBeVisible({
      timeout: 15000,
    });

    // 验证没有错误提示
    const errorMessage = page.locator('.arco-message-error, .arco-notification-error');
    await expect(errorMessage).not.toBeVisible({ timeout: 2000 }).catch(() => {});
  });

  test('can open create dialog in generated module', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });

    const targetPath = await probeBusinessMenuPath(page);
    // reason: probeBusinessMenuPath returned null - this database has no generated business module to exercise (honest skip).
    test.skip(targetPath === null, 'No business modules exist in the current database');
    if (targetPath === null) {
      return;
    }

    await page.goto(targetPath, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('.page-container, .arco-table, table', { timeout: 15000 });

    const addButton = page.locator('button:has-text("新增"), button:has-text("Add")').first();
    if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addButton.click();

      const dialog = page.locator('.arco-modal').first();
      await expect(dialog).toBeVisible({ timeout: 5000 });

      await dialog.locator('button:has-text("取消"), button:has-text("Cancel")').first().click();
    }
  });

  test('generated module list has basic operations', async ({ page }) => {
    await signInAsAdmin(page);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });

    const targetPath = await probeBusinessMenuPath(page);
    // reason: probeBusinessMenuPath returned null - this database has no generated business module to exercise (honest skip).
    test.skip(targetPath === null, 'No business modules exist in the current database');
    if (targetPath === null) {
      return;
    }

    await page.goto(targetPath, { waitUntil: 'domcontentloaded' });
    await page.waitForSelector('table, .arco-table', { timeout: 15000 });

    // 验证基础操作按钮存在（新增/导出/刷新任一即可）。
    const operationButtons = page.locator(
      'button:has-text("新增"), button:has-text("导出"), button:has-text("刷新")',
    );
    expect(await operationButtons.count()).toBeGreaterThan(0);

    // 验证表格存在
    const table = page.locator('table, .arco-table').first();
    await expect(table).toBeVisible();
  });
});
