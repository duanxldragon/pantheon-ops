## Summary

**ARCH status：`BLOCK`，仅针对本次 O2 独立架构审查。** 当前源码存在日志/事件授权粒度不一致，以及重试成功受理后无法取得执行 attempt 的路径；v9 的 12/12 成功不能消除这些问题。建议先修复业务域的授权与恢复准入，再复审；不批准整个 O2/P0，也不涉及合并或发布。

## Analysis

### 1. 阻断：任务可见不等于所有主机日志可见

任务可见性只要求 **任一目标主机可见**：[deploy_service.go:2607](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2607)、[deploy_service.go:2644](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2644)。详情已过滤当前 CMDB 可见主机，但日志接口只检查任务可见性，随后按任务 ID 和客户端提交的 host/step 条件读取：[deploy_service.go:715](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:715)、[deploy_handler.go:766](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_handler.go:766)、[deploy_service.go:2072](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2072)。

因此，混合 DataScope 任务中，可见主机 A 足以通过入口检查；省略 `hostId` 或指定不可见主机 B，仍能读取 B 的输出。step/host 关联过滤验证的是记录归属，**没有验证调用者权限**。

现有源隔离测试使用 admin；负向权限测试覆盖整个任务不可见，未覆盖上述混合场景：[workbench_test.go:179](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/workbench_test.go:179)、[workbench_test.go:219](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/workbench_test.go:219)。这违反详情与日志流 DataScope 一致性的现有要求：[BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md:190](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md:190)。

**最小修补：**通过现有 CMDB reader 得到当前可见任务主机集合，在日志查询的排序、500 条上限和 first-error 查询之前求交；不能靠前端隐藏主机完成授权。

### 2. 阻断：已建立 SSE 不重新检查当前主机可见性

SSE 只在连接入口检查权限，后续每秒直接读取全部任务事件，并序列化完整事件：[deploy_handler.go:833](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_handler.go:833)、[deploy_handler.go:852](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_handler.go:852)。因此，连接建立后发生主机归属撤销，原连接仍可收到新事件。

UI 又把 silent refresh 的错误吞掉，保留旧 task；只要 SSE 仍 connected，就停止详情 fallback polling：[DeployTaskDetail.tsx:925](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:925)、[DeployTaskDetail.tsx:949](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:949)。这会使旧详情继续表现为 live。

**最小修补：**每次事件批次读取前重新检查当前 CMDB 可见性，过滤 host/step 事件；撤销后终止流。UI 收到 forbidden 时清除详情及冻结快照，普通刷新失败则显式 stale。这里无需增加新的系统域会话模型。

### 3. 阻断：耗尽 attempt 后，Retry 可以返回成功但永久停在 running

worker 的累计非 paused attempt 上限为 3；耗尽时 `claimAttempt` 返回 nil，而调用者直接退出：[deploy_service.go:1638](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1638)、[deploy_service.go:1423](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1423)。

但 Retry 准入只检查 failed、当前 scope/host，然后把 task 和失败主机改为 running；没有检查剩余 attempt：[deploy_service.go:1867](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1867)、[deploy_service.go:1891](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1891)。主机仍 running 时，终态计算返回空值：[deploy_service.go:2695](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2695)。

**最小修补：**在控制事务提交前检查执行预算；耗尽时拒绝并保持 failed，提供明确错误。不要静默清空历史或取消三次限制。若业务确实需要开启新的重试周期，应另行明确预算语义。

### 4. WATCH：租约、远端取消与持久化边界仍有风险

- **租约不能证明旧执行已经停止。** attempt 租约固定五分钟，任务默认超时三十分钟、最大二十四小时；过期调和直接允许后续 claim：[deploy_service.go:1624](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1624)、[deploy_service.go:47](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:47)、[deploy_service.go:1661](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1661)。checkpoint 更新未校验当前 worker/attempt 所有权：[deploy_service.go:3276](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:3276)。另一个进程在旧执行仍存活时调和，可产生重叠执行和旧回写。Git HEAD 已存在该租约模型；它不是本次 UI 引入的新问题，但需要向上报告执行边界评审。
- **取消证明本地连接结束，不能证明远端副作用撤销。** 当前关闭 SSH connection 并等待输出复制结束，修复有效：[deploy_service.go:2974](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2974)。普通 script 可直接重新执行，组件路径才有健康检查判定：[deploy_service.go:3521](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:3521)、[deploy_service.go:3546](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:3546)。不能据此宣称任意脚本 exactly-once 或取消无残余副作用。
- **checkpoint/history 原子提交成立，event/log 尚不具备同等保证。** 历史与 checkpoint 同事务：[deploy_service.go:3287](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:3287)。`task.started` 在 claim 提交后另写，step event 也单独追加；日志写入失败只记录服务器日志：[deploy_service.go:1144](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1144)、[deploy_service.go:3218](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:3218)、[deploy_service.go:2040](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2040)。成功 checkpoint 可以缺少相应事件或输出，UI 不能把日志空白解释为执行无输出。

最小方向是保留现有模型，补齐准入、所有权保护和缺失投影提示；完整 lease/fencing 修订应走明确的执行引擎评审，不能借 O2 工作台顺带扩展。

### 5. 当前成立的边界与剩余验收缺口

控制命令先检查可见性，再比较 opaque stateVersion；task/host/command event 同事务，worker effects 在提交后执行：[task_command.go:175](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/task_command.go:175)、[task_command.go:234](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/task_command.go:234)。手动恢复/重试不会启动 SSH。执行计划来自 frozen snapshot，而不是当前 package/template：[deploy_service.go:3667](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:3667)。这些设计应保留。

仍不能扩大以下证明范围：

- source/host ownership 在每条 SSH command 前重读，但 credential 在 worker 启动时解析并复用；不是每条命令实时撤销检查：[task_command.go:122](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/task_command.go:122)、[deploy_service.go:1368](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1368)、[deploy_service.go:1574](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1574)。v9 credential cases证明的是暂停后 recovered execution 被拒绝：[v9 matrix:2839](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v9-matrix-result-1815605de83a.json:2839)。
- UI Cancel 只允许 pending/running，后端还允许 paused/pause_requested；详情动作区缺少 rollback 入口：[DeployTaskDetail.tsx:542](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:542)、[deploy_service.go:1721](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1721)、[DeployTaskDetail.tsx:442](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:442)。只应暴露后端已支持的 rollback；复杂脚本/DAG inverse 被明确拒绝，是能力边界：[task_rollback_snapshot.go:24](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/task_rollback_snapshot.go:24)。
- 正式 Base 消费仍锁定 v0.18.6。当前 rail 与该正式快照哈希一致，仍直接显示 raw status，CSS 未给步骤按钮定义主题表面：[ExecutionStepRail.tsx:49](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/components/operational/ExecutionStepRail.tsx:49)、[operational.css:177](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/components/operational/operational.css:177)。已查看历史 v16 desktop/mobile dark 渲染，白色 rail 和难辨状态确实可见；没有产生当前源码完整视觉矩阵，不能给 UI polish 批准。共享修复必须 Base-first、正式发布后消费：[PROJECT_INHERITANCE.md:9](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/PROJECT_INHERITANCE.md:9)。

## Root Cause

**业务授权和执行的粒度没有贯通。** 任务级“任一主机可见”被日志/事件当成整任务授权；控制事务的“可以恢复”又没有包含 worker 的 attempt 预算与所有权条件。新增详情过滤、版本控制和历史事务修复了各自路径，却没有把同一不变量延伸到所有读取、派发和回写边界。

## Recommendations

1. **优先修复日志与 SSE 的当前主机授权交集** — 中等工作量 — 消除越权；补混合 DataScope、已连接后撤销的回归证据。
2. **事务内拒绝不可执行的 Retry，并修正 UI action matrix** — 小至中等 — 避免成功受理后永久 running；保留现有预算和历史。
3. **上报租约/旧 worker 回写与远端不确定结果边界** — 中等以上 — 单独评审，避免在 O2 内隐式改变引擎模型。
4. **完成正式 Base 修复消费及当前源码验证** — 跨仓库 — 保留原五项验收，不以已有 VM 或历史截图替代。

## Architectural Status

`BLOCK`

## Trade-offs

| 方向                                  | 收益                   | 代价                                       |
| ------------------------------------- | ---------------------- | ------------------------------------------ |
| 保持部分任务可见，逐主机过滤日志/事件 | 保留混合任务的操作价值 | 查询与游标必须在授权过滤后处理             |
| 任一不可见主机即拒绝整任务            | 授权实现较简单         | 改变现有部分可见语义，不宜作为默认修补     |
| 耗尽预算时拒绝 Retry                  | 最小修补，保留执行限制 | 用户需要明确的后续处理路径                 |
| 自动重置预算、扩大恢复能力            | 更多自动恢复           | 改变引擎合同与副作用风险，超出 O2 最小修补 |

**最强批准反论：**v9 已真实验证成功、失败重试、暂停、取消、并发、撤销和冷启动，且源码一致；新增控制和历史事务也有明确保护。这个论据足以认可已覆盖场景，但未覆盖混合主机授权、既有 SSE 撤销、耗尽预算后的 Retry，以及仍存活 worker 的租约过期，所以不能据此批准当前 ARCH lane。

## References / Source identity

最终只读核对：reviewed request 的 **185/185 编译源 SHA-256 全部匹配，零差异**。主要当前哈希如下：

| 源文件                         | SHA-256                                                            |
| ------------------------------ | ------------------------------------------------------------------ |
| `deploy_service.go`            | `8ea2ee60bb99d9a2577bf342ac7e826953670db04e4aa3c8cfb399383f7efedc` |
| `deploy_handler.go`            | `36c81d49130dc181de52326d6a03b119db700229ab3608a9639b23fcc6ac6fff` |
| `task_command.go`              | `e256db4c85400efe8d44cc87e062f486e77221c2daa4e8d8b417b4d3cfc683ac` |
| `step_history.go`              | `6bbfbc6249f2afef10e7f2e3ba6bb333b318921ab2af390ccdbec7772a2dc441` |
| `cmdb/capability.go`           | `87da5f9bde5e3f69022617cf6c92b7e4a4c0bdfee106a6876e511c925c16f566` |
| `DeployTaskDetail.tsx`         | `1ea8b443bfe32f86a55ab15f3c0adfca44b3071c5456b28920f2942b6567f4f3` |
| `DeployExecutionWorkbench.tsx` | `9356ccd861ba5b10e7f0bf58b58f489d14634be06c40d64cbddf109137f73286` |
| `useDeployTaskLogs.ts`         | `9fc4b84175c54cd43a98dae50784eb23c59a6af87549ba8531f79811908da957` |
| `useDeployTaskStream.ts`       | `7043751fd26ae00910d41914b7e8b99d441f89e099b8d729cd19911dfabcc227` |

v9 汇总确认 12/12、152 正向、31 负向、零失败、`sourceStable=true`，并明确仅属 generator evidence：[matrix:4781](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v9-matrix-result-1815605de83a.json:4781)。

历史 fullrace 的 service/command 哈希分别为 `95c27e19…`、`008241a4…`，与当前不同：[race hashes](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continuation-full-race-hashes-after.json)。**不将该 race 提升为当前树最终验证。**

本次未编辑文件、未读取运行态凭据、未运行 VM/凭据测试、未委派。诊断已达到源码证据停止条件；独立 CODE lane 仍保持分离。
