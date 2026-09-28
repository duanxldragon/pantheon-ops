# 安装部署业务模块设计

English version: [BUSINESS_DEPLOY_MODULE_DESIGN.en.md](./BUSINESS_DEPLOY_MODULE_DESIGN.en.md)

更新时间：2026-08-24

类型：Design
归属层：business/deploy
状态：Active（V1 冻结目标；实现和发布状态以 `../V1_RELEASE_CHECKLIST.md` 为准）

本文定义运维平台第二个业务模块：安装部署（Deploy）。它与 `business/cmdb`、`business/bizscope` 平级，挂在“工作台 / 运维平台”菜单下，用 CMDB 主机和业务域作为目标来源，为后续 Agent 执行、软件安装、任务编排、凭据托管和变更审计打基础。

本文按 Pantheon Base 当前 canonical 收口：

- 跨业务模块只允许通过显式 capability / facade 查询，不允许直读对方表
- 业务权限统一使用 `business:<module>:<resource>:<action>`
- 视觉 token 数值以 `pantheon-base/docs/designs/THEME_TOKENS_REFERENCE.md` 为准
- 响应式断点与退化行为以 `pantheon-base/docs/designs/MOBILE_RESPONSIVE_BREAKPOINTS.md` 为准

本文同时吸收 `.harness/v1.2-docs/V1_FREEZE_REMEDIATION_DEV_PLAN.md` 的 FZ-01 至 FZ-04 合同。下文标注的 V1 冻结目标不代表已经实现；与历史“第一版”描述冲突时，以冻结目标和 V1 检查表为准。

## 1. 模块概述

安装部署模块负责把“要装什么、装到哪里、执行到什么状态、结果如何”沉淀成可追踪任务。

本模块解决：

- 维护可安装的软件组件。
- 维护软件组件、任务模板和部署任务三层对象。
- 维护固定模板任务和可编排任务两类模板定义。
- 基于业务域下的 CMDB 主机或分组创建部署任务。
- 记录任务和每台目标主机的执行状态。
- 记录任务和每台目标主机的执行过程步骤。
- 支持上传源码包，满足内网离线部署场景。
- 保留手动/模拟/SSH 基线，并在 V1 收口为统一 provider、持久化 checkpoint 和可恢复编排。
- 通过认证 SSE 提供追加式、脱敏、可重连的任务事件与日志。
- Agent、Kubernetes 和 API provider 仅作为未注册扩展点，不提供伪成功实现。

本模块不负责：

- 主机台账和分组维护，属于 `business/cmdb`。
- 实时连通性监控，属于后续监控能力。
- 凭据明文托管；本模块只消费 `CredentialRef` 和安全 metadata。
- Agent 进程本身；V1 只定义 provider 扩展点。
- 回退的独立执行入口，当前仍作为后续生命周期能力扩展，不在第一版主链路内。
- 平台用户、角色、菜单、权限实现，属于 `system/iam`。

## 2. 边界与依赖

| 类别              | 允许依赖                                                 | 禁止依赖                                                                               |
| :---------------- | :------------------------------------------------------- | :------------------------------------------------------------------------------------- |
| platform          | `pkg/common`、统一响应、审计元数据、模块注册契约         | 修改平台壳层业务逻辑                                                                   |
| system/auth       | `gin.Context` 中登录主体                                 | 直接 import auth Service                                                               |
| system/iam        | 菜单、权限点、Casbin 策略结果                            | 业务模块内写角色授权逻辑                                                               |
| system/org        | 数据范围上下文                                           | 直接依赖 org repository                                                                |
| business/cmdb     | 通过显式只读查询 capability 获取可见主机、分组和目标快照 | 直读 `biz_cmdb_*` 表、直接依赖 CMDB repository/service、修改 CMDB 内部状态机或标签逻辑 |
| business/bizscope | 通过业务域 ID 和名称约束部署信任边界                     | 绕过业务域直接为未分配主机下发主机级部署                                               |

当前代码基线说明：

- 设计目标仍然是 capability / facade 边界
- 当前仓库已经把 Deploy 的目标解析、主机状态回写、已安装组件写回收敛到 `business/cmdb` 内部 capability
- Deploy 服务层不再直接查询或更新 `biz_cmdb_host`、`biz_cmdb_group`
- 后续仍可继续把 capability 注册方式做成更统一的跨模块契约，但当前边界已经从“直读直写表”收敛到“显式能力调用”

第一版边界结论：

- Deploy 不得直接读取 `biz_cmdb_host`、`biz_cmdb_group` 表结构
- Deploy 只能消费 `business/cmdb` 暴露的只读查询契约，用于目标选择、成员预览和任务快照
- Deploy 不得更新 CMDB 主机配置、分组条件、标签规则或运维状态机

### 2.1 CMDB 查询契约

Deploy 允许消费的最小能力面建议固定为：

| capability             | 输入                                  | 输出                    | 用途               |
| :--------------------- | :------------------------------------ | :---------------------- | :----------------- |
| `ListSelectableHosts`  | 关键字、状态、数据范围、分页          | Host 摘要列表           | 任务目标选择       |
| `ListSelectableGroups` | 关键字、数据范围                      | Group 摘要列表          | 分组目标选择       |
| `PreviewGroupMembers`  | `groupIds[]`、数据范围                | 分组成员数量 / 主机摘要 | 表单预览           |
| `ResolveTaskTargets`   | `targetType`、`targetIds[]`、数据范围 | 去重后的 Host 快照列表  | 启动任务时固化目标 |

契约要求：

- 所有结果都必须经过 `DataScopeReq` 或等价上下文过滤
- Deploy 只拿“当前任务执行需要的快照字段”，不反向绑定 CMDB 的内部 JSON 结构
- 该 capability 后续可以落到 `pkg/contracts`、模块 facade 或独立 query service，但不能只停留在口头约定

## 3. 菜单与命名

一级归属：`运维平台`。

二级模块：`安装部署`。

菜单中文保持四字：

| 菜单 key                      | 路径                        | titleKey                          | routeName              | module            | component key                                 | pagePermission                  | activeMenu              | 类型 | 说明                     |
| :---------------------------- | :-------------------------- | :-------------------------------- | :--------------------- | :---------------- | :-------------------------------------------- | :------------------------------ | :---------------------- | :--- | :----------------------- |
| `business.deploy.package`     | `/business/deploy/package`  | `operations.deploy.package.menu`  | `deploy-package-list`  | `business.deploy` | `business/deploy/package/DeployPackageList`   | `business:deploy:package:view`  | —                       | `C`  | 软件组件列表             |
| `business.deploy.template`    | `/business/deploy/template` | `operations.deploy.template.menu` | `deploy-template-list` | `business.deploy` | `business/deploy/template/DeployTemplateList` | `business:deploy:template:list` | —                       | `C`  | 任务模板列表             |
| `business.deploy.task`        | `/business/deploy/task`     | `operations.deploy.task.menu`     | `deploy-task-list`     | `business.deploy` | `business/deploy/task/DeployTaskList`         | `business:deploy:task:view`     | —                       | `C`  | 部署任务列表             |
| `business.deploy.task.detail` | `/business/deploy/task/:id` | `operations.deploy.task.detail`   | `deploy-task-detail`   | `business.deploy` | `business/deploy/task/DeployTaskDetail`       | `business:deploy:task:detail`   | `/business/deploy/task` | `C`  | 任务详情页，不进侧边菜单 |

约束：

- `module` 固定为 `business.deploy`
- 菜单只表达导航，不表达执行规则
- 列表页和详情页的页面权限分离：列表页走 `view`，详情页走 `detail`
- 任务模板页使用独立模板模型：模板头 + 模板步骤，任务创建优先选模板，软件组件保留为底层可复用资产

## 4. 核心对象

| 对象               | 说明                  | 关键字段                                                                                                                 |
| :----------------- | :-------------------- | :----------------------------------------------------------------------------------------------------------------------- |
| DeployPackage      | 软件组件              | name、version、execution_mode、template_code、source_object_key、install_command、uninstall_command、status              |
| DeployTemplate     | 任务模板              | name、version、category、default_action、package_id、parameter_schema、status                                            |
| DeployTemplateStep | 模板步骤              | template_id、step_code、step_name、step_type、action、package_id、template_params、sort                                  |
| DeployTask         | 部署任务              | name、package_id、business_scope_id、action、target_type、executor_type、execution_mode、status、started_at、finished_at |
| DeployTaskHost     | 主机执行明细          | task_id、host_id、host_ip、status、attempt_no、state_version                                                             |
| DeployTaskAttempt  | 一次不可变执行尝试    | task_id、attempt_no、mode、status、worker、时间                                                                          |
| DeployTaskStep     | 持久化步骤 checkpoint | task/host、step_key、依赖、条件、failure_policy、status、attempt_no、state_version                                       |
| DeployTaskEvent    | 追加式状态事件        | sequence、event_type、from/to、actor、request_id、reason                                                                 |
| DeployTaskLogChunk | 追加式脱敏日志        | task/host/step、sequence、stream、content、timestamp                                                                     |

## 5. 状态流转与编排

### 5.1 Task 状态机

| From                  | Allowed next states                                        |
| :-------------------- | :--------------------------------------------------------- |
| `pending`             | `running`, `canceled`                                      |
| `running`             | `pause_requested`, `cancel_requested`, `success`, `failed` |
| `pause_requested`     | `paused`, `cancel_requested`, `failed`                     |
| `paused`              | `running`, `cancel_requested`                              |
| `cancel_requested`    | `canceled`, `failed`                                       |
| `failed`              | `running`，仅通过 Retry 创建新 attempt                     |
| `success`, `canceled` | 无                                                         |

`draft` 只用于尚未提交执行的编辑态；进入 `pending` 后由上述状态机管理。所有 task/host/step 转换必须通过一个 CAS service function，并校验 `state_version`。业务代码禁止直接 `Updates({"status": ...})`。

### 5.2 Step 与 Attempt

- Step 状态：`pending`、`running`、`success`、`failed`、`skipped`、`canceled`。
- Retry 创建新 attempt，不修改上一成功 attempt 的输出。
- `failed_step` 只从失败步骤继续；`failed_host` 重新调度失败主机的未完成步骤。
- Pause 停止新步骤调度，在当前步骤返回后进入 `paused`。
- Cancel 传播 context cancellation，并把未开始步骤标为 canceled。
- Worker 重启调和过期 attempt，保留成功 checkpoint，只调度未完成工作。

### 5.3 Provider 与编排

- V1 注册 `manual`、`simulated`、`ssh` provider；Agent、Kubernetes、API 保持未注册扩展点。
- SSH provider 独占拨号、host-key 校验、文件传输、命令执行和取消；业务 Service 不直接调用 SSH。
- Start 前持久化并校验 DAG，拒绝环和缺失依赖。
- condition：`always`、`on_success`、`on_failure`、`param_equals`。
- failure policy：`fail_fast`、`continue_hosts`、`continue_steps`。
- 默认行为：ordered steps、`fail_fast`、有界 host 并发；只有所有 required host/step 成功时 task 才是 `success`。

### 5.4 日志与事件

- 每个状态变化写追加式 `DeployTaskEvent`，记录单调 sequence、from/to、actor、request ID 和 reason。
- stdout/stderr 先经过统一脱敏，再按最大 16 KiB 写入 `DeployTaskLogChunk`；大日志分段，不覆盖 task row。
- UI 通过认证 SSE `GET /tasks/:id/events?after=<sequence>` 获取事件、heartbeat 和断线重连。
- 历史 `trace_steps/stdout/stderr` 只作为迁移兼容字段，不能继续作为 V1 canonical checkpoint/log store。

### 5.5 组件编排模型（声明式组件 provider，V1 核心）

安装部署是 pantheon-ops 最核心的基本功能。V1 冻结把 Deploy 从"脚本执行工具"升级为**声明式组件编排**：模板 = 一张**经过校验的组件编排图**，每个步骤引用一个**幂等组件 provider**，组件自带 install / uninstall / healthcheck，由 healthcheck 判定 `changed/ok`，形成可复用、可编排、幂等、可审计的安装单元。**明确不引入图灵完备的模板 DSL**（不重造 Ansible 的 loops / tags / blocks / roles / Jinja2）。

> 本节是 V1 冻结目标。标注"已有"的能力已在代码落地；标注"V1 补"的能力是冻结前必须补齐的缺口，实现与发布状态以 `../V1_RELEASE_CHECKLIST.md` 与 `FZ-07` 为准。

#### 5.5.1 概念映射（Ansible 价值 ↔ pantheon-ops 对应）

| Ansible 概念             | pantheon-ops 对应                                             | 状态 |
| :----------------------- | :------------------------------------------------------------ | :--- |
| Playbook                 | `DeployTemplate`（`execution_mode=orchestrated`，组件编排图）   | 已有 |
| Inventory（hosts/groups）| CMDB 主机 + 标签动态分组 + 业务域                               | 已有 |
| Task                     | `DeployTemplateStep`（引用组件类型 + 参数）                    | 已有 |
| Module                   | `ComponentProvider`（幂等组件 provider）                       | V1 补 |
| Variable                 | 受控 `{{ var }}` 字面替换（白名单参数 + CMDB 快照）             | V1 补 |
| Condition（when）        | typed condition 枚举 + 新增 `on_changed`                       | 半有 |
| Changed/Ok（幂等）       | `StepResult.Changed` + 组件 healthcheck                         | V1 补 |
| Handler/notify           | 用 `on_changed` 条件 + DAG 依赖替代（不引入 handler 对象）       | 替代 |
| 任意自动化逃生舱         | `script` 步骤（裸 shell，可跑 `ansible-playbook`）              | 已有 |

**明确不在 V1**：loops、tags、blocks/rescue/always、roles、register、通用模板引擎（text/template / Jinja2）——这些是 Ansible 的重量，不是安装部署的刚需。

#### 5.5.2 对象模型扩展

- 新增 `ComponentProvider` registry（对齐 `executor.Registry` 的注册 / 查找模式）：契约含 `Type() / Name() / Validate() / Resolve(ComponentInput) (ComponentPlan) / HealthCheck(ComponentOutput) (Changed)`。
- `DeployTemplateStep.step_type` 从 `package/script` 扩展为对 `ComponentProvider` 的类型引用；nginx / redis / jdk / mysql / minio / harbor / prometheus 七个固定模板迁移为组件 provider 实现，保留旧固定模板迁移兼容入口。
- 不新增 `DeployTemplateHandler`、`loop`、`tags`、`register`、`block`、`changed_when` 等字段；`changed` 由组件 healthcheck 判定，不引入表达式。

#### 5.5.3 变量与受控替换

- 来源：`template_params`（白名单校验）+ CMDB 主机快照（hostname / OS / CPU / 内存 / 磁盘 / IP，只读）。
- 替换：**字面 `{{ key }}` 替换**，无函数 / 过滤器 / 表达式 / 管道；键名白名单 `^[A-Za-z_][A-Za-z0-9_.]*$`。
- `secret_params` 只在结构化传输中引用（如 SFTP mode 0600 的配置文件），**禁止进入 shell 文本**（见 FZ-02）。
- 作用：组件 provider 在 `Resolve` 阶段用受控替换生成最终 install / uninstall 命令与配置；所有输出经统一脱敏与注入校验后才传给 executor。

#### 5.5.4 幂等组件 provider（ComponentProvider）

| 组件类型     | 用途                                    | 幂等判定（healthcheck）        |
| :----------- | :-------------------------------------- | :----------------------------- |
| `nginx`      | Nginx 源码安装 / systemd 服务           | 服务 active + 端口 / 版本探测   |
| `redis`      | Redis 安装 / 配置（requirepass + 绑定） | `AUTH` 后 PONG + 端口探测       |
| `jdk`        | JDK 安装 / 环境变量 / `java -version`    | 版本探测 + `JAVA_HOME` 存在     |
| `mysql`      | MySQL 离线初始化 / root 密码            | 连接探测 + 数据目录存在         |
| `minio`      | MinIO 单机 / API 端口                   | 端口探测 + 进程存在             |
| `harbor`     | Harbor 离线部署                         | 服务 active + 健康端点          |
| `prometheus` | Prometheus 安装 / systemd               | 端口探测 + `/metrics` 可达      |
| `script`     | 任意自动化逃生舱（裸 shell，可 ansible）| 由脚本退出码 / 用户声明         |
| `file`       | 文件 / 配置上传（SFTP mode 0600）        | 目标文件内容比对                |
| `service`    | systemd 服务控制（enable/start/stop）    | 服务状态探测                    |

- 每个组件 provider 实现统一契约：`Resolve` 产出幂等的 install / uninstall 计划，`HealthCheck` 判定 `changed/ok`。
- 组件不得持久化 secret；日志经 `LogSink` 脱敏。
- 幂等原则：install / uninstall 本身幂等（重复执行不产生副作用）；`HealthCheck` 通过则 `Changed=false`（`ok`），否则 `Changed=true`（`changed`）。

#### 5.5.5 执行语义

- **changed / ok**：`StepResult` 增加 `Changed bool`；组件 `HealthCheck` 通过则 `ok`（不重复执行副作用），否则执行 install 并报 `changed`。
- **on_changed 条件**：新增到 typed condition 枚举，替代 handler / notify——"仅当上游步骤 `changed` 时才执行"（如"仅当配置变化才重启服务"），与 DAG 依赖组合覆盖 notify 语义，不引入 handler 对象。
- **保留现有能力**：DAG 依赖、typed condition（always / on_success / on_failure / param_equals / on_changed）、`fail_fast / continue_hosts / continue_steps`、`ContinueOnHostFailure`、有界并行、协作取消、frozen 快照、持久化 checkpoint、CAS 状态机、SSE 均不变。

#### 5.5.6 验收标准（组件编排）

- 一个 `orchestrated` 模板能声明多步骤组件、参数、condition（含 `on_changed`）、依赖、失败策略，并整体启动 / 暂停 / 恢复 / 取消 / 重试。
- 组件 provider 的 install / uninstall 幂等；`HealthCheck` 通过时重跑返回 `ok`（`Changed=false`），不重复执行副作用。
- `on_changed` 条件只在上游 `changed` 时触发下游，与 DAG 依赖组合可表达"配置变化才重启"。
- 受控 `{{ var }}` 替换对引号、反引号、`$()`、`{{`、`}}`、未声明键的对抗输入全部拒绝或转义，无函数求值。
- `script` 逃生舱可执行任意脚本（含 `ansible-playbook`），但其输出不参与组件幂等判定。
- 组件输出结构化，`changed/ok` 与 task 终态一致，不产生虚假 success。

## 6. 数据模型

### 6.1 biz_deploy_package

| 字段                                 | 类型         | 说明                                                                                                  |
| :----------------------------------- | :----------- | :---------------------------------------------------------------------------------------------------- |
| id                                   | bigint       | 主键                                                                                                  |
| name                                 | varchar(128) | 组件名称                                                                                              |
| version                              | varchar(64)  | 版本                                                                                                  |
| description                          | varchar(512) | 说明                                                                                                  |
| install_command                      | text         | 安装命令模板                                                                                          |
| uninstall_command                    | text         | 卸载命令模板                                                                                          |
| execution_mode                       | varchar(32)  | `fixed` / `orchestrated`                                                                              |
| template_code                        | varchar(64)  | 固定模板编码，如 `nginx_systemd`、`mysql_systemd`、`redis_systemd`、`minio_systemd`、`harbor_offline` |
| template_config                      | json         | 模板元信息                                                                                            |
| source_object_key                    | varchar(255) | 上传源码包对象键                                                                                      |
| source_file_name                     | varchar(255) | 上传源码包原始文件名                                                                                  |
| source_url                           | varchar(512) | 源码包访问地址                                                                                        |
| status                               | varchar(32)  | enabled / disabled                                                                                    |
| created_at / updated_at / deleted_at | datetime     | 审计字段                                                                                              |
| created_by / updated_by              | varchar(64)  | 操作人                                                                                                |

唯一约束：`uk_deploy_package_name_version_deleted` (`name`, `version`, `deleted_at`)。

### 6.2 biz_deploy_task

| 字段                                    | 类型             | 说明                                                    |
| :-------------------------------------- | :--------------- | :------------------------------------------------------ |
| id                                      | bigint           | 主键                                                    |
| name                                    | varchar(128)     | 任务名称                                                |
| template_id                             | bigint           | 任务模板 ID，可为空                                     |
| template_name / template_version        | varchar          | 创建任务时的模板快照                                    |
| package_id                              | bigint           | 软件组件 ID                                             |
| package_name / package_version          | varchar          | 创建任务时的组件快照                                    |
| business_scope_id / business_scope_name | bigint / varchar | 业务域快照                                              |
| action                                  | varchar(32)      | install / uninstall / upgrade / reinstall               |
| target_type                             | varchar(32)      | host / group                                            |
| target_ids                              | json             | 目标主机或分组 ID                                       |
| executor_type                           | varchar(32)      | manual / simulated / agent / ssh                        |
| execution_mode                          | varchar(32)      | fixed / orchestrated                                    |
| template_params                         | json             | 固定模板参数，如安装目录、服务名                        |
| status                                  | varchar(32)      | draft / pending / running / success / failed / canceled |
| remark                                  | varchar(512)     | 备注                                                    |
| external_task_id                        | varchar(128)     | Agent/外部执行任务 ID                                   |
| started_at / finished_at                | datetime         | 执行时间                                                |
| created_at / updated_at / deleted_at    | datetime         | 审计字段                                                |
| created_by / updated_by                 | varchar(64)      | 操作人                                                  |

### 6.3 biz_deploy_task_host

| 字段                     | 类型         | 说明                                           |
| :----------------------- | :----------- | :--------------------------------------------- |
| id                       | bigint       | 主键                                           |
| task_id                  | bigint       | 部署任务 ID                                    |
| host_id                  | bigint       | CMDB 主机 ID                                   |
| hostname / host_ip / os  | varchar      | 主机快照                                       |
| status                   | varchar(32)  | pending / running / success / failed / skipped |
| trace_steps              | json         | 执行过程步骤                                   |
| stdout / stderr          | text         | 执行输出                                       |
| error_message            | varchar(512) | 错误说明                                       |
| executor_id              | varchar(128) | Agent 或执行器 ID                              |
| started_at / finished_at | datetime     | 执行时间                                       |
| reported_at              | datetime     | Agent 回写时间                                 |
| created_at / updated_at  | datetime     | 审计字段                                       |

### 6.4 biz_deploy_template

| 字段                                 | 类型         | 说明                                      |
| :----------------------------------- | :----------- | :---------------------------------------- |
| id                                   | bigint       | 主键                                      |
| name                                 | varchar(128) | 模板名称                                  |
| version                              | varchar(64)  | 模板版本                                  |
| description                          | varchar(512) | 说明                                      |
| category                             | varchar(64)  | 模板分类                                  |
| execution_mode                       | varchar(32)  | `fixed` / `orchestrated`                  |
| default_action                       | varchar(32)  | install / uninstall / upgrade / reinstall |
| package_id                           | bigint       | 默认软件组件                              |
| package_name / package_version       | varchar      | 默认组件快照                              |
| template_code                        | varchar(64)  | 固定模板编码                              |
| template_config                      | json         | 模板元信息                                |
| parameter_schema                     | json         | 默认参数                                  |
| status                               | varchar(32)  | enabled / disabled                        |
| created_at / updated_at / deleted_at | datetime     | 审计字段                                  |
| created_by / updated_by              | varchar(64)  | 操作人                                    |

### 6.5 biz_deploy_template_step

| 字段                           | 类型         | 说明                                   |
| :----------------------------- | :----------- | :------------------------------------- |
| id                             | bigint       | 主键                                   |
| template_id                    | bigint       | 模板 ID                                |
| step_code                      | varchar(64)  | 步骤编码                               |
| step_name                      | varchar(128) | 步骤名称                               |
| step_type                      | varchar(32)  | `package` / `script`                   |
| action                         | varchar(32)  | 步骤动作                               |
| package_id                     | bigint       | 步骤组件                               |
| package_name / package_version | varchar      | 步骤组件快照                           |
| template_code                  | varchar(64)  | 固定模板编码                           |
| template_params                | json         | 步骤参数                               |
| step_config                    | json         | 脚本正文、前置校验、后置校验等扩展配置 |
| sort                           | int          | 步骤顺序                               |
| created_at / updated_at        | datetime     | 审计字段                               |

### 6.6 biz_deploy_task_attempt

每次 Start/Retry 生成独立 attempt，至少保存 `task_id`、`attempt_no`、`mode`、`status`、`worker_id`、`lease_until`、开始/结束时间和结果摘要。attempt 只追加，不覆盖历史成功输出。

### 6.7 biz_deploy_task_step

至少保存 `task_id`、`task_host_id`、稳定 `step_key`、顺序、类型、动作、dependency/condition JSON、failure policy、status、attempt number、时间、错误 key/message 和 `state_version`。

### 6.8 biz_deploy_task_event

至少保存 task/host/step ID、单调 sequence、event type、from/to status、actor、request ID、reason 和 timestamp。事件只追加。

### 6.9 biz_deploy_task_log_chunk

至少保存 task/host/step ID、单调 sequence、stream、已脱敏 content 和 timestamp。单 chunk 最大 16 KiB；retention 到期按策略清理，不修改历史 task row。

### 6.10 租户就绪判断

- 当前按单租户运行，第一版不新增 `tenant_id`
- `DeployPackage(name, version)` 当前按平台全局唯一理解；若未来出现多租户软件目录隔离，应调整为 `(tenant_id, name, version)` 或等价 scope 唯一
- `DeployTask` 与 `DeployTaskHost` 的列表、导出、审计查询必须保留统一 scope 注入点，不能把“当前只有一个业务空间”写死在 handler 或 repo
- 任务执行审计未来大概率需要按 tenant / business space 检索，因此启动、取消、结果回写的审计结构不能只保留纯全局流水语义

## 7. API 设计

前缀：`/api/v1/business/deploy`

### 7.1 通用契约

- 返回统一走 `common.Success` / `common.Fail`
- 列表接口统一返回：`items / total / page / pageSize`
- 写接口统一返回：资源主键、核心状态字段、最近更新时间
- 仓库级 canonical 清单以 `docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md` 为准；本节保留 Deploy 局部摘要
- 错误 key 前缀统一为：
  - `business.deploy.package.*`
  - `business.deploy.task.*`
  - `business.deploy.taskHost.*`
- 所有接口都必须显式标注对应权限，不允许“页面能进就默认接口都能调”
- 所有写接口都必须进入统一操作审计；启动、取消、结果回写属于高敏动作

### 7.2 接口清单

| 接口                     | 方法   | 说明                                                                                            | 权限                                                                                                                                                 |
| :----------------------- | :----- | :---------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------- |
| `/packages`              | GET    | 软件组件列表                                                                                    | `business:deploy:package:list`                                                                                                                       |
| `/packages/:id`          | GET    | 软件组件详情                                                                                    | 权限映射缺口：路由已实现，但 `business:deploy:package:list` 当前只映射 `GET /packages`，未覆盖本路由；见任务卡 `2026-07-17-business-permission-gaps` |
| `/packages`              | POST   | 新建软件组件                                                                                    | `business:deploy:package:create`                                                                                                                     |
| `/packages/:id`          | PUT    | 编辑软件组件                                                                                    | `business:deploy:package:update`                                                                                                                     |
| `/packages/:id`          | DELETE | 删除软件组件                                                                                    | `business:deploy:package:delete`                                                                                                                     |
| `/templates`             | GET    | 任务模板列表                                                                                    | `business:deploy:template:list`                                                                                                                      |
| `/templates`             | POST   | 新建任务模板                                                                                    | `business:deploy:template:create`                                                                                                                    |
| `/templates/:id`         | GET    | 任务模板详情                                                                                    | `business:deploy:template:list`                                                                                                                      |
| `/templates/:id`         | PUT    | 编辑任务模板                                                                                    | `business:deploy:template:update`                                                                                                                    |
| `/templates/:id`         | DELETE | 删除任务模板                                                                                    | `business:deploy:template:delete`                                                                                                                    |
| `/tasks`                 | GET    | 部署任务列表                                                                                    | `business:deploy:task:list`                                                                                                                          |
| `/tasks`                 | POST   | 创建部署任务                                                                                    | `business:deploy:task:create`                                                                                                                        |
| `/tasks/:id`             | GET    | 任务详情                                                                                        | `business:deploy:task:detail`                                                                                                                        |
| `/tasks/:id`             | PUT    | 编辑未启动任务                                                                                  | `business:deploy:task:update`                                                                                                                        |
| `/tasks/:id`             | DELETE | 删除未启动任务（`draft / pending`，同步清理 task-host 明细）                                    | `business:deploy:task:delete`                                                                                                                        |
| `/tasks/:id/start`       | POST   | 启动任务                                                                                        | `business:deploy:task:start`                                                                                                                         |
| `/tasks/:id/pause`       | POST   | 请求暂停；当前步骤返回后进入 paused                                                             | `business:deploy:task:pause`                                                                                                                         |
| `/tasks/:id/resume`      | POST   | 从 checkpoint 恢复调度                                                                          | `business:deploy:task:resume`                                                                                                                        |
| `/tasks/:id/cancel`      | POST   | 取消任务                                                                                        | `business:deploy:task:cancel`                                                                                                                        |
| `/tasks/:id/retry`       | POST   | 按 failed_step/failed_host 创建新 attempt                                                       | `business:deploy:task:retry`                                                                                                                         |
| `/tasks/:id/events`      | GET    | 认证 SSE；支持 after sequence 重连                                                              | `business:deploy:task:detail`                                                                                                                        |
| `/task-hosts/:id/result` | POST   | 标记主机执行结果                                                                                | `business:deploy:task:mark-result`                                                                                                                   |
| `/task-hosts/:id/report` | POST   | Agent 结果回写预留；当前与 `/result` 绑定同一 handler，“人工标记 vs 执行器上报”来源区分尚未实现 | Agent 阶段待办，第一版不挂菜单                                                                                                                       |

### 7.3 请求 / 响应要点

#### 7.3.1 `GET /packages`

- query：`keyword`、`status`、`page`、`pageSize`
- resp item 最少包含：`id`、`name`、`version`、`description`、`status`、`updatedAt`
- 空列表与筛选空结果要能区分前端状态

#### 7.3.2 `POST /packages` / `PUT /packages/:id`

- body：`name`、`version`、`description`、`installCommand`、`uninstallCommand`、`executionMode`、`templateCode`、`templateConfig`、`sourceObjectKey`、`sourceFileName`、`sourceUrl`、`status`
- 关键错误：
  - `business.deploy.package.nameRequired`
  - `business.deploy.package.versionRequired`
  - `business.deploy.package.nameVersionConflict`
  - `business.deploy.package.commandTooLong`

#### 7.3.3 `GET /tasks`

- query：`keyword`、`status[]`、`packageId`、`targetType`、`executorType`、`startedFrom`、`startedTo`、`page`、`pageSize`
- resp item 最少包含：`id`、`name`、`templateName`、`templateVersion`、`packageName`、`packageVersion`、`targetType`、`executorType`、`status`、`successCount`、`failedCount`、`totalCount`、`startedAt`、`finishedAt`

#### 7.3.4 `POST /tasks`

- body：
  - `name`
  - `templateId` 或 `packageId`
  - `businessScopeId`
  - `targetType` (`host | group`)
  - `targetIds[]`
  - `executorType` (`manual | simulated | ssh`)
  - `templateParams`
  - `credentialRef`（SSH 时必填，只允许引用）
  - `remark`
- 服务端必须通过 `ResolveTaskTargets` capability 固化主机快照后再落任务
- 若传 `templateId`，服务端必须回填模板快照，并允许从模板默认组件/默认动作派生任务头信息
- host 目标必须和动作匹配：`install / reinstall` 允许 `assigned / online`，`uninstall / upgrade` 仅允许 `online`
- 关键错误：
  - `business.deploy.task.nameRequired`
  - `business.deploy.task.packageRequired`
  - `business.deploy.task.packageDisabled`
  - `business.deploy.task.scopeRequired`
  - `business.deploy.task.scopeInvalid`
  - `business.deploy.task.targetRequired`
  - `business.deploy.task.invalidTargetType`
  - `business.deploy.task.invalidExecutorType`
  - `business.deploy.task.invalidAction`
  - `business.deploy.task.targetOutOfScope`
  - `business.deploy.task.targetStatusMismatch`

#### 7.3.5 `GET /tasks/:id`

- 返回任务头信息 + 主机执行明细 + 汇总统计
- 若资源不存在，返回 `business.deploy.task.notFound`
- 若当前用户无详情权限或数据范围不覆盖，返回 `business.deploy.task.forbidden`

#### 7.3.6 `POST /tasks/:id/start`

- 只允许 `draft / pending` 状态启动
- 启动时生成 `biz_deploy_task_host` 快照记录
- `ssh` 执行方式只接受 `credentialRef` 和 `hostFingerprint`，禁止在 Start DTO 传 password/private key
- 每次 Start 必须携带幂等键；同键同 payload 返回既有结果，同键不同 payload 返回 409
- `ssh` 启动前必须先完成固定模板参数、脚本模板变量和命令缺失检查；前置配置不合法时不得把任务推进到 `running`
- 关键错误：
  - `business.deploy.task.templateNotFound`
  - `business.deploy.task.templateDisabled`
  - `business.deploy.task.packageNotFound`
  - `business.deploy.task.invalidStartState`
  - `business.deploy.task.emptyResolvedTargets`
  - `business.deploy.task.templateParamsInvalid`
  - `business.deploy.task.templateInvalid`
  - `business.deploy.task.installCommandRequired`
  - `business.deploy.task.uninstallCommandRequired`
  - `business.deploy.task.packageSourceMissing`
  - `business.deploy.task.sshHostKeyRequired`
  - `business.deploy.task.credentialRefRequired`
  - `business.deploy.task.credentialRefInvalid`
  - `business.deploy.task.sshHostKeyMismatch`
  - `business.deploy.task.sshAuthFailed`
  - `business.deploy.task.sshConnectFailed`

#### 7.3.7 `POST /tasks/:id/cancel`

- 允许 `pending / running / pause_requested / paused` 发起取消；运行中先进入 `cancel_requested`
- 关键错误：`business.deploy.task.invalidCancelState`

#### 7.3.8 `POST /tasks/:id/pause|resume|retry`

- `pause` 只允许 running；`resume` 只允许 paused；`retry` 只允许 failed。
- `retry.mode` 只允许 `failed_step | failed_host`，并创建新 attempt。
- 三个接口均要求独立权限、DataScope、审计 metadata 和幂等键。

#### 7.3.9 `GET /tasks/:id/events`

- query：`after` 为最后确认的 sequence。
- 返回 `text/event-stream`，包含 heartbeat、task/host/step、timestamp、redacted content 和 elapsed time。
- 断线后从 `after` 重连；无详情权限或数据范围返回 403。

#### 7.3.10 `DELETE /tasks/:id`

- 只允许删除 `draft / pending` 状态任务
- 删除任务时同步清理对应 `biz_deploy_task_host` 明细
- 关键错误：`business.deploy.task.invalidDeleteState`

#### 7.3.11 `POST /task-hosts/:id/result`

- body：`status` (`success | failed | skipped`)、`attemptNo`、`stateVersion`、`errorKey`、`errorMessage`
- stdout/stderr 通过 LogSink 写追加式脱敏 chunk，不随结果 DTO 直接提交
- `failed` 时 `errorMessage` 必填
- 关键错误：
  - `business.deploy.taskHost.notFound`
  - `business.deploy.taskHost.invalidResultState`
  - `business.deploy.taskHost.markFailed.reasonRequired`

## 8. 前端与 UI

前端目录：`frontend/src/modules/business/deploy`。

### 8.1 复用平台现有组件

| 组件                | 用途                              |
| :------------------ | :-------------------------------- |
| `PageContainer`     | 整页骨架                          |
| `PageHeader`        | hero 区，含标题、面包屑、主操作   |
| `FilterPanel`       | 列表页筛选区                      |
| `AppTable`          | 标准表格                          |
| `ListHeaderActions` | 列表操作条（批量、新建）          |
| `FormSection`       | 表单分区                          |
| `SubmitBar`         | 表单底部提交栏                    |
| `AppModal`          | 弹窗                              |
| `PageState`         | 空 / 加载 / 错误 / 无权限统一容器 |

### 8.2 通用视觉与状态约束

- **不新增 UI 库**。所有组件取自 Arco Design + base 平台组件。
- **不新增独立视觉体系**。配色走 base 的 `THEME_TOKENS_REFERENCE.md`；暗色模式自动适配。
- 表格、筛选、按钮、详情页卡片沿用 CMDB 和系统管理节奏（参考 `BUSINESS_CMDB_MODULE_DESIGN.md` §9）。
- 状态统一用 `Tag`，颜色映射 `THEME_TOKENS_REFERENCE.md` §4 状态色 token：
  - `success` → 任务/明细 success
  - `warning` → running
  - `error` → failed
  - `info` → pending / draft
  - 中性灰 → canceled / skipped
- 操作类按钮使用图标 + 文案，危险动作（删除、取消任务、标记失败）使用 base 的二次确认 Modal。
- 状态变体（加载/空/错/无权限）遵循 base 的 `EMPTY_LOADING_ERROR_STATES.md`。

### 8.3 通知与提示规则

- 成功使用 `Message.success`，i18n key 走 `business.deploy.*.success`
- 失败走 request interceptor 与 i18n 错误 key，**不**硬编码英文 fallback
- 启动、暂停、恢复、取消和重试使用状态感知确认
- V1 使用认证 SSE；仅在 SSE 不可用或断线时按文档化策略降级轮询

---

### 8.4 软件组件列表 `DeployPackageList`

- **骨架类型**：标准列表页（hero + 筛选 + 表格 + 分页）
- **筛选区**：关键词（搜索 name / version / description）、状态（enabled / disabled）
- **表格列**：组件名、版本、执行模式、状态 (Tag)、最近部署时间、说明、操作
- **操作按钮**：编辑、删除、停用/启用切换
- **新建入口**：列表头部右上「新建组件」按钮，权限 `business:deploy:package:create`
- **页面状态**：empty-initial（引导新建第一个组件）、empty-filtered、loading、error、forbidden

### 8.5 软件组件表单 `DeployPackageForm`（Modal 内）

- **骨架类型**：标准 Modal 表单
- **字段**：组件名（必填，唯一）、版本（必填）、状态、执行模式、模板类型、源码包、安装命令、卸载命令、说明
- **固定模板**：当前已内置 `nginx_systemd`、`mysql_systemd`、`redis_systemd`、`minio_systemd`、`harbor_offline`
- `nginx_systemd`：源码安装、安装目录参数、systemd 服务注册
- `mysql_systemd`：离线二进制安装、数据目录初始化、root 密码和端口参数
- `redis_systemd`：源码安装、数据目录和端口参数、systemd 服务注册
- `minio_systemd`：单机 MinIO 安装、API/控制台端口、root 用户密码参数
- `harbor_offline`：Harbor 离线安装包部署、hostname/http 端口/admin 密码参数
- **源码包**：支持通过系统上传能力上传 `tar.gz`、`tgz`、`zip`、`tar`、`gz`，用于内网离线部署
- **表单校验**：组件名 + 版本组合唯一（前端预校验 + 后端 4xx 兜底）；命令字段限制长度 ≤ 8192；固定模板需校验 `templateCode`
- **提交反馈**：成功关闭 Modal + 列表刷新；失败保留表单数据并显示 banner

### 8.6 部署任务列表 `DeployTaskList`

- **骨架类型**：标准列表页
- **筛选区**：关键词（任务名）、状态（pending/running/success/failed/canceled）
- **表格列**：任务名、关联组件 (name@version)、目标类型、业务域、执行方式、状态 (Tag)、创建时间、执行时间、操作
- **操作按钮**：详情、启动（draft/pending 状态）、取消（pending/running 状态）、编辑（draft 状态）、删除
- **批量动作**：第一版不做任务级批量启动；列表头操作仅“新建任务”
- **实时刷新**：详情使用 SSE；列表可消费摘要事件，断线时才按退避策略轮询并显示 stale 状态

### 8.7 部署任务表单 `DeployTaskForm`（Modal 内）

- **骨架类型**：标准 Modal 表单
- **基础信息**：任务名、关联组件、业务域、服务/服务实例、目标类型、目标主机/分组、执行方式、备注
- **凭据**：SSH 只选择 `CredentialRef`，表单不包含 password/private key 字段
- **模板参数**：当组件为固定模板时，表单联动显示参数区；参数区按 `templateCode` 动态渲染，而不是只写死 `nginx`
- **目标选择交互**：
- host 模式：先选业务域，再从该业务域下状态为 `assigned/online` 的主机中多选
- 真实闭环脚本会先判断主机是否已安装目标组件：已安装则执行 `uninstall -> reinstall`，未安装则跳过卸载直接执行 `install`
  - group 模式：分组多选
- **校验**：组件状态必须为 `enabled`；`action` 仅允许 `install / uninstall / upgrade / reinstall`；`targetType` 仅允许 `host / group`；host 模式必须先选业务域；目标至少 1 个；主机目标状态必须与动作匹配（`install / reinstall` 允许 `assigned / online`，`uninstall / upgrade` 仅允许 `online`）；固定模板任务默认切到 `ssh`

### 8.8 部署任务详情 `DeployTaskDetail`

- **骨架类型**：详情页（hero + 多 section）
- **hero 区**：任务名、状态 Badge、关联组件、目标类型与数量、执行人、时间
- **section 1 任务摘要**：执行方式、目标快照、组件/模板版本、credential metadata、备注、`external_task_id`（如有）；不展示秘密值或未脱敏命令
- **section 2 状态流转可视化**：以 §5 状态机为模板，渲染当前所处状态 + 已经走过的路径
- **section 3 主机执行明细表**：
  - 列：主机名、IP、OS、状态 Tag、开始时间、结束时间、耗时、错误摘要、操作
  - 状态分布统计条放在表格上方（success X 个，failed Y 个，pending Z 个）
  - 单行操作：「标记成功」「标记失败」按钮根据当前状态条件渲染
- - 每台主机下方按 attempt/step 展示 checkpoint、事件和耗时
- - 日志查看器按 host/step/stream 读取脱敏 chunk，显示 sequence、连接和 stale 状态
- **section 4 任务级操作区**：启动、暂停、恢复、取消、重试、编辑、删除按钮根据状态和权限渲染
- **section 5 事件/审计时间线**：展示追加式状态事件和审计关联，不得省略 V1 高风险动作
- **页面状态**：loading（hero + section skeleton）、not-found（资源不存在）、error-server、forbidden

### 8.9 标记结果交互

- 入口：详情页主机明细表的「标记成功」「标记失败」按钮
- 弹窗：标记成功无需输入；标记失败需填写错误描述（必填，i18n key `business.deploy.taskHost.markFailed.reasonRequired`）
- 提交后该行状态切换，任务状态可能自动汇总；列表无需手动刷新（局部更新）

### 8.10 i18n key 前缀

- `business.deploy.package.*` 软件组件
- `business.deploy.task.*` 任务
- `business.deploy.taskHost.*` 主机明细
- `business.deploy.state.*` 状态名称（success / pending / running / failed / canceled / skipped）

最低必补 key：

- 菜单：`business.deploy.package.menu.list`、`business.deploy.task.menu.list`
- 页头：`business.deploy.package.page.title`、`business.deploy.task.page.title`、`business.deploy.task.detail.title`
- 状态：`business.deploy.state.draft`、`business.deploy.state.pending`、`business.deploy.state.running`、`business.deploy.state.success`、`business.deploy.state.failed`、`business.deploy.state.canceled`、`business.deploy.state.skipped`
- 错误：见 §7.3 各接口关键错误
- 成功反馈：`business.deploy.package.create.success`、`business.deploy.package.update.success`、`business.deploy.task.create.success`、`business.deploy.task.start.success`、`business.deploy.task.cancel.success`

### 8.11 响应式

- 表格列按 `MOBILE_RESPONSIVE_BREAKPOINTS.md` §4 的列优先级声明
- 任务详情在 `md` 以下：section 单列堆叠，主机明细表切换为卡片视图（每行一卡）

### 8.12 引用规范（位于 base）

- `../../../pantheon-base/docs/designs/FRONTEND_UI_SPEC.md`
- `../../../pantheon-base/docs/designs/FRONTEND_PAGE_TEMPLATES.md`
- `../../../pantheon-base/docs/designs/BACKOFFICE_STYLE_CONSTRAINTS.md`
- `../../../pantheon-base/docs/designs/ACCESSIBILITY.md`
- `../../../pantheon-base/docs/designs/THEME_TOKENS_REFERENCE.md`
- `../../../pantheon-base/docs/designs/DARK_MODE_DESIGN.md`
- `../../../pantheon-base/docs/designs/EMPTY_LOADING_ERROR_STATES.md`
- `../../../pantheon-base/docs/designs/MOBILE_RESPONSIVE_BREAKPOINTS.md`

## 9. 权限与审计

权限按导航、页面、操作、接口四层理解。

页面 / 导航权限：

- `business:deploy:package:view`
- `business:deploy:task:view`
- `business:deploy:task:detail`

权限点：

- `business:deploy:package:view`
- `business:deploy:package:list`
- `business:deploy:package:create`
- `business:deploy:package:update`
- `business:deploy:package:delete`
- `business:deploy:template:list`
- `business:deploy:template:create`
- `business:deploy:template:update`
- `business:deploy:template:delete`
- `business:deploy:task:view`
- `business:deploy:task:list`
- `business:deploy:task:detail`
- `business:deploy:task:create`
- `business:deploy:task:update`
- `business:deploy:task:delete`
- `business:deploy:task:start`
- `business:deploy:task:pause`
- `business:deploy:task:resume`
- `business:deploy:task:cancel`
- `business:deploy:task:retry`
- `business:deploy:task:mark-result`

模板权限现状：

- `business:deploy:template:list/create/update/delete` 四项均已在 `backend/pkg/contracts/permission_policies.go` 定义 API 路由映射
- `deploy_seed.go` 当前只 seed 模板列表 C 菜单（`PagePerm=business:deploy:template:list`）
- `create/update/delete` 三项 F 按钮 seed 缺失，不能写成已完成的角色授权事实；缺口见任务卡 `2026-07-17-business-permission-gaps`

审计动作：

- 新建/编辑/删除软件组件
- 新建/编辑/删除部署任务
- 启动、暂停、恢复、取消、重试部署任务
- 标记主机执行结果

高敏约束：

- 启动、暂停、恢复、取消、重试和结果回写必须记录操作者、请求 ID、幂等键、任务 ID、目标数量、状态变更前后值
- 若任务目标通过分组解析，审计中至少保留解析时的主机数量与快照时间
- Agent 结果回写后续接入时，必须单独区分“人工标记”与“执行器上报”来源

## 10. 验收标准

- 软件组件、模板和任务基础 CRUD 在权限/DataScope 内闭环。
- Start 只接受 CredentialRef，重复 Start 幂等，明文 credential/组件密码不进入数据库、API、日志、审计、导出或 evidence。
- 模板对引号、反引号、`$()`、分号、换行和重定向输入的对抗测试通过。
- task/host/attempt/step/event/log chunk 持久化完整，所有状态转换使用 CAS/version guard。
- pause/resume/cancel/failed-step retry/failed-host retry/Worker restart recovery 结果确定。
- provider registry、DAG、condition、failure policy 和有界并发覆盖正常、部分失败和取消场景。
- SSE 鉴权、DataScope、heartbeat、after-sequence 重连、脱敏和 polling fallback 通过。
- CMDB/ServiceInstance 回写与 task 终态一致，不产生虚假 running/success。
- Deploy 不直读 `biz_cmdb_*` 或其他 owner 表，只通过 capability/facade。
- 列表/详情/按钮/接口权限分离；普通用户对每个 mutation 的越权请求返回 403。
- 菜单、权限、i18n、migration、构建、Windows CGO race 和外部 VM 运行矩阵通过。
- 响应式和可访问性按 Base 合同验收，任务详情覆盖 loading、empty、error、forbidden、conflict、stale 和 partial failure。

发布状态只由 [V1 冻结检查表](../V1_RELEASE_CHECKLIST.md) 和对应 evidence/review 更新。
