# Pantheon Ops

English version: [README.en.md](./README.en.md)

Pantheon Ops 消费 Pantheon Base 的正式 foundation release，扩展运维业务。平台层与系统域由 Base 提供；Ops 负责业务域、业务装配和业务验收。

当前处于试点交付准备阶段，尚未完成试点验收。五个核心模块（CMDB、BizScope、Service、Deploy、K8s）已有实现；部署与实例关联、目录维护、K8s Service 编辑和当前版本真机验收仍有缺口。观测、文件分发、GitOps、备份已存在实现或装配，按实际开放范围独立验收；Helm 专用模块目前主要登记记录，不能据此宣称真实 Chart 安装。

当前提交锁定 Base v0.18.9；业务审查基于升级前提交，候选版本验收须重新采证。前端采用 React、TypeScript、Arco Design、Vite；后端采用 Go、Gin、GORM，依赖 MySQL 和 Redis。

Helm 专用接口只提供发布记录登记、查询和删除登记，不执行 Chart 安装、升级、回滚或卸载；这些执行能力当前延期且未开放。新建记录固定为 `registered`，请求中的运行状态不作为执行结果保存。所有响应的 `status` 和兼容字段 `effectiveStatus` 均为 `registered`，历史原始状态在 `storedStatus` 中保留，数据库历史行不改写；历史 `deployed` 值和 K8s 镜像 Release 均不能证明 Chart 已安装。删除登记也不会卸载集群资源。

## 当前计划

- [试点交付计划](./.harness/tasks/2026-11-16-ops-pilot-delivery/plan.md)：6 个首期必交包、4 个条件扩展包，含依赖与可检验退出条件。
- [执行队列](./.harness/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md)：从 C01 开始；C03/C04 可独立推进。
- [当前任务状态](./.harness/STATUS.md)与[归档索引](./.harness/ARCHIVE.md)：12 个历史任务按原范围归档，证据保留。
- [业务功能设计](./docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md)：当前能力与边界。

## 开发入口

1. 阅读 [文档索引](./docs/README.md)、[继承说明](./docs/PROJECT_INHERITANCE.md)和 [AGENTS.md](./AGENTS.md)。
2. 按 Base 的 [部署指南](../pantheon-base/docs/DEPLOYMENT_GUIDE.md)准备隔离 MySQL/Redis 与环境变量；Ops 使用业务迁移与运行时 seed。
3. 在 backend/ 执行 go run ./cmd/server；在 frontend/ 执行 npm install、npm run dev。

在仓库根运行 npm run check:business-overlay、npm run check:shared-drift；重建使用锁定的官方 release，参见继承说明。业务修改按子任务运行对应测试、smoke 与完整 CGO race；编译通过不替代外部环境与用户流程验收。

共享能力改动先进入 [pantheon-base](https://github.com/duanxldragon/pantheon-base)。许可证见 [LICENSE](./LICENSE)。
