# Task: Ops V1.2 Production Readiness Round 2

Task ID: `2026-08-31-ops-v1-2-production-readiness-round2`
Priority: P0 production gate
Level: L2
Owner: `business/observability`, `business/service`, `business/k8s`, business smoke
Parent evidence: `.harness/v1.2-tasks/`, `.harness/evidence/2026-08-28-v1-real-acceptance-r2/`

## Goal

Re-audit Pantheon Ops against the v1.2 requirements, close repository-local
production blockers, and produce evidence for a release decision based on the
current source tree rather than stale binaries or self-reported completion.

## In Scope

- Reorganize observability navigation around monitoring, alerting, logs, and tracing.
- Provide real Prometheus-compatible, Loki, and Tempo query paths before exposing pages.
- Enforce observability data scope, action permissions, audit metadata, and reference protection.
- Complete the Application -> Service -> ServiceInstance management workflow.
- Add the missing Kubernetes Service route, menu, page, permissions, i18n, and smoke coverage.
- Repair file-distribution terminal-state smoke behavior and unstable page wait conditions.
- Run focused backend tests, frontend contract gates, build, business smoke, and rendered UI checks.

## Out Of Scope

- Copying or overriding platform/system behavior from `pantheon-base`.
- Implementing Prometheus, Loki, Tempo, Kubernetes, Argo CD, or Velero themselves.
- Claiming external integration acceptance without reachable real endpoints and credentials.
- Expanding the service catalog beyond Application, Service, and ServiceInstance.
- Replacing the existing Go stack with Python components.

## Acceptance

- Navigation and component registries are contract-clean and no menu lands on a placeholder.
- Monitoring targets are data-scoped and creation validates an authorized active BizScope.
- Datasource deletion is blocked while referenced; default changes are transactional and reversible.
- Logs and traces execute real backend queries with loading, empty, and error states.
- Service catalog supports create/update/delete for its three entities with permission-aware controls.
- Kubernetes Service can be listed, created, and deleted through the existing client-go backend.
- Target smoke passes against a freshly built backend on an isolated port.
- Desktop and mobile screenshots exist for the changed user-facing flows.
- Final review distinguishes closed blockers from external real-machine gaps.

## Verification

```text
backend: go test -count=1 ./modules/business/observability/... ./modules/business/service/... ./modules/business/k8s/... ./modules/business/filedist/...
frontend: npm run type-check; npm run build; npm run check:menu-contract; npm run check:business-overlay
smoke: observability, service catalog, Kubernetes Service, file distribution, deploy pages
render: 1440x900 and 390x844 plus loading/empty/error states
```

Evidence root: `.harness/evidence/2026-08-31-ops-v1-2-production-readiness-round2/`.

## Stop Conditions

Stop and record a gap if completion requires a Base/system-domain override, production
credentials, destructive external actions, or an unavailable Prometheus/Loki/Tempo/K8s endpoint.
