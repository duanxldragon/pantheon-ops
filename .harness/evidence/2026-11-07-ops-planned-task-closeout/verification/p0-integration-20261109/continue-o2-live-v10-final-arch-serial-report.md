## Summary

**Architectural Status: BLOCK — one HIGH finding in this exact v10 tooling proposal.** The sealed request matches the supplied SHA256 and every recorded pin matches current bytes, but its inventory omits inputs used by the actual Windows race build. This decision does not reopen the retained product-source reviews or approve runtime/P0 acceptance.

## Analysis

**HIGH — Build discovery does not match the verified build configuration.** [contract.py:155](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-contract.py:155) discovers Windows dependencies without `-race`; [verify.py:48](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-verify.py:48) tests and builds Windows with `-race`.

Independent inspection of the existing executable confirms `-race=true`, `CGO_ENABLED=1`, Windows/amd64. FINAL go-list streams instead select `race0.go` and ignore `race.go`. The sealed [inventory:5121](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v10/local-verification-1794295469416895500/source-after.json:5121) pins that non-race implementation. At least five race inputs are absent from both source and tool pins, including:

- `runtime/race.go`, explicitly selected by the [race build constraint:5](/D:/cache/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.windows-amd64/src/runtime/race.go:5).
- `internal/race/race.go`.
- `runtime/race/race_v1_amd64.go`, which [imports the detector package:9](/D:/cache/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.10.windows-amd64/src/runtime/race/race_v1_amd64.go:9).
- That package’s `doc.go` and `race_windows.syso`.

There is **no evidence these omitted bytes changed**. The defect is that repeated inventory equality cannot detect their drift, so it does not establish the required complete source-to-build binding ([contract.py:288](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-contract.py:288)).

Other scoped boundaries hold:

- Root checks same-request CODE/ARCH/Dispatcher receipts before reservation and private imports; reservation precedes runner execution ([root-execute.py:23](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-root-execute.py:23), [contract.py:295](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-contract.py:295)).
- Original v9 driver/tests, 45 assertions and protected runner bodies remain unchanged ([contract.py:224](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-contract.py:224)).
- Cold restart retains abrupt exit75, fresh processes, actual lease waiting and exact crashed-claim checks ([driver.go:610](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-driver.go:610), [runner.py:203](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-runner.py:203), [driver.go:889](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v10-driver.go:889)).
- Both Dispatcher adapters contain verbatim original reports; original report/CLI hashes match. Their declared boundary adds no review or approval ([adapter:3](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-service-transaction-20261110/final-arch-report.bound.md:3)).

## Root Cause

Discovery and verification use different build selectors. Before/after equality proves stability of the non-race projection, while the Windows executable adopts race-specific inputs.

## Recommendations

1. **Small tooling repair:** add Windows `-race` discovery for named-file and linked package-test dependencies; retain discovery needed by non-race checks and Linux builds. Require race-specific source/Syso coverage.
2. **Moderate verification effort:** produce a corrected successor candidate with fresh FINAL evidence and request-bound reviews. Preserve this sealed request, historical inventories, handoffs and concurrent work.

## Trade-offs

| Option                                   | Benefit                                                                      | Cost                                                         |
| ---------------------------------------- | ---------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Rely on existing binary and owner hashes | Strongest counterargument: executed bytes and business sources already match | Does not satisfy the declared complete build-input guarantee |
| Match discovery to actual build flags    | Repairs that guarantee with a bounded change                                 | Requires fresh verification and review                       |

## Validation and Runtime Gap

Independently rehashed **12 candidate, 27 evidence, 188 historical, 65 owner, 5,246 source and 390 tool pins**: zero mismatches. All **5,256 archive members** matched. Six additional pure-memory offline regressions passed. Existing FINAL evidence records seven successful commands, including 11 Python and 15 Go regressions; these were not all rerun.

No files were written, builds/seals performed, or remote/credential operations invoked. Zero v10 runtime cases remain recorded ([receipt:429](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v10/local-verification-1794295469416895500/receipt.json:429)). Public Base identity/IAM/audit, HTTP/browser/SSE and physical VM acceptance remain open ([request:328](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-current-live-v10/sealed-candidate-request.json:328)). Diagnosis stop condition is met.

```json
{
  "lane": "ARCH",
  "decision": "BLOCK",
  "independent": true,
  "subjectSHA256": "f5bdc55e94a56dbeec56429b20189eeedd4cfc5524a85cdf81ce0bd0f294c2a2",
  "requestSHA256": "f5bdc55e94a56dbeec56429b20189eeedd4cfc5524a85cdf81ce0bd0f294c2a2",
  "architectureStatus": "BLOCK"
}
```
