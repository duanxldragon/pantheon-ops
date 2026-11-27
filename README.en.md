# Pantheon Ops

Chinese version: [README.md](./README.md)

Pantheon Ops consumes official Pantheon Base foundation releases and adds operations business modules. Base owns the platform and system domains; Ops owns its business domains, assembly and business acceptance.

The product is preparing for a pilot and has not passed pilot acceptance. CMDB, BizScope, Service, Deploy and K8s have implementations. Deployment-instance integrity, catalog maintenance, K8s Service editing and acceptance on the current candidate remain open. Observability, file distribution, GitOps and backup have implementations or runtime assembly and require acceptance for every capability exposed in the delivery. The dedicated Helm module currently records releases; that does not prove Chart installation.

The current product version is recorded in [`VERSION`](./VERSION). The consumed Base release tag and commit are recorded in [`foundation-release.lock.json`](./foundation-release.lock.json). Changes by version are listed in the [Base release notes index](https://github.com/duanxldragon/pantheon-base/blob/main/docs/release-notes/README.en.md).

Existing business audit evidence applies to the reviewed commit. Collect acceptance evidence for the delivery candidate. The frontend uses React, TypeScript, Arco Design and Vite. The backend uses Go, Gin and GORM with MySQL and Redis.

The dedicated Helm API registers, lists and deletes release records only. Chart install, upgrade, rollback and uninstall are deferred and unavailable. New records always store `registered`; a requested runtime status is not saved as an execution result. Every response reports `status=registered` and compatible `effectiveStatus=registered`; `storedStatus` preserves the original source value without rewriting historical database rows. Neither a historical `deployed` value nor a K8s image Release proves Chart installation. Deleting a registration does not uninstall cluster resources.

## Current plan

- [Pilot delivery plan](./.harness/tasks/2026-11-16-ops-pilot-delivery/plan.md): six required core packages and four conditional extension packages, with dependencies and testable exit criteria.
- [Execution queue](./.harness/tasks/2026-11-16-ops-pilot-delivery/EXECUTION_QUEUE.md): start with C01; C03/C04 can proceed independently.
- [Task status](./.harness/STATUS.md) and [archive index](./.harness/ARCHIVE.md): 12 historical tasks archived within their original scope; evidence preserved.
- [Business functional design](./docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md): current capabilities and boundaries.

## Development entry

1. Read the [document index](./docs/README.en.md), [inheritance guide](./docs/PROJECT_INHERITANCE.en.md) and [AGENTS.md](./AGENTS.md).
2. Prepare isolated MySQL/Redis and environment variables using the Base [deployment guide](../pantheon-base/docs/DEPLOYMENT_GUIDE.md); Ops applies its business migrations and runtime seeds.
3. Run go run ./cmd/server from backend/; run npm install and npm run dev from frontend/.

Run npm run check:business-overlay and npm run check:shared-drift from the repository root. Rebuild from the locked official release as described in the inheritance guide. Business changes need their package tests, smoke suites and full CGO race checks. Compilation does not replace provider or user-flow acceptance.

Shared capability changes belong in [pantheon-base](https://github.com/duanxldragon/pantheon-base). See [LICENSE](./LICENSE).
