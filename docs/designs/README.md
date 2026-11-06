---
title: 设计文档目录
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-11-04
---

# 设计文档目录

English version: [README.en.md](./README.en.md)

本文是 `docs/designs/` 的中文优先入口，用来快速定位 Pantheon Base 的主设计文档。

目标不是把所有设计稿平铺出来，而是先给出中文开发者最常用的阅读顺序和主题入口。

覆盖范围可反证：本目录共 **62 份中文主文档**（不含本索引、不含 `.en.md`），下面的索引逐份引用；新增设计文档时必须同步登记，死链由 `npm run check:harness-docs` 兜住。

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
- [运营工作台组件设计](./OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md)

## system/auth

- [Auth 模块拆分设计](./AUTH_MODULE_DESIGN.md)
- [安全中心设计](./SECURITY_CENTER_DESIGN.md)
- [认证 Provider 抽象设计](./AUTH_PROVIDER_ABSTRACTION.md)
- [安全策略深化路线图](./SECURITY_POLICY_ROADMAP.md)
- [SSO / OIDC 设计](./SSO_OIDC_DESIGN.md)
- [登录风控设计](./LOGIN_RISK_CONTROL_DESIGN.md)

## system/iam 与导航治理

- [权限工作台治理深化设计](./PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md)
- [数据权限 Hook 设计](./DATA_PERMISSION_HOOK.md)
- [权限点和菜单解耦实施指南](./PERMISSION_MENU_DECOUPLING.md)

## system/org：组织与租户

- [system/org 组织域设计](./SYSTEM_ORG_DESIGN.md)
- [多租户架构设计](./MULTI_TENANT_DESIGN.md) — 愿景文档，**不是 V1 承诺**；V1 的绑定范围以 [租户合同 V1](../contracts/TENANT_CONTRACT_V1.md) §1 为准，冲突时合同优先
- [单租户先行、租户就绪设计](./TENANT_READY_SINGLE_TENANT_DESIGN.md)
- [租户资源 scope 矩阵模板与全局唯一键登记](./TENANT_RESOURCE_SCOPE_MATRIX.md)

## system/config

- [字典与系统设置设计](./DICT_AND_SETTING_DESIGN.md)
- [system/config 扩展设计](./SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [i18n 模块设计](./I18N_MODULE_DESIGN.md)
- [上传与存储设计](./UPLOAD_AND_STORAGE_DESIGN.md)
- [业务字典接入指南](./BUSINESS_DICT_INTEGRATION_GUIDE.md)
- [用户偏好设计](./USER_PREFERENCE_DESIGN.md)

## platform.lowcode 工作域

- [动态模块治理设计](./DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [模块生成器设计](./GENERATOR_MODULE_DESIGN.md)
- [模块生成模板扩展设计](./MODULE_GENERATOR_EXTENSION.md)
- [运行时模块引擎演进约束](./RUNTIME_MODULE_ENGINE_EVOLUTION.md)
- [低代码生成器使用指南](./LOWCODE_GENERATOR_GUIDE.md)
- [业务模块建模评审清单](./BUSINESS_MODELING_REVIEW_CHECKLIST.md)

## 业务模块与模板

- [业务模块设计模板](./BUSINESS_MODULE_TEMPLATE.md)
- [业务资源列表页模板](./BUSINESS_RESOURCE_LIST_PATTERN.md)
- [业务生命周期详情页模板](./BUSINESS_LIFECYCLE_DETAIL_PATTERN.md)
- [审批流骨架设计](./APPROVAL_WORKFLOW_DESIGN.md)
- [工作流引擎选型](./WORKFLOW_ENGINE_SELECTION.md)

## 运维与观测中心（设计稿）

下列 6 份是中心类能力的设计稿；是否已实现以各文档自身 frontmatter 的 `status` 与「未实现声明」为准，本索引不代述状态。

- [监控告警中心设计](./ALERT_MONITORING_CENTER_DESIGN.md)
- [通知中心设计](./NOTICE_CENTER_DESIGN.md)
- [消息通知服务设计](./NOTIFICATION_SERVICE_DESIGN.md)
- [报表中心设计](./REPORT_CENTER_DESIGN.md)
- [调度中心设计](./SCHEDULER_CENTER_DESIGN.md)
- [任务中心设计](./TASK_CENTER_DESIGN.md)

## 基础规则与参考

- [仓库目录布局](./REPOSITORY_LAYOUT.md)
- [数据库设计规范与详细说明](./DATABASE.md)
- [错误码与多语言设计](./ERROR_CODE_AND_I18N.md)
- [代码质量与安全治理策略](./QUALITY_AND_SECURITY_STRATEGY.md)
- [全局异常处理设计](./GLOBAL_EXCEPTION_HANDLING.md)
- [Foundation Release Model](./FOUNDATION_RELEASE_MODEL.md)
- [P2 规模化能力路线图](./P2_SCALE_ROADMAP.md)
- [gstack Windows 使用清单](./GSTACK_WINDOWS_GUIDE.md)

## 说明

- 中文 `.md` 是中文主入口。
- 英文 companion 的真实覆盖：**62 份中文主文档中 57 份有 `.en.md`**，另有 5 份只有中文版本，本索引不声称它们存在——`LOGIN_RISK_CONTROL_DESIGN.md`、`MULTI_TENANT_DESIGN.md`、`OPERATIONAL_WORKBENCH_COMPONENTS_DESIGN.md`、`PERMISSION_MENU_DECOUPLING.md`、`RUNTIME_MODULE_ENGINE_EVOLUTION.md`。此前本节写作「当前目录下主题文档均已补齐 `.en.md` companion」，与目录实测不符，本次就地更正；补齐这 5 份英文副本登记为后续文档项，不在本包范围内。
- 中文主入口优先级保持不变，英文 companion 用于国际协作与后续扩展；英文入口索引见 [README.en.md](./README.en.md)。
