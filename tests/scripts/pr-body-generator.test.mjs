import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');
const generator = path.join(repoRoot, 'scripts', 'harness', 'generate-pr-body.mjs');

// The generator used to hardcode `Quality Profile: none`, which the governance
// checker rejects for every non-trivial change ("Quality Profile must not be none
// for non-trivial changes"), so the sanctioned path could not produce a compliant
// body. These tests pin the manifest as the source of those fields.
function runGenerator(manifest) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-pr-body-'));
  try {
    const taskId = 'fixture-task';
    const taskDir = path.join(root, '.harness', 'tasks', taskId);
    fs.mkdirSync(taskDir, { recursive: true });
    fs.writeFileSync(path.join(taskDir, 'manifest.json'), JSON.stringify(manifest, null, 2));

    const result = spawnSync(process.execPath, [generator, taskId], {
      cwd: root,
      encoding: 'utf8',
    });
    assert.equal(result.status, 0, result.stderr);
    return result.stdout;
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
}

function baseManifest(overrides = {}) {
  return {
    taskId: 'fixture-task',
    goal: 'fixture goal',
    primaryLayer: 'platform',
    status: 'completed',
    runtimeSensitive: false,
    dependencyLayers: ['platform'],
    scope: { in: ['a'], out: ['b'] },
    expectedFiles: { modify: ['scripts/harness/generate-pr-body.mjs'] },
    harnessProfile: { qualityProfile: 'ci-workflow' },
    verificationPlan: { commands: ['true'] },
    ...overrides,
  };
}

function fieldValue(body, label) {
  const match = new RegExp(`^- ${label}：(.*)$`, 'm').exec(body);
  return match ? match[1].trim() : null;
}

test('emits the manifest quality profile instead of a hardcoded none', () => {
  const body = runGenerator(baseManifest());
  assert.equal(fieldValue(body, 'Quality Profile'), 'ci-workflow');
});

test('supports every quality profile the governance checker accepts', () => {
  for (const profile of [
    'auth-security',
    'permission-policy',
    'i18n',
    'ui-runtime',
    'generator',
    'ci-workflow',
  ]) {
    const body = runGenerator(
      baseManifest({ harnessProfile: { qualityProfile: profile } }),
    );
    assert.equal(fieldValue(body, 'Quality Profile'), profile);
  }
});

test('never emits none for a non-trivial change', () => {
  const body = runGenerator(baseManifest());

  assert.equal(fieldValue(body, 'Trivial change'), 'no');
  assert.notEqual(fieldValue(body, 'Quality Profile'), 'none');
});

test('lets a trivial change keep the none profile', () => {
  const body = runGenerator(
    baseManifest({
      runtimeSensitive: false,
      dependencyLayers: [],
      primaryLayer: 'platform',
      harnessProfile: {},
    }),
  );

  assert.equal(fieldValue(body, 'Trivial change'), 'yes');
  assert.equal(fieldValue(body, 'Quality Profile'), 'none');
});

test('emits the manifest ratchet decision and defaults to no-repeat-observed', () => {
  const withDecision = runGenerator(
    baseManifest({ harnessProfile: { qualityProfile: 'ci-workflow', ratchetDecision: 'gate-updated' } }),
  );
  assert.equal(fieldValue(withDecision, 'Ratchet Decision'), 'gate-updated');

  const withoutDecision = runGenerator(baseManifest());
  assert.equal(fieldValue(withoutDecision, 'Ratchet Decision'), 'no-repeat-observed');
});

// End-to-end compliance (the checker also verifies the referenced task packet and
// evidence files exist in the repository) is enforced by the Docs Governance job
// on every PR, so it is deliberately not duplicated with fixture artifacts here.
test('emits every governance field the checker parses', () => {
  const body = runGenerator(
    baseManifest({ harnessProfile: { qualityProfile: 'ci-workflow', ratchetDecision: 'gate-updated' } }),
  );

  for (const label of [
    'Task ID',
    'Task Manifest',
    'Evidence',
    'Verification evidence',
    'Review Artifact',
    'Trivial change',
    'Quality Profile',
    'Ratchet Decision',
    'GitHub Signal',
  ]) {
    assert.notEqual(fieldValue(body, label), null, `body must carry the ${label} field`);
  }
});
