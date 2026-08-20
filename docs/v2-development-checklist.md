# Pantheon-Ops V2 开发检查清单

## ✅ Week 1 已完成 (2026-08-20)

### ServiceInstance 抽象层 (P0-1)
- [x] Service 数据模型定义
- [x] ServiceInstance 数据模型定义
- [x] TargetRef 多态关联设计
- [x] Repository 数据访问层
- [x] Service 业务逻辑层
- [x] Handler HTTP 处理层
- [x] RESTful 路由注册
- [x] 前端 API 客户端
- [x] 前端服务列表页面
- [x] 前端服务实例列表页面
- [x] 数据库 Schema
- [x] 模块文档 (README + IMPLEMENTATION)

### Observability 模块 (P0-3)
- [x] MetricSource 数据模型
- [x] AlertRule 数据模型
- [x] AlertRecord 数据模型
- [x] NotificationChannel 数据模型
- [x] Repository 数据访问层
- [x] Service 业务逻辑层
- [x] Handler HTTP 处理层 (24 个端点)
- [x] RESTful 路由注册
- [x] 前端 API 客户端
- [x] 前端指标源列表
- [x] 前端告警规则列表
- [x] 前端告警历史列表
- [x] 前端活跃告警列表 (30秒自动刷新)
- [x] 前端通知渠道列表
- [x] 数据库 Schema
- [x] 模块文档 (README + IMPLEMENTATION)

---

## 🔄 待集成 (Week 1 后续)

### 后端集成
- [ ] Service 模块初始化到主程序
- [ ] Observability 模块初始化到主程序
- [ ] Service 路由注册到主路由
- [ ] Observability 路由注册到主路由
- [ ] 数据库表初始化 (执行 schema.sql)
- [ ] 验证 API 端点可访问性

### 前端集成
- [ ] Service 路由合并到主路由
- [ ] Observability 路由合并到主路由
- [ ] 菜单配置添加"服务管理"
- [ ] 菜单配置添加"可观测性"
- [ ] 国际化配置 (中英文)
- [ ] 权限配置 (菜单权限/数据权限)

---

## 📋 Week 2 计划

### Service 模块增强
- [ ] ServiceInstance 健康状态聚合 API
- [ ] 服务拓扑图数据结构设计
- [ ] 服务拓扑图可视化组件 (使用 G6 或 Cytoscape.js)
- [ ] Deploy 模块回写 ServiceInstance 状态
- [ ] K8s Release 关联 ServiceInstance
- [ ] 服务实例详情页面

### Observability 模块增强
- [ ] 指标源创建表单弹窗
- [ ] 指标源编辑表单弹窗
- [ ] 指标源连接测试功能
- [ ] Prometheus API 客户端封装
- [ ] Prometheus 指标查询接口
- [ ] 指标数据展示图表
- [ ] VictoriaMetrics 适配器
- [ ] PromQL 编辑器 (Monaco Editor)

### Deploy 模块改造
- [ ] DeployTask 添加 ServiceInstanceID 字段
- [ ] 部署成功后回写 ServiceInstance 状态
- [ ] 回滚能力实现 (备份 + 恢复)
- [ ] 批量部署策略设计 (金丝雀/蓝绿/滚动)
- [ ] 批量部署策略实现
- [ ] 部署前预检 (磁盘/端口/依赖)
- [ ] 部署后健康检查

### CMDB/Deploy 边界收敛
- [ ] installed_components 从 Host 表迁移到 ServiceInstance
- [ ] Deploy 模块不再直写 Host.status
- [ ] 状态管理统一到 ServiceInstance
- [ ] 历史数据迁移脚本

---

## 📋 Week 3-4 计划

### Observability 深化
- [ ] 告警规则创建表单
- [ ] 告警规则编辑表单
- [ ] PromQL 验证逻辑 (调用 Prometheus API)
- [ ] 告警规则测试功能
- [ ] 通知渠道创建表单
- [ ] 通知渠道编辑表单
- [ ] 邮件通知集成 (SMTP)
- [ ] 钉钉 Webhook 集成
- [ ] 企业微信 Webhook 集成
- [ ] Slack Webhook 集成
- [ ] 通知模板管理

### 告警引擎 (Week 5-6)
- [ ] 告警评估调度器
- [ ] PromQL 执行器
- [ ] 告警触发逻辑
- [ ] 告警恢复检测
- [ ] 通知发送调度器
- [ ] 通知防抖策略
- [ ] 告警风暴抑制
- [ ] 告警聚合规则

---

## 🎯 里程碑

### M1: V2 基础能力 (Week 1-2)
- [x] ServiceInstance 抽象层
- [x] Observability 基础架构
- [ ] 模块集成完成
- [ ] 烟雾测试通过

### M2: V2 核心能力 (Week 3-4)
- [ ] Prometheus 指标查询
- [ ] 告警规则配置
- [ ] 通知渠道集成
- [ ] Deploy 回滚能力

### M3: V2 完整闭环 (Week 5-6)
- [ ] 告警评估引擎
- [ ] 通知发送调度
- [ ] ServiceInstance 状态聚合
- [ ] 服务拓扑图

### M4: V2 生产就绪 (Week 7-8)
- [ ] 性能优化
- [ ] 安全加固
- [ ] 文档完善
- [ ] 用户验收测试

---

## 📊 进度追踪

| 模块 | 进度 | 状态 |
|---|---|---|
| Service | 80% | 🟡 待集成 |
| Observability | 40% | 🟡 开发中 |
| Deploy 改造 | 0% | ⚪ 未开始 |
| CMDB 边界 | 0% | ⚪ 未开始 |

---

## 🐛 已知问题

暂无

---

## 📝 备注

1. Week 1 所有代码均为直接实现,未调用 Codex
2. 遵循 pantheon-base 架构约定和代码风格
3. 完整的三层架构 (Repository → Service → Handler)
4. 前端使用 React + TypeScript + Arco Design
5. 后端使用 Go + Gin + GORM

---

最后更新: 2026-08-20
