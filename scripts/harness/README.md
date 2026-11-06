# Harness Scripts

Chinese version: [README.zh.md](./README.zh.md)

Repo-local Harness checks for `pantheon-base`.

The portable method source of truth lives in the workspace sibling `../pantheon-harness/`. This directory is the `pantheon-base` execution layer and contains both portable checker mirrors and base-specific gates.

Default upstream resolution order:

1. `--method-root <path>`
2. `PANTHEON_HARNESS_ROOT`
3. `config/method.config.json` -> `pantheonHarnessRoot`
4. `../pantheon-harness`

## Quick Validation

```powershell
node scripts/harness/check-method-health.mjs --root . --strict
node scripts/harness/check-adoption.mjs --root . --strict
node scripts/harness/check-template-health.mjs --root . --strict
node scripts/harness/check-doc-frontmatter.mjs --root . --strict
node scripts/harness/check-doc-links.mjs --root . --strict
node scripts/harness/check-doc-inventory.mjs --root . --strict
node scripts/harness/check-sync-drift.mjs --root . --strict
node scripts/harness/check-encoding.mjs --root . --strict
node scripts/harness/check-review.mjs --root . --strict
node scripts/harness/check-runtime-evidence.mjs --root . --strict   # warnings also fail --strict
node scripts/harness/check-task-packet.mjs --root .   # all task packets; pre-template docs would be listed in config/task-packet-legacy-docs.json, which is now empty
node scripts/harness/check-checker-wiring.mjs --root .   # every checker must be invoked by an npm script or a workflow, or hold a documented exception
node scripts/harness/task-evidence-state.mjs --root . --json   # evidence-based lifecycle evaluator (criteria now enforced by check-task-packet.mjs wiring, 2026-10-31)
node scripts/harness/check-evidence.mjs --root . --strict
node scripts/harness/check-failure-registry.mjs --root . --strict
node scripts/harness/check-boundaries.mjs --root .
node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base --baseline config/boundary-baseline.json   # CI form; bare --strict is stricter than the gate
node scripts/harness/check-inheritance-contract.mjs --root .. --strict   # workspace-level: --root . reports the workspace and ops docs as "missing"
node scripts/harness/check-ui-quality-gate.mjs --root . --strict
node scripts/harness/check-visual-evidence.mjs --root .
node scripts/harness/check-feature-ledger.mjs --root . --strict
node scripts/harness/check-generated.mjs --root . --strict
```

## Wired checker scripts (2026-10-31)

Wave `2026-10-31-audit-p2-ci-checker-wiring` triaged every `check-*.mjs` with the order
"does it go red right now?" before "should it be wired?", and every wired checker was
negative-verified (a fabricated violation exits 1) before the step landed. Root constraint
(FR-026): a new checker must land in the same diff with its wiring - a workflow step plus
npm script, or an explicit mirror-only / workspace-only decision recorded in this table.

| npm script | checker | wiring |
|---|---|---|
| `check:harness-evidence` | `check-evidence.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:harness-review` | `check-review.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:harness-runtime-evidence` | `check-runtime-evidence.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:harness-graph-review` | `check-graph-review.mjs --root . --strict` | `quality.yml` / Docs Governance (dormant until `structuralScope`) |
| `check:harness-audit-coverage` | `check-audit-coverage.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:harness-feature-ledger` | `check-feature-ledger.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:harness-permission-contract` | `check-permission-contract.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:backend-dto-contract` | `check-backend-dto-contract.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:backend-response-contract` | `check-backend-response-contract.mjs --root . --strict` | `quality.yml` / Docs Governance |
| `check:inheritance-contract` | `check-inheritance-contract.mjs --root .. --strict` | workspace runs only; NOT a base PR gate (`--root .` always fails in a single-repo checkout) |
| - (mirror-only) | `check-doc-frontmatter.mjs` | not wired: `scripts/frontmatter-check.mjs` is the single implementation (W5 decision; double-running the same rule is the spec-duplication class) |
| `check:checker-wiring` | `check-checker-wiring.mjs --root .` | `quality.yml` / Docs Governance (blocking since 2026-11-04) |
| `check:security-patterns` | `scripts/check-security-patterns.mjs` | `quality.yml` / Docs Governance (blocking since 2026-11-04; wired 2026-11-03 after a zero-reference audit; Node scanner since 2026-11-04, 46 legacy hits baselined in `config/security-pattern-baseline.json`) |
| `check:ui-tokens` | `scripts/check-ui-tokens.mjs` | `quality.yml` / Docs Governance (blocking since 2026-11-04; wired 2026-11-03 after a zero-reference audit; Node scanner since 2026-11-04) |

`check-task-packet.mjs` now enforces the `task-evidence-state.mjs` lifecycle predicate: a
task doc `status` that mismatches its evidence (Archived without `summary.md`, or complete
evidence with a non-Archived status) is an error, no longer a manual step.

## Checker scope (2026-11-04, FR-027)

Archiving is the normal end state of a packet, so a checker that only walks the live tree
loses the data it governs without going red. Three strict evidence checkers did exactly that
after the 2026-10-29 / 2026-10-31 archive rounds: they scanned 15 (or 0) files while 196
artifacts sat under `.harness/archive/<YYYY-MM>/`, and exited 0.

- `check-evidence.mjs`, `check-review.mjs` and `check-runtime-evidence.mjs` now discover
  `.harness/evidence/**` **and** `.harness/archive/**/*/evidence/**`. Their reported file
  counts are the observable proof of scope: a count that does not grow when the archive grows
  means the checker is blind again, whatever its exit code says.
- Generational scope, not deletion, handles age. `check-evidence.mjs` gates current-schema
  findings from `CURRENT_SCHEMA_FROM = 2026-10-26` onward and *counts, without gating*, the
  pre-cutoff structural findings (today 205 across 43 archived artifacts, printed in the text
  report). Those readings are things this repository should restate but must not invent:
  `commands[].status`, `cwd` and `agent` for a round nobody observed are unfalsifiable
  history. The counted number must shrink; `File does not exist.` and `Invalid JSON` are
  always gated, in any generation.
- Where a restatement is impossible without falsifying the record, the artifact goes into a
  named, dated allowlist instead of being edited: `config/review-loop-legacy-artifacts.json`
  (8 entries) and `config/runtime-evidence-legacy-artifacts.json` (2 entries,
  `--legacy-config`). A stale allowlist entry is an error, so exemptions cannot outlive the
  artifact they exempt.
- **New evidence must be written to `.harness/evidence/<task-id>/` and archived with its
  packet.** A round recorded anywhere else is invisible to `check-evidence`'s live-tree
  discovery, and a packet whose artifacts do not move together breaks
  `check-task-packet`'s status-versus-evidence consistency - either way the round is unchecked
  while the gate stays green.
- The registry table is scope-sensitive too: `check-failure-registry.mjs` used to stop at the
  first blank line, so a row added after one (`FR-026`) rendered as an orphan header-less
  table and was never validated. The scan now continues across blank rows and stops at prose;
  `tests/scripts/harness-check-failure-registry.test.mjs` pins both halves.

## Graph Review Tools

- `build-graph-review-import.mjs` - normalize CodeGraph JSON for import.
- `check-graph-review.mjs` - validate manifest structural scope, evidence `graphChecks`, and review `structuralReview` consistency.
- `scaffold-graph-review.mjs` - seed `graphChecks` and `structuralReview` from task manifest structural scope.

## Shared Utilities

- `sort-utils.mjs` - shared string sorting helper used by harness checks.
- `upstream-root.mjs` - resolves the current `pantheon-harness` method source for workspace mode.
- `scan-tree.mjs` - dependency-free recursive text scanner (extension filter, suffix exclusion, build-dir skips, missing-root reporting). Used by `check-security-patterns.mjs` and `check-ui-tokens.mjs` instead of ripgrep: no external binary, no CI install step, and look-ahead regexes work.
- `pattern-baseline.mjs` - fingerprint baseline helper (`config/security-pattern-baseline.json`). Accepted legacy hits are listed; anything else fails. Regenerate with `node scripts/check-security-patterns.mjs --write-baseline` after triage.

## Pantheon Base Additions

The following scripts are repo-specific and not part of the portable method:

- `check-checker-wiring.mjs` - enumerates `scripts/check-*.mjs` and `scripts/harness/check-*.mjs`, and fails when one is referenced by no npm script and no workflow (exceptions are declared in the script and must match this table).

- `check-audit-coverage.mjs` - audit coverage validation.
- `check-backend-dto-contract.mjs` - backend DTO contract checks.
- `check-backend-response-contract.mjs` - backend response contract checks.
- `check-coverage.mjs` - unit-test coverage gate (Go cover profile / vitest coverage-summary via `--format json`).
- `check-generated.mjs` - generated-artifact marker + drift gate (REPOSITORY_LAYOUT §7).
- `check-inheritance-contract.mjs` - base-to-ops inheritance validation.
- `check-structure-contract.mjs` - file placement + naming against REPOSITORY_LAYOUT.md (complements check-boundaries import rules).
- `check-ui-quality-gate.mjs` - validates the canonical UI policy, CI integration, and post-adoption UI task evidence declarations.
- `check-permission-contract.mjs` - permission model contract checks.
- `generate-pr-body.mjs` - generate a task-linked PR body for governance validation.
- `triage-base-drift.mjs` - base drift triage.
- `migrate-task-manifests.mjs` - one-shot task-manifest migrator (dry-run by default, `--write` to apply). Collected into `scripts/harness/` on 2026-11-03: it had lived in `scripts/` with no npm script and no workflow reference, and it is a maintenance tool rather than a gate.

## Tests

```powershell
node --test tests/scripts/harness-*.test.mjs
```
