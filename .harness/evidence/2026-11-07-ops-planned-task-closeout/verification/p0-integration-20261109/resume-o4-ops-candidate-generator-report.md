指定的 Ops 候选工具已完成并验证。

## Changes Made

- [foundation-candidate.mjs](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:572)：固定哈希验证、安全提取、快照稳定性和构建身份校验。
- [rebuild-from-candidate.mjs](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:53)：复用装配/漂移接口，保护暂存目标并输出候选收据。
- [foundation-candidate.test.mjs](D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.test.mjs)：恶意归档、源变化、目标保护和锁稳定性测试。

## Verification

- 候选测试：48 项通过。
- 既有 overlay 测试：62 项通过。
- Node 诊断、Prettier、overlay 契约、文档门禁和空白检查通过。
- 正式工具、锁文件及旧报告哈希均未变化；初始失败日志已保留。

## Assumptions / Notes

实际产品候选封存、独立 CODE/ARCH 评审、产品构建与配对 runtime 仍由 root 完成。

## Summary

本工具分支已验证完成。[完整证据与源码哈希](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o4-ops-candidate-report.md)。
