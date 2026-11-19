# E01/E02 independent review — first candidate

Verdict: REQUEST CHANGES. Reviewed at 2026-11-18T07:09:23Z by /root/e01_e02_independent_review. Fourteen bound files, two HIGH and one MEDIUM. Root persists the read-only reviewer's returned report; the adjacent JSON preserves its original fourteen SHA256 values.

E01-R1 HIGH: claimTarget turns transaction errors into false. After a worker consumes a queue entry its caller returns; the target remains Pending/Retrying under a Running parent while the dispatcher only scans Pending parents. Query, lock or commit failure can strand work until restart. Return (claimed,error), prevent remote IO on failure and persist guarded terminal failure with bounded cancellable calls and sanitized evidence. Add query/ambiguous-commit, persistence-failure and terminal-cancellation regressions.

E02-R1 HIGH: reconcileServer returns when /api/version fails before scanning applications. A persistent outage therefore prevents the ten-minute Unknown timeout. Existing direct reconcileApplication tests bypass this path. Expire aged local pending history independently of provider-probe success, preserving source/current-history transactional fences and visible outage evidence. Regress via reconcile or reconcileServer.

E01-R2 MEDIUM: the dispatcher discards dispatchDueTasks errors. Failed schedule discovery retries without application evidence. Keep bounded retry; record sanitized failure class/subsystem and recovery without SQL or credentials.

Independent gopls check on fourteen files exited 0 without diagnostics. Native lsp_diagnostics, ast_grep and CodeGraph unavailable. Review covered cancellation, CAS, schedule/restart/ownership, SFTP cancellation, sync deduplication, uncertainty and correlation. Reviewer ran no MySQL, full race, browser or live-provider validation.

Architecture boundaries remain: accepted POST plus SQL commit failure is dual-write uncertainty; application row fences do not establish server generation/request-source/remote UID ownership; provider timestamp precision vs local subsecond StartedAt needs runtime evidence; advisory ownership is not SSH fencing; progress goroutines cancel without independent join. Original ownership/legacy/integration/provider/human gaps stay OPEN. This report retains REQUEST CHANGES after later repair.
