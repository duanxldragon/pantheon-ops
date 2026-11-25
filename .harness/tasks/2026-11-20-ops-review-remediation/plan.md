# 11月20日审查整改与真实交付计划

本计划产出1个父任务和17个子任务；2026-11-25 T04/T16 收口时又拆出 T18/T19 两个后继包承接残余。最新维护者指令优先验收现有候选的V1–V8和旧十包未验收事项；Pantheon Ops本地启动，从CMDB开始验证全部在册业务功能，授权服务器115.190.251.143只作为SSH部署目标。父任务与T16已进入in-progress；Claude负责执行内容与失败修复，Codex负责审查和实际结果验收。任务实施覆盖业务域，平台层和系统域的共享缺口通过Base正式release与Ops重建消费闭合。

最新入口见[本地候选准备](./CLAUDE_RUNTIME_WRITE_NOW.md)与[CMDB→SSH派发](./CLAUDE_CMDB_SSH_FIRST.md)。root密码SSH登录与主机指纹已经真实核实；候选构建通过，本地隔离运行脚本及业务验证仍在准备，未跑项不计通过。远端旧派发保留历史，不再要求把Ops部署到该服务器。

## 来源与基线

- 主来源为[独立评审](../../../docs/reviews/pantheon-ops独立评审与外部报告核实-2026-11-20.md)，报告最初基线71c86cf、文末更新444ce8e。
- 配套来源为[真实环境验证清单](../../../docs/reviews/pantheon-ops真实环境验证清单-2026-11-20.md)，使用V1-V8全部八项；清单“6个问题”的描述与编号数量存在差异。
- 规划读取候选为 `a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc`，foundation为v0.18.9/6316f9b7189c86542424145fa49344ad737e952b。
- 结论优先级遵循文末逐条权限核对、基线与CI核对、正文。来源是静态审查，任务开工先给每条发现绑定当前候选事实。
- 当前旧十包已经归档为completed并保留原范围验收residual；承接依据为[全板处置](../../../.harness/evidence/2026-11-20-ops-board-complete/dispositions.json)。

## 范围与验收

本轮按报告采用多租户目标，覆盖模板Shell边界、凭据/任务/namespace授权、核心与扩展租户归属、部署恢复与远程取消、K8s配置/缓存/日志、服务实例闭环、启用扩展验收和交付包。

filedist和ArgoCD实施包按实际开放状态激活。T13模板体验增量在核心P1闭合后推进。资产关系与Pod→Node→VM、真实Chart执行、Nacos/RocketMQ、完整Agent协议登记独立后继；Agent无协议时禁用入口并拒绝执行。T01需冻结扩展开放状态与凭据资源边界，偏好选择记录为规划假设交维护者验收。

完成标准要求全部发现有当前判定、owner和证据/后继，核心和已启用扩展通过同候选真实验收，T17隔离交付操作成功，维护者功能/视觉验收留证。报告中P0为交付完整性，安全P1保留原语义。SUP-08按文末降为P2授权功能缺口，受限角色可用性为条件性P1。

## 子任务与依赖

| 包                                                                           | 目标                                  | 优先级 | 前置包                                | 激活条件         | 来源发现                                                     |
| ---------------------------------------------------------------------------- | ------------------------------------- | ------ | ------------------------------------- | ---------------- | ------------------------------------------------------------ |
| [T01](../2026-11-20-ops-review-baseline-and-decisions/HANDOFF.md)            | 当前基线核实与业务边界决策            | P1     | 无                                    | core             | ARC-001、NEW-02、NEW-03、SUP-08                              |
| [T02](../2026-11-20-ops-deploy-fixed-template-security/HANDOFF.md)           | 固定部署模板参数与卸载路径安全        | P1     | 无                                    | core             | SEC-001                                                      |
| [T03](../2026-11-20-ops-business-permissions-and-task-visibility/HANDOFF.md) | 业务权限绑定与部署任务数据可见性      | P1     | T01                                   | core             | SEC-002、NEW-01、SUP-08、路由与权限绑定逐条核对              |
| [T04](../2026-11-20-ops-core-business-tenant-isolation/HANDOFF.md)           | 核心业务资源租户归属与兼容迁移        | P1     | T01                                   | core             | ARC-001                                                      |
| [T05](../2026-11-20-ops-deploy-credential-ownership-and-ui/HANDOFF.md)       | 部署凭据资源授权与管理页面            | P1     | T03、T04                              | core             | SEC-004、DEP-003                                             |
| [T06](../2026-11-20-ops-k8s-resource-authorization-and-mutations/HANDOFF.md) | K8s 资源读写授权与变更准入            | P1     | T03、T04                              | core             | SEC-003、SEC-005、SEC-006、SEC-008、NEW-05、R-08、R-10、R-13 |
| [T07](../2026-11-20-ops-k8s-client-lifecycle-and-config-safety/HANDOFF.md)   | K8s 凭据导入、连接轮换与请求边界      | P1     | T01、T04                              | core             | R-03、R-04、R-06、R-07、R-14                                 |
| [T08](../2026-11-20-ops-k8s-log-websocket-lifecycle/HANDOFF.md)              | Pod 日志 WebSocket 来源与连接生命周期 | P1     | T06、T07                              | core             | SEC-007、R-05                                                |
| [T09](../2026-11-20-ops-deploy-recovery-and-remote-execution/HANDOFF.md)     | 部署任务恢复、协程与远程执行一致性    | P1     | T02、T03、T04                         | core             | R-01、R-02、R-11、R-12、DEP-004                              |
| [T10](../2026-11-20-ops-filedist-durable-lifecycle/HANDOFF.md)               | 文件分发队列、恢复与取消              | P1     | T03、T14                              | filedist-enabled | R-02、R-09                                                   |
| [T11](../2026-11-20-ops-service-deploy-business-closure/HANDOFF.md)          | 服务目录与实例部署关联闭环            | P0     | T03、T04、T05、T06、T09               | core             | DEP-001、DEP-002                                             |
| [T12](../2026-11-20-ops-argocd-ownership-and-webhook/HANDOFF.md)             | ArgoCD 归属、Webhook 与状态收敛       | P1     | T03、T14                              | argocd-enabled   | R-16                                                         |
| [T13](../2026-11-20-ops-deploy-template-and-task-experience/HANDOFF.md)      | 模板编辑与回滚追溯体验                | P2     | T05、T08、T09、T11                    | after-core-p1    | DEP-005、DEP-006、DEP-007、R-15、前端长耗时操作              |
| [T14](../2026-11-20-ops-extension-and-async-tenant-isolation/HANDOFF.md)     | 扩展业务与异步链路租户隔离            | P1     | T03、T04                              | core             | ARC-001                                                      |
| [T15](../2026-11-20-ops-review-docs-and-test-governance/HANDOFF.md)          | 能力文档、测试资产与治理对账          | P2     | T01、T03                              | core             | NEW-02、NEW-03、NEW-04、文档与版本、冗余与遗留代码           |
| [T16](../2026-11-20-ops-review-live-runtime-acceptance/HANDOFF.md)           | 同候选真实环境与受限角色验收          | P0     | 无；实际失败驱动对应整改包            | core             | V1、V2、V3、V4、V5、V6、V7、V8                               |
| [T17](../2026-11-20-ops-review-delivery-and-closeout/HANDOFF.md)             | 试点交付包、安装回滚与审查收口        | P0     | T16                                   | core             | 交付决策与后续工作                                           |
| [T18](../2026-11-25-ops-tenant-detail-route-binding/HANDOFF.md)              | Base 发版后恢复业务详情路由租户绑定   | P1     | Base 正式 release（#525）；T04 已收口 | core             | ARC-001、V2                                                  |
| [T19](../2026-11-25-ops-residual-live-acceptance/HANDOFF.md)                 | 真实环境残余验收                      | P0     | T05、T11、T12、T13、T14               | core             | V4、V7、V8、K8s shelf                                        |

依赖字段使用任务ID，表中的T编号用于阅读。T16先验当前候选，原整改前置包保留为followupRemediationTasks，实际失败后按局部整改推进；扩展条件包按实际开放范围验收。T17只能使用T16认证的候选。

## 执行顺序与文件交接

1. T01核实来源与范围，T02可直接开始复用模板校验的核实与反例。
2. T03授权契约、T04核心租户契约冻结后，再推进凭据、K8s及部署。T04先交付模型/索引/迁移，T03对同一部署文件的变更依序集成。
3. T05/T06/T07推进凭据、K8s资源授权与client配置，T08跟进日志生命周期，T09推进部署恢复。T14承接扩展租户与异步身份。
4. T10/T12按开放清单推进扩展，T11关闭服务实例用户流；T15核实文档与实际测试调用，T13按后续体验范围实施。
5. T16冻结候选并做完整真实环境验收，T17做隔离安装升级回滚与最终交付收口。

本计划不规定工具或模型。实际执行遇到共同文件必须由当前owner串行修改，按合同/候选哈希交接。部署的deploy_service.go/模型/handler由T03、T04、T05、T09、T11依序交接；K8s cluster文件由T04、T06、T07依序交接；业务路由/seed由T03统一契约后各owner实现；T04与T14分配不同迁移编号；frontend同页由后端契约完成后交接。共享文件的新owner保留他人改动。

## 发现到任务的完整映射

机器映射见[FINDINGS.json](./FINDINGS.json)。该文件包含37条有编号的SEC/DEP/ARC/NEW/R发现、SUP-08及补审主题、V1-V8和十包历史承接。

| 发现                   | 处理包        | 核实要求或更正                                                                    |
| ---------------------- | ------------- | --------------------------------------------------------------------------------- |
| SEC-001                | T02           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-002                | T03           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-003                | T06           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-004                | T05           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-005                | T06           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-006                | T06           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-007                | T08           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| SEC-008                | T06           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| DEP-001                | T11           | 已有#195推进，当前缺口与C01原范围验收需重新核实                                   |
| DEP-002                | T11           | 已有#195推进，当前缺口与C02原范围验收需重新核实                                   |
| DEP-003                | T05           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| DEP-004                | T09           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| DEP-005                | T13           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| DEP-006                | T13           | 外部结论未逐行复核，先复现再实现                                                  |
| DEP-007                | T13           | 部分复核，按当前API与页面重新核实                                                 |
| ARC-001                | T01、T04、T14 | 报告记录多租户目标已确认，本规划沿用该目标；业务表分类和覆盖范围由T01冻结         |
| NEW-01                 | T03           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| NEW-02                 | T01、T15      | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| NEW-03                 | T01、T15      | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| NEW-04                 | T15           | 按基线更新撤销核心业务smoke未接CI的旧结论；核实单测/旧e2e和安全反例的真实调用缺口 |
| NEW-05                 | T06           | 源码行为已读，授权影响仍需反例核实                                                |
| R-01                   | T09           | 代码路径静态核实，重启卡死时序为推断，需租约跨重启复现                            |
| R-02                   | T09、T10      | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| R-03                   | T07           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| R-04                   | T07           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| R-05                   | T08           | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| R-06                   | T07           | 无校验为静态结论，可利用性需无副作用负例证明                                      |
| R-07                   | T07           | 子审查，实施前核实全集群Pod读取和分页                                             |
| R-08                   | T06           | 校验标签已核实，零副本拒绝为静态推断                                              |
| R-09                   | T10           | 子审查，需当前队列/恢复/超时反例                                                  |
| R-10                   | T06           | 需对照Base SecureAction合同和实际危险操作，复用公共能力                           |
| R-11                   | T09           | 子审查，先核实当前stream/缓冲与错误i18n                                           |
| R-12                   | T09           | 子审查推断，真实SSH副作用与重试是验收重点                                         |
| R-13                   | T06           | 子审查，验证同名Deployment不会被DaemonSet请求修改                                 |
| R-14                   | T07、T05      | 子审查，确认敏感输入与截图脱敏                                                    |
| R-15                   | T13           | 请求拦截器补审已将缩容错误反馈降P3；保留业务范围静默失败的核实                    |
| R-16                   | T12           | 处理器静态存在、路由可达性待实测，#195后仍需确认                                  |
| SUP-08                 | T03           | 文末降为P2授权模型/功能缺口；细粒度角色可用性仍为条件性P1，不按已证实越权整改     |
| 路由与权限绑定逐条核对 | T03           | 以文末精确接口数为审查基线，保留详情/export有意缺口，当前路由再次核实             |
| 前端长耗时操作         | T07、T13、T16 | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |
| 文档与版本             | T15           | 只修当前有效版本语义与失效链接，英文补齐数量不直接形成项目                        |
| 冗余与遗留代码         | T15           | Evict应接入；生成迁移副本有意保留；deadcode有调用证据后才删                       |
| 交付决策与后续工作     | T16、T17      | 来源为静态审查，开工确认当前实现、回归与真实运行缺口                              |

## 旧十包的原验收承接

原manifest.acceptanceCriteria、原task packet、review、summary和historyCloseout.residuals共同构成输入。表中列出剩余检查重点，每包开工需逐条带入原标准并标记对应当前证据，保留原失败和未跑事实。

| 原包 | 已归档输入                                                                                                      | 后继包 | 当前核实重点                                                   |
| ---- | --------------------------------------------------------------------------------------------------------------- | ------ | -------------------------------------------------------------- |
| C01  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-service-deploy-integrity/manifest.json)     | T11    | 租户/引用/host策略/恢复及真实角色provider                      |
| C02  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-service-deploy-user-flow/manifest.json)     | T11    | 浏览器实例部署全过程/失败越权/终态日志审计                     |
| C03  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-service-catalog-usability/manifest.json)    | T11    | 目录生命周期/NamespaceBinding/environment/action/101+对象/角色 |
| C04  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-k8s-service-mutations/manifest.json)        | T06    | 最小角色/真实binding CRUD/kubectl冲突/全部UI状态               |
| C05  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-business-runtime-acceptance/manifest.json)  | T16    | 空库升级/故障回滚/角色provider/UI原完整矩阵                    |
| C06  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-pilot-release-closeout/manifest.json)       | T17    | 包/校验和/隔离安装升级失败恢复回滚/离线/终验                   |
| E01  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-filedist-product-closure/manifest.json)     | T10    | 真实SSH/异步job授权/重启取消/两域role/shutdown                 |
| E02  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-argocd-ownership-convergence/manifest.json) | T12    | ownership/legacy/UID/token/webhook/双写/时间戳/provider        |
| E03  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-helm-capability-truth/manifest.json)        | T15    | 登记状态/历史raw/UI菜单口径，实际开放操作按清单验收            |
| E04  | [原manifest](../../../.harness/archive/2026-11/tasks/2026-11-16-ops-extensions-live-acceptance/manifest.json)   | T16    | 按实际开放能力执行九行provider矩阵和清理/终验                  |

C01->C02、C01-C04->C05->C06与已开放扩展->E04->C06的原因果顺序由T11/T06->T16->T17和条件依赖保持。

## 真实验证的修复前后预期

V1-V8中的接口、字段、租户模式配置名和启动环境变量均需按当前路由/合同校准。V1/V3全部403、V4可见对方数据、V8旧client失效时间等是报告给出的修复前预测，修复后的预期由下表限定。

| 项  | 修复前核实                                                                            | 修复后验收                                                                                     | 包                 |
| --- | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ------------------ |
| V1  | 按报告推断查看角色的tasks/create/start/credentials均403；实际结果用于核实既有授权路径 | 查看角色可读授权任务；创建、启动和凭据操作按独立权限拒绝                                       | T03、T05、T16      |
| V2  | 列表200，export403，详情可能403，核实keyMatch2有意缺口                                | 列表/详情按读取合同可用，export需要独立权限；若共享匹配器仍阻碍安全详情授权，记录Base依赖和gap | T03、T16           |
| V3  | 按报告未绑定模块可能403，记录按钮显示和后端行为                                       | 已授予的查询/写动作成功，未授予动作拒绝，跨域/租户不可越界                                     | T03、T06、T16      |
| V4  | multi业务隔离缺口为待实测推断                                                         | 租户B看不到A资源，猜ID/修改/导出/回调无跨租户副作用；compat既有数据可用                        | T04、T14、T16      |
| V5  | 核实凭据脱敏、加密和归属面；以当前已注册路由为准                                      | 明文/密文不进入响应或日志；跨归属按ID使用拒绝，合法凭据执行成功                                | T05、T16           |
| V6  | 记录真实耗时、浏览器10秒超时与后端结果，源清单中的超时现象是待核实预测                | 受测长操作可取消/有界/有可见结果，重复提交无重复副作用                                         | T07、T09、T13、T16 |
| V7  | 隔离SSH验证模板注入、指纹、租约跨重启、远程取消                                       | 注入在执行前拒绝且无标记，指纹不符拒绝，重启终态可收敛，远程取消/重试副作用有证据              | T02、T09、T11、T16 |
| V8  | 源清单SUP-01对应本报告R-03，旧凭据缓存是静态预测                                      | 持续请求下轮换后使用新身份，旧client不能复活，撤销旧令牌后合法请求正常                         | T07、T16           |

清单中的远程IP作为原报告示例输入保留。真实SSH/provider目标使用执行时已授权的隔离环境；规划不会默认把该IP登记为授权目标。V7注入应在后端拒绝，隔离目标检查无副作用。不要将生产凭据、完整kubeconfig或密钥写入证据。

## 验证与证据

- 在 backend/ 执行 Go 命令；race 使用 D:\msys64\mingw64\bin，CGO_ENABLED=1 和隔离 MySQL DSN；最终提交门禁执行 go test -race ./...
- 在 frontend/ 执行 lint/type-check/build/test:smoke:*；业务 overlay/重建/治理命令在仓库根执行
- UI 前后使用 impeccable 与 Base DESIGN.md，1440px/390px、明暗主题和相关 loading/empty/error/forbidden/submitting/键盘状态必须有 rendered evidence 或显式 gap
- 服务端二次确认、路由匹配器、tenant公共扩展和 shutdown 公共接口若缺失，登记 Base 上游任务与 release/消费依赖，不在 Ops override
- 只在实际完成验收时关闭任务；缺 provider/凭据/维护者终验保持 gap，安全可逆准备可继续

每个执行包记录真实commands.json、summary.md和review.md，status从planned开始按实际结果更新。当前Review File为none，开工和真实review产生后同步manifest与task packet，evidence与review必须引用同一taskId。最终验收提供candidate.json或等价receipt绑定源码、构建、lock、migration、provider与角色。

本批任务正文遵循仓库canonical位置 `docs/harness/tasks/<task-id>.task.md`；可执行manifest、handoff、status、父计划与发现映射位于 `.harness/tasks/`。批量task packet检查使用显式文件列表，仓库默认检查也会发现这18个新task packet。

## 规划交付检查

```powershell
node scripts/harness/check-task-packet.mjs --root . docs/harness/tasks/2026-11-20-ops-review-baseline-and-decisions.task.md docs/harness/tasks/2026-11-20-ops-deploy-fixed-template-security.task.md docs/harness/tasks/2026-11-20-ops-business-permissions-and-task-visibility.task.md docs/harness/tasks/2026-11-20-ops-core-business-tenant-isolation.task.md docs/harness/tasks/2026-11-20-ops-deploy-credential-ownership-and-ui.task.md docs/harness/tasks/2026-11-20-ops-k8s-resource-authorization-and-mutations.task.md docs/harness/tasks/2026-11-20-ops-k8s-client-lifecycle-and-config-safety.task.md docs/harness/tasks/2026-11-20-ops-k8s-log-websocket-lifecycle.task.md docs/harness/tasks/2026-11-20-ops-deploy-recovery-and-remote-execution.task.md docs/harness/tasks/2026-11-20-ops-filedist-durable-lifecycle.task.md docs/harness/tasks/2026-11-20-ops-service-deploy-business-closure.task.md docs/harness/tasks/2026-11-20-ops-argocd-ownership-and-webhook.task.md docs/harness/tasks/2026-11-20-ops-deploy-template-and-task-experience.task.md docs/harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation.task.md docs/harness/tasks/2026-11-20-ops-review-docs-and-test-governance.task.md docs/harness/tasks/2026-11-20-ops-review-live-runtime-acceptance.task.md docs/harness/tasks/2026-11-20-ops-review-delivery-and-closeout.task.md docs/harness/tasks/2026-11-20-ops-review-remediation.task.md
```

检查本批18个manifest的schema、父子对称、依赖存在及无环、37条编号发现与V1-V8覆盖、启用条件、旧十包链接和taskDoc/linkage一致性，结果写入 `.harness/evidence/2026-11-20-ops-review-remediation/planning-validation.json`。这些检查用于确认任务可执行，实施验收等待各包真实执行。

本次自检采用来源与格式对照。独立规划review尚未执行，源码漏洞复现、真实provider与视觉证据尚未执行，全部在对应任务的Known Gaps记录。
