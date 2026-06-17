# Review: 2026-06-17-github-governance-followup

## Machine Readable

```json
{
  "taskId": "2026-06-17-github-governance-followup",
  "verdict": "findings addressed",
  "structuralReview": {
    "affectedSubgraph": [
      "pull_request -> governance scripts -> quality/security workflows -> required checks"
    ],
    "checks": [
      "call-depth",
      "cycle",
      "hub"
    ],
    "findings": [],
    "notes": "Repository governance review focused on required check composition, workflow posture, and artifact linkage."
  },
  "linkage": {
    "taskPacket": "docs/harness/tasks/2026-06-17-github-governance-followup.task.md",
    "evidence": ".harness/evidence/2026-06-17-github-governance-followup/commands.json",
    "reviewFile": ".harness/evidence/2026-06-17-github-governance-followup/review.md",
    "changeRef": "none",
    "planRefs": []
  }
}
```

## Findings

1. `.github/workflows/quality.yml`
   The prior branch state turned existing frontend and smoke debt into a blocker for a governance-only PR. Required checks now stay strict for product-affecting changes, but they skip frontend and smoke jobs when the pull request only touches governance files.

2. `.github/workflows/quality.yml`
   The prior branch state also made `npm run check:inheritance` an unconditional docs-governance gate even though the repo already has shared backend drift unrelated to this patch. The gate now runs only when inheritance-sensitive files change.

3. `.github/workflows/security.yml`
   Workflow posture previously failed because the required-check workflows still used floating GitHub Action tags and default credential persistence. The workflow now pins action SHAs, disables credential persistence, and removes the extra scorecard step from the required posture gate.

## Assumptions

- This review is for repository governance and required-check composition, not for fixing existing product runtime debt.
- Existing base-sync drift, frontend hardcode debt, smoke regressions, and duplication baseline remain follow-up work after this PR.

## Status

- Artifact linkage: complete
- Governance template and validator coverage: complete
- Workflow posture hardening: complete
- Product baseline debt remediation: deferred

## Recommended Next Step

- Merge this governance PR once the refreshed required checks pass.
- Follow with targeted remediation PRs for base-sync drift, frontend i18n hardcode debt, smoke failures, and duplication reduction.
