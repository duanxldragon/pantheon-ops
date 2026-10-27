---
title: 工作流引擎选型
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# 工作流引擎选型

English version: [WORKFLOW_ENGINE_SELECTION.en.md](./WORKFLOW_ENGINE_SELECTION.en.md)

## 1. 选型背景

Pantheon Base 当前没有工作流引擎，已有的设计文档 `WORKFLOW.md` 描述了审批流程与 RBAC 的集成需求，但尚未落地实现。

本文档对主流工作流引擎进行评估，为 `P2.1 工作流引擎选型` 提供决策依据。

---

## 2. 候选引擎对比

### 2.1 Camunda

| 维度       | 评分       | 说明                      |
| ---------- | ---------- | ------------------------- |
| 成熟度     | ⭐⭐⭐⭐⭐ | BPMN 2.0 标准，工业级稳定 |
| 学习曲线   | ⭐⭐       | 陡峭，概念多              |
| 集成难度   | ⭐⭐       | 需要部署独立服务          |
| 功能覆盖   | ⭐⭐⭐⭐⭐ | 会签、或签、子流程、事件  |
| 社区活跃度 | ⭐⭐⭐⭐   | 成熟社区                  |
| 开源协议   | ⭐⭐⭐⭐   | Apache 2.0                |
| Go 集成    | ⭐⭐       | 提供 REST API，需网络调用 |

### 2.2 Temporal

| 维度       | 评分       | 说明                     |
| ---------- | ---------- | ------------------------ |
| 成熟度     | ⭐⭐⭐⭐   | 云原生，微服务友好       |
| 学习曲线   | ⭐⭐⭐     | 概念新，但文档清晰       |
| 集成难度   | ⭐⭐⭐     | 需要部署服务             |
| 功能覆盖   | ⭐⭐⭐⭐   | 长时运行任务、重试、补偿 |
| 社区活跃度 | ⭐⭐⭐⭐   | 活跃，Airbnb 背书        |
| 开源协议   | ⭐⭐⭐⭐   | MIT                      |
| Go 集成    | ⭐⭐⭐⭐⭐ | 原生 Go SDK              |

### 2.3 Flowise / n8n

| 维度       | 评分       | 说明             |
| ---------- | ---------- | ---------------- |
| 成熟度     | ⭐⭐⭐     | 轻量级           |
| 学习曲线   | ⭐⭐⭐⭐⭐ | 低代码，拖拽配置 |
| 集成难度   | ⭐⭐⭐⭐   | 部署简单         |
| 功能覆盖   | ⭐⭐⭐     | 适合简单流程     |
| 社区活跃度 | ⭐⭐⭐⭐   | 快速增长         |
| 开源协议   | ⭐⭐⭐⭐   | Apache 2.0       |
| Go 集成    | ⭐⭐       | 提供 REST API    |

### 2.4 自建轻量引擎

| 维度       | 评分       | 说明               |
| ---------- | ---------- | ------------------ |
| 成熟度     | ⭐⭐       | 从头构建           |
| 学习曲线   | ⭐⭐⭐     | 无外部依赖         |
| 集成难度   | ⭐⭐⭐⭐⭐ | 同进程，无网络开销 |
| 功能覆盖   | ⭐⭐       | 基础审批流         |
| 社区活跃度 | N/A        | 维护成本高         |
| 开源协议   | N/A        | 自有代码           |
| Go 集成    | ⭐⭐⭐⭐⭐ | 直接 import        |

---

## 3. 推荐方案

### 3.1 短期方案（基础审批）

对于简单的审批流程（单人审批、会签），建议自建轻量引擎：

```go
// backend/modules/workflow/workflow.go
type Workflow struct {
    ID          string
    Name        string
    Steps       []WorkflowStep
    CurrentStep int
    Status      WorkflowStatus
}

type WorkflowStep struct {
    StepID     string
    Name       string
    Type       StepType  // approve, reject, notify
    AssigneeID uint64
    Deadline   time.Time
}
```

优点：

- 无额外依赖
- 与现有系统紧耦合，查询性能好
- 满足基础审批需求

缺点：

- 无 BPMN 图形化建模
- 复杂场景需要扩展

### 3.2 长期方案（Temporal）

对于复杂工作流（长时任务、分布式事务、补偿机制），建议采用 Temporal：

优点：

- 原生 Go SDK
- 完善的错误处理、重试、活动历史
- 与 Go 微服务天然契合
- Cadence 已经在生产环境验证

缺点：

- 需要额外部署 Temporal 服务
- 学习曲线

---

## 4. 选型决策矩阵

| 场景                       | 推荐     | 理由                  |
| -------------------------- | -------- | --------------------- |
| 简单审批（≤3步）           | 自建     | 无依赖，快速落地      |
| 复杂审批（会签/或签/条件） | Camunda  | BPMN 标准             |
| 长时任务/分布式事务        | Temporal | 原生 Go，错误处理完善 |
| 低代码配置                 | Flowise  | 拖拽配置，快速上线    |

---

## 5. 落地计划

### Phase 1: 基础审批引擎

- 设计工作流数据模型
- 实现状态机
- 与 RBAC 集成
- 基础 API

### Phase 2: 扩展能力

- 会签/或签
- 条件分支
- 审批历史

### Phase 3: 可视化（可选）

- BPMN 建模工具集成
- 流程监控面板

---

## 6. 数据模型

```sql
CREATE TABLE system_workflow (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    code        VARCHAR(64) NOT NULL UNIQUE COMMENT '工作流编码',
    name        VARCHAR(128) NOT NULL COMMENT '工作流名称',
    description TEXT COMMENT '描述',
    status      TINYINT DEFAULT 1 COMMENT '1=draft, 2=active, 3=suspended',
    version     INT DEFAULT 1 COMMENT '版本号',
    created_at  DATETIME,
    updated_at  DATETIME
);

CREATE TABLE system_workflow_instance (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    workflow_id BIGINT NOT NULL COMMENT '工作流定义ID',
    biz_type    VARCHAR(64) NOT NULL COMMENT '业务类型',
    biz_id      VARCHAR(64) NOT NULL COMMENT '业务ID',
    status      TINYINT DEFAULT 1 COMMENT '1=running, 2=completed, 3=cancelled',
    current_step INT DEFAULT 0,
    created_at  DATETIME,
    updated_at  DATETIME
);

CREATE TABLE system_workflow_task (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    instance_id BIGINT NOT NULL,
    step_id     VARCHAR(64) NOT NULL,
    assignee_id BIGINT COMMENT '审批人',
    status      TINYINT DEFAULT 1 COMMENT '1=pending, 2=approved, 3=rejected, 4=cancelled',
    comment     TEXT COMMENT '审批意见',
    deadline    DATETIME,
    completed_at DATETIME,
    created_at  DATETIME,
    updated_at  DATETIME
);
```

---

## 7. API 设计

| 接口                         | 方法 | 说明           |
| ---------------------------- | ---- | -------------- |
| `/workflow`                  | POST | 创建工作流定义 |
| `/workflow/:id`              | GET  | 获取工作流详情 |
| `/workflow/:id/deploy`       | POST | 部署工作流     |
| `/workflow/instance`         | POST | 启动流程实例   |
| `/workflow/instance/:id`     | GET  | 查询实例状态   |
| `/workflow/task/pending`     | GET  | 待办任务列表   |
| `/workflow/task/:id/approve` | POST | 审批通过       |
| `/workflow/task/:id/reject`  | POST | 审批拒绝       |

---

## 8. 与 RBAC 集成

```go
// 审批权限检查
func CheckApprovalPermission(userID uint64, taskID uint64) error {
    var task WorkflowTask
    if err := db.First(&task, taskID).Error; err != nil {
        return err
    }

    // 检查用户是否是任务 assignee
    if task.AssigneeID != userID {
        return ErrForbidden
    }

    // 检查用户是否在对应角色中
    if !HasRole(userID, task.RequiredRole) {
        return ErrForbidden
    }

    return nil
}
```

---

## 9. 验收标准

- [ ] 工作流定义支持 CRUD
- [ ] 流程实例可启动、审批、查询
- [ ] 待办任务列表正确过滤
- [ ] 审批日志完整记录
- [ ] 与现有 RBAC 权限集成
- [ ] 单元测试覆盖核心路径
