# 文档类型与状态说明

更新时间：2026-04-30

类型：Contract
归属层：platform
状态：Active

本文用于定义 Pantheon Base 文档系统的统一元信息字段、文档类型枚举和生命周期状态枚举。

它是 [DOCUMENT_GOVERNANCE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md) 的配套规则文档。

---

## 1. 目标

如果只定义“文档类型”，不定义“文档状态”，项目很快又会回到旧问题：

- 文档知道自己属于 `Design` 还是 `Assessment`，但不知道自己是不是已经过期；
- 主索引里混着 `Draft`、`Active`、`Superseded` 文档；
- AI 和新人看不出当前应该信哪份；
- 旧评估稿虽然还在，但没有任何退场信号。

因此 Pantheon 后续所有主文档都应补齐统一元信息。

---

## 2. 统一元信息字段

建议每份主文档头部至少包含以下字段：

- `更新时间`
- `类型`
- `归属层`
- `状态`

建议重点文档再增加：

- `Owner`
- `Last Reviewed`
- `关联合同`

推荐格式：

```text
更新时间：2026-04-30

类型：Design
归属层：system/config
状态：Active
关联合同：SYSTEM_CONFIG_CONTRACT.md
```

说明：

- 不要求第一轮为所有历史文档一次性补齐全部扩展字段；
- 但从现在开始新增或重写的主文档，应至少补 `类型 / 归属层 / 状态`。

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
