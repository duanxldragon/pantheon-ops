import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  extractTaskIdFromManifestPath,
  normalizeRepoRelativePath,
  readTaskManifest,
} from './task-manifest.mjs';

const root = process.cwd();
const TEMPLATE_CANDIDATES = [
  path.join(root, '.github', 'PULL_REQUEST_TEMPLATE.md'),
  path.join(root, '.github', 'pull_request_template.md'),
];

const REQUIRED_SECTIONS = [
  '## Summary',
  '## Scope',
  '## Verification',
  '## Evidence',
  '## Review',
  '## Release Risk',
];

const REQUIRED_FIELDS = [
  { label: 'Target repo', allowExplicitNone: false },
  { label: 'Layer', allowExplicitNone: false },
  { label: 'Task mode', allowExplicitNone: false },
  { label: 'Sync expectation', allowExplicitNone: false },
  { label: 'In scope', allowExplicitNone: false },
  { label: 'Out of scope', allowExplicitNone: true },
  { label: 'Commands', allowExplicitNone: false },
  { label: 'Result', allowExplicitNone: false },
  { label: 'Task ID', allowExplicitNone: true },
  { label: 'Task Manifest', allowExplicitNone: true },
  { label: 'Evidence', allowExplicitNone: false },
  { label: 'Human gate', allowExplicitNone: true },
  { label: 'Review status', allowExplicitNone: false },
  { label: 'Review artifact', allowExplicitNone: false },
  { label: 'Known gaps', allowExplicitNone: true },
  { label: 'GitHub signal', allowExplicitNone: false },
];

const EXPLICIT_NONE = new Set(['none', 'n/a', 'na', 'not-applicable', 'not applicable']);
const PLACEHOLDER_PATTERNS = [
  /^<.*>$/i,
  /^\[.*fill.*\]$/i,
  /^(todo|tbd|pending)$/i,
];
const TASK_PACKET_TEMPLATE_PATHS = new Set([
  'docs/TASK_PACKET_OPS_TEMPLATE.md',
]);
const EVIDENCE_PLACEHOLDERS = new Set([
  'inline command summary',
  'screenshot path',
  'runtime gap',
  'smoke artifact',
]);
const REVIEW_ARTIFACT_PLACEHOLDERS = new Set([
  'inline review summary',
  'evidence/review note',
]);

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function normalizeValue(value) {
  return value.trim().replace(/^`+/, '').replace(/`+$/, '').trim();
}

function parseField(content, label) {
  const pattern = new RegExp(`^[\\-*]\\s+${escapeRegExp(label)}:\\s*(.+)$`, 'mi');
  const match = content.match(pattern);
  return match ? normalizeValue(match[1]) : null;
}

function isPlaceholder(value) {
  return PLACEHOLDER_PATTERNS.some((pattern) => pattern.test(value));
}

function hasExistingRepoFile(rootDir, value) {
  if (value.includes('://')) {
    return false;
  }
  const normalized = normalizeRepoRelativePath(value);
  if (!normalized || path.isAbsolute(value) || normalized.startsWith('..')) {
    return false;
  }
  return fs.existsSync(path.join(rootDir, normalized));
}

function extractTaskIdFromEvidencePath(value, suffix) {
  const pattern = new RegExp(`^\\.harness/evidence/(.+)/${suffix}$`, 'i');
  const match = normalizeRepoRelativePath(value).match(pattern);
  return match ? match[1] : null;
}

function validateArtifactLinkage(content, rootDir, findings) {
  const taskIdValue = parseField(content, 'Task ID');
  const taskManifestValue = parseField(content, 'Task Manifest');
  const evidenceValue = parseField(content, 'Evidence');
  if (taskIdValue !== null) {
    const normalizedTaskId = normalizeValue(taskIdValue);
    if (EXPLICIT_NONE.has(normalizedTaskId.toLowerCase())) {
      findings.push('Task ID must reference a concrete task identifier; explicit none is not allowed');
    } else if (!/^[A-Za-z0-9][A-Za-z0-9-]*$/.test(normalizedTaskId)) {
      findings.push('Task ID must be a normalized task identifier such as YYYY-MM-DD-task-name');
    }
  }
  if (taskManifestValue !== null) {
    const normalizedTaskManifest = normalizeRepoRelativePath(taskManifestValue);
    if (EXPLICIT_NONE.has(taskManifestValue.toLowerCase())) {
      findings.push('Task Manifest must reference a repository task manifest file; explicit none is not allowed');
    } else if (!/^\.harness\/tasks\/.+\/manifest\.json$/i.test(normalizedTaskManifest)) {
      findings.push('Task Manifest must reference .harness/tasks/<task-id>/manifest.json');
    } else if (!hasExistingRepoFile(rootDir, normalizedTaskManifest)) {
      findings.push(`Task Manifest file does not exist in repository: ${normalizedTaskManifest}`);
    } else {
      try {
        readTaskManifest(rootDir, normalizedTaskManifest);
      } catch (error) {
        findings.push(error.message);
      }
    }
  }

  if (evidenceValue !== null) {
    const normalizedEvidence = normalizeRepoRelativePath(evidenceValue);
    if (EVIDENCE_PLACEHOLDERS.has(evidenceValue.toLowerCase())) {
      findings.push('Evidence must reference a repository commands artifact, not an inline summary placeholder');
    } else if (!/^\.harness\/evidence\/.+\/commands\.json$/i.test(normalizedEvidence)) {
      findings.push('Evidence must reference .harness/evidence/<task-id>/commands.json');
    } else if (!hasExistingRepoFile(rootDir, normalizedEvidence)) {
      findings.push(`Evidence file does not exist in repository: ${normalizedEvidence}`);
    }
  }

  const reviewArtifactValue = parseField(content, 'Review artifact');
  if (reviewArtifactValue !== null) {
    const normalizedReviewArtifact = normalizeRepoRelativePath(reviewArtifactValue);
    if (REVIEW_ARTIFACT_PLACEHOLDERS.has(reviewArtifactValue.toLowerCase())) {
      findings.push('Review artifact must reference a repository review file, not an inline summary placeholder');
    } else if (!/^\.harness\/evidence\/.+\/review\.md$/i.test(normalizedReviewArtifact)) {
      findings.push('Review artifact must reference .harness/evidence/<task-id>/review.md');
    } else if (!hasExistingRepoFile(rootDir, normalizedReviewArtifact)) {
      findings.push(`Review artifact file does not exist in repository: ${normalizedReviewArtifact}`);
    }
  }

  const taskManifestTaskId = taskManifestValue
    ? extractTaskIdFromManifestPath(taskManifestValue)
    : null;
  const evidenceTaskId = evidenceValue ? extractTaskIdFromEvidencePath(evidenceValue, 'commands\\.json') : null;
  const reviewTaskId = reviewArtifactValue
    ? extractTaskIdFromEvidencePath(reviewArtifactValue, 'review\\.md')
    : null;
  const canonicalTaskId =
    (taskIdValue && !EXPLICIT_NONE.has(taskIdValue.toLowerCase()) ? normalizeValue(taskIdValue) : null) ??
    taskManifestTaskId;

  if (canonicalTaskId && taskManifestTaskId && canonicalTaskId !== taskManifestTaskId) {
    findings.push('Task ID and Task Manifest must reference the same task-id');
  }
  if (canonicalTaskId && evidenceTaskId && canonicalTaskId !== evidenceTaskId) {
    findings.push('Task ID and Evidence must reference the same task-id');
  }
  if (canonicalTaskId && reviewTaskId && canonicalTaskId !== reviewTaskId) {
    findings.push('Task ID and Review artifact must reference the same task-id');
  }
}

export function validatePrTemplate(content) {
  const findings = [];

  for (const section of REQUIRED_SECTIONS) {
    if (!content.includes(section)) {
      findings.push(`missing required section: ${section}`);
    }
  }

  for (const field of REQUIRED_FIELDS) {
    if (parseField(content, field.label) === null) {
      findings.push(`missing required field: ${field.label}`);
    }
  }

  return findings;
}

export function validatePrBody(content, options = {}) {
  const findings = [];
  const rootDir = options.rootDir ?? root;

  if (!content.trim()) {
    return ['pull request body is empty'];
  }

  for (const section of REQUIRED_SECTIONS) {
    if (!content.includes(section)) {
      findings.push(`missing required section in PR body: ${section}`);
    }
  }

  for (const field of REQUIRED_FIELDS) {
    const value = parseField(content, field.label);
    if (value === null) {
      findings.push(`missing required field in PR body: ${field.label}`);
      continue;
    }
    if (!value) {
      findings.push(`empty required field in PR body: ${field.label}`);
      continue;
    }
    if (isPlaceholder(value)) {
      findings.push(`placeholder value is not allowed for PR field: ${field.label}`);
      continue;
    }
    if (!field.allowExplicitNone && EXPLICIT_NONE.has(value.toLowerCase())) {
      findings.push(`explicit none is not allowed for PR field: ${field.label}`);
    }
  }

  validateArtifactLinkage(content, rootDir, findings);

  return findings;
}

export function resolveTemplatePath(templateCandidates = TEMPLATE_CANDIDATES) {
  for (const templatePath of templateCandidates) {
    if (fs.existsSync(templatePath)) {
      return templatePath;
    }
  }
  throw new Error(`PR template file does not exist: ${templateCandidates.join(', ')}`);
}

function readTemplate(templatePath = resolveTemplatePath()) {
  return fs.readFileSync(templatePath, 'utf8');
}

function readPrBodyFromEvent(eventPath) {
  if (!eventPath) {
    throw new Error('missing --event <github-event.json>');
  }
  const payload = JSON.parse(fs.readFileSync(eventPath, 'utf8'));
  return String(payload?.pull_request?.body ?? '');
}

function reportAndExit(findings, successMessage) {
  if (findings.length > 0) {
    console.error('Pantheon Ops PR governance check failed');
    for (const finding of findings) {
      console.error(`- ${finding}`);
    }
    process.exit(1);
  }
  console.log(successMessage);
}

function main(argv) {
  const args = [...argv];
  const eventFlagIndex = args.indexOf('--event');

  if (eventFlagIndex >= 0) {
    const eventPath = args[eventFlagIndex + 1];
    const body = readPrBodyFromEvent(eventPath);
    reportAndExit(validatePrBody(body), 'OK pantheon-ops PR body governance fields are present');
    return;
  }

  reportAndExit(
    validatePrTemplate(readTemplate()),
    'OK pantheon-ops PR template governance fields are present',
  );
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2));
}
