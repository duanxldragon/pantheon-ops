---
title: Pantheon Ops 业务菜单信息架构改造（bizscope 三菜单 / 服务目录拆分 / 应用发布去品牌化）
doc_type: Remediation
layer: business/*
status: Active
updated_at: 2026-11-14
linked_contracts:
  - docs/harness/TASK_PACKET_SPEC.md
  - docs/harness/VERIFICATION_EVIDENCE_SPEC.md
  - docs/harness/VISUAL_QUALITY_PROTOCOL.md
---

# Task Packet: 2026-11-13-ops-business-menu-ia-refactor

## Goal

把 pantheon-ops 业务导航从两个单薄单页重构为两个 M 父目录共六个 C 二级菜单：bizscope 拆为业务域列表 / 主机绑定 / 环境管理（新增 `biz_business_environment` 子资源 CRUD），服务目录把应用 / 服务 / 服务实例三个 Tab 拆为左侧菜单（复用现有 list API，零 API 改动）；旧菜单路径走退役 overlay 清理；ArgoCD 菜单仅做显示文案去品牌化（「应用发布」/「Application Delivery」），内部标识符全部保留。seed、注册表、i18n、smoke、overlay 契约在同一交付内同步关闭。L2 任务，执行细节与精确标识符见 `plan.md`。

## Priority

P2；L2（触碰共享菜单/权限/i18n 合同与业务模块继承边界，需要 task packet + evidence + 独立评审 + 人工验收）。

## Primary Layer

business/*

## Dependency Layers

- business/*（bizscope、service、cicd、cmdb、retired overlay）
- system/iam（组件白名单注册表 `business_overlay_component_registry.go`）

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base（锁定的 foundation release，提供 seed/retirement/menu 合同）、pantheon-harness（方法论）
- Downstream Consumers: none
- Sync Expectation: none
- Release Requirement: none

## Harness Profile

- Template: ui-heavy-product
- Overlay: pantheon-ops
- Quality Profile: ci-workflow
- Portable Failure Class: menu-ia-contract-gap
- Owner Layer: business-consumer
- Ratchet Decision: sensor-added（新增 bizscope smoke 套件与 seed 测试，菜单契约漂移由现有机械门禁持续拦截）
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality

## Contract Anchors

- `docs/harness/TASK_PACKET_SPEC.md`
- `docs/harness/VERIFICATION_EVIDENCE_SPEC.md`
- `docs/harness/REVIEW_LOOP_SPEC.md`
- `docs/harness/VISUAL_QUALITY_PROTOCOL.md`

## Scope

### In

- bizscope：单 C 菜单重构为 M 父目录 + 3 个 C（业务域列表、主机绑定、环境管理）；现有 BizScope CRUD 迁至 `/business/bizscope/scopes`。
- 新实体 BusinessEnvironment：新表 `biz_business_environment`（业务域子资源，`(business_scope_id, code)` 复合唯一、软删除）、AutoMigrate、`/business/bizscope/environments` 下 4 个 CRUD 端点、service 测试与 seed 测试。
- 主机绑定页复用 CMDB 主机列表 API 既有的 `businessScopeId` 过滤，零新后端端点。
- 服务目录：548 行 ServiceCatalog 三 Tab Monolith 拆为 M 父目录 + 3 个 C（`/business/service-catalog/{applications,services,instances}`），三页复用现有 application/service/instance list API 与现有 `business:service:list` 权限；删除 `ServiceCatalog.tsx`。
- 退役 overlay：清理 `/business/business-scope[/:id]`、`/business/service` 菜单行及 `system_role_menu` 关联，清理组件 key `business/service/ServiceCatalog`；清理先于模块 seed 执行。
- ArgoCD 去品牌化：仅改 i18n 显示值（中英，含 6 条权限文案），path / 权限 key / 组件目录 / routeName / API 全部保留。
- 前后端组件注册表、`business-overlay.json`、双语 i18n seed 与前端语言包、smoke spec 重写与新增 bizscope 套件、smoke README、`package.json` 脚本登记同步。
- pantheon-ops 侧 L2 治理：本 task packet、`plan.md` 与交付期 evidence/review。

### Out

- CI 能力建设；现有 CD/GitOps 行为保持不变，CI 半部分明确不做。
- cicd 的路径、权限 key、组件目录、API 路由、常量与类型标识符改名。
- 现有 `BizScope.Environment` 枚举字段与服务实例 `environment` 自由文本字段的引用改造。
- `e2e-tests/` 遗留手动套件（不在 CI；仅当文案断言被改名打破时做最小文本修复）。
- `BUSINESS_ARGOCD_MODULE_DESIGN.md` 等设计文档重命名。
- 任何新依赖、门禁弱化、告警 dismiss/suppress；生产数据与凭证。
- pantheon-base 的 `backend/`、`frontend/` 产品代码；本包与治理产物归 ops，不产生 foundation release。

## Structural Scope

- Affected Subgraph: bizscope/service/cicd seed → iam 组件注册表与菜单合同 → 前端 defineModule 路由/菜单 → smoke 与 overlay 契约
- Boundary Crossings: business/* 模块内部重构；system/iam 仅追加/移除组件白名单 key；任务、证据与实现均归 pantheon-ops；共享规范引用 pantheon-base
- Risk Nodes: 退役清理的 role_menu 关联删除、seed upsert 的同 path 类型覆写、ServiceCatalog 删除前的三向逻辑分流、双语 i18n key 对齐
- Graph Focus: 交付期在 pantheon-ops 做人工子图与敏感流审查（cycle/hub/call-depth/sensitive-flow）；base 无改动，本包不声明 manifest structuralScope

## Expected Files

### Create

- `docs/harness/tasks/2026-11-13-ops-business-menu-ia-refactor.task.md`（pantheon-ops）
- `.harness/tasks/2026-11-13-ops-business-menu-ia-refactor/manifest.json`（pantheon-ops）
- `.harness/tasks/2026-11-13-ops-business-menu-ia-refactor/plan.md`（pantheon-ops）
- `.harness/evidence/2026-11-13-ops-business-menu-ia-refactor/{commands.json,summary.md,review.md}`（交付期，pantheon-ops）
- `backend/modules/business/bizscope/environment_{model,dto,service,handler}.go` 与 `environment_service_test.go`、`seeds_test.go`（pantheon-ops）
- `backend/modules/business/service/seed_test.go`（pantheon-ops）
- `frontend/src/modules/business/bizscope/BizScopeHostBindingList.tsx`
- `frontend/src/modules/business/bizscope/environment/{EnvironmentList.tsx,api.ts}`
- `frontend/src/modules/business/service/ApplicationList.tsx`
- `frontend/tests/smoke/business/bizscope/bizscope-pages.spec.ts`

### Modify

- `backend/modules/business/bizscope/module.go`、`bizscope_service.go`
- `backend/modules/business/service/seed.go`
- `backend/modules/business/cicd/seed.go`（仅 i18n value）
- `backend/modules/business/retired_modules_overlay.go`
- `backend/modules/system/iam/menu/business_overlay_component_registry.go`
- `frontend/src/modules/business/bizscope/index.ts`、`BizScopeList.tsx`
- `frontend/src/modules/business/service/index.ts`、`ServiceList.tsx`
- `frontend/src/modules/business/serviceinstance/ServiceInstanceList.tsx`
- `frontend/src/core/router/businessOverlayComponentRegistry.ts`
- `frontend/src/i18n/resources/business/{zh-CN,en-US}.ts`
- `frontend/tests/smoke/business/service/service-pages.spec.ts`、`frontend/tests/smoke/README.md`、`frontend/package.json`
- `business-overlay.json`

### Do Not Touch

- `backend/modules/business/service/handler.go`（API 路由与合同不变）
- `backend/modules/business/bizscope/bizscope_model.go`（`BizScope.Environment` 枚举保留）
- cicd 路径、常量、API 路由与权限 key（只动 i18n value）
- `frontend/src/modules/business/cicd/argocd/ArgoCDList.tsx`（JSX 无硬编码品牌文案）
- `e2e-tests/`（除被改名打破的断言文本）
- `foundation-release.lock.json`、`.github/workflows/` 门禁定义与一切质量阈值

## Assumptions and Open Questions

- 需求两轮澄清已锁定：三菜单形态、显示文案改名深度（仅 i18n value）、旧路径退役策略、环境实体为业务域子资源。无未决问题。
- 假设现有 `testmysql` 测试夹具与本地空库启动流程在执行环境可用；Windows 跑 `-race` 需先挂 mingw64 到 PATH（见 plan.md 验证方案）。
- 计划命令不是验证结果；实际退出码与截图以交付期 `commands.json` 与 browserEvidence 为准。

## Minimum Viable Approach

最小复杂度阶梯：主机绑定零新端点（复用 CMDB 过滤）、服务拆分零新 API 零新权限 key、环境管理复用 bizscope 既有 model/service/handler 分层与 i18n 错误键风格、建表复用 AutoMigrate。唯一新增 CRUD 实体是环境管理；不引入新依赖、不做编号 SQL 迁移、不抽象通用菜单框架。

## Implementation Notes

- **Hard Stop**：`backend/` 与 `frontend/src/` 全部由 Codex executor 在 pantheon-ops 内实现（`codex exec ... -C pantheon-ops`），按 plan.md 的 5 提交序列（①环境实体后端+测试 ②seed 重构+退役+seed 测试 ③前端页面/模块/注册表 ④smoke 与契约文件 ⑤i18n 去品牌化）；Claude 协调者只拥有本治理包与 evidence/review。
- 路径前缀约定：expectedFiles 中 `backend/`、`frontend/` 路径相对 pantheon-ops 根；`.harness/`、`docs/harness/` 路径同样相对 pantheon-ops 根；任务、证据与产品改动都在 ops，本任务不需要 base 发行版。
- seed upsert 按 path 匹配并全字段覆写（含 type/component/parent_id），同 path 的单 C → M+3C 转换不留孤儿；只有换 path 的两个旧路径需要 overlay 退役，且退役清理先于 seed 执行。
- 环境表软删行仍占用 `(business_scope_id, code)` 唯一索引，与现有 BizScope 限制一致，代码注释标明。
- 服务目录三个 C 的 pagePerm 统一复用 `business:service:list`，不新增服务权限 key；bizscope 侧共新增 6 个权限 key（主机查看 1 + 环境 5）。
- 自定义非 admin 角色在升级后需重新授权新菜单（退役清理删除其 `system_role_menu` 关联），必须写入发布说明。
- 开工时把 manifest 状态翻为 in-progress、创建 evidence 目录、`linkage.reviewFile` 从 `none` 改为 canonical 路径，并在 `verificationPlan` 增加 `visualEvidence`（1440px/390px 两视口 + 六条 C 路由），让 `check:harness-visual` 跟踪交付。

## Method Readiness

- Consumer-Specific Controls: check-menu-contract 五字段双向比对、check-smoke-coverage-contract、business overlay 四命令、空库双启动幂等检查。
- Required Sensors: 后端 `go test -race`（含新 environment/seeds 测试）、前端 prebuild 机械门禁组、两个 business smoke 套件实跑、Playwright 双视口截图。
- Required Evidence: 真实命令/退出码/cwd 的 commands.json、测试与 overlay 回执、browserEvidence 截图、独立 review.md。
- Minimal Complexity Rung: reuse（API、权限、组件模式、AutoMigrate 全部复用）。
- Ratchet Decision: sensor-added（新 smoke 目录与 seed 测试把菜单 IA 漂移纳入持续拦截）。
- Deferred Code Issues: `BizScope.Environment` 枚举与实例自由文本 environment 的统一治理显式留待后续任务，本任务不 infer 扩边。

## Execution Roles

- Implementer Posture: Codex executor 拥有 pantheon-ops 全部产品代码与测试改动，按 5 提交序列交付；Claude 不触碰 Hard Stop 边界。
- Reviewer Posture: Claude 协调者对照 plan.md 审 diff、跑机械门禁并收 evidence；独立非作者产出 review.md machine-readable verdict；维护者执行最终视觉/功能人工门禁。

## Verification Plan

- 后端：`go build ./...`；bizscope（含 environment + seeds）、service（含新 seed 测试）、business 根（retired overlay）目标包 `go test -race`。
- 前端：`tsc -b`、eslint、`npm run prebuild`（menu-contract / i18n 双向对齐 / smoke-coverage / overlay 四关为重点）。
- 烟测：`npm run test:smoke:business:service` 与新 `test:smoke:business:bizscope` 真实本地栈实跑。
- overlay 四命令：`npm run check:business-overlay`、`npm run check:shared-drift`、`npm run rebuild:from-base -- --target .tmp/business-overlay-rebuild`、`check:task-packet`。
- 空库连续启动两次：seed 幂等、无孤儿菜单行、退役 path 无菜单、admin 菜单树 M+3C / M+3C 结构正确。
- Impeccable：六个 C 页面与改名后的应用发布页全部产出 1440px + 390px Playwright 渲染证据，核查 Source Sans 3、CSS tokens、radius 4/6/8/12、无 Inter/radial-gradient。
- pantheon-ops 治理门禁：`check:task-packet-template`、`check:task-packet`、`check:docs-frontmatter`、`check:harness-docs`、`check:harness-inventory`、`check:harness-encoding`、`check:harness-evidence`、`check:harness-review`、`check:harness-runtime-evidence`、`check:harness-visual`、`check:harness-graph-review`、`check:harness-adoption`、`check:structure`、`check:duplication`。

## Success Criteria

- check-menu-contract 绿：每个前端 C 菜单存在同 path 后端 C seed 且五字段一致；titleKey 中英双语前后端均存在；组件 key 与权限 key 双向对齐。
- check-smoke-coverage-contract 绿；两个 business smoke 套件实跑通过、console 干净、无裸 i18n key。
- 后端构建与目标 race 测试通过，含新环境 CRUD 生命周期测试与两个 seed 测试。
- 前端 type-check/lint/prebuild 全绿；overlay 四命令通过且 rebuild 可复现。
- 空库双启动幂等；退役 path 与 ServiceCatalog 组件 key 彻底移除，无孤儿 role_menu。
- 1440px/390px 渲染证据覆盖全部新/改页面并满足 impeccable 视觉门禁。
- 发布说明记录自定义角色重新授权事项；独立 review.md 归档，维护者通过最终人工验收。

## Linkage

- Task ID: 2026-11-13-ops-business-menu-ia-refactor
- Task Manifest: .harness/tasks/2026-11-13-ops-business-menu-ia-refactor/manifest.json
- OpenSpec Change: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-13-ops-business-menu-ia-refactor/plan.md
- Evidence Directory: .harness/evidence/2026-11-13-ops-business-menu-ia-refactor/
- Review File: none

## Evidence Required

- `commands.json`：按 VERIFICATION_EVIDENCE_SPEC 记录每条命令、cwd 与 passed/failed/not-run 状态，含 overlay 与 smoke 回执。
- 后端 environment service 测试、两个新 seed 测试与空库双启动幂等检查的输出。
- 六个 C 页面 + 应用发布改名页的 Playwright 截图与 browserEvidence（桌面 1440px、移动 390px，含 checkedStates 与 consoleErrors）。
- `summary.md` 验证读数与显式 knownGaps；`review.md` 含 machine-readable 独立 verdict。
- evidence 中不得出现凭证、密钥、生产数据或认证痕迹。

## Human Gates

- 维护者批准本 planned 任务包（In/Out/验收标准）后 Codex 方可开工；菜单形态、改名深度、环境数据模型与退役策略已由两轮澄清锁定。
- 维护者基于渲染证据做最终视觉/功能验收；不预授权任何门禁豁免、红灯放行或规则变更。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
- [ ] Menu contract and smoke coverage contracts green
- [ ] Impeccable visual evidence captured at 1440px and 390px
- [ ] Retirement and seed idempotency verified on an empty database
- [ ] Custom-role re-authorization release note recorded
