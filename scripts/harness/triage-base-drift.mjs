#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_BASE = 'pantheon-base';
const DEFAULT_BUSINESS = 'pantheon-ops';
const CATEGORY_NAMES = [
  'pseudo-drift',
  'business mount',
  'generic drift',
  'business-specific drift',
  'noise',
  'base-only',
  'business-only',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/triage-base-drift.mjs [--json] [--root <path>] [--base <repo>] [--business <repo>]

Default behavior:
  Scan shared files between pantheon-base and pantheon-ops and classify drift without modifying code.

Examples:
  node scripts/harness/triage-base-drift.mjs
  node scripts/harness/triage-base-drift.mjs --json
  node scripts/harness/triage-base-drift.mjs --business pantheon-ops`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    help: false,
    root: DEFAULT_ROOT,
    base: DEFAULT_BASE,
    business: DEFAULT_BUSINESS,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else if (arg === '--base') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--base requires a repository name');
      }
      options.base = value;
      index += 1;
    } else if (arg === '--business') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--business requires a repository name');
      }
      options.business = value;
      index += 1;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function walkFiles(rootDir) {
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
        if (shouldSkipDirectory(entry.name)) {
          continue;
        }
        stack.push(entryPath);
      } else {
        files.push(entryPath);
      }
    }
  }

  return sortStrings(files);
}

function relativeUnix(root, filePath) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function readText(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function isTextFile(filePath) {
  const ext = path.extname(filePath).toLowerCase();
  return [
    '.go',
    '.ts',
    '.tsx',
    '.js',
    '.jsx',
    '.json',
    '.md',
    '.yaml',
    '.yml',
    '.sql',
    '.txt',
    '.css',
    '.scss',
    '.html',
  ].includes(ext);
}

function isNoisePath(relativePath) {
  return (
    /(^|\/)generated(\/|$)/.test(relativePath) ||
    /(^|\/)\.gocache(\/|$)/.test(relativePath) ||
    /(^|\/)\.tmp(\/|$)/.test(relativePath) ||
    /(^|\/)uploads(\/|$)/.test(relativePath) ||
    /(^|\/)vendor(\/|$)/.test(relativePath) ||
    /(^|\/)node_modules(\/|$)/.test(relativePath) ||
    /(^|\/)dist(\/|$)/.test(relativePath) ||
    /(^|\/)build(\/|$)/.test(relativePath) ||
    relativePath.endsWith('.sum') ||
    relativePath.endsWith('.lock')
  );
}

function shouldSkipDirectory(name) {
  return (
    ['.git', '.gocache', '.tmp', 'node_modules', 'dist', 'build', 'vendor', 'uploads', 'tmp', 'test-results'].includes(
      name,
    ) || name.startsWith('.tmp')
  );
}

function normalizeModuleName(content, options) {
  const baseBackend = `${options.base}/backend`;
  const businessBackend = `${options.business}/backend`;
  return content
    .replaceAll('pantheon-base/backend', 'MODNAME/backend')
    .replaceAll(`${options.base}/backend`, 'MODNAME/backend')
    .replaceAll(baseBackend, 'MODNAME/backend')
    .replaceAll(businessBackend, 'MODNAME/backend');
}

function diffLineCount(a, b) {
  const left = a.split(/\r?\n/);
  const right = b.split(/\r?\n/);
  const max = Math.max(left.length, right.length);
  let count = 0;

  for (let index = 0; index < max; index += 1) {
    if (left[index] !== right[index]) {
      count += 1;
    }
  }

  return count;
}

function classifyShared(relativePath, baseContent, businessContent, options) {
  if (isNoisePath(relativePath)) {
    return 'noise';
  }

  if (/^(AGENTS|CLAUDE|DESIGN|README|agent)\.md$/.test(relativePath)) {
    return 'business-specific drift';
  }

  const isBackend = relativePath.startsWith('backend/');
  const normalizedBase = isBackend ? normalizeModuleName(baseContent, options) : baseContent;
  const normalizedBusiness = isBackend ? normalizeModuleName(businessContent, options) : businessContent;

  if (normalizedBase === normalizedBusiness) {
    return 'pseudo-drift';
  }

  if (/^backend\/modules\/business\/(?:business|module|routes?)\.go$/.test(relativePath)) {
    return 'business mount';
  }

  if (
    /^backend\/modules\/business\//.test(relativePath) ||
    /^frontend\/src\/modules\/business\//.test(relativePath) ||
    /\b(cmdb|deploy|biz_|business:)/i.test(businessContent)
  ) {
    return 'business-specific drift';
  }

  return 'generic drift';
}

function createRecord(relativePath, category, basePath, businessPath, options) {
  let diffLines = 0;

  if (basePath && businessPath && fs.existsSync(basePath) && fs.existsSync(businessPath) && isTextFile(basePath)) {
    const baseContent = readText(basePath);
    const businessContent = readText(businessPath);
    const normalizedBase = relativePath.startsWith('backend/') ? normalizeModuleName(baseContent, options) : baseContent;
    const normalizedBusiness = relativePath.startsWith('backend/')
      ? normalizeModuleName(businessContent, options)
      : businessContent;
    diffLines = diffLineCount(normalizedBase, normalizedBusiness);
  }

  return {
    path: relativePath,
    category,
    diffLines,
    action: recommendedAction(category),
  };
}

function recommendedAction(category) {
  switch (category) {
    case 'pseudo-drift':
      return 'delete or collapse once shared workspace/go.work ownership is ready';
    case 'business mount':
      return 'keep as business repo mount or replace with an explicit extension API';
    case 'generic drift':
      return 'review for backport to base';
    case 'business-specific drift':
      return 'keep in business domain or expose a base extension contract';
    case 'noise':
      return 'exclude from drift decisions';
    case 'base-only':
      return 'review whether business repo needs the base file';
    case 'business-only':
      return 'review whether this is intentional business extension';
    default:
      return 'review manually';
  }
}

function scan(options) {
  const baseRoot = path.join(options.root, options.base);
  const businessRoot = path.join(options.root, options.business);
  const warnings = [];
  const records = [];
  const categoryCounts = Object.fromEntries(CATEGORY_NAMES.map((category) => [category, 0]));

  if (!fs.existsSync(baseRoot)) {
    warnings.push(`Base repository root not found: ${options.base}`);
  }
  if (!fs.existsSync(businessRoot)) {
    warnings.push(`Business repository root not found: ${options.business}`);
  }
  if (warnings.length > 0) {
    return { records, categoryCounts, warnings };
  }

  const baseFiles = new Map(walkFiles(baseRoot).map((filePath) => [relativeUnix(baseRoot, filePath), filePath]));
  const businessFiles = new Map(walkFiles(businessRoot).map((filePath) => [relativeUnix(businessRoot, filePath), filePath]));
  const allPaths = sortStrings(Array.from(new Set([...baseFiles.keys(), ...businessFiles.keys()])));

  for (const relativePath of allPaths) {
    const basePath = baseFiles.get(relativePath);
    const businessPath = businessFiles.get(relativePath);
    let category;

    if (!basePath) {
      category = isNoisePath(relativePath) ? 'noise' : 'business-only';
    } else if (!businessPath) {
      category = isNoisePath(relativePath) ? 'noise' : 'base-only';
    } else if (!isTextFile(basePath) || !isTextFile(businessPath)) {
      category = 'noise';
    } else {
      const baseContent = readText(basePath);
      const businessContent = readText(businessPath);
      if (baseContent === businessContent) {
        continue;
      }
      category = classifyShared(relativePath, baseContent, businessContent, options);
    }

    const record = createRecord(relativePath, category, basePath, businessPath, options);
    records.push(record);
    categoryCounts[category] += 1;
  }

  return { records, categoryCounts, warnings };
}

function printTextReport(report) {
  console.log(`Base drift triage: ${report.records.length} drift record(s), ${report.warnings.length} warning(s)`);
  console.log(`base: ${report.base}`);
  console.log(`business: ${report.business}`);

  for (const category of CATEGORY_NAMES) {
    console.log(`\n${category}: ${report.categoryCounts[category] || 0}`);
    for (const record of report.records.filter((item) => item.category === category).slice(0, 50)) {
      console.log(`  ${record.path} (${record.diffLines} diff line(s))`);
      console.log(`    action: ${record.action}`);
    }
  }

  for (const warning of report.warnings) {
    console.log(`\nwarning: ${warning}`);
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

  const scanResult = scan(options);
  const report = {
    root: options.root,
    base: options.base,
    business: options.business,
    categoryCounts: scanResult.categoryCounts,
    warningCount: scanResult.warnings.length,
    warnings: scanResult.warnings,
    records: scanResult.records,
  };

  if (options.json) {
    console.log(JSON.stringify(report, null, 2));
  } else {
    printTextReport(report);
  }

  return 0;
}

process.exitCode = main();
