# Pantheon 平台层冒烟归档报告（2026-04-20）

更新时间：2026-04-20

类型：Acceptance
归属层：platform
状态：Archived

执行者：Pantheon 专家
层级归属：`platform` + `system/auth` + `system/iam` + `system/org` + `system/config`

## 1. 背景

本次验收属于平台层发布前冒烟，不涉及 `business/*`。
目标是从登录页出发，使用 gstack 内置 Chrome 覆盖当前已交付的系统底座页面，并确认此前发现的问题已经完成闭环。

## 2. 执行环境

- 操作系统：Windows
- 前端地址：`http://127.0.0.1:5173`
- 后端地址：`http://127.0.0.1:8080`
- 浏览器方式：gstack 内置 Chrome / `browse.exe`
- 登录账号：`admin / 123456`

## 3. 验收方法

- 先通过 `POST /api/v1/auth/login` 获取 access / refresh token
- 再通过单条 `browse chain` 注入登录态，避免 Windows 下多次调用时上下文漂移
- 对每个页面采集：
  - 最终 URL
  - console error 状态
  - 页面截图
  - 无障碍快照

## 4. 覆盖范围

### 4.1 `platform`

- `/dashboard`

### 4.2 `system/auth`

- `/login`
- `/auth/security`
- `/system/login-log`
- `/system/session`

### 4.3 `system/iam`

- `/system/profile`
- `/system/user`
- `/system/user/1`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/system/operation-log`

### 4.4 `system/org`

- `/system/dept`
- `/system/post`

### 4.5 `system/config`

- `/system/dict`
- `/system/setting`

## 5. 结果摘要

- 本次覆盖的 16 个页面全部 `PASS`
- 登录、导航、页面加载、核心列表页和详情页均可正常访问
- console error 已清零
- 安全中心、会话管理、登录日志、操作日志已完成针对性回归

## 6. 关键修复闭环

### 6.1 `system/auth` / `system/audit`

- 根因不是前端路由，而是本地 SQLite 历史 schema 漂移
- `system_user_session`、`system_log_login`、`system_log_oper` 的时间列曾是 `TEXT`
- 现已在启动迁移中为 `auth` 与 `audit` 子域补齐自修复逻辑，并将相关列修正为 `DATETIME`

### 6.2 前端壳层

- 面包屑切换为 `routes` 模式，移除旧的 key 告警
- React 与 Arco 调整到兼容版本，移除 `element.ref` 开发态告警

## 7. 验收证据

- 最终 JSON 汇总：`.gstack/qa-reports/summary-20260420-final.json`
- 原始逐页输出：`.gstack/qa-reports/raw/20260420-final`
- 最终截图目录：`.gstack/qa-reports/screenshots/20260420-final`

关键截图：

- 登录页：`.gstack/qa-reports/screenshots/20260420-final/login.png`
- 仪表盘：`.gstack/qa-reports/screenshots/20260420-final/dashboard.png`
- 安全中心：`.gstack/qa-reports/screenshots/20260420-final/security-center.png`
- 会话管理：`.gstack/qa-reports/screenshots/20260420-final/session.png`
- 用户管理：`.gstack/qa-reports/screenshots/20260420-final/user-list.png`
- 角色管理：`.gstack/qa-reports/screenshots/20260420-final/role-list.png`
- 系统设置：`.gstack/qa-reports/screenshots/20260420-final/setting-page.png`

## 8. 风险与说明

- Windows 下 gstack 的 `browse.exe` 偶发出现浏览器上下文被关闭、截图超时或回到空白页
- 该现象在本轮中通过“优先单条 `browse chain` + 必要时提权运行”完成规避
- 复核后确认这些现象属于工具运行特性，不构成页面真实失败

## 9. 结论

- 当前平台层与系统底座层已经完成一轮可归档的冒烟验收
- 验收范围内页面全部通过
- 可以将本报告作为后续阶段评审与发布前收口的参考基线
