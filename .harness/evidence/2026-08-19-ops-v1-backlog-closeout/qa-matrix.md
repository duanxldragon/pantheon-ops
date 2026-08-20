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
| P0-AUTH-03 | pass | MySQL concurrency regression: ownership move holds cluster lock, updates scope/dept atomically, stale scoped writer receives `k8s.cluster.not_found` |
| P0-BOUNDARY-01 | pass | Boundary checker and four regression cases |
| P0-RELEASE-01 | pass | Deploy idempotency/start/callback targeted tests |
| P0-RELEASE-02 | pass | Lease reconciliation, retry and worker recovery tests |

## P1

| QA area | Status | Evidence / disposition |
|---|---|---|
| CMDB/BizScope/Service binding | pass | Business package tests and boundary contracts |
| Deploy async lifecycle | pass | Full Go suite, race gate, authenticated Deploy page smoke |
| K8s ownership/credential/conflict | pass | Unit/contract plus isolated cluster ConfigMap/Secret writes; stale resourceVersion rejected with Conflict and namespace cleaned |
| Pagination and import/export | pass | Backend tests, frontend build and list-surface smoke |
| Browser responsive rendering | pass | Desktop screenshots plus 390x844 mobile screenshot, no horizontal overflow |
| Production migration duplicate/index rehearsal | pass | Isolated full snapshot restored; active-key duplicates=0, generated columns/indexes verified; `22 -> 16 -> 22` reapply preserved row counts |
| SSH fingerprint/credential mutation | pass | Isolated host write/read/hash/delete completed with verified ED25519 host key; no remote residue |

## Harness adoption

| QA area | Status | Evidence / disposition |
|---|---|---|
| Clean Base overlay Harness method/adoption | pass | Rebuilt tree from locked Base snapshot has 0 strict method findings and 0 strict adoption findings (one no-changed-files warning) |
| Consumer-root Harness strict checks | blocked external | Locked `pantheon-base-v0.10.22` predates the current shell landing contract; Base commit `303c6bdb` has 156 pre-existing unresolved SonarCloud issues and its Release Gate is red. No copied fork was added to Ops. |
