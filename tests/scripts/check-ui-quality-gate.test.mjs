import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import { scan } from '../../scripts/harness/check-ui-quality-gate.mjs';

const repoRoot = path.resolve(import.meta.dirname, '../..');

function copyFile(root, relativePath) {
  const target = path.join(root, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.copyFileSync(path.join(repoRoot, relativePath), target);
}

function createFixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'ui-quality-gate-'));
  for (const relativePath of [
    'config/ui-quality-gate.json',
    'docs/designs/OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md',
    'frontend/scripts/check-shell-visual-contract.mjs',
    'frontend/scripts/check-ui-contract.mjs',
    'frontend/scripts/check-contrast.mjs',
    'frontend/scripts/check-important-budget.mjs',
    'scripts/harness/check-visual-evidence.mjs',
    'frontend/playwright.visual.config.ts',
    'frontend/tests/visual/visual-baseline.spec.ts',
    'frontend/tests/visual/README.md',
  ]) {
    copyFile(root, relativePath);
  }
  fs.mkdirSync(path.join(root, '.github', 'workflows'), { recursive: true });
  fs.writeFileSync(path.join(root, 'package.json'), JSON.stringify({ scripts: { 'check:ui-quality-gate': 'node scripts/harness/check-ui-quality-gate.mjs --root . --strict' } }));
  fs.writeFileSync(path.join(root, '.github', 'workflows', 'quality.yml'), 'run: npm run check:ui-quality-gate\n');
  return root;
}

test('accepts the canonical policy and repository integration', () => {
  const root = createFixture();
  try {
    const result = scan(root, 'config/ui-quality-gate.json');
    assert.deepEqual(result.findings, []);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('rejects policy weakening and post-adoption UI tasks without evidence declarations', () => {
  const root = createFixture();
  try {
    const configPath = path.join(root, 'config', 'ui-quality-gate.json');
    const policy = JSON.parse(fs.readFileSync(configPath, 'utf8'));
    policy.requiredMatrix.viewports = ['1440x900'];
    fs.writeFileSync(configPath, JSON.stringify(policy));

    const manifestDir = path.join(root, '.harness', 'tasks', '2026-09-01-ui-change');
    fs.mkdirSync(manifestDir, { recursive: true });
    fs.writeFileSync(path.join(manifestDir, 'manifest.json'), JSON.stringify({
      taskId: '2026-09-01-ui-change',
      qualityProfiles: ['ui-runtime'],
      verificationPlan: {},
    }));

    const result = scan(root, 'config/ui-quality-gate.json');
    assert.ok(result.findings.some((finding) => finding.code === 'ui_gate_viewport_missing'));
    assert.ok(result.findings.some((finding) => finding.code === 'ui_gate_visual_plan_missing'));
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test('accepts only explicit governance-only visual exemptions', () => {
  const root = createFixture();
  try {
    const manifestDir = path.join(root, '.harness', 'tasks', '2026-09-01-gate-change');
    fs.mkdirSync(manifestDir, { recursive: true });
    fs.writeFileSync(path.join(manifestDir, 'manifest.json'), JSON.stringify({
      taskId: '2026-09-01-gate-change',
      qualityProfiles: ['ui-runtime'],
      verificationPlan: {
        visualEvidenceExemption: {
          scope: 'governance-only',
          reason: 'No rendered surface changed.',
          noRenderedSurfaceChanged: true,
          humanApprovalRequired: true,
        },
      },
    }));

    const result = scan(root, 'config/ui-quality-gate.json');
    assert.deepEqual(result.findings, []);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});
