## 变更摘要

- 改动层级：`backend/modules/business/**`（43 个 Go 文件，含 1 个新增测试文件）、`frontend/src/modules/business/**`（22 个 TSX/TS 文件）、`.github/workflows/{quality,security}.yml`（聚合判定步的去重），以及同一任务包的 manifest/evidence/任务包文档
- 改动模块：deploy、k8s（backup/cluster/ingress/install/namespace/release/storage/workload + seed）、cicd（argocd client/handler/service/webhook/reconciler + seed）、observability（notification/observability/seed）、service（handler/seed/service/state/capability/module）、cmdb（capability/host/group/label）、filedist、bizscope、capability 契约与 opsinit 装配；前端 deploy/observability/k8s/service/serviceinstance/filedist/cmdb/bizscope 页面与 api 层
- 目标问题：SonarCloud `duanxldragon_pantheon-ops` 在 round-4 安全流转之后仍开放的 **158 条 CODE_SMELL**（`go:S1192`×68、`go:S3776`×27、`typescript:S3358`×20、`typescript:S3863`×10、`godre:*`×10、`typescript:S3776`×6、`typescript:S6571`×6、`typescript:S6551`×3、`go:S107`×2、`go:S1135`×2、`githubactions:S1192`×2、`typescript:S7770`×1、`typescript:S7755`×1），使 Issues 计数无法归零
- 预期影响：按规则逐条以真实重构清除——重复字面量下沉为包级 `const`（`Updates(map…)` 键刻意保留行内引号字面量以兼容 GORM 字面量传感器）、认知复杂度用卫语句与分派辅助函数下降、嵌套三元改为模块级 `if/else`、重复 import 合并、`… | string` 改为命名联合、`String(obj)` 改为 `typeof` 收窄、单方法接口改为施事后缀名（`InstanceCreator`/`StateTransitioner`/`ServiceInstanceStateApplier`/`LogWriter`）、9/8 参数函数改为单一意图结构体。两条 `go:S1135` 一真实现（ArgoCD webhook 自动同步 + 3 条测试）一真删除（reconciler 服务实例桩，因 schema 无关联）。不引入任何 `//nolint`/`NOSONAR`/`eslint-disable`/`@ts-ignore`，不弱化任何断言，不改 Sonar 指标、new-code 基线或门禁定义；安全侧保持 0 vulnerability / 0 bug / 0 hotspot

## Harness 链路

- Task ID：2026-10-30-sonar-findings-remediation
- Task Manifest：.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json
- Evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json
- Verification evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/summary.md
- Review Artifact：.harness/evidence/2026-10-30-sonar-findings-remediation/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-30-sonar-findings-remediation
- task manifest: `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- evidence: `.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json`
- boundaries: 代码改动全部落在 `business/*`（backend `modules/business`、frontend `src/modules/business`）加两个跨切面装配点（`capability` 契约名、`opsinit` 注入）与 `.github/workflows` 的去重；不改 base、不改 overlay 声明与 lock、不改 Sonar 门禁定义与指标、不改任何测试断言
- backend response contract: not-applicable（无响应结构/字段变更；`CreateApplicationPayload` 的匿名内层结构改为同名同 JSON tag 的命名类型，序列化字节一致）
- backend DTO contract: not-applicable（无 DTO 字段变更；`godre:S8196` 重命名的接口均为包内/跨包引用，非对外契约）
- permission contract: not-applicable（未新增路由或权限键；seed 的权限键字面量原样保留，`check-menu-contract` 回读 40 菜单/50 路由/221 权限/52 组件键一致）
- audit coverage: not-applicable（无审计链变更；`appendTaskEvent` 由 9 个位置参数改为 `taskEventInput` 结构体，字段与写入顺序逐一对齐原调用）
- visual evidence: not-applicable（前端改动为等价重构：辅助函数上提到模块作用域、三元改 `if/else`、import 合并、`String()` 改 `typeof` 收窄；无渲染差异需要新证据，运行侧缺口见下）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变
- base drift: `node --test tests/scripts/*.test.mjs` 14 条既有失败与 `check:harness-generated` 红灯在 `a5b5910` 干净 worktree 上以同一集合复现，本 PR 不修；`frontend/scripts/check-menu-contract.mjs` 只能从 `const ( name = "value" )` 块解析 seed 字段标识符，这是 base 侧传感器的通用缺陷，按 Base-first 原则回 base 提，不在 ops 手抄修改
- Base/ops inheritance: 本轮新增/改动的传感器均为 ops-only（`scripts/check-gorm-sql-literal.mjs` 依赖 `backend/modules/business`，base 不含该目录）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = 未变化（本 PR 不推进 lock）
- generic drift: not-applicable（未新增平台/系统域能力的副本）
- pseudo-drift: not-applicable
- business-only: no（代码全部在 business 作用域，但含 `.github/workflows/{quality,security}.yml` 聚合步的字面量去重，属高风险面）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 两个框都不勾选是准确的：代码改动全部落在 `backend/modules/business` 与 `frontend/src/modules/business`，但同一 changeset 含 `.github/workflows/quality.yml`、`.github/workflows/security.yml` 的去重（高风险面，因此 Quality Profile 取 `ci-workflow` 并按高风险要求双人评审），以及同一任务包的 manifest/evidence/任务包文档。前端为等价重构，无视觉面变更；后端两处行为性改动（webhook 自动同步实现、reconciler 桩删除）各自带测试或带 schema 依据，不是顺手清理。

## 验证记录

- [x] `cd backend && gofmt -l modules/business && go build ./... && go vet ./...`
- [x] `cd backend && golangci-lint run --timeout=25m --new-from-rev=a5b5910 ./...`（与托管 `Go Lint` 同一 new-code 模式）→ **0 issues**
- [x] `node scripts/check-gorm-sql-literal.mjs`（exit 0，4 条 waiver 可见）与 `node --test tests/scripts/check-gorm-sql-literal.test.mjs`（13/13）
- [x] `node scripts/check-duplication.mjs`、`check:structure`、`check:docs-frontmatter`、`check:task-packet-template`、`check:harness-adoption --strict`、`check:harness-method/template/failure-registry`、`check:encoding --strict`、`check:business-overlay` + `test:business-overlay`、`check:business-boundaries`、`check:ui-quality-gate`、`check:generated-modules` 全绿
- [x] `cd frontend && node ./node_modules/eslint/bin/eslint.js .`（无输出）+ `tsc -b`（exit 0）+ `vitest run`（230/230）+ `PANTHEON_CI=true npm run build`（exit 0）
- [x] `check-menu-contract` / `check-i18n-hardcode`(302 文件) / `check-i18n-generated-scope` 全绿
- [ ] `cd backend && go test ./... -count=1`（本地 MySQL 下 5 包 11 条失败，逐条在 `a5b5910` 干净 worktree 复现同一消息，见残留风险 ①；托管 `backend-tests` 用全新 `mysql:8.0`，是唯一权威）
- [x] 业务 smoke 或明确 runtime gap：本轮不新增页面行为，前端等价重构的运行侧证据缺口已显式记录为「以 `tsc -b` + `vitest` + CI 契约构建 + 菜单/i18n 合同代替浏览器 smoke」
- [ ] GitHub required checks 通过（合并前要求 `Quality Gates` / `Security Gates` 全绿，`main` 的 `strict=false`）
- [x] Copilot review 已请求（由 PR Automation 自动请求）

补充说明：清零判定用度量而非声明。以 `gocognit` 把 HEAD 的每条 `go:S3776` 映射到所属函数再复测工作树：**27/27** 已降至阈值以内（SonarGo 报告的复杂度不会高于 `gocognit`，因此 `gocognit ≤ 15` 是可靠的本地oracle）；对 `backend/modules/business` 的重复字面量清单从 871 个候选簇降到 802 个且**无任何新增簇**（表面新增的两处是扫描器把新 webhook 测试里反引号原始串中的 `"automated"`/`"Synced"` 误当字面量，SonarGo 按整 token 处理）。命令与逐条结果见 `commands.json`（本轮新增 14 条记录）。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：pending（本 PR 改动后端 Go 与前端 TS 源码，`CodeQL Security Scan` 与 `CodeQL Alert Gate` 需在 PR 运行后读回；已核验 diff 未新增 `os.Create`/路径拼接/SQL 拼接等 sink，SQL 文本字面量由必跑的 `check:gorm-sql-literal` 传感器阻断）
- GitHub checks 结果：pending（`Quality Gates` / `Security Gates` 为 `main` 的 required contexts）
- Auto-merge：enabled（`gh pr merge --auto --squash`，合并后删分支，仓库仅保留 `main`）
- Duplication Gate 结果：ran（`node scripts/check-duplication.mjs` PASS；43 文件后端扇出最容易踩的正是这个 base 侧复制粘贴门禁）
- 是否高风险改动：yes（① 触及 `.github/workflows/*` 的门禁聚合步；② 一次覆盖 65 个源文件的可维护性重构，等价性是主要风险）。因此按高风险要求：至少两位非作者批准，其中一位为域/安全/架构负责人
- Ratchet 说明：本轮不加新传感器、不改门禁规则，两条新暴露的耦合以任务包 + evidence 记录并给出机器可核依据——① `EXPECTED_RESULT` 是 fail-closed（未设变量展开为空串即比较不匹配、exit 1），且 `tests/scripts/security-workflow.test.mjs` 以精确文本钉住 dependency-vulnerability 分支的字面量，故该分支保留一处 `"success"`、其余四处才参数化；② base 的 `check-menu-contract.mjs` 只从 `const ( name = "value" )` 块建表，行内 `const` 会被解析为空串，`serviceModuleKey` 因此放回 const 块，通用修复按 Base-first 回 base（任务 #38）。扇出结果不信自述：两个独立对抗审计 + 逐行亲读高风险抽取（`state.go`/`host_state.go`/seed/导入路径/前端辅助函数上提）判为无阻塞后，仍以 `golangci-lint --new-from-rev` 复核，并因此抓到 2 条真实回归（一个只被死函数 `executeSSHHost` 调用从而 `unused` 的新提取辅助函数、一个 `goconst` 重复字面量），两者都以改代码收敛而非注解豁免
- Residual risk / follow-up：① 本地 MySQL 下 11 条 Go 测试失败（deploy×4、bizscope×3、observability×2、cmdb/host×1、lowcode/dynamicmodule×1）在 `a5b5910` 干净 worktree 以同一消息复现、单独 `-run` 亦复现，判为环境相关的既有失败；托管 `backend-tests` 为权威。② `node --test tests/scripts/*.test.mjs` 的 14 条失败与 `check:harness-generated` 红灯同样逐条复现于 HEAD，属 base 漂移既有项。③ 「Issues 归零」的验收以 push 触发的托管分析为准，本地 `gocognit`/重复扫描只是近似；分析落地后回读 `/api/issues/search?issueStatuses=OPEN,CONFIRMED` 与 `/api/qualitygates/project_status` 并把读数写入任务包。④ `biz_argocd_application` 与服务实例无关联列，reconciler 无法把 ArgoCD 状态推进服务实例状态机，需先做 schema 与接口设计，已写入 manifest 的 `deferredCodeIssues`。⑤ `check-menu-contract.mjs` 对内联 `const` 静默解析为空串，是 base 侧传感器的通用缺陷，回 base 修。⑥ 等价重构依赖人审 + 既有测试覆盖，`deploy`/`observability` 部分路径本地被环境失败遮挡，若托管侧出现新失败需以 `git revert` 回退本 PR 而非就地放宽。⑦ Dependabot 运行仍读不到 `SONAR_TOKEN`、`sonar.sources=.` 会把 PNG evidence 当源码读、无覆盖率报告（`new_coverage` 未被真正评估）、ArgoCD `caCert` 仅 API 可达、Vercel 检查项按维护者指示移出范围——均为先前轮次已记录的既有缺口

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（代码归 business 作用域，工作流去重按高风险面单独声明）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（`check-menu-contract` 的 const 块解析缺陷回 base 提，ops 侧只把声明改成现有传感器能解析的形状；未手抄 base 文件）
- [x] 前端新增展示文案已使用 i18n（无新增展示文案；`check-i18n-hardcode` 302 文件通过）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（seed 常量下沉后 `check-menu-contract` 回读一致；未新增路由/权限键；`check:business-overlay`、`check:business-boundaries`、`test:business-overlay` 全绿）
- [x] UI 改动已有渲染证据或明确 runtime gap（等价重构无渲染面变更，gap 已显式说明）
- [x] 已确认不会泄露敏感配置、账号密码或 Token（`.env` 的 DSN/密钥仅以 shell 变量注入测试环境，未 echo 值；diff 不含任何 token、DSN 或凭据字面量；新增测试只使用 `git.example.com` 与 RFC 5737 文档地址段）
