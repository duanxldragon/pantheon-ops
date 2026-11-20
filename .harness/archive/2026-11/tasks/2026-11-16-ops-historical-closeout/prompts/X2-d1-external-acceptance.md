# Prompt X2 — D1 外部集成真实验收（gate-prepared，端点授予后才执行）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。
> **这是外部 gate 批**：仅当维护者已授予 scoped 端点/凭证与有限窗口才可运行。未授予则**不运行、不假报完成**，本提示词作为 ready-to-run 清单保留。

## 当前事实（对账核实）

截至本轮，真实验收仅有 SSH deploy（单 Ubuntu 主机 `115.190.251.143`）+ 隔离单节点 K3s v1.34.5（2026-08-28 r1/r2）。**无**对 Prometheus/VictoriaMetrics、Loki、Tempo、K8s Service CRUD、ArgoCD、Velero 的活环境验收证据。`datasource_live_acceptance_test.go` 仅在 env 设置时跑且只测 deploy-test host，不覆盖这六系统。

## 范围（D1 矩阵，紧密对齐原始 packet）

- Prometheus/VM：query + timeout。
- Loki：log query + failure 路径。
- Tempo：trace query + failure。
- K8s：Service CRUD 含 resourceVersion/conflict。
- ArgoCD：health/sync/rollback。
- Velero：backup/restore/drill。
- 每项补 permission/DataScope/audit/cleanup/idempotency 证据。

## 精确文件所有权（运行测试 + 证据）

- `backend/modules/business/observability/{datasource_live_acceptance_test.go,datasource_*}`
- `backend/modules/business/k8s/cluster/*`（K8s Service CRUD/conflict）
- `backend/modules/business/cicd/argocd/*`（ArgoCD）
- `v1/2` real-material 证据、velero 面
- 新建/更新 `.harness/evidence/2026-08-28-v1-real-acceptance-*` 或新 D1 证据目录：`commands.json`、逐系统 summary、`review.md`、redacted 结果

## 硬约束

- **无 mocks/无截图**；需真实接口调用；不存储/回显任何凭证、kubeconfig 内容，只在报告中标"可用/不可用 + owner"。
- 每个系统含失败路径与回滚/清理；idempotency。
- 若某系统端点未授予：该条目标 `external-gate` + 真实前置（无端点即不宣称覆盖）。
- 遵守 Windows race 门禁 + `impeccable`（不适用 UI 时说明）。

## 真实验收（Codex）

- 逐系统有 redacted 真实验证 + 维护者/租户侧 review；cleanup 证据。
- 未覆盖系统保留 external-gate，不并入"全完成"。
