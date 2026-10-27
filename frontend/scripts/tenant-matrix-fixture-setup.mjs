// Provision the tenant matrix that tests/smoke-core/tenant-hostile-*.spec.ts and
// tenant-protected-resources.spec.ts run against.
//
// Phase: `up` provisions, `down` tears down. A manifest is written to
// ../.tmp/smoke-tenant-matrix.json so the specs (and an interrupted run) can
// find the tenants without guessing.
//
// Why this no longer logs the platform admin in with an explicit tenantId
// ---------------------------------------------------------------------
// This script used to reuse tenants 101/202 and authenticate as the platform
// `admin` with `tenantId` in the login body. Under the tenant-isolation model
// (task 2026-10-13-tenant-true-isolation) that is no longer a valid subject:
// a session carrying a tenant claim resolves its role set INSIDE that tenant,
// so a platform operator who owns no tenant-scoped role gets an EMPTY role set
// and every console call is refused with 403 permission.denied. Membership is
// not authority (contract §4: the global and tenant namespaces never merge).
//
// The supported shape - and the one the application itself produces - is a
// tenant provisioned through POST /system/tenant: it owns its tenant_admin
// role, its Casbin domain policies and its own administrator user. The matrix
// now provisions exactly that, and each spec signs in as its tenant's own
// administrator. That is also what the platform "tenant viewpoint" browser
// spec does, which is why it has been green all along.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const base = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8080/api/v1';
const phase = process.argv[2] ?? 'up';
const run = Date.now();
// Repo-level .tmp (gitignored). This file lives in frontend/scripts, so it
// takes TWO hops to reach the repository root — one hop lands in frontend/.tmp,
// which is not where the specs look.
const manifestUrl = new URL('../../.tmp/smoke-tenant-matrix.json', import.meta.url);

// `smoke_tmatrix_` is registered in scripts/cleanup-smoke-fixtures.mjs so an
// interrupted run is swept by `npm run smoke:cleanup`.
const adminUsername = (key) => `smoke_tmatrix_${key}_${run}`;
const adminPassword = 'SmokeMatrix#2026';

const tenantSpecs = [
  { key: 'a', code: `smoke-matrix-a-${run}`, name: `Smoke Matrix Acme ${run}` },
  { key: 'b', code: `smoke-matrix-b-${run}`, name: `Smoke Matrix Globex ${run}` },
];

function readManifest() {
  try {
    return JSON.parse(readFileSync(manifestUrl, 'utf8'));
  } catch {
    return { tenants: [] };
  }
}

function writeManifest(payload) {
  mkdirSync(dirname(fileURLToPath(manifestUrl)), { recursive: true });
  writeFileSync(manifestUrl, `${JSON.stringify(payload, null, 2)}\n`);
}

async function sessionFromResponse(res) {
  const body = await res.json();
  if (body.code !== 200) throw new Error(`login: ${body.code} ${body.message ?? ''}`.trim());
  const cookies = (res.headers.getSetCookie?.() ?? [])
    .map((c) => c.split(';')[0])
    .filter((c) => c.startsWith('pantheon_'));
  const accessToken = cookies.find((c) => c.startsWith('pantheon_access_token='))?.split('=')[1];
  if (!accessToken) throw new Error(`login: no access cookie (${JSON.stringify(body).slice(0, 200)})`);
  const csrfToken = cookies.find((c) => c.startsWith('pantheon_csrf_token='))?.split('=')[1] ?? '';
  return { accessToken, cookie: cookies.join('; '), csrfToken, body };
}

async function loginPlatformAdmin() {
  const res = await fetch(`${base}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
    body: JSON.stringify({
      username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
      password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
    }),
  });
  const session = await sessionFromResponse(res);
  // The platform admin may carry matrix memberships from a previous run, which
  // would turn this into the picker. Re-running `down` first clears them.
  if (session.body.data?.tenantSelectionRequired) {
    throw new Error(
      'platform admin still has active tenant memberships - run this script with `down` first',
    );
  }
  return session;
}

async function loginTenantAdmin(tenant) {
  const res = await fetch(`${base}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
    body: JSON.stringify({
      username: tenant.adminUsername,
      password: tenant.adminPassword,
      tenantId: tenant.id,
    }),
  });
  const session = await sessionFromResponse(res);
  if (session.body.data?.user?.id === undefined && !session.body.data?.sessionId) {
    throw new Error(`tenant admin login t${tenant.id}: ${JSON.stringify(session.body).slice(0, 200)}`);
  }
  return session;
}

async function getOperationToken(
  session,
  password = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
) {
  const res = await fetch(`${base}/auth/operation-verify`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': session.csrfToken,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ password }),
  });
  const body = await res.json();
  if (body.code !== 200 || !body.data?.operationToken) {
    throw new Error(`operation verify: ${body.code} ${body.message ?? ''}`.trim());
  }
  return body.data.operationToken;
}

async function provisionTenant(session, spec) {
  const response = await fetch(`${base}/system/tenant`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': session.csrfToken,
      'X-Operation-Token': await getOperationToken(session),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      code: spec.code,
      name: spec.name,
      plan: 'free',
      // snake_case on the wire - CreateTenantDTO binds snake_case and Gin
      // silently ignores camelCase, which would produce a tenant with no admin.
      admin_username: adminUsername(spec.key),
      admin_password: adminPassword,
      admin_nickname: `Smoke Matrix Admin ${spec.key} ${run}`,
    }),
  });
  const body = await response.json();
  if (body.code !== 200 || !body.data?.id) {
    throw new Error(`provision ${spec.code}: ${body.code} ${body.message ?? ''}`.trim());
  }
  return {
    key: spec.key,
    id: body.data.id,
    code: spec.code,
    name: spec.name,
    dictCode: `matrix_browser_${spec.key}`,
    adminUsername: adminUsername(spec.key),
    adminPassword,
  };
}

async function listDictTypes(session, dictCode) {
  const res = await fetch(`${base}/system/dict/type/list?dictCode=${encodeURIComponent(dictCode)}`, {
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
    },
  });
  const body = await res.json();
  if (body.code !== 200) throw new Error(`list ${dictCode}: ${body.code} ${body.message ?? ''}`);
  return Array.isArray(body.data) ? body.data : [];
}

async function createDictType(session, dictCode, name) {
  const res = await fetch(`${base}/system/dict/type`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': session.csrfToken,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      dictCode,
      dictName: name,
      module: 'system',
      status: 1,
      remark: 'tenant-matrix fixture',
    }),
  });
  const body = await res.json();
  if (body.code !== 200) throw new Error(`create ${dictCode}: ${body.code} ${body.message ?? ''}`);
}

async function deleteDictType(session, operationToken, typeId) {
  const res = await fetch(`${base}/system/dict/type/${typeId}`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': session.csrfToken,
      'X-Operation-Token': operationToken,
    },
  });
  const body = await res.json().catch(() => ({}));
  if (![200, 404].includes(res.status) || (body.code !== undefined && body.code !== 200 && body.code !== 404)) {
    throw new Error(`delete type ${typeId}: HTTP ${res.status} ${body.message ?? ''}`.trim());
  }
}

async function listMembers(platform, tenantId) {
  const res = await fetch(`${base}/system/tenant/${tenantId}/members?page=1&pageSize=100`, {
    headers: {
      Authorization: `Bearer ${platform.accessToken}`,
      Cookie: platform.cookie,
      'X-Requested-With': 'XMLHttpRequest',
    },
  });
  const body = await res.json();
  if (body.code !== 200) return [];
  return Array.isArray(body.data?.items) ? body.data.items : [];
}

async function deleteMember(platform, tenantId, userId, operationToken) {
  const res = await fetch(`${base}/system/tenant/${tenantId}/members/${userId}`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${platform.accessToken}`,
      Cookie: platform.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': platform.csrfToken,
      'X-Operation-Token': operationToken,
    },
  });
  const body = await res.json().catch(() => ({}));
  if (![200, 404].includes(res.status)) {
    throw new Error(`remove member ${userId} of t${tenantId}: HTTP ${res.status} ${body.message ?? ''}`.trim());
  }
}

async function archiveTenant(platform, tenantId, operationToken) {
  const res = await fetch(`${base}/system/tenant/${tenantId}`, {
    method: 'PUT',
    headers: {
      Authorization: `Bearer ${platform.accessToken}`,
      Cookie: platform.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': platform.csrfToken,
      'X-Operation-Token': operationToken,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ status: 'archived' }),
  });
  const body = await res.json().catch(() => ({}));
  if (![200, 404].includes(res.status)) {
    throw new Error(`archive tenant ${tenantId}: HTTP ${res.status} ${body.message ?? ''}`.trim());
  }
}

async function deleteTenant(platform, tenantId, operationToken) {
  const res = await fetch(`${base}/system/tenant/${tenantId}`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${platform.accessToken}`,
      Cookie: platform.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': platform.csrfToken,
      'X-Operation-Token': operationToken,
    },
  });
  const body = await res.json().catch(() => ({}));
  if (![200, 404].includes(res.status)) {
    throw new Error(`delete tenant ${tenantId}: HTTP ${res.status} ${body.message ?? ''}`.trim());
  }
}

async function provision() {
  const platform = await loginPlatformAdmin();
  const tenants = [];
  for (const spec of tenantSpecs) {
    const tenant = await provisionTenant(platform, spec);
    // Seed the dictionary fixture through the TENANT ADMIN session. A tenant
    // administrator owns the tenant-scoped role and its Casbin domain policy, so
    // this call is authorised - the platform admin naming a tenant would not be.
    const session = await loginTenantAdmin(tenant);
    const existing = await listDictTypes(session, tenant.dictCode);
    if (existing.length === 0) {
      await createDictType(session, tenant.dictCode, tenant.name);
    }
    const after = await listDictTypes(session, tenant.dictCode);
    if (after.length !== 1) {
      throw new Error(`t${tenant.id}: expected exactly 1 ${tenant.dictCode}, got ${after.length}`);
    }
    tenants.push(tenant);
    console.log(`t${tenant.id}: ${tenant.code} provisioned (dict ${tenant.dictCode} ready)`);
  }
  writeManifest({ run, tenants });
  console.log(`tenant-matrix tenants ready: ${tenants.map((t) => t.id).join(', ')}`);
}

async function cleanup() {
  const manifest = readManifest();
  if (manifest.tenants.length === 0) {
    console.log('tenant-matrix fixtures: nothing recorded, nothing to remove');
    return;
  }
  const platform = await loginPlatformAdmin();
  for (const tenant of manifest.tenants) {
    // Dict fixtures first, while the tenant admin can still reach the tenant.
    try {
      const session = await loginTenantAdmin(tenant);
      const existing = await listDictTypes(session, tenant.dictCode);
      if (existing.length > 0) {
        const operationToken = await getOperationToken(session, tenant.adminPassword);
        for (const item of existing) await deleteDictType(session, operationToken, item.id);
      }
    } catch (error) {
      console.warn(`t${tenant.id}: dict cleanup skipped (${error.message})`);
    }
    // §2.2 refuses to remove the last owner, so a tenant cannot reach "no active
    // members" by purging memberships alone. §2.1 makes archived the retirement
    // path, and deleting an archived tenant disables the memberships it still
    // has. Purge first anyway, so the fixture does not depend on the archived
    // path for tenants it can clean up directly.
    const operationToken = await getOperationToken(platform);
    for (const member of await listMembers(platform, tenant.id)) {
      await deleteMember(platform, tenant.id, member.user_id, operationToken);
    }
    await archiveTenant(platform, tenant.id, operationToken);
    await deleteTenant(platform, tenant.id, operationToken);
    console.log(`t${tenant.id}: removed (${tenant.code})`);
  }
  writeManifest({ run: 0, tenants: [] });
  console.log('tenant-matrix fixtures removed');
  console.log(
    'note: DELETE /system/tenant is a SOFT delete and the tenant admin system_user row survives it; ' +
      '`npm run smoke:cleanup` sweeps the smoke_tmatrix_ users.',
  );
}

if (phase === 'down') {
  await cleanup();
  process.exit(0);
}
if (phase !== 'up') {
  throw new Error(`usage: node tenant-matrix-fixture-setup.mjs [up|down], got ${phase}`);
}

await provision();
