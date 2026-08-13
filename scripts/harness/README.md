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
node scripts/harness/check-runtime-evidence.mjs --root . --strict
node scripts/harness/check-task-packet.mjs --root .
node scripts/harness/check-evidence.mjs --root . --strict
node scripts/harness/check-failure-registry.mjs --root . --strict
node scripts/harness/check-boundaries.mjs --root .
node scripts/harness/check-visual-evidence.mjs --root .
node scripts/harness/check-feature-ledger.mjs --root . --strict
```

## Graph Review Tools

- `build-graph-review-import.mjs` - normalize CodeGraph JSON for import.
- `check-graph-review.mjs` - validate manifest structural scope, evidence `graphChecks`, and review `structuralReview` consistency.
- `scaffold-graph-review.mjs` - seed `graphChecks` and `structuralReview` from task manifest structural scope.

## Shared Utilities

- `sort-utils.mjs` - shared string sorting helper used by harness checks.
- `upstream-root.mjs` - resolves the current `pantheon-harness` method source for workspace mode.

## Pantheon Base Additions

The following scripts are repo-specific and not part of the portable method:

- `check-audit-coverage.mjs` - audit coverage validation.
- `check-backend-dto-contract.mjs` - backend DTO contract checks.
- `check-backend-response-contract.mjs` - backend response contract checks.
- `check-coverage.mjs` - unit-test coverage gate (Go cover profile / vitest coverage-summary via `--format json`).
- `check-inheritance-contract.mjs` - base-to-ops inheritance validation.
- `check-structure-contract.mjs` - file placement + naming against REPOSITORY_LAYOUT.md (complements check-boundaries import rules).
- `check-permission-contract.mjs` - permission model contract checks.
- `triage-base-drift.mjs` - base drift triage.

## Tests

```powershell
node --test tests/scripts/harness-*.test.mjs
```
