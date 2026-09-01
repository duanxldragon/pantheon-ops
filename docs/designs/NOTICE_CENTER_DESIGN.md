---
title: 通知中心设计
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-18
---

# 通知中心设计

English version: [NOTICE_CENTER_DESIGN.en.md](./NOTICE_CENTER_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 平台层通知中心的最小产品骨架。

目标不是直接做成 IM 系统，而是先把平台级消息中心做成可治理、可跳转、可审计的统一入口。

## 1. 设计目标

通知中心要解决：

1. 平台级消息不再散落在各系统页角落
2. 跨域消息有统一入口、统一已读状态和统一跳转模型
3. dashboard、壳层顶部和后续任务/告警摘要可共享一套消息语义

## 2. 非目标

当前阶段不做：

- 即时聊天
- 群聊 / 私信
- 富文本消息编辑器
- 多端推送网关
- 完整消息模板引擎

## 3. 归属边界

### 3.1 `platform` 负责

- 通知中心入口
- 通知聚合视图
- 已读/未读状态
- 跳转契约
- 平台级消息摘要

### 3.2 系统域 / 业务域负责

- 产生消息的业务语义
- 源数据有效性
- 跳转目标页面的详情承载

一句话：

> `platform` 负责收口消息入口，不负责替代源域业务。

## 4. 消息模型

建议最小字段：

- `id`
- `category`
- `sourceDomain`
- `title`
- `summary`
- `severity`
- `status`
- `jumpType`
- `jumpTarget`
- `createdAt`
- `readAt`
- `actorId`

### 4.1 分类

最小分类建议：

- `task`
- `alert`
- `audit`
- `system`
- `business`

### 4.2 严重级别

- `info`
- `warning`
- `critical`

### 4.3 状态

- `unread`
- `read`
- `archived`

## 5. 跳转契约

每条消息必须明确：

- 来源域
- 跳转目标
- 是否需要权限

推荐跳转类型：

- `route`
- `external-url`
- `none`

### 5.1 跳转约束

- 没有稳定目标时，允许 `jumpType = none`
- 不允许伪造跳转按钮
- 跳转前仍受页面权限控制

## 6. UI 骨架

第一阶段最小 UI：

- 顶部通知入口
- 未读数量提示
- 通知中心列表页或抽屉
- 基础筛选：全部 / 未读 / 分类

暂不强制：

- 复杂分组
- 高级搜索
- 多栏详情布局

## 7. 审计与留痕

至少记录：

- 通知生成来源
- 用户查看
- 用户标记已读
- 用户归档

通知中心本身不是审计系统，但它必须能够回溯“消息从哪里来，被谁处理过”。

## 8. 与 dashboard 的关系

dashboard 不应各自保存另一套消息模型。

正确关系：

- dashboard 只消费通知中心摘要
- 通知中心是平台级消息入口
- 业务域通过统一消息模型向平台暴露摘要

## 9. 完成定义

第一阶段完成标准：

- 有统一消息模型
- 有已读/未读语义
- 有平台入口
- 有最小跳转契约
- 有基础审计留痕

不是以“功能像企业微信”作为完成标准。
