# E03 registration-only continuation plan

Parent: `2026-11-16-ops-helm-capability-truth`. L2; implementer `/root/e03_claude_partial_closeout`; independent review and whole-package acceptance remain with the coordinator.

Scope: retain Claude's partial `helm.go` / `helm_test.go` implementation, normalize new registrations to `registered`, preserve legacy stored status as provenance with response-only `effectiveStatus=registered`, and state the capability boundary in both README languages. No frontend Helm consumer exists in the current business K8s source, so no UI source changes are planned. No new dependencies, migrations, Chart execution, shared/platform/system edits, provider access, service restart or publication.

Claude supervisor receipts remain unchanged: initial turn-limit failure with zero writes; resumed run exit124 / timedOut=true after600s, with partial source edits but no verification/handoff. This continuation does not convert those attempts into success.

Minimum complexity: reuse the current registration service and standard JSON/HTTP tests. Keep the historical database column default and raw rows unchanged; Create explicitly supplies the registration status. Tighten existing HTTP regression assertions for successful envelopes, newly created record identity and complete list rows. Correct comments and formatting rather than add another status model.

Verification before handoff: scoped gofmt, package compile with no tests selected, package build, vet, gopls for both owned Go files, scoped diff check and source hashes. The three actual MySQL tests are `TestCreateForcedMisleadingStatusIsRegistrationOnly`, `TestListResponseNormalizesLegacyStoredStatus`, `TestHelmRegistrationCapabilityBoundaryHTTP`; the coordinator owns their mandatory-DSN Windows CGO/race run, the full K8s/full backend suites, official overlay rebuilding and current-candidate runtime evidence.

Stop: the registration implementation/docs are locally verified and frozen for central database/race and independent review. E03 remains unaccepted until its original runtime/capability-boundary criteria and human gate are evidenced; actual Chart installation/upgrade/rollback/uninstall remain deferred and unavailable.
