# Ops consumption of official Base v0.18.7

Ops consumes the immutable official foundation release `pantheon-base-v0.18.7` (tag -> commit `3203624e55438866edb951bd74e671eab38d2f16`) through the standard assembler only: GitHub four-asset acquisition with sidecar/hosted-digest verification, exact `foundation-release.lock.json` update, isolated staging rebuild, and full validation of the staged tree before materialization into the consumer PR.

## Official acquisition

- `npm run fetch:base-release -- --release pantheon-base-v0.18.7` locked `duanxldragon/pantheon-base@pantheon-base-v0.18.7` at `3203624e...` with bundle checksum `1daa3e7f2eb7c243d51e417aa651be2feeb045dfc875ac3556038c2df74e380a`.
- A first acquisition attempt produced a truncated cached tgz because the local disks were full (the same root cause as the Go linker "disk full" failure). The partial cache was purged, both archives were re-downloaded with authenticated requests, and `sha256sum -c` passed against the published sidecars; hosted asset digests from the GitHub API matched exactly (`1daa3e7f...` tgz, `6149f9e9...` repo.tar). No gate or assembler was modified; only official assets entered the tree.

## Staging gates (exact tree)

- `npm run rebuild:from-base -- --target .tmp/clean-base-overlay`: OK, base commit 3203624e, 954 declared/generated files.
- `check-business-overlay --root .tmp/clean-base-overlay`: OK, 53 declared paths.
- `check:shared-drift --root .tmp/clean-base-overlay`: OK, 866 shared files compared, 856 identical to base, 10 allowlisted drift, 164 business-overlay files skipped, 2 derived business migrations verified. (Re-run identically on the materialized tree; the synced checker script itself is outside the shared-file manifest.)

## Full validation on the staged tree (completed before materialization)

- Readonly CGO full race: 67 packages ok, 0 failures, 0 data races (`go test -timeout=45m -buildvcs=false -mod=readonly -race -p 2 -count=1 ./...`, rerun after the authorized disk-cache purge).
- Frontend lint/type-check/build/unit: 240 unit tests across 35 files passed.
- `check:business-foundation` incl. 59 business-overlay tests and tree-level drift: passed.
- Task-owned fresh runtime: backend startup on isolated `pantheon_ops187` database with Redis 26317; fresh startup migrations reached `version 32, dirty 0`.
- Four standard generator smoke suites (governance, auto-recycle, master-detail, many-to-many) against that runtime: all exit 0.
- The validated staging tree was then materialized into the consumer tree (junction-safe copy; no post-assembly source edits) and re-verified at tree level.

## Governance gates on the materialized tree

- Every consumption-scope check passes: docs-frontmatter, encoding, structure, inventory, sync, method, template, task-packet for both Nov packets, audit-coverage, feature-ledger, permission-contract, inheritance-contract, ui-tokens, ui-quality-gate, duplication, failure-registry, backend DTO/response contracts, generated-modules, business-boundaries, business-foundation, gorm-sql-literal, and the evidence/review artifacts of this task.
- `scripts/frontmatter-check.mjs` was synced to the immutable v0.18.7 snapshot copy: the v0.18.7 docs are prettier-formatted (blank line between a relation heading and its list) and the already-official-synced `tests/docs/frontmatter-check.test.mjs` asserts that tolerance; without the sync the CI Docs Governance frontmatter gate failed. The checker test suite and the gate now pass.
- Nine local gates still exit 1 exclusively on pre-existing legacy findings that are identical on origin/main (stale archive allowlist entries, legacy task packets/evidence, base task-local links inside inherited TENANT/LOWCODE docs, legacy unwired checkers, baseline business security-pattern and generated-marker findings). None references this task; none runs in the hosted required checks. The base-doc-link finding class is routed back to Base per the inheritance boundary.

## Pending (recorded as they actually complete)

- Independent CODE/ARCH review of the assembled delta (review.md currently records `blocked`).
- Consumer PR with required hosted checks, merge, main-only consolidation.

No consumption is claimed until the exact assembled source passes every sensor above and the consumer PR merges with required checks green.

## Explicit residuals

Real S3 provider skip unwaived; maintenance freeze unsigned; Base FR028 oversized audit storage deferred; separate Ops CMDB/O2/O3/O4/D2 product-acceptance and QA002/host-import runtime fixture gaps remain. Rollback anchor: previous v0.18.6 lock (`5553bc0e`) remains in git history.
