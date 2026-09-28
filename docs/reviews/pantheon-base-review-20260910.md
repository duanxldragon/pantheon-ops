最终以 Base 1.x 可冻结为目标
# Pantheon Base 功能补充与租户设计方案

> 文档定位：Pantheon Base 通用企业级后台基础平台功能补充设计
>
> 核心原则：
> 1. 只补通用基础能力，不追求功能大而全
> 2. 不为了对标其他后台系统而机械增加功能
> 3. 优先完善 IAM、审计、API、文件、缓存、任务等基础能力
> 4. 租户能力纳入规划，但采用分阶段建设方式
> 5. Base 不承载 OA、审批、消息中心等业务系统能力
> 6. 保持 Base 与 pantheon-ops 的低耦合、高内聚

---

# 一、项目定位

Pantheon Base 的定位不是一个完整的 OA 系统，也不是一个 SaaS 业务系统。

它应该定位为：

> **通用企业级后台应用基础平台 / Backend Admin Foundation**

为上层业务系统提供：

- 用户与身份管理
- 组织与部门管理
- 角色与权限管理
- 菜单与 API 权限
- 系统配置
- 字典
- 审计
- 文件存储
- 缓存
- 通用任务
- 租户基础能力

上层业务系统，例如：

- pantheon-ops
- CMDB
- 运维平台
- 其他企业管理系统
- SaaS 应用

在此基础上进行业务扩展。

---

# 二、功能补充总体结论

根据当前 Base 的功能定位，对参考功能清单进行重新划分。

| 功能 | 当前状态 | 最终决策 | 优先级 |
|---|---|---|---|
| 用户管理 | 已有 | 保留并补强 | P0 |
| 角色管理 | 已有 | 保留并补强 | P0 |
| 组织管理 | 已有 | 保留并补强 | P0 |
| 部门管理 | 已有 | 保留并补强 | P0 |
| 权限管理 | 已有 | 保留并补强 | P0 |
| 菜单管理 | 已有 | 保留并补强 | P0 |
| 权限点 | 已有 | 保留并补强 | P0 |
| 字典管理 | 已有 | 保留并补强 | P0 |
| 登录日志 | 已有 | 保留并补强 | P1 |
| 操作日志 | 已有 | 保留并补强 | P1 |
| 个人中心 | 已有 | 保留 | P1 |
| API 管理 | 待确认/补充 | 建议增加 | P1 |
| 文件管理 | 待补充 | 建议增加 | P1 |
| 缓存管理 | 待补充 | 建议增加 | P1 |
| 任务系统 | 待补充 | 建议增加 | P1 |
| 租户管理 | 未完整实现 | **纳入规划并建设** | P1 |
| 租户套餐 | 未实现 | 暂不实现 | P3 |
| 租户管理员登录 | 未实现 | 随租户一起设计 | P2 |
| 角色组 | 未实现 | 暂不实现 | P2 |
| 用户模拟登录 | 未实现 | 暂不实现 | P2 |
| 上级/主管 | 未实现 | 暂不实现 | P2 |
| 消息分类 | 未实现 | 不实现 | P3 |
| 消息管理 | 未实现 | 不实现 | P3 |
| 站内信 | 未实现 | 不实现 | P3 |
| OA/审批 | 未实现 | 明确排除 | P3 |

---

# 三、第一阶段重点：IAM

IAM 是 Pantheon Base 最核心的能力。

目标模型：

User → Organization / Department → Role → Permission

权限模型：

Permission
├── Menu
├── Permission Point
└── API

---

## 3.1 用户管理

### 必须具备

- 用户新增
- 用户编辑
- 用户查询
- 用户启用/禁用
- 密码重置
- 用户删除
- 用户角色关联
- 用户部门关联
- 用户组织关联
- 用户登录状态管理

### 建议补强

- 用户状态
- 最后登录时间
- 登录 IP
- 密码策略
- 密码过期策略
- 用户删除约束
- 管理员保护
- 批量启用/禁用
- 批量分配角色

### 暂不实现

- 主管/上级体系
- 用户模拟登录
- OA 人员关系
- HR 数据同步

这些不属于 Base 的核心职责。

---

# 四、组织与部门

组织模型建议保持通用。

```text
Organization
    │
    ├── Department
    │       ├── Department
    │       └── Department
    │
    └── User

重点保证：

树形结构
层级查询
用户归属
部门负责人（如果已有）
删除约束
移动部门
数据权限扩展能力

不要加入：

考勤
汇报关系
HR
薪资
OA 审批
五、角色与权限

这是 Base 的核心竞争力之一。

建议最终形成：

User
  ↓
Role
  ↓
Permission
  ├── Menu
  ├── Permission Point
  └── API
5.1 角色

支持：

创建角色
编辑角色
删除角色
启用/禁用
用户关联
菜单授权
权限点授权
API 授权

暂不增加：

角色组
复杂角色继承

除非后续实际业务出现需求。

六、API 管理

建议作为 Base 的重要补充能力。

目标：

API
 ↓
Permission Point
 ↓
Role
 ↓
User

API 管理建议支持：

API 列表
HTTP Method
Path
Service/Module
API 描述
API 分组
权限点绑定
API 自动同步
API 树形展示

例如：

用户管理
├── GET /users
├── POST /users
├── PUT /users/{id}
└── DELETE /users/{id}
6.1 为什么 API 管理值得加入 Base

因为未来：

pantheon-ops
CMDB
K8s
安装部署

都会拥有大量 API。

如果权限只绑定菜单：

菜单 → 权限

实际上无法很好解决接口授权。

最终应该形成：

菜单
  │
权限点
  │
API

但三者不要强耦合。

七、字典管理

字典已经存在，不需要重复开发。

重点应该从“有没有”转向“是否企业级”。

重点检查：

字典类型唯一性
字典项唯一性
启用/禁用
排序
默认值
删除约束
字典缓存
缓存一致性
多语言
前后端统一使用
是否存在硬编码枚举
权限控制
审计

例如：

sys_user_status

1 → 启用
0 → 禁用

业务代码不应该到处出现：

if (status == 1)

而应该通过统一字典/枚举机制进行管理。

八、文件管理

建议补充。

但不要把它设计成一个简单的文件 CRUD。

应该设计成：

FileService
      │
StorageProvider
      │
 ┌────┼────────┐
Local   S3/OSS   MinIO

建议抽象：

StorageProvider
├── LocalStorageProvider
├── S3StorageProvider
└── OssStorageProvider

支持：

上传
下载
删除
文件查询
文件元数据
文件大小
MIME Type
MD5/SHA256
文件访问地址
图片预览
文件权限

Base 不应该绑定：

阿里云 OSS
腾讯云 COS
MinIO

而应该通过 Provider 抽象。

九、缓存管理

建议补充，但保持轻量。

主要提供：

Redis Key 查询
Key 删除
批量删除
TTL 查询
TTL 修改
缓存统计
缓存命名空间

例如：

pantheon:user:*
pantheon:role:*
pantheon:permission:*
pantheon:dict:*

必须特别注意：

缓存管理不能允许管理员无约束地操作所有 Redis 数据。

需要：

权限控制
Key 前缀隔离
敏感 Key 保护
操作审计
十、任务系统

建议加入，但不要做成大型任务调度平台。

Base 只负责通用任务模型。

建议：

Task
 ↓
Task Execution
 ↓
Execution Log
 ↓
State Machine

任务状态：

PENDING
   ↓
RUNNING
   ↓
SUCCESS

RUNNING
   ↓
FAILED
   ↓
RETRY
   ↓
MANUAL

核心能力：

创建任务
启动
暂停
恢复
取消
立即执行
执行记录
执行日志
失败状态
重试
超时
执行历史
10.1 Base 与 Ops 的任务边界

这是非常重要的一点。

Base：

Task
Execution
Executor
Log
State
Retry

Ops：

InstallExecutor
UninstallExecutor
UpgradeExecutor
RollbackExecutor
HealthCheckExecutor
BackupExecutor

这样可以直接复用。

例如：

pantheon-base
      │
      └── Task Framework
              │
              └── pantheon-ops
                     ├── VM Install
                     ├── VM Uninstall
                     ├── Upgrade
                     ├── Rollback
                     ├── K8s Deploy
                     └── Velero Backup
十一、租户设计
11.1 为什么加入租户

租户是 Pantheon Base 后续商业化的重要基础能力。

未来可以形成：

Pantheon Base
     │
     ├── 单租户企业部署
     │
     └── 多租户 SaaS

因此租户值得加入。

但是：

租户不是简单增加一个 Tenant 表。

它属于横向架构能力。

十二、租户设计原则

建议采用：

Tenant Context + 数据隔离

核心模型：

Tenant
   │
   ├── User
   ├── Organization
   ├── Department
   ├── Role
   └── Permission

即：

Tenant
  ↓
Organization
  ↓
Department
  ↓
User
  ↓
Role
  ↓
Permission
十三、Tenant 核心模型

建议至少包含：

Tenant
├── id
├── code
├── name
├── status
├── description
├── created_at
├── updated_at
└── ...

Tenant Code 应作为稳定的业务标识。

例如：

tenant_id = 10001
tenant_code = acme
十四、Tenant Context

这是租户体系最重要的基础设施之一。

请求进入系统：

HTTP Request
      ↓
Authentication
      ↓
Resolve Tenant
      ↓
TenantContext
      ↓
Business Service

业务代码可以通过：

TenantContext.getTenantId()

获取当前租户。

不要让业务代码到处自己解析：

Header
Token
Cookie
Session

否则以后维护会非常困难。

十五、租户识别方式

第一阶段建议支持：

登录 Token
      ↓
tenant_id

未来可以扩展：

域名
 ↓
tenant_code
 ↓
Tenant

例如：

acme.example.com
      ↓
acme
      ↓
Tenant

不要第一阶段就同时支持：

子域名
独立域名
Header
Cookie
URL
多种 Token

否则复杂度会快速上升。

十六、租户数据隔离

第一阶段建议采用：

共享数据库 + tenant_id

模型：

tenant
user
role
department
organization
...

业务表：

id
tenant_id
...

例如：

user

id | tenant_id | username
---|-----------|---------
1  | 10001     | admin
2  | 10002     | admin
十七、Tenant ID 的核心原则

一旦进入多租户模式：

所有租户业务数据必须明确属于某个 Tenant。

例如：

User
Role
Department
Organization
File
Task
AuditLog

都需要考虑租户归属。

但是：

不是所有系统表都必须强制 tenant_id。

例如：

SystemConfig
DictionaryDefinition
API Definition
Menu Definition

有些属于平台级资源。

因此应该明确区分：

Platform Resource
        vs
Tenant Resource
十八、平台资源与租户资源

建议分为两层。

Platform
Platform
├── API Definition
├── Menu Definition
├── Permission Definition
├── Dictionary Definition
└── System Configuration
Tenant
Tenant
├── User
├── Organization
├── Department
├── Role
├── User Permission
├── File
├── Task
└── Audit

这样避免所有数据都机械增加 tenant_id。

十九、租户权限模型

推荐：

Platform
   │
   └── Tenant
          │
          ├── User
          ├── Role
          └── Permission

租户管理员：

Tenant Admin
    │
    ├── Tenant Users
    ├── Tenant Departments
    ├── Tenant Roles
    └── Tenant Configuration

平台管理员：

Platform Admin
    │
    ├── Tenant Management
    ├── Tenant Status
    ├── Tenant Configuration
    └── Platform Management
二十、租户管理员

建议最终支持：

Platform Admin
       │
       └── Tenant
              │
              └── Tenant Admin

Tenant Admin 只能够管理自己租户。

例如：

Tenant A Admin
    ↓
只能看到 Tenant A

Tenant B Admin
    ↓
只能看到 Tenant B
二十一、租户状态

至少：

ACTIVE
DISABLED

未来可以增加：

EXPIRED
DELETED

第一阶段不要设计过多状态。

二十二、租户删除策略

不建议真正物理删除租户。

推荐：

ACTIVE
   ↓
DISABLED
   ↓
ARCHIVED

租户删除应该属于高风险操作。

必须：

权限控制
二次确认
操作审计
数据影响提示
二十三、租户套餐暂不实现

虽然很多 SaaS 系统会有：

Tenant
 ↓
Package
 ↓
Features

但是 Pantheon Base 当前阶段不建议做。

暂时不要增加：

套餐
计费
订阅
订单
支付
License
配额

这些属于商业 SaaS 层。

未来如果真正进入商业化，再单独设计：

Tenant
 ↓
Subscription
 ↓
Package
 ↓
Feature
 ↓
Quota
二十四、租户开发的分阶段策略

这是整个租户设计最重要的部分。

Phase 1：架构准备

目标：

不实现完整租户，只保证未来能够加入。

工作：

明确 Platform Resource / Tenant Resource
设计 Tenant Context
明确用户身份模型
明确权限模型
避免核心代码写死单租户逻辑
预留租户扩展点
Phase 2：核心 Tenant

实现：

Tenant
Tenant Context
Tenant Admin
Tenant Status
Tenant User
Tenant Organization
Tenant Department
Tenant Role

实现基础数据隔离。

Phase 3：完整租户体系

再实现：

租户配置
租户级缓存
租户级文件
租户级任务
租户级审计
租户级数据权限
租户域名
租户初始化
Phase 4：商业 SaaS

最后才考虑：

套餐
Feature
License
Quota
Subscription
Billing
计费
二十五、消息系统明确不做

以下功能不建议放进 Pantheon Base：

消息分类
消息管理
站内信
公告
审批
OA
工作流
考勤
通讯录

原因：

这些功能虽然“企业后台经常有”，但它们并不是后台底座的基础能力。

一旦加入，很容易形成：

Base
 ├── IAM
 ├── OA
 ├── Message
 ├── Workflow
 ├── HR
 └── Business

最终 Base 会失去边界。

二十六、最终 Base 模块结构

建议最终形成：

pantheon-base
│
├── iam
│   ├── user
│   ├── organization
│   ├── department
│   ├── role
│   ├── permission
│   ├── menu
│   └── api
│
├── tenant
│   ├── tenant
│   ├── tenant-context
│   ├── tenant-admin
│   └── tenant-isolation
│
├── system
│   ├── dictionary
│   ├── configuration
│   └── parameter
│
├── audit
│   ├── login-log
│   └── operation-log
│
├── storage
│   ├── file
│   └── provider
│
├── cache
│   └── cache-management
│
├── task
│   ├── task
│   ├── execution
│   ├── executor
│   └── execution-log
│
└── personal
    └── profile
二十七、Base 最终不应该出现的模块

明确排除：

OA
审批
考勤
HR
CRM
财务
消息中心
站内信
工作流业务
业务监控
Kubernetes
CMDB
部署
CI/CD

这些应该属于上层应用。

例如：

Pantheon Base
      │
      ├── IAM
      ├── Tenant
      ├── Audit
      ├── Storage
      ├── Cache
      └── Task
             │
             ↓
       Pantheon Ops
             │
             ├── CMDB
             ├── Business Domain
             ├── Deployment
             ├── Kubernetes
             ├── Monitoring
             └── CI/CD
二十八、推荐的最终开发优先级
P0：必须完成
IAM
├── User
├── Organization
├── Department
├── Role
├── Permission
├── Menu
└── Dictionary

Audit
├── Login Log
└── Operation Log

重点不是增加功能，而是做企业级质量。

P1：建议补齐
API Management
File Storage
Cache Management
Task Framework
Tenant Foundation

这些能力具有较强复用价值。

P2：租户核心
Tenant
Tenant Context
Tenant Admin
Tenant User
Tenant Organization
Tenant Department
Tenant Role
Tenant Isolation

完成后，Base 才真正具备多租户能力。

P3：明确暂缓
Tenant Package
Subscription
Billing
Quota
License
Domain Management

这些属于商业 SaaS 阶段。

二十九、最终验收标准

Pantheon Base 不应该以：

“后台功能越多越好”

作为完成标准。

而应该以：

核心基础能力是否稳定、可复用、可扩展、低耦合

作为完成标准。

最终验收重点：

IAM
 用户模型稳定
 组织模型稳定
 角色模型稳定
 权限模型稳定
 菜单权限稳定
 API 权限稳定
 数据权限具备扩展能力
System
 Dictionary
 Configuration
 Parameter
Audit
 Login Audit
 Operation Audit
 Sensitive Operation Audit
Infrastructure
 Storage Provider
 Cache
 Task Framework
Tenant
 Tenant Model
 Tenant Context
 Tenant Isolation
 Tenant Admin
 Tenant-aware IAM
 Tenant-aware Audit
 Tenant-aware Storage
 Tenant-aware Task
三十、最终建议

Pantheon Base 的目标不是：

“把别人后台系统有的功能全部做一遍。”

而应该是：

做一个真正能够支撑多个企业应用的基础平台。

因此最终建议收敛为：

                 Pantheon Base
                       │
        ┌──────────────┼──────────────┐
        │              │              │
       IAM           Tenant         System
        │              │              │
        │              │         Dictionary
        │              │         Configuration
        │              │
        └──────────────┼──────────────┘
                       │
                Infrastructure
                       │
        ┌──────────────┼──────────────┐
        │              │              │
     Storage         Cache          Task
        │              │              │
        └──────────────┼──────────────┘
                       │
                     Audit
                       │
              ┌────────┴────────┐
              │                 │
        Login Audit       Operation Audit
最终冻结边界

建议 Pantheon Base 在完成以下能力后进入功能冻结：

IAM
+
Tenant
+
Dictionary
+
API
+
Audit
+
Storage
+
Cache
+
Task

之后新需求原则上进入上层业务项目，而不是继续往 Base 里面塞。

特别是：

租户值得做，但不要一次性把“完整 SaaS”做出来。

先把 Tenant + TenantContext + 基础隔离 + Tenant Admin 做出来，套餐、计费、配额等全部后置。

这样既满足你希望 Pantheon Base 最终具备商业化能力的目标，又不会因为租户把整个 Base 的开发周期拖得过长。


我建议你把这份作为 **`pantheon-base` 后续开发的补充设计基线**。其中最关键的不是“多做几个页面”，而是**在现在这个阶段把 Tenant Context、Platform Resource / Tenant Resource 这两个概念设计正确**，这样以后真正做多租户时，才不会出现全项目大规模返工。