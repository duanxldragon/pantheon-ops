## Summary

**独立 ARCH 判定：修复 delta 为 `BLOCK`；引擎方案为 `REVISION REQUIRED`。** 当前 16 文件哈希稳定，但只读内存复现发现一个新的刷新遗漏；CODE3 仍为未修复的 HIGH，不批准 O2/P0 收口。

## Analysis

**[MEDIUM，delta 阻断] 最后一次 SSE 刷新可能丢失。** [useDeployTaskSnapshot.ts:19](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/useDeployTaskSnapshot.ts:19) 在请求进行中直接退出，没有记录待刷新意图；旧请求随后将 `stale` 清为 false。若终态事件落在该窗口，[DeployTaskDetail.tsx:944](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:944) 又因 SSE connected 停止轮询，页面可以一直显示旧 running 状态。只读 React/JSDOM 内存复现得到：`requests=2, retainedStatus=running, stale=false, trailingRefreshStarted=false`。现有[合并请求测试:71](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/tests/unit/business/operational/deploy-read-revocation.test.tsx:71)只检查没有重复请求，未检查末次刷新最终执行。

其余指定修复在源码与局部回归层面成立：

- 当前主机交集、host-step 关联在日志 cap/firstError 前处理；事件页有限，整任务撤销发送 forbidden 并退出：[read_scope.go:31](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/read_scope.go:31)、[deploy_handler.go:855](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_handler.go:855)。
- Retry 事务内、写状态前检查累计非 paused 预算；失败导航使用实际失败步骤自身 attempt：[deploy_service.go:1862](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1862)、[DeployExecutionWorkbench.tsx:89](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployExecutionWorkbench.tsx:89)。
- 撤销清空与 generation fencing、支持的 cancel/rollback 控件及两处 TS 修正已有对应实现：[snapshot:25](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/useDeployTaskSnapshot.ts:25)、[detail:550](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:550)、[detail:1286](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:1286)、[cursor:89](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/tests/unit/business/operational/deploy-log-cursor.test.tsx:89)、[fixture:43](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/tests/smoke/business/deploy/deploy-pages.spec.ts:43)。

## Root Cause

刷新合并没有保留“请求期间发生变化”的意图。引擎方案方向正确，但尚未把全部锁、双租约和失权后的写回路径定义成可直接实现的不变量。

## Recommendations

1. **小工作量，解除 delta 阻断：**增加 generation 内的 pending-refresh 标记；请求结束后执行一次后续刷新，deny/dispose 清除标记。补“终态事件发生于旧请求期间”的回归，保留现有禁止并发请求与撤销 fencing。
2. **引擎方案最小修订：**
   - **锁序：**日志/事件的 [withTaskSequence:1999](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1999)会锁 task；不能在持有 host 锁后调用它，与 task→host 控制事务倒序。需要 task 的写路径先锁 task；纯 renewal 保持 host-only。CMDB 写回须通过已有事务绑定扩展，与 fence 同事务提交，避免检查后另写：[CMDB:139](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cmdb/capability.go:139)。
   - **双租约：**明确续期五分钟 attempt **和**三十分钟物理主机租约，否则长任务仍可被另一 task 接管：[service:133](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:133)、[service:2971](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2971)。保持当前 attemptID/workerID、最新 attempt、running、非空且未过期条件。
   - **失权：**确定 DB 时钟边界、保守本地单调截止时间及有界 DB 调用；最小策略是首次 DB/ownership 失败立即取消。所有持久化点必须 fence，包括 [requeue:1490](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1490)和最终任务状态；旧 callback 不得转成失败写回或再次排队。

[方案要求的两服务真实 MySQL、阻塞执行器、失权、停止续期及竞争回归:20](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-engine-ownership-plan.md:20)仍是实现批准前提，尚非成功证据。保留预算、历史和现有恢复合同；任意远端脚本副作用仍不确定，不作 exactly-once 或逆操作保证。

## Trade-offs

| 选择             | 收益             | 代价                 |
| ---------------- | ---------------- | -------------------- |
| 合并后补一次刷新 | 保证末次变化可见 | 增加一次请求         |
| DB 失败立即停止  | 最小失权边界     | 短暂故障也会中断执行 |

## References / Validation

已核对真实 MySQL targeted race **21 PASS、零 SKIP**及前端 **46 PASS**日志；CGO、build/lint/typecheck/vet 的退出记录见[交接:29](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-review-blockers-fix-handoff.md:29)。它们属于局部本地验证。

末次核对：**16/16 修复文件、43/43 未改所有权文件及 v9 请求/receipt 均匹配**。`exact-hashes.json` 未找到，实际核对的是命名修复清单：

- `resume-o2-review-blockers-fix-hashes.json`：`2478846b345e336369a5890645f5a42b898713e9b2ebc01089e89175fc49af25`
- `useDeployTaskSnapshot.ts`：`cb1a487107711798af476edfbe5880b09496e75a8ccb0fc361e3ccc1ba9abbdd`
- `deploy_service.go`：`7b4695ff601636e905ef3f6903df15b48d1f23fd4c5f2a33282c6c281670b9a7`

未写文件。按分工未产出新浏览器渲染；新源码 VM/browser、全树验证、Base 消费及 session/IAM/audit 验收继续由 root 持有。诊断证据已足够，停止于整改交接。
