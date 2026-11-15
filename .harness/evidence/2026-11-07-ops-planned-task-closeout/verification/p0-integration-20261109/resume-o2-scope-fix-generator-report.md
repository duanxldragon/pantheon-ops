已修复业务域 Deploy 的 start/resume/retry 准入和最终 SSH 命令边界：失活范围拒绝执行，pause/cancel 保留清理能力，冻结快照行为保持。

- 必需 MySQL + MinGW CGO1 race：248 项通过；vet/build 通过，源码哈希稳定。
- v7 双平台构建、17 项安全检查通过；185 份编译源码及依赖已封存，v6 证据保留。
- 无新增诊断；1 条既有提示已记录。

[验证报告](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v7-generator-report.md) · [源码哈希](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v7-readiness.json) · [root 执行请求](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v7-phase-request.json)

本 lane 未执行远端操作。完整 12 场景、全树验证和独立审查仍待 root；P0 未完成，完整角色/会话撤权仍属未来 Base 接口范围。
