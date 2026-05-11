# 字典与系统设置设计

更新时间：2026-04-28

类型：Design
归属层：system/config
状态：Active

本文定义 Pantheon Base 的字典管理和系统设置设计。

这两个能力属于 `system/config` 能力域，是企业后台“通用底座”的关键组成：

- 字典解决“枚举值和下拉选项不要硬编码”
- 系统设置解决“平台参数不要散落在代码和环境变量里”

当前落地状态：

- `system/setting` 已完成基础闭环：模型、迁移、默认配置、公开读取、管理端分组读取与保存、前端设置页、设置缓存刷新；
- `system/dict` 已完成基础闭环：模型、迁移、默认种子、字典类型/字典项 CRUD、公共 options 接口、前端双标签工作台、字典项服务端分页筛选、类型项数统计、options 缓存刷新；
- 上传配置分组、敏感配置加密存储、配置变更审计详情、设置缓存策略已完成基础实现。
- `system/setting` 已补配置健康总览：可直接查看公开配置数量、敏感配置数量、缺失必填项、运行时风险与当前驱动/语言/主题摘要。
- 运行时接线状态已补第一批：`site.name/site.logo` 已接入登录页与应用壳层，`security.password_min_length` 已接入当前用户改密、用户创建与管理员重置密码，`login.max_failed_attempts/login.lock_minutes` 已接入登录失败锁定策略，`login.session_idle_minutes` 已接入平台壳层空闲超时与 `system/auth` 会话失效判定，`login.max_active_sessions_per_user` 已接入同账号活跃会话上限治理，`audit.login_log_retention_days/audit.operation_log_retention_days/audit.session_retention_days` 已接入登录日志、操作日志与历史会话的自动保留治理，`i18n.default_language` 已接入无显式语言偏好时的默认语言初始化，`ui.default_theme/ui.enable_tab_bar` 已接入平台壳层主题与标签栏显示策略。
- 平台壳层显式偏好已完成与默认值解耦：`i18n.default_language`、`ui.default_theme` 仍只负责公开默认值；用户一旦在登录后显式切换 `theme / language / layoutMode / densityMode`，该选择由 `auth/me/preferences` 持久化并优先于默认值生效。
- 上传配置也已形成基础运行时闭环：平台新增统一上传能力，`upload.max_file_size / upload.allowed_types / upload.local_path / upload.public_base_url` 已接入真实上传接口与文件访问地址生成；当前已支持 `local` 与 `s3-compatible` 两类驱动。
- 国际化配置治理已补运行时收口：当前平台内置 fallback locale 为 `zh-CN / en-US / ja-JP / ko-KR / fr-FR`，默认语言设置只在用户没有显式语言偏好时生效；一旦用户已切换语言，平台应优先尊重用户显式选择，而不是反向被默认配置覆盖。
- 当前语种策略为“按市场扩展，不做无依据预扩”：除非出现明确客户、区域交付或合规需求，否则不继续预置更多 locale；后续新增 locale 时，默认沿现有 i18n 导入导出、缺失检测和 fallback 校验链路扩展。
- `system/config -> generator` 已补第一阶段“受管数据源”治理：代码生成器保留当前平台库为默认来源，同时允许管理员维护外部 MySQL 只读数据源，并按数据源选择表结构导入。当前只开放元数据读取，不支持任意 SQL 执行。

## 1. 设计目标

- 统一管理业务枚举、状态选项、下拉选项
- 统一管理平台配置、安全配置、上传配置、UI 配置
- 支持缓存和刷新
- 支持模块化注册
- 支持后续业务模块复用

## 2. 能力边界

## 2.1 字典管理负责

- 字典类型
- 字典项
- 字典排序
- 字典状态
- 字典缓存刷新
- 前端下拉选项下发

## 2.2 系统设置负责

- 平台基础信息
- 安全策略
- 上传配置
- 登录策略
- 国际化默认设置
- UI 默认偏好
- 当前允许的默认语言值：`zh-CN`、`en-US`、`ja-JP`、`ko-KR`、`fr-FR`
- 后续如需新增默认语言值，必须先完成对应 locale 的 fallback 资源、运行时翻译资产、导入导出模板与回归验证，再进入设置可选项

`generator` 同样归属 `system/config`，但职责与 `setting` 不同：

- `setting` 管平台运行参数
- `generator` 管研发接入治理元数据（如受管数据源、导入来源）

## 2.3 不负责

字典和设置不负责：

- 用户权限判断
- 业务流程状态机
- 大体量业务数据
- 私密凭据明文存储

## 3. 模块归属

建议模块：

```text
backend/modules/system/dict/
backend/modules/system/setting/

frontend/src/modules/system/dict/
frontend/src/modules/system/setting/
```

能力域：

```text
system/config
```

## 4. 字典管理设计

## 4.1 字典模型

建议拆两张表：

- `system_dict_type`
- `system_dict_item`

## 4.2 `system_dict_type`

字段建议：

| 字段 | 说明 |
| :--- | :--- |
| `id` | 主键 |
| `dict_code` | 字典编码，唯一 |
| `dict_name` | 字典名称 |
| `module` | 模块归属 |
| `status` | 状态 |
| `remark` | 备注 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |
| `deleted_at` | 软删除 |

示例：

```text
system_user_status
system_yes_no
biz_order_status
```

## 4.3 `system_dict_item`

字段建议：

| 字段 | 说明 |
| :--- | :--- |
| `id` | 主键 |
| `dict_code` | 字典编码 |
| `item_label_key` | 展示文案 i18n key |
| `item_value` | 实际值 |
| `item_color` | 标签颜色 |
| `sort` | 排序 |
| `status` | 状态 |
| `remark` | 备注 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |
| `deleted_at` | 软删除 |

## 4.4 字典 i18n 规则

字典展示文案不直接存自然语言，优先存：

```text
item_label_key
```

例如：

```text
dict.system_user_status.enabled
dict.system_user_status.disabled
```

## 4.5 字典使用规则

前端使用字典时：

- 通过 `dict_code` 获取字典项
- 使用 `item_label_key` 翻译展示
- 使用 `item_value` 提交后端
- 使用 `item_color` 显示状态标签

后端使用字典时：

- 校验值是否合法
- 不依赖自然语言文案

## 4.6 字典管理台当前实现约束

- 字典管理前端已调整为“字典类型 / 字典项”双标签工作台，不再强依赖左右主从布局。
- 字典类型列表会返回聚合统计：`itemCount / activeItemCount / disabledItemCount / lastItemUpdatedAt`，用于工作台摘要与运维判断。
- 字典项列表已升级为服务端分页查询，支持 `dictCode + status + keyword + page + pageSize`，避免字典项增多后一次性全量拉取。
- 字典项导出会复用当前筛选条件，但不受分页限制。

## 5. 系统设置设计

## 5.0 生成器受管数据源补充

建议表：

- `system_generator_datasource`

字段职责：

- `name / driver / host / port / database_name / username`
- `password_encrypted`：只存加密后密码，不回传前端明文
- `status`：启用 / 禁用
- `readonly_scope`：当前固定为 `metadata_only`
- `last_checked_at / last_check_status / last_check_error`：连接测试审计

约束：

- 第一阶段仅支持 `MySQL`
- 仅允许读取 `information_schema`
- 默认平台库作为虚拟内置数据源，不落 `system_generator_datasource`

## 5.1 设置模型

建议表：

- `system_setting`

字段建议：

| 字段 | 说明 |
| :--- | :--- |
| `id` | 主键 |
| `setting_key` | 配置 key |
| `setting_value` | 配置值 |
| `value_type` | 值类型 |
| `group_key` | 配置分组 |
| `module` | 模块归属 |
| `is_public` | 是否允许前端公开读取 |
| `is_encrypted` | 是否加密存储 |
| `remark` | 备注 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |

当前管理接口返回的 `SettingResp` 除了当前值外，还会额外下发 `defaultValue` 元数据，供前端实现“恢复默认值”这类通用设置治理交互；前端不应再把默认值硬编码回页面。

## 5.2 value_type

建议支持：

- `string`
- `number`
- `boolean`
- `json`

## 5.3 group_key

建议分组：

- `basic`
- `security`
- `login`
- `audit`
- `upload`
- `i18n`
- `ui`

## 5.4 setting_key 示例

```text
site.name
site.logo
security.password_min_length
security.password_expire_days
login.max_failed_attempts
login.lock_minutes
login.session_idle_minutes
login.max_active_sessions_per_user
audit.login_log_retention_options
audit.operation_log_retention_options
audit.session_cleanup_retention_options
audit.login_log_retention_days
audit.operation_log_retention_days
audit.session_retention_days
upload.max_file_size
i18n.default_language
ui.default_theme
```

## 5.4.2 日志治理设置当前运行时语义

- `audit.login_log_retention_options`
  - 使用 JSON 数组维护登录日志清理允许的保留天数，例如 `[1,7,30]`
  - 由 `system/auth` 在执行 `/system/login-log/cleanup` 时动态校验
- `audit.login_log_retention_days`
  - 使用 number 维护登录日志自动保留天数，例如 `90`
  - 由 `system/auth` 在登录日志写入、列表和导出链路中按节流策略执行自动清理
- `audit.operation_log_retention_days`
  - 使用 number 维护操作日志自动保留天数，例如 `180`
  - 由 `system/audit` 在操作日志列表、详情和导出链路中按节流策略执行自动清理
- `audit.session_retention_days`
  - 使用 number 维护历史会话保留天数，例如 `90`
  - 由 `system/auth` 在登录建会话、当前用户会话查询、管理员会话查询前自动清理已下线或已过期的历史会话
  - 目标是限制 `system_user_session` 数据规模，避免审计列表总量长期无界增长
- `audit.session_cleanup_retention_options`
  - 使用 JSON 数组维护管理员手动清理历史会话时可选的保留天数，例如 `[1,7,30]`
  - 由 `system/auth` 在“会话管理 -> 清理历史会话”动作中动态校验
  - 目标是让会话清理交互与登录日志/操作日志保持一致
- `audit.operation_log_retention_options`
  - 使用 JSON 数组维护操作日志清理允许的保留天数，例如 `[1,7,30]`
  - 由 `system/audit` 在执行 `/system/operation-log/cleanup` 时动态校验
- 前端登录日志页与操作日志页会读取 `audit` 分组设置来渲染清理下拉选项，避免再次把保留期硬编码回页面

## 5.4.1 上传配置当前运行时语义

- `upload.storage_driver`
  - 当前支持值：`local`、`s3`
  - `s3` 表示接入 S3 兼容对象存储（如 MinIO / AWS S3 / OSS 兼容网关）
- `upload.max_file_size`
  - 单文件最大体积，单位 MB
  - 已接入 `/api/v1/system/upload`
- `upload.allowed_types`
  - JSON 数组，例如 `["jpg","jpeg","png"]`
  - 已接入上传扩展名白名单校验
- `upload.local_path`
  - 本地存储根目录
  - 已接入本地文件落盘与 `/api/v1/system/upload/files/*filepath` 文件读取
- `upload.public_base_url`
  - 用于生成上传后的公开访问 URL
  - 为空时默认回退到平台内置文件访问路径
- `upload.s3_endpoint`
  - 对象存储 Endpoint
  - 支持 `http(s)://host[:port]` 或裸 `host[:port]`
- `upload.s3_bucket`
  - 对象存储 Bucket 名称
- `upload.s3_region`
  - 对象存储 Region，默认 `us-east-1`
- `upload.s3_access_key_id / upload.s3_secret_access_key`
  - 对象存储访问凭据
  - 按敏感配置加密保存，管理端不回显明文

其中 `ui.default_theme` 当前建议枚举为：

```text
indigo
emerald
violet
slate
```

## 5.5 公开配置

允许前端公开读取的配置：

- 站点名称
- logo
- 默认语言
- 默认主题

当前默认主题应与平台层主题 token 体系一致，不再使用历史占位值 `light`。

禁止公开读取：

- 密钥
- token secret
- 存储凭据
- 任何敏感安全配置明文

## 6. API 设计

## 6.1 字典接口

管理接口：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/system/dict/type/list` | 字典类型列表 |
| `POST` | `/api/v1/system/dict/type` | 创建字典类型 |
| `PUT` | `/api/v1/system/dict/type/:id` | 更新字典类型 |
| `DELETE` | `/api/v1/system/dict/type/:id` | 删除字典类型 |
| `GET` | `/api/v1/system/dict/item/list` | 字典项列表 |
| `POST` | `/api/v1/system/dict/item` | 创建字典项 |
| `PUT` | `/api/v1/system/dict/item/:id` | 更新字典项 |
| `DELETE` | `/api/v1/system/dict/item/:id` | 删除字典项 |

公共读取接口：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/system/dict/options?codes=a,b` | 批量获取字典选项 |

## 6.2 设置接口

管理接口：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/system/setting/list` | 配置列表 |
| `GET` | `/api/v1/system/setting/group/:groupKey` | 按分组获取配置 |
| `POST` | `/api/v1/system/setting/cache/refresh` | 刷新设置缓存 |
| `PUT` | `/api/v1/system/setting/group/:groupKey` | 批量保存配置 |

公开读取接口：

| 方法 | 路径 | 说明 |
| :--- | :--- | :--- |
| `GET` | `/api/v1/system/setting/public` | 获取公开配置 |

## 7. 前端页面设计

## 7.1 字典管理页

页面模板：

- `ListPage`

建议布局：

```text
DictPage
  ├── DictTypeList
  └── DictItemList
```

交互：

- 左侧选择字典类型
- 右侧维护字典项
- 支持启用/停用
- 支持排序
- 当前版本未提供独立“刷新缓存”按钮，后续增强时再补缓存刷新入口

## 7.2 系统设置页

页面模板：

- `ConfigPage`

当前前端基线已补齐：

- 按 `basic / security / login / upload / i18n / ui` 分组维护配置
- 敏感配置不回显明文，提供“留空保持不变”交互
- 页面底部展示当前分组最近配置变更审计
- 审计项展示操作人、操作 IP、变更字段、状态、操作时间
- 敏感字段只显示“已变更”，不展示前后值

## 7.3 字典缓存刷新

当前基线实现：

- `GET /api/v1/system/dict/options` 使用进程内缓存提升公共字典读取效率
- 字典类型 / 字典项发生增删改时，自动失效对应 `dict_code` 缓存
- 管理端提供 `system:dict:refresh` 权限点与手动刷新按钮
- 手动刷新支持按 `codes` 精准刷新，未指定时清空全部缓存

建议分组：

- 基础信息
- 安全策略
- 登录策略
- 上传配置
- 国际化
- UI 偏好

## 8. 权限设计

建议权限点：

```text
system:dict:list
system:dict:create
system:dict:update
system:dict:delete

system:setting:list
system:setting:update
system:setting:refresh
```

其中当前已落地的字典权限点为：

```text
system:dict:list
system:dict:create
system:dict:update
system:dict:delete
```

`system:dict:refresh` 已落地，用于手动刷新字典 options 缓存。

当前已落地的系统设置权限点为：

```text
system:setting:list
system:setting:update
system:setting:refresh
```

### 8.1 系统设置页面的双轨权限约束

系统设置页当前不是“只有一个前端页面权限点”就足够。

要让一个角色稳定进入并加载 `/system/setting`，至少要同时满足两类权限：

1. 页面 / 操作权限
   - `system:setting:list`：允许进入设置页
   - `system:setting:update`：允许保存当前分组
   - `system:setting:refresh`：允许点击“刷新设置缓存”
2. Casbin 接口权限
   - `GET /api/v1/system/setting/list`
   - `GET /api/v1/system/menu/tree`
   - 如果还要展示统一审计详情或跳转审计页，还需要相应的 `system/audit` 页面权限与接口策略

这意味着：

- 只授予 `system:setting:list`，但没有配套 Casbin `GET /system/setting/list` 时，用户可能“能进路由但页面数据加载失败”；
- 只授予页面权限，不代表接口会自动放行；
- 角色联调时必须同时检查“前端页面权限”与“后端接口策略”两层。

### 8.2 低权限用户的审计区块降级规则

系统设置页底部的“配置变更审计”属于 `system/audit` 能力，不应强绑到所有能查看设置页的角色。

规则：

- 仅当用户具备统一审计查看能力时，才展示并拉取设置审计区块；
- 对只读设置角色，页面主体应可正常加载，但不默认请求审计数据；
- 不能因为缺少 `system/audit` 能力，就让 `system/config` 主页面直接退化成失败态。

## 9. 菜单设计

建议菜单：

```text
平台配置
  ├── 字典管理
  └── 系统设置
```

如果暂不新增一级“平台配置”，也可以先挂在系统管理下。

## 10. i18n key 规划

建议：

```text
system.menu.config
system.menu.dict
system.menu.setting

system.dict.type
system.dict.item
system.dict.dictCode
system.dict.dictName
system.dict.itemLabelKey
system.dict.itemValue
system.dict.refreshCache

system.setting.basic
system.setting.security
system.setting.login
system.setting.upload
system.setting.i18n
system.setting.ui
```

## 11. 缓存设计

## 11.1 字典缓存

字典适合缓存：

- Redis 可用时写 Redis
- Redis 不可用时走内存或数据库

缓存 key：

```text
dict:{dict_code}:{lang}
```

## 11.2 设置缓存

设置适合缓存：

```text
setting:public
setting:group:{group_key}
setting:list:{group_key}:{module}
```

## 11.3 刷新策略

- 修改字典项后刷新对应字典缓存
- 修改设置后自动失效 setting 相关缓存
- 支持按 `groupKeys` 手动刷新并预热 group 缓存
- 提供管理员手动刷新缓存入口

## 12. 与 i18n 的关系

字典项展示文案使用 i18n。

注意：

- 字典项不是 i18n 表
- i18n 负责翻译文案
- 字典负责枚举值和可选项

## 13. 与业务模块的关系

业务模块可以声明自己的字典：

```text
biz_order_status
biz_ticket_priority
```

但仍由底座字典模块统一管理和下发。

业务模块新增字典时必须同步：

- dict type seed
- dict item seed
- i18n seed
- 文档说明

## 14. 安全规则

系统设置里如果出现敏感配置：

- 不允许明文展示完整值
- 不允许公开接口返回
- 必须标记 `is_encrypted`
- 保存时必须加密或交给专门密钥系统

当前阶段建议：

- 敏感配置主密钥仍使用环境变量管理
- 敏感配置值允许存入系统设置，但必须以加密形式保存，管理端只显示“已配置”状态，不回显明文

## 15. 分阶段实现

## 15.1 Phase 1：字典管理

- DDL
- 后端 CRUD
- 前端字典页
- 字典 options 接口
- options 缓存刷新

## 15.2 Phase 2：系统设置（已完成基础闭环）

- DDL
- 后端分组读取/保存
- 前端 ConfigPage
- public setting 接口
- 敏感配置加密存储
- 配置变更审计详情

## 15.3 Phase 3：模块 seed

- 模块注册 seed

## 16. 当前落地差距

当前剩余增强项：

- 业务模块字典接入样例

## 17. 验收清单

完成时必须满足：

- 可以维护字典类型
- 可以维护字典项
- 前端可以批量获取字典 options
- 字典文案走 i18n
- 可以按分组维护系统设置
- 公开配置和敏感配置有边界
- 修改后缓存刷新
- 权限和审计接入

## 18. 下一份建议补的文档

下一份建议补：

- `docs/designs/BUSINESS_MODULE_TEMPLATE.md`
- `docs/business/ORDER_MODULE_DESIGN.md`

因为配置底座设计完成后，下一步就可以：

- 先定义业务模块接入模板；
- 再写一个真实业务模块样例验证整套底座。
