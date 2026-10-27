import { test } from '../../fixtures/coverage';
import { expect } from '@playwright/test';
import {
  apiBaseUrl,
  createDeptByApi,
  createPostByApi,
  createUserByApi,
  deleteDeptByName,
  deletePostByCode,
  deleteUserByUsername,
  expectVisiblePageTitle,
  findPostByCode,
  flattenDeptTreeNodes,
  getDeptTree,
  getFirstActiveRole,
  installOperationToken,
  runOptionalSmokeCleanup,
  signInAsAdmin,
  verifiedHeaders,
} from './system-pages-helpers';

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

test('dept smoke: create dialog and root edit are reachable through the UI', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  await page.goto('/system/dept', { waitUntil: 'domcontentloaded' });
  await installOperationToken(page, accessToken);
  await expectVisiblePageTitle(page, '部门管理');
  await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

  await page.getByRole('button', { name: '新增' }).click();
  const createDialog = page.getByRole('dialog').filter({ hasText: '新增部门' });
  await expect(createDialog).toBeVisible();
  await expect(createDialog.getByText('Pantheon Base').first()).toBeVisible();
  await expect(createDialog.getByRole('textbox').first()).toBeVisible();
  await createDialog.locator('.submit-bar').getByRole('button', { name: '取消' }).click();
  await expect(createDialog).not.toBeVisible();

  const rootRow = page.getByRole('row', { name: /Pantheon Base/ }).first();
  await rootRow.getByRole('button', { name: '编辑' }).click();
  const editDialog = page.getByRole('dialog').filter({ hasText: '编辑部门' });
  await expect(editDialog).toBeVisible();
  await expect(editDialog.getByRole('textbox').first()).toHaveValue('Pantheon Base');
  await expect(editDialog.getByText('启用').first()).toBeVisible();
  await page.keyboard.press('Escape');
});

test('dept smoke: create dialog defaults the parent even when the dept tree loads late', async ({
  page,
}) => {
  const accessToken = await signInAsAdmin(page);
  await page.route('**/api/v1/system/dept/tree**', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 3000));
    await route.continue();
  });
  await page.goto('/system/dept', { waitUntil: 'domcontentloaded' });
  await installOperationToken(page, accessToken);
  await page.getByRole('button', { name: '新增' }).click();
  const createDialog = page.getByRole('dialog').filter({ hasText: '新增部门' });
  await expect(createDialog).toBeVisible();
  await expect(createDialog.getByText('Pantheon Base').first()).toBeVisible({ timeout: 15000 });
  await createDialog.locator('.submit-bar').getByRole('button', { name: '取消' }).click();
});

test('dept smoke: blocked delete through API is covered', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const deptName = `烟测部门-${now}`;
  const postCode = `SMOKE_DEPT_POST_${now}`;
  const postName = `烟测部门岗位-${now}`;
  const deptTree = await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' });
  const rootDept = flattenDeptTreeNodes(deptTree).find(
    (item) => item.isRoot || item.parentId === 0,
  );
  expect(rootDept).toBeTruthy();

  await deletePostByCode(page, accessToken, postCode);
  await deleteDeptByName(page, accessToken, deptName);

  try {
    const createdDept = await createDeptByApi(page, accessToken, {
      parentId: rootDept!.id,
      deptName,
      sort: 11,
      email: `dept-${now}@example.com`,
      phone: '021-12345678',
    });

    await createPostByApi(page, accessToken, {
      deptId: createdDept.id,
      postCode,
      postName,
      sort: 10,
      remark: 'dept smoke',
    });

    const deleteResponse = await page.request.delete(
      `${apiBaseUrl}/system/dept/${createdDept.id}`,
      {
        headers: await verifiedHeaders(page, accessToken),
      },
    );
    expect(deleteResponse.ok()).toBeTruthy();
    const deletePayload = await deleteResponse.json();
    expect(deletePayload.code).not.toBe(200);
    expect(deletePayload.message).toBe('dept.delete.error.has_posts');
  } finally {
    await runOptionalSmokeCleanup('system-pages:dept-smoke-blocked-delete', async () => {
      await deletePostByCode(page, accessToken, postCode);
      await deleteDeptByName(page, accessToken, deptName);
    });
  }
});

test('post smoke: edit through UI and blocked delete through API are covered', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const now = Date.now();
  const deptName = `烟测岗位部门-${now}`;
  const postCode = `SMOKE_POST_${now}`;
  const postName = `烟测岗位-${now}`;
  const username = `smoke_post_user_${now}`;
  const password = 'ChangeMe123';
  const role = await getFirstActiveRole(page, accessToken);
  const deptTree = await getDeptTree(page, accessToken, { sortField: 'sort', sortOrder: 'asc' });
  const rootDept = flattenDeptTreeNodes(deptTree).find(
    (item) => item.isRoot || item.parentId === 0,
  );
  expect(rootDept).toBeTruthy();

  await deleteUserByUsername(page, accessToken, username);
  await deletePostByCode(page, accessToken, postCode);
  await deleteDeptByName(page, accessToken, deptName);

  try {
    const dept = await createDeptByApi(page, accessToken, {
      parentId: rootDept!.id,
      deptName,
      sort: 10,
      email: `post-dept-${now}@example.com`,
      phone: '13800000002',
    });
    const createdPost = await createPostByApi(page, accessToken, {
      deptId: dept.id,
      postCode,
      postName,
      sort: 21,
      remark: 'post smoke',
    });

    await page.goto('/system/post', { waitUntil: 'domcontentloaded' });
    await installOperationToken(page, accessToken);
    await expectVisiblePageTitle(page, '岗位管理');
    await expect(page.locator('.system-list__table-card')).toBeVisible({ timeout: 30000 });

    {
      const toolbarKeyword = page.locator('.search-toolbar').getByPlaceholder(/搜索/);
      await toolbarKeyword.fill(postCode);
      await toolbarKeyword.press('Enter');
    }
    const postRow = page.getByRole('row', { name: new RegExp(postCode) }).first();
    await expect(postRow).toBeVisible();
    await expect(postRow).toContainText(postName);

    await postRow.getByRole('button', { name: '编辑' }).click();
    const editDialog = page.getByRole('dialog').filter({ hasText: '编辑岗位' });
    await expect(editDialog).toBeVisible();
    await expect(editDialog.getByRole('textbox').first()).toHaveValue(postCode);
    await expect(editDialog.getByRole('textbox').nth(1)).toHaveValue(postName);
    await page.keyboard.press('Escape');

    const post = await findPostByCode(page, accessToken, postCode);
    expect(post).toBeTruthy();
    await createUserByApi(page, accessToken, {
      username,
      password,
      nickname: `岗位占用用户${now}`,
      roleIds: [role.id],
      deptId: dept.id,
      postId: post!.id,
      email: `post-user-${now}@example.com`,
    });

    const deleteResponse = await page.request.delete(
      `${apiBaseUrl}/system/post/${createdPost.id}`,
      {
        headers: await verifiedHeaders(page, accessToken),
      },
    );
    expect(deleteResponse.ok()).toBeTruthy();
    const deletePayload = await deleteResponse.json();
    expect(deletePayload.code).not.toBe(200);
    expect(deletePayload.message).toBe('post.delete.error.has_users');
  } finally {
    await runOptionalSmokeCleanup('system-pages:post-smoke-edit-blocked-delete', async () => {
      await deleteUserByUsername(page, accessToken, username);
      await deletePostByCode(page, accessToken, postCode);
      await deleteDeptByName(page, accessToken, deptName);
    });
  }
});
