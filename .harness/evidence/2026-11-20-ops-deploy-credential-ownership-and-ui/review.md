# T05 Review — 部署凭据资源授权与管理页面

## 结论

`changes requested`（就完整验收而言）。SEC-004 的资源范围授权与 DEP-003 的管理页面已在代码层闭合并有真实 MySQL 回归；真实 SSH、浏览器渲染与租户级隔离仍未验证。

## Findings

1. **P1 fixed — 凭据跨业务域可读/可改/可删。**
   凭据新增 `business_scope_id`，列表、更新、删除都经过数据范围子查询；越权 ID 统一按 not_found 处理。`TestDeployCredentialListRespectsBusinessScope` PASS（真实 MySQL），跨域 update/delete 均被拒绝。

2. **P1 fixed — 按 ID 用他域凭据启动任务。**
   `resolveStartCredential` 要求凭据业务域与任务一致，`TestDeployCredentialStartRequiresMatchingBusinessScope` 断言 `credential.out_of_scope`；队列恢复路径显式 `allowGlobal=true`，只重放已授权快照。

3. **P2 fixed — 删除在用凭据破坏历史任务引用。**
   删除前按 `credential_ref_id` 计数，命中返回 `credential.in_use`；测试在任务启动后验证拒绝。

4. **P2 fixed — 旧凭据默认变成全局共享。**
   历史行保持 `business_scope_id = 0`，视为管理员平台级：受限角色既不能创建也不能用于有业务域的任务；`TestDeployCredentialGlobalScopeIsAdminOnly` 覆盖两侧。

5. **P1 open — 没有真实 SSH 使用/轮换证据（V5）。**
   本环境无授权部署主机，`TestDeploySSHTransportBusinessExecution` 只证明 loopback 传输契约，不代表真实主机凭据轮换。

6. **P2 open — 管理页面无渲染证据。**
   仅 `tsc -b`、`eslint`、`vite build` 通过；1440px/390px、明暗主题和 loading/empty/error/forbidden 状态需要浏览器截图或 smoke 才能满足本包视觉验收。

7. **P2 open — 凭据级租户隔离未落地。**
   `tenant_id` 列存在但业务路由未接收租户声明（ARC-001），本包不臆造来源；登记为 T14/T19 的前置。

8. **info — overlay 扩展点。**
   `businessOverlayComponentRegistry.ts` 位于 `frontend/src/core`，是业务 overlay 注册新页面的既定入口，本次只追加一行注册，不改变共享 core 行为。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-deploy-credential-ownership-and-ui",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "changes requested",
  "decision": "changes requested",
  "scope": "deploy credential business-scope ownership, start scope binding, delete-in-use guard, credential management UI",
  "taskManifest": ".harness/tasks/2026-11-20-ops-deploy-credential-ownership-and-ui/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-deploy-credential-ownership-and-ui",
  "summaryFile": ".harness/evidence/2026-11-20-ops-deploy-credential-ownership-and-ui/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-deploy-credential-ownership-and-ui/review.md",
  "changeRef": "none",
  "findings": [
    {
      "id": "T05-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "cross-scope credential read/update/delete denied by data-scope query",
      "verification": "TestDeployCredentialListRespectsBusinessScope PASS"
    },
    {
      "id": "T05-R2",
      "severity": "P1",
      "status": "fixed",
      "summary": "start/preview require credential business scope to match the task",
      "verification": "TestDeployCredentialStartRequiresMatchingBusinessScope PASS"
    },
    {
      "id": "T05-R3",
      "severity": "P2",
      "status": "fixed",
      "summary": "credentials referenced by tasks cannot be deleted",
      "verification": "credential.in_use assertion PASS"
    },
    {
      "id": "T05-R4",
      "severity": "P2",
      "status": "fixed",
      "summary": "legacy scope-0 rows stay admin-owned instead of global",
      "verification": "TestDeployCredentialGlobalScopeIsAdminOnly PASS"
    },
    {
      "id": "T05-R5",
      "severity": "P1",
      "status": "open",
      "summary": "no live SSH credential use/rotation evidence (V5)"
    },
    {
      "id": "T05-R6",
      "severity": "P2",
      "status": "open",
      "summary": "credential page has no rendered browser evidence at 1440px/390px"
    },
    {
      "id": "T05-R7",
      "severity": "P2",
      "status": "open",
      "summary": "credential tenant_id stays 0 until business routes carry the tenant claim"
    },
    {
      "id": "T05-R8",
      "severity": "info",
      "status": "verified",
      "summary": "overlay component registry change is a single page registration at the documented extension point"
    }
  ],
  "verification": {
    "goCredentialScope": "PASS 6.58s (real MySQL, -race)",
    "goNeighbouringPaths": "PASS 48.68s",
    "frontendTypeCheck": "PASS",
    "frontendLint": "PASS (changed files)",
    "frontendBuild": "PASS 2.32s"
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-deploy-credential-ownership-and-ui/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-deploy-credential-ownership-and-ui.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-deploy-credential-ownership-and-ui/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-deploy-credential-ownership-and-ui/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
