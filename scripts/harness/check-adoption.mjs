#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { sortStrings } from './sort-utils.mjs';
import { execFileSync } from 'node:child_process';
import { formatExternalPath, resolvePantheonHarnessRoot } from './upstream-root.mjs';

const DEFAULT_ROOT = process.cwd();
const IMPLEMENTATION_ROOTS = [
  'backend/',
  'frontend/',
  'docs/contracts/',
  'docs/designs/',
  'docs/acceptances/',
];
const TASK_MANIFEST_ROOT = '.harness/tasks/';
const EVIDENCE_ROOT = '.harness/evidence/';
const OPEN_SPEC_CHANGES_ROOT = 'openspec/changes';
const PR_TEMPLATE_CANDIDATES = [
  '.github/pull_request_template.md',
  '.github/PULL_REQUEST_TEMPLATE.md',
];

const REQUIRED_METHOD_FILES = [
  'patterns/harness-core-model.md',
  'patterns/harness-coverage-model.md',
  'patterns/harness-template-taxonomy.md',
  'patterns/tool-adapter-matrix.md',
];

const REQUIRED_FILES = [
  'docs/harness/HARNESS_CORE_MODEL.md',
  'docs/harness/HARNESS_COVERAGE_MODEL.md',
  'docs/harness/HARNESS_TEMPLATE_TAXONOMY.md',
  'docs/harness/TOOL_ADAPTER_MATRIX.md',
  'docs/harness/HARNESS_METHOD_PLAYBOOK.md',
  'docs/harness/HARNESS_ENGINEERING_CONTRACT.md',
  'docs/harness/AGENT_INTERFACE_CONTRACT.md',
  'docs/harness/TASK_PACKET_SPEC.md',
  'docs/harness/VERIFICATION_EVIDENCE_SPEC.md',
  'docs/harness/REVIEW_LOOP_SPEC.md',
  'docs/harness/VISUAL_QUALITY_PROTOCOL.md',
  'scripts/harness/check-inheritance-contract.mjs',
  '.agents/README.md',
  '.agents/adapters/codex.md',
  '.agents/adapters/claude-code.md',
  '.agents/adapters/cursor.md',
  '.agents/adapters/github-copilot.md',
  '.agents/adapters/openhands.md',
  '.agents/adapters/human.md',
];

const REQUIRED_PR_MARKERS = [
  'Task ID',
  'Task Manifest',
  'Verification evidence',
  'OpenSpec change',
  'task id',
  'task manifest',
  'evidence',
  'Review Artifact',
  'Trivial change',
  'boundaries',
  'backend response contract',
  'backend DTO contract',
  'permission contract',
  'audit coverage',
  'visual evidence',
  'inheritance contract',
  'base drift',
  'Base/ops inheritance',
];

const REQUIRED_AGENT_PROMPT_MARKERS = [
  'Task manifest',
  'Record verification results',
  'Do not claim completion without fresh verification evidence',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-adoption.mjs [--json] [--strict] [--root <path>] [--changed-file <path> ...]

Checks that Phase 7 Harness adoption entrypoints are present:
- shared contracts
- tool adapters
- PR template task/evidence/trivial markers
- implementation prompt completion evidence rules
- implementation changes are paired with task manifest and evidence files`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    root: DEFAULT_ROOT,
    changedFiles: [],
    methodRoot: null,
    config: undefined,
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
    } else if (arg === '--method-root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--method-root requires a path');
      }
      options.methodRoot = value;
      index += 1;
    } else if (arg === '--config') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--config requires a path');
      }
      options.config = value;
      index += 1;
    } else if (arg === '--changed-file') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--changed-file requires a path');
      }
      options.changedFiles.push(value.replaceAll('\\', '/'));
      index += 1;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }
  return options;
}

function readText(root, repoPath) {
  return fs.readFileSync(path.join(root, repoPath), 'utf8');
}

function findExistingRepoPath(root, candidates) {
  return candidates.find((candidate) => fs.existsSync(path.join(root, candidate))) ?? null;
}

function hasAllMarkers(content, markers) {
  return markers.filter((marker) => !content.includes(marker));
}

function toRepoPath(filePath, root) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function discoverChangedFiles(root) {
  try {
    const tracked = execFileSync(
      'git',
      ['diff', '--name-only', '--relative', 'HEAD'],
      { cwd: root, encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] },
    )
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
    const untracked = execFileSync(
      'git',
      ['ls-files', '--others', '--exclude-standard'],
      { cwd: root, encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] },
    )
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean);
    return sortStrings(Array.from(new Set([...tracked, ...untracked])));
  } catch {
    return [];
  }
}

function isTaskManifestFile(file) {
  return /^\.harness\/tasks\/.+\/manifest\.json$/u.test(file);
}

function scanChangedFiles(root, findings, warnings, changedFiles) {
  const normalizedFiles = sortStrings(
    Array.from(new Set(changedFiles.map((file) => file.replaceAll('\\', '/')))),
  );
  if (normalizedFiles.length === 0) {
    warnings.push({
      file: '.',
      reason: 'no changed files detected; implementation adoption linkage was not evaluated',
    });
    return;
  }

  const implementationChanges = normalizedFiles.filter((file) =>
    IMPLEMENTATION_ROOTS.some((prefix) => file.startsWith(prefix)),
  );
  if (implementationChanges.length === 0) {
    return;
  }

  const hasTaskManifestChange = normalizedFiles.some((file) => isTaskManifestFile(file));
  const hasEvidenceChange = normalizedFiles.some(
    (file) => file.startsWith(EVIDENCE_ROOT) && /\/commands\.json$/.test(file),
  );

  if (!hasTaskManifestChange) {
    findings.push({
      file: implementationChanges[0],
      reason: 'implementation change detected without a matching task manifest change',
    });
  }

  if (!hasEvidenceChange) {
    findings.push({
      file: implementationChanges[0],
      reason: 'implementation change detected without matching verification evidence',
    });
  }
}

function listActiveOpenSpecChanges(root) {
  const changesRoot = path.join(root, OPEN_SPEC_CHANGES_ROOT);
  if (!fs.existsSync(changesRoot)) {
    return [];
  }

  return fs
    .readdirSync(changesRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && entry.name !== 'archive')
    .map((entry) => `${OPEN_SPEC_CHANGES_ROOT}/${entry.name}/`)
    .sort((left, right) => left.localeCompare(right));
}

function parseManifestChangeRef(root, repoPath) {
  const fullPath = path.join(root, repoPath);
  if (!fs.existsSync(fullPath)) {
    return null;
  }

  try {
    const payload = JSON.parse(fs.readFileSync(fullPath, 'utf8'));
    return payload?.linkage?.changeRef ?? null;
  } catch {
    return null;
  }
}

function parseEvidenceChangeRef(root, repoPath) {
  const fullPath = path.join(root, repoPath);
  if (!fs.existsSync(fullPath)) {
    return null;
  }

  try {
    const payload = JSON.parse(fs.readFileSync(fullPath, 'utf8'));
    return payload?.linkage?.changeRef ?? null;
  } catch {
    return null;
  }
}

function scanOpenSpecLinkage(root, findings, changedFiles) {
  const activeChanges = listActiveOpenSpecChanges(root);
  if (activeChanges.length === 0) {
    return;
  }

  const changedTaskManifests = changedFiles.filter((file) => isTaskManifestFile(file));
  const changedEvidenceFiles = changedFiles.filter(
    (file) => file.startsWith(EVIDENCE_ROOT) && /\/commands\.json$/.test(file),
  );

  for (const taskManifest of changedTaskManifests) {
    const changeRef = parseManifestChangeRef(root, taskManifest);
    if (!changeRef || changeRef === 'none') {
      findings.push({
        file: taskManifest,
        reason: 'active OpenSpec change exists; changed task manifest must declare a real linkage.changeRef',
      });
      continue;
    }
    if (!activeChanges.includes(changeRef)) {
      findings.push({
        file: taskManifest,
        reason: `task manifest linkage.changeRef must reference an active change: ${activeChanges.join(', ')}`,
      });
    }
  }

  for (const evidenceFile of changedEvidenceFiles) {
    const changeRef = parseEvidenceChangeRef(root, evidenceFile);
    if (!changeRef || changeRef === 'none') {
      findings.push({
        file: evidenceFile,
        reason: 'active OpenSpec change exists; changed evidence must declare a real linkage.changeRef',
      });
      continue;
    }
    if (!activeChanges.includes(changeRef)) {
      findings.push({
        file: evidenceFile,
        reason: `evidence linkage.changeRef must reference an active change: ${activeChanges.join(', ')}`,
      });
    }
  }
}

function scanAdoption(root, changedFiles, options = {}) {
  const findings = [];
  const warnings = [];
  const methodRoot = resolvePantheonHarnessRoot(root, options);

  for (const repoPath of REQUIRED_METHOD_FILES) {
    if (!fs.existsSync(path.join(methodRoot, repoPath))) {
      findings.push({
        file: formatExternalPath(root, methodRoot, repoPath),
        reason: 'required upstream Harness method file is missing',
      });
    }
  }

  for (const repoPath of REQUIRED_FILES) {
    if (!fs.existsSync(path.join(root, repoPath))) {
      findings.push({
        file: repoPath,
        reason: 'required Harness adoption file is missing',
      });
    }
  }

  const prTemplatePath = findExistingRepoPath(root, PR_TEMPLATE_CANDIDATES);
  if (!prTemplatePath) {
    findings.push({
      file: PR_TEMPLATE_CANDIDATES[0],
      reason: 'required Harness adoption file is missing',
    });
  } else {
    const missing = hasAllMarkers(readText(root, prTemplatePath), REQUIRED_PR_MARKERS);
    for (const marker of missing) {
      findings.push({
        file: prTemplatePath,
        reason: `PR template missing adoption marker: ${marker}`,
      });
    }
  }

  const implementationPromptPath = '.agents/prompts/implementation.md';
  if (!fs.existsSync(path.join(root, implementationPromptPath))) {
    findings.push({
      file: implementationPromptPath,
      reason: 'implementation prompt is missing',
    });
  } else {
    const missing = hasAllMarkers(readText(root, implementationPromptPath), REQUIRED_AGENT_PROMPT_MARKERS);
    for (const marker of missing) {
      warnings.push({
        file: implementationPromptPath,
        reason: `implementation prompt missing adoption marker: ${marker}`,
      });
    }
  }

  scanChangedFiles(root, findings, warnings, changedFiles);
  scanOpenSpecLinkage(root, findings, changedFiles);

  return { findings, warnings };
}

function printTextReport(result, strict) {
  const mode = strict ? 'strict' : 'report-only';
  console.log(
    `Adoption check (${mode}): ${result.findings.length} finding(s), ${result.warnings.length} warning(s)`,
  );
  if (result.findings.length === 0 && result.warnings.length === 0) {
    console.log('\nno findings');
  }
  for (const finding of result.findings) {
    console.log(`\nfinding: ${finding.file}`);
    console.log(`  reason: ${finding.reason}`);
  }
  for (const warning of result.warnings) {
    console.log(`\nwarning: ${warning.file}`);
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

  const changedFiles = options.changedFiles.length > 0 ? options.changedFiles : discoverChangedFiles(options.root);
  const result = scanAdoption(options.root, changedFiles, options);
  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          changedFileCount: changedFiles.length,
          findingCount: result.findings.length,
          warningCount: result.warnings.length,
          changedFiles,
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
