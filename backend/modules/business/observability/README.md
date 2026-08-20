# Observability Module

## 概述

可观测性模块提供企业级监控告警能力，支持指标采集、告警规则、通知渠道等核心功能。

## 功能范围

### 1. 指标源管理 (Metric Sources)
- 支持 Prometheus、VictoriaMetrics
- 指标源注册与认证
- 健康检查与状态监控

### 2. 告警管理 (Alert Management)
- PromQL 告警规则配置
- 告警严重级别（critical/warning/info）
- 告警记录与历史查询
- 活跃告警实时监控

### 3. 通知渠道 (Notification Channels)
- 邮件、钉钉、企业微信、Slack
- 通知模板配置
- 渠道测试能力

## 数据模型

### MetricSource（指标源）
```go
type MetricSource struct {
    ID             uint64
    Name           string
    Type           string // prometheus, victoria-metrics
    Endpoint       string
    CredentialRef  string // 凭据引用
    BusinessScopeID uint64
    Status         string // active, inactive
    Config         JSON
}
```

### AlertRule（告警规则）
```go
type AlertRule struct {
    ID                      uint64
    MetricSourceID          uint64
    Name                    string
    BusinessScopeID         uint64
    Environment             string // prod, test, dev
    PromQL                  string
    Duration                string
    Severity                string // critical, warning, info
    Labels                  JSON
    Annotations             JSON
    NotificationChannelIDs  []uint64
    Status                  string // enabled, disabled
}
```

### AlertRecord（告警记录）
```go
type AlertRecord struct {
    ID                 uint64
    AlertRuleID        uint64
    FiredAt            time.Time
    ResolvedAt         *time.Time
    Severity           string
    Labels             JSON
    Annotations        JSON
    NotificationsSent  JSON
}
```

### NotificationChannel（通知渠道）
```go
type NotificationChannel struct {
    ID             uint64
    Name           string
    Type           string // email, dingtalk, wechat, slack
    Config         JSON
    BusinessScopeID uint64
    Status         string // active, inactive
}
```

## API 端点

### 指标源
- `GET /api/v1/observability/metrics/sources` - 列表
- `POST /api/v1/observability/metrics/sources` - 创建
- `GET /api/v1/observability/metrics/sources/:id` - 详情
- `PUT /api/v1/observability/metrics/sources/:id` - 更新
- `DELETE /api/v1/observability/metrics/sources/:id` - 删除

### 告警规则
- `GET /api/v1/observability/alerts/rules` - 列表
- `POST /api/v1/observability/alerts/rules` - 创建
- `GET /api/v1/observability/alerts/rules/:id` - 详情
- `PUT /api/v1/observability/alerts/rules/:id` - 更新
- `DELETE /api/v1/observability/alerts/rules/:id` - 删除
- `POST /api/v1/observability/alerts/rules/validate` - PromQL 验证

### 告警记录
- `GET /api/v1/observability/alerts/records` - 历史记录
- `GET /api/v1/observability/alerts/active` - 活跃告警

### 通知渠道
- `GET /api/v1/observability/alerts/channels` - 列表
- `POST /api/v1/observability/alerts/channels` - 创建
- `GET /api/v1/observability/alerts/channels/:id` - 详情
- `PUT /api/v1/observability/alerts/channels/:id` - 更新
- `DELETE /api/v1/observability/alerts/channels/:id` - 删除
- `POST /api/v1/observability/alerts/channels/:id/test` - 测试通知

## 前端路由

```
/business/observability
├── /metrics
│   └── /sources              # 指标源列表
├── /alerts
│   ├── /rules                # 告警规则
│   ├── /records              # 告警历史
│   ├── /active               # 活跃告警
│   └── /channels             # 通知渠道
```

## 开发计划

### Week 1 (当前)
- ✅ 数据模型定义
- ✅ 基础 CRUD API
- ✅ 前端页面框架
- ⏳ 指标源表单弹窗

### Week 2
- 指标源连接测试
- Prometheus API 集成
- VictoriaMetrics 支持

### Week 3
- 告警规则配置表单
- PromQL 编辑器
- 告警规则启用/停用

### Week 4
- 通知渠道配置
- 邮件/钉钉/企业微信集成
- 通知模板管理

### Week 5
- 告警评估引擎
- 告警触发与恢复
- 通知发送调度

### Week 6
- 活跃告警看板
- 告警历史查询优化
- 告警统计报表

## 技术栈

### 后端
- Gin 路由
- GORM ORM
- Prometheus Go Client
- 通知客户端（email/dingtalk/wechat）

### 前端
- React 18
- TypeScript
- Arco Design
- React Router v6

## 依赖关系

### 业务域关联
- 关联 Business Scope（业务范围）
- 关联 Department（部门）
- 支持多租户隔离

### 凭据管理
- 指标源凭据存储在凭据模块
- 通知渠道凭据独立管理

## 扩展性设计

### 1. 多指标源支持
当前支持 Prometheus/VictoriaMetrics，未来可扩展：
- Thanos
- Mimir
- Cortex
- InfluxDB

### 2. 告警规则引擎
预留告警规则扩展点：
- LogQL（Loki 日志查询）
- TraceQL（Tempo 链路查询）
- 自定义脚本告警

### 3. 通知渠道
预留通知渠道扩展点：
- Webhook
- PagerDuty
- OpsGenie
- 短信通知

## 注意事项

1. **告警评估性能**
   - 告警规则应避免高基数查询
   - 建议设置合理的评估间隔（>=1m）

2. **通知防抖**
   - 同一告警的重复通知间隔
   - 告警风暴抑制策略

3. **数据保留**
   - 告警记录建议保留 90 天
   - 历史数据归档策略

4. **权限控制**
   - 告警规则的创建/修改权限
   - 通知渠道的测试权限
   - 跨部门告警规则访问控制
