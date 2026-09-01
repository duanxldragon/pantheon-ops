---
title: 用户偏好设计
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-05-18
---

# 用户偏好设计

English version: [USER_PREFERENCE_DESIGN.en.md](./USER_PREFERENCE_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_AUTH_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 平台层用户偏好体系的边界。

用户偏好不是“顺手存几个前端 localStorage 字段”，而是平台级体验控制面的一部分。

## 1. 设计目标

用户偏好要解决：

1. 把导航模式、表格密度、工作台布局等平台级体验偏好统一起来
2. 区分“会话临时选择”和“长期用户偏好”
3. 为未来跨设备同步预留统一模型

## 2. 非目标

当前阶段不做：

- 完整个人设置中心重构
- 任意组件都可随意写入偏好
- 无边界地把业务页私有状态都收进平台偏好

## 3. 归属边界

### 3.1 `platform` 负责

- 偏好键定义
- 偏好存储模型
- 平台壳层和工作台对偏好的消费
- 默认值与回退规则

### 3.2 其他域负责

- 提供需要被偏好影响的页面或模块
- 不自行发明另一套持久化规则

## 4. 偏好范围

第一阶段建议只覆盖平台级偏好：

- `navMode`
- `tableDensity`
- `workbenchLayout`
- `defaultLanguage` 的长期偏好位

### 4.1 `navMode`

示例值：

- `side`
- `top`

### 4.2 `tableDensity`

示例值：

- `comfortable`
- `compact`

### 4.3 `workbenchLayout`

示例值：

- `summary-first`
- `task-first`

## 5. 会话覆盖与长期偏好

必须明确两层：

- 会话级选择
- 用户长期偏好

例如语言优先级：

- 本次登录选择
- 用户长期偏好
- 系统默认

不能把一次会话切换自动解释成“永久修改偏好”。

## 6. 存储模型

建议最小模型：

- `userId`
- `preferenceJson`
- `updatedAt`

`preferenceJson` 内推荐按稳定键管理：

- `navMode`
- `tableDensity`
- `workbenchLayout`
- `language`

## 7. 前端约束

- 平台壳层统一消费偏好
- 不允许每个页面自己发明密度键名或导航键名
- 临时 UI 状态不默认进用户偏好

## 8. 后端约束

- 提供读取 / 更新偏好接口
- 对偏好键和值做白名单校验
- 非法值回退默认而不是污染存储

## 9. 审计与安全

用户偏好不是高敏配置，但仍应记录：

- 谁修改了偏好
- 修改了哪些键
- 何时修改

不需要把偏好系统做成高敏治理页，但也不能完全无记录。

## 10. 完成定义

第一阶段完成标准：

- 有统一偏好模型
- 有平台级消费点
- 有会话覆盖与长期偏好区分
- 有基本读写与校验规则

不是以“个人中心很复杂”作为完成标准。
