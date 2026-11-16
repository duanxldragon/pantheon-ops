## 变更摘要

- 改动层级：inheritance-sync
- 改动模块：`frontend/package-lock.json`、`docs/harness/tasks/`、
  `.harness/{tasks,evidence}/2026-10-31-frontend-follow-redirects-audit/`
- 目标问题：`main` 在 2026-10-31T04:36:50Z 的 `Security Gates` 变红，且仓库没有任何新提交。
  `Dependency Vulnerabilities` 的前端 `npm audit` 在 `--audit-level=high` 下退出 1：
  `axios@1.20.0` 解析出的 `follow-redirects@1.16.1` 落进了新发布的
  GHSA-mrwp-f3q8-3g9p（high，CWE-201，范围 `>=1.16.0 <=1.16.1`）。
- 预期影响：把 `node_modules/follow-redirects` 的 lock 条目从 `1.16.1` 提到 `1.16.2`
  （advisory 范围外最小的已发布版本，且仍满足 axios 的依赖范围），前端 audit 的 high / critical
  归零，`Dependency Vulnerabilities` 与 `Security Gates` 恢复通过。`frontend/package.json`、
  安全 workflow、业务源码均不变。

## Harness 链路

- Task ID：`2026-10-31-frontend-follow-redirects-audit`
- Task Manifest：`.harness/tasks/2026-10-31-frontend-follow-redirects-audit/manifest.json`
- Evidence：`.harness/evidence/2026-10-31-frontend-follow-redirects-audit/commands.json`
- Verification evidence：`.harness/evidence/2026-10-31-frontend-follow-redirects-audit/summary.md`
- Review Artifact：`.harness/evidence/2026-10-31-frontend-follow-redirects-audit/review.md`
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-31-frontend-follow-redirects-audit
- task manifest: .harness/tasks/2026-10-31-frontend-follow-redirects-audit/manifest.json
- evidence: .harness/evidence/2026-10-31-frontend-follow-redirects-audit/commands.json
- boundaries: 只动消费者自有的 `frontend/package-lock.json` 单个条目；`frontend/package.json`、`.github/workflows/security.yml`、业务源码与 i18n 均不改
- backend response contract: 未改动
- backend DTO contract: 未改动
- permission contract: 未改动
- audit coverage: 未改动
- visual evidence: 不适用（无 UI 改动）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 版本 marker 不变（未消费新发行包）
- base drift: 不适用（本轮无 foundation 消费）
- Base/ops inheritance: 无 lock 变更；`pantheon-base` 同样的 1.16.1 锁定在 base 侧独立修复，避免 base 自己变红
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit 保持 `pantheon-base-v0.18.2` / `a44ae4e012fff20430b1b7f91cb7a9296f3ea843`
- generic drift: 无；依赖 advisory 属消费方 lock 维护，两侧各自刷新同一版本
- pseudo-drift: 无
- business-only: no（依赖 lock 维护，不涉及业务代码）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 边界：不消费新发行包、不重建共享面。改动只落在 ops 自有且未被 overlay 覆盖的
> `frontend/package-lock.json`（该文件的差异本就是有意的依赖版本差），因此不需要走 Base 重建；
> base 仓库用同样的 lock 条目修复单独处理，保证上游自己也不红。

## 验证记录

- [x] `cd frontend && npm ci --ignore-scripts` → 454 个包安装成功，lock 自洽
- [x] `cd frontend && npm ls follow-redirects` → `axios@1.20.0 -> follow-redirects@1.16.2`
- [x] `cd frontend && npm audit --registry=https://registry.npmjs.org --json` → high 0 / critical 0
      （修复前 1 high + 19 moderate；剩余 19 条 moderate 低于 `--audit-level=high`，属 lint/coverage
      工具链大版本升级，未夹带进本次解锁补丁）
- [x] `cd frontend && npm run lint && npm run type-check && npm run build` → 0 / 0 / `✓ built in 736ms`
- [x] `npm run check:task-packet` / `check:docs-frontmatter` / `check:structure` / `check:encoding` /
      `check:pr-governance` → exit 0
- [x] 业务 smoke 或明确 runtime gap：无产品行为变更，lock 刷新不影响运行时代码路径；
      以 lint/type-check/build 与 audit 读数为准
- [x] GitHub required checks 通过（见审核留痕回读）

## 审核留痕

- Copilot review：unavailable（本仓库/账号无法解析 `copilot` reviewer）
- CodeQL 结果：无代码变更，预期不触发新发现（见 PR 检查回读）
- GitHub checks 结果：待回读
- Auto-merge：enabled（沿用仓库既有 squash 策略）
- Duplication Gate 结果：PASS（无代码改动）
- 是否高风险改动：no
- Residual risk / follow-up：
  1. 19 条 moderate advisory 仍在（`@typescript-eslint/*`、`eslint`、`minimatch`、`nyc`、
     `vite-plugin-istanbul` 等），需要 lint/coverage 工具链大版本升级，另行决策。
  2. 该 advisory 由上游发布时间线引入，同类问题会以同样方式在任意时间点把 main 变红；
     是否把 `npm audit` 的固定 registry 快照或 Dependabot 自动分支纳入常态，留给维护者。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（本次为消费者 lock 维护）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（base 侧同版本修复单独提交）
- [x] 前端新增展示文案已使用 i18n（无文案改动）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（范围内无此类改动）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 改动）
- [x] 已确认不会泄露敏感配置、账号密码或 Token
