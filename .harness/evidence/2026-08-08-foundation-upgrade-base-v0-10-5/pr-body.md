## 变更摘要

- 改动层级：foundation inheritance adapter + `business/*`
- 改动模块：foundation consumer、lowcode generator、business registries、i18n、smoke governance
- 目标问题：完成历史业务收口，并消费修复共享 smoke runner 测试闭包的 `pantheon-base-v0.10.5`
- 预期影响：生成代码稳定落入 `business/*`，手写业务模块保留，Ops 与不可变 v0.10.5 共享合同完全对齐

## Harness 链路

- Task ID：`2026-08-08-foundation-upgrade-base-v0-10-5`
- Task Manifest：`.harness/tasks/2026-08-08-foundation-upgrade-base-v0-10-5/manifest.json`
- Evidence：`.harness/evidence/2026-08-08-foundation-upgrade-base-v0-10-5/commands.json`
- Verification evidence：`.harness/evidence/2026-08-08-foundation-upgrade-base-v0-10-5/summary.md`
- Review Artifact：`.harness/evidence/2026-08-08-foundation-upgrade-base-v0-10-5/review.md`
- OpenSpec change：`none`
- Trivial change：`no`
- Quality Profile：`generator`
- Ratchet Decision：`adapter-updated`
- GitHub Signal：`runtime-evidence-gate`

## Harness adoption markers

- task id: `2026-08-08-foundation-upgrade-base-v0-10-5`
- task manifest: `.harness/tasks/2026-08-08-foundation-upgrade-base-v0-10-5/manifest.json`
- evidence: `.harness/evidence/2026-08-08-foundation-upgrade-base-v0-10-5/commands.json`
- boundaries: `immutable Base v0.10.5 artifact -> exact Ops consumer allowlist -> business/* overlays`
- backend response contract: `unchanged`
- backend DTO contract: `unchanged`
- permission contract: `existing corrected lowcode policies retained`
- audit coverage: `existing operation-token and purge audit middleware retained`
- visual evidence: `no new visual design; hosted smoke is the runtime gate`
- inheritance contract: `pantheon-base-v0.10.5 b97c0f6d288e2c984fbd9215d6d3626929f68d85`
- base drift: `shared smoke closure returned to Base and consumed through the release artifact`
- Base/ops inheritance: `archive SHA-256 a49457b96ec1ff18bd716c149708433798e5e99db8698fa686f6346c26f555f0`

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 foundation 继承，已说明 Base-first 边界和消费方式

Base owns the shared generator and smoke runner contracts. Ops owns the exact consumer adapter, lock, evidence, and `business/*` overlays. No immutable release content was modified or copied by hand.

## 验证记录

- [x] `npm run check:inheritance`
- [x] `npm run test:foundation-release`（73/73）
- [x] `npm --prefix frontend run test:smoke:scripts`（10/10）
- [x] `npm --prefix frontend run lint && npm --prefix frontend run type-check && npm --prefix frontend run build`
- [x] `$env:PATH='D:\msys64\mingw64\bin;' + $env:PATH; $env:CGO_ENABLED='1'; go test -race ./...`
- [ ] GitHub required checks：最新 head 推送后验证

## 审核留痕

- Copilot review：`automatic-policy`
- CodeQL 结果：`pending latest hosted head`
- GitHub checks 结果：`pending latest hosted head`
- Auto-merge：`not-enabled`
- Duplication Gate 结果：`pending latest hosted head`
- 是否高风险改动：`yes，foundation inheritance + generator/runtime closeout`
- Residual risk / follow-up：`PR #75 合并后继续清理 Dependabot PR、告警与遗留分支`

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 foundation inheritance adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 菜单、权限、接口、审计和 smoke 在范围内同步
- [x] 未保留 `business/*` 或 generated registry 漂移
- [x] 已确认不会泄露敏感配置、账号密码或 Token
