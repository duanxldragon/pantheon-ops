import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { resolveArchiveAwarePath } from '../../../scripts/task-manifest.mjs';

const root = fs.realpathSync(process.cwd());
assert.equal(path.basename(root), 'pantheon-ops');
const audit = '.harness/evidence/2026-11-17-ops-history-audit';
const hash = (bytes) => createHash('sha256').update(bytes).digest('hex');
const checked = (relative) => {
  const absolute = path.resolve(root, relative);
  const rel = path.relative(root, absolute);
  assert.ok(rel && !rel.startsWith('..') && !path.isAbsolute(rel), `outside repository: ${relative}`);
  for (let ancestor = absolute; ancestor !== root; ancestor = path.dirname(ancestor)) {
    if (fs.existsSync(ancestor)) assert.ok(!fs.lstatSync(ancestor).isSymbolicLink(), `linked path: ${ancestor}`);
  }
  return absolute;
};
const exists = (file) => fs.existsSync(checked(file));
const read = (file) => fs.readFileSync(checked(file));
const json = (file) => JSON.parse(read(file));
const write = (file, bytes, exclusive = false) => {
  fs.mkdirSync(path.dirname(checked(file)), { recursive: true });
  // Stage outside the moved packages; process interruption never truncates a
  // manifest/card and a retained stage can be safely replayed.
  const stage = `${audit}/continuation-write-staging/${hash(Buffer.from(file))}-${hash(Buffer.from(bytes))}.tmp`;
  fs.mkdirSync(path.dirname(checked(stage)), { recursive: true });
  if (exists(stage) && !read(stage).equals(Buffer.from(bytes))) {
    // This deterministic scratch file is owned solely by this writer. The
    // historical target was never opened for truncation; discard a partial stage.
    fs.unlinkSync(checked(stage));
  }
  if (!exists(stage)) {
    const fd = fs.openSync(checked(stage), 'wx');
    try { fs.writeFileSync(fd, bytes); fs.fsyncSync(fd); }
    finally { fs.closeSync(fd); }
  }
  if (exclusive) {
    // Hard-link creation is atomic and refuses an existing destination.
    fs.linkSync(checked(stage), checked(file));
    fs.unlinkSync(checked(stage));
  } else fs.renameSync(checked(stage), checked(file));
};
const writeJSON = (file, value, exclusive = false) => write(file, JSON.stringify(value, null, 2) + '\n', exclusive);
const tree = (relative) => {
  const rows = [];
  const base = checked(relative);
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      assert.ok(!entry.isSymbolicLink(), `linked entry: ${entry.name}`);
      const file = path.join(dir, entry.name);
      if (entry.isDirectory()) walk(file);
      else {
        assert.ok(entry.isFile(), `unsupported entry: ${file}`);
        rows.push({ file: path.relative(base, file).split(path.sep).join('/'), sha256: hash(fs.readFileSync(file)) });
      }
    }
  };
  walk(base);
  return rows.sort((a, b) => a.file.localeCompare(b.file));
};
const snapshotFile = `${audit}/continuation-dispositions.json`;
const snapshot = json(snapshotFile);
const snapshotSHA256 = hash(read(snapshotFile));
assert.equal(snapshot.total, 85);
const ids = snapshot.pairArchiveCandidates;
assert.deepEqual([...ids].sort(), [
  '2026-08-07-foundation-upgrade-base-v0-10-3', '2026-08-08-foundation-upgrade-base-v0-10-5',
  '2026-08-12-foundation-v0-10-14-clean-consumer', '2026-10-12-foundation-upgrade-base-v0-16-1', 'cmdb-host-import-export'
].sort());
const frozenFiles = ['dispositions.json', 'archive-dry-run.json', 'archive-result.json', 'second-batch-result.json'];
const frozen = frozenFiles.map((file) => ({ file: `${audit}/${file}`, sha256: hash(read(`${audit}/${file}`)) }));
const preflightFile = `${audit}/continuation-archive-preflight.json`;
const sealFile = `${audit}/continuation-archive-preflight-seal.json`;
const resultFile = `${audit}/continuation-archive-result.json`;
const journalFile = `${audit}/continuation-archive-journal.jsonl`;
const apply = process.argv.includes('--apply');
let preflight;
if (!apply) {
  assert.ok(!exists(preflightFile) && !exists(journalFile), 'preflight already frozen; use --apply to resume');
  const operations = ids.map((id) => {
    const decision = snapshot.records.find((row) => row.id === id);
    assert.ok(decision.canPairArchive && decision.lifecycleChangeAuthorized);
    assert.ok(['completed', 'abandoned'].includes(decision.disposition));
    const sourceTask = `.harness/tasks/${id}`;
    const sourceEvidence = `.harness/evidence/${id}`;
    const targetTask = `.harness/archive/2026-11/tasks/${id}`;
    const targetEvidence = `.harness/archive/2026-11/evidence/${id}`;
    assert.ok(exists(sourceTask) && exists(sourceEvidence));
    assert.ok(!exists(targetTask) && !exists(targetEvidence), `archive collision: ${id}`);
    for (const file of ['commands.json', 'review.md', 'summary.md']) assert.ok(exists(`${sourceEvidence}/${file}`));
    const manifest = json(`${sourceTask}/manifest.json`);
    const taskDoc = id === 'cmdb-host-import-export' ? null : manifest.taskDoc ?? manifest.linkage?.taskPacket;
    assert.ok(taskDoc || id === 'cmdb-host-import-export', `missing original card: ${id}`);
    return { id, status: decision.disposition, owner: decision.owner, successor: decision.successor,
      remainingAcceptance: decision.remainingAcceptance, proofPaths: decision.proofPaths,
      sourceTask, sourceEvidence, targetTask, targetEvidence, taskDoc,
      originalTask: tree(sourceTask), originalEvidence: tree(sourceEvidence),
      originalDocSHA256: taskDoc ? hash(read(taskDoc)) : null };
  });
  preflight = { recordedAt: new Date().toISOString(), snapshotSHA256, frozen, operations };
  writeJSON(preflightFile, preflight, true);
  writeJSON(sealFile, { preflightSHA256: hash(read(preflightFile)), snapshotSHA256 }, true);
  console.log(JSON.stringify({ dryRun: true, candidates: operations.length, snapshotSHA256 }));
} else {
  preflight = json(preflightFile);
  assert.deepEqual(json(sealFile), { preflightSHA256: hash(read(preflightFile)), snapshotSHA256 }, 'preflight seal mismatch');
  assert.equal(preflight.snapshotSHA256, snapshotSHA256, 'snapshot changed after preflight');
  assert.deepEqual(preflight.frozen, frozen, 'prior batch changed');
  assert.deepEqual(preflight.operations.map((row) => row.id), ids);
  const journal = (phase, id) => fs.appendFileSync(checked(journalFile), JSON.stringify({ at: new Date().toISOString(), phase, id, snapshotSHA256 }) + '\n');
  const backup = (file, target, expected) => {
    if (!exists(target)) {
      const bytes = read(file);
      assert.equal(hash(bytes), expected, `original changed: ${file}`);
      write(target, bytes, true);
    }
    assert.equal(hash(read(target)), expected, `backup changed: ${target}`);
    return read(target);
  };
  const move = (source, target) => {
    assert.notEqual(exists(source), exists(target), `ambiguous source/target: ${source}`);
    if (exists(source)) {
      fs.mkdirSync(path.dirname(checked(target)), { recursive: true });
      fs.renameSync(checked(source), checked(target));
    }
  };
  let evidenceFiles = 0;
  for (const op of preflight.operations) {
    const decision = snapshot.records.find((row) => row.id === op.id);
    assert.equal(op.sourceTask, `.harness/tasks/${op.id}`);
    assert.equal(op.sourceEvidence, `.harness/evidence/${op.id}`);
    assert.equal(op.targetTask, `.harness/archive/2026-11/tasks/${op.id}`);
    assert.equal(op.targetEvidence, `.harness/archive/2026-11/evidence/${op.id}`);
    assert.equal(op.status, decision.disposition);
    assert.deepEqual(op.owner, decision.owner);
    assert.deepEqual(op.successor, decision.successor);
    assert.deepEqual(op.remainingAcceptance, decision.remainingAcceptance);
    assert.deepEqual(op.proofPaths, decision.proofPaths);
    journal('start-or-resume', op.id);
    assert.notEqual(exists(op.sourceTask), exists(op.targetTask));
    assert.notEqual(exists(op.sourceEvidence), exists(op.targetEvidence));
    const currentTask = exists(op.sourceTask) ? op.sourceTask : op.targetTask;
    const currentEvidence = exists(op.sourceEvidence) ? op.sourceEvidence : op.targetEvidence;
    // Reject concurrent additions/deletions and edits before preparing or moving.
    const currentFiles = tree(currentTask);
    assert.deepEqual(currentFiles.map((row) => row.file), op.originalTask.map((row) => row.file));
    for (const row of op.originalTask.filter((row) => row.file !== 'manifest.json')) assert.equal(hash(read(`${currentTask}/${row.file}`)), row.sha256);
    const closeout = `# Historical disposition — 2026-11-17\n\nDisposition: ${op.status}\n\nOwner: ${op.owner}\nSuccessor: ${op.successor}\n\n${op.remainingAcceptance}\n\nDecision and independent reviews: ${snapshotFile}. Original logs, commands, summaries and review retained byte-for-byte. This closes only the original scope; current product, external, production and final maintainer acceptance remain separate.\n`;
    assert.ok(!op.originalEvidence.some((row) => row.file === 'history-closeout.md'), 'original closeout must not be overwritten');
    const expectedEvidence = [...op.originalEvidence];
    if (exists(`${currentEvidence}/history-closeout.md`)) expectedEvidence.push({ file: 'history-closeout.md', sha256: hash(Buffer.from(closeout)) });
    assert.deepEqual(tree(currentEvidence), expectedEvidence.sort((a, b) => a.file.localeCompare(b.file)), `evidence inventory changed: ${op.id}`);
    const manifestFile = `${currentTask}/manifest.json`;
    const originalHash = op.originalTask.find((row) => row.file === 'manifest.json').sha256;
    const original = backup(manifestFile, `${audit}/continuation-originals/manifests/${op.id}.json`, originalHash);
    const manifest = JSON.parse(original);
    assert.equal(op.taskDoc, op.id === 'cmdb-host-import-export' ? null : manifest.taskDoc ?? manifest.linkage?.taskPacket);
    manifest.status = op.status;
    manifest.historyCloseout = { date: '2026-11-17', originalStatus: JSON.parse(original).status ?? null,
      disposition: op.status, owner: op.owner, successor: op.successor, remainingAcceptance: op.remainingAcceptance,
      evidence: snapshotFile, snapshotSHA256,
      taskDocGap: op.taskDoc ? null : 'Original manifest and plan retained; no historical docs card fabricated.' };
    const expectedManifest = JSON.stringify(manifest, null, 2) + '\n';
    assert.ok(hash(read(manifestFile)) === originalHash || read(manifestFile).toString('utf8') === expectedManifest, `concurrent manifest edit: ${op.id}`);
    write(manifestFile, expectedManifest);
    write(`${currentEvidence}/history-closeout.md`, closeout);
    journal('prepared', op.id);
    move(op.sourceTask, op.targetTask);
    journal('task-moved', op.id);
    move(op.sourceEvidence, op.targetEvidence);
    journal('pair-moved', op.id);
    for (const row of op.originalEvidence) {
      assert.equal(hash(read(`${op.targetEvidence}/${row.file}`)), row.sha256);
      evidenceFiles++;
    }
    for (const row of op.originalTask.filter((row) => row.file !== 'manifest.json')) assert.equal(hash(read(`${op.targetTask}/${row.file}`)), row.sha256);
    if (op.taskDoc) {
      const doc = backup(op.taskDoc, `${audit}/continuation-originals/cards/${op.id}.md`, op.originalDocSHA256).toString('utf8');
      const updated = /^---\r?\n/.test(doc)
        ? doc.replace(/^---\r?\n([\s\S]*?)\r?\n---/, (_, fields) => {
          const status = /^status:.*$/m.test(fields) ? fields.replace(/^status:.*$/m, 'status: Archived') : fields + '\nstatus: Archived';
          const dated = /^updated_at:.*$/m.test(status) ? status.replace(/^updated_at:.*$/m, 'updated_at: 2026-11-17') : status + '\nupdated_at: 2026-11-17';
          return `---\n${dated}\n---`;
        }) : `---\ntitle: ${op.id}\ndoc_type: Remediation\nlayer: business\nstatus: Archived\nlinked_contracts:\n  - docs/PROJECT_INHERITANCE.md\nupdated_at: 2026-11-17\n---\n\n${doc}`;
      const expected = updated + `\n\n## Result — Historical Disposition 2026-11-17\n\n${op.status}. ${op.remainingAcceptance}\n\nOwner / successor / independent original-scope reviews: ${snapshotFile}. Original card body retained; only historical scope archived.\n`;
      assert.ok(read(op.taskDoc).toString('utf8') === doc || read(op.taskDoc).toString('utf8') === expected, `concurrent card edit: ${op.id}`);
      write(op.taskDoc, expected);
    }
    assert.ok(resolveArchiveAwarePath(root, `${op.sourceTask}/manifest.json`));
    assert.ok(resolveArchiveAwarePath(root, `${op.sourceEvidence}/summary.md`));
    if (!op.taskDoc) assert.ok(resolveArchiveAwarePath(root, `${op.sourceTask}/plan.md`));
    journal('complete', op.id);
  }
  assert.deepEqual(frozenFiles.map((file) => ({ file: `${audit}/${file}`, sha256: hash(read(`${audit}/${file}`)) })), preflight.frozen);
  const result = { snapshotSHA256, archived: 5, originalEvidenceFiles: evidenceFiles, hashesPreserved: true,
    priorBatchesUnchanged: true, phasesComplete: true, operations: preflight.operations };
  if (exists(resultFile)) assert.deepEqual(json(resultFile), result);
  else writeJSON(resultFile, result, true);
  console.log(JSON.stringify({ archived: 5, originalEvidenceFiles: evidenceFiles, hashesPreserved: true, priorBatchesUnchanged: true }));
}
