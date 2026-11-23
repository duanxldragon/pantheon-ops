# T03 业务权限绑定与部署任务数据可见性 — 实施摘要

- Task: `2026-11-20-ops-business-permissions-and-task-visibility`
- 日期: 2026-11-23
- 候选: pantheon-ops worktree 未提交改动；本地候选二进制 `pantheon-ops-codex-t03-20261123-162526.exe` sha256 `0586a9e2…`（sourceStable=true）
- 状态: **implemented + 代码级验证 + 本地 live 角色矩阵 18/18 PASS；远端同候选重跑与维护者终验仍开放**
- 交付层级: L2（独立 review 见 `review.md`）

## 来源发现处置

| Finding | 处置 | 说明 |
|---|---|---|
| **SEC-002** 任务详情泄露范围外主机 IP/日志/输出 | 已修复 | `loadTaskHostsByTaskIDs` 对两个分支统一应用 `visibleTaskHosts(hosts, dataScope)`；新增回归 `TestListDeployTasksHidesOutOfScopeHosts` |
| **NEW-01** 任务列表先全表加载再过滤 | 已收敛 | scoped 分支改为 id-only `Pluck("id")` → 单次批量 `hostReader.GetByIDs` → `total=len(visible)` → 按可见 ID 集合取当前页（id DESC）；unscoped 分支为真正 SQL `Count`+`Limit`/`Offset`。外部 CMDB reader 无法下推 join，ID 候选列仍需全量读取，已用 `shortcut:` 注释登记升级条件 |
| **SUP-08** 细粒度角色不可用（只有 package 绑定） | 已修复（deploy 域） | 新增 template(6)/task(10, 不含 export)/credential(4) 绑定，共 26 键；每动作独立权限点（查看/创建/启动/取消/凭据写） |
| 路由与权限绑定逐条核对 | deploy 域已闭；其他业务域显式 gap | bizscope / cicd(argocd) / filedist / k8s-cluster-workload 仍 default-deny，未在本轮绑定 |

## 变更文件（本任务）

后端：
- `backend/modules/business/deploy/permission_routes.go` — 重写注册：package 6、template 6、task 10（**不含** `task:export`）、credential 4。详情路由 `GET /packages|templates|tasks/:id` 刻意不绑定（keyMatch2 会把 `/:id` 放宽到静态 `/export`），守卫测试锁定。
- `backend/modules/business/deploy/deploy_seed.go` — 新增 4 个 F 菜单行（`deployTaskListRoute` Sort 12–15，perms `business:deploy:credential:list|create|update|delete`）+ zh/en i18n（group `permission`）。
- `backend/modules/business/deploy/deploy_service.go` — `ListTasks` 重写（NEW-01）；`filterVisibleTaskIDs` 取代 `filterVisibleTasks`；`loadTaskHostsByTaskIDs` 应用 SEC-002。
- `backend/modules/business/deploy/permission_route_actions_test.go` — `deployPermKeys` 26 键 + anti-widening Casbin 矩阵 + 详情路由保留 gap 断言。
- `backend/modules/business/deploy/deploy_service_test.go` — 新增 SEC-002 回归测试。

前端（独立 review P1：菜单契约门禁红灯修复）：
- `frontend/src/i18n/resources/business/zh-CN.ts` / `en-US.ts`
- `frontend/src/modules/business/deploy/locales/zh-CN.json` / `en-US.json` — 补 `business.deploy.credential.permission.{list,create,update,delete}` 4 键。

证据/矩阵：
- `.harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/acceptance-matrix.json` — V1/V5/deploy-lane 追加 `t03Recheck`（历史探测字段保留）。
- 本目录 `t03-role-matrix-local.json`（18 探针 + cleanup 记录）、`commands.json`、`summary.md`、`review.md`。
- 本地探针脚本 `.local/acceptance-20261121/probe-t03-role-matrix.mjs`（本地专用，不提交）。

## 验证结果

| 命令 | 结果 |
|---|---|
| anti-widening + detail-guard 测试 | PASS（26 键每键只授予自身路由） |
| 聚焦 DB 测试（4 个，含新增 SEC-002） | PASS（exit 0） |
| `TestServiceTransactionFactoryRefusesInvalidOwnerBeforeMutation` 单跑 | PASS 17.58s（证明默认 600s 全量 FAIL 为累计 fixture 慢，不是正确性失败） |
| 全量 `./modules/business/deploy/...`（`-timeout 1800s`） | **PASS**（ok, 1047.796s，真实 MySQL；含 attempt ownership 套件） |
| `check-method-consumption.mjs` | PASS（1.5.0, 7 policies） |
| `check:business-boundaries` | PASS |
| `check:business-overlay` | PASS（55 declared paths） |
| `check-menu-contract.mjs` | 修复前 RED（缺 4×2 权限 i18n）；修复后 PASS（45 菜单 / 55 路由 / 250 权限） |
| TD03 候选构建 | PASS（exit 0, sourceStable=true, sha256 `0586a9e2…`） |
| 本地 live 角色矩阵 | **18/18 PASS**（候选 0586a9e2，后端 127.0.0.1:8081，DB `ops_acceptance_formal_20261122_codex`） |

### 本地 live 角色矩阵关键结论

- 只持 `business:deploy:task:view`：`GET /tasks` → 200；`POST /tasks`、`POST /tasks/1/start`、`DELETE /tasks/1`、`GET /tasks/export`、`GET /templates`、`GET /credentials` → 全部 `code=403 permission.denied`（导出未被 view 键放宽）。
- 只持 `business:deploy:credential:list`：`GET /credentials` → 200，响应无任何 secret/private/password/pem/token 字段；`POST/PUT/DELETE credentials` 与 `GET /tasks` → 403。
- 只持 `business:deploy:package:view`：`GET /packages` → 200；`GET /packages/1` → 403（详情路由保留 gap）；`GET /packages/export` 与 `POST /packages` → 403。
- 清理：3 个一次性用户 + 3 个一次性角色经系统 API 删除；DB 计数复核为 0。

## 验收标准映射

1. **查看角色可读、写操作拒绝** — 代码级 + 本地 live 均成立（见上表）。
2. **tenant+DataScope 过滤，分页总数准确，不先全表加载** — 代码级：SEC-002 + NEW-01 有测试；ID 候选列全量读取为登记的 shortcut。
3. **V2 查看角色无法导出；回调伪造/跨任务/过期 attempt 无副作用；共享匹配器缺陷移交 Base** — `task:export` 由既有 `log_download.go` 单独注册、本任务未重复；`template:export` 独立键；live 证明 view 键不能导出。attempt 归属逻辑未改动（既有 `attempt_ownership_test.go` 覆盖，全量套件执行）。Base keyMatch2 静态路由守卫仍是 Base-first 依赖，Ops 不 override。

## 显式 gaps

- **REMOTE-REPROBE**：远端候选（k8s33/k8s35 部署实例）未用 0586a9e2 重跑角色矩阵；本地 live 等价已证明绑定与 seed 生效。
- **NON-DEPLOY-BINDINGS**：bizscope/cicd/filedist/k8s 业务域仍无细粒度绑定，非管理员 default-deny。
- **SEC-004**：`DeployCredential` 无 owner/scope 字段，CRUD 全部无创建者/范围过滤；新增 `credential:list|create|update|delete` 后，任何持有对应键的非管理员可读/轮换/删除其他范围的凭据引用（密文不外泄，暴露面为元数据与轮换能力）。需 schema 变更，未在本轮做。
- **CREDENTIAL-DETAIL**：不存在 `GET /credentials/:id` 详情路由。
- **BASE-MATCHER**：`task:detail` 已 seed 但无法绑定详情路由；需 Base 静态路由守卫正式 release。
- **RACE**：`go test -race ./...` 未在本轮跑（维护者 T02 裁定接受 TSan 限制）。

## 安全七项检查

- 空值：`GetByIDs` 仅在 hostIDs>0 调用；空可见集合直接返回空页。
- 重复：26 键无重复；`task:export` 未重复注册（避免 `ErrConflict` 启动 panic）。
- 并发：改动为只读查询路径，无共享可变状态。
- 权限：default-deny 保持；anti-widening 测试 + live 矩阵锁定。
- 超时：无新增外部调用；hostReader 批量一次。
- 异常：reader 未配置错误码保持；查询错误原样上抛。
- 敏感信息：证据文件不含 DSN/密码/token；探针只输出状态码与响应键名扫描结果。

## 回滚

仅代码可逆回退本任务后端 5 文件 + 前端 4 文件；seed 新增 4 行菜单/i18n 可通过既有 seed 同步机制回退（DB 中已 upsert 的 4 行需手工删除或保留为无害空授权）；无数据库 schema 变更。本地候选二进制与探针脚本只存在于 `.local/`。
