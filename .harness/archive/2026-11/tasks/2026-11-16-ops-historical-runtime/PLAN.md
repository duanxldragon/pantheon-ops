# Lightweight Plan — `2026-11-16-ops-historical-runtime`

Date: 2026-11-16. Owner: runtime-verification Claude (D1/D4 + old CMDB/K8s/menu runtime conditions).

## Status

**in-progress / await-artifacts** (not self-approved; Codex independently accepts). D4/G2/G3 are
`external-gate` and stay open.

## Owned write scope (exclusive)

- `.harness/tasks/2026-11-16-ops-historical-runtime/**`
- `.harness/evidence/2026-11-16-ops-historical-runtime/**`
  Everything else read-only. No business source edits, no state flips, no merges, no signing.

## What was performed / verified (real, this session)

1. **Local candidate stack is live and healthy** — probes (credential-free, via `node`, since
   `docker`/`curl`/`mysql`/`kubectl`/`git` are auto-denied in this headless session):
   - app `http://127.0.0.1:8080/api/v1/health` → HTTP 200, `service=pantheon-platform`,
     `database/ok`, `migrations/ok`, `redis/ok`.
   - MySQL `:3306` OPEN, handshake banner **8.0.36** (== G2 required version).
   - Redis `:6379` OPEN, `PING → -NOAUTH` (requirepass profile, no secret exposed).
   - Observability (Prom 9090 / VM 8428 / Loki 3100 / Tempo 3200 / Grafana 3000), K8s (6443),
     ArgoCD (8085), MinIO (9000): **not listening locally**.
2. **Business overlay contract** `node scripts/business-overlay/check-business-overlay.mjs` → OK
   (53 declared paths).
3. Raw evidence saved under `.harness/evidence/.../raw/` (reachability, health, banners).

## Key constraint (honest, non-bypassed)

Headless session has no approval surface; tooling vetoes `docker`/`git`/`kubectl`/`curl`/`mysql`
execution. Therefore the D1 D4 live cluster/observability/backup checks and D4/G2/G3 remain
**external prerequisites**, not executed-here. Matrix + commands for a permissioned runner:
`ENVIRONMENT_MATRIX.md`, `commands.json`.

## Reuse for downstream

- Live localhost candidate (`127.0.0.1:8080`, MySQL 8.0.36, Redis) → reusable by C05/UI/SSRF workers.
- `scripts/probe-reachability.mjs` + `scripts/probes.json` = credential-free reusable probe.

## Remaining (owner-listed, see commands.json)

D1 Prom/VM, Loki/Tempo, K8s, ArgoCD, S3; D2 SSRF-negative; D4/G2 + D4/G3 (DBA/maintainer only).

## Files

- `ENVIRONMENT_MATRIX.md` (evidence)
- `summary.md` (evidence)
- `commands.json` (task)
- `scripts/probe-reachability.mjs`, `scripts/probes.json` (task, reusable)
- `task-manifest.json` (task)
