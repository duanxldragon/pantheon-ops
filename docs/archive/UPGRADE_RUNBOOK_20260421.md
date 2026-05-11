# 老环境升级运维 SOP（2026-04-21）

更新时间：2026-04-21

类型：Design
归属层：platform
状态：Archived

本文是给运维/发布执行人的顺序版 SOP，用于将已有 Pantheon 环境升级到 2026-04-21 这轮模块目录与菜单信息架构版本。

详细解释、完整 SQL 和回归项见 `docs/archive/UPGRADE_EXECUTION_CHECKLIST_20260421.md`；本文只保留执行顺序和放行/回滚判断。

---

## 0. 执行边界

本次升级涉及：

- `platform`：dashboard 仍是平台聚合层，但物理目录已顶层化；
- `system/auth`：auth 已物理拆到顶层模块，但逻辑上仍是系统安全域；
- `system/iam`：用户、角色、菜单、权限仍在系统治理域；
- `system/org`：部门、岗位仍在系统治理域；
- `system/config`：字典、设置仍在系统治理域；
- `system/audit`：操作日志仍在系统治理域。

本次升级不涉及：

- 业务域 `business/*` 数据结构调整；
- API 前缀批量改名；
- 旧库全量重建。

---

## 1. 升级前 10 分钟检查

- [ ] 确认当前环境是老环境升级，不是全新初始化。
- [ ] 记录当前后端制品版本 / 镜像 tag / commit。
- [ ] 记录当前前端制品版本 / 镜像 tag / commit。
- [ ] 完成数据库备份，并确认备份文件可访问。
- [ ] 保留上一版后端制品或镜像。
- [ ] 保留上一版前端静态资源或镜像。
- [ ] 确认 `PANTHEON_DSN` 指向目标数据库。
- [ ] 确认 Redis 配置仍可用，如启用则检查 `PANTHEON_REDIS_ADDR`。
- [ ] 确认敏感配置密钥未变化，如启用则检查 `PANTHEON_SETTING_SECRET`。
- [ ] 明确回滚负责人和回滚窗口。

红线：

- [ ] 老环境不要重新执行 `database/system_init.sql`。
- [ ] 不要手工删除旧菜单数据。
- [ ] 不要手工改 API 路径。

---

## 2. 发布前制品验证

在发布机或 CI 环境确认：

```bash
go test ./...
```

```bash
cd frontend && npm run build
```

放行条件：

- [ ] 后端测试通过。
- [ ] 前端构建通过。
- [ ] 新后端制品已生成。
- [ ] 新前端静态资源或镜像已生成。

不满足以上任意一项时，不进入正式升级。

---

## 3. 正式升级步骤

### 3.1 停旧服务

- [ ] 停止旧版后端实例。
- [ ] 确认没有新的后端请求继续进入旧实例。
- [ ] 如有多实例，按发布策略逐台替换或进入维护窗口。

### 3.2 升级后端

- [ ] 替换为新版本后端制品或镜像。
- [ ] 保持原有环境变量不变。
- [ ] 启动新版本后端。
- [ ] 观察启动日志。

后端启动时预期自动完成：

- [ ] 自动补齐 `system_menu.page_perm`。
- [ ] 自动创建/迁移 `system_role_permission`。
- [ ] 自动补种 `dashboard / access / org / config / security` 一级目录。
- [ ] 自动把历史平铺菜单重挂到新一级目录下。
- [ ] 自动更新登录日志、会话管理菜单的组件路径。
- [ ] 自动给管理员角色绑定新一级目录菜单。

后端阻断条件：

- [ ] 数据库连接失败。
- [ ] 自动迁移失败。
- [ ] Casbin 初始化失败。
- [ ] 服务启动后登录接口不可用。

出现任意阻断条件，进入第 8 节回滚。

### 3.3 升级前端

- [ ] 替换为新版本前端静态资源或镜像。
- [ ] 发布前端。
- [ ] 清理 CDN / Nginx / 网关缓存，如适用。
- [ ] 使用无缓存窗口或浏览器强刷验证。

---

## 4. 数据库升级后核查

### 4.1 查一级菜单

```sql
SELECT id, title_key, path, module, sort
FROM system_menu
WHERE parent_id = 0
ORDER BY sort ASC, id ASC;
```

必须包含：

- [ ] `system.menu.dashboard`
- [ ] `system.menu.access`
- [ ] `system.menu.org`
- [ ] `system.menu.config`
- [ ] `system.menu.security`

### 4.2 查历史菜单重挂

```sql
SELECT c.path AS child_path, p.path AS parent_path, c.module
FROM system_menu c
LEFT JOIN system_menu p ON p.id = c.parent_id
WHERE c.path IN (
  '/system/user',
  '/system/role',
  '/system/permission',
  '/system/menu',
  '/system/dept',
  '/system/post',
  '/system/dict',
  '/system/setting',
  '/system/login-log',
  '/system/session',
  '/system/operation-log'
)
ORDER BY c.path;
```

必须满足：

- [ ] `/system/user`、`/system/role`、`/system/permission`、`/system/menu` 的父级是 `/system/access`。
- [ ] `/system/dept`、`/system/post` 的父级是 `/system/org`。
- [ ] `/system/dict`、`/system/setting` 的父级是 `/system/config`。
- [ ] `/system/login-log`、`/system/session`、`/system/operation-log` 的父级是 `/system/security`。

### 4.3 查菜单元数据

```sql
SELECT path, component, page_perm, module
FROM system_menu
WHERE path IN ('/dashboard', '/system/login-log', '/system/session')
ORDER BY path;
```

必须满足：

- [ ] `/dashboard` 的 `module` 是 `platform`。
- [ ] `/system/login-log` 的 `component` 是 `auth/LoginLogList`。
- [ ] `/system/session` 的 `component` 是 `auth/SessionList`。
- [ ] `/system/login-log`、`/system/session` 的 `module` 是 `system.auth`。

---

## 5. 最小页面回归

按顺序执行：

- [ ] 打开 `/login`，确认可以登录。
- [ ] 打开 `/dashboard`，确认指标卡和最近登录活动可加载。
- [ ] 检查左侧一级菜单为 `工作台 / 访问控制 / 组织架构 / 平台配置 / 安全审计`。
- [ ] 打开 `/auth/security`，确认安全概览可加载。
- [ ] 打开 `/system/login-log`，确认登录日志列表可加载。
- [ ] 打开 `/system/session`，确认会话列表可加载。
- [ ] 打开 `/system/user`，确认用户列表可加载。
- [ ] 打开 `/system/role`，确认角色列表可加载。
- [ ] 打开 `/system/menu`，确认菜单树可加载且层级正确。
- [ ] 打开 `/system/dept`，确认部门树可加载。
- [ ] 打开 `/system/dict`，确认字典页可加载。
- [ ] 打开 `/system/setting`，确认设置页可加载。
- [ ] 打开 `/system/operation-log`，确认操作日志可加载。
- [ ] 检查右上角用户菜单，确认 `个人中心 / 安全中心 / 退出登录` 可用。

---

## 6. 最小 API 回归

使用登录后的 token 验证：

- [ ] `POST /api/v1/auth/login`
- [ ] `POST /api/v1/auth/refresh`
- [ ] `GET /api/v1/auth/me`
- [ ] `GET /api/v1/auth/security`
- [ ] `GET /api/v1/auth/sessions`
- [ ] `GET /api/v1/platform/dashboard/summary`
- [ ] `GET /api/v1/system/menu/tree?scope=nav`
- [ ] `GET /api/v1/system/menu/tree?scope=manage`
- [ ] `GET /api/v1/system/user/list`
- [ ] `GET /api/v1/system/setting/group/basic`

兼容接口至少验证：

- [ ] `POST /api/v1/system/login`
- [ ] `GET /api/v1/system/user/info`
- [ ] `POST /api/v1/system/logout`

---

## 7. 放行标准

全部满足后可宣布升级完成：

- [ ] 后端启动成功且日志无迁移错误。
- [ ] 前端资源发布成功。
- [ ] 数据库一级菜单补齐。
- [ ] 历史菜单重挂正确。
- [ ] 菜单组件路径更新正确。
- [ ] 左侧菜单分组正确。
- [ ] 右上角用户菜单可用。
- [ ] 最小页面回归通过。
- [ ] 最小 API 回归通过。
- [ ] 旧兼容 API 仍可用。

---

## 8. 回滚步骤

出现阻断条件且无法在发布窗口内定位时：

- [ ] 停止新版本后端。
- [ ] 回滚上一版后端制品或镜像。
- [ ] 回滚上一版前端静态资源或镜像。
- [ ] 如数据库数据异常，使用升级前备份恢复。
- [ ] 清理前端缓存或 CDN 缓存。
- [ ] 验证 `/login` 可登录。
- [ ] 验证 `/dashboard` 可打开。
- [ ] 验证 `/system/user` 可打开。
- [ ] 记录失败日志、数据库核查结果和回滚时间。

回滚后不要直接再次执行升级；先根据失败点回到代码或数据层定位根因。
