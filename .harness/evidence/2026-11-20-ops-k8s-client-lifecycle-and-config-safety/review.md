# T06 Review — K8s 凭据导入、连接轮换与请求边界

## 结论

`approved with documented P2 follow-up`。连接池身份/并发/Evict、kubeconfig 嵌入凭据边界、QPS/Burst 节流、Pods 分页、凭据输入默认掩码有代码与渲染证据；真实集群凭据轮换（k13）与 k8s 页面/日志在 live 验收 17/17 PASS。仅断网/取消的网络分区探针保留为显式 gap。

## Findings（严重度从高到低）

1. **fixed — V8 真实集群轮换未在本候选重跑（原 P2）。**
   main `697c0771` 候选后端在 `115.190.251.143` K3s 上执行 k13：warm pool 后轮换为无效 token，version 递增且 `/nodes` 被拒；恢复原凭据后 version 再递增且 `/nodes` 返回真实节点。17/17 PASS。

2. **fixed — k8s 页面 smoke。**
   PR #207 CI Smoke Sanity（run 45308078162，17m30s）在隔离后端执行 `test:smoke:business:k8s` 绿。该套件覆盖集群/发布/服务路由、手机视口、Service 写前置条件与只读门禁，但不覆盖凭据表单；掩码仍以定向 Playwright 探针（`disc`→`none` + 截图）为渲染证据。

3. **P2 — 断网/取消仅有单测级覆盖。**
   provider 调用超时/取消的运行时证据（网络分区、挂起连接）未采集。

4. **info — Base 上游依赖。**
   `keyMatch2` 会把 `GET /packages/:id` 放宽到 `/export`，本包与 T03 均因此不绑定详情路由；需 Base 正式 release 的静态路由守卫，不在 Ops override。

## 变更面

- `backend/modules/business/k8s/pkg/{client_factory.go,client_factory_test.go,client_pool.go}`
- `backend/modules/business/k8s/workload/{workload_service.go,workload_handler.go,workload_dto.go}`
- `frontend/src/components/patterns/forms/{SecretTextArea.tsx,secret-text-area.css}`
- `frontend/src/modules/business/{k8s/cluster/ClusterForm.tsx,k8s/cluster/ClusterDetail.tsx,filedist/FiledistList.tsx,cmdb/host/CmdbHostDetail.tsx,cmdb/host/CmdbHostForm.tsx}`

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-k8s-client-lifecycle-and-config-safety",
  "reviewer": "primary agent self-review (no independent review on this candidate)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "client pool identity/eviction, embedded-only kubeconfig, QPS/Burst, pod pagination, credential masking (k8s/filedist/cmdb)",
  "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety",
  "summaryFile": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/review.md",
  "changeRef": "none (uncommitted worktree)",
  "findings": [
    {
      "id": "T06-R1",
      "severity": "P2",
      "status": "fixed",
      "summary": "live credential rotation re-run on candidate backend against real K3s",
      "verification": "k13 PASS in k8s-live run omx-accept-20261121-k8s38 (17/17)"
    },
    {
      "id": "T06-R2",
      "severity": "P2",
      "status": "fixed",
      "summary": "k8s browser smoke green in CI; suite does not cover the credential form",
      "verification": "PR #207 Smoke Sanity PASS 17m30s (run 45308078162)"
    },
    {
      "id": "T06-R3",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "offline/cancel evidence is unit-level only"
    },
    {
      "id": "T06-R4",
      "severity": "info",
      "status": "base-first-dependency",
      "summary": "Base keyMatch2 blocks detail-route bindings; upstream release needed"
    }
  ],
  "verification": {
    "k8sModuleTests": "11/11 packages ok",
    "poolTests": "6 PASS",
    "factoryThrottlePin": "PASS",
    "frontendTypeCheck": "PASS",
    "passwordToggleContract": "PASS (352 files)",
    "uiContract": "PASS (405 files)",
    "eslintChangedFiles": "PASS",
    "k8sBrowserSmoke": "CI Smoke Sanity PASS 17m30s (run 45308078162)",
    "liveClusterRotation": "k13 PASS, 17/17 total (omx-accept-20261121-k8s38)",
    "renderedMaskEvidence": [
      "k8s-cluster-kubeconfig-masked.png",
      "k8s-cluster-kubeconfig-revealed.png",
      "t06-mask-runtime.json",
      "cmdb-host-ssh-key-masked.png",
      "cmdb-host-ssh-key-revealed.png",
      "t06-cmdb-mask-runtime.json"
    ]
  },
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-k8s-client-lifecycle-and-config-safety.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
