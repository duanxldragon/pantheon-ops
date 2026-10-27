---
title: 任务中心设计
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# 任务中心设计

English version: [TASK_CENTER_DESIGN.en.md](./TASK_CENTER_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 标准后台产品位中的“任务中心”最小骨架。

目标不是马上做完整 workflow 引擎，而是先形成一个真实、可跳转、可聚合、可验收的任务承载位。

## 1. 设计目标

任务中心要解决：

1. 平台层有统一的“待我处理 / 我发起 / 已完成”任务入口
2. 任务可以从不同系统域或业务域接入，但呈现模型一致
3. dashboard 与通知中心可引用同一套任务摘要

## 2. 非目标

当前阶段不做：

- 通用 BPM 引擎
- 任意流程建模器
- 复杂审批编排
- 任务评论系统

## 3. 归属边界

### 3.1 `platform` 负责

- 任务中心入口
- 统一任务列表模型
- 任务摘要聚合
- 跳转契约

### 3.2 任务源域负责

- 任务业务语义
- 任务完成逻辑
- 详情页或处理页实现

## 4. 任务模型

建议最小字段：

- `id`
- `taskType`
- `sourceDomain`
- `title`
- `summary`
- `status`
- `assigneeId`
- `creatorId`
- `priority`
- `dueAt`
- `jumpTarget`
- `createdAt`
- `completedAt`

### 4.1 状态

最小状态：

- `pending`
- `processing`
- `completed`
- `cancelled`

### 4.2 视图

最小视图：

- `assigned-to-me`
- `created-by-me`
- `completed`

## 5. 跳转契约

任务中心不负责完成所有业务处理，它负责：

- 找到任务
- 解释任务
- 跳到正确页面

每条任务必须提供：

- 来源域
- 目标路由
- 权限约束

## 6. UI 骨架

第一阶段最小 UI：

- 任务中心列表页
- 视图切换：待我处理 / 我发起 / 已完成
- 基础筛选：状态 / 优先级
- 行级跳转

不要求第一阶段就有：

- 甘特图
- 看板
- 批量流转

## 7. 与通知中心 / dashboard 的关系

- 通知中心展示“任务消息”
- dashboard 展示“任务摘要卡片”
- 任务中心承接统一任务列表

这三者应共享同一任务主模型，而不是各自维护一套副本。

## 8. 审计与权限

至少需要：

- 页面进入权限
- 任务查看权限
- 源域处理权限仍由源域控制

写操作如“完成 / 驳回 / 认领”不一定都在第一阶段实现，但未来必须由源域审计留痕。

## 9. 完成定义

第一阶段完成标准：

- 有统一任务模型
- 有平台入口
- 有任务列表页
- 有来源域跳转契约
- 能被 dashboard / 通知中心引用摘要

不是以“已经有完整审批系统”作为完成标准。
