# Service Foundation Evidence

Task: `ops-service-instance-foundation`.

## Outcome

The minimum business-owned service identity layer is present:

- `Application -> Service -> ServiceInstance`
- typed in-process Reader/Command capability
- VM and K8s target validation through owner APIs
- service and instance routes under `/business/service`
- business overlay registry and menu/i18n seeds wired

## Validation

- `go test -count=1 ./modules/business/service/...` pass
- `go test -race ./modules/business/service/...` pass with MinGW CGO
- `go vet ./...` pass
- `npm run type-check` pass
- `npm run build` pass
- `npm run check:menu-contract` pass
- `node scripts/check-business-module-boundaries.mjs` pass
- `git diff --check` pass

## Compatibility Decision

Legacy `Host.installed_components` stays read-compatible only and is no longer
treated as the source of service identity. This task does not add a new dual-write
path.

## UI Note

No new visual layout was introduced; the existing service list surface was kept
stable. Evidence here is static build and contract validation rather than a new
rendered screenshot.

