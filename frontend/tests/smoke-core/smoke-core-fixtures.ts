import { expect, type Locator, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  installOperationToken,
  loginByApi,
  signInAsAdmin,
  verifiedApiHeaders,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';

type UserListItem = {
  id: number;
  username: string;
};

type TreeListItem = {
  id: number;
  children?: TreeListItem[];
};

async function deleteUserByUsername(page: Page, login: BrowserLoginResult, username: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/user/list`, {
    headers: authHeaders(login.accessToken),
    params: { username, page: 1, pageSize: 10 },
  });

  if (!response.ok()) {
    return;
  }

  const payload = await response.json();
  const users = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const user of users as UserListItem[]) {
    if (user.username === username) {
      // User deletion requires CSRF + operation-token verification on the backend.
      await page.request
        .delete(`${apiBaseUrl}/system/user/${user.id}`, {
          headers: await verifiedApiHeaders(page.request, login),
        })
        .catch(() => undefined);
    }
  }
}

async function deleteTreeItemByName(
  page: Page,
  login: BrowserLoginResult,
  listPath: '/system/dept/tree' | '/system/menu/tree',
  deletePath: '/system/dept' | '/system/menu',
  keyName: string,
  getName: (item: Record<string, unknown>) => string | undefined,
) {
  const response = await page.request.get(`${apiBaseUrl}${listPath}`, {
    headers: authHeaders(login.accessToken),
  });

  if (!response.ok()) {
    return;
  }

  const payload = await response.json();
  const items = Array.isArray(payload.data) ? payload.data : [];

  const walk = async (nodes: TreeListItem[]) => {
    for (const node of nodes) {
      if (getName(node as unknown as Record<string, unknown>) === keyName) {
        // Tree deletion requires CSRF + operation-token verification on the backend.
        await page.request
          .delete(`${apiBaseUrl}${deletePath}/${node.id}`, {
            headers: await verifiedApiHeaders(page.request, login),
          })
          .catch(() => undefined);
      }
      if (Array.isArray(node.children)) {
        await walk(node.children as TreeListItem[]);
      }
    }
  };

  await walk(items as TreeListItem[]);
}

async function deleteRoleByKey(page: Page, login: BrowserLoginResult, roleKey: string) {
  const response = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(login.accessToken),
    params: { roleKey, page: 1, pageSize: 10 },
  });

  if (!response.ok()) {
    return;
  }

  const payload = await response.json();
  const roles = Array.isArray(payload.data?.items) ? payload.data.items : [];
  for (const role of roles as Array<{ id: number; roleKey: string }>) {
    if (role.roleKey === roleKey && role.roleKey !== 'admin') {
      // Role deletion requires CSRF + operation-token verification on the backend.
      await page.request
        .delete(`${apiBaseUrl}/system/role/${role.id}`, {
          headers: await verifiedApiHeaders(page.request, login),
        })
        .catch(() => undefined);
    }
  }
}

export async function prepareUserSmokeFixture(page: Page, username: string) {
  const login = await loginByApi(page, adminCredentials);
  await deleteUserByUsername(page, login, username);
  return {
    cleanup: async () => {
      const cleanupLogin = await loginByApi(page, adminCredentials);
      await deleteUserByUsername(page, cleanupLogin, username);
    },
  };
}

export async function prepareDeptSmokeFixture(page: Page, deptName: string) {
  const login = await loginByApi(page, adminCredentials);
  await deleteTreeItemByName(
    page,
    login,
    '/system/dept/tree',
    '/system/dept',
    deptName,
    (item) => item.deptName as string | undefined,
  );
  return {
    cleanup: async () => {
      const cleanupLogin = await loginByApi(page, adminCredentials);
      await deleteTreeItemByName(
        page,
        cleanupLogin,
        '/system/dept/tree',
        '/system/dept',
        deptName,
        (item) => item.deptName as string | undefined,
      );
    },
  };
}

export async function prepareMenuSmokeFixture(page: Page, menuTitleKey: string) {
  const login = await loginByApi(page, adminCredentials);
  await deleteTreeItemByName(
    page,
    login,
    '/system/menu/tree',
    '/system/menu',
    menuTitleKey,
    (item) => item.titleKey as string | undefined,
  );
  return {
    cleanup: async () => {
      const cleanupLogin = await loginByApi(page, adminCredentials);
      await deleteTreeItemByName(
        page,
        cleanupLogin,
        '/system/menu/tree',
        '/system/menu',
        menuTitleKey,
        (item) => item.titleKey as string | undefined,
      );
    },
  };
}

export async function prepareRoleSmokeFixture(page: Page, roleKey: string) {
  const login = await loginByApi(page, adminCredentials);
  await deleteRoleByKey(page, login, roleKey);
  return {
    cleanup: async () => {
      const cleanupLogin = await loginByApi(page, adminCredentials);
      await deleteRoleByKey(page, cleanupLogin, roleKey);
    },
  };
}

/**
 * Signs in, opens a system page, and installs the operation token required by
 * backend-verified UI mutations (create/edit/batch operations). Returns the
 * admin access token for follow-up API assertions.
 */
export async function openSystemPageWithOperationToken(page: Page, path: string) {
  const accessToken = await signInAsAdmin(page);
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await installOperationToken(page, accessToken);
  return accessToken;
}

/**
 * Creates an entity through the API with the verified (CSRF + operation token)
 * headers the backend requires for mutations. Returns the parsed payload data.
 */
export async function createVerified(
  page: Page,
  path: string,
  login: BrowserLoginResult,
  data: Record<string, unknown>,
) {
  const response = await page.request.post(`${apiBaseUrl}${path}`, {
    headers: await verifiedApiHeaders(page.request, login),
    data,
  });
  const payload = await response.json();
  return {
    ok: response.ok() && payload.code === 200,
    data: payload.data as Record<string, unknown> | undefined,
  };
}

/**
 * Locates a form input inside an Arco form item by its visible label text.
 * Arco does not render `name` attributes on inputs; the visible label is the
 * stable anchor (same pattern as the full smoke suite's `formItem` helper).
 */
export function formInputByLabel(scope: Page | Locator, label: string) {
  return scope
    .locator('.arco-form-item')
    .filter({ has: scope.getByText(label, { exact: true }) })
    .first()
    .locator('input');
}

/**
 * Clicks the modal footer submit button rendered inside the shared
 * `.submit-bar` footer (新增 for create flows, 保存 for edits).
 */
export function submitButtonInDialog(dialog: Locator, action: 'add' | 'save') {
  return dialog.locator('.submit-bar').getByRole('button', {
    name: action === 'add' ? '新增' : '保存',
    exact: true,
  });
}

/**
 * Opens a list page and waits for its table card to render.
 */
export async function openSystemListPage(page: Page, path: string, title: string) {
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await expectVisiblePageTitle(page, title);
  await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });
}

/**
 * Runs one search-and-expand attempt for revealTreeRow: fills the tree-table
 * search box, then clicks inline expand toggles layer by layer until the
 * target row becomes visible. Returns the row locator, or null if the
 * attempt did not surface it.
 */
async function attemptTreeRowReveal(
  page: Page,
  searchInput: Locator,
  targetText: string,
  targetRow: Locator,
): Promise<Locator | null> {
  await searchInput.fill(targetText);
  await searchInput.press('Enter');

  if (await targetRow.isVisible().catch(() => false)) {
    return targetRow;
  }

  // 目标行可能藏在折叠的树节点内。Arco 树表格的展开控件是行内第一个无可见文本的
    // 按钮 (有名称的按钮是 编辑/删除 等操作), 逐层点击直到目标行出现。
  for (let depth = 0; depth < 3; depth += 1) {
    const expandToggles = page
      .locator('.arco-table-tr')
      .getByRole('button', { name: /^$/ })
      .filter({ visible: true });
    const toggleCount = await expandToggles.count();
    if (toggleCount === 0) {
      break;
    }
    for (let index = 0; index < toggleCount; index += 1) {
      await expandToggles
        .nth(index)
        .click({ force: true })
        .catch(() => undefined);
      if (await targetRow.isVisible().catch(() => false)) {
        return targetRow;
      }
    }
  }

  return null;
}

/**
 * Reveals a row inside an Arco tree table: searches for the target (the
 * backend returns the ancestor chain of the match), expands the matched
 * ancestor if needed, then resolves to the target row once visible.
 */
export async function revealTreeRow(page: Page, targetText: string, targetMatchOverride?: string) {
  const searchInput = page.locator('.search-toolbar__keyword input').first();

  // Arco Typography.Text ellipsis 会在 DOM 里把长文本截断为 "前缀...",
  // 因此可见文本匹配默认用 targetText; 菜单标题这类被截断的列需传入更短的前缀。
  const targetMatchText = targetMatchOverride ?? targetText;
  const targetRow = page.locator('.arco-table-tr').filter({ hasText: targetMatchText }).first();

  // 增删改会触发 refresh-topic 失效并重拉未过滤列表, 关键字搜索结果可能被
  // 覆盖; 整个 "搜索 → 展开" 流程重试, 等刷新沉淀后再过滤一次。
  for (let attempt = 0; attempt < 3; attempt += 1) {
    if (attempt > 0) {
      // 重试前固定退避: refresh-topic 失效重拉没有可观测的就绪信号,
      // 无法用 Playwright expect 替代 (SonarCloud typescript:S3516)。
      await page.waitForTimeout(800);
    }
    const revealed = await attemptTreeRowReveal(page, searchInput, targetText, targetRow);
    if (revealed) {
      return revealed;
    }
  }

  await expect(targetRow).toBeVisible({ timeout: 15000 });
  return targetRow;
}

/**
 * Expands a tree anchor row (e.g. the always-visible root) in the unfiltered
 * tree and returns the anchor row locator. 刚创建的子节点在未过滤树里跟随锚点行,
 * 直接展开锚点可避开创建后 refresh-topic 失效与关键字搜索的竞态。
 */
export async function expandTreeRow(page: Page, anchorText: string) {
  const anchorRow = page.locator('.arco-table-tr').filter({ hasText: anchorText }).first();
  await expect(anchorRow).toBeVisible({ timeout: 15000 });
  const toggle = anchorRow.getByRole('button', { name: /^$/ }).first();
  if ((await toggle.count()) > 0) {
    await toggle.click({ force: true }).catch(() => undefined);
  }
  return anchorRow;
}

async function expectVisiblePageTitle(page: Page, title: string) {
  // 无条件轮询组件可见性: 非幂等 UI 状态没有可等待的 promise,
  // 显式 sleep 会触发 SonarCloud typescript:S3516。
  await expect
    .poll(async () => {
      try {
        return await page.getByText(title, { exact: false }).filter({ visible: true }).count();
      } catch {
        return 0;
      }
    }, { timeout: 15000 })
    .toBeGreaterThan(0);
}

/**
 * Confirms the visible Arco popconfirm / modal confirm dialog by clicking 确定.
 */
export async function confirmVisiblePopconfirm(page: Page, title: RegExp | string) {
  const confirmPopup = page
    .locator(
      '.arco-popconfirm:visible, .arco-trigger-popup:visible, .arco-modal-confirm:visible, .arco-modal:visible',
    )
    .filter({ hasText: title })
    .last();
  await expect(confirmPopup).toBeVisible({ timeout: 5000 });
  await confirmPopup.getByRole('button', { name: '确定', exact: true }).click();
}

/**
 * Waits for the Arco success message toast.
 */
export async function expectSuccessMessage(page: Page, timeout = 10000) {
  await expect(page.locator('.arco-message-success').first()).toBeVisible({ timeout });
}
