# O4 Ops candidate tooling handoff

Outcome: the assigned Ops resolver, thin staging entry and hostile regression tests are implemented and verified. This is Generator/executor evidence; independent concrete CODE/ARCH tool/API review, actual product candidate creation and paired runtime remain root-owned and open.

Parent: `.harness/tasks/2026-11-07-ops-planned-task-closeout/manifest.json`. Paired parent: `../pantheon-base/.harness/tasks/2026-11-07-ops-formal-queue-foundation-gaps/manifest.json`. Approved revision4 plan SHA256: `9aaa0e53b43c040a4d04949693d3ef6ab4b086336fe953c3e04105400e4caaf7`. The Base producer handoff is `resume-o4-candidate-tooling.md/json` under the paired Base evidence parent. Exact exports, command/log digests and linkage are in [report JSON](resume-o4-ops-candidate-report.json); before/after source and protected-file hashes are in [source receipt](resume-o4-ops-candidate-source-receipt.json).

## Changes and contract

- `scripts/business-overlay/foundation-candidate.mjs`: self-contained schema1 validation and `resolveCandidateArtifact({artifactPath,descriptorPath,expectedDescriptorSha256,cacheRoot})`. Finite capability/export bindings match the Base producer. Validate provenance, detached descriptor digest, archive digest, sorted inventory, aggregate identity and required Go declaration presence. Only strict regular-file ustar is accepted; reject links/PAX/GNU/special entries, unsafe/secret/config paths, aliases, duplicate/case/file-directory collisions, malformed framing, tamper and bounds violations. Extract once into a new `.tmp/foundation-candidates/<candidateId>/snapshot`, preserve immutable descriptor semantics and compare complete file/directory stamps and hashes. `verifyCandidateExecutable` separately validates externally pinned build/executable identity; it never launches a binary.
- `scripts/business-overlay/rebuild-from-candidate.mjs`: `rebuildFromCandidate({opsRoot,artifactPath,descriptorPath,expectedDescriptorSha256,targetRoot,manifestPath})`. Canonical Ops manifest only because the unchanged assembler copies it into the product. Freeze declared business/repository inputs, implicit package/go.sum inputs, the official lock and actual Git file selection. Verify snapshot and Ops sources immediately before/after invoking `assembleConsumerTree`, call `checkSharedDrift`, byte-check only planner-declared migration destinations, and emit `.foundation-candidate-receipt.json` with staging identity, source/shared hashes, migrations, review/evidence references and `officialConsumption:false`. A supplied target must exactly equal the default new `.tmp/foundation-candidates/<candidateId>/assembled`; existing identities/targets refuse. Cleanup requires process-recorded ownership and real bounded ancestors.
- `scripts/business-overlay/foundation-candidate.test.mjs`: disposable genuine-Git provenance/dirty-source fixtures created by explicitly located Base tooling, producer/consumer semantic checks, hostile tar/descriptor/capability/build tests, real assembly/drift tests, mutation/edit-revert/Git-selection races and foreign/junction/hard-link/lock protection.

The product consumer never loads current Base source or tooling. Protocol validation is local because candidate tooling is excluded from the formal consumed Base tree; producer archive encoding/source capture are not duplicated. No dependency was added. Formal release identity, producer/fetch/resolver/rebuild/drift behavior, current lock, canonical ledgers/manifests, backend/UI/SQL and old evidence remain untouched. The existing seam modification already present on entry was preserved by identical hashes.

## CLI

```powershell
node scripts/business-overlay/rebuild-from-candidate.mjs --help
node scripts/business-overlay/rebuild-from-candidate.mjs --descriptor "../pantheon-base/.tmp/foundation-candidates/<reviewed-attempt>/candidate-descriptor.json" --descriptor-sha256 "<independently-reviewed-64-hex-descriptor-digest>" --artifact "../pantheon-base/.tmp/foundation-candidates/<reviewed-attempt>/candidate-<aggregateSha256>.tar"
```

Replace quoted placeholders with root-reviewed values. The second line is syntax only; no actual product candidate was sealed while Base API/generator/source work is in flight. `--ops-root` may identify the Ops repository; optional `--target` accepts only the exact default candidate assembled path. `--base`, alternate manifest ownership, missing pins, unknown/duplicate options and existing targets refuse. No package script alias or formal lock replacement is needed.

## Fresh verification

- Final `node --test scripts/business-overlay/foundation-candidate.test.mjs`: **48 passed, 0 failed/skipped**, 74508.8491 ms. [Final output](resume-o4-ops-candidate-tests-verified-final.txt).
- Existing `npm run test:business-overlay`: **62 passed, 0 failed/skipped**, 16160.9946 ms. [Existing overlay output](resume-o4-ops-candidate-existing-overlay-tests.txt).
- Current-source supplemental ownership/mutation tests: 18 passed, 0 failed/skipped. [Output](resume-o4-ops-candidate-tests-ownership-final.txt).
- `node --check` for all three files, Prettier check, `check:business-overlay` (56 declared paths), docs frontmatter and task-packet template: passed.
- Tracked diff check passed; each untracked file's `git diff --no-index --check -- NUL <file>` emitted no whitespace diagnostics. Its raw exit1 means a new-file diff, and is recorded without treating it as a test failure.
- Exact tested source hashes matched after verification. All protected tool/lock/manifest/root-report/qualified old drift-report hashes matched their before values. Real lock SHA256 stays `95de1a87270e15c0d4c672418a5df906818a88652f8a3e6c480f1484ee9fa70c`.

[Initial failures](resume-o4-ops-candidate-tests-initial.txt) are preserved: 44 passes, with the migration-conflict fixture and parent reported failed because the fixture omitted the required down pair. The corrected fixture now exercises the genuine Base-version conflict. No existing failure/report was overwritten or removed. Fixture directories are removed by test teardown; no debug source or current product candidate remains.

## Findings-first self-review and remaining acceptance

No unresolved finding in the assigned tooling scope after fixing the missing fixture pair, freezing Git selection/capability mappings and binding cleanup to actual created-stage ownership. The archive/descriptor establishes local integrity and reviewed interface identity; source declaration presence and synthetic build receipts do not prove API behavior, executable correctness or independent approval. The Base provenance commit is never relabeled as a formal release identity. Assembled source inventory excludes only the receipt subsequently added to that assembly.

CodeGraph tools were unavailable in this leaf executor; focused source reads covered the exact resolver/assembly/drift call chain. No sub-agent or independent review lane was spawned. Root must review the concrete Base APIs and both tools, provide the complete reviewed product source/build/docs inventory, create the actual stable candidate after source writers finish, and collect separate executable/build, schema/migration, supported-start/old-stop, generator and paired runtime evidence. Final official immutable release and formal Ops consumption remain open. No commit/push/release was performed.

Gate Outcomes: candidate integrity/extraction caught tamper, false capabilities, unsafe tar and source instability | stage ownership caught destructive/foreign targets and unowned cleanup | inheritance caught drift/migration conflicts | none false-positive.
