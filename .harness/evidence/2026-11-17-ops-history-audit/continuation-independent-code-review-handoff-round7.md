# C01 independent code review — handoff round 7

Provenance: coordinator transcription of the independent `/root/c01_handoff_code_review` report, condensed with its verdict and evidence boundaries retained. Reviewed at 2026-11-17T16:38:15.8003445Z. The reviewer was read-only and did not run tests or change runtime.

Recommendation: **COMMENT**. Seven scoped files reviewed; zero new CRITICAL/HIGH/MEDIUM/LOW issues. Static narrow fix PASS; runtime test-build source binding pending at review time. Whole C01, merge and archive approval are false.

Scope: `deploy_service.go`, `cancel_unclaimed_test.go`, `worker_handoff_test.go`, `attempt_ownership.go`, `task_command.go`, `service_link.go`, `target_selection.go`. Prior code rounds 5/6 remain historical COMMENT, not full acceptance.

## Findings and evidence

- Reservation and retirement share the slot mutex (`deploy_service.go:1459`, `:1484`); stale retired pointers retry the current map (`:1476`), and identity deletion protects successors (`:1493`).
- After host drain, once-only cleanup captures pending and retires the original slot before final settlement (`:1553`, `:1556`). Resume during owned-pause commit reply is exercised by synchronous and actual asynchronous dispatch tests (`worker_handoff_test.go:151`).
- Pending admission locks current task/hosts/attempts/physical leases, requires running SSH with unfinished hosts, rejects valid attempts and foreign leases, and binds CMDB then Service reads to the same transaction (`deploy_service.go:1572`, `:1589`, `:1594`, `:1600`). Successor dispatch occurs only after commit (`:1615`).
- Cancel/superseded intent is rejected; existing fenced owner and lease helpers preserve foreign facts (`service_link.go:160`, `attempt_ownership.go:662`, `:689`).
- The old context is checked before each next owned-host retry (`deploy_service.go:1625`). This is a pre-loop guard, not atomic cancellation versus claim.
- The last-running `done` defer covers final settlement and cleanup (`:1504`). The deterministic commit-reply fixture proves map deletion precedes whole worker exit (`worker_handoff_test.go:85`, `:504`).
- Manual Resume does not create a worker; database errors remain observable rather than masked (`worker_handoff_test.go:463`, `:484`).

Two gopls batches across seven files returned exit 0, no type errors. One existing unnecessary fmt.Sprintf diagnostic at `deploy_service.go:4880` is outside the new handoff area. Scoped diff check and three core gofmt checks passed. Native CodeGraph/AST-grep tools were unavailable; installed gopls plus source reading/rg were used, with no claim of AST-grep execution.

The independently inspected `continuation-c01-owned-resume-final-settlement-fixed.log` and receipt show 18 top-level PASS, 43.248s, exit 0, CGO=1, race, required disposable MySQL, and no 1213/ERROR/DATA RACE. The preserved final-settlement RED returned exit 1 because fixture cleanup returned while the commit reply remained pending.

The original GREEN receipt contains no before-build hashes. Reviewer start/end hashes bind static review only. Coordinator execution order supports frozen-source operation, but does not turn post-run hashes into pre-build evidence. A later source-bound rerun must be recorded separately.

Retire lifecycle remains OPEN/WATCH (`continuation-retire-lifecycle-owner-gap.md`). Cross-execution owner changes are owned by the independent architect. Full CGO race, official-lock rebuild/source equivalence and current runtime/original functional acceptance remain open.

## Machine Readable

```json
{
  "taskId": "2026-11-16-ops-service-deploy-integrity",
  "parentEvidenceTaskId": "2026-11-17-ops-history-audit",
  "reviewRound": 7,
  "agentRole": "code-reviewer",
  "reviewer": "/root/c01_handoff_code_review",
  "reviewedAtUTC": "2026-11-17T16:38:15.8003445Z",
  "recommendation": "COMMENT",
  "staticNarrowCodeDisposition": "PASS",
  "filesReviewed": 7,
  "findings": [],
  "approvalGranted": false,
  "wholeTask": false,
  "archiveApproved": false,
  "retireLifecycleGap": "OPEN_WATCH",
  "validation": {
    "goplsExitCodes": [0, 0],
    "typeErrors": 0,
    "scopedDiffCheck": "PASS",
    "coreGofmtDiff": "PASS",
    "reviewerRanTests": false
  },
  "dynamicEvidence": {
    "log": "continuation-c01-owned-resume-final-settlement-fixed.log",
    "logSha256": "12970b598122625a10ba934e70ae698a25f4e044db3177ae5c07840d156916b8",
    "receiptSha256": "03b6f05f60c517e7866dbcb1975b3f91b2970e57b51491dec559ebdc5488b7d9",
    "exitCode": 0,
    "topLevelPassCount": 18,
    "packageDurationSeconds": 43.248,
    "testBuildSourceBinding": "PENDING_AT_REVIEW"
  },
  "sourceHashesSha256": {
    "deploy_service.go": "6d23de28303ce416cae65088ba67db31abe0d5031a5bda27d0983ecb91a52800",
    "cancel_unclaimed_test.go": "b064bbef402afe6fb52fe4bdf3134e25696152eef740df5e50c963130da58a5b",
    "worker_handoff_test.go": "dee06ffc35063d6cc722ebdec63ffc52dd3fa2d08b8937168405049e0162faa6",
    "attempt_ownership.go": "767da276e8686a56046f830b4c5f06946197cb91d563ca188f13b638b7d8680e",
    "task_command.go": "14a944a69e82a768b060a0882c170ab76c4dfbe5c4fb3cab9c4212363bc48f8a",
    "service_link.go": "fd636428774a5cc7985c0906a3dff16bb212119490b9858b9baf740cef1b9e1f",
    "target_selection.go": "ce5041bacf74db0ddf0409f82ff9d0bc62310f1e116fdbae2378365a24f79b0f"
  },
  "sourceHashScope": "Static review snapshot, not test-before hashes"
}
```
