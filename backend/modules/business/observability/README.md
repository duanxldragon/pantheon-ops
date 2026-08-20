# Observability Module

**状态:** V2 Sprint 1 开发中  
**创建日期:** 2026-08-20  
**负责人:** SRE Team

## 概述

Observability 模块提供完整的可观测性能力，包括：

- **Metrics（指标）** - 集成 Prometheus/VictoriaMetrics
- **Logs（日志）** - 集成 Loki/Elasticsearch
- **Alerting（告警）** - 规则配置、通知路由、降噪

## 模块结构

```
observability/
├── model.go                # 数据模型
├── repository.go           # 数据访问层
├── service.go              # 业务逻辑层
├── handler.go              # HTTP 处理器
├── router.go               # 路由注册
├── prometheus/             # Prometheus 客户端
│   ├── client.go           # HTTP API 客户端
│   ├── query.go            # 查询封装
│   └── alertmanager.go     # Alertmanager 集成
├── loki/                   # Loki 客户端（Sprint 2）
│   ├── client.go
│   └── query.go
└── notifier/               # 通知发送器
    ├── email.go            # 邮件通知
    ├── dingtalk.go         # 钉钉通知
    └── wechat.go           # 企业微信通知
```

## 数据模型

### MetricSource（指标源）
```go
type MetricSource struct {
    ID              uint64
    Name            string         // "生产环境 Prometheus"
    Type            string         // "prometheus" | "victoria-metrics"
    Endpoint        string         // "http://prometheus.prod.svc:9090"
    CredentialRef   string         // 凭据引用
    BusinessScopeID uint64
    Status          string         // "active" | "inactive"
}
```

### AlertRule（告警规则）
```go
type AlertRule struct {
    ID                     uint64
    MetricSourceID         uint64
    Name                   string
    BusinessScopeID        uint64
    Environment            string         // "prod" | "test" | "dev"
    PromQL                 string         // 'up{job="node-exporter"} == 0'
    Duration               string         // "5m"
    Severity               string         // "critical" | "warning" | "info"
    Labels                 JSON           // {"team": "ops"}
    Annotations            JSON           // {"summary": "主机宕机"}
    NotificationChannelIDs JSON           // [1, 2, 3]
    Status                 string         // "enabled" | "disabled"
}
```

### AlertRecord（告警历史）
```go
type AlertRecord struct {
    ID                uint64
    AlertRuleID       uint64
    FiredAt           time.Time
    ResolvedAt        *time.Time
    Severity          string
    NotificationsSent JSON
}
```

### NotificationChannel（通知渠道）
```go
type NotificationChannel struct {
    ID              uint64
    Name            string
    Type            string         // "email" | "dingtalk" | "wechat"
    Config          JSON           // 不同类型的配置
    BusinessScopeID uint64
    Status          string
}
```

### LogSource（日志源）
```go
type LogSource struct {
    ID              uint64
    Name            string
    Type            string         // "loki" | "elasticsearch"
    Endpoint        string
    CredentialRef   string
    BusinessScopeID uint64
    Status          string
}
```

## API 端点

### Metrics API

```
GET    /api/v1/observability/metrics/sources           # 列表指标源
POST   /api/v1/observability/metrics/sources           # 创建指标源
GET    /api/v1/observability/metrics/sources/:id       # 详情
PUT    /api/v1/observability/metrics/sources/:id       # 更新
DELETE /api/v1/observability/metrics/sources/:id       # 删除

POST   /api/v1/observability/metrics/query             # 即时查询
POST   /api/v1/observability/metrics/query_range       # 范围查询
GET    /api/v1/observability/metrics/label/:name/values # Label 值列表
```

### Alerting API

```
GET    /api/v1/observability/alerts/rules              # 列表告警规则
POST   /api/v1/observability/alerts/rules              # 创建规则
GET    /api/v1/observability/alerts/rules/:id          # 详情
PUT    /api/v1/observability/alerts/rules/:id          # 更新
DELETE /api/v1/observability/alerts/rules/:id          # 删除
POST   /api/v1/observability/alerts/rules/:id/validate # 验证 PromQL

GET    /api/v1/observability/alerts/records            # 告警历史
GET    /api/v1/observability/alerts/active             # 活跃告警

GET    /api/v1/observability/alerts/channels           # 通知渠道
POST   /api/v1/observability/alerts/channels           # 创建渠道
PUT    /api/v1/observability/alerts/channels/:id       # 更新
DELETE /api/v1/observability/alerts/channels/:id       # 删除
POST   /api/v1/observability/alerts/channels/:id/test  # 测试通知
```

### Logs API（Sprint 2）

```
GET    /api/v1/observability/logs/sources              # 列表日志源
POST   /api/v1/observability/logs/sources              # 创建日志源
POST   /api/v1/observability/logs/query                # 日志查询
WS     /api/v1/observability/logs/stream               # 日志流
```

## 开发进度

### Sprint 1 (Week 1-4)

- [x] Week 1: 数据模型设计 ✅
- [ ] Week 1: 模块骨架和路由注册
- [ ] Week 2: Prometheus 客户端集成
- [ ] Week 3: 告警规则管理
- [ ] Week 4: 告警通知

### Sprint 2 (Week 5-8)

- [ ] Week 5-6: Loki 集成
- [ ] Week 7: CMDB 边界清理
- [ ] Week 8: 凭据管理

### Sprint 3 (Week 9-10)

- [ ] Week 9: Dashboard
- [ ] Week 10: 变更管理

## 技术选型

| 组件 | 选型 | 版本 |
|---|---|---|
| Metrics Backend | Prometheus | 2.45+ |
| Logs Backend | Loki | 2.9+ |
| Alerting | Prometheus Alertmanager | 0.26+ |
| Dashboard | Grafana | 10.0+ |
| 采集器（主机） | node_exporter | 1.6+ |
| 采集器（K8s） | kube-state-metrics | 2.10+ |
| 日志采集 | Promtail | 2.9+ |

## 参考资料

- [Prometheus HTTP API](https://prometheus.io/docs/prometheus/latest/querying/api/)
- [PromQL 语法](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Alertmanager 配置](https://prometheus.io/docs/alerting/latest/configuration/)
- [Loki HTTP API](https://grafana.com/docs/loki/latest/api/)
- [LogQL 语法](https://grafana.com/docs/loki/latest/logql/)

## 相关文档

- [V2_DEVELOPMENT_PLAN.md](../../../../docs/V2_DEVELOPMENT_PLAN.md)
- [BUSINESS_OBSERVABILITY_MODULE_DESIGN.md](../../../../docs/designs/BUSINESS_OBSERVABILITY_MODULE_DESIGN.md)（待创建）
