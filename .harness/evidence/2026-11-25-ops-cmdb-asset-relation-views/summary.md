# T21 摘要 — CMDB 主机与服务实例、业务域关联视图

- 状态：completed（代码 + 后端测试 + 真实浏览器 smoke + 1440/390 渲染证据）
- 候选：`04c230dc` + 本包工作树（分支 `feature/ops-t20-service-instance-deploy-entry`）
- 基线：`pantheon-base-v0.19.0`（`cc2fe737`）
- 来源发现：NEW-02

## 已交付

### S1 后端读取接口（复用既有列表，不新增跨模块依赖）

- `InstanceQuery.HostID`：`GET /business/service/instances?hostId=` 在 SQL 里按 `host_id` 过滤，数据范围与服务目录一致。
- `TaskQuery.HostID`：`GET /business/deploy/tasks?hostId=` 用 `JSON_CONTAINS(target_ids, ?)` 过滤，可见性过滤仍在其后执行，过滤不会扩大可见集合。
- `GET /business/service/summary?businessScopeId=`：按数据范围统计该业务域下的应用/服务/实例数，登记到 `business:service:list` 与 `business:service:view` 权限路由。

### S2 一致性

- 实例环境与业务域环境的对应关系继续由 service 侧准入负责，cmdb 不复制规则；主机关联视图只读，不写实例状态，因此没有新的不一致入口。

### S3 bizscope 反例测试

- 应用引用阻止删除：返回 `business.bizscope.inUse`，应用软删除后可删除，行确实消失。
- 并发删除：两个 goroutine 同时删同一业务域，恰好一个成功、一个失败，不留半删状态。
- 既有数据范围测试（列表/详情 hostCount）继续通过。

### S4 主机详情

- 新增「承载服务实例」只读表格（服务名可跳到该服务的实例列表、环境、当前版本、运行状态）与「最近部署」只读列表（任务名可跳详情、状态、完成时间）。
- 无 `business:service:list|view` 时整块隐藏；无 `business:deploy:task:view` 时部署块隐藏，而不是渲染空白。

### S5 业务域详情

- 新增「服务目录统计」卡片：应用/服务/实例三个计数，点击分别跳到服务目录对应 tab。
- 无服务目录读权限时整块隐藏。

### S6 受限角色

- 关联视图通过既有列表接口读取，数据范围由后端保证；后端用例覆盖“域外主机对 dept 范围调用方返回空、admin 可见”。

### S7 i18n

- 新增 15 个 key（cmdb 10 + bizscope 5），zh-CN/en-US/fr-FR/ja-JP/ko-KR 齐全；cmdb 模块 locales 同步 zh/en，bizscope 模块 locales 同步五语言。

## 验证

- `go build ./...`、三个包 `go vet` exit 0。
- `go test ./modules/business/bizscope/ ./modules/business/service/ ./modules/business/cmdb/...` 全绿（6 个包）。
- 新增后端用例：`TestListInstancesFiltersByHostAndScopeSummary`、`TestListTasksFiltersByHostWithinDataScope`、`TestBizScopeDeleteRejectsApplicationReference`、`TestBizScopeDeleteIsSingleWinnerUnderConcurrency`。
- `tsc -b` exit 0；改动文件 ESLint 0/0；`npm run build` 连同全部 prebuild 契约门禁通过。
- `tests/smoke/business/cmdb/cmdb-pages.spec.ts` **10 passed (37.8s)**：新用例造 scope/host/app/service/instance/package/task，断言主机详情出现承载实例与最近部署、业务域详情出现统计卡片，390px 无横向溢出。
- 截图：`cmdb-host-relations-1440.png`、`cmdb-host-relations-390.png`、`bizscope-catalog-summary-1440.png`、`cmdb-host-detail.png`、`bizscope-list.png`。

## 未关闭项（显式 gap）

1. 主机详情为两个额外读请求（实例 + 部署），未做服务端聚合端点。
2. 实例没有显示名，主机卡片以服务名 + 版本 + 状态呈现。
3. 业务域统计的三个跳转落到未按业务域过滤的目录 tab（目录尚无 businessScopeId 过滤）。
4. “跨租户”以部门数据范围验证，测试库 bizscope 表无 tenant 列。

### 11 月 28 日基线复验

- 已移到 Ops main 875331ba 与锁定 Base v0.19.1。Go Lint 无新增问题，前端 lint、type-check、build，以及业务 overlay、干净重建和共享漂移检查通过。
- 首轮完整 Windows race 的可观测 HTTP 取消测试出现时序失败。隔离串行复验已通过。

- 同候选使用 versioned migrations、隔离 UTC MySQL 和 Redis，部署 API 5/5，服务目录与 CMDB 浏览器 smoke 25/25。二进制 SHA-256 b2e4a7e2f68bb656f9d156811b91740baad574b40becc51c2b84d26312e99153。桌面 1440px 和窄屏 390px 证据见 screenshots；相关布局和已覆盖权限状态可复验。

## 11 月 28 日交付复核

- 在 Windows MinGW CGO、隔离 UTC MySQL 和 Redis 上串行执行 `go test -race ./...`，78 个包通过，15 个包跳过，退出码 0。产品树 `730c46af902502d9abe1e05654ae63ccde7459f4`。[完整门禁回执](./windows-cgo-race-20261128.json)
- 同候选部署 API 5/5，服务目录与 CMDB 浏览器 smoke 25/25。已核看 390px 主机详情，关联区使用现有详情页面布局。
- 业务域统计跳转的过滤和双租户同候选证据由后续租户运行态验证继续跟踪。
- 本地证据已按候选绑定，暂留独立审批和 GitHub 门禁记录。
