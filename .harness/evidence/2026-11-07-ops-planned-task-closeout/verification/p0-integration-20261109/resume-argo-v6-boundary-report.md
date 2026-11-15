**Architectural status: BLOCK for the current Argo submission path.** The diagnosis is grounded; a bounded repair can proceed to implementation, but neither provider acceptance nor full P0 completion is established.

The smallest correct repair is an **Argo-side conditional action capability that reuses native Sync/Rollback policy and operation construction**, while preserving the existing Ops reservation and recovery semantics.

**Required contract**

Original O3 requires “预览后对象变化必须重新预览,” binds preview and execute to the same intent/version, requires drift to return 409, and explicitly calls for TOCTOU evidence. These requirements predate the current implementation; I checked the committed packet as well as the working copy. See [O3:19](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O3-change-preview-and-risk.md:19), [O3:35](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O3-change-preview-and-risk.md:35), and [O3:43](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O3-change-preview-and-risk.md:43).

Those clauses require a condition enforced at operation acceptance, or an equivalent guarantee. A fresh client recheck followed by an unconditional mutation leaves a TOCTOU gap. **They do not prescribe JSON Patch, three separate UID/version/spec tests, or freezing every managed resource until terminal completion.** UID plus resourceVersion on one conditional Application update is sufficient to protect that Application’s reviewed identity/spec, including ABA.

D1 requires real health/sync/rollback, permission/DataScope/audit/idempotency, and positive/negative/timeout/retry evidence. It does not independently specify CAS or require an unmodified Argo distribution. See [D1:13](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/packets/D1-external-integration-acceptance.md:13) and [D1:27](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/packets/D1-external-integration-acceptance.md:27). Test-server authorization does not waive O3.

**Root cause and official semantics**

- **PATCH cannot submit this operation.** Official `Patch` applies JSON Patch to an Application, then calls `validateAndUpdateApp`, which rereads the Application. `updateApp` copies spec and selected metadata, omits operation, and retries conflicts against fresh reads. Thus the JSON tests are not write-boundary preconditions. See [official Patch:1058](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-server-application-application.go.txt:1058), [validation reread:936](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-server-application-application.go.txt:936), and [updateApp:986](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-server-application-application.go.txt:986). This explains the recorded HTTP200 with unchanged version and absent operation/state/Service: [v4 observation:3555](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v4-evidence/resume-argo-v4-live-result.json:3555).
- **Native Sync genuinely submits an operation, but cannot enforce the reviewed version.** It checks sync windows and sync RBAC, constructs the native operation, then calls `SetAppOperation`. That helper rereads by name, uses `context.Background()`, and retries conflicts indefinitely without reviewed UID/version conditions. See [Sync:1998](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-server-application-application.go.txt:1998) and [SetAppOperation:869](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-util-argo-argo.go.txt:869).
- **Native Rollback restores historical source configuration.** It selects history by ID, copies its source/revision, enforces sync RBAC and auto-sync restrictions, and uses the same retrying helper. Unlike Sync, this reviewed implementation contains no sync-window check; its request also has no `infos` field. See [Rollback:2156](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-server-application-application.go.txt:2156) and [request definitions:126](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-server-application-application.proto.txt:126). No reviewed evidence supports invented precondition fields or `If-Match`.

V5 fixes submission reporting, not this provider boundary: [client:272](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:272).

**Feasible options**

| Option                                  | Benefit                                                         | Limitation                                                                                                                           |
| --------------------------------------- | --------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Fresh recheck + native Sync/Rollback    | Lowest deployment burden; native authority retained             | Leaves O3 TOCTOU unresolved; native rollback lacks the required correlation marker                                                   |
| Direct Kubernetes CRD conditional write | Atomic Application conditions                                   | Argo token authority, sync RBAC and windows are not enforced by that write; unacceptable without a separately established equivalent |
| Conditional Argo action capability      | Retains native authority and closes Application acceptance race | Requires maintaining and testing a provider extension                                                                                |

The strongest counterargument against the recommendation is operational: a provider extension complicates upgrades and standard-provider support. That burden is real. Nevertheless, neither simpler option currently proves both O3 consistency and native policy preservation. Keep stock-provider limitations explicit rather than labeling extension evidence as unmodified-provider acceptance.

**Recommended bounded implementation plan**

1. **Provider capability — medium effort, required boundary repair.**  
   In the test-provider fork, change `server/application/application.go`, `server/application/application.proto`, `util/argo/argo.go`, their tests, and regenerated API/gateway/Swagger outputs. The exact generator output manifest remains to be inspected.

   Introduce distinct, authenticated routes:

   - `GET /api/v1/applications/{name}/reviewed-actions/capabilities`
   - `POST /api/v1/applications/{name}/reviewed-actions/sync`
   - `POST /api/v1/applications/{name}/reviewed-actions/rollback`

   Advertise a versioned protocol, UID/resourceVersion enforcement, native policy preservation, and correlated historical rollback. Require nonempty expected UID/version, resolved Git revision, and exactly one action marker; rollback additionally carries the reviewed history ID.

   Distinct routes are important: an older server cannot silently ignore new fields and execute an unconditional native action. Missing/malformed capability fails closed before reservation; disappearance of the route never triggers fallback.

2. **Server transaction — medium effort, highest safety impact.**  
   Reuse native authorization, project lookup, Sync window checks, revision resolution, restrictions, operation construction and event logging. Build the operation from the same freshly read Application whose UID/version matches the review. Persist operation and cleared operationState in **one context-bound update using that resourceVersion**. Return conflict without reload/retry on a Kubernetes conflict.

   Preserve native Rollback’s policy explicitly; adding its missing window check would be a separately declared policy change. Preserve source/destination enforcement in the normal provider execution path and verify it live; the reviewed project helper alone checks Application namespace eligibility, not all source/destination permissions: [GetAppProject:726](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v5-evidence/resume-argo-v5-official-util-argo-argo.go.txt:726).

3. **Ops adapter — medium effort, bounded business-domain change.**  
   Change `argocd_client.go`, `argocd_preview.go`, `argocd_action.go`, and corresponding tests. Add capability binding and selected history ID to private review data. Preserve public signatures and existing result recovery; old reserved tickets remain recoverable, while obsolete undispatched reviews conflict and require a new preview.

   Populate expected sync source: native Sync supplies it, whereas today’s plan sets it only for rollback. Retain the current same-source historical rollback restriction until a historical-source preview capability exists: [preview:147](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:147).

   If terminal validation is expanded to source/prune, extend private action metadata and `argocd_reconciler.go`; current reconciliation checks UID, revision, marker and terminal timestamps, but not source/prune: [reconciler:160](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_reconciler.go:160).

**Proof obligations and recovery**

Retain reservation-before-dispatch, one dispatch attempt, exact-ticket recovery before expiry checks, and unfinished Unknown after lost reply, cancellation or outcome-write failure: [action:58](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:58), [reservation:82](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:82), [outcome recording:185](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:185).

Required fresh evidence must cover:

- Drift before submission and between provider read/write, UID replacement, and spec A→B→A: conflict, zero accepted operation.
- Native policy denials, token revocation, source/destination restrictions and supported historical rollback.
- Real controller terminal plus independently observed Service restoration; preserve the existing strict live assertions: [live test:121](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_final_live_test.go:121).
- Lost reply and cold restart: recover the original Unknown and exact terminal without redispatch.
- Cancellation before dispatch versus during submission: cancellation is not native operation termination. Do not call name-only `TerminateOperation` to resolve uncertainty.
- Overwritten/missing operationState: retain Unknown. Eventual recovery after that evidence disappears is **not proven** and would require a durable provider receipt capability if demanded.

Base current-session/IAM wiring remains a separate shared-auth dependency. The inspected handler does not propagate SessionID; the business port intentionally fails closed. Use the Base-owned reader contract through formal consumption, not a local system-domain implementation: [requestActor:21](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:21), [authorization port:38](D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_authorization.go:38), [Base reader:47](D:/workspace/go/pantheon-platform/pantheon-base/backend/pkg/contracts/businessauth/businessauth.go:47).

**Implementation-readiness verdict:** ready for this bounded provider/adapter repair; blocked for current-path acceptance and full P0 approval. Five official source hashes/blob SHAs verified, and 23 current Go files match the recorded v5 verification. No fresh runtime tests or deployment occurred.

No files, providers, locks or ledgers were changed. The requested `resume-argo-v6-boundary-handoff.md` was not created because the explicit read-only constraint forbids writes; this report supplies its review content without claiming an on-disk artifact.
