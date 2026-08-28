# Pantheon-Ops V1 发布检查表

更新时间：2026-08-28
类型：Checklist
归属层：platform
状态：Active
依据：`.harness/evidence/2026-08-19-ops-v1-backlog-closeout/`（2026-08-22 基线 `ee21350` 全部通过）

本文是 v1.0 冻结与发布的唯一状态口径。业务模块设计文档（`designs/BUSINESS_*`）标注的"V1 冻结目标"不代表已实现，以本文勾选状态为准。

## 1. 编译与测试门禁（全部必过）

- [x] `go build ./...`（2026-08-28 补全 10 个缺失包 + deploy 辅助函数 + metrics/中间件后已通过）
- [x] `go vet ./...`（2026-08-28 通过）
- [x] `go test ./... -count=1`（2026-08-28：51 包 0 失败）
- [ ] `go test -race ./...`（Windows CGO）
- [x] 前端 `npm run build`（2026-08-28 通过；补 ingress/observability 组件、移除 mdqaorder 幽灵引用）
- [x] 前端 `tsc --noEmit`（2026-08-28 通过）
- [ ] overlay 重建测试（business-overlay 契约）

## 2. 迁移与数据

- [ ] 版本化迁移在隔离 schema 演练通过（基线版本 22→25；2026-08-28 已在开发库实测应用 23/24/25，隔离重演待补）
- [ ] 生产快照回滚/重放演练（22→16→22 模式）
- [ ] 活跃键唯一索引零重复（business scope / cmdb host+label / deploy package+template / k8s cluster）
- [ ] system_init.sql 与 gorm tag 与 migrations 三方一致性抽查（2026-08-28 已全量比对核心表模型-数据库列，修复 3 处漂移：rollback 列 / step.changed / task_secret 结构）

## 3. 功能门禁（真机验收，参照评审报告 14 步流程）

- [x] CMDB 添加主机（115.190.251.143 已在 CMDB，host 111，2026-08-28 R1）
- [x] 创建业务域并绑定主机（scope 161 创建 + host 111 绑定，2026-08-28 R1）
- [x] 安装 Nginx / Redis（源码编译方式，2026-08-28 R2：nginx 1.24 任务 136 success / redis 7.2.4 任务 140 success）
- [ ] 安装 JDK（fixed 模板）——2026-08-28 R1/R2：执行链路全通，**真机外网带宽受限导致 180MB 包下载超时（curl 28）**，建议内网镜像源或离线包后补验
- [x] 任务日志与步骤事件可查（2026-08-28 R1：trace 步骤 + 日志流 API 实测）
- [x] 卸载→重装幂等（2026-08-28 R2：任务 137 重复安装 nginx success 且无脏数据、任务 138 卸载 success）
- [x] 制造失败→断点续跑（2026-08-28 R2：3 步模板任务 139，step1/step2 成功后 step3 失败，重试 **只重跑 step3**，checkpoint 生效）
- [ ] K8s 集群纳管（kubeconfig 加密入库、连通性探测）——6443 端口可达，但真机无 kubeconfig 文件，**需用户提供后补验**
- [ ] Release 镜像发布与回滚（依赖 K8s 纳管）
- [x] ~~ArgoCD 应用发布~~（v1.0 下架，移 v1.2，见 `designs/BUSINESS_ARGOCD_MODULE_DESIGN.md`）
- [x] ~~Velero 备份恢复~~（v1.0 下架，移 v1.2，见 `designs/BUSINESS_BACKUP_MODULE_DESIGN.md`）
- [x] 平台自监控（2026-08-28 R2：8090 `/metrics` 返回 212 条 `pantheon_*` 指标；业务指标实时准确——`pantheon_deploy_task_hosts_total{status="success"} 5` 与真机执行记录吻合；`pantheon_hosts_total{state="pending"}` / `pantheon_host_connectivity` 正常输出）

> 真机验收证据：
> - `.harness/evidence/2026-08-28-v1-real-acceptance-r1/summary.md`（R1：部署引擎真实链路 10 项全通）
> - `.harness/evidence/2026-08-28-v1-real-acceptance-r2/summary.md`（R2：源码编译安装 / 幂等 / 卸载 / 断点续跑 四项全通）

## 4. 安全门禁

- [ ] 无明文凭据入库（部署凭据/kubeconfig 均走 `v1:` 加密）
- [x] backup/cicd 路由已下架（2026-08-28 执行：菜单/路由/权限种子摘除，代码保留 v1.2 启用）
- [x] CMDB UpdateStatus 无法绕过 retire 引用检查（2026-08-28 修复，附单测）
- [x] 业务域删除引用检查覆盖主机/应用/集群/任务（2026-08-28 修复，附单测）
- [ ] 危险操作（删资产/删 K8s 资源/执行部署）有权限点+审计记录
- [ ] 生产配置启动强校验（加密密钥、初始管理员密码）

## 5. 发布物

- [ ] 后端二进制 + SHA-256
- [ ] 前端 dist + index.html SHA-256
- [ ] 迁移 SQL 基线固化（版本号记录）
- [ ] docker compose 离线包与安装文档
- [ ] git tag `v1.0.0`，发布说明含已知问题清单（P1 遗留项）

## 6. 流程约束

- 任何提交前本地过 §1 前两项（main 编译门禁是硬约束）。
- AI 会话不得直推 main；一律 feature 分支 + 本检查表 §1 通过后合并。
- 真机验收证据（截图/日志/命令输出）落 `.harness/evidence/<task>/`。
