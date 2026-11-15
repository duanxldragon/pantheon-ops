# SeaweedFS diagnosis: root cause remains unresolved

Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md` (L2).

Existing reproduction remains red: three MinIO modes and ordinary nonstreaming SigV4 PUT return `500 InternalError`; GET returns 404 and isolated comparison fixtures were cleaned. See `.local/ops-acceptance/continue-base-s3-manual-protocol-20261110T091331339761/manual-protocol-result.json:10` and its `http-metadata.jsonl:5`.

One hypothesis was disproved by saved native 4.53 help and storage status: default minimum free disk is 1%, available space is 97.6167%, and volume ReadOnly is false. Installer omits the threshold override (`.local/ops-acceptance/resume-velero-s3-v4-remote.py:315`); saved storage receipt gives ReadOnly=false (`.local/ops-acceptance/continue-base-s3-storage-readonly-20261110T090617303822.json:38`). The extracted help and receipt hashes are in `result.json`.

The checksum classifier is inconclusive: it scans the entire XML body with a `sha256` pattern (`.local/ops-acceptance/continue-base-s3-manual-protocol.py:69`), while the requested object path itself contains `signed-sha256.txt` (`:176`). No isolated XML Message or checksum mismatch is recorded.

The new helper failed during SSH socket connect: PermissionError / WinError 10013 (`.local/ops-acceptance/remote.py:37`). No remote command was submitted. Prior unit journal receipt also records TimeoutExpired (`.local/ops-acceptance/continue-base-s3-readonly-diagnosis-20261110T090747472988.json:11`) with causalEvidence=false (`:16`). No broad journal query was repeated.

Root handoff: run `python -B .local/ops-acceptance/continue-seaweed-diagnosis-topology.py` in an existing permitted session. It reuses connect()/known-host checks, verifies exact unit executable/argv/private listeners, and reads numeric ownership/mount/cgroup metadata, master/volume topology and one-entry original-bucket filer metadata. It attempts at most one exact current journal per journal location, only if <=16 MiB, for the verified invocation with a 4-second timeout; raw messages are never output. The script never allocates volumes, writes objects, restarts or changes caps. Check collection mapping because the original installer sets `-volume.max=1` (`:316`); no collection-capacity cause is claimed without topology/log evidence. Exact original argv remains enforced at `.local/ops-acceptance/resume-velero-s3-v6-native.py:274`.

Repair proposal: none, because no backend cause is proven. Root owns repair, rollback and subsequent ordinary signed-PUT/content-read/absence verification plus the existing real Base S3 upload test. Provider health, Velero success and P0 acceptance are not established by this diagnosis.

Validation: local and embedded Python compile checks pass; local read-only endpoint review passes. Remote helper runtime is unverified. Self-review only under the leaf/no-delegation constraint. Only new diagnosis-prefixed helpers and evidence were added; no product build/test rerun is warranted by these local-only additions. Remote mutations: 0. Gate Outcomes: none.
