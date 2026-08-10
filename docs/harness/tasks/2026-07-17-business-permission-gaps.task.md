目标仓库：pantheon-ops
层级：business/deploy + business/cmdb
任务模式：implement

先读：
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-ops/backend/pkg/contracts/permission_policies.go
- pantheon-ops/backend/modules/business/deploy/deploy_seed.go
- pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md §9
- pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md §6.3 / §7.3

背景（2026-07-17 文档 truthing 轮停点发现的三个代码缺口；已对
permission_policies.go 全量 business 路由映射与 handler 路由做过交叉比对，
确认缺口仅此三处）：
1. deploy template：permission_policies.go:153-168 已定义
   `business:deploy:template:list/create/update/delete` 四项的 API 路由映射，
   但 deploy_seed.go 只 seed 了 template 列表菜单（C 项），
   create/update/delete 三个 F 按钮权限没有 seed——角色授权页无法勾选，
   模板 CRUD 的写接口对非超管角色实际不可达。
2. cmdb label options：`GET /labels/options` 路由已实现
   （label_handler.go:21），但 `business:cmdb:label:list` 的路由映射
   （permission_policies.go:96-99）只覆盖 `GET /labels`，Casbin 会拒绝
   非超管角色访问 /labels/options。
3. deploy package detail：`GET /packages/:id` 路由已实现
   （deploy_handler.go:23），但 `business:deploy:package:list` 的路由映射
   （permission_policies.go:137-140）只覆盖 `GET /packages`，
   `/packages/:id` 无任何权限映射，非超管角色不可达。
   （对照：template:list 的映射同时覆盖 /templates 与 /templates/:id，
   package 应对齐同样口径。）

实现范围：
- deploy_seed.go 补三条 F seed：template create/update/delete，
  ParentKey 挂 `deploy-template-list`，TitleKey 走
  `business.deploy.template.permission.*`，同步补 i18n seed（zh-CN/en-US）
- permission_policies.go 的 `business:cmdb:label:list` case 增加
  `{Path: "/api/v1/business/cmdb/labels/options", Method: "GET"}`
- permission_policies.go 的 `business:deploy:package:list` case 增加
  `{Path: "/api/v1/business/deploy/packages/:id", Method: "GET"}`
- 同步补 seed/permission 相关测试（参照 cmdb/seeds_test.go 既有模式）

不处理：
- 新增权限点或改权限命名
- 菜单结构调整
- 文档改动（由 docs-truthing 任务收口）

同步要求：
- 仅本仓业务改动

验证方式：
- Backend: `go test ./backend/modules/business/deploy ./backend/modules/business/cmdb/... ./backend/pkg/contracts`
- 手动/测试验证：授予角色 label:list 后可访问 /labels/options；
  角色授权页出现 template 三个按钮权限
- Smoke: `npm run check:menu-contract`

停点：
- permission_policies.go 属于 pkg/contracts（deep 层）：只允许上述两处
  单行追加，若发现需要更大结构调整，停下报告
