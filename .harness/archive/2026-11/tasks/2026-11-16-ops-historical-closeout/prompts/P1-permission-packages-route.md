# Prompt P1 — 权限缺口收口：/packages/:id 授权（insufficient-evidence → 定论）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。

## 问题

旧记录 `2026-07-17-business-permission-gaps` 的三个 PR 中：

- PR1（deploy template F 权限种子）与 PR2（cmdb `/labels/options` → `label:view`）已在树内（deploy_seed.go / cmdb/permission_routes.go）。
- **PR3 悬而未决**：原 `backend/pkg/contracts/permission_policies.go` 已被删除，现机制为 `permissionroutes.Register` + `keyMatch2`；`backend/modules/business/deploy` 只有 `business:deploy:task:export` 注册（log_download.go），seed 用 `business:deploy:package:view`；未找到 `GET /packages/:id` 的显式授权绑定，而 `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md` 仍标该缺口为开放。

## 范围 / 方法

- 判定 `GET /packages/:id`（及所属 package list/view）在当前权限路由下是否被实际授权：
  - 若 `keyMatch2` 按路由前缀自动覆盖，则写一个**通过性/负向测试**锁定该结论，并更新设计文档缺口注释为已验证/已授权，附路由前缀证据。
  - 若不覆盖，则补齐最小授权（显式 `permissionroutes.Register` 项或等价声明），并使设计文档一致。
- 同步刷新 deploy 设计文档中旧"缺口开放"标注。

## 精确文件所有权

- `backend/modules/business/deploy/*`（授权路由/测试面）
- `backend/modules/business/cmdb/permission_routes.go`（只读参考，除非需对照模式）
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`
- `.harness/{tasks,evidence}/2026-07-17-business-permission-gaps/`（关闭链；若结论为既有授权则标注 verified，否则 missing-implementation 并列出补丁）

## 约束

- 最小复杂度、Base-first（权限路由机制属 base/system 域，若需改机制则 STOP 回 Base；本批只允许在 business 侧补业务子路径授权）。
- 不覆盖他人改动；不触碰 cmdb `/labels/options` 既有实现。
- 不提交/不 push/不发布；不允许伪造测试通过证据。

## 真实验收

- 明确的授权结论（authorized-with-test 或 补显式绑定），设计文档 + 权限记录一致。
- `go test ./backend/modules/business/deploy ./backend/modules/business/cmdb/...` 通过；`npm run check:menu-contract` 通过。
- 无鉴权削弱。
