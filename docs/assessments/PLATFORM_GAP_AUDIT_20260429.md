# 平台功能与设计缺口审计（2026-04-29）

更新时间：2026-04-29

类型：Assessment
归属层：platform
状态：Active

本次审计遵循 Pantheon 分层边界，按 **platform / system/auth / system/iam / system/org / system/config / business/\*** 做只读评估。

本报告不评估单个页面视觉细节，也不进入“发现即修复”的代码变更流程，重点回答两个问题：

- 当前哪些能力已经进入实现，但文档和验收没有跟上？
- 当前哪些能力虽然已有设计目标，但运行时仍未真正闭环？

---

## 一、结论摘要

当前 Pantheon Base 的主结论不是“功能做得少”，而是：

- `platform` 与主要系统域主链路已经基本成型；
- `system/config` 与 `business/cmdb/*` 的实现扩张速度，已经开始超过设计文档和验收基线；
- 若不及时补文档与验收矩阵，后续最容易出现的不是“不会做”，而是“越做越散、越做越难判断边界”。

一句话判断：

> 当前的主要风险，是 **实现版图领先于设计与验收版图**。

---

## 二、范围与边界

本轮审计覆盖：

- `platform`：壳层、仪表盘、跨域聚合入口
- `system/auth`：登录、refresh、logout、安全中心、会话、登录日志
- `system/iam`：用户、角色、菜单、权限工作台
- `system/org`：部门、岗位、组织治理
- `system/config`：字典、设置、i18n、上传、动态模块管理、生成器
- `business/*`：以 `business/cmdb`、`business/cmdb/host`、`business/cmdb/vendor` 为样本

本轮不做：

- 逐页交互冒烟
- 样式一致性复检
- 单元测试和 smoke 重新执行

---

## 三、核心发现

### 3.1 platform

结论：

- 平台层边界总体正确，`dashboard` 已作为聚合层存在，而不是回塞 `system/*`
- 当前缺口不在平台首页本身，而在“平台层如何持续吸纳新增系统域/业务域能力”的验收机制

已知依据：

- 设计文档已明确 `dashboard` 属于 `platform` 聚合层
- 前端模块已声明 `scope: 'platform'`
- 后端已有 `GET /api/v1/platform/dashboard/summary`

缺口：

- 验收清单对 `platform` 仍偏静态，主要停留在 `/dashboard` 是否可打开，缺少“新增聚合卡片如何声明来源域、权限和跳转归属”的检查项

### 3.2 system/auth

结论：

- `auth` 从 `user` 中拆出这件事基本成立
- 当前主缺口不是认证主链路，而是审计与管理员安全治理的文档仍分散

已闭环能力：

- `/api/v1/auth/login`
- `/api/v1/auth/refresh`
- `/api/v1/auth/logout`
- `/api/v1/auth/security`
- `/api/v1/auth/sessions`
- `/api/v1/auth/login-logs`
- 管理员登录日志与全局会话页

缺口：

- `system/auth` 与 `system/audit` 的边界还没有独立域文档收口，登录日志、会话清理、危险动作二次验证分散在多个文档中
- `system/auth` 验收主要覆盖页面可用性，缺少“历史会话清理 / 登录日志批量删除 / 二次验证保护”这样的治理型验收项

### 3.3 system/iam

结论：

- `system/iam` 已完成第一层闭环，能发现问题、解释问题、导出报表
- 但还没有进入“工作台内可直接整改”的第二层闭环

已闭环能力：

- 用户、角色、菜单、权限页面
- 页面权限守卫
- Casbin 接口策略管理
- 权限工作台缺口识别
- 权限治理报表导出

功能缺失：

- 权限工作台仍缺少批量整改动作
- 风险角色整改后的回写引导不足
- “发现问题”到“完成治理”的路径仍需回到多个页面人工拼接

判断：

- 这不是实现错误，而是功能阶段尚未走完
- 应明确标注为“治理第一层已完成，治理第二层未完成”

### 3.4 system/org

结论：

- `system/org` 的真实能力已超出普通部门/岗位 CRUD
- 但文档层还缺少独立的组织治理设计文档

已存在能力：

- 部门树
- 岗位管理
- 组织总览
- 治理任务列表
- 治理任务导出

缺口：

- `docs/` 中没有独立 `system/org` 设计文档
- 当前组织治理能力主要散落在 `BACKEND.md`、阶段评估和代码中
- 后续若继续扩展“无人负责部门 / 空部门 / 无岗位部门”之类治理语义，缺少统一文档锚点

### 3.5 system/config

结论：

- 这是当前“实现领先于文档/验收”最明显的系统域

当前真实能力已经包含：

- 字典管理
- 系统设置
- i18n 管理与生命周期治理
- 上传配置与统一上传入口
- 动态模块管理
- 模块生成器

主要缺口：

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md) 的固定页面覆盖曾停留在 `/system/dict`、`/system/setting`
- 未把 `/system/i18n`、`/system/modules`、`/system/generator` 纳入固定验收矩阵
- 缺少独立的 `i18n` 设计文档
- 缺少独立的动态模块/生成器治理设计文档
- 上传能力已有实现与配置消费，但没有独立“上传与存储边界”设计文档

判断：

- 当前不应该再把 `system/config` 理解成“设置页附属能力”
- 它已经演进为一个包含配置、国际化、上传、模块治理的复合系统域

### 3.6 business/cmdb

结论：

- `business/cmdb` 已不再只是单一入口样板，而是开始出现子模块分化
- 设计文档与业务验收基线此前没有跟上这个变化，本轮已补齐 CMDB 总设计和业务验收矩阵

当前已发现的真实模块：

- `business/cmdb`
- `business/cmdb/host`
- `business/cmdb/vendor`

关键问题：

- 文档已迁移为 [BUSINESS_CMDB_MODULE_DESIGN.md](../designs/BUSINESS_CMDB_MODULE_DESIGN.md)，并明确纳入 `cmdb/host` 与后续 `cmdb/vendor`
- `vendor` 当前不再按“已实现”统计，而是按“设计归属已明确，待实现”管理
- 后续实现 `vendor` 前必须补 DDL、菜单、权限、i18n、审计与 smoke 验收证据

判断：

- 这属于典型的“业务模块已长出新切片，但设计仍停留在旧样板”

---

## 四、缺口矩阵

| 类别 | 层级 | 当前状态 | 缺口性质 | 优先级 |
| :--- | :--- | :--- | :--- | :--- |
| 权限工作台第二层治理 | `system/iam` | 已有发现、导出、受控补齐与深化设计 | 进入整改追踪实现阶段 | P1 |
| 组织治理域文档 | `system/org` | 已补 `docs/designs/SYSTEM_ORG_DESIGN.md` | 已收口 | P1 |
| i18n 独立域设计 | `system/config` | 已补 `docs/designs/I18N_MODULE_DESIGN.md` | 已收口 | P0 |
| 动态模块治理设计 | `system/config` | 已补 `docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md` | 已收口 | P0 |
| 上传与存储边界设计 | `system/config` | 已补 `docs/designs/UPLOAD_AND_STORAGE_DESIGN.md` | 已收口 | P1 |
| `/system/i18n` 固定验收基线 | `system/config` | 已纳入 `SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md` | 已收口 | P0 |
| `/system/modules` 固定验收基线 | `system/config` | 已纳入 `SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md` | 已收口 | P0 |
| `/system/generator` 固定验收基线 | `system/config` | 已纳入 `SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md` | 已收口 | P0 |
| `business/cmdb/vendor` 设计定义 | `business/cmdb` | 已纳入 `BUSINESS_CMDB_MODULE_DESIGN.md` | 待实现 | P0 |
| `business/*` 业务域专项验收矩阵 | `business/*` | 已补 `BUSINESS_MODULE_ACCEPTANCE_MATRIX.md` | 已收口 | P0 |

---

## 五、最需要先补的文档

本轮已按 typed docs 结构补齐：

1. `docs/designs/I18N_MODULE_DESIGN.md`
2. `docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
3. `docs/designs/UPLOAD_AND_STORAGE_DESIGN.md`
4. `docs/designs/SYSTEM_ORG_DESIGN.md`
5. `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
6. `docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`
7. `docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`

---

## 六、最需要先补的验收项

固定验收入口已拆出到：

- `docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
- `docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`

### 6.1 system/config 固定页面覆盖补齐

至少纳入：

- `/system/i18n`
- `/system/modules`
- `/system/generator`

并补充以下检查项：

- 页面是否能正常加载
- 对应 `pagePermission` 是否正确生效
- 导入/导出/刷新缓存/生成/注册/卸载等危险动作是否受权限和二次验证保护
- i18n、动态模块、生成器是否保留固定 smoke 证据

### 6.2 business 域专项验收矩阵

至少纳入：

- `/business/cmdb/list`
- `/business/cmdb/host`
- `/business/cmdb/vendor`

并检查：

- 菜单 seed
- 页面权限
- 动作权限
- i18n namespace
- 审计点
- 字典依赖
- 导入导出或批量治理能力

---

## 七、根因判断

当前缺口的根因不是“团队忘了设计”，而是：

1. 第一轮设计文档主要围绕 `platform + system` 主链路展开，业务域和扩展配置域还只是样板阶段
2. 近期实现重点落在 `system/config` 扩张和业务模块生成/注册能力，导致真实能力边界发生了二次增长
3. 验收清单仍沿用较早的系统页固定覆盖范围，没有随着模块版图同步升级

所以现在最该做的不是继续补零散说明，而是把 **system/config 与 business/* 的设计锚点重新立起来**。

---

## 八、建议的下一步

### P0，本周应完成

- 已完成：补 `system/config` 扩展设计文档与高敏验收基线
- 已完成：把 `/system/i18n`、`/system/modules`、`/system/generator` 纳入固定验收入口
- 已完成：明确 `business/cmdb/vendor` 的设计归属与验收要求

### P1，下一阶段完成

- 已完成：为 `system/org` 补独立设计文档
- 已完成：为 `system/auth` 安全策略深化补统一说明
- 已完成：为 `business/*` 建立统一验收矩阵模板

### P2，后续增强

- 已完成设计锚点：把 `system/iam` 权限工作台从“发现 + 导出”推进到“发现 + 整改 + 追踪”
- 已完成设计锚点：把业务模块验收从手工文档升级到固定 smoke 套件与报告归档
- 待实现：权限整改追踪落表、业务 smoke 常态化报告归档

---

## 九、最终判断

Pantheon 当前最需要警惕的，不是“没有能力”，而是“能力长出来了，但没有及时被文档和验收接住”。

只要这一轮把 `system/config` 和 `business/*` 的设计与验收锚点补上，平台层、系统域、业务域三层语言才会重新对齐。

这一步必须尽快做。否则后面每新增一个模块，判断成本都会变高。
