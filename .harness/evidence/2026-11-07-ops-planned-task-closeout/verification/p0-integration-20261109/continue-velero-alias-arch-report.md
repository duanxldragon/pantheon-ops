`ARCH BLOCK` — 仅限当前恢复工具与 Root 暂存调用。两个 normalized alias 是有证据支持的最小修复方向，但以下调用边界需先收口。

1. **P1：validate/cleanup 的可变操作前缺少完整运行态复核。**  
   Root 在入口核对远端文件后，直接调用原验证或清理流程；新增的 native、22 UID/spec、CRD、namespace LIST 和 alias/content 检查只用于 diagnostic/repair。[root.py:173](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-root.py:173)、[root.py:199](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-root.py:199)、[remote.py:303](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-remote.py:303)  
   原 fixture 确实检查 namespace UID、BSL、监听端口和 store mirror，但未覆盖本次完整状态契约。因此，成功 repair 后发生的 Deployment/spec、凭据或 native 状态漂移，可能在 fixture CREATE 前未被拒绝。[v6-fixture.py:26](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-velero-s3-v6-fixture.py:26)

2. **P1：diagnostic 上传失败的部分变更记录不真实。**  
   diagnostic 会顺序独占上传两个 helper；第一份上传成功、第二份失败或响应丢失时，顶层异常处理仍设置 `partialMutationMayRemain=false`，且部分文件捕获仅针对 repair。[root.py:67](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-root.py:67)、[root.py:175](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-root.py:175)、[root.py:222](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-root.py:222)  
   本次全 mock、内存执行复现了该 false 标志；没有连接远端或写文件。后续 baseline 会拒绝残留 helper，限制了重放，但未补足失败证据。

**根因与已成立的设计判断：**实际报错明确指向 normalized containerd 引用缺失；现有列表包含 unnormalized index 和 canonical official manifest，CRI 却返回 normalized repoDigest。[runtime-diagnostic.json:42](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-runtime-diagnostic-20261110T052807663710.json:42)、[image-diagnostic.json:16](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-image-diagnostic-20261110T052901682299.json:16)、[image-diagnostic.json:34](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-image-diagnostic-20261110T052901682299.json:34)

修复契约固定 `k8s.io`、两条 absent alias、原 index→linux/amd64 manifest/config/layers；matching-present alias 同样拒绝。[contract.py:77](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-contract.py:77)、[remote.py:66](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-remote.py:66) 独占 intent/outcome、lost-reply 记录以及失败/错阶段/过期 predecessor 拒绝机制方向正确；canonical inventory 保留原字节，provider 如实标记 `alias-recovery`。[remote.py:329](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-remote.py:329)、[remote.py:407](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-remote.py:407)、[contract.py:123](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-contract.py:123)

**最小建议：**

1. **中等工作量，高影响：**在 validate/cleanup 可变入口复用完整只读 diagnostic，以两条已修复 alias 为预期，核对已审查 repair 状态；紧邻可变边界再次检查 predecessor freshness。保留原 fixture、driver 和验收断言。
2. **小工作量，高影响：**独立记录 helper 上传尝试、完成与不确定状态；上传开始后失败应标记可能残留，并只捕获有限 helper 文件供检查。不得自动续传、重放或覆盖。
3. **小工作量：**补充跨阶段 native/spec 漂移及上传 lost-reply 用例；由独立 CODE reviewer 核验实际 containerd tag 的并发存在行为。

| 选择                     | 收益                   | 代价                             |
| ------------------------ | ---------------------- | -------------------------------- |
| 补复核后 alias-only 恢复 | 保留 UID、数据和失败链 | 更多只读检查                     |
| 直接使用当前暂存调用     | 少一次检查             | 不满足可变边界与部分结果记录要求 |

当前 Request SHA 为 `f053fb6d…a81d70`；五个源码 hash 与 request、preparation、verification 全部一致，五份源码内存编译通过。记录中的 27 测试不能代替运行态证明。[verification.json:15](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-verification.json:15)

停止条件：上述两项完成并对新封存字节复审。实际 Ready、S3/BSL、Backup/Restore、持久化和精确清理仍待 Root 分阶段验证；本审查不授予产品 HTTP、Base IAM/audit 或整个 P0 结论。全程只读，未生成 candidate assembly 或落盘 review。
