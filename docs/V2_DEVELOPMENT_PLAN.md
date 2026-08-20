# pantheon-ops V2 开发计划

**创建日期:** 2026-08-20  
**状态:** Active - Sprint 1 Ready to Start  
**基线:** V1 完成度 92%，Service 模块已实现  
**目标:** 补齐可观测性能力，成为真正的 SRE 平台  
**预计周期:** 10 周（2 个月）

## 一、V2 核心目标

### 1.1 主要目标

让 pantheon-ops 从"资产管理 + 部署工具"升级为**具备完整可观测性的 SRE 平台**。

**三大支柱:**
1. ✅ **Metrics（指标）** - 实时监控主机、容器、应用性能
2. ✅ **Logs（日志）** - 聚合、检索、分析应用日志
3. ✅ **Alerting（告警）** - 规则配置、通知路由、降噪

### 1.2 不包含的内容（留给 V3）

- ❌ Tracing 链路追踪（OpenTelemetry）
- ❌ SLO/SLI/Error Budget 体系
- ❌ Incident Management（事件管理、值班）
- ❌ GitOps（ArgoCD/Flux 集成）
- ❌ 灰度发布、金丝雀部署
- ❌ 多租户支持

---

## 二、V2 开发路线图

### Sprint 1: Observability 基础（Week 1-4）

**目标:** 集成 Prometheus，实现基础指标采集和告警

#### Week 1: Observability 模块骨架
- [ ] 创建 `backend/modules/business/observability/` 模块
- [ ] 设计 Observability 数据模型：
  - `MetricSource` - 指标源（Prometheus/VictoriaMetrics）
  - `AlertRule` - 告警规则
  - `AlertRecord` - 告警历史
  - `NotificationChannel` - 通知渠道
- [ ] 设计 API 契约（RESTful）
- [ ] 前端模块骨架 `frontend/src/modules/business/observability/`

#### Week 2: Prometheus 集成
- [ ] 实现 Prometheus HTTP API 客户端
- [ ] 实现指标查询（PromQL）
- [ ] 实现 Range Query / Instant Query
- [ ] 实现 Label Values 查询
- [ ] 集成测试（本地 Prometheus）

#### Week 3: 告警规则管理
- [ ] 实现告警规则 CRUD API
- [ ] 告警规则验证（PromQL 语法检查）
- [ ] 告警规则同步到 Prometheus（通过配置文件或 API）
- [ ] 告警历史记录
- [ ] 前端告警规则列表、创建、编辑页面

#### Week 4: 告警通知
- [ ] 实现邮件通知渠道
- [ ] 实现钉钉 Webhook 通知
- [ ] 实现企业微信 Webhook 通知（可选）
- [ ] 告警通知路由（按业务域、环境、标签）
- [ ] 告警静默和抑制
- [ ] 前端通知渠道管理页面

**Sprint 1 交付物:**
- ✅ Prometheus 集成可用
- ✅ 至少 10 条预置告警规则（CPU/内存/磁盘/网络/K8s）
- ✅ 邮件和钉钉通知可用
- ✅ 前端可视化配置告警规则

---

### Sprint 2: 日志聚合与 CMDB 边界清理（Week 5-8）

**目标:** 集成 Loki 或 Elasticsearch，清理 CMDB 边界问题

#### Week 5-6: 日志聚合
- [ ] 选型：Loki vs Elasticsearch（推荐 Loki，更轻量）
- [ ] 实现日志查询 API（LogQL 或 ES Query DSL）
- [ ] 实现日志流式查询（WebSocket）
- [ ] 实现日志上下文查询（Context Query）
- [ ] 前端日志检索页面（时间范围、关键词、标签过滤）
- [ ] 日志采集方案文档（Promtail/Vector/Fluent Bit）

#### Week 7: CMDB 边界清理（P1）
- [ ] 检查 Deploy 模块是否回写 `InstalledComponents`
- [ ] 如果是，迁移逻辑到 ServiceInstance
- [ ] 标记 `Host.InstalledComponents` 为 deprecated
- [ ] 简化 `Host.Status` 字段语义
- [ ] 更新文档和 API 说明
- [ ] Migration 脚本（如果需要数据迁移）

#### Week 8: 凭据管理增强（P1）
- [ ] 设计凭据管理方案：
  - 方案 A: 集成 HashiCorp Vault
  - 方案 B: 自建加密凭据存储（AES-256-GCM）
- [ ] 实现 SSH 凭据托管
- [ ] 实现 K8s kubeconfig 外部化
- [ ] 凭据引用替换明文存储
- [ ] 凭据审计日志

**Sprint 2 交付物:**
- ✅ Loki 集成可用，可检索应用日志
- ✅ CMDB 边界清理完成
- ✅ 凭据管理方案落地

---

### Sprint 3: Dashboard 与变更管理（Week 9-10）

**目标:** 提供可视化看板，补充变更管理能力

#### Week 9: 轻量 Dashboard
- [ ] 方案选择：
  - 方案 A: 集成 Grafana（iframe 嵌入）
  - 方案 B: 自建轻量看板（基于 ECharts/Recharts）
- [ ] 主机监控看板（CPU/内存/磁盘/网络）
- [ ] K8s 监控看板（Pod/Node/资源使用）
- [ ] 业务域监控看板（按 BizScope 聚合）
- [ ] 告警趋势看板

#### Week 10: 变更管理基础
- [ ] Deploy Rollback 能力
  - 版本历史记录
  - 一键回滚到上一版本
  - 回滚失败保护
- [ ] 变更窗口定义（时间段、环境、业务域）
- [ ] 变更审批流（简单的单级审批）
- [ ] 变更日历（可视化时间轴）

**Sprint 3 交付物:**
- ✅ 可视化监控看板
- ✅ Deploy Rollback 能力
- ✅ 变更窗口和审批流

---

## 三、技术选型

### 3.1 可观测性技术栈

| 组件 | 选型 | 理由 |
|---|---|---|
| **Metrics** | Prometheus | 事实标准，生态完善，PromQL 强大 |
| **Logs** | Loki | 轻量、与 Prometheus 一致的标签模型 |
| **Alerting** | Prometheus Alertmanager | 成熟的告警路由和降噪能力 |
| **Dashboard** | Grafana（嵌入） | 事实标准，开箱即用 |
| **采集器（主机）** | node_exporter | Prometheus 官方，覆盖全面 |
| **采集器（K8s）** | kube-state-metrics | Kubernetes 状态指标 |
| **日志采集** | Promtail | Loki 官方，配置简单 |

### 3.2 为什么不自建？

**坚持"集成优于自建"原则:**
- ✅ 时序数据库：用 Prometheus/VictoriaMetrics，不自建
- ✅ 日志存储：用 Loki/Elasticsearch，不自建
- ✅ 告警引擎：用 Alertmanager，不自建
- ✅ 看板：用 Grafana，不自建

**pantheon-ops 的价值在于:**
- 统一认证和权限
- 与 CMDB/BizScope/Service 深度集成
- 提供面向业务域的可观测性视图
- 简化配置和使用门槛

---

## 四、数据模型设计

### 4.1 Observability 核心模型

```go
// MetricSource 指标源
type MetricSource struct {
    ID              uint64
    Name            string         // "生产环境 Prometheus"
    Type            string         // "prometheus" | "victoria-metrics"
    Endpoint        string         // "http://prometheus.prod.svc:9090"
    CredentialRef   string         // 凭据引用
    BusinessScopeID uint64         // 关联业务域
    Status          string         // "active" | "inactive"
}

// AlertRule 告警规则
type AlertRule struct {
    ID              uint64
    Name            string
    MetricSourceID  uint64
    BusinessScopeID uint64         // 关联业务域
    Environment     string         // "prod" | "test" | "dev"
    PromQL          string         // 'up{job="node-exporter"} == 0'
    Duration        string         // "5m"
    Severity        string         // "critical" | "warning" | "info"
    Labels          JSON           // {"team": "ops", "service": "web"}
    Annotations     JSON           // {"summary": "主机宕机", "description": "..."}
    NotificationChannelIDs []uint64
    Status          string         // "enabled" | "disabled"
}

// AlertRecord 告警历史
type AlertRecord struct {
    ID              uint64
    AlertRuleID     uint64
    FiredAt         time.Time
    ResolvedAt      *time.Time
    Severity        string
    Labels          JSON
    Annotations     JSON
    NotificationsSent JSON         // 已发送的通知记录
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
    ID              uint64
    Name            string
    Type            string         // "email" | "dingtalk" | "wechat" | "slack"
    Config          JSON           // 不同类型的配置
    BusinessScopeID uint64
    Status          string
}

// LogSource 日志源
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

### 4.2 与 ServiceInstance 关联

```go
// ServiceInstance 扩展字段（已存在）
type ServiceInstance struct {
    // ... 已有字段 ...
    
    // 可观测性标签（用于 Prometheus/Loki 查询）
    ObservabilityLabels JSON `gorm:"type:json" json:"observabilityLabels"`
    // 例如: {"service": "order-api", "env": "prod", "instance": "10.0.1.5:8080"}
}
```

---

## 五、API 设计

### 5.1 Metrics API

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

### 5.2 Alerting API

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

### 5.3 Logs API

```
GET    /api/v1/observability/logs/sources              # 列表日志源
POST   /api/v1/observability/logs/sources              # 创建日志源
GET    /api/v1/observability/logs/sources/:id          # 详情
PUT    /api/v1/observability/logs/sources/:id          # 更新
DELETE /api/v1/observability/logs/sources/:id          # 删除

POST   /api/v1/observability/logs/query                # 日志查询
WS     /api/v1/observability/logs/stream               # 日志流（WebSocket）
POST   /api/v1/observability/logs/context              # 上下文查询
```

---

## 六、前端页面设计

### 6.1 菜单结构

```
运维平台
├─ CMDB
│   ├─ 主机管理
│   ├─ 分组管理
│   └─ 标签管理
├─ 业务域
│   └─ 业务域管理
├─ 服务
│   ├─ 应用管理
│   ├─ 服务管理
│   └─ 服务实例
├─ 部署
│   ├─ 部署包
│   └─ 部署任务
├─ Kubernetes
│   ├─ 集群管理
│   ├─ Workload
│   ├─ ConfigMap
│   └─ Secret
└─ 可观测性 ⬅️ 新增
    ├─ 监控看板
    ├─ 告警规则
    ├─ 告警历史
    ├─ 日志检索
    ├─ 指标查询
    └─ 通知渠道
```

### 6.2 核心页面

#### 监控看板
- 主机监控：CPU/内存/磁盘/网络时序图
- K8s 监控：Pod/Node/资源使用
- 业务域监控：按 BizScope 聚合
- 自定义看板：拖拽式图表配置（V3）

#### 告警规则
- 列表：规则名称、业务域、PromQL、状态、最近触发时间
- 创建/编辑：表单 + PromQL 编辑器（Monaco Editor）
- PromQL 验证：实时语法检查和测试查询
- 预置模板：主机/K8s/应用常见告警规则

#### 告警历史
- 列表：告警名称、严重级别、触发时间、恢复时间、持续时长
- 详情：Label/Annotation 展开、通知记录
- 过滤：按时间范围、严重级别、业务域、服务

#### 日志检索
- 查询框：关键词 + 时间范围 + 标签过滤
- 结果列表：时间戳、日志内容、高亮关键词
- 上下文：点击某条日志，展开前后 N 行
- 实时流：WebSocket 实时推送新日志

---

## 七、验收标准

### Sprint 1 验收标准

✅ **Metrics:**
- 能连接到 Prometheus 并查询指标
- 能创建告警规则并同步到 Prometheus
- 能收到邮件和钉钉告警通知
- 预置至少 10 条告警规则生效

✅ **前端:**
- 告警规则列表、创建、编辑、删除可用
- 告警历史可查询和过滤
- 通知渠道可配置和测试

✅ **文档:**
- Observability 模块设计文档
- Prometheus 集成文档
- 告警规则配置指南

### Sprint 2 验收标准

✅ **Logs:**
- 能连接到 Loki 并查询日志
- 支持全文检索和标签过滤
- 支持日志上下文查询
- 前端日志检索页面可用

✅ **CMDB 边界:**
- Deploy 不再回写 CMDB `InstalledComponents`
- `Host.InstalledComponents` 标记为 deprecated
- 相关文档更新

✅ **凭据管理:**
- SSH/K8s 凭据托管可用
- 明文凭据已迁移
- 凭据审计日志可查询

### Sprint 3 验收标准

✅ **Dashboard:**
- 至少 3 个预置看板可用
- 看板可按业务域、环境过滤
- 看板可自定义时间范围

✅ **变更管理:**
- Deploy 任务可一键回滚
- 变更窗口可定义和查询
- 变更审批流可提交和审批

---

## 八、风险与缓解

### 风险 1: Prometheus 集成复杂度

**风险:** Prometheus HTTP API 有限，部分功能需直接操作配置文件

**缓解:**
- 优先使用 HTTP API（查询、Label）
- 告警规则通过配置文件挂载（ConfigMap）
- V3 考虑集成 Thanos 或 VictoriaMetrics（更好的 API）

### 风险 2: 日志量大导致性能问题

**风险:** 日志检索慢，ES/Loki 资源占用高

**缓解:**
- 设置日志保留策略（默认 7 天）
- 限制单次查询结果数量（默认 1000 条）
- 推荐使用标签过滤而非全文检索
- V3 引入日志采样和压缩

### 风险 3: 告警风暴

**风险:** 大量告警同时触发，通知渠道被打爆

**缓解:**
- 使用 Alertmanager 的 grouping 和 inhibition
- 实现告警静默功能
- 限制单个渠道的发送频率
- V3 引入智能告警降噪

---

## 九、后续 V3 规划

**V3 目标:** SRE 完整实践

1. **Tracing（链路追踪）**
   - OpenTelemetry 集成
   - Jaeger/Tempo 后端
   - 分布式调用链可视化

2. **SLO/SLI 体系**
   - 服务质量目标定义
   - SLI 自动计算
   - Error Budget 追踪
   - SLO 报表

3. **Incident Management**
   - 事件创建/分级/升级
   - On-call 值班排班
   - Postmortem 模板

4. **GitOps**
   - ArgoCD/Flux 集成
   - Git → K8s 自动同步
   - 配置即代码

5. **高级部署策略**
   - 金丝雀发布
   - 蓝绿部署
   - 流量分流

---

## 十、资源需求

### 开发资源
- 后端开发：1-2 人
- 前端开发：1 人
- QA 测试：0.5 人（兼职）
- SRE 顾问：0.5 人（兼职，Prometheus/Loki 配置）

### 基础设施
- Prometheus 服务器：2C4G（测试环境）
- Loki 服务器：2C4G（测试环境）
- Grafana 服务器：1C2G（可选，或使用 Grafana Cloud）
- 存储：至少 100GB（指标和日志保留 7 天）

---

**下一步:** 立即启动 Sprint 1 Week 1，创建 Observability 模块骨架。
