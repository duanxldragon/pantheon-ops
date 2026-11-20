# E02 accepted sync and observed outcome

L2 continuation of the original E02 task and manifest. Implementation ownership is limited to backend/modules/business/cicd/argocd and package-local tests. Other workers own C01, C04, E01, E03; preserve their edits. Root owns opsinit wiring, manifests, frontend, candidate integration and heavy verification.

## Facts and scope

SyncApplication currently changes Running history to Succeeded and application to Synced when the remote sync POST returns HTTP200. That response only acknowledges a request. A request error is shadowed and not returned. Reconciler currently copies raw sync/health and can append Succeeded history even when health is Degraded; it does not finish the originally requested Running history. Disconnected servers are skipped, so they cannot reconnect through this loop. StartAll exists but no package-local startup call is present.

Existing models contain no business owner/UID or persisted remote operation token. This narrow repair cannot certify the original scope/legacy-isolation gate or remote recreation identity; no schema, ownership contract, provider policy or shared middleware change is authorized here.

## Minimum approach

Reuse existing client, application/history models and reconciler. Keep accepted requests Running with unfinished history and no successful-sync timestamp. Derive aggregate completion from operation phase plus sync and health: operation success alone while Progressing/Degraded is not deployment success. Raw provider sync/health remain separately visible and do not imply aggregate success.

Finalize only the current pending history. Fetch its identity before remote observation and recheck it with the current application source binding under a transaction before applying observations; discard observations from a superseded request/source. Require an observed operation started at or after a local request before closing its history; older/undated operations fail closed. This conservative timestamp fence assumes comparable provider/local clocks; skew can delay completion and remains a documented external gap. Pending observations have a bounded timeout and become Unknown rather than falsely successful; a later live observation can still update actual application status.

Bound per-request reconciliation contexts, paginate registry scans, retry disconnected servers, reconcile on startup, and make Start/Stop idempotent. Do not create an uncontrolled timer/goroutine per application or weaken existing TLS/SSRF transport.

## Verification and handoff

Before fixes write a DB/fake-HTTP regression against the current reconcileApplication method: existing Running request + provider Succeeded/Synced/Degraded must not become aggregate Synced or history Succeeded. Root runs this RED and later targeted Windows CGO/race integration tests; this worker does not run heavy DB or full race during concurrent writes. Pure observation/lifecycle tests and package compilation are allowed with readonly module loading. Add coverage for accepted, progressing, failure, successful healthy observation, old/superseded observations, timeout, disconnected retry and stop cancellation where feasible.

Provide exact changed files/hashes/test filters and root startup hook. No live ArgoCD, production, credentials, namespace side effects, commit or publication. Narrow implementation completion is not E02 acceptance; original ownership/legacy/runtime/role and final review remain open.

## Implementation review adjustment

Explicit HTTP4xx rejection can finish Failed, while transport/5xx failures can mean a lost acceptance response and must remain Running until observation/timeout. HTTP submission now uses a 10-second child budget inside the existing 30-second transaction context so the DB can persist the result after request timeout. This reuses existing contexts/models and does not create an outbox; acceptance followed by DB commit failure remains an explicit dual-write boundary. Running/Terminating operations can expose old rollout health and remain unfinished until the operation leaves Running. Provider bodies are excluded from returned sync errors and new server-reconciliation logging.

New error keys follow the canonical business namespace; root integrates business.cicd.application.syncInProgress, business.cicd.application.syncObservationTimeout and business.cicd.application.sourceChanged. Root should use aggregate status for readiness. Final package-local source binding, command receipts and remaining gaps are recorded in the continuation-sync-observation handoff under the original E02 evidence directory.

## Independent review amendment: persistent server outage

The actual dispatcher returned before scanning applications whenever server credential/client/probe setup failed. Consequently the direct-application timeout-on-outage test did not prove dispatcher expiry and old Running histories could remain unfinished indefinitely. Add an actual reconcile-level regression with /api/version503 and aged Running history before the repair. Preserve the pre-amendment source/test bytes and binding separately from the earlier source/command receipts. Root owns the disposable-MySQL RED/GREEN runs.

Continue the bounded application scan even when server setup/probe fails, using the existing transaction/source/latest-history fences to expire only aged local pending histories as Unknown without fetching remote applications. Leave recent/terminal histories untouched, preserve disconnected-server retry and original failure classification. New dispatcher failure logs record a stable subsystem/error class instead of arbitrary provider text.
