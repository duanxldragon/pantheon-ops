---
title: business/* 业务模块验收矩阵
doc_type: Acceptance
layer: business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-23
---

# business/* 业务模块验收矩阵

English version: [BUSINESS_MODULE_ACCEPTANCE_MATRIX.en.md](./BUSINESS_MODULE_ACCEPTANCE_MATRIX.en.md)

本文把业务模块验收从“参考模板”升级为固定矩阵，适用于独立业务仓库中的所有 `business/*` 模块。

---

## 1. 九维验收矩阵

| 维度 | 必答问题 | 验收证据 |
| :--- | :--- | :--- |
| 边界 | 是否明确属于 `business/*`，并说明不负责哪些 system 能力 | 业务设计文档 |
| 数据 | DDL、索引、唯一键、审计字段、租户就绪判断是否完整 | DDL / migration / 设计文档 |
| 接口 | API 前缀、请求、响应、错误 key、权限点是否完整 | 后端测试 / API 清单 |
| 页面 | loading / empty / error / forbidden / submitting 是否覆盖 | 页面实现 / smoke 记录 |
| 菜单 | 菜单、路由、组件键、titleKey 是否一致 | `check:menu-contract` |
| 权限 | 页面权限、按钮权限、接口权限是否分层 | 权限 seed / Casbin / smoke |
| i18n | locale key 是否完整且无硬编码 | `check:i18n-hardcode` / locale audit |
| 审计 | 新增、编辑、删除、导入、导出等关键动作是否审计 | 操作日志记录 |
| 回归 | 是否有固定自动化或手工验收记录 | smoke / 测试报告 |

## 2. 业务模块准入

新业务模块进入实现前必须具备：

- 业务模块设计文档。
- 数据模型与租户就绪判断。
- API 与权限清单。
- 菜单与组件键清单。
- i18n namespace 与关键文案清单。
- 字典/配置依赖清单。
- 审计点清单。

## 3. 业务模块上线

业务模块进入“当前已完成”前必须通过：

- 后端模块测试。
- 前端构建。
- 菜单契约检查。
- i18n 硬编码检查。
- 至少一条主链路 smoke。
- 权限账号验证，覆盖有权限和无权限两类场景。

## 4. 独立业务仓库验收重点

- 业务仓库必须声明业务域设计文档、模块清单和 smoke 命令。
- 平台仓库只固定通用验收矩阵，不内置具体业务模块验收项。
- 业务仓库从平台底座升级时，必须重新执行菜单契约、权限、i18n、构建和主链路 smoke。

## 5. 固定命令

推荐固定执行：

- `go test ./backend/modules/business/...`
- `go test ./backend/modules/system/iam/permission`
- `cd frontend && npm run check:menu-contract`
- `cd frontend && npm run check:i18n-hardcode`
- `cd frontend && npm run build`

如果业务模块提供真实浏览器 smoke，必须在本矩阵或对应业务设计文档中记录命令与证据路径。

### 5.1 2026-05-23 真实浏览器 smoke 快照

| 套件 | 当前状态 | 说明 | 证据 |
| :--- | :--- | :--- | :--- |
| `business:generated` | 通过 | 生成/注册基础链路通过 | `frontend/business-generated-smoke.log` |
| `business:database-import` | 跳过 | 当前依赖源表存在性，日志应按 skip 解释，不记失败 | `frontend/business-database-import-smoke.log` |
| `business:master-detail` | 断言通过，teardown 挂起 | 用例主体通过，命令停在 Playwright/webServer 退出阶段 | `frontend/business-master-detail-smoke.log` |
| `business:many-to-many` | 断言通过，teardown 挂起 | 用例主体通过，命令停在 Playwright/webServer 退出阶段 | `frontend/business-many-to-many-smoke.log` |
| `business:auto-recycle` | 已修复并定向通过 | 首次失败含错误代理端口与前置契约误判；2026-05-23 定向复测通过 | `frontend/business-auto-recycle-smoke.log`、`frontend/business-auto-recycle-smoke-remediation-20260523.log` |

本矩阵后续维护要求：

- “断言通过但 teardown 挂起”必须单列记录，不能直接记成失败产品项。
- 若真实浏览器 smoke 依赖外部前置数据，必须在用例内自建前置或在文档中明确 `skip` 条件，避免把环境残留误记为模块失败。
