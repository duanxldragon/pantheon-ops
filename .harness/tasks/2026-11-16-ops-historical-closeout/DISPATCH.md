# 历史任务收尾：派发与验收契约

日期：2026-11-16。维护者指令：先把历史任务全部完成收尾；Claude 执行，Codex 只派发任务和验收成果。

## Scope

- In：`2026-11-16-ops-pilot-delivery/OPEN_TASKS.md` 的全部历史任务、O1–O7、D1/D2/D4/D5、SSH 配置外部化、缺完成链旧卡和已完成待归档记录。已有归档的12包只核查引用完整性。
- In：逐项追溯原始验收，补真实实现、测试和运行证据；有证据支持的 completed/superseded/upstream-owned 记录规范收口并保全归档。
- Out：无范围关联的新功能、生产写操作、替维护者/DBA签署、未经验证的状态关闭、发布/合并、修改正式Base锁版本或手抄共享源码。
- 新C/E任务只有与历史任务真实重叠的部分可实施；禁止因映射或计划移交自动认定实现完成。

## Roles and Ownership

- Codex：派发、读证据、执行验收检查、退回修订、维护最终验收结论。不直接修改业务代码。
- Claude intake：历史全量对账和任务拆分。只写本父包、`.harness/evidence/2026-11-16-ops-historical-closeout/`；其它源文件只读。
- 后续执行Claude：按对账结果分配独立文件所有权；共享文件串行。各执行者知道共享工作区存在他人改动，不覆盖或撤销既有改动。

## Acceptance

1. 每一历史ID/子项有且只有一个结论及原始依据、当前事实、实施责任、验收证据、剩余条件。
2. 缺实现项实施并验证；缺真实环境项实际验证，无法取得端点/凭证/生产签署则明确保留开放，不能“转交”后报全完成。
3. O1–O7逐项对原packet验收；原延期O7属于此次检查范围，若成熟度条件缺失应说明真实前置条件。
4. 有闭环证据的历史记录才归档，task/evidence成对、原字节与raw日志保全、引用有效；不得回写虚构历史命令。
5. 相关Harness检查、文档链接、业务overlay、代码/运行检查通过；按代码范围应用repo-verify，UI使用impeccable并给渲染证据。
6. final acceptance逐项记录accepted/rejected/external-gate；只在无剩余必做项时报告全部完成。

## Dispatch Sequence

1. Claude intake：读全部原始任务和关闭证据，生成`INVENTORY.json`、`PLAN.md`与按所有权拆分的任务提示词。
2. Codex核查完整性和不可绕过的门禁后派发执行批次。
3. Codex对成果进行独立验收；发现缺口原Claude返修，直到完成或留下经验证的外部阻塞。
