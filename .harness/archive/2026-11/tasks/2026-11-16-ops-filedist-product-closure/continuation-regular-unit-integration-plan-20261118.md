# E01 FileDist regular-unit integration plan

Parent: `2026-11-16-ops-filedist-product-closure`; implementer `/root/e03_claude_partial_closeout`; business/filedist, L2 inherited.

Reuse the UI executor's actual regression fixture only after it announces a final freeze. Preserve its original evidence fixture, runner and real failed runs. Copy the frozen suite into `frontend/tests/unit/business/filedist/FiledistLifetime.test.ts` and adjust only source imports needed for the standard frontend Vitest root. Add exactly this file path to `business-overlay.json`; preserve the earlier C03 declaration and all other ownership entries.

The UI executor owns production Filedist sources and the original evidence fixture; this lane owns new standard unit files and the narrow manifest declaration only. Do not edit production, backend, shared UI/config/locales, old fixtures, global worker/process settings or dependencies.

Impeccable acceptance scope: progress/detail request activation, canceled/stale requests, action permission/mutual exclusion, safe resolved payload/schedule, and file-picker activation/search behavior supported by the final frozen fixture. No visual layout changes or desktop/mobile rendering are performed by test integration; final rendered evidence remains coordinator-owned.

Validation: native Node with existing hidden-child preload, regular Vitest config and default unit include; no shell shim, include override or claim that preserved manual runs are canonical-suite evidence. Require all frozen fixture cases to pass with zero skips. Run scoped ESLint and relevant targeted TypeScript, preserving actual diagnostics and any transitive baseline gap. Run overlay checker without exemptions and record hashes for the new test/manifest and preserved fixture/production bindings.

Stop: regular test and manifest declaration are frozen, fresh results and exact evidence boundaries recorded; no entire E01 approval, backend lifecycle/provider/partition proof, original role/current runtime or human acceptance inferred.

## Authorized fixture completion after first verification

The first byte-identical canonical copy passed 28 cases but strict TypeScript reported TS2352 for an incomplete DeployTargetResolution mock. The original fixture remains frozen; the UI owner authorized completing the canonical mock. Supply resolvedAt/sources and required host status/businessScopeId/source, remove the redundant cast, preserve every assertion/import/case, and retain the initial actual failure. Re-run the same standard unit suite, strict targeted TypeScript and scoped gates; compare original/regular contents allowing only those typed-data completions. This changes no production contract, dependency or shared configuration.
