#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';
import { readTaskManifest, TASK_ARCHIVE_ROOT } from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_LEGACY_CONFIG = 'config/runtime-evidence-legacy-artifacts.json';

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT, legacyConfig: DEFAULT_LEGACY_CONFIG };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') options.json = true;
    else if (arg === '--strict') options.strict = true;
    else if (arg === '--help' || arg === '-h') options.help = true;
    else if (arg === '--legacy-config') {
      const value = argv[index + 1];
      if (!value) throw new Error('--legacy-config requires a path');
      options.legacyConfig = value;
      index += 1;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) throw new Error('--root requires a path');
      options.root = path.resolve(value);
      index += 1;
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-runtime-evidence.mjs [--json] [--strict] [--root <path>] [--legacy-config <path>]');
}

function discoverEvidenceFiles(root) {
  const files = [];
  const base = path.join(root, '.harness', 'evidence');
  function walk(dirPath) {
    if (!fs.existsSync(dirPath)) return;
    for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
      const fullPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) walk(fullPath);
      else if (entry.isFile() && entry.name === 'commands.json') files.push(fullPath);
    }
  }
  walk(base);
  // Evidence trees move to `.harness/archive/<YYYY-MM>/evidence/` when a round
  // closes; scanning only the live tree would drop every archived packet out of
  // this gate's scope the moment it is archived.
  const archiveRoot = path.join(root, TASK_ARCHIVE_ROOT);
  if (fs.existsSync(archiveRoot)) {
    for (const entry of fs.readdirSync(archiveRoot, { withFileTypes: true })) {
      if (entry.isDirectory() && /^\d{4}-\d{2}$/.test(entry.name)) {
        walk(path.join(archiveRoot, entry.name, 'evidence'));
      }
    }
  }
  return sortStrings(files);
}

function containsRuntimeSignal(value) {
  if (Array.isArray(value)) return value.length > 0;
  return typeof value === 'string' && value.trim() !== '';
}

// `verificationPlan.runtimeEvidence` was written by packet authors as "what will
// prove this round", so it also carries CI runs, grep output and explicit
// negations ("not runtime-sensitive: read-only inventory"). A boolean flag is
// the author's declaration and wins over that prose. Any explicit true keeps
// the requirement when packet and evidence flags disagree. The free-text list is
// consulted only when neither the evidence nor the packet declares one, and
// then only for entries that actually name a runtime artifact.
const RUNTIME_PLAN_ITEM =
  /\b(log|logs|metric|metrics|trace|traces|performance|latency|benchmark|profile|memory|cpu|smoke|browser|screenshot|p95|p99|uptime|restart)\b/i;
const RUNTIME_PLAN_NEGATION = /^\s*(not\s+runtime|none\b|n\/a\b|no\s+runtime)/i;

function scanFile(filePath, root) {
  const result = { file: path.relative(root, filePath).replaceAll(path.sep, '/'), warnings: [], errors: [] };
  let payload;
  try {
    payload = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    result.errors.push(`invalid JSON: ${error.message}`);
    return result;
  }

  const taskId = typeof payload.taskId === 'string' ? payload.taskId : path.basename(path.dirname(filePath));
  const taskManifestReference =
    typeof payload.linkage?.taskManifest === 'string' ? payload.linkage.taskManifest : null;

  let declaredSensitivity = null;
  if (typeof payload.runtimeSensitive === 'boolean') declaredSensitivity = payload.runtimeSensitive;
  let plannedRuntimeItems = [];
  if (taskManifestReference) {
    try {
      const taskManifest = readTaskManifest(root, taskManifestReference);
      const taskSensitivity = taskManifest.payload.runtimeSensitive;
      if (typeof taskSensitivity === 'boolean') {
        declaredSensitivity = declaredSensitivity === true || taskSensitivity;
      }
      plannedRuntimeItems = Array.isArray(taskManifest.payload.verificationPlan?.runtimeEvidence)
        ? taskManifest.payload.verificationPlan.runtimeEvidence
        : [];
    } catch {
      // ignore invalid manifests here; structural validators report that separately
    }
  }

  const runtimeSensitive =
    declaredSensitivity !== null
      ? declaredSensitivity
      : plannedRuntimeItems.some((item) => RUNTIME_PLAN_NEGATION.test(String(item)) === false && RUNTIME_PLAN_ITEM.test(String(item)));

  if (!runtimeSensitive) return result;

  const hasSignal =
    containsRuntimeSignal(payload.runtimeLogs) ||
    containsRuntimeSignal(payload.runtimeMetrics) ||
    containsRuntimeSignal(payload.runtimeTraces) ||
    containsRuntimeSignal(payload.runtimePerformance);
  const hasGap =
    containsRuntimeSignal(payload.runtimeGap) ||
    (Array.isArray(payload.knownGaps) && payload.knownGaps.some((gap) => /runtime|log|metric|trace|performance/i.test(String(gap))));

  if (!hasSignal && !hasGap) {
    result.warnings.push({
      taskId,
      reason: 'runtime-sensitive task evidence has no runtime logs/metrics/traces/performance signal and no recorded runtime gap',
    });
  }

  return result;
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function loadLegacyEntries(root, legacyConfigPath) {
  const absolute = path.isAbsolute(legacyConfigPath)
    ? legacyConfigPath
    : path.join(root, legacyConfigPath);
  // A missing allowlist means "nothing is legacy": check every artifact. Deleting
  // the file re-exposes the exempted rounds loudly instead of hiding them.
  if (!fs.existsSync(absolute)) return [];
  let payload;
  try {
    payload = JSON.parse(fs.readFileSync(absolute, 'utf8'));
  } catch (error) {
    throw new Error(`legacy runtime-evidence allowlist is not valid JSON: ${legacyConfigPath}: ${error.message}`);
  }
  const entries = Array.isArray(payload.entries) ? payload.entries : [];
  for (const entry of entries) {
    if (typeof entry !== 'string' || entry.trim() === '') {
      throw new Error(`legacy runtime-evidence allowlist entries must be non-empty strings: ${JSON.stringify(entry)}`);
    }
  }
  return entries.map((entry) => entry.replaceAll('\\', '/'));
}

function partitionLegacy(root, files, entries) {
  const legacySet = new Set(entries);
  const checked = [];
  const skipped = [];
  for (const file of files) {
    (legacySet.has(toRepoPath(file, root)) ? skipped : checked).push(file);
  }
  // The allowlist must shrink: an entry whose artifact no longer exists is an
  // error, so archiving or deleting a round forces someone to edit this file.
  const stale = entries.filter((entry) => !fs.existsSync(path.join(root, entry)));
  return { checked, skipped, stale };
}

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  if (options.help) return printHelp(), 0;
  let legacyEntries;
  try {
    legacyEntries = loadLegacyEntries(options.root, options.legacyConfig);
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  const { checked, skipped, stale } = partitionLegacy(options.root, discoverEvidenceFiles(options.root), legacyEntries);
  const results = checked.map((file) => scanFile(file, options.root));
  for (const staleEntry of stale) {
    results.push({ file: staleEntry, errors: ['stale legacy runtime-evidence allowlist entry: the artifact no longer exists'], warnings: [] });
  }
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  if (options.json) console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', fileCount: results.length, legacySkipped: skipped.length, errorCount, warningCount, results }, null, 2));
  else {
    if (skipped.length > 0) console.log(`Legacy runtime-evidence artifacts exempted by ${options.legacyConfig}: ${skipped.length} (must shrink; see the file's description)`);
    console.log(`Runtime evidence check (${options.strict ? 'strict' : 'report-only'}): ${results.length} file(s), ${errorCount} error(s), ${warningCount} warning(s)`);
    for (const result of results) {
      for (const error of result.errors) console.log(`error: ${result.file}\n  reason: ${error}`);
      for (const warning of result.warnings) console.log(`warning: ${result.file}\n  task: ${warning.taskId}\n  reason: ${warning.reason}`);
    }
  }
  return options.strict && (errorCount > 0 || warningCount > 0) ? 1 : 0;
}

process.exitCode = main();
