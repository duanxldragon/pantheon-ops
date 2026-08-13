# SonarCloud Ops Remediation

- **Target repository:** `pantheon-ops`
- **Layer:** `inheritance-sync`, `business/cmdb`, `business/deploy`, `business/bizscope`, repository tooling, and CI security
- **Task mode:** `review` + `implement`
- **Tier:** L2

## Context

SonarCloud's latest `pantheon-ops` analysis is revision `69c59f8151f3f0c856ab0897af6717c0a475a4e5` (2026-08-10), while local `main` is `4604ea48ed01550919f856e39c3029ed024d9b9c`. The frozen baseline contains 486 unresolved findings versus zero on `pantheon-base`.

## In Scope

- Align automatic-analysis exclusions with `pantheon-base` for non-product tooling, SQL seed/migrations, and i18n resource maps.
- Remediate confirmed executable defects and security findings still in scope.
- Classify remaining findings by business ownership, locked foundation source, and missed base consumption.
- Repair generic shared-source omissions in `pantheon-base` first; consume them through a foundation release rather than copying files into ops.

## Out of Scope

- Business feature or schema redesign.
- Bulk SonarCloud `WONTFIX` transitions.
- Local overrides of `platform` or `system/*` behavior.

## Sync Expectation

- Current foundation release: `pantheon-base-v0.10.12` at `16918771e2650f8c045b0e086144eb290e774704`.
- Keep `business/*` code local. Any generic frontend source absent from `foundation-release.lock.json` must be classified and routed to `pantheon-base` before an ops release upgrade.

## Verification

- Run `npm run check:inheritance` and the focused affected tests/builds.
- Run the existing base-drift triage and compare post-analysis SonarCloud snapshots against the raw baseline under `.harness/evidence/2026-08-11-sonarcloud-ops-remediation/`.
- Treat hosted analysis of the candidate SHA as final; no current SonarCloud result is considered final until it analyzes `4604ea48ed01550919f856e39c3029ed024d9b9c` or its successor.

## Stop Points

- Stop before changing a foundation version, deleting an inherited overlay, changing a business schema, or carrying a generic shared fix locally.
