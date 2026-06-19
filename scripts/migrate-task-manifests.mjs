#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import {
  buildTaskManifestPath,
  ensureTrailingSlash,
} from './task-manifest.mjs';

const DEFAULT_ROOT = process.cwd();
const TASK_DOC_ROOT = path.join('docs', 'harness', 'tasks');
const RUNTIME_EVIDENCE_PATTERN =
  /\b(runtime|smoke|browser|playwright|screenshot|trace|metric|log|workflow|github|merge|branch|session|auth|permission|upload|import|export)\b/i;

function parseArgs(argv) {
  const options = {
    root: DEFAULT_ROOT,
    write: false,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    if (arg === '--write') {
      options.write = true;
    } else if (arg === '--root') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--root requires a path');
      }
      options.root = path.resolve(value);
      index += 1;
    } else {
      throw new Error(`unknown argument: ${arg}`);
    }
  }

  return options;
}

function discoverTaskPackets(root) {
  const taskDir = path.join(root, TASK_DOC_ROOT);
  if (!fs.existsSync(taskDir)) {
    return [];
  }
  return fs
    .readdirSync(taskDir)
    .filter((name) => name.endsWith('.task.md'))
    .sort((left, right) => left.localeCompare(right))
    .map((name) => path.join(taskDir, name));
}

function toRepoPath(root, filePath) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function parseFrontmatter(content) {
  if (!String(content).startsWith('---\n') && !String(content).startsWith('---\r\n')) {
    return {};
  }

  const lines = String(content).split(/\r?\n/);
  if (lines[0].trim() !== '---') {
    return {};
  }

  const frontmatter = {};
  for (let index = 1; index < lines.length; index += 1) {
    const line = lines[index];
    if (line.trim() === '---') {
      break;
    }
    const match = line.match(/^([A-Za-z0-9_-]+):\s*(.+)\s*$/);
    if (!match) {
      continue;
    }
    frontmatter[match[1]] = match[2].trim();
  }
  return frontmatter;
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
  if (!section) {
    return '';
  }
  const start = section.end;
  const next = headings.find(
    (heading) => heading.index > section.index && heading.level <= section.level,
  );
  const end = next ? next.index : content.length;
  return content.slice(start, end).trim();
}

function stripCodeFormatting(value) {
  return String(value ?? '')
    .trim()
    .replace(/^`+/, '')
    .replace(/`+$/, '')
    .trim();
}

function normalizeLineValue(value) {
  const normalized = unwrapWholeCodeSpan(String(value ?? '').trim());
  return normalized.toLowerCase() === 'none' ? '' : normalized;
}

function escapeRegExp(value) {
  return String(value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function getFirstMeaningfulLine(sectionContent) {
  return String(sectionContent ?? '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line && !line.startsWith('<!--'));
}

function extractBulletItems(sectionContent) {
  return String(sectionContent ?? '')
    .split(/\r?\n/)
    .map((line) => line.match(/^\s*[-*]\s+(.+?)\s*$/)?.[1] ?? '')
    .map((line) => normalizeLineValue(line))
    .filter(Boolean);
}

function extractChecklistItems(sectionContent) {
  return String(sectionContent ?? '')
    .split(/\r?\n/)
    .map((line) => line.match(/^\s*-\s+\[[ xX]\]\s+(.+?)\s*$/)?.[1] ?? '')
    .map((line) => normalizeLineValue(line))
    .filter(Boolean);
}

function extractParagraphLines(sectionContent) {
  return String(sectionContent ?? '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#') && !/^\s*[-*]\s+/.test(line));
}

function extractListOrParagraph(sectionContent) {
  const bullets = extractBulletItems(sectionContent);
  if (bullets.length > 0) {
    return bullets;
  }
  return extractParagraphLines(sectionContent);
}

function unwrapWholeCodeSpan(value) {
  const trimmed = String(value ?? '').trim();
  return /^`[^`]*`$/u.test(trimmed) ? trimmed.slice(1, -1).trim() : trimmed;
}

function extractNestedBulletItems(sectionContent, label) {
  const lines = String(sectionContent ?? '').split(/\r?\n/);
  const labelPattern = new RegExp(`^\\s*[-*]\\s+${escapeRegExp(label)}\\s*$`, 'u');
  const items = [];
  let collecting = false;

  for (const line of lines) {
    if (!collecting) {
      if (labelPattern.test(line.trimEnd())) {
        collecting = true;
      }
      continue;
    }

    if (!line.trim()) {
      continue;
    }

    const nestedMatch = line.match(/^\s{2,}[-*]\s+(.+?)\s*$/u);
    if (nestedMatch) {
      const normalized = normalizeLineValue(nestedMatch[1]);
      if (normalized) {
        items.push(normalized);
      }
      continue;
    }

    if (/^\s*[-*]\s+/u.test(line)) {
      break;
    }
  }

  return items;
}

function splitInlineValues(value) {
  let normalized = unwrapWholeCodeSpan(value);
  if (!normalized) {
    return [];
  }
  if (/`/.test(normalized)) {
    normalized = normalized.replace(/`([^`]+)`/g, '$1');
  }
  if (!normalized || normalized.toLowerCase() === 'none') {
    return [];
  }
  return normalized
    .split(/\s*\|\s*|\s*,\s*/u)
    .map((entry) => entry.trim())
    .filter(Boolean)
    .filter((entry) => entry.toLowerCase() !== 'none');
}

function parseKeyValueBullets(sectionContent) {
  const values = new Map();
  for (const line of String(sectionContent ?? '').split(/\r?\n/)) {
    const match = line.match(/^\s*[-*]\s+([^:]+):\s*(.+?)\s*$/);
    if (!match) {
      continue;
    }
    values.set(match[1].trim(), match[2].trim());
  }
  return values;
}

function readJsonIfExists(filePath) {
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

function readUtf8IfExists(filePath) {
  try {
    return fs.readFileSync(filePath, 'utf8');
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

function parseTaskPacket(taskPath, root) {
  const content = fs.readFileSync(taskPath, 'utf8');
  const frontmatter = parseFrontmatter(content);
  const headings = getHeadingMap(content);
  const topHeading = headings.find((heading) => heading.level === 1);
  const fileTaskId = path.basename(taskPath).replace(/\.task\.md$/, '');

  const goalSection = getSectionContent(content, headings, findSection(headings, 'Goal'));
  const primaryLayerSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Primary Layer'),
  );
  const dependencyLayersSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Dependency Layers'),
  );
  const harnessProfileSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Harness Profile'),
  );
  const contractAnchorsSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Contract Anchors'),
  );
  const scopeSection = getSectionContent(content, headings, findSection(headings, 'Scope'));
  const scopeHeadings = getHeadingMap(scopeSection);
  const expectedFilesSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Expected Files'),
  );
  const expectedFilesHeadings = getHeadingMap(expectedFilesSection);
  const implementationNotesSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Implementation Notes'),
  );
  const structuralScopeSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Structural Scope'),
  );
  const verificationPlanSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Verification Plan'),
  );
  const linkageSection = getSectionContent(content, headings, findSection(headings, 'Linkage'));
  const evidenceRequiredSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Evidence Required'),
  );
  const humanGatesSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Human Gates'),
  );
  const completionChecklistSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Completion Checklist'),
  );
  const methodReadinessSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Method Readiness'),
  );
  const deliveryGovernanceSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Delivery Governance'),
  );
  const executionRolesSection = getSectionContent(
    content,
    headings,
    findSection(headings, 'Execution Roles'),
  );

  const linkageValues = parseKeyValueBullets(linkageSection);
  const taskId = normalizeLineValue(linkageValues.get('Task ID')) || fileTaskId;
  const evidenceDir =
    normalizeLineValue(linkageValues.get('Evidence Directory')) ||
    `.harness/evidence/${taskId}/`;
  const reviewFile =
    normalizeLineValue(linkageValues.get('Review File')) ||
    `.harness/evidence/${taskId}/review.md`;
  const changeRef =
    normalizeLineValue(linkageValues.get('OpenSpec Change')) || 'none';
  const planRefValue =
    linkageValues.get('Superpowers Plan') ??
    linkageValues.get('Plan References') ??
    'none';
  const verificationCommands = extractBulletItems(verificationPlanSection);
  const evidenceRequired = extractBulletItems(evidenceRequiredSection);
  const runtimeEvidence = [...verificationCommands, ...evidenceRequired].filter((entry) =>
    RUNTIME_EVIDENCE_PATTERN.test(entry),
  );

  const harnessProfileValues = parseKeyValueBullets(harnessProfileSection);
  const methodReadinessValues = parseKeyValueBullets(methodReadinessSection);
  const deliveryGovernanceValues = parseKeyValueBullets(deliveryGovernanceSection);
  const executionRolesValues = parseKeyValueBullets(executionRolesSection);
  const structuralScopeValues = parseKeyValueBullets(structuralScopeSection);

  return {
    taskId,
    title:
      normalizeLineValue(frontmatter.title) ||
      normalizeLineValue(topHeading?.title?.replace(/^Task Packet:\s*/u, '')) ||
      taskId,
    goal: extractParagraphLines(goalSection).join(' ').trim() || normalizeLineValue(goalSection),
    primaryLayer: normalizeLineValue(getFirstMeaningfulLine(primaryLayerSection)),
    dependencyLayers: extractBulletItems(dependencyLayersSection),
    taskDoc: toRepoPath(root, taskPath),
    scope: {
      in: extractBulletItems(
        getSectionContent(scopeSection, scopeHeadings, findSection(scopeHeadings, 'In', 3)),
      ),
      out: extractBulletItems(
        getSectionContent(scopeSection, scopeHeadings, findSection(scopeHeadings, 'Out', 3)),
      ),
    },
    implementationNotes: extractListOrParagraph(implementationNotesSection),
    contractAnchors: extractBulletItems(contractAnchorsSection),
    expectedFiles: {
      create: extractBulletItems(
        getSectionContent(
          expectedFilesSection,
          expectedFilesHeadings,
          findSection(expectedFilesHeadings, 'Create', 3),
        ),
      ),
      modify: extractBulletItems(
        getSectionContent(
          expectedFilesSection,
          expectedFilesHeadings,
          findSection(expectedFilesHeadings, 'Modify', 3),
        ),
      ),
      doNotTouch: extractBulletItems(
        getSectionContent(
          expectedFilesSection,
          expectedFilesHeadings,
          findSection(expectedFilesHeadings, 'Do Not Touch', 3),
        ),
      ),
    },
    harnessProfile: {
      template: normalizeLineValue(harnessProfileValues.get('Template')),
      overlay: normalizeLineValue(harnessProfileValues.get('Overlay')) || 'none',
      qualityProfile: normalizeLineValue(harnessProfileValues.get('Quality Profile')),
      portableFailureClass: normalizeLineValue(
        harnessProfileValues.get('Portable Failure Class'),
      ),
      ownerLayer: normalizeLineValue(harnessProfileValues.get('Owner Layer')),
      coverageDimensions: extractNestedBulletItems(
        harnessProfileSection,
        'Coverage Dimensions:',
      ),
    },
    methodReadiness: {
      consumerSpecificControls: splitInlineValues(
        methodReadinessValues.get('Consumer-Specific Controls'),
      ),
      requiredSensors: splitInlineValues(methodReadinessValues.get('Required Sensors')),
      requiredEvidence: splitInlineValues(methodReadinessValues.get('Required Evidence')),
      ratchetDecision: normalizeLineValue(methodReadinessValues.get('Ratchet Decision')),
      deferredCodeIssues: splitInlineValues(methodReadinessValues.get('Deferred Code Issues')),
    },
    deliveryGovernance: {
      designGate: splitInlineValues(deliveryGovernanceValues.get('Design Gate')),
      developmentGate: splitInlineValues(deliveryGovernanceValues.get('Development Gate')),
      qaAcceptanceGate: splitInlineValues(deliveryGovernanceValues.get('QA Acceptance Gate')),
      githubGovernanceGate: splitInlineValues(
        deliveryGovernanceValues.get('GitHub Governance Gate'),
      ),
    },
    executionRoles:
      executionRolesSection
        ? {
            implementerPosture: normalizeLineValue(
              executionRolesValues.get('Implementer Posture'),
            ),
            reviewerPosture: splitInlineValues(
              executionRolesValues.get('Reviewer Posture'),
            ),
          }
        : undefined,
    structuralScope: {
      affectedSubgraph: splitInlineValues(
        structuralScopeValues.get('Affected Subgraph'),
      ),
      boundaryCrossings: splitInlineValues(
        structuralScopeValues.get('Boundary Crossings'),
      ),
      riskNodes: splitInlineValues(structuralScopeValues.get('Risk Nodes')),
      graphFocus: splitInlineValues(structuralScopeValues.get('Graph Focus')),
    },
    verificationPlan: {
      commands: verificationCommands,
      runtimeEvidence,
    },
    runtimeSensitive: runtimeEvidence.length > 0,
    linkage: {
      evidenceDir: ensureTrailingSlash(evidenceDir),
      reviewFile,
      summaryFile: `${ensureTrailingSlash(evidenceDir)}summary.md`,
      changeRef,
      planRefs: splitInlineValues(planRefValue),
    },
    evidenceRequired,
    humanGates: extractBulletItems(humanGatesSection),
    completionChecklist: extractChecklistItems(completionChecklistSection),
  };
}

function pruneEmptyArrays(value) {
  return Array.isArray(value) && value.length === 0;
}

function pruneEmptyObjects(value) {
  return value && typeof value === 'object' && !Array.isArray(value) && Object.keys(value).length === 0;
}

function cleanupManifest(manifest) {
  const next = structuredClone(manifest);
  for (const key of ['dependencyLayers', 'contractAnchors', 'implementationNotes', 'evidenceRequired', 'humanGates', 'completionChecklist']) {
    if (pruneEmptyArrays(next[key])) {
      delete next[key];
    }
  }
  if (next.executionRoles) {
    if (!next.executionRoles.implementerPosture) {
      delete next.executionRoles.implementerPosture;
    }
    if (pruneEmptyArrays(next.executionRoles.reviewerPosture)) {
      delete next.executionRoles.reviewerPosture;
    }
    if (pruneEmptyObjects(next.executionRoles)) {
      delete next.executionRoles;
    }
  }
  for (const sectionKey of ['harnessProfile', 'methodReadiness', 'deliveryGovernance', 'expectedFiles', 'structuralScope', 'verificationPlan']) {
    if (!next[sectionKey]) {
      continue;
    }
    for (const [fieldKey, fieldValue] of Object.entries(next[sectionKey])) {
      if (!fieldValue || pruneEmptyArrays(fieldValue) || pruneEmptyObjects(fieldValue)) {
        delete next[sectionKey][fieldKey];
      }
    }
    if (pruneEmptyObjects(next[sectionKey])) {
      delete next[sectionKey];
    }
  }
  return next;
}

function rewriteTaskPacketLinkage(taskPath, manifest) {
  const content = fs.readFileSync(taskPath, 'utf8');
  const manifestLine = `- Task Manifest: \`${buildTaskManifestPath(manifest.taskId)}\``;
  if (/^- Task Manifest:\s*`[^`]+`$/imu.test(content)) {
    const rewritten = content.replace(
      /^- Task Manifest:\s*`[^`]+`$/imu,
      manifestLine,
    );
    fs.writeFileSync(taskPath, rewritten, 'utf8');
    return;
  }
  const rewritten = content.replace(
    /^- Task ID:\s*`?[^`\r\n]+`?\s*$/imu,
    (line) => `${line}\n${manifestLine}`,
  );
  fs.writeFileSync(taskPath, rewritten, 'utf8');
}

function rewriteReviewLinkage(reviewPath, taskId, manifest) {
  const content = readUtf8IfExists(reviewPath);
  if (content === null) {
    return;
  }
  const manifestPath = buildTaskManifestPath(taskId);
  const evidencePath = `.harness/evidence/${taskId}/commands.json`;
  const match = content.match(/## Machine Readable\s+```json\s*([\s\S]*?)\s*```/m);
  let rewritten = content;
  if (match) {
    const payload = JSON.parse(match[1]);
    payload.taskId = taskId;
    if (payload.verdict === 'findings addressed') {
      payload.verdict = 'approved';
    }
    payload.linkage = {
      ...(payload.linkage ?? {}),
      taskManifest: manifestPath,
      evidence: evidencePath,
      reviewFile: `.harness/evidence/${taskId}/review.md`,
      changeRef: payload.linkage?.changeRef ?? manifest.linkage.changeRef,
      planRefs: payload.linkage?.planRefs ?? manifest.linkage.planRefs,
    };
    delete payload.linkage.taskPacket;
    rewritten = rewritten.replace(
      /## Machine Readable\s+```json\s*[\s\S]*?\s*```/m,
      ['## Machine Readable', '', '```json', JSON.stringify(payload, null, 2), '```'].join('\n'),
    );
  }

  rewritten = rewritten.replace(
    /^- Task Packet:\s*`[^`]+`$/gim,
    `- Task Manifest: \`${manifestPath}\``,
  );
  if (!/^- Task Manifest:\s*`[^`]+`$/imu.test(rewritten) && /^## Linkage$/imu.test(rewritten)) {
    rewritten = rewritten.replace(
      /^## Linkage$/imu,
      `## Linkage\n\n- Task Manifest: \`${manifestPath}\``,
    );
  }
  fs.writeFileSync(reviewPath, rewritten, 'utf8');
}

function rewritePrBody(prBodyPath, taskId) {
  let rewritten = readUtf8IfExists(prBodyPath);
  if (rewritten === null) {
    return;
  }
  const manifestPath = buildTaskManifestPath(taskId);

  rewritten = rewritten.replace(
    /^(\s*[-*]\s+)Task Packet([：:])\s*`?[^`\r\n]+`?\s*$/gim,
    `$1Task ID$2 \`${taskId}\``,
  );
  rewritten = rewritten.replace(
    /^(\s*[-*]\s+)task packet([：:])\s*`?[^`\r\n]+`?\s*$/gim,
    `$1task manifest$2 \`${manifestPath}\``,
  );

  if (!/^(\s*[-*]\s+)Task Manifest[：:]\s*`[^`]+`$/imu.test(rewritten)) {
    rewritten = rewritten.replace(
      /^(\s*[-*]\s+)Task ID([：:])\s*`[^`]+`\s*$/imu,
      `$&\n$1Task Manifest$2 \`${manifestPath}\``,
    );
  }

  if (!/^(\s*[-*]\s+)task manifest[：:]\s*`[^`]+`$/imu.test(rewritten)) {
    rewritten = rewritten.replace(
      /^(\s*[-*]\s+)task id([：:])\s*`[^`]+`\s*$/imu,
      `$&\n$1task manifest$2 \`${manifestPath}\``,
    );
  }

  fs.writeFileSync(prBodyPath, rewritten, 'utf8');
}

function migrate(root, write) {
  const results = [];
  for (const taskPath of discoverTaskPackets(root)) {
    const manifest = cleanupManifest(parseTaskPacket(taskPath, root));
    const manifestRepoPath = buildTaskManifestPath(manifest.taskId);
    const manifestAbsolutePath = path.join(root, manifestRepoPath);
    const commandsPath = path.join(root, `.harness/evidence/${manifest.taskId}/commands.json`);
    const reviewPath = path.join(root, `.harness/evidence/${manifest.taskId}/review.md`);
    const prBodyPath = path.join(root, `.harness/evidence/${manifest.taskId}/pr-body.md`);
    const commands = readJsonIfExists(commandsPath);

    if (commands?.linkage) {
      commands.taskId = manifest.taskId;
      commands.linkage.taskManifest = manifestRepoPath;
      commands.linkage.evidenceDir = ensureTrailingSlash(
        commands.linkage.evidenceDir ?? manifest.linkage.evidenceDir,
      );
      commands.linkage.reviewFile =
        commands.linkage.reviewFile ?? manifest.linkage.reviewFile;
      commands.linkage.summaryFile =
        commands.linkage.summaryFile ?? manifest.linkage.summaryFile;
      commands.linkage.changeRef =
        commands.linkage.changeRef ?? manifest.linkage.changeRef;
      commands.linkage.planRefs =
        commands.linkage.planRefs ?? manifest.linkage.planRefs;
      delete commands.linkage.taskPacket;
    }

    results.push({
      taskId: manifest.taskId,
      taskDoc: toRepoPath(root, taskPath),
      manifest: manifestRepoPath,
      commands: commands ? toRepoPath(root, commandsPath) : null,
      review: readUtf8IfExists(reviewPath) !== null ? toRepoPath(root, reviewPath) : null,
      prBody: readUtf8IfExists(prBodyPath) !== null ? toRepoPath(root, prBodyPath) : null,
    });

    if (!write) {
      continue;
    }

    fs.mkdirSync(path.dirname(manifestAbsolutePath), { recursive: true });
    fs.writeFileSync(manifestAbsolutePath, `${JSON.stringify(manifest, null, 2)}\n`, 'utf8');
    rewriteTaskPacketLinkage(taskPath, manifest);
    if (commands) {
      fs.writeFileSync(commandsPath, `${JSON.stringify(commands, null, 2)}\n`, 'utf8');
    }
    rewriteReviewLinkage(reviewPath, manifest.taskId, manifest);
    rewritePrBody(prBodyPath, manifest.taskId);
  }
  return results;
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  const result = migrate(options.root, options.write);
  console.log(JSON.stringify({ migrated: result.length, results: result }, null, 2));
}

main();
