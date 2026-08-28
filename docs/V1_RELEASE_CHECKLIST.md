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

- [ ] 版本化迁移在隔离 schema 演练通过（基线版本 22）
- [ ] 生产快照回滚/重放演练（22→16→22 模式）
- [ ] 活跃键唯一索引零重复（business scope / cmdb host+label / deploy package+template / k8s cluster）
- [ ] system_init.sql 与 gorm tag 与 migrations 三方一致性抽查

## 3. 功能门禁（真机验收，参照评审报告 14 步流程）

- [ ] CMDB 添加主机（115.190.251.143 或等效隔离主机）
- [ ] 创建业务域并绑定主机
- [ ] 安装 JDK / Nginx / Redis（fixed 模板）
- [ ] 任务日志与步骤事件可查（SSE/轮询）
- [ ] 卸载→重装（幂等验证：已安装组件跳过）
- [ ] 制造失败→断点续跑（失败 step 重试，已成功 step 不重跑）
- [ ] K8s 集群纳管（kubeconfig 加密入库、连通性探测）
- [ ] Release 镜像发布与回滚
- [ ] ~~ArgoCD 应用发布~~（v1.0 下架，移 v1.2，见 `designs/BUSINESS_ARGOCD_MODULE_DESIGN.md`）
- [ ] ~~Velero 备份恢复~~（v1.0 下架，移 v1.2，见 `designs/BUSINESS_BACKUP_MODULE_DESIGN.md`）
- [ ] 平台自监控（/metrics + Prometheus 抓取正常）

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
