# Argo adoption v2 — R1 sealed-byte Generator handoff

Task: argo-adoption-v2-20261110. L2 parent: 2026-11-07-ops-planned-task-closeout.
Parent packet: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md.
Plan: .harness/evidence/argo-adoption-v2-20261110/argo-adoption-v2-plan.md.
Verification: .harness/evidence/argo-adoption-v2-20261110/argo-adoption-v2-final2-verification.json.

**The narrow R1 implementation and local verification are complete; independent delta CODE/ARCH and root deployment remain pending. This is a Generator handoff, not self-approval, server success or P0 approval.**

## Review input and ownership

Read the current `resume-argo-adoption-independent-code-report.md` and `resume-argo-adoption-independent-arch-report.md`. CODE requests only HIGH R1: the original runner hashes the seal/helper and then reopens them for JSON/command consumption. ARCH is CLEAR for the original bounded scope; its disposition does not waive CODE R1 or approve this new delta.

Only three new implementation/verification files were added under `.local/ops-acceptance/`: `argo-adoption-v2-root-run.py`, `argo-adoption-v2-deploy-test.py`, `argo-adoption-v2-verify.py`. Evidence is new under `.harness/evidence/argo-adoption-v2-20261110/`; this handoff is new in the current EV directory. No original helper, original seal, log, failed receipt, Argo business/backend/provider/frontend/shared source, config, credential, fork, lock or ledger was changed. No remote/private transport was loaded during local verification. No delegation, model selection, commit, PR, release or production action occurred. Other active lanes were left in place.

This is a logical tooling correction within the business-domain delivery lane; no platform/system contract or physical package split. Python stdlib and the unchanged remote deployment helper are reused.

## Exact repair and its bounds

1. `prepare` captures the new seal through one bounded regular-file open (1 MiB maximum), hashes that immutable `bytes` value against the caller's explicit `--seal`, then parses **the same value**. There is no seal reopen. Duplicate JSON keys and unsafe/noncanonical relative paths are rejected. The original seal identity, original final3 receipt, new verification receipt, required helper/entry pins and complete inventory are checked before importing/connecting transport.
2. The original deployment helper is captured once while checking the inventory. Its captured SHA256 must match the captured seal pin. `execute` builds the exact `timeout --signal=TERM --kill-after=5s 250s python3 -B -c <quoted captured helper> <action>` command from that same UTF-8 buffer. Text-mode newline conversion is avoided. There is no helper reopen or fallback to the original vulnerable runner.
3. Capture checks absolute paths, real directory ancestors, regular final file, one hardlink, no symlink or Windows reparse traversal, and matching path/fd device/inode/mode/link count/size/mtime around the read. The binary is also checked before the preflight yield. POSIX `O_NOFOLLOW`/`O_NONBLOCK` are used when available. Python 3.12 Windows `lstat`/`fstat` disagree on ctime change/creation semantics; ctime comparison is therefore POSIX-only, while device/inode/shape/size/mtime and the cryptographic byte binding remain mandatory.
4. Altered captured bytes or unsafe paths discovered during capture refuse before connection. A path change **after completed safe capture** cannot change the immutable captured seal/helper; that case uses only the approved captured bytes. Namespace atomicity is not claimed. Other inventory hashes are preflight identity evidence, not an immutable filesystem snapshot of all source paths.
5. The binary is not buffered into a 206 MiB allocation. One regular fd remains open from bounded preflight hash through stream; install rewinds that fd, sends at most the pinned size, checks the streamed source hash and refuses short/extra/changed bytes. The original helper is byte-identical and still validates the remotely received digest, disk binary/ELF identity and immutable bit **before native dry-run and the single actual Update**. A stream mismatch cannot authorize the native Update. Rollback sends no binary.
6. Root's existing private transport is imported only in `execute`, after preflight. Only peer `115.190.251.143` can receive `exec_command`. Output remains bounded to 65536 stdout / 8192 stderr bytes, with a 275-second receive deadline and the original remote/native timeouts. A lost reply remains uncertain and is never automatically retried. New redacted root receipts use exclusive new v2 paths; preflight refusal records no remote mutation, and ambiguous execution retains `remoteMutationUnknown`.

The helper still retains the inspected namespace/deployment UID/spec, host-directory identity, sole-node pin, dedicated owner path, stock image/security/RBAC/env/probes and 192Mi/500m limits, immutable binary/read-only mount, original UID/RV on dry-run/single Update, and sealed private staged rollback artifacts. Those semantics/source were not edited.

## Fresh verification and regression sensitivity

Command: `python -B .local/ops-acceptance/argo-adoption-v2-verify.py --label final2` → **exit 0**; all six recorded checks passed, new source hashes stable before/after, **67 preserved existing source/evidence files stable**, and **all original 31 inventory entries identical**.

- `python -B .local/ops-acceptance/argo-adoption-v2-deploy-test.py` → **20/20 PASS**. The fake transport only records commands; it never executes them. Small temporary files provide real fd metadata, while both R1 replacement attacks modify memory-backed reads only. An altered first capture refuses with zero connect/exec. A seal replacement after digest cannot substitute an empty inventory: an altered helper still fails its original pin before connection. A helper replacement after its digest executes only the captured approved bytes. Valid capture preserves exact UTF-8/CRLF/shell quoting, with one seal open, one helper open and one binary fd. Other probes cover unsafe links/reparse/parents/shapes, path/fd drift, size/stream/output/deadline bounds, wrong peer, rollback/no upload and lost-reply/no retry.
- `python -B .local/ops-acceptance/argo-adoption-deploy-test.py` → **7/7 PASS**, original test file unchanged.
- Python 3.9 AST grammar and fresh in-memory `compile` of all three new files → **PASS**, no imports, bytecode or diagnostic output. No Python language-server/static-linter executable was available; AST/compile, runtime tests and source review are the recorded diagnostic coverage.
- Scoped `git diff --check --no-index` of old/new runner → **empty diagnostics**. Its exit 1 means the expected no-index source delta exists. All three new files were additionally checked for trailing whitespace; the exact runner delta and complete new tests/verifier are saved in the sealed verification receipt's `.diff` artifact.
- Two in-memory sensitivity mutants reintroduce seal or helper rereads. Each selected regression produces exactly **one assertion failure**, not an import/setup error. Their expected nonzero results are PASS signals for the checker. Neither mutant was written to a source file or run against a real transport.
- Actual local `prepare` using the full new candidate seal and actual original inventory/helper/binary → **PASS**. Independent AST assertions find one `capture(SEAL)` site and one inventory `capture(path)` site, no `read_text/read_bytes/open` consumption in `prepare`; shell command roundtrip preserves the captured helper SHA. `remote` is absent before and after. Receipt: `argo-adoption-v2-final2-actual-preflight.json`, SHA256 `82aa35dc23f4ba2cb60ff561cf5ecc4319407d256019a375e3d2dd5056090ee4`.

The new `local1` failed verification receipt/logs remain intact and are **not** promoted: offline/AST checks passed, but the verifier initially misclassified git's no-index exit 1 and its mutant subprocess used the wrong unittest module namespace. Only the new verifier was corrected before final2. Initial Windows ctime incompatibility was corrected before the sealed final source; no original artifact was edited.

Gate Outcomes: memory replacement probes catch seal/helper reopening | fd/path/bounds/peer guards catch unsafe capture or exec | mutation sensitivity verifies the probes | Windows ctime mismatch and no-index exit semantics were local false positives, corrected without weakening checked-byte binding.

## New seal and preserved source identity

New candidate request seal: `.harness/evidence/argo-adoption-v2-20261110/argo-adoption-v2-artifact-seal.json`.

**New SHA256: `fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03`.**

The new seal retains every original `files` key and value, original final3 receipt reference/pin and binary pin/size, then adds exactly three new source pins (**34 total files**). It separately binds the new verification receipt. The original artifact seal is a distinct immutable input and was not rewritten.

| Artifact                                  | SHA256                                                             |
| ----------------------------------------- | ------------------------------------------------------------------ |
| Original 31-file artifact seal            | `08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec` |
| Original final3 verification receipt      | `051e9fd95eead702b16da2e6092fc3f4bafc8d4129ae070089741aac11444d97` |
| New final2 verification receipt           | `e37070716df92af87a5a8a959c7a2ee951caee16ee118c73fc9b67876b2cb735` |
| New v2 runner                             | `7957878150d1d84b449fd7a8a0c8d277d34dc71a0c5eb999c00adbdc27d02419` |
| New v2 tests                              | `851bd8c8139b503138cf24567d9c687d4f48adbf0bf90848d562404a860743f8` |
| New v2 verifier                           | `3a69d2b33a0524c43aa7859c30b5a82d8c884d169629c021c0f954b7410d5e79` |
| Exact source delta                        | `b08a95edb1fe558a7e85720d8d239f4560b30f0ddcad790ed2956f971da0308e` |
| Unchanged deployment helper               | `bd71ed1edb5caa8d4dadd87beb49eaad0a282ba79a4bc236056eeee7cbac514e` |
| Original vulnerable root runner, retained | `0106df0229fd740daaa49294eb6c10833c84d074aafc3cd6c262cad225f2ab87` |
| Unchanged provider README                 | `18aeb848cecccd0b3e0a19091f9eb3ddbbbbc296223b6cd4990e35ca89d56192` |
| Unchanged provider patch                  | `a21740245ef6f35a6da8337ab05e6771e8c376fb4bcc70493cc51ac2e5dfc852` |
| Unchanged Linux binary, 206041319 bytes   | `cd68356983230edfa0c013235eccba4381a5fbe00d7076f47b7d4b7b8578138b` |

Provider upstream remains `66b2f302d91a42cc151808da0eec0846bbe1062c`. Existing independently reviewed 4828-source provider evidence remains historical evidence; the fork was not changed or rerun. The original 25 Go files and original 181 non-group/22-parent passing package evidence remain unchanged and approved-scope ready. They were not rerun or relabeled as fresh full-race evidence.

## Root commands after independent delta review

Root must independently review the exact source delta, new seal/receipt, R1 probes and unchanged-helper identity. The SHA below is a **candidate supplied for that review**; deployment requires root's reviewed SHA supplied explicitly by the caller. The runner does not discover or accept its own local seal hash as authorization.

From repository root, using root's already authorized private test-host transport:

```powershell
python -B .local/ops-acceptance/argo-adoption-v2-root-run.py install --seal fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03
```

Root-owned reversal with the same independently reviewed seal:

```powershell
python -B .local/ops-acceptance/argo-adoption-v2-root-run.py rollback --seal fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03
```

Use the v2 entry and v2 seal together. The old runner remains preserved evidence and has no fallback path from v2. Root receipts go only to `.harness/evidence/argo-adoption-v2-20261110/argo-adoption-v2-root-<action>-<fresh-UTC-stamp>.json`; old failure receipts remain unchanged. An uncertain install is not retried: independently inspect and, only in the helper's exact recorded UID/spec/owner state, use staged rollback.

## Remaining gates and stop condition

Independent delta CODE/ARCH are pending; this handoff does not close them. No actual server/native deployment or rollback success is claimed. Root still owns host filesystem/ABI/node proof, actual immutable inode and mounted digest, native UID/RV/receipt seals, readiness/restarts/RSS within 192Mi, real reviewed capability/Sync/Rollback/controller Service restoration, token/policy/session/IAM/DataScope/bizscope/audit composition, lost reply/cold recovery and overwritten-evidence Unknown behavior. Full stable CGO repository race and formal overlay/rebuild/compatibility remain root-owned. O3/D1/P0 acceptance and release remain open.

Stop condition reached: only R1 tooling changed, deterministic local verification and full actual preflight passed, exact new candidate seal and commands delivered for independent review. No further Generator-side scope is required.

```json
{
  "taskId": "argo-adoption-v2-20261110",
  "parentTask": "2026-11-07-ops-planned-task-closeout",
  "role": "Generator",
  "status": "implemented-and-locally-verified-awaiting-independent-delta-review",
  "independentApproval": false,
  "P0Approved": false,
  "changedOriginalFiles": 0,
  "newPythonFiles": 3,
  "originalInventoryFilesPreserved": 31,
  "newInventoryFiles": 34,
  "offlineTestsPassed": 20,
  "originalHelperTestsPassed": 7,
  "rereadMutantsCaught": 2,
  "actualLocalPreflightPassed": true,
  "remoteExecuted": false,
  "credentialsRead": false,
  "fullRaceRerun": false,
  "sealSHA256": "fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03",
  "originalSealSHA256": "08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec"
}
```
