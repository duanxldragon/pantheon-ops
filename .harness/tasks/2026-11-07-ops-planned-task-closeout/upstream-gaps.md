# Base-first follow-up requirements

## Business permission to API binding

Official Base v0.18.6 stores CMDB import/export permission keys but only maps system permission keys to routes in `backend/modules/system/iam/permission/permission_workbench.go`. Role saves consume that table in `role_policy_sync.go`. There is no provider-owned business registration extension in the consumed snapshot. Key-only CMDB CSV runtime: view-role negative check passes, export/import positive checks return permission.denied (2/3 failed). Explicit route grants pass 3/3. Evidence: current `cmdb/import-export-permissions-red.log` and `cmdb/import-export-permissions.log`.

Required Base change: expose a public typed permission-to-route registration contract, use it for both role synchronization and workbench coverage/remediation, and validate live route existence. Ops then registers provider-owned routes and consumes an official immutable release. Verify grant/revoke, wildcard policy coexistence, fail-closed unknown keys, tenant restrictions, import/export separation and scoped runtime acceptance. Do not put business route constants in Base or patch shared Ops sources.

## Inherited document packaging

19 broken targets originate in Base documents pointing to Base tasks, evidence or workflows that Ops excludes/replaces. The exact list is recorded in `d5/check-harness-docs.log`. Fix canonical links or consumer-safe release packaging in Base, release it, and rebuild Ops. Do not copy excluded Base evidence into Ops or add a docs gate exemption.

## Operational Dashboard

The consumed Dashboard renders quick-action/domain-overview only. Operational kinds and metadata exist, but there is no released operational renderer/data adapter extension. O6 requires that contract before business widget implementation.

## Business artifact ownership during local smoke

The inherited cleaner already preserves paths in the generated overlay report. The source tree's report preceded the new migration directory, so it did not contain those owned files. The new SQL paths were restored from the verified business assembly and temporarily marked intent-to-add; the final formal reconstruction refreshes derived ownership. This is a stale local generated projection, not a missing Base cleanup extension.

## Shared operational component contracts

Rendered dark ExecutionStepRail uses the browser's default white button surface, because the shared `.execution-step-rail__step` rule does not set background/text tokens. The shared Tag renders raw enum status text without a localized status-label contract. Evidence: workbench-combined-final dark Deploy desktop screenshot. Fix both in Base, test dark/light/keyboard/contrast/i18n, release and rebuild Ops. Do not override the shared component in business CSS.

The released ConditionBuilder is a read-only tree: its public props have no onChange, and add-rule is disabled. O4 structured editing needs a Base-owned editing contract, validated compiler boundary and official release. The raw Loki lane remains usable and separately tested.

## Remaining business contract gates

O1 requires provider-owned source/exclusion/manual resolution, scope version and audited submission revalidation; existing CMDB typed reads and start-time target checks are insufficient. O3 requires immutable intent/current-target version binding and conflict-on-change across Deploy/K8s/Argo CD. The parent handoff explicitly gates new schema/public APIs and final acceptance. No client-generated version or external mock is treated as that contract. The Deploy selection and preview portions are now implemented; other consumers, context propagation and full acceptance remain open. See current summary and independent review triage.

## Inherited legacy review registry packaging

The final strict scoped review validates this task review but reports 8 stale legacy allowlist entries whose archived Base review artifacts are excluded from the consumer snapshot. Exact entries: verification/verified-scoped-review.txt. Fix consumer-safe registry/archive packaging in Base and consume an official release. Do not copy missing Base evidence or remove entries in Ops to waive the checker. This is separate from the 19 inherited docs-link failures.

Base source continuation: permission registry, optional shared editing/theme and all 19 canonical link repairs are implemented and locally verified; release/consumer validation remains open. Dashboard and owner-aware legacy packaging still require implementation.

Latest source: Dashboard bounded provider rendering and Base-owned archive-projection metadata are implemented and tested. Current Ops remains on old release; registry, UI, Dashboard and packaging consumption stays open. Request-context and hostile selected-update follow-ups are now fixed and included in passing full CGO race.

## FileDist ownership/target integration prerequisite

FileDist current rows store string CreatedBy/UploadedBy and address-only targets; no numeric owner/department/BizScope or stable CMDB target identity exists. Generic WithDataScope cannot safely authorize these rows. Public routes now fail closed for missing/empty/unknown/restricted scopes, while trusted all/admin scopes retain existing route authorization. This is temporary business-only containment, not full O1 completion. The O1 packet records additive ownership migration, provider injection, source-file access, transactional stable-ID/version revalidation, post-commit dispatch and authoritative scheduled actor-policy resolution prerequisites. Restricted roles cannot use FileDist until these are implemented; do not grant all scope as a workaround. Existing historical credential snapshots and whole-source review remain open.

O3 K8s image review adds POST releases/preview and releases/:id/rollback/preview under the existing authentication/Casbin/DataScope group. Provider-owned create/rollback route bindings must be registered through the reviewed formal Base permission registry release; key-only positive runtime is still blocked in v0.18.6. No broad grant or shared Ops override was introduced. Public signed mutation works with already authorized routes/admin fixtures, but does not establish key-only-role acceptance.
