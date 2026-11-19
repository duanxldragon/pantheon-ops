# ENVIRONMENT_MATRIX — pantheon-ops historical runtime (D1/D4) 2026-11-16

> Prepared by the runtime-verification Claude for the historical-closeout batch.
> This matrix records **what actually is / is not reachable and executable**, not what
> a runbook claims. It is the ground truth the D1/D4 acceptance and C05/E04 candidates
> build on. Evaluated: 2026-11-16 10:37–10:40 (Asia/Shanghai), host = this machine.
> Repository baseline at evaluation: branch `chore/ops-task-closeout-2026-11-16`, latest
> commit `04da3b4f`. The running `backend/server.exe` is a prebuilt binary (file date
> Sep-29); its exact commit could NOT be traced in-session (`git` execution denied — see §5).

## 1. Legend / method

| Status      | Meaning                                                                                                                                                    |
| ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `OPEN`      | TCP handshake succeeded on the port (reachability fact only — NOT service-health proof)                                                                    |
| `CLOSED`    | TCP connect refused / no listener                                                                                                                          |
| `NOT-LOCAL` | Not listening on localhost; candidate endpoint lives in an external/prod/ci env not reachable here                                                         |
| `DENIED`    | The binary/tool is present on PATH but this session could not execute it (permission system blocks it; no interactive approval surface — headless session) |
| `ABSENT`    | Binary not found on PATH in this session                                                                                                                   |
| `EXT-GATE`  | Requires an external gate owned by another role (DBA / maintainer / release), not executable or signable by me                                             |

Probing method: credential-free, non-destructive `net.connect` TCP probes and a read-only
HTTP `GET /api/v1/health` via `node` (NodeJS is permitted in this session). No service
credentials were transmitted, read, or echoed.

## 2. Runtime availability matrix (evaluated this session)

| #   | Subsystem       | Candidate endpoint (repo-documented)                          | Local reachability (this host)                               | Version / identity evidence                                                                                                   | Executable here?                                                |
| --- | --------------- | ------------------------------------------------------------- | ------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| R1  | MySQL           | `127.0.0.1:3306` (ops root `docker-compose.yml`; `mysql:8.0`) | **OPEN**                                                     | Handshake banner reports **8.0.36** (matches G2 required 8.0.36)                                                              | client CLI `mysql` **DENIED**; probes via node only             |
| R2  | Redis           | `127.0.0.1:6379` (compose `redis:7-alpine`, `--requirepass`)  | **OPEN**                                                     | `PING → -NOAUTH Authentication required.` → confirmed Redis with `requirepass` set (no secret exposed)                        | `redis-cli` **ABSENT**; probes via node only                    |
| R3  | Pantheon app    | `127.0.0.1:8080` (`.env.example` `PANTHEON_PORT=8080`)        | **OPEN**                                                     | HTTP `GET /api/v1/health` → **200**, `service=pantheon-platform`, `database/ok, migrations/ok, redis/ok` (raw evidence saved) | n/a (verified via HTTP)                                         |
| R4  | Prometheus      | `9090`                                                        | `CLOSED` (ECONNREFUSED)                                      | none                                                                                                                          | tool **ABSENT** locally                                         |
| R5  | VictoriaMetrics | `8428`                                                        | `CLOSED`                                                     | none                                                                                                                          | tool **ABSENT** locally                                         |
| R6  | Loki            | `3100`                                                        | `CLOSED`                                                     | none                                                                                                                          | tool **ABSENT** locally                                         |
| R7  | Tempo           | `3200`                                                        | `CLOSED`                                                     | none                                                                                                                          | tool **ABSENT** locally                                         |
| R8  | Grafana         | `3000`                                                        | `CLOSED`                                                     | none                                                                                                                          | tool **ABSENT** locally                                         |
| R9  | Kubernetes API  | `6443`                                                        | `CLOSED` (local) → **NOT-LOCAL**                             | n/a                                                                                                                           | `kubectl` present at `C:\Users\xiaolong\kubectl` but **DENIED** |
| R10 | ArgoCD          | `8085`                                                        | `CLOSED` → **NOT-LOCAL**                                     | None                                                                                                                          | `argocd` **ABSENT**                                             |
| R11 | Velero          | external (k8s pod/cluster)                                    | **NOT-LOCAL**                                                | None                                                                                                                          | `velero` **ABSENT**                                             |
| R12 | S3 / MinIO      | `9000`                                                        | `CLOSED` → **NOT-LOCAL**; real S3 is prod/cloud **EXT-GATE** | None                                                                                                                          | `aws`, `mc` **ABSENT**                                          |

Key facts:

- The ops-local candidate stack (**MySQL 8.0.36 + Redis + Pantheon app**) is **live and
  self-consistent right now**: app health reports DB/migrations/Redis all `ok`. This is
  genuinely reusable D1 runtime evidence for the DB/cache/app pillars (raw evidence in
  `.harness/evidence/2026-11-16-ops-historical-runtime/raw/`).
- The observability stack (Prometheus/VM/Loki/Tempo/Grafana), Kubernetes, ArgoCD, Velero
  and any object store are **not running on this host** and are **not present as local
  tooling**. They are external-candidate/production prerequisites (see §4).

## 3. Subsystem tool executability in this session

| Binary                                                                                                            | On PATH?                          | Executes in this session?   | Note                                                               |
| ----------------------------------------------------------------------------------------------------------------- | --------------------------------- | --------------------------- | ------------------------------------------------------------------ |
| `node` (v24.14.0)                                                                                                 | yes                               | ✅                          | Used for all live probes                                           |
| `go` (1.26.5)                                                                                                     | yes                               | ✅                          | Allowed                                                            |
| `python` (3.12.8)                                                                                                 | yes                               | ✅                          | Allowed                                                            |
| `docker`                                                                                                          | yes (Docker Desktop path)         | ❌ **DENIED**               | cannot run containers (`docker compose up` etc. not possible here) |
| `kubectl`                                                                                                         | yes (`C:\Users\xiaolong\kubectl`) | ❌ **DENIED**               | cannot run cluster commands                                        |
| `git`                                                                                                             | yes                               | ❌ **DENIED**               | cannot read/run git                                                |
| `curl`                                                                                                            | yes                               | ❌ **DENIED**               | HTTP probes done via `node` instead                                |
| `mysql`                                                                                                           | yes (`D:\MySQL\mysql\bin`)        | ❌ **DENIED**               | DB probes done via `node` banner/handshake (no auth)               |
| `ssh`/`plink`                                                                                                     | yes (PuTTY on PATH)               | ❌ **DENIED** (not invoked) | SSH target verification not executed                               |
| `kind`, `helm`, `redis-cli`, `psql`, `grafana`, `promtool`, `velero`, `argocd`, `vault`, `minio`, `rclone`, `aws` | no                                | n/a                         | **ABSENT**                                                         |

Why: this is a headless session with **no interactive approval surface**. The harness
permission layer auto-denies any command not on the default safe-list (`node`, `go`,
`python`, `ls`/`cat` in the working dir). `docker`, `git`, `kubectl`, `curl`, `mysql`
were auto-denied on attempt. I did not and will not bypass permissions. THIS is the
primary reason most live subsystem checks are recorded as prerequisites, not failures.

## 4. True external prerequisites by gate (who can unblock)

| Gate / candidate                 | Missing                                         | Required condition                                                                                                                                                                                                                                                                                            | Owner                            | Status                                   |
| -------------------------------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- | ---------------------------------------- |
| D1 Prometheus/VM                 | no endpoint reached locally                     | Provide VM/Prom read-query endpoint (host:port) + read credentials (mark available/unavailable, don't echo)                                                                                                                                                                                                   | observability owner / maintainer | `EXT-GATE` open                          |
| D1 Loki/Tempo                    | no endpoint reached locally                     | Provide Loki/Tempo query endpoints; or run via permissioned session with docker                                                                                                                                                                                                                               | observability owner              | `EXT-GATE` open                          |
| D1 Kubernetes                    | no local cluster; kubectl DENIED                | Provide kubeconfig/context for an isolated dev/test cluster; grant kubectl permission                                                                                                                                                                                                                         | platform owner                   | `EXT-GATE` open                          |
| D1 ArgoCD                        | absent locally                                  | Provide ArgoCD server host:port + isolated app-of-apps namespace                                                                                                                                                                                                                                              | platform owner                   | `EXT-GATE` open                          |
| D1 Velero / D4/G2 backup-restore | absent + prod-track                             | **Real G2 = DBA drill**: needs production backup storage access, isolated MySQL 8.0.36 staging host (≥2× prod volume), `xtrabackup`/`mysqlbinlog`, prod _read-only_ DDL-free binlog queries, measured RPO/RTO numbers, DBA + maintainer signature. See `docs/runbooks/PRODUCTION_BACKUP_RESTORE_DRILL_G2.md`. | **DBA** + maintener              | `EXT-GATE` open — **NOT signable by me** |
| D1 S3                            | real provider is prod/cloud                     | Provide an isolated bucket + provider creds (available/unavailable) in a test region                                                                                                                                                                                                                          | maintainer / cloud owner         | `EXT-GATE` open                          |
| D4/G3 production volume          | prod data + window                              | Independent of G2; needs prod table row counts + §4.2 window calc                                                                                                                                                                                                                                             | DBA + maintener                  | `EXT-GATE` open                          |
| D2 SSRF negative acceptance      | needs a running deploy target + outbound policy | Provide isolated dev/test target and egress ruleset; negative scenarios                                                                                                                                                                                                                                       | deliverable owner                | `EXT-GATE` open                          |

## 5. Honest limitations recorded (no fabrication)

1. **I cannot run containers, clusters, git, curl, mysql-cli, or ssh in this session**
   (permission auto-deny, no approval surface). Attempted `docker`, `kubectl`, `git`,
   `curl`, `mysql` — all **DENIED**. Recorded as environmental evidence, not bypassed.
2. **Reachability ≠ health.** `OPEN` on 3306/6379/8080 is a TCP handshake; only the app
   health (R3) proves actual service health (`database/ok` etc. — a real read via HTTP).
   No authenticated DB write, no migration-to-prod, no backup-restore was executed.
3. **The running binary's exact commit is unknown.** Repo HEAD is `04da3b4f` but the live
   `server.exe` (built Sep-29) commit could not be traced because `git` is DENIED here.
4. **D4/G2 and D4/G3 are production external gates.** They require DBA, production
   read-only access, backup storage, isolated staging MySQL 8.0.36, measured RPO/RTO, and
   a maintainer signature. No local stub/mock substitutes for these. I record prerequisites
   and keep them open; I do not sign.

## 6. Reusable for downstream UI/SSRF/extensions workers

- A **live, isolated-on-localhost candidate stack** (MySQL 8.0.36 + Redis + app on :8080)
  is reachable now — API workers and the C05/UI acceptance can hit `127.0.0.1:8080`
  (read-only endpoints verified; authenticated flows need admin creds from the repo `.env`,
  provided by maintainer as available/unavailable without echoing values).
- Reusable probe artifact: `.harness/tasks/2026-11-16-ops-historical-runtime/scripts/probe-reachability.mjs`
  - `probes.json` (credential-free TCP reachability; runnable with only `node`).
- A permissioned runner can bring up the full op stack from the ops root
  `docker-compose.yml` (MySQL+Redis) and the base candidate compose for observability
  (`.foundation/releases/pantheon-base-v0.18.9/repo/docker-compose.yml`), then re-run the
  matrix for the missing headings (R4–R12). Exact commands in `commands.json`.
