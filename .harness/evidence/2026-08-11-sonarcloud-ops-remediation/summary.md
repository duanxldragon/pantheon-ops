# SonarCloud Ops Remediation Evidence

## Baseline

- Captured: 2026-08-11.
- `pantheon-ops`: 486 unresolved findings (1 Bug, 72 Vulnerabilities, 413 Code Smells); no unreviewed hotspots.
- `pantheon-base`: 0 unresolved findings; no unreviewed hotspots.
- Ops analysis revision: `69c59f8151f3f0c856ab0897af6717c0a475a4e5` (2026-08-10).
- Ops local `main`: `4604ea48ed01550919f856e39c3029ed024d9b9c` (2026-08-11).

## Initial Classification

- 195 tooling/workflow findings: automatic-analysis scope mismatch or separately triaged CI security.
- 91 SQL/i18n findings: the same known exclusions recorded in `pantheon-base/.sonarcloud.properties`.
- 23 locked foundation-source findings: an issue-disposition mismatch between separate SonarCloud projects.
- 21 ops overlay/unlocked common-source findings: require base-consumption review.
- 156 business-owned findings: require focused remediation; none are SonarCloud Bugs or Vulnerabilities in the baseline.

## Completed Remediation

- Added `.sonarcloud.properties` with the approved `pantheon-base` automatic-analysis exclusions: repository tooling, SQL, and i18n resource maps. Business modules, shared product code, tests, Dockerfile, and GitHub Actions remain scanned.
- Replaced the release-tree implicit `Array.sort()` ordering with an explicit code-point comparator in `scripts/foundation-release/shared-foundation-rules.mjs`, resolving `javascript:S2871` while preserving the existing signed release-tree hash ordering.
- Added `--ignore-scripts` to all 11 CI `npm ci` installs, addressing the workflow-install posture behind `githubactions:S6505`.
- Removed the redundant workflow-level `contents: read` declaration from `.github/workflows/inheritance-drift-detection.yml`; job permissions remain least-privileged.

## Ownership Result

- Full classification: `ownership-classification.md`.
- `frontend/src/modules/business/{cmdb,deploy,bizscope}/**` is legitimate operations business source.
- `frontend/src/{App.tsx,main.tsx,hooks/usePermission.ts,api/file.ts,api/upload.ts,api/importExport.ts}` is generic shared-source drift outside the locked foundation path list; it must be reconciled in `pantheon-base` and delivered through a foundation release.
- No bulk `WONTFIX` transitions were used, and no inherited foundation source was locally changed to hide an ops-only SonarCloud result.

## Remaining Work and Boundary

- The 156 business-owned findings remain a focused remediation backlog. The priority is behavior, accessibility, and safety fixes; historical complexity debt requires tests before structural refactors.
- The generic frontend candidates above are intentionally not repaired in ops. The base release manifest now declares them as foundation-owned, and the ops checker rejects a future local omission before it can be mistaken for business code.
- A hosted SonarCloud analysis of the candidate SHA is still required after commit and push. The 486-item baseline is stale and cannot prove post-change counts.
- No visual render was produced because this batch did not modify UI implementation. Future `business/*` UI remediation needs desktop, narrow viewport, and interaction-state evidence under this evidence directory.

## Foundation-Consumption Repair

- `pantheon-base` now declares `frontend/src/App.tsx`, `frontend/src/main.tsx`, `frontend/src/vite-env.d.ts`, `frontend/src/api`, and `frontend/src/hooks` in its release manifest and rejects a bundle when an existing required surface is unowned.
- `pantheon-ops` now reports `UNOWNED` generic frontend sources, verifies its lock path set against an installed release manifest, and requires the same path coverage in `check:inheritance`.
- Focused verification passed: 13 base manifest/bundle/cut tests; 31 ops sync, inheritance-contract, and release-consumer tests; `git diff --check` passed for both repositories.
- No base commit, release tag, GitHub release, archive publication, ops lock upgrade, push, or hosted SonarCloud run has been performed. Those operations require a committed base revision and a separate outward-facing authorization.

## Raw Inputs

- `artifacts/sonarcloud-ops-baseline-issues.json`
- `artifacts/sonarcloud-base-baseline-issues.json`
- `artifacts/sonarcloud-ops-hotspots.json`
- `artifacts/sonarcloud-ops-analyses.json`
- `artifacts/sonarcloud-base-analyses.json`
