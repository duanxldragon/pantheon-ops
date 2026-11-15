**Architectural status: `BLOCK` — one HIGH finding.** O2 v3’s persisted ownership protocol is supported by the inspected code and regression evidence, but ServiceInstance completion can commit independently before deployment/CMDB rollback. The disclosed limitation is a concrete partial-side-effect path.

### Finding and root cause

**HIGH — ServiceInstance completion escapes the fenced transaction.**

Owned completion invokes `finishServiceInstanceState` before the outer transaction returns ([attempt_ownership.go:875](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership.go:875)). Manual reporting has the same ordering ([deploy_service.go:2221](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2221)).

The transaction service copies the original `serviceState` unchanged ([target_selection.go:167](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/target_selection.go:167)). Its capability accepts context but provides no transaction binding ([capability.go:138](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/capability/capability.go:138)); the actual Service owner updates through its own `m.db` ([state.go:75](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/service/state.go:75)).

Consequently, ServiceInstance state/version can persist, followed by an ownership/deadline refusal or failed outer commit that rolls back task, attempt, host, CMDB and release changes ([attempt_ownership.go:151](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership.go:151)). Uninstall introduces an additional window between two independent state commands ([deploy_service.go:2383](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2383)).

This diagnosis is source-grounded; I did not execute a new fault injection. The existing rollback regression injects failure at CMDB, before ServiceInstance completion, and its fixture has no ServiceInstance linkage ([attempt_ownership_test.go:98](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:98), [701](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:701)). Its green result therefore does not cover this window.

The [handoff limitation:43](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-engine-ownership-20261110/handoff.md:43) discloses the gap; it does not approve an exception to [v2’s atomic finish requirement:19](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-engine-ownership-plan-v2.md:19).

### Supported architectural conclusions

- **Admission and exclusion:** nonlocking discovery precedes task/host/attempt/physical locks and exact-footprint validation ([deploy_service.go:1131](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1131), [1283](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1283), [task_preview.go:299](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/task_preview.go:299)). Latest attempt, worker, physical owner and strict DB-clock expiry are checked under persisted locks ([attempt_ownership.go:314](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership.go:314)).
- **Renewal and stale closures:** both leases renew atomically; pure renewal never subsequently locks task. The independent watchdog and latched failure prevent late-response resurrection ([attempt_ownership.go:422](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership.go:422), [495](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership.go:495)). Fresh stale-token callbacks are checked against snapshots of nine deploy/CMDB tables ([attempt_ownership_test.go:228](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:228)).
- **Projection and lifecycle:** outermost READ COMMITTED and unknown-transaction refusal are explicit ([attempt_ownership.go:184](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership.go:184)). Regression evidence covers component merge barriers, queued-own-finish retirement, reversed reconcile winners, frozen DB-clock equality/null and cumulative/free-paused attempt accounting ([attempt_ownership_test.go:400](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:400), [504](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:504), [717](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:717), [810](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:810), [892](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/attempt_ownership_test.go:892)). SSH runs after command-fence transactions return ([task_command.go:135](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/task_command.go:135)).

### Recommendations and trade-offs

1. **Medium effort, HIGH impact:** Dispatcher should authorize the smallest business-domain scope extension: an owner-provided ServiceInstance transaction-binding capability, consumed by `targetTransactionService`. Preserve Service validation and authorization; bind all finish/uninstall/retire mutations to the existing outer transaction.
2. **Small–medium effort:** add linked-ServiceInstance MySQL regressions for failure after owner mutation, outer commit failure, and uninstall’s second-command failure. Assert ServiceInstance and all existing fenced tables roll back together.

| Option                     | Benefit                                    | Cost                                                                          |
| -------------------------- | ------------------------------------------ | ----------------------------------------------------------------------------- |
| Bind owner capability      | Preserves supported behavior and atomicity | Narrow capability/Service scope extension                                     |
| Move callback after commit | Avoids this rollback window                | Creates deployment-success/Service-failure divergence; needs durable recovery |

### Verification and stop condition

I independently rehashed **46/46 deployment files**, verified all **10 final delta hashes**, and checked retained bindings: zero mismatches. Critical SHA256 values match [source-delta-final.json](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-engine-ownership-20261110/source-delta-final.json):

- `attempt_ownership.go`: `1c3073b1a29dcc27f9a49f8e9b89902df01f1a4d92f9757fe85861a48b6e2f0c`
- `deploy_service.go`: `1f60a774c742cfd72d0c3fb15414637e4fa690703c6af5ebf4eaec5706e3287b`

The inspected final log contains **157 passes, two explicit live-VM skips, zero failures**; race/vet/build/gopls receipt is green ([receipt](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/o2-engine-ownership-20261110/stable-v2-focused-receipt.json:12)). No files were written or tests rerun.

ARCH remains blocked pending repair and matching regression evidence. Root-owned acceptance and release gates remain open.
