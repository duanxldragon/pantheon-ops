#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const DEFAULT_ROOT = process.cwd();
const GENERATED_SCHEMA_ROOT = 'schema/generated';
const FEATURE_LEDGER_PATH = 'schema/generated/feature-ledger.json';
const VALID_SCOPES = new Set(['system', 'business']);
const REQUIRED_ENTRY_FIELDS = [
  'moduleKey',
  'name',
  'scope',
  'displayName',
  'owner',
  'boundedContext',
  'sourceMode',
  'source',
  'maturity',
  'status',
  'tableName',
  'schemaPath',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-feature-ledger.mjs [--json] [--strict] [--root <path>]

Checks:
- generated schema files and feature ledger snapshot can be parsed
- feature ledger entries match schema/generated module files one-to-one
- feature ledger entries stay sorted and include the required fields
- strict mode exits non-zero when drift is detected`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
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
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function relative(root, filePath) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function discoverGeneratedSchemaFiles(root) {
  const schemaRoot = path.join(root, GENERATED_SCHEMA_ROOT);
  if (!fs.existsSync(schemaRoot)) {
    return [];
  }

  const files = [];
  const walk = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) {
        walk(entryPath);
        continue;
      }
      if (!entry.isFile() || !entry.name.endsWith('.json')) {
        continue;
      }
      if (entry.name === path.basename(FEATURE_LEDGER_PATH)) {
        continue;
      }
      const relativePath = relative(schemaRoot, entryPath);
      const descriptor = splitSchemaRelativePath(relativePath);
      if (!descriptor) {
        continue;
      }
      files.push({
        file: entryPath,
        relativePath,
        ...descriptor,
      });
    }
  };

  walk(schemaRoot);
  return files.sort((left, right) => left.relativePath.localeCompare(right.relativePath));
}

function splitSchemaRelativePath(relativePath) {
  const normalized = relativePath.replaceAll('\\', '/').replace(/^\/+/, '');
  if (normalized === '') {
    return null;
  }

  const parts = normalized.split('/');
  if (parts.length < 2) {
    return null;
  }

  const scope = parts[0].trim();
  if (!VALID_SCOPES.has(scope)) {
    return null;
  }

  const name = parts.slice(1).join('/').replace(/\.json$/i, '').trim();
  if (name === '') {
    return null;
  }

  return {
    scope,
    name,
    moduleKey: `${scope}.${name}`,
  };
}

function safeJsonParse(filePath, findings, code, fallbackDetail) {
  if (!fs.existsSync(filePath)) {
    return { ok: false, value: null };
  }

  try {
    return { ok: true, value: JSON.parse(fs.readFileSync(filePath, 'utf8')) };
  } catch (error) {
    findings.push({
      code,
      severity: 'error',
      file: filePath,
      detail: fallbackDetail ?? error.message,
    });
    return { ok: false, value: null };
  }
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function getString(value) {
  return typeof value === 'string' ? value.trim() : '';
}

function compareLedgerEntry(left, right) {
  if (left.scope === right.scope) {
    return left.name.localeCompare(right.name);
  }
  return left.scope.localeCompare(right.scope);
}

function buildRequiredFieldFinding(moduleKey, field, detail, file) {
  return {
    code: 'ledger_required_field_missing',
    severity: 'error',
    moduleKey,
    field,
    file,
    detail,
  };
}

function buildLedgerFinding(code, moduleKey, detail, file, field, severity = 'error') {
  const finding = {
    code,
    severity,
    file,
    detail,
  };
  if (moduleKey) {
    finding.moduleKey = moduleKey;
  }
  if (field) {
    finding.field = field;
  }
  return finding;
}

function validateEntryFields(entry, moduleKey, snapshotPath, findings) {
  for (const field of REQUIRED_ENTRY_FIELDS) {
    if (field === 'moduleKey') {
      if (getString(entry.moduleKey) !== moduleKey) {
        findings.push(
          buildLedgerFinding(
            'ledger_entry_key_mismatch',
            moduleKey,
            `entry.moduleKey "${getString(entry.moduleKey)}" does not match schema module "${moduleKey}"`,
            snapshotPath,
            'moduleKey',
          ),
        );
      }
      continue;
    }

    if (field === 'schemaPath') {
      if (getString(entry.schemaPath) === '') {
        findings.push(
          buildRequiredFieldFinding(
            moduleKey,
            field,
            'feature ledger entry is missing schemaPath',
            snapshotPath,
          ),
        );
      }
      continue;
    }

    if (getString(entry[field]) === '') {
      findings.push(
        buildRequiredFieldFinding(
          moduleKey,
          field,
          `feature ledger entry is missing required field "${field}"`,
          snapshotPath,
        ),
      );
    }
  }
}

function validateSnapshotSourceOfTruth(snapshot, snapshotPath, findings) {
  if (!isObject(snapshot.sourceOfTruth)) {
    findings.push(
      buildLedgerFinding(
        'ledger_snapshot_invalid',
        null,
        'feature ledger snapshot is missing sourceOfTruth metadata',
        snapshotPath,
      ),
    );
    return;
  }

  const { registrations, schemas, snapshot: snapshotRef } = snapshot.sourceOfTruth;
  if (getString(registrations) !== 'system_module_registration') {
    findings.push(
      buildLedgerFinding(
        'ledger_source_of_truth_invalid',
        null,
        'feature ledger sourceOfTruth.registrations must point at system_module_registration',
        snapshotPath,
        'registrations',
      ),
    );
  }
  if (getString(schemas) !== GENERATED_SCHEMA_ROOT) {
    findings.push(
      buildLedgerFinding(
        'ledger_source_of_truth_invalid',
        null,
        `feature ledger sourceOfTruth.schemas must be "${GENERATED_SCHEMA_ROOT}"`,
        snapshotPath,
        'schemas',
      ),
    );
  }
  if (getString(snapshotRef) !== FEATURE_LEDGER_PATH) {
    findings.push(
      buildLedgerFinding(
        'ledger_source_of_truth_invalid',
        null,
        `feature ledger sourceOfTruth.snapshot must be "${FEATURE_LEDGER_PATH}"`,
        snapshotPath,
        'snapshot',
      ),
    );
  }
}

function scanFeatureLedger(root) {
  const findings = [];
  const warnings = [];
  const schemaFiles = discoverGeneratedSchemaFiles(root);
  const snapshotPath = path.join(root, FEATURE_LEDGER_PATH);
  const snapshotExists = fs.existsSync(snapshotPath);
  const snapshotParse = snapshotExists
    ? safeJsonParse(snapshotPath, findings, 'ledger_snapshot_invalid')
    : { ok: false, value: null };
  const snapshot = snapshotParse.value;
  const buildResult = (entryCount, issueCount) => {
    findings.sort((left, right) => {
      if (left.code !== right.code) {
        return left.code.localeCompare(right.code);
      }
      if ((left.moduleKey ?? '') !== (right.moduleKey ?? '')) {
        return (left.moduleKey ?? '').localeCompare(right.moduleKey ?? '');
      }
      if ((left.field ?? '') !== (right.field ?? '')) {
        return (left.field ?? '').localeCompare(right.field ?? '');
      }
      return (left.detail ?? '').localeCompare(right.detail ?? '');
    });

    warnings.sort((left, right) => {
      const leftText = typeof left === 'string' ? left : JSON.stringify(left);
      const rightText = typeof right === 'string' ? right : JSON.stringify(right);
      return leftText.localeCompare(rightText);
    });

    return {
      findings,
      warnings,
      schemaCount: schemaFiles.length,
      entryCount,
      issueCount,
      snapshotPath: relative(root, snapshotPath),
    };
  };

  const schemaModules = new Map();
  for (const file of schemaFiles) {
    const parsedSchema = safeJsonParse(
      file.file,
      findings,
      'ledger_schema_invalid',
      `invalid JSON: ${relative(root, file.file)}`,
    );
    if (!parsedSchema.ok) {
      continue;
    }
    const schema = parsedSchema.value;
    if (!isObject(schema)) {
      findings.push(
        buildLedgerFinding(
          'ledger_schema_invalid',
          file.moduleKey,
          'generated schema file must contain a JSON object',
          file.file,
        ),
      );
      continue;
    }

    const schemaName = getString(schema.name);
    const schemaScope = getString(schema.scope);
    if (schemaName === '' || schemaScope === '') {
      findings.push(
        buildLedgerFinding(
          'ledger_schema_invalid',
          file.moduleKey,
          'generated schema file must include non-empty name and scope fields',
          file.file,
        ),
      );
      continue;
    }

    if (schemaScope !== file.scope || schemaName !== file.name) {
      findings.push(
        buildLedgerFinding(
          'ledger_schema_mismatch',
          file.moduleKey,
          `schema file path "${file.relativePath}" does not match schema.name/scope "${schemaScope}.${schemaName}"`,
          file.file,
        ),
      );
    }

    schemaModules.set(file.moduleKey, {
      ...file,
      schema,
    });
  }

  let entries = [];
  let issues = [];
  if (!snapshotExists) {
    if (schemaFiles.length > 0) {
      findings.push(
        buildLedgerFinding(
          'ledger_snapshot_missing',
          null,
          `feature ledger snapshot is missing at ${FEATURE_LEDGER_PATH}`,
          snapshotPath,
        ),
      );
    }
    return buildResult(0, 0);
  }

  if (!snapshotParse.ok) {
    return buildResult(0, 0);
  }

  if (!isObject(snapshot)) {
    findings.push(
      buildLedgerFinding(
        'ledger_snapshot_invalid',
        null,
        'feature ledger snapshot must be a JSON object',
        snapshotPath,
      ),
    );
    return buildResult(0, 0);
  }

  validateSnapshotSourceOfTruth(snapshot, snapshotPath, findings);

  if (!Array.isArray(snapshot.entries)) {
    findings.push(
      buildLedgerFinding(
        'ledger_snapshot_invalid',
        null,
        'feature ledger snapshot.entries must be an array',
        snapshotPath,
      ),
    );
    return buildResult(0, 0);
  }

  entries = snapshot.entries.filter((entry) => isObject(entry));
  if (entries.length !== snapshot.entries.length) {
    findings.push(
      buildLedgerFinding(
        'ledger_snapshot_invalid',
        null,
        'feature ledger snapshot entries must be objects',
        snapshotPath,
      ),
    );
  }

  if (Array.isArray(snapshot.issues)) {
    issues = snapshot.issues.filter((issue) => isObject(issue));
    if (issues.length !== snapshot.issues.length) {
      findings.push(
        buildLedgerFinding(
          'ledger_snapshot_invalid',
          null,
          'feature ledger snapshot issues must be objects',
          snapshotPath,
        ),
      );
    }
  } else if ('issues' in snapshot) {
    findings.push(
      buildLedgerFinding(
        'ledger_snapshot_invalid',
        null,
        'feature ledger snapshot issues must be an array when present',
        snapshotPath,
      ),
    );
  }

  const entryByModuleKey = new Map();
  for (const entry of entries) {
    const moduleKey = getString(entry.moduleKey);
    if (moduleKey === '') {
      findings.push(
        buildLedgerFinding(
          'ledger_entry_invalid',
          null,
          'feature ledger entry is missing moduleKey',
          snapshotPath,
          'moduleKey',
        ),
      );
      continue;
    }

    if (entryByModuleKey.has(moduleKey)) {
      findings.push(
        buildLedgerFinding(
          'ledger_entry_duplicate',
          moduleKey,
          'feature ledger snapshot contains duplicate moduleKey entries',
          snapshotPath,
        ),
      );
      continue;
    }

    entryByModuleKey.set(moduleKey, entry);
  }

  if (entries.length > 0) {
    const sortedEntries = [...entries].sort(compareLedgerEntry);
    const actualOrder = entries.map((entry) => getString(entry.moduleKey)).join('\n');
    const expectedOrder = sortedEntries.map((entry) => getString(entry.moduleKey)).join('\n');
    if (actualOrder !== expectedOrder) {
      findings.push(
        buildLedgerFinding(
          'ledger_entries_unsorted',
          null,
          'feature ledger entries must be sorted by scope then name',
          snapshotPath,
        ),
      );
    }
  }

  for (const [moduleKey, schemaFile] of schemaModules.entries()) {
    const entry = entryByModuleKey.get(moduleKey);
    if (!entry) {
      findings.push(
        buildLedgerFinding(
          'ledger_entry_missing',
          moduleKey,
          `feature ledger snapshot is missing an entry for ${moduleKey}`,
          snapshotPath,
        ),
      );
      continue;
    }

    validateEntryFields(entry, moduleKey, snapshotPath, findings);

    if (getString(entry.schemaPath) && getString(entry.schemaPath) !== schemaFile.relativePath) {
      findings.push(
        buildLedgerFinding(
          'ledger_schema_path_mismatch',
          moduleKey,
          `feature ledger schemaPath "${getString(entry.schemaPath)}" does not match "${schemaFile.relativePath}"`,
          snapshotPath,
          'schemaPath',
        ),
      );
    }

    if (getString(entry.scope) && getString(entry.scope) !== schemaFile.scope) {
      findings.push(
        buildLedgerFinding(
          'ledger_entry_scope_mismatch',
          moduleKey,
          `feature ledger entry scope "${getString(entry.scope)}" does not match schema scope "${schemaFile.scope}"`,
          snapshotPath,
          'scope',
        ),
      );
    }

    if (getString(entry.name) && getString(entry.name) !== schemaFile.name) {
      findings.push(
        buildLedgerFinding(
          'ledger_entry_name_mismatch',
          moduleKey,
          `feature ledger entry name "${getString(entry.name)}" does not match schema name "${schemaFile.name}"`,
          snapshotPath,
          'name',
        ),
      );
    }
  }

  for (const [moduleKey, entry] of entryByModuleKey.entries()) {
    if (!schemaModules.has(moduleKey)) {
      findings.push(
        buildLedgerFinding(
          'ledger_entry_orphaned',
          moduleKey,
          `feature ledger contains an entry for ${moduleKey} without a matching generated schema`,
          snapshotPath,
        ),
      );
    }
  }

  if (Array.isArray(issues)) {
    for (const issue of issues) {
      const moduleKey = getString(issue.moduleKey);
      const detail = `${getString(issue.code) || 'issue'}${getString(issue.field) ? `:${getString(issue.field)}` : ''}`;
      findings.push(
        buildLedgerFinding(
          'ledger_issue_open',
          moduleKey || null,
          detail,
          snapshotPath,
          getString(issue.field) || undefined,
          getString(issue.severity) || 'warn',
        ),
      );
    }
  }

  return buildResult(entries.length, issues.length);
}

function printTextReport(result, strict) {
  const mode = strict ? 'strict' : 'report-only';
  console.log(
    `Feature ledger check (${mode}): ${result.findings.length} finding(s), ${result.warnings.length} warning(s)`,
  );
  console.log(`schema count: ${result.schemaCount}`);
  console.log(`entry count: ${result.entryCount}`);
  console.log(`issue count: ${result.issueCount}`);

  if (result.findings.length === 0 && result.warnings.length === 0) {
    console.log('\nno findings');
  }

  for (const finding of result.findings) {
    console.log(`\nfinding: ${finding.code}`);
    if (finding.moduleKey) {
      console.log(`  moduleKey: ${finding.moduleKey}`);
    }
    if (finding.field) {
      console.log(`  field: ${finding.field}`);
    }
    console.log(`  reason: ${finding.detail}`);
  }

  for (const warning of result.warnings) {
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

  const result = scanFeatureLedger(options.root);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          findingCount: result.findings.length,
          warningCount: result.warnings.length,
          schemaCount: result.schemaCount,
          entryCount: result.entryCount,
          issueCount: result.issueCount,
          snapshotPath: result.snapshotPath,
          findings: result.findings,
          warnings: result.warnings,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(result, options.strict);
  }

  return options.strict && result.findings.length > 0 ? 1 : 0;
}

process.exitCode = main();

export { discoverGeneratedSchemaFiles, scanFeatureLedger, splitSchemaRelativePath };
