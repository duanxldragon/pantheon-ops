# V2 Sprint 1 Week 1 进度报告

**日期:** 2026-08-20  
**Sprint:** V2 Sprint 1 - Observability 基础  
**Week:** Week 1 - Observability 模块骨架  
**状态:** ✅ 第一阶段完成

---

## 本周目标

- [x] 创建 `backend/modules/business/observability/` 模块
- [x] 设计 Observability 数据模型
- [ ] 设计 API 契约（RESTful）
- [ ] 注册到 business 模块
- [ ] 前端模块骨架
- [ ] 数据库 Migration

## 已完成工作

### 1. Observability 模块骨架 ✅

**创建文件:**
- `backend/modules/business/observability/model.go` - 数据模型定义
- `backend/modules/business/observability/repository.go` - 数据访问层
- `backend/modules/business/observability/service.go` - 业务逻辑层
- `backend/modules/business/observability/README.md` - 模块文档

### 2. 数据模型设计 ✅

**5 个核心模型:**

1. **MetricSource（指标源）**
   - 支持 Prometheus/VictoriaMetrics
   - 凭据引用外部化
   - 业务域隔离

2. **AlertRule（告警规则）**
   - PromQL 表达式
   - 多级严重性（critical/warning/info）
   - 环境标签（prod/test/dev）
   - 通知渠道绑定

3. **AlertRecord（告警历史）**
   - 触发时间 + 恢复时间
   - 通知发送记录
   - 支持查询活跃告警

4. **NotificationChannel（通知渠道）**
   - 支持多种类型（email/dingtalk/wechat/slack）
   - 配置 JSON 可扩展
   - 测试通知接口

5. **LogSource（日志源）**
   - 支持 Loki/Elasticsearch
   - 为 Sprint 2 预留

### 3. 三层架构实现 ✅

**Repository 层:**
- 完整的 CRUD 操作
- 分页查询（默认 20 条/页，最大 100 条）
- 过滤器支持（业务域/环境/状态）
- 软删除支持

**Service 层:**
- 参数验证
- 默认值设置
- 关联校验（MetricSource 存在性）
- TODO 标记（Week 3-4 补充）

**特性:**
- 多租户隔离（BusinessScopeID + DeptID）
- 审计字段（CreatedBy/UpdatedBy）
- 状态管理（active/inactive, enabled/disabled）

---

## 下一步工作（Week 1 后续）

### 剩余 Week 1 任务

- [ ] **实现 HTTP Handler**
  - MetricSource CRUD API
  - AlertRule CRUD API
  - NotificationChannel CRUD API
  - 统一错误处理

- [ ] **路由注册**
  - 创建 `router.go`
  - 注册到 `business.go`
  - 权限中间件集成

- [ ] **数据库 Migration**
  - 创建 5 张表的 Migration SQL
  - 添加索引（businessScopeId, status, environment）
  - 测试可重复执行

- [ ] **前端模块骨架**
  - `frontend/src/modules/business/observability/`
  - 菜单注册
  - 路由配置

---

## 技术债务

**标记为 TODO 的功能:**
1. `ValidatePromQL` - 调用 Prometheus API 验证 PromQL 语法（Week 3）
2. `TestNotificationChannel` - 发送测试通知（Week 4）
3. AlertRule 删除前检查依赖（Week 3）

**设计决策:**
- 凭据管理暂时使用 `CredentialRef` 字符串引用，V2 Sprint 2 Week 8 统一实现
- Prometheus 告警规则同步方式待定（配置文件 vs API），Week 2 调研

---

## 风险与问题

**无阻塞风险。**

**需要确认:**
- Prometheus 部署方式（用户已有 or 我们提供）
- 告警规则同步机制（Prometheus Operator or ConfigMap）

---

## 下周计划（Week 2）

**目标:** Prometheus HTTP API 客户端集成

**任务清单:**
1. 研究 Prometheus HTTP API 文档
2. 实现 Prometheus 客户端（query/query_range/labels）
3. 实现指标查询 API Handler
4. 集成测试（本地 Prometheus）
5. 前端指标查询页面

**预计产出:**
- `backend/modules/business/observability/prometheus/client.go`
- `POST /api/v1/observability/metrics/query` API 可用
- 前端能展示 Prometheus 指标查询结果

---

**提交记录:** `feat(observability): 创建 Observability 模块骨架 - V2 Sprint 1 Week 1`
