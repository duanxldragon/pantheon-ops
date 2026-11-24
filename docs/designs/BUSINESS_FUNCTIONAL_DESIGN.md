---
title: Pantheon Ops 全局业务功能设计
doc_type: Design
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-16
---

# Pantheon Ops 全局业务功能设计

目标是 CMDB → BizScope → Application/Service/Instance → Deploy → SSH/K8s 的试点闭环。当前执行与退出条件以[审查整改父计划](../../.harness/tasks/2026-11-20-ops-review-remediation/plan.md)、[执行队列](../../.harness/tasks/2026-11-20-ops-review-remediation/EXECUTION_QUEUE.md)和[当前状态](../../.harness/STATUS.md)为准；原试点计划仅作历史与未验收 residual 来源。业务审查基线为 691dd455（当时 Base v0.18.7）；本轮计划基线 07d53988 已提交 Base v0.18.9 消费，不能把旧审查结果视为新版验收。

## 11/16 能力检查点与历史任务归属

“已有实现”描述源码与装配；只有同一交付候选的真实业务证据才能证明可交付。

| 业务模块      | 已有能力                                                  | 剩余交付条件                                                                                     |
| ------------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| CMDB          | 主机、分组、标签、状态与引用机制                          | C01/C05 验证部署引用、导入 fixture、隔离迁移与权限；八月 UpdateStatus 缺陷不是当前重新确认的缺陷 |
| BizScope      | 域、资源绑定；新增环境目录与菜单                          | C05 验证环境 CRUD、两域权限与菜单用户流程                                                        |
| Service       | 应用、服务、实例、生命周期与部署回写接口                  | C01 关联/引用一致性；C02 普通 UI 关联及结果回读；C03 编辑、分页与受限角色查询                    |
| Deploy        | 包、模板、任务、SSH、队列/租约/checkpoint、取消重试与回滚 | C01 强校验 service/instance/目标；C02 页面闭环；C05 真执行、并发与恢复                           |
| K8s           | 集群、namespace 绑定、工作负载、镜像 Release 等真实 API   | C04 Service Update 与陈旧更新/删除保护；C05 真集群发布回滚与权限                                 |
| Observability | Prometheus/VM、Loki、Tempo 等 provider 查询与告警实现     | E04 对所选 provider 的查询、通知、失败及隔离验收；自动 exporter/file_sd 不在首期范围             |
| FileDist      | 真实 SFTP、校验与排队执行，已有出队取消保护               | E01 创建 UI、分页、在途取消、未来触发与重启收敛；E04 真传输验收                                  |
| ArgoCD/GitOps | 真实 Argo API、手工 Refresh，当前已装配                   | E02 DataScope 行归属、Sync 状态真实性与自动 reconcile；E04 真环境验收                            |
| Velero/Backup | 真实 CRD/reconciler，当前已装配                           | E04 backup/restore/schedule、PVC、失败和归属验收，不沿用“伪实现/下架”口径                        |
| Helm 专用模块 | 记录登记；创建默认 deployed 不代表安装 Chart              | E03 纠正能力与状态口径；真实 Chart 执行只有明确范围才新增                                        |

## T01 冻结边界（2026-11-20-ops-review-baseline-and-decisions）

本批来源（独立评审 + 真实环境清单）在当前 HEAD 逐条核实后，冻结以下边界（详见 [T01 证据](../../.harness/evidence/2026-11-20-ops-review-baseline-and-decisions/baseline-verification-20261123.json)）：

- **多租户目标沿用报告**：按基础平台四分类（platform-global / tenant-owned / tenant-overridable / derived）登记业务表。当前 HEAD 仅 observability 含 `TenantID`；核心业务表（主机/集群/凭据/部署任务及明细/bizscope/service）均无租户字段，为 T04 核心隔离主范围；argocd/filedist/helm 等扩展表为 T14 扩展隔离主范围。T04 迁移编号 **000041**、T14 **000042**，均不与存量迁移（34/35/36/39/40）冲突。
- **租户唯一键变体**：`uk_deploy_credential_name_deleted`、`uk_ip_deleted`、`uk_k8s_cluster_code`、bizscope name、argocd/filedist/velero 唯一键等全局唯一键在租户化时改为 `(tenant_id, 原键, deleted_at)` 组合，防止跨租户撞名。
- **凭据归属模型冻结**：采用"凭据按资源归属授权（业务域/租户作用域）"；不采用全局共享。T05 按此承接模型与 UI。
- **Agent 执行器冻结**：无完整执行协议，本期**关闭入口**并对旧数据启动拒绝启动；不对外宣称 Agent 能力。T09 承接该契约闭环。
- **资产关系（Pod→Node→VM）**：当前代码无实体承接（NEW-02），**收窄定位**，登记为独立后续能力，不进入本批。
- **Helm 真实安装 / Nacos / RocketMQ**：当前 Helm 为 DB 登记快照（NEW-03，设计如此），真实 Chart 安装、Nacos/RocketMQ 均登记为独立后续能力；本批不宣称真实 Helm 安装能力。

以上 4 项产品边界为规划假设，待维护者最终范围验收（T01 Human Gate）。

## 首期流程和边界

1. CMDB 资源与业务域、环境绑定，应用/服务建立可见实例目标。
2. 部署创建、更新、启动在副作用前验证关联和目标一致性，保留历史快照。
3. 有权限用户从 UI 选择实例、预览并执行部署，终态回写相同实例，日志与审计可追溯。
4. K8s 已绑定 namespace 的写操作带冲突保护，发布与回滚按真实外部结果验收。
5. C05 固定候选执行隔离迁移/真 SSH/K8s/权限/失败恢复与桌面窄屏验收，C06 输出可复现交付包并演练安装升级回滚。

六个核心包属于首期；四个扩展包按实际开放能力激活。当前扩展已装配，若试点不交付，必须在候选菜单、路由/权限、可见能力与说明中实施并验证裁剪，不能只在文档列 Out。开放 ArgoCD 就必须关闭 E02 的 P0。

实时健康平台、自动 exporter/file_sd 纳管、商业 License/ModuleGate、租户生产迁移属于后续范围。Service 的当前 reconcile 是超时收敛，不能以此认定首期必须新增完整健康平台。

## 状态、权限与证据

业务状态机以对应模块源码和模块设计为准；关联部署 ID 缺失、错配、失效引用、重复/乱序回调、陈旧写和在途取消均是任务中的验收反例。部署普通表单目前未提交实例关联；关联检查不足的影响是静态高置信结论，具体误回写或越权尚未动态复现。

认证、IAM、组织、通用审计、加密、UI 规范沿用 Base 合同，Ops 不维护副本。业务域负责 capability owner、DataScope、namespace/target 验证及业务菜单/权限/i18n/审计同步。旧设计中的 ArgoCD/Velero 明文凭据结论不能自动套用当前实现。

菜单已改为业务域父级与主机绑定/环境子级、服务目录与应用/服务/实例子级；近期 menu-m3c 任务保留 in-progress，当前 rendered/E2E gap 由 C05 承接。普通编译/单测与现有 CI Smoke Sanity 不替代这些业务验收。

## 追溯

- [当前计划](../../.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-delivery/plan.md)、[执行队列](../../.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md)、[任务状态](../../.harness/STATUS.md)。
- [归档索引](../../.harness/ARCHIVE.md)：12 个历史任务按原范围关闭，119 个原证据文件逐文件 hash 保全。
- [八月 V1 检查点](../V1_RELEASE_CHECKLIST.md)和[历史开发估算](./BUSINESS_DEVELOPMENT_PLAN.md)已被当前计划接管；OPEN 条目不能因为任务归档而视为通过。
