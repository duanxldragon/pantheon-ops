# E01 read-only rendered verification script handoff

Owner: `/root/e01_ui_partial_closeout`; production UI and frozen regression fixture remain untouched. Script: `.local/manual-preview-20261116/verify-ten-packages-ui-20261118.mjs`.

Impeccable's skill and `references/visual-quality-checklist.md` were read for this lane. Intended surface: a calm, compact operations list and task form using the existing design system. Primary checks are desktop 1440×900 and mobile 390×844 real task list/empty state, real basic create modal, and real detail 404. Fixture captures additionally cover populated list, bounded file paging through ID101 and remote search ID125, password/private-key mutual exclusion and reset, per-host fingerprint/schedule fields, loading/403/503/offline detail renders, future-pending start disabled, partial retry visible, and terminal controls absent. No distribution submit or control action is executed.

## Exact execution command

From the Ops repository root:

```text
node .local/manual-preview-20261116/verify-ten-packages-ui-20261118.mjs ten-packages-20261118 ten-packages-20261118-01 ten-packages-started-20261118 ten-packages-20261118-02
```

The command binds the successful candidate source-equivalence receipt, backend build receipt/binary hash, successful frontend lint/type/build/unit receipt, and running-binary readiness receipt. It retains the original independent CIM listener/PID/executable check. Root has separately verified the new binary identity and readiness; that fact does not make this independent probe successful.

Expected output: `ui-readonly-ten-packages-20261118-01.{log,json}` plus `ui-readonly-ten-packages-20261118-01-artifacts/render.json` and up to 19 screenshots in this evidence directory. The aggregate receipt records the source inventory before/after, actual hidden child exit/signal/error, hashes of proof/build/quality/runtime receipts, screenshot hashes and boundary classification. Existing run artifacts are protected with exclusive creation; failed executions are preserved, and a fresh suffix is required once artifacts exist.

## Network and fixture boundary

The script imports the existing `frontend/tests/smoke/helpers/auth.ts` session functions and existing Playwright installation; it installs no dependency. Login uses the existing local API helper at 127.0.0.1:8080. A headless browser loads 127.0.0.1:5173. The only real POST is local auth login. Browser requests permit localhost/127.0.0.1 ports 5173/8080 GET/HEAD. All other writes and external HTTP URLs are aborted and cause failure.

Root explicitly authorized one browser-only `POST /api/v1/business/deploy/targets/resolve` fixture so the production modal can show its target configuration fields. The browser route fulfills this response locally with documentation-only host address 192.0.2.71, never forwards it to the backend, and records `browser-fixture-only-POST-not-forwarded`. Other fixture data and status responses use GET. Fixture screenshots prove rendering and interaction only, not live authorization/provider admission. Authentication strings used to test mode clearing are inert fixtures and are cleared before screenshots. Tokens, auth passwords, request headers and browser storage are not exported.

The wrapper launches its Node worker with `windowsHide: true` and the existing `hide-child-windows.mjs` descendant preload. Chromium is headless. Hidden flags establish the launch configuration, not proof that every third-party instantaneous window event is impossible.

## Preparation evidence and actual first attempt

`node --check` passed. `--prepare-only` passed without browser launch or network requests: the actual installed Chromium executable exists, the existing TS auth helper imports under Node24, and four local/write/external/fixture request-policy checks pass. Its receipt is `ui-readonly-script-preparation-20261118-01.json`. A later pagination-read counter hardening changes the script hash; the final handoff hash is separate from that historical preparation hash.

The executor's first actual full command exited 1 at its independent preflight process check: a nested PowerShell `Get-NetTCPConnection` returned permission denial (HRESULT 0x80041003) under the managed sandbox; the probe returned status2 and the assertion failed. This happened before artifact creation, browser launch, local login or business reads. No render is claimed. `ui-readonly-preflight-failure-20261118-01.json` records this after the tool output.

A diagnostic showed that netstat/Get-Process could read the same local process, but root instructed retaining the original CIM probe and executing the full command through its already-authorized local command path. The script was not changed to use the alternative, and no independent-probe success or rendered acceptance is inferred from that diagnostic. Root will request escalation only if its authorized run still fails due to the sandbox.

Production code, prior evidence and acceptance status were not changed by this script preparation. Root owns final execution and screenshot inspection; actual SSH/provider acceptance and full visual acceptance remain separate.
