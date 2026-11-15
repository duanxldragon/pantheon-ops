已完成限定范围内的 O4 Generator 修复，保留其他人的改动及封存失败现场。

- `check-shared-drift.mjs`、`rebuild-from-base.mjs`：重放固定业务装配输出，验证字节、报告哈希和声明所有权；范围外 allowance 明确标记未比较，真正 stale 的条目仍失败。
- `foundation-candidate.mjs`、`rebuild-from-candidate.mjs`：登记目录创建身份，安全清理已证明归属的目录；未知或替换目录保留，并保存原错误 `cause`。新增 `--attempt`，独立装配并重验原始快照。
- 两个 focused test 文件补齐生成输出篡改、伪造报告、共享字节篡改、stale、目录替换及快照重试回归。

验证：**155/155 合同测试通过，0 失败、0 跳过**；6 个 JS 文件通过 Node 语法检查；11 个生成 TS 文件无语法诊断；overlay contract、diff 检查通过。

真实失败树只读复查确认：801 个共享文件一致、17 个派生输出及10个迁移通过；1,141 个快照文件和4,634个报告文件哈希保持一致。**未执行新的实际装配，仍无 consumer receipt。**

[完整交接与 root 下一条命令](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-integration-repair/summary.md) · [六个文件的完整旧/新 SHA256](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-integration-repair/source-hashes.json) · [仅本次修复差异](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-integration-repair/repair.patch)

descriptor pin 保持 `2f105d76e38c82e8c301d816269ea26669a7a0291a5860be74d6ca71281a9ce6`；官方 lock、manifest、allowlist 未变。独立 CODE/ARCH、源码静止后的实际装配和完整产品验证仍由 root 执行；O4 C–F 未启动，无 P0 自批准。
