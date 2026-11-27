# Summary: T14 phase 1 — schema, and what the audit found

## Delivered (phase 1)

Migration `000043` adds `tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0` to the 18 extension tables migration `000041` did not cover, and rebuilds five natural unique keys tenant-first so two tenants may reuse a name. Existing rows keep `tenant_id = 0`, so compat/platform data and behaviour are unchanged. The 18 GORM models carry the matching field and key tags, so a fresh install (AutoMigrate) and an upgrade (SQL) describe the same schema.

| Domain        | Tables                                                                                                              |
| ------------- | ------------------------------------------------------------------------------------------------------------------- |
| ArgoCD        | `biz_argocd_server`, `_repository`, `_application`, `_sync_history`, `biz_cicd_webhook_event`                       |
| K8s           | `biz_k8s_helm_release`, `biz_k8s_install`, `biz_k8s_release`                                                        |
| Observability | `biz_observability_target`, `obs_alert_rule`, `obs_alert_event`, `obs_notification_channel`, `obs_notification_log` |
| Filedist      | `filedist_storage`, `_file`, `_target`, `_task`                                                                     |
| Service       | `biz_application`                                                                                                   |

Keys rebuilt as `(tenant_id, …)`: `biz_argocd_server.name`, `filedist_storage.name`, `obs_alert_rule.name`, `obs_notification_channel.name`, `biz_k8s_release.idempotency_key`. Soft-delete semantics are unchanged (no `deleted_at` added to the keys, exactly like the keys they replace).

The old keys were created by AutoMigrate without a stable name, so the migration finds them by column set instead of guessing an index name; a test uses a deliberately oddly named legacy key.

## Findings

1. **The T01 table list was wrong.** `biz_preflight_control` is a test-only synthetic table; `biz_application` (also missed by T04), `obs_alert_rule/event` and `obs_notification_channel/log` were absent. The real count is 18, not 14.
2. **T04 delivered schema only.** No business code uses `tenant.Scoper` or `TenantOwnerID`, and cmdb/deploy/k8s/cicd/filedist still mount the legacy `DataScopedGroup`. Outside observability, `tenant_id` is not enforced on reads or writes; in multi-tenant mode rows are written with the default 0 and read unfiltered. **A column without enforcement is not isolation.** The T04 closeout deferred the write path to T14, and it is not done.
3. **Alert evaluation skips tenant datasources.** `alert_evaluator.go` ignores any datasource with `TenantID != 0`, so alerts only work for compat data.
4. **`000042` was never assembled.** T05's migration was missing from `businessMigrations`, so it was not copied into the product tree. Registered now; AutoMigrate had been covering the column, so the practical impact was small.
5. **Boundary violation from T05.** `deploy_credential.go:59` uses `bizscope.BizScope` directly. `check:business-boundaries` is manual-only in CI, so required checks did not catch it. Not fixed here.

## Not delivered (phase 2)

Read scoping and write ownership in the extension services, switching their route groups to `TenantScopedBusinessGroup` (which is fail-closed and needs permission routes registered for every cicd/filedist route), and threading tenant identity into the six background entry points (ArgoCD reconciler, filedist lifecycle and transfer, alert evaluator, k8s backup and release). The same gap exists for the core modules T04 covered, so doing it only for the extensions would leave multi-tenant mode half-enforced.

## Phase-1 closeout

- T14 phase 1 is closed at merge commit `81b8089efb943f7c3d20532eb766eb2c9e8764ef` via PR #225. The PR's hosted Backend Tests job passed with the MySQL-enabled gate in 6m36s; Smoke Sanity, Go Lint, Quality Gates, Docs Governance and security checks passed. The non-required Automate Solo PR job remained red due to the absent `RELEASE_GATE_TOKEN`; it did not block the merge.
- The real-MySQL migration suite passed 5/5 on the pre-merge candidate, covering legacy-key upgrades, arbitrary legacy index names, AutoMigrate-first recovery, and T04 compatibility. PR #225 then passed the MySQL-backed required Backend Tests job.
- An independent CODE/ARCH review of migration `000043`, model tags, and migration tests found no significant issues.
- **Scope boundary:** this closeout covers schema ownership only. It does not claim tenant isolation is enforced at runtime: read/write scoping, tenant-scoped routes, async tenant identity, callbacks, alert evaluation and file distribution remain assigned to `2026-11-25-ops-tenant-runtime-enforcement` and its module tasks.
- Local full business-package tests had one unrelated Redis-dependent observability test fail because the local Redis service was unavailable; the hosted PR Smoke Sanity and Backend Tests jobs passed.
- The credential capability boundary finding in `deploy_credential.go` belongs to T05 and remains tracked separately.
