import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

import {
  classifyFrontendSource,
  findUnauthorizedFrontendResiduals,
} from '../../scripts/check-clean-consumer.mjs';

const lock = {
  consumerMode: 'foundation-release-consumer',
  releaseVersion: 'pantheon-base-v0.10.14',
  baseCommit: '68d2bd714ddbeae9e61c0209e54559b96916f316',
  sharedPaths: {
    frontend: [
      'frontend/src/App.tsx',
      'frontend/src/components',
      'frontend/src/modules/system',
    ],
  },
};

test('classifies approved consumer source ownership', () => {
  assert.equal(classifyFrontendSource('frontend/src/App.tsx', lock), 'foundation');
  assert.equal(classifyFrontendSource('frontend/src/components/table/index.ts', lock), 'foundation');
  assert.equal(classifyFrontendSource('frontend/src/modules/business/cmdb/index.ts', lock), 'business');
  assert.equal(classifyFrontendSource('frontend/src/i18n/resources/generated/en-US.ts', lock), 'generated-overlay');
  assert.equal(classifyFrontendSource('frontend/src/modules/generated/business.ts', lock), 'generated-overlay');
  assert.equal(classifyFrontendSource('frontend/src/core/router/componentRegistry.ts', lock), 'explicit-overlay');
});

test('rejects unowned generic frontend source', () => {
  assert.equal(classifyFrontendSource('frontend/src/assets/hero.png', lock), 'unauthorized-residual');
  assert.equal(classifyFrontendSource('frontend/src/shared/localOnly.ts', lock), 'unauthorized-residual');
});

test('ignores tracked paths that are deleted from the working tree', () => {
  const opsRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-clean-consumer-'));
  try {
    fs.mkdirSync(path.join(opsRoot, 'frontend', 'src', 'assets'), { recursive: true });
    fs.writeFileSync(
      path.join(opsRoot, 'foundation-release.lock.json'),
      `${JSON.stringify(lock)}\n`,
      'utf8',
    );
    const residualPath = path.join(opsRoot, 'frontend', 'src', 'assets', 'hero.png');
    fs.writeFileSync(residualPath, 'residual', 'utf8');
    for (const args of [
      ['init'],
      ['config', 'user.email', 'test@example.com'],
      ['config', 'user.name', 'Test'],
      ['add', '.'],
      ['commit', '-m', 'fixture'],
    ]) {
      const result = spawnSync('git', args, { cwd: opsRoot, encoding: 'utf8' });
      assert.equal(result.status, 0, result.stderr);
    }
    assert.deepEqual(findUnauthorizedFrontendResiduals(opsRoot), ['frontend/src/assets/hero.png']);
    fs.rmSync(residualPath);
    assert.deepEqual(findUnauthorizedFrontendResiduals(opsRoot), []);
  } finally {
    fs.rmSync(opsRoot, { recursive: true, force: true });
  }
});
