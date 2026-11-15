---
title: Sealed Velero alias v7 exact native ReplicaSet spec successor
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-11
---

L2 parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
This bounded business acceptance-tooling repair preserves platform/system and product ownership.

The sealed v6 counterexample admits ReplicaSet `spec.minReadySeconds=120` and an unknown `foreignSpecField` while allowing the Deployment revision annotation. Actual sealed-v6 execution reproduces both offline. The fresh v7 successor requires exactly `replicas`, `selector`, and `template`, with integer `replicas=1`, at both annotation admission and pod-state checks. The reviewed Deployment omits minReadySeconds; its native zero is omitted. Explicit zero and nonzero extra fields both refuse. Existing selector, full original template, UID/generation, native owner/revision, inventory and policy gates remain mandatory. No new provider policy allowance or dependency is introduced.

Request `continue-velero-alias-v7-request.json` SHA256: `6bba684f283d6eca328ffe3e66e46391db67ae8606b634941183ea5a99e83ecc`. Operation `20261111t030005219177`.

| Source                                                       | SHA256                                                             |
| ------------------------------------------------------------ | ------------------------------------------------------------------ |
| `.local/ops-acceptance/continue-velero-alias-v7-contract.py` | `c33a31f6d6af824a9fdff7560433a6f4a6a2e617ae4f93cc33fa12c7b9e5a2f5` |
| `.local/ops-acceptance/continue-velero-alias-v7-remote.py`   | `637368b547c41c0e11f234873998defe297435513aa2cccef17fddf1d82dfb71` |
| `.local/ops-acceptance/continue-velero-alias-v7-root.py`     | `a9ec79669b0ff5733842319a618415741bfa6b2ef30848d6762eac1936e347d3` |
| `.local/ops-acceptance/continue-velero-alias-v7-generate.py` | `541c6ab85957a7b5b973a6e6ba30993cec46cf7f55cf8d38b6db6782b73aa870` |
| `.local/ops-acceptance/continue-velero-alias-v7-test.py`     | `da3cd46b5693b8dce507a064c693b5833a24513b7838d05f1b5699eb4fc4122a` |

The freshly exclusive request protects all 347 historical assets, including exact v6 request/all five sources. Streaming actual hashes before/after sealing match all retained files. Existing large binary validation remains under the original reviewed artifact gate; the successor reuses the existing inherited-seal check. The expected remote baseline stays exactly37 files with the two uploaded v5 helpers; v6 was never executed remotely. Two new v7 helper uploads remain exclusive. No old generator ran, no failure or historical review was overwritten, and no remote operation occurred.

`continue-velero-alias-v7-verification.json`: 331 executions/278 unique, zero errors/failures/skips, 52 forged-evidence denials. Includes273 inherited v6 unique checks plus5 new tests and expanded pod-state subtests. Actual Pyright1.1.414 LSP returned zero errors/warnings on all five current sources; five memory compiles pass. Root `check-inputs` exits0 with exact request/five sources,22 owned objects and37 remote baseline files. Checked sources and all v6 bytes remained stable.

Both new independent CODE APPROVE and ARCH APPROVE/CLEAR must bind this exact request/all five hashes before connection, using `continue-velero-alias-v7-report-format.json`. Prior v6 ARCH approval and CODE request-changes remain history and never authorize v7. Approval scope is tooling only; product/P0 acceptance remains open.

Root command prefix:

```powershell
$v7Tool = '.local/ops-acceptance/continue-velero-alias-v7-root.py'
$v7SHA = '6bba684f283d6eca328ffe3e66e46391db67ae8606b634941183ea5a99e83ecc'
python -B $v7Tool check-inputs --request-sha256 $v7SHA
```

After both exact approvals, execute **diagnostic → repair → validate → cleanup** separately. Verify each emitted artifact hash and actual receipt: exit0, `remoteExecuted=true`, `sourceHashesStable=true`, and matching `rootRequestSHA256` before using it as predecessor. Use `--receipt <artifact> --sha256 <hash>` for each subsequent phase. Failed or inspection-only receipts do not permit continuation; inspect the exact failure via `inspect-failed`, preserve uncertainty and never replay. Never regenerate this request.

Actual Ready/BSL/native alias/S3/BackupRestore/persistence/cleanup and independent tooling reviews remain pending with Root. Full postseal request-selection recheck is recorded separately when completed; all current test evidence is finite offline tooling evidence.

Gate Outcomes: independent CODE review detected the native ReplicaSet spec gap; exact-key/type regressions catch known and unknown drift. The initial local focused failure was the default4MiB capture bound on an inherited binary, not changed bytes; the successor now uses the existing inherited-seal validation pattern, preserving original artifact validation and all expected hashes. That failed local receipt remains retained under owned `.local`.
