# Findings-first Review — SonarCloud CI Analysis Restored for pantheon-ops

## Findings

### P1 - An undeclared Ops-only root file was deleted by a foundation consumption and stayed deleted

`sonar-project.properties` is Ops-owned configuration: `pantheon-base` ships only
`.sonarcloud.properties`. Because it was never listed in `business-overlay.json`, the assembly
had no instruction to keep it, and `5c12964f` (consume `pantheon-base-v0.17.7`) removed it from the
live tree. Nothing complained: the overlay checker only asserts that declared paths exist, so an
*undeclared* Ops file is invisible to it, and the loss surfaced three weeks later as a scanner
error rather than as a gate failure.

That is the same failure class the 0.18.1 round recorded for the pinned
`scripts/task-manifest.mjs` overlay — overlay declarations, not file contents, decide what survives
a consumption — but mirrored: there a pin froze a base file at the consumer, here a missing pin
deleted a consumer file. The repair is the declaration plus the restored file, and the rebuild that
goes from 758 to 761 declared/generated files is the sensor that proves it.

### P1 - The credential failure masked the configuration failure

`SONAR_TOKEN` returned HTTP 403 on every run, so the scanner never got far enough to complain about
missing mandatory properties. With the rotated token installed, dispatch `40849180395` reached the
analysis stage and failed with `You must define the following mandatory properties:
sonar.projectKey, sonar.organization`. A single red check was hiding two independent defects; the
before/after error text is the evidence that both were real.

### P2 - SonarCloud's automatic scope file is not a substitute for the CI configuration

`.sonarcloud.properties` survived every consumption because it is declared, and it looks like
enough: it carries the same exclusion rationale. It is not read by the scanner CLI, so keeping it
declared while the CI file went undeclared made the repository look Sonar-configured while the CI
channel was already dead. The two files have different consumers and both have to be declared.

### P2 - Coverage exclusions are load-bearing, not cosmetic

The restored file carries `sonar.coverage.exclusions=**/*` and the test-file `sonar.cpd.exclusions`.
The organization is on the SonarCloud Free plan, which cannot attach a custom quality gate, so the
built-in `Sonar way` `new_coverage` condition can only be neutralised by making coverage
`not computed`. Losing the file silently re-armed a condition the repository cannot satisfy, on top
of losing the project identity.

### P2 - Dependabot runs are still structurally unable to scan

GitHub does not expose repository secrets to Dependabot-triggered workflows, so on those runs
`SONAR_HOST_URL` is empty and the scanner dies with `URI with undefined scheme`. Two legitimate
fixes exist — provision Dependabot secrets, or skip the job for `dependabot[bot]` — and both change
who is allowed to be scanned, which is gate policy rather than plumbing. Neither was chosen here;
it is recorded as a maintainer decision.

## What was verified instead

- Token validated against `api/authentication/validate`, organization resolved to `duanxldragon`,
  and the existing project `duanxldragon_pantheon-ops` confirmed through `api/projects/search`, so
  no provisioning or project creation was needed.
- The restored file is byte-for-byte the last known-good revision from `5c12964f^`, not a rewrite.
- `check:business-overlay` (53 declared paths), `test:business-overlay` (fail 0) and
  `rebuild:from-base` (761 declared/generated files, rebuilt tree contains the file) all green after
  the declaration.
- This packet's `check:task-packet`, `check:harness-adoption` and PR governance body check all green.

## Machine Readable

```json
{
  "verdict": "approved",
  "findings": [
    {"id": "S1", "severity": "P1", "title": "Undeclared Ops-only root config deleted by a foundation consumption and invisible to the overlay checker", "state": "fixed-with-declaration"},
    {"id": "S2", "severity": "P1", "title": "HTTP 403 credential failure masked the missing projectKey/organization defect", "state": "closed-by-rotated-token-and-restored-file"},
    {"id": "S3", "severity": "P2", "title": ".sonarcloud.properties is not read by the scanner CLI and cannot substitute for sonar-project.properties", "state": "documented-both-declared"},
    {"id": "S4", "severity": "P2", "title": "Coverage and cpd exclusions are load-bearing on the Free plan", "state": "restored"},
    {"id": "S5", "severity": "P2", "title": "Dependabot runs cannot read repository secrets so the scan stays red there", "state": "maintainer-gate-policy-decision"}
  ],
  "structuralReview": {
    "affectedSubgraph": [
      "business-overlay.json repositoryOverlayPaths -> rebuild-from-base assembly -> sonar-project.properties",
      ".github/workflows/sonarcloud.yml -> sonar-scanner CLI -> SonarCloud project duanxldragon_pantheon-ops"
    ],
    "checks": ["hub", "sensitive-flow"],
    "findings": [
      "S1: the overlay checker enforces declared paths that exist but not Ops files that must be declared, so an undeclared consumer-only file is an unguarded deletion channel",
      "S2: one red check with two independent causes means the failure text, not the conclusion, is the useful evidence"
    ],
    "notes": "No product code changed. One restored Ops configuration file and one overlay declaration; no workflow semantics, no gate policy and no secret value committed."
  },
  "residualRisk": [
    "Dependabot PR runs keep a red SonarCloud Scan until the maintainer provisions Dependabot secrets or approves skipping the job for that actor.",
    "The project quality gate reads ERROR on new_security_rating with 4 vulnerabilities and 245 code smells; the scan now surfaces that backlog again instead of failing to run.",
    "Another Ops-only root file added without a repositoryOverlayPaths declaration will be deleted by the next consumption in exactly the same way."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-10-29-sonar-ci-analysis-restore.task.md",
    "evidence": ".harness/evidence/2026-10-29-sonar-ci-analysis-restore/commands.json",
    "reviewFile": ".harness/evidence/2026-10-29-sonar-ci-analysis-restore/review.md",
    "changeRef": "none",
    "planRefs": [
      "docs/harness/tasks/2026-10-29-sonar-ci-analysis-restore.task.md",
      "docs/harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1.task.md"
    ]
  }
}
```
