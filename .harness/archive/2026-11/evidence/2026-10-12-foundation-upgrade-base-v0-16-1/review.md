# Review — foundation upgrade to pantheon-base-v0.16.1

## Boundary

The change stays on the consumer side of the inheritance boundary. Base is consumed only through
the immutable GitHub release `pantheon-base-v0.16.1`; no base file was hand-copied, and no release
or tag was created or modified.

## What was reviewed

- **Lock shape.** The root lock keeps the existing schema and only moves version, release line,
  commit, checksums and materialization path. `sharedPaths` was proven identical across the lock,
  the previous manifest and the new manifest before it was left untouched.
- **Artifact integrity.** Both digests were verified twice: against the published `.sha256`
  sidecars and against GitHub's own asset digests. The `repo.tar` extraction was additionally
  compared file-by-file with the tag tree; the two differences are `.gitattributes` `eol=crlf`
  files, which is expected for a `git-archive` snapshot.
- **Overlay assembly.** All six assembly anchors resolve exactly once on the new base, the 53
  declared overlay paths resolve, and the 10 allowed drift entries are the known extension hooks —
  no stale allowlist entries and no undeclared drift.
- **Consumer tree.** The swap is 381 files (~16.4k insertions, ~10.1k deletions). The 23 deletions
  are 13 root-level completion/report artifacts that were never declared product files, the
  release-push scripts, templates base dropped and openspec placeholders. No business module, page,
  permission, menu, i18n or migration asset changed.
- **Verification depth.** Backend build, vet and tests were run on the rebuilt module graph, not
  only on the pre-swap tree; the frontend was reinstalled from the swapped lock and type-checked.

## Risks and decisions

- The baseline swap is the L2 step in `docs/PROJECT_INHERITANCE.md`. It was executed only after the
  staged tree passed the contract, drift, backend and frontend gates, which is the gate condition
  the document sets.
- Base main is 13 commits ahead of the release tag. The drift gate compares against a base
  checkout, so it will stay red for those commits until base publishes a release that contains
  them. Allowlisting them would hide real shared-file drift and was rejected.
- `.foundation` had to become a declared repository overlay. Without it a rebuild cannot carry
  `allowed-drift.json`, which is why an earlier staged drift run reported the 10 known hooks as
  unexpected drift.
- `docs/harness/tasks` had to be pruned from the base snapshot. The directory is ops-owned, but the
  rebuild only overlays files, so base's release task packets survived the copy, then survived the
  baseline swap, and failed the task-packet gate on files ops does not own.
- The local cgo failure is environmental (a trivial cgo program also fails), so it is recorded as
  a limitation rather than worked around with build-tag or test changes.

## Verdict

Accept for the consumer side. The remaining gaps (base release cut for the 13 post-tag commits,
push and hosted workflow run) are explicit and outside this repository's authority.
