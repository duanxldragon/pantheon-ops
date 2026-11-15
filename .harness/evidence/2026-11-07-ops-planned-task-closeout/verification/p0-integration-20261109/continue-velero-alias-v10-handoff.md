# Velero alias v10: native typed CRD inventory successor

Parent task: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`.

Scope is the existing bounded alias recovery helper lane. No platform/system/business product source, dependencies, model/config settings, permissions, administrator startup, waivers or production policy changes are involved. Root owns remote diagnostic → repair → validate → cleanup and independent final CODE+ARCH reviews.

## Concrete cause and repair

The consumed v9 diagnostic failed at the unchanged4MiB native API output gate. Root's read-only status observation proved the full CRD inventory was6,540,046bytes, with exit0 and no stderr; RBAC list outputs were within budget. Root then actually executed an unfiltered native `jsonpath-as-json` control. Its12,765byte output preserved all53CRDs and13ownedCRDs; names/UIDs/groups/alllabels/presence/resourceVersion and ordered UID coverage compared exactly with the full list kept only in diagnostic memory.

v10 adds `crd_inventory()` using that exact reviewed native template and the unchanged command wrapper:4MiB stdout, empty stderr, request timeout20s and process timeout30s. It replaces only the bulk CRD list's UID-set check with the typed projected inventory gate. Full individual CRD object reads still enforce pinned UID, full original labels/API version/annotations/spec and Established/non-Terminating conditions. All22individual object checks, native controller/RS/template/Pod chain, image/content/archive/alias/probe, RBAC and failure/inspection gates remain in force.

Typed projection rejects duplicate JSON map keys, malformed JSON/UTF8, nonfinite constants, wrong/absent required field types or singleton arity, duplicate CRD names/UIDs, invalid canonical identities, incomplete/reordered UID coverage and unknown trailing fields. Optional labels/resourceVersion preserve absence versus present empty values. All rows are projected without selectors; any Velero group, expected name or matching owner-label sibling must match exactly the original owned name→UID set and group/owner.

Native template:

```text
{.kind}{.metadata.resourceVersion}{.items[*].metadata.uid}{range .items[*]}{.metadata.name}{.metadata.uid}{.spec.group}{.metadata.labels}{end}
```

## Exact reviewed bytes

Request: `continue-velero-alias-v10-request.json`; schema11; SHA256`935242362fc459942bcef49e955d6216eee95438a25121cf1eda443e390fcace`; operation`20261111t053326667811`.

| Source                                                        | SHA256                                                             |
| ------------------------------------------------------------- | ------------------------------------------------------------------ |
| `.local/ops-acceptance/continue-velero-alias-v10-contract.py` | `56b4932709872f6ef9f34cdf1a88a5c0b952ddd9d2c04c1e15c08adb4efa1a33` |
| `.local/ops-acceptance/continue-velero-alias-v10-remote.py`   | `0d95ec93fa7c292cb0a653d6e35dadb35196c90ed23d117a8f85b77d32450c59` |
| `.local/ops-acceptance/continue-velero-alias-v10-root.py`     | `8283a467f24ef6bf81f44374a8a33342ab6cafb1afe5d627602c6b8c1ca729a4` |
| `.local/ops-acceptance/continue-velero-alias-v10-generate.py` | `4c9728711768d0111043c63160820d0f1993ce8e92842bb97594db92f45a36fe` |
| `.local/ops-acceptance/continue-velero-alias-v10-test.py`     | `dcb4041a11bcab07ff55800f65ed08325fc477db3a15184b320dee9438a5a869` |

Current remote baseline is exactly43files: immutable41pre-v9files plus only the two fully recorded v9 helper uploads. No failed v9 phase is replayed. Request protects443assets, including every original/v9 seal, consumed failed request/diagnostic/inspection and independent reviews, both actual read-only controls and their helpers/envelopes, official printer sources and retained failed local verification receipts. Canonical22object inventory is never rewritten.

Actual projection receipt:
`.local/ops-acceptance/post-config-velero-v10/projection-observed-20261111T052012737733.json`
SHA256`51c357a0024718f09e95411b4da2de9fde768ad145c053a68851894fbad10714`.

Prepared projection envelope SHA256`ef0ec9afc9f32b585c5714dc077b5d2feedb704c403ad61a31661b98ad600420`.

Official Kubernetesv1.33.0 printer evidence is pinned in that envelope. Typed JSON array emission, including empty arrays, is implemented by `client-go/util/jsonpath` JSON output and `cli-runtime/pkg/printers/jsonpath.go`. The control's8MiB full-response cap applied solely to its diagnostic memory comparison; no production budget is enlarged.

## Fresh verification

- Full inherited suite:373executions/320unique regressions,16new v10tests,52existing forgery denials; zero failures/errors/skips.
- Focused current projection/preparation/readback:17tests pass.
- Actual Pyright1.1.414 on all five exact sources:0errors/0warnings, stable hashes.
- Actual sealed v9 code rejects a schema-bearing bulk response over4MiB; current production projection passes the unchanged gate for the same identities. Root's actual complete53row comparison validates the real native rendering.
- AST evidence proves every existing remote function except the single inventory portion of `objects()` is identical after version-name normalization; complete22object validation prefix and remaining RBAC validation suffix are identical.
- All443protected hashes and exact43file baseline pass readback.

Initial full-run failures were confined to the new test adapter still targeting the inherited v9 fixture prefix. The new v10 test adapter now targets v10; the original helper bytes and assertions were preserved. The failed receipt/log remain in `.local`; the fresh complete suite passes. Postseal full/readback must also pass before Root execution.

## Required independent review

CODE: verify actual native projection syntax/types/absence/duplicate/coverage refusal; full individual CRD gates; no raw schema/annotation/status/metadata/stderr leakage into public telemetry; unchanged budgets; v9 failure/inspection adoption, exact43files and every original seal. Re-run focused/full and actual LSP as needed using only local effects; never execute old/consumed or remote phases.

ARCH: verify this is a representation-size fix in the approved existing alias recovery lane, with complete unfiltered ownership visibility and full owned-object semantics preserved. No shared/product policy, trust boundary, provider/S3 mutation scope, alias/probe/fixture recovery, network policy or approval gate expands. Confirm exact current request/five source pins and independent review gating before connection.

Fresh reports must use exactly one JSON fence, bind`935242362fc459942bcef49e955d6216eee95438a25121cf1eda443e390fcace` and the complete five-source map, and record the reviewer's actual decision. Existing dispatcher requires independent CODE=APPROVE and ARCH=APPROVE with architecturalStatus=CLEAR. No predecessor approval is inherited. Required format: `continue-velero-alias-v10-report-format.json`.

## Root execution after both fresh approvals

```powershell
& 'D:/Python/Python312/python.exe' -B '.local/ops-acceptance/continue-velero-alias-v10-root.py' check-inputs --request-sha256 935242362fc459942bcef49e955d6216eee95438a25121cf1eda443e390fcace
& 'D:/Python/Python312/python.exe' -B '.local/ops-acceptance/continue-velero-alias-v10-root.py' diagnostic --request-sha256 935242362fc459942bcef49e955d6216eee95438a25121cf1eda443e390fcace
```

Root then passes each exact successful predecessor receipt filename/SHA into repair, validate and cleanup. A failure remains inspection-only and cannot become a success predecessor or authorize replay. No remote phases have been executed by this successor author. Real acceptance and P0 approval remain pending.
