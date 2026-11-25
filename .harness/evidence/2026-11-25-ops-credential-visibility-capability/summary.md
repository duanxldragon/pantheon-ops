# Summary: credential visibility through the capability contract

## Problem

After T05, `deploy_credential.go` built its visibility subquery from `bizscope.BizScope`, a sibling module's model. `check:business-boundaries` flags that, but it is manual-only in CI, so the required checks stayed green and the violation merged.

## Change

`BizScopeReader` gained `VisibleIDs(ctx, scope)`, implemented in bizscope with the same `WithDataScope` it used before. Deploy now asks the capability for the visible scope ids and filters `business_scope_id IN ids`. It fails closed: an error or a missing reader denies, and an empty list matches nothing.

## Verified

- Boundary check: red before, green after.
- T05's four credential tests pass on a real MySQL with unchanged results.
- A negative control (VisibleIDs returning nothing) makes the list test fail.

## Gaps

- No independent CODE/ARCH review.
- The boundary check is still manual-only in CI; wiring it in is a gate-policy decision for the maintainer.
- A subquery became an id list; with very many business scopes the `IN` list grows. Acceptable at current scale.
