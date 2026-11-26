# Summary: the Backend Tests gate gets a real MySQL and Redis

## Problem

The required Backend Tests job ran `go test -race ./...` with no `PANTHEON_TEST_DSN`, so every `testmysql` test skipped and read as green: T04's tenant migration tests, T14's, the credential scope tests, async recovery. It also set only the observability-specific Redis address, so the generic `testredis` tests skipped as well. Base's own job already avoids this with a MySQL service and `*_REQUIRED` flags, and documents the trap.

## Change

`backend-tests` now mirrors Base: a MySQL service, a generic Redis service next to the existing observability one, and `PANTHEON_TEST_DSN`, `PANTHEON_TEST_DSN_REQUIRED`, `PANTHEON_TEST_REDIS_ADDR`, `PANTHEON_TEST_REDIS_REQUIRED`. With the REQUIRED flags, a missing DSN or address fails the job instead of skipping. A workflow test pins all of it, and a negative control showed it fails when the guard is removed.

The MySQL service is UTC, not the Asia/Shanghai used elsewhere. The DSN pins `loc=UTC` and the deploy code compares Go-written lease times with `CURRENT_TIMESTAMP(6)` in SQL, so a non-UTC server clock makes a fresh lease look expired. This is why `TestMarkHostResultDefersTerminalTaskUntilActiveAttemptFinishes` failed on a real database: a configuration mismatch, not a product defect.

## What the change exposes

Running the suite with MySQL before the change found that exact failure and nothing else. With a UTC clock (a simulation of the CI service), 26 business packages pass.

## Closeout and remaining limits

- Task status is `completed` in the closeout worktree. The service configuration has passed on `main`, and the locally verified regression assertions and status changes are ready for integration.
- PR #227 was merged as `08bdcf6fca828553a4c33085874ef03729f88c59`. Its hosted Backend Tests job passed in 7m24s (run [45661674309](https://github.com/duanxldragon/pantheon-ops/actions/runs/45661674309)); a subsequent `main` run passed in 9m28s (run [45727657847](https://github.com/duanxldragon/pantheon-ops/actions/runs/45727657847)). The separate Automate Solo PR check failed because `RELEASE_GATE_TOKEN` was unavailable; required quality checks and Backend Tests passed, and no required-check policy was changed.
- The independent CODE/ARCH review found that the regression test did not pin the generic Redis service or the UTC settings. Follow-up reviews found the UTC assertion was not scoped to MySQL and the Redis address was neither exact nor scoped to the job env. The test now checks `TZ: UTC` inside `mysql.env`, pins `127.0.0.1:6379` inside the job `env`, and includes negative controls for misplaced/wrong Redis configuration. `node --test tests/scripts/quality-workflow.test.mjs` passes all 14 tests; final independent review found no remaining issues.
- A fresh run of the workflow and dispatch regression suites passed all 19 tests. The single-task packet, evidence, and review checks passed, and the regenerated index matched the manifest.
- The assertion hardening and evidence are recorded on `chore/backend-tests-closeout-20261126` for integration separately from PR #227.
- Windows CGO go test -race ./... passed against isolated UTC MySQL and Redis on 81b8089e. 77 packages passed and 14 reported no test files. Deploy took 1434.715s. The earlier ThreadSanitizer allocation error 87 did not recur. Log SHA-256 08a2f49c3f543ef6619894b957c8a87d567463cd8598eeeaf11f2df1449809b8.
- Local `deploy` tests took about 25 minutes. Hosted Backend Tests took 7m24s on PR #227 and 9m28s on a later main run; continue monitoring duration.
- Wiring `check:business-boundaries` into CI is a separate gate decision and remains out of scope.
