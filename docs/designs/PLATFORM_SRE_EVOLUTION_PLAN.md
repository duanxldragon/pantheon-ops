# Pantheon-Ops SRE 演进计划

更新时间：2026-05-11

类型：Design (Roadmap)
归属层：platform
状态：Active
作者：duanxldragon

本文定义 pantheon-ops 从 Web 运维管理平台向 K8s-native SRE 平台演进的完整路线图。
基于当前代码基线（2026-05-11）的实际能力缺口制定，与 `BUSINESS_DEPLOY_MODULE_DESIGN.md` §1
中预留的 Agent/SSH 执行方向衔接。

---

## 1. 当前代码基线分析

### 1.1 已有能力（做对了的事）

| 能力 | 代码位置 | 成熟度 |
|---|---|---|
| CMDB 主机/分组/标签管理 | `backend/modules/business/cmdb/` | 生产可用 |
| 部署任务编排（状态机、目标解析、标签条件引擎） | `backend/modules/business/deploy/` | 编排层可用 |
| 4 层权限模型 + Casbin | `backend/pkg/database/casbin.go` | 成熟 |
| 低代码模块生成器 | `backend/modules/system/generator/` | 可扩展 |
| 动态模块生命周期 | `backend/modules/system/dynamicmodule/` | 可扩展 |
| DSL 条件表达式引擎（AND/OR + eq/neq/in/notIn） | `deploy_service.go:583-642` | 可用，后续迁至 K8s label selector |

### 1.2 能力缺口

| 缺口 | 当前状态 | 目标 |
|---|---|---|
| 真实执行引擎 | `StartTask` 只改状态，未真正执行 | SSH / Agent / K8s Job 三模式执行 |
| Agent 通信 | `ExecutorTypeAgent` 枚举已定义，无实现 | Agent Sidecar 拉取任务并回写结果 |
| K8s 原生集成 | 无 `client-go` 依赖，无 kubeconfig | Operator + CRD + Webhook |
| 可观测性 | 无 metrics 端点，无结构化日志 | Prometheus exporter + Loki + 链路追踪 |
| 容器化部署 | 仅 docker-compose 用于开发 | Dockerfile + Helm Chart + 一键部署 |
| CI/CD | 无 | 代码提交 → 镜像构建 → Helm 部署 |

---

## 2. 五阶段演进路线

### 阶段一：真实执行引擎（SSH + Agent 通信）

**目标**：`POST /tasks/:id/start` 不再只改状态，而是真正在远程机器上执行命令。

**对应能力缺口**：真实执行引擎、Agent 通信基础

**改动范围**：

| 步骤 | 文件/模块 | 工作内容 | 习得能力 |
|---|---|---|---|
| 1.1 | `backend/pkg/executor/ssh_executor.go` | 新建 SSH 执行器：从 CMDB host 读取 IP/SSH 端口，建立连接，执行 `InstallCommand`，返回 stdout/stderr | Go SSH 编程、连接池、超时控制 |
| 1.2 | `backend/pkg/executor/agent_executor.go` | 新建 Agent 执行器：通过 HTTP POST 向 Agent 下发任务 | HTTP client 设计、重试、幂等 |
| 1.3 | `backend/modules/business/deploy/deploy_service.go` | 改造 `StartTask`：根据 `executor_type` 选择执行器，goroutine 并发执行多主机，结果自动回写 `DeployTaskHost` | 并发模型、`context.Context` 取消、`sync.WaitGroup` |
| 1.4 | `backend/modules/business/deploy/deploy_service.go` | `MarkHostResult` 增加来源区分（人工标记 vs 执行器上报） | 审计完整性 |

**关键 Go 知识点（边写边查）**：
- `golang.org/x/crypto/ssh` —— SSH 客户端
- `context.WithTimeout` —— 执行超时
- `sync.WaitGroup` —— 并发等待

**验收标准**：
- 创建任务 → 启动 → SSH 连接到目标主机执行命令 → 结果自动回写
- 多台主机并发执行，单台失败不影响其他
- 在 `biz_deploy_task_host` 中正确记录 stdout/stderr/error_message

---

### 阶段二：Agent Sidecar 开发

**目标**：写一个独立的 Go 服务（Agent），部署在 K8s 节点上，接收任务并执行，作为 K8s 胶水程序的第一份作品。

**对应能力缺口**：Agent 通信完整闭环、可观测性起步、容器化

**新建项目**：`pantheon-agent/`（独立 Go module，不在 pantheon-ops 仓库内）

**项目结构**：
```
pantheon-agent/
├── cmd/agent/main.go          # 入口
├── internal/
│   ├── executor/executor.go   # 命令执行
│   ├── reporter/reporter.go   # 结果上报到 pantheon-ops
│   └── metrics/metrics.go     # Prometheus 指标
├── Dockerfile                  # 多阶段构建
└── go.mod
```

**改动范围**：

| 步骤 | 工作内容 | 习得能力 |
|---|---|---|
| 2.1 | 新建 Agent 项目，HTTP long-poll 从 pantheon-ops 拉取待执行任务 | 独立服务设计、HTTP client、JSON 序列化 |
| 2.2 | 在 Agent 内执行 shell 命令，采集 stdout/stderr/exit_code | Go `os/exec`、进程管理 |
| 2.3 | 实时回传日志到 pantheon-ops（SSE 或 chunked 上报） | SSE / streaming HTTP |
| 2.4 | Agent 上报心跳，pantheon-ops 新增心跳接收端点 | 心跳协议、健康检查 |
| 2.5 | 给 Agent 加 `/metrics` 端点（任务执行次数、耗时直方图、成功率） | `prometheus/client_golang`、Counter/Gauge/Histogram |
| 2.6 | 写 Dockerfile（多阶段构建），打镜像 | Docker 镜像优化、scratch/alpine 选择 |
| 2.7 | 以 DaemonSet 方式部署到 K8s 集群 | DaemonSet、hostNetwork/hostPID、RBAC |

**验收标准**：
- Agent 以 DaemonSet 在每个 K8s 节点运行
- pantheon-ops 创建 Agent 类型任务 → 启动 → Agent 拉取 → 执行 → 结果自动回写
- Agent `/metrics` 端点可被 Prometheus 抓取
- Agent 离线 30s 后，pantheon-ops 将该节点的运行中任务标记为失败

---

### 阶段三：K8s Operator 开发

**目标**：用 kubebuilder 写 Operator，把 CMDB 主机管理和部署任务原生接入 K8s API。

**对应能力缺口**：K8s 原生集成、CRD/Operator/Webhook

**新建项目**：`pantheon-operator/`（kubebuilder 脚手架生成）

**项目结构**：
```
pantheon-operator/
├── PROJECT                           # kubebuilder 元数据
├── api/v1/
│   ├── host_types.go                 # Host CRD（对标 cmdb host_model.go）
│   ├── deploytask_types.go           # DeployTask CRD
│   └── groupversion_info.go
├── internal/controller/
│   ├── host_controller.go            # Host Reconcile
│   └── deploytask_controller.go      # DeployTask Reconcile
├── internal/webhook/
│   ├── host_validating_webhook.go
│   └── host_mutating_webhook.go
├── config/                           # Kustomize + RBAC + Webhook 配置
│   ├── crd/
│   ├── rbac/
│   └── webhook/
├── charts/pantheon-operator/         # Helm Chart
│   ├── templates/
│   └── values.yaml
├── Dockerfile
└── Makefile
```

**你已有代码 → K8s 概念的对照映射**：

| pantheon-ops 现有代码 | K8s 对应概念 | 衔接方式 |
|---|---|---|
| `host.Host` struct (GORM model) | `HostSpec` + `HostStatus` in CRD | 字段直接映射，去掉 GORM tag，换 json tag |
| `host_handler.go` 的 CRUD | Reconcile 循环 | HTTP 被动响应 → Controller 主动调和 |
| `host_service.go` 的校验逻辑 | Validating Webhook | 搬校验到 webhook |
| `deploy_service.go` 的状态流转 | `DeployTask` CRD Status conditions | 状态机逻辑保留，存入 K8s Status |
| `groupMatchesHost()` DSL | K8s `matchLabels` / `matchExpressions` | DSL 替换为标准 label selector |

**改动范围**：

| 步骤 | 工作内容 | 习得能力 |
|---|---|---|
| 3.1 | 用 kubebuilder 初始化项目，定义 `Host` CRD（hostname, ip, sshPort, os, osVersion, cpuCores, memoryGb, diskGb, labelValues, status） | kubebuilder 脚手架、CRD spec/status 设计 |
| 3.2 | 写 Host Controller Reconcile：创建 Host CR → 自动同步到 pantheon-ops CMDB 数据库 | Reconcile 模式、幂等设计、OwnerReference |
| 3.3 | 定义 `DeployTask` CRD（对标 deploy model），Reconcile 在 K8s 内创建 Job 执行 | Job 编排、K8s API 操作（client-go） |
| 3.4 | 写 Validating Webhook：校验 Host IP 格式、DeployTask 必须引用存在的 Package | Admission Webhook、cert-manager TLS 证书 |
| 3.5 | 写 Mutating Webhook：Host 创建时自动注入默认 sshPort=22、默认 status=pending | 默认值注入 |
| 3.6 | Host Controller 增加定期探活：TCP ping 主机 IP:SSH 端口，更新 Status.Conditions | Conditions 设计模式、探活逻辑 |
| 3.7 | 打包 Helm Chart：Operator + CRD + RBAC + ServiceMonitor | Helm 模板、Chart 结构 |

**关键 Go 依赖（首次引入）**：
- `sigs.k8s.io/controller-runtime` —— Reconcile 框架核心
- `sigs.k8s.io/kubebuilder` —— 代码生成
- `k8s.io/client-go` —— K8s API 客户端
- `k8s.io/apimachinery` —— API 类型基础库

**验收标准**：
- `kubectl apply -f host.yaml` 创建 Host CR → 数据自动出现在 pantheon-ops CMDB 页面
- `kubectl get hosts` 可查看所有纳管主机及状态
- 非法 Host YAML（IP 格式错误）被 webhook 拒绝
- `helm install pantheon-operator ./charts/pantheon-operator` 一键部署

---

### 阶段四：可观测性体系

**目标**：pantheon-ops + Agent + Operator 全链路可观测（指标、日志、告警）。

**对应能力缺口**：可观测性从零搭建

**改动范围**：

| 步骤 | 工作内容 | 习得能力 |
|---|---|---|
| 4.1 | 给 pantheon-ops 后端加 `/metrics` 端点，埋入 API QPS、延迟 P50/P99、错误率 | Prometheus 指标类型选型（Counter/Gauge/Histogram）、`promhttp` |
| 4.2 | 给 Agent 加 `/metrics`（在阶段二已完成） | — |
| 4.3 | 部署 Prometheus + Grafana，配置 ServiceMonitor 抓取 pantheon-ops 和 Agent | Prometheus Operator、ServiceMonitor、Grafana Dashboard |
| 4.4 | 创建 3 张核心仪表盘：服务健康总览（QPS/Latency/Error）、Agent 执行大盘（任务量/耗时/成功率）、CMDB 资源大盘 | Grafana Dashboard JSON model、PromQL |
| 4.5 | 部署 Loki + Promtail，收集结构化日志 | Loki LogQL、Promtail pipeline |
| 4.6 | 配置 Alertmanager 规则：任务失败率 > 10%、Agent 离线 > 5 分钟、API 5xx > 1% | 告警规则、分组、静默、路由 |
| 4.7 | 在 pantheon-ops 通知模块中接入 Alertmanager webhook | Webhook 集成 |

**Prometheus 指标设计**：

| 指标名 | 类型 | 标签 | 说明 |
|---|---|---|---|
| `pantheon_api_requests_total` | Counter | method, path, status | API 总请求数 |
| `pantheon_api_request_duration_seconds` | Histogram | method, path | API 延迟分布 |
| `pantheon_deploy_tasks_total` | Counter | executor_type, status | 部署任务计数 |
| `pantheon_agent_heartbeat_timestamp` | Gauge | agent_id, hostname | Agent 最后心跳时间 |
| `pantheon_agent_task_duration_seconds` | Histogram | agent_id, status | Agent 任务执行耗时 |

**验收标准**：
- Grafana 仪表盘能展示实时 API QPS 和延迟
- 部署任务失败时，钉钉/企微/邮件至少一种通知到人
- Loki 中可检索到任意一次任务执行的详细日志

---

### 阶段五：打包与一键部署

**目标**：全栈可一键部署到裸 K8s 集群。

**改动范围**：

| 步骤 | 工作内容 |
|---|---|
| 5.1 | 给 pantheon-ops 写 Dockerfile（多阶段构建，Go build → alpine runtime） |
| 5.2 | 写顶层 Helm Chart `pantheon-platform/`：包含 MySQL + Redis + pantheon-ops + pantheon-agent + pantheon-operator + Prometheus stack |
| 5.3 | 写 `deploy.sh`：kubeadm/二进制装 K8s → kubectl apply CRD → helm install 全栈 |
| 5.4 | 写团队操作手册（1-2 页）：集群要求、部署命令、常见排障 |

**验收标准**：
- 从空 K8s 集群到全栈运行 ≤ 30 分钟
- 新团队成员按照手册能独立完成部署

---

## 3. 技能矩阵

| 技能维度 | 当前水平 | 阶段一后 | 阶段三后 | 阶段五后 |
|---|---|---|---|---|
| Go 编码 | 会用框架（GORM + Gin） | 能写独立 package | 能写完整 Operator | 独立的 Go 项目 |
| K8s 运维 | 安装和日常运维 | — | 会写 CRD/Operator/Webhook | 能设计 K8s 原生架构 |
| 网络编程 | 基本 HTTP | SSH + HTTP client | gRPC + K8s API | — |
| 可观测性 | 会用 Prometheus | — | 会写 exporter | 会设计全栈可观测方案 |
| Docker/Helm | docker-compose 开发 | 会写 Dockerfile | 会写 Helm Chart | 一键部署脚本 |
| CI/CD | 无 | — | — | Git → Image → Deploy 全链路 |

---

## 4. 预期时间线

| 阶段 | 预估周数 | 启动条件 |
|---|---|---|
| 阶段一：执行引擎 | 3-4 周 | 立即 |
| 阶段二：Agent Sidecar | 4-5 周 | 阶段一完成 |
| 阶段三：K8s Operator | 5-6 周 | 阶段二完成 |
| 阶段四：可观测性 | 3-4 周 | 阶段二完成（与阶段三可并行） |
| 阶段五：打包部署 | 2 周 | 阶段三+四完成 |

总计：每日 2 小时 → 约 5-6 个月完成全路线。

---

## 5. 立即启动（阶段一第一步）

**目标**：100 行 Go 代码打通 SSH 远程执行。

**新建文件**：`backend/pkg/executor/ssh_executor.go`

**核心逻辑**：
```go
package executor

// SSHExecutor 连接远程主机执行命令，返回 stdout/stderr/exitCode
// 依赖 golang.org/x/crypto/ssh
// 配置来源：CMDB Host（IP + SSHPort）+ 凭据（第一阶段可硬编码测试）
```

**这 100 行是从 "Web CRUD 开发者" 到 "SRE 工程师" 的质变点。**

---

## 6. 引用规范

- 基座架构：`../../pantheon-base/DESIGN.md`
- 部署模块设计：[BUSINESS_DEPLOY_MODULE_DESIGN.md](./BUSINESS_DEPLOY_MODULE_DESIGN.md)
- CMDB 模块设计：[BUSINESS_CMDB_MODULE_DESIGN.md](./BUSINESS_CMDB_MODULE_DESIGN.md)
- 权限模型：`../../pantheon-base/docs/designs/PERMISSION_MODEL.md`
- 模块契约：`../../pantheon-base/docs/designs/MODULE_CONTRACT.md`
