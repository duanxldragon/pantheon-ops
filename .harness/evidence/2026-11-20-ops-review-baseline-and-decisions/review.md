# Review — 当前基线核实与业务边界决策

- Reviewer: primary agent (Claude Code) self-review; independent Codex review unavailable in this window (记录为显式 gap，非独立审查)
- Candidate: `main` `697c0771`（T01 判定原文基于规划基线 `a2d69d4a`，本审查对照当前 HEAD 区分"基线快照"与"后续实现"）
- Scope: `ARC-001` / `NEW-02` / `NEW-03` / `SUP-08` 判定、51 业务表分类、迁移编号分配、两份设计文档冻结边界

## Findings

1. **T01-R1（P2，fixed）— review 占位未生成真实审查。**
   原 `review.md` 仅为占位说明，缺机器可读块。本文件替换为 findings-first 审查并补齐 Machine Readable JSON。

2. **T01-R2（P2，accepted-gap）— 维护者最终范围验收未过。**
   四项规划假设（凭据按资源归属、Agent 关闭入口、资产关系收窄、Helm 登记口径）按维护者三触点契约属最终功能验收范围，本包只完成证据与冻结文本。

3. **T01-R3（P2，accepted-gap）— ARC-001 判定是基线快照，已被 T04 实现推进。**
   `baseline-verification-20261123.json` 记录"仅 observability 含 `TenantID`"对应规划基线 `a2d69d4a`；当前 HEAD 上 T04 已落 migration `000041_core_business_tenant_ownership`，`bizscope`/`cmdb`/`deploy`/`k8s`/`service` 模型均带 `TenantID`。这是计划执行结果，不是 T01 判定错误；逐表分类是否与 000041 实际迁移一致由 T04 审查复核。

4. **T01-R4（P2，accepted-gap）— T14 的 `000042` 尚未创建。**
   当前迁移目录只有 `000041`；`000042` 仍属 T14 范围，T14 受正式 foundation release 阻塞。编号分配本身无冲突（存量 34/35/36/39/40）。

5. **T01-R5（P2，accepted-gap）— 文档冻结文本的实际提交点在 #205。**
   两份设计文档的 T01 冻结段落经 `git log -S` 定位在 `5e4dac4f`（PR #205）落库，而 T01 `commands.json` 的 `changeRef` 仍为 `none/uncommitted`。内容与 T01 证据一致；traceability 偏差登记为后续收口项，不影响判定。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-review-baseline-and-decisions",
  "reviewer": "primary agent self-review (independent review unavailable)",
  "verdict": "approved with documented P2 follow-up",
  "decision": "approved with documented P2 follow-up",
  "scope": "source findings verification, business table classification, tenant migration allocation, frozen design boundaries",
  "taskManifest": ".harness/tasks/2026-11-20-ops-review-baseline-and-decisions/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-review-baseline-and-decisions",
  "summaryFile": ".harness/evidence/2026-11-20-ops-review-baseline-and-decisions/summary.md",
  "reviewFile": ".harness/evidence/2026-11-20-ops-review-baseline-and-decisions/review.md",
  "changeRef": "none",
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-review-baseline-and-decisions/manifest.json",
    "evidence": ".harness/evidence/2026-11-20-ops-review-baseline-and-decisions/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-review-baseline-and-decisions/review.md",
    "changeRef": "none",
    "planRefs": [
      ".harness/tasks/2026-11-20-ops-review-remediation/plan.md"
    ]
  },
  "findings": [
    {
      "id": "T01-R1",
      "severity": "P2",
      "status": "fixed",
      "summary": "review placeholder replaced with findings-first review and machine-readable block"
    },
    {
      "id": "T01-R2",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "maintainer final scope acceptance (human gate) open"
    },
    {
      "id": "T01-R3",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "ARC-001 verdict is baseline-bound; T04 migration 000041 has since added TenantID to core models"
    },
    {
      "id": "T01-R4",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "T14 migration 000042 not yet created (blocked on formal foundation release)"
    },
    {
      "id": "T01-R5",
      "severity": "P2",
      "status": "accepted-gap",
      "summary": "frozen design paragraphs landed in #205 while T01 changeRef records none"
    }
  ]
}
```
