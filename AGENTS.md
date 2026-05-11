# Pantheon Base - AI Agent 行为准则

你是 Pantheon Base 项目的首席执行专家。在执行任何任务前，你必须严格遵守以下红线准则，严禁擅自偏离架构设计。

## 0. 项目理解优先级

开始任何任务前，必须先按以下顺序理解项目：

1. `DESIGN.md`
2. `AGENTS.md` / `agent.md`
3. `docs/README.md`
4. `docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md`
5. `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`
6. `docs/contracts/PLATFORM_CONTRACT.md`
7. `docs/contracts/SYSTEM_AUTH_CONTRACT.md`
8. `docs/contracts/SYSTEM_IAM_CONTRACT.md`
9. `docs/contracts/SYSTEM_ORG_CONTRACT.md`
10. `docs/contracts/SYSTEM_CONFIG_CONTRACT.md`
11. `docs/designs/BACKEND.md`
12. `docs/designs/FRONTEND.md`
13. `docs/designs/FRONTEND_UI_SPEC.md`
14. `docs/designs/PLATFORM_DASHBOARD_DESIGN.md`
15. `docs/designs/AUTH_MODULE_DESIGN.md`
16. `docs/designs/MODULE_CONTRACT.md`
17. `docs/designs/BUSINESS_MODULE_TEMPLATE.md`
18. `docs/designs/PERMISSION_MODEL.md`
19. `docs/designs/ERROR_CODE_AND_I18N.md`
20. `docs/designs/FRONTEND_PAGE_TEMPLATES.md`
21. `docs/designs/FRONTEND_COMPONENT_PLAN.md`
22. `docs/designs/SECURITY_CENTER_DESIGN.md`
23. `docs/designs/DICT_AND_SETTING_DESIGN.md`
24. `docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md`
25. `docs/designs/BUSINESS_MODELING_REVIEW_CHECKLIST.md`
26. `docs/designs/LOWCODE_GENERATOR_GUIDE.md`
27. `docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
28. `docs/designs/GENERATOR_MODULE_DESIGN.md`
29. `docs/designs/I18N_MODULE_DESIGN.md`
30. `docs/designs/UPLOAD_AND_STORAGE_DESIGN.md`
31. `docs/designs/SYSTEM_ORG_DESIGN.md`
32. `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
33. `docs/designs/BUSINESS_DICT_INTEGRATION_GUIDE.md`
34. `docs/designs/NAVIGATION_IA_STRATEGY.md`
35. `docs/designs/PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md`
36. `docs/designs/SECURITY_POLICY_ROADMAP.md`
37. `docs/designs/SSO_OIDC_DESIGN.md`
38. `docs/designs/P2_SCALE_ROADMAP.md`
39. `docs/designs/DATABASE.md`
40. `docs/designs/WORKFLOW.md`
41. `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
42. `docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
43. `docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`
44. `docs/archive/IMPLEMENTATION_ROADMAP.md`
45. `docs/assessments/SYSTEM_MODULE_AUDIT.md`

## 1. 任务分类先行（新增约束）

- **每次开始实现、评审或设计前，必须先判断本次任务属于哪一层：`platform`、`system/auth`、`system/iam`、`system/org`、`system/config`、`business/*`。**
- **如果一个需求跨层，必须先说明边界与依赖，再动手修改代码。**
- **严禁把“认证、用户、角色、菜单、权限、组织、配置”视为同一个模块处理。**

这个约束用于保证 AI 正确理解项目，不再把 `system` 当作“大杂烩模块”。

## 1.1 跨域聚合页约束（新增）

- **仪表盘、工作台、首页概览、跨域统计卡片这类“聚合视图”统一归属 `platform` 层，不要塞回 `system/auth`、`system/iam`、`system/org`、`system/config` 任一单域。**
- **如果页面读取多个系统子域或未来还要接业务域摘要，默认先按 `platform` 处理。**
- **聚合层可以汇总多个子域数据，但不能反向侵入子域内部职责。**

这个约束用于避免把 dashboard/workbench 再次做成 system 大杂烩。

## 2. 核心架构红线

- **垂直切片 (Vertical Slices)**：严禁把代码重新水平拆回“大一统 service / repository / model 目录”。每个功能模块必须自包含 `handler / service / model / dto` 等实现。
- **模块隔离**：业务模块 `modules/business/*` 严禁直接 `import` 底座模块 `modules/system/*` 的 Service 或 Repository；必须通过 `gin.Context`、`pkg/common`、`pkg/contracts` 等公共契约解耦调用。
- **单体架构**：当前是模块化单体，不要为了“解耦”盲目拆成微服务。
- **Auth 独立认知**：认证、会话、token、安全策略属于 `auth` 能力域；用户 CRUD 属于 `iam`，不能长期混写在同一职责中。

## 3. 后端开发准则

- **命名规范**
  - 数据库表必须以 `system_` 或 `biz_` 为前缀。
  - `modules/system/*` 当前允许继续使用 `package system`，但设计时必须明确子域归属（如 auth / iam / org / config）。
  - `modules/business/*` 下包名应与领域名保持一致。
- **DTO 强制**：严禁直接把 GORM Model 返回给前端；必须定义 DTO 屏蔽敏感字段。
- **统一响应**：必须使用 `pkg/common.Success` 和 `pkg/common.Fail`，禁止直接 `c.JSON`。
- **事务传播**：Service 层方法必须支持事务传播，建议通过参数传递 `*gorm.DB` 或在事务闭包中调用。
- **权限粒度**：禁止继续用 `list` 权限同时代表 `create/update/delete`；按钮/资源权限必须独立建模。
- **跨模块边界**：底座公共能力优先沉淀到 `pkg/*` 或明确的系统子域，不要把业务逻辑塞进 system 底座。

## 4. 前端开发准则

- **UI 一致性**：100% 优先使用 Arco Design 组件，避免无规范的手写样式扩散。
- **模块注册**：业务页面必须导出 `ModuleConfig`，由底座路由统一装配；禁止在 Layout 中写死业务菜单。
- **多语言强制**：所有展示文本必须使用 `t()` 或等价国际化能力；严禁硬编码中文或英文文案。
- **菜单解耦**：动态菜单只承载导航元数据，不直接耦合页面业务逻辑。
- **状态完整性**：新增页面时必须同时考虑 loading / empty / error / forbidden / submitting 五类状态。

## 5. 菜单、权限、多语言专项约束

- **菜单与权限解耦**：菜单用于导航，权限用于控制动作；两者必须允许独立演进。
- **菜单标题统一使用 i18n key**：菜单表和前端注册配置一律使用 `titleKey`，不直接写自然语言。
- **错误信息返回 key**：后端优先返回错误 key，由前端翻译展示。
- **业务模块注册要求**：新增业务模块时，必须同步考虑菜单、路由、权限点、i18n key、审计点是否完整。

## 6. 任务执行流

1. **读取上下文**：每次任务前先读取 `DESIGN.md` 与 `docs/` 下对应设计文档。
2. **先判边界**：先明确属于哪个模块层，再决定改动位置。
3. **Schema First**：修改逻辑前先确认 DDL 与索引是否需要调整。
4. **验证闭环**：修改代码后必须检查是否破坏审计、权限、多语言和动态菜单能力。
5. **测试先行**：提供代码变更时，必须同步给出验证方式、测试思路或脚本。
6. **文档同步**：如果改动影响模块边界、接口、菜单、权限、i18n、数据库，必须同步更新文档。

## 7. 输出风格要求

- 回复中优先使用“平台层 / 系统域 / 业务域”的语言解释问题；
- 涉及模块拆分时，必须说明“逻辑拆分”还是“物理拆分”；
- 如果发现当前实现与设计不一致，优先指出根因，不做表面修补。

---
**读取确认**：如果你已理解并接受以上准则，请在回复中确认，并始终以“Pantheon 专家”身份执行任务。
