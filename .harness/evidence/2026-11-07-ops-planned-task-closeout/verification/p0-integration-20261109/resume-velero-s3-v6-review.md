---
title: Local self-review of pinned Velero v6 compatibility recovery
doc_type: Evidence
layer: business
status: Active
updated_at: 2026-11-10
---

No unresolved local finding identified in the scoped candidate. Independent
root source/request review and real execution remain required. Local mocks do
not establish controller, Backup/Restore or P0 acceptance.

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.
Evidence: [v6 handoff](resume-velero-s3-v6-handoff.md),
[commands](resume-velero-s3-v6-commands.json),
[source comparison](resume-velero-s3-v6-source-final.diff),
[request](resume-velero-s3-v6-root-request.json).

Reviewed chains: sealed local files -> explicit request hash -> exact retained
remote baseline -> exclusive new helper upload -> corrected official dry-run
and live five/mirror guard -> exact successful predecessor -> remaining sealed
v4 install tail -> original actual driver validation -> stronger fixture
namespace guard plus original UID-precondition cleanup.

The original tail AST is preserved except for the reviewed manifest assignment;
original validation and remote fixture adapter function code is unchanged
except the v6 dispatch name. Fixture apply/assert-owned/storage assertions are
unchanged; prepare/cleanup gain store relationship checks and namespace DELETE
gains full discovery/LIST checks. The restored target ConfigMap requires exact
original fixture content, not only a name or owner label.
Additional restore labels are limited to the two exact original Velero
backup/restore attempt names; local tests cover plain/finite labels, wrong
content, unknown labels and foreign restore names. No live restore is claimed.
No new dependency, platform/system behavior, public interface, SQL source or product assertion was
changed. No credentials, remote API or production state were accessed here.

Trust boundary: only the pinned, live controller-derived slice can be recorded
separately. It is never adopted into APPLIED or directly deleted. Exact
Service/Endpoints UIDs/specs and ownerReference booleans remain mandatory. Unknown
objects, additional slices, source changes, extra bytes, wrong optional retained
v5 helper seals and stale/failed predecessors fail closed. The request explicitly
accounts for three observed v5 uploads and three possible exact original helper
copies; it cannot refresh arbitrary remote files. The helper import dependencies
remain sealed existing files. No new recursive dependency/hub or product cycle
is introduced; the additional namespace guard calls make trust checks stricter.

Residuals: sequential Kubernetes LISTs and precondition checks cannot provide an
atomic cluster snapshot against concurrent foreign writers. Namespace DELETE is
checked immediately before its original UID-precondition DELETE; root must
keep the authorized isolated acceptance lane. The strict mirror contract can
refuse a naturally recreated mirror or previously unobserved API metadata. Such
a refusal is deliberate and requires read-only root inspection plus a new
sealed narrow request. Runtime-only timeout/headroom/provider behavior remains
unproven by local mocks. Provider/native data cleanup is outside this request.

Local mock issues corrected during review: synthetic extra objects lacked UIDs,
a Windows mock tried to stat POSIX `/opt`, and a mock resourceVersion used a
nonnumeric placeholder. These were test-fixture defects, not reasons to weaken
the guards. Each earlier request candidate is retained under its content hash.

## Machine Readable

```json
{
  "taskId": "2026-11-07-ops-planned-task-closeout",
  "parentTaskPacket": "docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md",
  "reviewType": "self-review",
  "independentReviewPending": true,
  "reason": "Generator scope prohibits delegation; root independently reviews and executes",
  "scope": "new local v6 acceptance helpers and current EV only",
  "findings": [],
  "rootRequestSHA256": "d89a0624f6fa9bae53371dd37af9d9550ebb8c227e8334f04ae108d1e8389e08",
  "v6TestsPassed": 46,
  "v5TestsPassed": 32,
  "v4TestsPassed": 25,
  "protectedOriginalFilesStable": 156,
  "newHelpersSyntaxPassed": 6,
  "graphChecks": {
    "toolAvailable": false,
    "sourceDerivationReviewed": true,
    "criticalBoundary": "pinned live ownership and explicit successful hash-bound predecessors before remote mutation",
    "newProductCycle": false
  },
  "remoteExecuted": false,
  "credentialsRead": false,
  "realControllerVerified": false,
  "backupRestoreVerified": false,
  "P0Approved": false
}
```
