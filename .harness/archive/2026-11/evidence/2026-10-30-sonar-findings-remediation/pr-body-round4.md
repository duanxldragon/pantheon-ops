## 变更摘要

- 改动层级：治理产物（`.harness/evidence/**`、`.harness/tasks/**`、`docs/harness/tasks/**`）；不改 `business/*` 代码、不改 base、不改工作流
- 改动模块：新增 `.harness/evidence/2026-10-30-sonar-findings-remediation/sonar-triage.json`；同任务包的 `commands.json`（+4 条 round-4 记录）、`summary.md`（新增「Recorded false positives, and the check that carries them」并回写历史积压计数与 coverage 缺口）、`review.md`（新增 CI round 3 与 triage 决策）、`docs/harness/tasks/2026-10-30-sonar-findings-remediation.task.md`（新增 Recorded False Positives (round 4) 与验证计划补项）、`.harness/tasks/.../manifest.json`（round-4 说明、期望文件、验证计划、陈旧积压计数回写）
- 目标问题：任务 #27 的收尾——在修复进入 `main` 且新一轮分析落地之后，把已机器核验的 `gosecurity:S3649`（33 条）与 `gosecurity:S2083`（3 条）以「记录理由 + 存档 issue key」的方式流转，并回读门禁是否脱离 ERROR。此前只完成代码修复与读数存档，门禁仍因 `new_security_rating` 单条件 ERROR。
- 预期影响：把 36 条 BLOCKER vulnerability 置为 `RESOLVED / FALSE-POSITIVE`，`main` 分支门禁回读为 **OK**（含 `new_security_rating`），开放 vulnerability 36→0、bug 保持 0、待复审 hotspot 0。不动任何指标、new-code 基线或门禁定义，不改生产代码；每条 issue key 均存档，可逐条 `to_reopen` 回退。

## Harness 链路

- Task ID：2026-10-30-sonar-findings-remediation
- Task Manifest：.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json
- Evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json
- Verification evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/summary.md
- Review Artifact：.harness/evidence/2026-10-30-sonar-findings-remediation/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：auth-security
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-30-sonar-findings-remediation
- task manifest: `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- evidence: `.harness/evidence/2026-10-30-sonar-findings-remediation/sonar-triage.json`
- boundaries: base 拥有共享面；本 PR 只改该任务包的 manifest/evidence 与任务包文档，不改生产代码、不改 `.github/workflows/*`、不改 Sonar 门禁定义与指标、不改 overlay 声明与 lock
- backend response contract: not-applicable（无响应结构变更）
- backend DTO contract: not-applicable（无 DTO 变更）
- permission contract: not-applicable（未新增路由或权限键）
- audit coverage: not-applicable（无审计链变更；对外部分析平台的状态流转本身按 issue key 逐条留痕，可回退）
- visual evidence: not-applicable（无前端改动）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变，仍为 `pantheon-base-v0.18.1`
- base drift: `test:quality-workflow` 存在 4 个先前即失败的用例（断言 base 版 `quality.yml` 的 paths-filter / governance-only 形状），本 PR 不修，已记入残留风险
- Base/ops inheritance: not-applicable（Sonar 误报依据所依赖的 `scripts/check-gorm-sql-literal.mjs` 是 ops-only 传感器，base 不含 `backend/modules/business`）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = 未变化（本 PR 不推进 lock）
- generic drift: not-applicable
- pseudo-drift: not-applicable
- business-only: no（治理产物 only，代码在 PR #156 / #158 已进入 `main`）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 两个框都不勾选是准确的：本 changeset 是同一任务包的治理产物，不含任何代码或 base/overlay 变更。被流转所依赖的代码与传感器（`scripts/check-gorm-sql-literal.mjs` 及其 13 条测试、`filedist` 三条净化测试）已由 PR #156 / #158 进入 `main`，本轮只记录外部分析平台状态流转与其回读证据。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `npm run check:gorm-sql-literal`
- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：流转前在 `52dfe28`（含传感器与净化测试的 `main`）重跑——`node scripts/check-gorm-sql-literal.mjs` exit 0 且 4 条 waiver 可见打印；`node --test tests/scripts/check-gorm-sql-literal.test.mjs` 13/13；承载 18 条仍开放 sink 的包 `go test ./modules/business/{filedist,service,cmdb,cmdb/host,cmdb/group,cmdb/label,deploy}/...` 全绿；三条净化测试逐条 `--- PASS`。逐条核验：12 处为 map 键全部由同函数内引号字面量赋值的 `Updates(map)`，6 处为 GORM 以绑定参数下发的 `First(&model, id)` 主键查询，无请求值进入 SQL 文本。流转后回读：36 条 key 全为 `RESOLVED / FALSE-POSITIVE`，门禁 OK，开放 vulnerability 0、bug 0、hotspot 待复审 0，剩余 158 条全为历史 `CODE_SMELL`。本轮不改运行时行为，runtime gap 显式为「无需 UI/服务渲染证据，证据为 CI 必跑的传感器/单测与托管侧读数」。逐条命令与结果见 `commands.json`。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：not-applicable（本 PR 无代码变更；PR #156 中 `CodeQL Security Scan` 与 `CodeQL Alert Gate` 均 SUCCESS）
- GitHub checks 结果：pending（合并前要求 required contexts `Quality Gates` / `Security Gates` 全绿；`main` 的 `strict=false`）
- Auto-merge：enabled
- Duplication Gate 结果：ran（`node scripts/check-duplication.mjs` PASS；本轮只增删文档与 JSON 证据）
- 是否高风险改动：yes（对外部分析平台 36 条 BLOCKER 安全条目做状态流转，属于「可能洗白真实漏洞」的一类动作）
- Ratchet 说明：顺序本身即门禁——流转排在修复进入 `main`、传感器与 13 条测试在该 commit 重跑绿、18 条 sink 逐行读毕之后；依据文本按规则写入 `sonar-triage.json` 并指向 CI 阻断步（`quality.yml` 的 `Check/Test GORM SQL literalness`，无 `continue-on-error`）。`POST /api/issues/comment` 在 SonarCloud 侧不可用（HTTP 404 `Unknown url`），故理由以版本控制文件承载；`/api/issues/search` 忽略 `issueKeys` 且 `issueStatuses` 只接受 `OPEN,CONFIRMED,FALSE_POSITIVE,ACCEPTED,FIXED`，这两点已写入证据以免后续读者把「查不到」误读为「未流转」。
- Residual risk / follow-up：① 回退路径为逐条 `POST /api/issues/do_transition transition=to_reopen`，全部 key 已存档；未改任何指标、new-code 基线或门禁定义。② 若 SonarGo 后续版本改变净化/字面量识别，这 36 条可能重新出现，届时以同一传感器与同一组测试为依据重估。③ `new_coverage` 显示 OK 仅因未上传覆盖率报告，该条件实际未被评估，不作为通过证据声称。④ 历史 `CODE_SMELL` 积压 158 条本轮不动（`go:S1192`×68、`go:S3776`×27、`typescript:S3358`×20、`typescript:S3863`×10、`godre:*`×10、`typescript:S3776`×6、`typescript:S6571`×6、`typescript:S6551`×3、`go:S107`×2、`go:S1135`×2、`githubactions:S1192`×2、`typescript:S7770`×1、`typescript:S7755`×1），属独立可维护性任务包。⑤ Dependabot 触发的运行读不到 `SONAR_TOKEN`，属维护者门禁策略。⑥ `test:quality-workflow` 4 个既有失败源于 ops `quality.yml` 缺 base 的 governance-only / paths-filter 形状。⑦ `sonar.sources=.` 使扫描器把 PNG evidence 当源码文本读。⑧ Vercel 检查项按维护者指示已移出范围（外部 GitHub App 集成、无仓库内配置、非 required context）。⑨ 流转驱动脚本因需读取维护者 token，刻意保留在仓库之外，不进入代码树。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（本轮为治理产物，归属该任务包）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（本轮无共享面改动；依赖的传感器为 ops-only）
- [x] 前端新增展示文案已使用 i18n（无前端改动）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（均未触及；任务包 manifest、evidence、PR 模板留痕已同步）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 改动，gap 已显式说明）
- [x] 已确认不会泄露敏感配置、账号密码或 Token（Sonar token 仅作 shell 变量使用，`sonar-triage.json` 内已核验不含 token，只存 issue key、规则名与文件行号）
