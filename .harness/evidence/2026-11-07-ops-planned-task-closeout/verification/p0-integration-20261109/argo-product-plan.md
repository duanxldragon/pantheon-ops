# Argo product generator scope

Parent: `2026-11-07-ops-planned-task-closeout`; O3/D1; L2; business/cicd/argocd only. Logical separation; no Base/system overrides, SQL, frontend, overlay, lock or ledger changes. Independent final review belongs to the coordinating root.

1. Add regressions for old/unrelated terminal states, uncertain submission/restart, stale caller scope, source/server revocation, provider error/diff redaction and review drift.
2. Reuse Argo native operations with an opaque per-action info marker and Kubernetes UID/resourceVersion conditional patch. Preserve provider historical source for the supported single-source rollback. Persist marker in the existing history message column, project only sanitized business messages, and never resend uncertain actions.
3. Add bounded provider-owned manifest/live diff preview and expiring authenticated review binding actor, immutable target revision, provider identity/version, local configuration and credential references. Recheck current authorization before provider dispatch.
4. Run required isolated MySQL red/green tests, scoped CGO race, vet/build and source hashes. Prepare a finite true-provider test driver using the existing isolated Argo namespace and smart Git fixture protocol. Emit a sanitized executable phase request for root; do not retry restricted SSH or claim provider-only checks as product checks.

Base dependency: released `common.DataScopeReq` is a request snapshot, not an authoritative actor resolver. `DataScopeMiddleware` uses request role keys and caches user departments for five minutes. No public fresh actor/role/DataScope authorization port is present. The module must fail closed without a composition-root supplied authority; it must never query system-domain tables or trust caller IsAdmin to resolve this gap. Route permission registry and shared wiring remain root/Base-owned.

Success: exact matching action reaches terminal on the original persisted row; unrelated state cannot complete it; unknown survives failures/restart and is not resent; denied current identity/source/config cannot dispatch; diff/API/audit omit raw configuration and provider errors. Runtime and unsupported historical-source limitations are recorded explicitly. This is Generator evidence, not whole O3/D1/P0 approval.
