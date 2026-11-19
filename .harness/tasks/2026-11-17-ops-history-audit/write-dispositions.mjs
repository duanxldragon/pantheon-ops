import fs from 'node:fs';
import { auditId, decisions } from './history-decisions.mjs';
const dir = `.harness/evidence/${auditId}`;
const scan = JSON.parse(fs.readFileSync(`${dir}/history-scan.json`, 'utf8'));
const adjudicated = new Map(decisions.map((decision) => [decision.id, decision]));
const records = scan.records.map((record) => {
  const decision = adjudicated.get(record.id);
  // All records retain their original readback; existence-only claims do not become approval.
  const remaining = record.remainingWork ?? 'Need evidence-backed scope closeout before any lifecycle change.';
  return { id: record.id, idType: record.idType, originalAcceptance: record.originalAcceptance,
    intakeDisposition: record.priorDisposition, intakeEvidence: record.currentEvidence,
    disposition: decision?.status ?? (record.priorDisposition === 'container' ? 'container' : 'keep-open'),
    reason: decision?.reason ?? `Intake classification ${record.priorDisposition}; retained as a claim, not a new completion verdict. ${remaining}`,
    lifecycleChangeAuthorized: Boolean(decision),
    residuals: decision?.residuals ?? [{ finding: remaining, owner: record.ownerFiles ?? [], successor: record.batch ?? record.batchId ?? 'history audit follow-up', state: 'open', proof: null }],
    sourcePaths: record.sourcePaths, preservedArtifactHashes: record.artifacts };
}).sort((a, b) => a.id.localeCompare(b.id));
fs.writeFileSync(`${dir}/dispositions.json`, JSON.stringify({ taskId: auditId, date: '2026-11-17',
  note: '84 records accounted for; only 18 independently adjudicated candidates receive lifecycle changes. Original intake and evidence remain unchanged. Current/runtime/production acceptance remains open.',
  archiveDecisions: decisions, records }, null, 2) + '\n');
fs.writeFileSync(`${dir}/DISPOSITIONS.md`, `# 历史记录逐项处置\n\n日期：2026-11-17。84条对账；18条经独立原范围审查，其余保留明确开放/路由，不能从文件存在推断完成。\n\n| 记录 | 本轮处置 | 依据与剩余责任 |\n|---|---|---|\n` + records.map((r) => `| ${r.id} | ${r.disposition} | ${r.reason.replaceAll('|', '/').replaceAll('\n', ' ')} |`).join('\n') + '\n\n每项 finding/owner/successor/proof 详见 dispositions.json。最终候选门禁仍开放。\n');
console.log(JSON.stringify({ records: records.length, adjudicated: decisions.length }));
