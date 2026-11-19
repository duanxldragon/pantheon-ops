# Summary — `2026-11-16-ops-historical-runtime`

Date 2026-11-16. Owner: runtime-verification Claude (D1/D4 + old CMDB/K8s/menu conditions).
Task status: **in-progress / await-review** (Codex accepts independently). D4/G2, D4/G3:
**external-gate**, open, not self-signed.

Baseline: ops repo HEAD `chore/ops-task-closeout-2026-11-16` @ `04da3b4f`; Base release
locked `pantheon-base-v0.18.9`. Running `backend/server.exe` is a pre-built binary (Sep-29)
whose exact commit could not be traced (git execution denied in-session — honest limitation).

## 1. What was actually verified this session (real, repeatable, raw evidence saved)

All probes were **credential-free and non-destructive**, run via `node` (this headless
session auto-denies `docker`/`git`/`kubectl`/`curl`/`mysql`; I did not bypass permissions).
Raw outputs under `.harness/evidence/2026-11-16-ops-historical-runtime/raw/`.

| #   | Check                                                | Result                                                                                  | Evidence                                     |
| --- | ---------------------------------------------------- | --------------------------------------------------------------------------------------- | -------------------------------------------- |
| V1  | App health `GET http://127.0.0.1:8080/api/v1/health` | **HTTP 200**, `service=pantheon-platform`, **database/ok, migrations/ok, redis/ok**     | `raw/health_1926080110808_20261116T1037.txt` |
| V2  | MySQL `127.0.0.1:3306` TCP                           | **OPEN**                                                                                | `raw/reachability_localhost_20261116.txt`    |
| V3  | MySQL identity banner                                | **8.0.36** (== G2 required version)                                                     | `raw/probe_banner_20261116.json`             |
| V4  | Redis `127.0.0.1:6379` TCP                           | **OPEN**                                                                                | reachability raw                             |
| V5  | Redis `PING`                                         | `-NOAUTH Authentication required.` → requirepass profile (no secret exposed)            | banner raw                                   |
| V6  | Observability / k8s / argo / minio                   | **NOT listening locally** (`9090,8428,3100,3200,3000,6443,8085,9000` all ECONNREFUSED)  | reachability raw                             |
| V7  | Business overlay contract                            | `node scripts/business-overlay/check-business-overlay.mjs` → **OK (53 declared paths)** | inline in commands.json                      |

**Bottom line for D1 (local pillars):** the ops-local candidate — MySQL 8.0.36 + Redis
(requirepass) + Pantheon app on :8080 — is **live and self-consistent RIGHT NOW**. This is
reusable D1 runtime evidence for the DB/cache/app acceptance. It is **not** evidence for
the observability / K8s / ArgoCD / Velero / S3 headings, none of which are running locally.

## 2. Environment availability matrix → `ENVIRONMENT_MATRIX.md`

- Reachable local: MySQL (3306), Redis (6379), app (8080).
- Not local / external prereq: Prometheus, VictoriaMetrics, Loki, Tempo, Grafana, Kubernetes,
  ArgoCD, Velero, object store.
- Executable here: `node`, `go`, `python`. Present-but-**DENIED** (no approval surface): `docker`,
  `kubectl`, `git`, `curl`, `mysql`, ssh family. **ABSENT**: `kind`, `helm`, `redis-cli`, `psql`,
  `grafana`, `promtool`, `velero`, `argocd`, `vault`, `minio`, `rclone`, `aws`.
- The repo expects **only MySQL + Redis** in its own compose; the observability platform, K8s,
  ArgoCD, Velero and S3 are **external services configured at runtime** (DB/settings), not
  provisioned in-repo (no Helm charts, no Makefile, no live-compatible observability compose).
  This matches my probes: only the DB/cache/app are up locally.

## 3. D4 / G2 / G3 reconciliation (vs runbook + old evidence)

Read `docs/runbooks/PRODUCTION_BACKUP_RESTORE_DRILL_G2.md` and `TENANT_MIGRATION_RUNBOOK.md`;
cross-checked old gray evidence (`.harness/evidence/2026-09-10-tenant-verification-and-gray/`,
`.../2026-09-28-ops-delivery-readiness-gap-closure/`).

- **Old gray D4 evidence is NOT reusable runtime evidence**: it is a local one-off replica
  rehearsal (`tenant_rehearsal`, MySQL 8.0.36 @ 127.0.0.1, isolated Redis DB15) + two UI
  screenshots (tenant picker) + an **unfilled** G2 procedure template (`____` fields). No
  production restore, no measured RPO/RTO, no signed agreement.
- Runbook authoritative state: **G1✓ G2✗ G3✗ G4✓** — production DDL/data-change remains
  **prohibited**. The G2 drill doc's own record template is empty (lines 166–201 of the drill).
- **G2 = real production drill** (DBA-owned): production backup storage access, isolated MySQL
  8.0.36 staging host ≥ 2× prod volume, `xtrabackup`/`mysqlbinlog`, prod **read-only** binlog
  queries, measured RPO (≤15 min target) / RTO, **DBA + maintainer signature**. Requires the
  maintenance window and prod read-only conditions. I verify prerequisites only; I do not sign.
- **G3 = production volume + migration-window**: independent, needs prod table row counts +
  runbook §4.2 window calc. DBA/maintainer only.
- Weird dangling ref noted: `TENANT_MIGRATION_RUNBOOK.md:298` cites `.harness/state/.../status.md`
  as the authoritative gate record, but `.harness/state/` **does not exist** in this repo.

## 4. Old-candidate runtime-evidence reuse check

- **menu-m3c** (`2026-11-16-ops-business-menu-m3c`): source/static only — `go build`/`go test`,
  business-overlay (53), menu-contract (45/55/250/54), i18n, tsc/eslint/vite — all green, but
  **no rendered smoke / no Playwright / no e2e runtime**; `review.md:15` flags it. → **Not live
  runtime evidence**; rendered/visual coverage still a gap (C05 UI worker).
- **CMDB host import/export** (`cmdb-host-import-export`): real modern CSV implementation
  (`host_import_export.go`, `pkg/impexp`), unit+contract verified, but **DB-backed smoke
  skipped** (MySQL root creds unavailable) and no desktop/narrow visual evidence. → **Code-only;
  DB smoke + visual remain gaps**.
- **k8s-recovery**: module recovered/reassembled from a Claude transcript (`recover.py`) and the
  repo builds/tests, but **explicit "No live-cluster verification"** (compile + tests only,
  deps pinned from cache, `go test -race` not run). → **Not live runtime evidence**.
- **argo-adoption-20261110 / -v2 / -v3**: **mislabeled** — contain logs of an unrelated Python
  helper-binary integrity "RootRunnerTests" suite, a Go `packages.Load` cache error, and **empty**
  diff files; greppable ArgoCD content = **0 matches**. Real ArgoCD ownership convergence is still
  `planned` with an empty evidence dir. → **No ArgoCD evidence; do not treat as reuse.**
- **D2 SSRF** (`2026-09-29-ops-ssrf-protection-integration`): a real `validator.go` is live and
  defensive — blocks private IPs/RFC1918/169.254 metadata, DNS-rebinding, redirect disabled
  (`MaxRedirects 0`), and is wired into datasource/notification/argocd/k8s clients, with negative
  tests. **But the "allowlist" is empty in every service path** (all use `DefaultValidator()`,
  `AllowedDomains: []string{}` = allow-all-by-domain), so the deployed rule is private-network
  blocking, NOT the strict domain allowlist D2 recommended. Provenance also contradicts the
  Base-first rule (validator copied byte-for-byte from base). → partial reuse; **production
  allowlist + governance overlap must be closed by the D2 owner**.

## 5. Immediate remaining gaps — each with concrete missing item + owner

Per-subsystem commands in `commands.json` `closeGaps[]`. Summary:

| Gap              | Missing (concrete)                                                                                                       | Owner                              |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------ | ---------------------------------- |
| D1 Prom/VM       | live read query endpoint; observability owner to provide host:port                                                       | observability owner                |
| D1 Loki/Tempo    | query endpoints                                                                                                          | observability owner                |
| D1 K8s           | dev/test cluster kubeconfig + kubectl permission                                                                         | platform owner                     |
| D1 ArgoCD        | server host:port + isolated app-of-apps namespace                                                                        | platform owner                     |
| D1 Velero/S3     | isolated bucket + provider creds (available/unavailable)                                                                 | cloud owner                        |
| D2 SSRF-negative | running isolated deploy target + egress ruleset (private/DNS/redirect); also fix empty allowlist + Base-first provenance | deliverable owner + upstream(base) |
| menu-m3c         | rendered/visual smoke, desktop+narrow                                                                                    | C05 UI worker                      |
| CMDB imp/exp     | DB-backed MySQL smoke (needs MySQL root creds available/unavailable) + visual                                            | deliverable owner / C05            |
| k8s-recovery     | live-cluster + race verification                                                                                         | platform owner / C05               |
| D4/G2            | real DBA drill (prod read-only, backup storage, staging MySQL 8.0.36, RPO/RTO, signature) + maintenance window           | **DBA + maintainer**               |
| D4/G3            | prod table row counts + window calc                                                                                      | **DBA + maintainer**               |

## 6. Reuse for downstream UI / SSRF / extensions workers

- **Live localhost candidate** is reachable now: `http://127.0.0.1:8080` (health verified), MySQL
  8.0.36 @ 3306, Redis @ 6379 (requirepass). API/UI acceptance can target it; authenticated flows
  need admin creds from the repo `.env` (owner supplies available/unavailable, no echo).
- **Reusable probe**: `scripts/probe-reachability.mjs` + `scripts/probes.json` (credential-free
  TCP reachability, runnable with only `node`).

## 7. Security & honesty notes

- No credentials read/echoed; availability marked only available/unavailable.
- No production write; read-only probes only (health/banner/PING); no state changed.
- No permission bypass; denials recorded as evidence (`commands.json` sessionNote).
- No D4/G2/G3 signing; no legacy status flips; no merges; nothing fabricated.
- Runbook and old evidence quoted as-is; all reconciliation claims carry file:line refs in the
  source audit (the reconciler sub-report) which accompanies this batch.
