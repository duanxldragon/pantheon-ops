# 当前基线核实与业务边界决策

- Status: in-progress
- Updated: 2026-11-23
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc（规划基线；来源报告 444ce8e）
- Owner: 业务规划负责人（Claude 执行；Codex 暂不可用）
- Priority: P1
- Activation: core
- Dependencies: none

## 首轮执行（2026-11-23）：4 条来源发现逐条判定 + 业务表四分类冻结

在当前 HEAD 对 T01 承接收的 4 条发现（ARC-001 / NEW-02 / NEW-03 / SUP-08）逐条核实，结果已写入父计划 `FINDINGS.json`（status=verified-at-head）与下述证据：

- **ARC-001（仍成立）**：遍历 51 个业务表模型，仅 observability（datasource_model.go + query_workspace_model.go 两张文件，覆盖 5–6 张表）含 `TenantID`；核心业务表（cmdb/host、k8s/cluster、deploy 全部、bizscope、service）**均无租户字段**。全局唯一键批量比对完成，列出 12 组需建租户内变体的唯一键（含报告点名的 `uk_deploy_credential_name_deleted` 仅 name 全局唯一）。→ T04 分配迁移 **000041**，T14 分配 **000042**（存量业务迁移已见 34/35/36/39/40，不冲突）。
- **NEW-02（仍成立，文档-实现偏差）**：业务代码零资产关系实体（Pod→Node→VM），仅 `cluster_service.go:854` Node capacity 元数据读取。→ 登记为独立后续能力，待维护者最终范围验收。
- **NEW-03（设计如此，非缺陷）**：`k8s/helm/helm.go` 包注释自明"DB-snapshot records / registration snapshot"——HelmChart 只登记不真装。口径保留（E03 登记态），真实 Chart 安装归独立后续范围。
- **SUP-08（维持 P2 降级）**：plan.md 已定"授权模型/功能缺口 P2"；T03 承接逐条权限绑定；T16 受限角色实测后再定条件性 P1 是否成立。

**冻结的规划假设**（T01 验收标准 3 项，待维护者终验）：凭据归属=按资源归属授权方案；Agent 无执行协议时关闭入口并拒绝旧数据启动；资产关系/Helm 真实安装/完整 Agent 协议均登记为独立后续范围。

## Next Atomic Action

T01 的核实与表分类已完成、FINDINGS.json 已回写。剩余：① 两份设计文档（`docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`、`BUSINESS_K8S_MODULE_DESIGN.md`）需将 T01 冻结的凭据/Agent/资产/扩展边界写入（我用 tiny diff 在声明文件内追加）；② commands.json/summary.md 按包要求补齐；③ findings-first review 生成后同步 linkage；④ 维护者对 4 项规划假设做最终范围验收后 T01 才可 close。

## Evidence

- [本轮逐条判定 + 表分类 + 迁移编号分配](../../evidence/2026-11-20-ops-review-baseline-and-decisions/baseline-verification-20261123.json)
- 父计划 FINDINGS.json 已回写 ARC-001/NEW-02/NEW-03/SUP-08（status=verified-at-head）

## Known Gaps

- 两份设计文档尚未写入冻结边界（本包 Modify 项，下一步）。
- commands.json / summary.md 尚未生成；review.md 为 none，真实 review 生成后同步 linkage。
- 维护者未对 4 项规划假设做最终范围验收（T01 Human Gate）。
- 业务表分类基于当前 HEAD 静态扫描；真实双租户对抗行为属 T04/T14 验收范畴，不算本包缺口。