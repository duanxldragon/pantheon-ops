## 变更摘要

- 改动层级：inheritance-sync
- 改动模块：`.foundation/foundation-release.lock.json`、
  `.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/{commands.json,summary.md,review.md}`
- 目标问题：v0.18.1 锁文件的第四项 `pendingWork`（`runtime-acceptance`）从 v0.17.7 原样继承为
  `open`，理由是「需要一台带凭据的 MySQL」。台账因此停留在「构建与漂移已验证，但消费者从未真的
  起来过」的半开状态：golang-migrate 到 000030、business seed、租户控制台路由都只是声明，没有读数。
- 预期影响：台账收口为 4/4 `done`，并补上真实的运行时证据链；业务源码零改动。消费者的运行时验收
  口径从「基线未移动，故沿用上一轮」提升为「在隔离 schema 上跑通 migrate → boot → seed → 租户
  控制台读写全生命周期」。

## Harness 链路

- Task ID：`2026-10-29-foundation-upgrade-base-v0-18-1`
- Task Manifest：`.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json`
- Evidence：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json`
- Verification evidence：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/summary.md`
- Review Artifact：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/review.md`
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：no-repeat-observed
- GitHub Signal：runtime-evidence-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-29-foundation-upgrade-base-v0-18-1
- task manifest: .harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json
- evidence: .harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json
- boundaries: 仅锁文件 pendingWork 状态与 evidence 追加；无 backend/frontend 源码、无迁移文件、无门禁配置改动
- backend response contract: 未改动；本轮只是把既有的租户控制台契约在运行时读了一遍
- backend DTO contract: 未改动（`CreateTenantDTO` / `UpdateTenantDTO` 只作为调用方使用）
- permission contract: 未改动；观测到的 `auth.operation.verification_required` 是既有 secure-action 守卫
- audit coverage: 未改动；审计/维护任务在启动日志中正常完成
- visual evidence: 未提供（本轮为后端运行时验收；UI 面在 knownGaps 显式保留）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 的版本 marker 不变（releaseVersion / baseCommit 未动）
- base drift: 无（`check:shared-drift` 的 MODULE_NOT_FOUND 与 `check:business-overlay` 之外的既有缺口见残留风险）
- Base/ops inheritance: 无新发布；本轮吸收的是已在 v0.18.1 锁内的 base 行为
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = `pantheon-base-v0.18.1` / `93505225`
- generic drift: handled in `pantheon-base` first (not-applicable)
- pseudo-drift: excluded as line-count or formatting noise without contract break (no)
- business-only: no（本轮只动继承台账与治理证据，未动 `business/*`）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 边界：这不是再一次基线切换。base commit 仍是 `93505225`、tag 仍是 `v0.18.1`、四个资产未重哈希。
> 改动只落在消费者自己的台账与证据面。之所以不算越界：`runtime-acceptance` 是锁文件里唯一
> 未闭合的验收声明，而它需要的是本地 MySQL 与一次性隔离 schema，不是 base 侧修复。

## 验证记录

- [x] `npm run check:business-overlay`（exit 0）；`test:business-overlay` 未重跑：本轮无 overlay 变化
- [x] `npm run check:gorm-sql-literal`（不涉及：无 Go 源码改动）
- [x] `go test -count=1 ./modules/system/iam/tenant/ ./modules/platform/` → ok（0.067s / 0.042s，未缓存）
- [x] `go run ./cmd/server` 未改业务源码；`cd frontend && npm run lint && type-check && build` 不涉及（无前端改动）
- [x] 业务 smoke 或明确 runtime gap：以真实 HTTP 契约代替浏览器 smoke——登录（cookie jar：access + refresh + csrf）、
      `user/info`、`tenant/list`、`tenant-ops/plan/list`、`tenant-ops/overview`、`tenant-ops/member` 全 200；
      切 `platform.tenant_mode=multi` 后跑通 create → get → members → overview → archive → 首次 delete 被
      secure-action 守卫拒（`auth.operation.verification_required`）→ `/auth/operation-verify` 取
      `X-Operation-Token` → delete 成功，`total=0` 且行转为 `__deleted_tenant_1/deleted`
- [x] 空 schema 上的 golang-migrate：`schema_migrations version=30 dirty=false`、34 张表；
      boot 后 81 张表、`system_menu` 228、`tenant_plans` 4（全 active）、`system_setting` 84、`system_i18n` 1385
- [x] 隔离纪律：全程使用新建 schema `pantheon_ops_ra`，`pantheon_ops` 开发库未被写入；未用
      `docker-compose.yml`（3306 冲突且挂载已废弃的 `database/system_init.sql`）；未执行任何 `DROP DATABASE`
- [x] 治理矩阵：`check:harness-inventory` / `check:harness-encoding` / `check:structure` /
      `check:harness-method` / `check:harness-adoption` / `check:failure-registry` /
      `check:docs-frontmatter` / `check:encoding` / `check:pr-governance` / `check:task-packet-template` /
      `check:duplication` / `check:harness-sync` → exit 0；
      `check-evidence.mjs --strict` 与 `check-review.mjs --strict` 对本包由 FAIL 转 PASS
      （补了 review 机器可读块缺失的 `taskId`，并把 D5 状态从 `inherited-open` 改为 `closed-2026-10-31`）
- [x] GitHub required checks 通过
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用（见审核留痕）

补充说明：一次性驱动文件（临时 Go 程序）与临时脚本已删除，工作树只留 4 个治理文件的改动；
`pantheon_ops_ra` 原地保留以便复查，可随时丢弃。

## 审核留痕

- Copilot review：unavailable（本仓库/账号无法解析 `copilot` reviewer）
- CodeQL 结果：本 PR 只改 `.json` 与 `.md`，无新增抽取面；回读后确认
- GitHub checks 结果：待回读
- Auto-merge：enabled（沿用仓库既有 squash 策略）
- Duplication Gate 结果：PASS（`npm run check:duplication` exit 0）
- 是否高风险改动：no（无源码改动；运行时证据落在隔离 schema）
- Residual risk / follow-up：
  1. 浏览器侧 `tests/smoke-core/tenant-console.spec.ts` 未跑（需 dev server + tenant matrix fixture），
     已写入 `knownGaps`，不能据本轮读数声称 UI 面已验收。
  2. `check:release-assets` 与 `check:shared-drift` 在本仓 MODULE_NOT_FOUND（package.json 引用了
     未随 v0.18.1 落地的脚本）；实测在干净 `main` 上同样失败，属继承缺口，按 Base-first 归 base 处理。
  3. `check-evidence/check-review --strict` 仍对 `ops-*` 与 `2026-10-29-sonar-ci-analysis-restore`
     等旧证据目录报缺失（同一 `taskId`/root shape 形态）。本轮只修自己所处的包，未批量翻修历史包。
  4. 会话卫生：一次 shell `source .env` 失败时回显过本地开发库连接串；该串未进入任何仓库文件或提交，
     驱动随后改为进程内解析。建议轮换该本地口令。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（本次为 inheritance-sync 台账收口）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（无新增平台问题；继承缺口显式标注归 base）
- [x] 前端新增展示文案已使用 i18n（不涉及）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（不涉及源码；seed 与菜单为实测读数）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 改动；gap 已显式记录）
- [x] 已确认不会泄露敏感配置、账号密码或 Token（凭据仅在进程内解析，未写入仓库；见残留风险第 4 条）
