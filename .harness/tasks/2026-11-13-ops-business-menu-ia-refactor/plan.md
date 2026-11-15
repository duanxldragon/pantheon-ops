# 实施计划：pantheon-ops 业务菜单信息架构改造

- Task ID：`2026-11-13-ops-business-menu-ia-refactor`
- 目标仓库：pantheon-ops（本文件位于 pantheon-ops 治理包内；所有 `backend/`、`frontend/` 路径均相对 pantheon-ops 仓库根）
- 状态：planned（需求经两轮 AskUserQuestion 澄清锁定，等待维护者批准任务包后交 Codex 执行）

## Context（背景与已确认决策）

用户从功能角度指出四个 IA 问题，已通过两轮澄清锁定决策：

1. **业务域（bizscope）单页太单薄** → 改为 M 父目录 + **3 个二级菜单**：
   - ① **业务域列表**：现有 BizScope CRUD 原样迁移
   - ② **主机绑定**：跨业务域的主机绑定只读总览（复用 CMDB 主机列表接口已有的 `businessScopeId` 过滤，零新后端端点）
   - ③ **环境管理**：**新实体**，业务域子资源（用户已选定数据模型，见下）
2. **服务目录（service）Tab 改左侧二级菜单** → M 父目录 + 3 个 C：应用 / 服务 / 服务实例。三个独立 list 接口（`/business/service/applications|services|instances`）已现成，零 API 改动。
3. **ArgoCD 去品牌化**：菜单显示名改为 **「应用发布」/「Application Delivery」**，**仅改 i18n 显示文案**；内部路径 `/business/cicd/argocd`、权限 key `business:cicd:argocd:*`、组件目录、API 路由全部保留。CI 功能不做。
4. **测试同步关闭**：菜单/路由/smoke 同补丁改动，机械门禁强制。

**环境管理数据模型（用户已确认）**：新表 `biz_business_environment`，业务域子资源：
`id`、`business_scope_id`（必填、索引）、`code`（域内唯一，复合唯一索引）、`name`、`env_type`（dev/test/prod）、`status`（active/inactive）、`remark`、时间戳、软删除。
现有 `BizScope.Environment` 枚举字段与服务实例 `environment` 自由文本**本任务不动**。

**旧路由策略（用户已确认）**：service 用新前缀 `/business/service-catalog/{applications,services,instances}`，旧 `/business/service` 进 `retired_modules_overlay.go` 退役清单。

### 已核实的关键机械约束

- **seed upsert 按 path 匹配、全字段覆写（含 type/component/parent_id）**；F 权限行按 perms 匹配，改 ParentKey 可原地换父，不留孤儿。换 path 的旧行只能靠退役清理。
- **退役清理时机**：base `business.go` preflight 先执行 `cleanupRetiredBusinessModules`，之后各模块才 seed。MenuPaths-only 条目只删菜单行 + `system_role_menu` 关联，不碰权限/i18n/业务表。
- **check-menu-contract 门禁**：每个前端 menu（C）必须有后端同 path 的 C seed，且 titleKey/module/routeName/pagePermission/component 五字段完全一致；titleKey 必须同时存在于前端语言包和后端 i18n seed（中英）；前端 route 的 componentKey 必须在前后端两个组件注册表中**双向对齐**；权限 key 前后端双向对齐。M 类型 seed 不参与前端 manifest 比对（cmdb 模板证实：前端 menus 只声明 C 子项，M 仅存在于后端 seed）。
- **check-smoke-coverage-contract**：每个 spec 必须被某个 `test:smoke:*` 脚本引用，且脚本名与 spec 路径都要在 `tests/smoke/README.md` 以反引号形式登记。
- **system-page-admission 只管 /system/\***，business 页面不在准入清单。
- **建表走 GORM AutoMigrate**（bizscope 先例），不需要编号 SQL 迁移文件。
- **Hard Stop**：pantheon-ops 中 Claude 只做计划/审查/文档，`backend/` 与 `frontend/src/` 实现全部交 Codex（`codex exec ... -C pantheon-ops`）。本任务属 L2，需 task packet + evidence + review + human gate。

---

## 改动一：bizscope → M + 3C（含新环境实体）

### 后端（`backend/modules/business/bizscope/`，Codex）

1. **新文件 `environment_model.go`**：`BusinessEnvironment` 结构体，表名 `biz_business_environment`，字段按上述模型；复合唯一索引 `idx_scope_env_code (business_scope_id, code)`；软删除约定与 `BizScope` 一致（沿用代码库已接受的"软删行仍占唯一索引"限制，加一行注释说明）。
2. **新文件 `environment_dto.go` / `environment_service.go` / `environment_handler.go`**：
   - Service：List（过滤 scopeId/envType/status/keyword + 分页）、Create、Update、Delete；校验 code 域内唯一、scope 存在、envType ∈ dev/test/prod、status ∈ active/inactive；错误走 i18n key 风格（参照 bizscope 现有 `*.required/*_invalid` 错误键）。
   - Handler 挂在现有 `DataScopedGroup(r, "/business/bizscope")` 下：`GET /environments`、`POST /environments`、`PUT /environments/:id`、`DELETE /environments/:id`。
3. **`bizscope_service.go` Migrate()** 扩展为 `AutoMigrate(&BizScope{}, &BusinessEnvironment{})`。
4. **`module.go` seeds 重构**（参照 `cmdb/module.go` 的 topLevelMenuSeeds + 子项模式）：
   - M：`Key:"bizscope", Path:"/business/bizscope", Type:"M", Icon:"apps", Sort:2`，TitleKey 复用 `operations.bizscope.menu`（值"业务域"）。
   - C1 列表：`Path:"/business/bizscope/scopes"`，component `business/bizscope/BizScopeList`，pagePerm `business:bizscope:list`，routeName 保留 `bizscope-list`，**新 titleKey** `operations.bizscope.scope.menu`（"业务域列表"/"Scope List"）；现有 6 个 F 权限 ParentKey 改为新 C 的 key。
   - C2 主机绑定：`Path:"/business/bizscope/host-bindings"`，component `business/bizscope/BizScopeHostBindingList`，pagePerm 新权限 `business:bizscope:host:view`，titleKey `operations.bizscope.hostBinding.menu`（"主机绑定"/"Host Bindings"），只读，配 1 个同 key 的 F seed。
   - C3 环境管理：`Path:"/business/bizscope/environments"`，component `business/bizscope/EnvironmentList`，pagePerm `business:bizscope:environment:list`，titleKey `operations.bizscope.environment.menu`（"环境管理"/"Environments"）；F 种子 `environment:list/view/create/update/delete`（list 作页面权限，其余 4 个动作）。
5. **i18nSeeds 增补**（中英，group=menu/permission/page）：3 个新菜单 key、5 个环境权限 + 1 个主机查看权限的 permission 文案、环境实体的 field/placeholder/error/empty 等页面 key。
6. **退役清单**（`backend/modules/business/retired_modules_overlay.go`）：新增一个 `retiredModuleSpec{MenuPaths: ["/business/business-scope", "/business/business-scope/:id"]}`（与现有 `/operations/*` 条目同风格；`:id` 行无对应菜单时查询无副作用，保持惯例）。
7. **测试**：
   - 新 `environment_service_test.go`：CRUD 生命周期、域内 code 重复拒绝、跨域 code 可重复、scope 不存在、envType/status 非法（testmysql 风格，参照 `bizscope_service_test.go`）。
   - 新 `seeds_test.go`（bizscope 包目前无 seed 测试）：断言 M 1 行 + C 3 行的 path/type/page_perm、F 权限行数、admin 角色绑定数（参照 `cmdb/seeds_test.go`）。

### 前端（`frontend/src/modules/business/bizscope/`，Codex）

1. **`index.ts` defineModule 重构**：routes 改为 4 条（scopes 列表、scopes/:id 详情带 activeMenu、host-bindings、environments）；menus 只声明 3 个 C（M 不声明，cmdb 模板如此）；permissions 追加 6 个新 key；i18nNamespaces 不变。
2. **`BizScopeList.tsx`**：详情跳转改 `/business/bizscope/scopes/${id}`，其余 CRUD 逻辑不动。
3. **新文件 `BizScopeHostBindingList.tsx`**：只读列表，复用 `../cmdb/host/api` 的 `getHostList`（传 `businessScopeId`）+ bizscope options 做过滤；列取 hostname/IP/业务域名/状态子集；行可跳转 `/business/cmdb/host/:id`；统一 `PageContainer + GovernanceSummaryBar + SearchToolbar + AppTable + list-page.css` 模式。
4. **新目录 `environment/`**：`EnvironmentList.tsx`（CRUD 表格 + FormModal，交互骨架复制 BizScopeList 的弹窗模式）+ `api.ts`（对接 4 个新端点）。
5. **i18n 资源**：`src/i18n/resources/business/zh-CN.ts`、`en-US.ts` 增补全部新 key（菜单/页面/字段/占位/错误/权限名）——与后端 i18n seed 双写，menu-contract 强制。
6. **组件注册表双侧对齐**：
   - 前端 `src/core/router/businessOverlayComponentRegistry.ts` 加 `business/bizscope/BizScopeHostBindingList`、`business/bizscope/EnvironmentList`。
   - 后端 `backend/modules/system/iam/menu/business_overlay_component_registry.go` 加同两个 key。
7. **`business-overlay.json`**：components 数组加 2 条；新 smoke 目录 `frontend/tests/smoke/business/bizscope` 加入 businessPaths。

---

## 改动二：service → M + 3C（拆 Tab，零 API 改动）

### 后端（`backend/modules/business/service/seed.go`，Codex）

1. 单 C seed 替换为：
   - M：`Key:"service-catalog", Path:"/business/service-catalog", Type:"M", Icon:"apps", Sort:2`，TitleKey 复用 `operations.service.menu`（"服务目录"）。
   - C 应用：`/business/service-catalog/applications`，component `business/service/ApplicationList`，routeName `service-application-list`，新 titleKey `operations.service.application.menu`（"应用"/"Applications"）。
   - C 服务：`/business/service-catalog/services`，component `business/service/ServiceList`，routeName `service-service-list`，titleKey `operations.service.service.menu`（"服务"/"Services"）。
   - C 实例：`/business/service-catalog/instances`，component `business/serviceinstance/ServiceInstanceList`，routeName `service-instance-list`，titleKey `operations.service.instance.menu`（"服务实例"/"Service Instances"）。
   - **三个 C 的 pagePerm 统一复用现有 `business:service:list`**（不新增权限 key；门禁只查 path/routeName/权限 key 唯一，不查 pagePerm 唯一）。现有 4 个 F 权限 seed 保留，ParentKey 挂到 M 的 key。
2. seedI18n 增补 3 个新菜单 key（中英）；权限文案不动。
3. **退役清单**：同一 overlay 文件加 `retiredModuleSpec{MenuPaths: ["/business/service"], ComponentKeys: ["business/service/ServiceCatalog"]}`。
4. **新 `seed_test.go`**：断言 M+3C、admin 绑定、旧 path 无菜单行。
5. `handler.go` 的 API 路由（`/business/service/applications|services|instances`）**完全不动**。

### 前端（Codex）

1. **`service/index.ts`**：3 routes + 3 menus，permissions 保持现有 5 个。
2. **新建 `ApplicationList.tsx`**：从 ServiceCatalog 抽出应用 Tab 的表格 + 应用 FormModal + 删除逻辑（api.ts 的 application 系列函数已现成）。
3. **`ServiceList.tsx`（已存在，提升挂载）**：删除内嵌的实例卡片第二张 Card（与实例页重复），GovernanceSummaryBar 标题换新子菜单 key，保留筛选/分页服务表 + 服务 FormModal。
4. **`serviceinstance/ServiceInstanceList.tsx`（已存在，补完工）**：补 GovernanceSummaryBar、标准分页（替换硬编码 pageSize=100）、实例创建 FormModal（从 ServiceCatalog 迁入）。
5. **删除 `ServiceCatalog.tsx`**（548 行 Monolith，逻辑已三向分流；生产引用仅 seed/index.ts/两个注册表/overlay.json 五处，已核实）。
6. **注册表双侧**：移除 `business/service/ServiceCatalog`，加 `business/service/ApplicationList`、`business/service/ServiceList`（与 k8s 的 `business/k8s/service/ServiceList` 不同 key，无冲突）、`business/serviceinstance/ServiceInstanceList`；`business-overlay.json` components 同步。
7. i18n：优先复用现有 `business.service.catalog.*` Tab、hero、field key；仅新增 3 个菜单 key（中英双写 + 后端 seed）。

---

## 改动三：ArgoCD → 应用发布（仅文案）

### 后端（`backend/modules/business/cicd/seed.go`，Codex）

- `seedArgocdI18n`：菜单值 zh `"ArgoCD 应用发布"`→`"应用发布"`；en `"ArgoCD GitOps"`→`"Application Delivery"`。
- `SeedPerms`：6 条权限文案去除 ArgoCD 字样（如"查看 ArgoCD 应用"→"查看应用发布"，en 同步）。
- 常量名、path、routeName、component key、pagePerm、API 一律不动。

### 前端（Codex）

- `resources/business/zh-CN.ts` / `en-US.ts`：**只改 value 不改 key**。菜单 key（zh 1176/en 1171 附近）、hero.eyebrow、hero.title 中的 ArgoCD 品牌词移除；全量 grep 两个语言文件 value 中的 `ArgoCD` 残留并清理。
- `ArgoCDList.tsx` 已核实无 JSX 硬编码品牌文案（grep 命中均为类型/函数标识符），无需改组件。
- 若 generated i18n 资源有副本，按仓库现有 i18n 生成脚本重新生成（`check:i18n-generated-scope` 会拦）。
- smoke 中凡是断言可见文案 "ArgoCD" 的地方改 "应用发布"（实现时先 grep `tests/` 定位）。

---

## 改动四：测试、契约与治理同步关闭

1. **重写 `frontend/tests/smoke/business/service/service-pages.spec.ts`**：
   - 保留 API 造数链路（scope/host/bind/application/service/instance）与 finally 清理；
   - 旧的单页 Tab 断言改为依次访问三个新路由，各自断言 governance bar 文案、token 行内容（`token-application`/`token-service`/`Host #id`）、无裸露 i18n key、console 无 error、桌面截图；
   - 手机视口测试改访问新路由，保留无横向滚动断言；
   - 增加侧边栏菜单断言（参照 `observability-menu.spec.ts`：父菜单展开 → 3 个子项点击 → URL 正则 + 404 监听）。
2. **新建 `frontend/tests/smoke/business/bizscope/bizscope-pages.spec.ts`**：菜单父项展开 3 子项；业务域 CRUD 回归；主机绑定页渲染并验证过滤；环境 CRUD UI 生命周期；无裸 key/console 干净/桌面+移动截图。
3. **`package.json`**：新增 `test:smoke:business:bizscope` 并挂进 `test:smoke:business` 链（service 脚本名不变）。
4. **`tests/smoke/README.md`**：按 coverage-contract 要求反引号登记新脚本名与两个 spec 的相对路径，更新 service 行。
5. **L2 task packet**：本包（`.harness/tasks/` + `docs/harness/tasks/`）承载 In/Out/验收标准；evidence 目录收纳全部验证记录与截图；PR 按 docs-governance 模板写 body。

## 验证方案（Codex 执行，Claude 审查 evidence）

- 后端：`go build ./...`；目标包 `go test -race`（Windows 先 `$env:PATH='D:\msys64\mingw64\bin;' + $env:PATH; $env:CGO_ENABLED='1'`），至少覆盖 bizscope（含新 environment 测试 + seeds 测试）、service（含新 seed 测试）、business 根（retired overlay）。
- 前端：`tsc -b`、eslint、`npm run prebuild`（含全部机械门禁，menu-contract/i18n/smoke-coverage/overlay 四关是本任务重点）。
- 烟测：`test:smoke:business:service` 与新 `test:smoke:business:bizscope` 实跑通过。
- overlay 四命令：`npm run check:business-overlay`、`npm run check:shared-drift`、`npm run rebuild:from-base -- --target .tmp/business-overlay-rebuild`、`check:task-packet`。
- **Impeccable 视觉证据**：每个新/改页面 Playwright 桌面 1440 + 移动 390 截图入 evidence；重点检查 Source Sans 3、CSS tokens、radius 4/6/8/12、无 Inter/radial-gradient。
- 本地空库启动两次，验证 seed 幂等（不报错、无孤儿菜单行、admin 菜单树结构正确、退役 path 无菜单行）。

## Out of scope（显式不做）

- CI 能力（用户明确 CI 先不做，只保留 CD/GitOps 现状）。
- argocd 的路径/权限 key/组件目录/API/类型标识符改名。
- `BizScope.Environment` 枚举字段、服务实例 environment 自由文本字段的引用改造。
- `e2e-tests/` 遗留手动套件（不在 CI；仅当其断言因文案失败时顺手改）。
- 设计文档文件名 `BUSINESS_ARGOCD_MODULE_DESIGN.md` 等文档重命名。
- 不引入任何新依赖。

## 风险与注意事项

- **自定义角色授权失效**：退役清理会删旧菜单行及 `system_role_menu` 关联，非 admin 自定义角色需在升级后重新授权新菜单——写入发布说明。
- **环境表软删与唯一索引**：与现有 BizScope 同限制（软删行仍占用 code 唯一约束），保持代码库一致约定，注释标明。
- **ServiceCatalog 拆分是最大前端 diff**：拆出的三个页面均小于原 548 行 Monolith，注意 Sonar S3776 认知复杂度阈值（该文件曾被压到 5）。
- **菜单 sort 冲突**：bizscope M 与 service M 同为 sort 2 是现状延续（同级不同 key），保持不动；若侧边栏排序异常再调。
- 建议 Codex 分 5 条可审查提交：①环境实体后端+测试 ②seed 重构+退役+seed 测试 ③前端页面/模块/注册表 ④smoke 与契约文件 ⑤i18n 文案去品牌化。

## Authorized execution correction — 2026-11-14

The maintainer request to complete planned Ops tasks authorizes implementation. The canonical startup path has AutoMigrate disabled, so environment schema uses paired business migration000038. Host Bindings cannot register CMDB API paths under a BizScope owner; the page requires an independently granted CMDB host-view permission. All six pages remain subject to final rendered/functional acceptance.
