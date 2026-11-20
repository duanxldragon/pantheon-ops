# E02 package-local implementation handoff

Original E02 remains OPEN. This continuation repairs accepted-sync versus observed-deployment outcome in business/cicd/argocd. Root owns the ten-package manifest, startup integration, shared/locale changes, disposable-MySQL validation, independent review, complete race and final acceptance. No schema, provider credentials, runtime restart, live ArgoCD, production or publication operation was performed by this executor.

## Implemented behavior

- Accepted HTTP200 sync remains Running/Progressing with unfinished original history and no successful-sync timestamp. Same pending revision reuses its history without another POST; a different pending revision is rejected.
- Explicit HTTP4xx rejection returns a stable failure key and records Failed. Transport or HTTP5xx uncertainty returns an error but keeps the request Running so a later observation can determine whether the remote accepted it. HTTP work has a 10-second budget inside the existing 30-second DB transaction budget, leaving time to persist the result.
- Aggregate success requires operation Succeeded, configuration Synced and health Healthy. Succeeded with Progressing stays unfinished; Succeeded with Degraded/Missing fails. Running/Terminating stays unfinished even if old health is degraded. Public/persisted request messages are stable keys and exclude provider bodies.
- Refresh and background reconciliation share this rule. Reconciliation finishes the original pending history once, without periodic synthetic successful rows. Operation date and requested revision correlate pending requests conservatively; old/undated/mismatched operations do not finish newer local work.
- Before remote I/O snapshot application/source/latest history; under an application row lock recheck source, UpdatedAt and history ID. Changed source or superseded requests discard the old observation. Pending operations expire after 10 minutes as Unknown; remote404 becomes Missing/Failed.
- Immediate startup reconciliation, disconnected-server retry, bounded keyset batches, scoped observation/probe timeouts and idempotent Stop with loop cancellation are implemented. Existing StartAll(ctx, db) remains the root startup hook.

## Fresh verification

Six pure-test groups passed with Windows MinGW, CGO=1, readonly modules and race: package 1.112s, exit0. They cover success/health combinations, Running on old degraded health, revision/date correlation, stopped/canceled startup, repeated Stop, HTTP404 versus outage and rejected versus uncertain sync responses. gopls check for all six changed files returned exit0/no diagnostics. Scoped git diff --check returned exit0.

The original Stop double-close defect has an actual earlier RED panic and GREEN receipt. The unhealthy business RED was not executed by this worker; preserved pre-change bytes allow root to reproduce it safely with an isolated overlay. Database integration tests are added and handed to root; no unconfigured SKIP is accepted as evidence. See continuation-sync-observation-commands-20261118.json and continuation-sync-observation-source-binding-20261118.json for exact filters, receipts and hashes.

## Integration requirements and open boundaries

1. Root should wire the existing argocd.StartAll(ctx, db) into application-owned startup/lifecycle and verify cancellation. No opsinit file was edited here.
2. Canonical new keys needing root inventory/locales/handler mapping: business.cicd.application.syncInProgress, business.cicd.application.syncObservationTimeout and business.cicd.application.sourceChanged. Existing cicd.argocd.app_sync_failed is reused. UI readiness must use aggregate status; raw syncStatus=Synced can coexist with Error/Degraded.
3. Existing application/server/history models lack ownership, remote UID and persisted operation token. This patch cannot establish original ownership/legacy isolation or safe identity after remote recreation. Server configuration changes are not fenced by a persisted server generation. Root must keep those E02 gates open.
4. Operation timestamp correlation requires comparable clocks and sufficiently precise remote startedAt. Clock skew/second precision, branch-to-SHA formats and missing requested revision can conservatively delay completion until Unknown. An operation token is needed for stronger guarantees; current timestamps/UpdatedAt are not absolute concurrency tokens.
5. The application row lock spans bounded HTTP submission to serialize duplicate admission. Remote acceptance followed by transaction/DB commit failure remains an uncertain dual-write boundary; no durable outbox or idempotency token was introduced.
6. The existing unsigned ArgoCD webhook updates raw sync/health by application name alone. It does not finish history or aggregate status, but authentication, owner/server correlation and verified observation routing remain original E02 security gaps. This scoped repair leaves the webhook untouched.
7. DB observation/business tests, live provider evidence (including ImagePullBackOff), original permission/legacy isolation, independent review and final acceptance remain unverified by this executor. Do not archive or mark the complete E02 work package done from this handoff.
