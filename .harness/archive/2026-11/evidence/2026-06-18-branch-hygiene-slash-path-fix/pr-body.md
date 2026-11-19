## Summary

- Target repo: `pantheon-ops`
- Layer: `platform/governance`
- Task mode: `implement`
- Sync expectation: `included base -> ops`

## Scope

- In scope: `preserve slash-separated branch names for hosted GitHub branch lookup/delete endpoints in branch-hygiene cleanup and add the matching regression coverage`
- Out of scope: `business-domain code, inheritance overlays, PR governance rules, merge strategy, and query-parameter encoding semantics`

## Verification

- Commands: `npm run test:branch-hygiene`
- Result: `the dedicated branch-hygiene regression suite passes locally after switching hosted GitHub branch lookup/delete paths to raw slash-separated branch names`

## Evidence

- Task ID: `2026-06-18-branch-hygiene-slash-path-fix`
- Task Manifest: `.harness/tasks/2026-06-18-branch-hygiene-slash-path-fix/manifest.json`
- Evidence: `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/commands.json`
- Human gate: `none`

## Review

- Review status: `passed`
- Review artifact: `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/review.md`

## Release Risk

- Known gaps: `hosted GitHub residue cleanup still needs one post-merge rerun to prove live slash-branch deletion end-to-end`
- GitHub signal: `repo-quality-gate`
