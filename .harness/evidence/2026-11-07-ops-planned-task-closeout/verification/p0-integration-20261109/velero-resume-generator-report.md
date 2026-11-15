## Changes Made

- [CreateRestore](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:497)：持久化前检查授权、凭据和真实源状态；提交时锁定集群与源备份，worker 再次校验。
- 补齐 namespace mapping 转换和授权回归测试。

## Verification

- 真实隔离 MySQL：28 个新增子测试通过，包含外部门拒绝、零 provider 调用、撤销、取消和成功入队。
- Scoped race、vet、build 全通过；cgo=1，使用 MinGW。
- 实时驱动 race build/vet、7 个 Python 脚本语法检查通过。
- red/green 日志、精确源码哈希：[证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/velero-resume-closeout.json)。

## Assumptions / Notes

远端 inspect/install/validate 已准备，额外内存上限 512Mi。[阶段请求](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/velero-resume-phase-request.json)等待 root 执行；本轮未尝试 SSH 或安装。

## Summary

本地修复已验证。真实 S3 备份、恢复内容、远端清理及独立 review 尚待回执；未宣称 D1/P0 完成，D4 签署恢复仍在范围外。
