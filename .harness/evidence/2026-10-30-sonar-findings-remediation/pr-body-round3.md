## 变更边界

- Primary Layer：business/*（filedist 存储路径边界的测试证据）
- 关键路径：`filedist/storage_service_test.go` 的 `TestUploadPathStaysInsideRootForAdversarialNames`
- 变更意图：为 PR #156 合并后仍留在 Sonar 门禁 `new_security_rating` 上的 3 条 `gosecurity:S2083` 建立可机器核验的理由，并记录合并后的门禁回读；不改生产代码、不改任何指标或基线
- 继承判断：base 无 `backend/modules/business`，本测试是 ops 业务域资产，不涉及底座能力，无需下沉；本轮无 lock 版本变化
- 共享触点：共享符号改动：no；涉及 UI 面：no；涉及运行时敏感面：no（只新增测试）
- 影响的模块：`backend/modules/business/filedist`（仅测试文件）；治理证据：`.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`、`.harness/evidence/2026-10-30-sonar-findings-remediation/{commands.json,summary.md,sonar-after-merge-vulnerabilities.json}`、`docs/harness/tasks/2026-10-30-sonar-findings-remediation.task.md`
- 受影响子图：`UploadFile` 的攻击者可控文件名 -> `sanitizeUploadFileName` -> `filepath.Join` -> `resolveWithinRoot` -> 文件写/删；本轮把该条链路的边界性质从「代码里看起来对」升级为「测试钉住」

## 需求澄清

- Goal：把合并后仍红的那一个门禁条件所需的事实补齐——3 条 `os.Remove` 上的 `S2083` 需要一条与 Sonar 报告的数据流同形的测试作为误报依据；同时把 242→194/bugs 6→0 的真实读数写进 evidence。
- Non-goals：不修改生产代码路径；不在测试合入前对 Sonar issue 做流转；不动 quality gate/new code 基线；不清理历史 code smell 积压。
- Acceptance Criteria：新测试在 Linux 语义下成立且断言的是最终解析结果（不是中间值）；既有 `TestResolveWithinRootKeepsPathsInsideStorage`/`TestSanitizeUploadFileNameReducesNameToOneElement` 不回退；`golangci-lint --new-from-rev=<main sha>` 对新代码 0 issues；`go test ./... -count=1` exit 0；治理门禁全绿；evidence 与 manifest 同步。
- Risk Level：medium（安全类证据，但不改运行时行为）
- 本 PR 不修哪些项：33 条 `S3649` 与 3 条 `S2083` 的 Sonar 流转结果（在测试进入 main 之后单独提交证据）；174→158 个 code smell 的历史积压；依赖漏洞告警。

## 任务包与 Harness 清单

- Task Manifest：`.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- 任务包：`docs/harness/tasks/2026-10-30-sonar-findings-remediation.task.md`
- 质量画像：auth-security
- 证据目录：`.harness/evidence/2026-10-30-sonar-findings-remediation/`
- Review 产物：`.harness/evidence/2026-10-30-sonar-findings-remediation/review.md`
- 本轮 Review 记录：post-merge read-back 写入 `summary.md` 与任务包的 `## Post-Merge Read-Back (round 3)`

## 方案对齐

- 契约影响：no；接口/路由/权限/i18n/菜单/DB/seed 均未变化，只新增一个测试函数与治理证据文件。
- 实现方式：`TestUploadPathStaysInsideRootForAdversarialNames` 复刻 Sonar 标记的传播路径（请求文件名 → 净化 → 拼接 → 解析），断言「要么被拒绝，要么解析结果在存储根内」；7 个敌对输入含 `..` 穿越、Windows 反斜杠、绝对路径、`....//` 变形、嵌套穿越、NUL、CRLF。
- 为什么这样实现：SonarGo 本轮已接受 `resolveWithinRoot` 对 `os.Create` 是净化子，但不认同一值到达 `os.Remove` 时仍被净化，因此单靠代码修不掉这 3 条；能诚实支撑「误报」结论的只有把该数据流钉住的测试，而不是文字论证。被拒绝的输入按「同样安全」处理，避免把测试写成对期望行为的虚构。

## 验证记录

- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [x] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：本轮实际执行集（逐条结果见 `commands.json`）——`cd backend && gofmt -l modules/business`（0 文件）+ `go build ./...` + `go vet ./...` + `go test ./... -count=1`（exit 0，64 包 ok）；新增测试单独跑：`--- PASS: TestUploadPathStaysInsideRootForAdversarialNames`；CI 同款新代码模式 `golangci-lint run --new-from-rev=95220488 ./...` → **0 issues**；治理集 `check:task-packet`（0 errors）/`check:encoding --strict`（0/2105）/`check:docs-frontmatter`（435 docs）/`check:structure`（0/2279）全绿。前端未改动，PR #156 的 lint/type-check/build 结论仍成立。Runtime gap：无运行态验证需求（未改生产代码路径），证据为 Go 单测与托管侧 Sonar 读数。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：ran（PR #156 中 `CodeQL Security Scan` 与 `CodeQL Alert Gate` 均 SUCCESS；本 PR 只新增测试，合并前以托管结论为准）
- GitHub checks 结果：pending（合并前要求 required contexts `Quality Gates`/`Security Gates` 全绿；`test:quality-workflow` 的 4 个既有失败属 base 漂移，不在本 PR 断言范围）
- Auto-merge：enabled
- Duplication Gate 结果：ran（`node scripts/check-duplication.mjs` PASS；新测试是表驱动的输入集合，未复制既有测试体）
- 是否高风险改动：no（不改生产代码，不改门禁与指标，不改工作流）
- Trivial change：no（安全类证据，需要评审确认论证链条而非看 diff 行数）
- Ratchet Decision：gate-updated（把「误报主张必须有可机器核验依据」这一约束实际落到一条钉住数据流的测试上；同类问题不再靠口头解释）
- GitHub Signal：repo-quality-gate（信号来自托管侧 `new_security_rating` 单条件 ERROR 的回读，而非本地主观判断）
- Residual risk / follow-up：① 本 PR 只做证据，33 条 `gosecurity:S3649` 与 3 条 `gosecurity:S2083` 的 Sonar 流转将在其进入 `main` 后执行，逐条 key 已存 `sonar-after-merge-vulnerabilities.json`，理由文本分别指向 `scripts/check-gorm-sql-literal.mjs` 与本轮新增测试，且回退方式为逐条 `to_reopen`。② 若 SonarGo 后续版本改变净化识别，这 3 条可能重新出现，届时以同一测试为依据重估。③ 历史 code smell 积压本轮不动（`go:S1192`×68、`go:S3776`×27、`typescript:S3358`×20 等），属独立可维护性任务包。④ Dependabot 触发的运行读不到 `SONAR_TOKEN`，该类 PR 上 Sonar 仍为红。⑤ `test:quality-workflow` 4 个既有失败属 ops `quality.yml` 缺 base 的 governance-only/paths-filter 形状。⑥ `sonar.sources=.` 使 PNG evidence 被当源码文本读。

## 检查清单

- [x] 需求、范围、验收标准已写入任务包
- [x] 分层、关键路径、风险级别已明确
- [x] 修改前已读取相关 contract anchors
- [x] 涉及共享符号/跨仓行为时，已判断 Base-first 或后续同步
- [x] 测试先行：已确认现有测试能否钉住行为
- [x] 单测和静态扫描已跑（`go test ./... -count=1` 64 包 ok、CI 同款 `golangci-lint --new-from-rev` 0 issues）
- [x] 类型和导入闭环已跑（前端未改动，PR #156 结论仍成立；`gofmt`/`go vet` 干净）
- [x] 端到端冒烟或 runtime gap 已说明（未改运行时行为，证据为单测与托管读数）
- [x] 同类问题检查已跑（`scripts/check-gorm-sql-literal.mjs` 绿；`gosec` 同类路径调用点已逐一核对并带 `#nosec G304` 行内依据）
- [x] 文档、测试、菜单、权限、i18n 已同步或显式说明无需同步
- [x] Evidence 已落盘，PR 描述包含验证记录与剩余风险
- [x] 当前提交只包含本任务边界内的文件（测试文件 + 该任务包的 manifest/evidence，无交叉）
