#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

import { formatExternalPath, resolvePantheonHarnessRoot } from './upstream-root.mjs';

const DEFAULT_ROOT = process.cwd();
const REQUIRED_METHOD_DOCS = [
  'patterns/harness-core-model.md',
  'patterns/harness-coverage-model.md',
  'patterns/harness-template-taxonomy.md',
  'patterns/tool-adapter-matrix.md',
];
const REQUIRED_REPO_DOCS = [
  'docs/harness/HARNESS_CORE_MODEL.md',
  'docs/harness/HARNESS_COVERAGE_MODEL.md',
  'docs/harness/HARNESS_TEMPLATE_TAXONOMY.md',
  'docs/harness/TOOL_ADAPTER_MATRIX.md',
];

function parseArgs(argv) {
  const options = { json: false, strict: false, help: false, root: DEFAULT_ROOT, methodRoot: null, config: undefined };
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
    } else if (arg === '--method-root') {
      const value = argv[index + 1];
      if (!value) throw new Error('--method-root requires a path');
      options.methodRoot = value;
      index += 1;
    } else if (arg === '--config') {
      const value = argv[index + 1];
      if (!value) throw new Error('--config requires a path');
      options.config = value;
      index += 1;
    } else throw new Error(`Unknown argument: ${arg}`);
  }
  return options;
}

function printHelp() {
  console.log('Usage: node scripts/harness/check-template-health.mjs [--json] [--strict] [--root <path>]');
}

function exists(root, repoPath) {
  return fs.existsSync(path.join(root, repoPath));
}

function detectTemplateSelection(root) {
  const configPath = path.join(root, 'config', 'method.config.json');
  if (!fs.existsSync(configPath)) return false;
  try {
    const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
    return typeof config.templateId === 'string' && config.templateId.trim() !== '';
  } catch {
    return false;
  }
}

function detectTemplateReview(root) {
  return exists(root, 'docs/harness/template-health.md') || exists(root, 'docs/harness/template-selection.md');
}

function scan(root, options = {}) {
  const findings = [];
  const warnings = [];
  const methodRoot = resolvePantheonHarnessRoot(root, options);
  for (const repoPath of REQUIRED_METHOD_DOCS) {
    if (!exists(methodRoot, repoPath)) {
      findings.push({
        file: formatExternalPath(root, methodRoot, repoPath),
        reason: 'required upstream method template document is missing',
      });
    }
  }
  for (const repoPath of REQUIRED_REPO_DOCS) {
    if (!exists(root, repoPath)) findings.push({ file: repoPath, reason: 'required repo template projection document is missing' });
  }
  if (!detectTemplateSelection(root) && !detectTemplateReview(root)) {
    warnings.push({
      file: 'config/method.config.json',
      reason: 'no templateId selection and no template review artifact found',
    });
  }
  return { findings, warnings };
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
  const result = scan(options.root, options);
  if (options.json) console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', findingCount: result.findings.length, warningCount: result.warnings.length, findings: result.findings, warnings: result.warnings }, null, 2));
  else {
    console.log(`Template health check (${options.strict ? 'strict' : 'report-only'}): ${result.findings.length} finding(s), ${result.warnings.length} warning(s)`);
    for (const finding of result.findings) console.log(`finding: ${finding.file}\n  reason: ${finding.reason}`);
    for (const warning of result.warnings) console.log(`warning: ${warning.file}\n  reason: ${warning.reason}`);
  }
  return options.strict && result.findings.length > 0 ? 1 : 0;
}

process.exitCode = main();
