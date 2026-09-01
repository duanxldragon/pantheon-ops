---
title: 文档 Frontmatter Schema 约定
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-05-18
---

# 文档 Frontmatter Schema 约定

English version: [DOCUMENT_FRONTMATTER_SCHEMA.en.md](./DOCUMENT_FRONTMATTER_SCHEMA.en.md)

本文用于定义 Pantheon Base 文档系统的统一 YAML frontmatter 约定。

它补充 [DOCUMENT_GOVERNANCE_CONTRACT.md](./DOCUMENT_GOVERNANCE_CONTRACT.md) 与 [DOCUMENT_METADATA_AND_STATUS.md](./DOCUMENT_METADATA_AND_STATUS.md)，把原本主要依赖正文行文本的元信息，收敛成可被脚本直接解析的固定头部。

---

## 1. 目标

本约定解决三个问题：

- 让 AI 与脚本能稳定读取文档类型、状态、关联合同、索引分组；
- 让 `docs/superpowers/specs/` 与 `docs/archive/*` 的保留逻辑真正 machine-readable；
- 为后续补 `lint / drift check / linkage check` 提供固定输入结构。

本轮目标不是一次性改造整个 `docs/` 目录，而是先为新文档和已治理目录建立统一格式。

---

## 2. 基本格式

每份纳入治理的 Markdown 文档，头部应优先使用 YAML frontmatter：

```yaml
---
title: 平台配置治理与低代码菜单重构设计
doc_type: Design
layer: platform
depends_on_layers:
  - system/config
status: Approved
index_group: superpowers-specs
retention_reason: 作为平台导航信息架构调整与低代码工作域上提的设计锚点保留
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-13
---
```

frontmatter 后仍保留正文标题：

```md
# 平台配置治理与低代码菜单重构设计
```

这样做的原则是：

- frontmatter 负责机读；
- H1 与正文负责人读；
- 不要求通过 frontmatter 取代正文语义。

---

## 3. 字段定义

### 3.1 必填字段

- `title`
  - 文档标题
- `doc_type`
  - 文档类型
- `layer`
  - 主归属层
- `status`
  - 生命周期状态
- `updated_at`
  - 更新时间，格式固定为 `YYYY-MM-DD`

### 3.2 条件必填字段

- `linked_contracts`
  - 对 `Design / Assessment / Remediation / Acceptance / specs / archive` 文档应填写
- `index_group`
  - 对 `docs/superpowers/specs/` 与 `docs/archive/*` 文档必填
- `retention_reason`
  - 对 `docs/superpowers/specs/` 与 `docs/archive/*` 文档必填

### 3.3 可选字段

- `depends_on_layers`
  - 当文档是跨层设计、跨层整改或跨层验收时填写
- `superseded_by`
  - 文档状态为 `Superseded` 时建议填写
- `owner`
  - 需要长期维护归属时填写
- `last_reviewed_at`
  - 需要审阅记录时填写

---

## 4. 字段类型

### 4.1 标量字段

以下字段为单值字符串：

- `title`
- `doc_type`
- `layer`
- `status`
- `index_group`
- `retention_reason`
- `updated_at`
- `superseded_by`
- `owner`
- `last_reviewed_at`

### 4.2 数组字段

以下字段必须使用 YAML 数组：

- `depends_on_layers`
- `linked_contracts`

即使只有一个值，也建议写成数组，避免后续脚本分支判断。

---

## 5. 推荐枚举

### 5.1 `doc_type`

推荐值：

- `Contract`
- `Design`
- `Assessment`
- `Remediation`
- `Acceptance`

说明：

- 当前仓库里仍存在 `Audit`、`Baseline`、`Design / Pattern` 等历史写法；
- 本轮不强制一次性清理全部历史文档；
- 但新增或重写文档应优先收敛到上述主枚举。

### 5.2 `status`

推荐值：

- `Draft`
- `Active`
- `Approved`
- `Superseded`
- `Archived`

说明：

- `Approved` 主要用于仍处于设计锚点角色、但未直接并入正式主设计目录的 `docs/superpowers/specs/`；
- 长期看可再评估是否与 `Active` 合并，但本轮先保留兼容。

### 5.3 `index_group`

当前固定值：

- `superpowers-specs`
- `archive/examples`
- `archive/baselines`
- `archive/upgrade`

后续若扩展到其他目录，再新增枚举，不要先预留大量空值。

---

## 6. 目录映射规则

### 6.1 `docs/superpowers/specs/`

- `doc_type` 通常为 `Design`
- `status` 通常为 `Approved` 或 `Superseded`
- `index_group` 固定为 `superpowers-specs`
- `retention_reason` 必填
- `linked_contracts` 必填

### 6.2 `docs/archive/examples/`

- `status` 固定为 `Archived`
- `index_group` 固定为 `archive/examples`
- `retention_reason` 必填
- `linked_contracts` 必填

### 6.3 `docs/archive/baselines/`

- `status` 通常为 `Archived` 或 `Superseded`
- `index_group` 固定为 `archive/baselines`
- `retention_reason` 必填
- `linked_contracts` 必填

### 6.4 `docs/archive/upgrade/`

- `status` 固定为 `Archived`
- `index_group` 固定为 `archive/upgrade`
- `retention_reason` 必填
- `linked_contracts` 必填

---

## 7. 兼容与迁移策略

采用渐进迁移，不做全量硬切换。

### 第一阶段

- 新增文档优先使用 YAML frontmatter
- `docs/superpowers/specs/` 与 `docs/archive/*` 率先迁移

### 第二阶段

- `contracts/`
- `designs/`
- `acceptances/`
- `remediations/`
- `assessments/`

逐步迁移到 frontmatter。

### 第三阶段

- 再决定是否移除正文顶部的中文元信息行写法；
- 在脚本完全接管前，不要求强行删掉所有旧格式痕迹。

---

## 8. 示例

### 8.1 `specs` 示例

```yaml
---
title: Language Session And Cross-Page Selection Design
doc_type: Design
layer: platform
depends_on_layers:
  - system/auth
  - system/config
  - system/iam
status: Approved
index_group: superpowers-specs
retention_reason: 作为语言运行时优先级和跨页批量选择规则的跨模块设计锚点保留
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-16
---
```

### 8.2 `archive/examples` 示例

```yaml
---
title: Platform 壳层 PR 描述样例
doc_type: Acceptance
layer: platform
status: Archived
index_group: archive/examples
retention_reason: 作为平台壳层改动的 PR 描述样例保留，供后续批次复用
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-30
---
```

---

## 9. 当前约束

本约定当前只解决“元信息如何机读”，暂不解决：

- 文档正文结构 schema
- 证据链字段 schema
- `change / task packet / evidence / review` 的跨文件自动校验

这些能力应在 frontmatter 稳定后，再逐步推进。

---

## 10. 当前校验命令

当前仓库已提供轻量 frontmatter 校验脚本：

```bash
npm run check:docs-frontmatter
```

如果需要同时查看“哪些文档还没有迁移到 frontmatter”，执行：

```bash
npm run check:docs-frontmatter:legacy
```

当前会检查：

- 全仓 `docs/*.md` 扫描
- 已采用 frontmatter 的文档是否满足基础字段约束
- `docs/superpowers/specs/` 与 `docs/archive/*` 是否满足目录专属约束
- `index_group` 是否与目录语义匹配
- `status` 是否符合该目录允许值
- `linked_contracts` 是否为数组且目标文件存在
- 非 `Contract` 主文档是否携带 `linked_contracts`
- `Superseded` 文档是否声明 `superseded_by`
- 输出尚未迁移、仍使用旧元信息头部的 legacy 文档清单

对应测试命令：

```bash
npm run test:docs-frontmatter
```
