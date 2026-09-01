---
title: 平台层合同文档
doc_type: Contract
layer: platform
status: Active
related_designs:
  - docs/designs/PLATFORM_DASHBOARD_DESIGN.md
  - docs/designs/FRONTEND_UI_SPEC.md
  - docs/designs/FRONTEND_PAGE_TEMPLATES.md
  - docs/designs/NAVIGATION_IA_STRATEGY.md
  - docs/designs/P2_SCALE_ROADMAP.md
related_acceptances:
  - docs/acceptances/ACCEPTANCE_CHECKLIST.md
  - docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md
  - docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md
updated_at: 2026-04-30
---

# 平台层合同文档

English version: [PLATFORM_CONTRACT.en.md](./PLATFORM_CONTRACT.en.md)

关联设计：
- `PLATFORM_DASHBOARD_DESIGN.md`
- `FRONTEND_UI_SPEC.md`
- `FRONTEND_PAGE_TEMPLATES.md`
- `NAVIGATION_IA_STRATEGY.md`
- `P2_SCALE_ROADMAP.md`

关联验收：
- `ACCEPTANCE_CHECKLIST.md`
- `PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- `PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`

---

本文用于定义 Pantheon `platform` 层的执行契约。

它不是某一页的视觉稿，也不是一次整改记录，而是平台层后续设计、实现、评估、整改、验收都必须回挂的上层边界文档。

---

## 1. 背景

Pantheon 当前已经不再是“只有登录 + 菜单 + CRUD”的后台壳。

但在平台层仍存在两个典型风险：

- `dashboard`、导航壳层、浮层、全局状态页这类跨域能力，容易被重新塞回某个 `system/*` 子域；
- 平台层设计、评估、整改、验收已经很多，但缺少一份能定义“什么属于 platform、什么算完成、什么不该继续扩散”的上层契约。

如果没有这份合同，平台层最容易发生的不是功能缺失，而是：

- 聚合层边界漂移
- 壳层风格再次分叉
- 新页面继续复制旧骨架
- 历史整改稿重新变成新的事实依据

## 2. 归属层

本合同归属 `platform` 层。

这里的 `platform` 是逻辑层，不强制要求物理目录名必须叫 `platform`。

它覆盖：

- 登录页语言选择与当前会话语言解析
- 登录后应用壳层
- 平台工作台 / 仪表盘
- 全局导航模式
- 平台级页面骨架与浮层基线
- 跨域聚合入口
- 平台级状态、反馈和验收纪律

它不等于：

- `system/auth`
- `system/iam`
- `system/org`
- `system/config`

这些系统域可以被平台层聚合，但不能反向替代平台层职责。

## 3. 目标

平台层合同的目标是锁定以下 5 件事：

1. 明确 `platform` 与各系统域的职责边界
2. 明确应用壳层、导航、工作台和浮层属于平台层统一治理
3. 明确平台层只做聚合、导航和公共骨架，不回塞单域业务
4. 明确后续 UI 收口必须围绕统一骨架，而不是继续复制历史模板
5. 明确平台层改动的完成定义和验收纪律

## 4. 非目标

本合同明确不负责：

- 设计某个 `business/*` 页面
- 承担 `system/auth`、`system/iam`、`system/org`、`system/config` 单域内部业务细节
- 规定每个系统页的所有字段和交互
- 直接替代单个设计文档的详细实现说明

本合同也不把“低代码平台”作为 Pantheon 当前主目标。

低代码能力可以存在于平台治理范围内，但不能反向改写平台层的核心定位。

## 5. 边界

### 5.1 覆盖对象

- `dashboard` / 工作台 / 平台首页
- 应用壳层：导航、顶部栏、页签、品牌区、布局切换
- 平台级公共骨架：`PageContainer / PageHeader / FilterPanel / PageSplitLayout / SideRail / AppModal / AppDrawer`
- 平台级全局反馈：loading / empty / error / forbidden / submitting 的统一表达
- 平台级导航图标语义
- 平台级验收模板、迁移矩阵和 PR 纪律

### 5.2 不覆盖对象

- `system/auth` 内部认证协议与会话模型
- `system/iam` 内部授权数据结构
- `system/org` 内部组织治理规则
- `system/config` 内部配置存储与运行时资产细节
- 具体业务模块页面的业务交互

## 6. 依赖

平台层合同依赖以下文档与约束：

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)
- [PLATFORM_DASHBOARD_DESIGN.md](../designs/PLATFORM_DASHBOARD_DESIGN.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 聚合层约束

- 仪表盘、工作台、首页概览、跨域统计卡片统一归属 `platform`
- 聚合层可以读取多个子域数据，但不能侵入子域内部职责
- 新增平台卡片必须说明来源域、跳转目标和权限边界

### 7.2 壳层约束

- 导航、顶部栏、页签、品牌区、布局切换属于 `platform` 壳层
- 左导航只承担导航，不混入说明卡、统计卡、帮助卡
- 竖版侧栏与横版顶栏必须共享同一套状态语言
- 运行时语言解析优先级固定为 `本次登录选择 > 用户偏好 > 系统默认`
- 登录页语言下拉属于 `platform` 会话级覆盖，不等于自动写入用户长期偏好
- 当前会话语言覆盖必须在登出时清理，不得污染下一位登录主体

### 7.3 骨架约束

- 平台层统一维护页面骨架、右侧辅助栏和浮层基线
- 业务页面不得重新引入旧右栏模板或绕过平台浮层封装
- 平台层允许系统域页面填内容，不允许系统域各自发明新的壳层模式

### 7.3.1 表格交互约束

- 跨页选择属于 `platform` 交互契约，不绑定单个当前页
- 仅翻页或修改每页条数时，不得清空选择集
- 搜索词、筛选条件、排序字段、排序方向和重置动作会改变查询上下文，必须清空选择集
- 批量动作必须基于完整选择集执行，而不是只基于当前页可见行
- 当前页复选框选中态应由“完整选择集 ∩ 当前页可见数据”派生

### 7.4 页面准入约束

- 任务优先页默认禁止治理抽屉和治理 Hero 抢占主叙事，例如 `/system/user`
- 配置页允许总览卡片和风险说明，但不允许再引入与页签、面包屑重复的右侧治理菜单，例如 `/system/setting`
- 只有治理型页面和审计型页面，才允许使用治理抽屉，例如角色、权限、组织、日志等页面
- 页面准入规则必须由共享清单维护，并进入自动检查与 smoke 白名单，不能再由单个页面自由决定

### 7.5 文档约束

- 平台层的设计、评估、整改、验收文档都必须回指本合同
- 平台层 dated 评估稿如果被后续矩阵或规范覆盖，应删除或转为归档

## 8. 完成定义

平台层达到“当前已完成”至少应满足：

### 8.1 职责完成

- `dashboard` 已稳定归属 `platform`
- 壳层、导航、浮层和平台级状态归属清晰

### 8.2 视觉与交互完成

- 左导航与横版顶栏已共享同一套菜单状态语言
- 右侧辅助栏已从历史模板切换到平台骨架语义
- 原生业务层静态 `Modal.*` 与原生 `<Drawer>` 不再扩散

### 8.3 文档与流程完成

- 平台层有稳定的迁移矩阵
- 平台层有固定双模式验收模板
- 平台壳层提交必须附验收记录和扫描摘要

### 8.4 回归控制完成

- 不再新增旧壳层样式类名
- 不再新增未纳入平台封装的业务层浮层入口
- 平台层索引和主文档不再被阶段性评估稿淹没
- 页面准入规则具备自动检查与冒烟白名单，不再依赖人工记忆

## 9. 验收标准

平台层相关改动至少应通过以下验收：

### 9.1 文档验收

- 符合 [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 UI 验收

- 参考 [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- 参考 [PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md](../acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md)

### 9.3 固定扫描

- `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src`
- `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src`
- `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components`

### 9.4 构建与回归

- `cd frontend && npm run build`
- 如果影响壳层交互或系统页主链路，需补页面级冒烟或验收记录

## 10. 关联文档

### 10.1 Design

- [PLATFORM_DASHBOARD_DESIGN.md](../designs/PLATFORM_DASHBOARD_DESIGN.md)
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- [PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md](../acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md)

## 变更日志
| 日期 | 版本 | 变更内容 | 原因 |
|------|------|---------|------|
| 2026-06-25 | v1.0 | 初始版本 | 评估审计添加 |

