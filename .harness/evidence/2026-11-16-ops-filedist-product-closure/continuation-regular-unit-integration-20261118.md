# E01 FileDist regular-unit integration receipt

Parent: `2026-11-16-ops-filedist-product-closure`; implementer `/root/e03_claude_partial_closeout`; observed UTC `2026-11-18T08:07:12Z`.
Plan: `../../tasks/2026-11-16-ops-filedist-product-closure/continuation-regular-unit-integration-plan-20261118.md`.

The UI executor's frozen 28-case regression suite now lives at `frontend/tests/unit/business/filedist/FiledistLifetime.test.ts`, covered by the standard Vitest unit glob. Its exact path is declared in `business-overlay.json`, preserving the C03 declaration and other owners' entries. Production sources, original evidence fixture and manual runner are preserved.

The first byte-identical copy passed all 28 cases (14.63s), while targeted strict TypeScript returned exit1/TS2352 at 163:46 because the resolution mock omitted required fields. This actual failure is retained in `regular-unit-initial-verification-20261118-01.json`; the initial tool output was not persisted as a raw file, so the receipt records only the inherited actual result and diagnostic summary. It does not manufacture a raw RED log.

The canonical mock now supplies fixed `resolvedAt`/`sources`, plus each host's required `status`/`businessScopeId`/`source`; the obsolete `as DeployTargetResolution` assertion is removed. The typed declaration directly checks the complete contract. A normalized original-vs-regular comparison passes with exactly those changes; all assertions, imports and 28 cases remain unchanged. The original fixture hash remains `0d3d66ea394e3c92dea41b5ce8fb19da5395fac4a23ff66b7a44ee54a0554256`.

## Fresh verification

Full command output is retained in `regular-unit-verification-20261118-02.json`.

| Check                                                                                  | Actual result                                               |
| -------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| Standard-config/default-include Vitest, native Node plus existing hidden-child preload | Exit0; 1 file, 28 passed, 0 skipped; 12.39s                 |
| Targeted strict TypeScript using `regular-unit-typecheck-20261118.json`                | Exit0; no diagnostics; test and imported production only    |
| Scoped ESLint `--max-warnings 0`                                                       | Exit0; no diagnostics                                       |
| Business overlay checker                                                               | PASS; 55 declared paths, rebuilt tree                       |
| Scoped `git diff --check`                                                              | Exit0                                                       |
| Original/regular fixture comparison                                                    | PASS; only complete typed mock data and cast removal differ |

Vitest used the threads pool, one worker and no file parallelism. All Node entries used the existing hidden-child preload. No new process implementation, dependencies, shared configuration or global settings were introduced. The pre-existing `__dirname` future native config-loader warning remains visible and was not suppressed. No new visible-window observation was performed.

## Frozen scope and remaining gates

The canonical test and narrow manifest declaration are frozen for the coordinator's official assembly. Their final hashes and the imported production bindings are in the adjacent JSON. The overlay checker above is not an official locked-release reconstruction or drift proof.

The 28 cases cover sibling-request settlement before polling, identity changes/late responses, terminal polling, background error recovery, unmount/malformed identity, 404/403/503 classification, cancellation mutual exclusion and permission, resolved target/auth/fingerprint intent, safe schedules, paging/search beyond 100 files, stale list refreshes, and a fresh mutation read after an older deletion-related GET.

Impeccable applies to these lifecycle/control states. This lane made no layout or production UI changes and produced no desktop/mobile rendering; no visual polish is claimed. Full assembled frontend type/build/smoke, official reconstruction/drift, backend lifecycle/partition, current runtime/provider and roles, rendered evidence, independent review and whole E01 human acceptance remain with the coordinator. E01 is not marked closed by this integration.
