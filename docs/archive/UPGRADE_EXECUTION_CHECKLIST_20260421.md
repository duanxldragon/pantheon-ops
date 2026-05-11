# 数据库 / 老环境升级执行 Checklist（2026-04-21）

更新时间：2026-04-21

类型：Design
归属层：platform
状态：Archived

本文用于指导已经运行中的 Pantheon 环境完成 2026-04-21 这一轮升级。

如果你需要发布窗口内“按顺序直接执行”的版本，请先看 `docs/archive/UPGRADE_RUNBOOK_20260421.md`；本文更偏完整核查与验收清单。

适用范围：

- 已存在历史数据库的数据环境
- 已使用旧版菜单信息架构的环境
- 已从旧目录结构部署过前后端代码的环境

本次升级的核心不是业务逻辑重写，而是：

- `auth` 物理拆到顶层 `modules/auth`
- `dashboard` 物理拆到顶层 `modules/dashboard`
- 左侧菜单重组为 `访问控制 / 组织架构 / 平台配置 / 安全审计`
- 启动时自动补种并重挂历史菜单
- 保持主要 API 路径兼容，不要求业务方同步改接口

---

## 1. 升级前必须确认

### 1.1 环境与回滚准备

- [ ] 记录当前部署版本、提交号或制品版本
- [ ] 备份生产数据库
- [ ] 保留上一版后端可执行文件 / 镜像
- [ ] 保留上一版前端静态资源 / 镜像
- [ ] 确认出现问题时可快速回滚

### 1.2 配置准备

- [ ] 确认 `PANTHEON_DSN` 正确
- [ ] 若启用了 Redis，确认 `PANTHEON_REDIS_ADDR` 与密码配置正确
- [ ] 若使用系统设置敏感字段加密，确认 `PANTHEON_SETTING_SECRET` 正确
- [ ] 确认新版本前后端制品已经构建完成

### 1.3 执行原则

- [ ] **老环境不要重新执行 `database/system_init.sql` 全量初始化**
- [ ] 老环境应通过**新版本服务启动时自动迁移**完成结构补齐
- [ ] 若是全新环境，才使用 `database/system_init.sql`

---

## 2. 本次升级后应保持不变的内容

以下接口和页面路径应继续可用：

### 2.1 不变的 API

- [ ] `POST /api/v1/auth/login`
- [ ] `POST /api/v1/auth/refresh`
- [ ] `POST /api/v1/auth/logout`
- [ ] `GET /api/v1/auth/me`
- [ ] `GET /api/v1/auth/security`
- [ ] `PUT /api/v1/auth/password`
- [ ] `GET /api/v1/auth/sessions`
- [ ] `DELETE /api/v1/auth/sessions/:id`
- [ ] `GET /api/v1/auth/login-logs`
- [ ] `GET /api/v1/platform/dashboard/summary`

### 2.2 兼容保留的旧 API

- [ ] `POST /api/v1/system/login`
- [ ] `POST /api/v1/system/refresh`
- [ ] `POST /api/v1/system/logout`
- [ ] `GET /api/v1/system/user/info`
- [ ] `PUT /api/v1/system/profile/password`
- [ ] `GET /api/v1/system/login-log/list`
- [ ] `POST /api/v1/system/login-log/export`
- [ ] `GET /api/v1/system/session/list`
- [ ] `DELETE /api/v1/system/session/:id`

### 2.3 不变的页面路由

- [ ] `/login`
- [ ] `/dashboard`
- [ ] `/auth/security`
- [ ] `/system/login-log`
- [ ] `/system/session`
- [ ] `/system/profile`
- [ ] `/system/user`
- [ ] `/system/role`
- [ ] `/system/menu`
- [ ] `/system/permission`
- [ ] `/system/dept`
- [ ] `/system/post`
- [ ] `/system/dict`
- [ ] `/system/setting`
- [ ] `/system/operation-log`

---

## 3. 启动后应自动发生的变化

新版本后端启动后，预期会自动完成以下事情：

### 3.1 数据结构自动补齐

- [ ] `system_menu` 表存在 `page_perm` 字段
- [ ] `system_role_permission` 表存在
- [ ] MySQL-only 运行时与测试夹具已完成切换

### 3.2 菜单目录自动补种

应自动存在以下一级目录菜单：

- [ ] `dashboard`
- [ ] `access`
- [ ] `org`
- [ ] `config`
- [ ] `security`
- [ ] 管理员角色自动绑定以上新一级目录菜单

### 3.3 历史平铺菜单自动重挂

以下页面菜单应自动归入新的一级目录，而不是继续平铺在根级：

- [ ] `/system/user` → `访问控制`
- [ ] `/system/role` → `访问控制`
- [ ] `/system/permission` → `访问控制`
- [ ] `/system/menu` → `访问控制`
- [ ] `/system/dept` → `组织架构`
- [ ] `/system/post` → `组织架构`
- [ ] `/system/dict` → `平台配置`
- [ ] `/system/setting` → `平台配置`
- [ ] `/system/login-log` → `安全审计`
- [ ] `/system/session` → `安全审计`
- [ ] `/system/operation-log` → `安全审计`

### 3.4 菜单元数据自动更新

- [ ] 登录日志菜单组件路径应为 `auth/LoginLogList`
- [ ] 会话管理菜单组件路径应为 `auth/SessionList`
- [ ] `dashboard` 菜单模块归属应保持为 `platform`
- [ ] `login-log / session` 菜单模块归属应为 `system.auth`

说明：

- 当前前端真实页面装配不依赖数据库 `component` 字段直接懒加载
- 但菜单元数据仍必须与现状一致，避免后续管理端误导

---

## 4. 推荐升级执行步骤

### 4.1 后端升级

- [ ] 停止旧版后端实例
- [ ] 替换为新版本后端制品
- [ ] 启动后端服务
- [ ] 观察启动日志，确认未出现迁移失败、数据库连接失败、Casbin 初始化失败

### 4.2 前端升级

- [ ] 替换为新版本前端静态资源 / 镜像
- [ ] 发布前端
- [ ] 清理 CDN / 反向代理缓存（如适用）
- [ ] 浏览器强刷一次或清理缓存后验证

### 4.3 启动后立即做的验证

- [ ] 后端健康检查通过
- [ ] 登录接口可用
- [ ] `/dashboard` 可打开
- [ ] 左侧主导航分组正确
- [ ] 右上角用户菜单中 `个人中心 / 安全中心 / 退出登录` 正常

---

## 5. 数据库快速验证 SQL

以下 SQL 用于老环境升级后的快速核查。

### 5.1 检查一级目录菜单

```sql
SELECT id, title_key, path, module, sort
FROM system_menu
WHERE parent_id = 0
ORDER BY sort ASC, id ASC;
```

期望至少看到：

- `system.menu.dashboard`
- `system.menu.access`
- `system.menu.org`
- `system.menu.config`
- `system.menu.security`

### 5.2 检查关键菜单是否已重挂

```sql
SELECT c.title_key AS child_title, c.path AS child_path, p.title_key AS parent_title
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

### 5.3 检查 `page_perm` 字段是否已存在并回填

```sql
SELECT path, page_perm, perms, module
FROM system_menu
WHERE path IN (
  '/dashboard',
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
ORDER BY path;
```

### 5.4 检查 `system_role_permission`

```sql
SELECT COUNT(*) AS total FROM system_role_permission;
```

说明：

- 升级后该表存在即可
- 是否有完整数据，还与角色保存、菜单种子同步和历史权限补齐有关

---

## 6. 页面回归清单

下面这组页面建议作为本轮最小回归集。

### 6.1 `platform`

- [ ] `/dashboard`
  - [ ] 指标卡正常显示
  - [ ] 快捷入口可跳转
  - [ ] 最近登录活动正常显示

### 6.2 `auth`

- [ ] `/login`
  - [ ] 登录成功
  - [ ] 语言切换正常
- [ ] `/auth/security`
  - [ ] 安全概览可加载
  - [ ] 修改密码可提交
- [ ] `/system/login-log`
  - [ ] 列表加载正常
  - [ ] 导出按钮按权限显示
- [ ] `/system/session`
  - [ ] 列表加载正常
  - [ ] 下线操作按权限显示

### 6.3 `system/iam`

- [ ] `/system/profile`
- [ ] `/system/user`
- [ ] `/system/user/1`
- [ ] `/system/role`
- [ ] `/system/menu`
- [ ] `/system/permission`

### 6.4 `system/org`

- [ ] `/system/dept`
- [ ] `/system/post`

### 6.5 `system/config`

- [ ] `/system/dict`
- [ ] `/system/setting`

### 6.6 `system/audit`

- [ ] `/system/operation-log`

---

## 7. API 回归清单

建议至少验证下面这几类：

### 7.1 登录与自助链路

- [ ] `POST /api/v1/auth/login`
- [ ] `POST /api/v1/auth/refresh`
- [ ] `GET /api/v1/auth/me`
- [ ] `GET /api/v1/auth/security`
- [ ] `GET /api/v1/auth/sessions`

### 7.2 平台聚合链路

- [ ] `GET /api/v1/platform/dashboard/summary`

### 7.3 系统治理链路

- [ ] `GET /api/v1/system/menu/tree?scope=nav`
- [ ] `GET /api/v1/system/menu/tree?scope=manage`
- [ ] `GET /api/v1/system/user/list`
- [ ] `GET /api/v1/system/setting/group/basic`

### 7.4 兼容链路

- [ ] `POST /api/v1/system/login`
- [ ] `GET /api/v1/system/user/info`
- [ ] `POST /api/v1/system/logout`

---

## 8. 推荐执行命令

### 8.1 后端测试

```bash
go test ./...
```

### 8.2 前端构建

```bash
cd frontend && npm run build
```

### 8.3 平台层最小冒烟

如果本地启动了前后端，至少验证：

```text
/login
/dashboard
/auth/security
/system/login-log
/system/session
/system/user
/system/setting
```

---

## 9. 升级成功判定

满足以下条件，可判定本轮升级完成：

- [ ] 后端启动成功
- [ ] 前端构建成功
- [ ] 数据结构自动迁移成功
- [ ] 左侧新一级菜单分组已生效
- [ ] 历史平铺菜单已自动重挂
- [ ] `auth` 自助接口与旧兼容接口均可用
- [ ] dashboard 页面正常读取 `/api/v1/platform/dashboard/summary`
- [ ] 最小回归页面全部通过

---

## 10. 回滚原则

若升级失败：

- [ ] 停止新版本服务
- [ ] 回滚前后端制品到上一版
- [ ] 如数据库结构或数据已异常变化，使用升级前备份恢复
- [ ] 恢复后重新验证 `/login`、`/dashboard`、`/system/user`

不建议在未确认问题根因前反复重跑初始化 SQL。
