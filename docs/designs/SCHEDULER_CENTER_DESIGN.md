---
title: 调度中心设计
doc_type: Design
layer: platform / system/config / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 调度中心设计

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

关联合同：
- [PLATFORM_CONTRACT.md](../contracts/PLATFORM_CONTRACT.md)
- [SYSTEM_CONFIG_CONTRACT.md](../contracts/SYSTEM_CONFIG_CONTRACT.md)
- [TENANT_CONTRACT_V1.md](../contracts/TENANT_CONTRACT_V1.md)

关联路线：
- [P2_SCALE_ROADMAP.md](./P2_SCALE_ROADMAP.md)
- [PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md](./PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md)

本文定义 Pantheon 标准后台产品位中的"调度中心"。目标不是立即把系统演进成重量级调度平台，
而是先形成统一的计划任务入口、执行记录视图和接入治理边界，并解决一个已经存在的运行事实：
**当前所有周期任务在每个副本里各自跑，既没有跨实例互斥，也没有执行历史与启停控制。**

---

## 1. 设计目标

调度中心要解决：

1. 系统级与业务级计划任务有统一注册与展示入口
2. 任务定义、启停状态、最近执行结果具备一致表达
3. 高风险执行动作具备权限与审计留痕
4. **多副本下同一任务只执行一次**（本仓库按多副本部署，见 §7）
5. 执行结果可追溯（历史、耗时、错误键、重试次数），供告警中心与 dashboard 消费

## 2. 非目标

当前阶段不做：

- 分布式作业编排平台
- 可视化 DAG 编排器
- 多云批处理控制台
- 任意脚本远程执行器
- 用一套通用执行引擎替换各域本地执行逻辑
- 任务间依赖编排（A 完成后触发 B）；跨任务协同在仓库里唯一已存在的形态是报表中心显式调用调度中心契约（§14）
- 秒级调度与实时流式触发

## 3. 归属边界

### 3.1 `platform` 负责

- 调度中心产品位
- 任务注册清单聚合
- 任务执行记录总览
- 统一状态摘要
- 触发循环与租约领取（§6/§7）

### 3.2 `system/config` 负责

- 平台级任务配置治理（默认执行策略、保留期、默认时区）
- 高敏操作保护策略
- 边界遵守 [SYSTEM_CONFIG_CONTRACT.md](../contracts/SYSTEM_CONFIG_CONTRACT.md) §7.4：
  `setting` 只输出配置值，不接管消费语义——调度语义由调度中心解释

### 3.3 源域负责

- 实际任务逻辑（处理器实现）
- 任务参数校验
- 执行结果业务语义
- **处理器幂等**：同一幂等键重复投递必须产生同一结果（§9）

分层依赖方向不变：`platform` 不 import 具体源域实现，处理器由组合根注入
（同 `backend/modules/platform/routes.go:13-16` 的 `OrgGovernanceTaskLoader` 注入纪律）。

## 4. 既有事实：仓库里现在有什么"周期任务"

这是本设计的地基，必须先说清现状，否则"调度中心"会写成一个与既有机制并立的第二套系统。

| 事实 | 出处 |
| :--- | :--- |
| 周期任务注册表 `maintenance.Registry`，任务字段仅 `Name / Interval / Run` | `backend/pkg/maintenance/maintenance.go:31-35` |
| 默认间隔 15 分钟；未给 interval 的任务用默认值 | `backend/pkg/maintenance/defaults.go:8-10`、`maintenance.go:99-101` |
| 注册幂等，重复注册不累积，且保留既有节流状态 | `maintenance.go:92-114` |
| 节流 + 重叠保护，且**重叠判定先于节流判定** | `maintenance.go:171-198`（`:181-187` overlap，`:188-191` skip） |
| 失败与 panic 被隔离：panic 转 error，一个坏任务不拖垮循环 | `maintenance.go:226-237` |
| 每次执行都出指标（按任务名与结局计数 + 耗时直方图） | `maintenance.go:239-244`，`backend/pkg/metrics/prometheus.go:98-112` |
| 阻塞式 runner：首轮立即执行，之后按 ticker | `maintenance.go:246-267` |
| runner 在**每个进程**里以 goroutine 启动，可用 env 关闭 | `backend/cmd/server/main.go:204-213`（`PANTHEON_MAINTENANCE_ENABLED`、`PANTHEON_MAINTENANCE_INTERVAL_SECONDS`） |
| 已登记的真实任务：`auth.login_log_retention`、`auth.security_event_retention`、`auth.session_inventory`、`audit.operation_log_retention` | `backend/modules/auth/login/login_service.go:51`、`backend/modules/auth/security/security_service.go:120`、`backend/modules/auth/session/session_service.go:83`、`backend/modules/system/audit/audit_service.go:30`（注册点同文件的 `RegisterMaintenanceTasks`） |
| 保留期天数从 `system_setting` 读，读不到用代码默认值 | `backend/modules/system/audit/audit_service.go:325-341` |

结论（三条必须原样继承的现状事实）：

1. 现有 registry 的状态是**进程内**的（`sync.Mutex` + `map`，`maintenance.go:70-75`），
   因此重叠保护只在单进程内成立。三副本滚动部署时，同一清理任务会在三个进程里各跑一次。
   今天这是可接受的（任务幂等），但它是"没有跨实例互斥"这一缺口的事实来源，而不是一个可以声称的能力。
2. 现有模型是**固定间隔**，没有 cron、没有时区、没有执行历史、没有启停开关、没有手动触发的权限与审计。
3. 现有任务的删除类工作已经明确移出请求路径，本设计不得把它们再放回 handler 里
   （`maintenance.go:1-12` 包注释记录的教训：一次普通读取曾为无界清理买单）。

## 5. 调度任务模型

草稿字段（`id / jobType / sourceDomain / name / description / scheduleType / scheduleExpr /
enabled / lastRunAt / lastRunStatus / lastRunSummary / owner`）全部保留，并补齐实现所需维度，
含草稿英文版要求的"下次触发时间"与"跳转目标"。
列级定义见 [SCHEDULER_CENTER_DB_DESIGN.md](./SCHEDULER_CENTER_DB_DESIGN.md) §2。

| 分组 | 字段 | 约束来源 |
| :--- | :--- | :--- |
| 身份 | `job_key`（全局唯一，点分域前缀，如 `auth.login_log_retention`） | 沿用既有任务名形状（§4 表） |
| 归属 | `source_domain`、`owner`、`job_type` | `system_log_oper.source_domain` 已有同形维度（`backend/internal/middleware/operation_log_middleware.go`） |
| 调度 | `schedule_type`（`cron/manual/event`）、`schedule_expr`、`schedule_tz` | 草稿 §4.2 |
| 状态 | `enabled`、`paused`、`next_fire_at`、`prev_fire_at`、`concurrency_policy` | §6、§8 |
| 执行策略 | `timeout_ms`、`retry_max`、`retry_backoff_seconds`、`misfire_policy` | §8、§9 |
| 呈现 | `name_key`（i18n key）、`description_key`、`source_route_key`（来源上下文跳转目标） | 不存自然语言（[DESIGN.md](../../DESIGN.md) §6） |
| 可观测 | `last_run_at`、`last_run_status`、`last_run_summary`、`last_duration_ms` | 草稿 §4 + §10 |

### 5.1 注册形态：处理器在代码，调度状态在库

- **处理器（handler）在代码注册**：`handler_key` 必须映射到一个已注册实现，未注册的键在启用前
  校验即拒绝（`report`/`schedule` 都不提供"库里写一段逻辑"的能力，对齐 §2 非目标）。
  注册形状照抄 `maintenance.Registry.Register`（幂等、空名拒绝、重复注册不累积，
  `backend/pkg/maintenance/maintenance.go:92-114`）。
- **调度状态在库**：cron 表达式、启停、超时、重试策略可运行期调整，因此必须有行 + 缓存收敛策略（§12）。
- 迁移期允许一个任务同时存在于 maintenance registry 与调度中心清单里，但必须显式标注
  "双跑窗口"，并在实现 PR 中给出下线旧注册的时间点（OD-S-4）。

### 5.2 执行状态

草稿状态集合保留，并与既有 maintenance 结局词表对齐（`maintenance.go:40-52`：
`succeeded / failed / skipped / overlap / unknown`）：

| `schedule_job` 视图态 | 含义 |
| :--- | :--- |
| `idle` | 从未执行且未暂停 |
| `running` | 持有租约正在执行（跨实例可见） |
| `success` | 最近一次成功 |
| `failed` | 最近一次失败（含重试用尽） |
| `paused` | 人为暂停，不参与触发 |

`schedule_execution.status` 另有 `pending / running / succeeded / failed / cancelled / skipped / overlap`，
其中 `skipped`（未到间隔）与 `overlap`（上一轮未结束）**直接沿用 maintenance 的既有结局名**，
不引入第二套词汇。

### 5.3 调度类型

第一阶段支持 `cron / manual / event`（草稿 §4.2）；草稿英文版提到的 `one-off` 不新增第四种类型，
用 `manual` 任务 + 一次 `run-once` 表达：

- `cron`：由 §6 的表达式语法驱动。
- `manual`：无自动触发，只有 `run-once` 动作（§11）。
- `event`：由业务代码显式调用 `Trigger(jobKey, idempotencyKey)`；第一阶段**不提供**通用事件总线，
  事件源必须是代码内的显式调用点（OD-S-1）。

## 6. cron 解析与时区

### 6.1 语法子集（第一阶段承诺）

仓库当前**没有** cron 依赖：`backend/go.mod` 的 require 列表里没有 `robfig/cron`
或任何调度库（核对结果：`grep -r cron backend/go.mod` 无命中）。因此第一阶段锁定一个
**五段最小语法**，避免为一个产品位引入未获批的依赖：

| 段 | 取值 | 支持写法 |
| :--- | :--- | :--- |
| minute | 0-59 | `*`、`*/n`、`a`、`a-b`、`a,b,c`、`a-b/n` |
| hour | 0-23 | 同上 |
| day-of-month | 1-31 | 同上 |
| month | 1-12 | 同上 |
| day-of-week | 0-6（0=周日） | 同上 |

明确**不支持**：秒级段、`@every`、`@daily` 等宏、`L`/`W`/`#`、年段、别名月份名。
非法表达式在校验阶段拒绝并返回 `schedule.expr.invalid`，不做"尽力解析"。

是否改为引入成熟 cron 库属于开放决策 OD-S-2（引入依赖需按 AGENTS「不引入未要求的新依赖」走审批）。

### 6.2 时区

时区必须显式建模，因为仓库现状是**不统一**的：

| 事实 | 出处 |
| :--- | :--- |
| 后端代码中没有任何 `time.LoadLocation` 调用（即今天没有 IANA 时区解析能力） | 全库核对：`grep -rn "LoadLocation" backend/ --include=*.go` 无命中 |
| DSN 归一化会在缺省 `loc` 时把驱动的解析时区设为 `time.Local`（进程环境时区） | `backend/pkg/database/gorm.go:36-41` |
| CI 的 DSN 显式写 `loc=UTC`，而 MySQL 服务容器 `TZ=Asia/Shanghai` | `.github/workflows/quality.yml:411`、`:393` |
| 前端时间呈现只做"浏览器本地时区渲染"，无法按任意具名时区展示；且把无偏移字符串当本地时间解析 | `frontend/src/core/format/dateTime.ts:52-70`、`:83-99` |

由此得到四条硬约束：

1. **每个任务显式携带 `schedule_tz`**（IANA 名，如 `Asia/Shanghai`），默认值来自平台设置
   `schedule.default_timezone`；不允许"跟着进程时区走"的隐式语义。
2. 计算 `next_fire_at` 必须在该具名时区内做（DST/闰秒边界见 §8.3），
   但**落库统一为 UTC** 的 `DATETIME(3)` 瞬时值；API 返回 RFC3339 带偏移。
3. Go 标准库的 `time/tzdata` 是**标准库内**的时区表（`import _ "time/tzdata"`），
   用它保证精简镜像里没有系统 tzdata 也能解析具名时区；这不属于新增外部依赖。
   该 import 属于实现动作，写进实现 PR 清单而非本设计承诺。
4. "未来 5 次触发时间"预览必须**由服务端计算**返回，前端只渲染；
   不得在浏览器里用 `Intl`/本地 getter 复算 cron（依据上一条前端事实）。

### 6.3 校验与预览接口

`POST /platform/schedule/job/validate` 只解析不回写，返回规范化表达式 + `nextFireAt` 列表；
`schedule_expr` 落库前必须经过同一套校验代码（校验与解析共用一个函数，避免"预览能过、落库不能过"）。

## 7. 触发、派发与多实例 single-flight

### 7.1 触发循环

每个实例启动一个 tick 循环（形状照抄 `maintenance.Run`：首轮立即、之后 ticker，
`backend/pkg/maintenance/maintenance.go:246-267`），周期固定 60s（`schedule.tick_interval_seconds`）。
**不做 leader 选举**：所有实例都 tick，靠 §7.2 的租约在单个任务上决出唯一执行者。
理由：leader 选举要引入新的生命周期状态与故障模式，而本仓库已有的能力（MySQL 行锁）
足以在任务粒度上解决；单任务粒度也让一个实例崩溃只影响它持有的任务。

每轮 tick 的工作集是 `enabled = 1 AND paused = 0 AND next_fire_at <= now()` 的任务，
按 `next_fire_at` 升序取有限批量（批大小上限走 `common.NormalizePagination` 同款夹取思路，
默认 50，禁止无界扫描）。

### 7.2 single-flight 的来源：MySQL 租约表，不是 GET_LOCK，也不是 Redis

必须把"锁从哪来"写成可核对的事实：

| 候选机制 | 仓库事实 | 结论 |
| :--- | :--- | :--- |
| `GET_LOCK()` | 仓库业务代码**没有**任何 `GET_LOCK` 调用；唯一使用它的是 golang-migrate 的 MySQL driver，用于跨 Pod 串行化迁移（`docs/runbooks/MIGRATION_000031_PRODUCTION_DROP.md` §1.2，第 48 行；`docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md:1070` 记录了"最先取到 `GET_LOCK`（`mysql.go:298`，等待 10s）的 Pod 执行"） | 不作为运行时任务锁：它的持有者是迁移路径，语义是连接级、随连接释放，长任务持有跨秒等待的锁会引入不可控耦合 |
| 进程内 `sync.Mutex` | 已有，且只在单进程内成立（`maintenance.go:70-75`、`:171-198`） | 保留为实例内的第二层重叠保护，不能充当跨实例保证 |
| Redis | `database.RDB` 可能为 `nil`：非生产且未开 `PANTHEON_REDIS_REQUIRED` 时连接失败会降级为 nil（`backend/pkg/database/redis.go:27-54`） | 不作为租约主存储（否则开发/测试环境调度会静默停摆）；可用于跨实例广播（§12.2） |
| **MySQL 行锁 + 租约行** | `SELECT ... FOR UPDATE` 在本仓库生产代码里已广泛使用（`backend/modules/system/iam/role/role_service.go:560`、`backend/modules/system/org/dept/dept_service.go:170`），且有现成测试夹具能证明锁语义（`backend/pkg/testmysql/locking.go:18-32` 用 `FOR UPDATE NOWAIT` 探测，`:39` 起 `AssertSerialisedWriters` 证明两个写者不可交错） | **采用**：`schedule_lease` 一任务一行，领取/续租/释放都是对该行的条件写 |

领取协议（`schedule_lease`，字段定义见 DB 设计 §2.3）：

1. `BEGIN` → `SELECT ... FOR UPDATE NOWAIT` 锁住该任务租约行；已被别的实例锁住则立即放弃本轮
   （NOWAIT 的取舍已有同款先例：`backend/pkg/testmysql/locking.go:13-17` 说明探测用
   `FOR UPDATE NOWAIT` 是为了"缺锁时快速失败"，而不是等锁超时）。
2. 判定 `lease_expire_at < now() OR holder = ''`：不成立则记 `skipped`（不是 `failed`）。
3. 成立则写入 `holder = <instanceID>`、`lease_expire_at = now() + lease_seconds`、
   `lease_version = lease_version + 1`，`COMMIT`。**`lease_version` 是 fencing token**：
   执行结束时只有携带相同 version 的写回才生效，防止被抢租约的旧持有者覆盖结果（§7.3）。
4. 执行期间每 `lease_seconds/3` 续租一次；续租失败 = 失去所有权，立即取消任务 ctx。
5. 完成后释放（`holder=''`），并写 `schedule_execution`。

`instanceID` 形状沿用既有实例标识：`pid-unixnano`（`backend/pkg/database/casbin_watcher.go:83`）。

### 7.3 崩溃与租约过期

- 实例崩溃后，任务租约在 `lease_expire_at` 到期，其他实例可在下一轮 tick 领取；
  上一轮遗留的 `running` 执行行由启动时的对账逻辑改为 `failed` 并写
  `schedule.execution.reconciled_orphan` 摘要（对账必须按 `lease_version` 判定，不得直接清空 `running`）。
- 优雅停机只有默认 15 秒预算（`PANTHEON_SHUTDOWN_TIMEOUT_SECONDS`，
  `backend/cmd/server/main.go:215-226`）。因此：任务处理器**必须**消费传入的 `ctx` 并可被取消，
  循环的 ctx 来自 `signal.NotifyContext`（同文件 `:192-195`）。
  停机时仍在执行的任务按 §9.2 的语义落 `cancelled`/`failed`，并由下一实例按 `misfire_policy` 决定是否补跑。

### 7.4 并发策略

`concurrency_policy` 第一阶段固定为 `forbid`（同一任务全局只允许一个执行者），
不提供 `replace`/`allow`（那需要取消策略与副作用治理，属 §2 非目标）。
`run-once` 与 cron 触发共用同一把租约，所以手动触发撞上自动触发时结果是 `overlap`
（沿用 `maintenance.go:47-49` 的结局定义），不是排队等待。

## 8. 错过触发（misfire）策略

先定义时钟事实：tick 周期 60s，因此任何"错过"最多是 `tick + 领取延迟` 量级；
真正的长时错过来自整栈停机或 DB 不可用。

| `misfire_policy` | 语义 | 第一阶段 |
| :--- | :--- | :--- |
| `skip` | 只按最新计划点推进，错过的点全部丢弃；一行 `skipped` 记录写明错过个数 | **默认** |
| `coalesce` | 错过多个点只补跑一次，然后推进到下一个未来点 | 允许 |
| `fire_all` | 逐个补跑错过的点 | **禁止**（清理/导出类任务补跑会放大负载，与"不引入未要求的能力"一致） |

推进规则：以 `next_fire_at` 为唯一游标，**不**用"上次实际执行时间"作为计算基准，
否则一次长任务会把后续计划点整体推移（这是 cron 语义的常见实现错误，写进设计以避免）。

时区边界：DST 春季跳过的时间点按"不存在即跳到下一个有效点"处理；秋季重复时间点只触发一次
（依据 §6.2 把 `next_fire_at` 存为 UTC 瞬时值，重复本地时刻天然映射到两个不同 UTC 瞬时，
需在推进时去重）。

时钟回拨/偏移超过一个 tick：不自动补跑，记 `skipped` 并在 UI 提示"计划点已错过"。

## 9. 重试、退避与幂等

### 9.1 重试

- `retry_max` 默认 0（不重试）、上限 5；`retry_backoff_seconds` 默认 60，按指数翻倍并封顶
  `schedule.retry_backoff_max_seconds`（默认 1800）。
- 退避实现方式：更新 `next_fire_at`（或一次独立执行的 `fire_at`）而不是 `time.Sleep` 占住 goroutine——
  与"停机预算只有 15s"这一事实一致（§7.3）。
- 分类不可重试错误（直接落 `failed`，不消耗重试额度）：
  `schedule.expr.invalid`、`schedule.handler_not_found`、`schedule.param.invalid`、
  `schedule.tenant_context_missing`。理由：这些都是配置类错误，重试只会掩盖问题。
- 可重试：处理器返回的其他错误、ctx 超时、DB 死锁类错误。
- 重试用尽后产出失败摘要供告警中心消费（§14），摘要只含 `error_key` + 计数，不含堆栈或原始参数。

### 9.2 取消与结局

`cancelled` 只在两种情况下产生：显式取消动作（需 `platform:schedule:cancel`，见 API 契约 §5）
或停机导致的 ctx 取消；其余失败一律 `failed`。
**任何结局都不允许"静默无记录"**：每次领取尝试都写 `schedule_execution` 行，
包括 `skipped` 与 `overlap`（否则 §10 的历史视图与"最近执行摘要"会出现无法解释的空洞）。

### 9.3 幂等键

- 自动触发：`fire_instance = job_key + ':' + UTC(schedule_fire_at)`（计划时间点，不是实际开始时间）。
- 手动触发：`fire_instance = job_key + ':manual:' + <客户端 idempotencyKey>`，缺省时服务端生成
  `sha1(job_key|operatorID|unixMilli)` 并回显。
- `schedule_execution` 上 `(job_key, fire_instance)` 唯一键承担最终去重；
  重复投递命中唯一键时返回既有执行行，不重复执行处理器。
- 该唯一键**不包含 `deleted_at`**，遵循本仓库唯一索引约定（软删靠改名释放键，
  见 [SCHEDULER_CENTER_DB_DESIGN.md](./SCHEDULER_CENTER_DB_DESIGN.md) §4 与既有先例
  `backend/pkg/database/migrations/000001_init_schema.up.sql:117-120`）。
- 处理器侧义务：同一 `fire_instance` 重复调用不得产生第二次业务写入。
  清理类任务的"重复调用"本然是幂等的（`maintenance.go:27-30` 已要求 Run 可重复调用）。

## 10. 执行历史与保留期

| 项 | 规则 | 依据 |
| :--- | :--- | :--- |
| 每次执行一行 | `schedule_execution`（开始/结束/耗时/结局/错误键/重试序号/持有实例/`fire_instance`） | §7、§9 |
| 任务行冗余摘要 | `last_run_at / last_run_status / last_run_summary / last_duration_ms` | 草稿 §4 |
| 保留期 | 设置键 `schedule.execution_retention_days`（默认 30），按天分批 `LIMIT` 删除 | 先例：`backend/modules/system/audit/audit_service.go:317-341`（保留期天数来自 setting，清理走注册任务） |
| 清理实现 | 注册进调度中心自己的任务，同时**保留** maintenance registry 的注册通道（OD-S-4） | `backend/pkg/maintenance/maintenance.go:92-114` |
| 清理不得在请求路径里 | 列表/详情 handler 只读 | `maintenance.go:1-12` 包注释记录的既有教训 |
| 清理开关关闭时 | `PANTHEON_MAINTENANCE_ENABLED=false` 时无人清理，必须在文档与 UI 提示里写明运维动作 | `backend/cmd/server/main.go:204-213` |
| 分页 | 历史列表走 `common.NormalizePagination(page, pageSize, 100)`，`pageSize` 服务端夹到 100 | `backend/pkg/common/handler.go:39` |

## 11. 启停、暂停与手动触发

草稿动作集合（`enable / disable / run-once / view-history`，以及草稿英文版的
`pause / resume / cancel / jump to source context`）保留，语义收紧如下：

| 动作 | 语义 | 与状态的关系 |
| :--- | :--- | :--- |
| `enable` | 恢复自动触发 | `enabled = 1`；若 `paused = 1` 则仍不触发，须显式 resume |
| `disable` | 关闭自动触发 | `enabled = 0`；**不取消在跑的执行**，只停止后续领取 |
| `pause` | 暂停但保留清单可见与历史 | `paused = 1`；`run-once` 仍可用（运维排障常需要） |
| `resume` | 解除暂停 | `paused = 0` |
| `run-once` | 立即入队一次（草稿英文版的 run now） | 高敏：独立权限键 + 二次验证 + 审计（草稿 §5 约束） |
| `view-history` | 查看执行历史 | 只读 |
| `view-source` | 跳转到来源上下文（草稿英文版 §5 的 jump to source context） | 只读导航；目标路由由源域自己鉴权，调度中心不代授权 |
| `cancel` | 取消在跑执行 | 需要独立权限；取消走 ctx 取消，结局 `cancelled`（§9.2） |
| `update-schedule` | 改 cron/时区/策略 | 变更后必须重算 `next_fire_at`，不得保留旧游标 |

约束：

1. `disable` 与 `pause` 是两个不同事实，不合并成一个布尔：前者是"这个任务不再属于本环境的计划"，
   后者是"临时静默"。草稿 §4.1 同时列了 `enabled` 与 `paused`，本设计沿用并给出区分。
2. `run-once` 必须显式选择是否沿用当前参数快照，参数在触发时定稿写入执行行，之后编辑任务不影响该次记录。
3. 状态变更（enable/disable/pause/resume）与 `run-once` 都属写操作，必须过 Casbin；
   高敏的两个（`run-once`、`cancel`）过 `SecureActionMiddleware`
   （`backend/internal/middleware/secure_action_middleware.go`，先例
   `backend/modules/auth/module.go:96-108`、`backend/modules/system/system_modules.go:188-191`）。

## 12. 权限、租户与缓存

### 12.1 权限三层

- **路由层**：`TokenAuth -> PlatformScopeTenantContext -> Casbin`，与平台 dashboard 同形
  （`backend/modules/platform/routes.go:34-37`；契约见
  [PLATFORM_CONTRACT.md](../contracts/PLATFORM_CONTRACT.md) §3.3）。
- **动作层**：`platform:schedule:{list,view,update,run,cancel,clear}`（完整登记见 API 契约 §5）。
  读/导航门禁用 `:list`，任何动作类菜单节点不得绑 `:list`——这是
  `scripts/harness/check-permission-contract.mjs:125-136` 唯一的阻断级 finding。
- **任务可见层**：第一阶段任务清单是**平台级运维资源**，不做租户行级隔离：
  `schedule_job` 不带 `tenant_id`，理由与既有平台全局表先例一致
  （`backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:11-17` 记录的平台全局判断）。
  分类登记义务见 [TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §1/§3（OD-S-5）。
- **例外**：处理器内部的取数仍必须在其自身域内带租户语义。调度中心不做行级数据范围，
  也不得替代它（`business/*` 在 V1 不参与租户隔离，
  `backend/pkg/contracts/route_groups.go:21-26`）。

### 12.2 配置与缓存收敛

- 平台设置键（走 `system_setting`，消费方式同 `audit.*` 既有先例）：
  `schedule.default_timezone`、`schedule.tick_interval_seconds`、`schedule.lease_seconds`、
  `schedule.execution_retention_days`、`schedule.retry_backoff_max_seconds`、`schedule.sync_row_cap`。
- 任务定义在多实例间的收敛：
  - 优先复用既有 Redis pub/sub 广播形状（`backend/pkg/database/casbin_watcher.go:17-33`、`:138-140`，
    env 门 `PANTHEON_CASBIN_WATCHER`），新增同类频道用于"定义已变更"通知；
  - 该广播是**尽力而为**：env 未开或 Redis 缺失时必须靠"每 tick 重新读取到期任务"自然收敛，
    因此**不得**把广播写成一致性保证（与报表中心缓存章节同一纪律）。

## 13. UI 骨架

第一阶段最小 UI（草稿 §6 保留并补齐）：

- 调度中心列表页：`SearchToolbar` 契约（[DESIGN.md](../../DESIGN.md) §7.2）、启用状态筛选、
  执行状态筛选、调度类型、来源域筛选、行内动作、最近执行结果、下次触发时间（服务端计算值）
- 任务详情：定义快照、执行策略、最近 N 次执行、租约持有者（仅运维可见性，不展示内部地址）
- 执行历史页/抽屉：状态、耗时、错误键、重试序号、`fire_instance`
- 动作区：enable/disable/pause/resume/run-once/cancel，高敏两个按 §11 走二次验证
- 来源上下文跳转入口（只读导航，目标页自行鉴权）
- 表达式校验/预览：输入 cron + 时区 → 显示未来 5 次触发（服务端返回）
- 四态齐备：无权限 / 空 / 加载 / 提交中（[EMPTY_LOADING_ERROR_STATES.md](./EMPTY_LOADING_ERROR_STATES.md)）

暂不强制（草稿 §6）：任务依赖拓扑图、实时日志流、复杂批量编排。
另加一条明确不做：**不在列表页轮询执行日志内容**（第一阶段只存摘要与错误键，不存 stdout）。

## 14. 与报表中心、告警中心、dashboard 的关系

- 报表中心是调度中心的**消费者**：报表侧注册 `report.generate` 处理器、只通过调度中心契约登记与启停，
  不自建 cron/定时器/锁（[REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md) §11）。
- 调度中心**不是**告警系统；它产出失败摘要供告警中心消费
  （[ALERT_MONITORING_CENTER_DESIGN.md](./ALERT_MONITORING_CENTER_DESIGN.md)），两者概念上必须保持分离。
- dashboard 消费调度健康摘要（成功率、连续失败数、最长运行时长），只读聚合，
  沿用 `platform:dashboard:view` 的授权形状（`backend/modules/system/seed.go:210`）。
- 指标复用既有形状：按任务名与结局计数 + 耗时直方图
  （`backend/pkg/metrics/prometheus.go:98-112`），调度侧新增同类指标时不改变既有标签基数约定。

## 15. 最小样板要求

草稿三类样板保留，并各给一个"今天已存在"的接入点，避免样板变成 demo。
至少一类真实可调度来源域接入后，本中心才算完成（草稿 §9）：

1. **审计/登录/安全事件保留期任务**：现有 4 个 maintenance 任务（§4 表）迁移进调度中心，
   获得跨实例互斥、历史与启停。这是最有说服力的样板，因为它直接消灭 §4 结论 1 的缺口。
2. **i18n 资产同步 / 动态模块同步**：作为 `manual` + `event` 型任务接入，验证非 cron 路径。
   注意 `dynamicmodule / generator` 属高敏系统能力，不按 `system/config` 子域处理
   （[SYSTEM_CONFIG_CONTRACT.md](../contracts/SYSTEM_CONFIG_CONTRACT.md) §7.2），
   其触发动作必须保留原有权限与环境守卫。
3. **报表生成任务**：`report.generate`（§14），验证"调度中心触发 + 他域产出写回"的分工。

样板要求的是"真实任务 + 真实权限 + 真实留痕"，不是演示数据。

## 16. 完成定义

第一阶段完成标准：

- 有统一调度任务模型与注册形态（§5），四张表落地并有迁移
  （[SCHEDULER_CENTER_DB_DESIGN.md](./SCHEDULER_CENTER_DB_DESIGN.md)）
- 有 cron 最小子集解析 + 具名时区计算 + 服务端预览（§6），并有对应测试
- 有多实例 single-flight 的**可证明实现**（§7.2 协议 + 用 `testmysql` 锁夹具的测试），
  不是"看起来会用锁"
- 有错过策略、重试退避与幂等键（§8/§9），且每次尝试都有执行记录（§9.2）
- 有执行历史视图与保留期清理（§10）
- 有启停/暂停/手动触发动作及其权限与审计（§11）
- 有调度中心入口（菜单 + 路由 + i18n + 权限点同步登记）
- 有至少一个真实任务样板接入（§15 第 1 项）
- 测试计划（SCHEDULER_CENTER_TEST_PLAN.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺））中 P0 用例全绿或有显式豁免

不是以"已经具备企业级作业编排能力"作为完成标准。

## 17. 开放决策 / Open decisions

| 编号 | 决策 | 选项 | 影响 | 决策人 |
| :--- | :--- | :--- | :--- | :--- |
| OD-S-1 | `event` 型任务的触发源形态 | A：只有代码内显式 `Trigger()` 调用；B：引入通用事件/消息总线 | B 会把调度中心变成消息中间件，本设计倾向 A | 维护者 |
| OD-S-2 | cron 解析实现 | A：本设计 §6.1 的五段最小子集；B：引入 `robfig/cron` 等依赖；C：只做固定间隔 | B 需按 AGENTS「不引入未要求的新依赖」获批；C 会让 `schedule_type = cron` 变成空承诺 | 维护者 |
| OD-S-3 | 租约存储 | A：MySQL 租约表（本设计）；B：Redis `SET NX` + TTL；C：`GET_LOCK` | B 在非生产 `RDB = nil` 时静默失效（`backend/pkg/database/redis.go:27-54`）；C 与迁移路径争用同一机制 | 维护者 |
| OD-S-4 | 既有 4 个 maintenance 任务是否下线 | A：迁入调度中心并删除旧注册；B：双跑一个窗口后删除；C：长期共存 | 影响 §15 样板能否立即消灭 §4 结论 1 的缺口；倾向 B（有回退路径） | 维护者 |
| OD-S-5 | 调度任务是否需要租户维度 | A：平台级资源、无 `tenant_id`（本设计）；B：`tenant_id` 可选并允许租户自定义任务 | B 会把"任意脚本远程执行"风险引回，且与 §2 非目标冲突；需要回写 scope 矩阵 | 维护者 |
| OD-S-6 | 默认时区取值 | A：平台设置 `schedule.default_timezone = Asia/Shanghai`；B：强制 `UTC` | A 贴合运营直觉但依赖 §6.2 的 tzdata 前提；B 最安全但列表页"下次触发"对使用者不直观 | 维护者 |

## 18. 关联文档

| 文档 | 中文 | 英文 |
| :--- | :--- | :--- |
| 主设计（本文） | `SCHEDULER_CENTER_DESIGN.md` | [SCHEDULER_CENTER_DESIGN.en.md](./SCHEDULER_CENTER_DESIGN.en.md) |
| 数据模型与迁移 | [SCHEDULER_CENTER_DB_DESIGN.md](./SCHEDULER_CENTER_DB_DESIGN.md) | SCHEDULER_CENTER_DB_DESIGN.en.md（planned/unavailable：`./SCHEDULER_CENTER_DB_DESIGN.en.md`；无当前交付承诺） |
| 接口契约 | SCHEDULER_CENTER_API_CONTRACT.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺） | SCHEDULER_CENTER_API_CONTRACT.en.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.en.md`；无当前交付承诺） |
| 测试计划 | SCHEDULER_CENTER_TEST_PLAN.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺） | SCHEDULER_CENTER_TEST_PLAN.en.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.en.md`；无当前交付承诺） |

报表侧配套：[REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md)。
目录索引：[README.md](./README.md)。

## 变更日志

| 日期 | 变更 |
| :--- | :--- |
| 2026-11-04 | 补齐既有周期任务事实核对、cron 最小子集与时区硬约束、多实例 single-flight（MySQL 租约 + fencing token）、错过策略、重试退避与幂等键、执行历史保留期、启停/暂停/手动触发语义、权限与缓存收敛、开放决策与文档索引 |
| 2026-10-31 | 初始草稿 |
