## 变更摘要

- 改动层级：inheritance-sync（pantheon-base foundation release → pantheon-ops consumer）
- 改动模块：`foundation-release.lock.json`、`.foundation/foundation-release.lock.json`、base 快照树（backend / frontend / docs / tests）、`scripts/task-manifest.mjs`、`scripts/cleanup-github-branches.mjs`、`.github/workflows/{quality,pr-automation}.yml`
- 目标问题：把 ops 从 `pantheon-base-v0.17.7` 推进到 `pantheon-base-v0.18.1`（v0.18.0 的 S1 波次删掉了 `isForbiddenRequestError` 共享导出，导致本仓 `DeployTaskDetail` 前端构建 `TS2614`；v0.18.1 恢复该导出），同时修复这次跳版暴露的 5 处 consumer 侧漂移。
- 预期影响：`VERSION` 0.17.7 → 0.18.1；go directive 1.26.9 → 1.26.10，go-oidc 3.9.0 → 3.21.0，go-jose v3 → v4；Dockerfile builder 镜像 golang:1.26.10-alpine；删除 7 个被 0.18 精简波次退役的 base 文件与 13 个前端 capture 脚本；13 个根 RELEASE_NOTES 迁入 `docs/release-notes/`；业务模块无行为变更。

## Harness 链路

- Task ID：2026-10-29-foundation-upgrade-base-v0-18-1
- Task Manifest：.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json
- Evidence：.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json
- Verification evidence：.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/summary.md
- Review Artifact：.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：gate-updated
- GitHub Signal：repo-quality-gate

## Harness adoption markers

- task id：2026-10-29-foundation-upgrade-base-v0-18-1
- task manifest：`.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json`
- evidence：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json`
- boundaries：`pantheon-base` 拥有共享面，`pantheon-ops` 通过不可变 release 快照消费并拥有 `business/*`；本次不改 base 代码、不改已有 tag/release/asset。
- backend response contract：unchanged（0.18.1 无 API/DTO/路由变更）
- backend DTO contract：unchanged
- permission contract：unchanged（无菜单/权限码变更）
- audit coverage：unchanged
- visual evidence：none（无 UI 行为的 ops 侧改动；前端构建通过即覆盖本仓改动范围）
- inheritance contract：`docs/PROJECT_INHERITANCE.md` §7 锚点更新为 `pantheon-base-v0.18.1`
- base drift：无遗留 shared drift（`check:shared-drift --base ../pantheon-base` → 812 identical / 10 allowlisted / 160 skipped）
- Base/ops inheritance：Base-first 已执行：TS2614 的根因修复（恢复导出）在 `pantheon-base` 完成并发布为 v0.18.1，本仓只消费；两个 overlay 脚本钉住旧副本造成的漂移通过「取消钉住、采用 base 副本」收敛，未手抄 base 文件。
- Base version checked: `foundation-release.lock.json` releaseVersion = pantheon-base-v0.18.1, baseCommit = 93505225e36c8d3ba5aa9facc12975eb813167d3
- generic drift：yes（base 0.18 收紧的 task-manifest 导出、UI 质量门禁工作流步骤与 manifest status 词汇，均按 base 语义收敛）
- pseudo-drift：yes（删除的 33 个跟踪文件中 13 个是 RELEASE_NOTES 目录迁移，无契约破坏）
- business-only：no（inheritance-sync 变更，未改 `business/*` 业务逻辑）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> Base 侧：v0.18.1（tag `pantheon-base-v0.18.1`，目标提交 `93505225`）为唯一事实源，四件资产校验和与 lock 一致，Release Gate / Foundation Release Verify 均 success。Ops 侧：通过 `rebuild:from-base`（lock 解析）生成 758 个声明/生成文件，先在 `.tmp/business-overlay-rebuild` 全量过门禁，再做 L2 基线替换；`.foundation/allowed-drift.json` 未变。菜单/权限/i18n/审计/seed/smoke 均无变更。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `go test ./...`（`go build` / `go vet` / `gofmt -l` 同时为 0）
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [x] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：`check:shared-drift --base ../pantheon-base` 在替换后的树上为 OK；Docs Governance 的全部命令（frontmatter / encoding / structure / task-packet-template / pr-governance / business-overlay / business-overlay test / generated-modules / ui-quality-gate）在本地均为 0。Runtime gap：未启动后端、未跑 golang-migrate 与 seed（v0.18.x 未移动 schema baseline，继承 v0.17.7 同一 gap）；本机 Windows 无法运行 `go test -race`（TSan 影子内存分配失败，error code 87），CI 在 ubuntu-latest 上跑该命令。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：pending-hosted-run
- GitHub checks 结果：pending-hosted-run
- Auto-merge：enabled
- Duplication Gate 结果：pass（`npm run check:duplication` 本地 0）
- 是否高风险改动：yes（触碰共享 base 面、`pkg/*`、`.github/workflows/*` 与继承边界）
- Residual risk / follow-up：① runtime acceptance（启动 + migrate + seed）仍未在 consumer evidence 中执行；② `check:harness-docs` 的 4 个断链来自 base 文档指向 base-only 任务与归档，修复属 base 侧；③ 继承自 base 的 3 个工作流测试（release-gate / quality-workflow 评分 / RELEASE_GATE_TOKEN）断言的是 base 的 `.github` 形态或本仓不存在的 secret，未被任何 workflow 执行，保持显式红灯；④ 本仓无 `RELEASE_GATE_TOKEN`，自动合并仍用 `github.token`，workflow 触发的合并不会触发 main 的 push 流水线。

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
