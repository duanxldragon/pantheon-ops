# O4/D3 declared-metadata count — Generator handoff

Generator implementation and focused verification are complete. **Independent CODE/ARCH review and actual stable-source product assembly remain root gates.** No full real assembly, adoption, release, push, baseline replacement or CODE/ARCH approval is claimed.

L2 platform-tooling boundary: Base candidate producer → Ops business consumer; logical packaging change, no physical backend/frontend split. Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Entry evidence: [prior handoff](../../resume-o4-real-packaging-fix-handoff.md), its named implementation receipt and count diagnostic. Required inheritance/Base/method entry documents and `repo-verify` were read. No CodeGraph MCP was callable. No delegation or model override occurred.

## Exact delta

`scripts/business-overlay/foundation-candidate.mjs` adds frozen `METADATA_LIMITS.files = 100000`; only `captureInputs` switches its unique-file count check and error to `Declared metadata input count exceeds limit`. The shared schema1 `LIMITS` remains identical to live Base: **50,000 candidate files, 32 MiB regular source, 512 MiB archive**. Streaming digest/stamp logic, the retained-archive shape exception and all curated PUBLIC leaves/config/tamper refusals are byte-preserved.

Two tests add real boundary coverage. Metadata traversal/hash/stamp checks process 100,000 synthetic names projected onto a real zero-byte file, preserve every unique entry despite overlapping declarations, and refuse 100,001. Both live validators accept a 50,000-file inventory/archive and refuse 50,001; the overflow tar has a valid external digest. All original 74 test bodies remain unchanged. No production file beyond the owned consumer/test was modified.

[Exact patch](metadata-bound-1794282381452/metadata-bound-delta.patch), [contract/whitespace checks](metadata-bound-1794282381452/delta-contract-checks.json), [captured original bytes](metadata-bound-1794282381452/before.json), [complete before/after hash ledger](metadata-bound-1794282381452/preservation-after.json).

| File             | Before SHA256                                                      | After SHA256                                                       |
| ---------------- | ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| Ops consumer     | `e8e3cb25c3c052ab1e8111b60600f803ee4bd5e7b06090c89b391908364a1fd1` | `894eb9401d05a6d272eaf9cd5582c88e5ff3b9a2aa5430d5a309d138827b06a9` |
| Ops focused test | `9fc65b83d994eb4fe0229a6a7d9428539c36c7f3fbce936775e270802f8d0acf` | `1b5f7b820b587e933b271434e47f077620e976845de82480a866580d5769d56c` |
| Official lock    | `95de1a87270e15c0d4c672418a5df906818a88652f8a3e6c480f1484ee9fa70c` | identical                                                          |

The ledger captures all six originally named sources, `rebuild-from-base.mjs`, manifest, lock and both original Base configs. Only the two owned files changed; 21 protected prior artifacts retain their hashes. Live Base's three files exactly match the prior proposed hashes before/after. Cache/credentials/DB and other agents' edits were not altered or deleted.

## Fresh verification

Logs and exact executable/argument/cwd/exit/pass-count receipts are retained in `metadata-bound-1794282381452/`.

| Command                                                                               | Result / log                                   |
| ------------------------------------------------------------------------------------- | ---------------------------------------------- |
| `node --test ../pantheon-base/scripts/foundation-release/candidate-artifact.test.mjs` | 38/38; `live-base-tests.log`                   |
| `node --test scripts/business-overlay/foundation-candidate.test.mjs`                  | 76/76; `ops-candidate-tests.log`               |
| `node --test tests/scripts/business-overlay/*.test.mjs`                               | 62/62; `overlay-tests-isolated-cache.log`      |
| `node scripts/business-overlay/check-business-overlay.mjs`                            | exit 0, 56 declared paths; `overlay-check.log` |
| `node --check <six consumer/test/local helper files>`                                 | six pass; `diagnostics.json`                   |

All GREEN suites have zero skipped/cancelled/todo. TypeScript 6.0.3 LanguageService reports zero diagnostics with `checkJs=false`; full static type safety is not claimed. Captured-baseline whitespace checks pass. RED uses the final tests and exact pre-change consumer loaded only in memory: metadata fails at old 50,000 bound, archive test passes (1/2; `metadata-boundary-red.log`). An initial overlay run (61/62) hit denied writes to `D:\cache\go\build`; rerun used a fresh `.local/ops-acceptance/metadata-bound-1794282381452-go-cache`, preserving the original cache/failure. An early recording-mock attempt was interrupted; final fixture avoids that overhead. No backend/frontend build, full Go race or runtime acceptance was run in this lane.

[Current observational count](metadata-bound-1794282381452/declared-input-count-current.json): **55,812** unique declared files; `.foundation` **48,886**, `.harness` **6,341**, no omitted inputs/unsupported types, manifest unchanged during enumeration. Later evidence increases counts. This is enumeration, not a source freeze or actual capture/assembly.

## Root's next one-shot invocation — not executed

After independent review and O2 source quiescence, copy [request template](metadata-bound-1794282381452/root-review-request.template.json) to the fresh request path below and fill actual reviewed `reviewIds` (currently empty). It retains all 1,133 prior selections plus eight PUBLIC leaves, including both original configs. Refresh provenance/selections if root's approved source scope changed.

```powershell
node .local/ops-acceptance/resume-o4-stable-source-producer-consumer.mjs .local/ops-acceptance/o4-metadata-bound-1794282381452-approved-request.json .local/ops-acceptance/o4-metadata-bound-1794282381452-stable-real
```

The syntax-checked helper imports **applied live Base**, creates a new sealed output, and passes the original in-memory producer pin directly to the actual Ops consumer. Receipts stay outside declared inputs; existing output/stage identities refuse reuse. Preserve any refusal/seal/receipt. Root must inspect actual assembly/drift/runtime evidence before O4/P0 closure.
