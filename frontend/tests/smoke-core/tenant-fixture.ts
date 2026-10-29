import { randomBytes } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

import { expect, type APIRequestContext, type Browser } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  getApiOperationToken,
  installClientSession,
  loginByApi,
  type BrowserLoginResult,
} from '../smoke/helpers/auth';

/**
 * Tenant matrix fixture for the hostile / protected-resource smoke specs.
 *
 * ## Why the specs sign in as a *tenant* administrator
 *
 * These specs used to authenticate as the platform `admin` with an explicit
 * `tenantId` in the login body, against two fixed tenants (101/202). Under the
 * tenant-isolation model that is not a usable subject: a session carrying a
 * tenant claim resolves its role set *inside* that tenant, and a platform
 * operator owns no tenant-scoped role there, so it ends up with an empty role
 * set and every console call returns 403 permission.denied. Membership is not
 * authority — contract §4 keeps the global and tenant namespaces separate.
 *
 * The shape the application itself produces — and the one these specs now use —
 * is a tenant provisioned through `POST /system/tenant`: it owns its
 * `tenant_admin` role, its Casbin domain policies and its own administrator
 * user. The platform "tenant viewpoint" browser spec already worked this way,
 * which is why it has been green throughout.
 *
 * ## Where the tenants come from
 *
 * `scripts/tenant-matrix-fixture-setup.mjs up` (passed as `--setup` by
 * `npm run test:smoke:tenant`) writes `../.tmp/smoke-tenant-matrix.json`. The
 * specs prefer that manifest so a run that provisions once stays coherent. When
 * the manifest is missing — an individual spec run without `--setup` — the
 * helper provisions its own pair and the caller tears them down in `afterAll`.
 */
export type MatrixTenant = {
  id: number;
  code: string;
  name: string;
  dictCode: string;
  adminUsername: string;
  adminPassword: string;
  /** True when this helper created the tenant for this run and owns its teardown. */
  owned: boolean;
};

type ApiEnvelope<T> = { code: number; message?: string; data: T };

const manifestPath = join(process.cwd(), '..', '.tmp', 'smoke-tenant-matrix.json');
const run = `${Date.now()}`;
// Ephemeral smoke-tenant admin password. It is generated per run rather than
// checked in, and the value the run used is recorded in the fixture manifest, so
// the specs and the teardown always agree on it. Set
// PANTHEON_SMOKE_MATRIX_ADMIN_PASSWORD to pin a specific value instead.
function generateEphemeralPassword() {
  // 3 + 12 + 4 characters: comfortably above the deployment's minimum length
  // and mixed enough for any composition rule the deployment configures.
  return `Smk${randomBytes(6).toString('hex')}#Aa1`;
}

const adminPassword =
  process.env.PANTHEON_SMOKE_MATRIX_ADMIN_PASSWORD ?? generateEphemeralPassword();

const RUN_TENANT_SPEC = [
  { key: 'a', dictCode: 'matrix_browser_a' },
  { key: 'b', dictCode: 'matrix_browser_b' },
] as const;

function readManifestTenants(): MatrixTenant[] {
  if (!existsSync(manifestPath)) return [];
  try {
    const payload = JSON.parse(readFileSync(manifestPath, 'utf8')) as {
      tenants?: Array<Partial<MatrixTenant>>;
    };
    return (payload.tenants ?? []).flatMap((tenant) =>
      tenant.id && tenant.adminUsername && tenant.adminPassword
        ? [{ ...tenant, owned: false } as MatrixTenant]
        : [],
    );
  } catch {
    return [];
  }
}

async function platformSession(request: APIRequestContext) {
  const login = await loginByApi(request, {
    username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
    password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
  });
  const operationToken = await getApiOperationToken(request, login);
  return { login, headers: { ...apiRequestHeaders(login), 'X-Operation-Token': operationToken } };
}

async function provisionOne(
  request: APIRequestContext,
  session: Awaited<ReturnType<typeof platformSession>>,
  spec: { key: string; dictCode: string },
): Promise<MatrixTenant> {
  const code = `smoke-matrix-${spec.key}-${run}`;
  const name = `Smoke Matrix ${spec.key.toUpperCase()} ${run}`;
  const adminUsername = `smoke_tmatrix_${spec.key}_${run}`;
  const created = await request.post(`${apiBaseUrl}/system/tenant`, {
    headers: session.headers,
    data: {
      code,
      name,
      plan: 'free',
      // snake_case on the wire: CreateTenantDTO binds snake_case and Gin
      // silently ignores camelCase, which yields a tenant with no administrator.
      admin_username: adminUsername,
      admin_password: adminPassword,
      admin_nickname: `Smoke Matrix Admin ${spec.key} ${run}`,
    },
  });
  const payload = (await created.json()) as ApiEnvelope<{ id: number }>;
  expect(payload.code, `provision ${code}: ${JSON.stringify(payload)}`).toBe(200);
  expect(payload.data.id).toBeGreaterThan(0);
  return {
    id: payload.data.id,
    code,
    name,
    dictCode: spec.dictCode,
    adminUsername,
    adminPassword,
    owned: true,
  };
}

/**
 * Resolves the two-tenant matrix, provisioning one when no fixture manifest was
 * passed by the runner. Call it from `test.beforeAll` and always pair it with
 * `teardownMatrixTenants` in `test.afterAll`.
 *
 * The dictionary fixture each tenant owns is seeded through that tenant's own
 * administrator, which is the only subject authorised to write inside the
 * tenant's Casbin domain.
 */
let matrixPromise: Promise<MatrixTenant[]> | undefined;

export function setupMatrixTenants(request: APIRequestContext): Promise<MatrixTenant[]> {
  // Memoised per worker process. Several spec files call this from their own
  // `beforeAll`; without the memo each one would provision its own pair, and
  // because the tenant code embeds a shared run id the second attempt would
  // fail with tenant.conflict instead of reusing the first one.
  matrixPromise ??= (async () => {
    const fromManifest = readManifestTenants();
    if (fromManifest.length >= 2) {
      return fromManifest.slice(0, 2);
    }
    const session = await platformSession(request);
    const tenants = await Promise.all(
      RUN_TENANT_SPEC.map((spec) => provisionOne(request, session, spec)),
    );
    // Each tenant owns a distinct dictionary fixture, so the seeds are
    // independent and run concurrently instead of one round-trip at a time.
    await Promise.all(tenants.map((tenant) => ensureDictFixture(tenant)));
    return tenants;
  })();
  return matrixPromise;
}

/**
 * Creates the tenant's dictionary fixture when it is not there yet.
 *
 * Plain fetch, not `loginByApi`: this runs inside `beforeAll`, where a browser
 * context does not exist, and the session is only used for two calls that a
 * tenant administrator is allowed to make.
 */
async function ensureDictFixture(tenant: MatrixTenant): Promise<void> {
  const response = await fetch(`${apiBaseUrl}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
    body: JSON.stringify({
      username: tenant.adminUsername,
      password: tenant.adminPassword,
      tenantId: tenant.id,
    }),
  });
  const payload = (await response.json()) as ApiEnvelope<{ sessionId?: string }>;
  expect(payload.code, `tenant admin login t${tenant.id}: ${JSON.stringify(payload)}`).toBe(200);
  const cookies = (response.headers.getSetCookie?.() ?? []).map((cookie) => cookie.split(';')[0]);
  const cookie = cookies.join('; ');
  const accessToken = cookies
    .find((entry) => entry.startsWith('pantheon_access_token='))
    ?.split('=')[1];
  expect(accessToken, `t${tenant.id}: tenant admin login returned no access cookie`).toBeTruthy();
  const csrfToken =
    cookies.find((entry) => entry.startsWith('pantheon_csrf_token='))?.split('=')[1] ?? '';
  const auth = {
    Authorization: `Bearer ${accessToken}`,
    Cookie: cookie,
    'X-Requested-With': 'XMLHttpRequest',
    'X-CSRF-Token': csrfToken,
  };

  const listed = (await fetch(
    `${apiBaseUrl}/system/dict/type/list?dictCode=${encodeURIComponent(tenant.dictCode)}`,
    { headers: auth },
  ).then((res) => res.json())) as ApiEnvelope<Array<unknown>>;
  if ((listed.data ?? []).length > 0) return;

  const created = (await fetch(`${apiBaseUrl}/system/dict/type`, {
    method: 'POST',
    headers: { ...auth, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      dictCode: tenant.dictCode,
      dictName: tenant.name,
      module: 'system',
      status: 1,
      remark: 'tenant-matrix fixture',
    }),
  }).then((res) => res.json())) as ApiEnvelope<unknown>;
  expect(created.code, `seed dict ${tenant.dictCode}: ${JSON.stringify(created)}`).toBe(200);
}

/** Signs a browser context in as the tenant's own administrator. */
export async function loginAsTenantAdmin(browser: Browser, tenant: MatrixTenant) {
  const context = await browser.newContext();
  const page = await context.newPage();
  const login = await loginByApi(page.request, {
    username: tenant.adminUsername,
    password: tenant.adminPassword,
    // The owning tenant is also the only usable one (contract §2.2 V1.7);
    // naming it keeps the request deterministic without relying on discovery.
    tenantId: tenant.id,
  } as Parameters<typeof loginByApi>[1]);
  await installClientSession(page, login);
  return { context, page, login, tenant, tenantId: tenant.id };
}

export type TenantSession = Awaited<ReturnType<typeof loginAsTenantAdmin>>;

/**
 * Removes the tenants this run owns.
 *
 * `DELETE /system/tenant/:id` refuses anything that is not in the archived
 * terminal state (W1, contract §2.1: `tenant.not_archived`), so each owned
 * tenant is archived first; the delete transaction disables any remaining
 * memberships itself. The old purge-memberships-then-delete order is both
 * unnecessary and broken now — removing the provisioning owner is refused as
 * the last-owner removal. Manifest tenants are left in place: the runner's
 * `down` step owns them.
 */
export async function teardownMatrixTenants(
  request: APIRequestContext,
  tenants: MatrixTenant[],
  login: BrowserLoginResult,
): Promise<void> {
  const owned = tenants.filter((tenant) => tenant.owned);
  if (owned.length === 0) return;
  const operationToken = await getApiOperationToken(request, login);
  const headers = { ...apiRequestHeaders(login), 'X-Operation-Token': operationToken };
  // Tenants are independent, so they are torn down concurrently.
  await Promise.all(owned.map((tenant) => teardownOneTenant(request, tenant, headers)));
}

/** Archives one owned tenant, then deletes the row. */
async function teardownOneTenant(
  request: APIRequestContext,
  tenant: MatrixTenant,
  headers: Record<string, string>,
): Promise<void> {
  const archived = await request.put(`${apiBaseUrl}/system/tenant/${tenant.id}`, {
    headers,
    data: { name: tenant.name, status: 'archived' },
  });
  const archivedPayload = (await archived.json()) as ApiEnvelope<unknown>;
  expect(
    archivedPayload.code,
    `archive tenant ${tenant.id}: ${JSON.stringify(archivedPayload)}`,
  ).toBe(200);
  const deleted = await request.delete(`${apiBaseUrl}/system/tenant/${tenant.id}`, { headers });
  const deletedPayload = (await deleted.json()) as ApiEnvelope<unknown>;
  expect(deletedPayload.code, `teardown tenant ${tenant.id}: ${JSON.stringify(deletedPayload)}`).toBe(
    200,
  );
}
