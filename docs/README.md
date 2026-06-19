# pantheon-ops 文档索引

English version: [README.en.md](./README.en.md)

`pantheon-ops` 是 Pantheon 平台的运维域业务仓库，从 `pantheon-base` 继承底座架构、契约和验收标准。

默认继承方式已经调整为消费 `pantheon-base` 的 foundation release，而不是直接追随 `main`。

本目录只承载本仓库**自有的业务文档**。架构、契约、UI 规范、通用验收等基座级文档全部由 `pantheon-base/docs/` 提供，不在 ops 复制副本。

---

## 1. 本仓库自有文档（中文主入口 + 英文 companion）

| 文档 | 用途 |
|---|---|
| [README.md](./README.md) / [README.en.md](./README.en.md) | 仓库入口与推荐阅读顺序 |
| [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md) / [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md) | 与 `pantheon-base` 的继承关系、版本锁定、本地业务范围与 override 边界 |
| [../.agents/skills/README.zh.md](../.agents/skills/README.zh.md) / [../.agents/skills/README.md](../.agents/skills/README.md) | repo-local Codex skills：继承校验、PR 收口、GitHub comments 自动处理、CI 红灯排查 |
| [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md) / [TASK_PACKET_OPS_TEMPLATE.en.md](./TASK_PACKET_OPS_TEMPLATE.en.md) | `pantheon-ops` 业务开发与 `base -> ops` 同步的最小任务包模板 |
| [designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md](./designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md) / [designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md](./designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md) | 业务域模块设计：业务域台账、主机绑定边界与 Deploy 信任来源 |
| [designs/BUSINESS_CMDB_MODULE_DESIGN.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.md) / [designs/BUSINESS_CMDB_MODULE_DESIGN.en.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.en.md) | CMDB 业务模块完整设计（含数据模型、API、UI、字典依赖、验收） |
| [designs/BUSINESS_DEPLOY_MODULE_DESIGN.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.md) / [designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md) | 安装部署业务模块完整设计 |
| [designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md) / [designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md) | ops 业务模块 canonical 错误 key 语义附录 |
| [designs/PLATFORM_SRE_EVOLUTION_PLAN.md](./designs/PLATFORM_SRE_EVOLUTION_PLAN.md) / [designs/PLATFORM_SRE_EVOLUTION_PLAN.en.md](./designs/PLATFORM_SRE_EVOLUTION_PLAN.en.md) | SRE 演进路线图：从 Web 平台到 K8s-native 的五阶段计划 |
| [../DESIGN.md](../DESIGN.md) / [../DESIGN.en.md](../DESIGN.en.md) | 仓库级总体设计与继承的基座设计立场 |
| [../frontend/README.md](../frontend/README.md) / [../frontend/README.en.md](../frontend/README.en.md) | 前端工作区说明 |
| [../frontend/tests/smoke/README.md](../frontend/tests/smoke/README.md) / [../frontend/tests/smoke/README.en.md](../frontend/tests/smoke/README.en.md) | smoke 测试布局与覆盖边界 |
| [../CONTRIBUTING.md](../CONTRIBUTING.md) / [../CONTRIBUTING.en.md](../CONTRIBUTING.en.md) | 协作规范、提交格式与验证要求 |
| [../SECURITY.md](../SECURITY.md) / [../SECURITY.en.md](../SECURITY.en.md) | 安全问题报告范围与披露规则 |

说明：

- 中文 `.md` 仍是本仓库主阅读面。
- `.en.md` companion 用于国际协作、外部分享和后续扩展。
- `PROJECT_INHERITANCE` 已切换到与其余文档一致的模式：`PROJECT_INHERITANCE.md` 为中文主入口，`PROJECT_INHERITANCE.en.md` 为英文 companion。
- GitHub 协作默认走原生 PR 自动化：`Quality Gates`、`Security Gates`、Copilot review 请求（可用时）与 squash auto-merge，不再依赖 Sonar/Codacy/OCR 门禁。

建议实际阅读时先看：

1. [README.md](../README.md)
2. [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)
3. [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md)
4. [designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md](./designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md)
5. `designs/BUSINESS_*`

---

## 2. 基座文档（去 pantheon-base 看）

以下文档**不要在 ops 复制副本**。读时直接看 base 仓库：

### 2.1 项目入口与文档治理
- `../../pantheon-base/DESIGN.md`
- `../../pantheon-base/AGENTS.md`
- `../../pantheon-base/docs/README.md`（base 文档总索引）
- `../../pantheon-base/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md`
- `../../pantheon-base/docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`

### 2.2 契约（系统域 API 与协议）
- `../../pantheon-base/docs/contracts/PLATFORM_CONTRACT.md`
- `../../pantheon-base/docs/contracts/SYSTEM_AUTH_CONTRACT.md`
- `../../pantheon-base/docs/contracts/SYSTEM_IAM_CONTRACT.md`
- `../../pantheon-base/docs/contracts/SYSTEM_ORG_CONTRACT.md`
- `../../pantheon-base/docs/contracts/SYSTEM_CONFIG_CONTRACT.md`

### 2.3 架构与规范
- `../../pantheon-base/docs/designs/BACKEND.md` 总体架构与后端规范
- `../../pantheon-base/docs/designs/FRONTEND.md` 前端架构与模块接入
- `../../pantheon-base/docs/designs/FRONTEND_UI_SPEC.md` 前端 UI 详细规范
- `../../pantheon-base/docs/designs/FRONTEND_PAGE_TEMPLATES.md` 前端页面模板规范
- `../../pantheon-base/docs/designs/FRONTEND_COMPONENT_PLAN.md` 前端组件规划
- `../../pantheon-base/docs/designs/BACKOFFICE_STYLE_CONSTRAINTS.md` 后台风格硬约束
- `../../pantheon-base/docs/designs/NAVIGATION_IA_STRATEGY.md` 导航信息架构
- `../../pantheon-base/docs/designs/PERMISSION_MODEL.md` 权限模型设计
- `../../pantheon-base/docs/designs/MODULE_CONTRACT.md` 模块契约设计
- `../../pantheon-base/docs/designs/BUSINESS_MODULE_TEMPLATE.md` 业务模块设计模板（写 ops 业务模块时套用）

### 2.4 系统域设计（base 自有）
- `../../pantheon-base/docs/designs/AUTH_MODULE_DESIGN.md`
- `../../pantheon-base/docs/designs/SYSTEM_ORG_DESIGN.md`
- `../../pantheon-base/docs/designs/DICT_AND_SETTING_DESIGN.md`
- `../../pantheon-base/docs/designs/I18N_MODULE_DESIGN.md`
- `../../pantheon-base/docs/designs/UPLOAD_AND_STORAGE_DESIGN.md`
- `../../pantheon-base/docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
- `../../pantheon-base/docs/designs/GENERATOR_MODULE_DESIGN.md`
- `../../pantheon-base/docs/designs/BUSINESS_DICT_INTEGRATION_GUIDE.md` 业务字典接入通用指南
- `../../pantheon-base/docs/designs/ERROR_CODE_AND_I18N.md`
- `../../pantheon-base/docs/designs/DATA_PERMISSION_HOOK.md`
- `../../pantheon-base/docs/designs/SECURITY_CENTER_DESIGN.md`
- `../../pantheon-base/docs/designs/SECURITY_POLICY_ROADMAP.md`
- `../../pantheon-base/docs/designs/SSO_OIDC_DESIGN.md`
- `../../pantheon-base/docs/designs/PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md`
- `../../pantheon-base/docs/designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md`
- `../../pantheon-base/docs/designs/PLATFORM_DASHBOARD_DESIGN.md`
- `../../pantheon-base/docs/designs/LOWCODE_GENERATOR_GUIDE.md`
- `../../pantheon-base/docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md`
- `../../pantheon-base/docs/designs/BUSINESS_MODELING_REVIEW_CHECKLIST.md`
- `../../pantheon-base/docs/designs/P2_SCALE_ROADMAP.md`
- `../../pantheon-base/docs/designs/WORKFLOW.md`
- `../../pantheon-base/docs/designs/GSTACK_WINDOWS_GUIDE.md`

### 2.5 验收
- `../../pantheon-base/docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- `../../pantheon-base/docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`
- `../../pantheon-base/docs/acceptances/CODE_REVIEW_STANDARD.md`
- `../../pantheon-base/docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
- `../../pantheon-base/docs/acceptances/SYSTEM_IMPORT_EXPORT_SMOKE_GUIDE.md`
- `../../pantheon-base/docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- `../../pantheon-base/docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`
- `../../pantheon-base/docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`
- `../../pantheon-base/docs/acceptances/PLATFORM_SHELL_PR_CHECKLIST_SNIPPET.md`

### 2.6 历史基线与样例
- `../../pantheon-base/docs/archive/`
- `../../pantheon-base/docs/assessments/`
- `../../pantheon-base/docs/remediations/`

---

## 3. 文档治理约束

参见 `AGENTS.md` 中「文档所有权」章节。简单规则：

- 本仓库**只能新增** `designs/BUSINESS_*` 命名的业务模块设计文档
- 本仓库**禁止**新增非业务文档；如需修改架构、契约、UI 规范，开 PR 到 `pantheon-base`
- 本仓库可以维护 `PROJECT_INHERITANCE.md` 锁定的 base 版本和本地业务范围
- 非 trivial 的业务实现或 `base -> ops` 同步，优先先写一份 `TASK_PACKET_OPS_TEMPLATE` 风格的最小任务包
- `docs/harness/tasks/` 仅保留 task packet 实例与 evidence linkage，不作为业务文档主索引；对应任务若被稳定设计文档完全吸收，应在后续文档治理中清理
