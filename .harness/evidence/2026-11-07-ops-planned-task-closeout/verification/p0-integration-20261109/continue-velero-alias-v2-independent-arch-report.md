## Summary

**Architectural Status：`BLOCK`。** 旧 CODE/ARCH 四项问题已有源码闭环，但最终封存 v2 仍有一处 freshness 缺口：repair 的 S3 PUT/DELETE 可以发生在 predecessor 过期后。建议先收口该边界；本结论仅覆盖业务域验收工具，未执行远程阶段或改动文件。

## Analysis

**P1：repair 的存储写入遗漏最终 freshness 检查。** 最后一次检查位于 [remote.py:509](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-remote.py:509)，随后等待 Ready/BSL、执行完整 diagnostic，再调用 [S3 PUT/DELETE:542](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-remote.py:542)。实际 AST 中，repair 只有第 504、509 行两次 freshness 调用；继承的 [s3():136](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-velero-s3-v4-remote.py:136) 也没有该守卫。

因此，收据在 tag 时仍有效、随后超过 [1800 秒期限](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-contract.py:134)，仍可能继续存储写入。这是源码确认的边界遗漏，本轮没有执行其运行态复现。

旧发现的闭环证据：

| 原问题                        | v2 状态与证据                                                                                                                                                                                                                                                                                                                        |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| validate/cleanup 缺少完整复核 | [preflight:447](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-remote.py:447) 校验有限 fixture、完整 diagnostic、已修复镜像关联及 freshness；[Root:230](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-root.py:230) 接入原验证入口。 |
| diagnostic 部分上传记录失真   | [上传记录:86](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-root.py:86) 区分 intent/completed/uncertain；失败保留可能残留标记及有限检查。                                                                                                                                           |
| AST 适配丢失 captured loader  | [compile_captured:160](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-contract.py:160) 保留原 loader globals，并重新应用 `CapturedSyntax`。                                                                                                                                          |
| 原五对象 spec 不完整          | [原五对象适配:203](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-remote.py:203) 固定 Service、Endpoints 和 Secret policy；其余资源继续走完整 policy 校验。                                                                                                                          |

阶段链保持 `diagnostic → repair → validate → cleanup`；[已尝试阶段拒绝重放](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-root.py:254)，[inspect-failed](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-root.py:348) 不授予续跑权限。修复命令限定两条 absent alias，provider 如实标记 [`alias-recovery`](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-v2-remote.py:566)，没有重执行安装。

## Root Cause

freshness 守卫覆盖了 tag 和原 fixture 效果，却没有覆盖随后复用的存储探针。阶段入口有效，不能保证阶段内后续写入仍有效。

## Recommendations

1. **小工作量／高影响：**在 S3 PUT、DELETE 前分别调用 `fresh_predecessor(inputs)`；过期时保留探针路径与不确定状态，转入 inspect-only。
2. **小工作量：**补两个离线回归：最后一次 tag 后过期；PUT 成功、DELETE 前过期。修复交由 Dispatcher 后续安排，本轮不修改或重封存。

## Trade-offs

| 方案                  | 收益                      | 代价                                   |
| --------------------- | ------------------------- | -------------------------------------- |
| PUT/DELETE 均遵守期限 | 满足逐效果 freshness 要求 | 过期可能留下待检查探针                 |
| 单独授权补偿 DELETE   | 减少探针残留              | 需要明确独立清理权限，当前不能隐式推定 |

## Verification and Reviewed Hashes

实际 request SHA256：

`4bc6858b5f1ad5ff531339b8152fcd24b3a9ef53ed367f7f5720fbc158d22292`

五文件实际哈希均与 request、handoff、verification、LSP、sealing 一致：

| 文件：`continue-velero-alias-v2-*.py` | SHA256                                                             |
| ------------------------------------- | ------------------------------------------------------------------ |
| contract                              | `4582eb92ef30ce23e3f8c20e513c0925de5b8ffea9a2d45cfeb8f2671a615a66` |
| remote                                | `acee8606947ed303e0eb2f9c31abbf61d2879192f7b61a1c814bd351fd55d2e6` |
| root                                  | `f437de93590d61740b2e2a962078fcb92873a679ff445b0852091e915f6f1c7d` |
| generate                              | `d6835a601c43f46a1613194293dea9c1f016e2699504043f1c2b7231d8f8a3ad` |
| test                                  | `47c614c02eabeefd703e6c3ba3067e1bd34fa0659e1ed8019f8fbf3c413407d8` |

本轮五份内存编译、14 项既有纯内存离线检查通过；223 个 protected 文件重新核对均未变化。既有 [230 项零 skips 记录](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v2-verification.json:376) 与真实 [Pyright 1.1.414 LSP](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v2-lsp-final-20261110T064237081854.json:3) 已核验；LSP 使用 basic/openFilesOnly，不能证明动态适配或运行态成功。

**停止条件：**上述 freshness 缺口修复并独立复审。实际 containerd tag 并发语义、Ready/BSL、S3、BackupRestore、持久化及精确清理仍待 Dispatcher 在最终批准后验证；不授予 source、public HTTP/IAM/audit 或 P0 完成结论。
