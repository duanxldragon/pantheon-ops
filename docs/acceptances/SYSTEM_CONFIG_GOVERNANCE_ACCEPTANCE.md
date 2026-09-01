---
title: system/config 高敏治理验收基线
doc_type: Acceptance
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-23
---

# system/config 高敏治理验收基线

English version: [SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.en.md](./SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.en.md)

本文固定 `system/config` 自身的高敏/中敏治理验收范围，并标注与低代码工作域的相邻验收基线。

---

## 1. 覆盖范围

| 页面 | 子域 | 风险等级 | 固定验收目标 |
| :--- | :--- | :--- | :--- |
| `/system/i18n` | `system/config/i18n` | 中 | 翻译资产生命周期、导入导出、缓存刷新 |

相邻低代码工作域页面：

| 页面 | 能力归属 | 工作域归属 | 风险等级 | 固定验收目标 |
| :--- | :--- | :--- | :--- | :--- |
| `/system/modules` | `system/dynamicmodule` | `platform.lowcode` | 高 | 动态模块注册、卸载、删除记录、清理 |
| `/system/generator` | `system/generator` | `platform.lowcode` | 高 | 代码生成、数据源治理、工作台注册开关 |

## 2. 通用验收项

- 页面可见权限和动作权限必须分离。
- 后端 Casbin 接口权限必须覆盖真实写操作。
- 写操作失败时必须能区分权限拒绝、环境限制、二次验证失败、业务失败和工具失败。
- 所有展示文案必须使用 i18n key。
- 后端错误优先返回 error key。
- 高敏写操作必须有审计记录。
- 危险动作必须使用二次验证或等价安全校验。

## 3. `/system/i18n`

### 3.1 权限

| 能力 | 页面/动作权限 | 接口权限 |
| :--- | :--- | :--- |
| 查看 | `system:i18n:list` | `GET /api/v1/system/i18n` |
| 新增 | `system:i18n:create` | `POST /api/v1/system/i18n` |
| 编辑 | `system:i18n:update` | `PUT /api/v1/system/i18n/:id` |
| 删除 | `system:i18n:delete` | `DELETE /api/v1/system/i18n/:id` |
| 导入 | `system:i18n:import` | `POST /api/v1/system/i18n/import` |
| 导出 | `system:i18n:export` | `POST /api/v1/system/i18n/export` |
| 刷新缓存 | `system:i18n:refresh` | `POST /api/v1/system/i18n/cache/refresh` |

### 3.2 验收

- 能按 locale、namespace、key、状态筛选。
- 导入失败能返回结构化错误行。
- 导出文件名和内容可解释。
- 缓存刷新有明确成功/失败反馈。
- key 重命名或删除必须说明影响范围。

## 4. `/system/modules`

### 4.1 权限

| 能力 | 页面/动作权限 | 接口权限 |
| :--- | :--- | :--- |
| 查看 | `system:module:list` | `GET /api/v1/system/dynamic-modules` |
| 注册 | `system:module:register` | `POST /api/v1/system/dynamic-modules` |
| 卸载 | `system:module:unregister` | `DELETE /api/v1/system/dynamic-modules/:name` |
| 删除记录 | `system:module:delete_record` | `DELETE /api/v1/system/dynamic-modules/:name/record` |
| 清理 | `system:module:purge` | `DELETE /api/v1/system/dynamic-modules/:name/purge` |

### 4.2 验收

- 页面必须解释“模块注册”与“业务菜单可见”的关系。
- 注册、卸载、删除记录、清理必须走权限、环境限制和二次验证。
- 生成注册表与数据库状态不一致时，必须给出可读状态。
- 失败不能静默吞掉，必须回传 error key。

## 5. `/system/generator`

### 5.1 权限

| 能力 | 页面/动作权限 | 接口权限 |
| :--- | :--- | :--- |
| 查看 | `system:generator:use` | 页面权限 |
| 生成 | `system:module:generate` | `POST /api/v1/system/dynamic-modules/generate` |
| 数据源管理 | `system:generator:datasource:manage` | `POST/PUT/DELETE /api/v1/system/generator/datasources` |

`system:generator:use` 只能作为短期页面可见兼容权限，不代表长期高敏动作权限。

### 5.2 验收

- `includeDashboardWidget=true` 时生成工作台组件注册和路由声明。
- `includeDashboardWidget=false` 时不生成工作台组件注册。
- 未显式传入 `includeDashboardWidget` 时使用生成器默认行为，并有自动化回归覆盖。
- 生成失败时明确区分 schema 校验失败、写文件失败、工具失败和权限失败。

## 6. 固定自动化

推荐固定执行：

- `cd frontend && npm run test:generator:dashboard-widget`
- `cd frontend && npm run test:generator:quality`
- `cd frontend && npm run test:generator:smoke`
- `go test ./backend/modules/system/i18n`
- `go test ./backend/modules/system/dynamicmodule`
- `go test ./backend/modules/system/generator`
- `cd frontend && npm run check:menu-contract`
- `cd frontend && npm run check:i18n-hardcode`

`test:generator:smoke` 固化低代码生成链路的非浏览器验收，当前聚合：

- 工作台快捷入口生成契约；
- 生成物质量契约快照，包括后端 module seed、前端 manifest、数据权限钩子、i18n key 和关系表不导航规则；
- 菜单 / 权限 / 组件注册表一致性；
- 前端展示型硬编码文案扫描。

涉及真实浏览器主链路时补充：

- `cd frontend && npm run test:smoke:business:database-import`

### 6.1 2026-05-23 相邻高敏页面实测状态

本轮针对 `system/config` 相邻低代码高敏链路，按真实 smoke 日志记录如下：

| 套件/链路 | 当前状态 | 说明 | 证据 |
| :--- | :--- | :--- | :--- |
| `platform:full` 覆盖 `/system/modules`、`/system/generator` | 通过 | 稳定后重跑 `57/57` 通过 | `frontend/platform-full-smoke-rerun.log` |
| `system:pages` 中 `/system/dict` 工作台结构 | 已修复并定向通过 | `dict-page__table-card` 工作面恢复，治理摘要留在工作面外 | `frontend/system-pages-smoke.log`、`frontend/system-pages-smoke-remediation-20260523.log` |
| `system:governance` 权限工作台真实后端整改 | 已修复并定向通过 | 首次失败是 UI 文案断言漂移，不是整改链路失效 | `frontend/system-governance-smoke.log`、`frontend/system-governance-smoke-remediation-20260523.log` |
| `business:database-import` | 跳过 | 当前按源表可用性决定是否执行，不应记为失败 | `frontend/business-database-import-smoke.log` |
| `business:auto-recycle` | 已修复并定向通过 | 修正代理默认端口后，按当前真实契约验证“临时模块 purge 文案 + 自动回收结果”通过 | `frontend/business-auto-recycle-smoke.log`、`frontend/business-auto-recycle-smoke-remediation-20260523.log` |

说明：

- `auto-recycle` 首次失败包含错误代理目标 `127.0.0.1:18080`；修复后默认回到本仓库实际后端 `127.0.0.1:8080`。
- `auto-recycle` 现行后端契约在 `/generate` 后先进入 `PendingActivation`，因此浏览器 smoke 不再把“同步建表完成”作为唯一通过条件，而是验证临时模块 purge 提示与自动回收结果。

## 7. 完成定义

只有当 `system/config` 自身页面，以及与其相邻的低代码高敏页面，在权限、接口、二次验证、审计、i18n、错误 key、自动化回归上都能被解释并复现时，当前阶段治理才算闭环。
