import process from 'node:process';
import { readAuthCookieSession } from './lib/auth-cookie-session.mjs';

const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8080/api/v1';
const adminUsername = process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin';
const adminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456';
const defaultModule = 'system.config';
const defaultPageSize = 500;
const deleteChunkSize = 200;

function hasFlag(flag) {
  return process.argv.includes(flag);
}

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return String(process.argv[index + 1] ?? '').trim();
}

async function login() {
  const response = await fetch(`${apiBaseUrl}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: adminUsername, password: adminPassword }),
  });
  if (!response.ok) {
    throw new Error(`Login failed: HTTP ${response.status}`);
  }
  const payload = await response.json();
  if (payload.code !== 200) {
    throw new Error(`Login failed: code ${payload.code}`);
  }
  return readAuthCookieSession(response.headers, {
    includeRefreshToken: false,
    csrfFallback: `pantheon-placeholder-cleanup-csrf-${Date.now()}`,
  });
}

function authHeaders(session) {
  return {
    Authorization: `Bearer ${session.accessToken}`,
    'X-CSRF-Token': session.csrfToken,
    Cookie: `pantheon_csrf_token=${session.csrfToken}`,
  };
}

async function apiGet(session, path, params = {}) {
  const url = new URL(`${apiBaseUrl}${path}`);
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') {
      continue;
    }
    url.searchParams.set(key, String(value));
  }
  const response = await fetch(url, {
    headers: authHeaders(session),
  });
  if (!response.ok) {
    throw new Error(`GET ${path} failed: HTTP ${response.status}`);
  }
  const payload = await response.json();
  if (payload.code !== 200) {
    throw new Error(`GET ${path} failed: code ${payload.code}`);
  }
  return payload.data;
}

async function apiPost(session, path, data = {}) {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: 'POST',
    headers: {
      ...authHeaders(session),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error(`POST ${path} failed: HTTP ${response.status}`);
  }
  const payload = await response.json();
  if (payload.code !== 200) {
    throw new Error(`POST ${path} failed: code ${payload.code}`);
  }
  return payload.data;
}

function chunk(items, size) {
  const chunks = [];
  for (let index = 0; index < items.length; index += size) {
    chunks.push(items.slice(index, index + size));
  }
  return chunks;
}

async function listModuleRows(session, moduleName, pageSize) {
  const firstPage = await apiGet(session, '/system/i18n/list', {
    module: moduleName,
    page: 1,
    pageSize,
    sortBy: 'key',
    sortOrder: 'asc',
  });
  const rows = Array.isArray(firstPage.items) ? [...firstPage.items] : [];
  const total = Number(firstPage.total ?? rows.length);
  const effectivePageSize = Number(firstPage.pageSize ?? pageSize) || pageSize;
  const totalPages = Math.max(1, Math.ceil(total / effectivePageSize));
  for (let page = 2; page <= totalPages; page += 1) {
    const payload = await apiGet(session, '/system/i18n/list', {
      module: moduleName,
      page,
      pageSize,
      sortBy: 'key',
      sortOrder: 'asc',
    });
    if (Array.isArray(payload.items)) {
      rows.push(...payload.items);
    }
  }
  return rows;
}

async function main() {
  const moduleName = readArg('--module', defaultModule) || defaultModule;
  const pageSize = Number.parseInt(readArg('--page-size', String(defaultPageSize)), 10) || defaultPageSize;
  const dryRun = hasFlag('--dry-run');
  const jsonOnly = hasFlag('--json');

  const session = await login();
  const audit = await apiGet(session, '/system/i18n/audit');
  const candidates = (Array.isArray(audit.unusedKeys) ? audit.unusedKeys : [])
    .filter((item) => item.module === moduleName && item.placeholder)
    .sort((a, b) => String(a.key).localeCompare(String(b.key)));

  const lifecycleCounts = candidates.reduce(
    (acc, item) => {
      const status = String(item.lifecycleStatus || 'active');
      acc[status] = (acc[status] || 0) + 1;
      return acc;
    },
    {},
  );

  const moduleRows = await listModuleRows(session, moduleName, pageSize);
  const candidateKeys = new Set(candidates.map((item) => String(item.key)));
  const matchedRows = moduleRows.filter((row) => candidateKeys.has(String(row.key || '')));
  const matchedIds = matchedRows.map((row) => Number(row.id)).filter((id) => Number.isFinite(id) && id > 0);
  const deleteChunks = chunk(matchedIds, deleteChunkSize);

  let deletedRows = 0;
  if (!dryRun) {
    for (const ids of deleteChunks) {
      if (ids.length === 0) {
        continue;
      }
      const result = await apiPost(session, '/system/i18n/batch-delete', { ids });
      deletedRows += Number(result.count ?? ids.length);
    }
  }

  const payload = {
    generatedAt: new Date().toISOString(),
    module: moduleName,
    dryRun,
    candidateKeyCount: candidates.length,
    candidateRowCount: matchedIds.length,
    lifecycleCounts,
    sampleKeys: candidates.slice(0, 20).map((item) => item.key),
    deletedRows,
  };

  if (jsonOnly) {
    console.log(JSON.stringify(payload, null, 2));
    return;
  }

  console.log(`unused placeholder cleanup @ ${payload.generatedAt}`);
  console.log(`module=${moduleName} dryRun=${dryRun}`);
  console.log(`candidateKeys=${payload.candidateKeyCount} candidateRows=${payload.candidateRowCount}`);
  console.log(`lifecycle=${JSON.stringify(lifecycleCounts)}`);
  console.log(`deletedRows=${deletedRows}`);
  console.log('');
  console.log('Sample keys');
  if (payload.sampleKeys.length === 0) {
    console.log('- none');
  } else {
    for (const key of payload.sampleKeys) {
      console.log(`- ${key}`);
    }
  }
}

await main();
