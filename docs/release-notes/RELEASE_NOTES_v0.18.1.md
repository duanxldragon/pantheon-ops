# Pantheon Base v0.18.1 — Release Notes

Patch release on the `release/0.18` line. Supersedes `pantheon-base-v0.18.0` as tag
`pantheon-base-v0.18.1`.

v0.18.0 shipped `frontend/src/api/request.ts` without `isForbiddenRequestError`. Rebuilding
`pantheon-ops` against that release failed in its own business code, so this patch restores the export
and nothing else.

## Summary

- **No schema change.** The migration chain still ends at `000030_tenant_plan_definitions`
  (schema 30).
- **No route, permission code, menu key or i18n key change.** No backend change.
- `sharedPaths` stays at 52 entries; a consumer only moves the version, the release line, the commit
  and the two checksums.
- The public surface gains two lines: one predicate restored to the request-error family.

## What changed

### Fixed — the shared request-error predicate

`frontend/src/api/request.ts` exports a four-member family that consumer business code branches on:

| Predicate | Error kind |
| --- | --- |
| `isTimeoutRequestError` | `timeout` |
| `isNetworkRequestError` | `network` / `timeout` |
| `isForbiddenRequestError` | `forbidden` |
| `isServerRequestError` | `server` |

The S1 unexport wave (#439, `b0a1952a`) removed `isForbiddenRequestError` on a "no caller" reading.
That reading covered this repository only: `pantheon-ops`
`frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:29` imports the predicate and uses it
at line 182 to distinguish a permission rejection from other request failures. The first rebuild
against v0.18.0 stopped with:

```
src/modules/business/deploy/task/DeployTaskDetail.tsx(29,10): error TS2614:
Module '"../../../../api/request"' has no exported member 'isForbiddenRequestError'.
```

The patch restores the two-line export. `isForbiddenRequestError` was the only member of the family
missing; the other three, and every other `api/request` symbol the consumer imports, are unchanged.

## Upgrade notes

- There is no migration and no configuration change. Update the lock to `pantheon-base-v0.18.1`
  (`release/0.18`), rebuild the tree and re-run the builds.
- If you consumed v0.18.0 and your own code imports `isForbiddenRequestError` from
  `api/request`, v0.18.1 is the build fix. If it does not, the patch is a no-op for you.

## Consumer impact

- `pantheon-ops` must move from `pantheon-base-v0.18.0` to `pantheon-base-v0.18.1`; the base commit
  changes, so both checksums change too.
- The frontend build is the acceptance reading for this patch: `npm run build` in the rebuilt
  consumer tree must pass, and the backend `go build ./...` stays green.
- Governance: `FR-021` registers the miss in the failure registry, and
  `docs/harness/AI_QUALITY_GOVERNANCE.md` §5.4 adds the rule that removing a shared export requires a
  workspace-wide consumer reference search and that a foundation release is not verified until the
  consumer builds against it.

## Verification

- Consumer staging tree against v0.18.0: `npm run build` → TS2614 (failure).
- Same tree with the export restored: `npm run build` → `✓ built in 751ms`.
- Base frontend after the restore: `npm run lint`, `npm run type-check`, `npm run build` green;
  backend untouched, so no backend re-run beyond the release gate's own checks.
- Release assets and `Foundation Release Verify` follow the same four-asset contract as v0.18.0.
