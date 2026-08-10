# Review: 2026-08-10-security-csrf-closeout

## Findings

- `business/deploy` used `len(base) + len(override)` as an allocation capacity. Removing the preallocation eliminates the CodeQL integer-overflow path while preserving parameter precedence, covered by a focused unit test.
- Dependabot proposed TypeScript 7 while the locked `typescript-eslint` peer range is `<6.1`, causing deterministic `npm ci` failures. The incompatible TypeScript major update is ignored; no peer-dependency override weakens CI.
- Base-derived quality controls now cancel stale runs, validate new Go code with golangci-lint on PRs and merge groups, and execute quality gates after main and release pushes. Full-repository lint remains report-only on push until inherited historical lint debt is retired.
- Zizmor's cache-poisoning check required the new Go lint job to disable `setup-go` dependency caching; the workflow now uses the same no-cache posture as the existing Ops jobs.
- OSV Scanner v2.4.0 cannot run under the repository's Go 1.25 toolchain. Pinning v2.3.3 and enabling Go call analysis preserves a working scanner. The gate parses JSON so `GO-2026-5932` is non-blocking only because the scanner proves `openpgp` is not called; any unanalysed or reachable finding remains blocking.
- Historical CodeQL alerts 4, 6, 7, 8, 9, 10, 12, and 14 were manually reviewed and dismissed as false positives: each reported sink is either in a removed path or guarded by the current Base-derived cookie, object-key, or workspace-containment controls. The GitHub alert API now reports zero open CodeQL alerts.

## Residual Risk

- Hosted Quality and Security runs for the new PR remain required, including hosted smoke. The current Windows workstation cannot reach npm's audit advisory endpoint; that external connectivity gap is not treated as an audit pass.

## Machine Readable

```json
{
  "taskId": "2026-08-10-security-csrf-closeout",
  "verdict": "awaiting-hosted-validation",
  "findings": [],
  "structuralReview": {
    "status": "CLEAR",
    "checks": ["foundation-release-alignment", "workflow-gate-scope", "csrf-cookie-contract", "workspace-containment"],
    "findings": []
  },
  "residualRisks": [
    "Hosted Quality and Security gates, including smoke, have not run for this PR.",
    "Local npm advisory API access is unavailable; hosted audit is the authority."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-08-10-security-csrf-closeout/manifest.json",
    "evidence": ".harness/evidence/2026-08-10-security-csrf-closeout/commands.json",
    "reviewFile": ".harness/evidence/2026-08-10-security-csrf-closeout/review.md"
  }
}
```