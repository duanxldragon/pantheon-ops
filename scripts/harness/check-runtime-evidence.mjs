#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';
import { readTaskManifest } from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') options.json = true;
    else if (arg === '--strict') options.strict = true;
    else if (arg === '--help' || arg === '-h') options.help = true;
    else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) throw new Error('--root requires a path');
      options.root = path.resolve(value);
      index += 1;
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-runtime-evidence.mjs [--json] [--strict] [--root <path>]');
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
  return sortStrings(files);
}

function containsRuntimeSignal(value) {
  if (Array.isArray(value)) return value.length > 0;
  return typeof value === 'string' && value.trim() !== '';
}

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
  let runtimeSensitive = payload.runtimeSensitive === true;
  if (taskManifestReference) {
    try {
      const taskManifest = readTaskManifest(root, taskManifestReference);
      runtimeSensitive ||=
        Array.isArray(taskManifest.payload.verificationPlan?.runtimeEvidence) &&
        taskManifest.payload.verificationPlan.runtimeEvidence.length > 0;
      runtimeSensitive ||= taskManifest.payload.runtimeSensitive === true;
    } catch {
      // ignore invalid manifests here; structural validators report that separately
    }
  }

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

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  if (options.help) return printHelp(), 0;
  const results = discoverEvidenceFiles(options.root).map((file) => scanFile(file, options.root));
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  if (options.json) console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', fileCount: results.length, errorCount, warningCount, results }, null, 2));
  else {
    console.log(`Runtime evidence check (${options.strict ? 'strict' : 'report-only'}): ${results.length} file(s), ${errorCount} error(s), ${warningCount} warning(s)`);
    for (const result of results) {
      for (const error of result.errors) console.log(`error: ${result.file}\n  reason: ${error}`);
      for (const warning of result.warnings) console.log(`warning: ${result.file}\n  task: ${warning.taskId}\n  reason: ${warning.reason}`);
    }
  }
  return options.strict && (errorCount > 0 || warningCount > 0) ? 1 : 0;
}

process.exitCode = main();
