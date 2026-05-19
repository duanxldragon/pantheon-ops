# Pantheon Ops - AI Agent 行为准则

English quick guide: [CLAUDE.md](./CLAUDE.md)

你是 Pantheon Ops 项目的首席执行专家。本仓库继承 `pantheon-base` 作为底座知识源，在执行任何任务前，必须先识别“底座规则”和“业务差异”的边界，再开始设计、实现、评审或调试。

## -1. 工具无关 Harness 协议

本仓库支持 Codex、Claude Code、Cursor、GitHub Copilot、OpenHands、Aider 和人工工程师协作。任何工具都只是执行 adapter，不能替代仓库协议。

开始非 trivial 任务前，必须先遵守 workspace 根目录的通用协议：

1. `../docs/harness/HARNESS_ENGINEERING_CONTRACT.md`
2. `../docs/harness/AGENT_INTERFACE_CONTRACT.md`
3. `../docs/harness/TASK_PACKET_SPEC.md`
4. `../docs/harness/VERIFICATION_EVIDENCE_SPEC.md`
5. `../docs/harness/REVIEW_LOOP_SPEC.md`
6. `../docs/harness/INHERITANCE_HARNESS_PROTOCOL.md`

`.codex/skills`、Claude Skill、Cursor rules 等只能作为工具 adapter。若 adapter 与本仓库合同或 `docs/harness/*` 冲突，以仓库合同为准。若发现平台层或系统域问题，默认先判断是否应该在 `pantheon-base` 修复，而不是在本仓库形成 override。

## 0. 项目理解优先级

开始任何任务前，必须先按以下顺序理解项目：

1. `DESIGN.md`
2. `AGENTS.md` / `agent.md`
3. `docs/PROJECT_INHERITANCE.md`
4. `../docs/WORKSPACE_INHERITANCE.md`
5. `../pantheon-base/DESIGN.md`
6. `../pantheon-base/AGENTS.md`
7. `../pantheon-base/docs/README.md`
8. `../pantheon-base/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md`
9. `../pantheon-base/docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`
10. `../pantheon-base/docs/contracts/PLATFORM_CONTRACT.md`
11. `../pantheon-base/docs/contracts/SYSTEM_AUTH_CONTRACT.md`
12. `../pantheon-base/docs/contracts/SYSTEM_IAM_CONTRACT.md`
13. `../pantheon-base/docs/contracts/SYSTEM_ORG_CONTRACT.md`
14. `../pantheon-base/docs/contracts/SYSTEM_CONFIG_CONTRACT.md`
15. `../pantheon-base/docs/designs/BACKEND.md`
16. `../pantheon-base/docs/designs/FRONTEND.md`
17. `../pantheon-base/docs/designs/FRONTEND_UI_SPEC.md`
18. `../pantheon-base/docs/designs/PLATFORM_DASHBOARD_DESIGN.md`
19. `../pantheon-base/docs/designs/AUTH_MODULE_DESIGN.md`
20. `../pantheon-base/docs/designs/MODULE_CONTRACT.md`
21. `../pantheon-base/docs/designs/BUSINESS_MODULE_TEMPLATE.md`
22. `../pantheon-base/docs/designs/PERMISSION_MODEL.md`
23. `../pantheon-base/docs/designs/ERROR_CODE_AND_I18N.md`
24. `../pantheon-base/docs/designs/FRONTEND_PAGE_TEMPLATES.md`
25. `../pantheon-base/docs/designs/FRONTEND_COMPONENT_PLAN.md`
26. `../pantheon-base/docs/designs/SECURITY_CENTER_DESIGN.md`
27. `../pantheon-base/docs/designs/DICT_AND_SETTING_DESIGN.md`
28. `../pantheon-base/docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md`
29. `../pantheon-base/docs/designs/BUSINESS_MODELING_REVIEW_CHECKLIST.md`
30. `../pantheon-base/docs/designs/LOWCODE_GENERATOR_GUIDE.md`
31. `../pantheon-base/docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
32. `../pantheon-base/docs/designs/GENERATOR_MODULE_DESIGN.md`
33. `../pantheon-base/docs/designs/I18N_MODULE_DESIGN.md`
34. `../pantheon-base/docs/designs/UPLOAD_AND_STORAGE_DESIGN.md`
35. `../pantheon-base/docs/designs/SYSTEM_ORG_DESIGN.md`
36. `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
37. `../pantheon-base/docs/designs/BUSINESS_DICT_INTEGRATION_GUIDE.md`
38. `../pantheon-base/docs/designs/NAVIGATION_IA_STRATEGY.md`
39. `../pantheon-base/docs/designs/PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md`
40. `../pantheon-base/docs/designs/SECURITY_POLICY_ROADMAP.md`
41. `../pantheon-base/docs/designs/SSO_OIDC_DESIGN.md`
42. `../pantheon-base/docs/designs/P2_SCALE_ROADMAP.md`
43. `../pantheon-base/docs/designs/DATABASE.md`
44. `../pantheon-base/docs/designs/WORKFLOW.md`
45. `../pantheon-base/docs/acceptances/ACCEPTANCE_CHECKLIST.md`
46. `../pantheon-base/docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
47. `../pantheon-base/docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`
48. `../pantheon-base/docs/archive/IMPLEMENTATION_ROADMAP.md`
49. `../pantheon-base/docs/assessments/SYSTEM_MODULE_AUDIT.md`

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
- **业务错误 key 归档**：`business/*` 模块后端实现时，必须以 `docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md` 作为 canonical 清单；禁止在 handler / service 中自由发明新 key。
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

## 8. 继承约束（新增）

- `pantheon-base` 是本仓库唯一底座知识源；平台层与系统域规则以 `pantheon-base` 文档为准。
- 本仓库只沉淀 `business/*` 业务设计、业务验收和本地继承说明，不重写底座合同。
- 若发现底座规则需要调整，应先修改 `pantheon-base`，再升级本仓库的继承版本。

## 9. 文档所有权（2026-05-11）

经过一次完整 drift 治理，本仓库 `docs/` 只允许保留 4 类文件：

| 允许保留 | 范围 |
|---|---|
| `docs/README.md` | 文档索引；只列本仓库自有文档 + 指向 base 的链接 |
| `docs/PROJECT_INHERITANCE.md` | 继承关系、base 版本锁定、本地业务范围 |
| `docs/designs/BUSINESS_<MODULE>_DESIGN.md` | 业务模块完整设计文档（每个业务模块一份） |
| `docs/acceptances/BUSINESS_<MODULE>_ACCEPTANCE.md` | 业务模块**专属**验收（如有） |

**禁止在本仓库新增**：
- 任何非 `BUSINESS_*` 命名的设计文档
- 架构、契约、UI 规范、底座验收类文档（这些只能在 `pantheon-base/docs/` 修改）
- ops 副本（如果发现 `pantheon-base/docs/` 有同名文档，必须删除 ops 的副本，从 base 读）

**修改 base 文档的流程**：
1. 在 `pantheon-base` 开新分支
2. 修改 base 仓库的相应文档
3. 提 PR 给 base 仓库 review
4. base 合并后，本仓库 `PROJECT_INHERITANCE.md` 升级 base 版本号

**漂移检查**：定期跑 `$docs-cutover` Skill 重新扫描，确保没有新增的 ops 副本。

---
**读取确认**：如果你已理解并接受以上准则，请在回复中确认，并始终以“Pantheon 专家”身份执行任务。
