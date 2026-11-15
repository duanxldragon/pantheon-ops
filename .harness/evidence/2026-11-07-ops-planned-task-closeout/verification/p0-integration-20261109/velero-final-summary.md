# Velero final scoped repair

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. L2, Ops Business K8s. Exact source scope: `backend/modules/business/k8s/backup/**`, `cluster/cluster_service.go`, new `cluster/credential_revocation_test.go`. Existing changes from other owners and prior failures were preserved. No shared/frontend/SQL/lock/manifest/ledger/commit/push/install changes.

## Resolved findings

- HIGH delayed second Backup GET: fresh cluster/reference/source/attempt/target checks; cluster -> reference -> source -> attempt -> sorted binding row locks held through one 30s-bounded CREATE; revocations yield CREATE0.
- MEDIUM contaminated denials: fresh MySQL database per case, valid source/reference and actual native/dynamic cached clients; exact department/source error asserted. Foreign source uses two existing authorized valid clusters.
- ARCH dual representation: explicit reference is authoritative; revoked/missing/empty/foreign reference fails closed despite inline material. Only reference ID zero uses inline; both native and dynamic credential queries use caller context.
- ARCH restore targets: actual effective mapped/unmapped namespaces require NamespaceBinding matching cluster business scope/environment and RequireWriteWithContext backup:restore. Empty/wildcard namespace sets denied; includeClusterResources=false; RestorePVs=true rejected. No fabricated cluster-admin actor.
- ARCH uncertain submission: Backup/Restore lost response becomes Unknown; explicit provider refusal remains Failed. New/Unknown/InProgress are observed under stable name plus record-id annotation; true accepted-response-timeout and new-reconciler correction tested with CREATE1, never resend.
- Additional safety: live source expiration rechecked after cluster lock wait. Namespace-only false PV intent persisted explicitly despite legacy GORM default:true, using model field and existing DDL.

## Fresh evidence

Required MySQL, CGO_ENABLED=1, `D:/msys64/mingw64/bin`, scoped `go test -mod=readonly -race -count=1 -timeout=180s -v ./modules/business/k8s/backup ./modules/business/k8s/cluster` passed, no SKIP. Scoped vet/build, gopls check and git diff check passed. Final driver race build/vet passed. All final receipts have identical stable source hashes; closeout rechecked current files against those hashes. See `velero-final-green-closure.json`, `velero-final-diagnostics-closure.json`, `velero-final-driver-closure.json`, and `velero-final-closeout.json`. Passed test/subtest lines: 78.

Original red, PV intent red, wrong-column green failure and observer-test failure are retained, with causes recorded in closeout. There is no independent final approval in these artifacts.

## Residual findings / root handoff

- Base live actor/session/current-action authorization is not integrated into Velero workers. Copied DataScope is a request snapshot; current role/user/session/BizScope revocation is not proven. Composition root/public Base port required; no business -> system dependency was added.
- Process death after committing New but before dispatch leaves an observable pending record. Reconciler does GET only and never replays CREATE. No durable replay/job engine was introduced. Historical Failed records and legacy unannotated identities are not backfilled or automatically reclassified.
- No established cluster-admin restore authorization exists here. Cluster resources/PV restore is refused; only explicitly bound namespace restore is supported in this repair. Active BizScope re-resolution requires the live authority port; binding model has no active flag.
- No remote Velero/S3/SA/HTTP/audit/cleanup phase executed. Root-owned artifact-fetch/install prerequisites remain external to this repair. Prepared live driver is a direct-service acceptance tool and cannot establish Base session/action/audit/HTTP gates.
- Independent review of final hashes, full-repository CGO=1 MinGW race, business overlay/rebuild and all D1/P0 release gates remain root-owned and open. Unknown state consumer/runtime acceptance also remains open.

## Exact Base dependencies

`backup_handler.go:CreateRestore` currently receives only `common.GetDataScope(c)` plus actor ID serialized as `createdBy`; its service/record contract carries no verified originating SessionID. `TokenAuthMiddleware` sets trusted `sessionId` at `backend/internal/middleware/token_middleware.go`, while `checkTokenBlacklist` is private middleware. Available public helpers are `pkg/authtoken.ValidateSession(ctx, rdb, accessToken)`, `SessionData{UserID, SessionID, RoleKeys}`, `BlacklistUserKey(userID)` and `BlacklistSessionKey(sessionID)`. ValidateSession only loads a token payload; it is not a complete current-session/action authority. Workers must not persist raw tokens or trust a body/header session ID.

`pkg/contracts.DataScopedGroup` / `ProtectedGroupWithRedis` expose HTTP TokenAuth -> Casbin -> DataScope middleware, not a background authorization API. Casbin/current DataScope resolution stays in internal middleware (`authorizeRoleKeys`, `applyRoleDataScopePolicy`, role-policy caches). The adjacent Argo `ActionAuthorizer.AuthorizeArgoAction` is an Argo-specific business port, not a generic Base authority to borrow; it also declares composition-root wiring pending. No public combined Base port for current user/role/dept/verified-session/action authorization was available in inspected contracts. Active business scope can be requested through public business capability `BizScopeReader.GetActive`, but BackupService has no such live authority wiring.

Smallest concrete handoff: Base/composition root must supply a context-bound live authority carrying verified actor ID and originating SessionID, resolve current actor/roles/DataScope and active BizScope, deny revoked/disabled/expired sessions and unavailable authority, and authorize the actual cluster, source and effective target namespace action at admission and the final single-CREATE boundary. Token/session storage, generic permission/audit plumbing and composition-root wiring are outside this exact ownership. Then run actual revoked-session/current-policy/negative-target HTTP and audit cases on the final source hashes. No tenant or system-domain behavior is overridden here.

## Prepared root-run driver

`.local/ops-acceptance/velero-final-live-driver.go` / `.exe` are built from the final source. Input is stdin JSON with DSN, approved Kubeconfig/DeniedKubeconfig and exact SourceNamespace/TargetNamespace/Content. Secrets/errors are not emitted. Root provisions real Velero/S3 and the finite SAs first. The driver registers the target NamespaceBinding with `backup:restore`, checks unbound remap and two valid cluster source mismatch, retains negative cases, and uses phase `velero-final-20261109` and distinct success Backup/Restore names. It reuses the original content ConfigMap name `velero-resume-content` and inherited failure-fixture names intentionally; root must provide those exact owned fixtures and cleanup inventory. No driver remote phase was run here. Driver/source/binary hashes are recorded separately.

Gate Outcomes: MySQL caught actual trust-boundary and state defects; schema mapping and completion-observer failures were retained and corrected. Independent review and D1/P0 remain open.
