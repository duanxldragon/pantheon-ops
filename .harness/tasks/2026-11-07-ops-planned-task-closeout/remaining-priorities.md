# 按优先级继续执行

更新：2026-11-14。**还有 6 个 P0 父任务未关闭：O2、O3、O4、D1、D2、D3。**

## 最新执行接续：正式版本已独立复核

本轮 fresh GitHub 查询确认最新稳定发行仍为 `pantheon-base-v0.18.8`，tag 对象与活动锁文件均指向 `34428814a9efc997fd93d4434406a53df6460323`，四项官方资产摘要相符。独立标准 shared drift 通过：857 项共享文件、856 项完全一致、1 项既有 frontend package scripts allowance；288 项业务文件和14项迁移按声明校验。Overlay contract 与66项回归通过。下文 r5/r6 活动版本18.6的安装失败记录是历史，不再代表活动版本。

当前按[执行计划](../2026-11-14-ops-latest-foundation-and-plans/current-execution-plan.md)推进 O1 FileDist、O5 日志、O6 摘要与 P0 公共运行态矩阵，最终业务变更后重新装配与验证。共享调度授权、完整首页时间范围和发布文档资产缺口按 Base-first 处理；六个 P0 父任务和最终人工验收仍开放。生产 D4 签署与 O7 既有延期不因代码验证自动关闭。

## 最新授权接续

正式18.8新候选 `.tmp/ops-v0188-install-r6` 已通过真实GitHub资产获取、标准重建/Go依赖加载/overlay/drift；836后端和165前端owned文件与既有已验证源码一致。新165项计划在60次写入尝试后再次被同一tenant测试文件拒绝，全部165个preimage和index恢复，活动仍18.6。维护者已授权提权，但本会话工具策略禁止提权调用，需允许提权的执行环境。Base53个发行展示名已规范，3个旧命名标Legacy，55个tag对象与资产/发布状态未改动，Latest18.8。后续publisher规范已在Base本地修正，未提交或发布新版本。六个P0及最终验收仍未关闭。

## 当前接续状态 — 2026-11-14

正式 GitHub 最新稳定版为 **v0.18.8**，四项资产校验及完整候选 `.tmp/ops-v0188-final-r5` 装配已通过；活动安装因 Windows 目录写入拒绝回滚，**活动 lock/report 仍为 v0.18.6**。未使用本地 Base/Ops Git 对比或手抄共享源码。

- 菜单 IA、环境配对迁移000038、服务三页拆分与实例创建/分页已实现；AutoMigrate=false 的隔离空库启动到迁移38。自定义角色需重新授权，host binding 另需 CMDB host-view。
- 20 个既有 CMDB 权限键已注册；隔离实际浏览器验证 key-only 授予、原会话撤权和 CSV 权限分离。旧 R3 失败及夹具保留，未重放。host/group 详情仍等待正式上游安全 static matcher。
- O6 四个 bounded recorded-state widgets 已集成；亮/暗、1440/390 的空态渲染无溢出/页面脚本异常。非空、实时 workload/alert、时间范围和全状态矩阵仍未验收。
- 候选前端 lint/type-check/build、414/414 units、24/24 实际浏览器 smoke 通过。浏览器采用私有 localhost IPC/真实 Chromium CDP 适配，`nativeRunnerPassed:false`，完整 Go race 结果以新证据为准。
- D5：升级候选构建时扫描9个链接 findings、49个 review errors/41文件；活动目录完整扫描为19个链接 findings、93个 review errors/67项。树和扫描范围不同，历史计数不替代本轮证据。共享 packaging 缺口回 Base，不能在 Ops 改共享文档或编造审批补绿。
- **O2/O3/O4/D1/D2/D3 六个 P0 父任务未关闭**。O1 FileDist owner/stable target/scheduling、O5 audited download/redaction/live-source、O6完整矩阵、D4生产RPO/RTO与容量签署仍开放。Velero 继续后置，O7 deferred。

当前证据：[正式候选与规划执行](../../evidence/2026-11-14-ops-latest-foundation-and-plans/summary.md)。下方旧检查点保留历史，不覆盖本节。

## 2026-11-12 进度同步

本次同步只登记已验证事实，不新增关闭结论；父任务仍为 6 个 P0 开放，正式 Base 发行消费与最终视觉/功能签署未完成。

- **CMDB 权限键 R3 实际执行结果 = failed**：69 个请求、57 个断言通过；失败在 `exact current CMDB permission keys without wildcard`，即公开把角色35撤为空权限并成功回读后，原会话 `/auth/me` 仍不符预期空 CMDB 权限集合。尚不能判定为产品撤权缺陷还是 oracle 问题；禁止重放 mutation。R3 roles35–38/users26–29、R2 roles31–34/users22–25 与 host1 夹具全部保留。
- **O2 共享权限产品修复**：design-r3 已获 CODE design APPROVE + ARCH design APPROVE/CLEAR（仅设计）；Ops 隔离草案已写完 11 个 owned 文件（16 keys/27 routes、init-first、六个静态 GET guard），preimage RED27 → draft GREEN0，完整 native compile/tests 等新正常 Base 候选装配。
- **Base 新门禁事实**：候选 74d43c72 正常生成、1166 源文件核对通过；完整 CGO race 50 个测试包/2197 个测试通过，1 个真实 S3 因端点未配置显式 skip；前端五门禁 402 单测通过。74 尚未消费。
- **Base 权限缓存撤权修复（2026-11-12 计划）**：新增 [auth-permission-cache-repair plan](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-non-velero-2353/checkpoint.md)，最小方案为移除 `Runtime.GetUserPerms` 的 Redis 权限数组读写，只走权威数据库查询；failed-before 证据、源码与原生测试尚未产出。
- **O4**：complete-positive runner r1 独立 CODE 发现三项真实 runner 缺陷（claim 仅绑 activation、build/run 日志无界、超时/启动失败丢日志），r1 不可运行，独立 r2 进行中，failed-before witness 保留；六严格负向＋两完整正向实际运行仍未完成。
- **D2**：公共 adapter 收口进行中（严格 TS 与 27 纯 oracle 通过），未执行 HTTP/SQL/SSH，不声明 D2 完成。
- **Velero**：按维护者排序继续后置，不新增实施、复审或远程恢复。 其中不少子项已有实现或通过验收，不能把“父任务未关闭”理解成“完全没做”。

## 当前接续状态 — 2026-11-11

最新事实以[非 Velero 收口检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-non-velero-2353/checkpoint.md)为准：Base完整CGO race50包/2197测试通过，1个未配置真实S3skip；6a正常组装及前端五门禁402单测通过。O2设计双审通过，Base/Ops隔离产品实现已启动，新的正常候选待最终源码。CMDB r3实际69请求/57已通过断言，all→self和权限隔离通过，但公开撤空权限后的原会话/auth/me校验失败，正定位产品或oracle原因；新8个owned IAM对象和旧失败均保留。O4完整正向runner独立评审发现一次执行/日志边界三项问题，独立r2正在修复，D2公共adapter同时收口。**下表旧启动8case中的2项GREEN仅证明局部退休清理，不证明完整启动；新6严格拒绝+2完整正向仍开放。** 正式消费、公共产品组合、整体验收和最终视觉功能签署未完成。

**维护者最新排序：Velero 排到最后，先完成其余 P0。** 已封存的 Velero 有限诊断集成包和维护者政策决定保留，暂停新增 Velero 实施、复审与远程恢复。当前优先顺序：Base/Ops 标准启动校验顺序修复 → 新候选正常配对 → O2/O3/O4、公共权限/租户/审计与 D2 出站策略验收 → 正式底座发行消费及相应任务收口 → 最后恢复 Velero。D1 中非 Velero 的外部集成仍继续；后置不等于豁免或关闭 D1，也不改变最终 P0 门禁。

最新证据入口：[非 Velero 优先检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-non-velero-priority/checkpoint.md)，预览运行说明见[实际前后端预览检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-manual-preview/checkpoint.md)。[16:56 后台进度检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-status-1656/checkpoint.md)、[Base 完整门禁与 Event 实际关联检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-lifecycle-gates/checkpoint.md)、[运行态定位与 CMDB 独立评审检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-runtime-findings/checkpoint.md)和[已评审继任候选检查点](../../evidence/2026-11-07-ops-planned-task-closeout/verification/continuation-20261111-reviewed-successor/checkpoint.md)保留历史。独立评审和 SSH 已可用；本地 Vite 可通过仅关闭可选网络盘探测的启动器运行，无需管理员。

| 范围                | 当前已验证事实                                                                                                                                                                                                                                                                                                                                                                                                                                                          | 接下来仍须完成                                                                                                                                                                                                                                          |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Base / D3           | 日志隐私source12及标准入口preflight source6-r2双审通过。最新完整CGO race50 tested packages/零FAIL/零race warning（1930.422秒）、vet/build全部通过，533源码前后稳定；前端三门禁461源码及8依赖pin匹配可复用。canonical新候选50ced已生成，Root独立descriptor/archive/1154当前源码/批准map校验PASS。                                                                                                                                                                        | Ops正常配对、正式发行消费及最终验收。本轮完整race未配置真实S3，不声明零provider skip；既有未变S3真实单项证据保留。                                                                                                                                      |
| O4                  | 抽屉单属性修复81项测试/48截图及双审通过。正常o4s3配对、实际owned/same-tx/rollback、四包完整scoped CGO race和标准入口8case均通过，双审关闭ARCH-TX-01/ARCH-LC-01范围内缺陷。独立服务18127/PID28876迁移37/dirty0，真实source创建/provider query和历史/视图隔离、恢复、修订与外租户DELETE无效果已实际通过。R4真实唯一audit及角色撤权通过。o4fb完整组装完成，824backend与native-tested阶段字节完全一致。canonical locale六文件R2及有效RED→GREEN/unitTS已双审批准。预览保留。 | R4在会话撤销后预期401断言失败；只读审计确认后续四次均denied且三业务行保留，具体响应分类正在定位，CMDB未进入。R4不重放，精确残留待新审查清理。需冻结后新完整组装验证五语言生成内容。旧失败及已消费请求保留，完整产品/provider/正式消费及最终验收仍开放。 |
| D1 / Velero（后置） | 维护者有限诊断规则决定、新native7读取及已封存实际集成successor保留；18本地tests/Pyright/607保护pin通过。                                                                                                                                                                                                                                                                                                                                                                | 按最新排序暂停新增实现、复审和remote recovery。其他任务后恢复新版本双审、全部原生门禁、新鲜验证、S3备份恢复及精确清理；Event始终不授归属或操作权限。                                                                                                    |
| CMDB 权限键         | 三个既有键/四条静态路由的 provider 注册修复通过 scoped race/vet/build。独立 CODE APPROVE / ARCH WATCH，无范围内源码缺陷；综合 COMMENT，报告已保存。                                                                                                                                                                                                                                                                                                                     | 新候选中公共 IAM 授予/撤销、CSV 能力分离、数据范围验收。详情及其他动作的映射缺口保留；不得把局部修复写成全 CMDB 完成。                                                                                                                                  |

O2/O3 已有源码及真机局部证据继续保留；D2 产品出站策略组合和六项共用的公共鉴权/审计、正式消费、最终功能/视觉验收仍开放。以下 2026-11-10 表与更早检查点均作为历史，不覆盖上述当前状态。

测试统一使用已授权服务器 `115.190.251.143`。此前失败/成功回执见 [2026-11-10 接续证据](../../evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/progress-continuation-20261110.md)与[较早进度证据](../../evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/progress-20261110.md)，均保留历史。

## 历史 P0 交付状态 — 2026-11-10

| 任务                | 已完成或已验证                                                                                                                                                                                                                                                                       | 尚未完成；关闭条件                                                                                                                                                                                        |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| O2 部署执行工作台   | 最终事务及取消修复已独立 **CODE APPROVE / ARCH CLEAR**，177项真实MySQL race及3项取消边界通过。v12依赖修复已完成封存，5522源码、401工具、10个非OS DLL固定；七命令、15 Python＋15 Go race通过。最新独立 **CODE APPROVE**。旧v9真机12/12仅作历史证据。                                  | v12最终ARCH仍在运行；批准后执行当前源码VM/冷重启。产品HTTP/IAM/审计、原模板/渲染、正式Base消费及最终验收开放。                                                                                            |
| O3 变更预览与回滚   | Argo 审核版已真机部署，四个核心 Pod Ready/零重启。真实 ArgoCDService v4验收 **通过**：两次 Sync终态、历史回滚恢复实际 Service、撤权、丢回复保持 Unknown、同票无重复提交、新 reconciler 恢复终态、四次实际 mutation及精确清理；源码稳定，CGO race及远端退出均0。                      | Argo 公共 HTTP/完整 Base 当前身份/租户/IAM/审计/公网策略组合仍需验收；Deploy/K8s 全部差异、参数、脚本与恢复场景，正式消费及最终验收仍未完成。                                                             |
| O4 可观测查询工作台 | 复制实际返回字节的哈希绑定修复已完成，**207/207**回归、27 focused通过；原反例已先红后绿，旧资产保全。                                                                                                                                                                                | 新请求最终CODE在运行，需重新绑定最新源码的ARCH结果；随后真实成功装配。C–F历史/保存视图/用户＋租户后端、迁移37、前端、两租户真机与正式消费尚未交付。                                                       |
| D1 外部集成验收     | VM/四类观测/K8s、真实ArgoCDService局部证据保留。Velero v5已封存并获独立 **CODE APPROVE / ARCH CLEAR**：260唯一回归、52/52伪造记录拒绝。                                                                                                                                              | v5实际diagnostic失败：Kubernetes原生Deployment revision注解未被接受；无新增别名或S3探测，失败检查完成。v6正在精确修复该兼容性。controller Ready、BSL、Backup/Restore、持久化/清理及公共产品组合仍未通过。 |
| D2 出站网络策略     | K8s/SMTP 网络边界修复已局部验证；测试服务器上现有 Base validator 的 allow/deny/私网/协议/端口等 **15/15** 场景通过。                                                                                                                                                                 | 通过真实产品路径验证部署策略、重定向/DNS、凭证转发和错误脱敏；产品审计、provider RBAC 与授权私网例外；正式 Base 合同消费。validator 通过不能关闭产品策略验收。                                            |
| D3 底座继承与权限   | 官方Base v0.18.6锁定重建/漂移已验证；共享权限/租户接口已有局部实现。Base前端 **240单测＋20生成器测试、lint/build通过**；历史完整CGO race **50包/2154测试通过、1项S3跳过**。后续独立临时环境的现有S3真测试已通过 **CGO race、1/1、零跳过、源码稳定**，对象/桶和临时实例精确清理通过。 | 真实页面/功能验收、正式发行→Ops正式消费→兼容重建；**CMDB仅权限key授予/撤销验收**；独立继承边界复审及后续Ops最终全量门禁。现有v0.18.6消费不能证明尚未发行的新修复已消费。                                  |

六项 P0 共用 Base 发行消费、独立整体验收与最终功能/视觉验收；O3 与 D1 的 Argo 验收也有重叠，不应把各行当作互不相关的工作量。历史全量 CGO race 通过只证明当时稳定源码，后续变更完成后仍需执行最终稳定源码门禁。尚无完整 P0 独立批准或最终签署。

## 非 P0 与后续事项

| 任务                          | 尚未完成                                                                                                                        |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| O1 统一上下文与目标选择（P1） | 其他消费者的稳定 ID/归属接入；FileDist 数字 owner、源文件授权、目标重验和调度时实时授权。目前受限角色仍被拒绝，不能称完整实现。 |
| O5 任务与 Pod 日志体验（P1）  | 审计下载、脱敏、真实 Pod/VM 来源的完整撤权/重连验收。                                                                           |
| O6 运维首页（P1）             | Base Dashboard 正式消费、业务 widget 数据接入与界面验收。                                                                       |
| D5 文档与治理收口（P1）       | 正式发行消费后修复现有 19 个继承文档链接和 8 个旧审查登记缺口，并重跑治理门禁。                                                 |
| D4 生产交付准备               | 备份恢复演练及可接受数据丢失/恢复时间、容量和升级窗口（G2/G3）签署。测试服务器授权不等于生产交付签署。                          |
| O7 关系拓扑（已延期）         | 等待原 CMDB 关系模型成熟度门禁；维持延期。                                                                                      |
| FileDist 历史数据             | 已有明文凭证快照的独立迁移与凭据评估；新快照修复没有清理历史数据。                                                              |

当前执行（2026-11-10 10:07 UTC）：O2 v12已封存并获CODE批准，等待ARCH后运行当前源码真机；O4复制字节修复已封存，最终CODE在运行，ARCH需绑定最终源码；Velero v5获双审批准后diagnostic发现原生控制器注解兼容性问题，v6正在修复。S3现有真测试1/1零跳过、临时环境精确清理已完成。六个P0父任务未新增关闭；耗时主要来自验收工具实际缺陷的修复与复审，O4 C–F及公共产品组合仍是业务交付缺口，不能据此估算完成百分比。测试环境、权限和租户范围已获授权，不再等待用户输入。

## 历史检查点

以下为以前的状态记录；与上表不一致时以上表及本次进度证据为准。历史568 packet错误/108警告仍是已登记旧债，不因本次说明更新而豁免。

下一阶段合同草案见 [next-contract-decisions.md](./next-contract-decisions.md)：明确 O2 步骤尝试记录与 O4 持久化/隔离的事实、最小方案及验证集合。O2 已实现局部检查点候选且已验证，O4 仍为未实现草案；独立批准及父任务关闭条件不变。

上轮 SSH/标签预算检查点：O2 本机真实 SSH 协议/业务落库13项回归，连接/会话取消与暂停/取消误判修复；O4 每行6标签/80字符预览及明确省略提示，56项前端回归、2项浏览器验收。完整源码哈希稳定 CGO race、正式重建与漂移已通过；真实 VM/端点、历史状态、Base 正式消费和最终评审仍开放。

最新 O2 局部闭环：七项真实 MySQL 历史/并发/迁移测试，61项前端回归，18项部署模块 smoke、2项历史浏览器和完整哈希稳定 CGO race通过。移动端约474px内容裁切已补断言并修复。下一步仍需完整动作/重启/来源撤权/真实 VM，以及正式 Base 消费和独立最终验收。

## P0 acceptance-host checkpoint — 2026-11-09

本轮 P0 范围是 O2/O3/O4/D1/D2/D3（含 CMDB key-only 授权依赖）；上表 P1/D5 等历史分类不缩减该范围。维护者已统一授权测试服务器 `115.190.251.143`，不再要求重复提供主机或授权。已有真机部署和 Prometheus/VictoriaMetrics/Loki/Tempo provider 证据通过；本轮 K8s Service 的真实 MySQL/协议 race、76 项前端回归、lint/构建与烟测覆盖修复通过。旧集群迟到撤权结果清空新集群界面的两项缺陷已先红后绿验证。

当前 SSH 验收执行被会话网络限制拒绝（WinError 10013）；验收配置已改用正式 SQL 迁移，新的 Linux/前端产物与 v5 runner 准备好，但尚未真机运行。完整 P0 功能、正式 Base 发行消费、最终源码 race/独立评审/视觉功能验收仍未关闭。完整逐项缺口与可接续文件见 evidence/verification/p0-integration-20261109/continuation-20261109.md；不得将局部通过写成父任务完成。

## O4 follow local implementation checkpoint — 2026-11-09

已新增明确标注的 Loki 跟随轮询：每次请求结束后间隔5秒刷新最近15分钟；停止、条件/来源变更、卸载、错误和撤权终止轮询，有界快照覆盖旧结果。17项 Loki/86项 operational 单测、完整lint/build、正式锁定重建/漂移/边界通过。真实服务器浏览器矩阵已扩展但未运行；当前SSH执行仍受网络限制。查询历史/保存视图与租户契约、其他来源/完整 partial/live-tail 验收、独立评审和正式Base消费仍未关闭。

## Current remote acceptance checkpoint — 2026-11-09

This checkpoint supersedes earlier network-restricted/race-running notes below. The existing authenticated SSH transport now connects to the authorized test server115.190.251.143. Remote v5 failed on a preserved AutoMigrate-era database (duplicate page_perm); the independent v6 test database completed official migrations36/dirty0 without schema forcing or changes to the old database. Remote v6 browser acceptance ran18 cases:7 pass/11 fail. Four explicit CMDB API-policy cases, two K8s image-preview browser cases and frozen rollback draft pass; these do not certify key-only grants or live K8s providers.

Final complete Windows CGO race passes with713 stable Go/SQL/mod/sum hashes, exit0 and1308.532s. Required MySQL33316/Redis26316/migration DSN were configured. Later changes are frontend/harness only. Loki17 and operational86 units, full frontend lint/prebuild/type-check/build, official v0.18.6 reconstruction/overlay/drift pass. The new browser repairs also pass86 operational cases, full lint/build and official assembly/drift;317 business ownership records reconciled with unchanged release lock and shared source.

Failed remote traces show10–15s asset loading and several30s total scenario deadlines expiring during cleanup; complex specs now have90s total budgets while individual expectations retain10s. The acceptance-only reverse proxy now sends/flushed SSE headers/events incrementally; its protocol check receives the first event in0.016s before upstream completion and preserves final body and403. It is not a product SSE acceptance claim. Deployment edit loading is isolated from translation/router callback identity changes; Tempo uses the existing localized common.detail action instead of a missing common.view key. Original failures are retained.

Exact v6 fixture cleanup is prepared using captured IDs/run tokens and formal API identity checks, with bounded list absence verification. v7/v8 cleanup harness failures are retained (legacy GET-not-found500 semantics and the BizScope /list route); both runners cleaned their own remote backend/web processes and forwarding. v9 executes this correction before the updated18-case browser matrix; results remain pending. Old test databases/evidence are retained. No production, commit, push, publishing, baseline replacement or gate waiver.

Impeccable rendered inspection covers v6 Deploy conflict desktop1440x900/mobile390x844: visible alert, retry, collapsed mobile navigation and readable wrapped metadata; bounded horizontal table scrolling remains. Shared ExecutionStepRail raw status/localization and dark styling are still release dependencies. Final visual/functional acceptance and independent CODE/ARCH approval are open; the installed review role lanes returned503 and cannot be replaced with author self-review. O2/O3/O4/D1/D2/D3 remain open.

Evidence: verification/p0-integration-20261109/{continuation-full-race-result.json,remote-migration-v6-result.json,remote-p0-browser-v6-result.json,proxy-sse-check.json,browser-repair-*-result.json,browser-repair-ownership.json,remote-v6-owned-fixtures.json}.

## 2026-11-14 official-release execution checkpoint

Official latest stable GitHub release pantheon-base-v0.18.8 was acquired and checksummed; final standard overlay rebuild r5 succeeded at commit34428814a9efc997fd93d4434406a53df6460323. Active installation was denied by Windows directory permissions and rolled back; the active lock/report remain v0.18.6. Six original P0 parents remain open until their actual acceptance gates pass. Menu/environment planning is implemented; O6 recorded-state subset is implemented, with live workload/alert and temporal/state matrix acceptance retained. CMDB20 safe key bindings and66/66 overlay regressions pass; host/group detail need an upstream static matcher. Final stable-source full CGO race passes70 packages/3583 tests, including63 cached packages/3105 tests and7 freshly executed packages/478 tests;6 external tests remain skipped. Frontend lint/type-check/build and414 units pass;24 actual browser tests pass using isolated localhost IPC/real Chromium CDP, with nativeRunnerPassed:false. Dashboard rendered evidence covers empty samples only; complete UI/provider acceptance and empty-schema double boot remain open. Current evidence: .harness/evidence/2026-11-14-ops-latest-foundation-and-plans/summary.md. Earlier failure receipts and fixtures remain preserved.
