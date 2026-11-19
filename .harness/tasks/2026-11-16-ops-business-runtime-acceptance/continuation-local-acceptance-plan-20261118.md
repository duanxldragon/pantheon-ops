# C05 本地 CI 准备计划（2026-11-18）

- Parent task: `2026-11-16-ops-business-runtime-acceptance`；父交付包：`2026-11-16-ops-pilot-delivery`。
- 层级：L2，Ops 业务 smoke 门禁；不改变平台层、系统域或 Base 规则。
- Minimal Complexity Rung: reuse。复用五个已有业务脚本、候选 SHA checkout、CI MySQL/Redis、健康检查、失败产物与 Quality Gates。
- Ownership: `business-overlay.json` 的 `replaceRoots` / `repositoryOverlayPaths` 声明 `.github`，并声明 `tests/scripts/quality-workflow.test.mjs`；父任务预期修改 workflow。

## In / Out

In：在 `smoke-sanity` 的平台/系统 smoke 后加入 CMDB、Deploy API、Deploy UI、Service、K8s 五个必过步骤入口；任何入口失败即停止并使既有汇总门禁失败。用实际 workflow shell 回归验证成功顺序及每个失败位置。

Out：不修改应用、fixture、共享脚本、Foundation lock、任务状态；不扩展到包含生成模块的完整 business umbrella；不执行 Go、浏览器、数据库、服务启停或发布。本地静态/脚本检查不代表 C05 运行态验收完成。

## 验证与停止条件

1. 先补回归，运行 `node --test tests/scripts/quality-workflow.test.mjs`，记录缺少业务门禁的 RED。
2. 加入最小 CI 步骤，再运行同一测试至 GREEN。
3. `node --check tests/scripts/quality-workflow.test.mjs`；检查 scoped diff、真实结果与文件 SHA-256。
4. 提交新增 handoff MD/JSON 给协调者评审，停止本子任务文件编辑。

实施者负责上述文件；评审者由协调者检查本地证据及候选集成。完整 CI hosted run、当前候选真实 MySQL/SSH/K8s、最小角色/双部门范围、并发/故障/回滚、桌面/窄屏渲染及最终维护者验收仍是显式 gap。现有质量门禁阈值和 PR scope 语义保持原契约。

## Linkage

- Task manifest: `.harness/tasks/2026-11-16-ops-business-runtime-acceptance/manifest.json`
- Task doc: `docs/harness/tasks/2026-11-16-ops-business-runtime-acceptance.task.md`
- Evidence: `.harness/evidence/2026-11-16-ops-business-runtime-acceptance/ci-preparation-handoff-20261118.md`
- Machine evidence: `.harness/evidence/2026-11-16-ops-business-runtime-acceptance/ci-preparation-handoff-20261118.json`
