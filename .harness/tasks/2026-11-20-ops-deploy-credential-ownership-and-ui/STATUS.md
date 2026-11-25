# 部署凭据资源授权与管理页面

- Status: in-progress
- Updated: 2026-11-25
- Baseline: main `cb880eea`（规划基线 a2d69d4a，foundation `pantheon-base-v0.18.9`）
- Owner: 部署凭据负责人
- Priority: P1
- Activation: core
- Dependencies: T03（completed）、T04（completed）

## Next Atomic Action

在有授权部署主机的环境执行 V5：创建业务域凭据 → 真实 SSH 启动 → 轮换密钥后再次启动 → 停用后启动被拒；同时用浏览器在 1440px/390px 截图管理页的 list/create/edit/error 状态，补齐后再置 completed。

## Evidence

- 后端：凭据按业务域归属，列表/更新/删除走数据范围；启动/预览要求凭据与任务同域；在用凭据拒绝删除；`scope=0` 为管理员平台级。
- 迁移：`000042_deploy_credential_scope` 幂等加列与索引，历史行保留 0。
- 前端：`business/deploy/credential/DeployCredentialList` 页面 + 菜单/路由/权限 + 五语言文案。
- 命令与结果：`.harness/evidence/2026-11-20-ops-deploy-credential-ownership-and-ui/commands.json`
- 结论与 gap：同目录 `summary.md`、`review.md`（verdict: changes requested）。

## Known Gaps

1. V5 真实 SSH 凭据使用/轮换未执行（无授权主机）。
2. 管理页面无浏览器渲染证据（仅 tsc/eslint/build 通过）。
3. `tenant_id` 仍为 0，凭据级租户隔离待业务路由携带租户声明（T14/T19）。
4. `businessOverlayComponentRegistry.ts` 仅追加页面注册，作为业务 overlay 扩展点记录在 manifest expectedFiles。
