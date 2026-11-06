---
title: 文档类型与状态说明
doc_type: Contract
layer: platform
status: Active
linked_contracts:
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
  - docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md
updated_at: 2026-10-31
---

# 文档类型与状态说明

English version: [DOCUMENT_METADATA_AND_STATUS.en.md](./DOCUMENT_METADATA_AND_STATUS.en.md)

本文用于定义 Pantheon Base 文档系统的统一元信息字段、文档类型枚举和生命周期状态枚举。

---

## 1. 目标

如果只定义“文档类型”，不定义“文档状态”，项目很快又会回到旧问题：

- 文档知道自己属于 `Design` 还是 `Assessment`，但不知道自己是不是已经过期；
- 主索引里混着 `Draft`、`Active`、`Superseded` 文档；
- AI 和新人看不出当前应该信哪份；
- 旧评估稿虽然还在，但没有任何退场信号。

因此 Pantheon 后续所有主文档都应补齐统一元信息。

---

## 2. 统一元信息字段（指针）

字段定义的唯一权威是 [DOCUMENT_FRONTMATTER_SCHEMA.md](./DOCUMENT_FRONTMATTER_SCHEMA.md)：
基础必填字段（`title / doc_type / layer / status / updated_at`）、保留目录专属字段
（`index_group / retention_reason / linked_contracts`）、类型与格式约束、强制覆盖范围
（schema §11）都以该文件为准，本文不再重复定义。

本文只负责 §3 `doc_type` 与 §4 `status` 两个枚举的**取值语义**。

---

## 3. 文档类型枚举

### 3.1 `Contract`

定义：

- 某个层级、模块或专题的执行契约
- 负责定义边界、目标、非目标、完成定义、验收标准

示例：

- `platform contract`
- `system/auth contract`
- `system/config contract`

### 3.2 `Design`

定义：

- 对合同的设计展开
- 负责描述结构、交互、API、状态、治理细节

示例：

- `FRONTEND_UI_SPEC.md`
- `AUTH_MODULE_DESIGN.md`

### 3.3 `Assessment`

定义：

- 对“当前现实”做盘点
- 负责描述差距、风险、成熟度、缺口矩阵

示例：

- `PLATFORM_GAP_AUDIT_20260429.md`
- `SYSTEM_MODULE_AUDIT.md`

### 3.4 `Remediation`

定义：

- 对某次问题盘点后的整改路径做规划
- 负责定义收口顺序、治理动作、迁移边界

示例：

- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`

### 3.5 `Acceptance`

定义：

- 对合同或设计的验收方式、模板、样例、矩阵和结论做沉淀

示例：

- `ACCEPTANCE_CHECKLIST.md`
- `PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`

### 3.6 `Archive`

定义：

- 已不再作为当前主依据，但保留历史参考价值的文档

示例：

- 某次 dated 冒烟样例
- 某次旧整改阶段样例

说明：

- `Archive` 不是“垃圾桶”；
- 只有在仍有复用价值时才保留；
- 被完全覆盖且没有引用价值的文档，应直接删除，而不是滥用 `Archive`。

---

## 4. 文档状态枚举

### 4.1 `Draft`

含义：

- 还在起草，不能作为正式依据

规则：

- 默认不进入主索引一线入口
- 不能作为验收通过依据

### 4.2 `Active`

含义：

- 当前正式生效
- 可以作为实现、评估、整改、验收依据

规则：

- 可进入主索引
- 后续新增相关文档应优先引用它

### 4.3 `Superseded`

含义：

- 已被更新文档覆盖，但暂未删除

规则：

- 不再作为当前主依据
- 文档内部应明确写出“被哪份文档替代”
- 主索引中默认降级或不再展示

### 4.4 `Archived`

含义：

- 已退出当前治理主链，但保留样例、基线或审计价值

规则：

- 不进入主索引一线入口
- 仅在历史基线、样例区或归档区出现

---

## 5. 类型与状态的关系

同一文档的“类型”和“状态”是两条独立维度。

例如：

- 一份 `Assessment` 可以是 `Active`
- 一份 `Assessment` 也可以后来变成 `Superseded`
- 一份 `Acceptance` 样例可以长期保持 `Archived`

不建议把二者混写成一句自然语言判断。

推荐思路：

```text
类型：Assessment
状态：Superseded
```

这样人和 AI 都能快速知道：

- 它本来是干什么的
- 它现在还能不能作为主依据

---

## 6. 主索引展示规则

### 6.1 应进入主索引一线入口

- `Contract` + `Active`
- `Design` + `Active`

### 6.2 可进入主索引二级入口

- `Remediation` + `Active`
- `Acceptance` + `Active`

### 6.3 默认不进入一线入口

- `Assessment` + `Active`
- 任意类型 + `Draft`
- 任意类型 + `Superseded`
- 任意类型 + `Archived`

例外：

- 某份 `Assessment` 如果仍被流程强依赖，可以进入二级入口；
- 某份 `Acceptance` 模板或长期矩阵可以进入一线或二级入口。

### 6.4 目录与索引分组映射

为避免后续再次堆回 `docs/archive/` 或 `openspec/specs/`，建议固定使用以下映射：

- `openspec/specs/`
  - `类型` 通常为 `Design`
  - `状态` 通常为 `Active` 或 `Superseded`
  - `索引分组` 固定写 `superpowers-specs`
- `docs/archive/examples/`
  - `状态` 固定写 `Archived`
  - `索引分组` 固定写 `archive/examples`
- `docs/archive/baselines/`
  - `状态` 固定写 `Archived` 或 `Superseded`
  - `索引分组` 固定写 `archive/baselines`
- `docs/archive/upgrade/`
  - `状态` 固定写 `Archived`
  - `索引分组` 固定写 `archive/upgrade`

如果一份文档无法给出合理的 `索引分组` 和 `保留原因`，默认说明它不该被保留。

### 6.5 Active 复核时效

`Active` 的含义是"当前正式生效"，所以它同时是一句**关于现在**的断言；断言会过期。规则：

- `updated_at` 超过 **90 天**未更新 → **复核逾期**（active review overdue）。
- 逾期**不等于失效**，也不自动降级；它表示"不能仅凭 `status: Active` 就相信这份文档"。
- 处置二选一：① 人工复核内容后重写 `updated_at`；② 降为 `Draft` 或 `Superseded`。
- 复核不是改日期：只把 `updated_at` 推到今天、内容一字未读，比逾期更糟——它把"没人看过"伪装成"刚看过"。

该判据由 `scripts/frontmatter-check.mjs` 输出为**份数**（informational，不阻断构建），需要明细时用 `--report-overdue`。
为什么故意不做成 error：存量逾期文档远超一次可完成的人工复核（2026-10-31 实测 144 份，其中 71 份 `.en.md` 镜像，最老 198 天：`docs/designs/BACKEND.md` / `BACKEND.en.md`，`updated_at` 2026-04-17）；做成 error 后唯一可行的过门方式是批量改 `updated_at`，恰好摧毁本条想保护的真实复核。判据是**可见**，不是**变绿**。

口径说明：审计报告 §11 记"102 份 Active / 60 天未更新"，与本节实测（144 份 / 90 天窗口）口径不同，本次未复现审计口径，以实测为准。

读数变化（2026-11-01 复核）：144 → **143**。差的一份是 `docs/designs/MODULE_CONTRACT.md`——W11 因 §4.3 真实内容变更把它的 `updated_at` 推到 2026-11-01，该文档退出逾期集合。这正是本节期望的处置方式（复核内容后重写日期），不是批量改日期。

---

## 7. 第一轮落地要求

从本轮开始：

1. 新增的合同文档必须补 `类型 / 归属层 / 状态`
2. 新增的核心设计文档必须补 `类型 / 归属层 / 状态`
3. `README` 中列出的主文档，后续应逐步补齐元信息
4. 新的阶段评估稿如果没有明确 `类型 / 状态 / 关联合同`，不应进入主索引

---

## 8. 后续建议

建议下一轮继续推进：

1. 为首批平台级合同建立统一模板
2. 为 `README` 中的一线主文档逐步补齐元信息
3. 对仍保留的历史样例文档补 `状态：Archived`
4. 为 `openspec/specs/` 与 `docs/archive/*` 文档逐步补 `索引分组 / 保留原因`
