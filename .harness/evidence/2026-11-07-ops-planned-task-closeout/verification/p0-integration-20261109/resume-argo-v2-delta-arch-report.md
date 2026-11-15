## Summary

**ARCH `CLEAR` for the new deployment delta.** The R1 repair binds seal parsing and helper execution to the exact bytes checked. No actionable architectural blocker remains in this bounded 业务域 tooling review; separate CODE disposition and downstream acceptance remain open.

## Analysis

- **Approved bytes are consumed directly.** `prepare` captures the seal once, hashes it and parses the same buffer. It captures the helper once, verifies its digest, and passes that captured UTF-8 content into the quoted command. See [runner:126](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:126), [runner:148](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:148), [runner:196](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:196).

- **Preflight precedes transport.** File shape, ancestors, links/reparse points, fd/path identity, canonical inventory paths and pins are checked before connection. Unsafe captured inputs refuse locally. See [runner:35](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:35), [runner:101](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:101), [runner:232](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:232).

- **Binary consumption remains bound.** One fd spans preflight hash, rewind and bounded streaming. The unchanged helper checks received digest, disk identity and immutable flag before dry-run and one actual Update. See [runner:164](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:164), [helper:160](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:160), [helper:236](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:236).

- **Failure behavior is conservative.** Peer enforcement, receive deadline and output caps remain; ambiguous execution retains Unknown without retry. Rollback sends no binary and uses recorded ownership/spec checks. See [runner:189](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:189), [runner:228](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:228), [helper:250](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-deploy.py:250).

## Root Cause

The original runner separated hash verification from later pathname reads for parsing and execution. The delta removes both reread windows; it does not require an atomic snapshot of the entire filesystem. See [exact diff:250](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/argo-adoption-v2-20261110/argo-adoption-v2-final2-source-delta.diff:250).

## Validation and identity

Fresh bounded execution passed **20 new tests, 7 original tests, both reread mutants, Python 3.9 AST/in-memory compilation, and actual local prepare**. Each mutant produced one assertion failure. No private transport was imported during actual prepare.

Independently verified **34 current inventory hashes, all 31 original entries and original metadata/pins, 67 preserved files, six recorded log hashes, and exact diff reproduction**. Post-regression checks found no drift. Source identities are recorded in [seal:37](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/argo-adoption-v2-20261110/argo-adoption-v2-artifact-seal.json:37).

| Artifact                | Verified SHA256                                                    |
| ----------------------- | ------------------------------------------------------------------ |
| New seal                | `fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03` |
| Original seal           | `08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec` |
| New final2 receipt      | `e37070716df92af87a5a8a959c7a2ee951caee16ee118c73fc9b67876b2cb735` |
| Original final3 receipt | `051e9fd95eead702b16da2e6092fc3f4bafc8d4129ae070089741aac11444d97` |
| Exact diff              | `b08a95edb1fe558a7e85720d8d239f4560b30f0ddcad790ed2956f971da0308e` |

## Recommendations

1. **Low effort:** Root should use the v2 entry with the verified seal for already authorized test execution once separate CODE review resolves.
2. **Root-owned downstream work:** Retain nativeReady, liveService restoration, session/IAM/audit composition, official release and full race as explicit acceptance gates. Local tests do not establish native success; see [handoff:89](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-adoption-v2-sealed-bytes-handoff.md:89).

## Trade-offs

| Choice                                         | Benefit                         | Cost                                                                 |
| ---------------------------------------------- | ------------------------------- | -------------------------------------------------------------------- |
| Captured seal/helper plus continuous binary fd | Binds approved-byte consumption | Other inventory paths remain preflight evidence                      |
| Strict rollback preflight                      | Preserves sealed identity       | Local sealed binary remains required, although rollback uploads none |
| Unknown without retry                          | Avoids duplicate mutation       | Requires root inspection and guarded recovery                        |

**Stop condition met:** diagnosis and disposition are evidence-backed. Source/evidence remained unchanged; no remote execution or live-v3 rereview occurred. O3/D1/P0 closure is not granted.
