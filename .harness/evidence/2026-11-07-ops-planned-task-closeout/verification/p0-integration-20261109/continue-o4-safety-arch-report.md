## Summary

**Architectural status: `WATCH`; ARCH CLEAR is withheld.** The four original findings are addressed at the observed mutation boundaries, but primary Go errors are still lost in one reachable failure path. The remaining ownership concern is the avoidable path-based unlink window.

This remains a logical change to consumer/business-owned assembly tooling. Base-first platform/system ownership and canonical derivation checks remain enforced ([manifest guard](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:180), [canonical replay](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:681)).

## Analysis

- **Original HIGH1:** Writes now use exclusive creation or non-truncating opens, compare path and descriptor identities, and mutate through the descriptor. Ancestors are checked before mutation and after close; receipt publication also checks the complete target ([writes](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:591), [receipt](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:266)). Replacement regressions cover copy/open and receipt boundaries ([tests](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/candidate-ownership.test.mjs:41)).

- **Original HIGH2/HIGH3:** Post-mkdir identity remains an observation, explicitly without directory deletion authority. Unknown, replaced, or missing objects fail preflight; whole-stage cleanup always retains the stage and preserves the supplied primary cause ([creation](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:554), [preflight](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:944), [cleanup](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:1090)). An empty replacement between mkdir and observation remains indistinguishable; this implementation does not authenticate directory creation.

- **Original MEDIUM4:** Exact file/directory exclusions are selected before copying, avoiding destructive pruning of normal Base inputs. Both exclusion forms retain drift validation in the regression cases ([selection](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:637), [tests](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/candidate-ownership.test.mjs:102)).

- **Go boundary:** Commands receive alternate modfiles outside `assembled`; successful stable-read outputs enter protected target descriptors. Workspace outputs receive no cleanup authority. This constrains assembled-target writes, but does not authenticate every external Go write against concurrent workspace replacement ([implementation](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:1037)).

- **Confirmed remaining error defect:** `mergeGoDependencies` calls `result.stderr.trim()` without handling `result.error`. A read-only, in-memory invocation returning `{status:null, stderr:null, error:primary}` produced `TypeError`, losing the primary error and nested cause ([failure branch](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:521)). Normalization likewise constructs a replacement error without preserving `cause` ([normalization](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:544)).

- **Strongest ownership concern:** Individual removal checks identity, then calls pathname `unlinkSync`. Replacement after the last check can redirect deletion; subsequent ancestor checks cannot undo it. The existing test inserts an unknown sibling before preflight, not at this syscall boundary ([unlink](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:1024), [test](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/candidate-ownership.test.mjs:197)). I did not execute a filesystem deletion repro and do not waive this limitation.

## Root Cause

Directory observation, authenticated file mutation, subprocess output provenance, and pathname deletion have different guarantees. The repair largely separates them, but pathname unlink remains, and subprocess error normalization is inconsistent.

## Recommendations

1. **Small effort:** Preserve `result.error` directly and consistently preserve causes for nonzero Go exits. Add returned-error coverage alongside the existing thrown-error tests.
2. **Small effort:** Remove destructive unlink from the candidate facade; retain pre-copy exclusions and refuse unexpected present entries. This eliminates an avoidable race without adding native dependencies.
3. **Separate review if required:** Hostile concurrent namespace writers need a native lifecycle design. Additional lstat checks cannot provide atomic creation or ancestor-relative deletion.

## Trade-offs

| Option                                     | Pros                                       | Cons                                        |
| ------------------------------------------ | ------------------------------------------ | ------------------------------------------- |
| Pre-copy exclusions + retained directories | Minimal; avoids candidate cleanup deletion | Retained disk usage                         |
| Native handle-based lifecycle              | Stronger namespace guarantees              | Platform-specific implementation and review |

## Validation and handoff

Fresh read-only checks confirmed all **7 source pins**, the exact descriptor SHA256, tar inventory, original **1141/1141 snapshot** and **4634/4634 report** bytes; five scoped syntax checks passed. The **171-pass** result is retained Generator evidence, not my rerun ([evidence](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/o4-generator-code-safety-repair-20261110-01/summary.md:16)). No files were written.

Stop condition: close the two recommendations above and obtain independent CODE plus ARCH CLEAR. Then root can use this fresh attempt, currently absent:

```powershell
$o4Sealed = '.local/ops-acceptance/o4-metadata-bound-current-v2-stable-real/sealed'
node scripts/business-overlay/rebuild-from-candidate.mjs --descriptor "$o4Sealed/candidate-descriptor.json" --descriptor-sha256 2f105d76e38c82e8c301d816269ea26669a7a0291a5860be74d6ca71281a9ce6 --artifact "$o4Sealed/candidate-4315cee606c249a42c8a3372c26dee9a4658a9259229485de85953430c7f439f.tar" --attempt continue-o4-safety-reviewed-20261110-arch-01
```

This ARCH review does not replace CODE review or establish actual assembly, runtime, P0, or release completion.
