# 菜单 Icon 审计与治理结果

更新时间：2026-04-28

类型：Assessment
归属层：platform
状态：Archived

## 1. 归属判断

本次治理属于 `platform` 层导航规范整改，目标是统一菜单图标语义、消除重复感，并收口前后端菜单 icon key。

不涉及：

- `system/auth`、`system/iam`、`system/org`、`system/config` 的业务逻辑改动
- 权限模型、菜单数据结构、路由装配方式调整

## 2. 根因结论

本次菜单栏图标重复主要由两类问题叠加导致：

1. 平台层共享 icon registry 过小，只提供了 `dashboard/user/safe/menu/settings/list/apps/storage` 8 个 key。
2. 前端模块 manifest 与后端菜单 seed 对 `language`、`code` 等 key 已有使用，但共享映射未补齐，导致实际渲染退回默认 `menu` 图标。

这意味着当前问题不只是“个别菜单重复”，而是 `platform` 层缺少一份完整、统一、可约束的菜单图标语义表。

## 3. 审计表

| 层级 | 菜单 | 原图标 | 问题 | 治理后 |
| :--- | :--- | :--- | :--- | :--- |
| platform | 工作台 | `dashboard` | 正常 | `dashboard` |
| system/auth | 登录日志 | `safe` | 与安全中心/会话/审计混用 | `clock` |
| system/auth | 会话管理 | `safe` | 与安全中心/日志混用 | `desktop` |
| system/iam | 用户 | `user` | 正常 | `user` |
| system/iam | 角色 | `safe` | 与权限/审计/安全混用 | `user-group` |
| system/iam | 权限 | `safe` | 与角色/审计/安全混用 | `lock` |
| system/iam | 菜单 | `menu` | 正常 | `menu` |
| system/org | 部门 | `storage` | 与岗位/CMDB 复用，语义弱 | `branch` |
| system/org | 岗位 | `storage` | 与部门/CMDB 复用，语义弱 | `tags` |
| system/config | 字典 | `list` | 正常 | `list` |
| system/config | 系统设置 | `settings` | 正常 | `settings` |
| system/config | 多语言 | `language` | registry 缺失，实际退回默认图标 | `language` |
| system/config | 模块中心 | `apps` | 正常 | `apps` |
| system/config | 代码生成器 | `code` | registry 缺失，实际退回默认图标 | `code` |
| system/audit | 操作日志 | `safe` | 与安全/角色/权限混用 | `file` |
| business/cmdb | CMDB 总览 | `storage` | 作为业务父域可接受 | `storage` |
| business/cmdb | 主机管理 | `storage` | 与 CMDB 根节点完全同图标 | `desktop` |
| business/cmdb | 供应商管理 | `apps` | 现阶段可接受 | `apps` |

## 4. 平台层最佳实践

### 4.1 图标分配原则

1. 一级域优先唯一。
2. 二级菜单允许少量复用，但必须保留明显语义差异。
3. `安全` 图标只给安全域核心入口，不再兜底承担角色、权限、审计、日志。
4. 业务域父菜单可以用一个总类 icon，子菜单再按实体类型细分。

### 4.2 注册约束

1. 前端 `ModuleConfig` 的 `route.icon` 与 `menu.icon` 必须只使用平台注册过的 `MenuIconKey`。
2. 后端 `system_menu.icon` 种子值必须与前端 registry 同名，不允许自由字符串漂移。
3. 菜单管理页图标选择器只能从共享枚举中选，不允许手写自然语言或未注册 key。

### 4.3 语义建议

- `dashboard`: 聚合页/工作台
- `safe`: 安全中心/安全域总入口
- `clock`: 登录日志、时间序列日志
- `desktop`: 会话、终端、主机
- `user`: 用户
- `user-group`: 角色、成员分组
- `lock`: 权限、授权策略
- `menu`: 菜单、导航元数据
- `branch`: 部门、组织树
- `tags`: 岗位、标签化实体
- `file`: 审计记录、操作流水
- `language`: 多语言
- `code`: 代码生成、开发工具
- `apps`: 模块中心、应用集合
- `storage`: 资产、仓储、资源总览

## 5. 已落地的治理动作

1. 扩充平台层共享 icon registry，并补齐 `language`、`code` 等真实使用 key。
2. 将前端模块 manifest 的 `icon` 类型收紧为 `MenuIconKey`。
3. 统一修改前端菜单 manifest 与后端菜单 seed，消除系统域重复和 fallback。
4. 保留 `renderMenuIcon()` 的兼容分支，避免存量脏数据在渲染时直接失效。

## 6. 后续约束

后续新增菜单时，必须同步检查：

1. 是否属于 `platform`、`system/*`、`business/*` 的正确边界。
2. icon 是否已存在于共享 registry。
3. 前端 manifest、后端 seed、菜单管理页选项是否一致。
4. 是否引入新的“一级域同图标堆叠”问题。
