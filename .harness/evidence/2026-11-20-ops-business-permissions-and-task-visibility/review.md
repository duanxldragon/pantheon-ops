# T03 Review — 业务权限绑定与部署任务数据可见性

## 结论

`pass-with-explicit-gaps`。独立只读审查发现 1 个 P1（前端权限 i18n 缺口导致菜单契约门禁红灯），已修复并由 `check-menu-contract.mjs` 复核通过；其余 P2 为已登记的残余风险或成本项。代码级验证、本地 live 角色矩阵（18/18 PASS）与三项治理检查均通过。远端同候选重跑与维护者终验保持 open gap，不代签。

## Findings（严重度从高到低）

1. **P1 — 4 个新权限 title key 缺少前端 fallback i18n（已修复）。**
   `deploy_seed.go` 新增 `business.deploy.credential.permission.{list,create,update,delete}` 后端 seed/i18n，但 `frontend/src/i18n/resources/business/{zh-CN,en-US}.ts` 与 `frontend/src/modules/business/deploy/locales/{zh-CN,en-US}.json` 未同步；`node frontend/scripts/check-menu-contract.mjs` 为 RED（8 条 fallback 缺键错误），离线/回退 UI 的角色权限树会显示原始 key。修复后检查为 PASS（45 菜单 / 55 路由 / 250 权限 / 54 组件键）。复核方式：命令重跑 + 4×2 键存在性。

2. **P2 — anti-widening 测试未锁定“`task:export` 不得重复注册”的不变量（accepted）。**
   当前 `permission_routes.go` 未重复注册（正确），但测试把 export 排除在 `deployPermKeys` 之外；未来若有人重复注册，单元测试不会报红，只会在生产启动时触发 `permissionroutes.ErrConflict` panic。残余风险低（启动即失败、不会静默放权），保留为后续可加断言项。

3. **P2 — 新 credential 绑定把无范围过滤的全局 CRUD 暴露给非管理员（accepted gap）。**
   `DeployCredential` 无 owner/scope 字段，`ListCredentials` 及 create/update/delete 均无创建者或数据范围谓词。此前仅 admin 通配可及，现持有 `business:deploy:credential:*` 的受限角色可读/轮换/删除其他范围的凭据引用。密文不进入响应，暴露面为元数据与轮换能力。需 schema 变更，登记为 `SEC-004` gap，本轮不做。

4. **P2 — scoped 列表成本随总量增长（documented shortcut）。**
   scoped `ListTasks` 先只取全部候选 ID 列，再批量 `hostReader.GetByIDs`（无上限），无主机行（草稿）任务回退 `resolveTaskTargets`（N+1）。正确性已验证（分页/总数/排序/空页），成本随任务与主机总量线性增长；代码内 `shortcut:` 注释已登记升级条件（外部 CMDB 支持 join 谓词后改为 SQL 下推）。

5. **Info — 详情路由与 `task:detail` 的保留缺口（Base-first）。**
   `GET /packages|templates|tasks/:id` 刻意不绑定，`task:detail` 虽已 seed 但无路由效力；根因是 released `keyMatch2` 会把 `/:id` 放宽到静态 `/export`。守卫测试 `TestDeployPackageDetailRemainsUnregisteredForView` 锁定，等待 Base 静态路由守卫正式 release。

## 已验证干净

- **注册契约**：module/key 模式合法、26 键唯一、无批内重复；`log_download.go` 单独拥有 `business:deploy:task:export` 且本包未重复；注册先于 `permissionroutes.Seal()`。
- **无放宽**：新增 `:id` 绑定均为写操作；静态 export/import 为不同段数的 GET/POST，`keyMatch2` 无法放宽；无新增 GET 详情策略。anti-widening + detail-guard 测试 PASS。
- **分页正确性**：`pageStart=(Page-1)*PageSize`，Page/PageSize 归一化；可见 ID 保持 `id DESC`；越界页返回空数组；unscoped 为真实 `Count`+`Limit/Offset`。
- **SEC-002**：两个分支统一应用 `visibleTaskHosts`；`HostCount=len(visible hosts)`；新增回归测试 PASS。
- **Seed 对齐**：4 个唯一 F 键挂 `deployTaskListRoute` Sort 12–15，perms 与前端 `index.ts` 一致；seed i18n (key,locale) 唯一；DB 复核已 upsert。
- **Live 矩阵**：本地候选 `0586a9e2` 18/18 PASS，含 granted-200 / denied-403 / 导出抗放宽 / 凭据响应无 secret 键 / 清理归零。

## 残余 gap（不代签）

- 远端候选（k8s33/k8s35）未用 `0586a9e2` 重跑角色矩阵；本地 live 为等价证据。
- bizscope / cicd / filedist / k8s 业务域仍 default-deny（未绑定）。
- `SEC-004` credential 归属/范围过滤缺失（见 finding 3）。
- 无 `GET /credentials/:id`；Base 静态路由守卫未发布。
- 完整仓库 `-race` 按 T02 维护者裁定接受 TSan 限制，未重跑。
- 独立 review 未在 P1 修复后二次复跑；修复为机械性 i18n 补键，由菜单契约检查器复核。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-business-permissions-and-task-visibility",
  "reviewer": "independent read-only subagent + primary disposition",
  "decision": "pass-with-explicit-gaps",
  "scope": "deploy permission bindings, credential seed/i18n, ListTasks visibility (SEC-002/NEW-01), SUP-08",
  "taskManifest": ".harness/tasks/2026-11-20-ops-business-permissions-and-task-visibility/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-business-permissions-and-task-visibility",
  "summaryFile": ".harness/evidence/2026-11-20-ops-business-permissions-and-task-visibility/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-business-permissions-and-task-visibility/review.md",
  "changeRef": "none (uncommitted worktree)",
  "candidate": {
    "binarySha256": "0586a9e2588910aece8fd19410e2eea3111c7a0015a739c97e6eb024fcf124a0",
    "sourceStable": true,
    "buildReceipt": ".harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/codex-build-t03-20261123-162526.json"
  },
  "findings": [
    {"id": "T03-R1", "severity": "P1", "status": "fixed", "summary": "missing frontend fallback i18n for 4 credential permission title keys; menu-contract gate RED", "verification": "node frontend/scripts/check-menu-contract.mjs -> PASS (250 permissions)"},
    {"id": "T03-R2", "severity": "P2", "status": "accepted", "summary": "anti-widening test does not assert task:export is never re-registered"},
    {"id": "T03-R3", "severity": "P2", "status": "accepted-gap", "summary": "new credential bindings expose unscoped global CRUD to granted non-admin roles (SEC-004 schema limitation)"},
    {"id": "T03-R4", "severity": "P2", "status": "accepted-shortcut", "summary": "scoped ListTasks cost grows with total task/host volume; documented shortcut"},
    {"id": "T03-R5", "severity": "info", "status": "base-first-dependency", "summary": "detail routes and task:detail remain unbound until Base keyMatch2 static-route guard ships"}
  ],
  "verification": {
    "liveRoleMatrix": {"file": "t03-role-matrix-local.json", "passed": 18, "failed": 0, "base": "http://127.0.0.1:8081"},
    "menuContract": "PASS",
    "methodConsumption": "PASS",
    "businessBoundaries": "PASS",
    "businessOverlay": "PASS",
    "fullDeploySuite": "PASS (ok, 1047.796s, real MySQL)"
  },
  "residualGaps": [
    "remote candidate re-probe (local equivalent evidence only)",
    "non-deploy business bindings still default-deny",
    "credential owner/scope filter schema change (SEC-004)",
    "credential detail route absent",
    "Base static-route guard not released",
    "full-repo -race per T02 maintainer ruling"
  ]
}
```
