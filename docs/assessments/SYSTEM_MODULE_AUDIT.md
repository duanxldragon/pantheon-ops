# 系统管理功能审计报告

更新时间：2026-04-17

类型：Assessment
归属层：platform
状态：Active

本文用于对齐”设计文档 / 数据库 / 后端 / 前端”四层现实状态，重点检查以下系统管理功能点是否真正闭环：

- 用户管理
- 角色管理
- 部门管理
- 岗位管理
- 权限管理
- 菜单管理
- 字典管理
- 系统设置
- 个人中心

## 总结

当前真正闭环的系统管理模块只有：

- 用户管理
- 角色管理
- 部门管理
- 岗位管理
- 权限管理
- 菜单管理
- 字典管理

部分能力已存在但未形成独立模块：

- 个人中心：已与安全中心形成自助闭环，支持资料维护、密码修改、登录设备/会话管理与最近登录记录

## 状态矩阵

| 功能点 | 设计/文档 | 数据库 | 后端 | 前端 | 结论 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 用户管理 | 有 | 有 | 有 | 有 | 已闭环 |
| 角色管理 | 有 | 有 | 有 | 有 | 已闭环 |
| 部门管理 | 有 | 有 `system_dept` | 有 | 有 | 已闭环 |
| 岗位管理 | 有 | 有 `system_post` | 有 | 有 | 已闭环 |
| 权限管理 | 有 | 有 `casbin_rule`、角色菜单关系 | 有 | 有 | 已闭环 |
| 菜单管理 | 有 | 有 | 有 | 有 | 已闭环 |
| 字典管理 | 有 | 有 `system_dict_type`、`system_dict_item` | 有 | 有 | 基础闭环 |
| 系统设置 | 有 | 有 `system_setting` | 有 | 有 | 基础闭环 |
| 个人中心 | 有 | 复用 `system_user` 与 `system_user_session/system_log_login` | 有 | 有 | 已闭环 |

## 逐项审查

### 1. 用户管理

**已有证据**

- 后端路由：`backend/modules/system/system.go`
- 前端页面：`frontend/src/modules/system/user/UserList.tsx`、`frontend/src/modules/system/user/UserDetail.tsx`
- 前端模块注册：`frontend/src/modules/system/user/index.ts`
- 文档说明：`docs/designs/BACKEND.md`、`docs/designs/FRONTEND.md`

**结论**

用户管理功能完整度最高，已具备：

- 列表
- 筛选
- 分页
- 排序
- 新增
- 编辑
- 删除
- 详情
- 角色绑定
- 部门绑定
- 岗位绑定
- 批量启用 / 禁用

### 2. 角色管理

**已有证据**

- 后端路由：`backend/modules/system/system.go`
- 前端页面：`frontend/src/modules/system/role/RoleList.tsx`
- 前端模块注册：`frontend/src/modules/system/role/index.ts`

**结论**

角色管理已闭环，具备：

- 列表
- 筛选
- 分页
- 排序
- 新增
- 编辑
- 删除
- 菜单授权
- 批量启用 / 禁用

**缺口**

- 没有权限点维度的独立授权视图
- 角色授权仍然绑定在菜单树，不是“菜单 + 按钮/资源权限”的双层模型

### 3. 部门管理

**已有证据**

- 后端路由：`backend/modules/system/system.go`
- 后端模块：`backend/modules/system/dept/`
- 前端页面：`frontend/src/modules/system/dept/DeptList.tsx`
- 前端模块注册：`frontend/src/modules/system/dept/index.ts`
- 启动补种：`backend/modules/system/seed.go`

**结论**

部门管理已闭环，支持树形读取、新增、编辑、删除、批量启用 / 禁用，并对父子层级、组织根节点和用户占用关系做校验；当前前端已补“组织架构”视图，可在部门树下联动查看岗位与成员归属，并支持在选中部门下直接新增岗位、查看直属成员详情。

### 4. 岗位管理

**已有证据**

- 后端路由：`backend/modules/system/system.go`
- 后端模块：`backend/modules/system/post/`
- 前端页面：`frontend/src/modules/system/post/PostList.tsx`
- 前端模块注册：`frontend/src/modules/system/post/index.ts`
- 启动补种：`backend/modules/system/seed.go`

**结论**

岗位管理已闭环，支持分页读取、新增、编辑、删除、批量启用 / 禁用，并对岗位编码唯一性、所属部门和用户占用关系做校验；岗位已明确归属 `system/org`，不再作为脱离部门的孤立字典。

### 5. 权限管理

**已有证据**

- Casbin 引擎：`backend/pkg/database/casbin.go`
- Casbin 持久化：`backend/pkg/database/casbin_adapter.go`
- 鉴权中间件：`backend/internal/middleware/casbin_middleware.go`
- 角色菜单关系：`system_role_menu`
- 权限后端模块：`backend/modules/system/permission/`
- 权限前端页面：`frontend/src/modules/system/permission/PermissionList.tsx`

**已补齐**

- 权限管理模块
- 路由策略 CRUD
- 策略可视化页面
- 前端按钮细粒度权限点

**结论**

权限管理已经从“只有引擎”升级为可维护的产品能力。当前采用双轨模型：

- 菜单/按钮权限走 `system_menu`
- 接口访问策略走 `casbin_rule`

这比之前的“只有运行时校验”状态完整得多。

### 6. 菜单管理

**已有证据**

- 后端路由：`backend/modules/system/system.go`
- 前端页面：`frontend/src/modules/system/menu/MenuList.tsx`
- 前端模块注册：`frontend/src/modules/system/menu/index.ts`
- 菜单作用域：`scope=nav/manage`

**结论**

菜单管理已闭环，并且和角色授权联动正常。

**已补齐**

- `routeName / module / isCache / isExternal / activeMenu` 元数据
- 图标枚举选择器与统一 icon 映射
- 基础系统菜单回填与老库补种

**仍可增强**

- 菜单和权限点更彻底解耦
- iframe / 标签页缓存等更深一层导航策略
- 更完整的信息架构规划

### 7. 字典管理

**已有证据**

- 数据表：`database/system_init.sql`
- 后端模块：`backend/modules/system/dict/`
- 前端页面：`frontend/src/modules/system/dict/DictPage.tsx`
- 前端模块注册：`frontend/src/modules/system/dict/index.ts`
- 启动补种：`backend/modules/system/seed.go`
- 设计文档：`docs/designs/DICT_AND_SETTING_DESIGN.md`

**结论**

字典管理已完成当前阶段基础闭环，具备：

- 字典类型 CRUD
- 字典项 CRUD
- 字典 options 公共读取接口
- 字典 options 缓存自动失效与手动刷新
- 菜单与权限点注册
- 前端主从维护页

**仍可增强**

- 业务模块字典接入样例
- 字典值被业务引用时的更严格保护策略

### 8. 系统设置

**已有证据**

- 数据表：`system_setting`
- 后端模块：`backend/modules/system/setting/`
- 前端页面：`frontend/src/modules/system/setting/SettingPage.tsx`
- 前端模块注册：`frontend/src/modules/system/setting/index.ts`

**已补齐**

- 公开配置读取
- 管理员配置列表
- 按分组读取配置
- 按分组批量保存配置
- 基础默认配置种子
- 菜单与页面权限注册

**结论**

系统设置已经完成增强后的基础闭环，上传配置分组、敏感配置加密存储、配置变更审计详情均已落地；后续重点转向缓存策略与更细的安全策略。

### 9. 个人中心

**已有证据**

- 当前用户信息接口：`GET /api/v1/system/user/info`
- 登录后会写入用户信息状态

**已补齐**

- 独立页面
- 编辑昵称/邮箱/头像
- 修改密码
- 顶部布局跳转入口

**仍可增强**

- 更细的安全设置项（如二次验证、登录告警）
- 更强的安全策略页（如密码策略、设备信任）

**结论**

个人中心 + 安全中心已经完成当前阶段自助闭环，会话管理不再属于缺口项。

## 设计文档与现实代码的主要偏差

### 偏差 1：系统管理范围比实际实现大得多

从数据库命名和用户预期看，系统管理应至少包含：

- 用户
- 角色
- 部门
- 岗位
- 权限
- 菜单
- 字典
- 设置
- 个人中心

但真正高质量闭环仍需继续完善菜单元数据、配置增强与业务样例接入。

### 偏差 2：权限管理被误认为“已经完成”

现在有 Casbin，不代表“权限管理模块”已经完成。

这个偏差现在已经被修正。当前不仅有权限引擎，还有权限页面、策略 CRUD 和细粒度按钮权限点。

### 偏差 3：数据库预留表容易制造“已经支持”的错觉

`system_dept`、`system_post` 已存在，但这只代表未来想做，不代表现在可用。

这个偏差现在已经被修正。当前代码和页面已经把这两个表真正接入业务流。

### 偏差 4：个人中心没有从用户管理中独立出来

对用户来说，“管理别人”和“管理自己”是两个完全不同的任务。现在只有前者。

### 偏差 5：系统菜单信息架构比目标范围少

SQL 初始化、i18n 资源和前端模块注册，目前已经覆盖：

- 仪表盘
- 用户管理
- 角色管理
- 部门管理
- 岗位管理
- 菜单管理
- 字典管理
- 系统设置

服务启动时还会自动补种部门、岗位菜单到老库，避免“代码有了但升级库后导航没有入口”的尴尬。

### 偏差 6：按钮权限模型还是粗粒度

当前前端页面把“列表权限”直接当作“写权限”使用，这会导致两个问题：

- 设计上无法区分查看、创建、编辑、删除
- 后续就算补了权限管理页面，也没有统一的按钮权限粒度可接入

这个偏差现在也已经被修正。系统页已改为使用 `create/update/delete` 级别的独立权限点。

## 建议的实现优先级

### P0，应该优先补

1. 已完成设计锚点：菜单与权限模型进一步解耦，详见 `docs/designs/NAVIGATION_IA_STRATEGY.md` 与 `docs/designs/PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md`
2. 已完成设计锚点：业务模块字典接入样例，详见 `docs/designs/BUSINESS_DICT_INTEGRATION_GUIDE.md`

原因：

- 安全中心与会话管理已闭环
- 字典与系统设置都已完成基础闭环，系统设置侧重缓存与策略深化，字典侧重业务接入

### P1，第二批补

3. MFA 已实现；登录告警、密码策略、风控详见 `docs/designs/SECURITY_POLICY_ROADMAP.md`

原因：

- 系统设置通常会成为后续所有模块的公共依赖
- 字典管理适合在更多业务枚举出现前定好底座

## 下一步建议

如果按“最小但完整”的系统管理路线推进，推荐顺序是：

1. 已完成设计收口：菜单与权限模型进一步解耦
2. 已完成设计收口：业务模块字典接入样例
3. 下一轮实现：登录告警、密码策略、风控规则

这样做的好处是，系统管理会从“权限、组织、安全中心、基础配置与菜单元数据已闭环”继续升级成“配置中心可运营、权限模型更清晰、业务接入路径更标准”的可运营后台。
