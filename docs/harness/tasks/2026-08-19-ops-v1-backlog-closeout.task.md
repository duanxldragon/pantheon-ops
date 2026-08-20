# Task Packet: ops-v1-backlog-closeout

## Goal

Close the remaining Pantheon Ops V1 implementation and high-risk acceptance
gates with reproducible evidence and explicit external boundaries.

## Primary Layer

business/*

## Dependency Layers

- platform
- system/iam

## Harness Profile

- Template: custom
- Overlay: business-overlay
- Quality Profile: release-sensitive
- Portable Failure Class: runtime-evidence-gate
- Owner Layer: business
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `business-overlay.json`
- `docs/designs/BUSINESS_DEVELOPMENT_PLAN.md`
- `docs/designs/BUSINESS_TEST_PLAN.md`
- `backend/modules/business/k8s/cluster/cluster_service.go`

## Scope

### In

- Concurrent Kubernetes cluster ownership validation and stale-scope rejection.
- Isolated MySQL snapshot duplicate/index and rollback/reapply rehearsal.
- Authorized isolated Kubernetes resourceVersion/Secret writes with cleanup.
- Authorized isolated SSH write/hash/delete validation.
- Base-first Harness overlay/adoption verification and evidence closeout.

### Out

- Direct production database rollback or destructive data conversion.
- Production/live-cluster mutation, new platform services, and V2 observability.
- Copying generic Harness files into Ops or changing pantheon-base in this branch.

## Expected Files

### Create

- `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/high-risk-gates-20260820.md`

### Modify

- `backend/modules/business/k8s/cluster/cluster_service.go`
- `backend/modules/business/k8s/cluster/cluster_service_test.go`
- `backend/pkg/database/snapshot_rehearsal_test.go`
- `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/`
- `.harness/tasks/2026-08-19-ops-v1-backlog-closeout/`
- `docs/designs/BUSINESS_DEVELOPMENT_PLAN.md`

### Do Not Touch

- `pantheon-base` source files and the locked foundation release during this branch.
- Production databases, production clusters, and unapproved remote paths.

## Implementation Notes

- Keep ownership mutation inside `WithClusterLock` and pass caller DataScope to
  BizScope validation; update scope name and department atomically.
- Use isolated schemas and temporary namespaces/files only; never record secrets.
- Consume Base through the locked foundation release and record release-gate gaps
  instead of creating a local Harness fork.

## Structural Scope

- Affected Subgraph: cluster handler -> scoped service -> MySQL row/advisory lock; migration runner -> business schema; K8s/SSH external gates -> evidence.
- Boundary Crossings: business k8s -> BizScope capability; Ops -> Base foundation release; test harness -> isolated MySQL/K8s/SSH runtimes.
- Risk Nodes: stale authorization, generated unique keys, resourceVersion conflicts, remote cleanup, foundation release drift.
- Graph Focus: no cross-owner table access, no new dependency cycle, sensitive writes remain scoped and reversible.

## Verification Plan

- `go test -count=1 ./modules/business/k8s/cluster ./pkg/database` with isolated MySQL.
- `go test -count=1 ./...`, `go vet ./...`, and Windows CGO `go test -race ./...`.
- `npm run check:business-foundation` and `npm run test:business-overlay`.
- Evidence checks: task packet, evidence, review, runtime evidence, and docs frontmatter.
- Isolated K8s and SSH mutation commands recorded in `commands.json`.

## Linkage

- Task ID: `2026-08-19-ops-v1-backlog-closeout`
- Task Manifest: `.harness/tasks/2026-08-19-ops-v1-backlog-closeout/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/designs/BUSINESS_DEVELOPMENT_PLAN.md`, `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`, `docs/designs/BUSINESS_TEST_PLAN.md`
- Evidence Directory: `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/`
- Review File: `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/review.md`

## Evidence Required

- `commands.json`, `summary.md`, `review.md`, `qa-matrix.md`, and high-risk gate notes.
- Fresh test, race, vet, overlay, runtime, and external mutation outcomes.
- Credentials, kubeconfig contents, and secret values excluded from artifacts.

## Human Gates

- Base must publish a green foundation release before consumer-root Harness strict checks can close.
- Direct production database rollback remains an operator change-window action.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
