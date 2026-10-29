import { randomBytes } from 'node:crypto';

import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import { clickVisibleConfirmButton } from '../smoke/system/system-pages-helpers';
import {
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  installClientSession,
  loginByApi,
} from '../smoke/helpers/auth';

/**
 * Tenant console end-to-end (W6 `2026-10-27-tenant-console-ui-automation`).
 *
 * Covers the surfaces the isolation/viewpoint specs deliberately do not touch:
 * tenant CRUD with lifecycle gating (W1), the member dialog with the bounded
 * user selector (W2) and the last-owner refusal, the ops overview numbers and
 * the plan editor lifecycle (create custom code, assign it, refuse disabling
 * an in-use plan, disable after unassigning).
 *
 * ## Prerequisites and run command
 *
 * - a live backend with multi-tenant mode on (defaults: admin/123456, override
 *   with PANTHEON_SMOKE_ADMIN_USERNAME / PANTHEON_SMOKE_ADMIN_PASSWORD)
 * - the full suite: `npm run test:smoke:tenant` (provisions the matrix pair
 *   and runs every tenant spec)
 * - this file alone: `npx playwright test tests/smoke-core/tenant-console.spec.ts`
 *
 * The specs provision their **own** disposable tenants through the console and
 * archive-then-delete them at the end (contract §2.1: delete refuses anything
 * not archived). They never mutate the shared matrix pair — archiving a
 * manifest tenant would break the isolation specs that run after this file.
 * The member dialog additionally pins the tenant viewpoint through the real
 * switcher, because the W2 picker reads the request's tenant scope (W4 rule).
 *
 * Destructive rows (tenant delete, member removal) sit behind the backend's
 * secure-action guard: the first destructive request of a session is refused
 * with `auth.operation.verification_required` and the shell pops the secondary
 * password dialog before retrying. `confirmDestructiveAction` fills that
 * dialog when it appears; once the operation token is cached the dialog is
 * skipped, so the helper tolerates both shapes.
 */

type ApiEnvelope<T> = { code: number; message?: string; data: T };

const run = `${Date.now()}`.slice(-8);
const tenantCode = `console-${run}`;
const tenantName = `控制台冒烟 ${run}`;
const adminPassword = `Cnsl${randomBytes(6).toString('hex')}#Aa1`;
const planCode = `plan_console_${run}`;
const planName = `控制台套餐 ${run}`;
const platformAdminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456';

let platformLogin: Awaited<ReturnType<typeof loginByApi>>;

async function platformHeaders(request: APIRequestContext) {
  const operationToken = await getApiOperationToken(request, platformLogin);
  return { ...apiRequestHeaders(platformLogin), 'X-Operation-Token': operationToken };
}

/** Opens the tenant list as a signed-in platform administrator. */
async function openTenantList(page: Page) {
  await installClientSession(page, platformLogin);
  await page.goto('/system/tenant', { waitUntil: 'domcontentloaded' });
  await expect(page.getByRole('button', { name: '新增' })).toBeVisible();
}

function tenantRow(page: Page, code: string) {
  return page.locator('.system-list__table tbody tr', { hasText: code });
}

async function openRowAction(page: Page, code: string, action: '成员' | '编辑' | '删除') {
  const row = tenantRow(page, code);
  await row.getByRole('button', { name: action }).click();
}

/** Fills and submits the tenant form; `status` only applies when editing. */
async function submitTenantForm(
  page: Page,
  values: {
    code?: string;
    name?: string;
    adminUsername?: string;
    adminPassword?: string;
    plan?: string;
    status?: string;
  },
  mode: 'create' | 'edit',
) {
  const modal = page.locator('.arco-modal', { hasText: mode === 'create' ? '新增租户' : '编辑租户' });
  if (values.code) await modal.getByPlaceholder('小写字母、数字或连字符，例如 acme').fill(values.code);
  if (values.name) {
    await modal.getByPlaceholder('租户展示名称').fill(values.name);
  }
  if (values.adminUsername) {
    await modal.getByPlaceholder('例如 tenant-admin').fill(values.adminUsername);
  }
  if (values.adminPassword) {
    await modal.locator('input[type="password"]').fill(values.adminPassword);
  }
  for (const [label, option] of [
    ['套餐', values.plan],
    ['状态', values.status],
  ] as const) {
    if (!option) continue;
    const formItem = modal.locator('.arco-form-item', { hasText: label });
    await formItem.locator('.arco-select').click();
    await page.locator('.arco-select-option', { hasText: option }).first().click();
  }
  await modal.getByRole('button', { name: mode === 'create' ? '新增' : '保存' }).click();
}

/**
 * Confirms a row Popconfirm and then the secondary password dialog that the
 * secure-action guard raises for the first destructive request of a session
 * (tenant delete, member removal). The first DELETE returning HTTP 403 means
 * `auth.operation.verification_required`: fill the dialog so the request
 * retries. Outcome assertions stay with the caller — the retried request may
 * succeed or hit a business refusal.
 */
async function confirmDestructiveAction(page: Page, password: string) {
  const firstAttempt = page.waitForResponse(
    (response) => response.request().method() === 'DELETE' && response.url().includes('/system/tenant/'),
    { timeout: 10000 },
  );
  await clickVisibleConfirmButton(page);
  const response = await firstAttempt.catch(() => null);
  if (!response || response.status() !== 403) return;
  const verifyModal = page.locator('.arco-modal', { hasText: '敏感操作验证' });
  const passwordInput = verifyModal.locator('input[type="password"]');
  await passwordInput.waitFor({ state: 'visible' });
  await passwordInput.fill(password);
  await verifyModal.getByRole('button', { name: '确定' }).click();
}

test.beforeAll(async ({ request }) => {
  platformLogin = await loginByApi(request, {
    username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
    password: platformAdminPassword,
  });
});

test.afterAll(async ({ request }) => {
  // Best-effort cleanup of anything a failed run left behind: archive then
  // delete (the W1 guard refuses direct deletes of non-archived tenants), and
  // retire leftover console plans — plans have no delete, retirement is the
  // terminal state a failed run can leave an active row in.
  if (!platformLogin) return;
  const headers = await platformHeaders(request);
  const list = await request.get(`${apiBaseUrl}/system/tenant/list?keyword=console-`, { headers });
  const payload = (await list.json()) as ApiEnvelope<{ items?: Array<{ id: number; status: string }> }>;
  for (const row of payload.data?.items ?? []) {
    if (row.status !== 'deleted') {
      await request.put(`${apiBaseUrl}/system/tenant/${row.id}`, {
        headers,
        data: { name: `${tenantName} ${row.id}`, status: 'archived' },
      });
      await request.delete(`${apiBaseUrl}/system/tenant/${row.id}`, { headers });
    }
  }
  const planList = await request.get(`${apiBaseUrl}/system/tenant-ops/plan/list`, { headers });
  const planPayload = (await planList.json()) as ApiEnvelope<{
    items?: Array<{ code: string; name: string; status: string }>;
  }>;
  for (const plan of planPayload.data?.items ?? []) {
    if (plan.code.startsWith('plan_console_') && plan.status === 'active') {
      await request.put(`${apiBaseUrl}/system/tenant-ops/plan/${plan.code}`, {
        headers,
        data: { name: plan.name, status: 'disabled' },
      });
    }
  }
});

test.describe.serial('Tenant console @priority:critical @smoke:tenant', () => {
  test('list renders summary cards and the create entry', async ({ page }) => {
    await openTenantList(page);
    await expect(page.getByText('租户总数')).toBeVisible();
    await expect(page.getByText('启用中').first()).toBeVisible();
    await expect(page.locator('.system-list__table')).toBeVisible();
  });

  let adminUsername = '';

  test('create a tenant with its initial administrator', async ({ page }) => {
    await openTenantList(page);
    await page.getByRole('button', { name: '新增' }).click();
    adminUsername = `console_admin_${run}`;
    await submitTenantForm(
      page,
      {
        code: tenantCode,
        name: tenantName,
        adminUsername,
        adminPassword,
      },
      'create',
    );
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
    const row = tenantRow(page, tenantCode);
    await expect(row).toContainText(tenantName);
    await expect(row).toContainText('启用中');
  });

  test('edit renames the tenant and switches its plan', async ({ page }) => {
    await openTenantList(page);
    await openRowAction(page, tenantCode, '编辑');
    const renamed = `${tenantName} 改`;
    await submitTenantForm(page, { name: renamed, plan: '基础版' }, 'edit');
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
    const row = tenantRow(page, tenantCode);
    await expect(row).toContainText(renamed);
    await expect(row).toContainText('基础版');
  });

  test('delete stays disabled until the tenant is archived, then succeeds (W1)', async ({
    page,
  }) => {
    await openTenantList(page);
    // Suspended: still not archive-terminal, delete must stay disabled.
    await openRowAction(page, tenantCode, '编辑');
    await submitTenantForm(page, { status: '已停用' }, 'edit');
    await expect(tenantRow(page, tenantCode)).toContainText('已停用');
    await expect(
      tenantRow(page, tenantCode).getByRole('button', { name: '删除' }),
    ).toBeDisabled();

    await openRowAction(page, tenantCode, '编辑');
    await submitTenantForm(page, { status: '已归档' }, 'edit');
    await expect(tenantRow(page, tenantCode)).toContainText('已归档');
    await page.screenshot({ path: 'test-results/tenant-console/w1-before-delete.png', fullPage: true });

    // Archived: delete is available and succeeds; the row leaves the default list.
    await openRowAction(page, tenantCode, '删除');
    await confirmDestructiveAction(page, platformAdminPassword);
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
    await expect(tenantRow(page, tenantCode)).toHaveCount(0);
    await page.screenshot({ path: 'test-results/tenant-console/w1-after-delete.png', fullPage: true });
  });

  test('member dialog: last-owner removal refused, re-add through the selector (W2)', async ({
    page,
    request,
  }) => {
    // A fresh tenant whose only member is its provisioning owner. W4 decision
    // A: AddMember only accepts users owned by the target tenant, and the
    // picker reads whichever tenant scope the request carries — so the shell
    // is pinned to this tenant. A tenant-scoped user is seeded via the API as
    // fixture: user creation auto-provisions an active membership
    // (ensureTenantMembership), so the candidate list only ever offers a user
    // whose membership was removed — the add scenario is remove-then-re-add.
    const memberTenantCode = `console-m-${run}`;
    const memberTenantName = `成员冒烟 ${run}`;
    const extraUsername = `console_extra_${run}`;
    const headers = await platformHeaders(request);
    const created = await request.post(`${apiBaseUrl}/system/tenant`, {
      headers,
      data: {
        code: memberTenantCode,
        name: memberTenantName,
        admin_username: `console_member_${run}`,
        admin_password: adminPassword,
      },
    });
    const createdPayload = (await created.json()) as ApiEnvelope<{ id?: number }>;
    expect(createdPayload.code, JSON.stringify(createdPayload)).toBe(200);
    const memberTenantId = createdPayload.data?.id;
    expect(memberTenantId, JSON.stringify(createdPayload)).toBeTruthy();

    const extraUser = await request.post(`${apiBaseUrl}/system/user`, {
      headers: { ...headers, 'X-Tenant-Id': String(memberTenantId) },
      data: {
        username: extraUsername,
        nickname: `成员冒烟额外 ${run}`,
        password: adminPassword,
        status: 1,
      },
    });
    const extraPayload = (await extraUser.json()) as ApiEnvelope<{ id?: number }>;
    expect(extraPayload.code, JSON.stringify(extraPayload)).toBe(200);
    const extraUserId = extraPayload.data?.id;
    expect(extraUserId, JSON.stringify(extraPayload)).toBeTruthy();

    await openTenantList(page);
    await page.getByRole('button', { name: '租户视角' }).click();
    await Promise.all([
      page.waitForLoadState('load'),
      page.locator('.arco-dropdown-menu-item', { hasText: memberTenantName }).first().click(),
    ]);
    await expect(page.locator('.tenant-viewpoint-banner')).toBeVisible();
    // Pinning lands on the tenant workbench; return to the console list.
    await openTenantList(page);

    await openRowAction(page, memberTenantCode, '成员');
    const dialog = page.locator('.arco-modal', { hasText: '租户成员' });
    await expect(dialog.locator('tbody tr')).toHaveCount(2);

    // Last-owner refusal: removing the owner row is refused with the
    // contract §2.2 message and the row survives.
    await dialog
      .locator('tbody tr', { hasText: '所有者' })
      .getByRole('button', { name: '移除' })
      .click();
    await confirmDestructiveAction(page, platformAdminPassword);
    await expect(page.locator('.arco-message-error').first()).toContainText('最后一名');
    await expect(dialog.locator('tbody tr')).toHaveCount(2);

    // Removing the seeded non-owner member succeeds and frees the user up as
    // a picker candidate again.
    const memberRow = dialog.locator('tbody tr', { hasText: '普通成员' });
    await memberRow.getByRole('button', { name: '移除' }).click();
    await confirmDestructiveAction(page, platformAdminPassword);
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
    await expect(dialog.locator('tbody tr')).toHaveCount(1);

    // The W2 selector: candidates come from the pinned tenant's user pool —
    // not a raw id field and not the platform-wide user list.
    await dialog.getByPlaceholder('搜索用户名或昵称').click();
    await dialog.getByPlaceholder('搜索用户名或昵称').fill(extraUsername);
    await page.locator('.arco-select-option', { hasText: extraUsername }).first().click();
    await dialog.getByRole('button', { name: '新增' }).click();
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
    await expect(dialog.locator('tbody tr')).toHaveCount(2);

    // Leave the tenant archived+deleted so afterAll or a re-run starts clean.
    // The viewpoint is unpinned first: the cleanup requests must run at
    // platform scope, and archiving the pinned tenant from inside its own
    // viewpoint is exactly the confusion this dialog warns about.
    await dialog.locator('.arco-modal-close-btn, .arco-modal-close-icon').first().click();
    await Promise.all([
      page.waitForLoadState('load'),
      page.getByRole('button', { name: '退出租户视角' }).click(),
    ]);
    await expect(page.locator('.tenant-viewpoint-banner')).toHaveCount(0);
    await openTenantList(page);
    await openRowAction(page, memberTenantCode, '编辑');
    await submitTenantForm(page, { status: '已归档' }, 'edit');
    await openRowAction(page, memberTenantCode, '删除');
    await confirmDestructiveAction(page, platformAdminPassword);
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
  });

  test('ops overview numbers and the plan editor lifecycle', async ({ page }) => {
    await installClientSession(page, platformLogin);
    await page.goto('/system/tenant-ops/overview', { waitUntil: 'domcontentloaded' });
    await expect(page.getByText('租户总数').first()).toBeVisible();
    await expect(page.getByText('成员总数').first()).toBeVisible();
    await expect(page.getByText('角色总数').first()).toBeVisible();

    // Plan editor: the seed rows are the four built-in ladder plans. Plans
    // cannot be deleted (retire is the terminal state), so earlier runs leave
    // their console plans behind — assert on the builtins, not a total count.
    await page.getByRole('button', { name: '套餐定义' }).click();
    const drawer = page.locator('.arco-drawer', { hasText: '套餐定义' });
    for (const code of ['free', 'basic', 'professional', 'enterprise']) {
      await expect(drawer.locator('tbody tr', { hasText: code })).toHaveCount(1);
    }
    for (const code of ['free', 'basic', 'professional', 'enterprise']) {
      await expect(drawer.locator('tbody tr', { hasText: code })).toHaveCount(1);
    }

    // Create an operator-defined code.
    await drawer.getByRole('button', { name: '新增套餐' }).click();
    const createModal = page.locator('.arco-modal', { hasText: '新增套餐' });
    await createModal.getByPlaceholder('例如 starter').fill(planCode);
    await createModal.getByPlaceholder('套餐展示名称').fill(planName);
    for (const label of ['成员上限', '角色上限', '部门上限', 'API 密钥上限', '存储上限']) {
      await createModal
        .locator('.arco-form-item', { hasText: label })
        .locator('input:not([type="checkbox"])')
        .fill('10');
    }
    await createModal.getByRole('button', { name: '新增' }).click();
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
    await expect(drawer.locator('tbody tr', { hasText: planCode })).toHaveCount(1);

    // Disabling while a tenant still sits on it is refused… but first the plan
    // must actually be assigned, so assign it through the tenant edit form.
    // The shell tenant is provisioned through the API and the list is
    // re-opened afterwards: the table was loaded before the row existed.
    const shellCode = `console-p-${run}`;
    const headers = await platformHeaders(page.request);
    const created = await page.request.post(`${apiBaseUrl}/system/tenant`, {
      headers,
      data: {
        code: shellCode,
        name: `套餐冒烟 ${run}`,
        admin_username: `console_plan_${run}`,
        admin_password: adminPassword,
      },
    });
    expect(((await created.json()) as ApiEnvelope<unknown>).code).toBe(200);
    await openTenantList(page);
    await openRowAction(page, shellCode, '编辑');
    await submitTenantForm(page, { plan: planName }, 'edit');
    // A custom plan has no i18n label, so the cell falls back to the code.
    await expect(tenantRow(page, shellCode)).toContainText(planCode);

    // Refusal: retire the in-use plan.
    await page.goto('/system/tenant-ops/overview', { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '套餐定义' }).click();
    await drawer.locator('tbody tr', { hasText: planCode }).getByRole('button', { name: '编辑' }).click();
    const editModal = page.locator('.arco-modal', { hasText: '编辑套餐' });
    await editModal.locator('.arco-form-item', { hasText: '状态' }).locator('.arco-select').click();
    await page.locator('.arco-select-option', { hasText: '停用' }).first().click();
    await editModal.getByRole('button', { name: '保存' }).click();
    await expect(page.locator('.arco-message-error').first()).toContainText('仍有租户使用该套餐');

    // Remedy: unassign, then the retirement succeeds.
    await openTenantList(page);
    await openRowAction(page, shellCode, '编辑');
    await submitTenantForm(page, { plan: '免费版' }, 'edit');
    await page.goto('/system/tenant-ops/overview', { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '套餐定义' }).click();
    await drawer.locator('tbody tr', { hasText: planCode }).getByRole('button', { name: '编辑' }).click();
    await editModal.locator('.arco-form-item', { hasText: '状态' }).locator('.arco-select').click();
    await page.locator('.arco-select-option', { hasText: '停用' }).first().click();
    await editModal.getByRole('button', { name: '保存' }).click();
    await expect(page.locator('.arco-message-success').first()).toBeVisible();

    // Clean the shell tenant the same way the lifecycle test does.
    await openTenantList(page);
    await openRowAction(page, shellCode, '编辑');
    await submitTenantForm(page, { status: '已归档' }, 'edit');
    await openRowAction(page, shellCode, '删除');
    await confirmDestructiveAction(page, platformAdminPassword);
    await expect(page.locator('.arco-message-success').first()).toBeVisible();
  });
});
