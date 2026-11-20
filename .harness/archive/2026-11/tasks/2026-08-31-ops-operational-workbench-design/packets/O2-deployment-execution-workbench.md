# O2 Deployment Execution Workbench

- Priority: `P0`
- Layer: `business/deploy`
- Status: `blocked`
- Base Dependency: B1 sticky actions + B3 `ExecutionStepRail` and `TaskLogViewer`
- Business Dependency: immutable task/host/attempt/step/event contracts

## Outcome

把部署任务详情升级为可快速定位目标、步骤、日志和恢复动作的执行工作台。

## In Scope

- 任务 hero、状态/新鲜度、目标列表、步骤轨、日志主区和审计/参数抽屉。
- host/attempt/step/stream selection 与 URL/deep-link 状态。
- SSE last-sequence reconnect、stale/polling fallback 和 partial failure。
- permission/state-aware pause/resume/cancel/retry/rollback entry points。

## Out Of Scope

- 不在前端推导状态机或伪造运行进度。
- 不改变 Deploy 引擎本身的幂等、租约、回写和恢复模型。
- 不显示明文 credential、秘密参数或未脱敏命令。

## Expected Files

- `frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx` 及拆分的业务 composition。
- Deploy API/event adapters、i18n、smoke/Playwright。
- 后端只补现有 task event/log DTO 的必要字段与测试。

## Acceptance

1. 用户能从失败汇总在两次交互内定位 host + step + first error log。
2. SSE 断开后保留内容并从 last sequence 恢复；fallback 显式 stale。
3. 动作严格匹配后端状态、权限、DataScope 和 state version；冲突要求刷新。
4. mobile 首屏优先显示状态和失败目标，目标/步骤/日志切换无内容遮挡。
5. 多主机、长日志和实时更新不引起布局跳动或无限 DOM 增长。

## Evidence

- backend：event authorization、sequence reconnect、redaction、state transition and idempotency tests。
- frontend：target/step/log selection、action matrix、stale/conflict/partial failure tests。
- browser：happy path、one-host failure、multi-host partial、disconnect/reconnect、forbidden。
- runtime：脱敏 SSE/log 片段；无真实 VM/credential 时记录明确 gap。
- screenshots：desktop/mobile x running/failed/stale/forbidden x light/dark。

## Gates

- Base lock、Deploy public event DTO、状态机/权限变化、真实主机动作和最终视觉验收。
