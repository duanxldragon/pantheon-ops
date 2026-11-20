---
title: Ops 业务导航修复
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-navigation-repair

## Goal

修复 pantheon-ops 整个业务导航：

- 收敛重复/404 的服务目录菜单：升级旧树 `id159 /business/service`（旧 id、无注册页面路由 → 404）与当前 `id1818 /business/service-catalog`（含子项 1819/1820/1821）合并为一个；**保留旧菜单 id** 与全部角色绑定（含 tenant_id），对连续 seed 幂等，只处理 `module=business.service` 拥有的旧菜单，永不删除同名自定义菜单。
- 让 ServiceCatalog 页内 Tabs 受控于 URL 并在切换时写回 URL（activeTab/onChange），使二级菜单/Tab 点击真正响应；深链/刷新/前进后退一致。
- 用后端 seed Sort 重排业务菜单，使全部业务项（cmdb、bizscope、service、deploy、filedist、k8s、argocd、observability）在系统治理之前形成稳定连续块；filedist 紧跟 deploy，argocd 紧邻 k8s/release。
- 给 ArgoCD 一个不含 K8s 发布语义的明确展示名，消除侧栏重复「应用发布」按钮冲突。
- 编写迁移回归（真实升级旧树 fixture：绑定保留含 tenant_id、重复收敛、连续两轮 seed、所有权边界）与有意义的 Playwright 导航回归；记录完整业务菜单顺序 + 服务 3 页/404 修复的桌面 1440x900 / 移动 390x844 截图与逐叶导航矩阵。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: custom
- Overlay: none
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
- Quality Profile: ui-runtime
- Portable Failure Class: nav-no-response
- Owner Layer: business

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `business-overlay.json`

## Scope

### In

- 业务菜单 seed / scope 注册、前端业务菜单/路由/页面/i18n、业务导航 smoke/回归、business-overlay 声明、本任务自身的 task/evidence 治理文件。

### Out

- `backend/modules/business/cmdb/host/**`（SSH 修复另行进行）
- 任何 lock 升级、commit、push、PR 或 release
- wholesale 删除菜单或仅本地 SQL 改写既有业务库数据
- 削弱 RBAC 权限树（每个 C 节点 pagePerm 数保持 1）
- 自动关闭 SSH、历史工作台或其它 C/E 业务包
- 用户 Claude 设置/凭据文件、浏览器 profile 或 localStorage/cookies

## Implementation Notes

- Minimal Complexity Rung: reuse。复用现有 `mergeLegacyServiceMenu` 收敛逻辑与 `testmysql` 隔离库；不得绕过门禁或豁免新 task。
- Claude 为本任务唯一执行者；Codex 只做派发/复现/独立验收。实现/验证/证据由 Claude 直接完成。

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-navigation-repair.task.md
- .harness/evidence/2026-11-16-ops-navigation-repair/summary.md
- .harness/evidence/2026-11-16-ops-navigation-repair/commands.json
- .harness/evidence/2026-11-16-ops-navigation-repair/NAVIGATION_MATRIX.md
- backend/modules/business/service/seed_migrate_test.go
- frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts

### Modify

- backend/modules/business/service/seed.go
- backend/modules/business/deploy/deploy_seed.go
- backend/modules/business/k8s/k8s_seed.go
- backend/modules/business/filedist/seed.go
- backend/modules/business/cicd/seed.go
- backend/modules/business/observability/seed.go
- frontend/src/modules/business/service/ServiceCatalog.tsx

### Do Not Touch

- backend/modules/business/cmdb/host/**

## Verification Plan

- 真实隔离 MySQL（`testmysql` helper / 定义独立测试库）跑 `go test -race -v ./modules/business/service/...` 迁移回归，不接受因缺 DSN 而 skip 的通过结果。
- `go vet`/`go build`；前端 `npx tsc -b && eslint . && npm run build`。
- `check-business-overlay`、`check:menu-contract`、`check:gorm-sql-literal`、`check:docs-governance`。
- Codex 用真实升级 DB 重启后端，跑 Playwright 导航回归（侧栏父/子、页内 tabs、深链刷新、前进后退、全部业务叶入口）+ 桌面/移动截图。

## Linkage

- Task ID: 2026-11-16-ops-navigation-repair
- Task Manifest: .harness/tasks/2026-11-16-ops-navigation-repair/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-navigation-repair/
- Review File: .harness/evidence/2026-11-16-ops-navigation-repair/review.md
- OpenSpec Change: none
- Plan References: .harness/tasks/2026-11-16-ops-navigation-repair/plan.md
- OMX Plan: .harness/tasks/2026-11-16-ops-navigation-repair/plan.md
- Superpowers Plan: none

## Evidence Required

- 服务 seed 迁移回归输出（真实 MySQL，非 skip）
- 前端 typecheck/lint/build 输出
- menu-contract + business-overlay + docs-governance 门禁输出
- Playwright 导航回归输出
- 真实渲染桌面 1440x900 / 移动 390x844 截图与逐叶导航矩阵

## Human Gates

范围已确认；最终功能/视觉验收由维护者判断，未取得前不宣称交付完成。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed

## Result — History Closeout 2026-11-20

Manifest disposition: completed

导航修复实施 + after-final 验收证据（41 张截图 + acceptance.json）随 #190/#195 落地 main；QA/安全门禁绿。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

导航修复实施 + after-final 验收证据（41 张截图 + acceptance.json）随 #190/#195 落地 main；QA/安全门禁绿。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
