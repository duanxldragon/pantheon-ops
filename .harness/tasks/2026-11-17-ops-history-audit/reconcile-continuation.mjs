import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { resolveArchiveAwarePath } from '../../../scripts/task-manifest.mjs';

const root = fs.realpathSync(process.cwd());
assert.equal(path.basename(root), 'pantheon-ops');
const audit = '.harness/evidence/2026-11-17-ops-history-audit';
const sha = (bytes) => createHash('sha256').update(bytes).digest('hex');
const read = (file) => fs.readFileSync(path.join(root, file));
const json = (file) => JSON.parse(read(file));
const write = (file, value) => fs.writeFileSync(path.join(root, file), value);
const frozen = json(`${audit}/dispositions.json`);
const groups = json('.harness/tasks/2026-11-17-ops-history-audit/continuation-groups.json');
const byId = new Map(frozen.records.map((r) => [r.id, r]));
assert.equal(byId.size, 84);
const recommendations = Object.entries(groups).flatMap(([disposition, ids]) => ids.map((id) => ({ id, disposition })));
assert.equal(recommendations.length, 66);
assert.equal(new Set(recommendations.map((r) => r.id)).size, 66);
const prior = new Map(frozen.archiveDecisions.map((r) => [r.id, r]));
const second = '2026-06-18-pr-branch-cleanup-automation';
assert.equal(json(`${audit}/second-batch-result.json`).status, 'passed');
prior.set(second, { id: second, status: 'abandoned', reason: 'Old closed-PR cleanup coupling superseded; second-batch independent review and original hashes retained.', residuals: [], proof: `${audit}/second-batch-result.json` });
assert.equal(prior.size, 19);
assert.equal([...prior.keys()].filter((id) => byId.has(id)).length, 18);
for (const row of recommendations) {
  assert.ok(byId.has(row.id), `unknown original id ${row.id}`);
  assert.ok(!prior.has(row.id), `already adjudicated id ${row.id}`);
}
assert.deepEqual([...byId.keys()].filter((id) => !prior.has(id)).sort(), recommendations.map((r) => r.id).sort());
const current = new Map(recommendations.map((r) => [r.id, r.disposition]));
const reviewRefs = [`${audit}/continuation-scope-code-review.md`, `${audit}/continuation-scope-architecture.md`];
for (const file of reviewRefs) assert.ok(fs.existsSync(path.join(root, file)), `missing independent review ${file}`);
const pairCandidates = new Set(['2026-08-07-foundation-upgrade-base-v0-10-3', '2026-10-12-foundation-upgrade-base-v0-16-1', 'cmdb-host-import-export', '2026-08-08-foundation-upgrade-base-v0-10-5', '2026-08-12-foundation-v0-10-14-clean-consumer']);
const overrides = {
  '2026-06-20-business-closure-permission-scope': ['Ops deploy/CMDB/bizscope', '2026-11-17-ops-history-audit', 'Dept positive/negative write regressions pass; original exhaustive endpoint scope and Self acceptance remain open. BizScope list subrequirement has a separate successor-design disposition.'],
  '2026-06-21-business-deploy-state-guard': ['Ops deploy', '2026-11-16-ops-business-runtime-acceptance', 'Map original guard and declared request-smoke acceptance to current evidence; do not infer it solely from broader main-flow coverage.'],
  '2026-06-21-business-deploy-task-delete': ['Ops deploy', '2026-11-16-ops-business-runtime-acceptance', 'Original contracts-package, generated-i18n, request-smoke/role evidence need exact mapping; runtime smoke must follow current preview intent contract.'],
  '2026-07-17-cmdb-ssh-config-externalize': ['Ops CMDB; Base public settings', '2026-11-17-business-runtime-settings then formal Ops consumption', 'Business parsing/persistence race passed. Base draft supplier and seed are reviewed/tested but unpublished; official downstream wiring, real context/tenant and live SSH remain open.'],
  '2026-08-12-pr-auto-merge-push-trigger': ['Ops governance/maintainer', '2026-11-17-ops-auto-merge-reconcile', 'Local token/dispatch tests do not establish the hosted post-merge exact-candidate signal or final workflow approval.'],
  '2026-09-10-tenant-verification-and-gray': ['Unresolved historical owner; Base tenant domain', 'Recover owning packet', 'Images alone do not establish scope/candidate/provenance or production/gray acceptance.'],
  '2026-11-07-ops-planned-task-closeout': ['Ops historical packaging', '2026-11-17-ops-history-audit', 'Materialized records corrected with original backups; historical review/red-stage failures retained. Packaging review does not establish unknown original project scope or final acceptance.'],
  '2026-11-14-ops-latest-foundation-and-plans': ['Ops historical packaging', '07d53988 official v0.18.9 consumption', 'v0.18.8 baseline superseded by the later 11/16 official consumer. Original review/provenance and business-plan acceptance remain open.'],
  'argo-adoption-20261110': ['Mixed ArgoCD/tooling record; owner unresolved', 'Recover original task identity', 'final3-race.log includes real ArgoCD unit/race tests; other logs include sealed-execution helpers. Neither establishes live ArgoCD acceptance; no owning packet.'],
  'argo-adoption-v2-20261110': ['Sealed-execution tooling; owner unresolved', 'Recover original task identity', 'RootRunnerTests pass but original scope/review/manifest are missing; no live business acceptance inferred.'],
  'argo-adoption-v3-20261110': ['Sealed-execution tooling; owner unresolved', 'Recover original task identity', 'RootRunnerTests pass but original scope/review/manifest are missing; no live business acceptance inferred.'],
  'cmdb-host-import-export': ['Ops CMDB host', '2026-11-16-ops-business-runtime-acceptance', 'Original manifest permits an explicit visual/runtime gap and findings-first self-review. Later CMDB 12/12 smoke adds CSV capability and phone evidence; current product/role/visual acceptance remains separate.'],
  'k8s-recovery': ['Ops K8s integration', 'C04/C05/D1', 'Only original local registration/dependency/build recovery completed. No original manifest: record-only disposition, no fabricated paired archive; real cluster and current product/race acceptance remain open.'],
  'D1-external-integration-acceptance': ['Ops integration/endpoint owners/maintainer', '2026-11-16-ops-extensions-live-acceptance', 'Real enabled-provider matrix, permission/DataScope/audit/cleanup/idempotency acceptance needs scoped endpoints/credentials; mocks cannot replace it.'],
  'D2-outbound-ssrf-protection': ['Base network policy; Ops consumers; maintainer', 'Base policy/release and deployed Ops path acceptance', 'Two negative unit regressions do not establish all outbound surfaces or private-network/allowlist/redirect/rebinding/timeout policy and audit.'],
  'D3-base-first-consumer-re-review': ['Ops inheritance', 'Current official v0.18.9 clean rebuild', 'Old report claims completion but defers detailed comparison; fresh same-candidate lock/report/clean-rebuild, boundaries and generated-system ownership are still required.'],
  'D4-G2-production-backup-restore': ['DBA/maintainer/production backup owner', 'Production G2 gate', 'Production-backup restore drill, measured RPO/RTO and DBA/maintainer sign-off absent; local rehearsal does not close it.'],
  'D4-G3-production-sizing-migration-window': ['DBA/maintainer/production DB owner', 'Production G3 gate', 'Approved production/restored-target sizing, migration-window assumptions and required scale rehearsal/signatures remain open.'],
  'D5-harness-generated-artifact-governance': ['Ops governance; Base contracts; maintainer', '2026-11-17-ops-history-audit', 'Current full gate/debt classification, generated ownership and bounded historical drift still required; delta-only results do not mean whole-repo green.'],
  'O1-unified-context-target-selector': ['Ops business context; Base selector contract', 'C02/C03 and shared capability consumption', 'Deploy-only selector does not establish shared context/provider adapter, scope invalidation, TOCTOU, 10k bound and keyboard/mobile acceptance.'],
  'O2-deployment-execution-workbench': ['Ops deploy', 'C02/C05', 'Original running/failed/stale/forbidden, sequence reconnect, partial-failure, version-aware actions and visual acceptance remain open.'],
  'O3-change-preview-and-risk': ['Ops deploy/K8s/ArgoCD; Base diff contract', 'C02/E02 plus original cross-domain preview', 'K8s/ArgoCD version/resource diff, immutable intent/conflict, redaction and risk/mobile/keyboard evidence are not fully covered by partial successors.'],
  'O4-observability-query-workbench': ['Ops observability', 'C05/E04/D1', 'Migration/recovery/fixture tests do not establish real providers, cancel/live-tail/privacy/history/isolation and desktop/mobile state matrices.'],
  'O5-task-and-pod-log-experience': ['Ops deploy/K8s log adapters; Base viewer', 'C05/E04/D1', 'Real pod/deploy stream, sequence reconnect/download authorization/audit/redaction/bounds, 10k behavior and mobile fullscreen remain unproved.'],
  'O6-business-dashboard-widgets': ['Ops widget registration; Base registry', 'C05 business dashboard acceptance', 'Mixed-permission no-fetch/no-render, freshness/error isolation/query bounds/deep links/mobile order lack original-scope evidence.'],
  'O7-cmdb-relationship-impact-discovery': ['Ops CMDB relationship providers', 'Separate L2 after five maturity conditions', 'Deferred until all provider/DataScope/fixture prerequisites pass; does not block O1-O6.']
};
const records = [...byId.values()].map((original) => {
  const disposition = prior.get(original.id)?.status ?? current.get(original.id);
  const specific = overrides[original.id];
  const owner = specific?.[0] ?? (disposition === 'upstream-owned' ? 'pantheon-base (shared producer); pantheon-harness where stated by original card' : original.residuals?.[0]?.owner ?? 'Ops historical scope');
  const successor = specific?.[1] ?? (disposition === 'upstream-owned' ? 'Owning Base/Harness historical task; current official foundation line' : disposition === 'abandoned' ? 'Official locked-snapshot release/consumer or explicit later security/governance mechanism' : original.residuals?.[0]?.successor ?? 'Current Ops business/runtime acceptance');
  const gap = specific?.[2] ?? (disposition === 'upstream-owned' ? 'Routing only. No upstream completion asserted and no Ops packet fabricated.' : disposition === 'abandoned' ? 'Stop repeating the superseded exact target. Missing historical evidence and acceptance failures remain facts; no retrospective success.' : disposition === 'container' ? 'Children control epic acceptance; no overall completion from implemented children.' : disposition === 'completed' ? 'Only original independently reviewed local scope completed; hosted/current-product/final-maintainer acceptance remains separate.' : original.residuals?.map((r) => r.finding).join('; ') || 'Original completion evidence or acceptance not established.');
  const directProof = (original.sourcePaths ?? []).filter((file) => !/[{*]/.test(file) && fs.existsSync(path.join(root, file)));
  const proof = [...new Set([...directProof, ...reviewRefs, ...(original.id === 'cmdb-host-import-export' ? ['.harness/evidence/2026-11-07-ops-planned-task-closeout/cmdb/smoke-final.log'] : []), ...(original.id === 'argo-adoption-20261110' ? ['.harness/evidence/argo-adoption-20261110/final3-race.log'] : [])])];
  return { ...original, disposition, owner, successor, remainingAcceptance: gap, independentReview: reviewRefs, proofPaths: proof, historicalAcceptancePreserved: true, canPairArchive: pairCandidates.has(original.id), lifecycleChangeAuthorized: pairCandidates.has(original.id), originalInventoryMember: true };
});
for (const decision of prior.values()) if (!byId.has(decision.id)) records.push({ id: decision.id, originalInventoryMember: false, originalAcceptanceSource: `.harness/archive/2026-11/tasks/${decision.id}/manifest.json`, disposition: decision.status, owner: 'Ops original isolated F-stage', successor: 'C05/C06/E04 and production/maintainer gates', remainingAcceptance: decision.residuals, priorDecision: decision, proofPaths: [`${audit}/dispositions.json`, `${audit}/archive-result.json`], historicalAcceptancePreserved: true });
records.sort((a, b) => a.id.localeCompare(b.id));
assert.equal(records.length, 85);
assert.equal(new Set(records.map((r) => r.id)).size, 85);
const counts = Object.fromEntries(Object.keys(groups).map((key) => [key, records.filter((r) => r.disposition === key).length]));
assert.deepEqual(counts, { 'upstream-owned': 26, completed: 20, abandoned: 10, 'keep-open': 26, container: 2, deferred: 1 });
const snapshot = { taskId: '2026-11-17-ops-history-audit', recordedAt: new Date().toISOString(), inputSHA256: sha(read(`${audit}/dispositions.json`)), originalInventory: { total: 84, priorDisposed: 18, remainingReviewed: 66 }, additionalSubject: '2026-08-19-ops-v1-backlog-closeout', total: 85, counts, basis: 'Two independent lanes reviewed all 66 outstanding original subjects. Frozen first/second batches preserved. Classification and archival are separate from current product completion.', pairArchiveCandidates: [...pairCandidates], records };
write(`${audit}/continuation-dispositions.json`, JSON.stringify(snapshot, null, 2) + '\n');
const table = ['# 历史清查当前对账（新快照）', '', '2026-11-17：原清单84条（前轮处置18、余66全部复审）＋额外8/19组＝85对象。前轮19组原结论不改；本轮新增4项原范围完成、7项按替代停止重复实施。当前20 completed、10 abandoned、26 upstream-owned、26 keep-open、2 container、1 deferred。', '', '本表的关闭仅指原始范围。生产、外部端点、正式Base发布/消费、当前候选质量门禁和最终功能/视觉验收仍按各自责任开放。配对归档结果以 continuation-archive-result.json 为准。', '', '| 记录 | 处置 | Owner / 后继 / 剩余验收 |', '|---|---|---|', ...records.map((r) => `| ${r.id} | ${r.disposition} | ${r.owner} / ${r.successor} / ${typeof r.remainingAcceptance === 'string' ? r.remainingAcceptance.replaceAll('|', '/') : '原冻结批次 residuals 逐项保留'} |`)];
write(`${audit}/CONTINUATION_DISPOSITIONS.md`, table.join('\n') + '\n');
console.log(JSON.stringify({ total: snapshot.total, counts, pairArchiveCandidates: snapshot.pairArchiveCandidates }));
