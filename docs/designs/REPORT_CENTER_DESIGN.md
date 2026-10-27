---
title: 报表中心设计
doc_type: Design
layer: platform / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# 报表中心设计

English version: [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 标准后台产品位中的“报表中心”最小骨架。

目标不是立刻做成 BI 平台，而是先统一报表清单、查看入口、导出边界和权限审计语义。

## 1. 设计目标

报表中心要解决：

1. 平台层对各域报表有统一入口，而不是散落在业务页面里
2. 报表定义、适用权限、导出动作有一致治理模型
3. 后续 dashboard、调度中心和报表归档可以复用同一套元数据

## 2. 非目标

当前阶段不做：

- 自助式拖拽分析器
- 多维 OLAP 引擎
- 可视化数据建模工作台
- 海量交互式图表设计器

## 3. 归属边界

### 3.1 `platform` 负责

- 报表中心入口
- 报表清单聚合
- 报表元数据视图
- 导出动作入口治理

### 3.2 源域负责

- 报表数据生成逻辑
- 数据口径定义
- 报表详情实现
- 导出内容正确性

## 4. 报表元数据模型

建议最小字段：

- `id`
- `reportType`
- `sourceDomain`
- `name`
- `description`
- `category`
- `visibility`
- `supportedFormats`
- `defaultFilters`
- `owner`
- `updatedAt`

### 4.1 可见性

最小可见性：

- `private`
- `team`
- `platform`

### 4.2 导出格式

第一阶段建议支持：

- `csv`
- `xlsx`
- `pdf`

## 5. 核心动作契约

第一阶段统一动作：

- `view`
- `export`
- `copy-link`

约束：

- 导出动作必须受权限和审计约束
- 不允许在没有稳定数据口径时包装成“官方报表”
- 报表中心展示的是注册后的报表能力，不是自由 SQL 查询入口

## 6. UI 骨架

第一阶段最小 UI：

- 报表中心列表页
- 分类筛选
- 来源域筛选
- 报表说明与更新时间展示
- 详情跳转或内嵌查看入口

暂不强制：

- 大屏设计器
- 图表布局编辑器
- 复杂订阅能力

## 7. 审计与权限

至少需要：

- 页面访问权限
- 报表查看权限
- 报表导出权限
- 高敏数据报表额外权限策略

必须记录：

- 谁查看了报表
- 谁导出了报表
- 导出格式
- 导出时间
- 过滤条件摘要

## 8. 与调度中心 / dashboard 的关系

推荐关系：

- 调度中心负责离线生成或定时归档报表
- 报表中心负责查看与导出入口
- dashboard 只消费报表摘要，不承载完整报表管理

## 9. 最小样板要求

第一阶段至少应接入一类真实报表：

- 用户与权限治理报表
- 业务资源盘点报表
- 审计事件汇总报表

## 10. 完成定义

第一阶段完成标准：

- 有统一报表元数据模型
- 有报表中心入口
- 有查看与导出动作契约
- 有权限与审计留痕
- 有至少一个真实报表样板接入

不是以“已经做成完整 BI 平台”作为完成标准。
