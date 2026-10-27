# O7 CMDB Relationship Impact Discovery

- Priority: `P2`
- Layer: `business/cmdb`
- Status: `deferred`
- Base Dependency: none for discovery; future visualization may reuse Base primitives
- Business Dependency: mature relationship model

## Outcome

在关系模型足够可信后，为资源详情提供可决策的邻接关系和变更影响列表；不先建设装饰性拓扑图。

## Entry Gate

以下条件全部满足才从 `deferred` 变为 `planned`：

1. 关系类型、方向、来源、置信度和生命周期有 canonical schema。
2. 每类关系有 provider ownership、DataScope 和增量/删除同步策略。
3. 至少两个真实决策场景需要关系影响，例如发布影响与资源下线。
4. 能定义误报/漏报边界、数据新鲜度和冲突解决方式。
5. 有可重复 fixture 验证关系创建、更新、删除和孤儿清理。

## Discovery Scope

- 盘点现有 Application/Service/ServiceInstance/Host/K8s resources 的事实关系。
- 定义 relationship envelope 与 provenance，不先定 UI 图形。
- 为两个决策场景写 impact query 和 acceptance examples。
- 评估列表/树/局部图哪种最小表达满足任务。

## Out Of Scope

- 不在 gate 前创建图数据库、全屏 topology canvas 或自动根因分析。
- 不把推测关系表述为事实。
- 不允许跨模块直接读 owner 表拼关系。

## Expected Files

- discovery design、relationship contract proposal、fixtures and review only。
- 任何 schema/API/production UI 进入单独 L2 implementation packet。

## Acceptance For Discovery

1. 每条关系可回答“谁提供、何时观察、可信度、何时失效”。
2. DataScope 在节点、边、impact result 和导出中一致。
3. 两个决策场景能用固定 fixture 得出可解释结果。
4. UI 选择由信息任务决定，优先邻接列表和影响分组。
5. 未满足 gate 时保持 deferred，不影响 O1-O6 发布。

## Evidence

- schema/ownership review、fixture query results、DataScope matrix、staleness cases。
- 若进入 UI 阶段，再补 desktop/mobile、keyboard、large graph and reduced-motion evidence。

## Gates

- 关系 schema、跨模块 capability、数据迁移、图数据库/新依赖和任何 production impact claim 均需维护者决策。
