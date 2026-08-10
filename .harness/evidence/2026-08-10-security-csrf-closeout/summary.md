# Evidence Summary

Dependency remediation is complete locally: official npm audit reports zero vulnerabilities; Go dependency upgrades are recorded in `go.mod` and `go.sum`. Ops consumes immutable `pantheon-base-v0.10.10` (`a95e6e52`), whose shared request client persists CSRF tokens from response headers and sends them on unsafe requests.

The remaining Ops-owned CodeQL allocation-size-overflow finding is remediated and tested. The quality workflow now follows the relevant Base controls: stale-run cancellation, `main`/`release/**` execution, and PR/merge-group new-code Go linting. Hosted smoke and CodeQL alert recalculation remain pending PR validation.
