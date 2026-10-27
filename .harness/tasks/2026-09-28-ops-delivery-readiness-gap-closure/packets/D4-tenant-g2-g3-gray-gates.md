# D4: Tenant G2/G3 Gray Gates

**Priority:** P1 for the multi-tenant track; not a blocker for a single-tenant RC  
**Estimate:** 3-7 engineer-days plus DBA and maintenance-window time  
**State:** blocked; G2 and G3 are explicitly OPEN

## In

- G2 production-backup restore drill with measured RPO/RTO and application checks.
- G3 production read-only sizing snapshot and migration-window calculation.
- Production-scale rehearsal for any table over 10M rows.
- Tenant canary isolation, rollback timing, performance, observability, and audit evidence.

## Out

- Production DDL, backfill, or unique-index changes before G1-G4 are green.
- Changing the frozen Tenant Contract V1 without a contract-change task.
- Calling a local one-off backup rehearsal a production-backup proof.

## Exit Evidence

- `docs/runbooks/PRODUCTION_BACKUP_RESTORE_DRILL_G2.md` record with DBA and maintainer sign-off.
- `tenantsizing snapshot` and `tenantsizing plan` output with row-volume and window assumptions.
- Two-tenant hostile access, batch, export, cache, async, and audit results.
- Updated runbook §8.1 showing G2/G3 status and links.

## Stop Conditions

Any cross-tenant read/write, rollback failure, unmeasured RPO/RTO, or missing production-scale input keeps the track blocked.
