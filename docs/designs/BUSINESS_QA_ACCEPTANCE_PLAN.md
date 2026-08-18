# pantheon-ops QA 验收问题与准入计划

更新时间：2026-08-18  
类型：Acceptance  
归属层：business  
状态：Proposed  
关联评审：[BUSINESS_ARCHITECTURE_REVIEW.md](./BUSINESS_ARCHITECTURE_REVIEW.md)  
关联测试：[BUSINESS_TEST_PLAN.md](./BUSINESS_TEST_PLAN.md)

本文把架构评审中的风险转成 QA 可以逐条执行、留证和判定的验收问题。任何一条 P0 不通过，不能发布；P1 未通过必须有延期决定、风险 owner 和补偿控制。

## 1. 验收前置条件

- 使用 clean checkout 和固定 base foundation 版本。
- 准备 MySQL 8 空库、升级库、重复迁移库和可恢复备份。
- 准备两个互不可见的部门/业务域、两个用户、一个 active scope 和一个 inactive scope。
- 准备至少一个 Host、一个 ServiceInstance、一个 K8s Cluster、两个 Namespace（一个已绑定、一个未绑定）。
- 准备 fake SSH executor 和 fake K8s API；真实 SSH/K8s 只在隔离环境执行。
- 所有测试请求带 trace id；高风险动作可在审计中按 actor、scope、target、attempt 查询。

## 2. P0 准入问题

| ID | QA 问题 | 预期结果 | 证据 |
|---|---|---|---|
| P0-DB-01 | 空数据库执行全部 versioned up migration，应用启动并读写 Release/ServiceInstance | 成功，无 unknown column、无隐式 AutoMigrate | migration log、schema snapshot、API smoke |
| P0-DB-02 | 在已有旧版本数据库上升级，再重复执行 migration | 升级幂等，重复执行无破坏、无重复索引/列 | before/after schema、重复执行日志 |
| P0-DB-03 | 在中途故障后重新执行 migration | 只完成缺失步骤，不产生半成品和不可恢复状态 | failure injection、recovery log |
| P0-DB-04 | 从 clean checkout 重建 backend/frontend 并打包 | 版本、migration、菜单、权限、前端组件可复现 | commit、构建摘要、artifact checksum |
| P0-AUTH-01 | 用户 A 在 scope A 创建/更新 Cluster，尝试使用 scope B 或 inactive scope | mutation 前拒绝；数据库无部分写入；审计记录拒绝原因 | HTTP 响应、DB diff、audit event |
| P0-AUTH-02 | 用户无 DataScope、传空 scope、篡改 department id 访问 Cluster/BizScope | 默认拒绝，不能把 nil/空值解释为全量可见 | API matrix、审计 |
| P0-AUTH-03 | 并发修改 Cluster ownership 和业务域状态 | 最终状态符合唯一约束和状态机，无跨域归属 | 并发测试、锁/冲突日志 |
| P0-BOUNDARY-01 | 扫描 Deploy/K8s/Service request path 是否直读 CMDB/BizScope 表或 GORM Model | checker 失败用例可拦截，生产路径无未授权引用 | checker output、allowlist review |
| P0-RELEASE-01 | 重复提交相同 idempotency key、重复 Start、重复 callback | 只创建/执行一次，返回已有 intent 或冲突，不覆盖历史 Attempt | API response、任务表、审计 |
| P0-RELEASE-02 | Worker 崩溃、租约超时、进程重启后恢复任务 | 任务可接管或进入确定失败态，不永久 running，不重复执行副作用 | lease timeline、Attempt history |

## 3. P1 准入问题

### 3.1 CMDB 与业务域

| ID | QA 问题 | 预期结果 |
|---|---|---|
| P1-CMDB-01 | 创建 Host、Group、LabelSchema，按 env/biz/region 查询和分页 | 结果只包含当前 DataScope，分页稳定，软删除不回显 |
| P1-CMDB-02 | 修改/删除 Host 时存在 ServiceInstance、Deploy snapshot 或 K8s 引用 | 给出引用冲突或 retire 流程，不静默级联破坏历史 |
| P1-CMDB-03 | 同一资源重复导入、外部 id 冲突、标签 key/value 非法 | 幂等或明确冲突，错误 key 稳定且可国际化 |
| P1-BIZ-01 | BizScope inactive、跨部门、被删除后创建 ServiceInstance | mutation 被拒绝，已有引用进入可观测失效状态 |
| P1-BIZ-02 | ServiceInstance 指向 Host A，但请求携带 Host B/其他 Cluster/Namespace | 绑定校验失败，无部分写入 |
| P1-BIZ-03 | 并发 bind/unbind/delete | 状态机和唯一约束保证单一有效绑定 |

### 3.2 Deploy 与执行中心

| ID | QA 问题 | 预期结果 |
|---|---|---|
| P1-DEP-01 | 创建任务后 HTTP 请求立即返回，远程执行超过请求超时 | API 返回 intent/task id；Worker 异步执行，状态可查询 |
| P1-DEP-02 | 多目标执行中单台失败、其他成功 | 每个 Attempt 独立记录 stdout/stderr/exit code；任务聚合状态可解释 |
| P1-DEP-03 | 超时、重试、取消和 Worker 重启 | context 取消通知 executor；重试不覆盖旧 Attempt；最终状态单调收敛 |
| P1-DEP-04 | 部署期间 CMDB 标签/业务归属改变 | 已启动任务使用原 target snapshot，不漂移到新目标 |
| P1-DEP-05 | SSH 指纹变化、凭据失效、命令返回非零 | 任务失败且不泄漏凭据；错误可定位、审计可追踪 |
| P1-DEP-06 | ListTasks、ListTargets、K8s List 处理大数据量 | 服务端分页和索引生效，无全量内存加载和明显超时 |

### 3.3 Kubernetes

| ID | QA 问题 | 预期结果 |
|---|---|---|
| P1-K8S-01 | 在已绑定 Namespace 发布 Workload/Release | 通过 scope、权限、resourceVersion 校验，生成完整 rollout observation |
| P1-K8S-02 | 在未绑定或其他业务域 Namespace 写入 | mutation 前拒绝，K8s 无副作用 |
| P1-K8S-03 | 同时从平台和 kubectl 修改对象 | stale resourceVersion 返回冲突，不覆盖外部更新 |
| P1-K8S-04 | Secret 创建、读取、日志、审计、错误响应 | 明文永不回显；只显示引用/脱敏摘要；权限按动作分离 |
| P1-K8S-05 | rollout 超时、Pod 失败、外部对象删除、重复 watch 事件 | Release condition 最终可解释，reconcile 可重试且幂等 |
| P1-K8S-06 | Cluster credential 轮换、旧版本失效 | 新任务使用新版本，旧任务按策略结束；审计记录版本 |

## 4. P2 质量问题

- 是否已定义每个 Service 的 SLI/SLO、责任人和错误预算？
- Metrics、Logs、Trace 的标签是否能从 ServiceInstance/TargetRef 稳定关联？
- 告警是否具备分组、抑制、静默、升级和 Incident 关联？
- GitOps 与平台执行的事实源、冲突策略、回滚责任是否明确？
- 资源退役、凭据轮换、租户迁移、跨区域灾备是否有演练？
- CMDB 关系图、容量、成本和供应链风险是否只引用外部事实而非复制原始数据？

## 5. 验收矩阵

| 类别 | 必测范围 | 通过标准 |
|---|---|---|
| 功能 | CMDB/BizScope/Service/Deploy/K8s 主流程 | 正向流程可完成，状态和错误可解释 |
| 授权 | DataScope、scope ownership、细粒度动作 | 越权在 mutation 前拒绝 |
| 数据 | migration、唯一约束、软删除、回滚 | 空库/升级/重复/故障恢复通过 |
| 可靠性 | 幂等、租约、重试、取消、reconcile | 重复和崩溃后不双执行、不永久卡住 |
| 安全 | SSH/kubeconfig/Secret、日志、审计 | 凭据不落明文，操作可追溯 |
| 性能 | 分页、索引、批量目标、K8s List | 达到环境基线，无全量加载 |
| 前端 | permission/menu/i18n、loading/error/empty/confirm | 可访问、可回退、无未注册路由 |
| 运维 | metrics/health/logging、backup/restore | 关键故障可定位，备份可恢复 |

## 6. 发布退出条件

发布候选只有同时满足以下条件才可进入人工验收：

1. 所有 P0 问题通过，或有维护者批准的明确豁免。
2. P1 问题全部关闭，或记录风险、owner、截止日期和临时控制。
3. `BUSINESS_TEST_PLAN.md` 中的 Gate A-D 证据齐全。
4. clean checkout 构建、migration rehearsal、业务 smoke 和回滚 rehearsal 完成。
5. 前端改动有桌面/移动渲染证据；无法渲染时记录环境 gap，不声称 UI 完成。
6. `.harness/evidence/<task-id>/summary.md`、`review.md` 和命令清单可被独立执行者复核。
