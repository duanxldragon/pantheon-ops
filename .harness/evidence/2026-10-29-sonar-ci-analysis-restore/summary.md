# SonarCloud CI Analysis Restored for pantheon-ops

## What was broken and why

`SonarCloud Scan` had two independent failures stacked on top of each other.

1. **Credential.** `SONAR_TOKEN` was rejected: every push and normal-PR run died with
   `GET https://api.sonarcloud.io/analysis/jres… failed with HTTP 403 Forbidden`. The
   maintainer supplied a replacement token, it validates against `api/authentication/validate`,
   and it belongs to organization `duanxldragon`, which already provisions
   `duanxldragon_pantheon-ops`. The secret was replaced through `gh secret set`; nothing about the
   token value is recorded in the repository.
2. **Project identity.** With authentication fixed, a fresh dispatch on `main` (run `40849180395`)
   got further and failed differently:
   `You must define the following mandatory properties: sonar.projectKey, sonar.organization`.
   The reason is that `sonar-project.properties` is gone from the tree. It was deleted by
   `5c12964f` — the `pantheon-base-v0.17.7` consumption — because the file is Ops-only (base ships
   just `.sonarcloud.properties`) and had never been declared in `business-overlay.json`, so the
   rebuild had no instruction to keep it.

The credential failure hid the configuration failure for as long as both were broken.

## What changed

- `sonar-project.properties` restored verbatim from `5c12964f^`:
  `sonar.projectKey=duanxldragon_pantheon-ops`, `sonar.organization=duanxldragon`,
  `sonar.sources=.` with the tooling/SQL/i18n-resource/generated-directory exclusions,
  `sonar.coverage.exclusions=**/*` and the test-file `sonar.cpd.exclusions`.
- `business-overlay.json` now declares `sonar-project.properties` in `repositoryOverlayPaths`, so
  the next foundation consumption cannot delete it again.

`.sonarcloud.properties` is untouched and stays declared: it scopes SonarCloud's *automatic*
analysis, whereas the scanner CLI that this workflow runs only reads `sonar-project.properties`.
They are not duplicates.

## Gates

| Gate | Result |
| --- | --- |
| `check:business-overlay` | OK, 53 declared paths |
| `test:business-overlay` | fail 0 |
| `rebuild:from-base` | 761 declared/generated files; rebuilt tree contains the restored file |
| `check:task-packet` on this packet | 1 file, 0 errors, 0 warnings |
| `check:harness-adoption` | 0 findings |
| `check-pr-governance.mjs --event` | OK pantheon-ops PR body governance fields are present |
| SonarCloud authentication | `{"valid":true}`; project and organization resolve for the new token |
| `SonarCloud Scan` on PR #153 | pass, run `40851210564`, log ends `ANALYSIS SUCCESSFUL` |
| `SonarCloud Scan` on `main` | pass, run `40852770827` at `8799b2e9` |
| PR #153 check rollup | green except the external `Vercel` deployment |

The rebuild is the part that matters for durability: before the declaration the assembly produced
758 files and no Sonar configuration, after it the assembly produces 761 files including
`sonar-project.properties`.

## Hosted verification after the merge

`/api/project_analyses/search?project=duanxldragon_pantheon-ops&branch=main` returns three
analyses. The first two are dated `2026-08-10` and `2026-08-22`; the third is dated
`2026-10-29T09:50:45+0000` at revision `8799b2e9`. That is the CI analysis channel moving the
main-branch baseline for the first time since August, which is the acceptance condition for this
packet: the credential authenticates, the mandatory properties resolve from the restored file, the
server accepts the report, and the project version on `main` advances.

## Known gaps

- Dependabot-triggered runs still cannot read repository secrets, so the scan stays red on
  Dependabot PRs (`empty SONAR_HOST_URL` → `URI with undefined scheme`). Provisioning Dependabot
  secrets or making the job skip that actor is a gate-policy decision for the maintainer and was
  deliberately not taken here.
- The SonarCloud quality gate for the project currently reads `ERROR` on `new_security_rating`,
  with 4 vulnerabilities and 245 code smells recorded. The workflow does not fail on a gate
  `ERROR`, so the pipeline is not reddened by it, but that is the real backlog the scan now
  surfaces again.
- `Vercel` remains red and is outside this repository: an external integration with no committed
  configuration.
- The restored file keeps `sonar.sources=.` exactly as the deleted copy did, so the scanner reads
  committed PNG evidence and visual snapshots as text and logs `Invalid character encountered`.
  Those are warnings, not failures; narrowing the source set is a separate decision from restoring
  the channel.
