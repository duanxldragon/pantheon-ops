#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';

const DEFAULT_ROOT = process.cwd();

const REQUIRED_COLUMNS = [
  'Failure ID',
  'Category',
  'Failure Class',
  'Owner Layer',
  'Occurrences',
  'Example',
  'Impact',
  'GitHub Signal',
  'Current Guide',
  'Current Sensor',
  'Current Gate',
  'Detected By',
  'Missed By',
  'Recommended Harness Change',
  'Promotion Decision',
  'Promotion Deadline',
  'Status',
];

const REQUIRED_VALUE_COLUMNS = [
  'Failure ID',
  'Category',
  'Failure Class',
  'Owner Layer',
  'Occurrences',
  'Example',
  'Impact',
  'GitHub Signal',
  'Recommended Harness Change',
  'Promotion Decision',
  'Promotion Deadline',
  'Status',
];

const VALID_CATEGORIES = new Set([
  'behaviour',
  'maintainability',
  'architecture-fitness',
  'runtime-quality',
  'method-health',
]);

const VALID_FAILURE_CLASSES = new Set([
  'instruction-gap',
  'task-boundary-gap',
  'architecture-drift',
  'test-gap',
  'static-sensor-gap',
  'runtime-evidence-gap',
  'security-boundary-gap',
  'ci-signal-noise',
  'method-health-gap',
]);

const VALID_OWNER_LAYERS = new Set([
  'portable-method',
  'consumer-template',
  'consumer-repository',
  'agent-adapter',
  'no-action',
]);

const VALID_GITHUB_SIGNALS = new Set([
  'method-gate',
  'repo-quality-gate',
  'runtime-evidence-gate',
  'external-flaky',
  'not-applicable',
]);

const VALID_HARNESS_CHANGES = new Set(['guide', 'sensor', 'gate', 'template', 'adapter', 'registry-only', 'no-action']);
const VALID_PROMOTION_DECISIONS = new Set([
  'no-repeat-observed',
  'guide-updated',
  'sensor-added',
  'gate-updated',
  'template-updated',
  'adapter-updated',
  'registry-only',
]);
const VALID_STATUSES = new Set(['open', 'accepted', 'implemented', 'rejected']);

const DEFAULT_FILES = [
  'docs/harness/failure-registry.md',
  'docs/harness/failures.md',
  'docs/harness/failure-registry/index.md',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-failure-registry.mjs [--json] [--strict] [--root <path>] [registry-file ...]

Defaults:
  Scans common registry paths such as <root>/docs/harness/failure-registry.md.
  Missing default registries are reported as warnings, not errors.

Examples:
  node scripts/harness/check-failure-registry.mjs
  node scripts/harness/check-failure-registry.mjs --json
  node scripts/harness/check-failure-registry.mjs --root /tmp/fixture
  node scripts/harness/check-failure-registry.mjs docs/harness/failure-registry.md`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    files: [],
    root: DEFAULT_ROOT,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else {
      options.files.push(arg);
    }
  }

  return options;
}

function normalizeInputFile(inputPath, root) {
  return path.isAbsolute(inputPath) ? inputPath : path.join(root, inputPath);
}

function discoverRegistryFiles(root) {
  const discovered = DEFAULT_FILES.map((repoPath) => path.join(root, repoPath)).filter((filePath) =>
    fs.existsSync(filePath),
  );
  const registryDir = path.join(root, 'docs', 'harness', 'failure-registry');
  if (fs.existsSync(registryDir) && fs.statSync(registryDir).isDirectory()) {
    for (const fileName of sortStrings(fs.readdirSync(registryDir))) {
      if (fileName.endsWith('.md')) {
        discovered.push(path.join(registryDir, fileName));
      }
    }
  }
  return Array.from(new Set(discovered));
}

function splitMarkdownTableRow(line) {
  const cells = [];
  let current = '';
  let escaped = false;

  for (const char of line.trim()) {
    if (escaped) {
      current += char;
      escaped = false;
      continue;
    }

    if (char === '\\') {
      escaped = true;
      continue;
    }

    if (char === '|') {
      cells.push(current.trim());
      current = '';
      continue;
    }

    current += char;
  }

  cells.push(current.trim());

  if (cells[0] === '') {
    cells.shift();
  }
  if (cells[cells.length - 1] === '') {
    cells.pop();
  }

  return cells.map((cell) => cell.replace(/<br\s*\/?>/gi, ' ').replace(/\s+/g, ' ').trim());
}

function isSeparatorRow(cells) {
  return cells.length > 0 && cells.every((cell) => /^:?-{3,}:?$/.test(cell.trim()));
}

function findRegistryTable(content) {
  const lines = content.split(/\r?\n/);

  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (!line.trim().startsWith('|')) {
      continue;
    }

    const header = splitMarkdownTableRow(line);
    if (!header.includes('Failure ID') || !header.includes('Category')) {
      continue;
    }

    const rows = [];
    for (let rowIndex = index + 1; rowIndex < lines.length; rowIndex += 1) {
      const rowLine = lines[rowIndex];
      if (!rowLine.trim().startsWith('|')) {
        break;
      }
      const cells = splitMarkdownTableRow(rowLine);
      if (!isSeparatorRow(cells)) {
        rows.push({ line: rowIndex + 1, cells });
      }
    }

    return { header, rows };
  }

  return null;
}

function stripBackticks(value) {
  return value.trim().replace(/^`+/, '').replace(/`+$/, '').trim();
}

function isBlankish(value) {
  const normalized = stripBackticks(value).toLowerCase();
  return normalized === '' || normalized === '-' || normalized === 'tbd' || normalized === 'todo';
}

function rowToEntry(header, row) {
  const entry = {};
  for (let index = 0; index < header.length; index += 1) {
    entry[header[index]] = row.cells[index] ?? '';
  }
  return entry;
}

function validateEnum(result, entry, column, values, line) {
  const value = stripBackticks(entry[column] ?? '');
  if (!values.has(value)) {
    result.errors.push(
      `Line ${line}: Invalid ${column} "${value}". Expected one of: ${Array.from(values).join(', ')}.`,
    );
  }
}

function validateOccurrences(result, entry, line) {
  const value = stripBackticks(entry.Occurrences ?? '');
  const occurrences = Number.parseInt(value, 10);
  if (!/^\d+$/.test(value) || occurrences < 1) {
    result.errors.push(`Line ${line}: Occurrences "${value}" must be an integer >= 1.`);
  }
}

function isTemplatePlaceholder(entry) {
  return (
    stripBackticks(entry['Failure ID'] ?? '') === 'FR-001' &&
    (entry.Category ?? '').includes('|') &&
    (entry['Failure Class'] ?? '').includes('|') &&
    (entry['Owner Layer'] ?? '').includes('|') &&
    (entry['GitHub Signal'] ?? '').includes('|') &&
    (entry['Recommended Harness Change'] ?? '').includes('/') &&
    (entry['Promotion Decision'] ?? '').includes('|') &&
    (entry.Status ?? '').includes('|')
  );
}

function validateFailureRegistry(filePath, root) {
  const relativePath = path.relative(root, filePath).replaceAll(path.sep, '/');
  const result = {
    file: relativePath,
    errors: [],
    warnings: [],
  };

  if (!fs.existsSync(filePath)) {
    result.errors.push('File does not exist.');
    return result;
  }

  const content = fs.readFileSync(filePath, 'utf8');
  const table = findRegistryTable(content);
  if (!table) {
    result.errors.push('Missing registry table with "Failure ID" and "Category" columns.');
    return result;
  }

  for (const column of REQUIRED_COLUMNS) {
    if (!table.header.includes(column)) {
      result.errors.push(`Missing required column "${column}".`);
    }
  }

  if (result.errors.length > 0) {
    return result;
  }

  if (table.rows.length === 0) {
    result.warnings.push('Registry table has no failure rows.');
    return result;
  }

  for (const row of table.rows) {
    const entry = rowToEntry(table.header, row);

    if (isTemplatePlaceholder(entry)) {
      result.errors.push(`Line ${row.line}: Row appears to be the template placeholder row; replace it with a real failure.`);
      continue;
    }

    for (const column of REQUIRED_VALUE_COLUMNS) {
      if (isBlankish(entry[column] ?? '')) {
        result.errors.push(`Line ${row.line}: Required value "${column}" is empty.`);
      }
    }

    const failureId = stripBackticks(entry['Failure ID'] ?? '');
    if (failureId && !/^FR-\d{3,}$/.test(failureId)) {
      result.errors.push(`Line ${row.line}: Failure ID "${failureId}" must match FR-001 style.`);
    }

    validateEnum(result, entry, 'Category', VALID_CATEGORIES, row.line);
    validateEnum(result, entry, 'Failure Class', VALID_FAILURE_CLASSES, row.line);
    validateEnum(result, entry, 'Owner Layer', VALID_OWNER_LAYERS, row.line);
    validateOccurrences(result, entry, row.line);
    validateEnum(result, entry, 'GitHub Signal', VALID_GITHUB_SIGNALS, row.line);
    validateEnum(result, entry, 'Recommended Harness Change', VALID_HARNESS_CHANGES, row.line);
    validateEnum(result, entry, 'Promotion Decision', VALID_PROMOTION_DECISIONS, row.line);
    validateEnum(result, entry, 'Status', VALID_STATUSES, row.line);
  }

  return result;
}

function printTextReport(results) {
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  console.log(`Failure registry check: ${results.length} file(s), ${errorCount} error(s), ${warningCount} warning(s)`);

  for (const result of results) {
    const status = result.errors.length > 0 ? 'FAIL' : 'PASS';
    console.log(`\n[${status}] ${result.file}`);

    for (const error of result.errors) {
      console.log(`  error: ${error}`);
    }

    for (const warning of result.warnings) {
      console.log(`  warning: ${warning}`);
    }
  }
}

function buildMissingDefaultResult(root) {
  return {
    file: path.relative(root, path.join(root, 'docs', 'harness', 'failure-registry.md')).replaceAll(path.sep, '/'),
    errors: [],
    warnings: ['No failure registry files found in default locations.'],
  };
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

  const files =
    options.files.length > 0
      ? options.files.map((file) => normalizeInputFile(file, options.root))
      : discoverRegistryFiles(options.root);

  const results =
    files.length > 0 ? files.map((file) => validateFailureRegistry(file, options.root)) : [buildMissingDefaultResult(options.root)];

  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  if (options.json) {
    console.log(JSON.stringify({ results, errorCount, warningCount }, null, 2));
  } else {
    printTextReport(results);
  }

  return errorCount > 0 ? 1 : 0;
}

process.exitCode = main();
