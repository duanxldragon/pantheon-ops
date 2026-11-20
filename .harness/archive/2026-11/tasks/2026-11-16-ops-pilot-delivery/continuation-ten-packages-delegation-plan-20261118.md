# Ten-package continuation and delegated implementation

User steering: prioritize the existing ten C/E work packages; some implementation may be assigned to Claude. L2, pantheon-ops business-only. Existing original manifests and task scopes remain the acceptance source; historical85 classification/22archives+2preserved copies do not change.

Minimum complexity: reuse current owner capabilities, UI patterns, runtime harness and locked Base v0.18.9. No new generic platform, system implementation, provider credentials, production operations, gate exemptions or Chart execution invented. Scope authorization covers local implementation; maintainer/external final acceptance remains explicit.

## Ownership and order

| Package | Implementation lane                             | Exclusive first-wave ownership                                                                   | Dependency/stop                                                                       |
| ------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- |
| C01     | Native executor                                 | business/deploy and necessary business/service backend                                           | Queue/blocked claim/host interleavings; preserve verified RC Retire proof             |
| C02     | Later executor                                  | Deploy UI and ServiceCatalog only after C01/C03 ownership release                                | C01 contract stabilization                                                            |
| C03     | Later executor                                  | ServiceCatalog/target-lifetime scope, serialized with C02 and C01 service backend                | Original target/role/provider acceptance retained                                     |
| C04     | Native executor                                 | business/k8s/service; narrowly extended cluster_service.go, namespace_service.go and owner tests | Owner locks through bounded external write; no live kubectl                           |
| C05     | Verifier preparation, then current-candidate QA | Acceptance map/new evidence only during source writes                                            | C01-C04 accepted candidate; no historical pass extrapolation                          |
| C06     | Delivery preparation, then packaging QA         | New delivery evidence only                                                                       | C05 plus E04 for exposed extensions; final human gate                                 |
| E01     | Claude frontend, backend lane separately        | frontend/modules/business/filedist; backend filedist only by assigned backend executor           | Existing authorized target selection, nil fail-closed, cancellation/schedule/recovery |
| E02     | Later independent executor                      | business/cicd/argocd; opsinit changes centralized by root integration lane                       | No hard dependency; live provider remains a separate gate                             |
| E03     | Claude                                          | business/k8s/helm and README.md/README.en.md                                                     | Registration truth only; no fictitious deployed state or added Chart execution        |
| E04     | Verifier preparation, then current-candidate QA | New enabled-provider acceptance evidence                                                         | Conditional E01/E02/E03 dependencies for exposed capabilities                         |

Root owns shared runtime locale bundles, task statuses, official overlay rebuild, candidate/source bindings, central MySQL/race scheduling and integration. All writers are told they are not alone and must preserve others' edits. Claude prompts explicitly exclude shared sources and ask for separate locale deltas. A worker may not self-approve, fabricate command success, restart services or publish.

## Evidence and verification

Prior verified frozen candidate: sourceSet28031c9735a926509aec1138f82f63cfad7aaac48128c383625f1b1d5790e8f0; full Windows MinGW/CGO race -04 exit0/885.490s with734backend sources stable, official rebuild1221 and navigation14/14. These are historical evidence after new source edits.

Claude uses the existing hidden native CLI runner, strict empty MCP, explicit Read/Edit/Write/Grep/Glob tools, per-invocation disabled hooks, default model/authentication, bounded turns/time and task-owned Job Object. Preserve raw result/status/process receipts, including errors/timeouts; Start-to-Job assignment race and third-party child visibility remain evidence limitations.

Per changed behavior: meaningful regression, targeted CGO race, scoped diff, independent code/architecture review, frontend lint/type/build and actual desktop/mobile/interaction rendering if UI changed. Run official locked-release rebuild/overlay/drift and complete readonly Windows CGO race on the final frozen candidate. C05/E04/C06 must bind their own original acceptance and same-candidate evidence; production/provider gaps are not replaced by mocks or skipped tests.

Root runtime/readiness artifacts remain in .harness/evidence/2026-11-17-ops-history-audit; package-local evidence must be materialized accurately, with original absent evidence recorded rather than retrospective success. All existing whole-repo governance red findings remain explicit; this work must introduce no new findings.

Stop condition: each original criterion is either evidenced by the final candidate or remains explicitly unaccepted with its owner/prerequisite. Do not mark all ten completed while provider or human gates remain open.
