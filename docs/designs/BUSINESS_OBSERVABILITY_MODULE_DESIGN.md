# 可观测性业务模块设计（主机监控优先）

更新时间：2026-08-28
类型：Design
归属层：business/observability
状态：Draft（v1.1 起；当前仅平台自监控静态配置，属免费版价值兑现的关键缺口，见 `.harness/v1.2-docs/V1_FREEZE_REMEDIATION_DEV_PLAN.md` Phase 3-3）

本文定义 pantheon-ops 的可观测能力演进。2026-08-28 评审确认：`grafana/` 目录下 prometheus.yml 仅覆盖平台自身（localhost 静态目标），**无监控对象管理、无 PromQL 查询 API、无 CMDB→Prometheus 配置生成**，registry 引用的 observability 模块目录不存在。本文给出分阶段目标。

## 1. 定位与分阶段

| 阶段 | 版本 | 内容 | 商业归属 |
|---|---|---|---|
| Phase 0（现状） | v1.0 | 平台自监控：API/DB/Redis 指标 + 3 张 Grafana 面板 + 115 行告警规则 | 不随产品发布 |
| Phase 1 | v1.1 | 主机监控：CMDB 主机→抓取配置下发→基础指标页 | 免费版 |
| Phase 2 | v1.2 | 监控对象管理 + PromQL API + 告警规则管理 | 免费版基础 + 商业版高级 |

## 2. Phase 1：主机监控（免费版兑现，v1.1）

目标链路：

```
CMDB 主机（assigned 状态）
→ 一键启用监控（安装 node_exporter，复用 deploy 引擎 SSH 执行）
→ 配置生成（file_sd 服务发现，写入 prometheus 配置目录）
→ 自托管 Prometheus 抓取
→ 平台内基础指标页（CPU/内存/磁盘/网络）
```

设计要点：

1. **不代理 PromQL**：V1.1 只做固定指标卡片（up、cpu、mem、disk usage、load），避免实现完整查询引擎。
2. **file_sd 而非静态配置**：`prometheus/sd/hosts.json` 由后端按 CMDB 主机清单生成，增删主机即生效，不需要 reload Prometheus。
3. **主机状态闭环**：CMDB retire 时自动从 sd 文件移除；deploy 引擎装 node_exporter 用固定模板（复用 nginx/redis 的 fixed template 模式）。
4. **采集架构**：平台自带 Prometheus（docker compose 内），后续支持外接 Prometheus（只生成 sd 文件，用户自管抓取）。
5. 指标存储在 Prometheus，平台不建时序表——避免自建存储这个大坑。

## 3. Phase 2：监控对象与告警（v1.2）

1. **监控对象管理**：CMDB 资源（主机/K8s 集群）标记监控启用状态，统一生成抓取配置（主机 node_exporter、K8s kube-prometheus-stack 集成）。
2. **PromQL 查询 API**：后端代理 Prometheus HTTP API（带白名单与超时），前端指标页可自定义查询时间范围。
3. **告警规则管理**：基础阈值规则（CPU>80% 持续 5min 等）在平台内创建，下发为 PrometheusRule；告警事件经 Alertmanager webhook 回传平台，与业务域/主机关联展示。
4. **Grafana 集成**：外链模式（免登录 iframe 或直接跳转），不做面板管理。

## 4. 边界

- 不做：APM/链路追踪（明确排除）、日志（Loki 属后续独立设计）、自研时序存储。
- 与 `business/deploy` 的关系：node_exporter 安装是 deploy 引擎的一个固定模板，observability 模块只消费部署结果与 CMDB 状态，不内嵌 SSH 逻辑。
- 与 K8s 模块关系：集群维度监控（Pod/Node 指标）依赖集群纳管，v1.2 与 K8s 商业版绑定。

## 5. 验收标准

Phase 1（v1.1）：

1. CMDB 添加主机→启用监控→node_exporter 安装成功→5 分钟内指标页出数。
2. 主机 retire 后从 sd 文件移除，指标停止采集。
3. 断电/断网主机：指标页显示采集断开，不误报为资源删除。

Phase 2（v1.2）：自定义 PromQL 查询返回正确；告警规则下发后触发 webhook 事件在平台可见。
