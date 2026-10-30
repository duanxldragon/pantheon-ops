## 变更摘要

- 改动层级：business/*（`backend/modules/business/filedist` 仅测试文件）+ 同一任务包的治理产物（manifest / evidence / 任务包）
- 改动模块：`filedist/storage_service_test.go` 新增 `TestUploadPathStaysInsideRootForAdversarialNames`；`docs/harness/tasks/2026-10-30-sonar-findings-remediation.task.md` 的 Post-Merge Read-Back；`.harness/tasks/.../manifest.json`；`.harness/evidence/.../{commands.json,summary.md,sonar-after-merge-vulnerabilities.json}`
- 目标问题：PR #156 合并后 `main` 的 SonarCloud 门禁只剩一个 ERROR 条件 `new_security_rating`，其全部残量是 36 条 BLOCKER vulnerability：33 条 `gosecurity:S3649` 与 3 条 `gosecurity:S2083`（`UploadFile` 里三处 `os.Remove(fullPath)` 清理调用）。代码已把路径收敛在存储根内，SonarGo 本轮也接受该收敛对 `os.Create` 是净化子，但不认同一值到达 `os.Remove` 时仍被净化。上一轮承诺不靠降噪过关，因此本 PR 先把「误报」的依据做成一条与 Sonar 报告同形的测试，再进入流转。
- 预期影响：不改生产代码、不改运行时行为、不改任何指标或门禁定义；把 3 条 `S2083` 的依据从口头论证升级为 CI 必跑的断言，并把合并后的真实读数（开放问题 242→194、vulnerability 62→36、bug 6→0、`new_reliability_rating` 转 OK）与逐条 live key 存档，使后续流转可逐条回退。

## Harness 链路

- Task ID：2026-10-30-sonar-findings-remediation
- Task Manifest：.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json
- Evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json
- Verification evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/summary.md
- Review Artifact：.harness/evidence/2026-10-30-sonar-findings-remediation/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：auth-security
- Ratchet Decision：gate-updated
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-30-sonar-findings-remediation
- task manifest: `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- evidence: `.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json`
- boundaries: base 拥有共享面；本 PR 只改 ops 自有 `backend/modules/business/filedist` 测试文件与该任务包的 manifest/evidence，不改 base 文件、不改 overlay 声明、不改 lock、不改 Sonar 门禁定义与指标
- backend response contract: not-applicable（无响应结构变更）
- backend DTO contract: not-applicable（无 DTO 变更）
- permission contract: not-applicable（未新增路由或权限键）
- audit coverage: not-applicable（无审计链变更）
- visual evidence: not-applicable（无前端改动；`check:ui-quality-gate` 沿用 PR #156 的 0 finding 结论）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变，仍为 `pantheon-base-v0.18.1`
- base drift: `test:quality-workflow` 存在 4 个先前即失败的用例（断言 base 版 `quality.yml` 的 paths-filter / governance-only 形状），本 PR 不修，已记入残留风险
- Base/ops inheritance: not-applicable（filedist 属 ops 业务模块，base 不含 `backend/modules/business`，该测试不需要下沉）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = 未变化（本 PR 不推进 lock）
- generic drift: not-applicable
- pseudo-drift: not-applicable
- business-only: yes（仅 `business/filedist` 测试文件 + 该任务包治理产物）

## 边界说明

- [x] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 主体只落在 `backend/modules/business/filedist` 的一个测试文件，其余是同一任务包的 manifest 与 evidence 更新。不改生产代码、不改 `.github/workflows/*`、不改 Sonar 配置或门禁定义，因此无需 base-first round trip，也不触发 overlay 变更。

## 方案对齐

- 契约影响：no；接口/路由/权限/i18n/菜单/DB/seed 均未变化。
- 实现方式：`TestUploadPathStaysInsideRootForAdversarialNames` 复刻 Sonar 标记的传播路径（请求文件名 → `sanitizeUploadFileName` → `filepath.Join` → `resolveWithinRoot`），断言「要么被拒绝，要么解析结果仍在存储根内」；7 个敌对输入含 `../../etc/passwd`、Windows 反斜杠路径、绝对路径、`....//` 变形、嵌套穿越、内嵌 NUL、CRLF 名称。被净化器拒绝的输入按「同样是安全结果」处理，因此测试断言的是性质而非虚构的期望行为。
- 为什么这样实现：能诚实支撑「误报」结论的只有把该数据流钉住的测试；单靠代码修不掉这 3 条（SonarGo 的净化识别只覆盖了 create 调用点）。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `npm run check:gorm-sql-literal`
- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：本轮实际执行集（逐条结果见 `commands.json`）——`cd backend && gofmt -l modules/business`（0 文件）+ `go build ./...` + `go vet ./...` + `go test ./... -count=1`（exit 0，64 包 ok）；新增测试单独跑：`--- PASS: TestUploadPathStaysInsideRootForAdversarialNames`，与既有 `TestResolveWithinRootKeepsPathsInsideStorage`、`TestSanitizeUploadFileNameReducesNameToOneElement` 同时通过；CI 同款新代码模式 `golangci-lint run --new-from-rev=95220488 ./...` → **0 issues**；治理集 `check:task-packet`（0 errors 0 warnings）/`check:harness-adoption --strict`（0 findings）/`check:encoding --strict`（0/2105）/`check:docs-frontmatter`（435 docs）/`check:structure`（0/2279）/`check:duplication`（PASS）/`check:gorm-sql-literal`（绿，4 条 waiver 可见）全绿。前端与生产后端代码未改动，PR #156 的 lint/type-check/build 结论仍成立。Runtime gap：本轮不改运行时行为，无需 UI 渲染证据，证据为 Go 单测与托管侧 Sonar 读数。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：ran（PR #156 中 `CodeQL Security Scan` 与 `CodeQL Alert Gate` 均 SUCCESS；本 PR 只新增测试，合并前以托管结论为准）
- GitHub checks 结果：pending（合并前要求 required contexts `Quality Gates` / `Security Gates` 全绿；`main` 的 `strict=false`，非必需的 `Vercel` 状态不阻塞）
- Auto-merge：enabled
- Duplication Gate 结果：ran（`node scripts/check-duplication.mjs` PASS；新测试是表驱动的输入集合，未复制既有测试体）
- 是否高风险改动：no（不改生产代码、不改门禁与指标、不改工作流）
- Ratchet 说明：把「误报主张必须有可机器核验依据」这条约束实际落到一条钉住数据流的测试上；同类 `S2083` 以后不再靠口头解释。信号来自托管侧 `new_security_rating` 单条件 ERROR 的回读，而非本地主观判断。
- Residual risk / follow-up：① 本 PR 只做证据。33 条 `gosecurity:S3649` 与 3 条 `gosecurity:S2083` 的 Sonar 流转将在其进入 `main` 后执行，逐条 live key 已存 `sonar-after-merge-vulnerabilities.json`，理由文本分别指向 `scripts/check-gorm-sql-literal.mjs` 与本轮新增测试，回退方式为逐条 `to_reopen`，结果将写入 `sonar-triage.json`。② 若 SonarGo 后续版本改变净化识别，这 3 条可能重新出现，届时以同一测试为依据重估。③ 历史 code smell 积压本轮不动（`go:S1192`×68、`go:S3776`×27、`typescript:S3358`×20、`typescript:S3863`×10、`typescript:S6571`×6、`typescript:S3776`×6、`godre:*`×10、`go:S1135`×2、`go:S107`×2），属独立可维护性任务包。④ Dependabot 触发的运行读不到 `SONAR_TOKEN`，该类 PR 上 Sonar 仍为红，属维护者门禁策略。⑤ `test:quality-workflow` 4 个既有失败源于 ops `quality.yml` 缺 base 的 governance-only / paths-filter 形状。⑥ `sonar.sources=.` 使扫描器把 PNG evidence 当源码文本读。⑦ Vercel 检查项按维护者指示已移出范围：外部 GitHub App 集成、仓库内无可改配置、非 `main` 的 required context，停用需在 GitHub App 或 Vercel 控制台操作。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理（本轮无共享面改动，base 无 `backend/modules/business`）
- [x] 前端新增展示文案已使用 i18n（无前端改动）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（均未触及；治理产物已同步）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 改动，gap 已显式说明）
- [x] 已确认不会泄露敏感配置、账号密码或 Token（Sonar token 只作 shell 变量使用，evidence 仅存 issue key 与规则名）
