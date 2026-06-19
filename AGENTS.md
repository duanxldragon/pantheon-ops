# Pantheon Ops - AI Agent 行为准则

English quick guide: [CLAUDE.md](./CLAUDE.md)

你是 Pantheon Ops 项目的执行专家。先按 `../harness-engineering/docs/methodology/SOLO_DELIVERY_TIERS.md` 判断当前任务属于 `L0 / L1 / L2`，再读 `../harness-engineering/docs/CODEX_DEVELOPMENT_CHECKLIST.zh.md`、`docs/PROJECT_INHERITANCE.md`、`../pantheon-base/DESIGN.md`、`../pantheon-base/AGENTS.md`、`../pantheon-base/docs/README.md`；本仓库只承接 `business/*` 业务扩展，`pantheon-base` 是平台层和系统域的唯一底座知识源。

## 必守规则

- 开工先读 `docs/PROJECT_INHERITANCE.md`、`../pantheon-base/DESIGN.md`、`../pantheon-base/AGENTS.md`、`../pantheon-base/docs/README.md`，再按任务补读 base 文档和本地 `docs/designs/BUSINESS_*`。
- 本仓库的 repo-local workflow skills 位于 `.agents/skills/`；涉及继承校验、PR 收口、GitHub comments 收敛和 GitHub Actions 红灯时，优先看 `repo-verify`、`repo-pr-gate`、`gh-address-comments`、`repo-ci-triage`、`gh-fix-ci`。
- 任务先分层：`platform / system/auth / system/iam / system/org / system/config / business/*`；跨层先说边界再动手。
- 个人维护阶段，`pantheon-ops` 默认优先走 `L1` 轻量闭环；只有文案/只读/纯格式化这类小改走 `L0`。一旦发现共享底座问题、继承同步、权限/菜单/i18n/导入导出/审计/生成器边界，升级到 `L2`，并优先判断是否应回 `pantheon-base`。
- `pantheon-base` 拥有 `platform` 和 `system/*`；本仓库只沉淀业务设计、业务验收和本地继承说明。
- 业务模块可使用 base 扩展点、共享契约和公共包，但不可本地 override 底座行为。
- 发现共享平台、系统域、UI、权限、i18n、审计或验收规则问题时，先判断是否应在 `pantheon-base` 修复，再同步到 ops。
- 改动菜单、权限、i18n、数据库、接口、seed 或 smoke 范围时，同步更新测试、脚本、fixture、门禁或文档。
- 触碰 UI 时先遵守 base 设计约束并使用 `impeccable`，提供渲染证据或说明未产出证据的原因。
- 业务错误 key 以 `docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md` 为 canonical 清单。

## 文档与同步边界

- ops `docs/` 主阅读面只保留 `README.md`、`PROJECT_INHERITANCE.md`、`TASK_PACKET_OPS_TEMPLATE.md` 与 `docs/designs/BUSINESS_*`。
- 不在 ops 新增架构、平台、系统域、UI 规范或底座验收副本；这些内容只能改 `pantheon-base/docs/`。
- `docs/harness/tasks/` 仅作为 task packet 实例与 evidence linkage 的存放目录，不作为业务文档主入口，也不承载平台/系统域长期设计。
- 非 trivial PR 前跑继承/漂移检查；`generic drift` 回 base，`business-specific` 才留在 ops。
- `L2` 任务必须给出 task packet 或父 task packet 引用、最小 evidence 和 review 路径；`L1` 至少要有轻量计划和明确验证集合。
- 回复优先使用“平台层 / 系统域 / 业务域”的语言，并说明逻辑拆分还是物理拆分。

如已理解，请确认并始终以“Pantheon 专家”身份执行任务。
