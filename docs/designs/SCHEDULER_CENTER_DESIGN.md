---
title: 调度中心设计
doc_type: Design
layer: platform / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-18
---

# 调度中心设计

English version: [SCHEDULER_CENTER_DESIGN.en.md](./SCHEDULER_CENTER_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_CONFIG_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 标准后台产品位中的“调度中心”最小骨架。

目标不是立即把系统演进成重量级调度平台，而是先形成统一的计划任务入口、执行记录视图和接入治理边界。

## 1. 设计目标

调度中心要解决：

1. 系统级与业务级计划任务有统一注册与展示入口
2. 任务定义、启停状态、最近执行结果具备一致表达
3. 高风险执行动作具备权限与审计留痕

## 2. 非目标

当前阶段不做：

- 分布式作业编排平台
- 可视化 DAG 编排器
- 多云批处理控制台
- 任意脚本远程执行器

## 3. 归属边界

### 3.1 `platform` 负责

- 调度中心产品位
- 任务注册清单聚合
- 任务执行记录总览
- 统一状态摘要

### 3.2 `system/config` 负责

- 平台级任务配置治理
- 默认执行策略约束
- 高敏操作保护策略

### 3.3 源域负责

- 实际任务逻辑
- 任务参数校验
- 执行结果业务语义

## 4. 调度任务模型

建议最小字段：

- `id`
- `jobType`
- `sourceDomain`
- `name`
- `description`
- `scheduleType`
- `scheduleExpr`
- `enabled`
- `lastRunAt`
- `lastRunStatus`
- `lastRunSummary`
- `owner`

### 4.1 执行状态

最小状态：

- `idle`
- `running`
- `success`
- `failed`
- `paused`

### 4.2 调度类型

第一阶段建议支持：

- `cron`
- `manual`
- `event`

## 5. 核心动作契约

第一阶段统一动作：

- `enable`
- `disable`
- `run-once`
- `view-history`

约束：

- `run-once` 属于高敏动作，默认需要单独权限与审计
- 平台层只触发标准动作，不承诺所有任务都有复杂参数化能力

## 6. UI 骨架

第一阶段最小 UI：

- 调度中心列表页
- 启用状态筛选
- 来源域筛选
- 最近执行结果展示
- 执行记录详情抽屉或详情页

暂不强制：

- 任务依赖拓扑图
- 实时日志流
- 复杂批量编排

## 7. 审计与权限

至少需要：

- 页面访问权限
- 查看任务权限
- 启停任务权限
- 手动执行权限

必须记录：

- 谁启用了或停用了任务
- 谁手动执行了任务
- 何时执行
- 执行结果摘要

## 8. 与告警中心的关系

调度中心不是告警系统，但调度失败应能向告警中心暴露摘要。

推荐关系：

- 调度中心管理任务定义与执行视图
- 告警中心消费失败事件与异常摘要
- dashboard 消费调度健康摘要

## 9. 最小样板要求

第一阶段至少应接入一类真实任务：

- i18n 资产同步
- 动态模块同步
- 业务数据同步或巡检任务

## 10. 完成定义

第一阶段完成标准：

- 有统一调度任务模型
- 有调度中心入口
- 有执行记录视图
- 有高敏动作权限与审计
- 有至少一个真实任务样板接入

不是以“已经具备企业级作业编排能力”作为完成标准。
