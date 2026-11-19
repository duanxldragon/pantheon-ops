# Retire metadata contract: independent architecture review

Provenance: Root's condensed transcription of c01_handoff_architecture's supplied report. Narrow metadata-contract CLEAR; overall C01 WATCH, wholeTaskComplete=false, archiveApproved=false. This report predates the separate confirmed async rejection finding; it does not approve that branch.

Create/Update require exact metadata marker, Manual, complete Service association, empty non-null Steps and no caller commands; historical records cannot be laundered into metadata records. Package/template identity is retained without copying commands or defaults. Existing transaction ports, task/hosts/attempts/physical-lease -> CMDB -> Service ordering, legitimate selfreservation and no-worker-attempt guards remain. The real Create -> Preview -> Start -> Complete -> replay test passes without steps/attempts or beginning an owner transition.

Manual Retire validates marker/summary before settlement, performs zero CMDB writeback calls, and preserves software version values. Callback desired/current values are blank. Precision: Service SQL still assigns preserved desired/current/rollback columns in its update map; this is unchanged values, not zero SQL column assignments. Real owner SQL failure rolls back deployment facts. Retry rotates generation without beginning owner or rewriting the persisted snapshot; Uninstall STOP-to-Retire and software removal remain, as do historical read/cancel.

The architect independently matched all 7 current/raw/manifest hashes to GREEN before/after and all 730 manifest entries unchanged. Four Retire groups PASS, 39.079s, exit0; no 1213/ERROR/DATA RACE/FAIL in that targeted log. No DB tests or file/runtime changes were performed by this lane.

Limits: stopped/stopped is this iteration's supported subset. HasExecutingServiceReferences was not newly integrated into Retire admission/settlement; existing per-task attempts and physical-host lease guards are the reviewed scope. Service PATCH, wider global reference/host policy and full runtime/final gates remain open. The newer async rejection branch is BLOCK/REQUEST CHANGES in the separate appendix.
