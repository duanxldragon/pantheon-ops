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
  node scripts/harness/check-task-packet.mjs [--json] [--root <path>] [task-file ...]

Defaults:
  Scans <root>/docs/harness/tasks/*.task.md when no task files are provided.

Examples:
  node scripts/harness/check-task-packet.mjs
  node scripts/harness/check-task-packet.mjs --json
  node scripts/harness/check-task-packet.mjs --root /tmp/fixture
  node scripts/harness/check-task-packet.mjs docs/harness/tasks/example.task.md`);
}

function parseArgs(argv) {
  const options = {
    json: false,
    help: false,
    files: [],
    root: DEFAULT_ROOT,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--json') {
      options.json = true;
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

function printTextReport(results) {
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  console.log(`Task packet check: ${results.length} file(s), ${errorCount} error(s), ${warningCount} warning(s)`);

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
  const files =
    options.files.length > 0
      ? options.files.map((file) => normalizeInputFile(file, root))
      : discoverTaskFiles(root);

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

  const results = files.map((file) => validateTaskPacket(file, root));
  const errorCount = results.reduce((count, result) => count + result.errors.length, 0);
  const warningCount = results.reduce((count, result) => count + result.warnings.length, 0);

  if (options.json) {
    console.log(JSON.stringify({ results, errorCount, warningCount }, null, 2));
  } else {
    printTextReport(results);
  }

  return errorCount > 0 ? 1 : 0;
}

process.exitCode = main();
