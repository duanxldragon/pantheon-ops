# Prompt G1 — 自动合并矛盾收口（pr-auto-merge-push-trigger + pr-auto-merge-token）

> 派发模型：按 DISPATCH 契约交由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。

## 问题（已对账核实，非新扫描）

`.github/workflows/pr-automation.yml` 的 "Enable squash auto-merge" step 用 `GH_TOKEN: ${{ github.token }}`（确定性实测），但：

- `tests/scripts/pr-automation-workflow.test.mjs` 与 `tests/scripts/ci-workflow-dispatch.test.mjs` 断言要求 `secrets.RELEASE_GATE_TOKEN` + 不存在的 `.github/workflows/ci.yml`。
- 这两个历史记录 `2026-08-12-pr-auto-merge-push-trigger`（要求 RELEASE_GATE_TOKEN + fail-closed + ci.yml 精确提交触发器）与 `2026-08-12-pr-auto-merge-token`（要求 github.token）相互矛盾。
  当前树内 ctest 与 workflow 自洽性被破坏，CI 可能红灯。这不是"旧记录待办"，是真实缺实现/缺验证。

## 精确文件所有权（本批独占，串行——G2 后续才补 2026-08-03 证据，不得与 G2 并行碰此文件）

- `.github/workflows/pr-automation.yml`（auto-merge step 唯一改动点）
- `.github/workflows/ci.yml`（若最终方案需要精确提交手动触发器，则新建）
- `tests/scripts/pr-automation-workflow.test.mjs`
- `tests/scripts/ci-workflow-dispatch.test.mjs`
- `.harness/tasks/2026-08-12-pr-auto-merge-push-trigger/{manifest.json,plan.md}`
- `.harness/evidence/2026-08-12-pr-auto-merge-push-trigger/{commands.json,summary.md,review.md}`
- 同上 `.harness/{tasks,evidence}/2026-08-12-pr-auto-merge-token/`

## Decentral decision（请先定方案，二选一，并在两记录 manifest 中显式记录决策与理由，禁止同时保留矛盾断言）

- 方案 A：采用 `secrets.RELEASE_GATE_TOKEN`，`if: secrets.RELEASE_GATE_TOKEN != ''` 前置，无 token 则 `fail closed`（job 以失败结束，不静默用 github.token）；并新建 `ci.yml` 提供 `workflow_dispatch` + `push` 的精确提交手动触发；把 `pr-auto-merge-token` 记录标为 superseded-by-push-trigger（或同步改其断言为同一契约）。
- 方案 B：保留 `github.token`（默认权限、无需额外 secret），去掉 fail-closed 到 RELEASE_GATE_TOKEN 的要求，把 `pr-auto-merge-push-trigger` 记录标为 superseded-by-token；并把 `ci.yml` 需求从推送触发改作可选用 `workflow_dispatch` 的手动恢复入口（仍建议补）。
- 无论 A/B：明确 `fail-on-error`（`if gh pr merge ...; then ...; else echo 'Unable to enable auto-merge' >&2; exit 1; fi`），并把两记录的输出统一到同一契约，使 workflow 断言通过。

## 约束

- 最小复杂度：只改自动合并 step/触发器/两测试断言，不重构整个 workflow。
- Base-first：`github.token`/`RELEASE_GATE_TOKEN` 属 repo CI 域，不改 pantheon-base。
- 不触碰 `dependabot-pr-governance`（2026-08-03）相关测试或 quality.yml（留给 G2，本批不改）。
- 不提交、不 push、不发布；产出本地 + evidence。

## 真实验收（Codex 会核）

- 选定方案在两 manifest 记录中显式写明；另一记录被明确 superseded（带指向合并 decision 的链接）。
- `node --test tests/scripts/pr-automation-workflow.test.mjs tests/scripts/ci-workflow-dispatch.test.mjs` 通过；`npm run check:docs-frontmatter`、task-packet 检查通过。
- 未在 CI 之外引入新 secret 依赖或绕过门禁的快捷路径。
- 本仓库 Claude 不出 Edit/Write 到这些文件（系统域由 Codex 执行）。
