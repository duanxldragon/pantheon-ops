# Execution Queue: Ops Delivery Readiness Gap Closure

## Dependency Map

| Packet | Priority | Depends on | State | Estimate |
| --- | --- | --- | --- | --- |
| D1 External integration acceptance | P0 | endpoint and credential inventory | blocked | 2-4d |
| D2 SSRF and outbound network policy | P0 | policy decision; Base owner identified | blocked | 1-2d |
| D3 Base-first consumer re-review | P0 | v0.13.0 lock already present | ready | 0.5-1d |
| D4 Tenant G2/G3 gray gates | P1 | DBA, backup access, maintenance window | blocked | 3-7d |
| D5 Harness and generated-artifact governance | P1 | current dirty-tree decision | ready | 1-2d active; 2-5d historical |

## D1 - External Integration Acceptance (P0)

Close the production-readiness evidence gap for Prometheus/VictoriaMetrics, Loki,
Tempo, Kubernetes Service CRUD, ArgoCD, and Velero using production-like but
scoped credentials. Record health, read, write, failure, timeout, cleanup, and
permission evidence with secrets redacted. No production mutation is in scope.

Acceptance:

- each integration has a reachable endpoint and an owner;
- positive and negative API paths pass with bounded timeouts;
- created test resources are cleaned up and cleanup is evidenced;
- permission, audit, and DataScope behavior is proved;
- evidence is linked from this parent packet and reviewed by the maintainer.

## D2 - SSRF and Outbound Network Policy (P0)

Freeze the deployment policy for stored datasource URLs before proxy queries are
enabled. The policy must state allowed schemes, DNS/IP/private-network rules,
redirect handling, timeout limits, credential forwarding, and audit behavior.
The implementation owner is `pantheon-base` if the control is generic; Ops only
consumes the released contract.

Acceptance:

- policy decision is recorded and linked to the owning Base task or release;
- allow, deny, redirect, private-address, DNS-rebinding, and timeout tests pass;
- no secret or unrestricted URL is written to evidence;
- Ops overlay and build remain green after consumption.

## D3 - Base-first Consumer Re-review (P0)

Reconcile the 2026-08-31 boundary finding against the current
`pantheon-base-v0.13.0` lock. Confirm that generic middleware, common packages,
frontend tooling, and shared registries are inherited from Base and that Ops
contains only declared business overlay assets. If a shared fix is still local,
move the fix to Base and consume a new immutable release; do not patch the Ops
snapshot by hand.

Acceptance:

- lock commit, release checksum, overlay report, and clean rebuild agree;
- `npm run check:business-foundation` and the staged rebuild pass;
- no unowned shared source remains in the Ops diff;
- a findings-first review records resolved and deferred items.

## D4 - Tenant G2/G3 Gray Gates (P1, separate release track)

Complete the evidence required before any production tenant DDL or data
conversion. G1 and G4 are already granted. G2 requires a real production-backup
restore drill with measured RPO/RTO and application verification. G3 requires a
production read-only sizing snapshot and a migration-window plan; tables over
10M rows require a production-scale rehearsal. Add the canary isolation,
rollback timing, performance, and observability evidence that the gray task
currently lacks.

Acceptance:

- `TENANT_MIGRATION_RUNBOOK.md` §8.1 records G2 and G3 with signed evidence;
- failure and rollback evidence is reproducible and secrets are excluded;
- two-tenant hostile reads/writes, batch, export, cache, async, and audit paths
  have runtime proof;
- only then may a separate human gate decide whether to run migration DDL.

## D5 - Harness and Generated-artifact Governance (P1)

Close the active gate failures without rewriting unrelated history. First decide
the current `frontend/src/modules/generated/system.ts` change: restore the
canonical generated marker, or create a matching task/evidence pair and update
the generator contract. Then repair the eight active manifest warnings, the
operational-workbench visual declaration/workflow step, and the 13 docs-link and
10 docs-inventory findings. The global task-packet result reports 558 historical
errors and 95 warnings across 61 files; classify those as historical migration
debt and create bounded follow-up packets instead of mixing them into the
production feature gate.

Acceptance:

- `check:harness-adoption`, `check:ui-quality-gate`, docs links, and inventory
  are green for active delivery artifacts;
- the generated file has a declared owner and evidence;
- historical task documents are either migrated, archived with valid links, or
  explicitly tracked with a non-blocking debt decision;
- the active parent packet and evidence pass their local schema checks.

## Suggested Batches

1. D3 and the active portion of D5.
2. D1 and D2 in parallel after endpoint/policy decisions.
3. D4 only when the DBA and maintenance-window gates are available.
