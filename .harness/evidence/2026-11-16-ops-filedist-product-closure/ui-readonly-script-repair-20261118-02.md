# E01 read-only QA script repair for second run

Owner: `/root/e01_ui_partial_closeout`; only `.local/manual-preview-20261116/verify-ten-packages-ui-20261118.mjs` changed in this lane. Prior run01 screenshots, log and JSON are preserved. Production repairs in `api.ts` and `FiledistList.tsx` are owned separately by E03 following root's screenshot review; this lane does not edit them.

## Observed run01 evidence

Root executed the full source-bound script through its approved elevated command path. Independent CIM preflight passed, actual child exited 1, source snapshot remained stable, and four screenshots were captured. Live desktop list, live basic create modal, live 404 render and populated fixture list passed the first script's checks. Bounded file picker reached page6/ID101, then remote search selected ID125 at pageSize20. The trace records the local fulfilled resolver POST and scoped host GET. The next field visibility assertion failed in `desktop-fixture-catalog`; its diagnostic only retained the assertion's first line, so the exact field/timing cause was not established by run01.

Root inspected the actual screenshots and identified two production issues: untranslated `common.view` and shared global error toasts from 404 sibling reads. Run01's limited assertions do not certify these surfaces as accepted. Root assigned the production repairs to E03.

## QA changes

- Configure standalone Playwright expectations with a 12-second timeout, rather than relying on a test-runner context that this standalone script lacks.
- Wait for the fixture resolver response (must be status200), then the SSH username form item to become visible before asserting the remaining fields. Fixture data, provider boundary and all network gates are unchanged.
- Record explicit scope/resolver/auth subphases, safe assertion summary, locator, expected static field label and script line frames. Never record received field values, raw HTML, cookies, headers or authentication data.
- On failure, preserve `failed-<phase>.png`; mask password inputs and all textareas before capture. Keep its failed assertion status and fixture/live diagnostic classification explicit.
- Reject raw `common.view` in rendered page text.
- Require a translated `详情` button on a nonempty live task list; an empty list is not required to invent an action row.
- Count every inserted `.arco-message` in each browser document with a browser-only observer. At a handled live detail404, require insertion count zero and no visible Arco message/message-content. Counting insertions prevents auto-dismissed global errors from producing a false pass.
- Apply the same zero-toast insertion/visibility check to the browser-fixture 403/503/offline detail errors. These additional checks remain explicitly fixture rendering evidence.

Original source-equivalence, current-source hashes, successful backend/frontend build receipts, actual running process/binary identity, dependency readiness, local-only read policy, hidden child/browser launch and final source-before/after comparisons remain required. No production acceptance claim is added.

`node --check` passes after these edits. Browser run02 has not been executed by this lane. Root will provide new proof/build/runtime/quality suffixes after the production fixes; an old proof must not be mixed with new source. The run02 artifact suffix is `ten-packages-20261118-02`, and root handles any required separate command-prefix approval.
