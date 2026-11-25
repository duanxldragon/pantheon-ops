# T06 Review — K8s 资源读写授权与变更准入

## 结论

`pass-with-explicit-gaps`（自审，无独立 reviewer）。变更分派缺陷已红→绿锁定；跨命名空间读授权与读路由绑定已在代码级完成并有真实 MySQL 回归；live K3s 的 k14 namespace ownership/写保护与 k8s 页面 CI smoke 已绿；浏览器侧草稿恢复已由 2026-11-24 探针补验；仅剩 Base 路由守卫发布后的二次绑定为显式 gap。

## Findings（严重度从高到低）

1. **fixed — P1 NamespaceAll 聚合读授权缺失。**
   `NamespaceService.ReadableNamespaces` 按 Cluster 的 `business_scope_id` + `environment` 取可见 namespace 集合；workload/pod/configmap/secret/service/ingress/storage PVC 的聚合列表按集合过滤，空集合直接返回空响应，显式 namespace 读先过 `RequireReadForClusterWithContext`。`TestReadableNamespacesFollowsClusterOwnership` 覆盖跨 scope / 跨 environment / scope=0 三种反例。

2. **fixed — P1 读路由权限绑定不完整。**
   `business:k8s` 现注册 cluster/namespace/workload/configmap/secret/service/release/ingress/storage/backup 的读与动作路由；`TestK8sPermissionRoutesMatchActualActions` 校验每条声明路由真实存在且方法与 action 语义一致。凭据/Helm/install 及静态-参数同层冲突路由保持未绑定（默认拒绝）。

3. **fixed — P2 真实 resourceVersion 冲突的 UI 草稿恢复。**
   2026-11-24 浏览器探针在真实 K3s 上用 kubectl annotate 制造并发写（rv 2543724→2543736），UI 提交收到真实 409；编辑器保持打开、草稿选择器未丢、冲突提示可见；关闭并显式刷新后以新 rv 重试 200，独立 kubectl 读到 `uiRevision=1`。证据 `browser-20261124-153621/probe.json`。

4. **fixed — DaemonSet 误改同名 Deployment。**
   `restartWorkload` 现按 kind 分派到 DaemonSets/StatefulSets/Deployments；`scaleWorkload` 拒绝 DaemonSet 与未知 kind。测试红→绿（先写测试复现 patch 落到 Deployment，后改实现）。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-k8s-resource-authorization-and-mutations",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "workload mutation dispatch, namespace read authorization, k8s permission route bindings, resourceVersion conflict mapping",
  "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations",
  "summaryFile": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {
      "id": "T07-R1",
      "severity": "P1",
      "status": "fixed",
      "summary": "NamespaceAll cross-namespace read aggregation lacked ownership filtering",
      "verification": "TestReadableNamespacesFollowsClusterOwnership PASS; k8s 12/12 packages ok"
    },
    {
      "id": "T07-R2",
      "severity": "P1",
      "status": "fixed",
      "summary": "read-route permission bindings incomplete",
      "verification": "TestK8sPermissionRoutesMatchActualActions PASS; IAM permission/role/tenant suites PASS"
    },
    {
      "id": "T07-R3",
      "severity": "P2",
      "status": "fixed",
      "summary": "browser draft recovery under real resourceVersion conflict",
      "verification": "browser probe omx-t09-browser-20261124-153621: kubectl annotate bump, UI 409, draft preserved, refresh retry 200, provider selector uiRevision=1"
    },
    {
      "id": "T07-R4",
      "severity": "fixed",
      "status": "fixed",
      "summary": "DaemonSet restart patched same-name Deployment; scale accepted unknown kinds",
      "verification": "TestRestartWorkloadPatchesNamedKindOnly + TestScaleWorkloadRejectsDaemonSetAndUnknownKind PASS"
    }
  ],
  "verification": {
    "targetedWorkloadTests": "3/3 PASS",
    "workloadPackage": "ok",
    "k8sModuleTests": "12/12 packages ok (real MySQL)",
    "iamSuites": "permission 31.083s ok; role 49.642s ok; tenant 121.462s ok",
    "browserConflictProbe": "PASS (omx-t09-browser-20261124-153621; real kubectl bump, 409, draft preserved, refresh retry 200)"
  },
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations.task.md",
    "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations/manifest.json",
    "evidence": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-resource-authorization-and-mutations/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
