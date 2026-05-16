import { expect, test, type Page } from '@playwright/test';
import { apiBaseUrl, authHeaders, signInAsAdmin } from '../helpers/auth';

async function deleteRoleByKey(page: Page, accessToken: string, roleKey: string) {
  const listResponse = await page.request.get(`${apiBaseUrl}/system/role/list`, {
    headers: authHeaders(accessToken),
    params: {
      roleKey,
      page: 1,
      pageSize: 10,
    },
  });
  if (!listResponse.ok()) {
    return;
  }
  const listPayload = await listResponse.json();
  const roles = Array.isArray(listPayload.data?.items) ? listPayload.data.items : [];
  for (const role of roles) {
    if (role.roleKey === roleKey && role.roleKey !== 'admin') {
      await page.request.delete(`${apiBaseUrl}/system/role/${role.id}`, {
        headers: authHeaders(accessToken),
      });
    }
  }
}

function modal(page: Page) {
  return page.locator('.arco-modal').filter({ hasText: '新增角色' }).first();
}

function authorizationCard(page: Page, title: string) {
  return modal(page).locator('.dialog-grid-card').filter({ has: page.getByText(title, { exact: true }) }).first();
}

async function openCreateRoleModal(page: Page) {
  await page.goto('/system/role', { waitUntil: 'networkidle' });
  await expect(page.getByText('角色管理', { exact: false }).filter({ visible: true }).first()).toBeVisible();
  await expect(page.locator('.governance-summary-bar, .system-list__table-card').first()).toBeVisible();
  await page
    .locator('.table-batch-action-bar__prefix-actions')
    .getByRole('button', { name: '新增', exact: true })
    .click();
  await expect(modal(page)).toBeVisible();
}

async function fillModalInput(page: Page, label: string, value: string) {
  const item = modal(page).locator('.arco-form-item').filter({ has: page.getByText(label, { exact: true }) }).first();
  await item.locator('input').first().fill(value);
}

async function selectedCount(card: ReturnType<typeof authorizationCard>) {
  const text = await card.locator('.arco-tag').filter({ hasText: /已选/ }).first().innerText();
  const match = text.match(/已选\s*(\d+)/);
  return match ? Number(match[1]) : 0;
}

async function clickTreeCheckbox(card: ReturnType<typeof authorizationCard>, nodeText: string) {
  const node = card.locator('.arco-tree-node').filter({ hasText: nodeText }).first();
  await expect(node).toBeVisible();
  await node.locator('.arco-checkbox-mask').first().click();
}

test.beforeEach(async ({ page }) => {
  await signInAsAdmin(page);
});

test('role authorization trees support search and top-level batch selection', async ({ page }) => {
  await openCreateRoleModal(page);

  const navigationCard = authorizationCard(page, '导航授权');
  const pageCard = authorizationCard(page, '页面授权');
  const actionCard = authorizationCard(page, '操作授权');

  for (const card of [navigationCard, pageCard, actionCard]) {
    await expect(card.getByPlaceholder('搜索名称或权限标识')).toBeVisible();
    await expect(card.getByRole('button', { name: '全展开' })).toBeVisible();
    await expect(card.getByRole('button', { name: '全收起' })).toBeVisible();
  }

  await pageCard.getByPlaceholder('搜索名称或权限标识').fill('system:role:list');
  await expect(pageCard.getByText('角色管理', { exact: true })).toBeVisible();
  await expect(pageCard.getByText('用户管理', { exact: true })).toHaveCount(0);
  await pageCard.getByPlaceholder('搜索名称或权限标识').clear();

  await clickTreeCheckbox(navigationCard, '访问控制');
  await clickTreeCheckbox(pageCard, '访问控制');
  await clickTreeCheckbox(actionCard, '访问控制');

  expect(await selectedCount(navigationCard)).toBeGreaterThan(1);
  expect(await selectedCount(pageCard)).toBeGreaterThan(1);
  expect(await selectedCount(actionCard)).toBeGreaterThan(1);
});

test('role authorization trees can save parent-driven grants without synthetic keys', async ({ page }) => {
  const accessToken = await signInAsAdmin(page);
  const roleKey = `qa_role_auth_${Date.now()}`;

  await deleteRoleByKey(page, accessToken, roleKey);

  try {
    await openCreateRoleModal(page);
    await fillModalInput(page, '角色名称', 'QA 角色授权回归');
    await fillModalInput(page, '角色标识', roleKey);

    await clickTreeCheckbox(authorizationCard(page, '导航授权'), '访问控制');
    await clickTreeCheckbox(authorizationCard(page, '页面授权'), '访问控制');
    await clickTreeCheckbox(authorizationCard(page, '操作授权'), '访问控制');

    await modal(page).getByRole('button', { name: '新增' }).click();
    await expect(page.getByText('创建成功')).toBeVisible();
    await expect(modal(page)).toHaveCount(0);

    const listResponse = await page.request.get(`${apiBaseUrl}/system/role/list`, {
      headers: authHeaders(accessToken),
      params: {
        roleKey,
        page: 1,
        pageSize: 10,
      },
    });
    expect(listResponse.ok()).toBeTruthy();
    const listPayload = await listResponse.json();
    const createdRole = listPayload.data.items.find((role: { roleKey: string }) => role.roleKey === roleKey);
    expect(createdRole).toBeTruthy();
    expect(createdRole.menuIds.length).toBeGreaterThan(1);
    expect(createdRole.permissionKeys).toContain('system:role:list');
    expect(createdRole.permissionKeys).toContain('system:role:create');
  } finally {
    await deleteRoleByKey(page, accessToken, roleKey);
  }
});
