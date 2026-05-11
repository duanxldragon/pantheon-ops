# 文档合同化治理方案

更新时间：2026-04-30

本文用于定义 Pantheon Base 的文档治理模型。

目标不是再新增一类“好看的文档名称”，而是把项目文档从“资料堆”收敛为一套可执行的治理系统，让设计、实现、评估、整改、验收都围绕同一份契约推进。

---

## 1. 结论

“合同文档”这个想法是合理的，而且适合 Pantheon 目前的阶段。

原因不是项目缺文档，而是当前文档存在三个结构性问题：

- **类型混杂**：设计文档、评估文档、整改记录、验收样例都混在同一层；
- **主次失真**：长期有效的设计规范和阶段性盘点稿一起进入主入口；
- **缺少锚点**：很多评估和整改都在推进，但缺少一份能定义“边界、目标、完成定义、验收口径”的上层契约。

所以这里的“合同文档”，本质上不是法律式文书，而是：

> **某个层级、某个模块或某个专题的执行契约文档。**

它回答的是：

- 这件事到底归谁？
- 它要做到什么程度才算完成？
- 明确不做什么？
- 后续设计、实现、评估、整改、验收分别挂到哪里？

---

## 2. 文档分层模型

建议 Pantheon 后续固定使用 5 类文档。

### 2.1 `Contract`

作用：

- 作为某个层级、模块或专题的**最高约束文档**
- 定义边界、目标、非目标、完成定义、验收标准

特征：

- 生命周期长
- 进入主索引
- 后续 `Design / Assessment / Remediation / Acceptance` 都应引用它

### 2.2 `Design`

作用：

- 说明“应该如何设计”
- 展开数据结构、交互、API、状态、策略等实现思路

特征：

- 生命周期长
- 进入主索引
- 必须隶属于某份 `Contract`

### 2.3 `Assessment`

作用：

- 说明“当前实现距离合同/设计还有多远”
- 输出问题盘点、风险判断、差距矩阵

特征：

- 生命周期中等
- 默认不作为主索引一线入口
- 必须指向对应 `Contract`

### 2.4 `Remediation`

作用：

- 说明“准备如何补齐 Assessment 识别出的缺口”
- 用于整改方案、治理计划、阶段收口路线

特征：

- 生命周期中等
- 可进入主索引，但应作为次级入口
- 必须挂在某份 `Contract` 或某个专题治理链路下

### 2.5 `Acceptance`

作用：

- 说明“是否已经达到合同要求”
- 包含模板、样例、验收矩阵、阶段验收记录

特征：

- 模板类和基线类可长期保留
- 一次性样例可保留为历史基线
- 必须回指 `Contract`

---

## 3. 文档关系模型

### 3.1 正确关系

```text
Contract
  -> Design
  -> Assessment
  -> Remediation
  -> Acceptance
```

### 3.2 不建议继续的关系

```text
Assessment -> Assessment -> Assessment
Remediation -> 新 Remediation -> 新评估稿
README 平铺所有 dated 文档
```

也就是说：

- 不能让评估稿自己变成新的主文档；
- 不能每整改一轮就新增一堆没有上层归属的 dated 文件；
- 不能让索引首页成为“时间线列表”。

---

## 4. 建议的粒度

合同文档不应一开始细到每个页面。

Pantheon 当前最合理的合同粒度是两层。

### 4.1 第一层：平台级合同

建议先建立以下 5 份平台级合同：

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`

它们是最稳定、最长期的治理边界。

### 4.2 第二层：专题级合同

只在某个问题跨多个页面、且具有长期治理价值时建立：

- `platform shell`
- `dynamic menu governance`
- `i18n runtime governance`
- `dynamic module governance`
- `generator governance`

不要为一次性问题建立合同文档。

判断标准：

- 是否跨多个页面或多个模块
- 是否会被反复引用
- 是否需要长期验收纪律
- 是否需要独立的完成定义

---

## 5. 合同文档标准结构

建议每份合同文档固定包含以下部分：

## 1. 背景

- 为什么需要这份合同
- 它要解决什么混乱或风险

## 2. 归属层

- 属于 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config`、`business/*` 哪一层

## 3. 目标

- 明确要达成什么

## 4. 非目标

- 明确本轮不解决什么

## 5. 边界

- 这个合同覆盖哪些页面、模块、接口、流程
- 不覆盖哪些对象

## 6. 依赖

- 依赖哪些设计文档、契约、数据库、公共组件或流程

## 7. 强约束

- 后续实现必须遵守的红线

## 8. 完成定义

- 这份合同如何判定“已完成”

## 9. 验收标准

- 必须通过哪些检查、矩阵、模板、命令

## 10. 关联文档

- Design
- Assessment
- Remediation
- Acceptance

---

## 6. 命名与目录建议

不建议现在直接用纯英文前缀重命名全部历史文件，会带来一次大迁移成本。

更稳的方式是：

### 6.1 先保留现有文件名

先在内容层标明类型：

- `类型：Contract`
- `类型：Design`
- `类型：Assessment`
- `类型：Remediation`
- `类型：Acceptance`

### 6.2 再逐步建立目录分组

长期建议目录形态：

```text
docs/
  README.md
  contracts/
  designs/
  assessments/
  remediations/
  acceptances/
  archive/
```

但第一轮不建议物理迁移全部文件。

原因：

- 当前仓库里已有大量交叉引用
- 一次性移动会把“治理问题”放大成“链接修复工程”
- 先把类型和主次理顺，收益更高

### 6.3 第一轮更务实的做法

第一轮只做：

- 重写索引
- 建立合同文档模板
- 产出首批合同文档
- 清理已被覆盖的中间评估稿
- 给剩余文档标注类型与归属

---

## 7. 与现有文档的关系判断

### 7.1 可以继续保留为 `Design`

- `BACKEND.md`
- `FRONTEND.md`
- `FRONTEND_UI_SPEC.md`
- `AUTH_MODULE_DESIGN.md`
- `PERMISSION_MODEL.md`
- `DICT_AND_SETTING_DESIGN.md`
- `SYSTEM_CONFIG_EXTENDED_DESIGN.md`
- `I18N_MODULE_DESIGN.md`
- `UPLOAD_AND_STORAGE_DESIGN.md`
- `DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
- `GENERATOR_MODULE_DESIGN.md`

### 7.2 更适合作为 `Contract`

建议新增，而不是直接把现有设计文档硬改成合同文档：

- `platform contract`
- `system/auth contract`
- `system/iam contract`
- `system/org contract`
- `system/config contract`

### 7.3 继续作为 `Assessment`

- `PLATFORM_GAP_AUDIT_20260429.md`
- `DYNAMIC_MENU_MATURITY_20260422.md`
- `SYSTEM_MODULE_AUDIT.md`

### 7.4 继续作为 `Remediation`

- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`

### 7.5 继续作为 `Acceptance`

- `ACCEPTANCE_CHECKLIST.md`
- `PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- `PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`
- `PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`
- `QA_SMOKE_REPORT_20260420.md`

---

## 8. 首批落地计划

建议按三步落地。

### 第一步：先建治理骨架

输出物：

- 本文档
- 一份合同文档模板
- 一份文档类型说明

目标：

- 先固定规则，不急着全量迁移

### 第二步：建立首批平台级合同

建议优先产出：

1. `platform contract`
2. `system/auth contract`
3. `system/iam contract`
4. `system/org contract`
5. `system/config contract`

目标：

- 让后续设计、评估、整改、验收都有稳定归属

### 第三步：做文档映射与收口

输出物：

- 文档矩阵
- 索引重构
- 删除或降级一批中间评估稿

目标：

- 首页只保留核心文档
- 阶段文档退到二级入口

---

## 9. 第一轮不做什么

为了避免把治理做成大迁移工程，第一轮明确不做：

- 不一次性重命名所有现有文档
- 不一次性移动全部文件到新目录
- 不要求每个页面都建立合同文档
- 不把所有历史报告都删光

第一轮的目标只有一个：

> 先让文档体系形成“合同 -> 设计 / 评估 / 整改 / 验收”的稳定主干。

---

## 10. 建议的下一步

如果确认按这个模型落地，下一步最合适的动作是：

1. 新建 `合同文档模板`
2. 产出首批 5 份平台级合同文档骨架
3. 更新 `docs/README.md`，让索引正式按文档类型和层级组织

等这三步完成后，再讨论是否做物理目录迁移。
