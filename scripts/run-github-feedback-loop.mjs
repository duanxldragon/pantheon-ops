import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import { fileURLToPath } from 'node:url';
import {
  buildGhOperationSpecs,
  executeGhOperationSpecs,
  parseJsonFile,
  planWritebackActions,
} from './address-github-feedback.mjs';
import {
  buildTaskManifestPath,
  deriveTaskContextFromManifest,
  normalizeRepoRelativePath,
  readTaskManifest,
  resolveRepoPath,
} from './task-manifest.mjs';

const BODY_FIELD_LABELS = {
  taskId: ['Task ID'],
  taskManifest: ['Task Manifest'],
  evidence: ['Evidence'],
  reviewArtifact: ['Review Artifact', 'Review artifact'],
};

const CLOSE_ISSUE_PATTERN =
  /\b(close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)\s+(?:#(\d+)|https:\/\/github\.com\/([^/\s]+)\/([^/\s]+)\/issues\/(\d+))/gi;

function ensureArray(value) {
  return Array.isArray(value) ? value : [];
}

function toLines(value) {
  return ensureArray(value).map((entry) => String(entry).trim()).filter(Boolean);
}

function uniqueLines(values) {
  return [...new Set(toLines(values))];
}

function formatBulletSummary(lines) {
  return toLines(lines)
    .map((line) => `- ${line}`)
    .join('\n');
}

function normalizeRepoFullName(repoFullName) {
  return String(repoFullName ?? '').trim().toLowerCase();
}

function readFileIfExists(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return null;
  }
  return fs.readFileSync(filePath, 'utf8');
}

function parseJsonFileIfExists(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return null;
  }
  return parseJsonFile(filePath, filePath);
}

function bodyFieldPattern(label) {
  return new RegExp(`^[\\-*]\\s+${label.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}[：:]\\s*(.+)$`, 'mi');
}

function parseBodyField(body, labels) {
  const content = String(body ?? '');
  for (const label of labels) {
    const match = content.match(bodyFieldPattern(label));
    if (match?.[1]) {
      return normalizeRepoRelativePath(match[1]);
    }
  }
  return null;
}

function resolveArtifactPath(repoRoot, relativePath) {
  return resolveRepoPath(repoRoot, relativePath);
}

function findCommandsJsonCandidates(rootDir) {
  const evidenceRoot = path.join(rootDir, '.harness', 'evidence');
  if (!fs.existsSync(evidenceRoot)) {
    return [];
  }

  const candidates = [];

  function walk(currentDir) {
    for (const entry of fs.readdirSync(currentDir, { withFileTypes: true })) {
      const fullPath = path.join(currentDir, entry.name);
      if (entry.isDirectory()) {
        walk(fullPath);
        continue;
      }
      if (entry.isFile() && entry.name === 'commands.json') {
        const stat = fs.statSync(fullPath);
        candidates.push({
          filePath: fullPath,
          mtimeMs: stat.mtimeMs,
        });
      }
    }
  }

  walk(evidenceRoot);
  candidates.sort((left, right) => right.mtimeMs - left.mtimeMs);
  return candidates;
}

function extractMarkdownSection(content, headingLevel, heading) {
  const lines = String(content ?? '').split(/\r?\n/);
  const targetHeading = `${'#'.repeat(headingLevel)} ${heading}`.trim().toLowerCase();
  let startIndex = -1;

  for (let index = 0; index < lines.length; index += 1) {
    if (lines[index].trim().toLowerCase() === targetHeading) {
      startIndex = index + 1;
      break;
    }
  }

  if (startIndex < 0) {
    return '';
  }

  const sectionLines = [];
  for (let index = startIndex; index < lines.length; index += 1) {
    if (/^#{1,6}\s+/.test(lines[index].trim())) {
      break;
    }
    sectionLines.push(lines[index]);
  }

  return sectionLines.join('\n').trim();
}

function extractMarkdownBullets(content) {
  return String(content ?? '')
    .split(/\r?\n/)
    .map((line) => line.match(/^\s*-\s+(.+?)\s*$/)?.[1]?.trim() ?? '')
    .filter(Boolean);
}

function extractMarkdownParagraph(content) {
  return String(content ?? '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#') && !line.startsWith('-') && !line.startsWith('|'))
    .join(' ')
    .trim();
}

function buildScopedSummary(prefix, bullets) {
  const lines = uniqueLines(bullets);
  if (lines.length === 0) {
    return '';
  }
  if (lines.length === 1) {
    return lines[0];
  }
  return `${prefix}: ${lines.join('; ')}`;
}

function extractCloseIssueNumbers(body, repoFullName) {
  const targetRepo = normalizeRepoFullName(repoFullName);
  const issueNumbers = [];
  const content = String(body ?? '');

  for (const match of content.matchAll(CLOSE_ISSUE_PATTERN)) {
    const shorthandNumber = match[2];
    const owner = match[3];
    const repo = match[4];
    const fullUrlNumber = match[5];

    if (shorthandNumber) {
      issueNumbers.push(Number.parseInt(shorthandNumber, 10));
      continue;
    }

    const matchedRepo = normalizeRepoFullName(`${owner}/${repo}`);
    if (matchedRepo !== targetRepo) {
      continue;
    }
    issueNumbers.push(Number.parseInt(fullUrlNumber, 10));
  }

  return [...new Set(issueNumbers.filter(Number.isInteger))];
}

function defaultOwnershipSummary(repoRoot) {
  const repoName = path.basename(repoRoot);
  if (repoName === 'pantheon-ops') {
    return 'This patch stays in pantheon-ops because it is repository governance and local PR closure logic, not shared platform runtime behavior.';
  }
  return 'This stays in pantheon-base because the PR automation gate is shared platform governance rather than a business overlay.';
}

function loadEvidenceArtifacts(snapshot, repoRoot) {
  const body = snapshot?.pullRequest?.body ?? '';
  const bodyTaskId = parseBodyField(body, BODY_FIELD_LABELS.taskId);
  const bodyTaskManifest = parseBodyField(body, BODY_FIELD_LABELS.taskManifest);
  const bodyEvidence = parseBodyField(body, BODY_FIELD_LABELS.evidence);
  const bodyReviewArtifact = parseBodyField(body, BODY_FIELD_LABELS.reviewArtifact);

  const commandsJsonPath = resolveArtifactPath(repoRoot, bodyEvidence);
  const commandsEvidence =
    parseJsonFileIfExists(commandsJsonPath) ??
    (() => {
      const fallback = findCommandsJsonCandidates(repoRoot)[0];
      return fallback ? parseJsonFileIfExists(fallback.filePath) : null;
    })();

  const taskId =
    bodyTaskId ??
    (typeof commandsEvidence?.taskId === 'string' ? commandsEvidence.taskId.trim() : '') ??
    '';
  const taskManifestRelativePath =
    bodyTaskManifest ||
    normalizeRepoRelativePath(commandsEvidence?.linkage?.taskManifest ?? '') ||
    (taskId ? buildTaskManifestPath(taskId) : '');
  const reviewRelativePath =
    bodyReviewArtifact ??
    normalizeRepoRelativePath(commandsEvidence?.linkage?.reviewFile ?? '');

  const resolvedCommandsJsonPath =
    commandsJsonPath ??
    (commandsEvidence
      ? path.join(
          repoRoot,
          normalizeRepoRelativePath(commandsEvidence.linkage?.evidenceDir ?? ''),
          'commands.json',
        )
      : null);
  let taskManifest = null;
  if (taskManifestRelativePath) {
    try {
      taskManifest = readTaskManifest(repoRoot, taskManifestRelativePath);
    } catch {
      taskManifest = null;
    }
  }
  const reviewPath = resolveArtifactPath(repoRoot, reviewRelativePath);
  const summaryPath =
    resolveArtifactPath(
      repoRoot,
      normalizeRepoRelativePath(commandsEvidence?.linkage?.summaryFile ?? ''),
    ) ??
    (resolvedCommandsJsonPath &&
    fs.existsSync(path.join(path.dirname(resolvedCommandsJsonPath), 'summary.md'))
      ? path.join(path.dirname(resolvedCommandsJsonPath), 'summary.md')
      : null);

  return {
    taskId,
    taskManifest,
    commandsEvidence,
    commandsJsonPath: resolvedCommandsJsonPath,
    reviewPath,
    reviewContent: readFileIfExists(reviewPath),
    summaryPath,
    summaryContent: readFileIfExists(summaryPath),
  };
}

function deriveVerificationSummary(commandsEvidence, reviewContent) {
  const passedCommands = ensureArray(commandsEvidence?.commands)
    .filter((entry) => String(entry?.status ?? '').toLowerCase() === 'passed')
    .map((entry) => String(entry.command ?? '').trim())
    .filter(Boolean);

  if (passedCommands.length > 0) {
    return [...new Set(passedCommands)];
  }

  const verificationSection = extractMarkdownSection(reviewContent, 2, 'Verification Checked');
  return uniqueLines(extractMarkdownBullets(verificationSection));
}

function deriveChangeSummary(taskManifest) {
  if (!taskManifest?.payload) {
    return [];
  }
  return deriveTaskContextFromManifest(taskManifest.payload).changeSummary;
}

function deriveOwnershipSummary(taskManifest, repoRoot) {
  const preferred = taskManifest?.payload
    ? deriveTaskContextFromManifest(taskManifest.payload).ownershipSummary
    : '';
  return preferred || defaultOwnershipSummary(repoRoot);
}

function deriveOutOfScopeSummary(taskManifest, commandsEvidence) {
  const manifestSummary = taskManifest?.payload
    ? deriveTaskContextFromManifest(taskManifest.payload).outOfScopeSummary
    : '';
  if (manifestSummary) {
    return manifestSummary;
  }

  const knownGaps = uniqueLines(commandsEvidence?.knownGaps);
  if (knownGaps.length > 0) {
    return `Follow-up work remains outside this change: ${knownGaps.join('; ')}`;
  }

  return 'Follow-up work remains outside the current PR scope.';
}

export function deriveFeedbackContext(snapshot, options = {}) {
  const repoRoot = path.resolve(options.repoRoot ?? process.cwd());
  const {
    taskManifest,
    commandsEvidence,
    reviewContent,
  } = loadEvidenceArtifacts(snapshot, repoRoot);

  return {
    changeSummary: deriveChangeSummary(taskManifest),
    verificationSummary: deriveVerificationSummary(commandsEvidence, reviewContent),
    ownershipSummary: deriveOwnershipSummary(taskManifest, repoRoot),
    outOfScopeSummary: deriveOutOfScopeSummary(taskManifest, commandsEvidence),
    closeIssueNumbers: extractCloseIssueNumbers(snapshot?.pullRequest?.body ?? '', snapshot?.repoFullName),
    resolveOutOfScopeReviewThreads: false,
  };
}

function buildActionableReply(item, context) {
  const summary = formatBulletSummary(context.changeSummary);
  const verification = formatBulletSummary(context.verificationSummary);
  if (!summary || !verification) {
    return null;
  }
  return `Handled in the latest update.\n\nWhat changed:\n${summary}\n\nVerification:\n${verification}`;
}

function buildExplanationReply(item, context) {
  const ownershipSummary = String(context.ownershipSummary ?? '').trim();
  if (!ownershipSummary) {
    return null;
  }
  return `${ownershipSummary}\n\nVerification:\n${formatBulletSummary(context.verificationSummary)}`;
}

function buildOutOfScopeReply(item, context) {
  const outOfScopeSummary = String(context.outOfScopeSummary ?? '').trim();
  if (!outOfScopeSummary) {
    return null;
  }
  return `${outOfScopeSummary}\n\nCurrent scope:\n${formatBulletSummary(context.changeSummary)}`;
}

export function deriveAutomatedWritebackPlan(snapshot, context) {
  const plan = {
    replies: {},
    resolveReviewThreadIds: [],
    markRepliedDiscussionItemIdsAsAnswer: [],
    markDiscussionAnswerIds: [],
    closeIssueNumbers: ensureArray(context.closeIssueNumbers),
  };
  const blockedItems = [];

  for (const item of snapshot.items ?? []) {
    if (item.classification === 'closed') {
      continue;
    }

    if (item.classification === 'actionable-change') {
      const reply = buildActionableReply(item, context);
      if (!reply) {
        blockedItems.push({
          id: item.id,
          reason: 'missing changeSummary or verificationSummary for actionable-change reply',
        });
        continue;
      }
      plan.replies[item.id] = reply;
      if (item.threadId) {
        plan.resolveReviewThreadIds.push(item.threadId);
      }
      continue;
    }

    if (item.classification === 'explanation-only') {
      const reply = buildExplanationReply(item, context);
      if (!reply) {
        blockedItems.push({
          id: item.id,
          reason: 'missing ownershipSummary for explanation-only reply',
        });
        continue;
      }
      plan.replies[item.id] = reply;
      continue;
    }

    if (item.classification === 'out-of-scope') {
      const reply = buildOutOfScopeReply(item, context);
      if (!reply) {
        blockedItems.push({
          id: item.id,
          reason: 'missing outOfScopeSummary for out-of-scope reply',
        });
        continue;
      }
      plan.replies[item.id] = reply;
      if (item.sourceType === 'discussion-comment' && !item.parentCommentId) {
        plan.markRepliedDiscussionItemIdsAsAnswer.push(item.id);
      }
      if (context.resolveOutOfScopeReviewThreads === true && item.threadId) {
        plan.resolveReviewThreadIds.push(item.threadId);
      }
      continue;
    }

    blockedItems.push({
      id: item.id,
      reason: `unsupported automated classification: ${item.classification}`,
    });
  }

  plan.resolveReviewThreadIds = [...new Set(plan.resolveReviewThreadIds)];
  plan.markRepliedDiscussionItemIdsAsAnswer = [
    ...new Set(plan.markRepliedDiscussionItemIdsAsAnswer),
  ];
  plan.closeIssueNumbers = [...new Set(plan.closeIssueNumbers)];

  return { plan, blockedItems };
}

function parseArgs(argv) {
  const options = {
    dryRun: false,
    json: false,
    help: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    switch (arg) {
      case '--snapshot':
        options.snapshotPath = argv[index + 1];
        index += 1;
        break;
      case '--context':
        options.contextPath = argv[index + 1];
        index += 1;
        break;
      case '--dry-run':
        options.dryRun = true;
        break;
      case '--json':
        options.json = true;
        break;
      case '--help':
      case '-h':
        options.help = true;
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }

  if (options.help) {
    return options;
  }

  if (!options.snapshotPath) {
    throw new Error('missing --snapshot <snapshot.json>');
  }

  return options;
}

function printUsage() {
  console.log(`Usage:
  node scripts/run-github-feedback-loop.mjs --snapshot <snapshot.json> --context <context.json> --dry-run --json
  node scripts/run-github-feedback-loop.mjs --snapshot <snapshot.json> --dry-run --json
  node scripts/run-github-feedback-loop.mjs --snapshot <snapshot.json>

Options:
  --snapshot <path>    Snapshot JSON generated from fetch-github-feedback.
  --context <path>     Optional context JSON. When omitted, derive context from repo-local task and evidence artifacts.
  --dry-run            Print derived plan and operations without executing them.
  --json               Emit JSON output.
  --help               Show this help message.
`);
}

export function main(argv) {
  const options = parseArgs(argv);
  if (options.help) {
    printUsage();
    return;
  }

  const snapshot = parseJsonFile(options.snapshotPath, 'snapshot');
  const context = options.contextPath
    ? parseJsonFile(options.contextPath, 'context')
    : deriveFeedbackContext(snapshot, { repoRoot: process.cwd() });
  const contextSource = options.contextPath ? 'provided' : 'derived';
  const { plan, blockedItems } = deriveAutomatedWritebackPlan(snapshot, context);
  const actions = planWritebackActions(snapshot, plan);
  const operations = buildGhOperationSpecs(snapshot.repoFullName, actions);
  const output = {
    repoFullName: snapshot.repoFullName,
    pullRequest: snapshot.pullRequest ?? null,
    contextSource,
    context,
    blockedItems,
    plan,
    actions,
    operations,
  };

  if (options.dryRun) {
    console.log(JSON.stringify(output, null, 2));
    return;
  }

  if (blockedItems.length > 0) {
    throw new Error(`feedback loop blocked by ${blockedItems.length} unresolved automation gaps`);
  }

  const results = executeGhOperationSpecs(operations);
  const executedOutput = {
    ...output,
    results,
  };

  if (options.json) {
    console.log(JSON.stringify(executedOutput, null, 2));
    return;
  }

  console.log(`Applied ${results.length} GitHub feedback loop operations.`);
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2));
}
