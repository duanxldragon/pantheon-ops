# 业务开发工作流与 AI 协作指南

更新时间：2026-04-30

类型：Design
归属层：platform
状态：Active

## 0. 文档合同化开发流

从 2026-04-30 起，Pantheon 默认采用“合同先行”的文档治理方式。

这不是额外的文书动作，而是为了避免以下问题继续反复出现：

- 设计文档、评估稿、整改稿互相覆盖，却没有主依据；
- 新需求直接进设计或代码，没有先确认归属层和完成定义；
- 旧 dated 评估稿继续漂在主索引里，AI 和新人不知道该信哪份；
- 代码已经改了，但验收和文档挂不到同一个锚点上。

当前合同主干入口：

- `docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md`
- `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`
- `docs/contracts/CONTRACT_TEMPLATE.md`
- `docs/contracts/PLATFORM_CONTRACT.md`
- `docs/contracts/SYSTEM_AUTH_CONTRACT.md`
- `docs/contracts/SYSTEM_IAM_CONTRACT.md`
- `docs/contracts/SYSTEM_ORG_CONTRACT.md`
- `docs/contracts/SYSTEM_CONFIG_CONTRACT.md`

### 0.1 基本规则

后续新需求或新专题默认按以下关系推进：

```text
Contract
  -> Design
  -> Assessment
  -> Remediation
  -> Acceptance
```

约束如下：

1. 先判断需求属于 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config` 还是 `business/*`
2. 先确认是否已有对应 `Contract`
3. 没有合同锚点时，先补合同或补合同骨架，再继续设计
4. `Design / Assessment / Remediation / Acceptance` 文档都必须回指对应合同
5. 新的阶段评估稿如果没有明确 `类型 / 状态 / 关联合同`，不应进入主索引

### 0.2 文档类型规则

后续文档统一使用以下类型：

- `Contract`
- `Design`
- `Assessment`
- `Remediation`
- `Acceptance`
- `Archive`

后续新增或重写的主文档，至少补：

- `类型`
- `归属层`
- `状态`

规则以 `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md` 为准。

## 1. 业务功能开发全生命周期 (SOP)

### 第一阶段：数据模型 (Design Phase)
1.  先判断本次需求归属层，并确认是否已有对应 `Contract`。
2.  若无合同锚点，先补合同或合同骨架；若已有合同，先核对当前设计是否越界。
3.  编写 SQL DDL，以业务模块名作为表前缀（例如 `biz_order_`）。
4.  在 `database/` 下记录 DDL 脚本。

### 第二阶段：后端逻辑 (Backend Phase)
1.  在 `modules/business/` 下创建包（例如 `order`）。
2.  生成 `order_model.go`, `order_dto.go`, `order_repo.go` 模板。
3.  在 `order_service.go` 中编写核心业务，调用底座提供的公共能力（如 ID 生成、Context 取用户信息）。
4.  在对应模块装配文件中注册路由；后端统一通过 `pkg/contracts.BackendModule` 与模块装配文件显式注册，不再依赖“大一统 system.go”思维。

### 第三阶段：前端 UI (Frontend Phase)
1.  在 `src/modules/business/` 下创建模块目录。
2.  实现基于 Arco Design 的页面，通过 API 文件调用后端接口。
3.  导出 `ModuleConfig` manifest，并注册到 `src/core/router/modules.ts`；禁止把业务菜单和页面写死到 Layout。

### 第四阶段：测试 (Test Phase)
1.  **接口测试**: 使用 Postman 或 curl 验证 200/401/403 响应。
2.  **Auth 测试**: 验证 access token 过期后 refresh token 能自动轮换，注销后会话失效。
3.  **权限测试**: 验证多角色用户在任一角色命中 Casbin 策略时可访问接口；验证非管理员用户访问未授权接口返回 403。
4.  **菜单测试**: 验证 `scope=nav` 仅返回当前用户可见导航菜单，`scope=manage` 返回完整授权树；补充校验菜单祖先节点自动补齐逻辑。
5.  **接口筛选测试**: 验证用户列表、角色列表和菜单树的 query 参数筛选结果正确；补充校验用户/角色分页、排序参数与返回总数，以及菜单树排序参数的 sibling 顺序。
6.  **UI 测试**: 验证用户/角色/菜单列表、筛选、分页、排序、表单提交与按钮权限控制是否能正确触发 `system_log_oper`。
7.  **数据校验测试**: 验证用户名唯一、角色标识唯一、邮箱格式、角色必选、菜单路径唯一、管理员保护等约束能返回正确错误码。

### 第五阶段：平台层冒烟 (Smoke Phase)

> 默认约定：凡是“浏览器页面链路 / UI 冒烟 / 截图验收 / 交互巡检”，本地默认使用 **gstack browse / gstack Browser**。Playwright 仅作为 CI 自动化、API smoke 或明确要求的补充工具，不作为 Windows 本地人工验收默认方案。

1.  **先判边界**: 先确认本轮冒烟属于 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config` 还是 `business/*`，不要把系统域页面混成一个“大 system 列表”。
2.  **先保运行态**: 确认前端 `:5173` 与后端 `:8080` 在线，再开始页面巡检。
3.  **先 API 登录**: 优先调用 `POST /api/v1/auth/login` 获取 `accessToken` 与 `refreshToken`，不要把 UI 表单输入当作唯一登录方式。
4.  **优先单条链路**: 使用 gstack 内置 Chrome 的单条 `browse chain` 完成“打开登录页 → 注入登录态 → 跳转目标页 → 等待 → 采集证据”，避免 Windows 下多次调用导致上下文漂移。
5.  **固定平台层覆盖**: 默认至少覆盖 `/login`、`/dashboard`、`/auth/security`、`/system/login-log`、`/system/session`、`/system/profile`、`/system/user`、`/system/role`、`/system/menu`、`/system/permission`、`/system/dept`、`/system/post`、`/system/dict`、`/system/setting`、`/system/operation-log`。
6.  **每页固定采集项**: 至少采集最终 URL、console error、截图与 `snapshot -i` 输出。
7.  **先区分真故障与工具故障**: 如果出现 `No active page`、`spawn EPERM`、截图超时或上下文关闭，先判断是否为 gstack / Windows 运行特性，再决定是否上升为真实页面缺陷。
8.  **保留验收证据**: 统一输出 JSON 汇总、原始日志目录与截图目录，供 `docs/acceptances/ACCEPTANCE_CHECKLIST.md` 与阶段评审复用。

---

## 2. AI 协作 Prompt 精准指南

在让 AI 生成方案或代码前，默认先补充以下上下文：

- 当前归属层：`platform / system/auth / system/iam / system/org / system/config / business/*`
- 当前合同文档：`<doc path>`
- 当前文档类型：`Design / Assessment / Remediation / Acceptance`
- 当前状态：`Draft / Active / Superseded / Archived`

如果以上 4 项说不清，默认先不要让 AI 直接生成实现。

当你需要让 AI 生成代码时，请附带以下约束信息：

### 后端 Prompt 模板
> “请基于 business 目录下的垂直切片架构生成一个 [功能名] 模块。要求：
> 1. 模型表前缀为 biz_；
> 2. 使用 DTO 屏蔽敏感字段；
> 3. 返回值统一使用 common.Success 或 common.Fail；
> 4. 逻辑写在 service 中，repo 仅负责原生 DB 操作。”

### 前端 Prompt 模板
> “请在 src/modules/business 下生成一个 [页面名] 组件。要求：
> 1. 使用 Arco Design 官方组件库进行布局；
> 2. 所有文本显示均使用 i18next 的 t() 函数翻译；
> 3. 页面符合 Pantheon Base 的 Indigo 主题色规范。”

## 3. 测试与部署流程
1.  运行 `docker compose up -d` 启动 MySQL/Redis 环境。
2.  初始化 `database/system_init.sql`；脚本只写入角色、菜单、权限等底座数据，首个 `admin` 用户由后端迁移创建。
3.  设置 `PANTHEON_DSN`，可选设置 `PANTHEON_REDIS_ADDR` 与 `PANTHEON_REDIS_PASSWORD`。
4.  生产环境额外设置 `PANTHEON_INITIAL_ADMIN_PASSWORD`，长度不少于 12 位；开发环境未设置时默认创建 `admin / 123456`。
5.  启动后端服务：`go run ./backend/cmd/server`，监听 8080；若已执行初始化 SQL，`casbin_rule` 会已存在，服务启动时会继续做迁移校验与策略同步。
6.  启动前端工程：`cd frontend && npm run dev`，访问登录界面。
7.  提交前执行 `go test ./...`、`cd frontend && npm run lint`、`cd frontend && npm run build`、`cd frontend && npm audit --registry=https://registry.npmjs.org --audit-level=high --omit=dev`。

### 3.1 文档同步门槛

提交前至少检查：

1. 当前改动是否仍符合对应 `Contract`
2. 若改动触发边界变化，是否先更新了 `Contract`
3. 若新增设计文档，是否已回链对应合同
4. 若新增评估或整改文档，是否已标明 `类型 / 状态 / 关联合同`
5. 若旧文档已被覆盖，是否已删除、降级或标记为 `Superseded / Archived`

## 4. 冒烟执行 SOP（gstack / Windows）

以下 SOP 面向平台层与系统域页面，不替代业务模块专项测试。

默认规则：

- **页面链路默认使用 gstack**：所有本地浏览器验收、组织架构/权限交互验证、截图留证，默认走 `browse chain` 或 gstack Browser。
- **Playwright 不做本地默认入口**：只有在 CI、API smoke、跨浏览器矩阵或用户明确要求时，才运行 `npm run test:smoke:*`。
- **失败归因优先级**：gstack 页面巡检发现问题时先判断业务页面 / 接口 / 权限数据，再判断工具上下文；Playwright 缺浏览器、`spawn EPERM` 等不作为业务失败依据。

### 4.1 执行前检查

1.  确认当前验收层级：`platform`、`system/auth`、`system/iam`、`system/org`、`system/config` 或 `business/*`。
2.  确认前端开发服务监听 `5173`，后端服务监听 `8080`。
3.  确认默认管理员账号可用，或准备好本轮测试账号与权限组合。

### 4.2 推荐执行顺序

1.  调用登录接口获取 token。
2.  使用单条 `browse chain` 注入登录态。
3.  从登录页开始，先跑 `platform`，再按 `system/auth` → `system/iam` → `system/org` → `system/config` 扩散。
4.  页面异常时先复核接口返回，再判断是页面问题还是工具问题。
5.  收尾时输出最终 JSON、原始日志和截图目录。

### 4.3 推荐命令模式

```powershell
browse chain "goto http://127.0.0.1:5173/login | wait --networkidle 15000 | storage set pantheon_access_token <AT> | storage set pantheon_refresh_token <RT> | goto http://127.0.0.1:5173/dashboard | wait --networkidle 15000 | url | console --errors | screenshot out.png | snapshot -i"
```

### 4.4 平台层最小覆盖清单

- `platform`：`/dashboard`
- `system/auth`：`/login`、`/auth/security`、`/system/login-log`、`/system/session`
- `system/iam`：`/system/profile`、`/system/user`、`/system/user/1`、`/system/role`、`/system/menu`、`/system/permission`、`/system/operation-log`
- `system/org`：`/system/dept`、`/system/post`
- `system/config`：`/system/dict`、`/system/setting`

### 4.5 输出物要求

- JSON 汇总：记录页面、层级、状态、原始日志路径、截图路径
- 原始日志：保留每页的 `url`、`console --errors`、`snapshot -i` 输出
- 截图目录：用于阶段评审与回归比对

### 4.6 Windows 特别说明

- `browse.exe` 偶发需要提权运行，出现 `spawn EPERM` 时优先按提权处理。
- Windows 下分散的 browse 调用更容易漂移到空白页，优先使用单条 `browse chain`。
- 遇到 `No active page`、截图超时、上下文关闭时，先重跑整条链路，不要直接把它判成业务故障。

### 4.7 文档落点

- 本次真实样例可参考 `docs/archive/QA_SMOKE_REPORT_20260420.md`
- 平台层验收对照结论可参考 `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- Windows 使用细节可参考 `docs/designs/GSTACK_WINDOWS_GUIDE.md`

## 5. `platform` 壳层提交流程要求

本节仅适用于 `platform` 壳层改动，尤其是以下对象：

- `frontend/src/core/layout/index.tsx`
- `frontend/src/core/layout/index.css`
- 动态菜单渲染链路
- 菜单图标映射
- 顶部横版导航
- 左侧竖版导航
- 与导航直接相邻的品牌区、页签区、顶部栏布局

### 5.1 提交门槛

凡是触达上述对象的改动，无论是 PR、阶段记录、周报式交付说明还是 AI 生成的阶段结论，都必须附：

1. 一份双模式验收记录文档链接
2. 一份构建通过结论
3. 一份固定扫描结果摘要

不满足以上三项时：

- 不应标记为 `Target`
- 不应标记为“已收口”
- 只能视为“代码已修改，待验收”

### 5.2 验收文档要求

双模式验收记录必须基于：

- `docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`

首个基准样例为：

- `docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`
- PR 描述模板：`docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`
- PR checklist 片段：`docs/acceptances/PLATFORM_SHELL_PR_CHECKLIST_SNIPPET.md`

后续提交要求：

- PR 描述中必须附验收文档链接
- 阶段记录中必须附验收文档链接
- 若本轮存在 `Pending` 例外，必须同时附矩阵文档链接并说明挂账位置
- PR 描述与阶段记录正文建议直接复用 `docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`

### 5.3 固定摘要格式

后续壳层提交说明至少包含以下四行：

- `双模式验收文档：<doc link>`
- `构建结果：npm run build Passed / Failed`
- `扫描结果：旧右栏 / 原生浮层 / 静态 Modal API / 双模式链路`
- `矩阵状态：Target / Pending`

### 5.4 禁止事项

- 不允许只附截图、不附验收文档链接
- 不允许只写“已验证横竖版正常”而无记录文件
- 不允许出现“横版后补”但仍标记完成
- 不允许在壳层提交说明中省略 `Pending` 挂账
