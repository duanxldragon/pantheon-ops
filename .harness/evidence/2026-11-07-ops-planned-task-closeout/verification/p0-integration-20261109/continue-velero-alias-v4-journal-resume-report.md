v4 最小修复已完成并单次封装，P0 保持开放。

## Changes Made

- 仅新增五个 v4 脚本及对应 EV 产物。
- 增加持久探针 intent/outcome 日志、无终态收据检查和 INCOMPLETE 状态；ARCH 必须同时为 APPROVE、CLEAR。

## Verification

- 实际 v3 RED：3 项测试、6 个预期断言失败。
- GREEN：250 项唯一回归、303 次执行；专项 12 项通过，零跳过。
- 五文件内存编译通过；原生 Pyright LSP **0 errors / 0 warnings**，无抑制。
- 全部 **267 项旧资产未变**；测试、LSP、封装前后哈希一致。

Request SHA256：`dd3804fec93693b331397bc9b74182540ba694fc12543dcb3e5b3aeedae8e0fb`

## Assumptions / Notes

独立最终 v4 CODE/ARCH 和实际运行仍待 Dispatcher；未执行远端操作或自批准。

## Summary

[568-word 交接文档](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v4-handoff.md)包含五文件精确哈希、分开的 Dispatcher 命令及报告契约。[最终审计证据](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v4-final-audit.json)。
