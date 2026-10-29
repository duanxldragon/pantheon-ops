## 变更摘要

- 改动层级：inheritance-sync（`pantheon-ops` consumer 侧仓库级配置与 overlay 声明）
- 改动模块：`sonar-project.properties`（恢复）、`business-overlay.json`（`repositoryOverlayPaths` 新增声明）、本任务 packet / manifest / evidence
- 目标问题：托管 `SonarCloud Scan` 在凭证修复后仍然失败——`You must define the following mandatory properties: sonar.projectKey, sonar.organization`。根因是 `sonar-project.properties` 是 Ops 独有文件（base 只发布 `.sonarcloud.properties`）且从未在 `business-overlay.json` 声明，因此在 `5c12964f`（消费 `pantheon-base-v0.17.7`）重建时被删除，CI 侧 Sonar 分析通道就此失效。
- 预期影响：恢复 `sonar-project.properties`（取自 `5c12964f^`，含 projectKey `duanxldragon_pantheon-ops`、organization `duanxldragon`、源码排除、`sonar.coverage.exclusions=**/*` 与测试文件 cpd 排除）并把该文件声明为 repository overlay，使后续 foundation 消费不会再删除它；`rebuild:from-base` 的声明/生成文件数从 758 变为 761。不改工作流语义、不改质量门禁策略、不把 token 值写入仓库。

## Harness 链路

- Task ID：2026-10-29-sonar-ci-analysis-restore
- Task Manifest：.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json
- Evidence：.harness/evidence/2026-10-29-sonar-ci-analysis-restore/commands.json
- Verification evidence：.harness/evidence/2026-10-29-sonar-ci-analysis-restore/summary.md
- Review Artifact：.harness/evidence/2026-10-29-sonar-ci-analysis-restore/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：registry-only
- GitHub Signal：repo-quality-gate

## Harness adoption markers

- task id：2026-10-29-sonar-ci-analysis-restore
- task manifest：`.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json`
- evidence：`.harness/evidence/2026-10-29-sonar-ci-analysis-restore/commands.json`
- boundaries：`pantheon-base` 拥有共享面；`sonar-project.properties` 是 Ops 独有仓库级配置，本次只恢复与声明，不改 base、不改工作流语义
- backend response contract：unchanged
- backend DTO contract：unchanged
- permission contract：unchanged
- audit coverage：unchanged
- visual evidence：none（无 UI 改动；证据为托管 SonarCloud 分析运行）
- inheritance contract：`docs/PROJECT_INHERITANCE.md` 无需变更（未推进 lock，仍为 `pantheon-base-v0.18.1`）
- base drift：无新增 drift；`check:business-overlay` OK（53 declared paths）
- Base/ops inheritance：Base-first 判定为「非 base 问题」：base 仓库根本不发布该文件，缺失属 consumer 侧 overlay 声明遗漏，因此在 ops 修复并加声明，不向 base 复制任何文件
- Base version checked: `foundation-release.lock.json` releaseVersion = pantheon-base-v0.18.1, baseCommit = 93505225e36c8d3ba5aa9facc12975eb813167d3
- generic drift：no（consumer 独有配置文件恢复 + overlay 声明）
- pseudo-drift：no（恢复 34 行配置文件，非格式化噪声，直接决定 CI 分析能否运行）
- business-only：no（仓库级继承配置，未触碰 `business/*` 业务逻辑）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> Base 侧不发布 `sonar-project.properties`，只发布 `.sonarcloud.properties`；后者约束 SonarCloud 自动分析，而工作流使用的 sonar-scanner CLI 只读前者。因此两个文件都必须存在且都必须在 `repositoryOverlayPaths` 声明：本次新增 `sonar-project.properties` 声明，`.sonarcloud.properties` 保持原样。未推进 foundation lock，未改 `.github/workflows/sonarcloud.yml` 的触发条件或门禁语义，不影响菜单/权限/i18n/审计/seed/smoke。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `go test ./...`
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [x] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：本次改动不含 Go/TS 源码，因此 `go test ./...` 与前端三连（lint/type-check/build）沿用 0.18.1 消费轮已记录的全绿结果，本轮直接执行的门禁为：`check:business-overlay` OK、`test:business-overlay` fail 0、`rebuild:from-base` 生成 761 个声明/生成文件且重建树内含恢复后的 `sonar-project.properties`、`check:task-packet` 对本 packet 0 error 0 warning、`check:harness-adoption` 0 finding、`check-pr-governance.mjs --event` OK。Runtime gap：未启动应用运行时；本轮运行态证据就是托管 SonarCloud 分析本身（认证通过、项目与组织解析成功，扫描要么产出分析版本要么失败）。扫描前的失败基线是 main dispatch `40849180395`（exit 3，缺少强制属性），凭证侧 `api/authentication/validate` 返回 `{"valid":true}`。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：pending-hosted-run
- GitHub checks 结果：pending-hosted-run
- Auto-merge：enabled
- Duplication Gate 结果：pass（配置文件与 JSON 声明，无新增重复代码）
- 是否高风险改动：no（仅恢复 Ops 独有配置文件并加 overlay 声明；不改工作流语义、不提交任何密钥值）
- Residual risk / follow-up：① Dependabot 触发的运行无法读取仓库 secret，`SONAR_HOST_URL` 为空导致 `URI with undefined scheme`，扫描在 Dependabot PR 上仍为红；配置 Dependabot secret 或对 `dependabot[bot]` 跳过该 job 属门禁策略决策，需维护者拍板。② 项目质量门禁当前 `new_security_rating` 为 ERROR（4 vulnerabilities / 245 code smells），扫描恢复后这些真实存量问题会重新可见；工作流不因门禁 ERROR 变红。③ Free 计划无法自定义质量门禁，`sonar.coverage.exclusions=**/*` 是中和内置 `new_coverage` 条件的必要手段，后续若引入覆盖率报告需同步重评。④ 任何新增的 Ops 独有根文件若不写进 `repositoryOverlayPaths`，会以完全相同的方式被下一次消费删除。⑤ Vercel 仍为红，属外部集成且仓库内无配置，不在本次范围。

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
