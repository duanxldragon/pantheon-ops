# Pantheon Base v0.18.7

This patch releases the formal-queue foundation contract closeout merged through Base PR #504 under task 2026-11-07-ops-formal-queue-foundation-gaps: provider-owned permission routes (`backend/pkg/permissionroutes`) registered by providers before the fail-closed startup `Seal()`, the permission-cache revocation repair (authoritative owning-tenant query only; the stale platform-only Redis permission-array path and `dropStalePlatformOnlyPerms` are removed, verified red->green), the shared ConditionBuilder/ExecutionStepRail contracts, bounded Dashboard extension points and consumer artifact governance. VERSION is 0.18.7; the separate frontend UI package version is unchanged. Two release-blocking remediations ship with the preparation: the audit-route log values are now emitted through `logging.SanitizeLogValue` (closing the CodeQL go/log-injection alert), and the system smoke helper no longer stalls on Arco's transient loading empty state.

## Upgrade notes

Ops must fetch `pantheon-base-v0.18.7` from GitHub with the standard four-asset hash/tag/manifest verifier, rebuild from the locked official snapshot, and run readonly CGO full race, frontend lint/type-check/build, overlay/drift checks and applicable smoke before replacing the consumer tree. Business providers must register permission routes before startup seal; permission revocation no longer tolerates the stale platform-only cache. No new migration or business change. Retain the previous v0.18.6 lock for rollback.

## Validation status

Preparation validated locally on the PR branch: affected-set CGO race, vet and build; middleware sanitization test; frontend lint/type-check/build/vitest; all doc-harness gates. The remaining Release Gate blockers on main were cleared by code-only SonarCloud remediation (PR #506 fixed all 63 unresolved issues; PR #507 removed the 4 residual issues from that refactor), bringing main to zero unresolved issues with no waivers, exclusions or gate changes. The immutable target commit 3203624e55438866edb951bd74e671eab38d2f16 passed GitHub required checks and the Release Gate Summary (run 43194854386), after which the annotated tag `pantheon-base-v0.18.7` and the four official assets were published and verified by `check:release-assets` locally and remotely. Actual Ops consumer validation (readonly CGO full race, frontend, assembler, migration, standard smoke) will be recorded here after it happens. None of the preparation or publication evidence implies consumption.

## Explicit residuals

One real S3 provider skip has no waiver; maintenance freeze is unsigned. Separate Ops CMDB/O2/O3/O4/D2 product acceptance, QA002/host-import runtime fixture gaps, policy/security/production and legacy-auth followups remain. FR028 oversized audit json_result storage remediation stays deferred.
