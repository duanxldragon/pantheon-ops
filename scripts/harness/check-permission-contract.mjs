#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();
const REPOSITORIES = ['pantheon-base', 'pantheon-ops'];
const ACTION_TOKENS = [
  'create',
  'update',
  'delete',
  'batch',
  'import',
  'export',
  'generate',
  'register',
  'unregister',
  'start',
  'cancel',
  'reset',
  'remediate',
  'refresh',
  'clear',
  'upload',
];

const PERMISSION_KEY_PATTERN = /(?:system|business|platform):[a-z0-9:-]+/g;
const LIST_PERMISSION_PATTERN = /(?:system|business|platform):[a-z0-9:-]+:list\b/g;

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-permission-contract.mjs [--json] [--strict] [--root <path>]

Default behavior:
  Report findings and warnings, then exit 0. Use --strict to exit 1 when findings exist.

Examples:
  node scripts/harness/check-permission-contract.mjs
  node scripts/harness/check-permission-contract.mjs --json
  node scripts/harness/check-permission-contract.mjs --strict
  node scripts/harness/check-permission-contract.mjs --root /tmp/fixture`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    root: DEFAULT_ROOT,
  };

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--root') {
      const value = argv[++i];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
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
        if (entry.name === 'node_modules' || entry.name === 'dist') {
          continue;
        }
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

function lineContext(lines, index, radius = 3) {
  const start = Math.max(0, index - radius);
  const end = Math.min(lines.length, index + radius + 1);
  return lines.slice(start, end).join('\n');
}

function hasActionToken(text) {
  const lower = text.toLowerCase();
  return ACTION_TOKENS.some((token) => lower.includes(token));
}

function scanGoFile(filePath, root) {
  const content = fs.readFileSync(filePath, 'utf8');
  const findings = [];
  const warnings = [];

  const actionMenuPattern =
    /\{[^{}\n]*(?:Key:\s*"([^"]+)")?[^{}\n]*Perms:\s*"([^"]+:list)"[^{}\n]*Type:\s*"F"[^{}\n]*\}|\{[^{}\n]*(?:Key:\s*"([^"]+)")?[^{}\n]*Type:\s*"F"[^{}\n]*Perms:\s*"([^"]+:list)"[^{}\n]*\}/g;
  let match;

  while ((match = actionMenuPattern.exec(content)) !== null) {
    findings.push({
      file: toRepoPath(filePath, root),
      line: lineNumberAt(content, match.index),
      permission: match[2] || match[4],
      reason: 'action menu Type F must not use a list permission',
    });
  }

  const lines = content.split(/\r?\n/);
  lines.forEach((line, index) => {
    const listPermissions = line.match(LIST_PERMISSION_PATTERN) || [];
    if (listPermissions.length === 0) {
      return;
    }

    if (/\bPagePerm:\s*"[^"]+:list"/.test(line) && /\bType:\s*"C"/.test(line)) {
      return;
    }

    if (/^\s*case\s+"[^"]+:list":/.test(line)) {
      const policyBlock = lines.slice(index, index + 6).join('\n');
      if (/\bMethod:\s*"GET"/.test(policyBlock) && !/\bMethod:\s*"(POST|PUT|PATCH|DELETE)"/.test(policyBlock)) {
        return;
      }
    }

    const context = lineContext(lines, index);
    if (hasActionToken(context) && /PermissionKey|permission_key|Perms|permission/i.test(context)) {
      warnings.push({
        file: toRepoPath(filePath, root),
        line: index + 1,
        permission: Array.from(new Set(listPermissions)).join(', '),
        reason: 'list permission appears near action wording; review whether a dedicated action permission is required',
      });
    }
  });

  return { findings, warnings };
}

function scanTsFile(filePath, root) {
  const content = fs.readFileSync(filePath, 'utf8');
  const findings = [];
  const warnings = [];
  const lines = content.split(/\r?\n/);

  lines.forEach((line, index) => {
    const listPermissions = line.match(LIST_PERMISSION_PATTERN) || [];
    if (listPermissions.length === 0) {
      return;
    }

    const context = lineContext(lines, index);
    const hasDirectPermissionCheck = /\bhasPerm\s*\(|\bcan\s*\(|\bcanAccess\s*\(/.test(context);
    const hasPermissionProp = /\bpermission\s*:\s*['"][^'"]+:list['"]/.test(line);
    const actionContext = hasActionToken(context);
    const isReadGatingVariable =
      /\bconst\s+can(?:View|Open|Access|Read|List|Show)[A-Z]\w*\s*=/.test(line) ||
      /\bconst\s+has(?:View|Open|Access|Read|List|Show)[A-Z]\w*\s*=/.test(line);

    if (hasDirectPermissionCheck && actionContext && !isReadGatingVariable) {
      warnings.push({
        file: toRepoPath(filePath, root),
        line: index + 1,
        permission: Array.from(new Set(listPermissions)).join(', '),
        reason: 'direct permission check uses list permission near action wording; confirm this is read/navigation gating, not action gating',
      });
      return;
    }

    if (hasPermissionProp && actionContext) {
      warnings.push({
        file: toRepoPath(filePath, root),
        line: index + 1,
        permission: Array.from(new Set(listPermissions)).join(', '),
        reason: 'permission property uses list permission near action wording; confirm this is page/navigation gating, not action gating',
      });
    }
  });

  return { findings, warnings };
}

function lineNumberAt(content, index) {
  return content.slice(0, index).split(/\r?\n/).length;
}

function scanRepository(repoName, root) {
  const repoRoot = path.join(root, repoName);
  const findings = [];
  const warnings = [];
  const scanWarnings = [];

  if (!fs.existsSync(repoRoot)) {
    scanWarnings.push(`Repository root not found: ${repoName}`);
    return { repo: repoName, findings, warnings, scanWarnings };
  }

  const backendRoot = path.join(repoRoot, 'backend');
  const frontendRoot = path.join(repoRoot, 'frontend', 'src');

  for (const filePath of walkFiles(backendRoot, ['.go'])) {
    if (filePath.endsWith('_test.go')) {
      continue;
    }
    const result = scanGoFile(filePath, root);
    findings.push(...result.findings);
    warnings.push(...result.warnings);
  }

  for (const filePath of walkFiles(frontendRoot, ['.ts', '.tsx'])) {
    const result = scanTsFile(filePath, root);
    findings.push(...result.findings);
    warnings.push(...result.warnings);
  }

  return { repo: repoName, findings, warnings, scanWarnings };
}

function printTextReport(results, strict) {
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const scanWarningCount = results.reduce((count, result) => count + result.scanWarnings.length, 0);
  const mode = strict ? 'strict' : 'report-only';

  console.log(
    `Permission contract check (${mode}): ${findingCount} finding(s), ${warningCount} warning(s), ${scanWarningCount} scan warning(s)`,
  );

  for (const result of results) {
    console.log(`\n${result.repo}`);

    if (result.findings.length === 0 && result.warnings.length === 0 && result.scanWarnings.length === 0) {
      console.log('  no findings');
    }

    for (const finding of result.findings) {
      console.log(`  finding: ${finding.file}:${finding.line}`);
      console.log(`    permission: ${finding.permission}`);
      console.log(`    reason: ${finding.reason}`);
    }

    for (const warning of result.warnings) {
      console.log(`  warning: ${warning.file}:${warning.line}`);
      console.log(`    permission: ${warning.permission}`);
      console.log(`    reason: ${warning.reason}`);
    }

    for (const warning of result.scanWarnings) {
      console.log(`  scan warning: ${warning}`);
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

  const results = REPOSITORIES.map((repo) => scanRepository(repo, options.root));
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const scanWarningCount = results.reduce((count, result) => count + result.scanWarnings.length, 0);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          findingCount,
          warningCount,
          scanWarningCount,
          results,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(results, options.strict);
  }

  return options.strict && findingCount > 0 ? 1 : 0;
}

process.exitCode = main();
