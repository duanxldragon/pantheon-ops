# Task Handoff: Capability Boundaries

Read the parent handoff before this file. This task is the dependency owner for
all callers. Do not edit BizScope, Deploy, or K8s business behavior until the
contract and package dependency plan is recorded in `STATUS.md`.

## Start

1. Run a fresh static search for the audit's direct-access evidence and record the
   current list; dirty-tree changes may already remove or add paths.
2. Draw the import graph. Decide whether contracts live in provider packages or a
   DTO-only neutral package before adding imports.
3. Freeze method signatures, error keys, scope arguments, and batch semantics.
4. Add fakes/contract tests, then migrate callers one bounded context at a time.

## Resume

Read contract diffs and scan evidence before touching code. Re-run only the failing
provider/caller contract test. Never restore a direct query as a temporary fix.

## Stop

Stop for base ownership, package cycles requiring GORM leakage, a generic query
API request, or an external REST breaking change. Record the exact alternative.
