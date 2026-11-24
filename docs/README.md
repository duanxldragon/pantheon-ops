# Pantheon Ops 文档索引

English version: [README.en.md](./README.en.md)

Ops 文档聚焦业务设计、业务验收与本地继承说明；平台层和系统域规范以 [Base 文档](https://github.com/duanxldragon/pantheon-base/blob/main/docs/README.md)为准。

## 当前交付与任务

当前执行以 [审查整改父计划](../.harness/tasks/2026-11-20-ops-review-remediation/plan.md)、[任务索引](../.harness/TASK_INDEX.md) 和 [STATUS](../.harness/STATUS.md) 为准。原试点计划已归档，未验收 residual 由当前主线承接；历史完成证据按原范围保存。

| 入口                                                                                           | 用途                                                                                  |
| ---------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| [试点交付计划](../.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-delivery/plan.md)        | 已按 2026-11-20 board-complete 收口归档（原唯一执行计划：6 个核心包＋4 个条件扩展包） |
| [执行队列](../.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md) | 依赖、优先级与执行顺序（已随 pilot delivery 归档）                                    |
| [当前状态](../.harness/STATUS.md)                                                              | 开放任务与验收 gap                                                                    |
| [归档索引](../.harness/ARCHIVE.md)                                                             | 已完成原范围的任务与原始证据                                                          |
| [任务包实例入口](./harness/tasks/README.md)                                                    | 任务维护与证据链，不作为长期设计副本                                                  |

## 业务与继承

- [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)：正式 Base release、业务 overlay 与所有权。
- [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md)：业务与 `base -> ops` 同步任务模板。
- [BUSINESS_FUNCTIONAL_DESIGN.md](./designs/BUSINESS_FUNCTIONAL_DESIGN.md)：现状能力矩阵与首期边界。
- [BUSINESS_TARGET_ARCHITECTURE_DESIGN.md](./designs/BUSINESS_TARGET_ARCHITECTURE_DESIGN.md)：业务目标架构。
- [BUSINESS_QA_ACCEPTANCE_PLAN.md](./designs/BUSINESS_QA_ACCEPTANCE_PLAN.md)：业务验收场景。
- [BUSINESS_TEST_PLAN.md](./designs/BUSINESS_TEST_PLAN.md)：业务验证集合。
- [BUSINESS_ERROR_SEMANTICS_APPENDIX.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md)：业务错误语义清单。

## 历史参考

- [BUSINESS_DEVELOPMENT_PLAN.md](./designs/BUSINESS_DEVELOPMENT_PLAN.md)：八月阶段估算，已由当前计划接管。
- [V1_RELEASE_CHECKLIST.md](./V1_RELEASE_CHECKLIST.md)：八月验收检查点；历史勾选不表示当前候选通过。
- [V1.0 冻结前综合评审](./audits/v1.0/pantheon-ops_v1.0冻结前综合评审报告.md)。
- [V1.0 业务闭环评审](./audits/v1.0/pantheon-ops_v1.0业务模块关联性与闭环检查报告.md)。
- [V1.2 设计第一部分](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN.md)、[第二部分](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN_PART2.md)、[第三部分](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN_PART3.md)：历史扩展设计，不代表全部已交付。

## 按需方法参考

以下是继承与方法维护的索引；普通业务任务按当前任务路由读取。Inherited shell references describe the locked Base snapshot; active method policy uses the explicit receipt.

- [Inherited shell contract](./harness/HARNESS_ENGINEERING_CONTRACT.md)
- [Inherited method playbook](./harness/HARNESS_METHOD_PLAYBOOK.md)
- [patterns/README.md](../../pantheon-harness/patterns/README.md)
- [patterns/harness-core-model.md](../../pantheon-harness/patterns/harness-core-model.md)
- [patterns/harness-coverage-model.md](../../pantheon-harness/patterns/harness-coverage-model.md)
- [patterns/harness-template-taxonomy.md](../../pantheon-harness/patterns/harness-template-taxonomy.md)
- [patterns/tool-adapter-matrix.md](../../pantheon-harness/patterns/tool-adapter-matrix.md)
- [patterns/method-playbook.md](../../pantheon-harness/patterns/method-playbook.md)
- [patterns/execution-guardrails.md](../../pantheon-harness/patterns/execution-guardrails.md)
- [patterns/context-engineering-protocol.md](../../pantheon-harness/patterns/context-engineering-protocol.md)
