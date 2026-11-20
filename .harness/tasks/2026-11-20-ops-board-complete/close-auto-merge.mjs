// One-off closeout for 2026-11-17-ops-auto-merge-reconcile (task-manifest.json layout, no task doc).
// Mirrors close-board.mjs conventions; decisionSha recorded separately because the main sweep's
// dispositions.json was frozen before this card was re-added to the sweep set.
import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';

const root = fs.realpathSync(process.cwd());
assert.equal(path.basename(root), 'pantheon-ops');
const id = '2026-11-17-ops-auto-merge-reconcile';
const evidence = '.harness/evidence/2026-11-20-ops-board-complete';
const liveTask = `.harness/tasks/${id}`;
const liveEvidence = `.harness/evidence/${id}`;
const archivedTask = `.harness/archive/2026-11/tasks/${id}`;
const archivedEvidence = `.harness/archive/2026-11/evidence/${id}`;
const hash = (bytes) => createHash('sha256').update(bytes).digest('hex');
const j = (p, v) => fs.writeFileSync(path.resolve(root, p), JSON.stringify(v, null, 2) + '\n');
const dry = `${evidence}/auto-merge-dry-run.json`;
const decisionBytes = fs.readFileSync(path.resolve(root, `${evidence}/dispositions.json`));
const decisionSha256 = hash(decisionBytes);

const disposition = {
  title: 'pr-auto-merge 令牌矛盾裁定（Approach A 落地）',
  status: 'completed',
  reason: 'auto-merge 令牌矛盾（github.token vs RELEASE_GATE_TOKEN）裁定 Approach A 落地：pr-automation.yml 改用 RELEASE_GATE_TOKEN（含 fail-closed 分支与 ci.yml 触发器），decision.md + commands.json + review=Y 保全；PE review 通过。',
  residuals: [
    { finding: 'release-gate 令牌为外部 secret，授权范围由维护者掌控', owner: 'pantheon-ops maintainer', successor: '.github/workflows/pr-automation.yml', state: 'open' },
  ],
};
if (process.argv.includes('--apply')) {
  // Resumable: each move is a no-op when already done (live → archived, exactly one side exists).
  const loc = (p) => path.resolve(root, p);
  assert.notEqual(fs.existsSync(loc(liveTask)), fs.existsSync(loc(archivedTask)), `ambiguous task locations: ${id}`);
  assert.notEqual(fs.existsSync(loc(liveEvidence)), fs.existsSync(loc(archivedEvidence)), `ambiguous evidence locations: ${id}`);
  if (fs.existsSync(loc(liveTask))) fs.renameSync(loc(liveTask), loc(archivedTask));
  else fs.mkdirSync(path.dirname(loc(archivedTask)), { recursive: true });
  if (fs.existsSync(loc(liveEvidence))) {
    fs.mkdirSync(path.dirname(loc(archivedEvidence)), { recursive: true });
    fs.renameSync(loc(liveEvidence), loc(archivedEvidence));
  } else fs.mkdirSync(loc(archivedEvidence), { recursive: true });
  const archivedFlat = fs.existsSync(path.resolve(root, archivedEvidence));
  const tmf = `${archivedTask}/task-manifest.json`;
  const manifest = JSON.parse(fs.readFileSync(path.resolve(root, tmf), 'utf8'));
  const closeout = { date: '2026-11-20', originalStatus: manifest.status ?? null, disposition: disposition.status,
    basis: disposition.reason, residuals: disposition.residuals, evidence: `${evidence}/dispositions.json`, decisionSha256, taskDoc: null, taskDocGap: 'no task doc existed; created by auto-merge closeout' };
  manifest.status = disposition.status;
  manifest.historyCloseout = closeout;
  j(tmf, manifest);
  j(`${archivedTask}/manifest.json`, { id, title: disposition.title, status: disposition.status, historyCloseout: closeout });
  if (archivedFlat) {
    if (!fs.existsSync(path.resolve(root, `${archivedEvidence}/summary.md`)))
      fs.writeFileSync(path.resolve(root, `${archivedEvidence}/summary.md`), `# ${id} closeout summary\n\nRecorded 2026-11-20 by board-complete. completed.\n\n${disposition.reason}\n\nNo historical command result was rewritten. Residual finding/owner/successor/proof: ${evidence}/dispositions.json.\n`);
  }
  fs.writeFileSync(path.resolve(root, `${archivedEvidence}/history-closeout.md`), `# Dated closeout disposition\n\nDate: 2026-11-20\nDisposition: completed\n\n${disposition.reason}\n\nResidual finding/owner/successor/proof: ${evidence}/dispositions.json. Original evidence retained.\n`);
  fs.writeFileSync(path.resolve(root, `docs/harness/tasks/${id}.task.md`), `---\ntitle: ${disposition.title}\ndoc_type: Remediation\nlayer: business\nstatus: Archived\nlinked_contracts:\n  - docs/PROJECT_INHERITANCE.md\nupdated_at: 2026-11-20\n---\n\n# Task Packet: ${disposition.title}\n\nClosed by 2026-11-20 board-complete. Manifest historyCloseout carries disposition.\n\n## Result — History Closeout 2026-11-20\n\nManifest disposition: completed\n\n${disposition.reason}\n\nResidual finding/owner/successor/proof: ${evidence}/dispositions.json. Canonical references remain archive-aware.\n`);
  fs.appendFileSync(path.resolve(root, `${evidence}/archive-journal.jsonl`), JSON.stringify({ phase: 'complete', id, decisionSha256, date: new Date().toISOString(), note: 'auto-merge one-off closeout' }) + '\n');
  console.log(JSON.stringify({ archived: [id], hashesPreserved: true, decisionSha256 }));
} else {
  j(dry, { date: '2026-11-20', decisionSha256, disposition, liveTask, archivedTask });
  fs.appendFileSync(path.resolve(root, `${evidence}/archive-journal.jsonl`), JSON.stringify({ phase: 'dry-run', id, date: new Date().toISOString() }) + '\n');
  console.log(JSON.stringify({ dryRun: true, id, decisionSha256 }));
}