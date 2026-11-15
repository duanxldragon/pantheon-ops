## Summary

**ARCH `CLEAR` for the sealed Argo adoption source and deployment helpers.** No actionable architectural defect was found within this lane. The artifact is ready to progress to the already authorized reversible test deployment; real readiness, authenticated product acceptance, O3/D1/P0 completion and release remain downstream.

This review covers logical changes inside the business-domain Argo package and four local helpers. CODE disposition remains separate.

## Analysis

- **Capability and dispatch contract holds.** Negotiation requires exactly six compatible fields and rejects duplicates/unknown fields. Submission constructs only distinct reviewed Sync/Rollback POST routes, with UID/RV/revision/marker and native scope fields; rollback preserves ID `0` and supplies neither source nor infos. Evidence: [protocol.go:53](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_reviewed_protocol.go:53), [client.go:271](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:271), [wire tests:85](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_adoption_test.go:85).

- **Review identity and current authority remain bound.** The private ticket carries protocol/history ID/configuration fingerprint/UID/RV/diff; execute compares them before reservation. Final authorization follows the last capability read immediately before submission. Evidence: [preview.go:85](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:85), [action.go:77](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:77), [action.go:162](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:162).

- **Uncertainty does not redispatch.** Reservation commits before dispatch; exact-ticket recovery precedes expiry/provider reads and checks current plus original authorization. Unconfirmed HTTP200 remains Unknown; terminal reconciliation requires matching UID/revision/unique marker and valid terminal timestamps. Evidence: [action.go:58](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:58), [action.go:231](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:231), [client.go:310](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:310), [reconciler.go:160](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_reconciler.go:160).

- **Deployment side effects are constrained.** The runner verifies seals before transport and requires peer `115.190.251.143`. The helper checks inspected directory device/inodes, namespace ownership, Deployment UID/spec, exclusive staging, bounded ELF hash, immutable flag and read-only mount. It preserves the original spec except the declared mount/argv/node/owner delta and carries the original UID/RV through dry-run and one actual Update. Evidence: [runner:46](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-root-run.py:46), [runner:68](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-root-run.py:68), [deploy.py:63](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:63), [deploy.py:123](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:123), [deploy.py:160](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:160).

- **Staged rollback works without an install receipt.** Sealed before/plan artifacts support reversal after lost installation response; changed spec/owner/artifacts are refused. Evidence: [deploy.py:250](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:250).

### Old-finding mapping

| Earlier finding                            | Current disposition                                                                                                                                                                                       |
| ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PUT authorized only the old target         | Resolved: current/proposed authorization occurs under the write lock — [service.go:410](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_service.go:410).       |
| Kind-only namespace permission             | Resolved: shared Group/Kind allowlist — [preview.go:365](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:365).                                      |
| Legacy webhook compatibility               | Explicitly fail-closed; automation is not restored by this adoption — [action.go:21](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:21).            |
| Caller could not recover its exact result  | Resolved by ticket-hash/private metadata recovery — [action.go:235](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:235).                            |
| PATCH HTTP200 without persistence evidence | Rejected as Unknown; regression retains one dispatch — [confirmation test:20](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_submit_confirmation_test.go:20). |

## Root Cause

The earlier architectural gap was incomplete binding between reviewed intent, provider acceptance and recoverable caller identity. This adoption connects those boundaries through strict negotiation, native conditional submission and durable exact-ticket recovery.

The remaining product gap is separate: `requestActor` still omits originating SessionID, and live tests inject authority/client replacements. Evidence: [requestActor:21](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:21), [live test:48](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_final_live_test.go:48).

## Validation and exact hashes

Independently checked all **31 current file hashes**, receipt/log hashes and binary identity; no mismatch. Recorded race independently counts **181 non-group passes, 22 grouping parents, two live skips**. Fresh read-only execution passed six existing helper guards and seven pure-memory rollback scenarios. These prove guards, not native readiness.

Complete exact source/helper hashes are in [artifact-seal.json:5](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/argo-adoption-20261110/artifact-seal.json:5).

```text
Artifact seal:
08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec

Verification receipt:
051e9fd95eead702b16da2e6092fc3f4bafc8d4129ae070089741aac11444d97

Linux ELF, 206041319 bytes:
cd68356983230edfa0c013235eccba4381a5fbe00d7076f47b7d4b7b8578138b
```

All helper grammars parse as Python 3.8; the local runner requires Python 3.9+ for `Path.is_relative_to` ([runner:54](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-root-run.py:54)). Linux ioctl/filesystem/ABI and retained 192Mi readiness were not exercised.

## Recommendations

1. **Proceed with the exact sealed test deployment — low preparation effort.** Record actual inode/immutable/mount identity, readiness/RSS/restarts and exact rollback restoration.
2. **Persist the seven rollback guard cases — low effort.** Existing helper tests omit rollback; this is a coverage improvement, not a demonstrated implementation defect.
3. **Complete originating-session/HTTP/audit and reviewed live-controller gates — medium effort.** Preserve lowercase SHA1, different-source rollback rejection and overwritten-evidence Unknown; demand no additional provider capability.

## Architectural Status

`CLEAR` — scoped source and helper readiness only.

## Trade-offs

| Choice                          | Benefit                                 | Cost                                                   |
| ------------------------------- | --------------------------------------- | ------------------------------------------------------ |
| Strict native reviewed routes   | Conditional acceptance without fallback | Stock providers remain unsupported                     |
| Exact durable recovery          | Prevents duplicate dispatch             | Overwritten provider evidence can remain Unknown       |
| Same-source historical rollback | Honest, reproducible preview            | Different historical configurations remain unavailable |

Stop condition met: findings and disposition are grounded in opened files and stable artifact evidence. No files were written, remote actions performed or upstream tree rereviewed; root retains report-saving and downstream execution ownership.
