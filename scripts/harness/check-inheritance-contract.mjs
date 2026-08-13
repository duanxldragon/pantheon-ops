#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const DEFAULT_ROOT = process.cwd();

const REQUIRED_OPS_CLAUDE_MARKERS = [
  'pantheon-ops inherits pantheon-base as its foundation',
  'Business work reading order',
  '../docs/WORKSPACE_INHERITANCE.md',
  'docs/PROJECT_INHERITANCE.md',
  '../pantheon-base/DESIGN.md',
  '../pantheon-base/AGENTS.md',
  'Do not fix platform or system-domain drift locally in pantheon-ops',
];

const REQUIRED_PROJECT_INHERITANCE_MARKERS = [
  'Base repository: `../pantheon-base`',
  'Base version:',
  'business/cmdb',
  'business/deploy',
  'If a foundation rule must change, update `pantheon-base` first',
];

const REQUIRED_WORKSPACE_MARKERS = [
  '`pantheon-base`: the only authority',
  '`pantheon-ops`: a derived business repository',
  'Change base rules in `pantheon-base`, then let business repositories upgrade',
];

const REQUIRED_PR_MARKERS = [
  'Base/ops inheritance',
  'Base version checked',
  'generic drift',
  'pseudo-drift',
  'business-only',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-inheritance-contract.mjs [--json] [--strict] [--root <path>]

Checks that pantheon-ops keeps pantheon-base as the foundation source of truth:
- workspace inheritance rules exist
- pantheon-ops entrypoint declares derived-repo reading order
- PROJECT_INHERITANCE pins a base version and business-only scope
- PR template forces base/ops drift review`);
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

function readText(root, repoPath) {
  return fs.readFileSync(path.join(root, repoPath), 'utf8');
}

function requireFile(root, repoPath, findings) {
  if (!fs.existsSync(path.join(root, repoPath))) {
    findings.push({
      file: repoPath,
      reason: 'required inheritance contract file is missing',
    });
    return false;
  }
  return true;
}

function checkMarkers(root, repoPath, markers, findings) {
  if (!requireFile(root, repoPath, findings)) {
    return;
  }

  const content = readText(root, repoPath);
  for (const marker of markers) {
    if (!content.includes(marker)) {
      findings.push({
        file: repoPath,
        reason: `missing required inheritance marker: ${marker}`,
      });
    }
  }
}

function checkProjectInheritanceVersion(root, findings) {
  const repoPath = 'pantheon-ops/docs/PROJECT_INHERITANCE.md';
  if (!fs.existsSync(path.join(root, repoPath))) {
    return;
  }

  const content = readText(root, repoPath);
  const versionMatch = content.match(/Base version:\s*`([^`]+)`(?:\s*\(`([0-9a-f]{40})`\))?/i);
  if (!versionMatch) {
    findings.push({
      file: repoPath,
      reason: 'Base version must be pinned to a commit or tag',
    });
    return;
  }

  const shortVersion = versionMatch[1];
  const fullVersion = versionMatch[2];
  const shortLooksValid = /^[0-9a-f]{7,40}$/i.test(shortVersion) || /^v?\d+\.\d+\.\d+/.test(shortVersion);
  if (!shortLooksValid && !fullVersion) {
    findings.push({
      file: repoPath,
      reason: 'Base version must be an explicit tag or commit hash',
    });
  }
}

function scanInheritance(root) {
  const findings = [];
  const warnings = [];

  checkMarkers(root, 'docs/WORKSPACE_INHERITANCE.md', REQUIRED_WORKSPACE_MARKERS, findings);
  checkMarkers(root, 'pantheon-ops/CLAUDE.md', REQUIRED_OPS_CLAUDE_MARKERS, findings);
  checkMarkers(
    root,
    'pantheon-ops/docs/PROJECT_INHERITANCE.md',
    REQUIRED_PROJECT_INHERITANCE_MARKERS,
    findings,
  );
  checkProjectInheritanceVersion(root, findings);
  checkMarkers(root, '.github/pull_request_template.md', REQUIRED_PR_MARKERS, findings);

  const opsAgentsPath = path.join(root, 'pantheon-ops', 'AGENTS.md');
  if (fs.existsSync(opsAgentsPath)) {
    const content = fs.readFileSync(opsAgentsPath, 'utf8');
    if (!content.includes('pantheon-base') || !content.includes('business/*')) {
      warnings.push({
        file: 'pantheon-ops/AGENTS.md',
        reason: 'ops agent entrypoint should keep base inheritance and business-only scope visible',
      });
    }
  }

  return { findings, warnings };
}

function printTextReport(result, strict) {
  const mode = strict ? 'strict' : 'report-only';
  console.log(
    `Inheritance contract check (${mode}): ${result.findings.length} finding(s), ${result.warnings.length} warning(s)`,
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

  const result = scanInheritance(options.root);
  if (options.json) {
    console.log(
      JSON.stringify(
        {
          mode: options.strict ? 'strict' : 'report-only',
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
