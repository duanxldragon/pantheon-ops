## Code Review Summary

**CODE: REQUEST CHANGES**

**Files checked:** 29 — 25 Go files and four Python helpers.  
**Issues:** CRITICAL 0 · **HIGH 1** · MEDIUM 0 · LOW 0.

### [HIGH] R1 — Deployment can execute helper bytes that were never sealed

File: [argo-adoption-root-run.py:58](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-root-run.py:58). The same defect affects the seal reread at [line 48](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-root-run.py:48).

**Cause:** The runner hashes files, then reopens `artifact-seal.json` to parse it and the deployment helper to construct the remote command. A concurrent replacement between those reads can change what executes after the approved hash check has passed.

**Reproduction:** Two deterministic, memory-only probes used the current runner with a mocked transport:

- After all original hashes passed, the helper’s second read supplied different content. That unsealed content reached the simulated root execution command.
- After the approved seal hash passed, its second read supplied an empty file inventory. The runner skipped the original helper verification and again constructed the unsealed command.

Neither probe wrote files, loaded credentials, connected remotely, or executed the captured command. Both demonstrate the missing binding between checked bytes and consumed bytes.

**Minimal repair:** Read the seal once, hash that captured byte buffer, then parse that same buffer. Capture the helper once, verify those bytes against the captured seal, and construct the command from those exact bytes. Retain regular-file/path checks. Add regressions for both replacement windows requiring rejection before transport connection, then produce a new seal for independent review.

This blocks readiness for authorized test deployment.

### Contract assessment and old-finding mapping

The **业务域 adapter contract passes** within the agreed provider scope:

- Strict six-field negotiation; distinct reviewed Sync/Rollback POST endpoints.
- UID/RV/revision/marker and private history-ID binding, including ID zero.
- Current authorization after the final capability read.
- One dispatch, no stock/native/PATCH fallback.
- Exact-ticket recovery preserves Unknown without redispatch.
- Different historical source configurations remain fail-closed; revision scope remains lowercase 40-character SHA1.

No masking fallback was found. The pinned-revision compatibility branch remains narrowly documented at [argocd_preview.go:201](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:201).

Earlier findings remain repaired:

| Earlier finding                                         | Current disposition                                               |
| ------------------------------------------------------- | ----------------------------------------------------------------- |
| Misleading state after bookkeeping failure              | Unknown reservation and sanitized persistence diagnostic retained |
| Private timestamp values exposed through decoder errors | Provider decoding returns a safe error                            |
| 408/499 classified as definite rejection                | Both remain uncertain                                             |
| 412/413/415 left Unknown                                | Recognized standard rejection codes are terminal                  |
| HTTP 200 without persisted-operation evidence           | Remains Unknown                                                   |

R1 is a new deployment-helper finding.

### Validation and artifact identity

Fresh `gopls check` completed without diagnostics on all 25 Go files. Python AST checks and **7/7 offline helper tests passed**. Dedicated `lsp_diagnostics`, `ast_grep_search`, and Python language-server tools were unavailable.

The retained race log and every verification-log checksum matched: **181 non-group passes, 22 grouping parents, zero failures**, and the two documented live skips. Race tests were inspected, not rerun.

All **31 sealed file hashes matched before and after review**:

- [Artifact seal and complete hash inventory](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/argo-adoption-20261110/artifact-seal.json): `08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec`
- Verification receipt: `051e9fd95eead702b16da2e6092fc3f4bafc8d4129ae070089741aac11444d97`
- Affected root runner: `0106df0229fd740daaa49294eb6c10833c84d074aafc3cd6c262cad225f2ab87`
- Deployment helper: `bd71ed1edb5caa8d4dadd87beb49eaad0a282ba79a4bc236056eeee7cbac514e`
- Linux binary, **206041319 bytes**: `cd68356983230edfa0c013235eccba4381a5fbe00d7076f47b7d4b7b8578138b`

The previously approved provider patch was not rereviewed across its 4,828 upstream files.

**Stop condition met:** the scoped CODE blocker is reproduced and documented. ARCH disposition remains with its separate lane. Base originating-session/HTTP/audit composition, real provider/controller/Service restoration, live rollback/recovery, 192Mi readiness, and full O3/D1/P0 acceptance remain downstream gates. No source edits, deployment, or release occurred.
