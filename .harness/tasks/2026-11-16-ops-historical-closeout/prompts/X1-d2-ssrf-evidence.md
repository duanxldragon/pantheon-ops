# Prompt X1 — D2 出站 SSRF 验收证据（missing-verification → verified）

> 派发模型：由执行 Claude 通过 `codex exec ... -C pantheon-ops` 实施；本仓库 Claude 为 planner/reviewer。

## 当前事实（对账核实）

SSRF 代码已在（`backend/pkg/security/ssrf/validator.go`：Default/StrictValidator、BlockPrivateIPs、DNS-rebinding、MaxRedirects、allowlist）。该目录是 **base-shared 路径**，非 business-overlay 违规；`datasource_service.go` 经 `pantheon-base/backend/pkg/security/ssrf` 在 create/query 校验。但：

- `2026-09-29-ops-ssrf-protection-integration` manifest 自 report done 时附"byte-for-byte copy-from-Base unblock compilation"的 Base-first 潜在问题，且**无 `.harness/evidence/...` 验收证据目录**。
- 缺"实际部署的 allowlist/private-network/redirect/DNS 规则、审计、完整负向验收"。

## 范围（In / Out）

- In：给 D2 写验收证据包——对端到端路径（datasource 代理、notification webhook、ArgoCD client、K8s API、SMTP）跑 allowed/denied/redirect/private-address/rebinding/timeout 用例；负向（`127.0.0.1` / `169.254.169.254` 创建 datasource 必须失败）；出审计说明。
- Out：新写 SSRF 实现、复制 Base 源码到 business overlay、绕权限。

## 精确文件所有权

- `backend/modules/business/observability/{datasource_service,notification_service}.go`（若需小改/测试面）
- `backend/modules/business/cicd/argocd/*`（只读/测试面）
- `backend/pkg/security/ssrf/validator.go`（只读；归属 base-shared，确认无 business patch）
- 新建 `.harness/evidence/2026-09-29-ops-ssrf-protection-integration/{commands.json,summary.md,review.md,D2-ssrf-acceptance.md}`
- 更新 `.harness/evidence/.../D2-ssrf-network-policy-analysis.md` 链

## 约束

- **维护者策略决定前置**：出站策略 Option A/B/C 需维护者/人批准；未获决定则本批产出为"策略待决" + 代码面负向测试，报 external-gate，不谎报。
- Base-first：不把 base-shared `ssrf` 包拷进 business overlay；若确认有 business patch 则回 Base。
- 不触碰 `foundation-release.lock.json`；不提交/push/发布；commands 只记真实运行；**不得回显任何端点/凭证值，只报可用/不可用与 owner**。

## 真实验收

- 负向用例（私有/云元数据地址被拒）在部署路径实跑通过；策略号与 allowlist 配置有依据；审计无 secret。
- 若策略/端点未决：记录 external-gate 及真实前置。
