import process from 'node:process';
import { readAuthCookieSession } from './lib/auth-cookie-session.mjs';

const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8080/api/v1';
const adminUsername = process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin';
const adminPassword = process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456';
const smokePrefixes = ['i18n.enter.', 'i18n.smoke.', 'i18n.import.', 'i18n.sync.'];

function readFlag(flag) {
  return process.argv.includes(flag);
}

function readArg(flag, fallback = '') {
  const index = process.argv.indexOf(flag);
  if (index < 0 || index + 1 >= process.argv.length) {
    return fallback;
  }
  return String(process.argv[index + 1] ?? '').trim();
}

function authHeaders(session) {
  return {
    Authorization: `Bearer ${session.accessToken}`,
    'X-CSRF-Token': session.csrfToken,
    Cookie: `pantheon_csrf_token=${session.csrfToken}`,
  };
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
    csrfFallback: `pantheon-audit-csrf-${Date.now()}`,
  });
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

function compareModuleRisk(a, b) {
  if (a.deleteEligibleKeyCount !== b.deleteEligibleKeyCount) {
    return b.deleteEligibleKeyCount - a.deleteEligibleKeyCount;
  }
  if (a.archivedKeyCount !== b.archivedKeyCount) {
    return b.archivedKeyCount - a.archivedKeyCount;
  }
  if (a.observingKeyCount !== b.observingKeyCount) {
    return b.observingKeyCount - a.observingKeyCount;
  }
  if (a.unusedKeyCount !== b.unusedKeyCount) {
    return b.unusedKeyCount - a.unusedKeyCount;
  }
  return String(a.module).localeCompare(String(b.module));
}

function toModuleSummary(item) {
  return {
    module: item.module,
    entryCount: item.entryCount,
    unusedKeyCount: item.unusedKeyCount,
    observingKeyCount: item.observingKeyCount,
    archivedKeyCount: item.archivedKeyCount,
    deleteEligibleKeyCount: item.deleteEligibleKeyCount ?? 0,
  };
}

function toUnusedKeySummary(item) {
  return {
    module: item.module,
    key: item.key,
    lifecycleStatus: item.lifecycleStatus,
    observingDays: item.observingDays,
    eligibleForArchive: item.eligibleForArchive,
    eligibleForDelete: item.eligibleForDelete,
  };
}

async function main() {
  const targetModule = readArg('--module', '');
  const jsonOnly = readFlag('--json');
  const session = await login();
  const audit = await apiGet(session, '/system/i18n/audit');

  const moduleRows = Array.isArray(audit.modules) ? audit.modules : [];
  const unusedRows = Array.isArray(audit.unusedKeys) ? audit.unusedKeys : [];

  const filteredModules = moduleRows
    .filter((item) => !targetModule || item.module === targetModule)
    .filter(
      (item) =>
        item.unusedKeyCount > 0 ||
        item.observingKeyCount > 0 ||
        item.archivedKeyCount > 0 ||
        (item.deleteEligibleKeyCount ?? 0) > 0,
    )
    .map(toModuleSummary)
    .sort(compareModuleRisk);

  const filteredUnused = unusedRows
    .filter((item) => !targetModule || item.module === targetModule)
    .map(toUnusedKeySummary);

  const smokeKeys = filteredUnused
    .filter((item) => smokePrefixes.some((prefix) => item.key.startsWith(prefix)))
    .sort((a, b) =>
      a.module === b.module ? a.key.localeCompare(b.key) : a.module.localeCompare(b.module),
    );

  const deleteEligible = filteredUnused
    .filter((item) => item.eligibleForDelete)
    .sort((a, b) =>
      a.module === b.module ? a.key.localeCompare(b.key) : a.module.localeCompare(b.module),
    );

  const payload = {
    generatedAt: new Date().toISOString(),
    module: targetModule || null,
    totalUnusedKeys: filteredUnused.length,
    totalModules: filteredModules.length,
    observationThresholdDays: audit.unusedObservationThresholdDays,
    archivedRetentionThresholdDays:
      typeof audit.archivedRetentionThresholdDays === 'number'
        ? audit.archivedRetentionThresholdDays
        : null,
    modules: filteredModules,
    smokeKeys,
    deleteEligible,
  };

  if (jsonOnly) {
    console.log(JSON.stringify(payload, null, 2)); // NOSONAR - build-only audit script, not a production service
    return;
  }

  const deleteThresholdLabel =
    payload.archivedRetentionThresholdDays === null
      ? 'n/a'
      : `${payload.archivedRetentionThresholdDays}d`;

  console.log(`system_i18n runtime audit @ ${payload.generatedAt}`); // NOSONAR - build-only audit script
  console.log( // NOSONAR - build-only audit script
    `unused=${payload.totalUnusedKeys} modules=${payload.totalModules} observe>=${payload.observationThresholdDays}d delete>=${deleteThresholdLabel}`,
  );
  console.log('');

  if (filteredModules.length === 0) {
    console.log('No unused/observing/archived modules found.');
  } else {
    console.log('Modules'); // NOSONAR - build-only audit script
    for (const item of filteredModules) {
      console.log( // NOSONAR - build-only audit script
        `- ${item.module}: unused=${item.unusedKeyCount}, observing=${item.observingKeyCount}, archived=${item.archivedKeyCount}, deleteEligible=${item.deleteEligibleKeyCount}`,
      );
    }
  }

  console.log('');
  console.log(`Smoke keys (${smokeKeys.length})`); // NOSONAR - build-only audit script
  if (smokeKeys.length === 0) {
    console.log('- none'); // NOSONAR - build-only audit script
  } else {
    for (const item of smokeKeys) {
      console.log( // NOSONAR - build-only audit script
        `- ${item.module} :: ${item.key} [${item.lifecycleStatus}] observingDays=${item.observingDays}`,
      );
    }
  }

  console.log('');
  console.log(`Delete-eligible keys (${deleteEligible.length})`); // NOSONAR - build-only audit script
  if (deleteEligible.length === 0) {
    console.log('- none'); // NOSONAR - build-only audit script
  } else {
    for (const item of deleteEligible) {
      console.log( // NOSONAR - build-only audit script
        `- ${item.module} :: ${item.key} [${item.lifecycleStatus}] observingDays=${item.observingDays}`,
      );
    }
  }
}

await main();
