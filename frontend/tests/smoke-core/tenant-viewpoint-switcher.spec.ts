/**
 * Tenant Viewpoint - 平台主体租户视角（真实浏览器验收）
 *
 * 契约 §3.1 source 1：平台主体（tenant 0）通过 `X-Tenant-Id` 把工作域钉到
 * 某个真实租户；前端由顶栏「租户视角」控件写入 sessionStorage，axios 请求
 * 拦截器负责加头。本 spec 用真实浏览器 + 真实后端走完整链路：
 *
 *   1. 默认平台全局视角：不加头、无横幅、切换器回显「平台全局」
 *   2. 钉住租户：全量重载、出现常驻横幅、业务请求带上 X-Tenant-Id、列表被限定
 *   3. 钉住视角内新建用户并「加入本租户」：membership 落库
 *   4. 该用户能真正登录（多租户模式下 membership 是登录闸门），且新会话不带头
 *   5. 退出视角回到平台全局
 *
 * fixture 自建自清：beforeAll 用 admin API 开一个租户，afterAll 删用户与租户。
 * （DELETE /system/tenant 是软删除，租户行的物理清理见 evidence 里的 SQL。）
 */

import { mkdirSync, existsSync } from 'node:fs';
import { join } from 'node:path';

import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  loginByApi,
  signInWithUi,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';

const run = `${Date.now()}`;
const tenantCode = `smoke-vp-${run}`;
const tenantName = `Smoke Viewpoint ${run}`;
const tenantAdminUsername = `smoke_tadmin_${run}`;
const tenantAdminPassword = 'SmokeVpAdmin#2026';
const viewpointUsername = `smoke_user_viewpoint_${run}`;
const viewpointPassword = 'SmokeVpUser#2026';

const VIEWPOINT_TITLE = '租户视角';
const PLATFORM_SCOPE_LABEL = '平台全局';
const EXIT_VIEWPOINT = '退出租户视角';
const BANNER_TEXT = new RegExp(`当前正以租户「${tenantName}」视角操作`);

let adminLogin: BrowserLoginResult;
let tenantId = 0;
let createdUserId = 0;

/**
 * Acceptance evidence is opt-in: set PANTHEON_VIEWPOINT_EVIDENCE_DIR to keep
 * screenshots of the states the human acceptance pass would look at.
 */
const evidenceDir = process.env.PANTHEON_VIEWPOINT_EVIDENCE_DIR;

async function capture(page: Page, name: string) {
  if (!evidenceDir) {
    return;
  }
  if (!existsSync(evidenceDir)) {
    mkdirSync(evidenceDir, { recursive: true });
  }
  await page.screenshot({ path: join(evidenceDir, `${name}.png`) });
}

/** Everything the shell sent on business calls, keyed by the header we care about. */
type HeaderSample = { url: string; tenantHeader: string | undefined };

function recordTenantHeader(page: Page): HeaderSample[] {
  const samples: HeaderSample[] = [];
  page.on('request', (request) => {
    const url = request.url();
    if (!url.includes('/api/v1/')) {
      return;
    }
    samples.push({ url, tenantHeader: request.headers()['x-tenant-id'] });
  });
  return samples;
}

const businessSamplesFor = (samples: HeaderSample[], fragment: string) =>
  samples.filter((sample) => sample.url.includes(fragment));

async function adminApiLogin(request: APIRequestContext) {
  return loginByApi(request, adminCredentials);
}

async function provisionTenant(request: APIRequestContext) {
  const login = await adminApiLogin(request);
  const response = await request.post(`${apiBaseUrl}/system/tenant`, {
    headers: {
      ...apiRequestHeaders(login),
      'X-Operation-Token': await getApiOperationToken(request, login),
    },
    data: {
      code: tenantCode,
      name: tenantName,
      plan: 'free',
      admin_username: tenantAdminUsername,
      admin_password: tenantAdminPassword,
      admin_nickname: `Smoke Viewpoint Admin ${run}`,
    },
  });
  const payload = await response.json();
  expect(payload.code, `tenant provision: ${JSON.stringify(payload)}`).toBe(200);
  return payload.data.id as number;
}

async function signInAsPlatformAdmin(page: Page) {
  await signInWithUi(page, adminCredentials);
  await expect(page.locator('.app-shell__header')).toBeVisible();
}

/** Pin the tenant through the real switcher (dropdown + full shell reload). */
async function pinTenantViaSwitcher(page: Page, name: string) {
  await page.getByRole('button', { name: VIEWPOINT_TITLE }).click();
  const item = page.locator('.arco-dropdown-menu-item', { hasText: name }).first();
  await expect(item).toBeVisible();
  await Promise.all([page.waitForLoadState('load'), item.click()]);
  await expect(page.locator('.tenant-viewpoint-banner')).toBeVisible();
}

async function openUserList(page: Page) {
  await page.goto('/system/user', { waitUntil: 'domcontentloaded' });
  await expect(page.locator('.app-shell__header')).toBeVisible();
  await expect(page.locator('.arco-table')).toBeVisible();
}

/**
 * Waits until the shell has actually issued the business call and returns every
 * sample of it. The header renders before the list XHR resolves, so a plain
 * "assert the array is non-empty" right after mounting is racy.
 */
async function settledBusinessSamples(samples: HeaderSample[], fragment: string) {
  await expect
    .poll(() => businessSamplesFor(samples, fragment).length, { timeout: 15_000 })
    .toBeGreaterThan(0);
  return businessSamplesFor(samples, fragment);
}

test.describe.configure({ mode: 'serial' });

test.describe('Tenant Viewpoint @priority:critical @smoke:tenant', () => {
  test.beforeAll(async ({ request }) => {
    adminLogin = await adminApiLogin(request);
    tenantId = await provisionTenant(request);
    expect(tenantId).toBeGreaterThan(0);
  });

  test.afterAll(async ({ request }) => {
    const login = await adminApiLogin(request);
    const operationToken = await getApiOperationToken(request, login);
    const headers = { ...apiRequestHeaders(login), 'X-Operation-Token': operationToken };
    const tenantHeaders = { ...headers, 'X-Tenant-Id': String(tenantId) };

    // Retire the tenant the way the contract says to. §2.2 refuses to remove the
    // last owner, so a tenant can never reach "no active members" by purging
    // memberships alone; §2.1 makes archived the retirement path, and deleting an
    // archived tenant disables whatever memberships are left. Purging is still
    // attempted first so the fixture does not depend on the archived path working
    // for tenants it could clean up directly.
    const members = await request.get(`${apiBaseUrl}/system/tenant/${tenantId}/members`, {
      headers,
    });
    const memberPayload = await members.json();
    const memberRows = (memberPayload.data?.items ?? []) as Array<{ user_id: number }>;
    for (const member of memberRows) {
      await request.delete(`${apiBaseUrl}/system/tenant/${tenantId}/members/${member.user_id}`, {
        headers: tenantHeaders,
      });
    }

    if (createdUserId > 0) {
      await request.delete(`${apiBaseUrl}/system/user/${createdUserId}`, { headers: tenantHeaders });
    }

    const archived = await request.put(`${apiBaseUrl}/system/tenant/${tenantId}`, {
      headers,
      data: { status: 'archived' },
    });
    expect((await archived.json()).code, 'tenant teardown: archive before delete').toBe(200);

    const deleted = await request.delete(`${apiBaseUrl}/system/tenant/${tenantId}`, { headers });
    const deletedPayload = await deleted.json();
    expect(deletedPayload.code, `tenant teardown: ${JSON.stringify(deletedPayload)}`).toBe(200);
  });

  test('平台管理员默认处于平台全局视角：不发 X-Tenant-Id，且没有横幅', async ({ page }) => {
    const samples = recordTenantHeader(page);
    await signInAsPlatformAdmin(page);
    await openUserList(page);

    // 切换器存在并回显平台全局（说明平台能力门控 + 活跃租户列表都就绪）。
    const trigger = page.getByRole('button', { name: VIEWPOINT_TITLE });
    await expect(trigger).toBeVisible();
    await expect(trigger).toContainText(PLATFORM_SCOPE_LABEL);
    await expect(page.locator('.tenant-viewpoint-banner')).toHaveCount(0);

    await settledBusinessSamples(samples, '/api/v1/system/user/list');
    expect(samples.filter((sample) => sample.tenantHeader !== undefined)).toEqual([]);
    await capture(page, '01-platform-scope-default');
  });

  test('钉住租户后：全量重载、横幅常驻、业务请求带上 X-Tenant-Id、列表被限定在该租户', async ({
    page,
  }) => {
    await signInAsPlatformAdmin(page);
    await pinTenantViaSwitcher(page, tenantName);

    await expect(page.locator('.tenant-viewpoint-banner')).toContainText(BANNER_TEXT);
    await capture(page, '02-viewpoint-pinned-banner');

    const samples = recordTenantHeader(page);
    await openUserList(page);

    const listCalls = await settledBusinessSamples(samples, '/api/v1/system/user/list');
    expect(listCalls.every((sample) => sample.tenantHeader === String(tenantId))).toBe(true);

    // 租户域内只应看到该租户自己的管理员，看不到平台账号。
    await expect(page.getByRole('cell', { name: tenantAdminUsername })).toBeVisible();
    await expect(page.getByRole('cell', { name: 'admin', exact: true })).toHaveCount(0);
  });

  test('钉住视角内新建用户并加入本租户：membership 落库', async ({ page, request }) => {
    await signInAsPlatformAdmin(page);
    await pinTenantViaSwitcher(page, tenantName);
    await openUserList(page);

    await page.getByRole('button', { name: '新增' }).click();
    await expect(page.getByText('新增用户')).toBeVisible();

    await page.getByLabel('用户名').fill(viewpointUsername);
    // PasswordInput's toggle button is labelled 「显示密码」, so getByLabel('密码')
    // is ambiguous; the modal carries exactly one password field.
    await page.locator('.arco-modal').locator('input[type="password"]').fill(viewpointPassword);
    await page.getByLabel('昵称').fill('Viewpoint User');

    // 「加入本租户（可登录）」默认勾选，成员角色默认 member。
    const joinCheckbox = page.getByRole('checkbox', { name: /加入本租户/ });
    await expect(joinCheckbox).toBeChecked();
    await capture(page, '03-create-modal-join-default-on');
    await page.locator('.arco-modal').getByRole('button', { name: '新增' }).click();

    await expect(page.getByText('用户已创建并加入租户成员')).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole('cell', { name: viewpointUsername })).toBeVisible();
    await capture(page, '04-user-created-and-joined');

    // 通过 API 复核：该用户确实成为租户的 active member，且行归属该租户。
    const response = await request.get(`${apiBaseUrl}/system/user/list`, {
      headers: { ...apiRequestHeaders(adminLogin), 'X-Tenant-Id': String(tenantId) },
      params: { username: viewpointUsername, page: 1, pageSize: 10 },
    });
    const payload = await response.json();
    const rows = (payload.data?.items ?? []) as Array<{ id: number; username: string; tenantId?: number }>;
    const created = rows.find((row) => row.username === viewpointUsername);
    expect(created, `created user not found in tenant scope: ${JSON.stringify(payload)}`).toBeTruthy();
    createdUserId = created!.id;

    const members = await request.get(`${apiBaseUrl}/system/tenant/${tenantId}/members`, {
      headers: apiRequestHeaders(adminLogin),
    });
    const memberPayload = await members.json();
    const memberRows = (memberPayload.data?.items ?? []) as Array<{ user_id: number; role: string; status: string }>;
    const membership = memberRows.find((row) => row.user_id === createdUserId);
    expect(membership, `membership missing: ${JSON.stringify(memberPayload)}`).toBeTruthy();
    expect(membership!.status).toBe('active');
  });

  test('该用户能真正登录：新会话回到平台全局视角，不再带头也不再显示切换器', async ({ page }) => {
    expect(createdUserId, 'run the create-user case first').toBeGreaterThan(0);

    const samples = recordTenantHeader(page);
    // 上一个用例的视角属于前一个浏览器上下文，这里用全新会话验证登录闸门。
    await signInWithUi(page, { username: viewpointUsername, password: viewpointPassword });

    await expect(page.locator('.app-shell__header')).toBeVisible();
    await expect(page.locator('.tenant-viewpoint-banner')).toHaveCount(0);
    await capture(page, '05-joined-user-signed-in');
    // 非平台主体：能力门控下切换器不渲染。
    await expect(page.getByRole('button', { name: VIEWPOINT_TITLE })).toHaveCount(0);
    expect(samples.filter((sample) => sample.tenantHeader !== undefined)).toEqual([]);
  });

  test('退出租户视角：回到平台全局，请求不再带头', async ({ page }) => {
    await signInAsPlatformAdmin(page);
    await pinTenantViaSwitcher(page, tenantName);

    const samples = recordTenantHeader(page);
    await Promise.all([
      page.waitForLoadState('load'),
      page.getByRole('button', { name: EXIT_VIEWPOINT }).click(),
    ]);
    await expect(page.locator('.tenant-viewpoint-banner')).toHaveCount(0);

    await openUserList(page);
    await capture(page, '06-viewpoint-exited-platform-scope');
    const trigger = page.getByRole('button', { name: VIEWPOINT_TITLE });
    await expect(trigger).toContainText(PLATFORM_SCOPE_LABEL);
    expect(samples.filter((sample) => sample.tenantHeader !== undefined)).toEqual([]);
  });
});
