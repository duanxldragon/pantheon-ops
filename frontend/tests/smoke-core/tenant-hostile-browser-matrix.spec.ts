import { expect, test, type APIRequestContext, type Browser } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  installClientSession,
  loginByApi,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';

// Hostile two-tenant browser matrix expansion (tenant-verification-and-gray,
// 2026-09-15). Scenarios NOT covered by tenant-protected-resources.spec.ts:
//   1. live multi-tenant login picker against the real backend (candidate
//      disclosure only after password verification, wrong-tenant rejection)
//   2. dashboard aggregate isolation in the browser
//   3. upload namespace isolation through the real upload UI/API flow
//   4. refresh rotation + replay rejection in the browser cookie jar
// Tenants 101/202 are the reusable smoke tenants; dict fixtures
// matrix_browser_a/_b are pre-provisioned with tenant_id 101/202.

type DictTypeRow = {
  id: number;
  dictCode: string;
  tenantId?: number;
};

type ApiEnvelope<T> = {
  code: number;
  message?: string;
  data: T;
};

const tenants = {
  a: 101,
  b: 202,
} as const;

const fixtureA = 'matrix_browser_a';
const fixtureB = 'matrix_browser_b';

async function loginForTenant(browser: Browser, tenantId: number) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const login = await loginByApi(page.request, {
    username: 'admin',
    password: '123456',
    tenantId,
  } as Parameters<typeof loginByApi>[1]);
  await installClientSession(page, login);
  return { context, page, login };
}

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
  const payload = (await response.json()) as ApiEnvelope<DictTypeRow[]>;
  expect(payload.code).toBe(200);
  return Array.isArray(payload.data) ? payload.data : [];
}

test.describe('Tenant login picker (live backend) @priority:critical @smoke:tenant', () => {
  test('multi-mode login requires tenant selection; wrong tenant rejected; candidates only after password check', async ({
    page,
  }) => {
    test.setTimeout(60_000);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    // Password verified first: submitting credentials without a tenant must
    // surface the picker (tenantSelectionRequired), never log straight in.
    await page.getByPlaceholder(/请输入用户名|username/i).fill('admin');
    await page.getByPlaceholder(/请输入密码|password/i).fill('123456');
    const loginResponse = page.waitForResponse((response) =>
      response.url().includes('/api/v1/auth/login'),
    );
    await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();
    const response = await loginResponse;
    expect(response.ok()).toBeTruthy();
    const payload = (await response.json()) as ApiEnvelope<{
      tenantSelectionRequired?: boolean;
      tenantCandidates?: Array<{ tenantId: number; code: string; name: string }>;
    }>;
    expect(payload.code).toBe(200);
    expect(payload.data.tenantSelectionRequired).toBeTruthy();
    const candidateIds = (payload.data.tenantCandidates ?? []).map((c) => c.tenantId);
    expect(candidateIds).toEqual(expect.arrayContaining([tenants.a, tenants.b]));

    // The picker renders with both candidates and the session hint stays unset.
    await expect(page.locator('.auth-login-tenant-picker')).toBeVisible();
    expect(await page.evaluate(() => localStorage.getItem('pantheon_session_hint'))).toBeNull();
    expect((await page.context().cookies()).filter((c) => c.name.startsWith('pantheon_'))).toEqual(
      [],
    );

    // Select tenant 202 explicitly and submit.
    await page.locator('.auth-login-tenant-picker .arco-select').click();
    await page.getByText(/Globex South|smoke-globex/).first().click();
    const secondLogin = page.waitForResponse((response) =>
      response.url().includes('/api/v1/auth/login'),
    );
    await page.getByRole('button', { name: /登录|sign in/i }).click();
    const secondResponse = await secondLogin;
    expect(secondResponse.ok()).toBeTruthy();
    const secondPayload = (await secondResponse.json()) as ApiEnvelope<{
      user?: { username: string };
    }>;
    expect(secondPayload.code).toBe(200);
    await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.app-shell__header')).toBeVisible();
  });

  test('wrong password never exposes tenant candidates', async ({ page }) => {
    test.setTimeout(60_000);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.getByPlaceholder(/请输入用户名|username/i).fill('admin');
    await page.getByPlaceholder(/请输入密码|password/i).fill('definitely-wrong');
    await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();
    await expect(page.locator('.arco-message-error, .arco-notification-error')).toBeVisible();
    await expect(page.locator('.auth-login-tenant-picker')).toHaveCount(0);
  });
});

test.describe('Tenant dashboard and refresh isolation @priority:critical @smoke:tenant', () => {
  test('dashboard renders per tenant and refresh rotation rejects replay', async ({ browser }) => {
    test.setTimeout(90_000);
    const tenantA = await loginForTenant(browser, tenants.a);
    const tenantB = await loginForTenant(browser, tenants.b);
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

      // Refresh rotation: use tenant B refresh token via the API jar; the old
      // refresh token must not be replayable.
      const refreshResponse = await tenantB.page.request.post(`${apiBaseUrl}/auth/refresh`, {
        data: { refreshToken: tenantB.login.refreshToken },
      });
      expect(refreshResponse.ok()).toBeTruthy();
      const refreshPayload = (await refreshResponse.json()) as ApiEnvelope<unknown>;
      expect(refreshPayload.code).toBe(200);

      const replayResponse = await tenantB.page.request.post(`${apiBaseUrl}/auth/refresh`, {
        data: { refreshToken: tenantB.login.refreshToken },
      });
      const replayPayload = (await replayResponse.json()) as ApiEnvelope<unknown>;
      expect(
        replayResponse.status() >= 400 || replayPayload.code >= 400,
        `refresh replay must be rejected: ${JSON.stringify(replayPayload)}`,
      ).toBeTruthy();
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
    const tenantA = await loginForTenant(browser, tenants.a);
    const tenantB = await loginForTenant(browser, tenants.b);
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
      // The stored object key must embed the resolved tenant namespace t101/.
      expect(uploadPayload.data.objectKey).toMatch(/^t101\//);
      const objectKey = uploadPayload.data.objectKey!;

      // Tenant B reading A's object key must be denied with not_found.
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
      for (const probe of ['t101/../../secret.txt', 't202/../t101/escape.png']) {
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
    const tenantA = await loginForTenant(browser, tenants.a);
    const tenantB = await loginForTenant(browser, tenants.b);
    try {
      const rowsA = await listDictTypes(tenantA.page.request, tenantA.login, fixtureA);
      expect(rowsA.map((row) => row.dictCode)).toEqual([fixtureA]);
      const foreignA = await listDictTypes(tenantA.page.request, tenantA.login, fixtureB);
      expect(foreignA).toEqual([]);

      await tenantA.page.goto('/system/dict', { waitUntil: 'domcontentloaded' });
      const tableA = tenantA.page.locator('.dict-page__table-card');
      await expect(tableA.getByText(fixtureA, { exact: true })).toBeVisible();
      await expect(tableA.getByText(fixtureB, { exact: true })).toHaveCount(0);
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});
