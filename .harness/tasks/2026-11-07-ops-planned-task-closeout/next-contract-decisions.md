# P0 下一阶段合同草案

状态：原方案草案保留为历史，未批准。下方 checkpoint-history candidate 已开始实现业务迁移35和现有 DTO 的局部扩展，独立评审/最终验收仍开放；公开路由和 Base 锁未改变。O4 草案尚未实现。

## O2：历史步骤状态

当前事实：DeployTaskStep 每个 task/host/step 只有一个可变检查点，运行时递增 attempt；日志已单独保存写入时的 attempt。DeployTaskEvent 没有 attempt 字段，步骤更新与事件追加不是同一事务。日志存在或事件顺序不能证明某次尝试的完整步骤状态。

最小方案：保留当前恢复检查点，新增独立的业务步骤尝试记录，键为 task/host/step/stepAttempt。步骤 attempt 与 host execution attempt 分别命名，不假定二者相等。开始、结果、状态版本与检查点在同一事务内提交；历史读模型只返回安全身份、状态、时间、已登记错误码，不返回命令、条件、配置、凭证或错误原文。任务/主机授权先于 attempt 查询校验。

关键评审：锁顺序与当前 task/host lease/CAS 的兼容性、重复回写规则、暂停/取消/重启的尝试语义、历史分页和响应预算。新增历史不能改变恢复/重试策略。旧步骤不回填推测状态，继续显示历史状态不可用；旧日志 attempt0 不自动关联。

验证集合：真实 MySQL 并发开始、重复完成、事务失败回滚、暂停/恢复、重试、重启；历史记录精确归属步骤尝试；旧库/空库/重放迁移保留历史；跨任务/主机/用户拒绝且不泄露存在性。浏览器验证当前/历史明确区分、来源切换不混日志、撤权清空与链接恢复；desktop/mobile、light/dark。完整 CGO race、正式装配/漂移、独立评审和原 O2 验收继续必需。

## O3：复杂变更与回滚

已验证的 Deploy 冻结计划和 K8s image review 不能扩大解释为任意配置、manifest 或旧版本恢复。下一实现先选择一个有权威 provider 的资源类型：读取实际版本与受权配置投影，生成进入 DTO 前已遮蔽秘密的差异，对执行对象使用原生版本比较。

Deploy 参数无实测来源时保持 unknown。Argo 本地 release 记录不等同于实际 revision/history；rollback 绑定 provider 可恢复的确切版本与资源集合。脚本、多步和不可逆动作先定义可信逆操作，不能重放 forward 命令充当回滚。缺实际端点时仅验证本机协议/契约，不关闭真实资源验收。

## O4：历史与保存视图

当前事实：Loki 查询和结果只存在组件内存，切换源或失败清空；没有持久化历史 provider。官方 Base 条件编辑修复待正式发行消费。业务域仍遵守 Base V1 单租户合同，不能自行宣称多租户隔离。

最小分段方案：先提供 actor-owned 视图元数据合同，保存 datasource ID、UTC 时间模式、limit、视图类型。原始表达式、标签值、响应、凭证和自由文本不写数据库、浏览器存储、URL 或审计载荷。历史只记录发生时间和安全结果状态；历史写失败不改变查询成败。恢复视图后重新授权数据源，由用户明确运行。

只有服务器能生成经白名单验证的无秘密结构化表达式时，才扩展为保存脱敏表达式；客户端 AST 或正则替换不能作为可信脱敏。所有读写由认证上下文确定 owner，重新验证 datasource DataScope，实时撤权后不暴露元数据。用户/租户双重隔离先获得 Base 明确业务合同与正式消费；compat actor 隔离不能替代原 O4 tenant 验收。

关键评审：生命周期/保留期/条数与字节预算、删除语义、撤权后的列表响应、名称/错误秘密处理、业务单租户到显式租户合同迁移。新增权限点/路由/迁移/seed/i18n/smoke 一起登记，不借 list 权限扩大写权限。

验证集合：actor/tenant 读写隔离与撤权；恶意自由文本/表达式/响应无法持久化；超限拒绝；存储失败不改变查询结果；恢复视图不自动运行；空库/升级/重放/保留历史。审计脱敏、desktop/mobile/light/dark、键盘/空/错误/拒绝/恢复及真实数据源验收继续必需。

## 执行顺序

1. 收口当前 SSH/标签预算验证；不更改运行中验证的 Go/SQL 源码。
2. 独立评审 O2 历史合同，先做事务/并发失败回归，再实现业务历史读模型/UI。
3. O3 选择有权威来源的下一资源类型；无来源的参数不制造差异。
4. O4 明确持久化/租户合同，先做无秘密元数据，再做可验证的结构化表达式。
5. 正式 Base 发行/消费解决共享组件、权限、归档依赖；外部环境与签署按原父任务验收，不用本地 fixture 替代。

## O2 bounded checkpoint-history implementation candidate

User renewed execution. Reuse the recorded root Generator fallback. Prior installed agents returned503, Claude timed out and Gemini is unauthenticated; no independent approval is inferred and release/final acceptance stays gated. Refine the previous per-attempt draft to an append-only checkpoint snapshot journal: capture only committed safe checkpoint identity/status/version/times, atomic with the existing step update. Do not change engine transitions, attempt increments, lease or recovery policy. Historical UI shows the last recorded checkpoint in each step attempt, not a reconstructed execution timeline or terminal guarantee. Legacy rows are not backfilled. Expose at most500 latest step/attempt snapshots in the existing authorized Task DTO, filter by its visible hosts and actual step ownership, and report truncation. No new public route/permission or error/command/config/secret exposure. Existing all-attempt logs use current rail; an explicit attempt uses its recorded history with capture-time context.

Verification: real MySQL failed-before lost-history regression, atomic write-failure rollback, serialized concurrent updates, pause/retry and unknown legacy cases, response budgets/host-step scope and private-field exclusion; migration empty/replay/retained down; frontend history/source/deep-link/denial regressions and rendered desktop/mobile light/dark. Full hash-stable CGO race/vet/build, official locked assembly/contract/drift and task/evidence gates follow. Schema35 is a business-owned additive local candidate; production migration, formal review and full original acceptance remain open.
