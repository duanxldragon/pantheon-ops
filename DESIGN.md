# Pantheon Ops - 运维业务平台总体设计

English version: [DESIGN.en.md](./DESIGN.en.md)

## Source of truth

- **状态**：Active；V1 发布状态仍为 `BLOCKED - NOT READY TO FREEZE`
- **最后刷新**：2026-08-24
- **主要产品表面**：业务域、服务目录、CMDB、安装部署、Kubernetes
- **功能总览**：[全局功能设计](./docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md)
- **冻结判定**：[V1 Release Freeze Checklist](./docs/V1_RELEASE_CHECKLIST.md)
- **继承基线**：[Pantheon Base 总体设计](../pantheon-base/DESIGN.md)
- **证据来源**：业务模块注册表、前后端路由与页面、现有 `BUSINESS_*` 设计、V1 冻结任务包与证据

本文是 `pantheon-ops` 仓库级产品与体验设计契约。它只描述 Ops 自有的 `business/*` 能力；平台壳层、认证、IAM、组织、配置、审计、i18n、上传和通用 UI 继续以 `pantheon-base` 为唯一事实源。

文档中的“设计目标”不等于“已经实现”。当前能力、V1 冻结目标和发布阻断必须按 [全局功能设计](./docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md) 的状态标签及 [V1 冻结检查表](./docs/V1_RELEASE_CHECKLIST.md) 判断。

## Brand

- **个性**：冷静、可信、工具化，面向需要重复操作和快速判断的运维团队。
- **信任信号**：真实状态、明确责任边界、可追溯操作、可恢复任务、脱敏输出和可复现证据。
- **避免**：营销式大标题、装饰性渐变、悬浮卡片堆叠、虚假实时状态、无后端行为的可点击控件，以及把高风险动作包装成普通 CRUD。

## Product goals

### Goals

1. 建立从业务责任到基础设施目标的统一运维上下文：`BizScope -> Application -> Service -> ServiceInstance -> Resource`。
2. 让 VM 与 Kubernetes 变更具备权限、数据范围、不可变目标快照、幂等、审计、恢复和外部运行证据。
3. 在保持模块化单体的前提下，通过 typed capability 隔离业务模块，避免跨模块直读表和实现耦合。
4. 让发布状态只由可复现门禁和证据决定，不被历史完成报告或页面展示覆盖。

### Non-goals

- V1 不建设完整 Metrics、Logs、Tracing、告警、事件和 On-call 平台。
- V1 不拆分微服务，不引入 Kafka、通用工作流引擎或 GitOps 控制面。
- Ops 不复制或覆盖 Base 的平台层、系统域和通用 UI 规范。
- CMDB 不复制 Kubernetes 运行态，业务主库不保存高基数日志和时序数据。

### Success signals

- 操作者能从业务域追踪到服务实例、VM/Kubernetes 目标、变更任务和审计证据。
- 普通用户在权限或数据范围外的所有 mutation 均稳定返回 403。
- 任务在重复启动、暂停、取消、失败重试和 Worker 重启后保持确定结果。
- 凭据、组件密码、kubeconfig、Secret 值不进入 API、日志、审计、导出或 evidence。
- clean checkout 可从锁定的 Base foundation 重建并通过 V1 必选门禁。

## Personas and jobs

| Persona          | 核心工作                                           | 关键约束                                                |
| ---------------- | -------------------------------------------------- | ------------------------------------------------------- |
| 平台管理员       | 配置业务模块权限、菜单和数据范围，维护发布基线     | 不能绕过 Base IAM、审计和 human freeze gate             |
| SRE / 运维工程师 | 维护资源、执行部署、管理集群、处理失败和恢复       | 需要明确目标、实时反馈、幂等控制和可追溯日志            |
| 业务负责人       | 维护业务域、应用、服务和责任归属，查看本域运行状态 | 只能访问所属数据范围，不直接操作底层凭据                |
| 安全 / 审计人员  | 检查高风险操作、权限、密钥暴露和变更证据           | 需要不可变事件、关联 ID、脱敏输出和导出授权             |
| 发布维护者       | 审核冻结检查、外部运行矩阵和清理结果               | 只有维护者能把 `READY FOR HUMAN FREEZE` 提升为 `FROZEN` |

## Information architecture

### Primary navigation

```text
运维平台
├── 业务域
├── 服务目录
├── CMDB
│   ├── 主机
│   ├── 动态分组
│   └── 标签模型
├── 安装部署
│   ├── 软件组件
│   ├── 任务模板
│   └── 部署任务
└── Kubernetes
    ├── 集群
    ├── 工作负载
    └── 发布记录
```

ConfigMap、Secret、Namespace、Node、Pod 日志等集群上下文能力优先放在集群详情或工作负载详情中，不为每种 Kubernetes 对象扩张一级导航。Kubernetes Service CRUD 是 V1 冻结目标，未完成前不能标为现有功能。Observability 是 V1 排除项，不出现在默认导航和默认运行时。

### Core routes/screens

- 业务域列表与详情
- 服务目录（Application、Service、ServiceInstance）
- CMDB 主机列表/详情、动态分组、标签模型
- 软件组件、任务模板、部署任务列表与任务详情
- Kubernetes 集群列表/详情、工作负载、发布记录

### Content hierarchy

1. 页面先回答“当前对象、归属和状态是什么”。
2. 再展示需要处理的异常、冲突或阻断。
3. 然后提供与当前状态相符、且有权限的动作。
4. 最后提供历史事件、日志、审计和关联资源。

## Design principles

1. **事实与意图分离**：业务目录表达责任和期望，CMDB/Kubernetes 表达资源事实，Deploy 表达变更意图和执行历史。
2. **权限先于交互**：菜单可见、页面可见、数据可见和动作可执行是四个独立判断；后端是最终门禁。
3. **危险动作可解释**：启动、暂停、取消、重试、删除、轮换凭据和 Kubernetes mutation 必须展示目标、影响和前置条件。
4. **失败是一等状态**：页面和 API 同时设计 loading、empty、error、forbidden、conflict、partial failure 和 recovery。
5. **快照保护历史**：执行目标、模板、组件版本和关键参数在任务创建/启动时固化，后续资源变化不改写历史。
6. **最小共享面**：跨模块只使用 typed capability 和稳定 Ref，不共享 GORM Model、repository 或裸表访问。

## Visual language

- **Color**：沿用 Base semantic tokens；品牌色只用于主动作与选中态，成功、警告、失败、信息和中性状态使用独立语义色。
- **Typography**：沿用 Base 字体和层级；运维数据、IP、版本、资源标识和日志使用适合等宽阅读的样式。
- **Spacing/layout rhythm**：4/8px 节奏；操作台信息密度高但分组清晰，固定工具栏和状态区不得挤压正文。
- **Shape/radius/elevation**：圆角不超过 Base 约束；层次主要依赖边框、间距与轻量阴影，不使用嵌套卡片制造层级。
- **Motion**：只用于状态反馈和视图过渡，150-300ms；执行进度不能用无限动画伪装真实状态。
- **Imagery/iconography**：统一使用 Base 已安装图标库；图标按钮必须有 tooltip，状态不能只靠颜色区分。

## Components

- **优先复用**：Base 的 `PageContainer`、`PageHeader`、`FilterPanel`、`AppTable`、`ListHeaderActions`、`FormSection`、`SubmitBar`、`AppModal`、`PageState`。
- **业务组件**：资源引用选择器、业务域选择器、目标快照预览、状态转换操作组、任务事件流、脱敏日志查看器、Kubernetes 资源版本冲突提示。
- **Variants and states**：默认、hover、focus、disabled、loading、success、warning、error、forbidden、conflict、stale、partial failure。
- **Ownership**：通用组件和 token 回 Base；只表达 Ops 业务语义的组合组件留在 `frontend/src/modules/business/*`。

## Accessibility

- **目标标准**：以 Base 可访问性契约为准，至少满足键盘可操作、可见焦点、语义标签和足够对比度。
- **Keyboard/focus**：弹窗、抽屉和确认流程正确管理焦点；执行动作不可只依赖 hover 菜单。
- **Contrast/readability**：状态同时使用文本/图标/形状；日志、表格和暗色主题保持可读。
- **Screen reader**：任务状态变化、校验错误和异步结果使用适当 live region；图标按钮提供可读名称。
- **Reduced motion**：尊重 reduced-motion；实时日志和轮询/SSE 更新不得抢夺焦点。

## Responsive behavior

- **Supported devices**：桌面为主要工作面，兼容窄桌面、平板和移动端查看/处置关键状态。
- **Layout adaptations**：宽表格允许受控横向滚动或列优先级收缩；详情页在窄屏改为单列，操作栏不能覆盖内容。
- **Touch/hover**：关键动作在触摸设备始终可发现；hover 只提供增强提示，不承载唯一入口。
- **Safety**：移动端不弱化危险操作确认、影响范围和权限反馈。

## Interaction states

- **Loading**：骨架保持布局稳定；mutation 使用局部 loading，防止重复提交。
- **Empty**：区分初始空、筛选空和权限不可见；只在有创建权限时显示创建引导。
- **Error**：保留用户输入和筛选上下文，展示稳定错误 key 对应的可执行恢复建议。
- **Success**：展示结果和下一步；异步任务创建成功不等于执行成功。
- **Disabled**：给出状态、权限或依赖原因，不能只把按钮置灰。
- **Offline/slow network**：实时事件断开时标记 stale，并按文档化策略降级到轮询；不得把旧状态显示成实时状态。

## Content voice

- **Tone**：直接、精确、可操作，不使用夸大和模糊成功文案。
- **Terminology**：统一使用业务域、应用、服务、服务实例、资源、目标快照、任务、尝试、步骤、事件、发布。
- **Microcopy**：动作使用动词；危险确认包含对象、数量和不可逆影响；错误提示通过 `business.*` i18n key 管理。
- **Status language**：区分“请求已接受”“任务执行中”“执行成功”“部分失败”“证据未完成”和“已冻结”。

## Implementation constraints

- **Framework/styling**：继承 Base 的 Go/Gin/GORM、React/TypeScript、Arco Design 和 Pantheon tokens；不新增 UI 库。
- **Architecture**：V1 保持模块化单体；业务模块通过 overlay 注册表装配，跨模块使用 `business/capability`。
- **Data**：schema 变更必须使用 `backend/pkg/database/migrations` 的成对版本化 migration；运行时事实的 owner 保持唯一。
- **Security**：凭据只通过 `CredentialRef`；秘密值写入后不可回显；日志、事件、审计、导出和 evidence 统一脱敏。
- **Performance/reliability**：外部调用有超时；任务并发有界；状态转换使用 version/CAS；Kubernetes mutation 使用 `resourceVersion`。
- **Compatibility**：完整 Base 快照加声明式 business overlay；共享问题回 Base 修复后再升级 foundation。
- **Verification**：文档不得把规划写成完成；实现完成必须通过冻结清单中的自动门禁、外部运行矩阵和独立复核。

## Release and roadmap

| 阶段 | 范围                                                         | 当前判定                     |
| ---- | ------------------------------------------------------------ | ---------------------------- |
| V1   | BizScope、Service、CMDB、Deploy、Kubernetes 的安全可恢复闭环 | `BLOCKED`，以冻结检查表为准  |
| P1   | 通用资源事实中心、授权/审计/可靠性 ratchet                   | V1 冻结后启动                |
| P2   | Observability、GitOps、告警、事件、SLO                       | 明确排除在 V1 默认运行时之外 |

## Open questions

- [ ] V1 外部 VM 与 Kubernetes 验收目标由维护者授权并记录；没有授权时只保留为 release blocker。
- [ ] V1 冻结后决定通用 Resource 模型的迁移窗口和兼容投影期限。
- [ ] Observability 进入默认运行时前，需要独立设计评审、权限矩阵、migration 和前端完整闭环。

## Document reading order

1. `DESIGN.md`
2. `AGENTS.md`
3. `docs/README.md`
4. `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`
5. `docs/V1_RELEASE_CHECKLIST.md`
6. `docs/PROJECT_INHERITANCE.md`
7. `../pantheon-base/DESIGN.md`
8. `../pantheon-base/AGENTS.md`
9. `../pantheon-base/docs/README.md`
10. `../pantheon-base/docs/contracts/` 中与任务匹配的共享契约
11. `../pantheon-base/docs/designs/` 中与任务匹配的平台/系统域设计
12. `../pantheon-base/docs/acceptances/` 中与任务匹配的共享验收标准
13. `docs/designs/BUSINESS_*` 模块事实源
14. 对应 `.harness` 任务、状态、证据和 review

本仓库不维护 `docs/contracts/`、`docs/acceptances/`、`docs/archive/` 或 `docs/assessments/` 的底座副本；历史过程稿统一放在 `archive/2026-08-process/`，且不能覆盖当前冻结状态。
