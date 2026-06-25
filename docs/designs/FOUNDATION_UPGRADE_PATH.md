# Foundation Release Upgrade Path

## base-v0.8.4 → base-v0.8.5

English version: [FOUNDATION_UPGRADE_PATH.en.md](./FOUNDATION_UPGRADE_PATH.en.md)

## 基本信息

| 字段 | 值 |
|---|---|
| 从 | `base-v0.8.4`（`412a465e7c3a2620b81e81edf3d5c9a16fb33952`）|
| 到 | `base-v0.8.5`（`185e31f030108ad0e85cf8a6c87912222656ba3a`）|
| 发布日期 | 2026-06-24 |
| 升级模式 | `foundation-release-consumer` |
| 变更摘要 | Auth foundation hardening、secure-action runtime updates、upload safety fixes、release artifact packaging |

## 变更范围

本次 release 涉及以下共享路径的变更：

### Backend 变更

| 路径 | 变更类型 | 说明 |
|---|---|---|
| `backend/internal/middleware/secure_action_middleware.go` | MODIFY | 安全操作中间件强化 |
| `backend/internal/middleware/operation_log_middleware.go` | MODIFY | 操作日志中间件变更 |
| `backend/modules/auth/login/login_handler.go` | MODIFY | 登录处理器更新 |
| `backend/modules/auth/login/login_runtime.go` | MODIFY | 登录运行时更新 |
| `backend/modules/auth/security/security_service.go` | MODIFY | 安全服务强化 |
| `backend/modules/auth/mfa/mfa_service.go` | MODIFY | MFA 服务更新 |
| `backend/modules/auth/session/session_service.go` | MODIFY | 会话服务更新 |
| `backend/modules/system/config/setting/setting_handler.go` | MODIFY | 设置处理器更新 |
| `backend/pkg/authtoken/token.go` | MODIFY | Auth Token 处理更新 |
| `backend/pkg/upload/service.go` | MODIFY | 上传服务安全修复 |

### Frontend 变更

| 路径 | 变更类型 | 说明 |
|---|---|---|
| `frontend/src/store/authTypes.ts` | MODIFY | Auth 类型定义更新 |
| `frontend/src/modules/system/user/*` | MODIFY | 用户管理页面更新 |
| `frontend/src/modules/system/setting/*` | MODIFY | 系统设置页面更新 |
| `frontend/src/modules/system/role/*` | MODIFY | 角色管理页面更新 |

### 新增文件

- `frontend/src/modules/system/post/PostList.tsx`（岗位列表）
- `frontend/src/modules/system/permission/PermissionWorkbenchTab.tsx`（权限工作台 Tab）

## ops 本地 Overlay 不受影响

以下路径是 ops 本地扩展点，升级过程不会覆盖：

- `backend/modules/business/generated_registry.go`
- `backend/modules/system/iam/menu/generated_component_registry.go`
- `frontend/src/core/router/componentRegistry.ts`
- `frontend/src/core/router/generatedComponentRegistry.ts`
- `frontend/src/core/router/modules.ts`
- `frontend/src/modules/system/generator/backend-generator.ts`
- `backend/modules/system/i18n/builtin_locale_resources.json`（`business.*` 词条会被保留合并）

## 升级步骤

### 步骤 1：记录当前基座版本

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-base
git rev-parse --short HEAD
# 记录当前 base commit（例如：185e31f0）
```

### 步骤 2：预览升级差异（推荐先执行）

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-ops
npm run upgrade:foundation:local-plan -- --release-version base-v0.8.5
```

此命令会以 dry-run 模式展示所有将被改动的共享文件，不会修改任何文件。

### 步骤 3：执行升级

```powershell
npm run upgrade:foundation:local-apply -- --release-version base-v0.8.5
```

此命令会：

1. Stash 本地未提交的 ops 业务改动
2. 应用 `base-v0.8.5` 的 shared-backend 和 shared-frontend 文件（自动重写模块引用）
3. 保留 ops 本地 overlay 文件（`generated_registry.go`、`componentRegistry.ts` 等）
4. 更新继承文档（`PROJECT_INHERITANCE.md`、双语文本、`foundation-release.lock.json`）
5. 合并 `builtin_locale_resources.json` 中的 `business.*` 词条
6. 运行 `go build ./...` 验证后端模块引用重写正确性
7. 运行 `check:inheritance` 完整继承检查

### 步骤 4：验证

```powershell
# 后端编译验证
cd D:\workspace\go\pantheon-platform\pantheon-ops
go build ./backend/...

# 前端构建验证
cd D:\workspace\go\pantheon-platform\pantheon-ops\frontend
npm run build

# 继承完整性验证
npm run check:inheritance
```

### 步骤 5：如需回滚

如果升级后出现非预期问题，可通过 git 回滚：

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-ops
git checkout -- .
git stash pop
```

然后手动还原 `foundation-release.lock.json`、`docs/PROJECT_INHERITANCE.md` 和 `docs/PROJECT_INHERITANCE.en.md`。

## 预期结果

升级成功后：

- `foundation-release.lock.json` 中 `releaseVersion` 变为 `base-v0.8.5`，`baseCommit` 变为 `185e31f030108ad0e85cf8a6c87912222656ba3a`
- 所有共享 backend/frontend 文件与 `base-v0.8.5` 字节级对齐（overlay 文件除外）
- `go build ./...` 和 `check:inheritance` 均通过

## 回滚注意事项

- `upgrade:foundation:local-apply` 在执行文件写入前会自动 git stash 本地改动
- 如果 `go build` 失败，脚本会输出错误并报告失败，不会自动回滚
- 手动回滚请使用 `git checkout -- .` + `git stash pop`
- `foundation-release.lock.json` 需要手动还原版本号
