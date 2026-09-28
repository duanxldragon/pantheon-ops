# Findings-first Review

## Findings

### P0 - Direct production delivery is still blocked by missing external evidence

The latest production-readiness review records real acceptance as missing for Prometheus/VictoriaMetrics, Loki, Tempo, Kubernetes Service CRUD, ArgoCD, and Velero. Local unit tests, builds, and screenshots prove repository behavior but cannot prove those external contracts.

Evidence: `.harness/evidence/2026-08-31-ops-v1-2-production-readiness-round2/summary.md`.

### P0 - Datasource proxy SSRF policy is not closed

The same review records stored datasource URLs as an unresolved SSRF policy decision. Production must define and test an explicit outbound allowlist/private-network policy before proxy queries are enabled.

Evidence: `.harness/evidence/2026-08-31-ops-v1-2-production-readiness-round2/summary.md`, child packet D2.

### P0 - Base-first ownership needs a fresh consumer review

The locked foundation is now v0.13.0 and the overlay checks pass, but the prior production review required generic compatibility/runtime edits to be upstreamed before delivery. A clean v0.13.0 consumer comparison and findings-first review are still required. The current dirty generated-system change also fails Harness adoption because it has no matching task/evidence linkage.

Evidence: `foundation-release.lock.json`, `.business-overlay-report.json`, `npm run check:business-foundation`, `npm run check:harness-adoption`.

### P1 - Tenant migration G2 and G3 remain open

The tenant runbook explicitly records G2 and G3 as `OPEN`: no production-backup RPO/RTO and restore evidence exists, and no production-scale sizing/window calculation exists. G1 and G4 being granted does not authorize production DDL.

Evidence: `docs/runbooks/TENANT_MIGRATION_RUNBOOK.md` §8.1.

### P1 - Active Harness governance gates remain partially red

The current checks report 13 docs-link findings, 10 docs-inventory findings, 8 legacy manifest warnings, and 2 UI-quality findings. The adoption check is now green after this task/evidence linkage. The global task-packet scan reports 558 errors and 95 warnings across 62 files, with the new parent packet passing independently. This is real governance debt, but it should be triaged into active-gate fixes and bounded historical cleanup rather than treated as a single product implementation deficit.

Evidence: `.harness/evidence/2026-09-28-ops-delivery-readiness-gap-closure/commands.json`.

## Positive Signals

- `go test -count=1 ./...`, `go vet ./...`, and Windows CGO `go test -race ./...` passed.
- Frontend type-check, lint, and build passed.
- `npm run check:business-foundation` passed, including business boundaries, overlay contract, and 8 overlay tests.
- Method, sync, encoding, and structure checks passed.

## Decision

Keep the release state at **RC-ready / production not approved**. Execute D3 and the active D5 items first. Only schedule D1 and D2 when scoped external access and the outbound policy decision are available. Keep D4 as a separate multi-tenant migration gate.

## Unknowns

- Whether the intended release includes tenant migration or only the single-tenant Ops platform.
- Which maintainer owns the SSRF policy decision and which Base release will carry it.
- Availability and maintenance windows for the six external integrations and the DBA restore/sizing drills.

## Machine Readable

```json
{
  "taskId": "2026-09-28-ops-delivery-readiness-gap-closure",
  "status": "reviewed",
  "verdict": "blocked",
  "findings": [
    {"id": "D1", "severity": "P0", "title": "External integration acceptance", "state": "blocked-on-endpoints-and-credentials"},
    {"id": "D2", "severity": "P0", "title": "Datasource proxy SSRF policy", "state": "decision-required"},
    {"id": "D3", "severity": "P0", "title": "Base-first consumer review", "state": "open"},
    {"id": "D4", "severity": "P1", "title": "Tenant migration G2/G3", "state": "open-separate-release-track"},
    {"id": "D5", "severity": "P1", "title": "Harness and generated-artifact governance", "state": "partially-open-active-gates"}
  ],
  "residualRisk": [
    "External systems, credentials, and maintainer policy decisions are unavailable.",
    "Historical Harness docs and packet migration debt remains outside the direct product implementation estimate.",
    "Multi-tenant migration is not authorized while G2/G3 remain OPEN."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure.task.md",
    "evidence": ".harness/evidence/2026-09-28-ops-delivery-readiness-gap-closure/commands.json",
    "reviewFile": ".harness/evidence/2026-09-28-ops-delivery-readiness-gap-closure/review.md",
    "changeRef": "none",
    "planRefs": [".harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/EXECUTION_QUEUE.md"]
  }
}
```
