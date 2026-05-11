# 模块物理目录升级说明（2026-04-21）

更新时间：2026-04-21

类型：Design
归属层：platform
状态：Archived

本文用于说明 2026-04-21 这一轮”逻辑层不变、物理目录优化”的升级。

如需在老环境执行实际升级，请配合 `docs/archive/UPGRADE_RUNBOOK_20260421.md` 与 `docs/archive/UPGRADE_EXECUTION_CHECKLIST_20260421.md` 一起使用；本文只负责解释边界与目录变化，不替代执行清单。

## 1. 目标

这次升级解决两个问题：

- `auth` 作为独立安全域，继续放在 `system/*` 物理目录下不够清晰；
- `platform/dashboard` 只有一个 dashboard 子模块，单独保留 `platform` 目录显得冗余。

## 2. 升级原则

- **逻辑层不变**：`platform` 仍然是聚合层，`system/auth` 仍然是认证安全子域语义；
- **物理目录优化**：目录结构更扁平，但不改变业务边界；
- **接口路径尽量不变**：避免因为目录调整引发额外 API 兼容成本。

## 3. 目录变化

### 3.1 后端

```text
旧：
backend/modules/platform/dashboard/
backend/modules/system/auth/

新：
backend/modules/dashboard/
backend/modules/auth/
```

### 3.2 前端

```text
旧：
frontend/src/modules/platform/dashboard/
frontend/src/modules/system/auth/

新：
frontend/src/modules/dashboard/
frontend/src/modules/auth/
```

## 4. 装配变化

- `dashboard` 不再通过 `platform.InitPlatformModule()` 装配，而是由独立 `dashboard.InitDashboardModule()` 装配；
- `auth` 不再通过 `system.InitSystemModule()` 装配，而是由独立 `auth.InitAuthModule()` 装配；
- `system/*` 继续承载 `iam / org / config / audit / i18n` 等系统治理域。

## 5. 不变项

- dashboard 逻辑归属仍是 `platform` 层；
- dashboard API 仍使用 `/api/v1/platform/dashboard/summary`；
- auth API 仍使用 `/api/v1/auth/*` 为主、`/api/v1/system/*` 为兼容路径；
- 左侧导航的信息架构仍按 `访问控制 / 组织架构 / 平台配置 / 安全审计` 分组。

## 6. 对老库的影响

### 6.1 菜单

服务启动时会自动补种和重挂以下目录菜单：

- `dashboard`
- `access`
- `org`
- `config`
- `security`

并把已有平铺菜单自动归到新的一级目录下。

### 6.2 菜单组件路径

初始化 SQL 与菜单补种已同步更新：

- 登录日志组件路径改为 `auth/LoginLogList`
- 会话管理组件路径改为 `auth/SessionList`

说明：

- 当前前端实际路由注册不依赖数据库里的 `component` 字段直接懒加载；
- 但菜单元数据仍应与当前物理目录保持一致，避免后续维护误导。

## 7. 推荐认知模型

建议后续统一按下面的方式理解项目：

- `platform`：逻辑层
- `auth`：独立顶层物理模块，逻辑上仍属于系统安全域
- `dashboard`：独立顶层物理模块，逻辑上仍属于 platform 聚合层
- `system/*`：系统治理域子模块
- `business/*`：业务域模块

## 8. 验证建议

升级后至少验证：

- `/login`
- `/dashboard`
- `/auth/security`
- `/system/login-log`
- `/system/session`
- `/system/user`
- `/system/setting`

同时执行：

```bash
go test ./...
cd frontend && npm run build
```

若是数据库已在线运行的老环境，执行顺序、菜单重挂核查、页面/API 回归项请直接使用：

- `docs/archive/UPGRADE_RUNBOOK_20260421.md`
- `docs/archive/UPGRADE_EXECUTION_CHECKLIST_20260421.md`
