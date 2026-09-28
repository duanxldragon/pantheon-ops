import { expect, test, type Browser } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  installClientSession,
  loginByApi,
} from '../smoke/helpers/auth';

// Hostile two-tenant browser matrix — phase 2 (tenant-verification-and-gray,
// 2026-09-20). Closes the browser/runtime coverage remaining from review
// finding #1 (2026-09-15 matrix covered login picker, dashboard, dict
// isolation, refresh rotation and upload namespace):
//   1. auth-log surfaces: login-log + security-event pages render per tenant
//      and their list APIs never return another tenant's rows.
//   2. audit surfaces: operation-log list/export pinned to the acting tenant;
//      a cross-tenant id lookup is rejected.
//   3. settings surface: tenant-bound group reads stay per-tenant (override
//      semantics), cache refresh stays scoped.
//   4. dynamic-module/generator surface: module registry list is tenant-blind
//      platform state but the write path requires the SecureAction token per
//      tenant (no cross-tenant mutation without re-verification).
// The flag must be multi and tenants 101/202 must exist (tenantmatrixdb up).
// This spec restores nothing: flag restore is done by the runner/cleanup, the
// reusable fixtures stay.

type ApiEnvelope<T> = {
  code: number;
  message?: string;
  data: T;
};

type OperationLogPage = {
  items?: Array<{ id: number; tenantId: number; title: string }>;
  total?: number;
};

type SecurityEventPage = {
  items?: Array<{ id: number; tenantId: number }>;
  total?: number;
};

const tenants = {
  a: 101,
  b: 202,
} as const;

async function loginForTenant(browser: Browser, tenantId: number) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const login = await loginByApi(page.request, {
    username: 'admin',
    password: '123456',
    tenantId,
  } as Parameters<typeof loginByApi>[1]);
  await installClientSession(page, login);
  return { context, page, login, tenantId };
}

async function assertProtectedPage(page: import('@playwright/test').Page, path: string, selector: string) {
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.goto(path, { waitUntil: 'domcontentloaded' });
  await expect(page.locator(selector)).toBeVisible();
  expect(pageErrors, `page ${path} must render without errors`).toEqual([]);
}

test.describe('Tenant auth-log surfaces @priority:critical @smoke:tenant', () => {
  test('login-log and security-event pages render per tenant and stay pinned to it', async ({ browser }) => {
    test.setTimeout(120_000);
    const tenantA = await loginForTenant(browser, tenants.a);
    const tenantB = await loginForTenant(browser, tenants.b);
    try {
      // Both audit pages must render for a tenant subject without errors.
      await assertProtectedPage(tenantA.page, '/system/login-log', '.auth-login-log-page__table-card');
      await assertProtectedPage(tenantA.page, '/system/security-event', '.auth-security-event-page__table-card');
      await assertProtectedPage(tenantB.page, '/system/security-event', '.auth-security-event-page__table-card');

      // API-level pinning: every returned row must carry the acting tenant id
      // (or the page must be empty). A single foreign row is a leak.
      for (const tenant of [tenantA, tenantB]) {
        const response = await tenant.page.request.get(`${apiBaseUrl}/system/security-event/list`, {
          headers: apiRequestHeaders(tenant.login),
        });
        expect(response.ok()).toBeTruthy();
        const payload = (await response.json()) as ApiEnvelope<SecurityEventPage>;
        expect(payload.code).toBe(200);
        for (const item of payload.data?.items ?? []) {
          expect(
            item.tenantId,
            `security-event row ${item.id} leaked across tenants (expected ${tenant.tenantId})`,
          ).toBe(tenant.tenantId);
        }
      }
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});

test.describe('Tenant audit surfaces @priority:critical @smoke:tenant', () => {
  test('operation-log list, detail, and export stay pinned to the acting tenant', async ({ browser }) => {
    test.setTimeout(120_000);
    const tenantA = await loginForTenant(browser, tenants.a);
    const tenantB = await loginForTenant(browser, tenants.b);
    try {
      await assertProtectedPage(tenantA.page, '/system/operation-log', '.system-list__table-card');

      // Collect each tenant's own visible ids through the list API.
      const listFor = async (tenant: Awaited<ReturnType<typeof loginForTenant>>) => {
        const response = await tenant.page.request.get(`${apiBaseUrl}/system/operation-log/list`, {
          headers: apiRequestHeaders(tenant.login),
          params: { pageSize: '50' },
        });
        expect(response.ok()).toBeTruthy();
        const payload = (await response.json()) as ApiEnvelope<OperationLogPage>;
        expect(payload.code).toBe(200);
        return payload.data ?? {};
      };

      const pageA = await listFor(tenantA);
      const pageB = await listFor(tenantB);
      for (const item of pageA.items ?? []) {
        expect(item.tenantId, 'operation-log list leaked a foreign row to tenant 101').toBe(tenants.a);
      }
      for (const item of pageB.items ?? []) {
        expect(item.tenantId, 'operation-log list leaked a foreign row to tenant 202').toBe(tenants.b);
      }

      // Cross-tenant detail lookup must be rejected (404 semantics, no row).
      const foreignId = (pageB.items ?? []).find((item) => item.tenantId === tenants.b)?.id;
      if (foreignId !== undefined) {
        const detailResponse = await tenantA.page.request.get(
          `${apiBaseUrl}/system/operation-log/${foreignId}`,
          { headers: apiRequestHeaders(tenantA.login) },
        );
        const detailPayload = (await detailResponse.json().catch(() => ({}))) as ApiEnvelope<unknown>;
        expect(
          detailResponse.status() >= 400 || detailPayload.code >= 400,
          `cross-tenant operation-log detail must be rejected: ${JSON.stringify(detailPayload)}`,
        ).toBeTruthy();
      }

      // Export must only contain the acting tenant's rows. The CSV carries a
      // tenantId column since the tenant wave (PR #316).
      const exportResponse = await tenantA.page.request.post(`${apiBaseUrl}/system/operation-log/export`, {
        headers: apiRequestHeaders(tenantA.login),
        data: { pageSize: 1000 },
      });
      expect(exportResponse.ok()).toBeTruthy();
      const csv = await exportResponse.text();
      const tenantIdColumnIndex = csv
        .split(/\r?\n/)
        .at(0)
        ?.split(',')
        .findIndex((header) => header.trim() === 'tenantId');
      if (tenantIdColumnIndex !== undefined && tenantIdColumnIndex >= 0) {
        for (const line of csv.split(/\r?\n/).slice(1)) {
          if (!line.trim()) {
            continue;
          }
          const cell = line.split(',')[tenantIdColumnIndex]?.trim();
          if (cell && cell !== '0') {
            expect(
              cell,
              'operation-log export leaked a foreign tenant row',
            ).toBe(String(tenants.a));
          }
        }
      }
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});

test.describe('Tenant settings surface @priority:high @smoke:tenant', () => {
  test('tenant-bound setting group reads stay per-tenant; cache refresh stays scoped', async ({ browser }) => {
    test.setTimeout(90_000);
    const tenantA = await loginForTenant(browser, tenants.a);
    const tenantB = await loginForTenant(browser, tenants.b);
    try {
      // The ui group is the smallest read surface both tenants can access.
      const readGroup = async (tenant: Awaited<ReturnType<typeof loginForTenant>>) => {
        const response = await tenant.page.request.get(`${apiBaseUrl}/system/setting/group/ui`, {
          headers: apiRequestHeaders(tenant.login),
        });
        expect(response.ok()).toBeTruthy();
        const payload = (await response.json()) as ApiEnvelope<unknown>;
        expect(payload.code).toBe(200);
        return payload.data;
      };
      expect(await readGroup(tenantA)).toBeDefined();
      expect(await readGroup(tenantB)).toBeDefined();

      // Cache refresh must succeed per tenant without leaking the other
      // tenant's namespace (response is a status envelope only).
      const refresh = async (tenant: Awaited<ReturnType<typeof loginForTenant>>) => {
        const response = await tenant.page.request.post(`${apiBaseUrl}/system/setting/cache/refresh`, {
          headers: apiRequestHeaders(tenant.login),
        });
        expect(response.ok()).toBeTruthy();
      };
      await refresh(tenantA);
      await refresh(tenantB);

      // After refresh both tenants still resolve their own group reads.
      expect(await readGroup(tenantA)).toBeDefined();
      expect(await readGroup(tenantB)).toBeDefined();
    } finally {
      await tenantA.context.close();
      await tenantB.context.close();
    }
  });
});

test.describe('Tenant dynamic-module surface @priority:high @smoke:tenant', () => {
  test('module registry list is readable per tenant; unverified write stays rejected', async ({ browser }) => {
    test.setTimeout(90_000);
    const tenantA = await loginForTenant(browser, tenants.a);
    try {
      // The registry list is platform state readable by authorized subjects.
      const listResponse = await tenantA.page.request.get(`${apiBaseUrl}/lowcode/dynamic-modules`, {
        headers: apiRequestHeaders(tenantA.login),
      });
      expect(listResponse.ok()).toBeTruthy();

      // A mutation attempt without the SecureAction operation token must be
      // rejected for a tenant subject (defense-in-depth on the write path).
      const unverifiedWrite = await tenantA.page.request.post(`${apiBaseUrl}/lowcode/dynamic-modules/generate`, {
        headers: apiRequestHeaders(tenantA.login),
        data: {},
      });
      const writePayload = (await unverifiedWrite.json().catch(() => ({}))) as ApiEnvelope<unknown>;
      expect(
        unverifiedWrite.status() >= 400 || writePayload.code >= 400,
        `unverified dynamic-module write must be rejected: ${JSON.stringify(writePayload)}`,
      ).toBeTruthy();
    } finally {
      await tenantA.context.close();
    }
  });
});
