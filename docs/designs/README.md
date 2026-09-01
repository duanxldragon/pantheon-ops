---
title: 设计文档目录
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-27
---

# 设计文档目录

English version: [README.en.md](./README.en.md)

本文是 `docs/designs/` 的中文优先入口，用来快速定位 Pantheon Base 的主设计文档。

目标不是把所有设计稿平铺出来，而是先给出中文开发者最常用的阅读顺序和主题入口。

## 推荐阅读顺序

1. [Pantheon Base 架构总览](./PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md)
2. [仓库目录布局](./REPOSITORY_LAYOUT.md)
3. [总体架构与后端规范](./BACKEND.md)
4. [前端架构与模块接入](./FRONTEND.md)
5. [前端 UI 详细规范](./FRONTEND_UI_SPEC.md)
6. [模块契约设计](./MODULE_CONTRACT.md)
7. [权限模型设计](./PERMISSION_MODEL.md)
8. [业务开发工作流与 AI 协作指南](./WORKFLOW.md)
9. [代码质量与安全治理策略](./QUALITY_AND_SECURITY_STRATEGY.md)

## 平台与前端

- [前端页面模板规范](./FRONTEND_PAGE_TEMPLATES.md)
- [前端组件规划](./FRONTEND_COMPONENT_PLAN.md)
- [导航信息架构深化设计](./NAVIGATION_IA_STRATEGY.md)
- [空 / 加载 / 错误状态规范](./EMPTY_LOADING_ERROR_STATES.md)
- [响应式断点与移动端适配](./MOBILE_RESPONSIVE_BREAKPOINTS.md)
- [主题 Token 参考表](./THEME_TOKENS_REFERENCE.md)
- [暗色模式设计](./DARK_MODE_DESIGN.md)
- [后台 UI 风格约束](./BACKOFFICE_STYLE_CONSTRAINTS.md)
- [无障碍设计规范](./ACCESSIBILITY.md)
- [平台仪表盘设计](./PLATFORM_DASHBOARD_DESIGN.md)

## system/auth

- [Auth 模块拆分设计](./AUTH_MODULE_DESIGN.md)
- [安全中心设计](./SECURITY_CENTER_DESIGN.md)
- [认证 Provider 抽象设计](./AUTH_PROVIDER_ABSTRACTION.md)
- [安全策略深化路线图](./SECURITY_POLICY_ROADMAP.md)
- [SSO / OIDC 设计](./SSO_OIDC_DESIGN.md)

## system/iam 与导航治理

- [权限工作台治理深化设计](./PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md)
- [数据权限 Hook 设计](./DATA_PERMISSION_HOOK.md)

## system/config

- [字典与系统设置设计](./DICT_AND_SETTING_DESIGN.md)
- [system/config 扩展设计](./SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [i18n 模块设计](./I18N_MODULE_DESIGN.md)
- [上传与存储设计](./UPLOAD_AND_STORAGE_DESIGN.md)
- [业务字典接入指南](./BUSINESS_DICT_INTEGRATION_GUIDE.md)

## platform.lowcode 工作域

- [动态模块治理设计](./DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [模块生成器设计](./GENERATOR_MODULE_DESIGN.md)
- [运行时模块引擎演进约束](./RUNTIME_MODULE_ENGINE_EVOLUTION.md)
- [低代码生成器使用指南](./LOWCODE_GENERATOR_GUIDE.md)
- [业务模块建模评审清单](./BUSINESS_MODELING_REVIEW_CHECKLIST.md)

## 业务模块与模板

- [业务模块设计模板](./BUSINESS_MODULE_TEMPLATE.md)
- [业务模块建模评审清单](./BUSINESS_MODELING_REVIEW_CHECKLIST.md)
- [单租户先行、租户就绪设计](./TENANT_READY_SINGLE_TENANT_DESIGN.md)
- [业务资源列表页模板](./BUSINESS_RESOURCE_LIST_PATTERN.md)
- [业务生命周期详情页模板](./BUSINESS_LIFECYCLE_DETAIL_PATTERN.md)
- [审批流骨架设计](./APPROVAL_WORKFLOW_DESIGN.md)

## 基础规则与参考

- [仓库目录布局](./REPOSITORY_LAYOUT.md)
- [数据库设计规范与详细说明](./DATABASE.md)
- [错误码与多语言设计](./ERROR_CODE_AND_I18N.md)
- [代码质量与安全治理策略](./QUALITY_AND_SECURITY_STRATEGY.md)
- [P2 规模化能力路线图](./P2_SCALE_ROADMAP.md)

## 说明

- 中文 `.md` 是中文主入口。
- 当前目录下主题文档均已补齐 `.en.md` companion，可在对应中文文档顶部直接进入英文版本。
- 中文主入口优先级保持不变，英文 companion 用于国际协作与后续扩展。
