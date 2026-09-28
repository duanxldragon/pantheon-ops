import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

const testDir = path.dirname(fileURLToPath(import.meta.url));
const moduleUrl = pathToFileURL(path.resolve(testDir, '../../scripts/create-pr.mjs')).href;
const { buildGhArguments, resolveBodyFile, validateBodyFile } = await import(moduleUrl);

test('buildGhArguments creates a PR with the validated body file', () => {
  assert.deepEqual(
    buildGhArguments({
      title: 'fix(governance): enforce PR body validation',
      bodyFile: 'D:/repo/pr-body.md',
      base: 'main',
      draft: true,
    }),
    [
      'pr',
      'create',
      '--title',
      'fix(governance): enforce PR body validation',
      '--body-file',
      'D:/repo/pr-body.md',
      '--base',
      'main',
      '--draft',
    ],
  );
});

test('buildGhArguments supports editing an existing PR', () => {
  assert.deepEqual(
    buildGhArguments({
      prNumber: '222',
      bodyFile: 'D:/repo/pr-body.md',
      title: 'ignored',
      base: 'main',
      draft: false,
    }),
    ['pr', 'edit', '222', '--body-file', 'D:/repo/pr-body.md'],
  );
});

test('resolveBodyFile rejects paths outside the repository', () => {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-create-pr-'));
  try {
    assert.throws(
      () => resolveBodyFile(repoRoot, path.join(repoRoot, '..', 'pr-body.md')),
      /must stay inside the repository/,
    );
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
});

test('a PR body with valid harness artifact links passes the CI validator', () => {
  // Self-contained fixture: build a minimal repoRoot with the harness
  // artifacts the validator checks for, so this test does not couple to any
  // specific checked-in PR body that archiving may relocate.
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-create-pr-'));
  try {
    const taskId = '2026-01-01-pr-body-validator-fixture';
    const manifestDir = path.join(repoRoot, '.harness/tasks', taskId);
    const evidenceDir = path.join(repoRoot, '.harness/evidence', taskId);
    fs.mkdirSync(manifestDir, { recursive: true });
    fs.mkdirSync(evidenceDir, { recursive: true });
    fs.writeFileSync(
      path.join(manifestDir, 'manifest.json'),
      JSON.stringify({
        taskId,
        goal: 'fixture',
        primaryLayer: 'governance',
        scope: { in: [], out: [] },
        linkage: {
          evidenceDir: `.harness/evidence/${taskId}/`,
          reviewFile: `.harness/evidence/${taskId}/review.md`,
          changeRef: 'none',
          planRefs: [],
        },
      }),
    );
    for (const file of ['commands.json', 'summary.md', 'review.md']) {
      fs.writeFileSync(path.join(evidenceDir, file), '{}');
    }
    const body = [
      '## 变更摘要',
      '',
      '- 改动层级：governance',
      '- 改动模块：tests',
      '- 目标问题：validator fixture',
      '- 预期影响：validator fixture coverage',
      '',
      '## Harness 链路',
      '',
      `- Task ID：${taskId}`,
      `- Task Manifest：.harness/tasks/${taskId}/manifest.json`,
      `- Evidence：.harness/evidence/${taskId}/commands.json`,
      `- Verification evidence：.harness/evidence/${taskId}/summary.md`,
      `- Review Artifact：.harness/evidence/${taskId}/review.md`,
      '- OpenSpec change：none',
      '- Trivial change：yes',
      '- Quality Profile：none',
      '- Ratchet Decision：no-repeat-observed',
      '- GitHub Signal：not-applicable',
      '',
      '## Harness adoption markers',
      '',
      `- task id: ${taskId}`,
      `- task manifest: .harness/tasks/${taskId}/manifest.json`,
      `- evidence: .harness/evidence/${taskId}/commands.json`,
      '- boundaries: none',
      '- backend response contract: none',
      '- backend DTO contract: none',
      '- permission contract: none',
      '- audit coverage: none',
      '- visual evidence: none',
      '- inheritance contract: none',
      '- base drift: none',
      '- Base/ops inheritance: none',
      '',
      '## 边界说明',
      '',
      '- [x] 本次改动仅涉及单一层级',
      '',
      '## 验证记录',
      '',
      '- [x] 后端测试：go test ./...',
      '',
      '## 审核留痕',
      '',
      '- Copilot review：unavailable',
      '- CodeQL 结果：not-applicable',
      '- GitHub checks 结果：pending',
      '- Auto-merge：enabled',
      '- Duplication Gate 结果：not-applicable',
      '- 是否高风险改动：no',
      '- Residual risk / follow-up：none',
      '',
      '## 检查清单',
      '',
      '- [x] 已明确本次改动归属',
    ].join('\n');
    const bodyPath = path.join(repoRoot, 'pr-body.md');
    fs.writeFileSync(bodyPath, body);
    assert.equal(
      path.relative(repoRoot, validateBodyFile(repoRoot, bodyPath)).replaceAll('\\', '/'),
      'pr-body.md',
    );
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
});
