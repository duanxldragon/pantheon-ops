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

The rebuild is the part that matters for durability: before the declaration the assembly produced
758 files and no Sonar configuration, after it the assembly produces 761 files including
`sonar-project.properties`.

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
