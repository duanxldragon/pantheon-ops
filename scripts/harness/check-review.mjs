#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';
import {
  buildTaskManifestPath,
  normalizeRepoRelativePath,
  readTaskManifest,
  resolveArchiveAwarePath,
  TASK_ARCHIVE_ROOT,
} from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_CONFIG = 'config/method.config.json';
const DEFAULT_EVIDENCE_DIR = '.harness/evidence';
const DEFAULT_LEGACY_CONFIG = 'config/review-loop-legacy-artifacts.json';
const VALID_VERDICTS = new Set([
  'approved',
  'changes requested',
  'blocked',
  'approved with documented P2 follow-up',
]);

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-review.mjs [--json] [--strict] [--root <path>] [--config <path>] [--legacy-config <path>] [review-file ...]`);
}

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT, config: DEFAULT_CONFIG, legacyConfig: DEFAULT_LEGACY_CONFIG, files: [] };
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

function listArchiveMonths(root) {
  const archiveRoot = path.join(root, TASK_ARCHIVE_ROOT);
  if (!fs.existsSync(archiveRoot)) return [];
  return fs
    .readdirSync(archiveRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && /^\d{4}-\d{2}$/.test(entry.name))
    .map((entry) => entry.name)
    .sort((left, right) => right.localeCompare(left));
}

function discoverReviewFiles(root, evidenceDir) {
  const files = [];
  function walk(dirPath) {
    if (!fs.existsSync(dirPath)) return;
    for (const entry of fs.readdirSync(dirPath, { withFileTypes: true })) {
      const fullPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) walk(fullPath);
      else if (entry.isFile() && entry.name === 'review.md') files.push(fullPath);
    }
  }
  walk(path.join(root, evidenceDir));
  // Archiving moves a packet's evidence tree wholesale under
  // `.harness/archive/<YYYY-MM>/evidence/`; the review contract still has to
  // hold for those artifacts, otherwise closing a round would silently drop
  // 200+ reviewed packets out of the gate's scope.
  for (const month of listArchiveMonths(root)) {
    walk(path.join(root, TASK_ARCHIVE_ROOT, month, 'evidence'));
  }
  return sortStrings(files);
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

// A plan reference is written when the artifact is created and then lives for
// years. Three spellings have to stay valid: a repo-root path, a canonical
// `.harness/{tasks,evidence}/...` path whose tree was later archived, and a
// sibling file addressed relative to the review itself. Resolving all three
// keeps the gate honest about the linked file existing without asking every
// archived round to rewrite its own history.
function resolvePlanRef(root, ref, artifactDir) {
  if (resolveArchiveAwarePath(root, ref)) {
    return ref;
  }
  const sibling = path.join(artifactDir, ref);
  return fs.existsSync(sibling) ? sibling : null;
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
    const artifactDir = path.dirname(filePath);
    review.linkage.planRefs.forEach((ref, index) => {
      if (typeof ref !== 'string' || ref.trim() === '') {
        result.errors.push(`root.linkage.planRefs[${index}] must be a non-empty string.`);
      } else if (!resolvePlanRef(root, ref, artifactDir)) {
        result.errors.push(`linked plan missing: ${ref}`);
      }
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
  else if (!resolveArchiveAwarePath(root, review.linkage.evidence)) result.errors.push(`linked evidence missing: ${review.linkage.evidence}`);
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
    throw new Error(`legacy review allowlist is not valid JSON: ${legacyConfigPath}: ${error.message}`);
  }
  const entries = Array.isArray(payload.entries) ? payload.entries : [];
  for (const entry of entries) {
    if (typeof entry !== 'string' || entry.trim() === '') {
      throw new Error(`legacy review allowlist entries must be non-empty strings: ${JSON.stringify(entry)}`);
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
  let config;
  let legacyEntries;
  try {
    config = loadConfig(options.root, options.config);
    legacyEntries = loadLegacyEntries(options.root, options.legacyConfig);
  } catch (error) {
    console.error(error.message);
    return 1;
  }
  const discovered =
    options.files.length > 0
      ? options.files.map((file) => normalizeInputFile(file, options.root))
      : discoverReviewFiles(options.root, config.evidenceDir);
  const { checked, skipped, stale } = partitionLegacy(options.root, discovered, legacyEntries);
  const results =
    checked.length > 0
      ? checked.map((file) => validateReview(file, options.root))
      : [buildMissingReviewResult(options.root, config.evidenceDir, options.strict)];
  for (const staleEntry of stale) {
    results.push({ file: staleEntry, errors: ['stale legacy review allowlist entry: the artifact no longer exists'], warnings: [] });
  }
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  if (skipped.length > 0) {
    console.log(`Legacy review artifacts exempted by ${options.legacyConfig}: ${skipped.length} (must shrink; see the file's description)`);
  }
  if (options.json) {
    console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', fileCount: results.length, errorCount, results }, null, 2));
  } else {
    printTextReport(results, options.strict);
  }
  return options.strict && errorCount > 0 ? 1 : 0;
}

process.exitCode = main();
