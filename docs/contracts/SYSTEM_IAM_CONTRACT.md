# system/iam 合同文档

更新时间：2026-04-30

类型：Contract
归属层：system/iam
状态：Active

关联设计：
- `PERMISSION_MODEL.md`
- `MODULE_CONTRACT.md`
- `FRONTEND_PAGE_TEMPLATES.md`
- `PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md`
- `NAVIGATION_IA_STRATEGY.md`

关联评估：
- `SYSTEM_MODULE_AUDIT.md`
- `DYNAMIC_MENU_MATURITY_20260422.md`

关联整改：
- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`

关联验收：
- `ACCEPTANCE_CHECKLIST.md`
- `PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- `QA_SMOKE_REPORT_20260420.md`

---

本文用于定义 Pantheon `system/iam` 能力域的执行契约。

它锁定的是用户、角色、菜单、权限四块能力的治理边界，避免后续把导航、页面、按钮、接口权限再次混回“一个 list 权限管全部”的旧模型。

---

## 1. 背景

Pantheon 的 `iam` 不是简单的“用户管理页集合”，而是系统底座里最容易发生职责混乱的一层：

- 用户管理容易和认证混写
- 菜单容易被误当成权限模型
- 页面权限、按钮权限、接口权限容易混成一个字段
- 角色授权、菜单树、页面守卫、Casbin 策略容易脱节

如果没有 `system/iam` 合同，后续最容易继续发生：

- `auth` 和 `iam` 边界重新模糊
- 菜单、页面、按钮、接口四层权限再次混写
- 新模块接入时菜单、权限、路由、i18n、seed 继续各管各的
- `list` 权限重新变成新增、编辑、删除的兜底权限

## 2. 归属层

本合同归属 `system/iam`。

它覆盖：

- 用户管理
- 角色管理
- 菜单管理
- 权限策略与权限工作台
- 导航、页面、按钮、接口四层权限治理
- 模块注册中的菜单/权限/i18n/seed 同步接入约束

它不等于：

- `system/auth` 登录、refresh、session、安全中心
- `system/org` 部门、岗位、组织治理
- `platform` 壳层导航样式和聚合层工作台

## 3. 目标

`system/iam` 合同的目标是锁定以下 6 件事：

1. 明确 `iam` 与 `auth` 的边界
2. 明确菜单不是权限，页面权限不等于列表权限
3. 明确用户、角色、菜单、权限工作台属于同一治理域
4. 明确模块接入时菜单、权限、路由、i18n、seed 必须同步注册
5. 明确权限工作台的职责是“发现问题、解释问题、导出问题、受控整改问题”
6. 明确 `iam` 的完成定义和验收口径

## 4. 非目标

本合同明确不负责：

- 登录、refresh、logout、session、password 策略
- 部门、岗位等组织治理规则
- 系统设置、字典、i18n 运行时资产和上传配置
- 业务域的数据权限细节
- 业务域具体菜单结构设计

同时，本合同不把“菜单配置”误认为“平台壳层视觉策略”。

导航样式属于 `platform`，导航元数据和授权语义属于 `system/iam`。

## 5. 边界

### 5.1 覆盖对象

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- 用户、角色、菜单、权限相关后端接口
- 动态菜单元数据
- 页面权限与按钮权限语义
- 接口权限与 Casbin 策略映射
- 模块接入中的菜单/权限/i18n/seed 契约

### 5.2 不覆盖对象

- `/login`
- `/auth/security`
- `/system/session`
- `/system/login-log`
- `/system/dept`
- `/system/post`
- `/system/setting`
- 平台壳层导航样式与双模式验收

## 6. 依赖

`system/iam` 合同依赖以下文档与约束：

- [DESIGN.md](D:/workspace/go/pantheon-ops/DESIGN.md)
- [AGENTS.md](D:/workspace/go/pantheon-ops/AGENTS.md)
- [BACKEND.md](D:/workspace/go/pantheon-ops/docs/designs/BACKEND.md)
- [FRONTEND.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND.md)
- [PERMISSION_MODEL.md](D:/workspace/go/pantheon-ops/docs/designs/PERMISSION_MODEL.md)
- [MODULE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/designs/MODULE_CONTRACT.md)
- [FRONTEND_PAGE_TEMPLATES.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND_PAGE_TEMPLATES.md)
- [ERROR_CODE_AND_I18N.md](D:/workspace/go/pantheon-ops/docs/designs/ERROR_CODE_AND_I18N.md)
- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 域边界约束

- `auth` 负责“你是谁、你能否登录、会话是否有效”
- `iam` 负责“你能看到什么、进入什么、操作什么、调用什么”
- 用户 CRUD 不得重新承担认证主链路职责

### 7.2 四层权限约束

- 导航权限、页面权限、按钮权限、接口权限必须分层
- 菜单不是权限模型本体
- 页面权限不等于列表查询权限
- 禁止继续使用一个 `list` 权限同时控制新增、编辑、删除

### 7.3 模块接入约束

- 新模块接入必须同步考虑菜单、路由、权限点、i18n、seed
- 不允许靠“记忆”手工维护分散注册信息
- 模块接入契约优先于具体实现

### 7.4 权限工作台约束

- 权限工作台的职责是治理，不是任意写策略
- 未知权限、页面缺口、API 缺口必须被识别和解释
- 受控整改可以存在，但必须受推荐映射和边界约束控制

### 7.5 文档约束

- `system/iam` 的设计、评估、整改、验收文档都必须回指本合同
- 任何新权限模型变更，不得绕开 [PERMISSION_MODEL.md](D:/workspace/go/pantheon-ops/docs/designs/PERMISSION_MODEL.md)

## 8. 完成定义

`system/iam` 达到“当前已完成”至少应满足：

### 8.1 职责完成

- 用户、角色、菜单、权限四块能力归属清晰
- `iam` 与 `auth / org / config` 边界清晰

### 8.2 权限模型完成

- 导航、页面、按钮、接口四层权限模型稳定
- 页面守卫、按钮显隐和后端接口鉴权语义一致
- 不再依赖粗粒度 `list` 权限兜底所有动作

### 8.3 接入契约完成

- 菜单、权限、路由、i18n、seed 具备同步注册机制
- 新模块接入不会继续发散成多套来源

### 8.4 治理闭环完成

- 权限工作台能够发现未知权限、页面缺口、API 缺口
- 支持导出盘点与受控整改

### 8.5 文档与验收完成

- `iam` 设计、整改、验收文档都能回链本合同
- 动态菜单、权限模型和 UI 模板的引用关系稳定

## 9. 验收标准

`system/iam` 相关改动至少应通过以下验收：

### 9.0 批量删除能力约束

- 用户、角色、权限策略支持受控批量删除，分别归属 `system/iam` 的用户治理、角色治理与接口策略治理。
- 批量删除不得复用单条删除权限或批量启停权限，必须使用独立权限点：`system:user:batch-delete`、`system:role:batch-delete`、`system:permission:batch-delete`。
- 批量删除接口必须复用单条删除服务校验，保留内置管理员、内置角色、角色占用、Casbin 策略刷新等保护逻辑。
- 批量删除属于高风险写操作，必须经过二次验证，并返回部分成功结果：`deletedCount`、`failedCount`、`failures[]`。

### 9.1 文档验收

- 符合 [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 后端与权限验收

- `go test ./backend/modules/system/permission`
- 相关用户、角色、菜单模块改动应补对应测试

### 9.3 前端与构建验收

- `cd frontend && npm run build`
- `cd frontend && npm run check:menu-contract`
- 如果涉及页面权限或角色授权主链路，补对应冒烟或专项验收

### 9.4 页面与主链路验收

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`

### 9.5 权限一致性验收

- 页面守卫与后端 Casbin 权限不能语义断裂
- 菜单可见性与页面可进入性不能混写
- 权限工作台发现的缺口必须有明确解释，而不是静默失败

## 10. 关联文档

### 10.1 Design

- [PERMISSION_MODEL.md](D:/workspace/go/pantheon-ops/docs/designs/PERMISSION_MODEL.md)
- [MODULE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/designs/MODULE_CONTRACT.md)
- [FRONTEND_PAGE_TEMPLATES.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND_PAGE_TEMPLATES.md)

### 10.2 Assessment

- [SYSTEM_MODULE_AUDIT.md](D:/workspace/go/pantheon-ops/docs/assessments/SYSTEM_MODULE_AUDIT.md)
- [DYNAMIC_MENU_MATURITY_20260422.md](D:/workspace/go/pantheon-ops/docs/assessments/DYNAMIC_MENU_MATURITY_20260422.md)

### 10.3 Remediation

- [BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md](D:/workspace/go/pantheon-ops/docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md)

### 10.4 Acceptance

- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](D:/workspace/go/pantheon-ops/docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- [QA_SMOKE_REPORT_20260420.md](D:/workspace/go/pantheon-ops/docs/archive/QA_SMOKE_REPORT_20260420.md)
