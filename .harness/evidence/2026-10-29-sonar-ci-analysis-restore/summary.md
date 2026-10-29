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

`/api/project_analyses/search?project=duanxldragon_pantheon-ops&branch=main` now returns four
analyses. The first two are dated `2026-08-10` and `2026-08-22`; the rest are dated
`2026-10-29T09:50:45+0000` at `8799b2e9` and `10:00:28` / `10:02:14` at `d79dc024`. That is the CI
analysis channel moving the main-branch baseline for the first time since August, which is the
acceptance condition for this packet: the credential authenticates, the mandatory properties
resolve from the restored file, the server accepts the report, and the project version on `main`
advances.

After both merges every GitHub Actions workflow on `main` is green: Code Quality Gates, Security
Gates, SonarCloud Scan, Branch Hygiene and Base Compatibility Rebuild all complete with success on
`d79dc024`, dispatched and pushed. One bookkeeping item is worth naming because it looks like a
failure in the history: the push-triggered Code Quality Gates run `40854031854` was cancelled
mid-flight by the concurrent dispatch sharing its concurrency group, which left its rollup red on
the commit; `gh run rerun 40854031854 --failed` re-executed the cancelled jobs and it now concludes
success.

The single remaining red context on `main` is SonarCloud's own quality-gate check, and it is red
because the restored analysis finally measures the code: on the new baseline the gate reads `ERROR`
on `new_security_rating` (E) and `new_reliability_rating` (C), with maintainability, duplication
(1.8% against a 3% threshold) and hotspot review passing. The open backlog is 62 vulnerability
issues, 6 bugs and 174 code smells, concentrated in GORM dynamic-condition taint reported as SQL
Injection across `backend/modules/business/{cmdb,deploy,service,observability,filedist,k8s,cicd}`,
plus two path-traversal findings in `filedist/storage_service.go`, disabled certificate and
hostname verification in `cicd/argocd/argocd_client.go:42`, npm/npx lifecycle-script exposure in
the e2e shell scripts, `pip` without `--only-binary` in `security.yml`, and unhandled promise
rejections in four business list pages. Those numbers replace the stale pre-restoration reading
recorded earlier in this file, which is precisely the blindness this packet removes.

## Known gaps

- Dependabot-triggered runs still cannot read repository secrets, so the scan stays red on
  Dependabot PRs (`empty SONAR_HOST_URL` → `URI with undefined scheme`). Provisioning Dependabot
  secrets or making the job skip that actor is a gate-policy decision for the maintainer and was
  deliberately not taken here.
- The SonarCloud quality gate on `main` reads `ERROR` on `new_security_rating` (E) and
  `new_reliability_rating` (C) against that 62-vulnerability / 6-bug backlog. GitHub Actions stays
  green because the workflow does not fail on a gate `ERROR`; clearing it is business-layer product
  work, and marking findings false-positive, moving the new-code baseline or attaching a laxer gate
  were all declined as maintainer-owned policy decisions.
- `Vercel` remains red and is outside this repository: an external integration with no committed
  configuration, and the Vercel CLI on this host is logged out, so neither its project settings nor
  its build logs are reachable from here.
- The restored file keeps `sonar.sources=.` exactly as the deleted copy did, so the scanner reads
  committed PNG evidence and visual snapshots as text and logs `Invalid character encountered`.
  Those are warnings, not failures; narrowing the source set is a separate decision from restoring
  the channel.
