Target repository: pantheon-ops
Layer: inheritance-sync (pantheon-base -> pantheon-ops)
Delivery tier: L2
Mode: formal release consumption, generator route remediation, verification, and debt assessment

Source release:
- releaseVersion: pantheon-base-v0.10.3
- releaseLine: release/0.10
- baseCommit: c8c5832d85f33f2f245526f4d0eab9df93225880
- GitHub release archive SHA-256: c0d3981c315cd323234e1272cc2cd473cbf114c213938abb4b4ff268a94e1239

In scope:
- Install and verify the immutable Base v0.10.3 release archive.
- Run the release consumer dry-run and rollback-protected apply.
- Restore generated business page routes, inferred menus, and activation summaries to `/business/*`.
- Preserve manually designed Ops navigation that intentionally remains under `/operations/*`.
- Preserve every `business/*` overlay and validate inherited backend/frontend contracts.
- Repair consumer compatibility defects exposed by the immutable artifact: stale lowcode API policy paths, excluded operation-token caching, canonical generated i18n output, empty registries, and safe purge ordering.
- Measure the remaining v0.10.3 consumption work and historical branch/PR debt.
- Record verification evidence, self-review, known gaps, and a decision-ready effort estimate.

Out of scope:
- Editing or retagging the immutable Base release.
- Manual copies from the Base worktree.
- Migrating manually designed CMDB, Deploy, or Bizscope routes from `/operations/*`.
- Merging or deleting remote branches and PRs without a separate final human gate.
- New UI design, database changes, or API prefix changes.

Landing-side decision:
- Base owns the shared generator fix and immutable release artifact.
- Ops owns formal artifact consumption, the lock, consumer adapter, inheritance evidence, and business overlays.
- v0.10.3 remains immutable. Consumer compatibility changes in shared paths are explicit overlays only when the release manifest omitted the required Base fix or runtime validation exposed a post-release defect; generic fixes must return to a later Base release.

Assumptions and open questions:
- The existing lock edit is the intended start of the v0.10.3 upgrade and must be preserved.
- The current branch's 21 local commits remain in scope as the integration branch; they must not be rewritten.
- Hosted checks and merge actions remain external gaps until a PR head is pushed.

Minimum viable approach:
- Reuse `foundation:install` and `upgrade:foundation:apply` with their existing checksum, clean-worktree, overlay, and rollback guards.
- Add no new dependency or packaging abstraction.
- Repair only consumer-specific breakage exposed by the exact v0.10.3 artifact.

Success criteria:
- Lock, installed marker, manifest, tag, and archive identify the exact v0.10.3 release.
- Generated business routes use `/business/*`; manually designed Ops routes remain `/operations/*`.
- Apply preserves Ops-owned business overlays.
- Dynamic generation and purge preserve handwritten business modules, emit canonical registries/i18n, and do not expose transient imports to file watchers.
- Inheritance, base-sync, generator, Go, frontend, and business-runtime checks pass or have an explicit evidence-backed gap.
- Open PRs and remote/local branches are inventoried, grouped by action, and estimated.

Structural scope:
- Affected subgraph: Base release manifest -> installer -> consumer -> shared lowcode generator -> generated business routes and menus.
- Boundary crossings: Base foundation release -> Ops shared paths -> Ops business overlay.
- Risk nodes: incomplete lock-only upgrade, `/operations/*` generator drift, business overlay replacement, stale PR bases.
- Graph focus: release identity, generated route flow, overlay preservation, merge ancestry.

Implementer posture:
- Apply the immutable release through repository tooling and make only consumer-owned compatibility changes.

Reviewer posture:
- Findings-first self-review because no independent agent was requested; verify release identity, route behavior, overlay preservation, and evidence completeness.

Stop points:
- Release checksum or commit mismatch.
- Consumer proposes deleting or replacing Ops-owned `business/*` files.
- Rollback protection is unavailable.
- Remote merge, branch deletion, or production action requires a separate human gate.

Verification plan:
- `npm run foundation:install`
- `npm run upgrade:foundation:plan`
- `npm run upgrade:foundation:apply`
- `npm run check:inheritance`
- `npm run check:base-sync`
- `npm run test:foundation-release`
- `go test -race ./...`
- `npm --prefix frontend run lint`
- `npm --prefix frontend run type-check`
- `npm --prefix frontend run build`
- `npm --prefix frontend run test:generator:smoke`
- `npm --prefix frontend run test:smoke:business`

Evidence:
- Task manifest: `.harness/tasks/2026-08-07-foundation-upgrade-base-v0-10-3/manifest.json`
- Evidence directory: `.harness/evidence/2026-08-07-foundation-upgrade-base-v0-10-3/`
- Review: `.harness/evidence/2026-08-07-foundation-upgrade-base-v0-10-3/review.md`

Human gate:
- Satisfied for local implementation and assessment by the maintainer request on 2026-08-07.
- Still required before remote merge, PR closure, or branch deletion.
