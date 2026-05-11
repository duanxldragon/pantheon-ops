# system/org 合同文档

更新时间：2026-04-30

类型：Contract
归属层：system/org
状态：Active

关联设计：
- `SYSTEM_ORG_DESIGN.md`
- `FRONTEND_PAGE_TEMPLATES.md`
- `BACKEND.md`

关联评估：
- `SYSTEM_MODULE_AUDIT.md`

关联整改：
- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`

关联验收：
- `ACCEPTANCE_CHECKLIST.md`
- `PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- `QA_SMOKE_REPORT_20260420.md`

---

本文用于定义 Pantheon `system/org` 能力域的执行契约。

它锁定的是部门、岗位、组织树、组织治理信息的职责边界，避免后续把组织结构治理重新混入 `iam` 用户授权或 `config` 配置页逻辑中。

---

## 1. 背景

`system/org` 在很多后台里最容易被弱化成“部门和岗位两个 CRUD 页”。

但在 Pantheon 里，它更准确的职责是：

> 负责组织结构和组织归属治理，而不是身份认证、角色授权或配置资产管理。

如果没有 `system/org` 合同，后续最容易继续发生：

- 部门、岗位逻辑被塞回用户管理页
- 组织治理摘要和授权治理摘要重复堆叠
- 用户归属、部门树、岗位信息的边界和归责不清
- `org` 页面为了对齐视觉而随意复用不合适的 `iam` 治理语义

## 2. 归属层

本合同归属 `system/org`。

它覆盖：

- 部门管理
- 岗位管理
- 组织树
- 用户组织归属所依赖的结构治理
- 组织治理上下文与问题定位入口

它不等于：

- `system/auth` 登录、会话、安全中心
- `system/iam` 角色、菜单、权限策略
- `system/config` 字典、设置、i18n、上传、动态模块、生成器
- `platform` 壳层导航与工作台聚合

## 3. 目标

`system/org` 合同的目标是锁定以下 5 件事：

1. 明确组织结构治理与身份授权治理的边界
2. 明确部门、岗位、组织树属于 `org`，不是 `iam` 的附属字段页
3. 明确 `org` 页面的右侧治理信息只能服务于组织定位，不重复 `iam` 摘要
4. 明确组织治理的完成定义和验收口径
5. 明确后续组织治理扩展不能侵入 `auth / iam / config`

## 4. 非目标

本合同明确不负责：

- 登录、refresh、session、安全策略
- 角色授权、菜单配置、权限工作台
- 字典、设置、i18n、上传等配置资产治理
- 业务域自己的组织语义扩展

同时，本合同不把“用户属于哪个部门/岗位”误判为 `auth` 或 `iam` 自己的子职责。

## 5. 边界

### 5.1 覆盖对象

- `/system/dept`
- `/system/post`
- 部门树、岗位列表、组织归属摘要
- 组织问题计数、治理定位入口

### 5.2 不覆盖对象

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/login`
- `/auth/security`
- `/system/setting`
- 平台壳层导航与聚合层工作台

## 6. 依赖

`system/org` 合同依赖以下文档与约束：

- [DESIGN.md](D:/workspace/go/pantheon-ops/DESIGN.md)
- [AGENTS.md](D:/workspace/go/pantheon-ops/AGENTS.md)
- [BACKEND.md](D:/workspace/go/pantheon-ops/docs/designs/BACKEND.md)
- [FRONTEND.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND.md)
- [FRONTEND_UI_SPEC.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND_UI_SPEC.md)
- [FRONTEND_PAGE_TEMPLATES.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND_PAGE_TEMPLATES.md)
- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 域边界约束

- 部门、岗位、组织树属于 `system/org`
- 用户归属可被 `iam` 消费，但组织结构定义不能回塞 `iam`
- `org` 不承担认证主链路或配置资产治理

### 7.2 页面与治理约束

- `org` 页面的主列承载真实组织任务流
- 右侧辅助栏只能承载组织定位、治理问题计数与下一步动作
- 不允许重复展示 `iam` 已有的授权摘要

### 7.3 数据与语义约束

- 组织树、岗位、用户归属必须以组织治理语义解释
- 不允许把岗位或部门当作临时字典项处理
- 组织治理扩展必须保持结构清晰，不反向侵入 `dict` 或 `setting`

### 7.4 文档约束

- `system/org` 的设计、评估、整改、验收文档都必须回指本合同
- 如果后续新增专题治理文档，应先说明仍归属 `org` 还是已经升级为跨层专题

## 8. 完成定义

`system/org` 达到“当前已完成”至少应满足：

### 8.1 职责完成

- 部门、岗位、组织治理边界清晰
- `org` 与 `iam / auth / config` 的边界清晰

### 8.2 页面完成

- 部门页、岗位页主链路稳定
- 组织治理信息不再与授权治理信息混写
- 右侧辅助栏符合轻量组织治理语义

### 8.3 数据完成

- 组织树与岗位结构具备稳定的管理语义
- 用户归属依赖组织结构，但不反向污染组织治理边界

### 8.4 文档与验收完成

- `org` 主设计、整改、验收文档都能回链本合同
- 验收口径明确区分组织治理与授权治理

## 9. 验收标准

`system/org` 相关改动至少应通过以下验收：

### 9.0 批量删除能力约束

- 部门、岗位支持受控批量删除，归属 `system/org`，不得回塞到 `iam` 或 `config`。
- 批量删除必须使用独立权限点：`system:dept:batch-delete`、`system:post:batch-delete`。
- 批量删除接口必须复用单条删除服务校验，保留根部门、子部门、岗位占用、用户占用等组织治理保护逻辑。
- 批量删除属于高风险写操作，必须经过二次验证，并返回部分成功结果：`deletedCount`、`failedCount`、`failures[]`。

### 9.1 文档验收

- 符合 [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 前端与构建验收

- `cd frontend && npm run build`
- 如果影响组织治理主链路，补页面级冒烟或验收记录

### 9.3 页面与主链路验收

- `/system/dept`
- `/system/post`

### 9.4 组织治理验收

- 组织右栏信息不能退化为第二主内容列
- 组织治理信息必须服务于定位和处理，不做无意义摘要堆叠

## 10. 关联文档

### 10.1 Design

- [BACKEND.md](D:/workspace/go/pantheon-ops/docs/designs/BACKEND.md)
- [FRONTEND_PAGE_TEMPLATES.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND_PAGE_TEMPLATES.md)
- [FRONTEND_UI_SPEC.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND_UI_SPEC.md)

### 10.2 Assessment

- [SYSTEM_MODULE_AUDIT.md](D:/workspace/go/pantheon-ops/docs/assessments/SYSTEM_MODULE_AUDIT.md)

### 10.3 Remediation

- [BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md](D:/workspace/go/pantheon-ops/docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md)

### 10.4 Acceptance

- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](D:/workspace/go/pantheon-ops/docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- [QA_SMOKE_REPORT_20260420.md](D:/workspace/go/pantheon-ops/docs/archive/QA_SMOKE_REPORT_20260420.md)
