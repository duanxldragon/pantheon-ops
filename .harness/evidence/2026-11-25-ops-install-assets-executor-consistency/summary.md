# T22 摘要 — 安装包、模板与执行器能力一致性

- 状态：completed（代码 + 门禁脚本 + 后端回归 + 真实浏览器 smoke）
- 候选：`18d7f08a` + 本包工作树（分支 `feature/ops-t20-service-instance-deploy-entry`）
- 基线：`pantheon-base-v0.19.0`（`cc2fe737`）
- 来源发现：DEP-004、NEW-03

## 已交付

### S1 一致性检查

- `deploy_fixed_assets.go` 新增机器可读目录 `FixedTemplateAssets`：每个固定模板要么声明 `files/amd64` 的包名前缀（nginx-/redis-/jdk-/bisheng-jdk-/prometheus-），要么显式标记 `NeedsOwnPackage`（mysql_systemd / minio_systemd / harbor_offline）。
- `UnbackedFixedTemplates` 做纯函数匹配，`TestFixedTemplateAssetCoverage` 对照真实的 `files/amd64`，`TestFixedTemplateAssetCoverageReportsMissingPackage` 是缺包反例。
- `fixedTemplateCodes()` 成为模板白名单的唯一来源，`TestFixedTemplateAssetCoverage` 要求目录覆盖全部模板，新增模板不能绕过资产决策。
- 新增 `npm run check:deploy-assets`（`go test -run TestFixedTemplateAsset`），有真实运行输出。

### S2 模板元数据

- “需自备包”不再隐含在注释里：MySQL/MinIO/Harbor 三个模板在目录中显式标记，缺包不会被误报为资产缺失，但也不会被当成有预置包。

### S3 Agent 执行器

- `validExecutorType` 只接受 `manual`、`simulated`、`ssh`；创建、更新两条路径对 `agent` 返回专用错误键 `business.deploy.task.executorTypeUnsupported`（其余非法值仍是 `invalidExecutorType`）。
- 前端创建表单移除 Agent 选项；smoke 打开真实创建弹窗，断言只有三者且没有 `Agent 执行`。

### S4 存量数据

- 无迁移、无改写：`TestLegacyAgentTaskStaysReadableButCannotStart` 直接写一行 `executor_type=agent`，验证详情可读、只改任务名可成功且类型不变、启动被专用错误键拒绝。
- 列表与详情对 `executorType === 'agent'` 渲染“不可执行”标记。

### S5 Helm 口径

- 仓库内没有 Helm 页面；`k8s/helm` 包注释与 `docs/designs/BUSINESS_K8S_MODULE_DESIGN.md` 已声明“仅 DB 登记快照、不执行真实安装”。本包未重复改写，只在 review 里登记为无可改面的 gap。
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md` 补充执行器收敛与资产检查两段，使文档与代码一致。

### S6 延后项登记

- nginx/redis/prometheus 的 fresh-install 实机验证仍归 T17；本包只保证“包存在且被模板认领”，不重复验收。

## 验证

- `npm run check:deploy-assets` exit 0：`TestFixedTemplateAssetCoverage` 与缺包反例 `PASS`。
- `go test -run 'TestFixedTemplateAsset|TestCreateTaskRejectsAgent|TestUpdateTaskRejectsAgent|TestLegacyAgentTask' ./modules/business/deploy/` 5 个用例通过。
- `go test ./modules/business/k8s/helm/` 通过；`-run 'TestDeployTaskExecutor|TestCreateTask|TestDeployTaskImport'` 通过。
- `tsc -b`、改动文件 ESLint、`npm run build`（含全部 prebuild 契约门禁）exit 0。
- `deploy-api` smoke 5 passed；新 UI 用例“does not offer the agent executor”通过并落图 `t22-executor-options-1440.png`。
- `deploy-pages` 9 passed / 1 failed，失败是既有 route-content-fallback 竞态（单跑通过），与本次无关。

## 未关闭项（显式 gap）

1. 仓库无 Helm UI 页面，S5 的“页面明示”没有可改面，仅在包注释与设计文档中有口径。
2. nginx/redis/prometheus 的实机 fresh-install 验证属 T17。
3. 资产检查按文件名前缀匹配，不校验哈希。
4. MySQL/MinIO/Harbor 仍可选，需操作员自备包；UI 未在选包时预检归档名。

### 11 月 28 日资产门禁复验

- 基线为 Ops main 875331ba 与 Base v0.19.1。普通 Go 测试验证模板目录的覆盖和缺包反例。打包入口验证给定的真实归档目录，未给路径时检查 files/amd64。
- 使用本地现有 Linux amd64 制品通过包装检查。缺失目录退出非零，缺包反例通过。组织级归档发布和分发来源尚未在 GitHub 记录。

- Windows 本地使用真实制品目录，部署 API 4/4、执行器选择浏览器用例 1/1。桌面 1440px 证据见 screenshots。前端 lint、type-check、build 及 Base 干净重建和共享漂移检查通过。
- 归档验证证据记录的是本机已有制品。nginx-1.31.9.tar.gz SHA-256 a11c5eeee2e257f4478f0e649f84b0040787d277d2d3e5233952aa6b6dcb80ad。redis-7.4.12.tar.gz SHA-256 324ab63d55c607639b419f547b213c62ae39ff50e09870e450a11c272567b84f。prometheus-3.16.0.linux-amd64.tar.gz SHA-256 68727009472a2c02e1ff7f062fd5b3e637285bffc8fc2d340c76fb449d6e0f34。bisheng-jdk-8u462-b11-linux-x64.tar.gz SHA-256 1c381c8a5e85d7c43d8093fffb42a6e085db235bef1ed5a932f9409729265035。

## 11 月 28 日交付复核

- 在 Windows MinGW CGO、隔离 UTC MySQL 和 Redis 上串行执行 `go test -race ./...`，78 个包通过，15 个包跳过，退出码 0。产品树 `2a7ec524939b4cc7e393e1fbcb27b412d5d1dcb7`。[完整门禁回执](./windows-cgo-race-20261128.json)
- 同候选部署 API 4/4，浏览器执行器用例 1/1。已核看 1440px 创建表单的执行方式控件，390px 执行器选项待补图。
- 本机 Linux amd64 归档目录和缺包反例已验证，正式制品来源仍待记录。Helm 登记说明保留在包和设计文档。
- 本地证据已按候选绑定，暂留独立审批和 GitHub 门禁记录。
