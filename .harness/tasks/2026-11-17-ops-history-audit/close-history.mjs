import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { resolveArchiveAwarePath } from '../../../scripts/task-manifest.mjs';
import { auditId, decisions } from './history-decisions.mjs';

const root = fs.realpathSync(process.cwd());
assert.equal(path.basename(root), 'pantheon-ops');
const evidence = `.harness/evidence/${auditId}`;
const apply = process.argv.includes('--apply');
const relocate = process.argv.includes('--relocate-preserved');
assert.ok(!(apply && relocate), 'relocation and archive apply must run separately');
const hash = (bytes) => createHash('sha256').update(bytes).digest('hex');
const checked = (relative) => {
  const absolute = path.resolve(root, relative);
  const rel = path.relative(root, absolute);
  assert.ok(rel && !rel.startsWith('..') && !path.isAbsolute(rel), `outside repository: ${relative}`);
  let ancestor = absolute;
  while (ancestor !== root) {
    if (fs.existsSync(ancestor)) assert.ok(!fs.lstatSync(ancestor).isSymbolicLink(), `linked path: ${ancestor}`);
    ancestor = path.dirname(ancestor);
  }
  return absolute;
};
const exists = (relative) => fs.existsSync(checked(relative));
const tree = (relative) => {
  const absolute = checked(relative);
  if (!fs.existsSync(absolute)) return [];
  const rows = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      assert.ok(!entry.isSymbolicLink(), `linked entry: ${entry.name}`);
      const file = path.join(dir, entry.name);
      if (entry.isDirectory()) walk(file);
      else if (entry.isFile()) rows.push({ file: path.relative(absolute, file).split(path.sep).join('/'), sha256: hash(fs.readFileSync(file)) });
      else throw new Error(`unsupported entry: ${file}`);
    }
  };
  walk(absolute);
  return rows.sort((a, b) => a.file.localeCompare(b.file));
};
const write = (relative, content) => {
  fs.mkdirSync(path.dirname(checked(relative)), { recursive: true });
  fs.writeFileSync(checked(relative), content);
};
const jsonWrite = (relative, value) => write(relative, JSON.stringify(value, null, 2) + '\n');
const decisionBytes = fs.readFileSync(checked(`${evidence}/dispositions.json`));
assert.deepEqual(JSON.parse(decisionBytes).archiveDecisions, decisions, 'decision artifact must match scope/status/residuals');
const decisionSha256 = hash(decisionBytes);
const dryPath = `${evidence}/archive-dry-run.json`;
let operations;
if (apply || relocate) {
  // Immutable preflight retains original hashes and destinations across interruption/resume.
  const preflight = JSON.parse(fs.readFileSync(checked(dryPath), 'utf8'));
  assert.equal(preflight.decisionSha256, decisionSha256, 'decisions changed since preflight');
  operations = preflight.operations;
  assert.deepEqual(operations.map((operation) => Object.fromEntries(Object.keys(decisions[0]).map((key) => [key, operation[key]]))), decisions);
} else {
  assert.ok(!exists(`${evidence}/archive-journal.jsonl`), 'archive started; resume --apply rather than replace preflight');
  operations = decisions.map((decision) => {
    const { id, taskDoc } = decision;
    assert.match(id, /^\d{4}-\d{2}-\d{2}-[a-z0-9-]+$/);
    if (taskDoc) assert.ok(exists(taskDoc), `explicit task document missing: ${id}`);
    else assert.ok(decision.taskDocGap, `missing task document disposition: ${id}`);
    const liveTask = `.harness/tasks/${id}`;
    const liveEvidence = `.harness/evidence/${id}`;
    const archivedTask = `.harness/archive/2026-11/tasks/${id}`;
    const archivedEvidence = `.harness/archive/2026-11/evidence/${id}`;
    const duplicate = exists(archivedTask);
    assert.equal(exists(archivedEvidence), duplicate, `unpaired archive: ${id}`);
    const taskTarget = duplicate ? `${evidence}/artifacts/preserved-live/tasks/${id}` : archivedTask;
    const evidenceTarget = duplicate ? `${evidence}/artifacts/preserved-live/evidence/${id}` : archivedEvidence;
    assert.ok(exists(`${liveTask}/manifest.json`) && exists(liveEvidence), `unpaired live package: ${id}`);
    for (const required of ['review.md', 'commands.json']) assert.ok(exists(`${liveEvidence}/${required}`), `missing ${required}: ${id}`);
    assert.ok(!exists(taskTarget) && !exists(evidenceTarget), `destination collision: ${id}`);
    return { ...decision, duplicate, liveTask, liveEvidence, taskTarget, evidenceTarget,
      originalTask: tree(liveTask), originalEvidence: tree(liveEvidence), originalDocSha256: taskDoc ? hash(fs.readFileSync(checked(taskDoc))) : null,
      existingArchiveTask: duplicate ? tree(archivedTask) : null, existingArchiveEvidence: duplicate ? tree(archivedEvidence) : null };
  });
  jsonWrite(dryPath, { date: '2026-11-17', decisionSha256, operations });
  console.log(JSON.stringify({ dryRun: true, candidates: operations.length, duplicates: operations.filter((o) => o.duplicate).map((o) => o.id) }));
}

const relocationPath = `${evidence}/preserved-live-relocation.json`;
const relocationSource = `${evidence}/artifacts/preserved-live`;
const relocationTarget = `.harness/archive/2026-11/preserved-live/${auditId}`;
const validateRelocation = (intent) => {
  assert.equal(intent.source, relocationSource);
  assert.equal(intent.target, relocationTarget);
  assert.equal(intent.preflightSha256, hash(fs.readFileSync(checked(dryPath))));
  assert.equal(intent.archiveResultSha256, hash(fs.readFileSync(checked(`${evidence}/archive-result.json`))));
  const duplicates = operations.filter((o) => o.duplicate);
  const mappings = duplicates.map((o) => ({ id: o.id, oldTask: o.taskTarget, oldEvidence: o.evidenceTarget,
    taskTarget: `${relocationTarget}/tasks/${o.id}`, evidenceTarget: `${relocationTarget}/evidence/${o.id}` }));
  assert.deepEqual(intent.mappings, mappings, 'relocation may only map the original duplicate operations');
  const expected = duplicates.flatMap((o) => [
    ...o.originalTask.map((r) => ({ ...r, file: `tasks/${o.id}/${r.file}` })),
    ...o.originalEvidence.map((r) => ({ ...r, file: `evidence/${o.id}/${r.file}` })),
  ]).sort((a, b) => a.file.localeCompare(b.file));
  assert.deepEqual(intent.files, expected, 'relocation must preserve exactly the original duplicate files');
  for (const o of duplicates) {
    assert.deepEqual(tree(`.harness/archive/2026-11/tasks/${o.id}`), o.existingArchiveTask);
    assert.deepEqual(tree(`.harness/archive/2026-11/evidence/${o.id}`), o.existingArchiveEvidence);
  }
};
if (relocate) {
  let intent;
  if (exists(relocationPath)) intent = JSON.parse(fs.readFileSync(checked(relocationPath), 'utf8'));
  else {
    assert.ok(exists(relocationSource) && !exists(relocationTarget), 'relocation destination must be absent');
    const result = JSON.parse(fs.readFileSync(checked(`${evidence}/archive-result.json`), 'utf8'));
    assert.equal(result.phasesComplete, true);
    assert.equal(result.hashesPreserved, true);
    assert.equal(result.decisionSha256, decisionSha256);
    intent = { source: relocationSource, target: relocationTarget,
      preflightSha256: hash(fs.readFileSync(checked(dryPath))), archiveResultSha256: hash(fs.readFileSync(checked(`${evidence}/archive-result.json`))),
      files: tree(relocationSource), mappings: operations.filter((o) => o.duplicate).map((o) => ({ id: o.id,
        oldTask: o.taskTarget, oldEvidence: o.evidenceTarget, taskTarget: `${relocationTarget}/tasks/${o.id}`, evidenceTarget: `${relocationTarget}/evidence/${o.id}` })) };
    validateRelocation(intent);
    jsonWrite(relocationPath, intent);
  }
  validateRelocation(intent);
  assert.notEqual(exists(relocationSource), exists(relocationTarget), 'exactly one relocation root must exist');
  assert.deepEqual(tree(exists(relocationSource) ? relocationSource : relocationTarget), intent.files);
  if (exists(relocationSource)) {
    fs.mkdirSync(path.dirname(checked(relocationTarget)), { recursive: true });
    fs.renameSync(checked(relocationSource), checked(relocationTarget));
  }
  assert.ok(!exists(relocationSource));
  assert.deepEqual(tree(relocationTarget), intent.files);
  validateRelocation(intent);
  fs.appendFileSync(checked(`${evidence}/archive-journal.jsonl`), JSON.stringify({ phase: 'preserved-live-relocated', date: new Date().toISOString(), source: relocationSource, target: relocationTarget }) + '\n');
  jsonWrite(`${evidence}/preserved-live-relocation-result.json`, { status: 'passed', sourceAbsent: true, originalHashesPreserved: true, canonicalArchivesUnchanged: true, target: relocationTarget });
  console.log(JSON.stringify({ relocated: intent.mappings.length, sourceAbsent: true, hashesPreserved: true }));
} else if (apply) {
  if (exists(relocationPath)) {
    const intent = JSON.parse(fs.readFileSync(checked(relocationPath), 'utf8'));
    validateRelocation(intent);
    assert.ok(!exists(intent.source) && exists(intent.target), 'finish relocation before resuming archive');
    assert.deepEqual(tree(intent.target), intent.files);
    operations = operations.map((o) => {
      const mapping = intent.mappings.find((m) => m.id === o.id);
      return mapping ? { ...o, taskTarget: mapping.taskTarget, evidenceTarget: mapping.evidenceTarget } : o;
    });
  }
  const journalPath = checked(`${evidence}/archive-journal.jsonl`);
  const journal = (phase, operation) => fs.appendFileSync(journalPath, JSON.stringify({ phase, id: operation.id, decisionSha256, date: new Date().toISOString() }) + '\n');
  const backup = (source, target, expectedHash) => {
    if (!exists(target)) {
      assert.equal(hash(fs.readFileSync(checked(source))), expectedHash, `source changed before backup: ${source}`);
      fs.mkdirSync(path.dirname(checked(target)), { recursive: true });
      fs.copyFileSync(checked(source), checked(target), fs.constants.COPYFILE_EXCL);
    }
    assert.equal(hash(fs.readFileSync(checked(target))), expectedHash, `backup differs: ${target}`);
    return fs.readFileSync(checked(target));
  };
  const moveOrResume = (source, target) => {
    assert.notEqual(exists(source), exists(target), `exactly one source/target must exist: ${source}`);
    if (exists(source)) {
      fs.mkdirSync(path.dirname(checked(target)), { recursive: true });
      fs.renameSync(checked(source), checked(target));
    }
  };
  for (const operation of operations) {
    const { id, liveTask, liveEvidence, taskTarget, evidenceTarget, duplicate, taskDoc } = operation;
    journal('resume-start', operation);
    // Every phase is repeatable; resume also handles interruption between the two moves.
    const currentTask = exists(liveTask) ? liveTask : taskTarget;
    const currentEvidence = exists(liveEvidence) ? liveEvidence : evidenceTarget;
    assert.notEqual(exists(liveTask), exists(taskTarget), `ambiguous task locations: ${id}`);
    assert.notEqual(exists(liveEvidence), exists(evidenceTarget), `ambiguous evidence locations: ${id}`);
    const manifestFile = `${currentTask}/manifest.json`;
    const originalManifestHash = operation.originalTask.find((row) => row.file === 'manifest.json').sha256;
    const originalManifest = backup(manifestFile, `${evidence}/artifacts/original-manifests/${id}.json`, originalManifestHash);
    if (!duplicate) {
      const manifest = JSON.parse(originalManifest);
      manifest.historyCloseout = { date: '2026-11-17', originalStatus: manifest.status ?? null,
        disposition: operation.status, basis: operation.reason, residuals: operation.residuals,
        evidence: `${evidence}/dispositions.json`, decisionSha256, taskDoc, taskDocGap: operation.taskDocGap };
      manifest.status = operation.status;
      jsonWrite(manifestFile, manifest);
      if (!operation.originalEvidence.some((row) => row.file === 'summary.md')) {
        const hosted = id.includes('branch-hygiene') ? `\nHistorical hosted proof recovered today: ${evidence}/branch-hygiene-27738932286.log and branch-hygiene-27737850759.log. No new cleanup executed.\n` : '';
        write(`${currentEvidence}/summary.md`, `# ${id} completion readout\n\nRecorded 2026-11-17 from retained commands.json and independent review.md.\n\n${operation.reason}\n${hosted}\nNo historical command result was rewritten. Residual finding/owner/successor/proof: ${evidence}/dispositions.json. Current candidate, production and maintainer gates remain separate.\n`);
      }
      write(`${currentEvidence}/history-closeout.md`, `# Dated historical disposition\n\nDate: 2026-11-17\nDisposition: ${operation.status}\n\n${operation.reason}\n\nResidual finding/owner/successor/proof: ${evidence}/dispositions.json. Original evidence retained; reconstructed missing summaries are dated.\n`);
    }
    journal('prepared', operation);
    moveOrResume(liveTask, taskTarget);
    journal('task-moved', operation);
    moveOrResume(liveEvidence, evidenceTarget);
    journal('pair-moved', operation);
    for (const row of operation.originalEvidence) assert.equal(hash(fs.readFileSync(checked(`${evidenceTarget}/${row.file}`))), row.sha256, `original evidence changed: ${id}/${row.file}`);
    for (const row of operation.originalTask.filter((row) => duplicate || row.file !== 'manifest.json')) assert.equal(hash(fs.readFileSync(checked(`${taskTarget}/${row.file}`))), row.sha256, `original task changed: ${id}/${row.file}`);
    if (duplicate) {
      assert.deepEqual(tree('.harness/archive/2026-11/tasks/' + id), operation.existingArchiveTask);
      assert.deepEqual(tree('.harness/archive/2026-11/evidence/' + id), operation.existingArchiveEvidence);
    }
    if (taskDoc) {
    const docBytes = backup(taskDoc, `${evidence}/artifacts/original-task-docs/${id}.md`, operation.originalDocSha256);
    const originalDoc = docBytes.toString('utf8');
    let updatedDoc;
    if (/^---\r?\n/.test(originalDoc)) {
      updatedDoc = originalDoc.replace(/^---\r?\n([\s\S]*?)\r?\n---/, (_, fields) => {
        const status = /^status:.*$/m.test(fields) ? fields.replace(/^status:.*$/m, 'status: Archived') : `${fields}\nstatus: Archived`;
        const updated = /^updated_at:.*$/m.test(status) ? status.replace(/^updated_at:.*$/m, 'updated_at: 2026-11-17') : `${status}\nupdated_at: 2026-11-17`;
        return `---\n${updated}\n---`;
      });
    } else {
      updatedDoc = `---\ntitle: ${id}\ndoc_type: Remediation\nlayer: business\nstatus: Archived\nlinked_contracts:\n  - docs/PROJECT_INHERITANCE.md\nupdated_at: 2026-11-17\n---\n\n${originalDoc}`;
    }
    const expectedDoc = updatedDoc + `\n\n## Result — History Closeout 2026-11-17\n\nManifest disposition: ${operation.status}\n\n${operation.reason}\n\nResidual finding/owner/successor/proof: ${evidence}/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Current candidate, production and maintainer gates remain separate.\n`;
    const currentDoc = fs.readFileSync(checked(taskDoc), 'utf8');
    assert.ok(currentDoc === originalDoc || currentDoc === expectedDoc, `concurrent document edit: ${taskDoc}`);
    write(taskDoc, expectedDoc);
    assert.equal(fs.readFileSync(checked(taskDoc), 'utf8'), expectedDoc);
    assert.match(expectedDoc.split(/\r?\n---/)[0], /\nstatus: Archived\r?\n/);
    }
    assert.ok(resolveArchiveAwarePath(root, `.harness/tasks/${id}/manifest.json`));
    assert.ok(resolveArchiveAwarePath(root, `.harness/evidence/${id}/summary.md`));
    journal('complete', operation);
  }
  const resultPath = exists(`${evidence}/archive-result.json`) ? `${evidence}/archive-resume-result-${Date.now()}.json` : `${evidence}/archive-result.json`;
  jsonWrite(resultPath, { date: '2026-11-17', decisionSha256, operations, hashesPreserved: true, phasesComplete: true });
  console.log(JSON.stringify({ archived: operations.filter((o) => !o.duplicate).length, liveDuplicatesPreserved: operations.filter((o) => o.duplicate).length, hashesPreserved: true }));
}
