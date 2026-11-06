---
title: 通知中心设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/designs/NOTICE_CENTER_API_CONTRACT.md
updated_at: 2026-11-04
---

# 通知中心设计

English version: [NOTICE_CENTER_DESIGN.en.md](./NOTICE_CENTER_DESIGN.en.md)

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

关联合同：
- `PLATFORM_CONTRACT.md`
- `TENANT_CONTRACT_V1.md`
- `SYSTEM_IAM_CONTRACT.md`

关联设计：
- `NOTICE_CENTER_DB_DESIGN.md`
- `NOTICE_CENTER_API_CONTRACT.md`
- `NOTICE_CENTER_TEST_PLAN.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

本文定义 Pantheon 平台层通知中心的产品与技术骨架，并作为「通知中心文档包」的入口。
本文是设计稿，不是实现说明：文中给出的表名、路由、权限 key 都是**待实现契约**，实现落地前不得对外宣称已具备该能力。

文档包四份，分工固定：

| 文档 | 承担 |
|------|------|
| `NOTICE_CENTER_DESIGN.md`（本文） | 目标、边界、模型、状态机、语义、开放决策 |
| `NOTICE_CENTER_DB_DESIGN.md` | 表/列/索引/迁移/清理 |
| `NOTICE_CENTER_API_CONTRACT.md` | REST 面、DTO、错误码、权限、审计头、i18n 命名空间 |
| `NOTICE_CENTER_TEST_PLAN.md` | 验收标准 → 用例 → 门禁挂点 → 运行态缺口 |

目标不是直接做成 IM 系统，而是先把平台级消息中心做成可治理、可跳转、可审计的统一入口。

## 1. 设计目标

通知中心要解决：

1. 平台级消息不再散落在各系统页角落
2. 跨域消息有统一入口、统一已读状态和统一跳转模型
3. dashboard、壳层顶部和后续任务/告警摘要可共享一套消息语义
4. 消息的产生方（源域）与消费方（用户）之间有一条**幂等、可重试、可追溯**的投递边界，而不是各模块各自写库

## 2. 非目标

当前阶段不做：

- 即时聊天
- 群聊 / 私信
- 富文本消息编辑器
- 多端推送网关（厂商 push / 小程序订阅消息 / App 角标）
- 完整消息模板引擎
- 消息已读的「谁看过」多人协作视图（回执只到收件人粒度）

## 3. 归属边界

### 3.1 `platform` 负责

- 通知中心入口与聚合视图
- 消息模型、收件人视图与已读/未读状态
- 跳转契约
- 渠道投递的执行与账本（含重试）
- 保留与清理策略的执行载体

### 3.2 系统域 / 业务域负责

- 产生消息的业务语义
- 源数据有效性
- 跳转目标页面的详情承载

一句话：

> `platform` 负责收口消息入口，不负责替代源域业务。

边界的具体落点：源域**不写**通知表，只调用 `platform` 的摄入接口（进程内 service 调用或 §13.2 声明的内部 HTTP 面）。理由是可引用条款：`docs/contracts/PLATFORM_CONTRACT.md` §7.1「聚合层可以读取多个子域数据，但不能侵入子域内部职责」的镜像方向——反过来，子域也不得反向持有平台消息表。

### 3.3 与相邻设计稿的分工

本仓库已有四份相邻的未实现设计稿，通知中心不吞并它们的概念：

| 相邻文档 | 它承担 | 通知中心承担 |
|----------|--------|--------------|
| `NOTIFICATION_SERVICE_DESIGN.md` | 渠道能力（email/sms/websocket）与服务侧发送 API、用户偏好 | 消息本体、收件人视图、跳转、平台入口；渠道执行以本文为调用方 |
| `ALERT_MONITORING_CENTER_DESIGN.md` | 告警规则、事件聚合与分级 | 把已定级的告警作为 `alert` 类消息呈现给收件人 |
| `TASK_CENTER_DESIGN.md` | 可执行工作项与其状态 | 任务的「分配/到期/结果」类消息 |
| `PLATFORM_DASHBOARD_DESIGN.md` | 首页摘要卡片 | 摘要的数据源与跳转语义 |

`NOTIFICATION_SERVICE_DESIGN.md` 内已给出 `system_notification` 草案表结构（同文档 §2.1，未实现）。它与本文的三张表**不能同时落库**，收敛路径登记为 §17 开放决策 OD-1。

### 3.4 现状基线（可反证）

- 壳层已有一个「通知中心」占位入口：`frontend/src/core/layout/AppLayout.tsx` 使用 `app.notice.*` 文案（`frontend/src/i18n/resources/zh-CN.ts` 起 `app.notice.title` 等），其空态文案明写「当前尚未接入实时通知」，展示的数据来自登录/会话/安全事件，而不是消息模型。
- 后端没有通知模块：`backend/modules/platform/` 目前只有 `dashboard_*` 与 `health_*`，`backend/cmd/server/main.go:169` 的 `RegisterPlatformRoutes` 只注册 `/dashboard/summary` 与健康检查。
- 后端没有 WebSocket 基础设施（`grep -ri "websocket\|Upgrader" backend/` 在出货代码中无命中），因此「实时推送」在本文一律按未落地能力处理。

## 4. 消息模型

### 4.1 本体与收件人视图分离

一条消息**一份本体**，每个收件人**一行视图**。原因是 §9 的已读语义必须以「收件人 × 消息」为单位存储；把已读时间写在消息本体上，第二条消息就永远读不完。

- `system_notice`：消息本体（标题/摘要/类别/严重级/来源域/跳转/payload/租户）
- `system_notice_target`：收件人视图（read/archived/delivered 状态）
- `system_notice_delivery`：站外渠道投递账本（仅在启用 email/sms/webhook 时写入）

列级定义见 `NOTICE_CENTER_DB_DESIGN.md` §4。

### 4.2 字段

最小字段（视图/本体合并后的对外字段）：

- `id`
- `category`
- `sourceDomain`
- `title`
- `summary`
- `severity`
- `status`
- `jumpType`
- `jumpTarget`
- `createdAt`
- `readAt`
- `actorId`

补充约束：`title` / `summary` 存**渲染后文本**，不存模板 key；本地化由源域在摄入时按收件人语言决定（见 §17 开放决策 OD-2）。

### 4.3 分类

最小分类：

- `task`
- `alert`
- `audit`
- `system`
- `business`

分类是**产品语义**，不是源域自由字符串。新增分类必须先扩展本文枚举，再进 `system_dict` 与 i18n，否则列表筛选会退化成自由输入（`docs/contracts/SYSTEM_CONFIG_CONTRACT.md`）。

### 4.4 严重级别

- `info`
- `warning`
- `critical`

`critical` 允许在列表里高亮并进入未读计数的独立分桶（§9.2），但**不**默认触发站外渠道：站外开关由渠道配置决定，不由级别决定。

### 4.5 状态

- `unread`
- `read`
- `archived`

状态是 `system_notice_target` 的派生视图，不是本体字段：`archived` 优先于 `read`，`read` 优先于 `unread`（读序见 §5.1）。

## 5. 状态机

本仓库 `docs/designs/` 不使用图语法（现存文档零 `mermaid` 命中），状态机一律用迁移表表达，避免引入渲染依赖。

### 5.1 收件人视图状态机

| 当前 | 事件 | 目标 | 前置条件 | 副作用 |
|------|------|------|----------|--------|
| `unread` | 打开详情 | `read` | 收件人是当前主体 | 写 `read_at`；未读计数减一 |
| `unread` | 标记已读 | `read` | 同上 | 写 `read_at` |
| `read` | 归档 | `archived` | 同上 | 写 `archived_at` |
| `unread` | 归档 | `archived` | 同上 | 写 `read_at`（若为空）与 `archived_at` |
| `archived` | 取消归档 | `read` | 同上 | 清 `archived_at`；不回退 `read_at` |
| 任意 | 删除 | 视图行移除（物理删除） | 同上 | 删除 `system_notice_target` 行；本体与审计事实保留（不做物理删除会钉死唯一键，见 `NOTICE_CENTER_DB_DESIGN.md` §5.3） |

不可逆约束：`read_at` 一旦写入不再清空（§9.5）。取消归档不允许把状态改回 `unread`，否则「已读」变成可造假字段。

### 5.2 投递状态机

`system_notice_target.delivered_at` 与 `system_notice_delivery.status`：

| 状态 | 含义 | 出边 |
|------|------|------|
| `pending` | 已登记，等待 worker 认领 | `success` / `failed` / `skipped` |
| `success` | 渠道确认送达 | 终态 |
| `failed` | 单次尝试失败 | `pending`（退避后）/ `dead` |
| `dead` | 尝试次数耗尽 | 终态，计入死信指标 |
| `skipped` | 收件人偏好或渠道未配置，不尝试 | 终态 |

### 5.3 非法迁移

写入前必须校验「当前状态 ∈ 允许的源状态」，用一条带条件的 `UPDATE` 表达（`WHERE status = ?`），而不是先读后写。判定依据：

- `unread → archived` 合法，但 `read → unread` 非法，一律返回 §API 契约的 `notice.state.invalid`。
- `dead → pending` 只能由显式重放动作触发（§8.3），不自动发生。
- 本体状态（`draft`/`published`/`withdrawn`）与视图状态分列，撤回（withdrawn）不删收件人视图，只在列表侧过滤并标记，保证 §12 的可追溯。

## 6. 跳转契约

每条消息必须明确：

- 来源域
- 跳转目标
- 是否需要权限

推荐跳转类型：

- `route`
- `external-url`
- `none`

### 6.1 跳转约束

- 没有稳定目标时，允许 `jumpType = none`
- 不允许伪造跳转按钮
- 跳转前仍受页面权限控制

### 6.2 目标校验

- `route` 必须是前端路由表内已知路径：`frontend/src/core/router/componentRegistry.ts` 把菜单 `Component` 字符串映射到懒加载入口，跳转目标校验以菜单/路由已注册为条件，不在通知侧新建第二套路径清单。
- `external-url` 只允许 https、只允许白名单主机（配置项见 §17 OD-3），落库前校验，渲染前再校验一次。
- 源域失效（目标记录已删）时不回收消息，跳转由目标页自己返回空态；通知侧不做「目标存在性」跨域查询——那是 §3.3 的反向依赖。

## 7. 渠道与传输模型

### 7.1 渠道矩阵

| 渠道 | 本文是否定义 | 仓库现状 | 结论 |
|------|--------------|----------|------|
| `in_app` | 是 | 无实现，但只需写 `system_notice_target`，无外部依赖 | V1 唯一必须可用渠道 |
| `email` | 是（接口） | 后端无 SMTP 依赖 | 只定义契约与配置位，不承诺 V1 落地 |
| `sms` | 是（接口） | 无云厂商依赖 | 同上 |
| `webhook` | 是（接口） | 无 | 出站走 HTTP client，需 SSRF 约束 |
| `websocket` | 是（接口） | 后端无 WS 基础设施 | 登记为运行态缺口，见 `NOTICE_CENTER_TEST_PLAN.md` §5 |

### 7.2 渠道抽象

```go
// backend/modules/platform/notice/channel/channel.go（待实现）
type Channel interface {
    Name() string
    Send(ctx context.Context, msg *Message) error
}
```

签名与 `NOTIFICATION_SERVICE_DESIGN.md` §3.2 保持一致，避免同一能力出现两套接口。约束：

- `Send` 必须幂等（§8.2）：入参带 `deliveryID`，渠道侧用它做外部去重。
- `Send` 不得阻塞请求路径：站外投递一律由 worker 执行（§8.2）。
- `Send` 失败必须返回 error，不在内部吞掉：与 `backend/pkg/maintenance/maintenance.go` 的 `Task.Run` 约定同形（该文件 §Task 注释明写「should return an error instead of logging-and-continuing so failures stay measurable」）。

### 7.3 出站投递单元

一个投递单元 = `(notice_id, recipient_id, channel)`。`in_app` 不写投递账本，直接更新视图行的 `delivered_at`；其余渠道写 `system_notice_delivery`。理由：`in_app` 的送达与视图创建是同一件事，为它建一行账本只会制造冗余状态。

### 7.4 用户偏好

收件人可按渠道关闭非 `critical` 消息，偏好载体是 `system/config` 的设置面还是独立表，登记为 §17 OD-4；本文只锁定「偏好影响渠道选择，不影响消息存在」——被关闭的消息仍然入收件人视图，只是不落站外投递，状态记 `skipped`。

## 8. 幂等与重试

### 8.1 摄入幂等

源域必须提供 `dedupKey`（建议 `(sourceDomain, sourceType, sourceId, eventType, recipientId)` 的稳定拼接）。幂等键落在**本体表**的唯一键 `(tenant_id, dedup_key)` 上；fan-out 的不重复由收件人表的唯一键 `(notice_id, recipient_id)` 保证（两处的列类型与「MySQL 唯一索引不含软删列」的取舍见 `NOTICE_CENTER_DB_DESIGN.md` §5）。

- 命中唯一键 → 不新建，返回既有 id，响应里标 `deduplicated: true`；这是**幂等 2xx**，不是 409。
- `dedupKey` 为空 → 允许，但失去重复保护，摄入侧计数进 `notice_intake_missing_dedup_total`（§14），用于把「忘传」变成可见指标。

### 8.2 投递重试与退避

| 参数 | 默认 | 语义 |
|------|------|------|
| `maxAttempts` | 5 | 含首次尝试 |
| `backoffBase` | 30s | 第 n 次失败后 `base * 2^(n-1)` |
| `backoffCap` | 30min | 退避上限 |
| `attemptTimeout` | 10s | 单次 `Send` 上下文超时 |

重试由 `next_attempt_at` 列驱动，不引入内存定时器。执行载体复用既有后台循环：`backend/pkg/maintenance/maintenance.go` 的 `Registry`（间隔节流 + 重叠保护 + panic 隔离 + 结果指标）与 `backend/cmd/server/main.go:205-212` 的 `maintenance.Run` 开关；理由是该注册表已经是「请求路径外做批量清理/扫描」的既有机制，新建第二条后台循环等于新建第二套调度语义。

### 8.3 死信与重放

尝试耗尽 → `dead`，只保留可读诊断（`last_error` 截断存储，禁止落渠道凭证）。重放是显式管理动作，要求 step-up 验证（§13.2 的 `X-Operation-Token`），且重放后 `attempts` 归零重置还是累加，登记为 §17 OD-5。

### 8.4 至少一次与去重责任

投递语义是 **at-least-once**：worker 崩溃在「已发送、未落状态」之间会产生重复投递。仓库当前没有分布式锁原语（`grep SetNX backend/` 无命中），因此本文不把「恰好一次」列为承诺；去重责任分两层——渠道侧用 `deliveryID`，收件人侧用 §8.1 的 `dedupKey`。

## 9. 已读/未读语义

### 9.1 定义

- 已读 = 收件人视图行 `read_at` 非空。触发点：打开详情接口成功返回后。
- 归档 ≠ 已读的反义；归档行不计入未读。
- 「打开」不等于「读完」，本文不引入阅读时长/滚动比例等信号，避免把 UX 猜测写进契约。

### 9.2 未读计数

- 徽章返回 `(total, byCategory)`，`critical` 单独分桶。
- 计数走 `COUNT(*)` 加覆盖索引（`(tenant_id, recipient_id, read_at)`），不缓存于进程内。理由：跨实例读同一份数才可能一致；`system/config` 的进程内缓存模式（`backend/modules/system/config/setting/setting_service.go:110` 起）针对的是极少变更的配置，与本节的每请求变更不符。
- 上限展示策略（超过 N 显示 `N+`）是纯前端决定，不进后端契约。

### 9.3 批量已读

`mark-all-read` 必须带过滤条件（`category` 可选），且只作用于**当前主体自己的行**。批量入参上限沿用既有实践：`backend/modules/system/audit/audit_service.go:357` 对 id 列表做 `>500 → param.invalid` 的守卫，通知侧同样在绑定层拒绝超限，而不是静默截断。

### 9.4 竞态

并发「标记已读」与「归档」用条件更新解决：`UPDATE ... SET read_at = ? WHERE id = ? AND read_at IS NULL`。`RowsAffected == 0` 不报错，解释为「已被并发更新」，响应仍返回 2xx 并让前端重新拉取计数——这与 `docs/contracts/TENANT_CONTRACT_V1.md` §2.2 记录的 `RowsAffected == 0` 歧义教训相反方向：本处**不**用 `RowsAffected` 区分「不存在」与「已更新」，因为寻址已经由主键 + 收件人谓词完成。

### 9.5 不做的事

- 不撤销已读。
- 不做「未读时间轴」等派生指标。
- 不给平台管理员读他人未读状态（跨收件人读只在 §13 声明的平台面，且只读聚合）。

## 10. 保留与清理

### 10.1 保留参数来自 `system_setting`

对齐既有 retention 实践：`backend/modules/system/audit/audit_service.go:302-313`（`RunOperationLogRetention` 读 `audit.operation_log_retention_days`，缺省 180）与 `backend/modules/system/config/setting/setting_seed.go:65-68`（四个 `audit.*_retention_days` 种子 + `system.setting.remark.*` 说明 key）。

通知中心建议 key（待实现，命名不含连字符以匹配 `backend/pkg/common/http/response.go:29` 的 i18n key 正则）：

| setting key | 建议默认 | 作用 |
|-------------|----------|------|
| `notice.read_target_retention_days` | 180 | 已读/归档视图行的保留期 |
| `notice.unread_retention_days` | 365 | 未读行的保留期（不静默删除未读） |
| `notice.delivery_ledger_retention_days` | 90 | 投递账本保留期 |

### 10.2 清理走维护循环，不走请求路径

清理任务是 `maintenance.Task`，名字建议 `notice.retention`；请求路径永不做全表删除——这是 `backend/pkg/maintenance/maintenance.go` 文件头记录的历史缺陷（「list/export handlers used to run full-table DELETE sweeps inline」）的正面要求。

### 10.3 硬删 vs 软删

- `system_notice_target`：到期**硬删**（收件人视图无审计价值，软删会让 §8.1 唯一键永远占位）。
- `system_notice`：本体软删（`deleted_at`），保留消息事实供 §12 回溯；撤回与删除分离。
- `system_notice_delivery`：到期硬删，`dead` 行额外保留到下一个保留期，便于排障。

### 10.4 归档与合规

需要「审计级留存」的消息（`category = audit`）不参与 §10.1 的清理，或由独立更长的保留 key 控制；该裁定属 §17 OD-6。

## 11. UI 骨架

第一阶段最小 UI：

- 顶部通知入口
- 未读数量提示
- 通知中心列表页或抽屉
- 基础筛选：全部 / 未读 / 分类

必须覆盖的状态（`docs/contracts/PLATFORM_CONTRACT.md` §7.3 与 `AGENTS.md`「页面要覆盖 loading / empty / error / forbidden / submitting」）：

| 状态 | 通知中心的具体形态 |
|------|--------------------|
| loading | 列表骨架屏；徽章数字不闪烁为 0 |
| empty | 区分「无消息」与「筛选后无结果」两种文案 |
| error | 读取失败保留上次数据并显示重试，不把列表清空 |
| forbidden | 入口可见但页面 403 时，徽章隐藏而非显示 0 |
| submitting | 标记已读/归档为乐观更新，失败回滚单行 |

暂不强制：

- 复杂分组
- 高级搜索
- 多栏详情布局

文案全部走 i18n：现有 `app.notice.*`（壳层占位）在实现时收敛为 `notice.*` 命名空间，迁移方式与 key 清单见 `NOTICE_CENTER_API_CONTRACT.md` §8。

## 12. 审计与留痕

至少记录：

- 通知生成来源
- 用户查看
- 用户标记已读
- 用户归档

通知中心本身不是审计系统，但它必须能够回溯「消息从哪里来，被谁处理过」。

机制约束（不是愿望）：

- 全局操作日志中间件对**所有非 GET** 请求落库：`backend/internal/middleware/operation_log_middleware.go:224-231`（`c.Request.Method == http.MethodGet` 直接放行不记录）。因此 §API 契约里所有写端点天然有审计行，无需额外接线。
- 「查看」是 GET，不会自动留痕。本文**不**把逐次查看记入操作日志（会淹没审计面），改为在 `system_notice_target` 上保存 `read_at` 这一事实字段作为最小留痕；更强的查看留痕需求属 §17 OD-7。
- 管理面写动作（撤回、重放、清理）必须显式设审计标题与业务类型：`common.SetAuditMetadata(c, titleKey, businessType)`（`backend/pkg/common/audit.go:31`），业务类型取 `BusinessDelete = 3` / `BusinessGrant = 4`（同文件 §20-23），与 `backend/modules/system/system_modules.go:535`（operation-log cleanup）同形。

## 13. 权限、菜单与 i18n 影响

新增一个平台级页面会同时触碰权限、路由、菜单、i18n 四个面，四个面必须同一批落地（`AGENTS.md`「同步关闭」条款）。

### 13.1 权限 key

| key | 含义 |
|-----|------|
| `platform:notice:view` | 页面准入 + 自己的列表读 |
| `platform:notice:update` | 标记已读 / 归档 / 删除自己的视图行 |
| `platform:notice:publish` | 源域/管理面发布消息 |
| `platform:notice:manage` | 撤回、重放、清理（destructive，需 step-up） |

命名遵循 `platform:<资源>:<动作>`，与既有 `platform:dashboard:view`（`backend/modules/system/seed.go:210`）同层同前缀；`scripts/harness/check-permission-contract.mjs` 的 key 正则只接受 `system|business|platform` 三段式前缀，跳出前缀会在门禁里失去归属识别。

### 13.2 路由组与中间件

- 用户面与平台面都挂在 `/api/v1` 下的 `notice` 组，链路对齐 `backend/modules/platform/routes.go:34-37` 的 dashboard 先例：`TokenAuth → PlatformScopeTenantContext → Casbin`。
- 系统控制台形态若使用 `contracts.SystemProtectedGroup`，其固定链路是 `TokenAuth → PlatformScopeTenantContext → Casbin →（可选 Capability/DataScope/RefreshSync）`，顺序是安全属性而非风格（`backend/pkg/contracts/route_groups.go:137-150`）。
- 业务域若通过 `contracts.ProtectedGroup` 摄入消息，必须清楚该链路**不含**租户中间件（同文件 §13-26 与 `docs/contracts/TENANT_CONTRACT_V1.md` §2.6 记录了这是合同要求，不是遗漏），因此摄入侧的租户归属必须由显式入参 + 服务端判定承担，不得假设上下文里已有租户。
- destructive 动作（撤回/清理/重放）加 `middleware.SecureActionMiddleware()`：校验 `X-Operation-Token`，缺失/过期/不匹配分别返回 403 `auth.operation.verification_required` / `verification_expired` / `verification_mismatch`（`backend/internal/middleware/secure_action_middleware.go:16-53`），并校验令牌用户与当前会话一致。

### 13.3 菜单 seed

菜单项通过 `menuSeed`（`backend/modules/system/seed.go:33-50`，字段含 `Key/ParentKey/TitleKey/Path/Component/PagePerm/Perms/Type/Icon/RouteName/Module/Sort/HideInNav`）落库，`ensureMenuSeeds` 负责幂等、去重与失效菜单清理（同文件 §88-103）。前端 `Component` 字符串必须能在 `frontend/src/core/router/componentRegistry.ts` 找到注册项，否则菜单可见但路由不可达。

### 13.4 i18n

i18n canonical 条目按 `(locale, key)` 落 `system_i18n`，每个 key 必须同时给 5 个 locale（`backend/modules/system/i18n/i18n_seed.go:108` 起的 `canonicalMenuLocaleEntries`；语言集合见 `backend/modules/system/i18n/i18n_helper.go:815` 的 `zh-CN/en-US/ja-JP/ko-KR/fr-FR`）。菜单标题、权限说明、错误 key、页面文案四类都要五语齐全，缺语言在门禁里表现为「读得到 key 但回落到默认语言」，属可测量缺陷。

### 13.5 门禁同步清单

| 变更 | 必须同步 |
|------|----------|
| 新增路由 | `backend/modules/system/iam/permission/permission_workbench.go:622` 的 `requiredAPIRoutesByPermission` 与 `permission_workbench_routes_test.go` 的探针（同文件 `:607-613` 写明漂移由该测试兜住） |
| 新增菜单/权限 | `menuSeed` 清单 + 五语 i18n + `npm run test:menu-contract` |
| 新增表 | `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §3 唯一键登记（`docs/contracts/TENANT_CONTRACT_V1.md` §2.3 要求新增表先登记） |
| 新增文档 | `docs/designs/README.md` / `README.en.md` 索引 + `npm run check:harness-docs -- --strict` + `npm run check:docs-frontmatter` |

## 14. 指标与可观测

指标经既有 Prometheus 面暴露：`backend/pkg/metrics/prometheus.go` 定义 `pantheon_http_requests_total` / `pantheon_http_request_duration_seconds`（HTTP 维度由 `backend/internal/middleware/prometheus_middleware.go` 打点），`/metrics` 端点默认关闭，需 `PANTHEON_METRICS_BEARER_TOKEN` 或 `PANTHEON_METRICS_PUBLIC=true` 才暴露（`backend/cmd/server/main.go:150-156`）。后台任务已有一个可直接复用的维度：`metrics.MaintenanceRunsTotal{name, outcome}`（`backend/pkg/maintenance/maintenance.go` 的 `recordOutcome`）。

通知中心建议指标（待实现，命名保持 `pantheon_` 前缀）：

| 指标 | 类型 | 标签 | 想回答的问题 |
|------|------|------|--------------|
| `pantheon_notice_intake_total` | counter | `source_domain, category, deduplicated` | 谁在发消息、重复率多高 |
| `pantheon_notice_intake_missing_dedup_total` | counter | `source_domain` | 哪些源域没传幂等键 |
| `pantheon_notice_delivery_total` | counter | `channel, outcome` | 各渠道送达/失败/跳过 |
| `pantheon_notice_delivery_duration_seconds` | histogram | `channel` | 渠道延迟分布 |
| `pantheon_notice_delivery_dead_total` | counter | `channel` | 死信堆积 |
| `pantheon_notice_pending_age_seconds` | gauge | `channel` | 最老未投递消息的年龄（唯一能发现「worker 卡住」的信号） |
| `pantheon_notice_retention_deleted_rows` | counter | `table` | 清理是否真的在跑 |

## 15. 与 dashboard / 任务中心的关系

dashboard 不应各自保存另一套消息模型。

正确关系：

- dashboard 只消费通知中心摘要
- 通知中心是平台级消息入口
- 业务域通过统一消息模型向平台暴露摘要

落地约束：dashboard 侧已有治理待办聚合（`backend/modules/platform/dashboard_dto.go:23-24` 的 `OrgGovernanceTaskCount` / `OrgGovernanceTasks`），它属于「工作项摘要」而不是「消息摘要」；实现时不要把这两者合成同一份数据，否则 dashboard 会变成第二套消息存储——这正是 `PLATFORM_CONTRACT.md` §7.1 禁止的聚合层越界。

## 16. 落点约束（既有机制挂点）

本文每条约束都对应一个真实存在的机制，实现者不需要自行发明：

| 本文条款 | 既有机制 | 出处（已读文件） |
|----------|----------|------------------|
| §8.2 后台执行载体 | maintenance registry（节流/重叠/panic 隔离/指标） | `backend/pkg/maintenance/maintenance.go` |
| §10 清理不走请求路径 | registry 文件头记录的缺陷与修正 | 同上，§4-9 注释 |
| §10.1 保留期可配 | `system_setting` + 默认值回落 | `backend/modules/system/audit/audit_service.go:302-313、317-323、325` |
| §12 非 GET 自动审计 | 操作日志中间件跳过 GET | `backend/internal/middleware/operation_log_middleware.go:224-231` |
| §13.2 中间件链路 | `SystemProtectedGroup` 顺序即安全属性 | `backend/pkg/contracts/route_groups.go:137-150` |
| §13.2 step-up | `X-Operation-Token` 校验 | `backend/internal/middleware/secure_action_middleware.go:16-53` |
| §13.5 权限↔路由漂移 | workbench 绑定表 + 引擎探针测试 | `backend/modules/system/iam/permission/permission_workbench.go:602-652` |
| §9.3 批量上限 | `>500 → param.invalid` 既有实践 | `backend/modules/system/audit/audit_service.go:357` |
| §14 指标暴露 | Prometheus 注册与开关 | `backend/pkg/metrics/prometheus.go`、`backend/cmd/server/main.go:150-156` |
| §13.4 五语 i18n | canonical locale 条目 + 支持语言集合 | `backend/modules/system/i18n/i18n_seed.go:109+`、`i18n_helper.go:815` |

## 17. 开放决策

以下不是本文可以自行裁定的产品/架构选择，逐条给出选项与建议，由维护者在「质量门禁策略 / 验收」触点拍板：

| 编号 | 议题 | 选项 | 建议与理由 | 决策人 |
|------|------|------|------------|--------|
| OD-1 | 与 `NOTIFICATION_SERVICE_DESIGN.md` 的 `system_notification` 草案表如何收敛 | A 通知中心三表吸收渠道投递，废弃该草案表；B 通知中心只存本体+视图，渠道表归 notification 服务 | 建议 A：两份都未实现，同时落库必然产生两份收件人状态；渠道抽象仍以 §7.2 接口共享，不牺牲对方语义 | 维护者 |
| OD-2 | 多语言消息由谁渲染 | A 源域摄入时按收件人语言渲染成品；B 存模板 key + 变量，平台侧渲染 | 建议 A（V1）：平台侧模板引擎在 §2 已列为非目标；B 需先补模板治理合同 | 维护者 |
| OD-3 | `external-url` 白名单载体 | A `system_setting`；B 独立渠道配置表；C 环境变量 | 建议 A：与 §10.1 的 retention 配置同面，避免新增配置存储 | 维护者 |
| OD-4 | 用户通知偏好的归属 | A `system_user.preference_json`；B `system/config` 用户级设置；C 独立表 | 建议 C 之外先问「V1 是否需要偏好」：§7.4 只承诺不影响消息存在，可推迟到有站外渠道时 | 维护者 |
| OD-5 | 死信重放后 `attempts` 是否归零 | A 归零（新一轮生命周期）；B 累加（累计成本可见） | 建议 A + 保留 `total_attempts` 观测列；语义更贴近「运维动作」 | 维护者 |
| OD-6 | `category = audit` 消息是否豁免清理 | A 豁免；B 独立更长保留 key；C 同等清理 | 建议 B：豁免会让「未读审计消息」无限堆积 | 维护者 |
| OD-7 | 逐次「查看」是否需要审计 | A 只存 `read_at`；B 每次 GET 额外写一条审计 | 建议 A：§12 说明 B 会淹没审计面 | 维护者 |
| OD-8 | 摄入面是进程内 service 还是内部 HTTP | A 进程内（模块组合根注入）；B 声明式内部 HTTP 端点 | 建议 A：业务域与平台域跨模块调用已有 `pkg/contracts` 端口先例（`backend/cmd/server/main.go:172-176` 的 authuser 端口注释），新增 HTTP 面会多一套鉴权与幂等 | 维护者 |

## 18. 完成定义

第一阶段完成标准：

- 有统一消息模型
- 有已读/未读语义
- 有平台入口
- 有最小跳转契约
- 有基础审计留痕
- 有 §13 的四类同步（权限/菜单/i18n/文档）与 §14 指标
- 至少一个真实源域通过 §8.1 的幂等摄入接入

不是以「功能像企业微信」作为完成标准。

验证口径以 `NOTICE_CENTER_TEST_PLAN.md` 为准；本文档包全部四份文档的 `status` 在该计划落地前保持 `Draft`。
