---
title: 消息通知服务设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# 消息通知服务设计

English version: [NOTIFICATION_SERVICE_DESIGN.en.md](./NOTIFICATION_SERVICE_DESIGN.en.md)

## 1. 概述

消息通知服务是平台基础设施，为用户提供及时的业务事件通知。

### 1.1 支持的渠道

| 渠道      | 实现          | 说明               |
| --------- | ------------- | ------------------ |
| 站内信    | 自建          | 通知中心           |
| Email     | SMTP          | 需要配置邮箱服务器 |
| SMS       | 阿里云/腾讯云 | 需要配置云短信服务 |
| WebSocket | Gin WebSocket | 即时通知           |

### 1.2 通知类型

| 类型     | 触发场景               | 优先级 |
| -------- | ---------------------- | ------ |
| 审批通知 | 提交审批、审批完成     | 高     |
| 任务通知 | 任务分配、到期提醒     | 高     |
| 系统通知 | 计划任务完成、系统公告 | 中     |
| 安全通知 | 异地登录、密码修改     | 高     |

---

## 2. 数据模型

### 2.1 通知记录表

```sql
CREATE TABLE system_notification (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id     BIGINT NOT NULL COMMENT '接收用户ID',
    type        VARCHAR(32) NOT NULL COMMENT '通知类型',
    title       VARCHAR(256) NOT NULL COMMENT '通知标题',
    content     TEXT COMMENT '通知内容',
    data        JSON COMMENT '扩展数据',
    channels    VARCHAR(64) NOT NULL DEFAULT 'in_app' COMMENT '发送渠道，逗号分隔',
    status      TINYINT DEFAULT 1 COMMENT '1=pending, 2=sent, 3=failed',
    sent_at     DATETIME COMMENT '发送时间',
    read_at     DATETIME COMMENT '阅读时间',
    created_at  DATETIME,
    updated_at  DATETIME,
    INDEX idx_user_status (user_id, status),
    INDEX idx_user_created (user_id, created_at)
);

CREATE TABLE system_notification_channel (
    id              BIGINT PRIMARY KEY AUTO_INCREMENT,
    notification_id BIGINT NOT NULL,
    channel         VARCHAR(32) NOT NULL COMMENT 'in_app, email, sms, websocket',
    status          TINYINT DEFAULT 1 COMMENT '1=pending, 2=success, 3=failed',
    error_message   VARCHAR(512) COMMENT '失败原因',
    sent_at         DATETIME,
    created_at      DATETIME,
    INDEX idx_notification (notification_id)
);

CREATE TABLE system_user_notification_preference (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id     BIGINT NOT NULL UNIQUE,
    email       VARCHAR(256) COMMENT '邮箱地址',
    phone       VARCHAR(32) COMMENT '手机号',
    email_enabled BOOLEAN DEFAULT TRUE,
    sms_enabled   BOOLEAN DEFAULT TRUE,
    created_at  DATETIME,
    updated_at  DATETIME
);
```

---

## 3. 服务设计

### 3.1 核心接口

```go
// backend/modules/notification/notification_service.go
package notification

type NotificationService struct {
    db          *gorm.DB
    channels    []Channel // Email, SMS, WebSocket
}

type SendNotificationReq struct {
    UserID    uint64                 `json:"user_id" binding:"required"`
    Type      string                 `json:"type" binding:"required"`
    Title     string                 `json:"title" binding:"required"`
    Content   string                 `json:"content"`
    Data      map[string]interface{} `json:"data"`
    Channels  []string               `json:"channels"` // 默认 in_app
    Priority  int                    `json:"priority"`
}

func (s *NotificationService) Send(ctx context.Context, req *SendNotificationReq) error
func (s *NotificationService) SendBatch(ctx context.Context, reqs []*SendNotificationReq) error

// 用户偏好
func (s *NotificationService) GetUserPreference(userID uint64) (*UserPreference, error)
func (s *NotificationService) UpdateUserPreference(userID uint64, pref *UserPreference) error
```

### 3.2 通知渠道接口

```go
// backend/modules/notification/channels/channel.go
type Channel interface {
    Name() string
    Send(ctx context.Context, msg *Notification) error
}

// 站内信渠道
type InAppChannel struct{}
func (c *InAppChannel) Name() string { return "in_app" }
func (c *InAppChannel) Send(ctx context.Context, msg *Notification) error

// Email 渠道
type EmailChannel struct {
    SMTPHost string
    SMTPPort int
    Username string
    Password string
}
func (c *EmailChannel) Name() string { return "email" }
func (c *EmailChannel) Send(ctx context.Context, msg *Notification) error

// WebSocket 渠道
type WebSocketChannel struct {
    hub *Hub
}
func (c *WebSocketChannel) Name() string { return "websocket" }
func (c *WebSocketChannel) Send(ctx context.Context, msg *Notification) error
```

### 3.3 通知模板

```go
// 模板支持变量替换
type NotificationTemplate struct {
    Type    string
    Title   string // 支持 {user_name}, {action}, {time} 等变量
    Content string
}

var templates = map[string]NotificationTemplate{
    "approval_submitted": {
        Title:   "您有一个待审批任务",
        Content: "{user_name} 提交了 {biz_type}，请尽快处理",
    },
    "approval_completed": {
        Title:   "审批已完成",
        Content: "您的 {biz_type} 申请已{result}，{comment}",
    },
    "task_reminder": {
        Title:   "任务提醒",
        Content: "您有一个任务即将到期：{task_name}，剩余 {remaining_time}",
    },
}
```

---

## 4. API 设计

### 4.1 后端接口

| 接口                         | 方法 | 说明             |
| ---------------------------- | ---- | ---------------- |
| `/notification/list`         | GET  | 通知列表         |
| `/notification/:id`          | GET  | 通知详情         |
| `/notification/:id/read`     | PUT  | 标记已读         |
| `/notification/read-all`     | PUT  | 全部已读         |
| `/notification/unread-count` | GET  | 未读数量         |
| `/notification/preference`   | GET  | 用户偏好         |
| `/notification/preference`   | PUT  | 更新偏好         |
| `/notification/send`         | POST | 发送通知（内部） |

### 4.2 WebSocket 接口

```go
// WebSocket 连接
GET /ws/notifications?token=xxx

// 推送消息格式
{
    "type": "notification",
    "data": {
        "id": 123,
        "title": "您有一个待审批任务",
        "content": "...",
        "priority": "high"
    }
}
```

---

## 5. 前端集成

### 5.1 通知中心组件

```typescript
// frontend/src/modules/notification/components/NotificationCenter.tsx
interface NotificationCenterProps {
  placement?: "dropdown" | "drawer" | "modal";
}

// 功能：
// - 未读数量徽章
// - 通知列表（支持分页）
// - 标记已读/全部已读
// - WebSocket 实时推送
```

### 5.2 通知偏好设置

```typescript
// frontend/src/modules/notification/components/NotificationPreference.tsx
interface NotificationPreferenceProps {
  // 渠道开关：站内信、邮件、短信
  // 通知类型开关：审批、任务、系统、安全
}
```

---

## 6. 国际化

通知模板支持 i18n：

```json
{
  "notification.approval.submitted.title": "You have a pending approval",
  "notification.approval.submitted.content": "{user_name} submitted {biz_type}, please process",
  "notification.approval.completed.title": "Approval completed",
  "notification.approval.completed.content": "Your {biz_type} request has been {result}"
}
```

---

## 7. 迁移路径

### Phase 1: 基础通知

- [ ] 数据模型和迁移
- [ ] 站内信渠道实现
- [ ] 通知列表 API
- [ ] 通知中心组件

### Phase 2: 多渠道

- [ ] Email 渠道
- [ ] WebSocket 实时推送
- [ ] 用户偏好设置

### Phase 3: 高级功能

- [ ] SMS 渠道
- [ ] 通知模板管理
- [ ] 通知聚合

---

## 8. 验收标准

- [ ] 站内信通知可发送和接收
- [ ] 通知列表支持分页、已读/未读过滤
- [ ] WebSocket 实时推送可用
- [ ] 用户可配置通知偏好
- [ ] 通知内容支持 i18n
- [ ] 失败通知有重试机制
