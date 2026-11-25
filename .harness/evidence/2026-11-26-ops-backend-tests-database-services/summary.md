# Summary: the Backend Tests gate gets a real MySQL and Redis

## Problem

The required Backend Tests job ran `go test -race ./...` with no `PANTHEON_TEST_DSN`, so every `testmysql` test skipped and read as green: T04's tenant migration tests, T14's, the credential scope tests, async recovery. It also set only the observability-specific Redis address, so the generic `testredis` tests skipped as well. Base's own job already avoids this with a MySQL service and `*_REQUIRED` flags, and documents the trap.

## Change

`backend-tests` now mirrors Base: a MySQL service, a generic Redis service next to the existing observability one, and `PANTHEON_TEST_DSN`, `PANTHEON_TEST_DSN_REQUIRED`, `PANTHEON_TEST_REDIS_ADDR`, `PANTHEON_TEST_REDIS_REQUIRED`. With the REQUIRED flags, a missing DSN or address fails the job instead of skipping. A workflow test pins all of it, and a negative control showed it fails when the guard is removed.

The MySQL service is UTC, not the Asia/Shanghai used elsewhere. The DSN pins `loc=UTC` and the deploy code compares Go-written lease times with `CURRENT_TIMESTAMP(6)` in SQL, so a non-UTC server clock makes a fresh lease look expired. This is why `TestMarkHostResultDefersTerminalTaskUntilActiveAttemptFinishes` failed on a real database: a configuration mismatch, not a product defect.

## What the change exposes

Running the suite with MySQL before the change found that exact failure and nothing else. With a UTC clock (a simulation of the CI service), 26 business packages pass.

## Gaps and risks

- **Duration**: `deploy` took about 25 minutes locally (252 tests, each builds and migrates a database). The hosted time is unknown and is the thing to watch; the job could get noticeably longer.
- **The hosted run is the authoritative result.** The local run is a forecast.
- The Base packages in the tree (auth/login, system/i18n, pkg/database) timed out locally at 6 minutes because this machine is slow per database test; Base's own CI runs them with MySQL and passes.
- Wiring `check:business-boundaries` into CI is a separate gate decision, not made here.
- No independent CODE/ARCH review.
