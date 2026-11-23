# Summary — 当前基线核实与业务边界决策

- Task: 2026-11-20-ops-review-baseline-and-decisions
- Status: in-progress（核实完成，Human Gate 未过）
- Updated: 2026-11-23
- Baseline: a2d69d4a（规划）/ 444ce8e（来源报告文末）

## 本轮成果

对 T01 承接的 4 条来源发现逐条在当前 HEAD 核实，判定 + 归属如下：

| 发现 | 判定 | 证据要点 | 任务归属 |
|---|---|---|---|
| ARC-001 | 仍成立 | 遍历 51 业务表，仅 observability（datasource/query_workspace 2 文件）含 TenantID；核心业务表全无租户字段；12 组全局唯一键需租户变体 | T04（000041）、T14（000042） |
| NEW-02 | 仍成立 | 无 Pod→Node→VM 资产关系实体，仅 Node capacity 元数据读取 | 独立后续能力（文档收窄） |
| NEW-03 | 设计如此 | helm.go 包注释自明 DB-snapshot 登记，非缺陷 | E03 口径保留；真实 Chart 独立后续 |
| SUP-08 | 维持 P2 | plan.md 已降 P2 授权模型缺口 | T03 承接，T16 实测定条件 P1 |

同时交付：业务表四分类（T04 核心 tenant-owned 28 表 + T14 扩展 13 表 + 已有 observability）、租户唯一键变体清单、迁移编号 000041/000042 分配、4 项规划假设冻结（凭据归属/Agent 关闭/资产收窄/Helm 口径）。

## 验收标准对照

- [x] 全部发现均有当前候选判定和任务归属（baseline-verification-20261123.json + FINDINGS.json verified-at-head）
- [x] 业务表清单覆盖实际表并逐表分类，T04/T14 迁移编号不冲突（枚举 51 表、迁移池核对）
- [~] 输出可执行的凭据归属/Agent 开放状态/资产定位和扩展启用清单——已完成并写入两份设计文档，**未定产品选项记为假设**，待维护者最终范围验收

## Gaps

- review.md 未生成（findings-first review 由独立审查轮执行）；Review File none
- 维护者终验未过（Human Gate）
- 本包为分析型 reuse，无产品代码，故无 UI rendered evidence 需求