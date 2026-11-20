# E03 live acceptance plan

Parent task: `2026-11-19-ten-package-acceptance`; original package:
`2026-11-16-ops-helm-capability-truth`. Scope is evidence only. No product source,
package status, service restart, external provider execution or unrelated data edit.

Files: `.local/acceptance-20261119/e03-live.mjs`, uniquely suffixed E03 receipts,
screenshots and this lane's report. Reuse Playwright, the existing actual-login
helper and descendant-hiding preloader. Bound every request and browser run.

UI surface: operational K8s application-release workbench. Intended feel: clear,
restrained operational controls whose image-release purpose cannot imply Chart
execution. Use existing design tokens; review 1440x900 and 390x844 output with
impeccable. No dedicated Helm page exists in the current registered routes.

| ID     | Intent / user                                  | Setup and harness                                                         | Expected signal                                                                        | Result                            | Cleanup            |
| ------ | ---------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------------- | ------------------ |
| E03-01 | Reject unauthenticated reads/writes            | Separate cookie-free request context; GET/POST Helm                       | 401/403, no fixture row                                                                | pending                           | own row check      |
| E03-02 | Prevent fake deployed registration / admin     | Unique name/namespace; POST status=deployed; GET exact name               | status/effectiveStatus/storedStatus=registered                                         | pending                           | delete returned ID |
| E03-03 | Reject malformed input / hostile caller        | Missing fields, malformed JSON, blank name, zero cluster                  | non-success; no additional namespace rows                                              | pending                           | verify count       |
| E03-04 | Preserve historical status provenance / reader | Read-only inspect existing Helm registrations                             | Any legacy deployed row projects registered and preserves storedStatus                 | pending; not-run if no legacy row | no mutation        |
| E03-05 | Keep deferred execution unavailable            | Source-bound absent install/upgrade/rollback/uninstall paths; POST probes | 404/405 without provider action                                                        | pending                           | none               |
| E03-06 | Expose truthful menu/page / signed-in admin    | Real release API, desktop and narrow browser render                       | 应用发布, image controls, no Helm/Chart execution claims, no raw keys or page overflow | pending                           | read only          |
| E03-07 | Registration deletion only / admin             | DELETE only returned fixture ID; exact-name GET                           | own namespace empty                                                                    | pending                           | required           |
| E03-08 | Reject no-permission caller                    | Coordinate task-owned restricted fixture with C03 lane                    | sanitized GET/POST/DELETE rejection witnesses                                          | pending external lane             | fixture owner      |

Dirty work is retained; before/after hashes bind relevant source, docs, helpers
and lock. Existing binary/source receipts are checked before runtime claims.
Misleading success output cannot pass: wrapper needs successful exit, stable
hashes, every required runnable scenario and cleanup. Fresh suffixes preserve old
receipts. Command/request timeouts and finally cleanup cover interruption/hangs.
Prompt injection and unrelated OMX state are irrelevant to this non-agent API;
malformed caller fields and fake status are the applicable hostile cases.

Stop: produce truthful criterion-by-criterion evidence. Criterion 2 requires real
Chart evidence or a visible deferral plus an explicit maintainer risk decision.
The current docs prove deferral, not that decision. No self-approval or package
acceptance claim will be generated.
