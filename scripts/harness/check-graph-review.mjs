#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import {
  listTaskManifestPaths,
  normalizeRepoRelativePath,
  readTaskManifest,
  resolveRepoPath,
} from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-graph-review.mjs [--json] [--strict] [--root <path>]

Default behavior:
  Report graph-review consistency warnings and exit 0. Use --strict to exit 1 when warnings exist.

Examples:
  node scripts/harness/check-graph-review.mjs
  node scripts/harness/check-graph-review.mjs --json
  node scripts/harness/check-graph-review.mjs --strict
  node scripts/harness/check-graph-review.mjs --root /tmp/fixture --strict`);
}

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT };
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

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function normalizeChecks(value) {
  const items = Array.isArray(value) ? value : typeof value === 'string' ? [value] : [];
  return items
    .map((entry) => String(entry).trim())
    .filter((entry) => entry && entry.toLowerCase() !== 'none')
    .sort((left, right) => left.localeCompare(right));
}

function normalizeAffectedSubgraph(value) {
  const items = Array.isArray(value) ? value : typeof value === 'string' ? [value] : [];
  return items
    .map((entry) => String(entry).trim())
    .filter((entry) => entry && entry.toLowerCase() !== 'none')
    .sort((left, right) => left.localeCompare(right));
}

function normalizeManifestGraphFocus(value) {
  const mapping = new Map([
    ['cycle-check', 'cycle'],
    ['hub-check', 'hub'],
    ['call-depth', 'call-depth'],
    ['sensitive-input-flow', 'sensitive-flow'],
    ['cycle', 'cycle'],
    ['hub', 'hub'],
    ['sensitive-flow', 'sensitive-flow'],
  ]);
  return (Array.isArray(value) ? value : [])
    .map((entry) => mapping.get(String(entry).trim()) || String(entry).trim())
    .filter((entry) => entry && entry.toLowerCase() !== 'none')
    .sort((left, right) => left.localeCompare(right));
}

function readJsonIfExists(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return { exists: false, value: null, error: null };
  }
  try {
    return { exists: true, value: JSON.parse(fs.readFileSync(filePath, 'utf8')), error: null };
  } catch (error) {
    return { exists: true, value: null, error: error.message };
  }
}

function extractReviewMachineReadable(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return { exists: false, value: null, error: null };
  }
  const content = fs.readFileSync(filePath, 'utf8');
  const match = content.match(/## Machine Readable\s+```json\s*([\s\S]*?)\s*```/m);
  if (!match) {
    return { exists: true, value: null, error: 'missing machine-readable JSON block' };
  }
  try {
    return { exists: true, value: JSON.parse(match[1]), error: null };
  } catch (error) {
    return { exists: true, value: null, error: error.message };
  }
}

function parseGraphChecks(payload) {
  if (!payload || typeof payload !== 'object' || payload === null || !('graphChecks' in payload)) {
    return null;
  }
  return {
    affectedSubgraph: normalizeAffectedSubgraph(payload.graphChecks?.affectedSubgraph),
    checks: normalizeChecks(payload.graphChecks?.checks),
  };
}

function parseStructuralReview(payload) {
  if (!payload || typeof payload !== 'object' || payload === null || !('structuralReview' in payload)) {
    return null;
  }
  return {
    affectedSubgraph: normalizeAffectedSubgraph(payload.structuralReview?.affectedSubgraph),
    checks: normalizeChecks(payload.structuralReview?.checks),
  };
}

function buildManifestScope(manifestPayload) {
  const scope = manifestPayload.structuralScope;
  if (!scope || typeof scope !== 'object' || Array.isArray(scope)) {
    return null;
  }
  return {
    affectedSubgraph: normalizeAffectedSubgraph(scope.affectedSubgraph),
    checks: normalizeManifestGraphFocus(scope.graphFocus),
  };
}

function arraysEqual(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function compareMetadata(taskId, label, left, right, warnings) {
  if (!left || !right) {
    return;
  }

  if (!arraysEqual(left.affectedSubgraph, right.affectedSubgraph)) {
    warnings.push({
      taskId,
      reason: `${label} affected subgraph does not match`,
    });
  }

  if (!arraysEqual(left.checks, right.checks)) {
    warnings.push({
      taskId,
      reason: `${label} structural checks do not match`,
    });
  }
}

function scan(root) {
  const warnings = [];
  const reviewedTasks = [];

  for (const manifestPath of listTaskManifestPaths(root)) {
    let manifest;
    try {
      manifest = readTaskManifest(root, manifestPath);
    } catch (error) {
      warnings.push({
        file: manifestPath,
        taskId: normalizeRepoRelativePath(manifestPath),
        reason: `task manifest is unreadable: ${error.message}`,
      });
      continue;
    }

    const taskId = manifest.payload.taskId;
    const manifestScope = buildManifestScope(manifest.payload);
    if (!manifestScope) {
      continue;
    }
    reviewedTasks.push(taskId);

    const evidencePath = resolveRepoPath(
      root,
      `${normalizeRepoRelativePath(manifest.payload.linkage.evidenceDir)}commands.json`,
    );
    const reviewPath = resolveRepoPath(root, manifest.payload.linkage.reviewFile);
    const evidence = readJsonIfExists(evidencePath);
    const review = extractReviewMachineReadable(reviewPath);
    const graphChecks = parseGraphChecks(evidence.value);
    const structuralReview = parseStructuralReview(review.value);

    if (evidence.exists && evidence.error) {
      warnings.push({
        file: toRepoPath(evidencePath, root),
        taskId,
        reason: `evidence graph review could not be checked because commands.json is unreadable: ${evidence.error}`,
      });
    } else if (!graphChecks) {
      warnings.push({
        file: manifest.payload.linkage.evidenceDir,
        taskId,
        reason: 'graph-reviewed task is missing evidence.graphChecks',
      });
    }

    if (review.exists && review.error) {
      warnings.push({
        file: manifest.payload.linkage.reviewFile,
        taskId,
        reason: `review structural consistency could not be checked because review.md is unreadable: ${review.error}`,
      });
    } else if (!structuralReview) {
      warnings.push({
        file: manifest.payload.linkage.reviewFile,
        taskId,
        reason: 'graph-reviewed task is missing review.structuralReview',
      });
    }

    compareMetadata(taskId, 'manifest vs evidence', manifestScope, graphChecks, warnings);
    compareMetadata(taskId, 'manifest vs review', manifestScope, structuralReview, warnings);
    compareMetadata(taskId, 'evidence vs review', graphChecks, structuralReview, warnings);
  }

  return { reviewedTasks, warnings };
}

function printTextReport(result, strict) {
  const mode = strict ? 'strict' : 'report-only';
  console.log(
    `Graph review check (${mode}): ${result.reviewedTasks.length} graph-reviewed task(s), ${result.warnings.length} warning(s)`,
  );
  if (result.warnings.length === 0) {
    console.log('\nno findings');
  }
  for (const warning of result.warnings) {
    console.log(`\nwarning: ${warning.file}`);
    console.log(`  task: ${warning.taskId}`);
    console.log(`  reason: ${warning.reason}`);
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

  const result = scan(options.root);
  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          reviewedTaskCount: result.reviewedTasks.length,
          warningCount: result.warnings.length,
          reviewedTasks: result.reviewedTasks,
          warnings: result.warnings,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(result, options.strict);
  }

  return options.strict && result.warnings.length > 0 ? 1 : 0;
}

process.exitCode = main();
