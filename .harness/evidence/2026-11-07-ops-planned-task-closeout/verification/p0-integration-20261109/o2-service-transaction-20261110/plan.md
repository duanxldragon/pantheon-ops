L2 parent: docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md.
Generator scope: capability/capability.go, Service owner state.go, Deploy target_selection.go, linked owner regressions and explicit recording-fake binding. Business-to-business logical extension; no physical/system/Base/frontend changes.
Reuse rung: existing CMDB owner transaction-binding pattern; no new dependency/framework.

1. Save current capability/service/deploy snapshots and real-owner fault-injection RED (owned + manual).
2. Add narrow WithServiceInstanceStateTransaction owner API; owner delegates all validation/state/retire commands using caller transaction, context and SkipDefaultTransaction. Bind only actual transactions; refuse unknown injected owner before invocation. Preserve nil/no-link and request compatibility.
3. Required isolated MySQL33316 RED->GREEN, revision/scope/transaction lifecycle regressions; complete focused capability/service/deploy race count1, vet/build/native gopls, source stability and exact isolated cleanup evidence.
4. Re-read independent CODE report and handoff <=850 words for independent review. Dispatcher retains full-tree/runtime/remote/source gates; no P0 closure/self-approval.
   CodeGraph index exists; configured CodeGraph MCP/CLI is not exposed in this execution environment. Reading identified current files directly; no index modification.
