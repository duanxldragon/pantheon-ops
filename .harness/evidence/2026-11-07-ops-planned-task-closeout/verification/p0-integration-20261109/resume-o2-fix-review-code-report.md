## Code Review Summary

**修复 delta：BLOCK / REQUEST CHANGES。Engine 计划：REVISION REQUIRED。**

审查范围：16 个变更文件及 ownership 计划。修复 delta 发现 **1 个 MEDIUM**；CRITICAL/HIGH/LOW 均为 0。既有 **CODE3 HIGH 保持未关闭**。

### 修复 delta

**[MEDIUM] 在途请求会丢弃后续 SSE 刷新，页面可能永久显示旧状态**

位置：[useDeployTaskSnapshot.ts:19](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/useDeployTaskSnapshot.ts:19)，关联 [DeployTaskDetail.tsx:944](D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:944)。

`session.running` 时直接返回，没有保留刷新需求。若详情请求读取了 `running`，随后完成事件触发的刷新被丢弃，旧响应仍设置 `stale=false`。SSE 保持连接时轮询停止，页面可能一直显示 `running`。

**证据：**执行当前 hook 的内存复现得到 `serverStatus=success`、`snapshotStatus=running`、`stale=false`、请求数仍为 2。

**最小修复：**在同一 generation 内记录待刷新标记，在途请求结束后执行一次合并刷新；保留 disposed/deny fencing。补充“完成事件到达在途请求期间”的回归。

其余指定修复已核对：日志授权先于 cap/firstError、host-step 关联、SSE 当前主机过滤与整任务撤销、有限序列分页、事务内 Retry 预算拒绝、paused 免费及历史递增、失败步骤导航、forbidden 清理、stale polling、既有 cancel/rollback draft 控件和两处 TS 修复。未发现新增掩盖根因的 fallback。

### Engine 计划

计划方向可继续推进，但尚不具备实施就绪条件：

- **共享主机租约必须同时续租。** [计划:12](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-engine-ownership-plan.md:12)只明确续五分钟 attempt；共享 host lease 为三十分钟，[过期后可被其他任务接管](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2971)。最长二十四小时执行仍可能跨任务重叠。应在当前 ownership fence 下同步续租现有两类 lease，并增加跨任务竞争回归。
- **明确所有投影的完整锁序。** [计划:11](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-engine-ownership-plan.md:11)只禁止 renewal 的 host→task；日志/事件的 [withTaskSequence](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1999)仍取得 task 锁。凡需 task sequence 的事务应先 task、后 host/attempt；同时明确 CMDB/shared lease 顺序，禁止检查后脱离事务写入或释放租约。
- **明确失联停止期限。** [计划:12](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-engine-ownership-plan.md:12)需规定数据库统一到期边界、独立 monotonic watchdog、数据库调用期限及 DB/ownership 失败后的取消与 join 行为。

两服务真实 MySQL blocked-executor、lost ownership、停续租恢复、旧回调及竞争测试是实施验收前提，当前没有这些成功证据。任意远端副作用仍存在不确定性，不支持 exactly-once 声明。

### 验证与身份

本地证据确认 targeted CGO race **21 PASS、零 SKIP/FAIL**；frontend **46/46**，本次重跑仍通过。已核对 build/lint/typecheck/vet 记录。原生 `gopls` 覆盖五个 Go 文件、TypeScript 覆盖九个 TS/TSX 文件，零类型错误；有三条非阻断提示。LSP/ast-grep 工具未提供。

最终 **16/16 变更、43/43 既有拥有路径匹配**；v9 request/receipt 保持原哈希。字面 `exact-hashes.json` 未找到，核对的是现有 `resume-o2-review-blockers-fix-hashes.json`：

- 哈希清单：`2478846b345e336369a5890645f5a42b898713e9b2ebc01089e89175fc49af25`
- Engine 计划：`a8acfef81ca4a33c6c0681e02f026807a6394a7e8dd570d813af47ace18a3649`

未写入任何文件。新源码 VM/browser、渲染、完整源、session/IAM/audit、正式消费及发布验收仍归 root；本次不关闭 O2/P0。
