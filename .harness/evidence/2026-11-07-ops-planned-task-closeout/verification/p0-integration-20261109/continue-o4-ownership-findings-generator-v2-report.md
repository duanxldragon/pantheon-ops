已完成四项 O4 CODE finding 的受限安全修复和本地验证，等待独立 CODE/ARCH 审查。

## Changes Made

- [foundation-candidate.mjs](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:554)：目录身份仅作观察 pin；清理前检查完整子树，拒绝未知文件、替换或消失对象。未证明的目录保留并明确延期清理；文件写入使用独占创建和已核验句柄。
- [rebuild-from-base.mjs](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:511)：单文件/目录 exclusion 在复制前筛选；剩余写入接入保护。Go 使用保留的 alternate modfile 工作区，再经受保护写入导入结果。
- [rebuild-from-candidate.mjs](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:267)：验证及 receipt 前后检查目标身份，保留原始错误对象和 cause。
- 新增 [16 项所有权测试](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/candidate-ownership.test.mjs)。原六文件修复产物、独立 findings 和封存树均保留；本轮仅改变其中三个脚本及 candidate 测试。

## Verification

- RED：9 项检查中 **7 项失败**，覆盖全部四个独立 finding。
- 最终完整 candidate/overlay 套件：**171/171 passed，0 failed，0 skipped**。
- Node syntax：**7/7 通过**；business-overlay checker：**56 declared paths OK**。
- 原快照 **1,141/1,141**、原报告文件 **4,634/4,634**、35 项保护产物哈希匹配。
- 六文件 before/after pins、NEW repair-only diff 和精确命令见 [交接证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/summary.md)。

## Assumptions / Notes

Node/Windows 的 mkdir、身份检查、unlink 和 Go 调用是分离 syscall，不能证明原子目录创建或抵御最后检查窗口内的敌对替换。因此目录不获删除权限；已认证文件的 unlink 仍有明确的非原子边界。CodeGraph/LSP 不可调用。

夹具验证不代表真实装配或 P0 收口。本轮未执行完整源码装配、Go race、前端语义检查或运行态验收，也未自行批准。

## Summary

独立 **CODE 和 ARCH 均通过后**，root 的下一条完整装配命令为：

```powershell
$o4Sealed = '.local/ops-acceptance/o4-metadata-bound-current-v2-stable-real/sealed'
node scripts/business-overlay/rebuild-from-candidate.mjs --descriptor "$o4Sealed/candidate-descriptor.json" --descriptor-sha256 2f105d76e38c82e8c301d816269ea26669a7a0291a5860be74d6ca71281a9ce6 --artifact "$o4Sealed/candidate-4315cee606c249a42c8a3372c26dee9a4658a9259229485de85953430c7f439f.tar" --attempt o4-code-safety-reviewed-20261110-01
```
