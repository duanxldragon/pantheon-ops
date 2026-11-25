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
