# Pantheon Ops 文档索引

English version: [README.en.md](./README.en.md)

Ops 文档聚焦业务设计、业务验收与本地继承说明；平台层和系统域规范以 [Base 文档](https://github.com/duanxldragon/pantheon-base/blob/main/docs/README.md)为准。

## 当前交付与任务

截至 2026-11-16，试点尚未验收完成；历史完成证据按原任务范围保存。

| 入口                                                                           | 用途                                         |
| ------------------------------------------------------------------------------ | -------------------------------------------- |
| [试点交付计划](../.harness/tasks/2026-11-16-ops-pilot-delivery/plan.md)        | 当前唯一执行计划：6 个核心包＋4 个条件扩展包 |
| [执行队列](../.harness/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md) | 依赖、优先级与执行顺序                       |
| [当前状态](../.harness/STATUS.md)                                              | 开放任务与验收 gap                           |
| [归档索引](../.harness/ARCHIVE.md)                                             | 已完成原范围的任务与原始证据                 |
| [任务包实例入口](./harness/tasks/README.md)                                    | 任务维护与证据链，不作为长期设计副本         |

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
