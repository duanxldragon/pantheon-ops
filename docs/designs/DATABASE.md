# 数据库设计规范与详细说明

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Active

## 1. 基础规范
- **引擎**: 统一使用 InnoDB。
- **编码**: `utf8mb4_general_ci`。
- **主键**: 统一使用 `bigint unsigned NOT NULL AUTO_INCREMENT`。
- **软删除**: 使用 GORM 默认的 `deleted_at` 字段，类型为 `datetime(3)`，默认值为 `NULL`。

## 2. 命名契约
- **底座表**: `system_` 前缀。
- **业务表**: `biz_` 前缀。
- **中间表**: `_rel_` 或 `_mapping` 风格（如 `system_rel_user_role`）。

## 3. 核心表详细说明

### 3.1 动态多语言存储 (`system_i18n`)
- **领域归属**: 归属 `system/config`，用于平台层与业务域共享的运行时翻译资源。
- **核心字段**:
  - `module`: 来源模块，例如 `system.auth`、`system.config`、`business.cmdb`。
  - `group_name`: 逻辑分组，例如 `menu / permission / messages`；数据库层避免使用保留字 `group`。
  - `key`: i18n key，建议层级命名（如 `auth.login.fail`）。
  - `locale`: 语言代码 (`zh-CN`、`en-US`)。
- **查询优化**: 当前保留 `(module, key)`、`(module, group_name)`、`(locale)` 索引，兼顾运行时查询与治理审计。
- **迁移约束**: 平台运行时标准数据库为 MySQL。SQLite 运行时与历史兼容修复逻辑已移除。

### 3.2 审计日志
- **操作日志 (`system_log_oper`)**:
  - `request_id`: 记录平台层生成或透传的请求标识，用于把接口响应、前端报错和统一审计串联起来。
  - `oper_param`: 存储 JSON 格式的请求体，需在中间件中进行脱敏处理。
  - `cost_time`: 记录 API 执行毫秒数，用于性能审计。
  - `source_domain / source_page / failure_category`: 存储统一审计的派生检索字段，分别用于来源域、来源页面和失败分类筛选；启动迁移会自动回填历史数据，运行期查询不再触发补偿回填。
  - **索引策略**: 建立 `idx_system_log_oper_request_id`、`idx_system_log_oper_source_domain_page`、`idx_system_log_oper_source_page`、`idx_system_log_oper_failure_category`，让请求追踪、来源域/页面/失败分类筛选直接命中数据库索引，而不是在应用层全量过滤。
- **登录日志 (`system_log_login`)**:
  - 必须记录 `request_id`, `ipaddr`, `browser` 和 `login_location`（离线 IP 库解析）。

### 3.3 用户会话 (`system_user_session`)
- **session_id**: 会话主键，同时写入 access/refresh token。
- **refresh_jti**: 当前有效 refresh token 的 JTI。每次刷新都会轮换，旧 refresh token 会失效。
- **revoked_at**: 注销或强退时写入，受保护接口会检查会话是否仍有效。
- **refresh_expires_at**: 会话最长有效期，超过后必须重新登录。

### 3.3.5 用户平台壳层偏好 (`system_user.preference_json`)
- **领域归属**: 存储载体仍挂在 `system_user`，但语义归属 `platform` 壳层偏好，不属于 `system/iam` 个人资料字段。
- **承载内容**: 当前只允许 `theme / language / layoutMode / densityMode` 四项。
- **迁移约束**: 启动迁移会自动把历史 JSON 中的兼容别名（如 `layout / density / lang`）重写为当前规范字段，并清理非法值。
- **默认值关系**: `system/config` 的公开设置继续提供默认主题和默认语言；一旦当前用户保存了显式偏好，运行时优先使用 `preference_json`。

### 3.3.4 生成器受管数据源 (`system_generator_datasource`)
- **领域归属**: `system/config -> generator`。
- **用途**: 为低代码生成器维护外部数据库的只读连接元数据，支持“按数据源选择表并导入字段”。
- **核心字段**:
  - `driver`: 当前第一阶段固定为 `mysql`。
  - `database_name`: 目标 schema 名称。
  - `password_encrypted`: 加密存储的访问密码，不回传前端明文。
  - `readonly_scope`: 当前固定为 `metadata_only`，表示只允许结构探查。
  - `last_checked_at / last_check_status / last_check_error`: 连接测试结果与最近失败原因。
- **边界约束**:
  - 当前平台库作为虚拟默认数据源，不落库；
  - 外部数据源只允许读取 `information_schema.tables` 与 `information_schema.columns`；
  - 不允许在生成器链路内执行任意 SQL 或写外部库。

### 3.3.1 登录来源节流 (`system_login_throttle`)
- **领域归属**: `system/auth`。
- **用途**: 记录同一来源/IP 在滑动时间窗口内的失败次数、最近尝试时间与临时锁定截止时间，用于登录抗喷洒策略。
- **核心字段**:
  - `source_key`: 归一化后的来源标识，当前默认以 `ip:<client_ip>` 形式落库。
  - `failure_count`: 当前统计窗口内的失败次数。
  - `window_started_at`: 当前失败统计窗口起点。
  - `blocked_until`: 命中来源阈值后的临时锁定截止时间。
- **索引策略**: `source_key` 唯一索引、`blocked_until` 辅助索引。

### 3.3.2 用户软删除与唯一键
- **system_user.username**: 仍保持物理唯一约束。
- **软删除复用策略**: 删除用户时，服务层会先把已删除账号的 `username` 归档为内部保留值，再执行软删除；启动迁移也会自动修复历史已删除但仍占用用户名的数据，保证用户名可被重新创建。

### 3.3.3 其他系统域软删唯一键
- **system_role.role_key**、**system_post.post_code**、**system_dict_type.dict_code** 以及 **system_dict_item(dict_code, item_value)** 同样采用“归档唯一标识后再软删除”的策略。
- **目标**: 既保留审计与历史记录，又允许管理员在删除旧记录后重新创建相同标识的新记录。

### 3.4 Casbin 策略表 (`casbin_rule`)
- **来源**: `database/system_init.sql` 会直接建表并写入管理员默认策略；应用启动时也会通过 GORM Adapter 再做一次迁移与同步，保证脚本初始化和运行时行为一致。
- **用途**: 持久化保存 `p` / `g` 策略，避免服务重启后权限丢失。
- **初始化策略**: 系统启动时会确保 `admin` 拥有 `/api/v1/*` 的 `GET/POST/PUT/PATCH/DELETE` 权限。
- **管理入口**: 现已提供系统权限管理页面，直接维护 `casbin_rule` 中的 `p` 策略记录。

### 3.5 组织结构表 (`system_dept`, `system_post`)
- **system_dept**: 使用 `parent_id + ancestors` 维护树形组织结构，祖级路径由服务层自动计算和刷新。
- **组织根节点**: `system_dept.is_root=1` 表示组织根节点，用于承载“公司 / 组织”这一最高层；根节点不允许删除或挂到其他上级之下。
- **system_post**: 通过 `dept_id` 与 `system_dept` 建立部门-岗位一对多逻辑关联，不使用物理外键；岗位必须归属具体部门，不直接挂组织根节点。
- **用户接入**: `system_user.dept_id`、`system_user.post_id` 已用于用户管理页面的组织字段维护；用户选择岗位时必须与所选部门匹配。

## 4. 索引与性能
- **状态字段索引**: `status` 字段由于区分度低，通常不需要独立索引，除非结合其他字段做复合索引。
- **外键约束**: 为了支持后续的模块化拆分，**严禁使用数据库物理外键**。通过程序逻辑（Service 层）确保数据完整性。

## 5. 初始初始化路径
- [DDL 脚本见这里](../../database/system_init.sql)
- `database/system_init.sql` 不再直接写入默认管理员密码账号；首个 `admin` 用户由后端迁移创建。
- 非生产环境如果未设置 `PANTHEON_INITIAL_ADMIN_PASSWORD`，运行时会创建 `admin / 123456` 作为本地开发默认账号。
- 生产环境必须显式设置 `PANTHEON_INITIAL_ADMIN_PASSWORD`，且长度不少于 12 位；不得依赖开发默认密码。
- 运行时 `PANTHEON_DSN` 必须是 MySQL DSN；后端测试也统一通过 MySQL 夹具执行。
