#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';
import {
  buildTaskManifestPath,
  normalizeRepoRelativePath,
  readTaskManifest,
} from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_CONFIG = 'config/method.config.json';
const DEFAULT_EVIDENCE_DIR = '.harness/evidence';
const VALID_VERDICTS = new Set([
  'approved',
  'changes requested',
  'blocked',
  'approved with documented P2 follow-up',
]);

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-review.mjs [--json] [--strict] [--root <path>] [--config <path>] [review-file ...]`);
}

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT, config: DEFAULT_CONFIG, files: [] };
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
    } else if (arg === '--config') {
      const value = argv[index + 1];
      if (!value) throw new Error('--config requires a path');
      options.config = value;
      index += 1;
    } else {
      options.files.push(arg);
    }
  }
  return options;
}

function loadConfig(root, configPath) {
  const absolute = path.isAbsolute(configPath) ? configPath : path.join(root, configPath);
  if (!fs.existsSync(absolute)) return { evidenceDir: DEFAULT_EVIDENCE_DIR };
  try {
    const config = JSON.parse(fs.readFileSync(absolute, 'utf8'));
    return { evidenceDir: typeof config.evidenceDir === 'string' && config.evidenceDir ? config.evidenceDir : DEFAULT_EVIDENCE_DIR };
  } catch {
    return { evidenceDir: DEFAULT_EVIDENCE_DIR };
  }
}

function normalizeInputFile(inputPath, root) {
  return path.isAbsolute(inputPath) ? inputPath : path.join(root, inputPath);
}

function discoverReviewFiles(root, evidenceDir) {
  const files = [];
  const base = path.join(root, evidenceDir);
  function walk(dirPath) {
    if (!fs.existsSync(dirPath)) return;
    for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
      const fullPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) walk(fullPath);
      else if (entry.isFile() && entry.name === 'review.md') files.push(fullPath);
    }
  }
  walk(base);
  return sortStrings(files);
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function extractMachineReadableBlock(content) {
  const match = content.match(/## Machine Readable\s+```json\s*([\s\S]*?)\s*```/m);
  if (!match) return { error: 'missing ## Machine Readable JSON block' };
  try {
    return { value: JSON.parse(match[1]) };
  } catch (error) {
    return { error: `invalid machine-readable JSON: ${error.message}` };
  }
}

function requireNonEmptyString(object, key, errors, label) {
  if (typeof object[key] !== 'string' || object[key].trim() === '') errors.push(`${label}.${key} must be a non-empty string.`);
}

function validateReview(filePath, root) {
  const content = fs.readFileSync(filePath, 'utf8');
  const extracted = extractMachineReadableBlock(content);
  const result = { file: toRepoPath(filePath, root), errors: [], warnings: [] };
  if (extracted.error) {
    result.errors.push(extracted.error);
    return result;
  }
  const review = extracted.value;
  if (!isObject(review)) {
    result.errors.push('machine-readable review artifact must be a JSON object');
    return result;
  }
  requireNonEmptyString(review, 'taskId', result.errors, 'root');
  requireNonEmptyString(review, 'verdict', result.errors, 'root');
  if (typeof review.verdict === 'string' && !VALID_VERDICTS.has(review.verdict)) {
    result.errors.push(`root.verdict must be one of: ${Array.from(VALID_VERDICTS).join(', ')}.`);
  }
  validateStructuralReview(review, result.errors);
  if (!isObject(review.linkage)) {
    result.errors.push('root.linkage must be an object.');
    return result;
  }
  requireNonEmptyString(review.linkage, 'taskManifest', result.errors, 'root.linkage');
  requireNonEmptyString(review.linkage, 'evidence', result.errors, 'root.linkage');
  requireNonEmptyString(review.linkage, 'reviewFile', result.errors, 'root.linkage');
  requireNonEmptyString(review.linkage, 'changeRef', result.errors, 'root.linkage');
  if (!Array.isArray(review.linkage.planRefs)) {
    result.errors.push('root.linkage.planRefs must be an array.');
  } else {
    review.linkage.planRefs.forEach((ref, index) => {
      if (typeof ref !== 'string' || ref.trim() === '') result.errors.push(`root.linkage.planRefs[${index}] must be a non-empty string.`);
      else if (!fs.existsSync(path.join(root, ref))) result.errors.push(`linked plan missing: ${ref}`);
    });
  }

  const expectedTaskId = path.basename(path.dirname(filePath));
  if (review.taskId !== expectedTaskId) result.errors.push(`root.taskId must match evidence directory name "${expectedTaskId}".`);
  const expectedTaskManifest = buildTaskManifestPath(expectedTaskId);
  const expectedEvidence = `.harness/evidence/${expectedTaskId}/commands.json`;
  const expectedReview = `.harness/evidence/${expectedTaskId}/review.md`;
  if (review.linkage.taskManifest !== expectedTaskManifest) {
    result.errors.push(`root.linkage.taskManifest must be "${expectedTaskManifest}".`);
  } else {
    try {
      const manifest = readTaskManifest(root, review.linkage.taskManifest);
      if (manifest.payload.taskId !== expectedTaskId) {
        result.errors.push(
          `linked task manifest task id "${manifest.payload.taskId}" must match "${expectedTaskId}".`,
        );
      }
    } catch (error) {
      result.errors.push(error.message);
    }
  }
  if (review.linkage.evidence !== expectedEvidence) result.errors.push(`root.linkage.evidence must be "${expectedEvidence}".`);
  else if (!fs.existsSync(path.join(root, review.linkage.evidence))) result.errors.push(`linked evidence missing: ${review.linkage.evidence}`);
  if (review.linkage.reviewFile !== expectedReview) result.errors.push(`root.linkage.reviewFile must be "${expectedReview}".`);
  if (typeof review.linkage.changeRef === 'string' && review.linkage.changeRef !== 'none' && !fs.existsSync(path.join(root, review.linkage.changeRef))) {
    result.errors.push(`linked OpenSpec change missing: ${review.linkage.changeRef}`);
  }
  return result;
}

function validateStructuralReview(review, errors) {
  if (!('structuralReview' in review)) {
    return;
  }

  if (!isObject(review.structuralReview)) {
    errors.push('root.structuralReview must be an object when present.');
    return;
  }

  const structuralReview = review.structuralReview;
  if ('affectedSubgraph' in structuralReview) {
    const { affectedSubgraph } = structuralReview;
    if (typeof affectedSubgraph === 'string') {
      if (affectedSubgraph.trim() === '') {
        errors.push('root.structuralReview.affectedSubgraph must not be empty when present.');
      }
    } else if (Array.isArray(affectedSubgraph)) {
      affectedSubgraph.forEach((entry, index) => {
        if (typeof entry !== 'string' || entry.trim() === '') {
          errors.push(`root.structuralReview.affectedSubgraph[${index}] must be a non-empty string.`);
        }
      });
    } else {
      errors.push('root.structuralReview.affectedSubgraph must be a string or an array of strings when present.');
    }
  }

  if ('checks' in structuralReview) {
    const validChecks = new Set(['cycle', 'hub', 'call-depth', 'sensitive-flow']);
    if (!Array.isArray(structuralReview.checks)) {
      errors.push('root.structuralReview.checks must be an array when present.');
    } else {
      structuralReview.checks.forEach((entry, index) => {
        if (typeof entry !== 'string' || !validChecks.has(entry)) {
          errors.push(`root.structuralReview.checks[${index}] must be one of: ${Array.from(validChecks).join(', ')}.`);
        }
      });
    }
  }

  if ('findings' in structuralReview) {
    if (!Array.isArray(structuralReview.findings)) {
      errors.push('root.structuralReview.findings must be an array when present.');
    } else {
      structuralReview.findings.forEach((entry, index) => {
        if (typeof entry !== 'string') {
          errors.push(`root.structuralReview.findings[${index}] must be a string.`);
        }
      });
    }
  }

  if ('notes' in structuralReview && typeof structuralReview.notes !== 'string') {
    errors.push('root.structuralReview.notes must be a string when present.');
  }
}

function printTextReport(results, strict) {
  const mode = strict ? 'strict' : 'report-only';
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  console.log(`Review check (${mode}): ${results.length} file(s), ${errorCount} error(s)`);
  for (const result of results) {
    console.log('');
    console.log(result.errors.length === 0 ? `[PASS] ${result.file}` : `[FAIL] ${result.file}`);
    for (const error of result.errors) console.log(`  error: ${error}`);
  }
}

function buildMissingReviewResult(root, evidenceDir, strict) {
  return {
    file: toRepoPath(path.join(root, evidenceDir), root),
    errors: strict ? ['No review artifacts found.'] : [],
    warnings: strict ? [] : ['No review artifacts found.'],
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
  if (options.help) return printHelp(), 0;
  const config = loadConfig(options.root, options.config);
  const files = options.files.length > 0 ? options.files.map((file) => normalizeInputFile(file, options.root)) : discoverReviewFiles(options.root, config.evidenceDir);
  const results =
    files.length > 0
      ? files.map((file) => validateReview(file, options.root))
      : [buildMissingReviewResult(options.root, config.evidenceDir, options.strict)];
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  if (options.json) {
    console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', fileCount: results.length, errorCount, results }, null, 2));
  } else {
    printTextReport(results, options.strict);
  }
  return options.strict && errorCount > 0 ? 1 : 0;
}

process.exitCode = main();
