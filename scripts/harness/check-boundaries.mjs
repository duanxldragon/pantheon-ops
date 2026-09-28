#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();
const REPOSITORIES = ['pantheon-base', 'pantheon-ops'];

// Cross-layer import rules. Contract source: docs/designs/REPOSITORY_LAYOUT.md §8.2.
// Test files are exempt by construction (see isTestFile) — tests may wire real
// modules together, production code may not.
const BUSINESS_RULES = {
  go: [
    { pattern: /\/backend\/internal\//, reason: 'business backend must not import backend/internal directly' },
    { pattern: /\/backend\/modules\/system\//, reason: 'business backend must not import system module internals directly' },
    { pattern: /\/backend\/modules\/auth\//, reason: 'business backend must not import auth module internals directly' },
    { pattern: /\/backend\/modules\/platform\//, reason: 'business backend must not import platform module internals directly' },
  ],
  ts: [
    { pattern: /(?:^|\/)modules\/system(?:\/|$)/, reason: 'business frontend must not import system module internals directly' },
    { pattern: /(?:^|\/)modules\/auth(?:\/|$)/, reason: 'business frontend must not import auth module internals directly' },
    { pattern: /(?:^|\/)modules\/platform(?:\/|$)/, reason: 'business frontend must not import platform module internals directly' },
    { pattern: /(?:^|\/)\.\.\/system(?:\/|$)/, reason: 'business frontend must not reach into sibling system modules' },
    { pattern: /(?:^|\/)\.\.\/auth(?:\/|$)/, reason: 'business frontend must not reach into sibling auth modules' },
    { pattern: /(?:^|\/)\.\.\/platform(?:\/|$)/, reason: 'business frontend must not reach into sibling platform modules' },
  ],
};

// platform may only reach system/auth/business through a composition-root adapter
// (see backend/cmd/server/platform_org_governance.go), never by importing the
// implementation from inside the platform module.
const PLATFORM_RULES = {
  go: [
    { pattern: /\/backend\/modules\/system\//, reason: 'platform must not import system module internals directly (use a composition-root adapter)' },
    { pattern: /\/backend\/modules\/auth\//, reason: 'platform must not import auth module internals directly' },
    { pattern: /\/backend\/modules\/business\//, reason: 'platform must not import business module internals directly' },
  ],
  ts: [
    { pattern: /(?:^|\/)modules\/system(?:\/|$)/, reason: 'platform frontend must not import system module internals directly' },
    { pattern: /(?:^|\/)modules\/auth(?:\/|$)/, reason: 'platform frontend must not import auth module internals directly' },
    { pattern: /(?:^|\/)modules\/business(?:\/|$)/, reason: 'platform frontend must not import business module internals directly' },
    { pattern: /(?:^|\/)\.\.\/system(?:\/|$)/, reason: 'platform frontend must not reach into sibling system modules' },
    { pattern: /(?:^|\/)\.\.\/auth(?:\/|$)/, reason: 'platform frontend must not reach into sibling auth modules' },
    { pattern: /(?:^|\/)\.\.\/business(?:\/|$)/, reason: 'platform frontend must not reach into sibling business modules' },
  ],
};

// auth belongs to the system foundation layer: it may consume system subdomains
// through their public api modules, but not through internal components/hooks/
// utils, and never through business or platform internals.
const AUTH_RULES = {
  go: [
    { pattern: /\/backend\/modules\/business\//, reason: 'auth must not import business module internals directly' },
    { pattern: /\/backend\/modules\/platform\//, reason: 'auth must not import platform module internals directly' },
    { pattern: /\/backend\/modules\/system\//, reason: 'auth must not import system module internals directly (use a public contract)' },
  ],
  ts: [
    { pattern: /(?:^|\/)modules\/business(?:\/|$)/, reason: 'auth frontend must not import business module internals directly' },
    { pattern: /(?:^|\/)modules\/platform(?:\/|$)/, reason: 'auth frontend must not import platform module internals directly' },
    { pattern: /(?:^|\/)\.\.\/business(?:\/|$)/, reason: 'auth frontend must not reach into sibling business modules' },
    { pattern: /(?:^|\/)\.\.\/platform(?:\/|$)/, reason: 'auth frontend must not reach into sibling platform modules' },
    { pattern: /system\/(components|hooks|utils)(?:\/|$)/, reason: 'auth frontend must not import system internal components/hooks/utils directly (system api contracts are allowed)' },
  ],
};

// system/* subdomains may only be wired together at the composition root
// (backend/modules/system/system_modules.go). A subdomain reaching into another
// subdomain's packages creates a hidden coupling that the contract docs
// (docs/contracts/SYSTEM_*_CONTRACT.md) do not sanction.
const SYSTEM_DOMAINS = ['audit', 'config', 'i18n', 'iam', 'org'];

function systemDomainRules(domain) {
  const others = SYSTEM_DOMAINS.filter((candidate) => candidate !== domain);
  return {
    go: [
      {
        pattern: new RegExp(`/backend/modules/system/(?:${others.join('|')})/`),
        reason: `system/${domain} must not import another system subdomain's packages (wire them at backend/modules/system/system_modules.go instead)`,
      },
    ],
    ts: [],
  };
}

// Each subdomain is scanned as its own layer so "another subdomain" can be
// expressed without a self-exemption escape hatch.
const SYSTEM_SCANS = SYSTEM_DOMAINS.map((domain) => ({
  layer: `system/${domain}`,
  backendDir: `backend/modules/system/${domain}`,
  frontendDir: `frontend/src/modules/system/${domain}`,
  rules: systemDomainRules(domain),
  requireBusinessDirs: false,
}));

const LAYER_SCANS = {
  'pantheon-base': [
    { layer: 'business', backendDir: 'backend/modules/business', frontendDir: 'frontend/src/modules/business', rules: BUSINESS_RULES, requireBusinessDirs: false },
    { layer: 'platform', backendDir: 'backend/modules/platform', frontendDir: 'frontend/src/modules/platform', rules: PLATFORM_RULES, requireBusinessDirs: false },
    { layer: 'auth', backendDir: 'backend/modules/auth', frontendDir: 'frontend/src/modules/auth', rules: AUTH_RULES, requireBusinessDirs: false },
    ...SYSTEM_SCANS,
  ],
  'pantheon-ops': [
    { layer: 'business', backendDir: 'backend/modules/business', frontendDir: 'frontend/src/modules/business', rules: BUSINESS_RULES, requireBusinessDirs: true },
  ],
};

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-boundaries.mjs [--json] [--strict] [--root <path>] [--repo <name>] [--baseline <path>]

Default behavior:
  Report findings and exit 0.  Use --strict to exit 1 when unbaselined findings exist.
  Scanned layers: business, platform, auth and each system/* subdomain.
  Use --repo <name> to scan only one repository (default scans all).
  Use --baseline <path> to treat recorded, review-dated findings as known debt.
  Under --strict, a baseline entry that no longer matches any finding (stale)
  fails: remove it so it cannot whitelist a reintroduced import.

Examples:
  node scripts/harness/check-boundaries.mjs
  node scripts/harness/check-boundaries.mjs --strict --baseline config/boundary-baseline.json
  node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    root: DEFAULT_ROOT,
    repo: null,
    baseline: null,
  };

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--root') {
      const value = argv[++i];
      if (!value) throw new Error('--root requires a path');
      options.root = path.resolve(value);
    } else if (arg === '--repo') {
      const value = argv[++i];
      if (!value) throw new Error('--repo requires a repository name');
      options.repo = value;
    } else if (arg === '--baseline') {
      const value = argv[++i];
      if (!value) throw new Error('--baseline requires a path');
      options.baseline = path.resolve(options.root, value);
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function walkFiles(rootDir, extensions) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  const files = [];
  const stack = [rootDir];

  while (stack.length > 0) {
    const current = stack.pop();
    const entries = fs.readdirSync(current, { withFileTypes: true });

    for (const entry of entries) {
      const entryPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(entryPath);
      } else if (extensions.some((extension) => entry.name.endsWith(extension))) {
        files.push(entryPath);
      }
    }
  }

  return sortStrings(files);
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function isTestFile(fileName) {
  return /_test\.go$/.test(fileName) || /\.(test|spec)\.[cm]?tsx?$/.test(fileName);
}

function collectImports(content, importPattern) {
  const imports = [];
  let match;
  while ((match = importPattern.exec(content)) !== null) {
    imports.push(match[1].replaceAll('\\', '/'));
  }
  return imports;
}

function scanFile(filePath, root, rules, language) {
  const content = fs.readFileSync(filePath, 'utf8');
  const importPattern =
    language === 'go'
      ? /"([^"]+)"/g
      : /(?:import|export)\s+(?:type\s+)?(?:[\s\S]*?\s+from\s+)?['"]([^'"]+)['"]/g;
  const findings = [];

  for (const importPath of collectImports(content, importPattern)) {
    for (const rule of rules) {
      if (rule.pattern.test(importPath)) {
        findings.push({
          file: toRepoPath(filePath, root),
          importPath,
          reason: rule.reason,
        });
      }
    }
  }

  return findings;
}

function loadBaseline(baselinePath) {
  if (!baselinePath) {
    return { entries: [], reviewBy: null, missing: false };
  }
  if (!fs.existsSync(baselinePath)) {
    throw new Error(`baseline file not found: ${baselinePath}`);
  }

  const payload = JSON.parse(fs.readFileSync(baselinePath, 'utf8'));
  const entries = Array.isArray(payload.entries) ? payload.entries : [];
  for (const entry of entries) {
    if (typeof entry.file !== 'string' || typeof entry.importPath !== 'string') {
      throw new Error(`baseline entry must have string "file" and "importPath": ${JSON.stringify(entry)}`);
    }
  }
  return { entries, reviewBy: payload.reviewBy ?? null, missing: false };
}

function baselineKey(file, importPath) {
  return `${file}|${importPath}`;
}

function scanLayer(layerScan, repoRoot, root) {
  const warnings = [];
  const findings = [];

  const backendRoot = path.join(repoRoot, layerScan.backendDir);
  const frontendRoot = path.join(repoRoot, layerScan.frontendDir);

  // Findings are reported relative to the repository root (not the invocation
  // root) so a baseline recorded in one checkout matches from any working dir.
  if (fs.existsSync(backendRoot)) {
    for (const filePath of walkFiles(backendRoot, ['.go'])) {
      if (isTestFile(path.basename(filePath))) continue;
      findings.push(...scanFile(filePath, repoRoot, layerScan.rules.go, 'go'));
    }
  } else if (layerScan.requireBusinessDirs) {
    warnings.push(`Backend ${layerScan.layer} root not found: ${toRepoPath(backendRoot, root)}`);
  }

  if (fs.existsSync(frontendRoot)) {
    for (const filePath of walkFiles(frontendRoot, ['.ts', '.tsx'])) {
      if (isTestFile(path.basename(filePath))) continue;
      findings.push(...scanFile(filePath, repoRoot, layerScan.rules.ts, 'ts'));
    }
  } else if (layerScan.requireBusinessDirs) {
    warnings.push(`Frontend ${layerScan.layer} root not found: ${toRepoPath(frontendRoot, root)}`);
  }

  return { findings, warnings };
}

function scanRepository(repoName, root, baseline) {
  const warnings = [];
  const findings = [];
  // 约定 root 为 workspace 根（含各仓库目录）。但当 root 本身就是目标仓库
  // （例如 CI 在仓库内 checkout 后 cwd = 仓库根，其目录名 == repoName），
  // 直接使用 root 作为仓库根，避免 path.join(root, repoName) 找不到。
  let repoRoot = path.join(root, repoName);
  if (!fs.existsSync(repoRoot) && path.basename(root) === repoName) {
    repoRoot = root;
  }

  if (!fs.existsSync(repoRoot)) {
    warnings.push(`Repository root not found: ${repoName}`);
    return { repo: repoName, findings, baselined: [], warnings };
  }

  const scans = LAYER_SCANS[repoName] ?? [];
  for (const layerScan of scans) {
    const result = scanLayer(layerScan, repoRoot, root);
    findings.push(...result.findings);
    warnings.push(...result.warnings);
  }

  const baselined = [];
  const remaining = [];
  const baselineKeys = new Set(baseline.entries.map((entry) => baselineKey(entry.file, entry.importPath)));
  const usedBaselineKeys = new Set();
  for (const finding of findings) {
    const key = baselineKey(finding.file, finding.importPath);
    if (baselineKeys.has(key)) {
      baselined.push(finding);
      usedBaselineKeys.add(key);
    } else {
      remaining.push(finding);
    }
  }

  // A stale entry is not harmless bookkeeping: it still whitelists its
  // file+import pair, so if that import ever comes back it passes silently.
  // Stale entries are therefore reported separately and fail --strict.
  const staleBaseline = [];
  for (const entry of baseline.entries) {
    const key = baselineKey(entry.file, entry.importPath);
    if (!usedBaselineKeys.has(key)) {
      staleBaseline.push(entry);
      warnings.push(
        `baseline entry no longer matches any finding (stale, remove it): ${entry.file} -> ${entry.importPath}`,
      );
    }
  }

  return { repo: repoName, findings: remaining, baselined, staleBaseline, warnings };
}

function printTextReport(results, strict, baseline) {
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const baselinedCount = results.reduce((count, result) => count + (result.baselined?.length ?? 0), 0);
  const staleCount = results.reduce((count, result) => count + (result.staleBaseline?.length ?? 0), 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const mode = strict ? 'strict' : 'report-only';

  console.log(`Boundary check (${mode}): ${findingCount} finding(s), ${baselinedCount} baselined, ${staleCount} stale baselined, ${warningCount} warning(s)`);
  if (baseline.reviewBy) {
    console.log(`Baseline review-by: ${baseline.reviewBy}`);
  }

  for (const result of results) {
    console.log(`\n${result.repo}`);

    if (result.findings.length === 0 && (result.baselined?.length ?? 0) === 0) {
      console.log('  no findings');
    }

    for (const finding of result.findings) {
      console.log(`  finding: ${finding.file}`);
      console.log(`    import: ${finding.importPath}`);
      console.log(`    reason: ${finding.reason}`);
    }

    for (const finding of result.baselined ?? []) {
      console.log(`  baselined: ${finding.file}`);
      console.log(`    import: ${finding.importPath}`);
    }

    for (const warning of result.warnings) {
      console.log(`  warning: ${warning}`);
    }
  }
}

function main() {
  let options;

  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }

  if (options.help) {
    printHelp();
    return 0;
  }

  if (options.repo && !REPOSITORIES.includes(options.repo)) {
    console.error(`Unknown repository: ${options.repo} (known: ${REPOSITORIES.join(', ')})`);
    return 1;
  }

  let baseline;
  try {
    baseline = loadBaseline(options.baseline);
  } catch (error) {
    console.error(error.message);
    return 1;
  }

  const repositories = options.repo ? [options.repo] : REPOSITORIES;

  const results = repositories.map((repo) => scanRepository(repo, options.root, baseline));
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const baselinedCount = results.reduce((count, result) => count + (result.baselined?.length ?? 0), 0);
  const staleCount = results.reduce((count, result) => count + (result.staleBaseline?.length ?? 0), 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          baselineReviewBy: baseline.reviewBy,
          findingCount,
          baselinedCount,
          staleCount,
          warningCount,
          results,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(results, options.strict, baseline);
  }

  // --strict fails on new violations AND on stale baseline entries. A stale entry
  // must be removed (not kept) so it cannot mask a reintroduced import later.
  return options.strict && (findingCount > 0 || staleCount > 0) ? 1 : 0;
}

process.exitCode = main();
