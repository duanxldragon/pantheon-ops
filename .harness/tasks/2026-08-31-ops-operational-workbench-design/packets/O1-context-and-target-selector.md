# O1 Unified Context And Target Selector

- Priority: `P1`
- Layer: `business/bizscope + business/cmdb`
- Status: `Deploy resolve/create/update and UI verified locally; other consumers and formal acceptance open`
- Base Dependency: B3 `ContextSelector` in immutable foundation release
- Consumers: Deploy, K8s release, file distribution, bulk operations

## Outcome

提供统一的业务上下文与目标选择体验，支持拓扑、分组、主机、集群资源、手工输入和排除集合，而不绕过 DataScope。

## In Scope

- BizScope/environment context header and selector adapter。
- CMDB topology/group/host adapters，remote search、分页/虚拟化和状态提示。
- 手工 IP/ID 输入的解析、去重、匹配、未知/越权项反馈。
- selected/excluded/invalid summary、目标快照和提交前 revalidation。

## Out Of Scope

- 不在 Ops 复制 `ContextSelector` 外壳。
- 不把客户端选择结果当作授权事实。
- 不扩大现有 BizScope/CMDB ownership 或直接跨模块读表。

## Contract Anchors

- `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md` §5。
- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`。
- BizScope/CMDB provider-owned typed capability contracts。

## Expected Files

- `frontend/src/modules/business/shared/` 下业务 adapter/composition。
- 各消费模块的最小接入和业务 i18n。
- provider-owned backend read/resolve endpoint 仅在现有合同不足时增加。

## Acceptance

1. 切换 BizScope 会清理或标记不再合法的选择，不保留越权幽灵目标。
2. 手工输入报告有效、重复、未知、越权和状态不符数量。
3. 服务端在创建/执行前按稳定 ID、scope version 和状态重新解析。
4. 审计保存来源分布、排除规则、解析时间和最终数量，不保存秘密。
5. desktop 多栏与 mobile 分步视图可完成全键盘选择；10k 候选不失控。

## Evidence

- unit/integration：parser、dedupe、DataScope、scope change、stale ID、TOCTOU revalidation。
- permission matrix：view/search/select/resolve 与消费动作。
- Playwright：topology/group/manual/exclusion、mobile、error/forbidden/partial invalid。
- screenshots：desktop/mobile x populated/invalid/forbidden x light/dark。

## Gates

- Base lock、跨模块 typed capability、新 API/schema 和共享组件 drift。
- 外部生产资源不在本包内创建或修改。

## Reconciliation 2026-11-07

The required Base primitive is present in official v0.18.6. This resolves the prior component-release dependency only; all implementation and acceptance criteria above remain open. See the parent EXECUTION_QUEUE.md and current execution evidence.

## Current implementation evidence

See `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md` for implemented Deploy scope, exact verification and remaining gaps. The packet remains in-progress.

## File distribution consumer finding

New file distribution task snapshots no longer duplicate plaintext credentials; encrypted execution rows and caller-owned input are preserved and covered by real MySQL persistence regression. This bounded integrity repair does not add the missing DataScope/stable-ID provider/selection binding to file distribution. Historical plaintext snapshots and credential cleanup require a separate reviewed migration; no historical data is mutated.

## FileDist unsupported-scope containment

All public FileDist routes in both aliases now require an explicitly IAM-resolved all scope or the trusted IAM administrator flag. Missing, empty, unknown, self, department, department-with-children and custom scopes return permission.denied before FileDist handler parsing/I/O/dispatch. FileDist is intentionally unavailable to restricted-scope roles until ownership integration is reviewed and implemented; do not widen role policies to work around this. Trusted internal worker/service methods are unchanged.

The route-wide denial matrix, explicit all/admin admission, sibling-route isolation and actual Gin/MySQL scheduled-create/read/denied-mutation persistence checks pass. JWT/Casbin/IAM-resolution, live revocation, browser forbidden recovery and complete O1 target-selection acceptance remain separate gaps. The supplied-source reviewer timed out and has not approved this delta.

### Next implementation prerequisites

1. Review additive, preserving business migrations for numeric task/file owners, department/BizScope ownership and stable CMDB target IDs. Resolve shared/default storage administration separately from file ownership. Do not map string usernames to numeric owner IDs or automatically grant access to legacy records; retain unknown legacy ownership for an explicit migration assessment.
2. Inject the existing provider-owned TargetSelectionCapability through business assembly and transaction binding. Consumers must not query CMDB/BizScope tables or copy provider logic. Decide how execution credentials and verified SSH endpoints are bound to the selected stable IDs without trusting client addresses.
3. Create/start/retry must re-resolve selection, DataScope, state and scopeVersion inside the committing transaction, validate source-file access, save credential-free source/exclusion/time/count facts, and dispatch only after commit. Scheduled execution and live permission revocation need an authoritative actor-policy resolution contract, not a captured creation-time scope treated as current authority.
4. Verify scope/owner changes, unknown legacy records, source-file denial, target/address/version drift, failed transactions and zero worker effects before adding the UI consumer. Full O1 remains in-progress.
