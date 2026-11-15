# O2 v9 receipt preservation repair — root handoff

Task: `2026-11-07-ops-planned-task-closeout`. Tier: L2. Parent: `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`. Owner of review/sealing/execution: root. Generator scope: new local preservation helpers and evidence only. Business-domain O2 acceptance tooling; no platform/system-domain change or physical domain split. Minimal approach: reuse the exact original sealer and root validator, with Python standard-library checks for one explicit receipt path migration.

The failed receipt was recovered exactly. The original continuation inventory remains unchanged. The successful current adapter receipt remains at its original path and has an independent exclusive-create archive. A detached helper is ready for root review; this leaf did not write a root-reviewed request, execute the matrix, load credentials, or approve O2/P0.

## Finding and recovery evidence

`p0-role-adapter.py` writes the same report receipt at the end of every invocation. The original continuation inventory captured the first failed adapter receipt before the successful continuation finished. Completion replaced the live path with the success receipt. The original sealer correctly rejects this mismatch; its source and the original candidate request/verification are preserved.

The copied first-failed event stream proves the failed turn but does not contain the adapter's timing receipt. The retained root terminal artifact does contain an exact `Get-Content` read of the original receipt:

- Terminal: `C:/Users/xiaolong/.codex/sessions/2026/11/09/rollout-2026-11-09T08-41-47-01a1bdc1-0ba7-74b1-8ba9-77da0c49d07c.jsonl`, line **8062**, timestamp `2026-11-10T01:27:57.166Z`, command exit 0.
- Exact raw terminal line SHA256: `7f3e692b0d36d46a474bae4dd94e4e99a14e81aceb2f61555ded3a4f7886501d`.
- Extracted original receipt: role `executor`, exitCode **1**, elapsedSeconds **606.797**, modelOverride `false`, report `null`.
- Recovery copies the exact CRLF JSON substring and trailing CRLF from that output. It does not infer timing, regenerate fields, or use the success receipt as failure evidence.
- Original failure is **122 bytes**, SHA256 **`2bf315de892fa85c24918844c55189bc4f1cb5d56abec79345578f3037a60c85`**, exactly the pre-existing expected hash. The LF-only representation would have a different hash and is not used.
- Current success is **249 bytes**, SHA256 **`a544e0d33cea2ee0887d20738fbb097de6a1b5f25ec7263adcb21788f53e4187`**. Its exact bytes were copied before recovery work, and its live path was not overwritten.

The retained terminal log was read only. Only the receipt substring and narrowly scoped provenance were retained locally; no full terminal-output copy or credential/config load was performed.

## New artifacts and exact hashes

All paths in this table are under ignored `.local/ops-acceptance/`. All receipts/logs use exclusive creation; subsequent content changes fail SHA256 checks. They are hash-bound evidence, not filesystem WORM storage.

| File                                                                                                              | SHA256                                                             |
| ----------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `resume-o2-v9-preservation-failed-2bf315de892fa85c24918844c55189bc4f1cb5d56abec79345578f3037a60c85.receipt.json`  | `2bf315de892fa85c24918844c55189bc4f1cb5d56abec79345578f3037a60c85` |
| `resume-o2-v9-preservation-success-a544e0d33cea2ee0887d20738fbb097de6a1b5f25ec7263adcb21788f53e4187.receipt.json` | `a544e0d33cea2ee0887d20738fbb097de6a1b5f25ec7263adcb21788f53e4187` |
| `resume-o2-v9-preservation-recovery.json`                                                                         | `26029fdb8504b5cdb13af2e0dda6fb31f6c8895165d9a3a0177b916994c58daf` |
| `resume-o2-v9-preservation-migration.json`                                                                        | `35bde400a39ed63428a5e3b5df56f28c46b8123d17509b0c607009038b0fce5b` |
| `resume-o2-v9-preservation-recover.py`                                                                            | `0436bb02fd7c7d382e455bfa072417ce33dfced846ba159b3702ba324bc03476` |
| `resume-o2-v9-preservation-root.py`                                                                               | `cb1cb242feea5d6eb61d20f6a2733f79d322c5a0bd57b36ba58e2a3943f2a190` |
| `resume-o2-v9-preservation-root_test.py`                                                                          | `8538e28a547af66a428a0eb7da137e8b077a038d0f5a3e8c97a5310eaf595cf0` |
| `resume-o2-v9-preservation-verify.py`                                                                             | `e4f9c04ed1563368e713487135dadabd1895a86c3f775ccc00e00ea6ff0b4703` |
| `resume-o2-v9-preservation-verification-1794276695676947300.json`                                                 | `179a1ca689afd5c9e02ef8d35bf403922e78a5109d836ac5c1c75cf56effc8d4` |

The requested handoff is this new P0 evidence file. No `.gitignore`, tracked source, original handoff, request, verification, inventory, or sealer was edited. Do not rerun `preservation-recover.py` over these archives: it deliberately refuses existing files.

## Every original expected hash is retained

Original inventory: `resume-o2-v9-continuation-preserved-before.json`, unchanged SHA256 **`cdbe5fef9d2ff368019e103587efa7f33766910b40858c368d4d42fa9a3fb67a`**. The following names are in the current P0 evidence directory except the explicitly marked local events path.

| Original inventory entry                                                 | Original expected SHA256                                           | Resolution                                       |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------ |
| `resume-o2-v9-failed-verification-1794273737440186200.json`              | `66dab6e8863e789d5aed1139e7fb56091f3986aa28c723c644021038cbd3391c` | Original path                                    |
| `resume-o2-v9-failed-verification-1794273852572054400.json`              | `8338e17f9da467209c35379e98efd52d40ed59b988fb0f659432cdb50d0d04d5` | Original path                                    |
| `resume-o2-v9-failed-verification-1794273871455159300.json`              | `ae68f3a57707d77f124061544cad768691288558677230d7d0054108b0bd83e2` | Original path                                    |
| `resume-o2-v9-generator-report.receipt.json`                             | `2bf315de892fa85c24918844c55189bc4f1cb5d56abec79345578f3037a60c85` | Explicit mapping to the new failed archive above |
| `resume-o2-v9-verify-1794273737440186200-0.txt`                          | `9b37cc5d7b2bb624ff5896d4bedede5f1c55f52ec62b6feab9926bf3be64af4e` | Original path                                    |
| `resume-o2-v9-verify-1794273852572054400-0.txt`                          | `719a3d9a80108d6f7767ab9cc9f48163c8b32ce924b583d76f4de662a8939285` | Original path                                    |
| `resume-o2-v9-verify-1794273871455159300-0.txt`                          | `ebd408c950a3f89659fd4a7c4a2caac69252759919e26362f1389f9322630579` | Original path                                    |
| `resume-o2-v9-verify-1794274208029252500-0.txt`                          | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Original path                                    |
| `resume-o2-v9-verify-1794274208029252500-1.txt`                          | `21eb04b41d8b6027c1217a37f13ed6fe418fc616b76362113dd18ae5fa6919e4` | Original path                                    |
| `resume-o2-v9-verify-1794274208029252500-2.txt`                          | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | Original path                                    |
| `.local/ops-acceptance/resume-o2-v9-generator-first-failed-events.jsonl` | `ef23b1039f318cbe779716cbf583733da0578db1de1b97b3e8d5a1c413d549ec` | Original path                                    |

The map changes only where the failure bytes are found. It never changes their expected hash. The live success receipt and success archive are separately checked against the success hash. The original unmapped predicate still returns false at the overwritten live path; this repair does not pretend that path still contains a failure.

## Detached root continuation behavior

`resume-o2-v9-preservation-root.py` pins the original candidate request, original inventory, original sealer, and migration receipt. It also checks all original candidate evidence seals. The original sealer hash is `76a71e7360368b02acce2ebbe4edbe70ae0a41bfca8ab7ed29e325ebe0f9b6b5`; the original local candidate request hash is `46aba1c0f2e5c451898e9b0efb54e32206f6e5219b268b27b581bbed037d64e2`.

The helper compiles the original sealer in memory after replacing exactly one unique expression:

```python
all(digest(ROOT / p) == h for p, h in continuation.items())
# becomes only:
preservation_continuation_check(continuation)
```

That function requires the exact original 11-entry inventory, the pinned one-entry migration, exact failure/success bytes, retained events and recovery provenance, and every other original expected hash. Additional mappings, arbitrary archive paths, changed contents, success-as-failure, missing/symlink archives, changed candidate/evidence/sealer, and changed expected hashes fail closed.

All remaining original sealer code runs unchanged, including the exact root-guard-only change check, original five-command successful verification and log checks, 102 prior artifact preservation, v8 product inventory comparison, fresh actual two-platform dependency inventories, complete prior matrix checks, and original root validator. No transport tests, matrix, product code, SQL, leases, or 45 assertion call sites are changed.

The helper wraps only original reviewed-request writes to add preservation evidence hashes without replacing any existing hash. Both original reviewed-request destinations receive identical augmented JSON. The request binds all 11 resolved failure artifacts, the current success receipt and its archive, migration/provenance, role adapter, recovery helper, root helper and its tests. Existing product/candidate seals remain intact. `preservationMigration` records the explicit map and original inventory SHA/count. The unchanged original root entry can check and execute that request; no new execution or case-subset entry is introduced.

`--check` stops before the first request write, then passes the proposed request in memory to the unchanged root `validate`. It writes no root-reviewed request or approval artifact. `--reviewed` is for root after review: it lets the original sealer exclusively create the separately reviewed requests and run its existing local root check. Root's prior `ROOT_REVIEWED = True` change remains untouched, current entry SHA256 `81b8a3a53646f2b46d75e9d69ee670f30111be6ae0046bd49625d9fde2b26486`.

## Fresh local verification

Executed `python -B .local/ops-acceptance/resume-o2-v9-preservation-verify.py`; failed **0**, exit **0**. Receipt and all command log hashes are listed above/below. This wrapper recorded **338** protected pre-existing files before and after; their hashes were identical. All four repair Python input hashes were stable throughout verification.

| Check                                                                       | Result                                                                                                                                                                   | Log SHA256                                                         |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------ |
| Python AST and compile diagnostics for all four new helpers/tests           | 4 passed, no bytecode writes                                                                                                                                             | Embedded in verification receipt                                   |
| `python -B .local/ops-acceptance/resume-o2-v9-preservation-root_test.py`    | **22 tests passed**, exit 0                                                                                                                                              | `32851b932b4ca3f14c26773ea2117e8e04bd895554eef3b151a767130b822882` |
| `python -B .local/ops-acceptance/resume-o2-v9-preservation-root.py --check` | Original reviewed sealer checks and original root validate passed; **185 dependencies**, **11 preserved failures**, **1 explicit migration**; no seal/remote/credentials | `64a6ff7f1bb281d01c0c54f4464aab73870c9cfff1c5ff93b4a6c67c1379376d` |
| `node scripts/frontmatter-check.mjs`                                        | Passed, 462 docs checked / 301 with frontmatter                                                                                                                          | `788c653eaa0219525083583fb7eedf5f5709889eb4b101adc8ff7b101a84112c` |
| `node scripts/check-task-packet-template.mjs`                               | Passed                                                                                                                                                                   | `5237988e973ab745de408597063e4d36f1f7530246ce423600fcf6362c5b3495` |

Logs are `.local/ops-acceptance/resume-o2-v9-preservation-checks-1794276695676947300-{0,1,2,3}.txt`. Tests use in-memory tamper fixtures and simulated request writers; they do not modify retained files or seal approval. The standalone `--check` row is the actual unchanged sealer/root validation with real dependency inventory, distinct from mocked test coverage.

Self-review: no unresolved receipt-byte recovery gap. The helper has no execution branch; root alone must review and invoke reviewed sealing and the unchanged complete matrix. Runtime/product/transport validation was not repeated because product and candidate code were prohibited from mutation and stayed sealed. This local repair is not VM acceptance or independent O2/P0 approval.

Gate Outcomes: original preservation gate caught adapter receipt overwrite | explicit mapping checks caught arbitrary substitutions, changed contents, widened mappings, and relabelled success in local tests | no waiver of original expected hashes.

## Exact root commands after review

Run from `D:/workspace/go/pantheon-platform/pantheon-ops`. First confirm the helper SHA256 equals **`cb1cb242feea5d6eb61d20f6a2733f79d322c5a0bd57b36ba58e2a3943f2a190`**, and inspect the pinned migration/provenance and verification receipt. Do not edit the original sealer, candidate request, original verification/inventories, driver/tests, or product sources.

```powershell
Get-FileHash -Algorithm SHA256 .local/ops-acceptance/resume-o2-v9-preservation-root.py
python -B .local/ops-acceptance/resume-o2-v9-preservation-root_test.py
python -B .local/ops-acceptance/resume-o2-v9-preservation-root.py --check
```

Root only, after reviewing this exact extension, consumes the explicit preservation map and creates the new reviewed requests:

```powershell
python -B .local/ops-acceptance/resume-o2-v9-preservation-root.py --reviewed
python -B .local/ops-acceptance/resume-o2-v9-root-execute.py --phase check --request .local/ops-acceptance/resume-o2-v9-root-reviewed-phase-request.json
```

Only after those commands pass, root may use the unchanged previously reviewed complete matrix command under its existing authority:

```powershell
python -B .local/ops-acceptance/resume-o2-v9-root-execute.py --phase execute --request .local/ops-acceptance/resume-o2-v9-root-reviewed-phase-request.json
```

At handoff both reviewed-request paths were absent. If reviewed outputs already exist by the time root acts, preserve them: the original exclusive-create writer refuses overwrites. Do not reseal over a changed source, evidence, helper, map, or live adapter receipt. Stop and retain the failed check evidence. No `--reviewed` or `--phase execute` command above was run by this leaf.

Stop condition reached: exact failure bytes recovered and hash-verified, exact success bytes preserved, additive migration and detached helper locally verified, original artifacts unchanged, and concrete root review/sealing commands supplied. Parent L2 VM matrix and O2/P0 review remain root-owned.
