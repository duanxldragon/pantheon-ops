# 治理面残留项决策（2026-10-27）

**决策日期**：2026-10-27
**决策对象**：简化审计（[REDUNDANT_CODE_CANDIDATES.md](REDUNDANT_CODE_CANDIDATES.md)）中五项无法由实现波裁决的治理面残留：RC-036、RC-037、RC-038、RC-031、RC-043
**任务包**：`.harness/tasks/2026-10-27-s4-governance-decisions/manifest.json`（S4，父程序 2026-10-27-codebase-simplification-program）
**性质**：决策文档。本波**不删除任何文件**；每个 delete 决策转为后续任务包；keep-and-check 命名具体检查器与运行位置。

---

## 0. 决策表（签名）

| 项 | 决策 | 理由（一段） | 成本 | 后续 |
|---|---|---|---|---|
| RC-036（95 份 `docs/**/*.en.md` 双语副本，机械同步检查缺失） | **keep-and-check** | 双语副本是面向用户的交付物（contract/acceptance 等关键文档的 EN 版本），「删除优先于归档」规则针对的是无复用价值的一次性文档，不适用；但存在性检查不比较内容，只会加噪声不加安全——检查器必须比内容。以**结构等价**（标题骨架 + 代码围栏序列 + 链接目标）为可比口径，不要求措辞逐句一致。 | 一个检查脚本 + 接入 Docs Governance 的 CI 审批（新门禁需维护者批准） | 后续包 `2026-10-27-docs-bilingual-content-check`（立项时创建） |
| RC-037（7 份 LLM 评审转写未被 docs 索引引用） | **keep-and-check（本波即补索引）** | 转写是历史评审的审计凭证（20260910 与六份多模型评审），有复查价值，不删；「未被任何文档引用」是索引缺口而非文档缺陷。本波在 `docs/README.md` §4.1 补 7 条索引（keep 的文档才更新索引，符合包范围）。 | ~7 行索引 + doc-links/doc-inventory 门禁复跑 | 无（索引即闭合） |
| RC-038（`.harness/archive` 659/1193 个 harness 文件） | **keep-as-is** | archive 是 ratchet 方法论的历史证据层（closeout/证据/评审的只读存档），删档等于销毁审计链；全部为文本，体积成本可测量且可控（约 659 个文件、纯文本）。若未来确需瘦身，走独立的归档压缩/搬迁包，不在治理决策波内做。 | 无（仅存储） | 无 |
| RC-031（`frontend/src/core/router/types.ts` 的 DashboardWidget 类型族导出但仅族内消费） | **keep-as-is** | 该类型族是一套自洽的设计面（widget 基类 + 六种 widget + union + 查询预算/新鲜度/运维元数据），对应低代码生成器与工作台的规划能力；删除是不可逆的设计损失，而保留的静态成本为零（类型不进产物）。已在族内闭环、不污染外部签名。S3 或低代码程序立项时复核。 | 无 | 复核点：S3-tenant-scope-unification 或低代码程序 |
| RC-043（`backend/pkg/common/re_export.go` 18 个再导出别名） | **delete（10 个零消费者别名）→ 后续包** | 逐符号计量（2026-10-27，base backend + pantheon-ops/backend 全量 `common\.<Sym>` 限定引用）：**10/18 零限定引用**（仅 pkg/common 子包内原名使用，不经 re-export）；其余 8 个（Fail、Success、FailWithError、FailWithCode、GetRequestID、InitSecurityConfig、Response、SuccessWithStatus）被 26/22/17/2/4/1/3/1 个 base 文件与 48/44/37/5/4/1/5/1 个 ops 文件消费。零消费者的别名是死的表面，删除后如未来需要可单行恢复（keep-and-check 对不存在的消费者无意义）。本波不删。 | 后续包一个 wave：删 10 别名 + 双仓编译 + 全套件等价证明 | **后续包 `2026-10-27-s4-re-export-alias-removal`（本波创建，status=planned）** |

**决策方式**：维护者授权委派（delegated decision），对维护者审查开放；改判不溯及本波（本波零删除，改判只影响后续包）。

## 1. RC-043 计量明细（2026-10-27，分支基线 = S1 删除波）

| 别名 | base 限定引用文件数 | ops 限定引用文件数 | pkg/common 内裸名文件数 | 结论 |
|---|---|---|---|---|
| Fail | 26 | 48 | 1 | 保留 |
| Success | 22 | 44 | 1 | 保留 |
| FailWithError | 17 | 37 | 2 | 保留 |
| GetRequestID | 4 | 4 | 3 | 保留 |
| Response | 3 | 5 | 2 | 保留 |
| FailWithCode | 2 | 5 | 1 | 保留 |
| InitSecurityConfig | 1 | 1 | 1 | 保留 |
| SuccessWithStatus | 1 | 1 | 1 | 保留 |
| **SetAccessTokenCookie** | 0 | 0 | 2 | **删除**（消费方走 `commonhttp.` 直连） |
| **SetCSRFCookie** | 0 | 0 | 2 | **删除** |
| **SetRefreshTokenCookie** | 0 | 0 | 1 | **删除** |
| **ClearTokenCookies** | 0 | 0 | 2 | **删除** |
| **ValidateRequiredProductionSecret** | 0 | 0 | 2 | **删除** |
| **DefaultDevSecrets** | 0 | 0 | 1 | **删除** |
| **GenerateRandomSecret** | 0 | 0 | 1 | **删除** |
| **IsProductionEnv** | 0 | 0 | 1 | **删除** |
| **ResolveSecret** | 0 | 0 | 1 | **删除** |
| **FailWithData** | 0 | 0 | 1 | **删除** |

计量口径：`grep -rE "common\.<Sym>\b"` 于 `backend/`（剔除 re_export.go 自身）与 `../pantheon-ops/backend/`；裸名列仅说明符号在 pkg/common 子包内有原名使用（即底层函数活着），与再导出别名是否被消费无关。审计原文记 12 个无引用符号；本波实测为 10——差异说明审计后部分别名被新增代码消费（或计数口径不同），以实测为准。

## 2. keep-and-check 检查器规格（RC-036）

- **名字**：`scripts/harness/check-doc-bilingual.mjs`
- **口径**：对每对 `<name>.md` ↔ `<name>.en.md` 比较——①标题层级骨架逐行一致；②代码围栏（``` 序列与语言标注）一致；③相对链接目标集合一致。措辞不比较。
- **运行位置**：Docs Governance workflow 与 `check-doc-links.mjs` 同位（CI 审批后接线；本地可先跑）。
- **失败行为**：strict 模式列出不配对项。
- **新门禁审批**：按 AGENTS.md，新增 CI gate 必须维护者批准——故本波只登记规格，不接线。

## 3. 明确不做

- 本波删除任何文档、归档文件或源码（含 RC-043 的 10 个别名——后续包执行）。
- 未经维护者批准新增 CI gate（RC-036 的检查器只登记规格）。
- 重新讨论 `docs/harness` 与 `.harness` 的分工（README 已论证为有意设计）。
- 触碰 `frontend/src/index.ts`（父程序 out）。
