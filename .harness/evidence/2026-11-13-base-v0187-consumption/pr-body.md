## 变更摘要

- 改动层级： business/* consumer of official Base assembly; platform/system owned by Base
- 改动模块： official v0.18.7 four-asset acquisition, standard staging rebuild and materialized shared-path delta (permissionroutes/auditpolicy/businessauth packages, business middleware, shared operational dashboard modules, inherited docs/gates), consumer lock/VERSION/anchors and this task packet/evidence
- 目标问题： consume immutable foundation release pantheon-base-v0.18.7 (tag -> 3203624e55438866edb951bd74e671eab38d2f16) carrying the formal-queue foundation contracts (Base PR #504) plus release-blocking CodeQL/Sonar remediations (#505/#506/#507)
- 预期影响： Ops runs the sealed provider-owned permission-route registry, repaired permission-cache revocation and shared operational UI against the exact official source; full local sensor set passed; hosted required checks and merge close the consumption

## Harness 链路

- Task ID： 2026-11-13-base-v0187-consumption
- Task Manifest： .harness/tasks/2026-11-13-base-v0187-consumption/manifest.json
- Evidence： .harness/evidence/2026-11-13-base-v0187-consumption/commands.json
- Verification evidence： .harness/evidence/2026-11-13-base-v0187-consumption/summary.md
- Review Artifact： .harness/evidence/2026-11-13-base-v0187-consumption/review.md
- OpenSpec change： none
- Trivial change： no
- Quality Profile： ci-workflow
- Ratchet Decision： registry-only
- GitHub Signal： repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-11-13-base-v0187-consumption
- task manifest: .harness/tasks/2026-11-13-base-v0187-consumption/manifest.json
- evidence: .harness/evidence/2026-11-13-base-v0187-consumption/commands.json
- boundaries: official four-asset acquisition, standard rebuild and materialized shared paths only; two declared manual alignments (scripts/frontmatter-check.mjs byte-sync from the immutable snapshot, system.ts generated-marker restoration to snapshot bytes) both surfaced and closed by review; no business runtime change
- backend response contract: unchanged
- backend DTO contract: unchanged
- permission contract: Base-owned provider-route seal and revocation repair consumed verbatim from snapshot; no consumer-side permission edits
- audit coverage: unchanged; oversized json_result storage error remains deferred Base-first (FR028), no waiver
- visual evidence: no UI bytes authored in Ops; shared operational dashboard components are snapshot bytes validated by 240 frontend unit tests and smoke suites
- inheritance contract: only four verified official assets entered the tree; standard locked-source assembly; no manual base/main copies
- base drift: 866 compared / 856 identical / 10 existing allowlisted; 164 business-overlay skipped; 2 derived migrations; no new drift
- Base/ops inheritance: Base owns platform/system; Ops owns business assets and consumer lock; standard assembly only
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = pantheon-base-v0.18.7 / 3203624e55438866edb951bd74e671eab38d2f16
- generic drift: handled in pantheon-base first (yes)
- pseudo-drift: excluded as line-count or formatting noise without contract break (yes)
- business-only: business asset ownership retained; official Base rebuild adapter is the landing side

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> Base owns the shared implementation (permission routes + seal, revocation repair, operational dashboard, Sonar/CodeQL remediations); Ops consumes the immutable v0.18.7 assembly through the unchanged assembler. The six inherited base task-local doc links (TENANT/LOWCODE) are generic drift routed back to Base. Menu/permission/i18n/audit behavior changes land only through the snapshot bytes above; business overlay hooks are unchanged.

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `npm run check:gorm-sql-literal`
- [x] `go test ./...` (readonly CGO full race: 67 packages ok / 0 failures / 0 races, D:\msys64\mingw64\bin + CGO_ENABLED=1)
- [x] `cd frontend && npm run lint && npm run type-check && npm run build` (+240 unit tests)
- [x] 业务 smoke 或明确 runtime gap (four standard generator smoke suites green on task-owned fresh runtime, migrations 32/0)
- [ ] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：Official acquisition first hit truncated cached downloads caused by full local disks; cache purged, assets re-downloaded and verified against published sidecars and hosted digests (tgz 1daa3e7f…, repo.tar 6149f9e9…) before locking. Independent two-round CODE/ARCH panel: round 1 REQUEST_CHANGES/BLOCKED (anchors, generated marker, provenance), fixed in 9b51415, round 2 APPROVE/APPROVE with architectural status CLEAR. Legacy local-gate noise disclosed in commands.json knownGaps; no gate or allowlist was changed.

## 审核留痕

- Copilot review： automatic-policy
- CodeQL 结果： no consumer-authored code; shared paths are snapshot bytes that passed Base-side CodeQL on 3203624e; hosted result per this PR
- GitHub checks 结果： pending on this PR; required checks must be green before merge
- Auto-merge： not-enabled
- Duplication Gate 结果： check:duplication exit 0 locally; hosted per PR checks
- 是否高风险改动： L2 inheritance sync; two-round independent CODE/ARCH approval including ARCH
- Residual risk / follow-up： S3 provider skip unwaived, maintenance freeze unsigned, FR028 deferred Base-side, separate QA002/host-import and Ops product-acceptance followups remain; .foundation/foundation-release.lock.json keeps its stale v0.18.2-era narrative (no tooling consumers; cleanup routed separately); rollback anchor is the v0.18.6 lock in git history

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
