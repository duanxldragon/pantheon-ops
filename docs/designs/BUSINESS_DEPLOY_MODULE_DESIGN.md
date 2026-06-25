# 安装部署业务模块设计

English version: [BUSINESS_DEPLOY_MODULE_DESIGN.en.md](./BUSINESS_DEPLOY_MODULE_DESIGN.en.md)

更新时间：2026-05-28

类型：Design
归属层：business/deploy
状态：Active

本文定义运维平台第二个业务模块：安装部署（Deploy）。它与 `business/cmdb`、`business/bizscope` 平级，挂在“工作台 / 运维平台”菜单下，用 CMDB 主机和业务域作为目标来源，为后续 Agent 执行、软件安装、任务编排、凭据托管和变更审计打基础。

本文按 Pantheon Base 当前 canonical 收口：

- 跨业务模块只允许通过显式 capability / facade 查询，不允许直读对方表
- 业务权限统一使用 `business:<module>:<resource>:<action>`
- 视觉 token 数值以 `pantheon-base/docs/designs/THEME_TOKENS_REFERENCE.md` 为准
- 响应式断点与退化行为以 `pantheon-base/docs/designs/MOBILE_RESPONSIVE_BREAKPOINTS.md` 为准

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
- 提供手动/模拟/SSH 执行闭环，先验证任务、权限、审计和页面流程。
- 已验证真实主机闭环可跑通主机接入、业务域绑定、SSH 卸载、SSH 安装/重装和 CMDB 回写状态切换。
- 为后续 Agent 上报和真实执行预留接口。

本模块不负责：

- 主机台账和分组维护，属于 `business/cmdb`。
- 实时连通性监控，属于后续监控能力。
- SSH 凭据托管，属于后续凭据能力。
- 真正 Agent 进程和执行器调度，第一版只预留协议字段。
- 回退的独立执行入口，当前仍作为后续生命周期能力扩展，不在第一版主链路内。
- 平台用户、角色、菜单、权限实现，属于 `system/iam`。

## 2. 边界与依赖

| 类别 | 允许依赖 | 禁止依赖 |
| :--- | :--- | :--- |
| platform | `pkg/common`、统一响应、审计元数据、模块注册契约 | 修改平台壳层业务逻辑 |
| system/auth | `gin.Context` 中登录主体 | 直接 import auth Service |
| system/iam | 菜单、权限点、Casbin 策略结果 | 业务模块内写角色授权逻辑 |
| system/org | 数据范围上下文 | 直接依赖 org repository |
| business/cmdb | 通过显式只读查询 capability 获取可见主机、分组和目标快照 | 直读 `biz_cmdb_*` 表、直接依赖 CMDB repository/service、修改 CMDB 内部状态机或标签逻辑 |
| business/bizscope | 通过业务域 ID 和名称约束部署信任边界 | 绕过业务域直接为未分配主机下发主机级部署 |

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

| capability | 输入 | 输出 | 用途 |
| :--- | :--- | :--- | :--- |
| `ListSelectableHosts` | 关键字、状态、数据范围、分页 | Host 摘要列表 | 任务目标选择 |
| `ListSelectableGroups` | 关键字、数据范围 | Group 摘要列表 | 分组目标选择 |
| `PreviewGroupMembers` | `groupIds[]`、数据范围 | 分组成员数量 / 主机摘要 | 表单预览 |
| `ResolveTaskTargets` | `targetType`、`targetIds[]`、数据范围 | 去重后的 Host 快照列表 | 启动任务时固化目标 |

契约要求：

- 所有结果都必须经过 `DataScopeReq` 或等价上下文过滤
- Deploy 只拿“当前任务执行需要的快照字段”，不反向绑定 CMDB 的内部 JSON 结构
- 该 capability 后续可以落到 `pkg/contracts`、模块 facade 或独立 query service，但不能只停留在口头约定

## 3. 菜单与命名

一级归属：`运维平台`。

二级模块：`安装部署`。

菜单中文保持四字：

| 菜单 key | 路径 | titleKey | routeName | module | component key | pagePermission | activeMenu | 类型 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `business.deploy.package` | `/operations/deploy/package` | `operations.deploy.package.menu` | `deploy-package-list` | `business.deploy` | `business/deploy/package/DeployPackageList` | `business:deploy:package:view` | — | `C` | 软件组件列表 |
| `business.deploy.template` | `/operations/deploy/template` | `operations.deploy.template.menu` | `deploy-template-list` | `business.deploy` | `business/deploy/template/DeployTemplateList` | `business:deploy:template:list` | — | `C` | 任务模板列表 |
| `business.deploy.task` | `/operations/deploy/task` | `operations.deploy.task.menu` | `deploy-task-list` | `business.deploy` | `business/deploy/task/DeployTaskList` | `business:deploy:task:view` | — | `C` | 部署任务列表 |
| `business.deploy.task.detail` | `/operations/deploy/task/:id` | `operations.deploy.task.detail` | `deploy-task-detail` | `business.deploy` | `business/deploy/task/DeployTaskDetail` | `business:deploy:task:detail` | `/operations/deploy/task` | `C` | 任务详情页，不进侧边菜单 |

约束：

- `module` 固定为 `business.deploy`
- 菜单只表达导航，不表达执行规则
- 列表页和详情页的页面权限分离：列表页走 `view`，详情页走 `detail`
- 任务模板页使用独立模板模型：模板头 + 模板步骤，任务创建优先选模板，软件组件保留为底层可复用资产

## 4. 核心对象

| 对象 | 说明 | 关键字段 |
| :--- | :--- | :--- |
| DeployPackage | 软件组件 | name、version、execution_mode、template_code、source_object_key、install_command、uninstall_command、status |
| DeployTemplate | 任务模板 | name、version、category、default_action、package_id、parameter_schema、status |
| DeployTemplateStep | 模板步骤 | template_id、step_code、step_name、step_type、action、package_id、template_params、sort |
| DeployTask | 部署任务 | name、package_id、business_scope_id、action、target_type、executor_type、execution_mode、status、started_at、finished_at |
| DeployTaskHost | 主机执行明细 | task_id、host_id、host_ip、status、trace_steps、stdout、stderr、error_message |

## 5. 状态流转

任务状态：

```text
draft -> pending -> running -> success
                         \-> failed
pending/running -> canceled
```

主机明细状态：

```text
pending -> running -> success
                 \-> failed
pending -> skipped
```

当前执行方式：

- `manual`：用户创建任务后点击启动，系统生成主机明细并置为运行中。
- `simulated`：和 manual 一样，但文案明确为模拟执行。
- `ssh`：用户在启动时输入 SSH 用户、密码或私钥、主机指纹，系统远程执行安装脚本。
- `action`：第一版已支持 `install`、`uninstall`、`upgrade`、`reinstall` 四类任务动作，其中 `upgrade` / `reinstall` 先复用固定模板安装链路。
- `uninstall` 会移除对应组件记录；如果主机卸载后仍存在其他已安装组件，则主机保持 `online`，只有最后一个组件卸载后才回写为 `assigned`。
- 任务模板按步骤顺序执行，当前支持 `step_type=package` 和 `step_type=script` 两类步骤。
- `script` 步骤通过 `step_config.script` 执行脚本正文，可选 `precheckCommand` / `postcheckCommand` 做前置和后置校验。
- 每一步都会写入主机执行轨迹，并把标准输出/错误输出按 `precheck / script / postcheck` 分段汇总。
- 固定模板脚本在 `reinstall / upgrade` 前会先停止对应 systemd 服务并等待旧进程退出，避免覆盖二进制时出现 `Text file busy`。
- 用户通过“标记成功 / 标记失败”完成主机级结果。
- 任务状态由主机明细汇总：全部成功为 `success`，存在失败为 `failed`，全部取消为 `canceled`。
- 主机级明细必须记录执行过程步骤，至少包含连接、脚本渲染、安装、服务状态和结果回写。

后续执行方式：

- `agent`：Agent 拉取任务或接收下发后回写结果。

## 6. 数据模型

### 6.1 biz_deploy_package

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | bigint | 主键 |
| name | varchar(128) | 组件名称 |
| version | varchar(64) | 版本 |
| description | varchar(512) | 说明 |
| install_command | text | 安装命令模板 |
| uninstall_command | text | 卸载命令模板 |
| execution_mode | varchar(32) | `fixed` / `orchestrated` |
| template_code | varchar(64) | 固定模板编码，如 `nginx_systemd`、`mysql_systemd`、`redis_systemd`、`minio_systemd`、`harbor_offline` |
| template_config | json | 模板元信息 |
| source_object_key | varchar(255) | 上传源码包对象键 |
| source_file_name | varchar(255) | 上传源码包原始文件名 |
| source_url | varchar(512) | 源码包访问地址 |
| status | varchar(32) | enabled / disabled |
| created_at / updated_at / deleted_at | datetime | 审计字段 |
| created_by / updated_by | varchar(64) | 操作人 |

唯一约束：`uk_deploy_package_name_version_deleted` (`name`, `version`, `deleted_at`)。

### 6.2 biz_deploy_task

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | bigint | 主键 |
| name | varchar(128) | 任务名称 |
| template_id | bigint | 任务模板 ID，可为空 |
| template_name / template_version | varchar | 创建任务时的模板快照 |
| package_id | bigint | 软件组件 ID |
| package_name / package_version | varchar | 创建任务时的组件快照 |
| business_scope_id / business_scope_name | bigint / varchar | 业务域快照 |
| action | varchar(32) | install / uninstall / upgrade / reinstall |
| target_type | varchar(32) | host / group |
| target_ids | json | 目标主机或分组 ID |
| executor_type | varchar(32) | manual / simulated / agent / ssh |
| execution_mode | varchar(32) | fixed / orchestrated |
| template_params | json | 固定模板参数，如安装目录、服务名 |
| status | varchar(32) | draft / pending / running / success / failed / canceled |
| remark | varchar(512) | 备注 |
| external_task_id | varchar(128) | Agent/外部执行任务 ID |
| started_at / finished_at | datetime | 执行时间 |
| created_at / updated_at / deleted_at | datetime | 审计字段 |
| created_by / updated_by | varchar(64) | 操作人 |

### 6.3 biz_deploy_task_host

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | bigint | 主键 |
| task_id | bigint | 部署任务 ID |
| host_id | bigint | CMDB 主机 ID |
| hostname / host_ip / os | varchar | 主机快照 |
| status | varchar(32) | pending / running / success / failed / skipped |
| trace_steps | json | 执行过程步骤 |
| stdout / stderr | text | 执行输出 |
| error_message | varchar(512) | 错误说明 |
| executor_id | varchar(128) | Agent 或执行器 ID |
| started_at / finished_at | datetime | 执行时间 |
| reported_at | datetime | Agent 回写时间 |
| created_at / updated_at | datetime | 审计字段 |

### 6.4 biz_deploy_template

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | bigint | 主键 |
| name | varchar(128) | 模板名称 |
| version | varchar(64) | 模板版本 |
| description | varchar(512) | 说明 |
| category | varchar(64) | 模板分类 |
| execution_mode | varchar(32) | `fixed` / `orchestrated` |
| default_action | varchar(32) | install / uninstall / upgrade / reinstall |
| package_id | bigint | 默认软件组件 |
| package_name / package_version | varchar | 默认组件快照 |
| template_code | varchar(64) | 固定模板编码 |
| template_config | json | 模板元信息 |
| parameter_schema | json | 默认参数 |
| status | varchar(32) | enabled / disabled |
| created_at / updated_at / deleted_at | datetime | 审计字段 |
| created_by / updated_by | varchar(64) | 操作人 |

### 6.5 biz_deploy_template_step

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | bigint | 主键 |
| template_id | bigint | 模板 ID |
| step_code | varchar(64) | 步骤编码 |
| step_name | varchar(128) | 步骤名称 |
| step_type | varchar(32) | `package` / `script` |
| action | varchar(32) | 步骤动作 |
| package_id | bigint | 步骤组件 |
| package_name / package_version | varchar | 步骤组件快照 |
| template_code | varchar(64) | 固定模板编码 |
| template_params | json | 步骤参数 |
| step_config | json | 脚本正文、前置校验、后置校验等扩展配置 |
| sort | int | 步骤顺序 |
| created_at / updated_at | datetime | 审计字段 |

### 6.6 租户就绪判断

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

| 接口 | 方法 | 说明 | 权限 |
| :--- | :--- | :--- | :--- |
| `/packages` | GET | 软件组件列表 | `business:deploy:package:list` |
| `/packages` | POST | 新建软件组件 | `business:deploy:package:create` |
| `/packages/:id` | PUT | 编辑软件组件 | `business:deploy:package:update` |
| `/packages/:id` | DELETE | 删除软件组件 | `business:deploy:package:delete` |
| `/templates` | GET | 任务模板列表 | `business:deploy:template:list` |
| `/templates` | POST | 新建任务模板 | `business:deploy:template:create` |
| `/templates/:id` | GET | 任务模板详情 | `business:deploy:template:list` |
| `/templates/:id` | PUT | 编辑任务模板 | `business:deploy:template:update` |
| `/templates/:id` | DELETE | 删除任务模板 | `business:deploy:template:delete` |
| `/tasks` | GET | 部署任务列表 | `business:deploy:task:list` |
| `/tasks` | POST | 创建部署任务 | `business:deploy:task:create` |
| `/tasks/:id` | GET | 任务详情 | `business:deploy:task:detail` |
| `/tasks/:id` | PUT | 编辑未启动任务 | `business:deploy:task:update` |
| `/tasks/:id/start` | POST | 启动任务 | `business:deploy:task:start` |
| `/tasks/:id/cancel` | POST | 取消任务 | `business:deploy:task:cancel` |
| `/task-hosts/:id/result` | POST | 标记主机执行结果 | `business:deploy:task:mark-result` |
| `/task-hosts/:id/report` | POST | Agent 结果回写预留 | 第一版不挂菜单 |

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
- `ssh` 执行方式要求补充 `sshUser`、认证信息和 `hostFingerprint`
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
  - `business.deploy.task.sshUserRequired`
  - `business.deploy.task.sshPasswordRequired`
  - `business.deploy.task.sshPrivateKeyRequired`
  - `business.deploy.task.sshHostKeyMismatch`
  - `business.deploy.task.sshAuthFailed`
  - `business.deploy.task.sshConnectFailed`

#### 7.3.7 `POST /tasks/:id/cancel`

- 只允许 `pending / running` 状态取消
- 关键错误：`business.deploy.task.invalidCancelState`

#### 7.3.8 `POST /task-hosts/:id/result`

- body：`status` (`success | failed | skipped`)、`stdout`、`stderr`、`errorMessage`
- `failed` 时 `errorMessage` 必填
- 关键错误：
  - `business.deploy.taskHost.notFound`
  - `business.deploy.taskHost.invalidResultState`
  - `business.deploy.taskHost.markFailed.reasonRequired`

## 8. 前端与 UI

前端目录：`frontend/src/modules/business/deploy`。

### 8.1 复用平台现有组件

| 组件 | 用途 |
| :--- | :--- |
| `PageContainer` | 整页骨架 |
| `PageHeader` | hero 区，含标题、面包屑、主操作 |
| `FilterPanel` | 列表页筛选区 |
| `AppTable` | 标准表格 |
| `ListHeaderActions` | 列表操作条（批量、新建） |
| `FormSection` | 表单分区 |
| `SubmitBar` | 表单底部提交栏 |
| `AppModal` | 弹窗 |
| `PageState` | 空 / 加载 / 错误 / 无权限统一容器 |

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
- 启动、取消、标记结果使用二次确认
- 第一版不做站内通知中心和 WebSocket，只做页面内状态轮询或手动刷新

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
- **轮询刷新**：列表中存在 `pending/running` 状态任务时，每 5 秒自动 `GET /tasks` 刷新

### 8.7 部署任务表单 `DeployTaskForm`（Modal 内）

- **骨架类型**：标准 Modal 表单
- **基础信息**：任务名、关联组件、业务域、目标类型、目标主机/分组、执行方式、备注
- **模板参数**：当组件为固定模板时，表单联动显示参数区；参数区按 `templateCode` 动态渲染，而不是只写死 `nginx`
- **目标选择交互**：
- host 模式：先选业务域，再从该业务域下状态为 `assigned/online` 的主机中多选
- 真实闭环脚本会先判断主机是否已安装目标组件：已安装则执行 `uninstall -> reinstall`，未安装则跳过卸载直接执行 `install`
  - group 模式：分组多选
- **校验**：组件状态必须为 `enabled`；`action` 仅允许 `install / uninstall / upgrade / reinstall`；`targetType` 仅允许 `host / group`；host 模式必须先选业务域；目标至少 1 个；主机目标状态必须与动作匹配（`install / reinstall` 允许 `assigned / online`，`uninstall / upgrade` 仅允许 `online`）；固定模板任务默认切到 `ssh`

### 8.8 部署任务详情 `DeployTaskDetail`

- **骨架类型**：详情页（hero + 多 section）
- **hero 区**：任务名、状态 Badge、关联组件、目标类型与数量、执行人、时间
- **section 1 任务摘要**：执行方式、目标类型、命令快照（折叠展开）、备注、`external_task_id`（如有）
- **section 2 状态流转可视化**：以 §5 状态机为模板，渲染当前所处状态 + 已经走过的路径
- **section 3 主机执行明细表**：
  - 列：主机名、IP、OS、状态 Tag、开始时间、结束时间、耗时、错误摘要、操作
  - 状态分布统计条放在表格上方（success X 个，failed Y 个，pending Z 个）
  - 单行操作：「标记成功」「标记失败」按钮根据当前状态条件渲染
  - 每台主机下方展示 `traceSteps` 过程表，记录阶段、时间和消息
  - 完整 stdout/stderr 在详情页直接展示，保留原始输出便于排障
- **section 4 任务级操作区**：启动、取消、编辑、删除按钮根据状态条件渲染
- **section 5 审计时间线**（可选，第一版可省）：列出 §10 §11 章节定义的审计记录
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
- `business:deploy:task:view`
- `business:deploy:task:list`
- `business:deploy:task:detail`
- `business:deploy:task:create`
- `business:deploy:task:update`
- `business:deploy:task:start`
- `business:deploy:task:cancel`
- `business:deploy:task:mark-result`

审计动作：

- 新建/编辑/删除软件组件
- 新建/编辑部署任务
- 启动部署任务
- 取消部署任务
- 标记主机执行结果

高敏约束：

- 启动、取消、结果回写必须记录操作者、任务 ID、目标数量、状态变更前后值
- 若任务目标通过分组解析，审计中至少保留解析时的主机数量与快照时间
- Agent 结果回写后续接入时，必须单独区分“人工标记”与“执行器上报”来源

## 10. 验收标准

- 软件组件可增删改查。
- 部署任务可选择主机或分组创建。
- 启动任务后生成主机执行明细。
- 主机明细可标记成功或失败，并汇总任务状态。
- 菜单、权限、i18n、构建检查通过。
- CMDB 和系统域现有烟测不被破坏。
- 业务模块不直接依赖 `modules/system/*` 内部实现。
- Deploy 不直接读取 `biz_cmdb_*` 表，跨业务查询通过显式 capability / facade 完成。
- 列表页 / 详情页 / 按钮 / 接口四层权限链路完整：页面 `view` 不等于列表 `list`、详情 `detail`、动作 `start/cancel/mark-result`
- 所有错误反馈走 i18n key，不出现英文硬编码 fallback
- 响应式表现符合 `MOBILE_RESPONSIVE_BREAKPOINTS.md`，至少覆盖 `xl / lg / md / sm`
