已完成最小 copy 入口修复：Base、Ops 和所选元数据的实际返回字节在写入前校验大小与哈希；未知输入拒绝，原稳定性错误和清理规则保留。

- 保留反例：RED **0/2** → GREEN **2/2**，均在 receipt 前拒绝。
- 最终 JS 回归：**207/207**，包含原 198 项契约，零跳过。
- Node 语法与 overlay 契约通过；原生 `checkJs=true` 保留 137 条已有诊断，新增 **0**。
- 19,069 个旧文件、1,141 个 snapshot、4,634 个 report 哈希全部匹配；并发 progress 更新保留。

[304-word 交接](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-o4-copy-binding-20261110-r1-handoff.md) · [精确命令、计数及前后哈希](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-o4-copy-binding-20261110-r1-evidence.json)

Request SHA256：`6dc636da5804c8c1783f702bfe9e4618b9398b7a965461f5c7d6471ff72dc90e`

当前请求的独立 CODE/ARCH review、实际完整装配及 O4 C–F 仍待完成；未作自审批。
