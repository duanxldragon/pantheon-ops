# C04 UI source handoff

Status: implementation candidate and static verification complete; browser/runtime and full delivery gates remain open.

## Changes

- Service API exposes UID/resourceVersion, nullable selector and numeric/named targetPort. PUT submits the original UID/resourceVersion; DELETE requires both as query preconditions.
- ServiceList gates create/update/delete independently, provides explicit refresh and a stable namespace/name/UID row key. Failed writes retain the original form and identity, with request feedback supplied once by shared default handling.
- Reused AppModal with labelled multi-port fields. Edit keeps immutable name/namespace, existing numeric/named target ports and allocated node ports. ClusterIP explicitly drops nodePort. Selectorless null is displayed safely and edited as an empty selector; malformed/duplicate entries are rejected without silently losing fields.
- Five canonical locales contain the same 18 C04 keys; backend owner synchronized 36 exact zh/en seed values. Error appendix records the real 409 contract. Generated fallback remains coordinator rebuild work.
- The spec discovers 15 tests: 2 existing K8s page cases plus 13 C04 UI cases. C04 covers create, edit, selectorless/invalid selector, multi-port/type conversion, stale writes, preconditioned deletion, identity/permissions, and desktop/mobile rendered-state capture scenarios.

## Fresh local verification

`verification.json` records exact commands, raw stdout/stderr, UTC timing, exit codes and 12 source SHA256 values before and after verification.

- Scoped ESLint: exit 0.
- Production `tsc -b`: exit 0.
- Playwright discovery: exit 0; 15 tests discovered. No browser test was executed by this executor.
- Scoped `git diff --check`: exit 0.
- Canonical JSON checks: 90 locale/key comparisons passed; zh/en values match the seed handoff source. Source hashes stayed unchanged during verification.

## Visual gate and remaining evidence

Applied `impeccable` before implementation and reviewed the source against its visual checklist at handoff. The surface uses existing table/modal/action components, Pantheon tokens, a two-column form that stacks below 600px, associated labels, named destructive actions and disabled/submitting states.

Rendered screenshots were not produced or inspected by this executor because the coordinator exclusively owns browser and runtime execution. This is not a visual completion claim. Run `C04 Kubernetes Service UI contract` to capture 1440x900 and 390x844 loading/error/empty/edit/ports/submitting/conflict/create states, inspect the resulting screenshots, and bind them to this frozen source receipt. Mock business fixtures exercise UI/HTTP behavior and do not prove live provider or real role admission.

Stop condition reached: assigned reversible frontend/locale/appendix implementation and static checks are complete. Coordinator still owns live browser/provider/role acceptance, independent review, full CGO race, official-lock rebuild/source binding, maintainer final acceptance and archive decisions.
