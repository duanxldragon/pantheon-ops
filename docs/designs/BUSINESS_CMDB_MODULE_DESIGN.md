# CMDB 业务模块设计

更新时间：2026-05-07

类型：Design
归属层：business/cmdb
状态：Active

本文定义运维平台 CMDB（轻量级配置管理数据库）模块设计，替代此前已退役的 `business/cmdb` 低代码验证样本设计。

CMDB 是运维平台的第一个子系统，负责主机资源台账、标签体系与主机分组，为后续部署管理、容器平台、监控告警、链路追踪和日志管理提供统一的基础资源底账。

本文覆盖子模块：

- `business/cmdb/host`：主机资源台账。
- `business/cmdb/group`：主机分组（标签过滤条件的持久化视图）。

---

## 1. 模块概述

CMDB 是业务域模块，属于运维平台能力域。它负责承载基础资源台账、标签体系与主机分组关系。

**本模块解决什么问题：**

- 统一管理混合环境（物理机、虚拟机、K8s 节点）的主机元数据。
- 通过标签体系支持环境（env）与业务系统（biz）的二维正交分类。
- 通过分组提供可复用的主机集合视图，作为下游部署、监控等运维子系统的输入。
- 支持手动录入与 SSH 一次性探测两种配置采集方式。

**本模块不负责：**

- 登录、MFA、会话、安全策略，属于 `system/auth`。
- 用户、角色、菜单、权限策略，属于 `system/iam`。
- 部门、岗位、组织树，属于 `system/org`。
- 字典、设置、i18n、上传，属于 `system/config`。
- 主机实时健康探测或存活检测，属于后续监控告警子系统。
- 完整的 ITIL 级 CI 依赖拓扑与变更影响分析。
- SSH 凭据的持久化存储，CMDB 仅支持即采即弃的一次性探测。

## 2. 边界与依赖

| 类别 | 允许依赖 | 禁止依赖 |
| :--- | :--- | :--- |
| platform | `pkg/common`、统一响应、审计元数据、数据权限上下文 | 直接修改平台壳层或工作台聚合逻辑 |
| system/auth | `gin.Context` 中的登录主体、会话上下文 | 直接 import `modules/auth` Service |
| system/iam | 权限点、页面权限、Casbin 接口鉴权结果 | 在业务模块内直接写角色授权逻辑 |
| system/org | 用户的组织归属、部门维度数据范围 | 直接依赖 org 内部 repository |
| system/config | 字典、配置、i18n key、加密配置能力 | 在业务模块内重复造字典、配置或上传协议 |

跨模块约束：

- 下游子系统（部署管理、监控告警等）通过 API 读取 CMDB 主机信息，不允许直接访问 CMDB 数据库。
- 主机状态（`status`）和已装组件（`installed_components`）由下游子系统通过 API 更新，CMDB 不主动探测。

## 3. 核心业务对象

| 对象 | 说明 | 关键属性 |
| :--- | :--- | :--- |
| **Host** | 主机、虚拟机或物理机资源 | hostname、IP、OS、SSH 端口、CPU/内存/磁盘、标签、已装组件、状态 |
| **Group** | 主机分组，由标签过滤条件动态计算成员 | 名称、条件表达式 (AND/OR + key-op-value 规则)、说明 |
| **Label** | 键值对标签，附属于 Host，分组过滤的数据源 | key (如 env/biz)、value (如 production/order) |
| **LabelSchema** | 标签规范，治理可用标签 key 和取值来源 | key、name、value_mode、dict_code、options、required、status |

Label 不设独立数据库表，以 JSON 字段存储在 Host 记录上。Group 的 conditions 同样以 JSON 字段存储，成员列表实时计算得出。

LabelSchema 是标签 key 的业务规范，不保存每台主机的标签实例。主机标签仍保存在 Host JSON 中，LabelSchema 负责避免 `env/ENV/environment`、`prod/production` 这类语义漂移影响分组和后续运维目标选择。

LabelSchema 可以维护 `options` 作为 CMDB 业务域内的常用值清单。主机标签录入和分组条件编辑根据已选 key 联动展示 value 候选值，减少用户回到标签规范页查询标签值的成本。`value_mode=free` 时仍允许自由输入；`value_mode=enum` 时优先使用 `options`；`value_mode=dict` 预留系统字典编码，但 CMDB 前端不直接耦合 `system/config/dict` Service。

**标签预置 Key（可扩展）：**

| Key | 含义 | 示例值 |
| :--- | :--- | :--- |
| `env` | 环境 | `production`、`test`、`dev` |
| `biz` | 业务系统 | `order-system`、`user-system` |
| `cluster` | 集群归属 | `cluster-shanghai`、`cluster-beijing` |
| `region` | 区域/机房 | `east-1`、`west-2` |
| `db_type` | 部署数据库类型 | `mysql`、`postgresql` |
| `os` | 操作系统大类 | `linux`、`windows` |

标签 key 不做硬枚举，用户可以扩展自定义 key。预置 key 仅提供约定俗成的语义。

标签规范可引用 `system/config/dict` 的字典编码，但边界必须清晰：

- `system/config/dict` 只提供通用字典治理能力；
- `business/cmdb` 定义哪些标签 key 有业务意义、是否必填、是否引用某个 `dict_code`；
- `business/cmdb` 不直接 import `system/config` 内部 Service 或 Repository。

## 4. 业务流程

### 4.1 主机录入与配置采集

主机支持三种配置获取模式：

| 模式 | 说明 | 当前状态 | 安全约束 |
| :--- | :--- | :--- | :--- |
| **手动录入** | 用户在表单中直接填写主机信息 | 实现 | 无凭据风险 |
| **SSH 探测** | 输入 SSH 凭据 → 连接 → 采集 (`lscpu`/`free`/`df`/`uname`) → 写入 CMDB → 凭据释放 | 实现 | 凭据仅存在于请求内存，不写库不落盘 |
| **Agent 上报** | 主机侧 Agent 定期推送配置信息 | 预留 | Token 认证、单向只读上报 |

**SSH 采集流程：**

```
用户点击"采集配置"
  → 弹窗输入 SSH 用户 + 密码/私钥（私钥优先）
  → POST /api/v1/business/cmdb/hosts/:id/collect
  → 后端 ssh.Dial 连接目标主机
  → 执行采集命令集 → 解析输出
  → 更新 host 记录的 cpu_cores/memory_gb/disk_gb/os/os_version
  → 凭据在 HTTP 请求结束后释放（不写库、不落盘、不缓存）
```

**Windows 主机预留：**

- `os` 字段预留 `windows` 值。
- 采集协议：WinRM (5985/5986)，Go 侧使用 `masterzen/winrm`。
- 初始版本 Windows 主机仅支持手动录入，WinRM 采集后续补。

### 4.2 主机分组计算

分组不存储物理主机列表，存储标签过滤条件。成员列表通过条件实时计算：

```
用户查看分组详情 / 调用 API
  → 读取 group.conditions（JSON 过滤表达式）
  → 执行过滤：host.label_values 匹配 conditions 规则
  → 返回匹配的主机列表
```

**Expression JSON 结构：**

```json
{
  "operator": "AND",
  "rules": [
    { "key": "env", "op": "eq", "val": "production" },
    { "key": "biz", "op": "eq", "val": "order-system" }
  ]
}
```

支持的 op：`eq`（等于）、`neq`（不等于）、`in`（包含于）、`notIn`。

分组成员计算必须使用当前请求的 `DataScopeReq`。成员数量和成员列表只能基于当前账号可见的 `biz_cmdb_host` 数据集计算，不能绕过部门数据范围直接全表扫描。

分组支持逻辑层级。父子关系既用于导航组织，也参与成员计算语义：子分组成员按“父级条件链 + 当前分组条件”依次过滤，等价于父级与当前级之间做 `AND` 约束。每一层分组内部仍按自身 `conditions.operator` 执行 `AND` / `OR`。

示例：

```json
{
  "parent": {
    "name": "西安开发环境",
    "conditions": {
      "operator": "AND",
      "rules": [{ "key": "region", "op": "eq", "val": "西安开发环境" }]
    }
  },
  "child": {
    "name": "监控服务器",
    "conditions": {
      "operator": "AND",
      "rules": [{ "key": "role", "op": "eq", "val": "monitor" }]
    }
  }
}
```

查看“监控服务器”子分组时，实际成员必须同时满足 `region = 西安开发环境` 与 `role = monitor`。这样主机标签仍保持正交治理，子分组无需重复维护父级标签规则。

分组统计提供两个成员口径：

- `memberCount`：当前分组条件链下的成员数量。
- `aggregateMemberCount`：当前分组及所有子分组成员的去重数量。

分组层级统计提供两个分组口径：

- `childCount`：直接子分组数量。
- `descendantGroupCount`：所有层级下级分组数量。

### 4.3 主机状态流转

```
pending（待上线） → online（在线） ⇄ offline（离线）
                                         ↘ maintenance（维护中）
```

- 新录入主机默认 `pending`。
- 状态由下游子系统（监控告警、部署管理）通过 API 更新。
- CMDB 自身不做存活检测。

`status` 表示运维状态，不表示实时网络连通性：

| status | 展示语义 | 运维含义 |
| :--- | :--- | :--- |
| `pending` | 待上线 | 已录入但尚未纳入正式运维目标 |
| `online` | 可运维 | 可作为常规运维任务目标 |
| `offline` | 已下线 | 不应作为常规运维任务目标 |
| `maintenance` | 维护中 | 默认排除自动化批量任务，除非显式选择 |

实时连通性应由后续监控、Agent 或连接探测能力维护，建议独立字段为 `connectivity_status`、`last_reachable_at`、`last_check_at`、`last_check_error`。不要把 `online/offline` 当作 SSH 是否可连接的判断。

## 5. 数据模型设计

### 5.1 Host 表

| 表名 | `biz_cmdb_host` |
| :--- | :--- |

| 字段 | 类型 | 说明 | 约束 |
| :--- | :--- | :--- | :--- |
| `id` | bigint | 主键 | PK, AUTO_INCREMENT |
| `hostname` | varchar(128) | 主机名 | NOT NULL |
| `ip` | varchar(45) | 管理 IP（v4/v6） | NOT NULL, INDEX |
| `ssh_port` | int | SSH 端口 | DEFAULT 22 |
| `os` | varchar(32) | 操作系统大类 | NOT NULL，`linux` / `windows` |
| `os_version` | varchar(128) | 操作系统版本 | 采集或手动填写 |
| `cpu_cores` | int | CPU 核数 | 采集或手动填写 |
| `memory_gb` | decimal(8,1) | 内存 (GB) | 采集或手动填写 |
| `disk_gb` | decimal(10,1) | 磁盘总容量 (GB) | 采集或手动填写 |
| `label_values` | json | 标签键值对 | `[{"key":"env","val":"production"}]` |
| `installed_components` | json | 已装组件 | `[{"name":"mysql","version":"8.0.35"}]` |
| `status` | varchar(32) | 主机状态 | DEFAULT `pending`，`online`/`offline`/`maintenance` |
| `dept_id` | bigint | 数据范围归属部门 | INDEX，新增时默认取当前登录主体部门 |
| `owner` | varchar(64) | 负责人 | 可选 |
| `remark` | text | 备注 | 可选 |
| `created_at` | datetime | 创建时间 | NOT NULL |
| `updated_at` | datetime | 更新时间 | NOT NULL |
| `created_by` | varchar(64) | 创建人 | |
| `updated_by` | varchar(64) | 更新人 | |
| `deleted_at` | datetime | 软删时间 | INDEX |

索引：`uk_ip_deleted` UNIQUE (`ip`, `deleted_at`)，`idx_status`，`idx_os`。

### 5.2 Group 表

| 表名 | `biz_cmdb_group` |
| :--- | :--- |

| 字段 | 类型 | 说明 | 约束 |
| :--- | :--- | :--- | :--- |
| `id` | bigint | 主键 | PK, AUTO_INCREMENT |
| `parent_id` | bigint | 上级分组 ID，`0` 表示根分组 | INDEX, DEFAULT 0 |
| `name` | varchar(128) | 分组名称 | NOT NULL |
| `conditions` | json | 过滤条件表达式 | NOT NULL |
| `description` | varchar(512) | 说明 | 可选 |
| `created_at` | datetime | 创建时间 | NOT NULL |
| `updated_at` | datetime | 更新时间 | NOT NULL |
| `deleted_at` | datetime | 软删时间 | INDEX |

### 5.3 LabelSchema 表

| 表名 | `biz_cmdb_label_schema` |
| :--- | :--- |

| 字段 | 类型 | 说明 | 约束 |
| :--- | :--- | :--- | :--- |
| `id` | bigint | 主键 | PK, AUTO_INCREMENT |
| `key` | varchar(64) | 标签键 | UNIQUE, NOT NULL |
| `name` | varchar(128) | 展示名称 | NOT NULL |
| `value_mode` | varchar(16) | 取值模式：`free` / `enum` / `dict` | NOT NULL |
| `dict_code` | varchar(64) | 当 `value_mode=dict` 时引用的业务字典编码 | 可选 |
| `options` | json | CMDB 标签常用值清单 | `["西安开发环境","西安测试环境"]` |
| `required` | boolean | 是否建议主机必须填写 | DEFAULT false |
| `status` | varchar(16) | `enabled` / `disabled` | INDEX |
| `description` | varchar(512) | 说明 | 可选 |
| `created_at` | datetime | 创建时间 | NOT NULL |
| `updated_at` | datetime | 更新时间 | NOT NULL |
| `deleted_at` | datetime | 软删时间 | INDEX |

### 5.4 设计约束

- 表前缀 `biz_`。
- 所有查询通过 GORM 构建，不写裸 SQL，确保多数据库兼容（MySQL / PostgreSQL / SQLite）。
- JSON 字段使用数据库原生 JSON 类型（GORM `datatypes.JSON`）。
- 主机列表、详情、编辑、删除、采集、状态更新均接入 `DataScopeReq + WithDataScope` 扩展位。
- 分组列表、详情、成员查询均基于 `DataScopeReq` 计算可见主机集。
- `dept_id` 预留数据范围归属，与系统域部门树联动；新增主机时默认落当前登录主体部门。
- 标签规范删除前必须检查 Host 标签和 Group 条件引用，避免分组条件静默失效。

### 5.5 租户就绪判断

- 当前为单租户运行，`tenant_id` 暂不加入 DDL。
- `ip` 唯一键当前为平台全局唯一。若进入多租户，需调整为 `(tenant_id, ip)` 组合唯一。
- 列表接口已预留 `DataScopeReq`，未来可统一注入 tenant 过滤。

## 6. API 设计

API 前缀：`/api/v1/business/cmdb`

### 6.1 Host API

| 方法 | 路径 | 说明 | 权限点 |
| :--- | :--- | :--- | :--- |
| `GET` | `/hosts` | 主机列表（支持标签筛选） | `business:cmdb:host:list` |
| `GET` | `/hosts/:id` | 主机详情 | `business:cmdb:host:detail` |
| `POST` | `/hosts` | 手动录入主机 | `business:cmdb:host:create` |
| `PUT` | `/hosts/:id` | 编辑主机信息 | `business:cmdb:host:update` |
| `DELETE` | `/hosts/:id` | 删除主机（软删） | `business:cmdb:host:delete` |
| `POST` | `/hosts/:id/collect` | SSH 一次性配置采集 | `business:cmdb:host:collect` |
| `PATCH` | `/hosts/:id/status` | 更新主机状态（下游系统调用） | `business:cmdb:host:status` |

列表查询参数：`?keyword=&status=&os=&label.env=&label.biz=&page=&pageSize=`。

`POST /hosts/:id/collect` 请求体：

```json
{
  "ssh_user": "root",
  "ssh_password": "",
  "ssh_private_key": "-----BEGIN OPENSSH PRIVATE KEY-----\n...",
  "auth_mode": "password"
}
```

- `auth_mode`：`password` 或 `private_key`。
- 密码或私钥不持久化，请求结束后立即从内存释放。

### 6.2 Group API

| 方法 | 路径 | 说明 | 权限点 |
| :--- | :--- | :--- | :--- |
| `GET` | `/groups` | 分组列表 | `business:cmdb:group:list` |
| `GET` | `/groups/:id` | 分组详情（含成员列表） | `business:cmdb:group:detail` |
| `GET` | `/groups/:id/members` | 分组内主机列表（实时计算） | `business:cmdb:group:detail` |
| `POST` | `/groups` | 创建分组 | `business:cmdb:group:create` |
| `PUT` | `/groups/:id` | 编辑分组条件 | `business:cmdb:group:update` |
| `DELETE` | `/groups/:id` | 删除分组 | `business:cmdb:group:delete` |

### 6.3 Label API

| 方法 | 路径 | 说明 | 权限点 |
| :--- | :--- | :--- | :--- |
| `GET` | `/labels` | 标签规范列表 | `business:cmdb:label:list` |
| `POST` | `/labels` | 创建标签规范 | `business:cmdb:label:create` |
| `PUT` | `/labels/:id` | 编辑标签规范 | `business:cmdb:label:update` |
| `DELETE` | `/labels/:id` | 删除标签规范 | `business:cmdb:label:delete` |

### 6.4 错误 Key

仓库级 canonical 清单以 `docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md` 为准；本节保留 CMDB 局部摘要。

后端错误 key 统一收口为 `business.cmdb.*` 命名空间，和 Deploy 保持同一层级。

推荐拆分：

- 主机资源：`business.cmdb.host.*`
- 分组资源：`business.cmdb.group.*`
- 标签规范：`business.cmdb.label.*`
- 采集链路：`business.cmdb.collect.*`

| Key | 说明 |
| :--- | :--- |
| `business.cmdb.host.notFound` | 主机不存在 |
| `business.cmdb.host.ipExists` | IP 已存在 |
| `business.cmdb.host.invalidLabel` | 标签格式无效 |
| `business.cmdb.host.unsupportedOs` | 不支持的操作系统 |
| `business.cmdb.collect.sshConnectFailed` | SSH 连接失败 |
| `business.cmdb.collect.sshAuthFailed` | SSH 认证失败 |
| `business.cmdb.collect.executionFailed` | 配置采集执行失败 |
| `business.cmdb.group.notFound` | 分组不存在 |
| `business.cmdb.group.invalidConditions` | 过滤条件表达式无效 |
| `business.cmdb.label.keyExists` | 标签键已存在 |
| `business.cmdb.label.invalid` | 标签规范配置无效 |
| `business.cmdb.label.inUse` | 标签已被主机或分组引用 |
| `business.cmdb.label.notFound` | 标签规范不存在 |

## 7. 权限模型

页面 / 导航权限：

- `business:cmdb:host:view`
- `business:cmdb:group:view`
- `business:cmdb:label:view`

### 7.1 Host 权限点

| 权限点 | 控制粒度 |
| :--- | :--- |
| `business:cmdb:host:view` | 主机列表页进入权限 / 导航可见性 |
| `business:cmdb:host:list` | 主机列表查询接口 |
| `business:cmdb:host:detail` | 详情按钮 + 接口 |
| `business:cmdb:host:create` | 新增按钮 + 接口 |
| `business:cmdb:host:update` | 编辑按钮 + 接口 |
| `business:cmdb:host:delete` | 删除按钮 + 接口 |
| `business:cmdb:host:collect` | 采集配置按钮 + 接口 |
| `business:cmdb:host:status` | 状态更新接口（供下游系统） |

### 7.2 Group 权限点

| 权限点 | 控制粒度 |
| :--- | :--- |
| `business:cmdb:group:view` | 分组列表页进入权限 / 导航可见性 |
| `business:cmdb:group:list` | 分组列表查询接口 |
| `business:cmdb:group:detail` | 详情按钮 + 接口 |
| `business:cmdb:group:create` | 新增按钮 + 接口 |
| `business:cmdb:group:update` | 编辑按钮 + 接口 |
| `business:cmdb:group:delete` | 删除按钮 + 接口 |

### 7.3 Label 权限点

| 权限点 | 控制粒度 |
| :--- | :--- |
| `business:cmdb:label:view` | 标签规范页进入权限 / 导航可见性 |
| `business:cmdb:label:list` | 标签规范列表查询接口 |
| `business:cmdb:label:create` | 新增标签规范 |
| `business:cmdb:label:update` | 编辑标签规范 |
| `business:cmdb:label:delete` | 删除标签规范 |

约束：

- 页面权限、按钮权限和接口权限分层，互不兜底。
- 不允许用 `list` 权限兜底新增/编辑/删除。
- `view` 只控制页面进入和导航可见；`list` 只控制列表数据查询。
- 详情页权限与列表页权限分离：有 `view` 不代表有 `detail`，有 `detail` 也不等于拥有更新或删除动作。

## 8. 菜单与路由设计

CMDB 归属一级导航"运维平台"（`operations`），作为运维平台第一个子模块。

| 菜单 key | 路径 | 标题 key | routeName | module | 组件键 | pagePermission | activeMenu | 类型 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `operations` | `/operations` | `operations.menu` | `operations-root` | `platform.operations` | — | — | — | `D` |
| `operations.cmdb` | `/operations/cmdb` | `operations.cmdb.menu` | `operations-cmdb-root` | `business.cmdb` | — | — | — | `M` |
| `operations.cmdb.host` | `/operations/cmdb/host` | `operations.cmdb.host.menu` | `operations-cmdb-host-list` | `business.cmdb` | `business/cmdb/host/CmdbHostList` | `business:cmdb:host:view` | — | `C` |
| `operations.cmdb.group` | `/operations/cmdb/group` | `operations.cmdb.group.menu` | `operations-cmdb-group-list` | `business.cmdb` | `business/cmdb/group/CmdbGroupList` | `business:cmdb:group:view` | — | `C` |
| `operations.cmdb.label` | `/operations/cmdb/label` | `operations.cmdb.label.menu` | `operations-cmdb-label-list` | `business.cmdb` | `business/cmdb/label/CmdbLabelSchemaList` | `business:cmdb:label:view` | — | `C` |

- 一级 `operations` 是目录（`D`），不绑定组件。
- 主机详情页/编辑页不作为侧边栏菜单，从列表页内跳转。
- 菜单标题统一使用 `titleKey`，组件键进入前端注册表和后端白名单。
- `module` 固定为 `business.cmdb`，页面进入权限统一走 `pagePermission`

## 9. 前端页面设计

### 9.1 主机列表页 `CmdbHostList`

- **骨架类型**：系统页模板（hero + 筛选区 + 表格 + 分页）。
- **筛选区**：关键词搜索、状态（online/offline/pending/maintenance）、OS（linux/windows）、标签 key-value 过滤、所属分组下拉。
- **表格列**：主机名、IP、OS、CPU/内存/磁盘、标签（Tag 组件）、状态（Badge）、已装组件（Tooltip）、操作按钮。
- **操作按钮**：详情、编辑、删除、采集配置。
- **页面状态**：加载中、空数据（引导新建或 SSH 采集）、无权限、服务端错误。

### 9.2 主机详情页 `CmdbHostDetail`

- **骨架类型**：系统页模板（hero + 详情卡片 + 标签/组件面板）。
- **基本信息区**：主机名、IP、SSH 端口、OS、负责人、备注。
- **硬件配置区**：CPU 核数、内存、磁盘，旁边放"采集配置"按钮（SSH 弹窗）。
- **标签区**：标签列表（可增删改），支持自定义 key。
- **已装组件区**：组件名 + 版本列表（由下游系统写入，CMDB 只读展示）。
- **状态**：状态 Badge + 变更操作（`PATCH /status`）。

### 9.3 主机表单 `CmdbHostForm`（新增/编辑）

- **骨架类型**：标准表单（Modal 或独立页）。
- **字段**：主机名、IP、SSH 端口、OS 选择（linux/windows）、OS 版本、CPU 核数、内存 (GB)、磁盘 (GB)、标签编辑、负责人、备注。
- **标签编辑**：key-value 多行编辑器，key 提供预置候选项（env/biz/cluster/region/db_type），value 自由输入。
- **表单校验**：IP 格式、端口范围、必填字段。

### 9.4 分组列表页 `CmdbGroupList`

- **骨架类型**：左树右表的系统页模板。
- **左侧树**：按 `parent_id` 展示真实分组层级，节点展示分组名称 + 成员数量，支持选择当前分组。
- **右侧表格**：展示分组列表，包含上级分组、条件摘要、成员数量。
- **操作按钮**：查看成员、编辑、新增子分组、删除。存在子分组时不允许删除父分组。

### 9.5 分组表单 `CmdbGroupForm`（新增/编辑）

- **骨架类型**：表单（Modal 或独立页）。
- **字段**：分组名称、上级分组、说明、条件编辑器。
- **条件编辑器**：多行 key-op-value 编辑器，AND/OR 切换。每行包含：key 输入（带下拉候选项）、操作符选择（eq/neq/in/notIn）、value 输入。支持增加/删除行。

### 9.6 标签规范页 `CmdbLabelSchemaList`

- **骨架类型**：标准列表 + Modal 表单。
- **字段**：标签键、标签名称、取值模式、字典编码、可选值、是否必填、状态、说明。
- **删除保护**：当标签键已出现在 Host 标签或 Group 条件中时，删除失败并返回 `business.cmdb.label.inUse`。
- **主机/分组表单联动**：主机标签和分组条件的 key 使用启用状态的 LabelSchema 作为候选；value 根据 key 联动展示 `options`，`free` 标签仍允许手动输入。

### 9.7 页面状态规范

所有页面必须覆盖：加载中、空数据、空筛选、无权限（403）、服务端错误、提交中（表单）、删除确认弹窗。状态变体的具体视觉、文案、ARIA 标注遵循 base 的 `EMPTY_LOADING_ERROR_STATES.md`。

### 9.8 主题与可达性

- 配色使用 base 的主题 token（`THEME_TOKENS_REFERENCE.md`），不在 CMDB 内引入新颜色变量
- 暗色模式适配遵循 base 的 `DARK_MODE_DESIGN.md`，不为 CMDB 单独写暗色样式
- 键盘、ARIA、焦点环遵循 base 的 `ACCESSIBILITY.md`；表格、抽屉、命令面板按其规范实现

### 9.9 响应式

- 主机列表表格列按 `MOBILE_RESPONSIVE_BREAKPOINTS.md` §4 的列优先级机制声明；`xs/sm` 断点下切换为卡片视图
- 分组页左树右表布局在 `md` 以下退化为单列（树折叠到顶部下拉）

引用规范（位于 base，通过 workspace 继承）：

- `../../../pantheon-base/docs/designs/FRONTEND_UI_SPEC.md`
- `../../../pantheon-base/docs/designs/FRONTEND_PAGE_TEMPLATES.md`
- `../../../pantheon-base/docs/designs/BACKOFFICE_STYLE_CONSTRAINTS.md`
- `../../../pantheon-base/docs/designs/ACCESSIBILITY.md`
- `../../../pantheon-base/docs/designs/THEME_TOKENS_REFERENCE.md`
- `../../../pantheon-base/docs/designs/DARK_MODE_DESIGN.md`
- `../../../pantheon-base/docs/designs/EMPTY_LOADING_ERROR_STATES.md`
- `../../../pantheon-base/docs/designs/MOBILE_RESPONSIVE_BREAKPOINTS.md`
- `../../../pantheon-base/DESIGN.md`

## 10. 多语言设计

i18n key 前缀：`business.cmdb.*`。

按层级：

| 层级 | 前缀 | 覆盖内容 |
| :--- | :--- | :--- |
| 菜单 | `operations.cmdb.*` | 导航标题 |
| 主机 | `business.cmdb.host.*` | 页面标题、表格列、表单字段、按钮、空态、错误 |
| 分组 | `business.cmdb.group.*` | 页面标题、表单字段、按钮、空态、错误 |
| 标签 | `business.cmdb.label.*` | 标签编辑、预置 key 名称 |
| 采集 | `business.cmdb.collect.*` | SSH 弹窗、连接错误、成功提示 |

至少补齐：

- 菜单标题、页面标题、面包屑
- 筛选区字段标签
- 表格列名
- 表单字段标签、占位符、校验消息
- 按钮文案
- 状态文案（pending/online/offline/maintenance）
- OS 文案（Linux/Windows）
- 预置标签 key 的显示名
- 错误提示、空状态文案、删除确认文案

后端错误 key 前缀：`business.cmdb.*`（见 6.4 节与 `BUSINESS_ERROR_SEMANTICS_APPENDIX.md`），与前端 i18n 分离，但必须共享同一业务语义命名空间。

## 11. 字典与配置依赖

### 11.1 依赖字典

| 字典类型 | 用途 | 字典项 |
| :--- | :--- | :--- |
| `cmdb_host_status` | 主机状态枚举 | `pending`、`online`、`offline`、`maintenance` |
| `cmdb_os_type` | 操作系统类型 | `linux`、`windows` |
| `cmdb_label_key` | 预置标签 key | `env`、`biz`、`cluster`、`region`、`db_type` |
| `cmdb_env` | 环境标签取值 | `dev`、`test`、`prod` |

CMDB 的标签规范归 `business/cmdb` 所有。它可以通过 `dict_code` 引用系统字典项作为候选值，但字典模块不理解"哪些标签用于分组、哪些标签必填、哪些标签被主机引用"这些业务规则。

### 11.2 依赖配置

| 配置 key | 用途 | 默认值 |
| :--- | :--- | :--- |
| `cmdb.ssh.collect_timeout` | SSH 采集连接超时（秒） | `10` |
| `cmdb.ssh.default_port` | 默认 SSH 端口 | `22` |

配置通过 `system/config` 的加密配置能力管理，不走环境变量或硬编码。

## 12. 审计与安全要求

### 12.1 审计点

| 动作 | 审计 action | 记录内容 |
| :--- | :--- | :--- |
| 新增主机 | `business.cmdb.host.audit.create` | 主机名、IP |
| 编辑主机 | `business.cmdb.host.audit.update` | 变更字段 |
| 删除主机 | `business.cmdb.host.audit.delete` | 主机名、IP |
| SSH 采集 | `business.cmdb.host.audit.collect` | 主机名、IP、采集结果 |
| 新增分组 | `business.cmdb.group.audit.create` | 分组名 |
| 编辑分组 | `business.cmdb.group.audit.update` | 变更条件 |
| 删除分组 | `business.cmdb.group.audit.delete` | 分组名 |

后续导入、导出、批量操作必须补对应审计点。

### 12.2 安全要求

- **零凭据持久化**：SSH 密码/私钥仅在 HTTP 请求生命周期内存存在，不回显、不入库、不写日志。
- **私钥优先**：SSH 采集弹窗中提示用户优先使用私钥认证。
- **审计不可抵赖**：所有 SSH 采集操作记录审计日志，含操作人、时间、目标主机。
- **表单防抖**：主机创建/编辑提交按钮防重复提交。
- **软删保护**：主机和分组均使用软删除，误删可恢复。
- **Windows WinRM**：后续实现时同样遵循"即采即弃"策略，不持久化凭据。

## 13. Seed 与初始化

CMDB 模块需要以下 seed：

| 类别 | 内容 | 说明 |
| :--- | :--- | :--- |
| 菜单 | `operations`、`operations.cmdb`、`operations.cmdb.host`、`operations.cmdb.group` | 一级导航 + 子模块 |
| 权限 | `business:cmdb:host:*`（8 项）、`business:cmdb:group:*`（6 项）、`business:cmdb:label:*`（5 项） | 见第 7 节 |
| i18n | `business.cmdb.*`、`operations.*` | 中英文语言包 |
| 字典 | `cmdb_host_status`、`cmdb_os_type`、`cmdb_label_key` | 见 11.1 节 |
| 配置 | `cmdb.ssh.collect_timeout`、`cmdb.ssh.default_port` | 见 11.2 节 |

组件键注册：

- `business/cmdb/host/CmdbHostList` → 前端 `componentRegistry` + 后端菜单组件白名单
- `business/cmdb/group/CmdbGroupList` → 前端 `componentRegistry` + 后端菜单组件白名单

## 14. 风险与边界外事项

### 14.1 明确不做

- 完整的 ITIL CMDB（CI 类型建模、关系拓扑图、变更影响分析）。
- 自动发现（IP 段扫描、云 API 同步），属于后续 Agent 或部署管理阶段。
- 主机存活探测与告警，属于监控告警子系统。
- 凭据库或堡垒机集成，CMDB 只需 SSH 即采即弃，堡垒机后续独立设计。
- 批量导入导出（Excel/CSV），第一版不做，后续按需加入。

### 14.2 风险

| 风险 | 缓解 |
| :--- | :--- |
| SSH 采集依赖网络可达 | 手动录入作为降级方案，采集失败不影响 CMDB 可用性 |
| 标签 key 自由扩展导致混乱 | 预置 key 提供语义约定，管理视图可展示未使用 key 的清理建议 |
| 分组条件复杂度膨胀 | 首版仅支持 AND/OR + eq/neq/in/notIn，不引入嵌套子表达式 |
| 多数据库兼容性 | 所有查询走 GORM，不使用原生 SQL 或数据库特定语法 |

## 15. 测试与验收

### 15.1 接口测试

- Host CRUD 全链路。
- Group CRUD + 成员计算。
- SSH 采集成功/失败场景（连接超时、认证失败、不支持 OS）。
- 状态流转。
- 重复 IP 拦截。

### 15.2 权限测试

- 无权限用户不可见运维平台菜单。
- 有 `view` 无 `list` 用户，可进入页面骨架但列表接口返回 403 或页面展示无权限数据态。
- 有 `list` 无 `create` 用户，新增按钮不可见且创建接口 403。
- 有 `view + list` 无 `detail` 用户，详情按钮不可见且详情接口 403。
- 无权限用户调用采集接口返回 403。

### 15.3 UI 状态测试

- 空列表 → 空状态引导。
- 筛选无结果 → 空状态提示调整筛选项。
- 删除确认弹窗。
- 表单校验错误提示。
- 提交按钮防抖。
- 加载态骨架屏。

### 15.4 多语言测试

- 中英文切换后菜单、页面、表单、状态文案完整。
- 错误提示中英文对应。

### 15.5 浏览器 smoke

- 命令：`cd frontend && npm run test:smoke:business:cmdb`
- 覆盖：登录后访问 `/operations/cmdb/host`、`/operations/cmdb/host/1`、`/operations/cmdb/group`。
- 断言：侧栏 `operations.cmdb.menu` 显示为 `CMDB`，主机页存在 hero 与筛选区，分组页存在左侧分组树容器；当无分组数据时展示空状态而不是隐藏业务入口。
- 证据截图：`tmp/cmdb-qa/cmdb-host-list.png`、`tmp/cmdb-qa/cmdb-host-detail.png`、`tmp/cmdb-qa/cmdb-group-list.png`。

### 15.6 审计测试

- 新增/编辑/删除主机产生审计记录。
- 新增/编辑/删除分组产生审计记录。
- SSH 采集产生审计记录（含结果）。

### 15.7 数据范围与权限收口（Host）

- 确认 `business/cmdb/host` 的菜单挂接在业务域入口下。
- 确认主机列表、详情、表单都沿用系统页模板的 hero / card / modal 视觉节奏，而不是平台页的旧式散装样式。
- 确认列表接口接入 `DataScopeReq + WithDataScope`。
- 确认业务路由接入 `DataScopeMiddleware`，角色数据范围策略由 `/system/permission` 的"数据权限"页统一配置。
- 确认 `biz_cmdb_host.dept_id` 作为数据范围字段参与 `dept / dept_and_children / custom` 过滤。
- 自动化证据：`go test ./backend/modules/business/cmdb/host` 覆盖 `dept_and_children` 有权限/无权限数据集过滤。
- 确认创建、编辑、删除有审计 action。
- 确认按钮权限不复用列表权限。
- 确认页面进入权限使用 `business:cmdb:host:view`，不与列表查询权限混用。
- 确认错误 key 使用 `business.cmdb.host.*` / `business.cmdb.collect.*`。

### 15.8 数据范围与权限收口（Group）

- 确认 `business/cmdb/group` 的成员列表和成员数量都基于当前请求的数据范围计算。
- 确认分组详情与分组成员接口不会越权读取不可见主机。
- 确认分组条件在创建/更新时做基础合法性校验，避免空规则和非法操作符。
- 确认分组页左侧采用树形结构，右侧承载当前选中分组的表格视图或成员抽屉，不再只用单表平铺。
- 自动化证据：`go test ./backend/modules/business/cmdb/group` 覆盖成员过滤和条件校验。
- 确认创建、编辑、删除有审计 action。
- 确认页面进入权限使用 `business:cmdb:group:view`，不与列表查询权限混用。
- 确认错误 key 使用 `business.cmdb.group.*` / `business.cmdb.label.*`。

引用验收标准（位于 base）：

- `../../../pantheon-base/docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- `../../../pantheon-base/docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`
