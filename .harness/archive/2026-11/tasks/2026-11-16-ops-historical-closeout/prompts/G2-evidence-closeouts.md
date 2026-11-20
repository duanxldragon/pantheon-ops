# Prompt G2 — 缺陷-验证补证据（missing-verification 闭卷）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。**与 G1 串行**（都可能在 `.github/workflows/pr-automation.yml` 口径上有关联；若 G1 未完成，先只做本批不冲突部分）。

## 目标记录（仅补关闭链/证据，不新增业务功能）

1. `2026-08-03-dependabot-pr-governance`：树内代码/测试已齐（`pr-automation.yml` dependabot 豁免、`quality.yml` dependabot 跳过 governance、`pr-automation-workflow.test.mjs` 断言），但 declarature 的 `.harness/tasks/2026-08-03-dependabot-pr-governance/manifest.json` 与 `.harness/evidence/.../{commands.json,review.md}` **从未创建**。补齐，并把 hosted 上"Dependabot PR 经 governance 前置并自动合并"的验证结果如实记录（若有 hosted 证据链）。若无法取得 hosted 观察，则在 review 中如实写明"本地一致、hosted 未复核"，**不得谎称 hosted 已通过**。
2. `2026-06-20-business-closure-permission-scope`：代码已分层（deploy/bizscope 的页面级 `PagePerm` + 动作 F-perm、数据域 `WithDataScope`），但对该 ID 无 manifest/证据/测试闭环。做**逐端点的数据域审计**（deploy task list/detail/update/cancel/result-writeback；bizscope detail/host-bind），补权限映射测试断言，写证据定案：要么 verified（全闭环），要么列真实缺口。

## 精确文件所有权

- `.harness/{tasks,evidence}/2026-08-03-dependabot-pr-governance/**`
- `backend/modules/business/deploy/*`、`backend/modules/business/bizscope/*`（仅审计/新增缺失测试；不重写现有逻辑）
- `.harness/{tasks,evidence}/2026-06-20-business-closure-permission-scope/**`
- 如发现 `/packages/:id` 等授权缺口，输入与 P1 合并（本批不重复处理授权路由）。

## 约束

- 最小复杂度；审计发现真实缺口才小改；不鉴权削弱。
- 不触碰 `.github/workflows/pr-automation.yml`/`tests/scripts/pr-automation-workflow.test.mjs`（G1 独占）；本批 quality 相关只读。
- 不提交/push/发布；commands.json 只记真实运行。

## 真实验收

- 两记录各一个明确 disposition（verified 或列出剩余必做项）；evidence 齐全、无悬空。
- `go test ./backend/modules/business/...` 通过；设计/权限记录与实现一致。
