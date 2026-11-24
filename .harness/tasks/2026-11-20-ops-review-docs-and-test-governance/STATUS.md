# 能力文档、测试资产与治理对账

- Status: completed（维护者最终视觉/功能验收 Human Gate 仍开放）
- Updated: 2026-11-24
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc；执行候选 b1086437
- Owner: 业务文档与验证负责人
- Priority: P2
- Activation: core
- Dependencies: T01, T03

## Implemented

- `VERSION` 0.18.7 → 0.18.9，与 `foundation-release.lock.json`（pantheon-base-v0.18.9）和 Base 快照 `6316f9b7:VERSION` 对齐。
- `docs/PROJECT_INHERITANCE.md` / `.en.md` 新增「版本字段口径」：区分 lock、`VERSION`、`SHELL_VERSION.json`、`.harness/method-consumption.json`、`frontend/package.json`、`CHANGELOG`/`release-notes`。
- 修复 `deploy-detail-read.test.tsx`、`metrics-query-state.test.tsx` 的 `src/components` mock，补 `buildStandardPagination` 导出（此前 15 个用例渲染抛错）。
- `quality.yml` Frontend Contract 作业新增 `Unit tests` 步骤（`npm run test:unit`）。
- 退役 `e2e-tests/`（v1.2.0、2026-08-28；19 个端点全部使用旧路由前缀；不在 CI/脚本中），同步移除 `business-overlay.json` 的 `repositoryOverlayPaths` 条目。
- NEW-02/NEW-03/Helm `registered` 口径与失效链接在开工核对时已由既有文档收敛，无需再改。

## Verification

- `npm run test:unit`：54 文件 / 430 用例全绿（修复前 2 文件 / 15 用例失败）。
- `npm run test:scripts`：242/242，含 workflow baseline 校验。
- 真实 MySQL（隔离验收库 `ops_acceptance_formal_20261121`）：租户迁移 2 用例、`TestReadableNamespacesFollowsClusterOwnership`、固定模板注入/路径/卸载根拒绝、背压与并发 scope 用例全部 PASS。
- `check-doc-links`（仓库根）0 findings；`check-doc-frontmatter` 0 error；`check-business-overlay` OK。
- 既有基线：`check-task-packet` 657 error / 118 warning（115 个历史 packet）；`check-encoding` 2 处归档 U+FFFD。本批零新增。
- 证据：`../evidence/2026-11-20-ops-review-docs-and-test-governance/`。

## Known Gaps

1. 维护者最终验收 Human Gate 开放（本批无 UI 改动，不需要渲染证据）。
2. `deadcode` 未运行：未声明为依赖，运行会迫使 go.mod 变更，因此未删除任何候选。
3. `.business-overlay-report.json` 仍列出退役 e2e 文件，属重建时点回执，无门禁读取，下次重建自动刷新。
4. 248 个中文文档无英文版属既有基线，不在本包范围。
