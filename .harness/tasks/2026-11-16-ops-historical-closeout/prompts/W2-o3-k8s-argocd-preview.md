# Prompt W2 — O3 K8s/ArgoCD 变更预览（真正缺实现）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。
> 原始接受定义：`packets/O3-change-preview-and-risk.md`。

## 当前事实

Deploy 路径已有预览（`DeployTaskPreview.tsx` + `task_preview.go`/`task_preview_plan.go`/`task_rollback_snapshot.go`）。但 **K8s Release 无 manifest/image/config 预览，ArgoCD 无 sync/rollback 版本树资源 diff 预览**（k8s 模块现只有 WorkloadList/ClusterList 等，无 preview）。

## 范围（In / Out）

- In：为 K8s Release 与 ArgoCD sync/rollback 补变更预览 + 风险汇总（reversible/health/rollback），与现有 Deploy 预览同契约（intent/current/target/objects/diff/risk）；可分辨 add/modify/delete/conflict/unmodified；版本漂移返回 409 conflict；secret 脱敏；大 diff 可折叠/搜索；移动端 review/cancel。
- Out：重写 Deploy 预览；改动 platform 底座 diff 组件；无边界新范围。

## 精确文件所有权

- `frontend/src/modules/business/k8s/release/*`（新建预览）
- `frontend/src/modules/business/cicd/*`（ArgoCD 版本树/resource diff）
- `backend/modules/business/deploy/task_preview.go`（复用契约/模型；最小扩点承接跨域）
- `.harness/{tasks,evidence}/` 对应 O3 关闭链

## 硬约束

- **Base-first 前置（先验证）**：基础 `B3 ChangeDiff` 是否在 v0.18.9 可用；若缺则记录 `blocked-on-upstream`。
- **ArgoCD live 端点**：预览应基于真实端点能力；若端点/凭证未授予，先把代码面与意图契约落地 + 单元测试，真实验收按 D1 一起门禁，不谎报"已在活环境通过"。
- 与计划 C02/E02 重叠协调：本批只做历史缺口，不重复 Deploy 预览。
- `impeccable`（UI）+ 渲染证据。
- 不提交/push/发布。

## 真实验收

- K8s + ArgoCD preview 达成 O3 packet；`go test ./modules/business/deploy -run Preview`、Playwright preview→confirm→conflict 通过；secrets 脱敏验证。
- 活环境/端点相关明确标注 external-gate 状态。
