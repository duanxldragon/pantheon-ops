## 变更摘要

- 改动层级：业务域 `business/*`、业务数据库迁移、Ops 验收证据
- 改动模块：Deploy 异步执行与凭据、K8s ownership/credential/conflict、CSV 导入导出、业务前端、legacy migration
- 目标问题：完成 Pantheon Ops V1 C-F backlog 中未完成的开发、迁移、权限边界和发布验收任务
- 预期影响：Deploy 任务具备持久化尝试/租约/重试/取消能力；K8s 写入受 NamespaceBinding、凭据引用和 resourceVersion 保护；业务列表提供 CSV import/export；legacy schema upgrade 可重复执行

## Harness 链路

- Task ID：`2026-08-19-ops-v1-backlog-closeout`
- Task Manifest：`.harness/tasks/2026-08-19-ops-v1-backlog-closeout/manifest.json`
- Evidence：`.harness/evidence/2026-08-19-ops-v1-backlog-closeout/commands.json`
- Verification evidence：`.harness/evidence/2026-08-19-ops-v1-backlog-closeout/summary.md`
- Review Artifact：`.harness/evidence/2026-08-19-ops-v1-backlog-closeout/review.md`
- OpenSpec change：none
- Trivial change：no
- Quality Profile：permission-policy
- Ratchet Decision：guide-updated
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: `2026-08-19-ops-v1-backlog-closeout`
- task manifest: `.harness/tasks/2026-08-19-ops-v1-backlog-closeout/manifest.json`
- evidence: `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/commands.json`
- boundaries: Ops-owned `business/*`; no Base source changes
- backend response contract: typed DTOs and resourceVersion metadata retained
- backend DTO contract: credential material is redacted; paging fields are bounded
- permission contract: K8s action-specific seeds and NamespaceBinding enforcement
- audit coverage: mutation paths retain existing audit middleware and conflict semantics
- visual evidence: `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/screenshots/`
- inheritance contract: business overlay and clean rebuild evidence recorded
- base drift: no Base source change; Base-first gap remains explicit
- Base/ops inheritance: Ops consumes the locked foundation and owns only business extensions

## 边界说明

- [x] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

本 PR 未修改 `pantheon-base`。共享平台/系统域行为保持在 Base；本次改动只落在 Ops 业务模块、业务 migrations、业务 UI 和证据。生产数据库、生产 Kubernetes 和生产 SSH mutation 未执行；独立授权的 Kylin 单节点 K3s 仅用于部署验收，结果见 `k8s-install-acceptance.md`。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke / runtime evidence：Deploy desktop/mobile smoke、隔离 K3s Deployment/Service/Job/DNS/PVC/metrics/restart 验收
- [ ] GitHub required checks 通过（PR 创建后等待 hosted checks）
- [x] Copilot review 已请求或由仓库自动策略处理；维护者 approval 仍为合并门禁

补充说明：Windows CGO `go test -race ./...`、`go vet ./...`、MySQL 8 isolated migration rehearsal、overlay rebuild 和 release artifact checksums 均记录在 summary/evidence 中。

## 审核留痕

- Copilot review：automatic-policy
- CodeQL 结果：等待 GitHub hosted check
- GitHub checks 结果：PR 创建后记录
- Auto-merge：not-enabled
- Duplication Gate 结果：local evidence pass; hosted check pending
- 是否高风险改动：yes，涉及权限、凭据引用、K8s mutation guards 和数据库迁移
- Residual risk / follow-up：生产快照迁移和生产/live-cluster mutation 仍需维护者明确 gate；独立验收服务器不是 HA 生产拓扑

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token；本地 `config/kubeconfig.yaml` 未纳入提交
