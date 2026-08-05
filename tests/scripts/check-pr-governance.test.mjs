import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(
  path.resolve(testDir, '../../scripts/check-pr-governance.mjs'),
).href;

const {
  resolveTemplatePath,
  validatePrTemplate,
  validatePrBody,
} = await import(moduleUrl);

function withFixtureRepo(callback) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-ops-pr-governance-'));
  try {
    callback(repoRoot);
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
}

function writeTaskManifest(repoRoot, taskId) {
  const manifestPath = path.join(repoRoot, '.harness', 'tasks', taskId, 'manifest.json');
  fs.mkdirSync(path.dirname(manifestPath), { recursive: true });
  fs.writeFileSync(
    manifestPath,
    `${JSON.stringify({
      taskId,
      title: 'Ops governance follow-up',
      goal: 'Keep GitHub governance writeback manifest-first in pantheon-ops.',
      primaryLayer: 'business/deploy',
      scope: {
        in: ['Repository governance automation', 'Local PR closure logic'],
        out: ['Shared platform runtime changes'],
      },
      implementationNotes: [
        'This stays in pantheon-ops because it is repository governance and local PR closure logic.',
      ],
      linkage: {
        evidenceDir: `.harness/evidence/${taskId}/`,
        reviewFile: `.harness/evidence/${taskId}/review.md`,
        summaryFile: `.harness/evidence/${taskId}/summary.md`,
        changeRef: 'none',
        planRefs: [],
      },
      verificationPlan: {
        commands: ['node --test tests/scripts/check-pr-governance.test.mjs'],
        runtimeEvidence: [],
      },
      runtimeSensitive: false,
      evidenceRequired: ['commands.json', 'summary.md'],
      humanGates: ['none'],
      completionChecklist: [
        'Layer and boundary declared',
        'Contract anchors read',
        'Verification run or exception recorded',
        'Evidence saved or summarized',
        'Review completed',
      ],
    }, null, 2)}\n`,
    'utf8',
  );
}

const validTemplate = `## 变更摘要

- 改动层级：platform / business/*
- 改动模块：module
- 目标问题：problem
- 预期影响：impact

## Harness 链路

- Task ID：task-id
- Task Manifest：task-manifest
- Evidence：evidence
- Verification evidence：verification-evidence
- Review Artifact：review-artifact
- OpenSpec change：none
- Trivial change：yes / no
- Quality Profile：auth-security / permission-policy / i18n / ui-runtime / generator / ci-workflow / none
- Ratchet Decision：no-repeat-observed / guide-updated / sensor-added / gate-updated / template-updated / adapter-updated / registry-only
- GitHub Signal：method-gate / repo-quality-gate / runtime-evidence-gate / external-flaky / not-applicable

## Harness adoption markers

- task id: task-id
- task manifest: task-manifest
- evidence: evidence
- boundaries: boundaries
- backend response contract: none
- backend DTO contract: none
- permission contract: none
- audit coverage: none
- visual evidence: none
- inheritance contract: none
- base drift: none
- Base/ops inheritance: none

## 边界说明

- [ ] 本次改动仅涉及单一层级

## 验证记录

- [ ] GitHub required checks 通过

## 审核留痕

- Copilot review：requested
- CodeQL 结果：pending
- GitHub checks 结果：pending
- Auto-merge：not-enabled
- Duplication Gate 结果：report-only
- 是否高风险改动：no
- Residual risk / follow-up：none

## 检查清单

- [ ] 已明确本次改动归属
`;

function buildPrBody({
  taskId = '2026-06-17-sample',
  taskManifest = `.harness/tasks/${taskId}/manifest.json`,
  evidence = `.harness/evidence/${taskId}/commands.json`,
  verificationEvidence = `.harness/evidence/${taskId}/summary.md`,
  reviewArtifact = `.harness/evidence/${taskId}/review.md`,
  trivialChange = 'no',
  qualityProfile = 'ci-workflow',
  githubSignal = 'repo-quality-gate',
} = {}) {
  return `## 变更摘要

- 改动层级：\`business/deploy\`
- 改动模块：\`repository governance\`
- 目标问题：\`keep PR governance artifacts machine-linked\`
- 预期影响：\`repository checks only\`

## Harness 链路

- Task ID：\`${taskId}\`
- Task Manifest：\`${taskManifest}\`
- Evidence：\`${evidence}\`
- Verification evidence：\`${verificationEvidence}\`
- Review Artifact：\`${reviewArtifact}\`
- OpenSpec change：\`none\`
- Trivial change：\`${trivialChange}\`
- Quality Profile：\`${qualityProfile}\`
- Ratchet Decision：\`gate-updated\`
- GitHub Signal：\`${githubSignal}\`

## Harness adoption markers

- task id: \`${taskId}\`
- task manifest: \`${taskManifest}\`
- evidence: \`${evidence}\`
- boundaries: \`business governance only\`
- backend response contract: \`none\`
- backend DTO contract: \`none\`
- permission contract: \`none\`
- audit coverage: \`none\`
- visual evidence: \`none\`
- inheritance contract: \`none\`
- base drift: \`none\`
- Base/ops inheritance: \`none\`

## 边界说明

- [x] 本次改动仅涉及单一层级

## 验证记录

- [x] GitHub required checks 通过

## 审核留痕

- Copilot review：\`requested\`
- CodeQL 结果：\`pending\`
- GitHub checks 结果：\`pending\`
- Auto-merge：\`not-enabled\`
- Duplication Gate 结果：\`report-only\`
- 是否高风险改动：\`no\`
- Residual risk / follow-up：\`tracked in task manifest\`

## 检查清单

- [x] 已明确本次改动归属
`;
}

test('validatePrTemplate accepts the governance template structure', () => {
  assert.deepEqual(validatePrTemplate(validTemplate), []);
});

test('resolveTemplatePath prefers the uppercase GitHub template filename', () => {
  const resolved = resolveTemplatePath([
    path.resolve(testDir, '../../.github/PULL_REQUEST_TEMPLATE.md'),
    path.resolve(testDir, '../../.github/pull_request_template.md'),
  ]);
  assert.match(resolved, /PULL_REQUEST_TEMPLATE\.md$/);
});

test('validatePrBody rejects invalid required enums', () => {
  const findings = validatePrBody(buildPrBody({ trivialChange: 'none', githubSignal: '<signal>' }));
  assert.match(findings.join('\n'), /Trivial change/);
  assert.match(findings.join('\n'), /GitHub Signal/);
});

test('validatePrBody rejects template and inline artifact placeholders', () => {
  const findings = validatePrBody(buildPrBody({
    taskManifest: 'docs/TASK_PACKET_OPS_TEMPLATE.md',
    evidence: 'inline command summary',
    verificationEvidence: 'inline verification summary',
    reviewArtifact: 'inline review summary',
  }));
  assert.match(findings.join('\n'), /Task Manifest/);
  assert.match(findings.join('\n'), /Evidence/);
  assert.match(findings.join('\n'), /Verification evidence/);
  assert.match(findings.join('\n'), /Review Artifact/);
});

test('validatePrBody rejects missing artifact files', () => {
  const findings = validatePrBody(buildPrBody({ taskId: '2026-06-17-missing' }), {
    rootDir: path.resolve(testDir, '../..'),
  });
  assert.match(findings.join('\n'), /Task Manifest/);
  assert.match(findings.join('\n'), /Evidence/);
  assert.match(findings.join('\n'), /Verification evidence/);
  assert.match(findings.join('\n'), /Review Artifact/);
});

test('validatePrBody accepts existing manifest and evidence artifacts', () => {
  withFixtureRepo((repoRoot) => {
    const taskId = '2026-06-17-sample';
    writeTaskManifest(repoRoot, taskId);
    for (const [name, content] of [
      ['commands.json', `${JSON.stringify({ taskId, commands: [] })}\n`],
      ['summary.md', '# Verification Summary\n'],
      ['review.md', '# Review Summary\n'],
    ]) {
      const filePath = path.join(repoRoot, '.harness', 'evidence', taskId, name);
      fs.mkdirSync(path.dirname(filePath), { recursive: true });
      fs.writeFileSync(filePath, content, 'utf8');
    }
    assert.deepEqual(validatePrBody(buildPrBody({ taskId }), { rootDir: repoRoot }), []);
  });
});

test('validatePrBody rejects mismatched evidence task id linkage', () => {
  withFixtureRepo((repoRoot) => {
    const taskId = '2026-06-17-sample';
    const evidenceTaskId = '2026-06-17-other';
    writeTaskManifest(repoRoot, taskId);
    const evidencePath = path.join(repoRoot, '.harness', 'evidence', evidenceTaskId, 'commands.json');
    fs.mkdirSync(path.dirname(evidencePath), { recursive: true });
    fs.writeFileSync(evidencePath, `${JSON.stringify({ taskId: evidenceTaskId, commands: [] })}\n`, 'utf8');
    for (const name of ['summary.md', 'review.md']) {
      const filePath = path.join(repoRoot, '.harness', 'evidence', taskId, name);
      fs.mkdirSync(path.dirname(filePath), { recursive: true });
      fs.writeFileSync(filePath, `# ${name}\n`, 'utf8');
    }
    const findings = validatePrBody(buildPrBody({
      taskId,
      evidence: `.harness/evidence/${evidenceTaskId}/commands.json`,
    }), { rootDir: repoRoot });
    assert.match(findings.join('\n'), /same task-id/);
  });
});
