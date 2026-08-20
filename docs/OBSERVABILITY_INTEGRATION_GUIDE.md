# Observability 模块完整集成指南

## 📋 概述

本文档提供 Observability 模块完整集成到 pantheon-ops 的详细步骤和说明。

**当前状态**:
- ✅ 后端代码实现完成
- ✅ 后端编译通过
- ✅ 前端代码实现完成
- ⏳ 后端模块注册待完成
- ⏳ 数据库表待初始化
- ⏳ 前端构建检查待通过

---

## 🎯 待完成任务清单

### 1. 数据库表初始化 ⏳

**文件位置**: `backend/modules/business/observability/schema.sql`

**执行方式**:
```bash
# 方式 1: 直接执行 SQL 文件
mysql -u root -p pantheon_base < backend/modules/business/observability/schema.sql

# 方式 2: 通过数据库迁移（如果项目使用迁移）
# 将 schema.sql 转换为迁移文件并执行
```

**创建的表**:
- `business_metric_source` - 指标源表
- `business_alert_rule` - 告警规则表
- `business_alert_record` - 告警记录表
- `business_notification_channel` - 通知渠道表

**注意事项**:
- 确保数据库连接正常
- 备份现有数据（如果是生产环境）
- 检查表是否创建成功：`SHOW TABLES LIKE 'business_%';`

---

### 2. 后端模块注册配置 ⏳

根据前端构建检查的错误，需要在后端添加以下配置：

#### 2.1 组件白名单注册

**需要修改的文件**: 可能是 `backend/modules/business/observability/module.go` 或类似的模块配置文件

**需要添加的组件键**:
```
business/observability/MetricSourceList
business/observability/AlertRuleList
business/observability/AlertRecordList
business/observability/ActiveAlertList
business/observability/NotificationChannelList
```

**示例参考**: 查看 `backend/modules/business/bizscope/module.go` 中的组件注册方式

#### 2.2 菜单种子数据

**需要添加的菜单项**:

```go
// 主菜单
{
    Path:       "/business/observability",
    TitleKey:   "operations.observability.menu",
    Icon:       "eye",
    RouteName:  "observability",
    Module:     "business.observability",
    Type:       "directory", // 目录类型
}

// 子菜单 - 指标源
{
    Path:       "/business/observability/metrics/sources",
    ParentPath: "/business/observability",
    TitleKey:   "operations.observability.metrics.sources.menu",
    Icon:       "code-block",
    RouteName:  "observability-metric-sources",
    Module:     "business.observability",
    Component:  "business/observability/MetricSourceList",
    PagePerm:   "business:observability:metric_source:view",
}

// 子菜单 - 告警规则
{
    Path:       "/business/observability/alerts/rules",
    ParentPath: "/business/observability",
    TitleKey:   "operations.observability.alerts.rules.menu",
    Icon:       "notification",
    RouteName:  "observability-alert-rules",
    Module:     "business.observability",
    Component:  "business/observability/AlertRuleList",
    PagePerm:   "business:observability:alert_rule:view",
}

// 子菜单 - 活跃告警
{
    Path:       "/business/observability/alerts/active",
    ParentPath: "/business/observability",
    TitleKey:   "operations.observability.alerts.active.menu",
    Icon:       "exclamation-circle",
    RouteName:  "observability-active-alerts",
    Module:     "business.observability",
    Component:  "business/observability/ActiveAlertList",
    PagePerm:   "business:observability:alert_record:view",
}

// 子菜单 - 告警历史
{
    Path:       "/business/observability/alerts/records",
    ParentPath: "/business/observability",
    TitleKey:   "operations.observability.alerts.records.menu",
    Icon:       "history",
    RouteName:  "observability-alert-records",
    Module:     "business.observability",
    Component:  "business/observability/AlertRecordList",
    PagePerm:   "business:observability:alert_record:view",
}

// 子菜单 - 通知渠道
{
    Path:       "/business/observability/alerts/channels",
    ParentPath: "/business/observability",
    TitleKey:   "operations.observability.alerts.channels.menu",
    Icon:       "send",
    RouteName:  "observability-notification-channels",
    Module:     "business.observability",
    Component:  "business/observability/NotificationChannelList",
    PagePerm:   "business:observability:notification_channel:view",
}
```

**参考示例**: `backend/modules/business/bizscope/module.go` 中的 `menuSeeds`

#### 2.3 国际化种子数据

**需要添加的 i18n keys** (中英文):

```go
// 中文 (zh-CN)
{
    Key:    "operations.observability.menu",
    Value:  "可观测性",
    Module: "business.observability",
    Locale: "zh-CN",
},
{
    Key:    "operations.observability.metrics.sources.menu",
    Value:  "指标源",
    Module: "business.observability",
    Locale: "zh-CN",
},
{
    Key:    "operations.observability.alerts.rules.menu",
    Value:  "告警规则",
    Module: "business.observability",
    Locale: "zh-CN",
},
{
    Key:    "operations.observability.alerts.records.menu",
    Value:  "告警历史",
    Module: "business.observability",
    Locale: "zh-CN",
},
{
    Key:    "operations.observability.alerts.active.menu",
    Value:  "活跃告警",
    Module: "business.observability",
    Locale: "zh-CN",
},
{
    Key:    "operations.observability.alerts.channels.menu",
    Value:  "通知渠道",
    Module: "business.observability",
    Locale: "zh-CN",
},

// 英文 (en-US)
{
    Key:    "operations.observability.menu",
    Value:  "Observability",
    Module: "business.observability",
    Locale: "en-US",
},
{
    Key:    "operations.observability.metrics.sources.menu",
    Value:  "Metric Sources",
    Module: "business.observability",
    Locale: "en-US",
},
{
    Key:    "operations.observability.alerts.rules.menu",
    Value:  "Alert Rules",
    Module: "business.observability",
    Locale: "en-US",
},
{
    Key:    "operations.observability.alerts.records.menu",
    Value:  "Alert History",
    Module: "business.observability",
    Locale: "en-US",
},
{
    Key:    "operations.observability.alerts.active.menu",
    Value:  "Active Alerts",
    Module: "business.observability",
    Locale: "en-US",
},
{
    Key:    "operations.observability.alerts.channels.menu",
    Value:  "Notification Channels",
    Module: "business.observability",
    Locale: "en-US",
},
```

**参考示例**: `backend/modules/business/bizscope/module.go` 中的 `i18nSeeds`

#### 2.4 权限种子数据

**需要添加的权限**:

```go
{
    Key:         "business:observability:metric_source:view",
    Description: "查看指标源",
    Module:      "business.observability",
},
{
    Key:         "business:observability:metric_source:create",
    Description: "创建指标源",
    Module:      "business.observability",
},
{
    Key:         "business:observability:metric_source:update",
    Description: "编辑指标源",
    Module:      "business.observability",
},
{
    Key:         "business:observability:metric_source:delete",
    Description: "删除指标源",
    Module:      "business.observability",
},
{
    Key:         "business:observability:alert_rule:view",
    Description: "查看告警规则",
    Module:      "business.observability",
},
{
    Key:         "business:observability:alert_rule:create",
    Description: "创建告警规则",
    Module:      "business.observability",
},
{
    Key:         "business:observability:alert_rule:update",
    Description: "编辑告警规则",
    Module:      "business.observability",
},
{
    Key:         "business:observability:alert_rule:delete",
    Description: "删除告警规则",
    Module:      "business.observability",
},
{
    Key:         "business:observability:alert_record:view",
    Description: "查看告警记录",
    Module:      "business.observability",
},
{
    Key:         "business:observability:notification_channel:view",
    Description: "查看通知渠道",
    Module:      "business.observability",
},
{
    Key:         "business:observability:notification_channel:create",
    Description: "创建通知渠道",
    Module:      "business.observability",
},
{
    Key:         "business:observability:notification_channel:update",
    Description: "编辑通知渠道",
    Module:      "business.observability",
},
{
    Key:         "business:observability:notification_channel:delete",
    Description: "删除通知渠道",
    Module:      "business.observability",
},
```

**参考示例**: `backend/modules/business/bizscope/module.go` 中的权限定义

---

### 3. 创建后端模块配置文件

如果 `backend/modules/business/observability/` 目录下还没有 `module.go` 文件，需要创建一个。

**文件**: `backend/modules/business/observability/module.go`

**内容参考** `backend/modules/business/bizscope/module.go`，包含：
- 模块常量定义
- `menuSeed` 结构体和数据
- `i18nSeed` 结构体和数据
- `InitModule()` 函数
- 菜单和 i18n 的数据库同步逻辑

---

## ✅ 验证步骤

完成上述配置后，按以下步骤验证：

### 1. 后端验证

```bash
cd backend

# 1. 编译检查
go build -o /tmp/pantheon-server ./cmd/server

# 2. 启动服务
go run cmd/server/main.go

# 3. 检查 API 端点
curl http://localhost:8080/api/v1/observability/metrics/sources
```

**预期结果**:
- 编译无错误
- 服务正常启动
- API 返回 200 或 401（需要登录）

### 2. 数据库验证

```sql
-- 检查表是否创建
SHOW TABLES LIKE 'business_%';

-- 检查菜单是否注册
SELECT * FROM system_menu WHERE path LIKE '/business/observability%';

-- 检查国际化是否注册
SELECT * FROM system_i18n WHERE module = 'business.observability';

-- 检查权限是否注册
SELECT * FROM system_permission WHERE `key` LIKE 'business:observability:%';
```

### 3. 前端验证

```bash
cd frontend

# 1. 重新构建检查
npm run build

# 2. 检查错误是否解决
# 应该没有关于 observability 的错误
```

**预期结果**:
- 前端构建通过
- 无组件白名单错误
- 无菜单/i18n 缺失错误

### 4. 完整功能验证

1. 启动后端服务
2. 启动前端开发服务：`cd frontend && npm run dev`
3. 登录系统
4. 检查左侧菜单是否出现"可观测性"菜单
5. 点击菜单，验证页面是否正常加载
6. 测试基本的 CRUD 功能

---

## 🐛 常见问题

### Q1: 前端构建检查失败

**问题**: 提示组件白名单缺失

**解决**: 确保在后端模块配置文件中注册了所有组件键

### Q2: 菜单不显示

**可能原因**:
1. 菜单种子数据未正确注册
2. 权限配置问题
3. 用户角色没有相关权限

**排查**:
```sql
-- 检查菜单是否存在
SELECT * FROM system_menu WHERE module = 'business.observability';

-- 检查用户权限
SELECT * FROM system_user_permission WHERE user_id = YOUR_USER_ID;
```

### Q3: API 404 错误

**可能原因**:
1. 路由未正确注册
2. 后端服务未重启

**解决**:
1. 检查 `backend/modules/business/business.go` 中是否调用了 `initObservabilityModule`
2. 重启后端服务

### Q4: 数据库表创建失败

**可能原因**:
1. SQL 语法错误
2. 权限不足
3. 表已存在

**解决**:
```bash
# 检查错误日志
mysql -u root -p pantheon_base < backend/modules/business/observability/schema.sql 2>&1

# 如果表已存在，删除后重新创建
DROP TABLE IF EXISTS business_metric_source;
DROP TABLE IF EXISTS business_alert_rule;
DROP TABLE IF EXISTS business_alert_record;
DROP TABLE IF EXISTS business_notification_channel;
```

---

## 📝 后续工作 (Week 2)

完成基础集成后，Week 2 的增强工作：

1. **Prometheus 集成**
   - Prometheus API 客户端封装
   - 指标查询功能
   - 指标数据可视化

2. **指标源表单**
   - 创建/编辑指标源弹窗
   - 表单验证
   - 连接测试功能

3. **告警规则管理**
   - PromQL 编辑器
   - PromQL 语法验证
   - 告警规则测试

4. **通知渠道配置**
   - 各类通知渠道表单
   - 通知测试功能
   - 通知模板配置

5. **活跃告警监控**
   - 实时告警刷新（30秒轮询）
   - 告警确认/恢复操作
   - 告警详情查看

---

## 🔗 相关文档

- [Observability 模块 README](../backend/modules/business/observability/README.md)
- [Observability 实现指南](../backend/modules/business/observability/IMPLEMENTATION.md)
- [后端集成完成报告](./observability-backend-integration-completed.md)
- [Week 1 完整报告](./v2-sprint1-week1-report.md)

---

## 📞 需要帮助？

如果在集成过程中遇到问题：

1. 查看本文档的常见问题部分
2. 检查相关模块的参考实现（如 bizscope）
3. 查看 Git 提交历史了解变更
4. 联系开发团队

---

**最后更新**: 2026-08-20  
**状态**: 前后端代码完成，待后端配置和数据库初始化  
**下一步**: 按本文档步骤完成后端模块注册和数据库初始化
