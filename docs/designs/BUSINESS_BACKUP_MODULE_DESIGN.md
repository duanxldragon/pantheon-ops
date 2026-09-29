---
title: Velero 备份恢复业务模块设计
doc_type: Design
layer: business/k8s/backup
status: Draft
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-28
---

# Velero 备份恢复业务模块设计

> 状态说明：Draft（v1.2 目标；当前代码为壳级实现，菜单与路由在 V1 冻结版下架，见 `.harness/v1.2-docs/V1_FREEZE_REMEDIATION_DEV_PLAN.md` Phase 0-2）

本文定义基于 Velero 的 Kubernetes 备份恢复设计。2026-08-28 评审确认当前代码未操作任何 Velero CRD（`tryCreateVeleroBackupCRD` 为空函数、状态永远停在 New），属 P0 伪实现，本文是整改后的目标设计。

## 1. 模块定位

负责 K8s 资源的备份计划、执行、状态跟踪与恢复，是运维生命周期闭环（创建→部署→运行→监控→**备份→恢复**）中缺失的一环。

不负责：ETCD 快照、VM 级备份（属主机备份域，V1 不做）、跨集群迁移。

## 2. 数据模型

四表结构保留（`27209f5` 已建），语义修正：

| 表 | 关键字段 | 语义 |
|---|---|---|
| `biz_velero_backup` | cluster_id, name, namespace_include/exclude, resource_include, include_volumes, ttl, status(执行态), velero_backup_name, business_scope_id | 一次备份执行记录；status 由 reconciler 从 CRD 回传，禁止本地写终态 |
| `biz_velero_schedule` | cluster_id, name, cron, backup_spec(引用 backup 模板), enabled, last_run | 定时计划；enabled=false 只删除本地记录不删 CRD |
| `biz_velero_restore` | backup_id, target_cluster_id, namespace_mapping(JSON), status | 一次恢复执行记录 |
| `biz_velero_storage_location` | name, provider, bucket, endpoint, credential_ref, default | 存储位置；凭据走 CredentialRef 加密，禁止明文 SecretAccessKey（当前 P1 缺陷） |

业务关联链：业务域→K8s Cluster（已挂 BusinessScopeID）→Namespace→按 namespace 维度圈定备份范围。备份记录继承 cluster 的业务域归属，DataScope 可见性与其他 K8s 资源一致。

## 3. 生命周期（真实实现要求）

```
注册 StorageLocation（探测可达后保存）
→ 创建 BackupPlan/Schedule（创建 Velero Schedule CRD，成功才落库）
→ 手动备份（创建 Backup CRD）
→ Reconciler 轮询 CRD 状态（New→InProgress→Completed/PartiallyFailed/Failed）
→ 恢复（创建 Restore CRD，支持 namespace_mapping 重定向）
→ 删除（删除 CRD + 本地记录）
```

实现依赖：`client-go` dynamic client 操作 `velero.io/v1` API（Backup/Schedule/Restore/BackupStorageLocation CRD）。前提：目标集群已安装 Velero（检测逻辑已有：velero namespace 探测，保留）。

## 4. 恢复能力矩阵（v1.2 目标）

| 能力 | 支持 | 方式 |
|---|---|---|
| Namespace 整体恢复 | ✓ | Restore CRD + namespace include |
| Deployment/Service 恢复 | ✓ | resource include filter |
| PVC 恢复 | ✓ | include_volumes=PodVolume 或 snapshot（取决于安装模式） |
| Secret/ConfigMap 恢复 | ✓ | resource filter |
| 跨 namespace 重定向 | ✓ | namespace_mapping |
| 跨集群恢复 | 不做 | v1.2 明确排除 |

## 5. 安全要求

1. `secret_access_key` 走统一 `v1:` 加密（当前明文属 P1 缺陷）。
2. 路由注册入 `DataScopedGroup`（当前裸路由属 P0 缺陷）。
3. 恢复是危险操作：权限键 `business:k8s:backup:restore` + SecureAction 二次确认 + 强制审计（记录操作者、backup id、目标 namespace）。

## 6. 验收标准（v1.2）

1. 真实集群（k3s + velero + MinIO 作为对象存储）完成：创建计划→触发备份→删除命名空间下工作负载→恢复→工作负载 Ready。
2. 恢复后 PVC 数据可读写（部署带 PVC 的应用验证）。
3. 备份失败场景（对象存储不可达）：状态回传 Failed，错误信息可见，无伪造 Completed。
4. Schedule 到期自动触发备份（观察一个周期）。
