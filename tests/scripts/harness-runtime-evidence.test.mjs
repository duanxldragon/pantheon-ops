import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const checker = fileURLToPath(new URL('../../scripts/harness/check-runtime-evidence.mjs', import.meta.url));

for (const [name, taskFlag, evidenceFlag, hasLogs, exitCode] of [
  ['task true cannot be suppressed by evidence false', true, false, false, 1],
  ['evidence true cannot be suppressed by task false', false, true, false, 1],
  ['explicit false still overrides ambiguous plan prose', false, false, false, 0],
  ['runtime plan is used when neither flag exists', undefined, undefined, false, 1],
  ['required runtime evidence satisfies the gate', true, false, true, 0],
]) {
  test(name, (t) => {
    const tempParent = fs.realpathSync(os.tmpdir());
    const root = fs.mkdtempSync(path.join(tempParent, 'runtime-evidence-'));
    t.after(() => {
      assert.equal(path.dirname(fs.realpathSync(root)), tempParent);
      fs.rmSync(root, { recursive: true, force: true });
    });
    const manifest = '.harness/tasks/runtime-fixture/manifest.json';
    const evidence = '.harness/evidence/runtime-fixture/commands.json';
    fs.mkdirSync(path.dirname(path.join(root, manifest)), { recursive: true });
    fs.mkdirSync(path.dirname(path.join(root, evidence)), { recursive: true });
    fs.writeFileSync(path.join(root, manifest), JSON.stringify({
      taskId: 'runtime-fixture', runtimeSensitive: taskFlag,
      goal: 'Check runtime evidence declarations', primaryLayer: 'platform',
      scope: { in: ['runtime evidence'], out: [] },
      linkage: { evidenceDir: '.harness/evidence/runtime-fixture', reviewFile: '.harness/evidence/runtime-fixture/review.md', changeRef: 'none', planRefs: [] },
      verificationPlan: { runtimeEvidence: ['browser smoke'] },
    }));
    fs.writeFileSync(path.join(root, evidence), JSON.stringify({
      taskId: 'runtime-fixture', runtimeSensitive: evidenceFlag,
      linkage: { taskManifest: manifest },
      ...(hasLogs ? { runtimeLogs: ['fixture-runtime.log'] } : {}),
    }));
    const result = spawnSync(process.execPath, [checker, '--root', root, '--strict', '--json'], { encoding: 'utf8' });
    assert.equal(result.status, exitCode, result.stdout + result.stderr);
    const report = JSON.parse(result.stdout);
    assert.equal(report.errorCount, 0);
    assert.equal(report.warningCount, exitCode);
  });
}
