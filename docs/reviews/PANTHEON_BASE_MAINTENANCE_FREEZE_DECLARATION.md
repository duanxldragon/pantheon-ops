---
title: Pantheon Base 维护冻结宣告（待签）
doc_type: Assessment
layer: platform
status: Draft
updated_at: 2026-11-05
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
---

# Pantheon Base 维护冻结宣告

English note: this file is the **text of the declaration**, prepared with the facts and
readings it depends on. It is not the declaration itself — the act of declaring belongs
to the maintainer (`programme` Human Gates: `FREEZE-DECLARATION`). Until §7 is signed and
the frozen commit SHA is filled in, this document records an *intent*, not a state.

**状态：未签署。** 未签署前，本文件不具备任何约束力。

---

## 当前执行状态（2026-11-05，优先于下文历史读数）

维护者已授权完成现有功能、保留租户、提交/合并/发行与 Ops 消费；执行任务为 [2026-11-05-base-freeze-release](../harness/tasks/2026-11-05-base-freeze-release.task.md)。11 月 1 日以下各节读数保留作为准备历史，不能作为当前验收结果。

- 原 10 月 31 日整改已合并至本地 main `525506035b3a843f85b984beb46866c1329cd964`（#488），旧审计分支的树与 main 一致。
- 当前候选分支 `feat/base-freeze-2026-11-05`，开发版本 `0.18.3`；Nov 3/4 现有修复纳入候选，本轮证据见 [summary](../../.harness/evidence/2026-11-05-base-freeze-release/summary.md)。最后正式发布仍为 v0.18.2。
- 租户 V1 全部保留；现有功能、真实 DB/Redis、核心/租户/生成器烟测与独立评审是冻结验收范围。
- 独立模型服务返回 503，GitHub API/CLI 网络被执行环境拒绝；没有 hosted CI、远程合并与正式发行证据，因此未签冻结，未更新 Ops 正式锁。
## 1. 冻结对象（2026-11-01 历史准备）

冻结必须指向**一个提交**，否则"冻结"没有主语。当前情况：

| 项 | 实测值（2026-11-01） |
|----|----------------------|
| 工作区 HEAD | `ef0073860ce03087d9cf05de6b5d7732687ef769` |
| 分支 | `fix/release-gate-checks-poll-resilience` |
| `VERSION` | `0.18.1` |
| 工作区状态 | **脏**：150 modified / 9 deleted / 4 renamed / 3 renamed+modified / 49 untracked |
| 整改产物是否已进 main | **否**——9 个子包的代码、证据、任务包均在工作区内 |

**因此今天不具备签署条件，且原因不是流程性的。**

冻结的含义是"此后变化的原因从'加能力'变成'修问题'"。而本次整改（生成器可编译、业务域租户语义、
入口文档纠错、文档归档、元数据规范、死代码与死表、token 缓存竞争、CI checker 接线）**尚未合入
`main`**——HEAD 上跑的仍是整改前的代码。在整改产物进入 `main` 之前签署，等于宣告冻结一份
并不包含整改的代码。

签署时必须回填：

- 冻结对象 SHA：`________________`（应为整改合并后的 `main` 提交）
- 该提交上的 `VERSION`：`________`
- 该提交上的 schema 版本：`________`（当前工作区迁移链止于 `000031`，但 `000031` 仍是
  untracked 文件，未进任何提交）

---

## 2. 判据：7 项冻结门槛

门槛定义见 `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md` §13。逐项对应与当前状态：

| # | 门槛 | 对应任务包 | 状态 | 证据位置 |
|---|------|-----------|------|----------|
| 1 | 生成器"生成 → `go build` → 建表"往返证据（AUD-001） | `2026-10-31-audit-p1-lowcode-generator-build` | 达成 | `.harness/evidence/2026-10-31-audit-p1-lowcode-generator-build/` |
| 2 | 业务域租户语义有结论并落到合同 + README（AUD-002） | `2026-10-31-audit-p1-business-route-tenant-context` | 达成（方案 C：V1 内不参与隔离） | 同上 `.../2026-10-31-audit-p1-business-route-tenant-context/` |
| 3 | 入口文档可执行性与失实修正（AUD-003/004/007/016/024） | `2026-10-31-audit-p1-entry-doc-truthfulness` | 达成 | `.../2026-10-31-audit-p1-entry-doc-truthfulness/` |
| 4 | 文档归档轮完成（AUD-006/011/017/018/019） | `2026-10-31-audit-p2-doc-lifecycle-archive` | 达成（92 份中 69 份转 `Archived`） | `.../2026-10-31-audit-p2-doc-lifecycle-archive/` |
| 5 | 元数据规范三合一 + scope 矩阵 + 门禁边界（AUD-008/009/010） | `2026-10-31-audit-p2-doc-metadata-and-index` | 达成 | `.../2026-10-31-audit-p2-doc-metadata-and-index/` |
| 6 | 死代码/死表/legacy 端点处理完毕（AUD-012/013/014） | `2026-10-31-audit-p2-dead-code-legacy-surface` | 达成，**生产侧执行属 human gate**（见 §4 的 `AUD-013-prod` / `AUD-014-followup`） | `.../2026-10-31-audit-p2-dead-code-legacy-surface/` |
| 7 | token 缓存竞争修复 + CI 去重 + checker 接线（AUD-026/015/027） | `2026-10-31-audit-p2-token-cache-race`、`2026-10-31-audit-p2-ci-checker-wiring` | 达成（2 × DATA RACE → 0） | 两个目录 |

门槛 6 需要一句说明：审计要求的是"处理完毕"，而"处理"的完成态是**代码与文档侧完成 + 生产执行
单独立项**。`000031` 的 DROP 只在本地一次性库验证过，legacy 端点的下线日期是 2026-12-31 的对外
承诺。两者都是已登记的 P2 残留，不是门槛未达成。

OIDC（AUD-005）不列入门槛，理由见 §13 注：不可达代码、无运行时风险，作为"冻结后接受的 gap"
显式登记即可。其决策记录为 W9（删除 4 文件 / 587 行，保留迁移与列）。

---

## 3. 今日实测读数（2026-11-01）

以下为本文撰写时**实际执行**的读数，不是引用既有记录（文档计数含本包新增的文档，签署时应重跑一次）：

| 检查 | 读数 |
|------|------|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `gofmt -l pkg/database/` | 空 |
| `go test -short ./...` | 见 §3.1 |
| `tsc -b`（frontend） | exit 0 |
| `check:search-toolbar-contract` | 0 findings / **106** 个模块文件 |
| `check:docs-frontmatter` | 421 docs / 306 带元数据 / 115 不强制 / **143** 份 Active 超 90 天（只报告） |
| `check:doc-links --strict` | 0 findings |
| `check:doc-inventory --strict` | 0 findings |
| `check-task-packet` | **94** 个任务包 / 0 error / 14 warning（warning 全为历史任务包既存项） |
| `check-evidence --strict` | **15** 份 / 0 error |
| `check-review --strict` | **15** 份 / 0 error |
| `check-encoding --strict` | 0 findings / 2609 文件 |

### 3.1 `go test -short ./...`

```
EXIT=0    ok=47    FAIL=0    no test files=5
```

带真实 MySQL/Redis（仓库根 `.env.test` 的 `PANTHEON_DSN` / `PANTHEON_REDIS_PASSWORD`）。
其中 `pkg/database` 单包 110s——迁移链的往返用例是真库执行，不是 mock。

未覆盖的两类读数，签署时若需要应另行补：

- **`-race`**：本机需 `export PATH="/d/msys64/mingw64/bin:$PATH"`，本轮未跑；
  竞争修复（门槛 7）的 `-race` 读数在 W7 证据里，不在本次读数中。
- **多实例压测**：未做，即 §4 的 `AUD-026-stress`。

---

## 4. 残留清单（11 条，均不阻断冻结）

冻结不是"问题清零"，是"剩余问题已知、已定级、已归属"。当前 open 项：

| ID | 级别 | 内容 | 归属 | 为什么不阻断 |
|----|------|------|------|--------------|
| `FREEZE-DECLARATION` | P2 | 本文件本身 | 维护者 | 即本动作 |
| `AUD-013-prod` | P2 | `000031` 的 DROP 未在生产执行 | 维护者 | 执行单已备好：`docs/runbooks/MIGRATION_000031_PRODUCTION_DROP.md` |
| `AUD-014-followup` | P2 | legacy 认证端点真正下线（2026-12-31） | 维护者 | 执行单已备好：`docs/runbooks/DECOMMISSION_LEGACY_AUTH_ROUTES_2026-12-31.md` |
| `AUD-001b` | P3 | 生成物 `gofmt` 不干净（模板写死对齐空格） | 待办 | lint-staged 提交时归位，CI 对 PR 仅报告，不影响构建 |
| `AUD-001c-ddl` | P3 | 生成表 DDL 不在迁移链里，无 down 迁移 | 待重估 | 需要与生成器设计一起重估，不是独立缺陷 |
| `AUD-021b` | P3 | `lowcode/templates` 1,894 行死代码 | 待决策 | 目录不参与构建（tsconfig exclude、零 import），删除属独立决策 |
| `AUD-025-review-round` | P3 | 143 份逾期 Active 文档需人工复核轮 | 待办 | 规则已落地且刻意不阻断（§6.5）；复核只能真读，不能脚本批量完成 |
| `AUD-026-race-only` | P3 | 竞争仅 `-race` 可见 | 观察项 | 已登记，不新增维护面 |
| `AUD-026-stress` | P3 | 多实例压测未做 | 观察项 | 同上 |
| `AUD-012-ops` | P3 | `pantheon-ops` 仍保留自有 `usePagination`/`useRequest` 副本 | 下游 | base 侧已删，下游同步属消费仓变更 |
| `AUD-001c-ops-doc` | P3 | `pantheon-ops` 部署手册可能抄了旧表述 | 下游 | 同上 |

三条 P2 的共同点是**都不是代码问题**：一条是宣告动作本身，两条是需要在真实环境/真实日期执行
的运维动作，且各自都有执行单与前置条件。把它们算作"冻结前必须做完"，等于要求维护者先在生产库
做一次破坏性 DDL、并提前 60 天撕毁一个对外承诺——那是把流程正确性排在正确性之前。

---

## 5. 冻结意味着什么

**冻结 = 变化的原因从"加能力"变为"修问题"。** 具体约束：

不做：
- 新增功能面、新模块、新路由组、新权限码、新菜单键
- 扩展架构契约的适用范围（含"顺手把某处也纳入租户隔离"这类）
- 为假设中的未来需求预埋抽象、开关、扩展点
- 新增门禁或把已有门禁的覆盖范围扩大（冻结期的门禁变更只允许修复假阳性/假阴性）

照常做：
- 缺陷修复、安全修复、依赖升级与 foundation release 补丁
- 文档勘误与已登记残留的收口
- 测试补强（补覆盖不是加能力）
- 上面 §4 中归属维护者/下游的动作

判据（遇到"这算不算新功能"时按此判断）：
> 该改动是否让系统**对外表现出新的能力**？是 → 需要解冻流程；否 → 属于冻结期允许范围。
> "对外"包括 API、权限码、菜单、i18n key、schema、以及下游 `sharedPaths` 可见的任何面。

---

## 6. 解冻条件

出现下列任一情况时，先改本文件再动代码：

1. 产品方向变化，需要新的能力面
2. 安全或合规要求变化，现有实现无法通过修补满足
3. 某项 P3 残留被重新定级为 P1/P2（此时按缺陷处理，不需要解冻，但需更新 §4）
4. 下游 `pantheon-ops` 提出 base 侧无法以补丁方式满足的需求

解冻不是"解除冻结"，是"记录一次例外"：在本文件追加一节，写明原因、范围、期限，由维护者签字。
例外结束即恢复冻结，不需要第二次签字。

---

## 7. 签署

签署即表示：签署人确认 §2 的 7 项门槛在其填写的冻结对象上达成、§3 的读数在该提交上可复现、
§4 的 11 条残留已知并接受其归属。

```
冻结对象 SHA      ：____________________________________________
VERSION           ：________________
schema 版本        ：________________
§3 读数复现于该 SHA：是 / 否（若否，说明差异：____________________）
§4 残留已逐条确认  ：是 / 否
签署人            ：________________
日期              ：________________
```

> 签署人应为维护者（`duanxldragon`）。代理不得代签——历史上有过代签被否定的先例，
> 见 `docs/runbooks/TENANT_MIGRATION_RUNBOOK.md` §8.1 审计注记。
