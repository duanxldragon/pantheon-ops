# O3 Change Preview And Risk

- Priority: `P0`
- Layer: `business/deploy + business/k8s + business/cicd`
- Status: `Deploy signed preview/start, frozen configuration and browser conflict flow verified locally; other domain previews and formal acceptance open`
- Base Dependency: B3 `ChangeDiff`
- Business Dependency: immutable intent/current-target version contracts

## Outcome

在 Deploy、K8s Release、Argo CD sync 和 rollback 执行前展示可追踪的差异、影响对象、风险和回滚条件。

## In Scope

- 统一 preview envelope：intent ID/version、current version、target version、objects、diff、risk、rollback。
- Deploy package/template/parameter preview。
- K8s manifest/image/config/resource preview。
- Argo CD sync/rollback version tree and resource diff。
- conflict-on-change：预览后对象变化必须重新预览。

## Out Of Scope

- 不在浏览器执行可信 diff 或安全判断。
- 不保证所有外部系统都能自动回滚。
- 不展示秘密、完整 credential 或受限配置值。

## Expected Files

- 各业务域 provider-owned preview endpoint、DTO、tests。
- `frontend/src/modules/business/**` 中基于 Base `ChangeDiff` 的业务组合。
- permission/audit/i18n/smoke 和风险 fixture。

## Acceptance

1. preview 与 execute 绑定同一 intent/version，版本漂移返回 409/conflict。
2. 新增/修改/删除/冲突和未变化项可区分，风险不只靠颜色表达。
3. 风险摘要指出目标数量、环境、不可逆项、健康检查与回滚可用性。
4. 秘密字段在进入 UI 前遮蔽，下载/复制同样遵守权限和审计。
5. 大 diff 可折叠/搜索/跳过未变化项，mobile 至少可完成审阅和取消。

## Evidence

- backend：intent binding、TOCTOU conflict、permission/DataScope、redaction、audit。
- domain fixtures：Deploy、K8s、Argo CD、rollback 的 safe/high-risk/irreversible cases。
- Playwright：preview -> confirm -> conflict -> refresh，keyboard and mobile。
- external evidence：可用测试集群/Argo CD 的实际 diff，或 endpoint/credential gap。
- screenshots：desktop/mobile x low/high-risk/conflict/forbidden x light/dark。

## Gates

- 新 schema/public API、外部 production mutation、风险策略影响发布权限、Base lock。

## Reconciliation 2026-11-07

The required Base primitive is present in official v0.18.6. This resolves the prior component-release dependency only; all implementation and acceptance criteria above remain open. See the parent EXECUTION_QUEUE.md and current execution evidence.

## Current implementation evidence

See `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md` for implemented Deploy scope, exact verification and remaining gaps. The packet remains in-progress.

## Frozen Deploy configuration continuation

The provider now projects the saved execution snapshot beside the host package diff: frozen template/package identities, effective step action/failure policy and bounded parameters. Current parameters are explicitly unknown. Only approved structural parameter values are returned; secrets/sentinels, freeform/nested values, unreadable-secret cases and oversized values are hidden server-side. Commands, URLs, conditions and StepConfig are excluded. Truncation is explicit at 100 steps / 200 parameters. This is proposed intent, not a measured parameter diff or a new execution policy.

Real MySQL race verifies live-definition independence, old-ticket rejection after frozen parameter drift, redaction and bounds. UI tests and rendered light/dark desktop/mobile flows cover safe/hidden values, search, keyboard collapse and conflict/refresh. Source review and the additional failure/motion/zoom matrix are tracked in the linked current evidence. K8s/Argo/rollback, native browser zoom, real external sources and final formal acceptance remain open.

## Scoped rollback integrity

Deploy rollback create/read now propagate request DataScope. A supported single-package uninstall inverse is created atomically from the frozen definition and secret set as a draft; its own public start requires a new signed preview. Real MySQL and HTTP handler denial tests pass; real local API verifies disabled live definitions, frozen versions, signed start and owned cleanup. Scripted/conditional/multi-step and prior-version restoration contracts remain open rather than replaying forward commands. Evidence stays in the current closeout packet.

## K8s image review continuation 2026-11-08

Provider-owned expiring authenticated review now binds public create/rollback to actor, immutable request/key, Cluster configuration/credential version and workload UID/resourceVersion. The guard is private persisted JSON and is checked on the exact updated Kubernetes object; drift conflicts409. A separate reviewed rollback restores the recorded image. Expired authenticated tickets can recover a matching scoped actor-owned persisted result only, preventing ambiguous retries from creating another release. No new schema/root credential or raw manifests/configuration exposure. Required MySQL CGO race passes for the K8s packages, operational frontend49/49, vet/build and official assembly pass; final browser/hash-stable full race and independent review are pending. Image-only scope does not close manifest/config/resource, Argo, real cluster or final acceptance. Permission-route registry consumption still requires the formal Base release.

## Final local priority checkpoint 2026-11-08

Current scoped source passes complete hash-stable CGO race, backend vet/build, operational54/54 units, frontend lint/type-check/build and final combined browser6/6 with28 inspected desktop/mobile light/dark screenshots. The final browser waits for mobile shell transitions. O4 now labels completed result source/range/limit, stale inputs and safe warning counts; upstream errors/result-type mismatch cannot become false empty/success and cross-stream limits keep newer rows. Official locked assembly/contract/drift and62 overlay tests pass. This supersedes prior pending local verification notes, while independent approval and every omitted formal criterion remain open. See current closeout summary.
