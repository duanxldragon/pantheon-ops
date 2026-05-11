# Contributing

本文定义 Pantheon Platform 的基础协作规则、提交规范和验证要求。

## 分层边界

提交前必须先判断改动归属：

- `platform`：应用壳层、工作台、跨域聚合、路由装配、公共设施。
- `system/auth`：登录、会话、Token、安全中心、登录日志。
- `system/iam`：用户、角色、菜单、权限点、角色授权。
- `system/org`：部门、岗位、组织架构、用户组织归属。
- `system/config`：系统设置、字典、配置缓存。
- `business/*`：具体业务域模块。

跨层改动必须在 PR 描述中说明边界、依赖和验证方式。

## 提交信息

使用 Conventional Commits：

```text
type(scope): subject
```

### Type

- `feat`：新增功能。
- `fix`：缺陷修复。
- `docs`：文档变更。
- `style`：格式、样式调整，不改变行为。
- `refactor`：重构，不改变外部行为。
- `perf`：性能优化。
- `test`：测试新增或修复。
- `build`：构建、依赖、打包相关。
- `ci`：CI/CD 配置。
- `chore`：工程杂项。
- `revert`：回滚提交。

### Scope 建议

- `platform`
- `system-auth`
- `system-iam`
- `system-org`
- `system-config`
- `system-audit`
- `business-cmdb`
- `frontend`
- `backend`
- `docs`
- `ci`
- `tests`

### 示例

```text
feat(system-iam): unify role authorization trees
fix(system-org): prevent inconsistent user post assignment
docs(platform): add repository README
test(system-iam): cover role authorization tree workflow
ci: run frontend contract and backend tests
```

## 本地启用提交规范

```bash
git config commit.template .gitmessage
git config core.hooksPath .githooks
```

启用后，`git commit` 会使用 `.gitmessage` 作为模板，`commit-msg` hook 会校验提交标题。

## 验证要求

按改动范围选择最小必要验证：

```bash
# 后端系统域
go test ./backend/modules/auth ./backend/modules/system/...

# 前端构建与菜单契约
cd frontend
npm run build

# 系统页 smoke
npm run test:smoke:system

# 角色授权专项 smoke
npm run test:smoke:role-auth

# 导入导出 smoke
npm run test:smoke:impexp

# 后台 UI smoke
npm run test:smoke:backoffice-ui
```

## PR Checklist

- [ ] 已说明改动归属层级与跨域依赖。
- [ ] 未把认证、用户、角色、菜单、权限、组织、配置混成一个“大 system”处理。
- [ ] 后端响应仍使用统一响应结构。
- [ ] 前端新增展示文本已走 i18n。
- [ ] 菜单、页面权限、操作权限、接口策略边界清晰。
- [ ] 已补充或更新必要文档。
- [ ] 已执行与改动范围匹配的测试。
