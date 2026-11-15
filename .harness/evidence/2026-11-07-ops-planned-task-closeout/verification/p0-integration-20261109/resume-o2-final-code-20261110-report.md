## Code Review Summary

**Recommendation: REQUEST CHANGES**

**Files reviewed:** 63 O2 source/test files, plus two runtime locale resources.  
**Issues:** 6 — CRITICAL: 0, HIGH: 3, MEDIUM: 1, LOW: 2.

This is an independent, read-only **business/deploy CODE/spec/security review**. O2/P0 remain open. Findings below distinguish newly introduced behavior from existing engine/security defects.

### Findings

**[HIGH] 1. Mixed-scope tasks expose hidden hosts’ logs**

Files: [deploy_handler.go:776](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_handler.go:776), [deploy_service.go:2072](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:2072).

`taskVisible` authorizes a task when **any** task host is visible. `ListTaskLogs` then queries logs using the task ID and caller-supplied filters, without constraining results to CMDB-authorized hosts.

A user who can see host A can retrieve host B’s output by omitting `hostId` or explicitly requesting B. `firstError=1` can similarly return B’s stderr. The new `GetTask` host/history filtering does not protect this endpoint.

**Fix:** Resolve authorized task-host IDs and constrain the database query before pagination and the first-error limit. Reject explicit hidden-host selections consistently. Add mixed-visibility regressions covering unfiltered logs, hidden-host filters and first-error retrieval.

**Evidence:** Direct authorization/query trace. `TestWorkbenchLogSourceIsolationAndFirstError` covers administrator filtering and complete task denial, but not one visible plus one hidden host. The underlying task-level exposure predates this change; the changed log provider retains it.

---

**[HIGH] 2. An established SSE stream continues after visibility is revoked**

File: [deploy_handler.go:852](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_handler.go:852).

`StreamTaskEvents` checks visibility once at line 833. Every subsequent replay/poll batch loads and emits events without another scope check.

After a CMDB ownership/department change removes the viewer’s task visibility, ordinary HTTP detail requests can be denied while the existing stream continues receiving event identifiers, statuses and reasons.

**Fix:** Revalidate current task/host visibility before each emitted batch, constrain host-specific events, and terminate with explicit denial behavior when access disappears. Add a test that revokes visibility after opening the stream.

**Evidence:** Direct handler trace. This is an existing endpoint defect. The VM receipt’s event cursor checks use REST reads; they do not establish active SSE revocation safety.

---

**[HIGH] 3. Attempt leases expire while supported long commands can still be running**

File: [deploy_service.go:1624](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/deploy/deploy_service.go:1624).

Attempts receive a fixed five-minute lease with no renewal. Execution allows 30 minutes by default and up to 24 hours. Reconciliation marks expired running attempts retryable at line 1661 without stopping their workers.

A second service process starting after minute five can reconcile and claim the same host while the original SSH command remains active:

- The worker guard at line 1358 is process-local.
- The host lease accepts the same task/owner at line 2903.
- Attempt admission excludes expired claims at line 1632.
- Completion at line 1712 checks only attempt ID/worker ID, without current ownership fencing.

This permits overlapping side effects and stale checkpoint/result writes.

**Fix:** Preserve exclusive ownership throughout supported execution, cancel work when ownership is lost, and fence stale checkpoint/result writes. Add a two-service regression with a blocked live executor crossing lease expiry.

**Evidence:** Source trace establishes the overlap condition; no live-worker reproduction was run. The cold-restart receipt deliberately crashes the previous process and tests abandoned-claim recovery, so it does not cover this condition.

**Ownership:** This is an existing engine defect. The original O2 packet excludes changing engine leases/recovery. Escalate this implementation evidence to the independent architect/engine owner; this report does not issue an architectural design verdict.

---

**[MEDIUM] 4. Failure navigation selects the wrong attempt and loses the failed step**

File: [DeployExecutionWorkbench.tsx:84](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/src/modules/business/deploy/task/DeployExecutionWorkbench.tsx:84).

The failure button takes the maximum attempt number across all host steps, then searches for failed steps only at that number. Backend checkpoint counters are **per step**: successful checkpoints are skipped on retry, and only executed checkpoints increment.

For a valid state with `prepare: success, attempt 2` and `install: failed, attempt 1`, the actual component generated:

```text
host=11&attempt=2&stream=stderr&firstError=1
```

It omitted the failed step and requested the wrong attempt, hiding the actual failure. This violates original acceptance criterion 1.

**Fix:** Select the actual failed current checkpoint first, then use that checkpoint’s ID and attempt number. Review the default maximum-attempt selection for the same assumption. Add a regression with unequal checkpoint attempts.

**Evidence:** Reproduced by executing the current component through in-memory transpilation with mocked dependencies. Existing tests use aligned step attempt numbers.

---

**[LOW] 5. New cursor regression test has a TypeScript error**

File: [deploy-log-cursor.test.tsx:92](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/tests/unit/business/operational/deploy-log-cursor.test.tsx:92).

`initialProps` infers a required `stepId: undefined`, while subsequent `rerender` values omit it or supply a number. Native TypeScript diagnostics report **TS2345**.

**Fix:** Explicitly type/widen the hook props and `initialProps` consistently.

**Evidence:** Reconfirmed with native TypeScript diagnostics. Passing transpiled unit tests does not resolve this type error.

---

**[LOW] 6. Existing smoke fixture type omits `remark`**

File: [deploy-pages.spec.ts:689](/D:/workspace/go/pantheon-platform/pantheon-ops/frontend/tests/smoke/business/deploy/deploy-pages.spec.ts:689).

The fixture accesses `TaskRow.remark`, but the type does not declare it. Native TypeScript diagnostics report **TS2339**.

**Fix:** Add the appropriate optional `remark` field to the fixture DTO type.

**Evidence:** Reconfirmed diagnostic. This error predates the current change and is reported separately from new regressions.

### Spec compliance and evidence boundaries

The original [O2 packet](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/tasks/2026-08-31-ops-operational-workbench-design/packets/O2-deployment-execution-workbench.md) was reconstructed before quality review.

The [completed VM receipt](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-o2-v9-matrix-result-1815605de83a.json) records **12/12 cases, 152 positive checks, 31 negative checks, zero failures, 670.329 seconds, `sourceStable=true`**, with exact-owned cleanup.

It supports the executed success/retry/pause/resume/cancel/concurrency scenarios, credential revocation/rotation, host/source revocation, invalid-start/host-key rejection, safe persisted projections and abandoned-lease cold recovery. It does **not** close complete original acceptance.

| Original criterion                                                          | Review result                                                                                                                                                             |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Locate host + step + first error within two interactions                    | Implemented path fails valid unequal-attempt states: finding 4.                                                                                                           |
| Preserve content and resume SSE from last sequence; explicit stale fallback | Adapter/unit and REST cursor evidence exists. Actual browser SSE disconnect/reconnect/revocation matrix remains unverified.                                               |
| Actions match backend state, permission, DataScope and version              | Version/CAS and selected execution revocation scenarios have evidence. Mixed-scope/stream defects remain; full live IAM/session/role/data-grant acceptance is unverified. |
| Mobile first screen prioritizes status/failure; unobstructed switching      | Selected desktop/mobile history renders were inspected. Complete first-screen, action and forbidden-state matrix remains unverified.                                      |
| Multi-host/live/long-log layout remains stable and bounded                  | Provider/window/viewer bounds of 500/1000/500 exist. Complete multi-host, live layout, zoom and reduced-motion evidence remains unverified.                               |

Complete audit/parameter drawer and recovery-entry acceptance is also not established. Official Base UI/permission consumption and final human acceptance remain downstream gates; they are not counted as source defects.

### Validation

- Native `gopls` checks completed for all **28 changed Go files**, plus five supporting Go files: no type errors.
- Application TypeScript check passed.
- Native diagnostics covered **21 changed TS/TSX files**, including tests; the two errors above remain.
- Six relevant frontend unit files passed: **43/43 tests**.
- Three safe Go tests passed under Windows CGO race: inline-secret rejection, execution cancellation and timeout bounds.
- Scoped diff whitespace checks passed.
- No additional newly introduced masking fallback, obvious hardcoded production secret, injection or XSS defect was established.

`impeccable` was applied, and existing desktop/mobile light/dark history screenshots were inspected. These are historical renders with explicit transport fixtures, not fresh complete UI acceptance.

Prior full-race results remain historical. Final stable-tree race belongs to root after active lanes finish. No product, driver, lock or ledger edits, credential-config reads, VM tests or delegation were performed.

### Source inventory and hashes

Final comparison found **185/185 sealed compiled-source paths matching current files** and **zero drift across the 63-file review inventory**.

| Artifact                                                                                                                                | SHA256                                                             |
| --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| [Reviewed request](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-o2-v9-root-reviewed-phase-request.json) | `807c967f6ce2993366c5cf76573e939bff43dd2e1c6bdcc438035e18e215eada` |
| VM receipt                                                                                                                              | `c3d169b12e82f0e6a60952393a104522be4d92e9d9138c7a08f7951ab529d043` |
| Canonical 63-file inventory                                                                                                             | `be955e8fc531e6c3f6651a39a3961f94d9ca038c049e336f134bdd58cf28fe0e` |
| Supplemental runtime `business/en-US.ts`                                                                                                | `61a8b61b5d42bc4fad887f47be1a8361fd32dbb51a21f0f12a91786df273cc98` |
| Supplemental runtime `business/zh-CN.ts`                                                                                                | `3d16834af35386c85557ba14507027dc7ac27726b95b09435775572ee93b47a1` |

The seven queried locale keys exist in both runtime resources; they are not findings.

<details>
<summary>Complete 63-file review inventory</summary>

Canonical digest format: repository-relative paths sorted ordinally, each `path SHA256\n`, UTF-8.

```text
backend/modules/business/bizscope/bizscope_service.go 4654f1b9077ccde03fc717223a63e2acfed0df8771cbeeca71a4d2f5027e9999
backend/modules/business/cmdb/capability.go 87da5f9bde5e3f69022617cf6c92b7e4a4c0bdfee106a6876e511c925c16f566
backend/modules/business/cmdb/target_selection.go c457445fc08b607e0dc83477a00343cb994c503084b472f1b30c8e8e57591cff
backend/modules/business/cmdb/target_selection_test.go 4342665a326db22c269a885981f16841920549f2c5af8eba80e3c5efccd7e846
backend/modules/business/deploy/deploy.go 2496bbe833477eb317d1cd999120cf7bbdf29fbe4db32e7a3965eb74efb66ce7
backend/modules/business/deploy/deploy_credential.go bdcf448935fd8aab487539d6dbb29d818419e59c99fc6e51b08c8e2bc843a540
backend/modules/business/deploy/deploy_dto.go eababd208f853cfcbd92be84a89633104dcf30ebf34c2152ee2ad480d2880a58
backend/modules/business/deploy/deploy_executor_test.go 9e26f4638b57384d26d65cde9771f99ca915bd91da693153a1fa5c35e3841194
backend/modules/business/deploy/deploy_fixed_templates.go e934086958e0c25f1d2844876df4bbc0985e0927dd83b2a7b76c0e6255d4caa0
backend/modules/business/deploy/deploy_handler.go 36c81d49130dc181de52326d6a03b119db700229ab3608a9639b23fcc6ac6fff
backend/modules/business/deploy/deploy_helpers.go 5b0388796b15e221202b94a5e0c686b10551764b5fe6c6166cf3e6e6f023e00f
backend/modules/business/deploy/deploy_live_acceptance_test.go 20f650c74d5ae83fde928ab6ad669990513086baab37132b526faa748f03a99f
backend/modules/business/deploy/deploy_model.go 5ab24f9e9674f809328d3a128a53c24d7dd4d1e19c2ddd9baf80f8ab08e22530
backend/modules/business/deploy/deploy_reliability_test.go 58ff415824ee9810596302b61e593d148b2b2fd52ae405be8dbe9c2bd6612c63
backend/modules/business/deploy/deploy_rollback.go cb054c1930b8afa125d24f6ca78c1cb323dbc72e0d5c59eb487e4a9bd306d3e3
backend/modules/business/deploy/deploy_seed.go ed4ed1585112aa2c0b0f06687ae2624e58324538ca1f2e79e2030be2f0df901d
backend/modules/business/deploy/deploy_service.go 8ea2ee60bb99d9a2577bf342ac7e826953670db04e4aa3c8cfb399383f7efedc
backend/modules/business/deploy/deploy_service_test.go 62eb9dab66526067912e160f08f4058fae6f46011cf87252303e8f4943db2106
backend/modules/business/deploy/deploy_ssh_transport_test.go 18d28e745fe711a40e02d213abbcf9d7fdd5ed89403fe1c1faefbb0de8df0302
backend/modules/business/deploy/log_attempt_migration_test.go da6e67c00bf545e1a60b0a5b7fdc3fcba28f51993be18b511b53f54a50c3c0cb
backend/modules/business/deploy/log_sequence_test.go a01a8bf86a2e3dd4639b7c3dedec3dbba5099fb06e5d3a9d2f03ca784200080f
backend/modules/business/deploy/migrations/000034_deploy_log_attempt.down.sql 79edb509a17da7461697d2567803b77dd08ad2ba139c1055d05da4f336590a37
backend/modules/business/deploy/migrations/000034_deploy_log_attempt.up.sql f175e065722f1ce721afa2cd7203ca8a71fde7dbebb46c5cca0fd19abd3d9342
backend/modules/business/deploy/migrations/000035_deploy_step_history.down.sql 6aeb009b5826e53e0d8b063c09fa0e5c65f9c3d04947375348ade9809c618840
backend/modules/business/deploy/migrations/000035_deploy_step_history.up.sql 1abf23eb1549d5929adc732bc1a35865c81494693efa468f6ef5d9b64b411cfa
backend/modules/business/deploy/migrations/000036_deploy_log_sequence_scope.down.sql 5cc784a06605b9abb174dedbd812fddbdf575b8347635db9bccff52a44952545
backend/modules/business/deploy/migrations/000036_deploy_log_sequence_scope.up.sql 98d94b65a5cd060010498b8b78277f34851561b9ee8bc25cdafd960bae52c569
backend/modules/business/deploy/resume_scope_revocation_test.go f2f36405292d07b46dec6b641c44d9985adf7c87f8ee3e911fb6f31e2b5feb8e
backend/modules/business/deploy/step_history.go 6bbfbc6249f2afef10e7f2e3ba6bb333b318921ab2af390ccdbec7772a2dc441
backend/modules/business/deploy/step_history_test.go 2059e33519344e3a691d6275056f67fc1ced8d7304d4ce2fb14e0b0e431e50b8
backend/modules/business/deploy/target_selection.go 239ac4b249da0d105234d74ae65a1ca1dfe03f0407b91ed28415709f0583b889
backend/modules/business/deploy/task_command.go e256db4c85400efe8d44cc87e062f486e77221c2daa4e8d8b417b4d3cfc683ac
backend/modules/business/deploy/task_command_test.go 7e885249ea4c078af8d158c6be1adb36a52106f8731c0f2b9601036daab5ef8b
backend/modules/business/deploy/task_preview.go 656255301998f2bbea30041f493d71f50622bbcde74de644eb74f5d06d067c15
backend/modules/business/deploy/task_preview_plan.go 94839e26c1c4b87f71097fd8127e95f79248c45ee18165ee486bfae76e650195
backend/modules/business/deploy/task_preview_test.go 78288f6914f16b531db798a63534053385c5462e6b2efa341771088d7756a2d0
backend/modules/business/deploy/task_rollback_snapshot.go 3d0a60fa0c9f388ab4faccd742bd443cc838bd8386fcce0475a14745a15deb87
backend/modules/business/deploy/task_rollback_snapshot_test.go df3c8671e98998777ba6aaf7dabcb3c0765258d78fed93a3f71ccc89d5942767
backend/modules/business/deploy/workbench_test.go 38ac2ef1ec9ae8b8108faa69cce2189cee4ec6e062c1672b9f16aeee0aa7b77b
frontend/src/modules/business/deploy/api.ts 0d0135cf3bdc9275e969a90ad709d3157c04fee01498965975ace3fc664573e5
frontend/src/modules/business/deploy/deploy.css afb45d3ecf4758374d33447440b9e2b179a7a6b4addb4eb8a3e4f34efd75dff7
frontend/src/modules/business/deploy/locales/en-US.json e231b371fecdf4a08400d7dd541b0ac3cd6c3ac77057160f0d947840350547ce
frontend/src/modules/business/deploy/locales/zh-CN.json 327da559ed5bac215340859b9a41f70411d2da2a453188f54adcaf1b28808b5b
frontend/src/modules/business/deploy/task/DeployExecutionWorkbench.tsx 9356ccd861ba5b10e7f0bf58b58f489d14634be06c40d64cbddf109137f73286
frontend/src/modules/business/deploy/task/DeployTargetSelector.tsx 71a81be72f8ad145d8cadfde97082afe2fa5440336d58d88f2c8197a50d5b131
frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx 1ea8b443bfe32f86a55ab15f3c0adfca44b3071c5456b28920f2942b6567f4f3
frontend/src/modules/business/deploy/task/DeployTaskList.tsx ec62431d4b3bb76f10d8188690ee12f3074c9a0fede2ec60fad1a1cf8ab29faa
frontend/src/modules/business/deploy/task/DeployTaskPreview.tsx 906e357bfbd62a25554778caef66671a3346f1f6e3ecb4774e06e6b3a7b37c8b
frontend/src/modules/business/deploy/task/useDeployTaskLogs.ts 9fc4b84175c54cd43a98dae50784eb23c59a6af87549ba8531f79811908da957
frontend/src/modules/business/deploy/task/useDeployTaskStream.ts 7043751fd26ae00910d41914b7e8b99d441f89e099b8d729cd19911dfabcc227
frontend/src/modules/business/shared/OperationalLogPanel.tsx 26e7fbcbe624ded48f06ddf8f813c1b8b7e4e689504b4741884bf66d8e6b27ab
frontend/tests/smoke/business/deploy/deploy-api.spec.ts 82d233d44085c4f2e8cde472a13a4b4f6aed2ff635ee8565e7a25c197ac17686
frontend/tests/smoke/business/deploy/deploy-command.spec.ts 3eeedc180c95c2586cc4d0b3db2ed0ef7ed05caa675a694c5b06fff753a5960d
frontend/tests/smoke/business/deploy/deploy-pages.spec.ts abdd9310709f15f3f8f0d869f8120344bc0f2bf6123b96a7b8b9a966022e774e
frontend/tests/smoke/business/deploy/deploy-selection-preview.spec.ts 601ad32e6ad64df3c005419aabf35e24d103cfc218db928e090adc45f08adbc8
frontend/tests/smoke/business/deploy/deploy-workbench.spec.ts 9ca48b5cffcf7886e4be08bdf0b6c133924b8624eb332772d8908de029748210
frontend/tests/smoke/business/deploy/reviewed-start.ts b7fefda17822370e80df5db6597eb15fd9872b46f0caa4b4221a11f1d5a695cd
frontend/tests/unit/business/operational/deploy-command.test.tsx 5e5a8e5199318a752cb99fb815762f56a39298bc7c3c89e64dbd99a2ab75a65a
frontend/tests/unit/business/operational/deploy-context.test.tsx 8aeaa42a0d0ddb6f88f17ff969ae6739dd7060e571de7f06325542106707752e
frontend/tests/unit/business/operational/deploy-log-cursor.test.tsx ceed93cae2c05eb9b4e5711c6f478ac590761841a46e2f240c43face172b9cb0
frontend/tests/unit/business/operational/deploy-preview.test.tsx 17bd0ecea77104215c36ca97fa236f2f040b21a7de9b3e5c9410ed5d833b9e41
frontend/tests/unit/business/operational/deploy-selection.test.tsx 9dd027c3dd472c48341dc2593f3a5bcf2518c75bdeb411cf459668fae2b89c4d
frontend/tests/unit/business/operational/log-lifecycle.test.tsx 27e55bc19d74f658382b13acedaba01d1b0cd42a7808cf62fa0dd121f50b91cc
```

</details>
