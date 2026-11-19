## Summary

**Architectural Status: `WATCH`；最终建议 `COMMENT`。** 本轮 locale 回迁未发现架构阻断：718 个缺失 key/locale pairs 已进入现有业务 JSON，10 个差异与原 fallback 对齐，五份 fallback 均能按现有生成器在内存中逐字节重建。三种语言的运行时接线、既有重复 key 所有权及最终候选验证仍须保留为明确边界。

以下是 `continuation-locale-architecture.md` 的独立报告正文；本 lane 受只读角色约束未写入报告文件。上一轮 WATCH 报告已原样返回，保存时不得覆盖既有证据。

## Analysis

### 回迁保持现有业务所有权

`business-overlay.json:39` 已声明整个 `frontend/src/modules/business` 为业务 overlay。迁移器优先保留既有 key 所在文件；缺失 key 按业务前缀落入既有模块，剩余落入 `v12locales`（`.local/manual-preview-20261116/reconcile-business-locales.mjs:8`、`:38`）。

`v12locales/README.md:3` 原本就定义该目录为 V1.2 业务 key 汇总，目的正是重现 generated fallback。此次沿用现有链条，没有新增共享 i18n 规范或运行时 loader 副本。

### 收据、原件和迁移后内容相符

独立只读校验重算 preflight 与 applied receipt，并核对当前 source 和 `continuation-locale-source-originals/`：

| Locale | 原 canonical 缺失 | 原 canonical/fallback 差异 | 当前生成 key 数 |
| ------ | ----------------: | -------------------------: | --------------: |
| zh-CN  |               196 |                          6 |            1669 |
| en-US  |               201 |                          4 |            1669 |
| ja-JP  |               107 |                          0 |             176 |
| ko-KR  |               107 |                          0 |             176 |
| fr-FR  |               107 |                          0 |             176 |
| 合计   |           **718** |                     **10** |               — |

收据分组见 `continuation-business-locale-reconciliation.json:6`、`:279`、`:552`、`:685`、`:818`。

校验结果：

- 15 份 canonical JSON 的 after SHA256 全部匹配，其中 6 份为新增文件。
- 9 份已有 canonical JSON 和 5 份 fallback 的原件备份，共 **14 份 before SHA256** 全部匹配。
- 五份 fallback 的 after SHA256 全部匹配。
- 未删除已有 canonical key，未发现清单外的语义修改。
- 五种语言原 fallback 的全部 key/value 均保留；三种非中英 fallback 的前后字节完全相同。

迁移器在写入前比较计划原件、以 `wx` 保存备份，最后以 `wx` 写收据（脚本 `:73`、`:90`）。这是已完成并有备份的单次迁移，不能由此宣称具备崩溃原子性或通用恢复协议。

### 五份 fallback 具备当前生成链的字节等价性

正式生成器递归读取业务 JSON 并 `Object.assign`，按 locale 输出固定模板（`scripts/business-overlay/rebuild-from-base.mjs:362`）。迁移器采用同样的收集和模板结构（迁移脚本 `:53`），在 canonical 文件写入后再次生成（`:85`）。

本 lane 按已打开生成器的逻辑仅在内存中重建，五份输出均与当前 fallback 字节相同。此证据证明**当前源内容的生成等价性**；新的官方完整重建及 target report 身份尚未在本 lane 验证，旧 target 不代表最终 locale 候选。

### WATCH：三种语言的生成文件不等于实际运行时覆盖

生成器只向 zh-CN/en-US 顶层 fallback 注入业务资源（`rebuild-from-base.mjs:388`）。当前 ja-JP/ko-KR/fr-FR 顶层 fallback 继承 en-US（各语言文件 `:1`）；运行时再合并对应 overrides 和 `resources/generated`（`frontend/src/i18n/index.ts:51`），后者三份当前为空（例如 `resources/generated/ja-JP.ts:3`）。

例如 `business.bizscope.permission.hostView` 的日文现在可在 canonical JSON 与 business fallback 中找到（`bizscope/locales/ja-JP.json:11`、`resources/business/ja-JP.ts:11`），但这不能证明运行时加载该 business fallback。

这是既有接线边界，本轮回迁未扩大它。不得据本次生成等价性关闭三语言运行时覆盖问题。

### WATCH：既有异值重复 key 仍依赖遍历覆盖顺序

回迁前后重复状态不变：zh-CN 有 602 次重复写入，其中 9 个异值；en-US 为 597/11；其余三语言为 0。未新增遮蔽。

例如同一个 `stepConditionKey` 在 `deploy/locales/zh-CN.json:126` 为“参数名”，在 `v12locales/zh-CN.json:762` 为“条件变量”。生成器递归遍历没有显式排序，并以最后一次 `Object.assign` 为准（`rebuild-from-base.mjs:370`）。

本次保留原 fallback 值，因此没有已证实的显示回归；单一 owner 和跨平台生成稳定性仍属既有治理缺口。

## Root Cause

业务 key 和文案曾直接维护在 generated fallback，canonical module JSON 没有同步。正式重建只读取 canonical JSON，因此会丢失手工生成产物中的 key，或恢复旧文案。

## Recommendations

1. **低工作量：** 对当前 locale 和已纠正文档重新生成官方候选，绑定 lock、report 及 source/target 哈希；不能沿用旧 target 验收。
2. **中工作量：** 等待当前 lint/build/nav/serial race 的实际结果，保留旧失败，不将内存等价性替代运行时验收。
3. **后续治理：** 单独跟踪三语言业务 fallback 接线和异值重复 key 所有权；按 Base-first 边界处理共享 loader，勿在此次回迁中扩展架构。
4. **低工作量：** 保留 preflight、applied receipt 和原件备份；不重复执行单次脚本或覆盖历史收据。

## Trade-offs

| 选择                           | 收益                           | 代价                                  |
| ------------------------------ | ------------------------------ | ------------------------------------- |
| 以当前 fallback 为迁移取值依据 | 保持已有显示文案，恢复重建能力 | 保留既有重复 key 和部分汇总目录所有权 |
| 先完成 canonical 回迁          | 修复正式重建丢 key 的直接原因  | 仍需独立验证 runtime 接线与最终候选   |

**停止条件：** 本轮迁移内容、原件和生成等价性已独立校验；最终官方候选及运行时门禁尚未返回，本报告不提供项目完成、生产/D4 或可合并结论。
