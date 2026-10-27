# Ops Delivery Readiness Gap Closure

## Verdict

`pantheon-ops` is **repository-local RC-ready** but **not approved for direct production delivery**. The local backend, frontend, business overlay, and race gates are green. The remaining release blockers are external integration evidence, an explicit outbound SSRF policy, and a current Base-first ownership review.

The multi-tenant migration track is separate and remains blocked: the tenant runbook records G1 and G4 as `GRANTED`, while G2 (production backup RPO/RTO and restore) and G3 (production-scale sizing and migration window) are `OPEN`.

## What Is Complete

- Backend tests, vet, and Windows CGO race gate pass.
- Frontend type-check, lint, build, menu/i18n/UI contract checks pass through the build.
- Business boundary, overlay, and overlay tests pass.
- Harness method, sync, encoding, and structure checks pass.
- The active parent task packet and Harness adoption check pass; the generated-system ownership decision is now explicitly linked to D5.
- Foundation lock is `pantheon-base-v0.13.0` at `c254e4c8c63f4409d49752eedb097291627d0b82`.

## Open Delivery Work

| ID | Work | Current state | Estimate |
| --- | --- | --- | --- |
| D1 | Real Prometheus/VictoriaMetrics, Loki, Tempo, Kubernetes, ArgoCD, and Velero acceptance | endpoints/credentials unavailable | 2-4d |
| D2 | SSRF allowlist/private-network policy and tests | policy decision required | 1-2d |
| D3 | Base-first consumer re-review against v0.13.0 | ready; evidence missing | 0.5-1d |
| D4 | Tenant G2/G3 gray gates | both OPEN | 3-7d + DBA window |
| D5 | Active Harness/generated/docs gate repair | partially failing | 1-2d active; 2-5d historical triage |

## Workload Reading

For a **single-tenant production release**, the remaining engineering effort is approximately **4.5-9 engineer-days** after endpoints, scoped credentials, and policy owners are available. D1 and D2 can overlap. Calendar time is unbounded while those dependencies are absent.

For a release that also claims **multi-tenant migration readiness**, add **3-7 engineer-days** and a DBA-backed maintenance window. That estimate excludes any schema or authorization defect discovered by the G2/G3 rehearsal.

These are medium-confidence estimates. They measure execution after prerequisites exist; they are not a promise that external systems or human approvals will arrive on a date.

## Evidence Boundaries

- The 2026-08-31 production-readiness review remains the authoritative external-gap record until D1-D3 produce a newer review.
- Current Harness debt is separated into active closure work (UI gate 2 findings, docs links 13, docs inventory 10, visual legacy warnings 8) and historical packet migration debt (62 files, 558 errors, 95 warnings in the full scan).
- The two 2026-09-10 tenant-picker screenshots are rendered UI evidence only; they do not prove cross-tenant runtime isolation or migration safety.
- Historical v1.2 documents claiming 30/30 or 100% completion describe an earlier feature checklist and do not close the current production gates.
