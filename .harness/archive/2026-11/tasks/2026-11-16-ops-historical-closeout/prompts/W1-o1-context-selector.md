# Prompt W1 — O1 统一上下文/目标选择器（真正缺实现）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。
> 原始接受定义：`docs/harness/tasks/2026-08-31-ops-operational-workbench-design` 的 `packets/O1-context-and-target-selector.md`。这是全量对账中真正缺实现的工作台项之一。

## 当前事实

仅存在 **Deploy 专用** target selector（`frontend/src/modules/business/deploy/task/DeployTargetSelector.tsx` + `backend/modules/business/{deploy,cmdb}/target_selection.go`，CMDB capability 驱动、预算上限 10000、去重/校验）。**没有**跨模块统一的 BizScope 环境上下文头 + 通用 context/target 选择器（`frontend/src/modules/business/shared/` 现只有 BusinessPageHeader/OperationalLogPanel/dashboardWidgets）。

## 范围（In / Out）

- In：构建共享 BizScope 环境上下文头 + 通用 context/target 选择器适配 CMDB capability；支持手动 IP/ID 输入 + 解析/去重/匹配 + unknown/permission/status 反馈；selected/excluded/invalid 汇总 + 目标快照 + 执行前服务端重校验；10k 候选有界；全键盘桌面 + 移动端；来源分布/排除/解析审计（无 secret）。
- Out：重写现有 DeployTaskSelector；改动 platform 系统域选择器底座；无边界的新领域选择。

## 精确文件所有权（与 C02/C03 重叠协调）

- `frontend/src/modules/business/bizscope/*`（上下文头）
- `frontend/src/modules/business/shared/*`（通用选择器；复用现有封装）
- `backend/modules/business/cmdb/target_selection.go`（复用 capability，最小扩点）
- `.harness/{tasks,evidence}/` 对应 O1 关闭链

## 硬约束

- **Base-first 前置（先验证）**：基础 `B3 ContextSelector` 是否已在当前 v0.18.9 中可用。若不可用，则记录 `blocked-on-upstream` 前置，不本地绕过实现。
- 复用现 `WithDataScope`/capability，不新增重复模型；不放宽 10000 有界；不做 secret/明文命令展示。
- 遵守 `../pantheon-base/DESIGN.md` 与 `impeccable`（此处为 UI，需渲染证据，或记录未产出证据原因）。
- 不提交/push/发布；commands.json 只记真实运行。

## 真实验收

- O1 packet 各条（含 10k 有界、全键盘、移动端、审计）逐项达成；`npm run type-check`、`go test ./modules/business/bizscope ./modules/business/cmdb` 通过；Playwright 覆盖拓扑/分组/手动输入/移动端。
- 若被 upstream B3 阻塞：记录真实前置 + Base link，不谎报完成。
