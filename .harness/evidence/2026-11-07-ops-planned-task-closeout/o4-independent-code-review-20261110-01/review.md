# Independent CODE review — O4 integration repair

Parent: `2026-11-07-ops-planned-task-closeout` (L2). Scope: exact six sources in `o4-generator-integration-repair/source-hashes.json`, relative to `repair.patch` and retained failed-stage baselines. Verdict: **REQUEST CHANGES**. Four findings: CRITICAL 0, HIGH 3, MEDIUM 1, LOW 0.

1. **[HIGH] File writes bypass directory identity guards; a replaced directory can receive a success receipt.** `scripts/business-overlay/foundation-candidate.mjs:915` exposes raw `copyFileSync`/`writeFileSync`; only mkdir/rm are wrapped. `scripts/business-overlay/rebuild-from-candidate.mjs:183` verifies bytes but never revalidates assembled-directory ownership before accepting. In the real small-fixture `rebuildFromCandidate`, rename `assembled/docs` after its mkdir and immediately before copying `docs/base.md`, recreate it with a foreign file at that name, then let native copy run. Foreign bytes are overwritten and the tool emits a receipt with `sourceStable: true`. Evidence: `replacement-assembly-results.json`, `replacement-assembly-check.log`. Fix: guard every target mutation and final receipt with directory/ancestor identity; include direct writes and Go subprocess mutations. Refuse observed replacements before writing. Preserve explicit non-atomic namespace limitations instead of treating content equality as ownership proof.

2. **[HIGH] Exclusive mkdir does not authenticate the directory subsequently enrolled.** `scripts/business-overlay/foundation-candidate.mjs:562`. Replace the newly created `assembled/docs` between native mkdir completion and `directoryIdentity()`: the replacement inode is registered as owned. Cleanup then attempts to unlink its foreign file. The independent unlink interceptor refused deletion, keeping both original and replacement. Evidence: the named mkdir-capture case in `adversarial-results.json`. Fix: bind creation and mutation/cleanup authority to the actual created directory instance through an appropriate handle-based primitive, or retain ambiguous instances without granting destructive ownership. A later path lstat cannot establish creation ownership. Cover precisely this syscall boundary.

3. **[HIGH] Directory ownership authorizes deletion of foreign regular files.** `scripts/business-overlay/foundation-candidate.mjs:884`. Add `foreign-sibling.txt` to an unchanged registered `assembled` directory. Preflight accepts it solely because it is a regular file; cleanup attempts unlink. Interceptor preserved it. This also exists in the prior cleanup model, but the repair expands it to fully assembled output and newly enables successful automatic cleanup there. Evidence: named foreign-regular-sibling case in `adversarial-results.json`. Fix: require creation/write receipts and exact file identity for unlink; reject unknown or replaced files before any deletion, without enrolling them by scanning. An unrelated sibling stage directory is correctly preserved.

4. **[MEDIUM] Valid single-file exclusions regress.** `scripts/business-overlay/foundation-candidate.mjs:929` always sends rm targets to a directory walker. A manifest `baseExcludedPaths: ["docs/base.md"]` now fails with `Unknown directory ownership` for that regular file; the previous native rm supported it. Current actual exclusions are directories, so this is a contract regression rather than the retained O4 failure. Fix: distinguish authenticated owned regular-file removal from directory traversal and retain unknown entries. Evidence: named single-file case in `adversarial-results.json`.

Spec/classification assessment: fixed replay uses existing assembler transformations; manifest/report identity, canonical bytes, declared imports and locale-source hashes are checked. Candidate package merging is constrained by original inputs. Forged arbitrary `generatedFiles` does not become an exemption. Out-of-coverage legacy allowances remain visible; genuinely stale in-coverage entries fail. Official release resolver/CLI source selection is unchanged; legacy package handling remains distinct from candidate merging. No newly introduced masking fallback found. Stage 1 fails on ownership safety and file-exclusion compliance; ordinary quality approval is withheld.

Validation: Node v24.14.0; focused existing regressions **85 passed, 0 failed, 0 skipped** (name-selected subset, not the full 155 suite); independent checks **five defect scenarios observed plus two positive checks**. BigInt inode IDs 9007199254740992/9007199254740993 remain distinct; primary cleanup cause survives; external sibling stage survives. Descriptor/tar pin, malformed archive, import/report tamper, legacy stale behavior, ancestor replacement, and immutable `--attempt` replay/tamper tests pass. Read-only original checker: 19 canonical outputs including packages, 801 identical shared files, 10 migrations; no assembly/runtime receipt inferred. Six source hashes and six baseline hashes match Generator receipts before/after; six Node syntax checks pass. Retained snapshot 1141/1141 and report-owned files 4634/4634 match; original report SHA256 remains `13ff60276a0c94df5ca26a371e70c40b56bf1d3bb178e5e3f900b0e980c7e337`. Official lock, overlay manifest and allowed-drift hashes match supplied pins.

Skips/gaps: CodeGraph, `lsp_diagnostics` and `ast_grep_search` are not callable. Native syntax checks and literal security scans do not replace LSP/type diagnostics. No full candidate assembly, remote/credential access, actual official CLI consumption, semantic frontend build/type-check, Go race or paired runtime was run. Heavy metadata-count/archive-volume and executable-receipt suites were not rerun. Rendered UI is outside this script-only scope. All original failed stages and all refused/replaced fixture directories remain. Initial review harness import failures are retained separately. Namespace atomicity/threat-boundary decisions go upward to root/ARCH; this CODE review grants no ARCH, P0 or release approval.

## Machine Readable

```json
{
  "taskId": "2026-11-07-ops-planned-task-closeout",
  "lane": "CODE",
  "tier": "L2",
  "verdict": "REQUEST CHANGES",
  "filesReviewed": 6,
  "severityCounts": { "CRITICAL": 0, "HIGH": 3, "MEDIUM": 1, "LOW": 0 },
  "evidenceDir": ".harness/evidence/2026-11-07-ops-planned-task-closeout/o4-independent-code-review-20261110-01",
  "productEdits": false,
  "sourceHashesStable": true,
  "lspDiagnostics": "unavailable",
  "actualCandidateAssembly": "not run",
  "releaseApproval": false
}
```
