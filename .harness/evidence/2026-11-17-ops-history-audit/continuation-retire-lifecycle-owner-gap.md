# Retire lifecycle — preserved owner acceptance gap

Owner: business/service lifecycle + business/deploy execution. Parent: `2026-11-16-ops-pilot-delivery`; current follow-up under G003, with C02/C05 runtime evidence required before functional acceptance. Status: open / WATCH. No task or historical archive is closed by this record.

Independent sources: `/root/history_scope_review` code review and `/root/c01_architecture_round2` contract analysis. Coordinator persists the findings; neither report ran SSH or modified source.

The original [lifecycle packet](../../../docs/harness/tasks/2026-08-17-ops-resource-service-state-machines.task.md) explicitly separates uninstall and retire and states:

> retire instance | stopped/uninstalled + no active lease | retired | retired | conflict

That packet is Superseded. The active [Service design](../../../docs/designs/BUSINESS_SERVICE_MODULE_DESIGN.md) retains Service-owner state commands and treats CMDB host components as a compatibility projection. The current Service Retire validator permits unknown/stopped/failed and does not itself check leases; it is a broader implementation than the original matrix.

Current generic/component/fixed renderers distinguish only uninstall from installation; Retire falls into installation. Empty successful manual summaries likewise upsert an installed component, while Service completion writes retired. The discrepancy existed when Retire was introduced (`949f47bc`, 2026-08-18, PR #112); it is not an HTTP/factory regression.

Retire means lifecycle termination after stopping/uninstalling, based on the above evidence. Mapping Retire to Uninstall would change business meaning and can delete data roots in fixed templates. It is not an authorized repair.

Smallest safe follow-up: reject unsupported Retire executable package/component/template plans and inappropriate manual success projections, preserving old records/snapshots; then implement an independent Service-owned lifecycle coordinator with stopped/uninstalled and no-active-lease prerequisites, owner locks/CAS and immutable change record. Do not add software removal to Retire. Reuse explicit Uninstall for software removal.

Required evidence: deterministic action admission/old-snapshot tests; no command or CMDB component mutation on rejection; owner/lease race checks; current UI and functional flow. No production, destructive cleanup, final acceptance or external-provider validation has occurred.
