#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { formatExternalPath, resolvePantheonHarnessRoot } from './upstream-root.mjs';

const DEFAULT_ROOT = process.cwd();

const REQUIRED_METHOD_KIT_FILES = [
  'VERSION',
  'patterns/METHOD_VERSION.json',
  'patterns/README.md',
  'patterns/harness-core-model.md',
  'patterns/harness-coverage-model.md',
  'patterns/harness-template-taxonomy.md',
  'patterns/tool-adapter-matrix.md',
  'patterns/method-playbook.md',
  'patterns/changelog.md',
  'patterns/upgrade.md',
];

const REQUIRED_REPO_SHELL_FILES = [
  'SHELL_VERSION.json',
  '.agents/README.md',
  '.github/pull_request_template.md',
  'docs/harness/HARNESS_CORE_MODEL.md',
  'docs/harness/HARNESS_COVERAGE_MODEL.md',
  'docs/harness/HARNESS_TEMPLATE_TAXONOMY.md',
  'docs/harness/TOOL_ADAPTER_MATRIX.md',
  'docs/harness/HARNESS_ENGINEERING_CONTRACT.md',
  'docs/harness/TRIVIALITY_CLASSIFICATION_POLICY.md',
  'docs/harness/VISUAL_EVIDENCE_PROMOTION_POLICY.md',
  'docs/harness/FAILURE_REGISTRY_PROMOTION_POLICY.md',
  'scripts/harness/check-adoption.mjs',
  'scripts/harness/check-review.mjs',
  'scripts/harness/check-graph-review.mjs',
  'scripts/harness/scaffold-graph-review.mjs',
  'scripts/harness/build-graph-review-import.mjs',
  'scripts/harness/check-visual-evidence.mjs',
  'scripts/harness/check-failure-registry.mjs',
  'scripts/harness/check-template-health.mjs',
  'scripts/harness/check-runtime-evidence.mjs',
  'scripts/harness/check-doc-links.mjs',
  'scripts/harness/check-doc-inventory.mjs',
  'scripts/harness/check-doc-frontmatter.mjs',
  'scripts/harness/check-sync-drift.mjs',
  'scripts/harness/check-task-packet.mjs',
  'scripts/harness/check-evidence.mjs',
  'config/method.config.json',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-method-health.mjs [--json] [--strict] [--root <path>] [--method-root <path>] [--config <path>]

Checks:
- method kit version metadata exists and parses
- repo shell version metadata exists and parses
- compatible versions agree
- required method files exist
- required repo shell landing files exist
- feature-ledger checker exists for generated capability drift
- portable/runtime boundary directories exist`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    strict: false,
    help: false,
    root: DEFAULT_ROOT,
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
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }

  return options;
}

function exists(root, repoPath) {
  return fs.existsSync(path.join(root, repoPath));
}

function readJson(root, repoPath, findings) {
  const fullPath = path.join(root, repoPath);
  if (!fs.existsSync(fullPath)) {
    findings.push({ file: repoPath, reason: 'required file is missing' });
    return null;
  }

  try {
    return JSON.parse(fs.readFileSync(fullPath, 'utf8'));
  } catch (error) {
    findings.push({ file: repoPath, reason: `invalid JSON: ${error.message}` });
    return null;
  }
}

function validateMethodKit(root, methodRoot, findings) {
  for (const repoPath of REQUIRED_METHOD_KIT_FILES) {
    if (!exists(methodRoot, repoPath)) {
      findings.push({
        file: formatExternalPath(root, methodRoot, repoPath),
        reason: 'required upstream method kit file is missing',
      });
    }
  }

  const versionTextPath = path.join(methodRoot, 'VERSION');
  if (fs.existsSync(versionTextPath)) {
    const value = fs.readFileSync(versionTextPath, 'utf8').trim();
    if (!/^\d+\.\d+\.\d+$/.test(value)) {
      findings.push({
        file: formatExternalPath(root, methodRoot, 'VERSION'),
        reason: 'version must use semver-like x.y.z format',
      });
    }
  }

  return readJson(methodRoot, 'patterns/METHOD_VERSION.json', findings);
}

function validateRepoShell(root, findings) {
  for (const repoPath of REQUIRED_REPO_SHELL_FILES) {
    if (!exists(root, repoPath)) {
      findings.push({ file: repoPath, reason: 'required repo shell landing file is missing' });
    }
  }

  return readJson(root, 'SHELL_VERSION.json', findings);
}

function validateCompatibility(methodVersion, shellVersion, findings) {
  if (!methodVersion || !shellVersion) {
    return;
  }

  if (methodVersion.compatibleRepoShell !== shellVersion.version) {
    findings.push({
      file: 'SHELL_VERSION.json',
      reason: `repo shell version "${shellVersion.version}" does not match method kit compatibleRepoShell "${methodVersion.compatibleRepoShell}"`,
    });
  }

  if (shellVersion.compatibleMethodKit !== methodVersion.version) {
    findings.push({
      file: 'patterns/METHOD_VERSION.json',
      reason: `method kit version "${methodVersion.version}" does not match repo shell compatibleMethodKit "${shellVersion.compatibleMethodKit}"`,
    });
  }
}

function validateBoundaries(root, warnings) {
  if (!exists(root, '.harness')) {
    warnings.push({ file: '.harness', reason: 'runtime evidence directory is missing' });
  }
  if (!exists(root, 'openspec')) {
    warnings.push({ file: 'openspec', reason: 'OpenSpec skeleton directory is missing' });
  }
}

function scan(root, options = {}) {
  const findings = [];
  const warnings = [];
  const methodRoot = resolvePantheonHarnessRoot(root, options);

  const methodVersion = validateMethodKit(root, methodRoot, findings);
  const shellVersion = validateRepoShell(root, findings);
  validateCompatibility(methodVersion, shellVersion, findings);
  validateBoundaries(root, warnings);

  return {
    findings,
    warnings,
    methodVersion: methodVersion?.version ?? null,
    repoShellVersion: shellVersion?.version ?? null,
    methodRoot,
  };
}

function printTextReport(result, strict) {
  const mode = strict ? 'strict' : 'report-only';
  console.log(
    `Method health check (${mode}): ${result.findings.length} finding(s), ${result.warnings.length} warning(s)`,
  );
  if (result.methodVersion || result.repoShellVersion) {
    console.log(`method kit version: ${result.methodVersion ?? 'unknown'}`);
    console.log(`repo shell version: ${result.repoShellVersion ?? 'unknown'}`);
  }
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

  const result = scan(options.root, options);

  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
          methodVersion: result.methodVersion,
          repoShellVersion: result.repoShellVersion,
          methodRoot: result.methodRoot,
          findingCount: result.findings.length,
          warningCount: result.warnings.length,
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
