import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import { auditId, decisions } from './history-decisions.mjs';

// Exercise the actual archival program, never the actual historical packages.
const fixtureParent = path.resolve('.local/manual-preview-20261116/history-archive-recovery-' + Date.now());
assert.ok(fixtureParent.startsWith(path.resolve('.local') + path.sep));
const write = (root, relative, value) => {
  const file = path.join(root, relative);
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, value);
};
const reports = [];
for (const phase of ['prepare', 'one-move', 'document']) {
  const root = path.join(fixtureParent, phase, 'pantheon-ops');
  const taskDir = `.harness/tasks/${auditId}`;
  const evidenceDir = `.harness/evidence/${auditId}`;
  for (const file of [`${taskDir}/close-history.mjs`, `${taskDir}/history-decisions.mjs`, 'scripts/task-manifest.mjs']) write(root, file, fs.readFileSync(file));
  write(root, `${evidenceDir}/dispositions.json`, JSON.stringify({ archiveDecisions: decisions }));
  for (const decision of decisions) {
    const manifest = JSON.stringify({ taskId: decision.id, goal: 'fixture', status: 'in-progress', primaryLayer: 'business', scope: { in: [], out: [] } });
    for (const prefix of ['.harness', ...(decision.id.includes('base-v0186') || decision.id.includes('base-v0187') ? ['.harness/archive/2026-11'] : [])]) {
      write(root, `${prefix}/tasks/${decision.id}/manifest.json`, manifest);
      for (const [file, content] of [['commands.json', '{"fixture":true}'], ['review.md', '# Historical review\n'], ['summary.md', '# Original summary\n']]) {
        if (file === 'summary.md' && decision.id.includes('2026-06-1')) continue;
        write(root, `${prefix}/evidence/${decision.id}/${file}`, content);
      }
    }
    if (decision.taskDoc) write(root, decision.taskDoc, '---\ntitle: fixture\ndoc_type: Remediation\nlayer: business\nstatus: Active\nupdated_at: 2026-06-17\n---\n\n# Task Packet: fixture\nOriginal body.\n');
  }
  const preload = path.join(root, 'fail-phase.mjs');
  write(root, 'fail-phase.mjs', `import fs from 'node:fs';\nconst rename=fs.renameSync, write=fs.writeFileSync;let count=0;\nfs.renameSync=(...args)=>{count++;if(process.env.HISTORY_FAIL_AT==='prepare'&&count===1||process.env.HISTORY_FAIL_AT==='one-move'&&count===2)throw new Error('intentional recovery probe');return rename(...args)};\nfs.writeFileSync=(file,...args)=>{if(process.env.HISTORY_FAIL_AT==='document'&&String(file).includes('docs'+${JSON.stringify(path.sep)}+'harness'))throw new Error('intentional document probe');return write(file,...args)};\n`);
  const run = (args, env = process.env) => {
    const result = spawnSync(process.execPath, args, { cwd: root, env, encoding: 'utf8', windowsHide: true });
    assert.ifError(result.error);
    return result;
  };
  const program = `${taskDir}/close-history.mjs`;
  assert.equal(run([program]).status, 0);
  const failure = run(['--import', pathToFileURL(preload).href, program, '--apply'], { ...process.env, HISTORY_FAIL_AT: phase });
  assert.notEqual(failure.status, 0);
  assert.match(failure.stderr, /intentional .*probe/);
  const resumed = run([program, '--apply']);
  assert.equal(resumed.status, 0, resumed.stderr);
  const repeated = run([program, '--apply']);
  assert.equal(repeated.status, 0, repeated.stderr);
  for (const decision of decisions) {
    assert.ok(!fs.existsSync(path.join(root, '.harness/tasks', decision.id)));
    assert.ok(!fs.existsSync(path.join(root, '.harness/evidence', decision.id)));
    if (decision.taskDoc) {
      const doc = fs.readFileSync(path.join(root, decision.taskDoc), 'utf8');
      assert.match(doc, /status: Archived/);
      assert.equal((doc.match(/## Result — History Closeout/g) ?? []).length, 1);
      assert.match(doc, /Original body\./);
    }
  }
  const report = JSON.parse(fs.readFileSync(path.join(root, `${evidenceDir}/archive-result.json`), 'utf8'));
  assert.equal(report.hashesPreserved, true);
  assert.equal(report.phasesComplete, true);
  const originalResult = fs.readFileSync(path.join(root, `${evidenceDir}/archive-result.json`));
  const relocationPreload = path.join(root, 'fail-relocation.mjs');
  write(root, 'fail-relocation.mjs', `import fs from 'node:fs';
const rename=fs.renameSync, write=fs.writeFileSync, append=fs.appendFileSync;
fs.renameSync=(...args)=>{if(process.env.HISTORY_RELOCATION_FAIL_AT==='prepare')throw new Error('intentional relocation probe');return rename(...args)};
fs.writeFileSync=(file,...args)=>{if(process.env.HISTORY_RELOCATION_FAIL_AT==='one-move'&&String(file).endsWith('preserved-live-relocation-result.json'))throw new Error('intentional relocation probe');return write(file,...args)};
fs.appendFileSync=(file,data,...args)=>{if(process.env.HISTORY_RELOCATION_FAIL_AT==='document'&&String(data).includes('preserved-live-relocated'))throw new Error('intentional relocation probe');return append(file,data,...args)};
`);
  const relocationFailure = run(['--import', pathToFileURL(relocationPreload).href, program, '--relocate-preserved'], { ...process.env, HISTORY_RELOCATION_FAIL_AT: phase });
  assert.notEqual(relocationFailure.status, 0);
  assert.match(relocationFailure.stderr, /intentional relocation probe/);
  const relocationResume = run([program, '--relocate-preserved']);
  assert.equal(relocationResume.status, 0, relocationResume.stderr);
  const relocationRepeat = run([program, '--relocate-preserved']);
  assert.equal(relocationRepeat.status, 0, relocationRepeat.stderr);
  const archiveAfterRelocation = run([program, '--apply']);
  assert.equal(archiveAfterRelocation.status, 0, archiveAfterRelocation.stderr);
  assert.deepEqual(fs.readFileSync(path.join(root, `${evidenceDir}/archive-result.json`)), originalResult, 'first archive result is immutable');
  assert.ok(!fs.existsSync(path.join(root, `${evidenceDir}/artifacts/preserved-live`)));
  assert.ok(fs.existsSync(path.join(root, `.harness/archive/2026-11/preserved-live/${auditId}`)));
  reports.push({ phase, injectedFailureExit: failure.status, resumeExit: resumed.status, repeatExit: repeated.status, all18Verified: true,
    relocationFailureExit: relocationFailure.status, relocationResumeExit: relocationResume.status, relocationRepeatExit: relocationRepeat.status,
    archiveAfterRelocationExit: archiveAfterRelocation.status, originalResultPreserved: true });
}
const reportPath = `.harness/evidence/${auditId}/archive-relocation-recovery-verification.json`;
assert.ok(!fs.existsSync(reportPath), 'choose a fresh report before repeating this verification');
fs.writeFileSync(reportPath, JSON.stringify({ fixtureParent, reports }, null, 2) + '\n');
console.log(JSON.stringify({ phases: reports.length, candidatesPerPhase: 18, resumedAndRepeated: true, relocationRecoveryVerified: true }));
