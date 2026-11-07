## 变更摘要

- 改动层级： inheritance-sync; Base平台/系统域正式release -> Ops business consumer
- 改动模块： official v0.18.5 lock, shared fixture and governance docs, normalized Go graph, evidence
- 目标问题： 发布夹具修正后消费者需从GitHub正式资产锁定重建，完成无overlay的全量验证
- 预期影响： 正式消费1da0834b，保留既有业务修复和全部原范围；源码与真实验证树一致

## Harness 链路

- Task ID： 2026-11-05-base-v0183-consumption
- Task Manifest： .harness/tasks/2026-11-05-base-v0183-consumption/manifest.json
- Evidence： .harness/evidence/2026-11-05-base-v0183-consumption/commands.json
- Verification evidence： .harness/evidence/2026-11-05-base-v0183-consumption/summary.md
- Review Artifact： .harness/evidence/2026-11-05-base-v0183-consumption/review.md
- OpenSpec change： none
- Trivial change： no
- Quality Profile： generator
- Ratchet Decision： registry-only
- GitHub Signal： runtime-evidence-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-11-05-base-v0183-consumption
- task manifest: .harness/tasks/2026-11-05-base-v0183-consumption/manifest.json
- evidence: .harness/evidence/2026-11-05-base-v0183-consumption/commands.json
- boundaries: Base-first official release; Ops owns composition and business overlay; protected source unchanged after validation
- backend response contract: unchanged
- backend DTO contract: unchanged
- permission contract: unchanged, no SSRF/auth relaxation
- audit coverage: historical failure evidence and current tests retained; runtime table gaps explicit
- visual evidence: no new UI behavior or polish claim; original smoke pages rendered, four exact standard suites passed
- inheritance contract: official four GitHub assets verified, exact immutable tag/archive provenance, 847 declared files
- base drift: 826 compared /816 identical /10 allowed; no new exclusion
- Base/ops inheritance: Base PR496 immutable1da0834b and Ops PR176 module normalization already merged; this PR consumes official corrective release
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = pantheon-base-v0.18.5 /1da0834bdce636fd2fc1063290e3f32339e2d27a
- generic drift: handled in `pantheon-base` first (yes)
- pseudo-drift: excluded as line-count or formatting noise without contract break (no)
- business-only: business source unchanged; source fixture and shared governance originate only from official Base archive

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 如果跨层，请说明 Base 与 Ops 各自职责、Base commit、overlay 变化，以及是否影响菜单/权限/i18n/审计。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `npm run check:gorm-sql-literal`
- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：完整 Windows CGO readonly race `go test -timeout=30m -buildvcs=false -mod=readonly -race -json -p 2 -count=1 ./...`：2170pass/0fail/1真实S3skip/64包。FE lint/type-check/build/218unit，59脚本，fresh migration32clean，四个标准smoke全部通过。

源码1031文件与真实验证树一致；模块/fixture/race无源码overlay。847report所有hash验证。独立报告/同snapshot见resume10-consumer-approval.json。额外未准备的目录smoke3fail/2pass保持历史，严格原入口与QA setup通过；QA-002及额外hostimport缺表保持显式gap。原消费task保持in-progress直到本PR CI及merge。

## 审核留痕

- Copilot review：automatic-policy
- CodeQL 结果： current PR hosted check pending; immutable Base security passed
- GitHub checks 结果： pending for this PR; local proof is not hosted CI
- Auto-merge： not-enabled
- Duplication Gate 结果： current hosted gate pending, thresholds unchanged
- 是否高风险改动： yes; two independent CODE/ARCH approvals on exact snapshot, both permit actual evidence updates
- Residual risk / follow-up： S3 real skip/no waiver; unsigned freeze; QA-002 missing deploy attempt table and extra host-import table in isolated versioned-migration runtime; original five Base closures await separate review; historical69packets/568 optional errors unchanged

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
