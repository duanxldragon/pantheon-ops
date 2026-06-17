import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';

import { analyzeDuplication, evaluateDuplication } from '../../scripts/check-duplication.mjs';

function withFixtureRepo(callback) {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'pantheon-ops-duplication-'));
  try {
    fs.mkdirSync(path.join(repoRoot, 'backend'), { recursive: true });
    fs.mkdirSync(path.join(repoRoot, 'frontend', 'src'), { recursive: true });
    fs.mkdirSync(path.join(repoRoot, 'frontend', 'src', 'modules', 'business', 'cmdb'), {
      recursive: true,
    });
    callback(repoRoot);
  } finally {
    fs.rmSync(repoRoot, { recursive: true, force: true });
  }
}

test('analyzeDuplication reports repository duplication above threshold', () => {
  withFixtureRepo((repoRoot) => {
    const sharedBlock = [
      'export function sharedBlock() {',
      "  const steps = ['alpha', 'beta', 'gamma'];",
      '  let output = 0;',
      '  for (const step of steps) {',
      '    output += step.length;',
      '  }',
      '  return output;',
      '}',
    ].join('\n');
    fs.writeFileSync(path.join(repoRoot, 'backend', 'alpha.ts'), `${sharedBlock}\n`, 'utf8');
    fs.writeFileSync(path.join(repoRoot, 'frontend', 'src', 'beta.ts'), `${sharedBlock}\n`, 'utf8');

    const report = analyzeDuplication(repoRoot, {
      includeRoots: ['backend', 'frontend/src'],
      minimumBlockLines: 8,
    });

    assert.equal(report.duplicates.length > 0, true);
    assert.equal(report.duplicatedLines > 0, true);
    assert.equal(evaluateDuplication(report, 3).ok, false);
  });
});

test('analyzeDuplication ignores generated low-code business paths', () => {
  withFixtureRepo((repoRoot) => {
    const sharedBlock = [
      'export function generatedOnly() {',
      "  const values = ['cmdb', 'host', 'label'];",
      '  let total = 0;',
      '  for (const value of values) {',
      '    total += value.length;',
      '  }',
      '  return total;',
      '}',
    ].join('\n');

    fs.writeFileSync(
      path.join(repoRoot, 'frontend', 'src', 'modules', 'business', 'cmdb', 'host.ts'),
      `${sharedBlock}\n`,
      'utf8',
    );
    fs.writeFileSync(
      path.join(repoRoot, 'frontend', 'src', 'modules', 'business', 'cmdb', 'label.ts'),
      `${sharedBlock}\n`,
      'utf8',
    );

    const report = analyzeDuplication(repoRoot, {
      includeRoots: ['frontend/src'],
      minimumBlockLines: 8,
    });

    assert.equal(report.files.length, 0);
    assert.equal(report.duplicatedLines, 0);
    assert.equal(evaluateDuplication(report, 3).ok, true);
  });
});
