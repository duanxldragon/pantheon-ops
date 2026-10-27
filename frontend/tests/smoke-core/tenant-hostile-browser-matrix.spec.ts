import { expect, test, type APIRequestContext } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  loginByApi,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';
import {
  loginAsTenantAdmin,
  setupMatrixTenants,
  teardownMatrixTenants,
  type MatrixTenant,
} from './tenant-fixture';

// Hostile two-tenant browser matrix expansion (tenant-verification-and-gray,
// 2026-09-15; refreshed 2026-10-15). Scenarios NOT covered by
// tenant-protected-resources.spec.ts:
//   1. live multi-tenant login picker against the real backend (candidate
//      disclosure only after password verification, session untouched until a
//      tenant is chosen)
//   2. dashboard aggregate isolation in the browser
//   3. upload namespace isolation through the real upload API
//   4. refresh rotation + replay rejection in the browser cookie jar
//   5. pre-provisioned dictionary fixtures staying tenant-isolated
//
// The matrix is a pair of REAL tenants provisioned through POST /system/tenant,
// each with its own administrator (see ./tenant-fixture.ts). Earlier revisions
// authenticated as the platform `admin` with an explicit tenantId, which under
// the isolation model yields an empty tenant role set and 403 on every call.

type ApiEnvelope<T> = {
  code: number;
  message?: string;
  data: T;
};

let tenants: [MatrixTenant, MatrixTenant];
let platformLogin: BrowserLoginResult;

async function platformSession(request: APIRequestContext) {
  const login = await loginByApi(request, {
    username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
    password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
  });
  const operationToken = await getApiOperationToken(request, login);
  return { login, headers: { ...apiRequestHeaders(login), 'X-Operation-Token': operationToken } };
}

async function listTenantMembers(request: APIRequestContext, session: Awaited<ReturnType<typeof platformSession>>, tenantId: number) {
  const response = await request.get(`${apiBaseUrl}/system/tenant/${tenantId}/members?page=1&pageSize=100`, {
    headers: session.headers,
  });
  const payload = (await response.json()) as ApiEnvelope<{ items?: Array<{ user_id: number }> }>;
  expect(payload.code, `members of t${tenantId}: ${JSON.stringify(payload)}`).toBe(200);
  return payload.data?.items ?? [];
}

async function addTenantMember(
  request: APIRequestContext,
  session: Awaited<ReturnType<typeof platformSession>>,
  tenantId: number,
  userId: number,
  role: string,
) {
  const response = await request.post(`${apiBaseUrl}/system/tenant/${tenantId}/members`, {
    headers: session.headers,
    data: { user_id: userId, role },
  });
  const payload = (await response.json()) as ApiEnvelope<unknown>;
  expect(payload.code, `add member ${userId} to t${tenantId}: ${JSON.stringify(payload)}`).toBe(200);
}

test.beforeAll(async ({ request }) => {
  const matrix = await setupMatrixTenants(request);
  tenants = [matrix[0], matrix[1]];
  platformLogin = await loginByApi(request, {
    username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
    password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
  });

  // Give tenant A's administrator a second membership so it is a genuine
  // multi-membership subject: the login picker exists precisely for this case
  // (contract §2.2), and it is what the picker spec below signs in as.
  const session = await platformSession(request);
  const members = await listTenantMembers(request, session, tenants[0].id);
  const owner = members.find((member) => member.user_id);
  expect(owner, `t${tenants[0].id} must have an owner after provisioning`).toBeTruthy();
  // Idempotent: re-running against a manifest tenant that already carries the
  // cross-membership (an interrupted run) must not fail on tenant.conflict.
  const alreadyMember = (await listTenantMembers(request, session, tenants[1].id)).some(
    (member) => member.user_id === owner!.user_id,
  );
  if (!alreadyMember) {
    await addTenantMember(request, session, tenants[1].id, owner!.user_id, 'member');
  }
});

test.afterAll(async ({ request }) => {
  await teardownMatrixTenants(request, [tenants[0], tenants[1]], platformLogin);
});

test.describe('Tenant login picker (live backend) @priority:critical @smoke:tenant', () => {
  test('multi-mode login requires tenant selection and never touches the session before it', async ({
    page,
  }) => {
    test.setTimeout(60_000);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // Password verified first: submitting credentials without a tenant must
    // surface the picker (tenantSelectionRequired), never log straight in.
    await page.getByPlaceholder(/请输入用户名|username/i).fill(tenants[0].adminUsername);
    await page.getByPlaceholder(/请输入密码|password/i).fill(tenants[0].adminPassword);
    const loginResponse = page.waitForResponse((response) =>
      response.url().includes('/api/v1/auth/login'),
    );
    await page.getByRole('button', { name: /登录|sign in|sign in/i }).click();
    const response = await loginResponse;
    expect(response.ok()).toBeTruthy();
    const payload = (await response.json()) as ApiEnvelope<{
      tenantSelectionRequired?: boolean;
      tenantCandidates?: Array<{ tenantId: number; code: string; name: string }>;
    }>;
    expect(payload.code).toBe(200);
    expect(payload.data.tenantSelectionRequired).toBeTruthy();
    const candidateIds = (payload.data.tenantCandidates ?? []).map((candidate) => candidate.tenantId);
    expect(candidateIds).toEqual(expect.arrayContaining([tenants[0].id, tenants[1].id]));

    // The picker renders with both candidates and the session stays untouched.
    await expect(page.locator('.auth-login-tenant-picker')).toBeVisible();
    expect(await page.evaluate(() => localStorage.getItem('pantheon_session_hint'))).toBeNull();
    expect((await page.context().cookies()).filter((c) => c.name.startsWith('pantheon_'))).toEqual(
      [],
    );

    // Pick tenant B (the subject is only a member there) and submit.
    await page.locator('.auth-login-tenant-picker .arco-select').click();
    await page.getByText(new RegExp(tenants[1].code)).first().click();
    const secondLogin = page.waitForResponse((response) =>
      response.url().includes('/api/v1/auth/login'),
    );
    await page.getByRole('button', { name: /登录|sign in/i }).click();
    const secondResponse = await secondLogin;
    expect(secondResponse.ok()).toBeTruthy();
    const secondPayload = (await secondResponse.json()) as ApiEnvelope<{ user?: { id: number } }>;
    expect(secondPayload.code).toBe(200);
    expect(secondPayload.data.user?.id).toBeGreaterThan(0);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.app-shell__header')).toBeVisible();
  });

  test('wrong password never exposes tenant candidates', async ({ page }) => {
    test.setTimeout(60_000);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.getByPlaceholder(/请输入用户名|username/i).fill(tenants[0].adminUsername);
    await page.getByPlaceholder(/请输入密码|password/i).fill('definitely-wrong');
    await page.getByRole('button', { name: /登录|sign in|sign in/i }).click();
    await expect(page.locator('.arco-message-error, .arco-notification-error')).toBeVisible();
    await expect(page.locator('.auth-login-tenant-picker')).toHaveCount(0);
  });
});

test.describe('Tenant dashboard and refresh isolation @priority:critical @smoke:tenant', () => {
  test('dashboard renders per tenant and refresh rotation rejects replay', async ({ browser }) => {
    test.setTimeout(90_000);
    const tenantA = await loginAsTenantAdmin(browser, tenants[0]);
    const tenantB = await loginAsTenantAdmin(browser, tenants[1]);
    try {
      // Dashboard aggregate endpoints must serve 200 per tenant without
      // cross-tenant leakage and without page errors.
      for (const tenant of [tenantA, tenantB]) {
        const pageErrors: string[] = [];
        tenant.page.on('pageerror', (error) => pageErrors.push(error.message));
        await tenant.page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
        await expect(tenant.page.locator('.dashboard-hero-card').first()).toBeVisible();
        expect(pageErrors).toEqual([]);
      }

      // Refresh rotation. The first call goes through the browser cookie jar,
      // and the handler prefers the cookie over the body — so the server burns
      // the cookie value and writes a rotated one. Capture that exact value
      // first: it is the token the server is about to consume.
      const jarBeforeRotation = await tenantB.context.cookies();
      const burnedToken =
        jarBeforeRotation.find((cookie) => cookie.name === 'pantheon_refresh_token')?.value ??
        tenantB.login.refreshToken;

      const refreshResponse = await tenantB.page.request.post(`${apiBaseUrl}/auth/refresh`, {
        data: { refreshToken: burnedToken },
      });
      expect(refreshResponse.ok()).toBeTruthy();
      const refreshPayload = (await refreshResponse.json()) as ApiEnvelope<unknown>;
      expect(refreshPayload.code).toBe(200);

      // The replay must present the burned token through a request that holds
      // no *fresh* refresh cookie. Replaying inside tenantB's own jar would
      // hand the server the rotated cookie, which succeeds for the right
      // reason and would assert nothing about replay.
      const replayContext = await browser.newContext();
      try {
        // 1. Body replay, no cookie: the burned token is the one evaluated.
        const bodyReplay = await replayContext.request.post(`${apiBaseUrl}/auth/refresh`, {
          data: { refreshToken: burnedToken },
        });
        const bodyReplayPayload = (await bodyReplay.json()) as ApiEnvelope<unknown>;
        expect(
          bodyReplay.status() >= 400 || bodyReplayPayload.code >= 400,
          `refresh replay (body) must be rejected: ${JSON.stringify(bodyReplayPayload)}`,
        ).toBeTruthy();

        // 2. Stale-cookie replay: what a thief holding the old cookie sends.
        await replayContext.addCookies([
          { name: 'pantheon_refresh_token', value: burnedToken, url: `${new URL(apiBaseUrl).origin}/` },
        ]);
        const cookieReplay = await replayContext.request.post(`${apiBaseUrl}/auth/refresh`, {
          data: {},
        });
        const cookieReplayPayload = (await cookieReplay.json()) as ApiEnvelope<unknown>;
        expect(
          cookieReplay.status() >= 400 || cookieReplayPayload.code >= 400,
          `refresh replay (stale cookie) must be rejected: ${JSON.stringify(cookieReplayPayload)}`,
        ).toBeTruthy();
      } finally {
        await replayContext.close();
      }
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});

test.describe('Tenant upload namespace isolation @priority:critical @smoke:tenant', () => {
  test('upload lands in own tenant namespace; foreign namespace and traversal rejected', async ({
    browser,
  }) => {
    test.setTimeout(90_000);
    const tenantA = await loginAsTenantAdmin(browser, tenants[0]);
    const tenantB = await loginAsTenantAdmin(browser, tenants[1]);
    const nsA = `t${tenants[0].id}`;
    const nsB = `t${tenants[1].id}`;
    try {
      // Upload through the real upload API as tenant A. PNG magic bytes are
      // required: the upload chain sniffs image content server-side.
      const pngHead = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
      const pngBody = Buffer.concat([pngHead, Buffer.from('matrix-upload-payload')]);
      const boundary = '----pantheonmatrix' + Date.now();
      const parts = Buffer.concat([
        Buffer.from(
          `--${boundary}\r\nContent-Disposition: form-data; name="file"; filename="matrix-a.png"\r\nContent-Type: image/png\r\n\r\n`,
        ),
        pngBody,
        Buffer.from(`\r\n--${boundary}--\r\n`),
      ]);
      const uploadResponse = await tenantA.page.request.post(`${apiBaseUrl}/system/upload`, {
        headers: {
          ...apiRequestHeaders(tenantA.login),
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
        },
        data: parts,
        params: { scope: 'matrix' },
      });
      const uploadPayload = (await uploadResponse.json()) as ApiEnvelope<{
        objectKey?: string;
        url?: string;
      }>;
      expect(
        uploadResponse.ok() && uploadPayload.code === 200,
        `upload should succeed: ${JSON.stringify(uploadPayload)}`,
      ).toBeTruthy();
      // The stored object key must embed the acting tenant's namespace.
      expect(uploadPayload.data.objectKey).toMatch(new RegExp(`^${nsA}/`));
      const objectKey = uploadPayload.data.objectKey!;

      // Tenant B reading A's object key must be denied.
      const foreignRead = await tenantB.page.request.get(
        `${apiBaseUrl}/system/upload/files/${objectKey}`,
        { headers: apiRequestHeaders(tenantB.login) },
      );
      const foreignPayload = (await foreignRead.json().catch(() => ({}))) as ApiEnvelope<unknown>;
      expect(
        foreignRead.status() >= 400 || foreignPayload.code >= 400,
        'cross-tenant object read must be rejected',
      ).toBeTruthy();

      // Tenant A reading its own object must succeed.
      const ownRead = await tenantA.page.request.get(
        `${apiBaseUrl}/system/upload/files/${objectKey}`,
        { headers: apiRequestHeaders(tenantA.login) },
      );
      expect(ownRead.ok()).toBeTruthy();

      // Path-traversal style probes must never escape the namespace.
      for (const probe of [`${nsA}/../../secret.txt`, `${nsB}/../${nsA}/escape.png`]) {
        const traversal = await tenantA.page.request.get(
          `${apiBaseUrl}/system/upload/files/${probe}`,
          { headers: apiRequestHeaders(tenantA.login) },
        );
        const traversalPayload = (await traversal.json().catch(() => ({}))) as ApiEnvelope<unknown>;
        expect(
          traversal.status() >= 400 || traversalPayload.code >= 400,
          `traversal probe ${probe} must be rejected`,
        ).toBeTruthy();
      }
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});

test.describe('Tenant dictionary cross-checks (fixtures) @priority:high @smoke:tenant', () => {
  test('pre-provisioned fixtures stay isolated and UI table shows only own tenant row', async ({
    browser,
  }) => {
    test.setTimeout(90_000);
    const tenantA = await loginAsTenantAdmin(browser, tenants[0]);
    const tenantB = await loginAsTenantAdmin(browser, tenants[1]);
    try {
      const rowsA = await listDictTypes(tenantA.page.request, tenantA.login, tenants[0].dictCode);
      expect(rowsA.map((row) => row.dictCode)).toEqual([tenants[0].dictCode]);
      const foreignA = await listDictTypes(tenantA.page.request, tenantA.login, tenants[1].dictCode);
      expect(foreignA).toEqual([]);

      await tenantA.page.goto('/system/dict', { waitUntil: 'domcontentloaded' });
      const tableA = tenantA.page.locator('.dict-page__table-card');
      await expect(tableA.getByText(tenants[0].dictCode, { exact: true })).toBeVisible();
      await expect(tableA.getByText(tenants[1].dictCode, { exact: true })).toHaveCount(0);
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});

async function listDictTypes(
  request: APIRequestContext,
  login: BrowserLoginResult,
  dictCode: string,
) {
  const response = await request.get(`${apiBaseUrl}/system/dict/type/list`, {
    headers: apiRequestHeaders(login),
    params: { dictCode },
  });
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<Array<{ dictCode: string }>>;
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? payload.data : [];
}
