---
title: Operational Workbench Components Design
doc_type: Design
layer: platform
status: Draft
updated_at: 2026-08-31
linked_contracts:
  - DESIGN.md
  - docs/designs/FRONTEND_UI_SPEC.md
  - docs/designs/FRONTEND_PAGE_TEMPLATES.md
  - docs/designs/ACCESSIBILITY.md
  - docs/designs/THEME_TOKENS_REFERENCE.md
---

# 运维工作台共享组件设计

## 1. 目的与结论

Pantheon Base 已经具备克制、统一的后台视觉基线，但共享能力仍偏向 CRUD 页面骨架。本文补齐面向运维场景的交互合同：对象上下文、执行步骤、风险预览、长日志、复杂筛选、表格视图与恢复动作。

本设计借鉴 BK Design 的成熟交互模型，但不引入 BK Design 依赖、不复制其视觉皮肤，也不在 Base 中实现 CMDB、发布、Kubernetes 或可观测性业务。所有实现继续使用 Arco Design、Pantheon token、现有 i18n 和权限合同。

## 2. 证据与外部参考

### 2.1 当前仓库事实

- `AppTable` 已处理响应式、横向滚动和分页，但没有列显隐、顺序、宽度、密度或保存视图合同。
- `SearchToolbar` 已处理关键词、行内/高级筛选和移动端弹层，可作为复杂查询入口的基础。
- `SubmitBar` 目前是普通右对齐按钮行，长表单滚动后无法持续暴露提交状态和主操作。
- Dashboard widget registry 目前只有 `quick-action` 与 `domain-overview` 两类语义槽位。
- Playwright visual baseline 只覆盖 `1440x900` 下的登录、Dashboard 和用户列表，缺少移动端与关键状态。

### 2.2 BK Design 参考

- 设计价值观：<https://bkdesign.bk.tencent.com/design/32>
- 兼容性：<https://bkdesign.bk.tencent.com/design/4>
- 文案：<https://bkdesign.bk.tencent.com/design/23>
- 按钮：<https://bkdesign.bk.tencent.com/design/9>
- 表单：<https://bkdesign.bk.tencent.com/design/33>
- 加载：<https://bkdesign.bk.tencent.com/design/138>
- 控件布局：<https://bkdesign.bk.tencent.com/design/17>
- 表格：<https://bkdesign.bk.tencent.com/design/35>
- 条件筛选：<https://bkdesign.bk.tencent.com/design/177>
- 业务选择器：<https://bkdesign.bk.tencent.com/design/149>
- IP 选择器：<https://bkdesign.bk.tencent.com/design/154>
- 任务日志：<https://bkdesign.bk.tencent.com/design/164>
- 日志检索：<https://bkdesign.bk.tencent.com/design/169>
- 测试步骤：<https://bkdesign.bk.tencent.com/design/158>
- 时间筛选：<https://bkdesign.bk.tencent.com/design/173>
- 版本树与 Diff：<https://bkdesign.bk.tencent.com/design/159>、<https://bkdesign.bk.tencent.com/design/180>

外部参考只证明交互模式有成熟先例，不替代 Pantheon 的权限、审计、i18n、可访问性和 Base-first 边界。

### 2.3 吸收与拒绝决策

| BK 原则 | Pantheon 采纳方式 | 门禁信号 |
| --- | --- | --- |
| 高效 | 优先默认值、选择和短流程，减少重复输入 | 重复操作有默认/复用路径；主流程不被低频操作阻塞 |
| 清晰 | 精确文案、状态可见、反馈明确、区域内单一主任务 | loading/error/forbidden 不混同；每个操作区只有一个主操作 |
| 一致 | 继续复用 Arco、Pantheon token、共享组件和统一术语 | shell/UI/i18n checker 与共享组件使用检查通过 |
| 可恢复 | 错误、stale、partial failure 保留有效上下文并给出恢复动作 | 状态矩阵、交互断言和恢复路径证据齐全 |
| 表格可扫读 | 默认约 `7 ± 2` 个可见列，低频列可配置，宽度与横向滚动有意设计 | 长中英文、移动端、固定列和横向滚动无覆盖 |
| 表单可控 | 复杂流程可拆分，即时校验，失败不丢有效输入 | error/submitting/conflict 与恢复断言 |
| 运行态可定位 | 测试步骤、日志、筛选、时区和 Diff 都保留上下文 | stale 失效、步骤跳转、日志定位和差异导航断言 |

不采纳过时或与本项目边界冲突的条目：IE9 兼容、固定 `460px` 弹窗尺寸、BK Design 运行时依赖和视觉皮肤复制。现代浏览器兼容意图保留为 Chrome/Firefox/Safari 与响应式验证要求。

## 3. 产品体验目标

界面属于高频运维后台，应呈现紧凑、安静、精确、风险可见的工具感：

1. 用户始终知道当前操作对象、作用范围、环境和权限边界。
2. 变更执行前能看到差异、影响对象、风险与不可逆项。
3. 长任务中能定位到目标、步骤、尝试和错误，并直接进入合法恢复动作。
4. 高频列表允许保存个人工作视图，但默认界面仍保持简单。
5. 移动端优先保障查看状态、定位失败与执行紧急动作，不强行复刻桌面密度。

## 4. 所有权边界

| 能力 | Base 所有权 | Ops 所有权 |
| --- | --- | --- |
| 表格列偏好、密度、视图存储合同 | 共享组件与偏好 key 规范 | 业务列、业务筛选和默认视图 |
| sticky 表单动作 | 布局、状态和可访问性 | 业务按钮、权限和提交逻辑 |
| 日志查看器 | 流式展示、搜索、暂停、下载适配口 | 日志源、脱敏、DataScope、重试语义 |
| Diff | 通用行级/结构化渲染与折叠 | 业务对象比较与风险解释 |
| 条件构建器 | AST、校验、键盘交互 | 字段目录、操作符白名单和查询映射 |
| 上下文选择器 | 插槽、选择摘要、排除集合合同 | BizScope、拓扑、主机、集群数据源 |
| 执行步骤轨 | 步骤状态、跳转、错误锚点 | 发布/任务状态机与动作权限 |
| Dashboard | widget 注册语义与资源预算 | 业务 widget 内容、权限和清理策略 |

共享合同必须先在 Base 实现并通过 foundation release 交付；Ops 不得复制 Base 源文件或维护第二套共享规范。

## 5. 共享组件合同

### 5.1 `SubmitBar` sticky 模式

- 新增显式 `sticky` 能力，默认保持当前非 sticky 行为，避免历史页面整体漂移。
- sticky 区显示主操作、次操作、提交中状态、未保存提示和表单级错误摘要。
- sticky 区不得遮挡最后一个字段；布局容器必须提供对应底部安全空间。
- 窄屏时主操作保持可见，低优先级动作进入菜单；不得因按钮文案长度造成横向溢出。
- 键盘焦点顺序与视觉顺序一致；提交中禁止重复提交但保留可感知状态。

### 5.2 `AppTable` 工作视图

- 列定义增加稳定 `columnKey`、默认可见性、可隐藏性、最小/最大宽度和移动端优先级。
- 支持列显隐、拖动排序、宽度调整、紧凑/标准密度和恢复默认。
- 支持页面声明 `viewKey` 后保存个人视图；没有稳定 key 时不得静默持久化。
- 保存内容只包含展示偏好与可序列化查询状态，不包含敏感数据、权限结果或整行数据。
- 服务端数据变化、列被权限移除或 schema 升级时，未知配置安全忽略并保留可恢复默认入口。

### 5.3 `TaskLogViewer`

- 支持结构化 chunk：`sequence`、`timestamp`、`source`、`stream`、`level`、`content`、`attempt`、`step`。
- 提供追尾/暂停、关键词搜索、级别和来源筛选、复制、下载、换行、全屏、重连与 stale 状态。
- 搜索和筛选不改变原始日志顺序；下载由调用方提供已鉴权、已脱敏的数据适配器。
- 断线后保留已加载内容并显式显示最后序号；不得用空白界面伪装连接正常。
- 大数据量采用窗口化或有界缓存，组件不得要求一次加载完整日志。

### 5.4 `ChangeDiff`

- 支持文本、键值、JSON/YAML 等结构化差异的统一外壳。
- 默认折叠未变化内容，突出新增、删除、修改和冲突；颜色之外必须有符号或文本语义。
- 暴露风险摘要插槽，业务层负责解释影响对象、回滚条件和不可逆项。
- 机密字段必须由数据提供方在进入组件前完成遮蔽，组件同时提供敏感 key 防误显检查。

### 5.5 `ConditionBuilder`

- 使用可序列化 AST，而不是拼接查询字符串；节点包含字段、操作符、值和 AND/OR 组。
- 字段与操作符由调用方白名单提供；不允许客户端 AST 直接变成 SQL、PromQL 或 LogQL。
- 支持键盘新增、删除、排序和组切换，错误就地关联到具体节点。
- 提供清晰的空条件、无匹配、无字段权限和非法历史条件恢复状态。

### 5.6 `ContextSelector`

- 统一外壳由来源导航、候选区、已选区、排除区和选择摘要组成。
- 数据源以 adapter 注册，Base 不认识 BizScope、拓扑、主机或集群业务实体。
- 选择值必须包含稳定 ID、展示快照、来源和可选范围版本；提交前由业务层重新授权与解析。
- 支持大量候选项的远程查询、分页/虚拟化和部分失效提示。

### 5.7 `ExecutionStepRail`

- 统一状态：`pending`、`running`、`success`、`warning`、`failed`、`skipped`、`canceled`。
- 展示步骤序号、名称、耗时、尝试次数和错误锚点，并允许跳转到对应日志或详情。
- 组件只渲染状态，不推导业务状态机，也不决定重试、跳过或回滚是否合法。

## 6. Dashboard 语义扩展

新增受控语义槽位，而不是允许任意业务组件占据首页：

- `status-summary`：有限数量的关键状态和异常计数。
- `attention-queue`：需要当前用户处理的失败、审批或过期事项。
- `trend-snapshot`：带明确时间范围和单位的轻量趋势。
- `recent-activity`：最近任务、变更或访问对象。

每个 widget 继续要求 owner、permission、cleanup policy，并新增数据新鲜度、最大查询预算、空态和错误隔离声明。首页不承载完整日志、复杂表格或高频轮询。

## 7. 文案合同

所有高风险交互文案按“对象 + 动作 + 结果/风险 + 恢复方式”组织：

- 按钮使用明确动作，如“重试失败主机”，避免只写“重试”。
- 确认框指出对象数量、作用范围和不可逆影响。
- 错误信息区分失败原因、当前状态和下一步合法动作。
- 空态区分首次使用、筛选无结果、权限不足、数据源断开和数据仍在加载。
- 技术细节放在可展开区域，不用堆栈或原始错误替代用户可执行说明。

## 8. 状态、响应式与可访问性矩阵

所有共享组件至少覆盖：

| 维度 | 必须验证 |
| --- | --- |
| 数据状态 | loading、empty-initial、empty-filtered、error、forbidden、partial、stale |
| 操作状态 | idle、submitting、success、conflict、retryable、non-retryable |
| 视口 | 1440x900、390x844；固定工具条和长文案不得遮挡内容 |
| 输入 | 键盘全流程、可见焦点、读屏名称、合理 tab 顺序 |
| 视觉 | 明/暗主题、200% 缩放、长中英文、非颜色状态提示 |
| 动效 | 遵循 reduced motion；实时更新不得造成布局跳动 |

## 9. 视觉回归基线

Base visual suite 扩展为“页面 x 视口 x 状态”矩阵：

- 页面：登录、Dashboard、标准列表、长表单、详情/执行轨、日志/Diff 样例页。
- 视口：至少桌面 `1440x900` 与移动 `390x844`。
- 状态：默认、loading、empty、error、forbidden；运行态组件追加 stale/partial failure。
- 主题：核心页面至少明暗主题各一组。

截图只能证明渲染结果；键盘、焦点、滚动、日志追尾和恢复动作还需要 Playwright 交互断言。

### 9.1 三层 UI 门禁

1. **静态合同**：`config/ui-quality-gate.json` 是机器可读真相源；`npm run check:ui-quality-gate` 防止原则、矩阵、来源、检查入口和维护者 gate 被静默削弱。现有 shell、UI、contrast、important-budget checker 继续负责具体代码反模式，不在新脚本重复扫描 CSS。
2. **渲染与交互证据**：自 `2026-08-31` 起，`ui-runtime` 任务必须在 manifest 声明桌面/移动、明/暗主题、基线状态、可访问性和路由/fixture。`check-visual-evidence` 继续核对实际证据覆盖；截图不能替代权限、功能和运行态测试。
3. **人工验收**：大面积或有意基线变更、最终视觉接受只能由维护者基于 before/after、交互结果和例外说明完成。Agent 不得用更新快照代替审查。

纯治理改动只有在 manifest 明确声明 `governance-only`、确认未改渲染表面并保留人工审批时，才能记录视觉证据豁免。生产组件、样式、文案、布局或交互一旦变化，不能使用该豁免。

## 10. 非目标

- 不替换 Arco Design，不引入 BK Design 包。
- 不在 Base 定义业务查询语言、部署状态机、CMDB 拓扑或 Kubernetes 对象。
- 不把所有列表升级成复杂工作台；只有高频、多列、重复配置的页面启用保存视图。
- 不在本轮文档任务中修改生产代码或声称新组件已经渲染完成。

## 11. 分阶段交付与验收

具体拆包见 `.harness/tasks/2026-08-31-base-operational-workbench-design/EXECUTION_QUEUE.md`。

Base 实现完成的必要条件：

1. 共享合同、Story/fixture 或示例页和类型测试同步交付。
2. 明暗主题、桌面/移动、长文案和关键状态有渲染证据。
3. 键盘/焦点、sticky 遮挡、日志追尾和表格偏好恢复有交互断言。
4. 不破坏历史默认行为；新增持久化有版本和恢复策略。
5. foundation release 发布后，Ops 才开始消费对应共享能力。

## 12. 本设计的证据边界

本轮只产出设计与执行包。现有截图证明当前 Base 视觉基线可用，但没有渲染未来组件，因此不得把本文状态表述为“UI 已完成”或“视觉已验收”。最终视觉验收属于各实现包的人类 gate。
