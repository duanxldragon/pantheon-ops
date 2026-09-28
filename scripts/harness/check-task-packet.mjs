#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import {
  buildTaskManifestPath,
  readTaskManifest,
} from '../task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();

const REQUIRED_SECTIONS = [
  'Goal',
  'Primary Layer',
  'Dependency Layers',
  'Harness Profile',
  'Contract Anchors',
  'Scope',
  'Expected Files',
  'Implementation Notes',
  'Verification Plan',
  'Linkage',
  'Evidence Required',
  'Human Gates',
  'Completion Checklist',
];

const REQUIRED_SUBSECTIONS = {
  Scope: ['In', 'Out'],
  'Expected Files': ['Create', 'Modify', 'Do Not Touch'],
};

const VALID_PRIMARY_LAYERS = new Set([
  'platform',
  'system/auth',
  'system/iam',
  'system/org',
  'system/config',
  'business/*',
  'app',
  'inheritance-sync',
]);

const VALID_REPOSITORY_ROLES = new Set([
  'method-source',
  'foundation-source',
  'business-consumer',
  'standalone',
]);
const VALID_SYNC_EXPECTATIONS = new Set(['none', 'not-required', 'deferred', 'required', 'completed']);
const VALID_RELEASE_REQUIREMENTS = new Set([
  'none',
  'method-release',
  'foundation-release',
  'consumer-lock-update',
]);

const VALID_HARNESS_TEMPLATES = new Set([
  'admin-platform',
  'api-service',
  'event-processor',
  'dashboard',
  'ui-heavy-product',
  'custom',
]);

const VALID_COVERAGE_DIMENSIONS = new Set([
  'behaviour',
  'maintainability',
  'architecture-fitness',
  'runtime-quality',
  'method-health',
]);

const REQUIRED_CHECKLIST_ITEMS = [
  'Layer and boundary declared',
  'Contract anchors read',
  'Verification run or exception recorded',
  'Evidence saved or summarized',
  'Review completed',
];

function printHelp() {
  console.log(`Usage:
  node scripts/harness/check-task-packet.mjs [--json] [--root <path>] [--legacy <path>]
                                         [--include-legacy] [task-file ...]

Defaults:
  Scans <root>/docs/harness/tasks/*.task.md when no task files are provided.
  Docs listed in the legacy allowlist (default <root>/config/task-packet-legacy-docs.json)
  are skipped, so a plain run reports whether current-format packets comply.
  A legacy entry that no longer exists is an error: the allowlist must shrink as
  docs are migrated, and it must never absorb a doc written to the template.

Examples:
  node scripts/harness/check-task-packet.mjs
  node scripts/harness/check-task-packet.mjs --json
  node scripts/harness/check-task-packet.mjs --root /tmp/fixture
  node scripts/harness/check-task-packet.mjs --include-legacy
  node scripts/harness/check-task-packet.mjs docs/harness/tasks/example.task.md`);
}

const DEFAULT_LEGACY_CONFIG = 'config/task-packet-legacy-docs.json';

// Legacy docs predate the section-based template. They are skipped by name so a
// plain run answers "do current packets comply?" instead of failing on history.
function loadLegacyDocs(configPath) {
  // A missing allowlist simply means "nothing is legacy": check every doc. That
  // keeps the checker usable against a bare fixture root, and deleting the file
  // re-exposes legacy docs loudly instead of hiding them.
  if (!fs.existsSync(configPath)) {
    return [];
  }

  const payload = JSON.parse(fs.readFileSync(configPath, 'utf8'));
  const entries = Array.isArray(payload.entries) ? payload.entries : [];
  for (const entry of entries) {
    if (typeof entry !== 'string' || entry.trim() === '') {
      throw new Error(`legacy allowlist entries must be non-empty strings: ${JSON.stringify(entry)}`);
    }
  }

  return entries.map((entry) => entry.replaceAll('\\', '/'));
}

function parseArgs(argv) {
  const options = {
    json: false,
    help: false,
    includeLegacy: false,
    legacyConfig: null,
    files: [],
    root: DEFAULT_ROOT,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
    } else if (arg === '--include-legacy') {
      options.includeLegacy = true;
    } else if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else if (arg === '--legacy') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--legacy requires a path');
      }
      options.legacyConfig = value;
      index += 1;
    } else {
      options.files.push(arg);
    }
  }

  return options;
}

function discoverTaskFiles(root) {
  const taskDir = path.join(root, 'docs', 'harness', 'tasks');
  if (!fs.existsSync(taskDir)) {
    return [];
  }

  return fs
    .readdirSync(taskDir)
    .filter((fileName) => fileName.endsWith('.task.md'))
    .sort((left, right) => left.localeCompare(right))
    .map((fileName) => path.join(taskDir, fileName));
}

function normalizeInputFile(inputPath, root) {
  return path.isAbsolute(inputPath) ? inputPath : path.join(root, inputPath);
}

function getHeadingMap(content) {
  const headings = [];
  const headingPattern = /^(#{1,6})\s+(.+?)\s*$/gm;
  let match;

  while ((match = headingPattern.exec(content)) !== null) {
    headings.push({
      level: match[1].length,
      title: match[2].trim(),
      index: match.index,
      end: headingPattern.lastIndex,
    });
  }

  return headings;
}

function findSection(headings, title, level = 2) {
  return headings.find((heading) => heading.level === level && heading.title === title);
}

function getSectionContent(content, headings, section) {
  const start = section.end;
  const next = headings.find(
    (heading) => heading.index > section.index && heading.level <= section.level,
  );
  const end = next ? next.index : content.length;
  return content.slice(start, end).trim();
}

function getFirstMeaningfulLine(sectionContent) {
  return sectionContent
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line && !line.startsWith('<!--'));
}

function hasListItem(sectionContent) {
  return /^[-*]\s+\S+/m.test(sectionContent);
}

function validateTaskPacket(filePath, root) {
  const relativePath = path.relative(root, filePath).replaceAll(path.sep, '/');
  const result = {
    file: relativePath,
    errors: [],
    warnings: [],
  };

  if (!fs.existsSync(filePath)) {
    result.errors.push('File does not exist.');
    return result;
  }

  const content = fs.readFileSync(filePath, 'utf8');
  const headings = getHeadingMap(content);
  const topHeading = headings.find((heading) => heading.level === 1);

  if (!topHeading || !topHeading.title.startsWith('Task Packet: ')) {
    result.errors.push('Missing top-level heading in the form "# Task Packet: <task-name>".');
  }

  for (const sectionTitle of REQUIRED_SECTIONS) {
    const section = findSection(headings, sectionTitle);
    if (!section) {
      result.errors.push(`Missing required section "## ${sectionTitle}".`);
      continue;
    }

    const sectionContent = getSectionContent(content, headings, section);
    if (!sectionContent) {
      result.errors.push(`Section "## ${sectionTitle}" is empty.`);
    }
  }

  for (const [sectionTitle, subsectionTitles] of Object.entries(REQUIRED_SUBSECTIONS)) {
    const section = findSection(headings, sectionTitle);
    if (!section) {
      continue;
    }

    const sectionContent = getSectionContent(content, headings, section);
    const sectionHeadings = getHeadingMap(sectionContent);

    for (const subsectionTitle of subsectionTitles) {
      const subsection = findSection(sectionHeadings, subsectionTitle, 3);
      if (!subsection) {
        result.errors.push(`Missing required subsection under "## ${sectionTitle}": "### ${subsectionTitle}".`);
      }
    }
  }

  validatePrimaryLayer(content, headings, result);
  validateWorkspaceContext(content, headings, result);
  validateHarnessProfile(content, headings, result);
  validateContractAnchors(content, headings, result, root);
  validateScope(content, headings, result);
  validateVerificationPlan(content, headings, result);
  validateOptionalStructuralScope(content, headings, result);
  validateLinkage(content, headings, result, root, filePath);
  validateOptionalExecutionRoles(content, headings, result);
  validateOptionalStopPoints(content, headings, result);
  validateOptionalStatePlan(content, headings, result);
  validateChecklist(content, headings, result);

  return result;
}

function validateWorkspaceContext(content, headings, result) {
  const primaryLayerSection = findSection(headings, 'Primary Layer');
  const primaryLayer = primaryLayerSection
    ? stripBackticks(getFirstMeaningfulLine(getSectionContent(content, headings, primaryLayerSection)) || '')
    : '';
  const required = primaryLayer === 'inheritance-sync';
  const section = findSection(headings, 'Workspace Context');

  if (!section) {
    if (required) {
      result.errors.push('Primary layer "inheritance-sync" requires section "## Workspace Context".');
    }
    return;
  }

  const items = parseLinkageItems(getSectionContent(content, headings, section));
  const requiredItems = [
    'Target Repository',
    'Repository Role',
    'Upstream Dependencies',
    'Downstream Consumers',
    'Sync Expectation',
    'Release Requirement',
  ];

  for (const item of requiredItems) {
    if (!items.has(item) || stripBackticks(items.get(item) || '') === '') {
      result.errors.push(`Section "## Workspace Context" must include "- ${item}: <value>".`);
    }
  }

  validateWorkspaceEnum(items, 'Repository Role', VALID_REPOSITORY_ROLES, result);
  validateWorkspaceEnum(items, 'Sync Expectation', VALID_SYNC_EXPECTATIONS, result);
  validateWorkspaceEnum(items, 'Release Requirement', VALID_RELEASE_REQUIREMENTS, result);
}

function validateWorkspaceEnum(items, key, allowed, result) {
  if (!items.has(key)) {
    return;
  }

  const value = stripBackticks(items.get(key) || '');
  if (value !== '' && !allowed.has(value)) {
    result.errors.push(
      `Workspace Context ${key} "${value}" must be one of: ${Array.from(allowed).join(', ')}.`,
    );
  }
}

function validateHarnessProfile(content, headings, result) {
  const section = findSection(headings, 'Harness Profile');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const templateMatch = sectionContent.match(/^- Template:\s*(.+)$/m);
  const overlayMatch = sectionContent.match(/^- Overlay:\s*(.+)$/m);
  const coverageHeaderMatch = sectionContent.match(/^- Coverage Dimensions:\s*$/m);

  if (!templateMatch) {
    result.errors.push('Section "## Harness Profile" must include "- Template: <template>".');
  } else {
    const template = stripBackticks(templateMatch[1]);
    if (!VALID_HARNESS_TEMPLATES.has(template)) {
      result.errors.push(
        `Invalid harness template "${template}". Expected one of: ${Array.from(VALID_HARNESS_TEMPLATES).join(', ')}.`,
      );
    }
  }

  if (!overlayMatch) {
    result.errors.push('Section "## Harness Profile" must include "- Overlay: <overlay>".');
  }

  if (!coverageHeaderMatch) {
    result.errors.push('Section "## Harness Profile" must include "- Coverage Dimensions:" with nested list items.');
    return;
  }

  const dimensions = [];
  const coverageBlock = sectionContent.slice(coverageHeaderMatch.index + coverageHeaderMatch[0].length);
  for (const match of coverageBlock.matchAll(/^\s{2,}[-*]\s+(.+)$/gm)) {
    dimensions.push(stripBackticks(match[1]));
  }

  if (dimensions.length === 0) {
    result.errors.push('Section "## Harness Profile" must list at least one coverage dimension.');
    return;
  }

  for (const dimension of dimensions) {
    if (!VALID_COVERAGE_DIMENSIONS.has(dimension)) {
      result.errors.push(
        `Invalid coverage dimension "${dimension}". Expected one of: ${Array.from(VALID_COVERAGE_DIMENSIONS).join(', ')}.`,
      );
    }
  }
}

function validatePrimaryLayer(content, headings, result) {
  const section = findSection(headings, 'Primary Layer');
  if (!section) {
    return;
  }

  const value = getFirstMeaningfulLine(getSectionContent(content, headings, section));
  if (!value) {
    return;
  }

  if (!VALID_PRIMARY_LAYERS.has(value)) {
    result.errors.push(
      `Invalid primary layer "${value}". Expected one of: ${Array.from(VALID_PRIMARY_LAYERS).join(', ')}.`,
    );
  }
}

function validateContractAnchors(content, headings, result, root) {
  const section = findSection(headings, 'Contract Anchors');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  if (!hasListItem(sectionContent)) {
    result.errors.push('Section "## Contract Anchors" must include at least one list item.');
    return;
  }

  const anchorPattern = /^[-*]\s+`([^`]+)`/gm;
  const anchors = [];
  let match;

  while ((match = anchorPattern.exec(sectionContent)) !== null) {
    anchors.push(match[1]);
  }

  if (anchors.length === 0) {
    result.warnings.push('Contract anchors should be listed as backticked file paths.');
    return;
  }

  for (const anchor of anchors) {
    const anchorPath = path.join(root, anchor);
    if (!fs.existsSync(anchorPath)) {
      result.warnings.push(`Contract anchor does not exist: ${anchor}`);
    }
  }
}

function validateScope(content, headings, result) {
  const section = findSection(headings, 'Scope');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const sectionHeadings = getHeadingMap(sectionContent);

  for (const subsectionTitle of ['In', 'Out']) {
    const subsection = findSection(sectionHeadings, subsectionTitle, 3);
    if (!subsection) {
      continue;
    }

    const subsectionContent = getSectionContent(sectionContent, sectionHeadings, subsection);
    if (!hasListItem(subsectionContent)) {
      result.errors.push(`Subsection "### ${subsectionTitle}" under "## Scope" must include at least one list item.`);
    }
  }
}

function validateVerificationPlan(content, headings, result) {
  const section = findSection(headings, 'Verification Plan');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  if (!hasListItem(sectionContent)) {
    result.errors.push('Section "## Verification Plan" must include at least one command or explicit none item.');
  }
}

function validateOptionalStructuralScope(content, headings, result) {
  const section = findSection(headings, 'Structural Scope');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const affectedSubgraphMatch = sectionContent.match(/^- Affected Subgraph:\s*(.+)$/m);
  const boundaryCrossingsMatch = sectionContent.match(/^- Boundary Crossings:\s*(.+)$/m);
  const riskNodesMatch = sectionContent.match(/^- Risk Nodes:\s*(.+)$/m);
  const graphFocusMatch = sectionContent.match(/^- Graph Focus:\s*(.+)$/m);

  if (!affectedSubgraphMatch || stripBackticks(affectedSubgraphMatch[1]) === '') {
    result.errors.push('Section "## Structural Scope" must include "- Affected Subgraph: <value>".');
  }

  if (!boundaryCrossingsMatch || stripBackticks(boundaryCrossingsMatch[1]) === '') {
    result.errors.push('Section "## Structural Scope" must include "- Boundary Crossings: <value>".');
  }

  if (!riskNodesMatch || stripBackticks(riskNodesMatch[1]) === '') {
    result.errors.push('Section "## Structural Scope" must include "- Risk Nodes: <value>".');
  }

  if (!graphFocusMatch || stripBackticks(graphFocusMatch[1]) === '') {
    result.errors.push('Section "## Structural Scope" must include "- Graph Focus: <value>".');
  }
}

function validateOptionalExecutionRoles(content, headings, result) {
  const section = findSection(headings, 'Execution Roles');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const implementerMatch = sectionContent.match(/^- Implementer Posture:\s*(.+)$/m);
  const reviewerMatch = sectionContent.match(/^- Reviewer Posture:\s*(.+)$/m);

  if (!implementerMatch || stripBackticks(implementerMatch[1]) === '') {
    result.errors.push('Section "## Execution Roles" must include "- Implementer Posture: <value>".');
  }

  if (!reviewerMatch || stripBackticks(reviewerMatch[1]) === '') {
    result.errors.push('Section "## Execution Roles" must include "- Reviewer Posture: <value>".');
  }
}

function validateOptionalStopPoints(content, headings, result) {
  const section = findSection(headings, 'Stop Points');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  if (!hasListItem(sectionContent)) {
    result.errors.push('Section "## Stop Points" must include at least one list item, even when the value is "none".');
  }
}

function validateOptionalStatePlan(content, headings, result) {
  const section = findSection(headings, 'State Plan');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const checkpointMatch = sectionContent.match(/^- Checkpoint Expectation:\s*(.+)$/m);

  if (!checkpointMatch || stripBackticks(checkpointMatch[1]) === '') {
    result.errors.push('Section "## State Plan" must include "- Checkpoint Expectation: <value>".');
  }
}

function parseLinkageItems(sectionContent) {
  const linkage = new Map();
  const lines = sectionContent.split(/\r?\n/);
  let currentKey = null;

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line) {
      continue;
    }

    const match = line.match(/^[-*]\s+([^:]+):\s*(.*)$/);
    if (match) {
      currentKey = match[1].trim();
      linkage.set(currentKey, (match[2] || '').trim());
      continue;
    }

    if (currentKey) {
      const previous = linkage.get(currentKey) || '';
      linkage.set(currentKey, `${previous}${line}`.trim());
    }
  }
  return linkage;
}

function stripBackticks(value) {
  return value.replace(/^`+/, '').replace(/`+$/, '').replace(/\s+/g, '').trim();
}

function validateLinkage(content, headings, result, root, filePath) {
  const section = findSection(headings, 'Linkage');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const linkage = parseLinkageItems(sectionContent);
  const requiredItems = [
    'Task ID',
    'Task Manifest',
    'OpenSpec Change',
    'Superpowers Plan',
    'Plan References',
    'Evidence Directory',
    'Review File',
  ];

  for (const item of requiredItems) {
    if (!linkage.has(item)) {
      result.warnings.push(`Section "## Linkage" is missing recommended item: ${item}.`);
    }
  }

  const expectedTaskId = path.basename(filePath).replace(/\.task\.md$/, '');
  const taskId = linkage.get('Task ID');
  if (taskId && stripBackticks(taskId) !== expectedTaskId) {
    result.errors.push(`Linkage Task ID "${stripBackticks(taskId)}" must match file name task id "${expectedTaskId}".`);
  }

  const taskManifest = linkage.get('Task Manifest');
  if (taskManifest) {
    const normalized = stripBackticks(taskManifest);
    const expectedManifest = buildTaskManifestPath(expectedTaskId);
    if (normalized !== expectedManifest) {
      result.errors.push(`Linkage Task Manifest must be "${expectedManifest}".`);
    } else {
      try {
        const manifest = readTaskManifest(root, normalized);
        if (manifest.payload.taskId !== expectedTaskId) {
          result.errors.push(
            `Linked task manifest task id "${manifest.payload.taskId}" must match "${expectedTaskId}".`,
          );
        }
      } catch (error) {
        result.errors.push(error.message);
      }
    }
  }

  const evidenceDir = linkage.get('Evidence Directory');
  if (evidenceDir) {
    const expectedDir = `.harness/evidence/${expectedTaskId}/`;
    if (stripBackticks(evidenceDir) !== expectedDir) {
      result.errors.push(`Linkage Evidence Directory must be "${expectedDir}".`);
    }
  }

  const reviewFile = linkage.get('Review File');
  if (reviewFile) {
    const normalized = stripBackticks(reviewFile);
    if (normalized !== 'none') {
      const expectedReview = `.harness/evidence/${expectedTaskId}/review.md`;
      if (normalized !== expectedReview) {
        result.errors.push(`Linkage Review File must be "${expectedReview}" or "none".`);
      } else if (!fs.existsSync(path.join(root, normalized))) {
        result.warnings.push(`Linked review file does not exist: ${normalized}`);
      }
    }
  }

  const changeRef = linkage.get('OpenSpec Change');
  if (changeRef) {
    const normalized = stripBackticks(changeRef);
    if (normalized !== 'none' && !fs.existsSync(path.join(root, normalized))) {
      result.warnings.push(`Linked OpenSpec change does not exist: ${normalized}`);
    }
  }

  const planRef = linkage.get('Superpowers Plan');
  if (planRef) {
    const normalized = stripBackticks(planRef);
    if (normalized !== 'none' && !fs.existsSync(path.join(root, normalized))) {
      result.warnings.push(`Linked superpowers plan does not exist: ${normalized}`);
    }
  }
}

function validateChecklist(content, headings, result) {
  const section = findSection(headings, 'Completion Checklist');
  if (!section) {
    return;
  }

  const sectionContent = getSectionContent(content, headings, section);
  const checklistPattern = /^-\s+\[[ xX]\]\s+(.+)$/gm;
  const checklistItems = [];
  let match;

  while ((match = checklistPattern.exec(sectionContent)) !== null) {
    checklistItems.push(match[1].trim());
  }

  if (checklistItems.length === 0) {
    result.errors.push('Section "## Completion Checklist" must include checkbox items.');
    return;
  }

  for (const requiredItem of REQUIRED_CHECKLIST_ITEMS) {
    const found = checklistItems.some((item) => item === requiredItem);
    if (!found) {
      result.errors.push(`Completion checklist is missing required item: ${requiredItem}`);
    }
  }
}

function toRepoRelative(file, root) {
  return path.relative(root, file).replaceAll(path.sep, '/');
}

function resultsFor(files, root) {
  return files.map((file) => validateTaskPacket(file, root));
}

function report(results, { legacySkippedCount, root, configPath, json }) {
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  if (json) {
    console.log(
      JSON.stringify(
        {
          results,
          errorCount,
          warningCount,
          legacySkippedCount,
          legacyConfig: configPath ? path.relative(root, configPath).replaceAll(path.sep, '/') : null,
        },
        null,
        2,
      ),
    );
  } else {
    printTextReport(results, { legacySkippedCount, root, configPath });
  }

  // Exit semantics are unchanged from before the allowlist existed: any error
  // fails the run, so a caller cannot mistake a red report for a pass.
  return errorCount > 0 ? 1 : 0;
}

function printTextReport(results, { legacySkippedCount = 0, root = DEFAULT_ROOT, configPath = null } = {}) {
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  console.log(`Task packet check: ${results.length} file(s), ${errorCount} error(s), ${warningCount} warning(s)`);
  if (legacySkippedCount > 0 && configPath) {
    console.log(
      `${legacySkippedCount} legacy doc(s) skipped via ${path.relative(root, configPath).replaceAll(path.sep, '/')} (use --include-legacy to check them)`,
    );
  }

  for (const result of results) {
    const status = result.errors.length > 0 ? 'FAIL' : 'PASS';
    console.log(`\n[${status}] ${result.file}`);

    for (const error of result.errors) {
      console.log(`  error: ${error}`);
    }

    for (const warning of result.warnings) {
      console.log(`  warning: ${warning}`);
    }
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

  const root = options.root;
  const explicitFiles = options.files.length > 0;
  const files = explicitFiles
    ? options.files.map((file) => normalizeInputFile(file, root))
    : discoverTaskFiles(root);

  // An explicit file argument always gets checked, so a single legacy doc can
  // still be inspected by name.
  let legacyAllowlist = [];
  let skippedLegacy = [];
  if (!explicitFiles && !options.includeLegacy) {
    const configPath = options.legacyConfig
      ? normalizeInputFile(options.legacyConfig, root)
      : path.join(root, DEFAULT_LEGACY_CONFIG);
    try {
      legacyAllowlist = loadLegacyDocs(configPath);
    } catch (error) {
      console.error(error.message);
      return 1;
    }

    const legacySet = new Set(legacyAllowlist);
    skippedLegacy = files.filter((file) => legacySet.has(toRepoRelative(file, root)));
    const checkedFiles = files.filter((file) => !legacySet.has(toRepoRelative(file, root)));

    // A stale entry means the allowlist is drifting out of date; fail on it so the
    // exemption list cannot quietly outlive the docs it excuses.
    const discoveredKeys = new Set(files.map((file) => toRepoRelative(file, root)));
    const staleLegacy = legacyAllowlist.filter((entry) => !discoveredKeys.has(entry));
    if (staleLegacy.length > 0) {
      const messages = staleLegacy.map((entry) => `legacy allowlist entry no longer exists (remove it): ${entry}`);
      if (options.json) {
        console.log(JSON.stringify({ errorCount: messages.length, warningCount: 0, legacySkippedCount: 0, results: messages.map((message) => ({ file: configPath, errors: [message], warnings: [] })) }, null, 2));
      } else {
        console.error(`Task packet check: ${messages.length} stale legacy allowlist entr(ies) in ${path.relative(root, configPath).replaceAll(path.sep, '/')}`);
        for (const message of messages) {
          console.error(`  error: ${message}`);
        }
      }
      return 1;
    }

    return report(resultsFor(checkedFiles, root), {
      legacySkippedCount: skippedLegacy.length,
      root,
      configPath,
      json: options.json,
    });
  }

  if (files.length === 0) {
    const result = {
      file: path
        .relative(root, path.join(root, 'docs', 'harness', 'tasks'))
        .replaceAll(path.sep, '/'),
      errors: ['No task packet files found.'],
      warnings: [],
    };

    if (options.json) {
      console.log(JSON.stringify({ results: [result], errorCount: 1, warningCount: 0 }, null, 2));
    } else {
      printTextReport([result]);
    }
    return 1;
  }

  return report(resultsFor(files, root), {
    legacySkippedCount: 0,
    root,
    configPath: null,
    json: options.json,
  });
}

process.exitCode = main();
