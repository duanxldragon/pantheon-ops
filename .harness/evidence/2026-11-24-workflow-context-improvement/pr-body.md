## 变更摘要

- 改动层级: business repository governance / clean Base rebuild adapter
- 改动模块: AI entrypoints, task recovery, checker wiring and pinned method CI
- 目标问题: Stale context and lexical script references can hide missing execution or policy drift.
- 预期影响: Find current tasks with less repeated reading and reject provenance/wiring counterexamples without weakening existing gates.

## Harness 链路

- Task ID: 2026-11-24-workflow-context-improvement
- Task Manifest: .harness/tasks/2026-11-24-workflow-context-improvement/manifest.json
- Evidence: .harness/evidence/2026-11-24-workflow-context-improvement/commands.json
- Verification evidence: .harness/evidence/2026-11-24-workflow-context-improvement/summary.md
- Review Artifact: .harness/evidence/2026-11-24-workflow-context-improvement/review.md
- OpenSpec change: none
- Trivial change: no
- Quality Profile: ci-workflow
- Ratchet Decision: sensor-added
- GitHub Signal: method-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-11-24-workflow-context-improvement
- task manifest: .harness/tasks/2026-11-24-workflow-context-improvement/manifest.json
- evidence: .harness/evidence/2026-11-24-workflow-context-improvement/commands.json
- boundaries: Method-only; no product source changes. Shared sensors in Base, portable helper in Harness, declared local overlays in Ops.
- backend response contract: none
- backend DTO contract: none
- permission contract: none
- audit coverage: none
- visual evidence: none
- inheritance contract: docs/PROJECT_INHERITANCE.md
- base drift: No shared product file copied from Base HEAD
- Base/ops inheritance: Locked Base release 0.18.9 unchanged; Ops uses owned workflow/adapter/assets
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = pantheon-base-v0.18.9 / 6316f9b7189c86542424145fa49344ad737e952b
- generic drift: yes; shared sensors land in Base; only declared repository overlays change in Ops
- pseudo-drift: yes; official locked snapshot drift was checked by contract
- business-only: owned workflow, method receipt, rebuild root command assembly and task documents

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 如果跨层，请说明 Base 与 Ops 各自职责、Base commit、overlay 变化，以及是否影响菜单/权限/i18n/审计。

## 验证记录

- [ ] `npm run check:business-overlay && npm run test:business-overlay`
- [ ] `npm run check:gorm-sql-literal`
- [ ] `go test ./...`
- [ ] `cd frontend && npm run lint && npm run type-check && npm run build`
- [ ] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：本次只提交机制改动，其他工具的 Ops 功能开发不在此 PR。隔离分支基于最新 main；具体命令与真实退出见 evidence。

Local validation: Harness 124/124, Base 19/19, Ops workflow/rebuild 29/29; read-only consumption, current indexes, docs/encoding, locked snapshot rebuild, overlay and 866-file shared drift passed. No UI/runtime source changed.

## 审核留痕

- Copilot review: unavailable; native independent code-reviewer and architect evidence recorded
- CodeQL 结果: Hosted scan pending; no product code change
- GitHub checks 结果: Pending hosted checks; merge only after required CI passes
- Auto-merge: not-enabled; authorized squash merge after CI
- Duplication Gate 结果: Pending hosted report
- 是否高风险改动: yes: CI workflow/provenance, independently reviewed
- Residual risk / follow-up: Formal release/foundation adoption and product acceptance remain separate; tooling diagnostic limitations recorded.

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [ ] 通用平台/系统域问题已在 Base-first 边界处理
- [ ] 前端新增展示文案已使用 i18n
- [ ] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [ ] UI 改动已有渲染证据或明确 runtime gap
- [ ] 已确认不会泄露敏感配置、账号密码或 Token

Reviewed method source: Harness PR #13 (`51fc783dc793b75ae91bdb936ba1418e2e95a354`); PR/main checks passed. Consumer CI must pass before squash merge.
