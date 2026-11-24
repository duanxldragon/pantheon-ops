# 能力文档、测试资产与治理对账结果

Task: 2026-11-20-ops-review-docs-and-test-governance；Status: completed（维护者最终验收 Human Gate 仍开放）；Review: [review.md](review.md)。

## 来源发现核对（2026-11-20 审查）

| 来源条目 | 2026-11-24 状态 | 处置 |
| --- | --- | --- |
| NEW-02 资产关系/Pod→Node→VM 在代码中不存在 | 已在 `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md:40` 收窄定位，登记为独立后续能力 | 复核，无需再改 |
| NEW-03 Helm 只有登记记录 | `README.md:11` 与设计文档已明确 `registered` 口径、历史原始状态保留在 `storedStatus`、删除登记不卸载资源 | 复核，无需再改 |
| NEW-04 业务单测未接入 CI、e2e 疑似过期 | 业务单测确未接入 CI 且 2 个文件已随共享组件 API 变更腐化；e2e 全部 19 个端点使用 2026-08 的旧前缀 | **本批修复（见下）** |
| 文档与版本：VERSION/lock/前端/CHANGELOG 口径不一 | `VERSION` 0.18.7 与 lock `pantheon-base-v0.18.9` 不一致（继承漂移）；其余字段是不同对象的独立版本 | **本批修复并文档化版本字段角色** |
| 文档与版本：7 条失效链接 | 已不存在；仓库根运行 `check-doc-links` 为 0 findings | 复核，无需再改 |
| 冗余：`normalizeListLimit` 三份重复 | 当前树 0 处出现 | 复核，已由先前变更合并 |
| 冗余：deadcode 系统扫描 | 工具未声明为依赖，运行会迫使 go.mod 变更 | 保留 gap，不删除候选 |
| 有意双份生成迁移 | 重建器维护的装配副本，不能当重复删除 | 保持 |

## 本批改动

1. **`VERSION` 0.18.7 → 0.18.9**：与 `foundation-release.lock.json` 的 `pantheon-base-v0.18.9` 及 Base 快照 `6316f9b7:VERSION` 对齐，消除继承漂移。
2. **版本字段口径写入继承合同**（`docs/PROJECT_INHERITANCE.md` + `.en.md`）：区分 `foundation-release.lock.json`、`VERSION`、`SHELL_VERSION.json`、`.harness/method-consumption.json`、`frontend/package.json`、`CHANGELOG.md`/`docs/release-notes/` 六个来源。
3. **修复两个腐化的业务单测**（`deploy-detail-read.test.tsx`、`metrics-query-state.test.tsx`）：共享 `src/components` 新增 `buildStandardPagination` 后，两处 `vi.mock` 工厂缺该导出，导致 15 个用例在渲染时抛错。补 stub 后 21/21 通过，全量 `npm run test:unit` **54 文件 / 430 用例全绿**（修复前 2 文件 / 15 用例失败）。
4. **业务单测接入 CI**：`quality.yml` Frontend Contract 作业新增 `Unit tests`（`npm run test:unit`）；`test:scripts` 242/242 与 workflow baseline 校验通过。
5. **退役过期 e2e 资产**：`e2e-tests/` 全套 9 个文件（v1.2.0、2026-08-28）不在 CI、不在任何 npm 脚本，19 个端点全部指向旧路由前缀；当前模块挂载为 `/api/v1/business/{cicd/argocd,filedist,k8s,observability,service}`。删除目录并同步移除 `business-overlay.json` 的 `repositoryOverlayPaths` 条目；五套 CI 业务 smoke 与 430 个单测继续覆盖其模块范围。历史任务文档中的引用保留为过去记录。

## 反例实际运行（真实 MySQL）

在隔离验收库 `127.0.0.1:33321`（`ops_acceptance_formal_20261121`，DSN 不落盘不打印）实跑：

- 租户迁移：`TestTenantOwnershipMigrationUpgradesLegacyNameKeys` PASS 0.45s、`TestTenantOwnershipMigrationPreservesLegacyRowsAndRecoversPartialState` PASS 0.43s。
- 授权/范围：`TestReadableNamespacesFollowsClusterOwnership` PASS 0.39s（跨 scope、跨 environment 不可见）；k8s permission-route 匹配、service mutation-admission deadline、log-writer 背压、cluster 陈旧 scope 并发、bizscope/cmdb data-scope 全部 PASS。
- 恢复/注入：固定模板注入、路径穿越、危险卸载根、非法端口、危险版本元数据拒绝用例 PASS。

## 门禁基线

- `check-doc-links`（仓库根）：0 findings。工作树内出现的 20 条是 `.local/wt-next` 嵌套导致的相对路径假阳性，不影响仓库根结论。
- `check-doc-frontmatter`：0 error / 2 既有 warning（legacy metadata）。
- `check-task-packet`（report-only）：115 文件、657 error、118 warning，全部为历史 packet；本包 packet 不在 FAIL 列表。
- `check-encoding`：2 处既有 U+FFFD，均位于 `.harness/evidence/2026-11-17-ops-history-audit/` 归档 JSON。
- `check-business-overlay`：OK（55 declared paths, rebuilt tree）。

## 显式 gap

1. 维护者最终功能/视觉验收（Human Gate）仍开放；本批无 UI 改动，不需要渲染证据。
2. deadcode 未运行（会迫使 go.mod 变更，违反锁定继承），未删除任何候选。
3. `.business-overlay-report.json` 仍列出退役的 e2e 文件；它是重建时点回执，无门禁读取，下次重建自动刷新。
4. 未新增全量英文翻译（248 个中文文档无英文版属既有基线，不在本包范围）。
