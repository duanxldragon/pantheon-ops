# Evidence Summary

Dependency remediation is complete: the Go upgrade is recorded in `go.mod` and `go.sum`; `govulncheck` reports zero reachable vulnerabilities; and hosted Security run `31445476446` passed the pinned OSV v2.3.3 call-analysis gate. Its gate parses the JSON report and blocks any finding without `called:false`, rather than treating OSV's generic non-zero finding exit as a reachable vulnerability.

Ops consumes immutable `pantheon-base-v0.10.10` (`a95e6e52`), whose shared cookie implementation sets `HttpOnly`, `Secure`, and `SameSite=Strict`, including the CSRF response-header contract. All eight historical CodeQL alerts are dismissed with individual evidence: the retired cookie and system paths no longer exist; current upload and generated-module paths normalize, restrict, and contain user-controlled values. Dependabot and CodeQL both report zero open alerts.

The quality workflow treats manual full-repository Go linting as a report-only baseline, matching push behavior; new-code lint remains blocking for pull requests and merge groups. Gitleaks passes with a single commit-level initial-import allowlist. Hosted Quality run `31445476431` and hosted Smoke job `93638664396` passed, closing the external validation loop.
