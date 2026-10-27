import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  clickVisibleConfirmButton,
  clickVisibleRowAction,
  createDeptByApi,
  createUserByApi,
  deleteDeptByName,
  deleteMenuById,
  deleteRoleByIdWithHeaders,
  deleteRoleByKey,
  deleteUserByIdWithHeaders,
  deleteUserByUsername,
  expectNoPageError,
  expectPageIdentityReady,
  expectVisiblePageTitle,
  fetchManageMenuTree,
  findUserByUsername,
  flattenDeptTreeNodes,
  flattenManageMenus,
  getDeptTree,
  getFirstActiveRole,
  getRoleByKey,
  installClientSession,
  installOperationToken,
  loginByApi,
  menuComponentKey,
  openUserMenu,
  runOptionalSmokeCleanup,
  signInAsAdmin,
  verifiedApiHeaders,
  waitForOkApiResponse,
  waitForVisibleRowByText,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

test('user page keeps list workflow primary without governance drawer entry', async ({ page }) => {
  await page.goto('/system/user', { waitUntil: 'domcontentloaded' });

  await expectVisiblePageTitle(page, '用户管理');
  await expect(page.locator('.system-user-list__hero')).toHaveCount(0);
  await expect(page.locator('.governance-summary-bar')).toBeVisible();
  await expect(page.getByRole('button', { name: '治理摘要' })).toHaveCount(1);
  await expect(page.locator('.governance-insight-drawer')).toHaveCount(0);
  await expect(page.locator('.system-list__table-card')).toBeVisible();
  await expect(page.locator('.table-batch-action-bar__prefix-actions')).toBeVisible();
  await expect(
    page.locator('.table-batch-action-bar__prefix-actions').getByText('新增'),
  ).toBeVisible();
  await expect(
    page.locator('.table-batch-action-bar__prefix-actions').getByText('导入'),
  ).toBeVisible();
  await expect(
    page.locator('.table-batch-action-bar__prefix-actions').getByText('导出'),
  ).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '状态' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '部门' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '岗位' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '角色' })).toBeVisible();
  await expect(page.getByRole('columnheader', { name: '邮箱' })).toBeVisible();

  const firstRowActions = page.locator('.system-user-list__row-actions').first();
  await expect(firstRowActions.getByRole('button', { name: '详情' })).toBeVisible();
  await expect(firstRowActions.getByRole('button', { name: '编辑' })).toBeVisible();
  await expect(firstRowActions.getByRole('button', { name: '重置密码' })).toBeVisible();
  await expect(firstRowActions.getByRole('button', { name: '删除' })).toHaveCount(0);
});

test('menu smoke: create dialog uses tree parent selector', async ({ page }) => {
  await signInAsAdmin(page);
  await page.goto('/system/menu', { waitUntil: 'domcontentloaded' });

  await page.getByRole('button', { name: '新增', exact: true }).click();
  const dialog = page
    .getByRole('dialog')
    .filter({ has: page.getByText('新增菜单', { exact: true }) });
  await expect(dialog).toBeVisible();
  await expect(dialog.locator('.arco-tree-select').first()).toBeVisible();
});

test('menu smoke: create child action preselects clicked parent', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const menus = flattenManageMenus(await fetchManageMenuTree(page, accessToken));
  const parentMenu =
    menus.find((item) => item.path === '/system/access') ??
    menus.find((item) => item.path && item.type !== 'F');
  expect(parentMenu).toBeTruthy();

  let createdMenuId: number | null = null;
  try {
    await page.goto('/system/menu', { waitUntil: 'domcontentloaded' });

    const parentRow = page
      .locator('.arco-table-tr')
      .filter({ hasText: parentMenu?.path || '' })
      .first();
    await expect(parentRow).toBeVisible();
    await parentRow.getByRole('button', { name: '新增子菜单', exact: true }).click();

    const dialog = page
      .getByRole('dialog')
      .filter({ has: page.getByText('新增菜单', { exact: true }) });
    await expect(dialog).toBeVisible();
    await expect(dialog.locator('.arco-tree-select').first()).toContainText('/system/access');

    const uniqueSuffix = Date.now();
    await dialog
      .getByPlaceholder('例如：system.menu.example')
      .fill(`system.menu.child.${uniqueSuffix}`);
    await dialog
      .getByPlaceholder('例如：/system/example')
      .fill(`/system/menu-child-${uniqueSuffix}`);
    await dialog.getByPlaceholder('例如：business/cmdb/CMDBTypeList').fill(menuComponentKey);
    await dialog.getByPlaceholder('例如：system-example').fill(`system-menu-child-${uniqueSuffix}`);
    await dialog
      .getByPlaceholder('例如：system.iam / system.auth / platform / business.order')
      .fill('system.iam');
    await dialog
      .getByPlaceholder('例如：system:example:list')
      .nth(0)
      .fill(`system:menu:child:${uniqueSuffix}`);

    const createResponsePromise = page.waitForResponse(
      (response) =>
        response.url().includes('/api/v1/system/menu') && response.request().method() === 'POST',
    );
    await dialog.locator('.submit-bar').getByRole('button', { name: '新增', exact: true }).click();
    const createResponse = await createResponsePromise;
    expect(createResponse.ok()).toBeTruthy();
    const requestBody = createResponse.request().postDataJSON() as { parentId: number };
    expect(requestBody.parentId).toBe(parentMenu?.id);
    const payload = await createResponse.json();
    expect(payload.code).toBe(200);
    createdMenuId = Number(payload.data?.id || 0) || null;
  } finally {
    await deleteMenuById(page, accessToken, createdMenuId);
  }
});

test('auth smoke: logout sends revoke request without stale invalid-session prompt', async ({
  page,
}) => {
  const tokens = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, tokens);

  let captureAuthFailures = false;
  const authFailures: string[] = [];

  page.on('response', async (response) => {
    if (
      !captureAuthFailures ||
      !response.url().includes('/api/v1/') ||
      response.url().endsWith('/auth/logout')
    ) {
      return;
    }
    try {
      const payload = await response.json();
      if (!payload || typeof payload !== 'object') {
        return;
      }
      const code = 'code' in payload ? payload.code : undefined;
      const message = 'message' in payload ? payload.message : undefined;
      const isAuthFailure =
        code === 401 ||
        message === 'session.invalid' ||
        (typeof message === 'string' && message.startsWith('token.'));
      if (isAuthFailure) {
        authFailures.push(
          `${response.request().method()} ${response.url()} -> ${String(message || code)}`,
        );
      }
    } catch {
      // ignore non-json responses
    }
  });

  await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
  captureAuthFailures = true;
  const logoutResponsePromise = page.waitForResponse((response) =>
    response.url().includes('/api/v1/auth/logout'),
  );
  await openUserMenu(page);
  await page.getByRole('menuitem', { name: /退出登录|Sign out|Logout/ }).click();
  const logoutPayload = await (await logoutResponsePromise).json();

  await expect(page).toHaveURL(/\/login$/);
  // Wait for the login page to settle (all in-flight requests done) so any
  // stray auth-failure message would have been rendered before the negative
  // assertions below — replaces a fixed 1s delay.
  await page.waitForLoadState('domcontentloaded').catch(() => {});
  expect(logoutPayload.code).toBe(200);
  await expect(
    page.locator('.arco-message').filter({ hasText: /无效会话|session\.invalid|token\./i }),
  ).toHaveCount(0);
  expect(authFailures).toEqual([]);
});

test('auth smoke: login page shows idle-timeout notice once', async ({ page }) => {
  test.setTimeout(20000);
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  
  // Wait for i18n to load
  await expect(page.getByRole('button', { name: '登录' })).toBeVisible({ timeout: 10000 });
  
  await page.evaluate(() => {
    sessionStorage.setItem('pantheon_login_notice', 'session.idle_timeout');
  });

  await page.reload({ waitUntil: 'domcontentloaded' });
  
  // Wait for i18n to load after reload
  await expect(page.getByRole('button', { name: '登录' })).toBeVisible({ timeout: 10000 });
  
  await expect(
    page.getByText('当前账号因超过会话空闲时长被自动退出，请重新登录继续操作。', { exact: true }),
  ).toBeVisible({ timeout: 10000 });

  await page.reload({ waitUntil: 'domcontentloaded' });
  
  // Wait for i18n to load after second reload
  await expect(page.getByRole('button', { name: '登录' })).toBeVisible({ timeout: 10000 });
  
  await expect(
    page.getByText('当前账号因超过会话空闲时长被自动退出，请重新登录继续操作。', { exact: true }),
  ).toHaveCount(0);
});

test('user governance smoke: cross-page selection keeps the full selected set', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const userPrefix = `smoke_cross_page_${now}`;
  const password = 'ChangeMe123';
  const roleId = (await getFirstActiveRole(page, accessToken)).id;
  const usernames = Array.from(
    { length: 11 },
    (_, index) => `${userPrefix}_${String(index + 1).padStart(2, '0')}`,
  );

  try {
    for (const username of usernames) {
      await deleteUserByUsername(page, accessToken, username);
      await createUserByApi(page, accessToken, {
        username,
        password,
        nickname: username,
        email: `${username}@example.com`,
        roleIds: [roleId],
      });
    }

    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await expectPageIdentityReady(page, '用户管理');
    await expectNoPageError(page);

    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(userPrefix);
      await Promise.all([
        waitForOkApiResponse(
          page,
          (response) =>
            response.url().includes('/system/user/list') &&
            decodeURIComponent(response.url()).includes(`keyword=${userPrefix}`) &&
            response.request().method() === 'GET',
        ),
        toolbarKeyword.press('Enter'),
      ]);
    }

    const pager = page.locator('.system-user-list__table');
    const selectedText = page.locator('.table-batch-action-bar__meta');
    await expect(
      pager.locator('.arco-pagination-item').filter({ hasText: /^2$/ }).first(),
    ).toBeVisible();

    const firstPageCheckbox = pager.locator('.arco-checkbox').nth(1);
    await firstPageCheckbox.click({ force: true });
    await expect(selectedText).toContainText('已选 1 条');

    const secondPageButton = pager
      .locator('.arco-pagination-item')
      .filter({ hasText: /^2$/ })
      .first();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${userPrefix}`) &&
          decodeURIComponent(response.url()).includes('page=2') &&
          response.request().method() === 'GET',
      ),
      secondPageButton.click(),
    ]);
    await expect
      .poll(async () => {
        return pager.locator('.arco-pagination-item-active').innerText();
      })
      .toBe('2');
    await expect(selectedText).toContainText('已选 1 条');

    const secondPageCheckbox = pager.locator('.arco-checkbox').nth(1);
    await secondPageCheckbox.click({ force: true });
    await expect(selectedText).toContainText('已选 2 条');

    const firstPageButton = pager
      .locator('.arco-pagination-item')
      .filter({ hasText: /^1$/ })
      .first();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${userPrefix}`) &&
          decodeURIComponent(response.url()).includes('page=1') &&
          response.request().method() === 'GET',
      ),
      firstPageButton.click(),
    ]);
    await expect
      .poll(async () => {
        return pager.locator('.arco-pagination-item-active').innerText();
      })
      .toBe('1');
    await expect(selectedText).toContainText('已选 2 条');
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-cross-page-selection', async () => {
      for (const username of usernames) {
        await deleteUserByUsername(page, accessToken, username);
      }
    });
  }
});

test('user smoke: edit and detail work through the UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const deptName = `烟测用户部门-${now}`;
  const username = `smoke_user_${now}`;
  const nickname = `烟测用户${now}`;
  const nextNickname = `${nickname}-已编辑`;
  const email = `smoke-user-${now}@example.com`;
  const nextEmail = `smoke-user-${now}-updated@example.com`;
  const password = 'ChangeMe123';
  const deptTree = await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' });
  const rootDept = flattenDeptTreeNodes(deptTree).find(
    (item) => item.isRoot || item.parentId === 0,
  );
  expect(rootDept).toBeTruthy();

  await deleteUserByUsername(page, accessToken, username);
  await deleteDeptByName(page, accessToken, deptName);

  try {
    const dept = await createDeptByApi(page, accessToken, {
      parentId: rootDept!.id,
      deptName,
      sort: 12,
      email: `user-dept-${now}@example.com`,
      phone: '13800000003',
    });
    await createUserByApi(page, accessToken, {
      username,
      password,
      nickname,
      email,
      deptId: dept.id,
      roleIds: [(await getFirstActiveRole(page, accessToken)).id],
    });

    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '用户管理');
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(username);
      await toolbarKeyword.press('Enter');
    }
    const userRow = page.getByRole('row', { name: new RegExp(username) }).first();
    await expect(userRow).toBeVisible();
    await expect(userRow).toContainText(nickname);

    await userRow.getByRole('button', { name: '编辑' }).click();
    const editDialog = page.getByRole('dialog').filter({ hasText: '编辑用户' });
    await expect(editDialog).toBeVisible();
    await editDialog.getByRole('textbox').nth(1).fill(nextNickname);
    await editDialog.getByRole('textbox').nth(2).fill(nextEmail);
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/') && response.request().method() === 'PUT',
      ),
      editDialog.locator('.submit-bar').getByRole('button', { name: '保存' }).click(),
    ]);
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.nickname)
      .toBe(nextNickname);
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.email)
      .toBe(nextEmail);
    await expect(userRow).toContainText(nextNickname);
    await expect(userRow).toContainText(nextEmail);

    await userRow.getByRole('button', { name: '详情' }).click();
    const detailDialog = page.getByRole('dialog').filter({ hasText: username });
    await expect(detailDialog).toBeVisible();
    await expect(detailDialog.getByText(nextNickname).first()).toBeVisible();
    await expect(detailDialog.getByText(nextEmail).first()).toBeVisible();
    await page.keyboard.press('Escape');
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-smoke-edit-detail', async () => {
      await deleteUserByUsername(page, accessToken, username);
      await deleteDeptByName(page, accessToken, deptName);
    });
  }
});

test('user and role smoke: role binding can be deferred to role management and removed there', async ({
  page,
}) => {
  test.setTimeout(120000);
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  const accessToken = login.accessToken;
  const now = Date.now();
  const username = `smoke_roleless_${now}`;
  const nickname = `待分配角色用户${now}`;
  const password = 'ChangeMe123';
  const roleKey = `role_member_smoke_${now}`;
  const roleName = `角色成员烟测${now}`;
  let createdUserId: number | null = null;
  let createdRoleId: number | null = null;
  const verifiedAuthHeaders = await verifiedApiHeaders(page.request, login);

  await deleteUserByUsername(page, accessToken, username);
  await deleteRoleByKey(page, accessToken, roleKey);

  try {
    const createRoleResponse = await page.request.post(`${apiBaseUrl}/system/role`, {
      headers: verifiedAuthHeaders,
      data: {
        roleName,
        roleKey,
        sort: 10,
        status: 1,
        menuIds: [],
        permissionKeys: [],
      },
    });
    expect(createRoleResponse.ok()).toBeTruthy();
    const role = await getRoleByKey(page, accessToken, roleKey);
    expect(role).toBeTruthy();
    createdRoleId = role?.id ?? null;

    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '用户管理');
    await page.getByRole('button', { name: '新增' }).click();
    const createDialog = page.getByRole('dialog').filter({ hasText: '新增用户' });
    await expect(createDialog).toBeVisible();
    await expect(
      createDialog.locator('.system-user-list__role-field').first().locator('strong'),
    ).toHaveCount(0);
    await createDialog.locator('.submit-bar').getByRole('button', { name: '取消' }).click();
    await expect(createDialog).toHaveCount(0);

    createdUserId = (
      await createUserByApi(page, accessToken, {
        username,
        password,
        nickname,
        roleIds: [],
      })
    ).id;
    expect(createdUserId).toBeTruthy();

    const userDetailBeforeResponse = await page.request.get(
      `${apiBaseUrl}/system/user/${createdUserId}`,
      {
        headers: authHeaders(accessToken),
      },
    );
    expect(userDetailBeforeResponse.ok()).toBeTruthy();
    const userDetailBeforePayload = await userDetailBeforeResponse.json();
    expect(userDetailBeforePayload.data.roleIds).toEqual([]);

    await page.goto('/system/role', { waitUntil: 'domcontentloaded' });
    await expectVisiblePageTitle(page, '角色管理');
    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(roleKey);
      await toolbarKeyword.press('Enter');
    }
    const roleRow = page.getByRole('row', { name: new RegExp(roleName) }).first();
    await expect(roleRow).toBeVisible();
    await roleRow.getByRole('button', { name: '角色成员' }).click();

    const memberDrawer = page.locator('.role-member-drawer');
    await expect(memberDrawer).toBeVisible();
    const candidateResponse = await page.request.get(
      `${apiBaseUrl}/system/role/${role!.id}/user-candidates`,
      {
        headers: authHeaders(accessToken),
        params: { keyword: username, page: '1', pageSize: '20' },
      },
    );
    expect(candidateResponse.ok()).toBeTruthy();
    const candidatePayload = await candidateResponse.json();
    expect(
      Array.isArray(candidatePayload.data?.items) &&
        candidatePayload.data.items.some(
          (item: { username: string }) => item.username === username,
        ),
    ).toBeTruthy();
    await memberDrawer.getByPlaceholder('搜索并选择待加入该角色的用户').fill(username);
    const candidateChip = memberDrawer
      .locator('.role-member-drawer__candidate-pill')
      .filter({ hasText: new RegExp(`${username}|${nickname}`) })
      .first();
    await expect(candidateChip).toBeVisible();
    await candidateChip.click({ force: true });
    await expect(memberDrawer.getByRole('button', { name: '新增' })).toBeEnabled();
    await memberDrawer.getByRole('button', { name: '新增' }).click();
    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/user/${createdUserId}`, {
          headers: authHeaders(accessToken),
        });
        const payload = await response.json();
        return payload.data.roleIds;
      })
      .toEqual([role!.id]);
    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/role/${role!.id}/users`, {
          headers: authHeaders(accessToken),
          params: { keyword: username, page: 1, pageSize: 10 },
        });
        const payload = await response.json();
        const items = Array.isArray(payload.data?.items) ? payload.data.items : [];
        return items.some((item: { username: string }) => item.username === username);
      })
      .toBeTruthy();
    await memberDrawer.getByPlaceholder('按用户名或昵称搜索当前成员').fill(username);
    await memberDrawer.getByRole('button', { name: '搜索' }).click();
    const memberRow = await waitForVisibleRowByText(memberDrawer, username, '删除');
    await clickVisibleRowAction(page, memberRow, '删除', '确认移除该成员的当前角色绑定？');
    await clickVisibleConfirmButton(page, '确认移除该成员的当前角色绑定？');
    await expect
      .poll(async () => {
        const response = await page.request.get(`${apiBaseUrl}/system/user/${createdUserId}`, {
          headers: authHeaders(accessToken),
        });
        const payload = await response.json();
        return payload.data.roleIds;
      })
      .toEqual([]);
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-role-membership-governance', async () => {
      await deleteUserByIdWithHeaders(page, verifiedAuthHeaders, createdUserId);
      await deleteRoleByIdWithHeaders(page, verifiedAuthHeaders, createdRoleId);
    });
  }
});

test('user smoke: batch disable enable and delete stay stable through the UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const username = `smoke_user_batch_${now}`;
  const password = 'ChangeMe123';

  await deleteUserByUsername(page, accessToken, username);

  try {
    await createUserByApi(page, accessToken, {
      username,
      password,
      nickname: `批量烟测用户${now}`,
      email: `smoke-user-batch-${now}@example.com`,
      roleIds: [(await getFirstActiveRole(page, accessToken)).id],
    });

    await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '用户管理');
    const tableCard = page.locator('.system-list__table-card');
    const userTable = page.locator('.system-user-list__table');
    const selectedText = page.locator('.table-batch-action-bar__meta');
    await expect(tableCard).toBeVisible({ timeout: 30000 });

    const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
    await toolbarKeyword.fill(username);
    const waitForFilteredUserList = () =>
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/list') &&
          decodeURIComponent(response.url()).includes(`keyword=${username}`) &&
          response.request().method() === 'GET',
      );
    const selectBatchUser = async () => {
      const rowCheckbox = userTable.locator('.arco-checkbox').nth(1);
      await expect(rowCheckbox).toBeVisible();
      await rowCheckbox.click({ force: true });
      await expect(selectedText).toContainText('已选 1 条');
    };

    await Promise.all([waitForFilteredUserList(), toolbarKeyword.press('Enter')]);
    await selectBatchUser();

    await page.getByRole('button', { name: '批量禁用' }).click();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/batch-status') &&
          response.request().method() === 'POST',
      ),
      clickVisibleConfirmButton(page),
    ]);
    await waitForFilteredUserList();
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.status)
      .toBe(2);
    await expect(selectedText).toContainText('已选 0 条');

    await selectBatchUser();
    await page.getByRole('button', { name: '批量启用' }).click();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/batch-status') &&
          response.request().method() === 'POST',
      ),
      clickVisibleConfirmButton(page),
    ]);
    await waitForFilteredUserList();
    await expect
      .poll(async () => (await findUserByUsername(page, accessToken, username))?.status)
      .toBe(1);
    await expect(selectedText).toContainText('已选 0 条');

    await selectBatchUser();
    await page.getByRole('button', { name: '删除所选' }).click();
    await Promise.all([
      waitForOkApiResponse(
        page,
        (response) =>
          response.url().includes('/system/user/batch-delete') &&
          response.request().method() === 'POST',
      ),
      clickVisibleConfirmButton(page),
    ]);
    await waitForFilteredUserList();
    await expect
      .poll(async () => await findUserByUsername(page, accessToken, username))
      .toBeUndefined();
  } finally {
    await runOptionalSmokeCleanup('system-pages:user-smoke-batch', async () => {
      await deleteUserByUsername(page, accessToken, username);
    });
  }
});
