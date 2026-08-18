# pantheon-ops Business Test Plan

Updated: 2026-08-18  
Type: Test Plan  
Status: Proposed

Normative test plan: [BUSINESS_TEST_PLAN.md](./BUSINESS_TEST_PLAN.md).

The test gates are:

- Gate A: Go/TypeScript unit tests, contracts, overlay checks, and boundary checker.
- Gate B: state-machine, idempotency, concurrency, lease, cancellation, stale callback, and secret-redaction invariants.
- Gate C: BizScope -> CMDB -> Service -> Deploy -> K8s integration and frontend smoke.
- Gate D: MySQL migration rehearsal, isolated K8s API tests, and safe SSH executor tests.

The current evidence has passing `go test ./...`, `go vet ./...`, and boundary checker results. Race, MySQL, frontend build, and real Kubernetes smoke remain explicit gaps until run in their required environments.
