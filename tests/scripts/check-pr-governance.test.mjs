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
    `${JSON.stringify(
      {
        taskId,
        title: 'Ops governance follow-up',
        goal: 'Keep GitHub governance writeback manifest-first in pantheon-ops.',
        primaryLayer: 'business/deploy',
        scope: {
          in: ['Repository governance automation', 'Local PR closure logic'],
          out: ['Shared platform runtime changes'],
        },
        implementationNotes: [
          'This stays in pantheon-ops because it is repository governance and local PR closure logic, not shared platform runtime behavior.',
        ],
        linkage: {
          evidenceDir: `.harness/evidence/${taskId}/`,
          reviewFile: `.harness/evidence/${taskId}/review.md`,
          changeRef: 'none',
          planRefs: [],
        },
        verificationPlan: {
          commands: ['node --test tests/scripts/check-pr-governance.test.mjs'],
          runtimeEvidence: [],
        },
        runtimeSensitive: false,
        evidenceRequired: ['commands.json'],
        humanGates: ['none'],
        completionChecklist: [
          'Layer and boundary declared',
          'Contract anchors read',
          'Verification run or exception recorded',
          'Evidence saved or summarized',
          'Review completed',
        ],
      },
      null,
      2,
    )}\n`,
    'utf8',
  );
}

const validTemplate = `## Summary

- Target repo: \`pantheon-ops\`
- Layer: \`business/cmdb\`
- Task mode: \`implement\`
- Sync expectation: \`business-only\`

## Scope

- In scope:
- Out of scope:

## Verification

- Commands:
- Result:

## Evidence

- Task ID: \`<task-id>\`
- Task Manifest: \`.harness/tasks/<task-id>/manifest.json\`
- Evidence: \`.harness/evidence/<task-id>/commands.json\`
- Human gate: \`none\`

## Review

- Review status: \`passed\`
- Review artifact: \`.harness/evidence/<task-id>/review.md\`

## Release Risk

- Known gaps: \`none\`
- GitHub signal: \`repo-quality-gate\`
`;

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

test('validatePrBody rejects explicit none for required PR fields', () => {
  const invalidBody = validTemplate
    .replace('`implement`', '`none`')
    .replace('`repo-quality-gate`', '<signal>');

  const findings = validatePrBody(invalidBody);

  assert.match(findings.join('\n'), /Task mode/);
  assert.match(findings.join('\n'), /GitHub signal/);
});

test('validatePrBody rejects template and inline artifact placeholders', () => {
  const findings = validatePrBody(`## Summary

- Target repo: \`pantheon-ops\`
- Layer: \`business/deploy\`
- Task mode: \`ui\`
- Sync expectation: \`deferred base -> ops\`

## Scope

- In scope: \`deploy page workflow hardening\`
- Out of scope: \`base foundation code\`

## Verification

- Commands: \`npm run check:inheritance && npm --prefix frontend run build\`
- Result: \`pass\`

## Evidence

- Task ID: \`2026-06-17-sample\`
- Task Manifest: \`docs/TASK_PACKET_OPS_TEMPLATE.md\`
- Evidence: \`inline command summary\`
- Human gate: \`none\`

## Review

- Review status: \`passed\`
- Review artifact: \`inline review summary\`

## Release Risk

- Known gaps: \`none\`
- GitHub signal: \`repo-quality-gate\`
  `);

  assert.match(findings.join('\n'), /Task Manifest/);
  assert.match(findings.join('\n'), /Evidence/);
  assert.match(findings.join('\n'), /Review artifact/);
});

test('validatePrBody rejects missing artifact files', () => {
  const findings = validatePrBody(`## Summary

- Target repo: \`pantheon-ops\`
- Layer: \`business/deploy\`
- Task mode: \`ui\`
- Sync expectation: \`deferred base -> ops\`

## Scope

- In scope: \`deploy page workflow hardening\`
- Out of scope: \`base foundation code\`

## Verification

- Commands: \`npm run check:inheritance && npm --prefix frontend run build\`
- Result: \`pass\`

## Evidence

- Task ID: \`2026-06-17-missing\`
- Task Manifest: \`.harness/tasks/2026-06-17-missing/manifest.json\`
- Evidence: \`.harness/evidence/2026-06-17-missing/commands.json\`
- Human gate: \`none\`

## Review

- Review status: \`passed\`
- Review artifact: \`.harness/evidence/2026-06-17-missing/review.md\`

## Release Risk

- Known gaps: \`none\`
- GitHub signal: \`repo-quality-gate\`
  `, { rootDir: path.resolve(testDir, '../..') });

  assert.match(findings.join('\n'), /Task Manifest/);
  assert.match(findings.join('\n'), /Evidence/);
  assert.match(findings.join('\n'), /Review artifact/);
});

test('validatePrBody accepts existing manifest, evidence, and review artifact files', () => {
  withFixtureRepo((repoRoot) => {
    const taskId = '2026-06-17-sample';
    const commandsArtifactPath = path.join(
      repoRoot,
      '.harness',
      'evidence',
      taskId,
      'commands.json',
    );
    const reviewArtifactPath = path.join(
      repoRoot,
      '.harness',
      'evidence',
      taskId,
      'review.md',
    );
    writeTaskManifest(repoRoot, taskId);
    fs.mkdirSync(path.dirname(commandsArtifactPath), { recursive: true });
    fs.mkdirSync(path.dirname(reviewArtifactPath), { recursive: true });
    fs.writeFileSync(commandsArtifactPath, '{"commands":[]}\n', 'utf8');
    fs.writeFileSync(reviewArtifactPath, '# Review Summary: sample\n', 'utf8');

    const body = `## Summary

- Target repo: \`pantheon-ops\`
- Layer: \`business/deploy\`
- Task mode: \`ui\`
- Sync expectation: \`deferred base -> ops\`

## Scope

- In scope: \`deploy page workflow hardening\`
- Out of scope: \`base foundation code\`

## Verification

- Commands: \`npm run check:inheritance && npm --prefix frontend run build\`
- Result: \`pass\`

## Evidence

- Task ID: \`${taskId}\`
- Task Manifest: \`.harness/tasks/${taskId}/manifest.json\`
- Evidence: \`.harness/evidence/${taskId}/commands.json\`
- Human gate: \`none\`

## Review

- Review status: \`passed\`
- Review artifact: \`.harness/evidence/${taskId}/review.md\`

## Release Risk

- Known gaps: \`none\`
- GitHub signal: \`repo-quality-gate\`
`;

    assert.deepEqual(validatePrBody(body, { rootDir: repoRoot }), []);
  });
});

test('validatePrBody rejects mismatched evidence task id linkage', () => {
  withFixtureRepo((repoRoot) => {
    const taskId = '2026-06-17-sample';
    const evidenceTaskId = '2026-06-17-other';
    const commandsArtifactPath = path.join(
      repoRoot,
      '.harness',
      'evidence',
      evidenceTaskId,
      'commands.json',
    );
    const reviewArtifactPath = path.join(
      repoRoot,
      '.harness',
      'evidence',
      taskId,
      'review.md',
    );
    writeTaskManifest(repoRoot, taskId);
    fs.mkdirSync(path.dirname(commandsArtifactPath), { recursive: true });
    fs.mkdirSync(path.dirname(reviewArtifactPath), { recursive: true });
    fs.writeFileSync(commandsArtifactPath, '{"commands":[]}\n', 'utf8');
    fs.writeFileSync(reviewArtifactPath, '# Review Summary: sample\n', 'utf8');

    const findings = validatePrBody(`## Summary

- Target repo: \`pantheon-ops\`
- Layer: \`business/deploy\`
- Task mode: \`ui\`
- Sync expectation: \`deferred base -> ops\`

## Scope

- In scope: \`deploy page workflow hardening\`
- Out of scope: \`base foundation code\`

## Verification

- Commands: \`npm run check:inheritance && npm --prefix frontend run build\`
- Result: \`pass\`

## Evidence

- Task ID: \`${taskId}\`
- Task Manifest: \`.harness/tasks/${taskId}/manifest.json\`
- Evidence: \`.harness/evidence/${evidenceTaskId}/commands.json\`
- Human gate: \`none\`

## Review

- Review status: \`passed\`
- Review artifact: \`.harness/evidence/${taskId}/review.md\`

## Release Risk

- Known gaps: \`none\`
- GitHub signal: \`repo-quality-gate\`
    `, { rootDir: repoRoot });

    assert.match(findings.join('\n'), /same task-id/);
  });
});
