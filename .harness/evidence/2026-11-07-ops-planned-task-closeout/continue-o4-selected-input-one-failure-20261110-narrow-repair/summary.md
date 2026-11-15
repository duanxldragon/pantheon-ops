# O4 selected-input remove assertion repair

Parent: [2026-11-07-ops-planned-task-closeout](../../../../docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md). L2 Generator scope: Ops consumer test-only; platform/system/product implementations unchanged.

Only `scripts/business-overlay/foundation-candidate.test.mjs:501` changed: one assertion replaced (10 added lines, 1 removed). Remove mutation accepts the original stability rejection or exact `ENOENT` for the fixture's `.harness/report.md`, unwrapping only `consumer.CandidateCleanupError`. Other mutation expectations remain the original stability regex. `assert.throws` and absence of `.foundation-candidate-receipt.json` still require refusal without consumer success. [Exact delta](delta.patch).

Fresh verification (Node v24.14.0):

- Original remove assertion reproduced: 1 test, 0 pass, 1 fail, 0 skipped. [Red TAP](../../../../.local/ops-acceptance/continue-o4-selected-input-one-failure-20261110-narrow-repair-red.log).
- `node --check scripts/business-overlay/foundation-candidate.test.mjs`: exit 0. Build/typecheck not applicable to this test-only assertion change.
- `node --test --test-reporter=tap --test-name-pattern='^selected inputs:' scripts/business-overlay/foundation-candidate.test.mjs`: 16 tests, 16 pass, 0 fail, 0 skipped, 34.037s. Name filtering selects 16 cases; the full run covers all cases. [Focused TAP](../../../../.local/ops-acceptance/continue-o4-selected-input-one-failure-20261110-narrow-repair-focused.log).
- `node --test --test-reporter=tap scripts/business-overlay/candidate-ownership.test.mjs scripts/business-overlay/foundation-candidate.test.mjs`: 124 tests, 124 pass, 0 fail, 0 skipped, 187.516s. [Two-suite TAP](../../../../.local/ops-acceptance/continue-o4-selected-input-one-failure-20261110-narrow-repair-two-suite.log).
- All runs have 0 cancelled. [Structured verification](verification.json).

[Preservation checks](preservation.json): exact pre-repair source/ignore/official lock/manifest hashes unchanged; owned test equals the exact assertion replacement. Original baseline protected hardlink pair remains inode `281474979169911`, nlink 2, 57 bytes with unchanged content hashes; original caches remain 62,530,476 and 55,594,218 bytes. Baseline device IDs/modes also match. Large caches were inspected only with `lstat`.

Historical-baseline limitation: all five `sourcesBefore` hashes in the supplied baseline already differed before this repair, including the test. Those existing changes were preserved; no historical source-hash equality is claimed. [New pre-repair snapshot](../../../../.local/ops-acceptance/continue-o4-selected-input-one-failure-20261110-narrow-repair-before.json).

Handoff to Dispatcher: scoped test repair verified. Actual complete candidate assembly and independent final delta reviews remain OPEN. This evidence is implementation/verification evidence, not approval, parent closeout, official consumption, or publication.
