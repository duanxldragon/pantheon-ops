## 变更摘要

- 改动层级：
- 改动模块：
- 目标问题：
- 预期影响：

## Harness 链路

- Task ID：
- Task Manifest：
- Evidence：
- Verification evidence：
- Review Artifact：
- OpenSpec change：
- Trivial change：yes / no
- Quality Profile：auth-security / permission-policy / i18n / ui-runtime / generator / ci-workflow / none
- Ratchet Decision：no-repeat-observed / guide-updated / sensor-added / gate-updated / template-updated / adapter-updated / registry-only
- GitHub Signal：method-gate / repo-quality-gate / runtime-evidence-gate / external-flaky / not-applicable

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id:
- task manifest:
- evidence:
- boundaries:
- backend response contract:
- backend DTO contract:
- permission contract:
- audit coverage:
- visual evidence:
- inheritance contract:
- base drift:
- Base/ops inheritance:

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 foundation 继承，已说明 Base-first 边界和消费方式

> 如果跨层，请说明 Base 与 Ops 各自职责、foundation release 身份，以及是否影响菜单/权限/i18n/审计。

## 验证记录

- [ ] `npm run check:inheritance`
- [ ] `go test ./...`
- [ ] `cd frontend && npm run lint && npm run type-check && npm run build`
- [ ] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：

## 审核留痕

- Copilot review：requested / automatic-policy / unavailable
- CodeQL 结果：
- GitHub checks 结果：
- Auto-merge：enabled / not-enabled / not-applicable
- Duplication Gate 结果：
- 是否高风险改动：
- Residual risk / follow-up：

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 foundation inheritance adapter
- [ ] 通用平台/系统域问题已在 Base-first 边界处理
- [ ] 前端新增展示文案已使用 i18n
- [ ] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [ ] UI 改动已有渲染证据或明确 runtime gap
- [ ] 已确认不会泄露敏感配置、账号密码或 Token
