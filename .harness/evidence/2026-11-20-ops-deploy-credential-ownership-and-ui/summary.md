# T05 摘要 — 部署凭据资源授权与管理页面

- 状态：in-progress（代码级闭合，真实环境与视觉验收未执行）
- 候选：main `cb880eea` + 本包工作树；foundation `pantheon-base-v0.18.9`
- 来源发现：SEC-004（凭据缺少资源范围授权）、DEP-003（凭据无管理页面）

## 已交付

### 后端（SEC-004）

- `DeployCredentialRef` 增加 `business_scope_id`，迁移 `000042_deploy_credential_scope`（幂等、只加列与索引；down 为保留式 no-op）。
- 列表/更新/删除统一走数据范围子查询：普通角色只命中自己数据范围内的业务域凭据；越权 ID 一律返回 `business.deploy.credential.not_found`，不泄露存在性。
- 创建要求业务域；`business_scope_id = 0` 为管理员平台级凭据，受限角色不能创建。
- 启动与预览要求凭据业务域与任务业务域一致（`out_of_scope`）；队列恢复路径按已授权快照重放（`allowGlobal=true`），不改变重启语义。
- 删除前检查 `credential_ref_id` 引用，被任务引用的凭据拒绝删除（`in_use`）。
- AES-GCM 加密、响应脱敏、密钥配置契约保持不变。

### 前端（DEP-003）

- 新增 `business/deploy/credential/DeployCredentialList`：列表（名称/业务域/用户名/认证方式/版本/状态/更新时间）、搜索、创建、编辑、轮换密钥、停用、删除确认。
- 菜单与路由 `business/deploy/credential`，页面权限 `business:deploy:credential:list`；后端菜单 seed 增加 C 型菜单。
- 五语言文案（zh-CN/en-US/fr-FR/ja-JP/ko-KR）。
- 复用 `PageContainer`、`GovernanceSummaryBar`、`SearchToolbar`、`AppTable`、`AppModal`、`SubmitBar` 与标准分页。

### 契约与设计

- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md` 增加 7.3.6a 凭据归属与作用域小节。

## 验证

- `go test -race -run TestDeployCredential`：PASS 6.58s（真实 MySQL），覆盖跨域不可见/不可改/不可删、启动域不一致、在用删除拒绝、管理员平台级限制、迁移幂等。
- 邻近用例 `TestDeployTaskSSHExecutorRunsInstallAndWritesBackHostState`、`TestDeployTaskVisibilityRespectsHostDataScope`、`TestDeploySSHTransportBusinessExecution`、`TestPreview*`：PASS 48.68s。
- `gofmt -l` 无输出、`go vet` 通过；前端 `tsc -b`、`eslint`（改动文件）、`vite build` 通过。

## 未关闭项

1. V5 真实 SSH 凭据使用/轮换未执行：当前环境没有可用授权主机。
2. 管理页面无 1440px/390px 渲染截图与浏览器 smoke；仅证明可编译、可打包。
3. `tenant_id` 仍为 0：业务路由尚未接收租户声明（ARC-001，T14/T19 承接），没有凭据级租户隔离证据。
4. `frontend/src/core/router/businessOverlayComponentRegistry.ts` 属于业务 overlay 扩展点，仅为注册新页面而修改。

因此本包不以 completed 收口；待上述真实环境/视觉证据补齐后关闭。

## 2026-11-27 Base v0.19.1 隔离候选

- 基线为 Ops `02088145`、Base `pantheon-base-v0.19.1`。只提取凭据管理、范围选择、任务预览/执行契约与相应测试。保留原工作树用于对账。
- 用真实隔离 UTC MySQL 和两套 Redis 执行 Windows MinGW CGO 定向 race，40 条通过事件。覆盖部署凭据管理、业务域归属及相关回归。
- 前端 smoke 入口补登记凭据页面用例。真实角色 smoke 6/6，其中只读角色有 1440px/390px 渲染。任务表单出现 React 重复 key 警告，未观察到用例断言失败。
- 本次不触碰生产凭据或主机。真实 SSH 证据来自旧候选，当前独立复核、完整 race 以及最终验收仍按本轮结果记录。
- 与 T14 开发树重叠 21 个产品路径，其中 14 个内容不同，须在各模块交接时合并语义。

### 候选验证结论

- 前端 lint、type-check、build、smoke 覆盖合同、干净重建、共享漂移检查通过。按 Impeccable 核对了真实浏览器 1440px 浅色、390px 深色只读列表。
- Windows 完整 `go test -race ./...` 未通过。76 个包输出通过，`business/deploy` 在 40 分钟时触发包超时。
- 完整 race 日志 SHA-256 为 `4f15fd1c6f80b6f71b7b1ec336b41ebf43620b6b78b7d3d77345ab9d633efdb9`。候选保留在本地，独立评审和 GitHub 交付仍开放。

### 2026-11-27 串行全量复验

- Windows MinGW CGO，隔离 UTC MySQL 与两套独立 Redis，单候选运行 `go test -race ./...`。77 个测试包通过，14 个无测试包，exit 0，耗时 352 秒。
- MySQL 为临时实例，禁用 redo 与 doublewrite 以缩短测试库初始化与清理开销。复验限定运行态与事务行为，生产耐久性不在本次验证范围内。
- 验证树 `48fd04eb38c69ec48ff0adcf2e79db2ce42dd8c1`，日志 SHA-256 `7c754398dc1f2e9954023f71e38e13a10b379909c8fb53b11913331f4f92a0c9`。当前候选仍待独立高风险评审与 GitHub 必过检查。
