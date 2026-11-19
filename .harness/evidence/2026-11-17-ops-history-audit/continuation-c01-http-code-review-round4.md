# C01 HTTP correction — independent code review round 4

Reviewer: `/root/history_scope_review` (`code-reviewer`). Coordinator-persisted synopsis of the completed independent report; no author self-review substituted.

Recommendation: **COMMENT**. The two round-3 MEDIUM findings are closed; no open findings in that correction. Cumulative files reviewed: 23. Four relevant files passed `gopls check`; named LSP/AST tools were unavailable.

- `service_link.go` maps only the documented Service not-found keys to invalid association. Unexpected owner errors retain their cause.
- `deploy_handler.go` classifies invalid association as 400, stale owner lifecycle as 409, infrastructure failures as 500, and binds manual reports to request context.
- The preserved HTTP red run failed as expected. `continuation-c01-http-cause-fixed.log.receipt.json` records real MySQL, CGO=1, race exit 0 covering ten admission mismatch cases, cause preservation across create/update/start, and handler classification.

Inherited Retire command/projection acceptance remains open: generic and fixed execution paths select installation for Retire; empty manual result summary writes an installation component while Service records retired. This is source evidence, not an SSH reproduction. Repair the lifecycle contract rather than changing only the component-removal projection. See [owner gap](./continuation-retire-lifecycle-owner-gap.md).

The later complete run [receipt](./continuation-c01-full-race-http-final.log.receipt.json) exited 1. The independent report predates that result; it does not approve those failures, C01, C02, or project completion.
