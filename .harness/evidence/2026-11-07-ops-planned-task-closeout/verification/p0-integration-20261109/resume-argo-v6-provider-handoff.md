# Argo reviewed-actions v1 provider handoff

**Bounded provider source and local evidence are ready for independent review.**
No provider binary/image or deployment is claimed. Ops adoption, authenticated
product evidence and full O3/D1/P0 acceptance remain open and root-owned.

## Scope and contracts

- Tier L2, business/cicd provider assets physically isolated from Ops/Base.
- Parent O3: `.harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O3-change-preview-and-risk.md` (same reviewed intent/version; drift409/TOCTOU).
- Parent D1: `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/packets/D1-external-integration-acceptance.md` (real sync/rollback plus authority/audit/idempotency evidence).
- Architecture: sibling `resume-argo-v6-boundary-handoff.md`.
- Official upstream v3.2.0 HEAD **66b2f302d91a42cc151808da0eec0846bbe1062c**, verified before edits; initial fork status clean.
- Fork: `.local/ops-acceptance/argo-reviewed-provider-v3.2.0`.
- Durable assets: `backend/modules/business/cicd/argocd/provider/README.md` and `reviewed-actions-v1.patch`.
- Existing Ops Go edits, Base, locks, credentials, common network/TLS and original failed v4/v5 evidence were not changed by this work. No remote action, commit, push or release occurred.

## Implemented provider contract

The independent `/reviewed-actions/capabilities`, `/reviewed-actions/sync` and
`/reviewed-actions/rollback` routes are native protobuf ApplicationService RPCs
and generated gateway handlers. Official stock source has none of these routes.
Unknown additive fields on a native stock Sync/Rollback route are not capability.

Protocol **pantheon-reviewed-actions/v1** advertises `single-git-sha1`,
`pantheon-action-id`, conditionalUpdate/nativePolicy/correlatedRollback=true.
Capabilities reuse ApplicationSyncWindowsQuery (name/appNamespace/project);
sync and rollback reuse their native request structs with strictly validated
additive `reviewed` fields: expectedUID, expectedResourceVersion,
resolvedRevision, actionMarker. Rollback's required native `id` is the reviewed
provider history ID, including valid ID0. See durable README for exact JSON and
bounds. Missing/incompatible capability or missing route must fail closed.

Argo's existing global authentication interceptors, claims, namespace/project
scope and RBAC remain the authority. All reviewed methods additionally require
the native session principal; anonymous default-role or disable-auth access
cannot advertise capability or submit an action. Unit claims fixtures are not
proof of real token authentication/revocation.

The same native Sync/Rollback implementations build both legacy and reviewed
operations. The reviewed operation is built from the SAME authorized Application
whose UID/resourceVersion matched. A single request-context-bound Application
Update uses that exact UID/version to persist operation and clear operationState.
No name-only reload or conflict retry follows. Pending/deleting/invalid and
revision/history mismatch conditions fail before mutation. Cached resource-tree
health projection is not persisted by reviewed submission.

Sync reuses native windows, sync RBAC, revision resolution, automatic-sync
restrictions, source/prune/options/retry/strategy/initiator and event logging.
V1 excludes multi-source, Helm chart-repository versions, local manifests and
caller infos. The reviewed operation has exactly one bounded correlation info.
Rollback uses the selected native immutable historical source/revision with
prune/options/apply/initiator, native sync RBAC and auto-sync restrictions.
**Native Rollback has no Sync-window check; the reviewed path preserves that
policy and does not invent one.** Controller-time source/destination/signature
policy still requires real provider evidence.

Version binding covers operation acceptance, not a freeze of every managed
resource through terminal. Native controller/retry semantics remain intact.
The provider returns the actual Update response, preserving the operation marker
when a controller subsequently consumes the operation quickly. No terminal
state is fabricated. Cancellation or lost reply can leave acceptance uncertain;
Ops must retain Unknown, recover the same ticket/UID/revision/marker, and never
redispatch or terminate a name-only operation to resolve uncertainty. There is
no durable provider receipt after later evidence overwrites.

## Fresh verification

- `go test -race -mod=readonly ./server/application ./pkg/apiclient/application -run ^(TestReviewed|TestSyncAndTerminate$|TestSyncHelm$|TestSyncGit$|TestSync_WithRefresh$|TestRollbackApp) -count=1 -json` → exit **0**, 38.906 s; source hashes stable.
- `go vet -mod=readonly ./server/application ./pkg/apiclient/application` → exit **0**, 3.219 s; source hashes stable.
- `go build -mod=readonly ./server/application ./pkg/apiclient/application` → exit **0**, 3.172 s; source hashes stable.

Toolchain: **go version go1.26.5 windows/amd64**; CGO_ENABLED=1, compiler PATH begins
`D:/msys64/mingw64/bin`. Both selected packages passed. **49 pass events
include 5 grouping parents: 44 non-group test cases, zero skips,
zero final failures.** This is a focused provider suite, not the full Argo suite
or a remote product test.

Coverage includes real native-method comparison for source/prune/options/retry
and historical source restoration; RBAC get/sync/rollback denial; active Sync
deny window; native Rollback without a window check; auto-sync rollback denial;
required fields/marker/history/revision mismatch; UID replacement and unchanged
spec with advanced ABA version; consumed-but-running/deleting/pending states;
single update conflict without reload; already-canceled and cancellation during
Update; lost reply without retry; fast controller consumption with original
correlated write response; protocol binary round trip and token metadata
transport; generated distinct gateway field binding and HTTP409; modeled
unimplemented peer with no native fallback; six unchanged legacy Sync/Rollback
regressions. New authentication tests reject absent native principals before
read/update. Real token and controller authority are separate open gates.

Exact fresh exits, source hashes, per-test events and timestamps:
`resume-argo-v6-provider-verification.json`. Raw logs remain in
`.local/ops-acceptance/resume-argo-v6-provider-evidence/final-auth-{race,vet,build}.log`.

Retained preliminary evidence is not silently relabeled: the initial unscripted
native baseline was canceled during dependency download; a preliminary combined
run exposed the test helper's deep-copy client wrapper assumption, corrected by
using the native GetUnderlyingClientSet helper. The recorded targeted-first run
had two gateway fixture failures because proto2 required name was omitted from
the body; explicit native name corrected them. An earlier race run passed but
was hash-unstable while health-projection isolation was being added, so it is
superseded by the final hash-stable auth runs. No initial failure is counted as
a final pass, and original v4/v5 product failures remain preserved.

## Generation, exact patch and unchanged paths

Official upstream pins: protoc29.3, gogofast1.3.2 and grpc-gateway/swagger1.16.0.
Actual generator commands, provenance/build metadata and hashes are in
`resume-argo-v6-provider-generation.json` and
`resume-argo-v6-provider-tool-provenance.json`.

Both baseline protobuf and gateway outputs reproduce official HEAD **byte for
byte**, before new output generation. No generated APIs are hand-fabricated.
The generated service's reviewed routes and four definitions are merged into
the existing consolidated Swagger with upstream collector transforms. Every
existing unrelated Swagger path/definition value is identical. No broad CRD,
controller/client or unrelated service generation was run.

The official Windows protoc zip was downloaded from the fixed public release
URL and hashed locally; no independent upstream Windows digest was obtained.
The optional GitHub API digest lookup through urllib failed with WinError10013,
and upstream's checksum inventory has no Windows checksum. This limitation is
recorded without claiming official digest verification. Go plugin build metadata
records official pinned module versions.

Exact patch: **162006 bytes**, SHA256
`a21740245ef6f35a6da8337ab05e6771e8c376fb4bcc70493cc51ac2e5dfc852`. All **8** changed paths are listed below. All
**4820** unrelated tracked files match their exact
official checkout bytes with the recorded Git checkout EOL representation;
individual hashes are in `resume-argo-v6-provider-changed-paths.json`.
Mechanically removing only the reviewed wrappers/branches reconstructs the
entire original application.go byte content (canonical LF), proving no unexplained
native-method drift. go.mod/go.sum are unchanged.

Forward patch --check with whitespace=error, actual application to original
changed-file blobs in an isolated no-Git scratch tree, patched-content comparison,
and reverse --check against the fork all exited0. No index staging/lock mutation.

| Changed path                                          | Working-file SHA256                                                |
| ----------------------------------------------------- | ------------------------------------------------------------------ |
| `assets/swagger.json`                                 | `eee7ccc1d563bcc6bd025867d87b5d88e7ee7d9845cd960054872aa21958f1de` |
| `pkg/apiclient/application/application.pb.go`         | `b6ae80edad7208f02617a6d2f6fee37c95b0f277c59cffdcc2f2c5de4e129026` |
| `pkg/apiclient/application/application.pb.gw.go`      | `16ac726e69ad4e30906209d0b832542b76031a02446c476caac7dad81eb37ad6` |
| `pkg/apiclient/application/reviewed_protocol_test.go` | `99f189b521675f96436ab307cbe76f73c7f04ac4fc89a7572dc2eca25e5c3b1f` |
| `server/application/application.go`                   | `9902808d66b291b7fe38e101b9b9627b2507ca7b66e487e8e540be30c0ca30b4` |
| `server/application/application.proto`                | `97f2d78b74c263400fe02c6b65aacb5351c0763ec3b3211f701d423bcb245b66` |
| `server/application/reviewed_actions.go`              | `deaf3671254dd804432c9f73713ec3274c90a0c0f556402e71da5ceeb7ba0a31` |
| `server/application/reviewed_actions_test.go`         | `fa77ac7f3e7b123ee7c445f5f0dd664314ed0afd6d4bc49214060db36790ae22` |

## Review disposition and root-owned remaining gates

Self-review found and resolved cached-health write side effects and anonymous
default-role capability/submission. It also checked native history semantics,
one-update context/identity binding, failure response conservatism, distinct route
registration, generated output authenticity and unchanged legacy bodies.
No independent reviewer was spawned in this leaf; parent/root must independently
review the exact patch and source hashes before adoption. This is the explicit
review path, not an approval claim.

Root still owns:

1. Independent provider review and any resulting repair, with updated exact hashes.
2. Full patched-server binary/image build and test-provider deployment. The local
   successful build covers server/application and pkg/apiclient/application only.
3. Ops capability negotiation/private review historyID/client adoption after
   independent provider approval; no stock/native fallback or cluster credentials.
4. Real native-token auth/RBAC/revocation, project source/destination/signature
   behavior, native windows and conditional UID/version/ABA conflicts against the
   actual Kubernetes API and deployed provider.
5. Real controller terminal, independently observed Service restoration, native
   immutable historical rollback, lost-reply/cold-restart recovery without
   redispatch, cancellation and overwritten-evidence Unknown behavior.
6. Base-owned current-session/IAM formal consumption, product DataScope/audit,
   existing O3/D1 scope, complete P0 acceptance and release gates.

Provider-extension compatibility/update burden and stock-provider safe limitation
are durable in README. No approval handoff is needed to inspect these local
artifacts; deployment/acceptance is deliberately outside this leaf's ownership.
