# 2026-11-16-ops-business-menu-m3c

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
