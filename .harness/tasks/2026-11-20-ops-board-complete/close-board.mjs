import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { resolveArchiveAwarePath } from '../../../scripts/task-manifest.mjs';

const root = fs.realpathSync(process.cwd());
assert.equal(path.basename(root), 'pantheon-ops');
const auditId = '2026-11-20-ops-board-complete';
const evidence = `.harness/evidence/${auditId}`;
const apply = process.argv.includes('--apply');
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
const dispositions = JSON.parse(decisionBytes).dispositions;
const decisionSha256 = hash(decisionBytes);
const dryPath = `${evidence}/archive-dry-run.json`;
let operations;
if (apply) {
  const preflight = JSON.parse(fs.readFileSync(checked(dryPath), 'utf8'));
  assert.equal(preflight.decisionSha256, decisionSha256, 'decisions changed since preflight');
  operations = preflight.operations;
  assert.deepEqual(operations.map((o) => o.id), Object.keys(dispositions));
} else {
  assert.ok(!exists(`${evidence}/archive-journal.jsonl`), 'archive started; resume --apply rather than replace preflight');
  operations = Object.entries(dispositions).map(([id, d]) => {
    const liveTask = `.harness/tasks/${id}`;
    const liveEvidence = `.harness/evidence/${id}`;
    const archivedTask = `.harness/archive/2026-11/tasks/${id}`;
    const archivedEvidence = `.harness/archive/2026-11/evidence/${id}`;
    assert.ok(!exists(archivedTask) && !exists(archivedEvidence), `destination collision: ${id}`);
    const taskDoc = exists(`docs/harness/tasks/${id}.task.md`) ? `docs/harness/tasks/${id}.task.md` : null;
    return {
      id, title: d.title, status: d.status, reason: d.reason, residuals: d.residuals || [],
      taskDoc, taskDocGap: taskDoc ? null : `no task doc existed; closeout created minimal ${id}.task.md`,
      liveTask, liveEvidence, archivedTask, archivedEvidence,
      originalTask: tree(liveTask), originalEvidence: tree(liveEvidence),
      originalDocSha256: taskDoc ? hash(fs.readFileSync(checked(taskDoc))) : null,
    };
  });
  jsonWrite(dryPath, { date: '2026-11-20', decisionSha256, operations });
  console.log(JSON.stringify({ dryRun: true, candidates: operations.length, noTaskDoc: operations.filter((o) => !o.taskDoc).map((o) => o.id) }));
}

if (apply) {
  const journalPath = checked(`${evidence}/archive-journal.jsonl`);
  const journal = (phase, operation) => fs.appendFileSync(journalPath, JSON.stringify({ phase, id: operation.id, decisionSha256, date: new Date().toISOString() }) + '\n');
  const manifestFileName = (dir) => exists(`${dir}/manifest.json`) ? 'manifest.json' : (exists(`${dir}/task-manifest.json`) ? 'task-manifest.json' : null);
  for (const op of operations) {
    const { id, liveTask, liveEvidence, archivedTask, archivedEvidence, taskDoc } = op;
    journal('resume-start', op);
    const currentTask = exists(liveTask) ? liveTask : archivedTask;
    const currentEvidence = exists(liveEvidence) ? liveEvidence : archivedEvidence;
    assert.notEqual(exists(liveTask), exists(archivedTask), `ambiguous task locations: ${id}`);
    // Evidence may legitimately be absent on both sides (e.g. superseded no-evidence cards); only guard against duplication.
    if (exists(liveEvidence) || exists(archivedEvidence)) assert.notEqual(exists(liveEvidence), exists(archivedEvidence), `ambiguous evidence locations: ${id}`);

    // Archive the task directory first (evidence may not exist yet).
    if (exists(liveTask)) {
      fs.mkdirSync(path.dirname(checked(archivedTask)), { recursive: true });
      fs.renameSync(checked(liveTask), checked(archivedTask));
    }
    journal('task-moved', op);

    // Manifest: append historyCloseout + flip status on whatever manifest exists; create manifest.json if neither (k8s-recovery).
    const manifestFile = manifestFileName(archivedTask);
    let manifest = {};
    if (manifestFile) {
      manifest = JSON.parse(fs.readFileSync(checked(`${archivedTask}/${manifestFile}`), 'utf8'));
    }
    const closeout = { date: '2026-11-20', originalStatus: manifest.status ?? null, disposition: op.status,
      basis: op.reason, residuals: op.residuals, evidence: `${evidence}/dispositions.json`, decisionSha256, taskDoc, taskDocGap: op.taskDocGap };
    manifest.status = op.status;
    manifest.historyCloseout = closeout;
    if (manifestFile) {
      jsonWrite(`${archivedTask}/${manifestFile}`, manifest);
      if (manifestFile !== 'manifest.json') jsonWrite(`${archivedTask}/manifest.json`, { id, title: op.title, status: op.status, historyCloseout: closeout });
    } else {
      jsonWrite(`${archivedTask}/manifest.json`, { id, title: op.title, status: op.status, historyCloseout: closeout });
    }
    journal('manifest-flipped', op);

    // Evidence: move when present, materialize a minimal dated summary + closeout otherwise.
    if (exists(liveEvidence)) {
      fs.mkdirSync(path.dirname(checked(archivedEvidence)), { recursive: true });
      fs.renameSync(checked(liveEvidence), checked(archivedEvidence));
    } else {
      fs.mkdirSync(checked(archivedEvidence), { recursive: true });
    }
    journal('evidence-moved', op);
    const currentEv = exists(archivedEvidence) ? archivedEvidence : currentEvidence;
    if (!fs.existsSync(`${checked(currentEv)}/summary.md`)) {
      write(`${currentEv}/summary.md`, `# ${id} closeout summary\n\nRecorded 2026-11-20 by board-complete. ${op.status}.\n\n${op.reason}\n\nNo historical command result was rewritten. Residual finding/owner/successor/proof: ${evidence}/dispositions.json. Candidate/production/maintainer gates remain separate.\n`);
    }
    write(`${currentEv}/history-closeout.md`, `# Dated closeout disposition\n\nDate: 2026-11-20\nDisposition: ${op.status}\n\n${op.reason}\n\nResidual finding/owner/successor/proof: ${evidence}/dispositions.json. Original evidence retained.\n`);
    for (const row of op.originalEvidence) assert.equal(hash(fs.readFileSync(checked(`${archivedEvidence}/${row.file}`))), row.sha256, `original evidence changed: ${id}/${row.file}`);
    journal('evidence-prepared', op);

    // Task doc: flip status: Archived + Result when one exists; otherwise create minimal.
    let updatedDoc;
    const archivedBody = `\n\n## Result — History Closeout 2026-11-20\n\nManifest disposition: ${op.status}\n\n${op.reason}\n\nResidual finding/owner/successor/proof: ${evidence}/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.\n`;
    if (taskDoc) {
      const originalDoc = fs.readFileSync(checked(taskDoc), 'utf8');
      updatedDoc = /^---\r?\n/.test(originalDoc)
        ? originalDoc.replace(/^---\r?\n([\s\S]*?)\r?\n---/, (_, fields) => {
            const status = /^status:.*$/m.test(fields) ? fields.replace(/^status:.*$/m, 'status: Archived') : `${fields}\nstatus: Archived`;
            const updated = /^updated_at:.*$/m.test(status) ? status.replace(/^updated_at:.*$/m, 'updated_at: 2026-11-20') : `${status}\nupdated_at: 2026-11-20`;
            return `---\n${updated}\n---`;
          }) + archivedBody
        : `---\ntitle: ${id}\ndoc_type: Remediation\nlayer: business\nstatus: Archived\nlinked_contracts:\n  - docs/PROJECT_INHERITANCE.md\nupdated_at: 2026-11-20\n---\n\n---ORIGINAL NOT FRONTMATTER---\n\n${originalDoc}${archivedBody}`;
    } else {
      updatedDoc = `---\ntitle: ${op.title || id}\ndoc_type: Remediation\nlayer: business\nstatus: Archived\nlinked_contracts:\n  - docs/PROJECT_INHERITANCE.md\nupdated_at: 2026-11-20\n---\n\n# Task Packet: ${op.title || id}\n\nClosed by 2026-11-20 board-complete. Manifest historyCloseout carries disposition.\n${archivedBody}`;
    }
    write(taskDoc ?? `docs/harness/tasks/${id}.task.md`, updatedDoc);
    journal('doc-updated', op);

    assert.ok(resolveArchiveAwarePath(root, `.harness/tasks/${id}/manifest.json`), `archive-aware manifest missing: ${id}`);
    const sm = fs.readdirSync(checked(archivedEvidence));
    assert.ok(sm.includes('summary.md') && sm.includes('history-closeout.md'), `dated evidence missing for ${id}`);
    journal('complete', op);
  }
  const resultPath = exists(`${evidence}/archive-result.json`) ? `${evidence}/archive-resume-result-${Date.now()}.json` : `${evidence}/archive-result.json`;
  jsonWrite(resultPath, { date: '2026-11-20', decisionSha256, operations: operations.map((o) => ({ ...o, originalTask: undefined, originalEvidence: undefined })), hashesPreserved: true, phasesComplete: true });
  console.log(JSON.stringify({ archived: operations.length, hashesPreserved: true, evidence: resultPath }));
}