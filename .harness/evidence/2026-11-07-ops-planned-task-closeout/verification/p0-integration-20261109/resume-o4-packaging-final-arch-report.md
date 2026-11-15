**ARCH CLEAR — 限于指定 L2 packaging delta 与新配对 helper。未发现阻断缺陷，可交 root 执行下一次稳定源码装配验证。**

- **metadata 与 archive 边界正确分离。** frozen `METADATA_LIMITS.files=100000` 仅用于 capture 的唯一文件计数；candidate inventory/tar 仍为 50,000，普通文件 32 MiB、archive 512 MiB 保持不变。[限制定义](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:7)、[capture 检查](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:638)。
- **流式读取保留完整核对。** 每次缓冲最多 1 MiB，使用显式 offset；bigint fd/path identity 包含 dev、ino、size、mtimeNs、ctimeNs、mode、nlink，最终核对完成后才关闭 fd。[实现](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:657)。
- **PUBLIC 范围没有扩成递归披露。** 仅八个精确叶子通过路径例外；两份原配置、registry 引用与 archive payload 必须对应。producer 拒绝允许叶子上的目录。[Base validator](D:/workspace/go/pantheon-platform/pantheon-base/scripts/foundation-release/candidate-descriptor.mjs:30)、[producer 拒绝条件](D:/workspace/go/pantheon-platform/pantheon-base/scripts/foundation-release/build-candidate-artifact.mjs:33)。两份 live 配置与 captured originals 字节一致，两端 curated validator block 完全一致。
- **生产者/消费者配对成立。** helper 导入 applied live Base，拒绝既有 output，并直接把 producer 返回的原始 `descriptorSha256` 交给实际消费者；当前 manifest 不包含 receipt 目录。[helper](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-o4-stable-source-producer-consumer.mjs:15)。消费者继续执行实际 assembler、source stability、drift 和 lock 检查，并拒绝 stage identity 复用。[消费者](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:133)。

根因是原 capture 将 retained metadata 套用了 candidate 的单文件与数量限制，同时原 Base 路径策略拒绝既有 registry 引用的全部 `.harness` 叶子。[原 capture](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-real-packaging-1794279171056/original-ops/foundation-candidate.mjs:592)。本次修复针对这两个不兼容点，未改变 archive 契约。

本轮 fresh verification：**Base 38/38、Ops 76/76、overlay 62/62**，contract 检查通过。新测试覆盖真实读取路径上的 100,000/100,001 synthetic leaves，以及 digest 正确的 50,000/50,001 tar。[边界测试](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.test.mjs:13)。retained RED 仅 metadata 边界失败；六文件 TypeScript 6.0.3 diagnostics 为空，`checkJs=false`，不构成完整类型证明。

最终复核：live Base 三文件等于 retained proposal；metadata delta 可精确反向还原；原测试与 producer seal/cleanup tail 保留。11 个审查 pin、11 个 ledger source、21 个 protected artifact 均无变化。helper SHA256：`e5e7303bab69a175427fd5a11fcea7a0539fc6d2bfaac4ee0f8330a587fe75d8`。

建议 root 在源码静止后填写真实 `reviewIds`，使用 fresh request/output 执行指定 helper。[交接要求](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o4-metadata-bound-fix-handoff.md:41)。

| 取舍                                           | 收益                                          | 代价                                                      |
| ---------------------------------------------- | --------------------------------------------- | --------------------------------------------------------- |
| 独立 metadata cap、流式 hash、精确 PUBLIC 例外 | 保留现有 registry/cache 兼容性和 archive 防线 | 更多读取调用、较大 metadata inventory；不提供原子快照保证 |

本审查到此收口。55,812 仅为既有枚举观察；九个实际大 archive 依据此前 focused evidence。actual full assembly、O4C-F、full race、privacy、双租户、UI 与 official release 仍由 root 验收。
