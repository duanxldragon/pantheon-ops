## 变更摘要

- 改动层级：business/*（`backend/modules/business/**`、`frontend/src/modules/business/**`、`e2e-tests/**`）+ ops 自有质量门禁（`scripts/**`、`.github/workflows/{quality,security}.yml`）
- 改动模块：`cicd/argocd`（服务端证书校验 + caCert 三态）、`filedist`（存储根路径收敛 + 上传名单元素化）、`bizscope`/`k8s/storage`/`retired_modules`（SQL 字面量不变式的落点）、`service/state.go`、`cmdb/host`（导入模板示例 IP）、新增 `scripts/check-gorm-sql-literal.mjs` 与其 node 测试、`e2e-tests` 两个脚本与 `playwright.config.ts`、`.harness/v1.2-tasks/generate_tasks.sh`、4 个业务列表页 + `FiledistDetail`/`BackupList` + business i18n 资源、`.github/PULL_REQUEST_TEMPLATE.md`、`package.json`
- 目标问题：SonarCloud `duanxldragon_pantheon-ops`（branch `main`）质量门禁持续 ERROR——`new_security_rating` 与 `new_reliability_rating` 两项超标，开放问题 242 个，其中 62 个 vulnerability + 6 个 bug。上一轮明确承诺不靠降噪手段（不标 false positive、不移 new code 基线、不换松门禁），因此本轮按规则逐类真实修复，并把唯一无法逐条手工论证的规则族（33 条 `gosecurity:S3649`）先变成机器可检的不变式。
- 预期影响：全部 vulnerability/bug 类规则要么被代码修复，要么被 CI 阻断的不变式覆盖后再做有依据的 Sonar 状态流转。`shell:S5332` 族通过删除硬编码可达公网 IP 与隐式 http 默认值消除（改为必须显式提供 `API_BASE_URL`/`WEB_BASE_URL`）；`go:S4830`/`S5527` 通过删除 `InsecureSkipVerify`、改为可选 CA 束消除；`gosecurity:S2083` 通过存储根收敛消除。174 个 code smell 中的历史积压（`go:S1192`×68、`go:S3776`×27、`typescript:S3358`×22 等）本轮显式不做，理由见 Residual risk。

## Harness 链路

- Task ID：2026-10-30-sonar-findings-remediation
- Task Manifest：.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json
- Evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json
- Verification evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/summary.md
- Review Artifact：.harness/evidence/2026-10-30-sonar-findings-remediation/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：auth-security
- Ratchet Decision：sensor-added
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-30-sonar-findings-remediation
- task manifest: `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- evidence: `.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json`
- boundaries: base 拥有共享面；本 PR 只改 ops 自有 `business/*`、`scripts/*`、`tests/scripts/*`、`e2e-tests/*` 与 ops 侧 workflow 步骤，不改 base 文件、不改 overlay 声明、不改 lock、不改 Sonar 门禁定义
- backend response contract: `argocd.ServerResponse` 新增只读布尔位 `hasCaCert`，不回证书内容；`UpdateServerRequest.caCert` 改为三态指针（缺省保持 / 空串清除 / 非空替换）
- backend DTO contract: `AddServerRequest.caCert`、`UpdateServerRequest.caCert`、`ArgoCDServer.CACert`（`json:"-"`，`type:text`），由 argocd 模块既有 `Migrate()`（GORM AutoMigrate）落列，无版本化 SQL 引用该表
- permission contract: not-applicable（未新增路由或权限键，权限键沿用 `business:cicd:argocd:*`）
- audit coverage: not-applicable（无审计链变更）
- visual evidence: 前端仅改占位符文案来源与 Promise 处置，无布局变更；`npm run lint`、`npm run type-check`、`PANTHEON_CI=true npm run build` 全绿，`check:ui-quality-gate` 严格模式 0 finding
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变，仍为 `pantheon-base-v0.18.1`
- base drift: `test:quality-workflow` 存在 4 个先前即失败的用例（断言 base 版 `quality.yml` 的 paths-filter / governance-only 形状），本 PR 不修，已记入残留风险
- Base/ops inheritance: not-applicable（新增 checker 只扫 `backend/modules/business/**`，base 不含该目录，属 ops 专有传感器，不下沉 base）
- generic drift: not-applicable
- pseudo-drift: not-applicable
- business-only: no（含 ops 自有 `.github/workflows` 步骤与根 `package.json` 脚本）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 主体落在 `business/*`，另含三处 ops 自有治理面：`scripts/check-gorm-sql-literal.mjs` + `tests/scripts/*`（新传感器）、`.github/workflows/quality.yml` 的 `docs-governance` 两个阻断步骤、`.github/workflows/security.yml` 的 zizmor 安装参数（已与 base `pantheon-base/.github/workflows/security.yml:206` 对齐）。`business-overlay.json` 未改动：`.github` 与 `scripts` 的 overlay 声明已覆盖新增文件，`check:business-overlay` 与 `check:structure` 均绿。Base-first 判断：字面量不变式的作用域是 ops 业务模块，base 无该目录，因此不下沉、不手抄。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：实际执行集（逐条结果见 `commands.json`）——`cd backend && gofmt -l modules/business`（0 文件）+ `go build ./...` + `go vet ./...` + `go test ./... -count=1`（exit 0，64 包 ok，无 FAIL）；`node scripts/check-gorm-sql-literal.mjs`（绿，4 条 waiver 全部打印可见）；`node --test tests/scripts/check-gorm-sql-literal.test.mjs`（13/13）；`cd frontend && npm run lint` + `npm run type-check` + `PANTHEON_CI=true npm run build`（全绿）；`bash -n` 三个 shell 文件；治理集 `check:structure`/`check:docs-frontmatter`/`check:task-packet-template`/`check:encoding --strict`/`check:business-boundaries`/`check:ui-quality-gate`/`check:generated-modules`/`check:harness-adoption`/`check:task-packet` 全部 0 finding。Runtime gap：未在本机启动后端并点击 ArgoCD 私有 CA 注册链路（该模块无前端注册表单，`caCert` 目前仅可经 API 传入），运行态证据是 Go 侧 `httptest.NewTLSServer` 的证书校验通过/失败断言与 SonarCloud 托管侧读数。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：pending（本 PR 触发 CodeQL Security Scan，合并前以托管结论为准；历史 main 上该扫描与 Alert Gate 均为 success）
- GitHub checks 结果：pending（合并前要求全绿；`test:quality-workflow` 的 4 个既有失败属 base 漂移，不在本 PR 的 CI 断言范围）
- Auto-merge：enabled
- Duplication Gate 结果：ran（`node scripts/check-duplication.mjs` 结果 PASS；`veleroState`/`applyMenuFieldFilter` 是为消除重复分支与嵌套模板而提取，未新增重复块）
- 是否高风险改动：yes（触碰 `system` 侧无、但含 `.github/workflows/*`、出网 TLS 信任、存储路径边界与新增阻断门禁，按 repo-pr-gate 归高风险）
- Residual risk / follow-up：① 174 个 code smell 的历史积压本轮显式不动（`go:S1192`×68 字符串重复、`go:S3776`×27 认知复杂度、`typescript:S3358`×22 嵌套三元、`typescript:S3863`×10、`godre:*`、`shell:S7688`×7、`go:S1135` TODO×2、`go:S107`×2）；这些在新代码期之外，且把它们塞进安全修复 PR 会让评审更难，需要独立的可维护性任务包。② 33 条 `gosecurity:S3649` 的流转只在本 checker 已合并且 CI 阻断之后执行：issue key 已完整存档于 `sonar-s3649-keys.json`，可逐条回退，理由文本指向 `scripts/check-gorm-sql-literal.mjs`。除此之外未采用任何降噪：不移 new code 基线、不换门禁、不把真实漏洞标成误报。③ Dependabot 触发的运行读不到仓库 secret，SonarCloud Scan 在该类 PR 上仍为红；provision Dependabot secrets 或让该 job 跳过此 actor 都属维护者的门禁策略决定。④ Vercel 检查项仍为红：它是外部 GitHub App 集成，本机无凭据且无提交配置，删除需在 GitHub UI 操作。⑤ `test:quality-workflow` 的 4 个既有失败源于 ops `quality.yml` 缺少 base 的 governance-only / paths-filter `all` 形状，修它等于改「哪些 job 允许跳过」，需单独决策。⑥ `sonar.sources=.` 使扫描器把 PNG evidence 当源码文本读；收窄该属性属 Sonar 配置决定。⑦ 本轮未为 ArgoCD 增加服务器注册表单，`caCert` 暂只经 API 可用。⑧ 两处 `go:S1135` TODO 背后是未实现的真实行为——`argocd_webhook.go:304` 在 auto-sync 策略未解析的情况下仍记录 "triggering sync"，`argocd_reconciler.go:183` 只打日志不改服务实例状态，这与本仓库「扩展点不提供伪成功实现」的原则冲突；补全需要 capability 接线与策略解析，属功能任务，本轮不冒充修复，只显式登记。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
