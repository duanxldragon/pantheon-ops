import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';

const audit = '.harness/evidence/2026-11-17-ops-history-audit';
const task = '.harness/tasks/2026-11-17-ops-history-audit';
const fixtureParent = path.resolve('.local/manual-preview-20261116/continuation-archive-recovery-' + Date.now());
assert.ok(fixtureParent.startsWith(path.resolve('.local') + path.sep));
const snapshot = JSON.parse(fs.readFileSync(`${audit}/continuation-dispositions.json`));
const sha = (bytes) => createHash('sha256').update(bytes).digest('hex');
const reports = [];
const copy = (root, file) => {
  const target = path.join(root, file);
  assert.ok(target.startsWith(root + path.sep));
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.cpSync(file, target, { recursive: true, errorOnExist: true });
};
for (const phase of ['prepare', 'one-move', 'document', 'partial-stage', 'after-document', 'extra-evidence', 'preflight-path']) {
  const root = path.join(fixtureParent, phase, 'pantheon-ops');
  for (const file of [`${task}/archive-continuation.mjs`, 'scripts/task-manifest.mjs',
    ...['continuation-dispositions.json', 'dispositions.json', 'archive-dry-run.json', 'archive-result.json', 'second-batch-result.json'].map((file) => `${audit}/${file}`)]) copy(root, file);
  for (const id of snapshot.pairArchiveCandidates) {
    const source = `.harness/tasks/${id}`;
    copy(root, source);
    copy(root, `.harness/evidence/${id}`);
    const manifest = JSON.parse(fs.readFileSync(`${source}/manifest.json`));
    if (id !== 'cmdb-host-import-export') copy(root, manifest.taskDoc ?? manifest.linkage.taskPacket);
  }
  const run = (args, env = process.env) => {
    const result = spawnSync(process.execPath, args, { cwd: root, env, windowsHide: true, encoding: 'utf8' });
    assert.ifError(result.error);
    return result;
  };
  const program = `${task}/archive-continuation.mjs`;
  assert.equal(run([program]).status, 0);
  const preflightFile = path.join(root, audit, 'continuation-archive-preflight.json');
  const preflight = JSON.parse(fs.readFileSync(preflightFile));
  if (phase === 'extra-evidence' || phase === 'preflight-path') {
    if (phase === 'extra-evidence') fs.writeFileSync(path.join(root, preflight.operations[0].sourceEvidence, 'new-evidence.txt'), 'new evidence must survive');
    else {
      preflight.operations[0].targetTask = '.harness/archive/2026-11/tasks/unrelated';
      fs.writeFileSync(preflightFile, JSON.stringify(preflight, null, 2) + '\n');
      // Even a matching seal cannot authorize a noncanonical operation path.
      fs.writeFileSync(path.join(root, audit, 'continuation-archive-preflight-seal.json'), JSON.stringify({
        preflightSHA256: sha(fs.readFileSync(preflightFile)), snapshotSHA256: sha(fs.readFileSync(path.join(root, audit, 'continuation-dispositions.json')))
      }));
    }
    const refused = run([program, '--apply']);
    assert.equal(refused.status, 1);
    assert.ok(fs.existsSync(path.join(root, preflight.operations[0].sourceTask)));
    if (phase === 'extra-evidence') assert.equal(fs.readFileSync(path.join(root, preflight.operations[0].sourceEvidence, 'new-evidence.txt'), 'utf8'), 'new evidence must survive');
    reports.push({ phase, refusedExit: refused.status, originalsUntouched: true });
    continue;
  }
  const preload = path.join(root, 'fail-phase.mjs');
  fs.writeFileSync(preload, `import fs from 'node:fs';
const rename=fs.renameSync,write=fs.writeFileSync,append=fs.appendFileSync;let moves=0,partial=false;
fs.renameSync=(source,target)=>{
  if(fs.existsSync(source)&&fs.statSync(source).isDirectory()){
    moves++;
    if(process.env.FAIL_PHASE==='prepare'&&moves===1||process.env.FAIL_PHASE==='one-move'&&moves===2)throw new Error('intentional recovery probe');
  }
  if(process.env.FAIL_PHASE==='document'&&String(target).includes('docs'+${JSON.stringify(path.sep)}+'harness'))throw new Error('intentional recovery probe');
  return rename(source,target);
};
fs.writeFileSync=(file,bytes,...rest)=>{
  if(process.env.FAIL_PHASE==='partial-stage'&&typeof file==='number'&&!partial){partial=true;write(file,Buffer.from(bytes).subarray(0,8));throw new Error('intentional recovery probe');}
  return write(file,bytes,...rest);
};
fs.appendFileSync=(file,bytes,...rest)=>{
  if(process.env.FAIL_PHASE==='after-document'&&String(bytes).includes('"phase":"complete"'))throw new Error('intentional recovery probe');
  return append(file,bytes,...rest);
};
`);
  const failed = run(['--import', pathToFileURL(preload).href, program, '--apply'], { ...process.env, FAIL_PHASE: phase });
  assert.equal(failed.status, 1);
  assert.match(failed.stderr, /intentional recovery probe/);
  const resumed = run([program, '--apply']);
  assert.equal(resumed.status, 0, resumed.stderr);
  const repeated = run([program, '--apply']);
  assert.equal(repeated.status, 0, repeated.stderr);
  const result = JSON.parse(fs.readFileSync(path.join(root, audit, 'continuation-archive-result.json')));
  assert.equal(result.archived, 5);
  assert.ok(result.hashesPreserved && result.priorBatchesUnchanged);
  for (const op of result.operations) {
    for (const row of op.originalEvidence) assert.equal(sha(fs.readFileSync(path.join(root, op.targetEvidence, row.file))), row.sha256);
    if (op.taskDoc) {
      const card = fs.readFileSync(path.join(root, op.taskDoc), 'utf8');
      assert.equal((card.match(/## Result — Historical Disposition 2026-11-17/g) ?? []).length, 1);
    } else assert.ok(fs.existsSync(path.join(root, op.targetTask, 'plan.md')));
  }
  reports.push({ phase, injectedFailureExit: failed.status, resumeExit: resumed.status, repeatExit: repeated.status, allFiveOriginalEvidenceVerified: true });
}
fs.writeFileSync(`${audit}/continuation-archive-recovery-verification.json`, JSON.stringify({ fixtureParent, reports }, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ recoveryPhases: 5, rejectionCases: 2, candidatesPerPhase: 5, originalEvidencePreserved: true }));
