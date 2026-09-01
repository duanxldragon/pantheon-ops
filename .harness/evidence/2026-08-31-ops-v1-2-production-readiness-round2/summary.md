# Ops V1.2 Production Readiness Round 2

Task: `2026-08-31-ops-v1-2-production-readiness-round2`
Status: complete
Source runtime: `http://127.0.0.1:8092`
Source binary: `.tmp-deploy-2026/pantheon-ops-smoke-r2-secure.exe`

## Scope and outcome

- Observability IA is `可观测中心 -> 监控中心 / 告警中心 / 日志中心 / 链路追踪`.
- Service Catalog owns `BizScope -> Application -> Service -> ServiceInstance`.
- Kubernetes owns the Kubernetes `Service` resource at `/business/k8s/service`.
- FileDist routes are protected by token auth, Casbin, and data-scope middleware on both canonical and compatibility paths.
- FileDist storage and observability notification secrets are encrypted at rest and redacted from responses.
- Observability datasource, alert rule/event, and notification records carry department ownership and enforce data scope.
- Service `runtimeType` is validated against instance `targetType`; Hybrid is the only mixed-target mode.
- Menu contract now includes observability and CICD modules: 37 menus, 47 routes, 208 permissions, 49 component keys.

## Verification evidence

- `go test ./... -count=1`: PASS
- `go vet ./...`: PASS
- Windows CGO race gate (`PATH=D:\msys64\mingw64\bin`, `CGO_ENABLED=1`, `go test -race ./...`): PASS
- `npm run lint`: PASS
- `npm run type-check`: PASS
- `npm run build`: PASS
- `npm run check:menu-contract`: PASS
- `npm run check:business-overlay`: PASS
- `npm run test:business-overlay`: PASS (8/8)
- Focused source smoke on port 8092: PASS (observability/FileDist 5/5; service/K8s 4/4)
- Cleanup verification after smoke: service applications/instances/scopes/hosts 0; observability scopes/targets/datasources/rules 0; FileDist files/tasks 0.

## Rendered evidence

All screenshots use exact `1440x900` desktop or `390x844` mobile viewports and are stored under `screenshots/`:

- `observability-monitoring-desktop.png`, `observability-monitoring-mobile.png`
- `observability-alerts-desktop.png`, `observability-alerts-mobile.png`
- `observability-logs-desktop.png`, `observability-logs-mobile.png`
- `observability-traces-desktop.png`, `observability-traces-mobile.png`
- `service-catalog-desktop.png`, `service-list-mobile.png`
- `k8s-service-desktop.png`, `k8s-service-mobile.png`
- `k8s-cluster-desktop.png`, `k8s-cluster-mobile.png`
- `k8s-release-desktop.png`
- `filedist-files-desktop.png`, `filedist-files-mobile.png`

Impeccable review found no text overlap or page-level horizontal overflow in the corrected desktop/mobile states. FileDist mobile evidence is captured after closing the responsive navigation drawer.

## Independent review synthesis

- Code reviewer: `REQUEST CHANGES` before remediation. Closed in this round: unauthenticated FileDist routes, plaintext FileDist/notification secrets, unscoped observability access, service runtime/target mismatch, and misleading menu-contract exclusions.
- Architect: `BLOCK`. The remaining block is evidence and delivery-boundary related: real Prometheus/Loki/Tempo/Kubernetes/ArgoCD/Velero endpoints and credentials were not available, and the current worktree still contains base-owned compatibility/runtime edits that must be upstreamed through a foundation release before production deployment.

## Remaining production gates

- Real external acceptance is still required for Prometheus/VictoriaMetrics, Loki, Tempo, Kubernetes Service CRUD, ArgoCD, and Velero with production-like credentials.
- Stored datasource URLs remain an SSRF policy decision for deployment: production must configure an explicit outbound allowlist/private-network policy before enabling proxy queries.
- Base-owned changes under `backend/cmd/server`, `backend/internal/middleware`, `backend/pkg/common`, and generic frontend tooling must be moved to `pantheon-base` and consumed through the foundation release; this ops tree must not be treated as the long-term owner.
- Service Catalog smoke drives API lifecycle and verifies all UI tabs; it does not exercise every create/update/delete modal exclusively through browser clicks.

## Verdict

Repository-local implementation is **RC-ready** with the listed tests and rendered evidence. Direct production delivery is **NOT YET APPROVED** until the external integration, outbound SSRF policy, and base-first inheritance gates are closed and re-reviewed.
