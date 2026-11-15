---
title: 权限工作台治理深化设计
doc_type: Design
layer: system/iam
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-11-12
---

# 权限工作台治理深化设计

English version: [PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.en.md](./PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.en.md)

本文定义 `system/iam` 权限工作台从“发现 + 导出 + 受控补齐”继续演进到“发现 + 整改 + 追踪”的下一阶段边界。

---

## 1. 当前状态

当前权限工作台已经具备：

- 角色维度盘点。
- 菜单授权统计。
- 页面权限、动作权限、未知权限识别。
- API 缺口识别。
- 工作台导出。
- 对推荐 API 缺口执行单角色受控补齐。

因此“权限工作台第二层治理”不再是从零实现，剩余重点是追踪整改过程和治理结果。

## 2. 非目标

本阶段不做：

- 前端任意提交 path/method 写 Casbin。
- 跨角色批量自动修复。
- 自动删除未知权限。
- 把权限工作台变成低代码权限编辑器。

## 3. 治理追踪模型

建议后续新增只读或轻写治理记录，用于沉淀整改历史。

建议记录字段：

- `id`
- `role_key`
- `issue_type`：`page-gap`、`api-gap`、`unknown-permission`
- `issue_key`
- `before_state`
- `after_state`
- `action`：`exported`、`remediated`、`ignored`
- `operator_id`
- `created_at`

是否落表应在实现前单独评估。当前可以先通过审计日志承载轻量追踪。

## 4. 追踪闭环

最小闭环：

1. 工作台发现缺口。
2. 用户导出或执行受控补齐。
3. 后端记录审计事件。
4. 再次计算工作台，缺口状态收敛。
5. 页面展示最近整改结果或提示“已收敛”。

### 4.1 默认浏览行为

- 首屏默认展示全部角色，保证权限工作台先满足“按角色浏览授权结果”的基本预期。
- 待整改角色仍然通过摘要指标、治理状态列与 `pending/all` 视图切换前置。
- 不允许因为默认视图过滤，导致 `clean` 角色在未显式筛选时完全不可见。

## 5. 权限与安全约束

- 查看工作台：`system:permission:list`。
- 导出工作台：`system:permission:export`。
- 执行补齐：建议独立权限 `system:permission:remediate`。
- 补齐动作必须经过二次验证。
- 补齐动作只能基于后端推荐映射，不能接受前端任意 path/method。

## 6. 验收

- 有 API 缺口的角色可以被识别。
- 执行受控补齐后，缺口在下一次工作台计算中收敛。
- 补齐动作生成审计记录。
- 无权限用户不能执行补齐。
- 前端不能绕过后端推荐映射提交任意策略。

## 7. Provider-owned business API bindings (source extension, unreleased)

`backend/pkg/permissionroutes` is the shared extension boundary. A business module registers `Binding{Permission, Routes}` with `Register("business:<domain>", bindings)` during startup, before role policy reconciliation and route serving. The module fails startup if registration returns an error. Registrations are trusted compiled provider declarations; no HTTP API accepts route mappings.

A declaration may bind only its own `business:<domain>:` permission prefix and `/api/v1/business/<domain>` paths. Methods are explicit HTTP methods. Wildcards, traversal, malformed Gin parameters and cross-domain paths are rejected. Batches are validated before atomic insertion. Each key's mapping is immutable; an identical replay is harmless and a conflicting replay fails. Snapshots and lookups return copies. The registry is symmetric with the other compiled registries: `Seal()` runs at the startup seal point in `backend/cmd/server/main.go` (after `registerAPIRoutes`, next to `contracts.SealBusinessTenantProviders()` and `auditpolicy.Seal()`), and every `Register` after that point fails closed with `ErrSealed` while `Snapshot`/`Lookup` keep working. Changes require a new process/release; dynamic removal and hot reload are not supported.

Registration creates no role grant. IAM merges provider mappings with required system routes for workbench coverage and normal role policy synchronization. Granting a registered key adds only its exact methods/routes; revoking it removes those desired API grants. Page permissions, DataScope, tenant policy and business resource checks still apply independently. Providers must include related search/preview/resolve routes explicitly in the appropriate action's mapping; keys do not automatically grant other domains or whole router groups.

Registry validation/concurrency/copy tests and role grant/revoke tests are linked from `.harness/evidence/2026-11-07-ops-formal-queue-foundation-gaps/`. Ops must consume an immutable foundation release before importing this package; the existing v0.18.6 consumer cannot yet register its business mappings.
