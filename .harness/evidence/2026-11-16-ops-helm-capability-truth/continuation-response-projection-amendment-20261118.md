# E03 response projection amendment

Parent: `2026-11-16-ops-helm-capability-truth`; implementer `/root/e03_claude_partial_closeout`; observed UTC `2026-11-18T07:37:13Z`.
Plan: `../../tasks/2026-11-16-ops-helm-capability-truth/continuation-response-projection-plan-20261118.md`.

The first `continuation-registration-handoff-20261118.md/json` and its central `continuation-c01-e03-frozen-green-20261118-01.log` remain historical. That candidate normalized only `effectiveStatus`; legacy main `status` still returned `deployed`, so it did not satisfy older consumers or the stronger original response requirement.

## Grounded RED before repair

The new HTTP regression was added while production remained at SHA256 `3cbe7f56ca38c895ad41b679b611873521568e5be425b26a794324712d01264e`; RED test SHA256 was `cd4685aeaae433cb342ec809ee969ee593d4bae3ba907356e27efaa8d8f36946`. The coordinator ran actual disposable-MySQL Windows CGO/race using required DSN, readonly modules and the unchanged ten-minute deadline:

`go test -race -count=1 -timeout=10m ./modules/business/k8s/helm -v -run '^TestHelmResponsesProjectRegistrationStatusAndPreserveStoredStatus$'`

Receipt: `../2026-11-17-ops-history-audit/continuation-e03-status-projection-red-20261118-01.log.receipt.json`. Actual exit1, package0.136s,745 backend files stable during the run. Both create and legacy-list failed: create omitted storedStatus; legacy list returned main status deployed and omitted storedStatus. Raw legacy database preservation remained required. Production repair started only after the coordinator released this RED freeze.

## Repaired behavior

List captures the original persisted status in response-only `StoredStatus`, then projects main `Status` and compatible `EffectiveStatus` to registered. Create explicitly stores registered and returns all three fields as registered. GORM ignores StoredStatus/EffectiveStatus; no legacy status, column default, schema or migration is changed.

Earlier API assertions now expect the strengthened projection, with independent raw-DB reload assertions retained. The added test checks both the HTTP main field and raw provenance and verifies the legacy database row still stores deployed. Both README languages reflect this exact contract and retain the unavailable Chart install/upgrade/rollback/uninstall boundary.

## Current local verification and freeze

| Check                                                         | Actual result                                          |
| ------------------------------------------------------------- | ------------------------------------------------------ |
| gofmt on the two Helm Go files                                | Exit0.                                                 |
| `go test -mod=readonly -run '^$' ./modules/business/k8s/helm` | Exit0, package0.041s; compile only, no tests selected. |
| `go build -mod=readonly ./modules/business/k8s/helm`          | Exit0; independently rerun, no output.                 |
| `go vet -mod=readonly ./modules/business/k8s/helm`            | Exit0; independently rerun, no diagnostics.            |
| gopls check on both Helm Go files                             | Exit0; no diagnostics.                                 |
| scoped diff whitespace check                                  | Exit0.                                                 |

The adjacent JSON binds the four revised source/document hashes. Current actual MySQL GREEN is coordinator-owned and pending at this receipt; all four Helm groups must run on this final frozen candidate. No actual GREEN or whole-package approval is invented here. Full K8s/full backend race, official rebuild/overlay/drift, original frontend checks/smoke, current runtime and human acceptance remain separate gates. No UI source changes/rendering or live Chart provider action was performed.
