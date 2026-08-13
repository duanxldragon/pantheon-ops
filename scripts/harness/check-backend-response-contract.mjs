#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();
const REPOSITORIES = ['pantheon-base', 'pantheon-ops'];
const ALLOWED_RESPONSE_WRAPPER = 'backend/pkg/common/response.go';
const DIRECT_JSON_PATTERN = /\bc\.JSON\s*\(/;

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-backend-response-contract.mjs [--json] [--strict] [--root <path>]

Default behavior:
  Report direct c.JSON findings and exit 0. Use --strict to exit 1 when findings exist.

Examples:
  node scripts/harness/check-backend-response-contract.mjs
  node scripts/harness/check-backend-response-contract.mjs --json
  node scripts/harness/check-backend-response-contract.mjs --strict
  node scripts/harness/check-backend-response-contract.mjs --root /tmp/fixture`);
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

function walkGoFiles(rootDir) {
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
      } else if (entry.name.endsWith('.go') && !entry.name.endsWith('_test.go')) {
        files.push(entryPath);
      }
    }
  }

  return sortStrings(files);
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function isAllowedWrapper(repoName, filePath, root) {
  const repoRelative = path.relative(path.join(root, repoName), filePath).replaceAll(path.sep, '/');
  return repoRelative === ALLOWED_RESPONSE_WRAPPER;
}

function scanGoFile(filePath, root) {
  const content = fs.readFileSync(filePath, 'utf8');
  const findings = [];
  const lines = content.split(/\r?\n/);

  lines.forEach((line, index) => {
    if (!DIRECT_JSON_PATTERN.test(line)) {
      return;
    }

    findings.push({
      file: toRepoPath(filePath, root),
      line: index + 1,
      snippet: line.trim(),
      reason: 'direct Gin JSON response; use pkg/common response helpers instead',
    });
  });

  return findings;
}

function scanRepository(repoName, root) {
  const warnings = [];
  const findings = [];
  const backendRoot = path.join(root, repoName, 'backend');

  if (!fs.existsSync(backendRoot)) {
    warnings.push(`Backend root not found: ${toRepoPath(backendRoot, root)}`);
    return { repo: repoName, findings, warnings };
  }

  for (const filePath of walkGoFiles(backendRoot)) {
    if (isAllowedWrapper(repoName, filePath, root)) {
      continue;
    }

    findings.push(...scanGoFile(filePath, root));
  }

  return { repo: repoName, findings, warnings };
}

function printTextReport(results, strict) {
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const mode = strict ? 'strict' : 'report-only';

  console.log(`Backend response contract check (${mode}): ${findingCount} finding(s), ${warningCount} warning(s)`);

  for (const result of results) {
    console.log(`\n${result.repo}`);

    if (result.findings.length === 0) {
      console.log('  no findings');
    }

    for (const finding of result.findings) {
      console.log(`  finding: ${finding.file}:${finding.line}`);
      console.log(`    snippet: ${finding.snippet}`);
      console.log(`    reason: ${finding.reason}`);
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

  const results = REPOSITORIES.map((repo) => scanRepository(repo, options.root));
  const findingCount = results.reduce((count, result) => count + result.findings.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          findingCount,
          warningCount,
          allowedWrapper: ALLOWED_RESPONSE_WRAPPER,
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
