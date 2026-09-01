---
title: 监控告警中心设计
doc_type: Design
layer: platform / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-18
---

# 监控告警中心设计

English version: [ALERT_MONITORING_CENTER_DESIGN.en.md](./ALERT_MONITORING_CENTER_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_CONFIG_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 标准后台产品位中的“监控告警中心”最小骨架。

目标不是在当前阶段直接做成完整可观测性平台，而是先形成统一告警入口、告警状态模型和处置留痕闭环。

## 1. 设计目标

监控告警中心要解决：

1. 平台层有统一异常入口，而不是错误散落在各业务模块
2. 系统级与业务级异常可以共享统一告警模型
3. 告警状态、认领、关闭和跳转具备一致表达

## 2. 非目标

当前阶段不做：

- 全量指标采集平台
- 自建日志检索引擎
- 全功能链路追踪系统
- 复杂规则拖拽编排器

## 3. 归属边界

### 3.1 `platform` 负责

- 告警中心入口
- 告警聚合视图
- 告警状态管理骨架
- 与通知中心、dashboard 的摘要联动

### 3.2 `system/config` 负责

- 平台级告警策略配置入口
- 阈值与开关类配置治理
- 告警渠道配置的后续扩展点

### 3.3 源域负责

- 异常判定逻辑
- 根因语义
- 修复动作实现
- 源域审计与业务上下文

## 4. 告警模型

建议最小字段：

- `id`
- `alertType`
- `sourceDomain`
- `title`
- `summary`
- `severity`
- `status`
- `occurredAt`
- `acknowledgedAt`
- `resolvedAt`
- `ownerId`
- `jumpTarget`

### 4.1 严重级别

最小级别：

- `info`
- `warning`
- `critical`

### 4.2 状态

最小状态：

- `open`
- `acknowledged`
- `resolved`
- `closed`

## 5. 核心动作契约

第一阶段统一动作：

- `acknowledge`
- `resolve`
- `close`
- `jump-to-source`

约束：

- 源域不支持的动作不应伪装为可用
- 告警关闭不等于根因修复，必须保留源域追踪能力

## 6. UI 骨架

第一阶段最小 UI：

- 告警中心列表页
- 严重级别筛选
- 状态筛选
- 来源域筛选
- 行级详情与跳转

暂不强制：

- 时序指标面板
- 拓扑依赖图
- 多渠道值班排班系统

## 7. 审计与权限

至少需要：

- 页面访问权限
- 告警查看权限
- 告警处置权限
- 高敏告警额外查看策略

必须记录：

- 谁认领了告警
- 谁关闭了告警
- 处置时间
- 处置备注摘要

## 8. 与通知中心 / 调度中心 / dashboard 的关系

推荐关系：

- 通知中心承载消息提醒
- 告警中心承载告警对象本身
- 调度中心暴露调度失败事件
- dashboard 消费告警摘要卡片

这几者应共享事件来源语义，而不是各自维护冲突状态。

## 9. 最小样板要求

第一阶段至少应接入一类真实异常：

- 调度任务失败
- 高敏操作异常
- 关键业务同步失败

## 10. 完成定义

第一阶段完成标准：

- 有统一告警模型
- 有告警中心入口
- 有最小处置动作契约
- 有权限与审计留痕
- 有至少一个真实异常样板接入

不是以“已经做成完整可观测性平台”作为完成标准。
