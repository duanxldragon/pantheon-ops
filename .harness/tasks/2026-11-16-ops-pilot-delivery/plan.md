# Pantheon Ops 首期试点交付计划

更新：2026-11-18。目标由维护者确认：按仓库业务设计评估试点交付。父包 in-progress；C01/C03/C04 继续实施，其余依赖按原队列。当前验证见 [历史清查进度](../../evidence/2026-11-17-ops-history-audit/progress.md)。

## Requirements Summary

交付 CMDB → BizScope → Application/Service/Instance → Deploy → SSH/K8s 的可验收闭环。五个核心模块均已有代码，当前不是增加插件阶段；本次审查六组 gap 拆为六个核心工作包，扩展另四个。任务数不是完成率，也不表示每包只有一个实现步骤。

## 2026-11-16 初始基线、事实与不确定性（历史）

- 业务审查提交：691dd4553c6e47c6eebe8e13a44af71cfdf7ebb5（当时 Base v0.18.7）；本轮规划提交：07d53988a13073d0293e44c906cee076d6d375c3，已提交锁：pantheon-base-v0.18.9 / 6316f9b7189c86542424145fa49344ad737e952b。
- 本轮开始工作树干净，锁文件为 pantheon-base-v0.18.9 / 6316f9b7189c86542424145fa49344ad737e952b。最新提交记录官方 Base v0.18.9 消费装配，本轮保留字节；消费提交不替代 Ops 业务验收。C05 对最终候选重新采证，不套用旧审查结果。
- 上一轮会话实测：完整 Windows CGO race、业务 vet、frontend type-check/build、业务边界/菜单契约与 overlay 59/59 通过；lint 1 warning。业务 pass/skip 事件 656/366（含子用例），0 fail；raw 不在本轮重新伪造。
- 旧 README 的 96%/GA 及八月状态不是现状证据。Service 已支持创建等操作，ArgoCD/Velero 已重新启用。
- 代码证实：普通部署 UI 不提交实例关联；后端关联准入不足；目录固定前100及跨模块查询耦合；K8s Service 缺更新/并发保护。具体越权、错配和在途取消影响未经本次动态复现。
- 近期菜单任务缺 rendered/E2E；CI Smoke Sanity 只跑平台/系统域，不能关闭业务烟测。

## Scope / In / Out

In：C01-C06；所有权限、引用、迁移、i18n、菜单与验收随业务同步。E01-E04 默认不计核心退出，但任何开放的扩展必须在交付前关闭对应 P0/验收。所谓裁剪必须在交付候选的菜单、路由/权限、可见能力和说明中可验证；不能只在文档写 Out。

Out：平台/系统域通用实现副本、自动 node_exporter/file_sd 纳管、完整实时健康平台、商业 License/ModuleGate、生产操作和租户生产迁移。只明确首期需求，不额外创造插件。

## 任务队列

| 编号 | 工作包                                                                                                               | 优先级/档位 | 依赖                           | 状态    | 退出口径 |
| ---- | -------------------------------------------------------------------------------------------------------------------- | ----------- | ------------------------------ | ------- | -------- |
| C01  | [部署与服务实例关联及引用保护](../../../docs/harness/tasks/2026-11-16-ops-service-deploy-integrity.task.md)          | P0 / L2     | 无                             | planned | 核心必交 |
| C02  | [页面服务实例到部署结果闭环](../../../docs/harness/tasks/2026-11-16-ops-service-deploy-user-flow.task.md)            | P1 / L2     | C01                            | planned | 核心必交 |
| C03  | [服务目录维护、分页与受限角色](../../../docs/harness/tasks/2026-11-16-ops-service-catalog-usability.task.md)         | P1 / L2     | 无                             | planned | 核心必交 |
| C04  | [Kubernetes Service 编辑与并发保护](../../../docs/harness/tasks/2026-11-16-ops-k8s-service-mutations.task.md)        | P1 / L2     | 无                             | planned | 核心必交 |
| C05  | [当前版本数据库、执行与业务视觉验收](../../../docs/harness/tasks/2026-11-16-ops-business-runtime-acceptance.task.md) | P0 / L2     | C01, C02, C03, C04             | planned | 核心必交 |
| C06  | [试点交付包、安装回滚与状态收口](../../../docs/harness/tasks/2026-11-16-ops-pilot-release-closeout.task.md)          | P0 / L2     | C05                            | planned | 核心必交 |
| E01  | [文件分发创建入口与可靠控制](../../../docs/harness/tasks/2026-11-16-ops-filedist-product-closure.task.md)            | P1 / L2     | 无                             | planned | 条件扩展 |
| E02  | [ArgoCD 归属隔离与真实状态收敛](../../../docs/harness/tasks/2026-11-16-ops-argocd-ownership-convergence.task.md)     | P0 / L2     | 无                             | planned | 条件扩展 |
| E03  | [Helm 能力口径与真实 Chart 安装边界](../../../docs/harness/tasks/2026-11-16-ops-helm-capability-truth.task.md)       | P1 / L2     | 无                             | planned | 条件扩展 |
| E04  | [已启用扩展真实外部验收](../../../docs/harness/tasks/2026-11-16-ops-extensions-live-acceptance.task.md)              | P1 / L2     | 按开放能力条件依赖 E01/E02/E03 | planned | 条件扩展 |

## Implementation Steps / 排序

1. 开始每包前核对实际候选和 CodeGraph/源码，重新定位 guard 与测试；本轮只完成规划和归档。
2. C01 的跨 owner 校验/引用保护先行，C02 同一关联接口稳定后接 UI。
3. C03 与 C04 可独立推进；C02 与 C03 都修改 ServiceCatalog，必须串行或明确拆分 owner，不并发覆盖。
4. C05 只验收完成后的同一候选，覆盖空库/升级/故障/回滚、真实 SSH/K8s 和业务 UI；环境未具备时保持 not-run，不以测试 skip 关闭。
5. C06 从已通过 C05 的 clean candidate 输出包并演练安装、升级、回滚，等待最终功能/视觉验收。
6. 条件扩展独立实施 E01/E02/E03，E04 按实际启用集合验证，E01-E03 依赖只对选中/开放能力生效；核心队列不能因未承诺的后续产品无限扩张。

## Acceptance Criteria

- 首期 C01-C06 全部完成；P0 全通过，任何 P1 延期必须有风险 owner、截止日期、补偿控制与维护者决定。
- 正向/越权/错配/重复/并发/取消/重试/故障恢复/回滚可由独立执行者复跑；关键必测用例无未解释 skip。
- source revision、foundation、migration、fixture、日志/截图、包 SHA-256 与全部 evidence 属于同一候选；历史证据只证明历史范围。
- UI 桌面/窄屏、主要交互与错误/权限状态有实际渲染；未产出时只能列 gap。
- 交付包安装/升级/回滚通过、扩展裁剪可验证、维护者最终功能与视觉验收后，parent 才能 completed。

## Risks and Mitigations

- 审查后 foundation 升级：保留最新提交与锁；不在旧树手抄 Base。按消费者流程记录装配来源，C05 固定实际候选重新验收。
- 目标错配与状态错误：C01 先以最小反例和真实事务验证，不先补 UI 后补边界。
- 外部环境不足：准备 task-owned 隔离依赖；按 not-run/skip 留证，不访问生产或伪造通过。
- 扩展已启用但声明 Out：C06 验证实际裁剪；未裁剪即对应扩展门禁必过。
- 历史任务字段/链接不完整：不扩大豁免；本次 raw archive 字节保全，历史门禁前后对比且单独列债务。

## Verification Steps

每个子任务中的 Verification Plan 是最小集合；业务源代码变更后补 business-overlay 契约与兼容重建。Windows完整 race 使用 D:\msys64\mingw64\bin 和 CGO_ENABLED=1。C05/C06 汇总 gate A-D，不用新计划文件本身的门禁为业务实现背书。

## 历史完成与开放继承

已归档：8 月 foundation hardening 主包+8 子包、follow-redirects 审计、Base v0.18.6 与已合并 v0.18.7 消费，共12包；见 [.harness/ARCHIVE.md](../../ARCHIVE.md)。它们完成的是原范围，不抹掉新审查发现。原始 evidence 完整保留。

继续开放：新菜单运行验收归 C05；CMDB 导入/迁移/引用 fixture 与服务目标一致性归 C01/C05；扩展观测/GitOps/备份/文件传输归 E 队列；最新 Base v0.18.9 消费提交保持不变，Ops 业务验收仍由 C05 验证。Base FR028/真实S3/维护冻结与租户G2/G3仍按上游决定和独立生产轨道跟踪，不写成 Ops 试点已经通过。

日志存在但无摘要的 2026-11-07 planned-task-closeout、2026-11-14 latest-foundation-and-plans、Argo adoption 目录保留，不据目录名归档。旧发布/综合验收清单含 OPEN 的包继续保留；详细 retained 原因见本轮 archive-inventory.json。

## Staffing / 安全交接

executor 负责明确 owner 的实施，test-engineer/verifier 负责隔离运行证据，code-reviewer 独立审查；当前 Dispatcher 仅维护任务/计划/档案。可使用会话内 native agents。仅在已有 tmux OMX runtime 时使用 team；不能把 native agents 冒充 tmux team。无具体排期基线，不估计剩余天数。

## Linkage

- Parent Task: docs/harness/tasks/2026-11-16-ops-pilot-delivery.task.md
- Machine state: .harness/tasks/2026-11-16-ops-pilot-delivery/manifest.json
- Queue: .harness/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md
- Current entry: .harness/STATUS.md
- Plan/archives governance evidence: .harness/evidence/2026-11-16-ops-readiness-plan-closeout/

归档兼容说明：八月 8 个子包的旧文档文件名含日期，但机器 taskId 不含日期；文档地址保留为 Superseded 历史入口，机器任务/evidence 按真实 taskId 归档。原始 evidence 不复制、不改写。PR183 的本轮 GitHub 复查因认证/限流失败，合并事实由上一轮 hosted 观察及本轮本地 commit ancestry 支撑；不冒充新的 GitHub raw 证据。

条件依赖执行规则：开始 E04 前保存试点开放能力清单，对 FileDist/ArgoCD/Helm 分别要求 E01/E02/E03 完成；未选能力必须由 C06 在候选中验证裁剪。C06 完成前，所有开放扩展的 E04 验收必须通过。仅启用 Observability/Velero 不等待未选 FileDist/ArgoCD/Helm。关联部署须两关联字段非0且一致；显式独立部署两字段均0且不回写实例，历史兼容有追溯。
