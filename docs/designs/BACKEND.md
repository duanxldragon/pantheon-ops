# 后端架构设计与开发规范

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Active

## 1. 架构核心：模块化单体 (Modular Monolith)
底座将系统拆分为物理隔离的模块，业务模块与底座通过 `pkg/common` 定义的接口契约进行“逻辑通信”。

## 2. 功能开发模式：垂直切片 (Vertical Slices)
每个功能包（如 `user`）必须自包含所有层级逻辑，严禁在 `modules/` 目录下进行“水平切包”（如把所有 service 放在一个 service 文件夹）。

### 2.1 垂直切片文件构成 (以 user 为例)
- `user_model.go`: 数据库物理映射 (GORM)。
- `user_dto.go`: 前后端通信契约 (Req/Resp)。
- `user_repository.go`: 数据持久化封装 (DAO)。
- `user_service.go`: 核心业务逻辑、权限计算、事务。
- `user_handler.go`: HTTP 路由逻辑、参数绑定、Response 组装。

## 3. 分层职责
- **Model**: 只对齐 DB，不带逻辑。
- **DTO**: 必须对敏感数据进行屏蔽（如不返回 Password）。
- **Service**: 唯一的业务流程控制中心，支持事务注入。
- **Handler**: 调用 Service，通过 `common.Success` 或 `common.Fail` 返回标准响应。

## 4. 关键底座能力
- **鉴权**: JWT access + refresh token 机制，Access Token 15 分钟有效，Refresh Token 7 天有效。
- **会话**: `system_user_session` 保存 refresh token 当前 JTI，刷新时轮换 JTI，注销时吊销会话；同时记录 `last_activity_at`，用于空闲超时判定。
- **权限**: Access Token 携带 `userId`、`username`、`roleKeys`、`sessionId`；Casbin 按 RESTful 路由匹配，`admin` 角色默认拥有 `/api/v1/*` 权限。
- **Casbin 持久化**: `database/system_init.sql` 会预建 `casbin_rule`，启动时再通过本地 GORM Adapter 自动迁移并同步策略，避免“只初始化 SQL 但没启动服务时表不存在”的落差。
- **审计**: `OperationLogMiddleware` 异步写入 `system_log_oper`，对 `password`、`token` 等敏感键递归脱敏，并持久化 `source_domain / source_page / failure_category` 三个派生字段；派生字段补偿仅发生在迁移/启动阶段，查询热路径直接按落库字段下推 SQL 过滤，配合组合/单列索引避免统一审计页退化为全表扫描；业务模块仍可通过上下文覆盖 `title / businessType / param / result`，让配置审计等场景复用统一操作日志底座。
- **平台偏好审计**: `PUT /api/v1/auth/me/preferences` 会把当前登录主体的平台壳层偏好变更写入统一审计，记录变更前/变更后快照，但不混入密码、token 等敏感值。
- **请求链路标识**: 平台壳层会为每个请求生成或透传 `X-Request-ID`，并同步回写 `X-Trace-ID` 响应头；统一审计会把 `request_id` 一并落库，用于串联前端报错、接口响应与操作日志。
- **多语言**: `/api/v1/system/i18n/pack` 返回语言包，优先查询 Redis，缓存失效后读取 `system_i18n`。
- **健康检查**: 平台层提供 `GET /api/v1/health`，通过统一 `common.Response` envelope 返回进程状态、数据库连通性与 Redis 状态（未启用时标记为 `disabled`）；依赖降级时保留 HTTP `503` 供部署探针识别。
- **平台聚合**: `platform` 层用于承载 dashboard/workbench 这类跨 `system/*` 子域的聚合接口，不应反向塞回单一系统子域；当前 `dashboard` 已物理放置在 `backend/modules/dashboard/`，但逻辑归属仍是 `platform`。
- **数据库运行时**: 后端运行时数据库已收敛为 MySQL；`PANTHEON_DSN` 必须是合法 MySQL DSN。SQLite 运行时代码与历史兼容修复已移除，测试统一通过 MySQL 夹具执行。

## 5. 已实现系统接口
当前认证接口已进入“**新旧路径并存**”阶段：

- 新路径：`/api/v1/auth/*`
- 兼容旧路径：`/api/v1/system/login`、`/api/v1/system/refresh`、`/api/v1/system/logout`、`/api/v1/system/user/info`、`/api/v1/system/profile/password`

| 方法 | 路径 | 鉴权 | 说明 |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/v1/health` | 否 | 平台层健康检查接口，返回进程、数据库与 Redis 依赖状态。 |
| `POST` | `/api/v1/auth/login` | 否 | 登录，新 auth 域主入口；在 `login.mfa_enabled=true` 时，密码校验成功后可能先返回 `mfaRequired=true` 与 challenge，而不是直接签发会话。 |
| `POST` | `/api/v1/auth/mfa/verify` | 否 | 校验登录阶段的 TOTP 二次验证 challenge；成功后签发登录会话。 |
| `POST` | `/api/v1/auth/refresh` | 否 | 刷新 token，新 auth 域主入口。 |
| `POST` | `/api/v1/auth/logout` | 是 | 注销当前用户会话，新 auth 域主入口。 |
| `POST` | `/api/v1/auth/activity` | 是 | 更新当前会话最近活动时间，供空闲超时与锁屏场景复用。 |
| `GET` | `/api/v1/auth/me` | 是 | 获取当前登录主体信息。 |
| `PUT` | `/api/v1/auth/me/preferences` | 是 | 更新当前登录主体的平台壳层偏好，仅承载 `theme / language / layoutMode / densityMode`，不混入 `system/iam` 个人资料字段。 |
| `GET` | `/api/v1/auth/security` | 是 | 获取当前登录账号安全概览，包含当前设备、活跃会话数、最近成功登录时间，以及运行时认证策略快照（密码长度、账号失败阈值、来源失败阈值、锁定时长、空闲超时、可配置安全特性开关）。 |
| `PUT` | `/api/v1/auth/password` | 是 | 校验旧密码后修改当前登录用户密码，并吊销其他设备会话。 |
| `GET` | `/api/v1/auth/sessions` | 是 | 获取当前登录账号会话列表，并标识当前会话。 |
| `DELETE` | `/api/v1/auth/sessions/:id` | 是 | 下线当前账号的其他会话；当前会话禁止通过该接口直接下线。 |
| `GET` | `/api/v1/auth/login-logs` | 是 | 获取当前登录账号最近登录日志。 |
| `GET` | `/api/v1/platform/dashboard/summary` | 是 | 平台层仪表盘汇总接口，返回用户、菜单、会话与最近登录活动概览。 |
| `POST` | `/api/v1/system/login` | 否 | 登录，返回 Token 与脱敏用户信息。 |
| `POST` | `/api/v1/system/refresh` | 否 | 使用 refresh token 轮换并返回新的 token pair。 |
| `GET` | `/api/v1/system/i18n/pack?lang=zh-CN` | 否 | 获取启动语言包。 |
| `POST` | `/api/v1/system/logout` | 是 | 兼容旧路径，吊销当前用户会话。 |
| `GET` | `/api/v1/system/user/info` | 是 | 兼容旧路径，获取当前用户信息、角色与权限。 |
| `GET` | `/api/v1/system/profile` | 是 | 获取个人中心资料，包含昵称、邮箱、手机号、角色与创建时间。 |
| `PUT` | `/api/v1/system/profile` | 是 | 修改当前登录用户的昵称、头像、邮箱、手机号。 |
| `PUT` | `/api/v1/system/profile/password` | 是 | 兼容旧路径，校验旧密码后修改当前登录用户密码，并吊销其他设备会话。 |
| `GET` | `/api/v1/system/user/list` | 是 | 获取用户分页列表 DTO，默认按 `id desc` 排序。 |
| `GET` | `/api/v1/system/user/list?username=&nickname=&deptId=&postId=&status=&page=&pageSize=&sortField=&sortOrder=` | 是 | 支持用户名、昵称、部门、岗位、状态筛选，以及分页与排序。 |
| `GET` | `/api/v1/system/user/import-template` | 是 | 下载用户导入 CSV 模板，表头为 `username,password,nickname,email,phone,deptPath,postCode,status,roleKeys`。 |
| `GET` | `/api/v1/system/user/:id` | 是 | 获取用户详情 DTO，返回头像、组织归属、角色集合以及创建/更新时间。 |
| `POST` | `/api/v1/system/user` | 是 | 创建用户，密码 bcrypt 加密，并绑定显式选择的角色集合。 |
| `POST` | `/api/v1/system/user/export` | 是 | 按当前筛选条件导出用户 CSV；使用 `POST` 以纳入统一操作审计。 |
| `POST` | `/api/v1/system/user/import` | 是 | 导入用户 CSV，按用户名做增量导入；单行校验失败时返回导入摘要与错误行。 |
| `POST` | `/api/v1/system/user/batch-status` | 是 | 批量启用/禁用用户；保护内置 `id=1` 管理员账号不可被批量禁用。 |
| `PUT` | `/api/v1/system/user/:id` | 是 | 更新用户资料、状态、角色；不再混入管理员重置密码语义。 |
| `PUT` | `/api/v1/system/user/:id/reset-password` | 是 | 管理员重置指定用户密码，并强制吊销该用户当前所有活跃会话。 |
| `DELETE` | `/api/v1/system/user/:id` | 是 | 删除用户并清理角色绑定与会话；内置保护 `id=1`。 |
| `GET` | `/api/v1/system/role/list` | 是 | 获取角色分页列表，并返回角色关联的菜单 ID 集合。 |
| `GET` | `/api/v1/system/role/list?roleName=&roleKey=&status=&page=&pageSize=&sortField=&sortOrder=` | 是 | 支持角色名称、角色标识、状态筛选，以及分页与排序。 |
| `POST` | `/api/v1/system/role` | 是 | 创建角色，并维护角色与菜单的绑定关系。 |
| `POST` | `/api/v1/system/role/export` | 是 | 按当前筛选条件导出角色基础信息 CSV，不包含菜单与权限绑定明细。 |
| `POST` | `/api/v1/system/role/batch-status` | 是 | 批量启用/禁用角色；保护内置 `admin` 角色不可被批量禁用。 |
| `PUT` | `/api/v1/system/role/:id` | 是 | 更新角色基本信息、状态与菜单权限；保护内置 `admin` 角色。 |
| `DELETE` | `/api/v1/system/role/:id` | 是 | 删除无用户绑定的角色，并清理导航、页面/按钮权限与 Casbin 接口策略关系。 |
| `GET` | `/api/v1/system/menu/tree` | 是 | 获取动态菜单树；默认按 `scope=nav` 返回当前用户可见导航菜单树，并附带 `icon / routeName / module / isCache / isExternal / activeMenu` 元数据。该模式属于登录后壳层自助接口，只要求已登录。 |
| `GET` | `/api/v1/system/menu/tree?titleKey=&path=&isVisible=&sortField=&sortOrder=&scope=manage` | 是 | 返回菜单管理页所需的全量菜单树，支持标题键、路径、显示状态筛选，以及树节点服务端排序。该模式仍属于 `system/iam` 管理接口，必须具备 `system:menu:list`。 |
| `POST` | `/api/v1/system/menu` | 是 | 创建菜单，并默认绑定 admin 角色；支持维护 `routeName / module / isCache / isExternal / activeMenu`。 |
| `PUT` | `/api/v1/system/menu/:id` | 是 | 更新菜单属性；校验路径唯一、父节点合法，并校验路由名称唯一。 |
| `DELETE` | `/api/v1/system/menu/:id` | 是 | 删除无子节点菜单并清理角色绑定。 |
| `GET` | `/api/v1/system/dept/overview` | 是 | 获取 `system/org` 组织健康总览，返回部门、岗位、缺负责人部门、无岗位部门、空部门等治理摘要。 |
| `GET` | `/api/v1/system/dept/governance/tasks` | 是 | 获取 `system/org` 组织治理任务清单，把缺负责人、无岗位、删除阻断、岗位在用等问题收敛成可执行任务，供前端治理工作台直接定位整改入口。 |
| `GET` | `/api/v1/system/dept/:id/leader-candidates` | 是 | 获取部门负责人候选人，仅返回当前部门内“已启用且已挂岗位”的真实成员，供部门治理编辑链路绑定负责人。 |
| `GET` | `/api/v1/system/dept/tree` | 是 | 获取部门树，支持按部门名称、状态、治理视角（`leaderless` / `no-post` / `empty`）筛选及树节点排序；同时返回子部门数、岗位数及治理标记，筛选命中子部门时会自动补齐祖先路径。 |
| `GET` | `/api/v1/system/dept/import-template` | 是 | 下载部门导入 CSV 模板，表头为 `parentDeptPath,deptName,sort,leader,phone,email,status`。 |
| `POST` | `/api/v1/system/dept` | 是 | 创建部门，自动计算祖级路径；新部门必须挂在组织根节点或已有部门下。 |
| `POST` | `/api/v1/system/dept/export` | 是 | 按当前筛选条件导出部门树为 CSV；支持复用治理视角（`leaderless` / `no-post` / `empty`）筛选，并按统一治理报表模板输出 `deptPath / childDeptCount / postCount / userCount / governanceScope / governanceTags / governanceProblemCount / governanceBlockedBy / governanceActions` 及对应 `*Label` 字典列，兼顾系统对接与人工整改阅读。 |
| `POST` | `/api/v1/system/dept/governance/export` | 是 | 导出 `system/org` 治理任务报表，输出任务键、治理范围、问题标签、阻断原因、建议动作及对应 `*Label` 字典列，并附带导出快照时间与查询摘要。 |
| `POST` | `/api/v1/system/dept/import` | 是 | 导入部门 CSV，按 `parentDeptPath + deptName` 做增量导入。 |
| `POST` | `/api/v1/system/dept/batch-status` | 是 | 批量启用/禁用部门；组织根节点状态固定为启用。 |
| `POST` | `/api/v1/system/dept/batch-leader` | 是 | 批量治理部门负责人，但提交体已收敛为“部门 -> 负责人候选人”绑定项；每个部门都必须选择本部门内已启用且已挂岗位的真实成员，不再接受自由文本批量写入。 |
| `PUT` | `/api/v1/system/dept/:id` | 是 | 更新部门，校验父子层级并同步子节点祖级路径；组织根节点允许改名，但不允许改挂载关系。 |
| `DELETE` | `/api/v1/system/dept/:id` | 是 | 删除无子部门、且未被用户引用的部门；组织根节点不可删除。 |
| `GET` | `/api/v1/system/post/list` | 是 | 获取岗位分页列表，支持所属部门、岗位编码、岗位名称、状态筛选及排序。 |
| `GET` | `/api/v1/system/post/import-template` | 是 | 下载岗位导入 CSV 模板，表头为 `deptPath,postCode,postName,sort,status,remark`。 |
| `POST` | `/api/v1/system/post` | 是 | 创建岗位，校验岗位编码唯一，并要求岗位归属到具体部门。 |
| `POST` | `/api/v1/system/post/export` | 是 | 按当前筛选条件导出岗位 CSV，并按统一治理报表模板输出 `assignedUserCount / governanceScope / governanceTags / governanceProblemCount / governanceBlockedBy / governanceActions` 及对应 `*Label` 字典列，兼顾系统对接与人工整改阅读。 |
| `POST` | `/api/v1/system/post/import` | 是 | 导入岗位 CSV，按岗位编码做增量导入，并通过 `deptPath` 维护所属部门。 |
| `POST` | `/api/v1/system/post/batch-status` | 是 | 批量启用/禁用岗位。 |
| `PUT` | `/api/v1/system/post/:id` | 是 | 更新岗位，校验岗位编码唯一，并维护所属部门。 |
| `DELETE` | `/api/v1/system/post/:id` | 是 | 删除未被用户引用的岗位。 |
| `GET` | `/api/v1/system/permission/list` | 是 | 获取 Casbin 路由策略分页列表，支持按角色、路径、方法筛选。 |
| `GET` | `/api/v1/system/permission/workbench` | 是 | 获取 IAM 权限工作台视图，统一返回角色的导航授权、页面/按钮权限、Casbin 接口策略，以及未知权限/页面缺口/API 缺口治理标记。 |
| `GET` | `/api/v1/system/permission/workbench/export` | 是 | 导出 IAM 权限工作台治理报表，按当前 `roleKey / status / integrity / coverage` 条件输出盘点 CSV。 |
| `GET` | `/api/v1/system/permission/import-template` | 是 | 下载权限策略导入 CSV 模板，表头为 `roleKey,path,method`。 |
| `POST` | `/api/v1/system/permission` | 是 | 创建 Casbin 路由策略，校验角色存在及策略唯一。 |
| `POST` | `/api/v1/system/permission/export` | 是 | 按当前筛选条件导出 Casbin 路由策略 CSV。 |
| `POST` | `/api/v1/system/permission/import` | 是 | 导入 Casbin 路由策略 CSV，按 `(roleKey,path,method)` 做增量导入。 |
| `PUT` | `/api/v1/system/permission/:id` | 是 | 更新 Casbin 路由策略，并实时重载 Enforcer。 |
| `DELETE` | `/api/v1/system/permission/:id` | 是 | 删除 Casbin 路由策略，并实时重载 Enforcer。 |
| `GET` | `/api/v1/system/login-log/list` | 是 | 获取管理员登录日志分页列表，支持用户名、状态筛选。 |
| `POST` | `/api/v1/system/login-log/export` | 是 | 按当前筛选条件导出管理员登录日志 CSV。 |
| `POST` | `/api/v1/system/login-log/cleanup` | 是 | 按保留期清理管理员登录日志；当前只允许保留最近 `1 / 7 / 30` 天，并要求二次验证。 |
| `POST` | `/api/v1/system/login-log/batch-delete` | 是 | 按选择集批量删除管理员登录日志，并要求二次验证。 |
| `GET` | `/api/v1/system/session/list` | 是 | 获取管理员全局会话分页列表，支持按用户名筛选。 |
| `DELETE` | `/api/v1/system/session/:id` | 是 | 管理员强制下线指定会话；当前会话禁止通过该接口直接下线。 |
| `GET` | `/api/v1/system/dict/options?codes=a,b` | 否 | 批量获取字典选项，返回已启用字典项，供系统页与业务模块下拉/标签复用。 |
| `GET` | `/api/v1/system/dict/type/list` | 是 | 获取字典类型列表，支持按 `dictCode`、`dictName`、`status` 筛选。 |
| `GET` | `/api/v1/system/dict/type/import-template` | 是 | 下载字典类型导入 CSV 模板，表头为 `dictCode,dictName,module,status,remark`。 |
| `POST` | `/api/v1/system/dict/type/export` | 是 | 按当前筛选条件导出字典类型 CSV。 |
| `POST` | `/api/v1/system/dict/type/import` | 是 | 导入字典类型 CSV，按 `dictCode` 做增量导入。 |
| `POST` | `/api/v1/system/dict/cache/refresh` | 是 | 刷新字典 options 缓存；支持按 `codes` 精准刷新指定字典，未传时清空全部缓存。 |
| `POST` | `/api/v1/system/dict/type` | 是 | 创建字典类型，校验 `dict_code` 唯一。 |
| `PUT` | `/api/v1/system/dict/type/:id` | 是 | 更新字典类型；若修改 `dict_code`，会同步变更所属字典项的 `dict_code`。 |
| `DELETE` | `/api/v1/system/dict/type/:id` | 是 | 删除无字典项引用的字典类型。 |
| `GET` | `/api/v1/system/dict/item/list` | 是 | 按 `dictCode` 获取字典项列表，默认按 `sort asc, id asc` 排序。 |
| `GET` | `/api/v1/system/dict/item/import-template` | 是 | 下载字典项导入 CSV 模板，表头为 `dictCode,itemLabelKey,itemValue,itemColor,sort,status,remark`。 |
| `POST` | `/api/v1/system/dict/item/export` | 是 | 按字典编码导出字典项 CSV。 |
| `POST` | `/api/v1/system/dict/item/import` | 是 | 导入字典项 CSV，按 `(dictCode,itemValue)` 做增量导入。 |
| `POST` | `/api/v1/system/dict/item` | 是 | 创建字典项，校验所属字典存在，且 `dict_code + item_value` 唯一。 |
| `PUT` | `/api/v1/system/dict/item/:id` | 是 | 更新字典项，校验所属字典存在及字典值唯一。 |
| `DELETE` | `/api/v1/system/dict/item/:id` | 是 | 删除字典项。 |
| `GET` | `/api/v1/system/setting/public` | 否 | 获取允许公开读取的系统设置，例如站点名称、默认语言、默认主题；`is_encrypted=1` 的敏感配置即使误配公开也不会被返回。 |
| `POST` | `/api/v1/system/upload?scope=profile/avatar` | 是（仅登录） | 统一上传入口，当前接入 `system/config` 的上传配置并支持本地存储；供头像等通用文件上传场景复用。 |
| `GET` | `/api/v1/system/upload/files/*filepath` | 否 | 本地上传文件访问入口；路径解析会受 `upload.local_path` 约束并阻止目录穿越。 |
| `GET` | `/api/v1/system/setting/overview` | 是 | 获取 `system/config` 配置健康总览，返回公开配置、敏感配置、缺失必填项与运行时风险摘要。 |
| `GET` | `/api/v1/system/setting/list` | 是 | 获取系统设置列表，支持按 `groupKey`、`module` 筛选；敏感配置不返回明文，仅返回 `hasValue` 状态。 |
| `GET` | `/api/v1/system/setting/group/:groupKey` | 是 | 按分组获取系统设置，当前已支持 `basic / security / login / audit / upload / i18n / ui`。 |
| `GET` | `/api/v1/system/setting/audit/list` | 是 | 获取系统设置分组变更审计，基于 `system_log_oper` 读取最近配置修改记录，支持按 `groupKey / settingKey / operName` 分页筛选。 |
| `POST` | `/api/v1/system/setting/audit/export` | 是 | 按当前 `groupKey / settingKey / operName` 条件导出系统设置变更审计 CSV。 |
| `POST` | `/api/v1/system/setting/cache/refresh` | 是 | 刷新系统设置缓存；支持按 `groupKeys` 精准预热分组缓存，未传时清空全部缓存。 |
| `PUT` | `/api/v1/system/setting/group/:groupKey` | 是 | 按分组批量保存系统设置，并校验 number/boolean/json 值类型；敏感配置留空表示保持当前已加密值不变。 |
| `GET` | `/api/v1/system/dynamic-modules` | 是 | 获取动态模块注册表；当前属于 `system/config` 高敏治理视图，受环境守卫控制。 |
| `GET` | `/api/v1/system/dynamic-modules/:name` | 是 | 获取单模块注册状态，区分已接入、待激活与已卸载。 |
| `POST` | `/api/v1/system/dynamic-modules` | 是 | 重新接入已生成但已卸载的模块；要求源码目录与 schema 仍存在，并进入待激活状态。 |
| `POST` | `/api/v1/system/dynamic-modules/generate` | 是 | 一键生成并注册业务模块源码、schema 与 generated registry；当前只支持 `business/*`，成功后进入待激活状态。 |
| `POST` | `/api/v1/system/dynamic-modules/repair` | 是 | 执行动态模块注册表自检/修复：按当前有效源码重写 generated registries，并把缺失源码的接入记录自动标记为已卸载。 |
| `DELETE` | `/api/v1/system/dynamic-modules/:name` | 是 | 卸载动态模块并重写 generated registry；默认保留工作区源码，仅解除接入状态。 |
| `POST` | `/api/v1/system/operation-log/export` | 是 | 按当前筛选条件导出操作日志 CSV。 |
| `POST` | `/api/v1/system/operation-log/cleanup` | 是 | 按保留期清理操作日志；当前只允许保留最近 `1 / 7 / 30` 天，并要求二次验证。 |
| `POST` | `/api/v1/system/operation-log/batch-delete` | 是 | 按选择集批量删除操作日志，并要求二次验证。 |

## 7. 输入校验补充
- **用户**: 创建时校验用户名唯一、邮箱格式合法、角色至少选择一项且角色存在；更新时禁止停用内置管理员或移除其管理员角色；管理员重置密码接口单独校验密码长度不少于 6 位。
- **用户删除与复用**: `system_user` 采用软删除，但删除时会先把用户名归档为内部保留值，再写入 `deleted_at`；这样既保留审计痕迹，也允许后续重新创建同名用户。
- **软删唯一键复用**: 对 `system_user.username`、`system_role.role_key`、`system_post.post_code`、`system_dict_type.dict_code`、`system_dict_item(dict_code,item_value)` 这类“软删除 + 唯一键”实体，删除时统一先归档唯一标识，再执行软删除；启动迁移也会自动修复历史遗留数据。
- **用户组织字段**: `dept_id`、`post_id` 允许为空；若选择岗位，则必须先选择部门，且岗位必须属于该部门。
- **平台能力开关**: `system/config` 公开下发 `platform.app_mode`、`org.enabled`、`org.required_for_user`。`platform.app_mode` 支持 `enterprise / consumer / hybrid`；关闭 `org.enabled` 只隐藏组织导航和用户组织字段，不删除 `system/org` 数据，也不改变已有用户记录。
- **用户扩展档案**: `system_user` 保持瘦核心表；C 端或混合模式下的非稳定档案字段统一进入 `system_user_profile_ext.profile_json`，通过 `profileExt` DTO 受控暴露。`preference_json` 仅用于平台壳层偏好，不承载业务档案。
- **角色**: 创建/更新时校验 `role_key` 唯一、导航菜单 ID 只能引用 `M/C` 菜单节点、权限 key 必须来自菜单元数据中的 `page_perm/perms`；删除时若仍绑定用户则拒绝。
- **菜单**: 创建/更新时校验路径唯一、父节点存在，且父节点不能指向自己；菜单类型为 `C` 时要求 `route_name` 非空且唯一；非外链菜单要求 `component` 非空；外链菜单要求 `path` 为合法 `http(s)` 地址。
- **部门**: 创建/更新时校验上级部门存在、不能把自己挂到自己或子孙节点下；删除时若存在子部门或被用户引用则拒绝。
- **部门负责人收敛**: `system/org` 已新增 `leader_user_id` 渐进约束；编辑部门时如传入 `leaderUserId`，后端会强制校验该用户属于当前部门、已启用且已挂具体岗位，并自动把展示名回填到 `leader` 文本列；创建部门阶段暂不允许直接绑定真实负责人，避免出现“部门未落库但负责人已跨域挂接”的脏状态。
- **组织治理阻断**: `system/org` 删除部门时必须先清空下级部门、岗位与成员；岗位仍被成员占用时，不允许直接禁用或删除，避免组织树与岗位挂接进入半失真状态。
- **组织根节点**: `dept` 模块启动迁移会确保存在唯一组织根节点，并把历史顶级部门自动收敛到根节点下，避免出现多个顶层组织入口。
- **岗位**: 创建/更新时校验所属部门存在且不能是组织根节点、岗位编码唯一；删除时若被用户引用则拒绝。
- **权限策略**: 管理 `casbin_rule` 中的 `p` 策略，要求 `role_key` 存在，且 `(role_key, path, method)` 唯一；角色保存和迁移会同步系统已知权限点对应的 API 策略，权限页仍可维护额外手工策略。
- **字典类型**: 创建/更新时校验 `dict_code` 非空且唯一；删除时若仍存在字典项则拒绝。
- **字典项**: 创建/更新时校验 `dict_code` 存在，且同一字典内 `item_value` 唯一。
- **字典缓存**: `GET /system/dict/options` 已接入进程内 options 缓存；字典类型/字典项变更后自动失效对应 `dict_code`，后台也支持手动刷新缓存。
- **菜单与权限解耦**: `system_role_menu` 只承载导航授权；页面/按钮权限改由 `system_role_permission` 保存，菜单元数据新增 `page_perm` 与原有 `perms` 分别承载页面权限和动作权限。
- **系统设置**: 更新时按 `value_type` 校验值类型；`is_encrypted=1` 的敏感配置会以应用主密钥加密后入库，管理接口不回显明文，公开接口强制忽略敏感配置。
- **组织可选化**: 企业后台默认启用组织能力；面向 C 端应用时可切到 `consumer` 并关闭 `org.enabled`。`system/iam` 用户 CRUD 不强制依赖 `system/org`，未来业务域只能通过 `pkg/capability` 或上下文能力判断消费开关，不允许直接依赖组织 Service。
- **C 端用户兼容**: 不把性别、生日、会员等级、营销来源、实名认证状态等 C 端字段直接铺到 `system_user`；先使用 `system_user_profile_ext` 承载，只有成为高频筛选/排序字段后才评估提取为独立列或业务域档案表。
- **单租户先行、租户就绪**: 当前平台继续按单租户运行，不引入真实租户模型；但新增 `biz_*` 表、唯一键、列表查询、导出和统计接口时，必须先判断未来是否需要 `tenant_id`、是否应采用“租户内唯一”，以及是否需要统一 tenant 过滤注入点。详见 `TENANT_READY_SINGLE_TENANT_DESIGN.md`。
- **上传能力**: 当前统一上传能力物理归于平台公共包、配置源归于 `system/config`；运行时已消费 `upload.max_file_size / upload.allowed_types / upload.local_path / upload.public_base_url / upload.s3_*`。当前支持 `local` 与 `s3-compatible` 驱动；本地驱动走平台文件访问路由，S3 驱动直接返回对象访问 URL。
- **系统设置缓存**: `GET /system/setting/list`、`GET /system/setting/group/:groupKey`、`GET /system/setting/public` 已接入进程内缓存；配置更新后自动失效，后台支持手动刷新与预热。
- **系统设置审计**: 配置变更不单独新建审计表，而是复用 `system_log_oper`；更新分组时会写入结构化变更 payload，并对敏感配置只记录“已变更”状态，不记录明文前后值。
- **日志治理约束**: 登录日志属于 `system/auth` 安全证据，操作日志属于 `system/audit` 统一审计流水；两者都不再默认开放“全表清空”，而是统一收敛为“按 `system/config -> audit` 设置中允许的保留期清理”或“按选择集删除”，并强制走二次验证链路。
- **敏感配置主密钥**: 当前通过环境变量 `PANTHEON_SETTING_SECRET` 提供；开发环境允许回退到内置 dev key，生产环境必须显式配置独立密钥。

## 7.1 系统域导入导出约定
- **边界选择**: 本轮在 `system/iam` 中开放“用户、权限策略导入导出 + 角色基础信息导出/批量状态”；在 `system/org` 开放“部门、岗位导入导出”；在 `system/config` 开放“字典类型、字典项导入导出”；在 `system/auth`、`system/audit` 开放日志导出；`role` 仍**不开放导入**，`menu/setting` 暂不开放导入导出。
- **格式选择**: 当前统一使用 CSV，而非 XLSX；优先保证后端零额外依赖、前后端协议稳定、便于模板维护。
- **审计约束**: 导出接口统一使用 `POST /export`，而不是 `GET`，这样可以进入统一操作审计并复用筛选查询 DTO。
- **模板约束**: 模板下载统一使用 `GET /import-template`；模板内包含表头、`#` 开头的说明行和可复制示例行，导入器会自动忽略这些注释/示例行。
- **结果返回**: 导入接口即使存在行级校验失败，也返回 `200 + ImportResultResp`，由前端展示“成功/失败/错误行”摘要，避免请求层直接吞掉结构化错误详情。
- **联调参考**: 接口级手工验证步骤与示例 CSV 请见 `docs/acceptances/SYSTEM_IMPORT_EXPORT_SMOKE_GUIDE.md`。

## 8. 用户列表分页约定
- **分页参数**: `page` 默认 `1`，`pageSize` 默认 `10`，最大 `100`。
- **组织筛选**: 支持可选 `deptId`、`postId` 精确筛选，供 `system/org` 组织架构视图与 `system/iam` 用户页共用。
- **排序参数**: `sortField` 白名单为 `id`、`username`、`nickname`、`email`、`phone`、`status`、`createdAt`；`sortOrder` 仅接受 `asc` / `desc`。
- **返回结构**: `items`、`total`、`page`、`pageSize`，便于前端表格直接驱动分页器。

## 9. 菜单树排序约定
- **默认排序**: 菜单树默认按 `sort asc, id asc` 构建，保证 sibling 节点稳定。
- **排序参数**: `sortField` 白名单为 `id`、`titleKey`、`path`、`routeName`、`pagePerm`、`perms`、`type`、`module`、`sort`、`isCache`、`isExternal`、`isVisible`；`sortOrder` 仅接受 `asc` / `desc`。
- **返回结构**: 仍返回树形 `children` 结构，排序仅影响同层级节点顺序。

## 11. 权限与菜单可见性约定
- **多角色判定**: Casbin 中间件会遍历 `roleKeys` 集合，只要任一角色命中策略即放行。
- **自助接口白名单**: `/system/logout`、`/system/user/info`、`/system/profile*` 以及 `/auth/logout`、`/auth/me`、`/auth/security`、`/auth/password`、`/auth/sessions*`、`/auth/login-logs` 允许所有已登录用户访问，不依赖角色菜单授权，避免普通用户无法维护自己的账号资料。
- **导航菜单**: `scope=nav` 只返回当前用户已授权、`is_visible=1`、且 `type <> 'F'` 的菜单；为保证树结构完整，会自动补齐其祖先节点。该接口归入平台壳层自助能力，避免普通登录用户在壳层初始化阶段被 `permission.denied` 阻断。

### 5.1 登录阶段 MFA 约束

- `POST /api/v1/auth/login` 在 `login.mfa_enabled=false` 时保持原账号密码登录链路。
- `POST /api/v1/auth/login` 在 `login.mfa_enabled=true` 时，密码验证成功后返回 MFA challenge，当前不直接签发 Access / Refresh Token。
- 已绑定因子的用户，前端应继续调用 `POST /api/v1/auth/mfa/verify` 提交 `challengeId + 6 位动态码`。
- 未绑定因子的用户，后端返回 `totpSecret` 与 `totpProvisionUri`，供前端渲染二维码和手动密钥。
- challenge 当前有效期为 5 分钟，只允许消费一次。
- 动态码由认证器本地根据 TOTP secret 和当前时间生成，后端不直接生成或回显当前验证码。
- **管理菜单**: `scope=manage` 返回完整菜单树，供菜单管理页与角色授权页使用；该接口继续保留在 `system/iam` 管理边界内，要求 `system:menu:list`。
- **管理员兜底**: 新增菜单后自动绑定 `admin` 角色，避免系统管理入口丢失。
- **系统菜单补种**: 服务启动时会自动补齐 `dashboard / access / org / config / security` 目录菜单，以及 `user / role / permission / menu / dept / post / dict / setting / login-log / session / operation-log` 等基础系统菜单，并回填菜单元数据，同时绑定到 `admin` 角色，避免老库升级后没有导航入口。
- **策略热加载**: 权限策略新增、编辑、删除后会立即触发 Casbin `LoadPolicy()`，无需重启服务。
- **系统设置页联调约束**: `/system/setting` 属于典型“双轨权限”页面。仅授予前端 `system:setting:list` 还不够，角色还需要 Casbin 放行至少 `GET /api/v1/system/setting/list` 与 `GET /api/v1/system/menu/tree`；否则会出现“路由可进入但主体数据加载失败”的假象。若角色还要操作设置缓存或查看统一审计，也需要分别补齐对应接口策略。

## 10. 角色列表分页约定
- **分页参数**: `page` 默认 `1`，`pageSize` 默认 `10`，最大 `100`。
- **排序参数**: `sortField` 白名单为 `id`、`roleName`、`roleKey`、`sort`、`status`、`createdAt`；`sortOrder` 仅接受 `asc` / `desc`。
- **返回结构**: `items`、`total`、`page`、`pageSize`，其中每个角色额外携带 `menuIds` 供前端角色授权表单回填。

## 6. 解耦准则
- **业务开发**: 只能在 `modules/business/` 下创建新包。
- **Context 注入**: 模块间禁止直接 import 对方 service。身份信息（UserID, Role）必须从 `gin.Context` 中获取。
- **模块装配**: 后端已新增 `pkg/contracts.BackendModule` 契约，系统模块通过 `RegisterBackendModules` 统一执行 migrate、seed 与路由注册；后续新增模块必须显式声明 `Name / Migrate / RegisterRoutes / SeedMenus / SeedPerms / SeedI18n`。
