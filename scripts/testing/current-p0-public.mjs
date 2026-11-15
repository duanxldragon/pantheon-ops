import { createHash, randomBytes } from 'node:crypto';
import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, extname, isAbsolute, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { readAuthCookieSession } from '../../frontend/scripts/lib/auth-cookie-session.mjs';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const evidence = resolve(root, '.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/verification/current-p0');
const check = (value, label) => { if (!value) throw new Error(label); };
const hash = (path) => createHash('sha256').update(readFileSync(path)).digest('hex');
const inside = (path, parent) => {
  const suffix = relative(parent, resolve(path));
  return suffix !== '' && !suffix.startsWith(`..${sep}`) && suffix !== '..' && !isAbsolute(suffix);
};

function sources(directory = resolve(root, 'backend')) {
  const result = {};
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) Object.assign(result, sources(path));
    else if (entry.isFile() && ['.go', '.sql', '.mod', '.sum'].includes(extname(path))) result[relative(root, path).replaceAll('\\', '/')] = hash(path);
  }
  return result;
}

function preflight(configPath) {
  check(configPath && inside(configPath, resolve(root, '.tmp')), 'Owned private config under Ops .tmp required');
  const config = JSON.parse(readFileSync(configPath, 'utf8'));
  const privateRoot = dirname(resolve(configPath));
  check(/^current_p0_\d{8}T\d{6}_[a-f0-9]{8}$/.test(config.runId), 'Fresh runtime run ID required');
  check(!existsSync(resolve(privateRoot, 'public-attempt.json')), 'One-use runtime attempt occupied');
  const endpoint = new URL(config.apiBaseUrl);
  check(endpoint.protocol === 'http:' && endpoint.hostname === '127.0.0.1' && endpoint.pathname === '/api/v1' && endpoint.port && !endpoint.search && !endpoint.hash && !endpoint.username && !endpoint.password, 'Exclusive local public API required');
  check(config.providerUrl === 'http://115.190.251.143:9090', 'Read-only provider endpoint outside current scope');
  check(/^ops_baseline_test_p0_[a-f0-9]{12}$/.test(config.schema), 'Exclusive synthetic schema required');
  check(inside(config.executable, resolve(root, '.tmp')) && inside(config.sourceManifest, resolve(root, '.tmp')), 'Build pins outside task scratch');
  check(hash(config.executable) === config.executableSHA256 && hash(config.sourceManifest) === config.sourceManifestSHA256, 'Build pins changed');
  const manifest = JSON.parse(readFileSync(config.sourceManifest, 'utf8'));
  const current = sources();
  check(Object.keys(manifest).length === Object.keys(current).length && Object.entries(manifest).every(([path, digest]) => current[path] === digest), 'Current source differs from built source');
  check(inside(config.runtimeReceipt, evidence), 'Runtime receipt outside owned evidence');
  const receipt = JSON.parse(readFileSync(config.runtimeReceipt, 'utf8'));
  check(receipt.status === 'ready' && receipt.sourceStable === true && receipt.apiPID === config.apiPID && receipt.redisPID === config.redisPID && receipt.schema === config.schema && receipt.apiBaseUrl === config.apiBaseUrl && receipt.executableSHA256 === config.executableSHA256, 'Runtime ownership/readiness receipt mismatch');
  check(config.platformAdmin?.username && config.platformAdmin.password, 'Private platform-admin credentials required');
  return { config, privateRoot, manifest };
}

async function run(configPath) {
  const { config, privateRoot, manifest } = preflight(configPath);
  const result = { runId: config.runId, coverage: 'current-source-public-HTTP', status: 'running', executableSHA256: config.executableSHA256,
    sourceManifestSHA256: config.sourceManifestSHA256, apiPID: config.apiPID, schema: config.schema,
    entrySHA256: hash(fileURLToPath(import.meta.url)), cookieParserSHA256: hash(resolve(root, 'frontend/scripts/lib/auth-cookie-session.mjs')),
    assertions: [], requests: [], owned: { tenants: [], roles: [], users: [], sources: [], views: [], histories: [] },
    gaps: ['This matrix covers O4, Base auth/IAM/tenant/audit composition and CMDB key isolation; it does not claim O2/O3/D1/D2/D3 completion.', 'Rendered browser states and synthetic Loki/Tempo/VictoriaMetrics provider acceptance are separate gates.'], credentialsPrinted: false, remoteMutations: 0 };
  const suffix = config.runId;
  const password = `P0${randomBytes(16).toString('hex')}!Aa1`;
  const sourceCanaries = [`P0Secret${randomBytes(16).toString('hex')}`, `P0Bearer${randomBytes(16).toString('hex')}`];
  const canaries = [password, config.platformAdmin.password, ...sourceCanaries];
  const cleanup = [];
  let sequence = 0, unknown = false, platform, tenantA, tenantB, owner;
  const assert = (value, label) => { check(value, label); result.assertions.push(label); };
  const safe = (text, label) => assert(!canaries.some((value) => text.includes(value)), `${label}: private canaries absent`);
  async function api(session, method, path, data, operation = false) {
    let operationToken;
    if (operation) operationToken = success(await api(session, 'POST', '/auth/operation-verify', { password: session.password })).operationToken;
    const requestSequence = ++sequence;
    const headers = { 'X-Requested-With': 'XMLHttpRequest', 'X-Request-ID': `${suffix}_${requestSequence}` };
    if (session) Object.assign(headers, { Authorization: `Bearer ${session.accessToken}`, 'X-CSRF-Token': session.csrfToken, Cookie: `pantheon_csrf_token=${session.csrfToken}` });
    if (operationToken) headers['X-Operation-Token'] = operationToken;
    if (data !== undefined && !(data instanceof FormData)) headers['Content-Type'] = 'application/json';
    const start = Date.now();
    let response, text;
    try {
      response = await fetch(`${config.apiBaseUrl}${path}`, { method, headers, body: data === undefined ? undefined : data instanceof FormData ? data : JSON.stringify(data), redirect: 'error', signal: AbortSignal.timeout(40000) });
      const reader = response.body.getReader();
      const chunks = []; let size = 0;
      for (;;) { const chunk = await reader.read(); if (chunk.done) break; size += chunk.value.byteLength; if (size > (2 << 20)) { await reader.cancel(); throw new Error('Body budget'); } chunks.push(chunk.value); }
      text = Buffer.concat(chunks).toString('utf8');
    } catch { unknown = true; throw new Error(`Transport Unknown at request ${requestSequence}; no retry`); }
    const requestId = response.headers.get('x-request-id');
    result.requests.push({ sequence: requestSequence, method, path: path.split('?')[0], status: response.status, elapsedMs: Date.now() - start, requestId });
    if (path.startsWith('/business/') || path.startsWith('/system/operation-log')) safe(text, `HTTP ${requestSequence}`);
    let payload;
    try { payload = JSON.parse(text); } catch { /* CSV responses intentionally have no JSON envelope. */ }
    return { response, payload, text, requestId, sequence: requestSequence };
  }
  function success(reply) { check(reply.response.ok && reply.payload?.code === 200, `Request ${reply.sequence}: expected native successful envelope`); return reply.payload.data; }
  function denied(reply, label, code = 403) { assert(reply.response.status === code, label); }
  async function login(credentials, tenantId) {
    const reply = await api(null, 'POST', '/auth/login', { ...credentials, ...(tenantId ? { tenantId } : {}) });
    success(reply);
    const session = { ...readAuthCookieSession(reply.response.headers, { csrfFallback: '' }), password: credentials.password, tenantId };
    check(session.csrfToken, 'Native login CSRF cookie/header missing');
    canaries.push(session.accessToken, session.refreshToken);
    session.user = success(await api(session, 'GET', '/auth/me'));
    return session;
  }
  async function provisionTenant(label) {
    const name = `${suffix}_${label}`;
    const row = success(await api(platform, 'POST', '/system/tenant', { code: name, name, plan: 'free', admin_username: name, admin_password: password, admin_nickname: name }, true));
    check(row.id > 0, 'Tenant ID missing'); result.owned.tenants.push(row.id);
    cleanup.push(async () => { success(await api(platform, 'PUT', `/system/tenant/${row.id}`, { name, status: 'archived' }, true)); success(await api(platform, 'DELETE', `/system/tenant/${row.id}`, undefined, true)); });
    return login({ username: name, password }, row.id);
  }
  async function provisionActor(label, keys, dataScope = 'all') {
    const name = `${suffix}_${label}`;
    const roleBody = { roleName: name, roleKey: name, status: 1, sort: 99, menuIds: [], permissionKeys: keys, dataScope };
    const role = success(await api(tenantA, 'POST', '/system/role', roleBody));
    check(role.id > 0, 'Role ID missing'); result.owned.roles.push(role.id);
    cleanup.push(async () => { success(await api(tenantA, 'DELETE', `/system/role/${role.id}`, undefined, true)); });
    const user = success(await api(tenantA, 'POST', '/system/user', { username: name, password, nickname: name, status: 1, roleIds: [role.id] }));
    check(user.id > 0, 'User ID missing'); result.owned.users.push(user.id);
    cleanup.push(async () => { success(await api(tenantA, 'DELETE', `/system/user/${user.id}`, undefined, true)); });
    const session = await login({ username: name, password }, tenantA.tenantId);
    assert(session.user.id === user.id && !session.user.roles?.some((key) => ['admin', 'tenant_admin'].includes(key)), `${label}: actual narrow actor`);
    assert(keys.every((key) => session.user.perms?.includes(key)), `${label}: current effective permission keys`);
    const list = success(await api(tenantA, 'GET', `/system/role/list?roleKey=${encodeURIComponent(name)}&page=1&pageSize=20`));
    const row = list.items.find((item) => item.id === role.id);
    assert(row && JSON.stringify([...row.permissionKeys].sort()) === JSON.stringify([...keys].sort()), `${label}: exact key-only public role readback`);
    return { ...session, roleId: role.id, roleBody };
  }
  async function audit(requestId, method, path) {
    check(requestId, 'Correlation request ID missing');
    let rows = [];
    for (let poll = 0; poll < 10; poll++) {
      const page = success(await api(tenantA, 'GET', `/system/operation-log/list?requestId=${encodeURIComponent(requestId)}&page=1&pageSize=20`));
      rows = page.items.filter((row) => row.requestId === requestId && row.method === method && row.operUrl.split('?')[0] === `/api/v1${path}` && row.tenantId === tenantA.tenantId);
      if (rows.length) break;
      await new Promise((done) => setTimeout(done, 500));
    }
    assert(rows.length === 1, `Audit ${method} ${path}: exact persisted row`);
    const detail = success(await api(tenantA, 'GET', `/system/operation-log/${rows[0].id}`));
    assert(detail.id === rows[0].id && detail.requestId === requestId, 'Audit detail/list correlation');
    safe(JSON.stringify(detail), 'Persisted audit');
    assert(!/"(?:rawQuery|password|bearerToken|result|recipe)"\s*:/.test(`${detail.operParam} ${detail.jsonResult}`), 'Audit omits sensitive body/result fields');
  }
  writeFileSync(resolve(privateRoot, 'public-attempt.json'), JSON.stringify({ startedAt: new Date().toISOString(), runId: suffix, configSHA256: hash(configPath), entrySHA256: result.entrySHA256 }, null, 2), { flag: 'wx' });
  try {
    platform = await login(config.platformAdmin);
    success(await api(platform, 'PUT', '/system/setting/group/platform', { items: [{ settingKey: 'tenant.mode', settingValue: 'multi' }] }, true));
    let tenantMode;
    for (let poll = 0; poll < 10; poll++) {
      tenantMode = success(await api(platform, 'GET', '/auth/tenants/current')).mode;
      if (tenantMode === 'multi') break;
      await new Promise((done) => setTimeout(done, 500));
    }
    assert(tenantMode === 'multi', 'Base tenant mode updated through public setting contract');
    tenantA = await provisionTenant('ta'); tenantB = await provisionTenant('tb');
    const keys = ['metric_source:view', 'metric_source:create', 'metric_source:delete', 'saved-view:view', 'saved-view:create', 'saved-view:update', 'saved-view:delete', 'saved-view:restore', 'query-history:view', 'query-history:delete', 'query-history:restore'].map((key) => `business:observability:${key}`);
    owner = await provisionActor('owner', keys);
    const peer = await provisionActor('peer', keys);
    const scoped = await provisionActor('self', keys, 'self');
    const sessions = success(await api(owner, 'GET', '/auth/sessions'));
    assert(sessions.filter((session) => session.isCurrent && !session.revokedAt).length === 1, 'Actual current Base session');
    const currentSession = sessions.find((session) => session.isCurrent).sessionId;
    const tenant = success(await api(owner, 'GET', '/auth/tenants/current'));
    assert(tenant.mode === 'multi' && tenant.tenantId === tenantA.tenantId, 'Base resolves exact current tenant');
    const sourceReply = await api(owner, 'POST', '/business/observability/datasources', { name: `${suffix}_source`, type: 'prometheus', url: config.providerUrl, accessMode: 'proxy', password: sourceCanaries[0], bearerToken: sourceCanaries[1], timeout: 5, isDefault: false });
    const source = success(sourceReply); check(source.id > 0, 'Datasource ID missing'); result.owned.sources.push(source.id);
    const sourcePath = `/business/observability/datasources/${source.id}`;
    cleanup.push(async () => { success(await api(owner, 'DELETE', sourcePath)); denied(await api(owner, 'GET', sourcePath), 'Cleanup: owned source absent', 404); });
    success(await api(peer, 'GET', sourcePath));
    denied(await api(scoped, 'GET', sourcePath), 'Base self DataScope hides another actor source', 404);
    denied(await api(tenantB, 'GET', sourcePath), 'Cross-tenant datasource hidden', 404);
    const recipe = { version: 1, sourceKind: 'prometheus', rules: [{ field: 'availability', op: 'instant' }], time: { mode: 'relative', duration: '5m' }, limit: 10, resultMode: 'table' };
    const view = success(await api(owner, 'POST', '/business/observability/saved-views', { datasourceId: source.id, recipe }));
    result.owned.views.push(view.id);
    const viewPath = `/business/observability/saved-views/${view.id}`;
    cleanup.push(async () => { success(await api(owner, 'DELETE', viewPath)); denied(await api(owner, 'GET', viewPath), 'Cleanup: owned view absent'); });
    denied(await api(null, 'GET', viewPath), 'Missing session denied', 401);
    denied(await api({ accessToken: 'invalid', csrfToken: 'invalid' }, 'GET', viewPath), 'Invalid session denied', 401);
    for (const actor of [peer, tenantB]) {
      denied(await api(actor, 'GET', viewPath), 'Foreign saved-view detail denied');
      denied(await api(actor, 'POST', `${viewPath}/restore`), 'Foreign saved-view restore denied');
      denied(await api(actor, 'PUT', viewPath, { datasourceId: source.id, expectedRevision: view.revision, recipe }), 'Foreign saved-view update denied');
      const deleted = await api(actor, 'DELETE', viewPath);
      assert(deleted.response.status === 403 || (deleted.response.ok && deleted.payload?.code === 200), 'Foreign saved-view delete denied or idempotent no-op');
      const list = success(await api(actor, 'GET', '/business/observability/saved-views'));
      assert(!list.items.some((row) => row.id === view.id), 'Foreign saved-view omitted from list');
    }
    const restored = success(await api(owner, 'POST', `${viewPath}/restore`));
    assert(restored.items.length === 1 && restored.items[0].revision === view.revision && JSON.stringify(restored.items[0].recipe) === JSON.stringify(recipe), 'Own restore preserves recipe/revision after foreign actions');
    const updated = success(await api(owner, 'PUT', viewPath, { datasourceId: source.id, expectedRevision: view.revision, recipe }));
    assert(updated.revision > view.revision, 'Saved-view revision advances');
    denied(await api(owner, 'PUT', viewPath, { datasourceId: source.id, expectedRevision: view.revision, recipe }), 'Stale saved-view revision conflicts', 409);
    denied(await api(owner, 'GET', `${sourcePath}/query?query=${encodeURIComponent(`up{acceptance_canary="${sourceCanaries[1]}"}`)}`), 'Legacy URL-expression query requires POST', 426);
    const queryReply = await api(owner, 'POST', `${sourcePath}/query`, { rawQuery: 'vector(1)', time: recipe.time, limit: 10, resultMode: 'table', historyMode: 'record' });
    success(queryReply); assert(queryReply.text.includes('"1"'), 'Actual Prometheus provider result through native public POST');
    const history = success(await api(owner, 'GET', '/business/observability/query-history'));
    assert(history.items.length === 1 && history.items[0].datasourceId === source.id && history.items[0].expressionUnavailable === true && !history.items[0].recipe, 'Raw history persists metadata without expression');
    const historyId = history.items[0].id; result.owned.histories.push(historyId);
    const historyPath = `/business/observability/query-history/${historyId}`;
    cleanup.push(async () => { success(await api(owner, 'DELETE', historyPath)); denied(await api(owner, 'GET', historyPath), 'Cleanup: owned history absent'); });
    for (const actor of [peer, tenantB]) {
      denied(await api(actor, 'GET', historyPath), 'Foreign history detail denied');
      denied(await api(actor, 'POST', `${historyPath}/restore`), 'Foreign history restore denied');
      const deleted = await api(actor, 'DELETE', historyPath);
      assert(deleted.response.status === 403 || (deleted.response.ok && deleted.payload?.code === 200), 'Foreign history delete denied or idempotent no-op');
      const list = success(await api(actor, 'GET', '/business/observability/query-history'));
      assert(!list.items.some((row) => row.id === historyId), 'Foreign history omitted from list');
    }
    const rawRestore = success(await api(owner, 'POST', `${historyPath}/restore`));
    assert(rawRestore.items[0].expressionUnavailable === true && !rawRestore.items[0].recipe, 'Raw-history restore cannot replay expression');
    await audit(sourceReply.requestId, 'POST', '/business/observability/datasources');
    await audit(queryReply.requestId, 'POST', `${sourcePath}/query`);
    success(await api(tenantA, 'PUT', `/system/role/${peer.roleId}`, { ...peer.roleBody, permissionKeys: [] }));
    denied(await api(peer, 'GET', sourcePath), 'Current role revocation denies stale session');
    const newOwner = await login({ username: `${suffix}_owner`, password }, tenantA.tenantId);
    success(await api(newOwner, 'DELETE', `/auth/sessions/${encodeURIComponent(currentSession)}`));
    denied(await api(owner, 'GET', viewPath), 'Revoked actual session denied', 401);
    owner = newOwner;
    for (const key of ['view', 'export', 'import']) {
      const actor = await provisionActor(key, [`business:cmdb:host:${key}`]);
      const list = await api(actor, 'GET', '/business/cmdb/hosts');
      if (key === 'view') { success(list); assert(true, 'CMDB view key admits list'); } else denied(list, `CMDB ${key} key denies list`);
      const exported = await api(actor, 'GET', '/business/cmdb/hosts/export');
      if (key === 'export') assert(exported.response.ok && exported.response.headers.get('content-type')?.includes('text/csv') && exported.text.includes('hostname'), 'CMDB export key admits CSV'); else denied(exported, `CMDB ${key} key denies export`);
      const template = await api(actor, 'GET', '/business/cmdb/hosts/import-template');
      if (key === 'import') assert(template.response.ok && template.response.headers.get('content-type')?.includes('text/csv'), 'CMDB import key admits template'); else denied(template, `CMDB ${key} key denies import template`);
      const data = new FormData(); data.append('file', new Blob(['hostname,ip,os\r\n,198.51.100.90,linux\r\n'], { type: 'text/csv' }), 'invalid.csv');
      const imported = await api(actor, 'POST', '/business/cmdb/hosts/import', data);
      if (key === 'import') { const row = success(imported); assert(row.applied === false && row.failed > 0, 'CMDB import key validates invalid CSV with zero host writes'); } else denied(imported, `CMDB ${key} key denies import`);
    }
    result.status = 'scoped-pass';
  } catch (error) { result.status = unknown ? 'unknown' : 'failed'; result.failure = error.message; }
  finally {
    if (unknown) result.cleanup = 'preserved-after-Unknown; inspect exact IDs before any successor mutation';
    else {
      const failures = [];
      for (let index = cleanup.length - 1; index >= 0; index--) {
        try { await cleanup[index](); } catch { failures.push(index); if (unknown) break; }
      }
      result.cleanup = failures.length ? 'incomplete; exact owned IDs retained' : 'exact-public-cleanup-complete';
      result.cleanupFailures = failures;
      if (failures.length) result.status = unknown ? 'unknown' : 'failed';
    }
    const current = sources();
    result.sourceStable = Object.keys(manifest).length === Object.keys(current).length && Object.entries(manifest).every(([path, digest]) => current[path] === digest);
    result.executableStable = hash(config.executable) === config.executableSHA256;
    if (!result.sourceStable || !result.executableStable) result.status = 'failed';
    result.runtimeLogs = {};
    for (const name of ['api.stdout.log', 'api.stderr.log']) {
      const path = resolve(privateRoot, name);
      const payload = readFileSync(path);
      result.runtimeLogs[name] = { bytes: payload.length, sha256: hash(path), privateCanariesAbsent: !canaries.some((value) => payload.includes(value)) };
      if (!result.runtimeLogs[name].privateCanariesAbsent) { result.status = 'failed'; result.failure = 'Native runtime log exposes a private canary'; }
    }
    result.finishedAt = new Date().toISOString();
    writeFileSync(resolve(privateRoot, 'public-result.json'), JSON.stringify(result, null, 2), { flag: 'wx' });
    writeFileSync(resolve(evidence, `${suffix.replaceAll('_', '-')}-public-http.json`), JSON.stringify(result, null, 2), { flag: 'wx' });
    console.log(JSON.stringify({ status: result.status, assertions: result.assertions.length, requests: result.requests.length, cleanup: result.cleanup, sourceStable: result.sourceStable }));
  }
  process.exitCode = result.status === 'scoped-pass' ? 0 : 1;
}

try {
  if (process.argv[2] === '--check') { preflight(process.argv[3]); console.log(JSON.stringify({ preflight: 'pass', runtimeExecuted: false })); }
  else await run(process.argv[2]);
} catch { console.error('Current public entry failed; credentials omitted'); process.exitCode = 1; }
