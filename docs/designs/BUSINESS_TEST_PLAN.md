# pantheon-ops 业务测试计划

更新时间：2026-08-18  
类型：Test Plan  
归属层：business  
状态：Proposed  
关联评审：[BUSINESS_ARCHITECTURE_REVIEW.md](./BUSINESS_ARCHITECTURE_REVIEW.md)  
关联验收：[BUSINESS_QA_ACCEPTANCE_PLAN.md](./BUSINESS_QA_ACCEPTANCE_PLAN.md)

本文定义从单元测试到发布验收的业务测试策略。测试目标不是增加测试数量，而是证明模块边界、授权、状态机、幂等、并发、迁移和外部系统调和满足 SRE 使用条件。

## 1. 测试金字塔

```text
                         +----------------------+
                         | Gate D 真实/隔离外部 |
                         | MySQL / K8s / SSH    |
                         +----------+-----------+
                                    |
                         +----------v-----------+
                         | Gate C 业务集成/Smoke|
                         | 跨模块 + API + UI    |
                         +----------+-----------+
                                    |
                         +----------v-----------+
                         | Gate B 不变量/并发   |
                         | 状态机/幂等/租约     |
                         +----------+-----------+
                                    |
                         +----------v-----------+
                         | Gate A 单元/契约      |
                         | Go + TS + checker    |
                         +----------------------+
```

测试替身只替换外部连接，不替换被测业务不变量；禁止为了通过测试绕过 DataScope、审计或 capability。

## 2. 测试环境和数据

### 2.1 环境

| 环境 | 用途 | 必要依赖 |
|---|---|---|
| Unit | 每次提交 | Go、Node，不需要外部凭据 |
| Integration | migration、repository、跨模块 capability | MySQL 8，`PANTHEON_TEST_DSN` |
| Contract | fake SSH/K8s API、provider、回调 | 可控 HTTP server、测试密钥 |
| Smoke | 前后端业务链路 | backend、frontend、浏览器/Playwright |
| External rehearsal | 发布前验证 | 隔离 K8s、测试 SSH 主机、可轮换凭据 |

### 2.2 最小 fixture

- 用户 `user-a`/`user-b`，分别属于部门 A/B。
- `scope-a-active`、`scope-b-active`、`scope-inactive`。
- Host A/B、Group A、Service/ServiceInstance A/B。
- Cluster A、Namespace A（绑定 scope A）、Namespace B（未绑定或 scope B）。
- 一个幂等 key、一个重复 callback、一个失败后可重试的 executor。
- 外部凭据版本 v1/v2，包含轮换和失效场景。

所有 fixture 使用独立前缀和清理钩子，不能依赖开发者本地数据库残留。

## 3. Gate A：单元、契约和静态边界

### 3.1 必跑命令

```text
cd pantheon-ops/backend
go test -count=1 ./modules/business/...
go vet ./...

cd ..
node --test tests/scripts/check-business-module-boundaries.test.mjs
node scripts/check-business-module-boundaries.mjs
```

前端适用命令：

```text
cd pantheon-ops/frontend
npm run type-check
npm run build
```

### 3.2 测试项

| ID | 范围 | 断言 |
|---|---|---|
| A-01 | CMDB capability | DTO 不含 provider Model/DB，DataScope、软删除和分页语义稳定 |
| A-02 | BizScope | active/visible scope、状态迁移、错误 key 稳定 |
| A-03 | Service | ServiceInstance target 类型、唯一有效绑定、健康状态迁移 |
| A-04 | Deploy | fingerprint、idempotency、状态流转、快照不可变 |
| A-05 | K8s | Cluster/Namespace/Release 请求归一化、resourceVersion 冲突 |
| A-06 | Boundary checker | 负例拦截跨 owner 表/Model，正例允许本模块内部和有理由的 migration/seed/fixture |
| A-07 | Frontend overlay | route/component/menu/permission/i18n 注册完整，type-check/build 通过 |

## 4. Gate B：不变量、并发和故障恢复

### 4.1 必测不变量

| ID | 场景 | 预期 |
|---|---|---|
| B-01 | 两个请求同时 bind 同一 Host/ServiceInstance | 只有一个有效绑定，另一个得到冲突，不出现双归属 |
| B-02 | bind 与 delete/retire 并发 | 无幽灵引用；失败请求不留下部分 mutation |
| B-03 | 同一 idempotency key 并发创建 | 一个 intent，其他请求可安全复用或得到一致冲突 |
| B-04 | 同一任务并发 Start | 一个执行批次，其他请求不重复下发 |
| B-05 | Worker 在执行中崩溃 | lease 超时后可接管或进入确定失败，不永久 running |
| B-06 | callback 重复、乱序、旧 generation | 幂等处理，旧状态不能覆盖新状态 |
| B-07 | cancel 与 finish 并发 | 终态由状态机规则决定，审计清楚记录竞态结果 |
| B-08 | K8s rollout 外部删除/失败/超时 | Reconcile 可恢复，condition 包含 reason/message/observed generation |
| B-09 | Secret/kubeconfig 出现在错误、日志、审计 | 明文匹配扫描为零，响应只含引用/脱敏摘要 |

### 4.2 Race 运行要求

Windows 仓库门禁要求显式启用 cgo 和 MinGW：

```powershell
$env:PATH='D:\msys64\mingw64\bin;' + $env:PATH
$env:CGO_ENABLED='1'
go test -race ./...
```

若因环境磁盘、链接器或外部依赖失败，必须记录失败阶段和 gap；普通 `go test` 不能替代 race 证据。

## 5. Gate C：跨模块集成和业务 Smoke

### 5.1 后端链路

```text
BizScope -> CMDB Host/Group -> Service -> ServiceInstance
        -> Deploy ChangeIntent -> TargetSnapshot -> Worker/Attempt
        -> K8s Release / SSH result -> Audit/summary
```

必须覆盖：

- active scope 正常创建到执行。
- inactive、跨部门、无 ownership 的请求在 mutation 前失败。
- Host/Namespace/Cluster 删除或 retire 后历史任务仍可解释。
- ServiceInstance 与 target 不一致时拒绝创建 Release/Deploy。
- 失败执行、重试、取消、回滚和最终查询。

### 5.2 前端 Smoke

至少覆盖：

- CMDB Host/Group/Label list/detail/import/export。
- BizScope list/detail、绑定和 scope 错误。
- Service/ServiceInstance list/detail、目标引用展示。
- Deploy package/template/task/detail、启动/取消/重试/日志状态。
- K8s cluster/namespace/workload/configmap/secret/release。
- loading、empty、error、permission denied、confirm、分页和窄屏退化。

用户界面改动必须提供桌面和移动渲染证据；没有可运行浏览器时，报告环境 gap，不以源码检查代替视觉验收。

## 6. Gate D：MySQL、K8s 和 SSH 外部验收

### 6.1 MySQL migration

按以下顺序运行：

1. 空库执行全部 up migration，启动应用并执行核心 CRUD。
2. 使用上一稳定版本 schema 升级到当前版本。
3. 重复执行迁移并检查 checksum、索引、约束和默认值。
4. 在每个关键 migration 步骤注入失败，重试并验证恢复。
5. 执行 down 或回滚演练，确认历史数据和审计可读。

### 6.2 K8s

- 使用两个 Cluster/Namespace 和两个业务域验证 ownership。
- 使用 fake API 做全量回归；隔离测试集群只运行少量 server-side apply、rollout、冲突和 secret 引用场景。
- 验证外部 kubectl 修改、watch 重复/丢失、对象删除和 credential rotation。

### 6.3 SSH/Executor

- 默认 fake executor；真实 SSH 只使用一次性测试凭据和无破坏命令。
- 验证指纹变化、连接超时、命令非零、stdout/stderr 截断、取消和重试。
- 任何失败都不得回写明文密码/私钥。

## 7. 回归门禁和证据

每个 task 使用 `.harness/evidence/<task-id>/`，至少包含：

- `commands.json`：实际运行命令、环境和退出码。
- `summary.md`：通过项、失败项、未执行项和影响。
- `review.md`：按严重性排序的发现和 reviewer 结论。
- 如涉及 UI：截图/视频或渲染 gap 说明。

推荐聚合门禁：

```text
go test -count=1 ./modules/business/...
go vet ./...
node --test tests/scripts/check-business-module-boundaries.test.mjs
node scripts/check-business-module-boundaries.mjs
npm run type-check
npm run build
```

需要外部环境的命令不得用空实现替代，应显式标为 `not run` 并记录原因。

## 8. 当前基线测试状态

截至 2026-08-18：`go test ./...`、`go vet ./...` 和业务边界 checker 已通过；race 因 Windows 临时磁盘空间不足未完成；MySQL、前端构建和真实 K8s smoke 未执行。以上缺口在 V1 发布前必须关闭或由维护者书面豁免。

## 9. 测试退出条件

- Gate A、B、C 全部通过。
- Gate D 的 MySQL migration rehearsal 通过；K8s/SSH 外部 gap 有明确环境和补测日期。
- P0 QA 问题全部关闭。
- P1 问题有关闭证据或批准的延期风险。
- 测试证据可由 clean checkout 的独立执行者复跑。
