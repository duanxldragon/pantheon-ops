# Argo Ops adoption — verified Generator handoff

Task: argo-adoption-20261110; parent: 2026-11-07-ops-planned-task-closeout.
Plan: plan.md. Generator self-review: review.md. Fresh evidence: final3-verification.json.
**Owned implementation and local verification are complete. Root independent review/deployment/runtime and P0 acceptance remain open. No remote operation, credential/config read, child agent, Base/shared/frontend/lock/ledger/provider-patch/fork edit, commit, push or release was performed.**

## Changes and compatibility

`argocd_reviewed_protocol.go` strictly validates the six reviewed-actions/v1 capability fields, rejects missing/null/duplicate/unknown/malformed/incompatible capability responses, and bounds the required UID/RV/revision/marker fields. `argocd_client.go` uses only distinct reviewed native Sync/Rollback POST routes with native name/appNamespace/project, expectedUID/expectedResourceVersion/resolvedRevision/actionMarker; rollback sends the exact required ID including valid zero and sends no source or infos. Its response must carry an advanced version, same UID/spec and exact source/revision/prune/marker; HTTP200 without evidence is Unknown.

`argocd_preview.go` binds protocol and selected native history ID in private encrypted review, supplies expected native Sync source, and admits only provider-supported lowercase 40-character Git SHA1. `argocd_action.go` compares/fixes private history identity across preview/execute/recovery and checks fresh authority after the final capability read. Existing scope, configuration/credential fingerprint, serialized reservation, audit/redaction, UID/revision/marker reconciliation and provider-free exact-result recovery remain intact. Obsolete undispatched reviews conflict; originally reserved obsolete records remain recoverable through the prior private format. Unknown is never resent or terminated name-only.

Public service/client method signatures and public request/response JSON shapes are unchanged; no dependency or schema/migration change. The native exported Sync/Rollback wrappers remain compatibility APIs and are not called by this business action path. Support is intentionally narrowed to negotiated single-source lowercase Git SHA1. Native immutable rollback is used for the existing supported same-source history scope; arbitrary historical source/configuration drift still fails closed because provider v1 cannot render that historical configuration for preview. UID/RV bind Application Update acceptance, not all managed objects through terminal.

## Fresh verification

`python -B .local/ops-acceptance/argo-adoption-verify.py --label final3` exited **0**, with **31/31 sealed source/helper/provider asset hashes stable**.

- `go test -mod=readonly -race -p 2 -count=1 -json -timeout 4m ./modules/business/cicd/argocd`: **PASS**, 43.797 seconds command time, **203 pass events = 181 non-group cases + 22 grouping parents**, zero failures. Two opt-in real live tests skipped: TestArgoFinalProductLive and TestArgoProductLive. Private MySQL was required and executed at the established password-free isolated harness; no DB tests were silently skipped. CGO_ENABLED=1, PATH begins D:/msys64/mingw64/bin, GOMAXPROCS=2.
- Scoped `go vet -mod=readonly`, `go build -mod=readonly`, `gopls check` all 25 Argo Go files: **PASS**, no diagnostic output.
- `gofmt -l` and scoped `git diff --check`: **PASS**, empty output.
- `python -B .local/ops-acceptance/argo-adoption-deploy-test.py`: **7/7 PASS**. All four new helpers also pass AST syntax checks without importing root transport or executing remote actions.

Focused regressions cover capability/no fallback, every required wire field, native ID0, same-revision/history-ID drift, private metadata redaction, disappearing capability/route, obsolete reviews, authority revocation during capability read, ambiguous reply/cold exact recovery and existing failure/cancellation/reconciliation behavior. Unit terminal fixtures are not real controller evidence.

Initial final1 vet/build/gopls failed on shared default Go cache ACLs while race passed. The task-owned cache resolved those failures. Final2 passed commands but was source-unstable while helper safety checks were completed; it is retained and not promoted as final verification. Final3 is the source-stable result. The task-owned build cache was removed after verification; cache-cleanup.json records the validated workspace-contained cleanup. Logs and source seals remain.

Gate Outcomes: capability gates caught unsupported/malformed peers and prevented fallback | final authority check caught revocation during capability read | source-stability gate caught concurrent helper edits | shared cache ACL false-positive isolated and resolved.

## Exact seals

Artifact: `.harness/evidence/argo-adoption-20261110/artifact-seal.json`

SHA256: **08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec**

The seal links the full 31-file source hash inventory and the exact final verification receipt. Key hashes:

| Artifact                                                           | SHA256                                                             |
| ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| `backend/modules/business/cicd/argocd/argocd_action.go`            | `cb7060c6a1b38d5d6fab7511af6d0ea509b8648bdf91f7cf9f509bf58b96b420` |
| `backend/modules/business/cicd/argocd/argocd_adoption_test.go`     | `6e0fb85064c56b41657a732580d81678e5ed0e02bfd1b1256e24c9e88780bb01` |
| `backend/modules/business/cicd/argocd/argocd_client.go`            | `5df752f84fe04df88d2b6a47d56c62c79b7e91e8010cae6db3f14203f2245dc6` |
| `backend/modules/business/cicd/argocd/argocd_preview.go`           | `1e0552c188cc3e3b23e21567cf7492d464a505b5f8651d739bd9bce1581a2aa0` |
| `backend/modules/business/cicd/argocd/argocd_reviewed_protocol.go` | `191a2200be0c5760edd3ed9bb9dd4f2b33d1c6ad58a8c4e3ad2b0c10afcc05b7` |
| `.local/ops-acceptance/argo-adoption-deploy.py`                    | `bd71ed1edb5caa8d4dadd87beb49eaad0a282ba79a4bc236056eeee7cbac514e` |
| `.local/ops-acceptance/argo-adoption-root-run.py`                  | `0106df0229fd740daaa49294eb6c10833c84d074aafc3cd6c262cad225f2ab87` |

Provider patch remains **a21740245ef6f35a6da8337ab05e6771e8c376fb4bcc70493cc51ac2e5dfc852** at upstream **66b2f302d91a42cc151808da0eec0846bbe1062c**. Durable README remains **18aeb848cecccd0b3e0a19091f9eb3ddbbbbc296223b6cd4990e35ca89d56192**. Reviewed Linux binary remains **cd68356983230edfa0c013235eccba4381a5fbe00d7076f47b7d4b7b8578138b**, **206041319 bytes**. The sealed upstream fork was not mutated; its prior independently verified 4828-source evidence is retained, not relabeled as a new full fork run.

## Root deployment and rollback

After root reviews this exact seal, run from repository root using root's existing authorized test-host transport:

```powershell
python -B .local/ops-acceptance/argo-adoption-root-run.py install --seal 08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec
```

Root-owned reversal, using the same reviewed seal:

```powershell
python -B .local/ops-acceptance/argo-adoption-root-run.py rollback --seal 08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec
```

The runner verifies local regular files/no symlink traversal, all sealed sources, the verification receipt and binary size/hash before connection, then rejects any peer other than 115.190.251.143. It streams only that bounded binary directly to the remote helper; no intermediate remote upload file is overwritten. It emits a fresh redacted root run receipt under this evidence directory and never automatically retries an uncertain execution.

The helper binds namespace UID **e6821e41-3e59-477f-af84-11ba32627360**, Deployment UID **dfb553ee-bec1-4f7e-b12f-ba96594684f0**, inspected spec SHA **5b38ce82391b455691355b3361cd5066b84d4ac6def249d0d8dbd4af5aaa82af** and inspected host-directory device/inodes. Inspected RV **1969595** is recorded; the actual current pre-RV is carried through native dry-run and one actual replace, so any intervening storage change conflicts without re-read/retry.

Dedicated owner path: `/opt/pantheon-p0-20261109/argo-reviewed-v6-cd6835698323`. Install refuses an existing path, symlink, foreign/writable ancestor, unexpected node/container/entrypoint/mount/spec or malformed upload. The hashed regular binary is made executable and Linux filesystem-immutable (actual FS_IMMUTABLE_FL; unsupported filesystems fail closed) and mounted read-only at `/opt/pantheon-reviewed/argocd-server`. The permitted spec delta is owner labels, dedicated File mount, executable argv path and pin to the inspected sole node. Stock quay.io/argoproj/argocd:v3.2.0 image and every original security/RBAC/env/volume/probe/resource setting remain, including **192Mi** memory and **500m** CPU limits. No unrelated Argo component is replaced.

Host artifacts include before-deployment.json, planned-deployment.json, actual accepted-deployment.json and install.json with exact pre/post UID/RV/spec/byte hashes. Rollback writes rollback-plan.json, actual rollback-accepted.json and rollback.json. Full original spec/metadata stays private on the owned host in mode0400 artifacts; only safe identities/hashes enter local receipts. If install loses its reply, rollback can still use the sealed staged before/plan artifacts when the exact current UID/spec/owner matches; it does not re-install to guess success. Different/unknown state stops. The immutable host binary remains after reversal for separately reviewed owned cleanup.

## Remaining root proof obligations

1. Independently review this adapter/helper seal, then run deployment and record actual inode/immutable-bit/read-only mount/binary identity, UID/RV/artifact seals and rollback restoration. Deployment Update acceptance is not server readiness or provider acceptance. Prove ABI/node compatibility and readiness/restarts/RSS under the retained 192Mi limit.
2. Prepare a new reviewed finite live relay phase. Existing `.local/ops-acceptance/argo-final-live.py` is PATCH-only in its remote allowlist and uncertainty trigger; it must not be used unchanged. Root's new phase needs exact application capability GET (project/appNamespace query), reviewed Sync/Rollback POST only, lost-reply injection after the actual reviewed POST acceptance, and the new source/driver seals. No native/PATCH/cluster fallback or name-only termination.
3. With root's private `PANTHEON_ARGO_PRODUCT_FIXTURE` environment supplied by that reviewed relay, run the opt-in suite from backend: `go test -mod=readonly -race -p 2 -count=1 -timeout 8m -run '^TestArgoFinalProductLive$' ./modules/business/cicd/argocd`. Keep fixture/auth bytes out of output. Independently inspect the native operation, real controller terminal and Service selector/restoration, immutable rollback history source and revision, permission/token/policy denials, UID/RV drift and lost reply/cold restart. Preserve old failed proof instead of rewriting it.
4. Prove real Base originating-session/IAM/DataScope/bizscope/cluster/audit composition and revocation. `requestActor` still lacks SessionID; consuming/wiring the Base-owned contract is outside this scope. Injected unit authority is not authenticated product acceptance.
5. Run root-owned full stable CGO repository race, formal business overlay/rebuild/compatibility and independent review. Keep Unknown if provider operation evidence is overwritten; v1 has no durable receipt capability. No P0 approval, production/release or omitted O3/D1 criteria are closed by this handoff.
