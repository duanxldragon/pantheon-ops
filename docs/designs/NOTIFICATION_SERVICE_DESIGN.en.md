---
title: Notification Service Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# Notification Service Design

Chinese version: [NOTIFICATION_SERVICE_DESIGN.md](./NOTIFICATION_SERVICE_DESIGN.md)

## 1. Overview

The notification service provides multi-channel notifications for users.

### 1.1 Supported Channels

| Channel   | Implementation        | Notes                       |
| --------- | --------------------- | --------------------------- |
| In-App    | Self-built            | Notification center         |
| Email     | SMTP                  | Requires mail server config |
| SMS       | Alibaba/Tencent Cloud | Requires cloud SMS service  |
| WebSocket | Gin WebSocket         | Real-time push              |

### 1.2 Notification Types

| Type     | Trigger                      | Priority |
| -------- | ---------------------------- | -------- |
| Approval | Submit, approve, reject      | High     |
| Task     | Assign, reminder             | High     |
| System   | Scheduled task, announcement | Medium   |
| Security | Login from new location      | High     |

---

## 2. Data Model

```sql
CREATE TABLE system_notification (
    id          BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id     BIGINT NOT NULL,
    type        VARCHAR(32) NOT NULL,
    title       VARCHAR(256) NOT NULL,
    content     TEXT,
    channels    VARCHAR(64) NOT NULL DEFAULT 'in_app',
    status      TINYINT DEFAULT 1, -- 1=pending, 2=sent, 3=failed
    read_at     DATETIME,
    created_at  DATETIME
);
```

---

## 3. API Design

| Endpoint                   | Method | Description        |
| -------------------------- | ------ | ------------------ |
| `/notification/list`       | GET    | List notifications |
| `/notification/:id/read`   | PUT    | Mark as read       |
| `/notification/preference` | PUT    | Update preferences |
| `/ws/notifications`        | WS     | Real-time push     |

---

## 4. Migration Path

| Phase   | Tasks                          |
| ------- | ------------------------------ |
| Phase 1 | In-app notifications, list API |
| Phase 2 | Email, WebSocket               |
| Phase 3 | SMS, templates                 |

---

## 5. Acceptance Criteria

- [ ] In-app notifications work
- [ ] Notification list with pagination
- [ ] WebSocket real-time push
- [ ] User preference settings
- [ ] i18n support
- [ ] Retry mechanism for failures
