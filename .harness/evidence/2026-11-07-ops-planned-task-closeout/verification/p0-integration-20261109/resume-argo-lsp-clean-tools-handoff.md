# Generator handoff: LSP-clean deployment v3 / reviewed live v4

L2 parent: `2026-11-07-ops-planned-task-closeout`; packet: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Scope: business-domain local acceptance tooling only. Eight new tools; original files/seals/requests/reviews/failed receipts preserved. Reused stdlib and the unchanged deployment helper; no platform/system/product/provider/frontend/config/dependency/release change.

Deployment v3 adds an explicit required-helper capture guard, actual transport/socket guards, truthful annotated BytesIO test stream, required spec/loader guards, and bool method annotation. Its receipt chain retains final3 and v2 and adds v3. Captured seal parsing/helper consumption, exact quoting, continuous checked binary fd, deadlines, bounded output and Unknown/no automatic retry are unchanged.

Live v4 preserves the exact v3 contract/function bytes and transforms the same original v2 generator input. Required source/spec/loader guards were added. Actual Pyright identified the colliding cleanup variable as `obj` (the Git-object callable); only cleanup metadata becomes `cleanup_obj`. `bare=None` initialization has explicit non-None guards at object writing and filesystem cleanup. All original cleanup conditions remain. Protocol/native get/sync policy, six outer/four reviewed fields, UID/RV/revision/marker/history ID 0, HTTP-200-only lost-reply injection, same-ticket/four-submit and Service restoration/product Go assertions remain unchanged. Optional `/fixture/phase` mismatch remains as recorded ARCH WATCH.

## Fresh verification and commands

Run from repository root:

- `python -B .local/ops-acceptance/argo-adoption-v3-verify.py --label local1`: PASS; 23 tests (original 20 + three capture/transport regressions), seven unchanged helper tests, Python 3.9 compile, whitespace check, two reread mutants each detected by the expected assertion failure. Exact subprocess commands/log hashes: `../../../argo-adoption-v3-20261110/argo-adoption-v3-local1-verification.json` (repository path `.harness/evidence/argo-adoption-v3-20261110/argo-adoption-v3-local1-verification.json`).
- `python -B .local/ops-acceptance/argo-reviewed-live-v4-prepare.py` and `python -B .local/ops-acceptance/argo-reviewed-live-v4.py --phase stage`: actual local generation PASS; new request only. Initial generation and its one-error LSP receipt retained under `resume-argo-lsp-clean-tools-local1-generation/` and original LSP filename.
- `python -B .local/ops-acceptance/argo-reviewed-live-v4-test.py`: nine tests PASS: original five guard groups plus source-None refusal, actual prepare byte reproduction, exact remote byte delta/reproduction, unchanged embedded contract. Log/result: `resume-argo-lsp-clean-tools-live-v4-tests.log/.json`.
- `python -B .local/ops-acceptance/python-lsp-check.py argo-v3-live-v4 argo-adoption-v3-root-run.py argo-adoption-v3-deploy-test.py argo-adoption-v3-verify.py argo-reviewed-live-v4-contract.py argo-reviewed-live-v4-prepare.py argo-reviewed-live-v4-test.py argo-reviewed-live-v4.py argo-reviewed-remote-staged-v4.py`: Pyright 1.1.414 true LSP, eight stable files, **0 errors / 0 warnings**; basic/openFilesOnly/actual LOCAL extraPaths unchanged.
- `python -B -`: eight-file Python 3.9 AST/in-memory compile/whitespace, actual v3 `prepare(sealSHA)` with 206041319-byte checked fd, phase/product pins and original preservation PASS; details `resume-argo-lsp-clean-tools-final-verification.json`. No private transport import.

## Identity / review pins

| New tool                          | SHA256                                                             |
| --------------------------------- | ------------------------------------------------------------------ |
| argo-adoption-v3-root-run.py      | `69477ec5d41ad5492eee66d876344cd96fff3106cc0e2825e15f4a60926a7f45` |
| argo-adoption-v3-deploy-test.py   | `b3946c1ca96deeb840407ae7fae1c01865e218f1339d42dd137e91f4327565f7` |
| argo-adoption-v3-verify.py        | `857dd597f81b81b6d6de17927240ff6c6bff5e0dd63fa53a3360f03256ae6352` |
| argo-reviewed-live-v4-contract.py | `5adb78dededc09860f5e0ece51c693e781019b1903f08a49e23d3b8e31717cbb` |
| argo-reviewed-live-v4-prepare.py  | `19ebb6b5c8e416e502afa8228d68ce99358ce20be9d9af0a6683c35e7d0bb70c` |
| argo-reviewed-live-v4-test.py     | `99dbed5421e621453f0208230463e39f5960590f944e58b44d6d5cf8a05a03fe` |
| argo-reviewed-live-v4.py          | `b650c47d5d2c7c3122a8c414c56cdc9a8adbad1f19517cd09827c8243192472b` |
| argo-reviewed-remote-staged-v4.py | `e476e7ff6a024de5234a369b1b91e02d857cad131ec4b036d586b524680c0a6b` |

- LSP `python-lsp-argo-v3-live-v4-20261110T041425075704.json`: `87ef3b21636f1b006239a67c5f3949fff55e17bb4efbd41eea989c0b02429f59`.
- New 34-entry seal (original 31 + new deployment three): `6f3174aa85fd8a7dd6369e3aef289363a36c327655f497e2d65148a5853868a3`. Candidate externally supplied `--seal` pin; not self-approval. Old v2 34 entries/entire evidence preserved separately and linked: old seal `fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03`.
- New phase `argo-reviewed-phase-request-v4.json`: `e7934f5656cd8aa537af487e898cd91431240aac955c76d3b7cfd9c0efc85b5d`.
- Original seal remains `08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec`; final3 `051e9fd95eead702b16da2e6092fc3f4bafc8d4129ae070089741aac11444d97`; Linux binary `cd68356983230edfa0c013235eccba4381a5fbe00d7076f47b7d4b7b8578138b`, size 206041319.
- Exact delta `resume-argo-lsp-clean-tools-source-delta.diff`: `cd7cec76900901725d48fe713bb4304c92cb50f4e840b3cc3f2658653ea8fd57`. Final verification: `c7a88f0ce6b9d4f48ce05dee30ab254de9e9c922c53646f2e2d827d61102dd9e`.

All 1566 original artifact baseline paths unchanged; only the active generator event stream appended externally. Original 15-error LSP receipt and root LSP helper unchanged. Stop condition: scoped tools locally verified, sealed and reviewable. Root independently reviews CODE/ARCH and owns actual deployment/readiness/live acceptance. No remote/credential operations, runtime success, full Go race rerun or P0 approval claimed. Gate Outcomes: LSP caught real nullability/callable collision; reread mutants caught checked-buffer regressions.
