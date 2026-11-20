# E03 response status projection amendment plan

Parent: `2026-11-16-ops-helm-capability-truth`; L2; implementer `/root/e03_claude_partial_closeout`.

The coordinator compared the first candidate with the original criterion and found an API compatibility gap: older consumers reading `status` still receive `deployed` for an uninstalled legacy registration. Merely adding `effectiveStatus` and documentation does not satisfy those consumers. The first handoff and its central GREEN remain historical; they do not prove this stronger response contract.

Scope: new HTTP regression before repair, then minimal response projection in the existing model/service. Responses expose `status=registered` and compatible `effectiveStatus=registered`; an independent response-only `storedStatus` carries the unmodified persisted source value. Keep the DB column/default and stored legacy rows untouched. New Create explicitly stores registered and returns storedStatus registered. No new execution, migrations, dependencies or UI source changes.

Order: (1) add `TestHelmResponsesProjectRegistrationStatusAndPreserveStoredStatus` with current production unchanged, freeze test/source hashes; (2) coordinator runs actual required-DSN MySQL/CGO race RED; (3) only after RED receipt, add the projection and update older API assertions to this stronger contract while retaining original DB provenance checks; (4) format, compile/build/vet/gopls and freeze; (5) coordinator runs all four Helm regression groups GREEN and independent review.

Stop: new frozen candidate has a real RED→GREEN source/receipt binding and fresh local diagnostics. Original full K8s/full backend gates, official assembly, current runtime capability evidence and human acceptance remain separate; no whole-package completion or Chart capability is claimed.
