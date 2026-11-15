# Candidate cleanup scoped plan (L2)

Parent: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md
Evidence: .harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o4-candidate-cleanup-*
Review/handoff: resume-o4-candidate-cleanup-handoff.md

Scope: only foundation-candidate.mjs and foundation-candidate.test.mjs. Ops candidate tooling owns this correction; no Base/platform/system/business runtime or physical layer split is changed.

1. Preserve current source, all prior parent evidence, formal CLIs, assembler, candidate CLI, locks, manifests and drift reports with SHA256 before/after.
2. Run the unchanged 48-test candidate baseline. Add real extraction-write and assembly-copy failure hooks that rename/recreate stage, snapshot, nested source directory, ancestor and assembled target; record red.
3. Replace path-only ownership with directly created-directory bigint lstat dev/ino identities and ancestor identities. Preflight all recursive descendants before deletion; check ancestors and directory identity immediately before individual unlink/rmdir calls. Refuse unknown/replaced identity with a typed error and original cause.
4. Do not infer creation from a later observation. The unchanged assembler supplies no creation-identity seam, so assembler-owned directories must be retained on failure. Adjust explicit baseline cleanup expectations to this fail-closed behavior while retaining every baseline test and mutation rejection assertion.
5. Run syntax/format, complete candidate and 62 overlay baseline tests, overlay contract. Review only the final scoped diff; save exact final source hashes and handoff mapping.

Minimum complexity: Node stdlib; no dependency, marker, arbitrary target deletion or locking framework.
Windows: use bigint stats to avoid loss of 64-bit file ID precision; reject nonpositive ino or unavailable dev/ino. Local rename/recreate experiment records distinct replacement identity and stable renamed-original identity.
Known boundary: Node pathname mkdir/lstat and check/unlink/rmdir are separate syscalls. No atomic creation or hostile-concurrent namespace deletion guarantee is claimed; all observed unknown/replaced directories fail closed. Do not add recursive rm after a guard.
Stop: scoped local correction/evidence only. Independent CODE and ARCH reruns belong to the coordinating root after its repairs; no delegation, product sealing or final P0 approval in this leaf lane.
