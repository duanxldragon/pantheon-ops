# Pantheon Base 代码审查修复报告 (fix-report)

- **日期**: 2026-07-15
- **审查依据**: `docs/PANTHEON_BASE_CODE_REVIEW_CHECKLIST.md`
- **分支**: `main`（未提交，工作区变更待人工确认后提交）
- **变更规模**: 40 文件修改 + 4 新增文件，+590 / -150 行

---

## 一、整体结论

- **是否达到企业级生产可交付标准**: **有条件通过**。核心闭环（认证/权限/组织/i18n/低代码/上传/审计）功能完整，构建、测试、lint 全绿；但存在若干需要架构决策的遗留项（见第四节），建议在下一个版本迭代内消化。
- **整体评分**: 修复前约 7/10，修复后约 **8.5/10**。
- **基线验证（修复前后均通过）**: `go build ./...`、`go vet ./...`、`gofmt -l`（无输出）、`go test ./...`（全部通过）、`tsc --noEmit`（0 错误）、`eslint --max-warnings=0`（0 告警）。
- **专项问题统计（本轮发现并处理）**:
  - 安全: 9 项修复，2 项待人工
  - 国际化: 88 处硬编码修复 + 111 组缺失 key 补齐
  - 低代码/动态模块: 5 项修复
  - 文件上传: 3 项修复
  - 前后端一致性: 2 项修复（含权限点种子缺失）
  - UI 门禁: 4 处 token 违规修复
  - CI/CD 与 DB: 3 项修复，若干项待人工

---

## 二、已修复问题清单

### 🔴 安全（P0/P1）

1. **刷新令牌不轮换（可重放）** — `backend/modules/auth/login/login_handler.go:212`
   刷新成功后旧 refresh token 在剩余 TTL 内仍然有效，可被重放。已在签发新 token pair 后立即删除旧 refresh token（Redis），失败时记 Warn 日志。
2. **高危操作缺少二次验证** — `backend/modules/system/system_modules.go`、`backend/modules/auth/module.go`
   为以下路由补挂 `SecureActionMiddleware()`（前端已有 `X-Operation-Token` 自动重试机制，无需前端改动）：
   - `DELETE /system/user/:id`、`PUT /system/user/:id/reset-password`
   - `DELETE /system/role/:id`、`DELETE /system/menu/:id`
   - `DELETE /system/session/:id`（管理员强制下线）
3. **CSP 双重来源冲突** — `backend/internal/middleware/security_headers_middleware.go`
   `SecurityHeadersMiddleware` 与 `CSPMiddleware` 都设置 CSP 且策略不同（前者会覆盖后者的环境感知策略，开发环境 Vite HMR 被破坏、Google Fonts 被阻断）。已移除前者的 CSP 设置，CSP 由 `CSPMiddleware` 单一来源负责。
4. **CSRF 豁免用前缀匹配** — `backend/internal/middleware/csrf_middleware.go`
   `strings.HasPrefix` 豁免意味着 `/api/v1/auth/refresh-anything` 等未来路由会被静默豁免。改为精确路径匹配，并补上 legacy 双入口 `/api/v1/system/login|refresh`。
5. **CSV 公式注入（导出）** — `backend/pkg/impexp/csv.go`
   所有导出单元格以 `= + - @ \t \r` 开头的非数值内容统一前置 `'` 中和，防止 Excel 公式执行。
6. **导入无大小/行数限制** — `backend/pkg/impexp/csv.go`
   `ReadCSV` 增加 10MB 字节上限 + 5000 行上限（`import.error.too_many_rows`）。
7. **批量操作无数量上限** — `backend/pkg/common/batch.go`
   `BatchDelete` 增加 `MaxBatchIDs=1000` 上限，超限整体拒绝并返回 `request.batch.too_large`。
8. **上传文件内容不校验（伪装图片）** — `backend/pkg/upload/service.go`
   图片扩展名（jpg/jpeg/png/gif/webp）增加 magic-bytes 内容嗅探（`http.DetectContentType`），内容与扩展名不符即拒绝。测试夹具已同步为真实 magic bytes。
9. **上传文件访问端点缺少防嗅探** — `backend/modules/system/config/setting/setting_handler.go`
   `GET /system/upload/files/*filepath` 增加 `X-Content-Type-Options: nosniff`；非图片类型强制 `Content-Disposition: attachment`，防止伪装内容被浏览器内联渲染（存储型 XSS 通路）。

### 🟠 低代码 / 动态模块

10. **卸载无注册记录模块可越权清理** — `backend/modules/lowcode/dynamicmodule/dynamic_module_lifecycle.go`
    `UnregisterModule` 原逻辑在查无注册记录时继续执行菜单/权限/源码清理，可对任意模块名（含 system/*）操作。现在无记录直接返回 `module.not_found`；内置模块（`ModelTableName` 为空）拒绝卸载。
11. **嵌套模块权限清理模式不匹配** — 同上 + `dynamic_module_naming.go`
    `splitModuleKey` 返回 `a/b` 形式而权限键为 `scope:a:b:action`，原 `LIKE scope:a/b:%` 永不匹配，嵌套模块卸载后权限残留。新增 `modulePermissionPrefix()` 统一转换。
12. **卸载遗留 role_menu 孤儿行** — 同上
    删除模块菜单前先收集菜单 ID 并清理 `system_role_menu` 关联，卸载后无悬挂引用。
13. **源码删除缺 business 限定** — 同上
    `FinalizeUnregister` 的 purgeSource 路径增加 `scope != "business"` 拒绝（纵深防御，`RemoveGeneratedModuleSource` 内部校验之外再加一层）。
14. **生成器 preview/download 绕过环境守卫** — `backend/modules/lowcode/generator/module.go`
    `POST /lowcode/generator/preview-files`、`/download-source` 原本不受 `DynamicModuleEnvGuard` 约束，生产环境仍可生成/下载代码。已补挂守卫。
15. **生成器数据源 SSRF 面收窄** — `generator_datasource_service.go`
    私网地址（RFC1918）增加 `PANTHEON_GENERATOR_DATASOURCE_ALLOW_PRIVATE` 开关（默认允许以兼容内网部署，可显式设 `false` 禁用），并对应 i18n key `generator.datasource.host_private_disabled`。

### 🟠 错误处理（静默吞错）

16. **认证域设置读取失败静默回退** — `backend/modules/auth/login/login_runtime.go`
    `fetchSetting*FromDB` 三个函数 DB 错误时静默用默认值（DB 故障时安全策略悄然放宽），现记录 Warn 日志。
17. **登录限流计数更新失败静默忽略** — `backend/modules/auth/login/login_service.go`
    throttle 重置/更新两处 `_ =` 改为 Error 级日志（暴力破解计数失效必须留痕）。
18. **安全事件/登录日志/审计清理写失败静默** — `login_runtime.go`、`login_service.go`、`security_service.go`、`audit_service.go`
    共 5 处 `_ = s.db.Create(...)` / `Delete(...)` 补 Warn 日志。

### 🟡 i18n（P0 专项）

19. **88 处审计标题硬编码中文** — 12 个 handler 文件
    所有 `common.SetAuditMetadata(c, "中文标题", ...)` 替换为结构化 i18n key（如 `user.batch_delete.title`、`dict.type.create.title`、`module.register.title`），共 86 处 + 中间件 2 处（`request.body_too_large`、`request.too_many`）。
20. **111 组 i18n key 补齐 zh-CN + en-US** — `frontend/src/i18n/resources/*.ts`
    含：86 组新审计标题、后端已引用但资源缺失的 10 组（`menu.create.title` 等）、前端引用缺失的 13 组（`generator.codePreview.*`、`i18n.lifecycle.*`、wizard 布局键等）、中间件 2 组。
21. **`system:role:import` 权限点种子缺失** — `backend/modules/system/seed.go`
    角色导入路由存在且前端有按钮，但权限点种子未定义（非 admin 角色永远无法被授予该权限）。已补种子 + zh/en 文案。

### 🟡 UI 门禁（DESIGN.md §7）

22. **硬编码品牌色阴影** — `frontend/src/modules/system/components/shared/list-page.css:1521,1528`
    `rgba(22, 93, 255, …)`（Arco 原始蓝）替换为 `color-mix(in srgb, var(--brand-primary) …%, transparent)`，四主题切换正确跟随。
23. **硬编码白色底** — `frontend/src/modules/lowcode/generator/pages/ModuleWizard.css:32,37`
    `color-mix(... , #fff)` 替换为 `var(--surface-lift)` token，暗色/主题安全。
24. **上传/导入请求超时缺失** — `frontend/src/api/upload.ts`、`importExport.ts`
    multipart 请求补 30s timeout，避免网络挂起时 UI 永久 pending。

### 🟢 DB / CI / 工程

25. **缺失索引迁移** — `backend/pkg/database/migrations/000009_review_hardening_indexes.{up,down}.sql`
    补 `system_menu(parent_id)`、`system_dept(parent_id)`、`system_role_menu(menu_id)` 索引（树查询/卸载清理路径）。迁移经 embed FS 自动加载。
26. **Husky pre-commit 钩子缺失** — `.husky/pre-commit` + `.gitignore`
    `package.json` 声明了 husky+lint-staged 但 `.gitignore` 把整个 `.husky/` 忽略导致钩子对克隆者不生效。已改为只忽略 `.husky/_/`，提交 `npx lint-staged` 钩子。
27. **生成器 preview/download 审计元数据缺失** — `generator_handler.go`
    补 `SetAuditMetadata`，`check-audit-coverage.mjs` 对 pantheon-base 现在 **118 条写路由 0 findings**。

---

## 三、验证结果

| 检查 | 结果 |
|---|---|
| `go build ./...` | ✅ 通过 |
| `go vet ./...` | ✅ 通过 |
| `gofmt -l` | ✅ 无格式问题 |
| `go test ./...` | ✅ 全部通过（含 upload 内容嗅探新逻辑的夹具更新） |
| `tsc --noEmit` | ✅ 0 错误 |
| `eslint src --max-warnings=0` | ✅ 0 告警 |
| `check-audit-coverage.mjs` | ✅ pantheon-base 0 findings |
| `check-permission-contract.mjs` | ✅ pantheon-base 0 findings |
| `check-boundaries.mjs` | ✅ 无发现 |

UI 变更说明（impeccable 门禁）：本轮 4 处 CSS 修复均为「硬编码颜色 → 既有 token」的等值替换，不改变布局与组件结构；默认主题下渲染结果与修复前一致，收益体现在非默认主题/暗色下的正确性。未产出截图证据，原因：无浏览器渲染环境可用；建议在 CI 截图基线任务中回归确认四主题。

---

## 四、需人工介入的问题及建议

> **2026-07-15 更新**：原 Top 3 决策项已经用户确认并按最佳实践实施完毕，见第 28-30 条修复记录。剩余项如下：

按优先级排序：

1. **【P1-架构】`business/*` 模块边界静态门禁** — 目前边界靠 `check-boundaries.mjs`（报告模式）约束。建议将其提升为 CI 阻断门禁，并在 golangci-lint 增加 `depguard` 规则禁止 `modules/business` import `modules/system/**` 非 contracts 包。
2. **【P1-产品】MFA 默认关闭** — `auth.mfa_enabled` 默认 false，Checklist 要求高敏操作强制二次校验。当前二次验证走操作密码（SecureAction），MFA/TOTP 已实现但需显式开启。建议在部署基线文档中把生产环境 `mfa_enabled=1` 列为必选项。
3. **【P2-工程】测试覆盖率门禁** — CI 有测试但未设覆盖率阈值。建议 `ci.yml` 增加 `go test -coverprofile` + 阈值检查（核心模块 ≥80% 为 Checklist 目标，可从 60% 起步 ratchet）。
4. **【P2-工程】UI 截图基线** — `smoke-full.yml` 有 e2e，但无四主题截图对比门禁。本轮 CSS token 修复的多主题正确性建议纳入下一次截图基线更新。
5. **【P3-治理】`system_i18n` 无 `updated_by` 字段** — 语言包动态更新无操作者追溯，依赖操作日志关联。可在下次 schema 迭代补列。

### 已实施的决策项（原 Top 3）

28. **i18n 资源表唯一索引** — `backend/pkg/database/migrations/000010_i18n_locale_key_unique.{up,down}.sql`
    确认事实后发现：模型无 GORM 软删除（`deleted_at` 列是历史遗留，模型早已不含该字段），且新装库（system_init.sql）与运行时 Bootstrap（`ensureLocaleKeyUniqueIndex`）都已有 `(locale, key)` 唯一索引——缺口只在"升级库依赖应用启动顺序"。新迁移先按"最新写入胜出"语义去重，再幂等创建唯一索引（information_schema 探测 + PREPARE），使唯一性保证进入版本化 schema。
29. **会话吊销级联删除 refresh token** — `pkg/authtoken/token.go`、`modules/auth/session/session_service.go`、`modules/auth/security/security_service.go`
    新增 `pantheon:sessref:<sessionID>` 反向索引（与 refresh token 同 TTL，轮换时 Set 覆盖指向最新 token）。`RevokeSessionRefresh` 级联删除 token+索引，幂等。接入全部四条吊销路径：logout/管理员强制下线（`RevokeSession`）、用户自助下线（`RevokeOwnedSession`）、批量强制下线（`BatchRevokeSessions`）、改密踢出其他会话（`RevokeOtherSessionsForUser`，事务提交后收集 sessionID 级联）。Redis 删除失败仅记 Warn——DB 侧 `revoked_at` 已生效，refresh 路径仍被 session 状态校验兜底，属双保险而非单点。新增 2 个测试覆盖级联删除、幂等性与轮换后索引指向。
30. **生成器私网数据源默认拒绝** — `generator_datasource_service.go`、`docs/DEPLOYMENT_GUIDE.md`
    默认值翻转为拒绝 RFC1918 私网地址（SSRF 最小面原则），需 `PANTHEON_GENERATOR_DATASOURCE_ALLOW_PRIVATE=true` 显式放行；环境变量已写入部署指南可选配置表。测试同步更新：默认拒绝 + opt-in 放行两个用例。

### 建议 roadmap

- **v0.9.x（本轮变更提交）**: 提交本报告全部修复（含已实施的原 Top 3 决策项 #28-30）。
- **v1.0（里程碑）**: 项 1、2（边界门禁阻断化、MFA 生产基线）、项 3 覆盖率 ratchet 起步。
- **v1.1+**: 项 4、5 及 pantheon-ops 侧同类问题同步（audit-coverage 检查显示 ops 侧存在与本轮 #27 相同的 2 处告警）。

---

## 五、变更文件清单

**后端（29 文件 + 5 新增）**: `internal/middleware/{body_limit,csrf,rate_limit,security_headers}_middleware.go`、`modules/auth/{login/login_handler,login/login_runtime,login/login_service,module,security/security_service,session/session_service}.go`、`modules/lowcode/{dynamicmodule/dynamic_module_{handler,lifecycle,naming},generator/{generator_datasource_service,generator_handler,generator_service_test,module}}.go`、`modules/system/{audit/audit_service,config/dict/dict_handler,config/setting/setting_handler,i18n/i18n_handler,iam/{permission/permission_handler,role/{role_handler,role_service},user/{user_handler,user_helper,user_service}},org/{dept/dept_handler,post/post_handler},seed,system_modules}.go`、`pkg/{authtoken/{token,token_test},common/batch,impexp/csv,upload/service,upload/service_test}.go`；新增 `pkg/common/builtin.go`、`pkg/database/migrations/000009_review_hardening_indexes.{up,down}.sql`、`pkg/database/migrations/000010_i18n_locale_key_unique.{up,down}.sql`。

**前端（6 文件）**: `src/api/{importExport,upload}.ts`、`src/i18n/resources/{zh-CN,en-US}.ts`、`src/modules/lowcode/generator/pages/ModuleWizard.css`、`src/modules/system/components/shared/list-page.css`。

**文档与工程（3 文件 + 1 新增）**: `.gitignore`、`docs/DEPLOYMENT_GUIDE.md`、新增 `.husky/pre-commit`。
