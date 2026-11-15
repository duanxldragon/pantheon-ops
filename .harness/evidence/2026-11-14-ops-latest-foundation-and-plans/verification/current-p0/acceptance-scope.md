# Current P0 acceptance scope

Parent task: `2026-11-14-ops-latest-foundation-and-plans`.

This worker owns fresh acceptance helpers and the native datasource-envelope oracle. It does not close any O2/O3/O4/D1/D2/D3 parent by itself.

## Verified

- Fresh SSH read-only preflight: `current-p0-20261114T112123-fe9f5deb-remote-readonly.json`, exit 0, zero remote mutations.
- Native wire-envelope regression RED: `current-p0-20261114T113241-fb65fcf6-envelope-oracle.json`, exit 1, the actual `TestDatasourceLiveEnvelopeMatchesNativeSuccess` assertion failed with the old code-0 comparison. All 842 backend source hashes were stable; no skips.
- Native wire-envelope regression GREEN: `current-p0-20261114T113258-2cc6415a-envelope-oracle.json`, exit 0, 1 test and 1 package passed after restoring `common.CodeSuccess`. All 842 backend source hashes were stable; no skips.
- `node --check scripts/testing/current-p0-public.mjs`: exit 0.
- `python -m py_compile scripts/testing/current-p0-runtime.py`: exit 0.

Exact native command: `python scripts/testing/current-p0-runtime.py envelope-oracle --timeout 300`, using the configured Go cache, `CGO_ENABLED=1`, the MSYS compiler, `-race -p=1 -count=1 -mod=readonly`.

## Prepared, not executed

`current-p0-runtime.py public-start` requires a task-built executable under Ops `.tmp` and a complete source manifest matching the current backend Go/SQL/module tree. It creates an exclusive local synthetic schema, API port and Redis instance. Formal migrations are required, automatic model migration is disabled, and startup/source identity is checked. Private credentials are never included in public evidence.

`current-p0-public.mjs` reserves one attempt per new runtime. It validates the binary, complete source map and startup receipt before public API requests. Its scope is O4 native Prometheus query, Base authentication/current IAM/tenant/data scope composition, saved-view/history ownership, revision conflict, metadata-only audit, and CMDB permission-key separation. Mutations use only public product APIs against the owned local schema. The remote provider endpoint is read-only. A transport Unknown is retained without retry or successor cleanup.

## Preserved failures and limits

- Initial build and SQL race attempts were interrupted by disk pressure while product source was still changing; they do not count as accepted build/test evidence. Their receipts and private logs remain intact.
- The earlier code-0 oracle attempt at `112703` failed during cache setup and does not establish assertion RED evidence. The later `113241` receipt does.
- Disposable build cache/scratch cleanup released 1,019,242,040 bytes; `disposable-build-cleanup.json` records the exact owned targets.
- Current public HTTP execution waits for root's product freeze and entry review.
- No historical one-use runner or receipt was replayed. The older API on port 18127 remains untouched.
- No rendered browser evidence, synthetic Loki/Tempo/VictoriaMetrics provider acceptance, O2/O3/D1/D2/D3 closure, or production deployment is claimed here.
