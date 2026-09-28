// Re-provision the pre-provisioned dict fixtures consumed by
// tests/smoke-core/tenant-hostile-browser-matrix.spec.ts (tenants 101/202).
// These were hand-provisioned during the 2026-09-15 matrix run. The workflow
// owns both provisioning and cleanup through the admin API so CI runs do not
// depend on database-level cascade behavior.
const base = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8081/api/v1';
const phase = process.argv[2] ?? 'up';

const fixtures = [
  { tenantId: 101, dictCode: 'matrix_browser_a', name: 'Smoke Acme North' },
  { tenantId: 202, dictCode: 'matrix_browser_b', name: 'Smoke Globex South' },
];

async function login(tenantId) {
  const res = await fetch(`${base}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
    body: JSON.stringify({ username: 'admin', password: '123456', tenantId }),
  });
  const body = await res.json();
  if (body.code !== 200 || !body.data?.sessionId) throw new Error(`login t${tenantId}: ${body.code}`);
  const cookies = (res.headers.getSetCookie?.() ?? [])
    .map((c) => c.split(';')[0])
    .filter((c) => c.startsWith('pantheon_'));
  const accessToken = cookies.find((c) => c.startsWith('pantheon_access_token='))?.split('=')[1];
  const csrfToken = cookies.find((c) => c.startsWith('pantheon_csrf_token='))?.split('=')[1] ?? '';
  if (!accessToken) throw new Error(`login t${tenantId}: no access cookie`);
  return { accessToken, cookie: cookies.join('; '), csrfToken };
}

async function listTypes(session, dictCode) {
  const res = await fetch(`${base}/system/dict/type/list?dictCode=${encodeURIComponent(dictCode)}`, {
    headers: { Authorization: `Bearer ${session.accessToken}`, Cookie: session.cookie, 'X-Requested-With': 'XMLHttpRequest' },
  });
  const body = await res.json();
  if (body.code !== 200) throw new Error(`list ${dictCode}: ${body.code} ${body.message}`);
  return Array.isArray(body.data) ? body.data : [];
}

async function createType(session, tenantId, dictCode, name) {
  const res = await fetch(`${base}/system/dict/type`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': session.csrfToken,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ dictCode, dictName: name, module: 'system', status: 1, remark: `tenant-matrix fixture (t${tenantId})` }),
  });
  const body = await res.json();
  if (body.code !== 200) throw new Error(`create ${dictCode}: ${body.code} ${body.message}`);
}

async function getOperationToken(session) {
  const res = await fetch(`${base}/auth/operation-verify`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${session.accessToken}`,
      Cookie: session.cookie,
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': session.csrfToken,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456' }),
  });
  const body = await res.json();
  if (body.code !== 200 || !body.data?.operationToken) {
    throw new Error(`operation verify: ${body.code} ${body.message ?? ''}`.trim());
  }
  return body.data.operationToken;
}

async function deleteType(session, operationToken, typeId) {
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

async function provision() {
  for (const fixture of fixtures) {
    const session = await login(fixture.tenantId);
    const existing = await listTypes(session, fixture.dictCode);
    if (existing.length === 0) {
      await createType(session, fixture.tenantId, fixture.dictCode, fixture.name);
      console.log(`t${fixture.tenantId}: created ${fixture.dictCode}`);
    } else {
      console.log(`t${fixture.tenantId}: ${fixture.dictCode} already present (${existing.length})`);
    }
    const after = await listTypes(session, fixture.dictCode);
    if (after.length !== 1) throw new Error(`t${fixture.tenantId}: expected exactly 1 ${fixture.dictCode}, got ${after.length}`);
  }
  console.log('tenant-matrix dict fixtures ready');
}

async function cleanup() {
  for (const fixture of fixtures) {
    const session = await login(fixture.tenantId);
    const existing = await listTypes(session, fixture.dictCode);
    if (existing.length === 0) {
      console.log(`t${fixture.tenantId}: ${fixture.dictCode} already absent`);
      continue;
    }
    const operationToken = await getOperationToken(session);
    for (const item of existing) {
      await deleteType(session, operationToken, item.id);
    }
    console.log(`t${fixture.tenantId}: removed ${fixture.dictCode} (${existing.length})`);
  }
  console.log('tenant-matrix dict fixtures removed');
}

if (phase === 'down') {
  await cleanup();
  process.exit(0);
}
if (phase !== 'up') {
  throw new Error(`usage: node tenant-matrix-fixture-setup.mjs [up|down], got ${phase}`);
}

await provision();
