---
title: Ops 业务菜单 M+3C
doc_type: Remediation
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-16
---

# Task Packet: 2026-11-16-ops-business-menu-m3c

## 目标

把 pantheon-ops 的业务导航按父级 + 子级（M+3C）菜单重组：

- **bizscope**：`业务域`（business-scope 列表）保留为父级；新增子级 `主机绑定`（HostBinding）与 `环境管理`（EnvironmentList/EnvironmentForm）。
- **service**：`服务目录` 父级 + 子级 `应用 / 服务 / 服务实例`，前缀改为 `/business/service-catalog/*`；ServiceCatalog 从路由深链接到对应 Tab，废弃扁平 `/business/service` 页面。
- 后端新增 `biz_business_environment` 迁移（dev/test/prod，scope+code 唯一），并暴露按业务域的环境 CRUD 路由 `/business/bizscope/:id/environments`。
- 同步菜单种子、权限、i18n 种子、business overlay 组件注册表，以及派生迁移链中的 `000040` 迁移副本。
- 关闭一个既有的 en-US fallback 缺口：补 `cmdb.host` / `deploy.package` 5 个缺失 bundle 键。

## 边界

- In：仅 `business/*` 模块、bizscope/service 菜单与页面、相应 i18n / 菜单 / 权限 / smoke / overlay 同步、派生迁移副本。
- Out：`pantheon-base` 共享路径改动、foundation release 升级、low-code/generator 模块行为、重生成已提交的权威 i18n fallback bundle。

## Why

- 已提交的 i18n fallback bundle（`frontend/src/i18n/resources/business/*.ts`）是 M+3C 键集的权威事实来源，且禁止重生成；新页面复用已提交键。
- 菜单层级由后端 `system_menu.parent_id` 表达；前端菜单是扁平的 1:1 菜单↔路由契约，故子页面以子路径路由 + Type-C seed 呈现。

## 验证

本地全绿：`go build ./...`、`go test ./modules/business/bizscope/... ./modules/business/service/... ./modules/system/iam/menu/...`、`check-business-overlay`（53 声明路径）、`check:menu-contract`（45 菜单/55 路由/250 权限/54 组件键）、`audit:i18n-locales`（各语言缺失=0）、`tsc`、`eslint`、`npm run build`、`check:gorm-sql-literal`、`check:generated-modules`。

## 关联

- [[ops-business-menu-ia-refactor-2026-11-13]]、[[ops_main_committed_push_blocked_2026_11_15]]、[[feedback_automate_dont_ask]]

## Goal

原目标与实现叙述见上文；本次只整理状态与治理字段，不重做菜单实现。实现已合并，运行和视觉验收仍开放。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
- Quality Profile: ui-runtime

## Contract Anchors

- docs/PROJECT_INHERITANCE.md
- `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`
- `business-overlay.json`

## Scope

### In

- 原范围见上文。整理合法 primaryLayer/status，C05 承接当前候选环境管理、菜单深链接与角色运行验收。

### Out

- 不修改既有实现或共享规范，不用历史编译证据宣称新版视觉验收。

## Expected Files

### Create

- none

### Modify

- docs/harness/tasks/2026-11-16-ops-business-menu-m3c.task.md
- .harness/tasks/2026-11-16-ops-business-menu-m3c/manifest.json

### Do Not Touch

- 本任务既有 .harness/evidence/2026-11-16-ops-business-menu-m3c/ 原始证据
- foundation-release.lock.json

## Implementation Notes

Minimal Complexity Rung: reuse。保留历史原文和证据，状态为 in-progress；C05 完成后再补本任务验收与归档。

## Verification Plan

- 当前候选环境 CRUD、两域/受限角色、菜单/Tab 深链接通过业务浏览器 smoke。
- 桌面/窄屏与 loading/empty/error/forbidden 状态截图；未执行写 not-run。

## Linkage

- Task ID: 2026-11-16-ops-business-menu-m3c
- Task Manifest: .harness/tasks/2026-11-16-ops-business-menu-m3c/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-business-menu-m3c/
- Review File: .harness/evidence/2026-11-16-ops-business-menu-m3c/review.md
- OpenSpec Change: none
- Plan References: .harness/tasks/2026-11-16-ops-pilot-delivery/plan.md
- OMX Plan: .harness/tasks/2026-11-16-ops-pilot-delivery/plan.md
- Superpowers Plan: none

## Evidence Required

- 保留原 commands/summary/review；新增候选运行证据由 C05 链回。

## Human Gates

范围已确认；最终功能/视觉验收由维护者判断，未取得前不宣称交付完成。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
