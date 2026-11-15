v9 本地候选已完成：每个 driver 复用独立、严格固定主机指纹的 SSH 连接；每条命令仍使用新会话和超时，无重连、重试或重放。

- 验证通过：15 项 Go 测试、4 项入口测试、vet/race、Windows/Linux 构建，均退出 0。
- 185 个产品依赖及既有 v7/v8 产物封存不变。
- [详细交接](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v9-handoff.md) 与 [root 请求](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-o2-v9-root-phase-request.json) 已生成。

执行门保持关闭；未加载真实凭据或执行远程矩阵。真实 12 案例验收由 root 评审、重新封存后执行。
