---
title: Business Operational Workbench Design
doc_type: Design
layer: business/*
status: Draft
updated_at: 2026-08-31
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md
  - docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md
  - docs/designs/BUSINESS_K8S_MODULE_DESIGN.md
  - docs/designs/BUSINESS_OBSERVABILITY_MODULE_DESIGN.md
---

# 业务运维工作台设计

## 1. 目的与结论

Pantheon Ops 已有 CMDB 列表、部署任务详情、Pod 日志和可观测性查询入口，但关键流程仍由普通表单、表格、卡片和段落拼接，尚未形成“选择作用域 -> 预览变更 -> 执行 -> 定位失败 -> 恢复”的统一体验。

本文把这些业务流程补成任务导向工作台。共享 UI 原语由 Pantheon Base 先交付，Ops 只实现 `business/*` 的数据 adapter、状态机组合、权限、DataScope、审计和业务文案。

## 2. 当前证据与适用边界

### 2.1 已有基础

- `CmdbHostList` 已采用 `SearchToolbar`、`TableBatchActionBar`、`AppTable`，说明标准列表模式可继续复用。
- `DeployTaskDetail` 已展示任务状态与主机轨迹，但 stdout/stderr 仍是普通段落，失败定位与跨主机比较成本高。
- `PodLogViewer` 已有连接、重连、清空、1000 行上限和自动滚动，但缺少搜索、暂停、下载、全屏、级别/容器筛选与 stale 语义。
- `LogExplorer` 当前主要是数据源、查询字符串和表格，缺少时间范围、历史、结构化/原始视图、趋势和字段统计。
- Ops 尚未注册业务 Dashboard widgets。

现有 desktop/mobile 截图位于 `.harness/evidence/2026-08-31-ops-v1-2-production-readiness-round2/screenshots/` 和 `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/screenshots/`，只作为当前基线，不证明本文目标界面已实现。

### 2.2 外部参考

参考 BK Design 的业务选择器、IP 选择器、条件筛选、任务日志、测试步骤、日志检索、时间筛选、版本树和 Diff。外部参考用于验证交互模式，不改变 Pantheon 的 Base-first、Arco、i18n、权限和审计合同。

## 3. 业务体验主线

所有高风险运维操作尽量遵循同一条认知路径：

```text
业务/环境上下文
-> 目标选择与排除
-> 参数与版本
-> 变更差异和影响摘要
-> 权限/风险/前置检查
-> 确认执行
-> 目标与步骤进度
-> 日志和失败定位
-> 重试/跳过/回滚/转人工
-> 审计与结果回写
```

页面不强求统一视觉模板，但对象、上下文、状态、风险和恢复动作的语义必须一致。

## 4. Base 依赖与 Ops 所有权

| 能力 | 依赖 Base | Ops 实现 |
| --- | --- | --- |
| 目标选择 | `ContextSelector` 外壳 | BizScope、CMDB 拓扑/分组/主机、集群 adapter 和 DataScope |
| 执行详情 | `ExecutionStepRail`、`TaskLogViewer` | Deploy/K8s 状态机、SSE、重试/取消/回滚权限 |
| 变更预览 | `ChangeDiff` | Deploy/K8s/Argo CD 版本解析、影响对象、风险规则 |
| 日志查询 | `ConditionBuilder`、表格视图 | Loki 查询 API、字段目录、时间范围和历史 |
| Dashboard | 新 widget kinds | 业务 widget 注册、数据查询、permission 和 cleanup |

如果 Base 组件尚未通过 foundation release，Ops packet 保持 blocked；不得在 Ops 复制共享组件实现。

## 5. 统一上下文与目标选择

### 5.1 页面结构

- 顶部固定显示 BizScope、环境、集群/地域等当前上下文摘要。
- 左侧或移动端首层选择来源：拓扑、分组、主机、集群资源、手工输入。
- 中部候选区支持搜索、分页/虚拟化、状态和权限提示。
- 右侧或移动端下一步显示已选、排除、失效项和最终数量摘要。

### 5.2 数据合同

- 选择结果包含 stable ID、display snapshot、source、scope version 和 exclusion flag。
- 手工输入先解析和去重，再与有权限资产匹配；未知或越权目标不得静默进入任务。
- 创建任务和执行前都由服务端重新做 DataScope、状态、版本和存在性校验。
- 目标数量、解析时间、来源分布和排除规则进入任务快照与审计。

### 5.3 适用场景

- Deploy 任务创建。
- K8s Release/回滚目标。
- 文件分发、批量运维和监控对象选择。
- CMDB 批量操作；普通单选表单不强制使用复杂选择器。

## 6. 部署执行工作台

### 6.1 桌面布局

- 顶部：任务对象、状态、目标数量、版本、执行人、持续时间、数据新鲜度和合法主操作。
- 左栏：目标列表及 success/failed/running/pending 聚合，可筛选失败和当前主机。
- 中栏：`ExecutionStepRail`，显示步骤、attempt、耗时、错误锚点和当前执行位置。
- 主区：`TaskLogViewer`，按 host/step/stream 切换并保持 sequence 和 stale 状态。
- 右侧抽屉：任务参数快照、风险、审计和事件时间线。

### 6.2 移动布局

移动端默认先呈现任务状态和失败目标，目标、步骤、日志通过 tabs/segmented control 切换。主操作保持可达，但批量配置和复杂比较提示转到桌面完成。

### 6.3 恢复动作

- 动作必须明确作用对象，如“重试 3 台失败主机”或“从失败步骤重试”。
- UI 只展示后端状态机与当前权限允许的动作。
- cancel/pause/resume/retry/rollback 使用幂等键、状态版本和状态感知确认。
- SSE 断线保留现有数据，按 last sequence 重连；轮询 fallback 必须显示 stale。

## 7. 执行前变更预览

适用于 Deploy、K8s Release、Argo CD sync 和 rollback：

1. **对象摘要**：业务域、环境、目标数量和排除项。
2. **版本差异**：当前版本、目标版本、镜像/tag/digest、配置差异。
3. **资源差异**：新增、修改、删除和冲突项。
4. **风险摘要**：不可逆项、健康检查缺失、容量风险、跨环境/大范围变更。
5. **回滚条件**：可回滚版本、所需快照、无法自动回滚的对象。
6. **确认**：高风险操作要求明确确认语句或二次授权，但不以多余确认打断低风险只读操作。

Diff 输入必须先脱敏。预览生成与最终执行使用同一 immutable intent/version；如果对象版本变化，执行返回 conflict 并要求重新预览。

## 8. 可观测性查询工作台

### 8.1 统一骨架

- 顶部 sticky 查询栏：数据源、时间范围、查询/条件、运行、停止和刷新间隔。
- 查询历史：仅保存脱敏后的查询表达式、数据源引用、时间范围和用户视图元数据。
- 结果 tabs：趋势、日志、字段统计、原始数据；无对应能力的数据源隐藏该 tab。
- 时间范围显示 timezone、绝对起止和采样/limit，避免“最近 1 小时”语义不清。

### 8.2 日志体验

- 结构化视图展示时间、级别、来源、关键字段和 message；原始视图保留完整脱敏事件。
- 支持字段 include/exclude、关键词高亮、上下文前后行和直达相关资源。
- 查询进行中、达到 limit、部分 shard 失败、数据源断开和 stale 必须有独立状态。
- 实时 tail 与历史查询是不同模式，使用 segmented control 明确切换。

### 8.3 查询安全

- 前端条件 AST 只用于表达意图；后端按数据源、DataScope、字段和操作符白名单编译。
- 设置超时、返回上限、并发和查询成本预算；停止按钮应取消后端请求。
- 查询历史不得保存秘密、token、完整敏感 label 或未经脱敏的响应。

## 9. K8s 与任务日志

Pod 日志和 Deploy 日志共用 Base `TaskLogViewer`，但 adapter 分离：

- Pod：cluster/namespace/workload/pod/container、previous、since、timestamps、follow。
- Deploy：task/host/attempt/step/stream、sequence、SSE reconnect、redaction metadata。

共同能力包括搜索、暂停追尾、复制/下载、换行、全屏、级别/来源筛选和 stale；权限、DataScope、审计和下载数据仍由业务 API 负责。

## 10. 业务 Dashboard widgets

首批只注册高价值、低查询成本 widget：

- 发布任务状态摘要。
- 需要关注的失败任务。
- K8s 集群/工作负载异常摘要。
- 告警与数据源健康摘要。
- 最近访问的业务资源或任务。

每个 widget 声明 owner、permission、cleanup policy、freshness 和 query budget。点击必须进入有相同上下文/筛选的业务页。禁止用 Dashboard 复制完整业务列表或日志工作台。

## 11. CMDB 关系与影响视图

这是 P2 且默认 deferred。只有满足以下成熟度 gate 才进入实现：

- 关系类型、方向、来源、置信度和生命周期有稳定模型。
- 关键关系有 provider ownership、DataScope 和增量更新策略。
- 至少两个真实决策场景需要影响分析，而不只是“画一张图”。
- 能定义可验证的影响语义和误报/漏报边界。

满足 gate 后，先做资源详情内的邻接关系和变更影响列表；不先做装饰性全屏拓扑图。

## 12. 权限、审计与状态

- 查看、查询、执行、取消、重试、回滚、下载日志和查看秘密相关 metadata 分离权限。
- DataScope 在列表、详情、SSE/日志流、Diff、导出和 Dashboard 聚合中保持一致。
- 高风险动作记录 request ID、idempotency key、intent/version、对象数量、前后状态和来源。
- UI 覆盖 loading、empty-initial、empty-filtered、error、forbidden、conflict、stale、partial failure 和 canceled。

## 13. 视觉、响应式与可访问性

- 运维工作台保持紧凑和扫描优先，不做营销式 hero、嵌套卡片或装饰背景。
- 桌面优先多栏定位，移动端改为任务摘要 + 分段视图；不把多栏按比例缩小。
- 固定工具条、tabs、日志区和计数器有稳定尺寸，实时数据不得引发布局跳动。
- 使用图标按钮时采用现有图标库并提供 tooltip/读屏名称。
- 验证 1440x900、390x844、明暗主题、长中英文、200% 缩放、键盘焦点和 reduced motion。

## 14. 非目标

- 不修改 `platform`、`system/*` 或 Base 共享合同副本。
- 不自研日志/时序存储、部署引擎、Kubernetes controller 或图数据库。
- 不在所有页面启用复杂条件和保存视图。
- 不把文档或旧截图表述为新工作台已实现。

## 15. 拆包与完成条件

具体 O1-O7 见 `.harness/tasks/2026-08-31-ops-operational-workbench-design/EXECUTION_QUEUE.md`。

每个实现包必须同时证明：

1. 使用已发布 Base contract 或记录明确 blocked 状态。
2. 权限、DataScope、审计、i18n、状态机与 API 契约一致。
3. desktop/mobile、关键状态、明暗主题和键盘交互有渲染/运行证据。
4. 外部 Prometheus/Loki/Tempo/K8s/Argo CD/主机动作没有凭据时记录真实 gap，不伪造验收。
5. 最终视觉/功能验收由维护者基于实际渲染与可用环境完成。
