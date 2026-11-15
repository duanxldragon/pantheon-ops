# Pantheon Ops

面向 SRE 和运维团队的业务平台，消费 Pantheon Base 正式 foundation release，并通过声明式 business overlay 扩展 CMDB、部署、Kubernetes、可观测和服务管理。

当前活动底座为正式 **v0.18.8**，提交 `34428814a9efc997fd93d4434406a53df6460323`。通过 GitHub 正式发行资产完整重建并保留业务 overlay；此前权限失败和回滚证据保留。前端构建、414 项单测、24 项实际浏览器烟测及完整 Go race 的证据见当前执行记录；六个 P0 父任务和最终功能/视觉验收仍开放。

- [当前执行证据](./.harness/evidence/2026-11-14-ops-latest-foundation-and-plans/summary.md)：正式资产来源、候选验证、活动安装与剩余门禁。
- [剩余规划](./.harness/tasks/2026-11-07-ops-planned-task-closeout/remaining-priorities.md)：O2/O3/O4/D1/D2/D3 六个 P0 父任务仍开放，Velero 后置、O7 deferred。
- [继承与升级流程](./docs/PROJECT_INHERITANCE.md)：正式锁定版本、完整重建和 business overlay 所有权。
- [业务设计与任务入口](./docs/README.md)。

## 本地开发

按仓库声明的工具链准备 Go 1.26.10、Node.js 24、MySQL 8 和 Redis 7。先按继承说明验证正式版本和 overlay，再启动后端与前端；数据库、Redis、初始化身份及安全环境变量沿用 [Base 文档](https://github.com/duanxldragon/pantheon-base/blob/pantheon-base-v0.18.8/docs/README.md) 的配置合同。

```powershell
# Ops 根目录：校验业务装配合同
npm run check:business-overlay
npm run test:business-overlay

# backend/：配置好隔离开发环境后启动
go run ./cmd/server

# frontend/：使用 lockfile 安装并启动
npm ci
npm run dev
```

Windows 后端提交门禁要求 MSYS MinGW GCC、显式 `CGO_ENABLED=1` 和完整 `go test -race ./...`。业务代码只落在 Ops；平台层与系统域共享修复先回 Base，再消费正式 release。历史测试与能力对标文档不能替代当前验收。
