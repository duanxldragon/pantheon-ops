# F-01 QA Acceptance Matrix

Task: `2026-08-19-ops-v1-backlog-closeout`

## P0

| QA ID | Status | Evidence / disposition |
|---|---|---|
| P0-DB-01 | pass | MySQL 8 isolated migration and runtime schema tests |
| P0-DB-02 | pass | Legacy upgrade and repeat-run tests |
| P0-DB-03 | pass | Dirty/current-schema recovery tests; production failure injection remains gated |
| P0-DB-04 | pass | Backend artifact, frontend dist, overlay rebuild and checksums |
| P0-AUTH-01 | pass | Capability and business module tests; live cross-scope mutation remains gated |
| P0-AUTH-02 | pass | DataScope/boundary regression tests |
| P0-AUTH-03 | deferred | Requires concurrent live ownership fixture and maintainer-selected environment |
| P0-BOUNDARY-01 | pass | Boundary checker and four regression cases |
| P0-RELEASE-01 | pass | Deploy idempotency/start/callback targeted tests |
| P0-RELEASE-02 | pass | Lease reconciliation, retry and worker recovery tests |

## P1

| QA area | Status | Evidence / disposition |
|---|---|---|
| CMDB/BizScope/Service binding | pass | Business package tests and boundary contracts |
| Deploy async lifecycle | pass | Full Go suite, race gate, authenticated Deploy page smoke |
| K8s ownership/credential/conflict | pass with gate | Unit/contract/read-only cluster evidence; write mutation requires isolated cluster approval |
| Pagination and import/export | pass | Backend tests, frontend build and list-surface smoke |
| Browser responsive rendering | pass | Desktop screenshots plus 390x844 mobile screenshot, no horizontal overflow |
| Production migration duplicate/index rehearsal | deferred | Requires production snapshot and maintainer approval |
| SSH fingerprint/credential mutation | deferred | No real SSH write was attempted by policy |
