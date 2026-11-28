# 安装包、模板与执行器能力一致性

- Status: completed（代码 + npm 门禁 + 后端回归 + 真实浏览器 smoke；Helm 无 UI 面按 gap 记录）
- Updated: 2026-11-26
- Baseline: 0.19.0（`origin/main` `5a1166c5`，foundation `cc2fe737`），本包基于 T21 提交 `18d7f08a`
- Owner: 安装部署负责人
- Priority: P1

## Source

DEP-004 / NEW-03：固定模板声明了 MySQL/MinIO/Harbor 等没有预置包的安装能力；`validExecutorType` 接受 agent，但 `dispatchStartedTask` 只处理 ssh，agent 任务会启动后静默不执行；Helm 仅为 DB 快照但缺少面向使用者的口径。

## Work Breakdown

- [x] S1 一致性检查：`deploy_fixed_assets.go` 的 `FixedTemplateAssets` 目录 + `UnbackedFixedTemplates`；`TestFixedTemplateAssetCoverage` 对照 `files/amd64`；缺包反例 `TestFixedTemplateAssetCoverageReportsMissingPackage`；`npm run check:deploy-assets`
- [x] S2 模板元数据：MySQL/MinIO/Harbor 显式 `NeedsOwnPackage`；`fixedTemplateCodes()` 收口模板白名单，目录必须覆盖全部模板
- [x] S3 Agent 执行器：`validExecutorType` 只留 manual/simulated/ssh；创建/更新返回 `business.deploy.task.executorTypeUnsupported`；start 对存量 agent 行同样拒绝；前端移除选项
- [x] S4 存量数据：无迁移；详情可读、改名保留类型、启动被拒；前端对 agent 行标注“不可执行”
- [x] S5 Helm 口径：包注释与 K8s 设计文档已有“仅登记快照”；仓库无 Helm UI 页面，登记为 gap
- [x] S6 延后项：nginx/redis/prometheus 的 fresh-install 实机验证归 T17
- [x] S7 证据：`npm run check:deploy-assets`、deploy 定向回归、helm 包、`tsc`/ESLint/`npm run build`、smoke 与截图

## 验证记录（2026-11-26）

- `npm run check:deploy-assets` exit 0：
  - `TestFixedTemplateAssetCoverage` PASS（真实目录：nginx/redis/jdk/bisheng-jdk/prometheus 有预置包；mysql/minio/harbor 标记需自备）
  - `TestFixedTemplateAssetCoverageReportsMissingPackage` PASS（缺包反例被报告）
- `go test -run 'TestFixedTemplateAsset|TestCreateTaskRejectsAgent|TestUpdateTaskRejectsAgent|TestLegacyAgentTask' ./modules/business/deploy/` 5/5 通过。
- `go test ./modules/business/k8s/helm/` 通过；`-run 'TestDeployTaskExecutor|TestCreateTask|TestDeployTaskImport'` 通过。
- `tsc -b` exit 0；改动文件 ESLint 0/0；`npm run build` 含全部 prebuild 契约门禁通过。
- `deploy-api` smoke 5 passed；新用例“the task create form does not offer the agent executor”通过（`t22-executor-options-1440.png`）。
- `deploy-pages` 9 passed / 1 failed：失败为既有 route-content-fallback 竞态（单跑通过），与本包无关。

## Known Gaps

- 仓库内没有 Helm UI 页面，S5 的“页面明示”无可改面；口径在 `backend/modules/business/k8s/helm/helm.go` 包注释与 `docs/designs/BUSINESS_K8S_MODULE_DESIGN.md`。
- nginx/redis/prometheus 的 fresh-install 实机验证仍归 T17。
- 资产检查按文件名前缀匹配，不校验哈希。
- MySQL/MinIO/Harbor 仍需操作员自备包，选包时未预检归档名。
- deploy 整包默认超时问题仍未处理（既有，见 T20 STATUS）。


## 11 月 28 日交付候选

- Windows 完整 CGO race 已通过，独立复核及 GitHub 交付待收口。

- 11 月 28 日候选串行完整 CGO race 通过，仍按各任务验收项收尾。[验证汇总](../../evidence/2026-11-25-ops-install-assets-executor-consistency/summary.md)
