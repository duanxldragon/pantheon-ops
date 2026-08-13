#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';
import {
  extractTaskIdFromManifestPath,
  normalizeRepoRelativePath,
  readTaskManifest,
} from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();
const VALID_AGENT_TOOLS = new Set([
  'codex',
  'claude-code',
  'cursor',
  'github-copilot',
  'openhands',
  'aider',
  'human',
  'other',
]);
const VALID_COMMAND_STATUSES = new Set(['passed', 'failed', 'not-run']);

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-evidence.mjs [--json] [--strict] [--root <path>] [commands-json ...]

Defaults:
  Scans <root>/.harness/evidence/**/commands.json when no files are provided.

Examples:
  node scripts/harness/check-evidence.mjs
  node scripts/harness/check-evidence.mjs --json
  node scripts/harness/check-evidence.mjs --strict
  node scripts/harness/check-evidence.mjs --root /tmp/fixture --strict`);
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

function discoverEvidenceFiles(root) {
  const files = [];
  const evidenceDir = path.join(root, '.harness', 'evidence');

  function walk(dirPath) {
    if (!fs.existsSync(dirPath)) {
      return;
    }

    for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
      const entryPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) {
        walk(entryPath);
      } else if (entry.isFile() && entry.name === 'commands.json') {
        files.push(entryPath);
      }
    }
  }

  walk(evidenceDir);
  return sortStrings(files);
}

function normalizeInputFile(inputPath, root) {
  return path.isAbsolute(inputPath) ? inputPath : path.join(root, inputPath);
}

function relative(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function requireString(object, key, errors, pathLabel) {
  if (typeof object[key] !== 'string' || object[key].trim() === '') {
    errors.push(`${pathLabel}.${key} must be a non-empty string.`);
  }
}

function validateBrowserEvidence(entries, errors) {
  entries.forEach((entry, index) => {
    const pathLabel = `root.browserEvidence[${index}]`;
    if (!isObject(entry)) {
      errors.push(`${pathLabel} must be an object.`);
      return;
    }

    requireString(entry, 'viewport', errors, pathLabel);
    requireString(entry, 'url', errors, pathLabel);

    if ('screenshot' in entry && (typeof entry.screenshot !== 'string' || entry.screenshot.trim() === '')) {
      errors.push(`${pathLabel}.screenshot must be a non-empty string when present.`);
    }

    if ('visualGap' in entry && (typeof entry.visualGap !== 'string' || entry.visualGap.trim() === '')) {
      errors.push(`${pathLabel}.visualGap must be a non-empty string when present.`);
    }

    if (!('screenshot' in entry) && !('visualGap' in entry)) {
      errors.push(`${pathLabel} must include a screenshot or explicit visualGap.`);
    }

    if (!Array.isArray(entry.consoleErrors)) {
      errors.push(`${pathLabel}.consoleErrors must be an array.`);
    } else {
      entry.consoleErrors.forEach((consoleError, consoleIndex) => {
        if (typeof consoleError !== 'string') {
          errors.push(`${pathLabel}.consoleErrors[${consoleIndex}] must be a string.`);
        }
      });
    }

    if ('checkedStates' in entry) {
      if (!Array.isArray(entry.checkedStates)) {
        errors.push(`${pathLabel}.checkedStates must be an array when present.`);
      } else {
        entry.checkedStates.forEach((state, stateIndex) => {
          if (typeof state !== 'string' || state.trim() === '') {
            errors.push(`${pathLabel}.checkedStates[${stateIndex}] must be a non-empty string.`);
          }
        });
      }
    }
  });
}

function validateLinkage(evidence, filePath, root, errors) {
  if (!isObject(evidence.linkage)) {
    errors.push('root.linkage must be an object.');
    return;
  }

  const requiredFields = ['taskManifest', 'evidenceDir', 'reviewFile', 'changeRef', 'planRefs'];
  for (const field of requiredFields) {
    if (!(field in evidence.linkage)) {
      errors.push(`root.linkage.${field} is required.`);
    }
  }

  if (typeof evidence.linkage.taskManifest === 'string') {
    const manifestRepoPath = normalizeRepoRelativePath(evidence.linkage.taskManifest);
    const manifestTaskId = extractTaskIdFromManifestPath(manifestRepoPath);
    const taskManifestPath = path.join(root, manifestRepoPath);
    if (!fs.existsSync(taskManifestPath)) {
      errors.push(`root.linkage.taskManifest does not exist: ${evidence.linkage.taskManifest}`);
    } else {
      try {
        const manifest = readTaskManifest(root, manifestRepoPath);
        if (manifest.payload.taskId !== evidence.taskId) {
          errors.push(
            `root.linkage.taskManifest task id "${manifest.payload.taskId}" must match root.taskId "${evidence.taskId}".`,
          );
        }
      } catch (error) {
        errors.push(error.message);
      }
      if (manifestTaskId && manifestTaskId !== evidence.taskId) {
        errors.push(
          `root.linkage.taskManifest path task id "${manifestTaskId}" must match root.taskId "${evidence.taskId}".`,
        );
      }
    }
  } else {
    errors.push('root.linkage.taskManifest must be a non-empty string.');
  }

  if (typeof evidence.linkage.evidenceDir === 'string') {
    const normalizedDir = evidence.linkage.evidenceDir.replaceAll('\\', '/');
    const actualDir = relative(path.dirname(filePath), root);
    if (normalizedDir !== `${actualDir}/` && normalizedDir !== actualDir) {
      errors.push(`root.linkage.evidenceDir must match the commands.json directory "${actualDir}/".`);
    }
  } else {
    errors.push('root.linkage.evidenceDir must be a non-empty string.');
  }

  if (typeof evidence.linkage.reviewFile === 'string') {
    if (evidence.linkage.reviewFile !== 'none' && !fs.existsSync(path.join(root, evidence.linkage.reviewFile))) {
      errors.push(`root.linkage.reviewFile does not exist: ${evidence.linkage.reviewFile}`);
    }
  } else {
    errors.push('root.linkage.reviewFile must be a non-empty string.');
  }

  if (typeof evidence.linkage.changeRef === 'string') {
    if (evidence.linkage.changeRef !== 'none' && !fs.existsSync(path.join(root, evidence.linkage.changeRef))) {
      errors.push(`root.linkage.changeRef does not exist: ${evidence.linkage.changeRef}`);
    }
  } else {
    errors.push('root.linkage.changeRef must be a non-empty string.');
  }

  if (!Array.isArray(evidence.linkage.planRefs)) {
    errors.push('root.linkage.planRefs must be an array.');
  } else {
    evidence.linkage.planRefs.forEach((planRef, index) => {
      if (typeof planRef !== 'string' || planRef.trim() === '') {
        errors.push(`root.linkage.planRefs[${index}] must be a non-empty string.`);
      } else if (!fs.existsSync(path.join(root, planRef))) {
        errors.push(`root.linkage.planRefs[${index}] does not exist: ${planRef}`);
      }
    });
  }

  if (
    'summaryFile' in evidence.linkage &&
    typeof evidence.linkage.summaryFile === 'string' &&
    evidence.linkage.summaryFile !== 'none' &&
    !fs.existsSync(path.join(root, evidence.linkage.summaryFile))
  ) {
    errors.push(`root.linkage.summaryFile does not exist: ${evidence.linkage.summaryFile}`);
  }
}

function validateEvidenceFile(filePath, root) {
  const result = {
    file: relative(filePath, root),
    errors: [],
    warnings: [],
  };

  if (!fs.existsSync(filePath)) {
    result.errors.push('File does not exist.');
    return result;
  }

  let evidence;
  try {
    evidence = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    result.errors.push(`Invalid JSON: ${error.message}`);
    return result;
  }

  if (!isObject(evidence)) {
    result.errors.push('Evidence root must be a JSON object.');
    return result;
  }

  requireString(evidence, 'taskId', result.errors, 'root');
  requireString(evidence, 'repo', result.errors, 'root');

  if (!isObject(evidence.agent)) {
    result.errors.push('root.agent must be an object.');
  } else {
    requireString(evidence.agent, 'tool', result.errors, 'root.agent');
    if (typeof evidence.agent.tool === 'string' && !VALID_AGENT_TOOLS.has(evidence.agent.tool)) {
      result.errors.push(
        `root.agent.tool must be one of: ${Array.from(VALID_AGENT_TOOLS).join(', ')}.`,
      );
    }
    if ('adapter' in evidence.agent && typeof evidence.agent.adapter !== 'string') {
      result.errors.push('root.agent.adapter must be a string when present.');
    }
  }

  if (!Array.isArray(evidence.commands)) {
    result.errors.push('root.commands must be an array.');
  } else if (evidence.commands.length === 0) {
    result.warnings.push('root.commands is empty.');
  } else {
    evidence.commands.forEach((command, index) => {
      const pathLabel = `root.commands[${index}]`;
      if (!isObject(command)) {
        result.errors.push(`${pathLabel} must be an object.`);
        return;
      }

      requireString(command, 'command', result.errors, pathLabel);
      requireString(command, 'cwd', result.errors, pathLabel);
      requireString(command, 'status', result.errors, pathLabel);

      if (typeof command.status === 'string' && !VALID_COMMAND_STATUSES.has(command.status)) {
        result.errors.push(
          `${pathLabel}.status must be one of: ${Array.from(VALID_COMMAND_STATUSES).join(', ')}.`,
        );
      }
      if ('durationMs' in command && typeof command.durationMs !== 'number') {
        result.errors.push(`${pathLabel}.durationMs must be a number when present.`);
      }
      if ('notes' in command && typeof command.notes !== 'string') {
        result.errors.push(`${pathLabel}.notes must be a string when present.`);
      }
    });
  }

  if (!Array.isArray(evidence.knownGaps)) {
    result.errors.push('root.knownGaps must be an array.');
  } else {
    evidence.knownGaps.forEach((gap, index) => {
      if (typeof gap !== 'string') {
        result.errors.push(`root.knownGaps[${index}] must be a string.`);
      }
    });
  }

  if ('browserEvidence' in evidence && !Array.isArray(evidence.browserEvidence)) {
    result.errors.push('root.browserEvidence must be an array when present.');
  } else if (Array.isArray(evidence.browserEvidence)) {
    validateBrowserEvidence(evidence.browserEvidence, result.errors);
  }

  validateGraphChecks(evidence, result.errors);

  validateRuntimeMetadata(evidence, result.errors);

  validateLinkage(evidence, filePath, root, result.errors);

  if ('completedAt' in evidence && typeof evidence.completedAt !== 'string') {
    result.errors.push('root.completedAt must be a string when present.');
  }

  return result;
}

function validateRuntimeMetadata(evidence, errors) {
  if ('runtimeSensitive' in evidence && typeof evidence.runtimeSensitive !== 'boolean') {
    errors.push('root.runtimeSensitive must be a boolean when present.');
  }

  for (const field of ['runtimeLogs', 'runtimeMetrics', 'runtimeTraces', 'runtimePerformance']) {
    if (!(field in evidence)) {
      continue;
    }

    const value = evidence[field];
    const pathLabel = `root.${field}`;
    if (typeof value === 'string') {
      if (value.trim() === '') {
        errors.push(`${pathLabel} must not be empty when present.`);
      }
      continue;
    }

    if (!Array.isArray(value)) {
      errors.push(`${pathLabel} must be a string or an array of strings when present.`);
      continue;
    }

    value.forEach((entry, index) => {
      if (typeof entry !== 'string' || entry.trim() === '') {
        errors.push(`${pathLabel}[${index}] must be a non-empty string.`);
      }
    });
  }

  if ('runtimeGap' in evidence && (typeof evidence.runtimeGap !== 'string' || evidence.runtimeGap.trim() === '')) {
    errors.push('root.runtimeGap must be a non-empty string when present.');
  }
}

function validateGraphChecks(evidence, errors) {
  if (!('graphChecks' in evidence)) {
    return;
  }

  if (!isObject(evidence.graphChecks)) {
    errors.push('root.graphChecks must be an object when present.');
    return;
  }

  const graphChecks = evidence.graphChecks;

  if ('usedCodeGraph' in graphChecks && typeof graphChecks.usedCodeGraph !== 'boolean') {
    errors.push('root.graphChecks.usedCodeGraph must be a boolean when present.');
  }

  if ('affectedSubgraph' in graphChecks) {
    const { affectedSubgraph } = graphChecks;
    if (typeof affectedSubgraph === 'string') {
      if (affectedSubgraph.trim() === '') {
        errors.push('root.graphChecks.affectedSubgraph must not be empty when present.');
      }
    } else if (Array.isArray(affectedSubgraph)) {
      affectedSubgraph.forEach((entry, index) => {
        if (typeof entry !== 'string' || entry.trim() === '') {
          errors.push(`root.graphChecks.affectedSubgraph[${index}] must be a non-empty string.`);
        }
      });
    } else {
      errors.push('root.graphChecks.affectedSubgraph must be a string or an array of strings when present.');
    }
  }

  if ('checks' in graphChecks) {
    const validChecks = new Set(['cycle', 'hub', 'call-depth', 'sensitive-flow']);
    if (!Array.isArray(graphChecks.checks)) {
      errors.push('root.graphChecks.checks must be an array when present.');
    } else {
      graphChecks.checks.forEach((entry, index) => {
        if (typeof entry !== 'string' || !validChecks.has(entry)) {
          errors.push(`root.graphChecks.checks[${index}] must be one of: ${Array.from(validChecks).join(', ')}.`);
        }
      });
    }
  }

  if ('findings' in graphChecks) {
    if (!Array.isArray(graphChecks.findings)) {
      errors.push('root.graphChecks.findings must be an array when present.');
    } else {
      graphChecks.findings.forEach((entry, index) => {
        if (typeof entry !== 'string') {
          errors.push(`root.graphChecks.findings[${index}] must be a string.`);
        }
      });
    }
  }

  if ('notes' in graphChecks && typeof graphChecks.notes !== 'string') {
    errors.push('root.graphChecks.notes must be a string when present.');
  }
}

function printTextReport(results, options) {
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);
  const mode = options.strict ? 'strict' : 'report-only';

  console.log(
    `Evidence check (${mode}): ${results.length} file(s), ${errorCount} error(s), ${warningCount} warning(s)`,
  );

  for (const result of results) {
    console.log('');
    console.log(result.errors.length === 0 ? `[PASS] ${result.file}` : `[FAIL] ${result.file}`);

    for (const error of result.errors) {
      console.log(`  error: ${error}`);
    }
    for (const warning of result.warnings) {
      console.log(`  warning: ${warning}`);
    }
  }
}

function buildMissingEvidenceResult(root, strict) {
  return {
    file: relative(path.join(root, '.harness', 'evidence'), root),
    errors: strict ? ['No evidence command files found.'] : [],
    warnings: strict ? [] : ['No evidence command files found.'],
  };
}

function main() {
  let options;
  try {
    options = parseArgs(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
    return;
  }
  if (options.help) {
    printHelp();
    return;
  }

  const root = options.root;
  const files =
    options.files.length > 0
      ? options.files.map((file) => normalizeInputFile(file, root))
      : discoverEvidenceFiles(root);
  const results =
    files.length > 0 ? files.map((file) => validateEvidenceFile(file, root)) : [buildMissingEvidenceResult(root, options.strict)];
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  if (options.json) {
    process.stdout.write(
      `${JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          fileCount: results.length,
          errorCount,
          warningCount,
          results,
        },
        null,
        2,
      )}\n`,
    );
  } else {
    printTextReport(results, options);
  }

  if (options.strict && errorCount > 0) {
    process.exitCode = 1;
  }
}

main();
