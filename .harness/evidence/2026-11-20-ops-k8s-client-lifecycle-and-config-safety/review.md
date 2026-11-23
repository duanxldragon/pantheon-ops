# T06 Review — K8s 凭据导入、连接轮换与请求边界

## 结论

`approved with documented P2 follow-up`。实现覆盖本包四条验收路径中可离线验证的部分：连接池身份/并发/Evict、kubeconfig 嵌入凭据边界、QPS/Burst 节流、Pods 分页、凭据输入默认掩码；凭据掩码有渲染证据。真实集群轮换（V8）与浏览器 smoke 未在本候选重跑，作为显式 gap 保留，不代签。

## Findings（严重度从高到低）

1. **P2 — V8 真实集群轮换未在本候选重跑。**
   池级单测（`TestPool*`）证明缓存键含凭据哈希、换凭据不复用旧 clientset、并发不复活；但 live 集群的令牌撤销时序未验证。风险：真实网络/服务端缓存层行为可能绕开客户端假设。

2. **P2 — `npm run test:smoke:business:k8s` 本会话未执行。**
   前端 dev server 被 harness 因系统内存不足回收，按 harness 指示未自行重启；掩码 UI 以定向 Playwright 探针取证（`disc`→`none` + 截图），但不等于整套 k8s 页面 smoke。

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
    {"id": "T06-R1", "severity": "P2", "status": "accepted-gap", "summary": "V8 live credential rotation not re-run on this candidate; pool unit tests cover cache identity"},
    {"id": "T06-R2", "severity": "P2", "status": "accepted-gap", "summary": "k8s browser smoke suite not run; frontend dev server reaped for low memory"},
    {"id": "T06-R3", "severity": "P2", "status": "accepted-gap", "summary": "offline/cancel evidence is unit-level only"},
    {"id": "T06-R4", "severity": "info", "status": "base-first-dependency", "summary": "Base keyMatch2 blocks detail-route bindings; upstream release needed"}
  ],
  "verification": {
    "k8sModuleTests": "11/11 packages ok",
    "poolTests": "6 PASS",
    "factoryThrottlePin": "PASS",
    "frontendTypeCheck": "PASS",
    "passwordToggleContract": "PASS (352 files)",
    "uiContract": "PASS (405 files)",
    "eslintChangedFiles": "PASS",
    "renderedMaskEvidence": ["k8s-cluster-kubeconfig-masked.png", "k8s-cluster-kubeconfig-revealed.png", "t06-mask-runtime.json", "cmdb-host-ssh-key-masked.png", "cmdb-host-ssh-key-revealed.png", "t06-cmdb-mask-runtime.json"]
  },
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-11-20-ops-k8s-client-lifecycle-and-config-safety.task.md",
    "evidence": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-11-20-ops-review-remediation/plan.md"]
  }
}
```
