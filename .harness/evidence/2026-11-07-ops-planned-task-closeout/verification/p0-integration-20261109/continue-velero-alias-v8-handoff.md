---
title: Sealed Velero alias v8 exact native serialization successor
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-11
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
This change is scoped to business acceptance tooling.

Actual sealed v7 failed before alias/provider/S3 mutations. Reviewed read-only observation and value controls isolate exactly two Deployment differences: missing `spec.template.metadata.creationTimestamp` when expected null, and extra `spec.template.spec.containers[].env[].valueFrom.fieldRef.apiVersion` equal to string `v1`. The actual control binds exactly one native fieldRef, matching layout/UID, 23 GETs, one reviewed dry-run, zero mutations and stable sources. Kubernetes v1.33.0 and API/apimachinery v0.33.0 official sources establish the exact `v1` default and null/omitted timestamp serialization; their bytes and actual control are protected by the new request. No other missing/null or nearby API-version path is admitted. Input objects are never rewritten. The v7 exact three-key/int1 ReplicaSet spec, UID/controller/full template, original policy and inventory gates remain mandatory.

Request `continue-velero-alias-v8-request.json` SHA256 `30c929d01d33c80e8419984436802c92af84362a1fbba332fa7a5b226582368b`; operation `20261111t040626789509`.

| Source                                                       | SHA256                                                             |
| ------------------------------------------------------------ | ------------------------------------------------------------------ |
| `.local/ops-acceptance/continue-velero-alias-v8-contract.py` | `dfb05b10db25a85f80a54f0426e603288ff147ba7b972cb98479799461912a31` |
| `.local/ops-acceptance/continue-velero-alias-v8-remote.py`   | `19b9d25d67eb32589b8a2b8c8aa9db59a8246f16e5a97fe34f89ab1c8733d026` |
| `.local/ops-acceptance/continue-velero-alias-v8-root.py`     | `77088fd9d3d56b7a5abdd6c0dcf20411d753fc9cc8f5c4b0883d470ec556f21e` |
| `.local/ops-acceptance/continue-velero-alias-v8-generate.py` | `e38d428cdd78711707257ec958afeddbaa0c98166e110808e881aca6b9ebb3e0` |
| `.local/ops-acceptance/continue-velero-alias-v8-test.py`     | `f4bbd8ad6d18019a748d5f8693c6ae1eeb2bd39c7d330b80fb43cbcff87ea214` |

The request protects 367 historical assets including all v7 sources/request/reviews/failure/inspection, both native observations, control helpers and official source bytes. Streaming pre/post seal hashes match all inherited assets. Exactly two completed v7 helper uploads are adopted from its failed diagnostic and COMPLETE inspection into the 39-file baseline; neither receipt grants continuation/replay. Two fresh v8 uploads remain exclusive.

`continue-velero-alias-v8-verification.json`: 344 executions/291 unique regressions, 13 new v8 checks, 52 forgery denials, zero failures/errors/skips. Sealed v7 rejects both exact native counterexamples; v8 accepts only the exact observed semantics and refuses wrong values, nearby paths, forged native controls, extra RS fields and tampered v7 upload/inspection metadata. Five memory compiles and actual Pyright1.1.414 five-surface LSP pass with zero errors/warnings. Actual Root check-inputs exits0 and selects this exact request/current five sources/22 owned objects/39-file inventory.

Both fresh independent CODE APPROVE and ARCH APPROVE/CLEAR must bind this exact request/all five source hashes before connection, using `continue-velero-alias-v8-report-format.json`. Old v7 approvals and failed/inspection-only receipts remain history. Review/P0/live Ready/BSL/alias/S3/BackupRestore/persistence/cleanup acceptance is pending with Root.

Root uses `.local/ops-acceptance/continue-velero-alias-v8-root.py check-inputs --request-sha256 30c929d01d33c80e8419984436802c92af84362a1fbba332fa7a5b226582368b`. After exact fresh approvals execute diagnostic → repair → validate → cleanup separately, verifying each actual receipt's exit0/remoteExecuted/sourceHashesStable/request binding before continuation. A failure requires exact inspect-failed, preserves uncertainty and forbids replay. Never regenerate this request.

Gate Outcomes: the actual reviewed v7 runtime gate caught native default serialization differences; fresh offline counterexamples and pinned official/actual value controls bound the smallest successor. No gate waiver, source override, original install replay, provider policy widening beyond the two exact observed paths, dependency or remote v8 operation was used. Postseal full request-selection recheck is recorded separately.
