#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const DEFAULT_ROOT = process.cwd();
const DEFAULT_CONFIG = 'config/ui-quality-gate.json';
const REQUIRED_ARTICLE_IDS = [4, 9, 23, 32, 33, 35, 138, 158, 164, 169, 173, 177, 180];
const REQUIRED_PRINCIPLES = ['clarity', 'consistency', 'efficiency', 'recoverability'];
const REQUIRED_VIEWPORTS = ['1440x900', '390x844'];
const REQUIRED_THEMES = ['dark', 'light'];
const REQUIRED_BASELINE_STATES = ['empty', 'error', 'forbidden', 'loading'];
const REQUIRED_OPERATIONAL_STATES = ['partial-failure', 'stale'];
const REQUIRED_ACCESSIBILITY = [
  'keyboard-path',
  'non-color-status',
  'reduced-motion',
  'screen-reader-name',
  'visible-focus',
  'zoom-200',
];
const REQUIRED_VISUAL_PLAN_FIELDS = ['accessibility', 'routes', 'states', 'themes', 'viewports'];

function parseArgs(argv) {
  const options = { root: DEFAULT_ROOT, config: DEFAULT_CONFIG, json: false, strict: false, help: false };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--root') {
      options.root = path.resolve(argv[++index] ?? '');
    } else if (arg === '--config') {
      options.config = argv[++index] ?? '';
    } else if (arg === '--json') {
      options.json = true;
    } else if (arg === '--strict') {
      options.strict = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else {
      throw new Error(`Unknown argument: ${arg}`);
    }
  }
  if (!options.root || !options.config) {
    throw new Error('--root and --config require values');
  }
  return options;
}

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-ui-quality-gate.mjs [--json] [--strict] [--root <path>] [--config <path>]

Checks the canonical UI policy, repository integration, and post-adoption UI task declarations.`);
}

function isObject(value) {
  return value !== null && typeof value === 'object' && !Array.isArray(value);
}

function readJson(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function normalizeStrings(values) {
  return Array.isArray(values)
    ? values.map((value) => String(value).trim()).filter(Boolean)
    : [];
}

function addFinding(findings, code, file, detail) {
  findings.push({ code, file, detail });
}

function requireMembers(findings, file, code, actual, required) {
  const values = new Set(normalizeStrings(actual));
  for (const value of required) {
    if (!values.has(value)) {
      addFinding(findings, code, file, `missing required value "${value}"`);
    }
  }
}

function validatePolicy(root, configPath, policy, findings) {
  if (!isObject(policy) || policy.schemaVersion !== 1) {
    addFinding(findings, 'ui_gate_schema_invalid', configPath, 'schemaVersion must be 1');
    return;
  }
  if (policy.owner !== 'pantheon-base') {
    addFinding(findings, 'ui_gate_owner_invalid', configPath, 'owner must be pantheon-base');
  }

  const canonicalDocument = String(policy.canonicalDocument ?? '').trim();
  if (!canonicalDocument || !fs.existsSync(path.join(root, canonicalDocument))) {
    addFinding(findings, 'ui_gate_anchor_missing', configPath, `canonical document does not exist: ${canonicalDocument}`);
  }

  const references = Array.isArray(policy.officialReferences) ? policy.officialReferences : [];
  const articleIds = new Set(references.map((reference) => reference?.articleId));
  for (const articleId of REQUIRED_ARTICLE_IDS) {
    if (!articleIds.has(articleId)) {
      addFinding(findings, 'ui_gate_reference_missing', configPath, `missing BK Design article ${articleId}`);
    }
  }
  for (const reference of references) {
    if (!String(reference?.url ?? '').startsWith('https://bkdesign.bk.tencent.com/design/')) {
      addFinding(findings, 'ui_gate_reference_unofficial', configPath, `article ${reference?.articleId ?? 'unknown'} must use the official BK Design URL`);
    }
  }

  requireMembers(findings, configPath, 'ui_gate_principle_missing', policy.principles?.map((entry) => entry?.id), REQUIRED_PRINCIPLES);
  requireMembers(findings, configPath, 'ui_gate_viewport_missing', policy.requiredMatrix?.viewports, REQUIRED_VIEWPORTS);
  requireMembers(findings, configPath, 'ui_gate_theme_missing', policy.requiredMatrix?.themes, REQUIRED_THEMES);
  requireMembers(findings, configPath, 'ui_gate_state_missing', policy.requiredMatrix?.baselineStates, REQUIRED_BASELINE_STATES);
  requireMembers(findings, configPath, 'ui_gate_state_missing', policy.requiredMatrix?.operationalStates, REQUIRED_OPERATIONAL_STATES);
  requireMembers(findings, configPath, 'ui_gate_accessibility_missing', policy.requiredMatrix?.accessibility, REQUIRED_ACCESSIBILITY);
  requireMembers(findings, configPath, 'ui_gate_task_contract_missing', policy.taskAdmission?.requiredVisualPlanFields, REQUIRED_VISUAL_PLAN_FIELDS);

  if (policy.taskAdmission?.effectiveTaskDate !== '2026-08-31') {
    addFinding(findings, 'ui_gate_adoption_date_invalid', configPath, 'effectiveTaskDate must remain 2026-08-31 or later changes need an explicit migration');
  }
  if (policy.adoption?.externalRuntimeDependency !== false || policy.adoption?.visualSkinCopied !== false) {
    addFinding(findings, 'ui_gate_dependency_boundary_invalid', configPath, 'BK Design must remain reference evidence only');
  }
  if (policy.enforcement?.humanGate?.owner !== 'maintainer') {
    addFinding(findings, 'ui_gate_human_owner_invalid', configPath, 'final visual gate owner must be maintainer');
  }

  const referencedFiles = [
    ...(policy.enforcement?.existingStaticChecks ?? []),
    policy.enforcement?.visualEvidenceChecker,
    policy.enforcement?.visualConfig,
    policy.enforcement?.visualSuite,
    policy.enforcement?.visualReviewGuide,
  ].filter(Boolean);
  for (const referencedFile of referencedFiles) {
    if (!fs.existsSync(path.join(root, referencedFile))) {
      addFinding(findings, 'ui_gate_enforcement_file_missing', configPath, `enforcement file does not exist: ${referencedFile}`);
    }
  }

  const visualConfig = String(policy.enforcement?.visualConfig ?? '').trim();
  if (visualConfig && fs.existsSync(path.join(root, visualConfig))) {
    const visualConfigSource = fs.readFileSync(path.join(root, visualConfig), 'utf8');
    requireMembers(
      findings,
      visualConfig,
      'ui_gate_visual_project_missing',
      normalizeStrings(policy.enforcement?.visualProjects).filter((projectName) => visualConfigSource.includes(`name: '${projectName}'`)),
      ['desktop-light', 'mobile-light', 'desktop-dark'],
    );
  }
}

function taskDate(taskId) {
  const match = /^(\d{4}-\d{2}-\d{2})/.exec(String(taskId ?? ''));
  return match?.[1] ?? null;
}

function listTaskManifests(root) {
  const taskRoot = path.join(root, '.harness', 'tasks');
  if (!fs.existsSync(taskRoot)) {
    return [];
  }
  return fs.readdirSync(taskRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => path.join(taskRoot, entry.name, 'manifest.json'))
    .filter((manifestPath) => fs.existsSync(manifestPath));
}

function validateTaskAdmission(root, policy, findings) {
  const effectiveDate = policy.taskAdmission?.effectiveTaskDate;
  const uiProfiles = new Set(normalizeStrings(policy.taskAdmission?.qualityProfiles));
  const requiredViewports = normalizeStrings(policy.requiredMatrix?.viewports);
  const requiredThemes = normalizeStrings(policy.requiredMatrix?.themes);
  const requiredStates = normalizeStrings(policy.requiredMatrix?.baselineStates);
  const requiredAccessibility = normalizeStrings(policy.requiredMatrix?.accessibility);

  for (const manifestPath of listTaskManifests(root)) {
    let manifest;
    try {
      manifest = readJson(manifestPath);
    } catch (error) {
      addFinding(findings, 'ui_gate_task_manifest_invalid', path.relative(root, manifestPath), error.message);
      continue;
    }
    const date = taskDate(manifest.taskId);
    if (!date || date < effectiveDate) {
      continue;
    }
    const profiles = new Set([
      ...normalizeStrings(manifest.qualityProfiles),
      ...normalizeStrings(manifest.methodReadiness?.qualityProfiles),
      String(manifest.qualityProfile ?? '').trim(),
    ]);
    if (![...profiles].some((profile) => uiProfiles.has(profile))) {
      continue;
    }

    const relativeManifest = path.relative(root, manifestPath).replaceAll(path.sep, '/');
    const visualPlan = manifest.verificationPlan?.visualEvidence;
    const exemption = manifest.verificationPlan?.visualEvidenceExemption;
    if (isObject(exemption)) {
      const exemptionValid =
        exemption.scope === policy.taskAdmission.governanceExemption?.scope &&
        exemption.noRenderedSurfaceChanged === true &&
        exemption.humanApprovalRequired === true &&
        String(exemption.reason ?? '').trim() !== '';
      if (!exemptionValid) {
        addFinding(findings, 'ui_gate_exemption_invalid', relativeManifest, 'governance exemption must include the fixed scope, reason, no-rendered-change assertion and human approval');
      }
      continue;
    }
    if (!isObject(visualPlan)) {
      addFinding(findings, 'ui_gate_visual_plan_missing', relativeManifest, 'UI task must declare verificationPlan.visualEvidence or a valid governance-only exemption');
      continue;
    }
    if (normalizeStrings(visualPlan.routes).length === 0) {
      addFinding(findings, 'ui_gate_route_missing', relativeManifest, 'visual plan must include at least one route or rendered fixture');
    }
    requireMembers(findings, relativeManifest, 'ui_gate_viewport_missing', visualPlan.viewports, requiredViewports);
    requireMembers(findings, relativeManifest, 'ui_gate_theme_missing', visualPlan.themes, requiredThemes);
    requireMembers(findings, relativeManifest, 'ui_gate_state_missing', visualPlan.states, requiredStates);
    requireMembers(findings, relativeManifest, 'ui_gate_accessibility_missing', visualPlan.accessibility, requiredAccessibility);
  }
}

function validateIntegration(root, findings) {
  const packagePath = path.join(root, 'package.json');
  const workflowPath = path.join(root, '.github', 'workflows', 'quality.yml');
  if (!fs.existsSync(packagePath) || readJson(packagePath).scripts?.['check:ui-quality-gate'] !== 'node scripts/harness/check-ui-quality-gate.mjs --root . --strict') {
    addFinding(findings, 'ui_gate_package_script_missing', 'package.json', 'check:ui-quality-gate must run the strict checker');
  }
  if (!fs.existsSync(workflowPath) || !fs.readFileSync(workflowPath, 'utf8').includes('npm run check:ui-quality-gate')) {
    addFinding(findings, 'ui_gate_workflow_step_missing', '.github/workflows/quality.yml', 'Docs Governance must run check:ui-quality-gate');
  }
}

function scan(root, configRelativePath) {
  const findings = [];
  const configPath = path.resolve(root, configRelativePath);
  const relativeConfig = path.relative(root, configPath).replaceAll(path.sep, '/');
  if (!fs.existsSync(configPath)) {
    addFinding(findings, 'ui_gate_config_missing', relativeConfig, 'UI quality gate config is missing');
    return { findings, checkedTaskCount: 0 };
  }
  let policy;
  try {
    policy = readJson(configPath);
  } catch (error) {
    addFinding(findings, 'ui_gate_config_invalid', relativeConfig, error.message);
    return { findings, checkedTaskCount: 0 };
  }
  validatePolicy(root, relativeConfig, policy, findings);
  validateTaskAdmission(root, policy, findings);
  validateIntegration(root, findings);
  findings.sort((left, right) => `${left.code}:${left.file}:${left.detail}`.localeCompare(`${right.code}:${right.file}:${right.detail}`));
  return { findings, checkedTaskCount: listTaskManifests(root).length };
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
  const result = scan(options.root, options.config);
  if (options.json) {
    console.log(JSON.stringify({ mode: options.strict ? 'strict' : 'report-only', findingCount: result.findings.length, ...result }, null, 2));
  } else {
    console.log(`UI quality gate (${options.strict ? 'strict' : 'report-only'}): ${result.findings.length} finding(s)`);
    for (const finding of result.findings) {
      console.log(`\nfinding: ${finding.code}\n  file: ${finding.file}\n  reason: ${finding.detail}`);
    }
    if (result.findings.length === 0) {
      console.log('\nno findings');
    }
  }
  return options.strict && result.findings.length > 0 ? 1 : 0;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  process.exitCode = main();
}

export { scan };
