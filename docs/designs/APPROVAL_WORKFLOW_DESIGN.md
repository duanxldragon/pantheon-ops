---
title: 审批流骨架设计
doc_type: Design
layer: platform / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# 审批流骨架设计

English version: [APPROVAL_WORKFLOW_DESIGN.en.md](./APPROVAL_WORKFLOW_DESIGN.en.md)

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 标准后台产品位中的“审批流骨架”。

当前目标不是直接建设通用 BPM 平台，而是先收口“待审批 / 已处理 / 我的提交”的统一产品位、接入契约和审计语义。

## 1. 设计目标

审批流骨架要解决：

1. 平台层有统一审批入口，而不是每个业务模块各做一套待办
2. 不同业务域可以接入统一审批视图，但仍保留各自业务语义
3. 审批动作、状态流转和审计留痕具备一致表达

## 2. 非目标

当前阶段不做：

- 可视化流程设计器
- 通用规则编排引擎
- 任意节点脚本执行
- 跨组织复杂会签编排

## 3. 归属边界

### 3.1 `platform` 负责

- 审批中心入口
- 审批单统一列表模型
- 统一状态视图
- 与任务中心的聚合关系

### 3.2 `business/*` 负责

- 审批单业务语义
- 节点流转规则
- 业务详情页与处理动作
- 领域侧审计补充

一句话：

> `platform` 负责统一承载审批产品位，业务域负责真正的审批处理逻辑。

## 4. 审批单模型

建议最小字段：

- `id`
- `workflowType`
- `sourceDomain`
- `bizType`
- `bizId`
- `title`
- `status`
- `currentStep`
- `applicantId`
- `currentApproverId`
- `submittedAt`
- `processedAt`
- `jumpTarget`

### 4.1 状态

最小状态：

- `draft`
- `submitted`
- `pending`
- `approved`
- `rejected`
- `cancelled`

### 4.2 默认视图

最小视图：

- `pending-for-me`
- `processed-by-me`
- `submitted-by-me`

## 5. 动作契约

第一阶段统一动作语义：

- `approve`
- `reject`
- `transfer`
- `cancel`

约束：

- 动作能力由源域决定是否暴露
- 平台层不直接承诺所有审批单都支持全部动作
- 不允许前端展示无后端语义支撑的伪按钮

## 6. 与任务中心的关系

审批流和任务中心不是两套平行模型。

推荐关系：

- 审批单是业务对象
- 审批待办是任务中心中的一种任务
- 任务中心展示审批摘要
- 审批中心承载审批专属筛选和处理入口

## 7. UI 骨架

第一阶段最小 UI：

- 审批中心列表页
- 三个视图页签：待我审批 / 我已处理 / 我的提交
- 状态筛选
- 行级跳转到源域详情或处理页

暂不要求：

- 拖拽流程图
- 节点级流程可视化编排
- 审批统计大屏

## 8. 审计与权限

至少需要：

- 审批中心访问权限
- 审批单查看权限
- 审批动作权限
- 审批动作审计记录

必须保留：

- 提交人
- 处理人
- 动作
- 时间
- 结果
- 备注摘要

## 9. 最小样板要求

第一阶段必须至少选一个真实业务场景接入，避免空壳审批中心。

推荐样板：

- 业务资源上下线审批
- 高敏配置变更审批
- 资产录入审核

## 10. 完成定义

第一阶段完成标准：

- 有统一审批单模型
- 有审批中心入口
- 有最小动作契约
- 有至少一个真实业务样板接入
- 有权限与审计留痕

不是以“已经做出完整 BPM 平台”作为完成标准。
