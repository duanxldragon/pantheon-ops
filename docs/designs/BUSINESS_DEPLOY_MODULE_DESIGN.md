# 安装部署业务模块设计

更新时间：2026-05-09

类型：Design
归属层：business/deploy
状态：Active

本文定义运维平台第二个业务模块：安装部署（Deploy）。它与 `business/cmdb` 平级，放在“运维平台”菜单下，用 CMDB 主机和分组作为目标来源，为后续 Agent 执行、软件安装、任务编排、凭据托管和变更审计打基础。

## 1. 模块概述

安装部署模块负责把“要装什么、装到哪里、执行到什么状态、结果如何”沉淀成可追踪任务。

本模块解决：

- 维护可安装的软件组件。
- 基于 CMDB 主机或分组创建部署任务。
- 记录任务和每台目标主机的执行状态。
- 提供手动/模拟执行闭环，先验证任务、权限、审计和页面流程。
- 为后续 Agent 上报和真实执行预留接口。

本模块不负责：

- 主机台账和分组维护，属于 `business/cmdb`。
- 实时连通性监控，属于后续监控能力。
- SSH 凭据托管，属于后续凭据能力。
- 真正 Agent 进程和执行器调度，第一版只预留协议字段。
- 平台用户、角色、菜单、权限实现，属于 `system/iam`。

## 2. 边界与依赖

| 类别 | 允许依赖 | 禁止依赖 |
| :--- | :--- | :--- |
| platform | `pkg/common`、统一响应、审计元数据、模块注册契约 | 修改平台壳层业务逻辑 |
| system/auth | `gin.Context` 中登录主体 | 直接 import auth Service |
| system/iam | 菜单、权限点、Casbin 策略结果 | 业务模块内写角色授权逻辑 |
| system/org | 数据范围上下文 | 直接依赖 org repository |
| business/cmdb | 通过数据库表读取可见主机基础信息，后续收敛为公共查询契约 | 修改 CMDB 内部状态机或标签逻辑 |

第一版允许部署模块读取 `biz_cmdb_host` 和 `biz_cmdb_group` 的只读数据，用于目标选择和任务快照；部署模块不得更新 CMDB 主机配置。

## 3. 菜单与命名

一级归属：`运维平台`。

二级模块：`安装部署`。

菜单中文保持四字：

| 菜单 | 路径 | 组件 |
| :--- | :--- | :--- |
| 软件组件 | `/operations/deploy/package` | `business/deploy/package/DeployPackageList` |
| 部署任务 | `/operations/deploy/task` | `business/deploy/task/DeployTaskList` |
| 任务详情 | `/operations/deploy/task/:id` | `business/deploy/task/DeployTaskDetail`，不进菜单 |

## 4. 核心对象

| 对象 | 说明 | 关键字段 |
| :--- | :--- | :--- |
| DeployPackage | 软件组件 | name、version、install_command、uninstall_command、status |
| DeployTask | 部署任务 | name、package_id、target_type、executor_type、status、started_at、finished_at |
| DeployTaskHost | 主机执行明细 | task_id、host_id、host_ip、status、stdout、stderr、error_message |

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

第一版执行方式：

- `manual`：用户创建任务后点击启动，系统生成主机明细并置为运行中。
- `simulated`：和 manual 一样，但文案明确为模拟执行。
- 用户通过“标记成功 / 标记失败”完成主机级结果。
- 任务状态由主机明细汇总：全部成功为 `success`，存在失败为 `failed`，全部取消为 `canceled`。

后续执行方式：

- `agent`：Agent 拉取任务或接收下发后回写结果。
- `ssh`：凭据能力成熟后补充，不在第一版实现。

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
| status | varchar(32) | enabled / disabled |
| created_at / updated_at / deleted_at | datetime | 审计字段 |
| created_by / updated_by | varchar(64) | 操作人 |

唯一约束：`uk_deploy_package_name_version_deleted` (`name`, `version`, `deleted_at`)。

### 6.2 biz_deploy_task

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | bigint | 主键 |
| name | varchar(128) | 任务名称 |
| package_id | bigint | 软件组件 ID |
| package_name / package_version | varchar | 创建任务时的组件快照 |
| target_type | varchar(32) | host / group |
| target_ids | json | 目标主机或分组 ID |
| executor_type | varchar(32) | manual / simulated / agent / ssh |
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
| stdout / stderr | text | 执行输出 |
| error_message | varchar(512) | 错误说明 |
| executor_id | varchar(128) | Agent 或执行器 ID |
| started_at / finished_at | datetime | 执行时间 |
| reported_at | datetime | Agent 回写时间 |
| created_at / updated_at | datetime | 审计字段 |

## 7. API 设计

前缀：`/api/v1/business/deploy`

| 接口 | 方法 | 说明 | 权限 |
| :--- | :--- | :--- | :--- |
| `/packages` | GET | 软件组件列表 | `business:deploy:package:list` |
| `/packages` | POST | 新建软件组件 | `business:deploy:package:create` |
| `/packages/:id` | PUT | 编辑软件组件 | `business:deploy:package:update` |
| `/packages/:id` | DELETE | 删除软件组件 | `business:deploy:package:delete` |
| `/tasks` | GET | 部署任务列表 | `business:deploy:task:list` |
| `/tasks` | POST | 创建部署任务 | `business:deploy:task:create` |
| `/tasks/:id` | GET | 任务详情 | `business:deploy:task:detail` |
| `/tasks/:id` | PUT | 编辑未启动任务 | `business:deploy:task:update` |
| `/tasks/:id/start` | POST | 启动任务 | `business:deploy:task:start` |
| `/tasks/:id/cancel` | POST | 取消任务 | `business:deploy:task:cancel` |
| `/task-hosts/:id/result` | POST | 标记主机执行结果 | `business:deploy:task:mark-result` |
| `/task-hosts/:id/report` | POST | Agent 结果回写预留 | 第一版不挂菜单 |

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
- **表格列**：组件名、版本、状态 (Tag)、说明、更新时间、操作
- **操作按钮**：编辑、删除、停用/启用切换
- **新建入口**：列表头部右上「新建组件」按钮，权限 `business:deploy:package:create`
- **页面状态**：empty-initial（引导新建第一个组件）、empty-filtered、loading、error、forbidden

### 8.5 软件组件表单 `DeployPackageForm`（Modal 内）

- **骨架类型**：标准 Modal 表单
- **字段**：组件名（必填，唯一）、版本（必填）、说明、安装命令（多行）、卸载命令（多行）、状态
- **表单校验**：组件名 + 版本组合唯一（前端预校验 + 后端 4xx 兜底）；命令字段限制长度 ≤ 8192
- **提交反馈**：成功关闭 Modal + 列表刷新；失败保留表单数据并显示 banner

### 8.6 部署任务列表 `DeployTaskList`

- **骨架类型**：标准列表页
- **筛选区**：关键词（任务名）、状态（draft/pending/running/success/failed/canceled）、组件下拉（关联 `DeployPackage`）、目标类型（host/group）、执行方式（manual/simulated/agent/ssh）、时间范围
- **表格列**：任务名、关联组件 (name@version)、目标类型、状态 (Tag)、执行进度（X/Y 成功 + 进度条）、创建时间、执行时间、操作
- **操作按钮**：详情、启动（draft/pending 状态）、取消（pending/running 状态）、编辑（draft 状态）、删除
- **批量动作**：第一版**不**做批量启动；列表头操作仅"新建任务"
- **轮询刷新**：列表中存在 `pending/running` 状态任务时，每 5 秒自动 `GET /tasks` 刷新

### 8.7 部署任务表单 `DeployTaskForm`（Modal 内）

- **骨架类型**：分步 Modal 表单
  - 步骤 1：基础信息（任务名、关联组件、执行方式、备注）
  - 步骤 2：目标选择（target_type = host → CMDB 主机多选；target_type = group → CMDB 分组多选）
  - 步骤 3：确认与提交
- **目标选择交互**：
  - host 模式：弹出主机选择 Drawer，左侧分组树（来自 CMDB），右侧主机表格，支持多选
  - group 模式：分组下拉，可多选；选择后显示分组当前成员数量预览
- **校验**：组件状态必须为 `enabled`；目标至少 1 个；执行方式必须为 `manual` 或 `simulated`（第一版）

### 8.8 部署任务详情 `DeployTaskDetail`

- **骨架类型**：详情页（hero + 多 section）
- **hero 区**：任务名、状态 Badge、关联组件、目标类型与数量、执行人、时间
- **section 1 任务摘要**：执行方式、目标类型、命令快照（折叠展开）、备注、`external_task_id`（如有）
- **section 2 状态流转可视化**：以 §5 状态机为模板，渲染当前所处状态 + 已经走过的路径
- **section 3 主机执行明细表**：
  - 列：主机名、IP、OS、状态 Tag、执行时间、退出码、stderr 摘要、操作（查看完整输出 / 标记结果）
  - 状态分布统计条放在表格上方（success X 个，failed Y 个，pending Z 个）
  - 单行操作：「标记成功」「标记失败」按钮根据当前状态条件渲染
  - 完整 stdout/stderr 通过 Drawer 展示，monospace 字体，可复制
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

权限点：

- `business:deploy:package:list`
- `business:deploy:package:create`
- `business:deploy:package:update`
- `business:deploy:package:delete`
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

## 10. 验收标准

- 软件组件可增删改查。
- 部署任务可选择主机或分组创建。
- 启动任务后生成主机执行明细。
- 主机明细可标记成功或失败，并汇总任务状态。
- 菜单、权限、i18n、构建检查通过。
- CMDB 和系统域现有烟测不被破坏。
- 业务模块不直接依赖 `modules/system/*` 内部实现。
