#!/usr/bin/env node
// Maps a changed-file list to the smoke module suites that must run for it.
//
// Contract: any file that is not *unambiguously* inside one module resolves to the full module
// set. A shared-core or wiring change must never be able to silently skip a module's suite.
//
// Usage (CI):
//   CHANGED_FILES="<space separated paths>" node scripts/resolve-smoke-modules.mjs
//   -> writes `modules=<comma separated npm script names>` to $GITHUB_OUTPUT
//
// Usage (local dry run):
//   node scripts/resolve-smoke-modules.mjs frontend/src/modules/system/iam/api/user.ts

import fs from 'node:fs';
import { pathToFileURL } from 'node:url';

const MODULE_SCRIPTS = {
  iam: 'test:smoke:module:iam',
  config: 'test:smoke:module:config',
  i18n: 'test:smoke:module:i18n',
  audit: 'test:smoke:module:audit',
  org: 'test:smoke:module:org',
  platform: 'test:smoke:module:platform',
};
const ALL_SCRIPT = 'test:smoke:module:all';

// Files that cannot change any runtime surface covered by smoke.
const IGNORED = [
  /^docs\//,
  /^\.harness\//,
  /^\.agents\//,
  /^dist\//,
  /^scripts\//,
  /^tests\/scripts\//,
  /^frontend\/tests\/smoke\/README\.md$/,
  /^config\/method\.config\.json$/,
  /^config\/ui-quality-gate\.json$/,
  /^(README|DESIGN|SECURITY|AGENTS|CLAUDE)(\.[a-z]{2})?\.md$/,
];

// First matching rule wins. A module owns its backend package, its frontend module tree, and the
// smoke specs that exercise it.
const MODULE_RULES = [
  {
    module: 'iam',
    patterns: [
      /^backend\/modules\/system\/iam\//,
      /^frontend\/src\/modules\/system\/iam\//,
      /^frontend\/tests\/smoke\/system\/system-iam\.spec\.ts$/,
      /^frontend\/tests\/smoke\/system\/role-authorization\.spec\.ts$/,
    ],
  },
  {
    module: 'config',
    patterns: [
      /^backend\/modules\/system\/config\//,
      /^frontend\/src\/modules\/system\/config\//,
      /^frontend\/tests\/smoke\/system\/system-config\.spec\.ts$/,
    ],
  },
  {
    module: 'i18n',
    patterns: [
      /^backend\/modules\/system\/i18n\//,
      /^frontend\/src\/modules\/system\/i18n\//,
      /^frontend\/tests\/smoke\/system\/system-i18n\.spec\.ts$/,
    ],
  },
  {
    module: 'audit',
    patterns: [
      /^backend\/modules\/system\/audit\//,
      /^frontend\/src\/modules\/system\/audit\//,
      /^frontend\/tests\/smoke\/system\/system-audit\.spec\.ts$/,
    ],
  },
  {
    module: 'org',
    patterns: [
      /^backend\/modules\/system\/org\//,
      /^frontend\/src\/modules\/system\/org\//,
      /^frontend\/tests\/smoke\/system\/system-org\.spec\.ts$/,
    ],
  },
  {
    module: 'platform',
    patterns: [
      /^backend\/modules\/platform\//,
      /^frontend\/src\/modules\/platform\//,
      /^frontend\/tests\/smoke\/system\/system-platform\.spec\.ts$/,
    ],
  },
];

const normalize = (file) => file.trim().replace(/\\/g, '/').replace(/^\.\//, '');

export function resolveSmokeModules(files) {
  const modules = new Set();
  let everything = false;

  for (const raw of files) {
    const file = normalize(raw);
    if (!file) continue;
    if (IGNORED.some((re) => re.test(file))) continue;

    const rule = MODULE_RULES.find(({ patterns }) => patterns.some((re) => re.test(file)));
    if (rule) {
      modules.add(rule.module);
    } else {
      everything = true;
    }
  }

  if (everything) return [ALL_SCRIPT];
  return [...modules].sort().map((module) => MODULE_SCRIPTS[module]);
}

function main() {
  const fromJson = (() => {
    const raw = process.env.CHANGED_FILES_JSON;
    if (!raw) return [];
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      throw new Error(`CHANGED_FILES_JSON is not a JSON array: ${raw.slice(0, 200)}`);
    }
  })();
  const fromEnv = (process.env.CHANGED_FILES ?? '').split(/\s+/);
  const fromArgv = process.argv.slice(2);
  const files = [...fromJson, ...fromEnv, ...fromArgv].filter(Boolean);

  const scripts = resolveSmokeModules(files);
  const value = scripts.join(',');

  console.log(`changed files : ${files.length}`);
  console.log(`smoke modules : ${value || '(none — nothing to run)'}`);

  const outputPath = process.env.GITHUB_OUTPUT;
  if (outputPath) {
    fs.appendFileSync(outputPath, `modules=${value}\n`, 'utf8');
  }
}

const invokedDirectly =
  process.argv[1] !== undefined && pathToFileURL(process.argv[1]).href === import.meta.url;
if (invokedDirectly) main();
