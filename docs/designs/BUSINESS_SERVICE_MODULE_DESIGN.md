---
title: 服务目录业务模块设计
doc_type: Design
layer: business/service
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-24
---

# 服务目录业务模块设计

English version: [BUSINESS_SERVICE_MODULE_DESIGN.en.md](./BUSINESS_SERVICE_MODULE_DESIGN.en.md)

> 状态说明：Active（V1 业务身份层）

## 1. 目标

服务目录提供 Ops 最小且稳定的业务身份链：

```text
BizScope -> Application -> Service -> ServiceInstance -> VM/K8s Target
```

它回答“哪个业务、哪个应用、哪个服务、在哪个环境运行到哪个目标”，但不复制 CMDB 主机或 Kubernetes 对象的全量事实。

## 2. 所有权与边界

- `Application` 属于一个 BizScope 和部门，表达可交付业务系统。
- `Service` 属于一个 Application，表达稳定服务身份和 runtime type。
- `ServiceInstance` 属于一个 Service，表达某环境中的一个具体运行实例。
- VM 资源事实由 CMDB 拥有；K8s 运行态由 Kubernetes API 拥有。
- Deploy 只能通过 `ServiceInstanceStateCommand` 回写实例状态，不能直接更新 Service 表。
- Service 只能通过 BizScope、CMDB、K8s typed capability 验证引用，不能查询其他 owner 的表或 GORM Model。

## 3. 核心对象

### 3.1 Application

关键字段：`code`、`name`、`business_scope_id`、`business_scope_name`、`dept_id`、`status`、`owner`、`remark`、审计字段。

不变量：

- `code` 在同一业务约束下稳定且唯一。
- 新建和更新必须验证 BizScope 存在、active 且在当前 DataScope 内。
- 删除前检查 Service 引用，不静默级联删除服务历史。

### 3.2 Service

关键字段：`application_id`、`code`、`name`、`runtime_type`、`description`、`status`、审计字段。

不变量：

- Service 的 scope 由 Application 继承，不能跨 Application/BizScope 移动实例。
- 停用 Service 后不得创建新实例或新 Deploy 任务，但历史可查询。
- 删除前检查 ServiceInstance 和 Deploy 引用。

### 3.3 ServiceInstance

关键字段：

- 身份：`service_id`、`environment`、`status`
- target：`target_type`、`host_id` 或 `k8s_cluster_id/namespace/workload_kind/workload_name`
- version：`desired_version`、`current_version`、`rollback_version`
- lifecycle：`desired_state`、`observed_state`、`health_state`、`lifecycle_version`
- observation：`health_definition`、`health_observed_at`、`health_message`、`health_revision`
- correlation：`last_transition_id`、`last_transition_at`

## 4. Target 规则

### VM

- `target_type=vm`。
- 必须且只能设置一个 `host_id`。
- K8s target 字段必须为空。
- Host 必须存在、在同一 BizScope/DataScope 中，且未被 retire/delete。

### Kubernetes

- `target_type=k8s`。
- 必须设置 `k8s_cluster_id`、`namespace`、`workload_kind`、`workload_name`。
- `host_id` 必须为空。
- Cluster 与 NamespaceBinding 必须与 Application 的 BizScope、环境和 allowed action 一致。

### 通用

- mixed target、missing target 和跨 scope target 一律拒绝。
- 同一 Service/environment/target 的有效实例必须唯一。
- target 校验通过 owner capability 完成；错误使用稳定 `business.service.*` key。

## 5. 状态模型

ServiceInstance 同时维护三条独立状态轴：

| 状态轴   | 值                                                                                                      | Owner                |
| -------- | ------------------------------------------------------------------------------------------------------- | -------------------- |
| desired  | `stopped`, `running`, `retired`                                                                         | 用户/变更意图        |
| observed | `unknown`, `installing`, `stopped`, `starting`, `running`, `stopping`, `upgrading`, `failed`, `retired` | Deploy/K8s reconcile |
| health   | `unknown`, `healthy`, `degraded`, `unhealthy`                                                           | 健康观测             |

规则：

- desired 不能被外部观测隐式改写。
- observed/health 更新必须携带 transition/revision，拒绝陈旧回写。
- `lifecycle_version` 用于 CAS；所有转换记录 actor、action、from/to、reason 和 transition ID。
- Deploy 失败不能让实例长期停留在错误的 `running` 投影；任务终态与状态回写使用事务/outbox 等价边界。
- Reconcile 只根据 owner 提供的当前事实修正 observed/health，不创建新的用户意图。

## 6. API

前缀：`/api/v1/business/service`

| Method         | Path                       | 功能                      |
| -------------- | -------------------------- | ------------------------- |
| GET/POST       | `/applications`            | Application 列表/创建     |
| GET            | `/applications/options`    | 授权范围内的选项          |
| PUT/DELETE     | `/applications/:id`        | 更新/删除                 |
| GET/POST       | `/services`                | Service 列表/创建         |
| GET            | `/services/options`        | 按 Application 获取选项   |
| PUT/DELETE     | `/services/:id`            | 更新/删除                 |
| GET/POST       | `/instances`               | ServiceInstance 列表/创建 |
| GET/PUT/DELETE | `/instances/:id`           | 详情/更新/删除            |
| PATCH          | `/instances/:id/state`     | 显式状态转换              |
| POST           | `/instances/:id/reconcile` | 与 owner 当前事实调和     |

列表、详情和 mutation 分别授权；所有查询和写入都携带 DataScope。删除和状态转换必须有操作审计。

## 7. 跨模块能力

- `BizScopeReader`：验证业务域和获取最小引用。
- `CMDBTargetReader`：验证 VM target 和 ownership。
- `K8sTargetReader`：验证 cluster/namespace/workload target。
- `ServiceInstanceStateCommand`：供 Deploy 以版本化请求回写 observed/health/version。
- `ServiceInstanceReferenceReader`：供 CMDB 删除/变更前检查 ServiceInstance 引用。

Capability DTO 只包含 ID、stable key、scope、状态、revision 和必要摘要。

## 8. UI

当前 `ServiceList` 是服务目录工作台，按 Application、Service 和 ServiceInstance 展示层级关系。

- 页面先显示业务域、owner、环境和状态，再显示 target 与版本。
- target 创建使用结构化 VM/K8s 选择器，不接受自由文本 locator。
- desired、observed 和 health 使用不同标签，不合并成一个“运行状态”。
- target 不可访问、引用失效、409 冲突和 reconcile 失败有独立页面状态。
- 危险状态转换展示实例、目标、当前/目标状态和影响；提交期间防止重复操作。
- 列表/详情在窄屏保持状态与恢复入口可见，宽表格使用受控滚动。

## 9. 兼容与迁移

- Host `installed_components` 只是兼容性检查投影，不是 Service 真相源。
- Service 模块不双写 Host JSON。
- 任何回填或 target schema 迁移必须使用版本化 migration，并提供可重复校验。
- P1 Resource 模型上线时，ServiceInstance 迁移到稳定 ResourceRef，但保留兼容读取窗口。

## 10. 验收

- Application/Service/Instance CRUD 在权限和 DataScope 内闭环。
- VM/K8s target 的 valid、missing、mixed、cross-scope、inactive 和 stale 用例全部覆盖。
- 状态转换与 reconcile 使用 CAS，重复/乱序回写结果确定。
- Deploy 成功、失败、取消和重试对 ServiceInstance 的投影符合状态合同。
- 删除引用检查阻止孤儿 target 或历史。
- 菜单、权限、i18n、审计、migration、type-check、build 和业务边界检查通过。

## 11. 验证入口

- `go test -count=1 ./modules/business/service/...`
- `go test -race ./modules/business/service/...`
- `go vet ./...`
- `npm run type-check`
- `npm run build`
- `npm run check:menu-contract`
- `node scripts/check-business-module-boundaries.mjs`

发布状态仍以 [V1 冻结检查表](../V1_RELEASE_CHECKLIST.md) 为准。
