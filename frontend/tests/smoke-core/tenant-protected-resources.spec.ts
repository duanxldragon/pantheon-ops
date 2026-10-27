import { expect, test, type APIRequestContext, type Page } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  loginByApi,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';
import {
  loginAsTenantAdmin,
  setupMatrixTenants,
  teardownMatrixTenants,
  type MatrixTenant,
} from './tenant-fixture';

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

/**
 * Provisioned in `beforeAll` (see ./tenant-fixture.ts). The matrix is a pair of
 * real tenants with their own administrators — a platform admin naming a tenant
 * has no tenant-scoped role there and would be refused with 403.
 */
let tenants: [MatrixTenant, MatrixTenant];
let platformLogin: BrowserLoginResult;

function listRows(payload: ApiEnvelope<DictTypeRow[] | { items?: DictTypeRow[] }>) {
  if (Array.isArray(payload.data)) {
    return payload.data;
  }
  return Array.isArray(payload.data?.items) ? payload.data.items : [];
}

async function createDictType(
  request: APIRequestContext,
  login: BrowserLoginResult,
  dictCode: string,
) {
  const response = await request.post(`${apiBaseUrl}/system/dict/type`, {
    headers: apiRequestHeaders(login),
    data: {
      dictCode,
      // The name must differ from the code: the dict table renders the code in
      // its own column AND the name column falls back to the raw value when no
      // translation exists (`t(value, value)`). Sharing one string makes
      // `getByText(code, { exact: true })` resolve to two cells, so the
      // isolation assertion below would be a strict-mode violation or a race.
      dictName: `E2E ${dictCode}`,
      module: 'system',
      status: 1,
      remark: 'tenant-browser-e2e',
    },
  });
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<DictTypeRow>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function deleteDictType(
  request: APIRequestContext,
  login: BrowserLoginResult,
  id: number,
) {
  const response = await request.delete(`${apiBaseUrl}/system/dict/type/${id}`, {
    headers: apiRequestHeaders(login),
  });
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<{ deleted: boolean }>;
  expect(payload.code).toBe(200);
}

async function listDictTypes(request: APIRequestContext, login: BrowserLoginResult, dictCode: string) {
  const response = await request.get(`${apiBaseUrl}/system/dict/type/list`, {
    headers: apiRequestHeaders(login),
    params: { dictCode },
  });
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<DictTypeRow[] | { items?: DictTypeRow[] }>;
  expect(payload.code).toBe(200);
  return listRows(payload);
}

async function assertProtectedPage(page: Page, path: string, selector: string) {
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await expect(page.locator(selector)).toBeVisible();
  expect(pageErrors).toEqual([]);
}

test.describe('Tenant protected resources @priority:critical @smoke:tenant', () => {
  test.beforeAll(async ({ request }) => {
    const matrix = await setupMatrixTenants(request);
    tenants = [matrix[0], matrix[1]];
    platformLogin = await loginByApi(request, {
      username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
      password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
    });
  });

  test.afterAll(async ({ request }) => {
    await teardownMatrixTenants(request, [tenants[0], tenants[1]], platformLogin);
  });

  test('isolates protected pages, dictionary reads, exports, and ID tampering', async ({ browser }) => {
    test.setTimeout(60_000);
    const suffix = `${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
    const dictCodeA = `tenant_e2e_a_${suffix}`;
    const dictCodeB = `tenant_e2e_b_${suffix}`;
    const tenantA = await loginAsTenantAdmin(browser, tenants[0]);
    const tenantB = await loginAsTenantAdmin(browser, tenants[1]);

    let dictA: DictTypeRow | undefined;
    let dictB: DictTypeRow | undefined;
    try {
      dictA = await createDictType(tenantA.page.request, tenantA.login, dictCodeA);
      dictB = await createDictType(tenantB.page.request, tenantB.login, dictCodeB);

      await assertProtectedPage(tenantA.page, '/dashboard', '.dashboard-hero-card');
      await assertProtectedPage(tenantB.page, '/dashboard', '.dashboard-hero-card');
      await assertProtectedPage(tenantA.page, '/system/dict', '.dict-page__table-card');
      await assertProtectedPage(tenantB.page, '/system/dict', '.dict-page__table-card');

      const tableA = tenantA.page.locator('.dict-page__table-card');
      const tableB = tenantB.page.locator('.dict-page__table-card');
      await expect(tableA.getByText(dictCodeA, { exact: true })).toBeVisible();
      await expect(tableA.getByText(dictCodeB, { exact: true })).toHaveCount(0);
      await expect(tableB.getByText(dictCodeB, { exact: true })).toBeVisible();
      await expect(tableB.getByText(dictCodeA, { exact: true })).toHaveCount(0);

      const rowsA = await listDictTypes(tenantA.page.request, tenantA.login, dictCodeA);
      const rowsAForeign = await listDictTypes(tenantA.page.request, tenantA.login, dictCodeB);
      const rowsB = await listDictTypes(tenantB.page.request, tenantB.login, dictCodeB);
      const rowsBForeign = await listDictTypes(tenantB.page.request, tenantB.login, dictCodeA);
      expect(rowsA.map((row) => row.dictCode)).toEqual([dictCodeA]);
      expect(rowsAForeign).toEqual([]);
      expect(rowsB.map((row) => row.dictCode)).toEqual([dictCodeB]);
      expect(rowsBForeign).toEqual([]);

      const exportA = await tenantA.page.request.post(`${apiBaseUrl}/system/dict/type/export`, {
        headers: apiRequestHeaders(tenantA.login),
        data: { dictCode: dictCodeA },
      });
      expect(exportA.ok()).toBeTruthy();
      const csvA = await exportA.text();
      expect(csvA).toContain(dictCodeA);
      expect(csvA).not.toContain(dictCodeB);

      const tamperedUpdate = await tenantB.page.request.put(
        `${apiBaseUrl}/system/dict/type/${dictA.id}`,
        {
          headers: apiRequestHeaders(tenantB.login),
          data: {
            dictCode: dictCodeA,
            dictName: 'cross-tenant-tamper',
            module: 'system',
            status: 1,
            remark: 'must be rejected',
          },
        },
      );
      const tamperedPayload = (await tamperedUpdate.json()) as ApiEnvelope<unknown>;
      expect(
        tamperedUpdate.status() >= 400 || tamperedPayload.code >= 400,
        `cross-tenant update must be rejected: ${JSON.stringify(tamperedPayload)}`,
      ).toBeTruthy();

      const tamperedBatch = await tenantB.page.request.post(
        `${apiBaseUrl}/system/dict/type/batch-status`,
        {
          headers: apiRequestHeaders(tenantB.login),
          data: { typeIds: [dictA.id], status: 2 },
        },
      );
      expect(tamperedBatch.ok()).toBeTruthy();
      const batchPayload = (await tamperedBatch.json()) as ApiEnvelope<{ updatedCount: number }>;
      expect(
        tamperedBatch.status() >= 400 || batchPayload.code >= 400,
        `cross-tenant batch update must be rejected: ${JSON.stringify(batchPayload)}`,
      ).toBeTruthy();
    } finally {
      if (dictA) {
        await deleteDictType(tenantA.page.request, tenantA.login, dictA.id);
      }
      if (dictB) {
        await deleteDictType(tenantB.page.request, tenantB.login, dictB.id);
      }
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});
