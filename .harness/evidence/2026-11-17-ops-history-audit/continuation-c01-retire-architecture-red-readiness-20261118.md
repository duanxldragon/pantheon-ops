# C01 independent first-stage RED readiness

Source: two independent native architect responses from c01_handoff_architecture received in this continuation, transcribed by the coordinator. Scope is test readiness only. Narrow RED-ready=CLEAR; whole Retire lifecycle=WATCH; no source modification or test run was performed by the architect.

The one optional Kind field is a minimal necessary distinction. Only server Create generates metadata_only; exact Retire + Manual + complete Service binding + explicit non-null empty Steps is required. Unknown kind, nonempty Steps, wrong action/executor and missing association are rejected. Old Retire Update cannot synthesize a marker. Preserve reads, cancellation, terminal replay and frozen bytes.

Four RED groups are sufficient for the initial slice: representation/admission/laundering; actual resolver/renderer/remote-step bypasses; completion with no CMDB write, atomic component-report rejection, three-version preservation and no-begin Retry/replay; real Uninstall compatibility. Reuse existing foreign-lease/attempt/stale-report coverage unless the new branch changes its risk. Preserve the old control fixture assertions when changing its legal construction from Update to fresh Create.

Successful metadata completion must skip the entire CMDB write, not merely empty the summary, because the normal package path otherwise upserts facts. It must omit all software versions. Keep Uninstall's existing STOP-to-Retire chain and lease-release ordering; do not add arbitrary task-exclusion parameters.

Reference clarification: TemplateID can be metadata-only even when the referenced template contains Steps, just as PackageID can reference installation commands. Keep ID/name/version for traceability; do not expand or copy Steps/commands/config. Reject caller-supplied executable script/pre/post/health parameters. The architect explicitly narrowed its earlier overly broad wording about executable templates: the standalone reference ID is not execution input.

Stopped-only remains this iteration's conservative source-state support. Service PATCH and wider host-wide lease policy remain open. This evidence approves writing compile-valid RED tests, not implementing or archiving the whole task.
