# C01 initial Retire architecture plan review

Provenance: condensed coordinator transcription of the earlier independent architect response, recovered from the native agent's completed status. This is not a verbatim original and does not constitute implementation approval.

Initial plan verdict WATCH. Retire must preserve installed component projection and must not become Uninstall. The Active Service design delegates lifecycle persistence and revision/CAS to the Service owner; the older stopped/uninstalled/no-active-lease matrix is explicitly Superseded. Stopped-only may be documented as a conservative supported subset, not universal proof against unknown/failed.

The previous snapshot had no distinct metadata-only representation, and zero Steps failed resolution. A Manual executor alone cannot authorize frozen executable Retire snapshots. Preserve old reads/cancellation/history and prohibit Update laundering. New metadata records must be distinguishable and testable.

Preserve task -> hosts -> attempts/physical leases -> CMDB -> Service ordering. Do not add physical lease locks after Service locks. Reuse HasExecutingServiceReferences: Start checks while the task is not executing; completion changes the task to success in the same transaction before owner finish. Do not add arbitrary excluded task IDs. Linked executing references and host-wide foreign leases are different policy scopes. Any own metadata reservation requires the same fenced transaction, matching task/host and no running attempt.

Prioritize direct-source RED coverage for executable rejection, component/version preservation and successful Uninstall STOP-to-Retire compatibility. Service PATCH retirement admission and wider host-level lease policy remain open. wholeTaskComplete=false; archiveApproved=false. Follow-up readiness and reference clarification are separately recorded; this historical WATCH is retained.
