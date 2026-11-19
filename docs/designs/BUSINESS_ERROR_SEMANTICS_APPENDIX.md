---
title: 运维域业务错误语义附录
doc_type: Design
layer: business/*
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-18
---

# 运维域业务错误语义附录

English version: [BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md](./BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md)

本文是 `pantheon-ops` 业务模块在真实实现阶段的**错误 key canonical 附录**。

目标只有一个：

- 让后端返回的业务错误 key 在编码前就收口
- 让前端 i18n、接口联调、验收、审计详情查看同一套语义
- 避免每个 handler / service 在实现时各自发明一套错误命名

适用范围：

- `business/cmdb`
- `business/deploy`
- 后续新增的 `business/*` 模块

---

## 1. 规则

### 1.1 canonical 命名

`pantheon-ops` 业务错误 key 统一使用：

```text
business.<module>.<resource>.<reason>
```

示例：

- `business.cmdb.host.notFound`
- `business.cmdb.collect.sshAuthFailed`
- `business.deploy.task.invalidStartState`
- `business.deploy.taskHost.markFailed.reasonRequired`

约束：

- `module` 取业务模块名，如 `cmdb`、`deploy`
- `resource` 取资源名或明确交互域，如 `host`、`group`、`label`、`collect`、`package`、`task`、`taskHost`
- `reason` 使用 lowerCamelCase
- 禁止继续新增历史短 key，如 `cmdbhost.*`、`cmdbgroup.*`、`cmdblabel.*`

### 1.2 返回责任

- 后端负责返回**稳定错误 key**
- 前端负责把错误 key 翻译为本地化文案
- 审计详情、导入导出结果、页面内 banner / toast 统一优先按错误 key 翻译，而不是硬编码自然语言

### 1.3 HTTP 语义建议

| 场景                    | 建议 HTTP      | 错误 key 语义                                                |
| :---------------------- | :------------- | :----------------------------------------------------------- |
| 参数缺失 / 格式错误     | `400`          | `*.nameRequired`、`*.invalidLabel`                           |
| 权限不足                | `403`          | `permission.denied` 或业务明确 `*.forbidden`                 |
| 资源不存在              | `404`          | `*.notFound`                                                 |
| 唯一键冲突 / 状态冲突   | `409`          | `*.ipExists`、`*.nameVersionConflict`、`*.invalidStartState` |
| 外部依赖失败 / 执行失败 | `422` 或 `500` | `*.sshAuthFailed`、`*.executionFailed`                       |

说明：

- 平台通用错误继续复用 base 的 `param.invalid`、`permission.denied`、`request.failed` 等
- 只有进入明确业务语义时，才返回 `business.*`

---

## 2. CMDB canonical 清单

来源设计文档：

- `BUSINESS_CMDB_MODULE_DESIGN.md`

### 2.1 Host

| Key                                | 说明             |
| :--------------------------------- | :--------------- |
| `business.cmdb.host.notFound`      | 主机不存在       |
| `business.cmdb.host.ipExists`      | IP 已存在        |
| `business.cmdb.host.invalidLabel`  | 标签格式无效     |
| `business.cmdb.host.unsupportedOs` | 不支持的操作系统 |

### 2.2 Collect

| Key                                      | 说明             |
| :--------------------------------------- | :--------------- |
| `business.cmdb.collect.sshConnectFailed` | SSH 连接失败     |
| `business.cmdb.collect.sshAuthFailed`    | SSH 认证失败     |
| `business.cmdb.collect.executionFailed`  | 配置采集执行失败 |

### 2.3 Group

| Key                                     | 说明               |
| :-------------------------------------- | :----------------- |
| `business.cmdb.group.notFound`          | 分组不存在         |
| `business.cmdb.group.invalidConditions` | 过滤条件表达式无效 |

### 2.4 Label

| Key                             | 说明                   |
| :------------------------------ | :--------------------- |
| `business.cmdb.label.keyExists` | 标签键已存在           |
| `business.cmdb.label.invalid`   | 标签规范配置无效       |
| `business.cmdb.label.inUse`     | 标签已被主机或分组引用 |
| `business.cmdb.label.notFound`  | 标签规范不存在         |

---

## 3. Deploy canonical 清单

来源设计文档：

- `BUSINESS_DEPLOY_MODULE_DESIGN.md`

### 3.1 Package

| Key                                           | 说明                 |
| :-------------------------------------------- | :------------------- |
| `business.deploy.package.nameRequired`        | 组件名称必填         |
| `business.deploy.package.versionRequired`     | 组件版本必填         |
| `business.deploy.package.nameVersionConflict` | 组件名与版本组合冲突 |
| `business.deploy.package.commandTooLong`      | 命令模板超长         |

### 3.2 Task

| Key                                             | 说明                                                                                                               |
| :---------------------------------------------- | :----------------------------------------------------------------------------------------------------------------- |
| `business.deploy.task.notFound`                 | 部署任务不存在                                                                                                     |
| `business.deploy.task.forbidden`                | 当前用户无权查看或操作该任务                                                                                       |
| `business.deploy.task.nameRequired`             | 任务名称必填                                                                                                       |
| `business.deploy.task.packageRequired`          | 组件必选                                                                                                           |
| `business.deploy.task.packageDisabled`          | 组件已禁用，不能用于新任务                                                                                         |
| `business.deploy.task.scopeRequired`            | 主机部署必须选择业务域                                                                                             |
| `business.deploy.task.scopeInvalid`             | 所选业务域不存在或不可用                                                                                           |
| `business.deploy.task.targetRequired`           | 目标主机或分组必填                                                                                                 |
| `business.deploy.task.invalidTargetType`        | 目标类型无效                                                                                                       |
| `business.deploy.task.targetOutOfScope`         | 目标不在当前数据范围内                                                                                             |
| `business.deploy.task.serviceLinkInvalid`       | 关联部署必须指定同一业务域中有效的服务与实例，且与唯一实际主机匹配；缺失 owner、单字段关联或 Kubernetes 错配均拒绝 |
| `business.deploy.task.invalidExecutorType`      | 执行方式无效                                                                                                       |
| `business.deploy.task.invalidAction`            | 任务动作无效                                                                                                       |
| `business.deploy.task.targetStatusMismatch`     | 目标当前状态不支持所选动作                                                                                         |
| `business.deploy.task.templateNotFound`         | 任务引用的模板不存在                                                                                               |
| `business.deploy.task.templateDisabled`         | 任务引用的模板已停用                                                                                               |
| `business.deploy.task.packageNotFound`          | 任务引用的组件不存在                                                                                               |
| `business.deploy.task.invalidStartState`        | 当前状态不允许启动                                                                                                 |
| `business.deploy.task.invalidUpdateState`       | 当前状态不允许编辑                                                                                                 |
| `business.deploy.task.invalidDeleteState`       | 当前状态不允许删除                                                                                                 |
| `business.deploy.task.emptyResolvedTargets`     | 解析后无可执行目标                                                                                                 |
| `business.deploy.task.invalidCancelState`       | 当前状态不允许取消                                                                                                 |
| `business.deploy.task.templateParamsInvalid`    | 启动或保存时模板参数不完整或模板变量无法解析                                                                       |
| `business.deploy.task.templateInvalid`          | 启动时固定模板定义无效                                                                                             |
| `business.deploy.task.installCommandRequired`   | 启动安装类任务时组件缺少安装命令                                                                                   |
| `business.deploy.task.uninstallCommandRequired` | 启动卸载类任务时组件缺少卸载命令                                                                                   |
| `business.deploy.task.packageSourceMissing`     | 启动固定模板任务时缺少可下载的软件包来源                                                                           |
| `business.deploy.task.sshHostKeyRequired`       | SSH 启动缺少主机指纹                                                                                               |
| `business.deploy.task.sshHostKeyMismatch`       | SSH 主机指纹校验失败                                                                                               |
| `business.deploy.task.sshUserRequired`          | SSH 启动缺少用户名                                                                                                 |
| `business.deploy.task.sshPasswordRequired`      | SSH 密码认证模式缺少密码                                                                                           |
| `business.deploy.task.sshPrivateKeyRequired`    | SSH 私钥认证模式缺少私钥                                                                                           |
| `business.deploy.task.sshAuthFailed`            | SSH 认证失败                                                                                                       |
| `business.deploy.task.sshConnectFailed`         | SSH 连接失败                                                                                                       |

### 3.3 TaskHost

| Key                                                  | 说明                       |
| :--------------------------------------------------- | :------------------------- |
| `business.deploy.taskHost.notFound`                  | 主机执行明细不存在         |
| `business.deploy.taskHost.invalidResultState`        | 当前状态不允许标记结果     |
| `business.deploy.taskHost.markFailed.reasonRequired` | 标记失败时必须填写错误原因 |

### 3.4 Template

Deploy Template CRUD 已上线，但当前尚未定义 Template 资源的 canonical 错误 key 清单。本段仅记录待补事实，本任务不发明新 key；后续应在新增或收口模板业务错误时先补本文，再同步模块设计与 i18n。

---

## 4. BizScope 已迁移状态

来源设计文档：

- `BUSINESS_BIZSCOPE_MODULE_DESIGN.md`

### 4.1 canonical 清单

| Key                            | 说明                                 | 来源                                             |
| :----------------------------- | :----------------------------------- | :----------------------------------------------- |
| `business.bizscope.codeExists` | 业务域编码重复                       | Create / Update 业务唯一性校验                   |
| `business.bizscope.inUse`      | 业务域仍绑定主机，不能删除           | Delete 引用保护                                  |
| `business.bizscope.notFound`   | 业务域、绑定目标或作用域内主机不存在 | Detail / Hosts / Bind / Unbind / Update / Delete |
| `param.invalid`                | 请求参数无效                         | base 通用参数错误 key                            |
| `request.failed`               | 非业务语义的查询、数据库或框架失败   | base 通用请求失败 key                            |

`backend/modules/business/bizscope` 已不再返回 `bizscope.code_exists`、`bizscope.in_use`、`bizscope.not_found`。

### 4.2 handler 流水 key 迁移结果

| 旧 handler Key                  | 已迁移处理                                                                                                                   |
| :------------------------------ | :--------------------------------------------------------------------------------------------------------------------------- |
| `bizscope.list.error`           | 无独立业务语义；回落 `request.failed`                                                                                        |
| `bizscope.options.error`        | 无独立业务语义；回落 `request.failed`                                                                                        |
| `bizscope.detail.error`         | 资源不存在时返回 `business.bizscope.notFound`；其他失败回落 `request.failed`                                                 |
| `bizscope.hosts.error`          | 业务域不存在时返回 `business.bizscope.notFound`；其他失败回落 `request.failed`                                               |
| `bizscope.availableHosts.error` | 业务域不存在时返回 `business.bizscope.notFound`；其他失败回落 `request.failed`                                               |
| `bizscope.bindHosts.error`      | 业务域/作用域内主机不存在时返回 `business.bizscope.notFound`，空 hostIds 返回 `param.invalid`；其他失败回落 `request.failed` |
| `bizscope.unbindHost.error`     | 业务域、主机或绑定关系不存在时返回 `business.bizscope.notFound`；其他失败回落 `request.failed`                               |
| `bizscope.create.error`         | 编码冲突返回 `business.bizscope.codeExists`；其他失败回落 `request.failed`                                                   |
| `bizscope.update.error`         | 资源不存在返回 `business.bizscope.notFound`，编码冲突返回 `business.bizscope.codeExists`；其他失败回落 `request.failed`      |
| `bizscope.delete.error`         | 仍被主机引用返回 `business.bizscope.inUse`，资源不存在返回 `business.bizscope.notFound`；其他失败回落 `request.failed`       |

### 4.3 旧 key fallback 决策

决策：**直接移除，不保留一个版本的旧 key fallback 映射**。

依据：前端请求层只对后端 `message` 做 i18n key 形态识别和精确翻译，不存在对旧 BizScope 短 key 的业务分支；翻译未命中时开发环境显示 `request.failed` 加原 key，生产环境只显示 `request.failed`。因此保留旧映射只会延长双 key 生命周期，不能提供必要兼容。zh-CN / en-US 模块 locale 和后端 seed 已改为 canonical key，seed 同时删除 `business.bizscope` 下三个 legacy 运行时记录。

---

## 5. 新模块接入要求

C04 Kubernetes Service 结构化修改与命名空间准入的 canonical 错误：

| Key                                     | 说明                                                                                                    |
| :-------------------------------------- | :------------------------------------------------------------------------------------------------------ |
| `k8s.service.resource_version_conflict` | 读取后的 Service 已改变或同名对象被重建；修改/删除返回实际 HTTP 409，保留对象和用户输入，刷新后重新核对 |
| `k8s.service.resource_version_required` | 修改缺少资源版本，或删除缺少 UID/资源版本；不得通过获取最新标识替代调用方预期版本                       |
| `k8s.service.update_failed`             | 无更具体业务 key 的 Kubernetes Service 更新失败                                                         |
| `k8s.namespace.binding_invalid`         | 命名空间绑定与集群业务域或环境不匹配；拒绝调用 provider                                                 |

`k8s.service.audit.update` 是更新操作的审计描述 key。前端完整提交类型、选择器和端口（保留数字/命名目标端口），沿用读取到的 UID/资源版本。合法无选择器 Service 使用 `{}`；格式错误或重复选择器键不能静默丢弃。以上 key 同步五语言业务 locale 和中文/英文运行态 seed；生成 fallback 由正式 overlay 重建刷新。

C01 Service 引用保护的 canonical 错误：

| Key                                                 | 说明                                                                                                          |
| :-------------------------------------------------- | :------------------------------------------------------------------------------------------------------------ |
| `business.service.instance_in_use`                  | 实例仍被执行中、暂停中或取消中的部署引用，不能删除或退役；删除和实际 Retire 状态变更均在同一 owner 事务中检查 |
| `business.service.deployment_reader_not_configured` | 部署引用校验未配置或无法绑定到 owner 事务；拒绝删除或退役                                                     |

Service 删除仍保留 `business.service.service_in_use` 的实例引用保护及执行引用保护；终态部署保留关联 ID 和目标快照，不静默级联。

后续新增 `business/*` 模块时，设计文档至少要补：

1. 模块自己的错误 key 前缀
2. 资源级 canonical 清单
3. 参数校验类错误
4. 状态流转类错误
5. 权限 / 数据范围类错误
6. 外部依赖失败类错误

若新增模块没有补这份清单，不应进入实现阶段。

---

C03 实例 owner 编辑及状态冲突的既有 canonical 错误（补齐清单与五语言资源）：

| Key                                 | 说明                                               |
| :---------------------------------- | :------------------------------------------------- |
| `business.service.state_stale`      | 期望生命周期版本已过期；刷新实例后再编辑，返回冲突 |
| `business.service.state_invalid`    | 当前状态或请求不允许该生命周期变更                 |
| `business.service.target_invalid`   | 目标字段、类型或定位信息无效；拒绝部分变更         |
| `business.service.target_not_found` | 目标不存在、已失效或在当前范围不可见               |
| `business.service.target_conflict`  | 目标绑定冲突或当前实例/执行引用不允许安全修改      |

HTTP 实例编辑不得写入运行态 currentVersion；观测版本由 owner 状态/运行命令发布。

### 5.1 E01/E02 生命周期与同步观察 canonical 错误

| Key                                                | 说明                                               |
| :------------------------------------------------- | :------------------------------------------------- |
| `business.filedist.task.executionInterrupted`      | 执行已中断；任务进入失败状态，请核对结果后重试     |
| `business.filedist.task.transferFailed`            | 传输失败；请核对目标结果后重试                     |
| `business.filedist.task.executionBusy`             | 另一执行进程正在管理文件分发，请稍后重试           |
| `business.filedist.task.notDue`                    | 任务尚未到计划执行时间                             |
| `business.filedist.task.notCancellable`            | 当前任务状态不允许取消                             |
| `business.filedist.task.invalidContext`            | 执行上下文无效，任务未开始                         |
| `business.cicd.application.syncInProgress`         | 应用正在同步，请等待当前操作完成                   |
| `business.cicd.application.syncObservationTimeout` | 同步结果观察超时；结果未知，请核对 ArgoCD 实际状态 |
| `business.cicd.application.sourceChanged`          | 应用来源已改变，请刷新后重新提交同步               |

六个临时 FileDist 短 key 不保留 alias。锁定底座的通用解析器只接受小写 key 段，业务 handler 对这些已声明 key 做显式映射。取消不回滚已写入远端的字节；观察超时表示 Unknown，不能显示同步成功。生产任务/源文件授权 provider 及进程级停机 join 仍是独立验收缺口。

## 6. 变更规则

- 新增、重命名、废弃业务错误 key 时，优先修改本文，再同步回业务模块设计文档
- 业务模块设计文档中的错误章节是模块局部说明，**本文是 ops 仓库级 canonical 附录**
- 若与 `pantheon-base/docs/designs/ERROR_CODE_AND_I18N.md` 冲突，以 base 的通用规则为上位约束，以本文的业务清单为 ops 层具体落点
