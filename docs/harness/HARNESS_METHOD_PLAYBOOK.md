---
title: Harness Method Playbook
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-24
---

# Harness Method Playbook

English version: [HARNESS_METHOD_PLAYBOOK.en.md](./HARNESS_METHOD_PLAYBOOK.en.md)

本文件不再承载完整方法定义。

## 当前关系

- `pantheon-harness/`：方法事实源，负责方法编排、模板、schema、portable checks
- `docs/harness/*`：当前仓库的 repo-local 合同与落地层
- `scripts/harness/*`：当前仓库的 mechanical gates

## 阅读顺序

1. 先读 `../../../pantheon-harness/patterns/README.md`
2. 再读 `../../../pantheon-harness/patterns/harness-core-model.md`
3. 再读 `../../../pantheon-harness/patterns/harness-coverage-model.md`
4. 再读 `../../../pantheon-harness/patterns/harness-template-taxonomy.md`
5. 再读 `../../../pantheon-harness/patterns/tool-adapter-matrix.md`
6. 再读 `../../../pantheon-harness/patterns/method-playbook.md`
7. 再读 `../../../pantheon-harness/patterns/execution-guardrails.md`
8. 再读 `../../../pantheon-harness/patterns/context-engineering-protocol.md`
9. 再读 `../../../pantheon-harness/patterns/method-first-delivery-policy.md`
10. 再读 `../../../pantheon-harness/patterns/cross-agent-ratchet-model.md`
11. 再读 `../../../pantheon-harness/patterns/minimal-complexity-ladder.md`
12. 再读 `../../../pantheon-harness/patterns/templates/task-packet.template.md`
13. 再读 `../../../pantheon-harness/patterns/templates/review.template.md`
14. 再读 `docs/harness/PANTHEON_BASE_DELIVERY_WORKFLOW.md`
15. 再读 `docs/harness/AI_QUALITY_GOVERNANCE.md`
16. 再读当前仓库需要落地执行的合同：
    - `HARNESS_ENGINEERING_CONTRACT.md`
    - `TRIVIALITY_CLASSIFICATION_POLICY.md`
    - `TASK_PACKET_SPEC.md`
    - `TOOL_ADAPTERS.md`
    - `VERIFICATION_EVIDENCE_SPEC.md`
    - `REVIEW_LOOP_SPEC.md`
    - `VISUAL_QUALITY_PROTOCOL.md`
    - `VISUAL_EVIDENCE_PROMOTION_POLICY.md`
    - `FAILURE_RATCHET_POLICY.md`
    - `FAILURE_REGISTRY_PROMOTION_POLICY.md`
    - `HARNESS_RETIREMENT_REVIEW.md`

## 默认执行护栏

在进入实现前，默认先过一遍 `../../../pantheon-harness/patterns/context-engineering-protocol.md`：

- 先区分 confirmed facts、working assumptions 和 open questions，不要在权限、审计、边界问题上静默猜测。
- 先走最小复杂度阶梯，把方案压到最小可承重复杂度。
- 先明确 entry sources、检索顺序和敏感上下文边界。
- 如果任务是长会话、跨 session、带 delegation 或成本敏感，再额外声明 `Response Budget`、`Retrieval Helpers`、`Promotion Target` 和 `Economics Watch`。
- 先声明 `Do Not Touch` 和 `Structural Scope`，再做手术式修改。
- 先写 `Success Criteria`、`Verification Plan` 和 evidence linkage，再声明完成。

## 当前仓库职责

当前仓库保留这些内容，是因为它们直接服务本仓库运行：

- `docs/harness/*`：合同、格式、治理规则
- `scripts/harness/*`：校验器和 CI 门禁
- 新增 generic gates：review、template health、runtime evidence、docs links、sync drift
- `.agents/*` / `.codex/skills/*`：工具适配层

如果方法层与仓库落地层出现冲突，以 `pantheon-harness/` 的方法定义为先，再同步仓库落地层。
