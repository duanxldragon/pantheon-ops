# C04 modal notice identity correction

Status: minimal source correction, pending coordinator fresh browser/render evidence.

The three failures in `continuation-c04-ui-toast-visibility-fixed.log` are real duplicate notices. Trace DOM snapshots show two sibling `.arco-message.arco-message-error` nodes under one message wrapper, not one notice matched through an ancestor. `trace-message-structure.json` records the exact trace hashes, single PUT request and duplicate snapshot structure for each failing case.

React 19.3.1 StrictMode replays the Arco hook's initial callback ref. In the first-instance branch of `Message.useMessage`, every non-null callback calls `instance.add(_noticeProps)`; without an explicit ID, the existing notification model allocates different random IDs and appends both. `maxCount` governs later additions and does not protect that initial branch. Memoizing the modal owner prevents form rerenders from changing ownership but does not disable StrictMode replay.

The dynamic probe uses the actual installed React/Arco with jsdom and StrictMode enabled. One `error` invocation without an ID produces two notices. The same invocation with a fixed modal-local ID produces one notice; a later same-ID update keeps one and updates its content. Unmounting the owner removes all notices. The probe asserts these results and records them in `strict-message-probe.json`.

The production correction only passes `id: 'c04-service-write-error'` with the local error content. This reuses Arco's existing notice replacement semantics. Shared success/load/delete behavior, StrictMode, modal ownership and all 13 C04 tests remain unchanged. Canonical notice count 1, total error-notice count 1, viewport/opacity/animation/hit-test checks, draft/UID/resourceVersion preservation and cancel/new-create lifecycle assertions are not relaxed.

`verification.json` records fresh scoped static checks and 17 before/after source hashes. The coordinator reruns the 13 browser cases serially; no concurrent frontend build is requested. Historical red runs and traces are preserved.

## Coordinator follow-up

Fresh 13 browser cases passed with all 17 sources stable. `coordinator-visual-binding.json` binds those sources, 20 actually viewed primary PNGs and 20 identical Playwright attachment copies. Fresh lint/type-check/build passed with recorded warnings. Independent code round2 APPROVE and architect narrow CLEAR; whole C04 WATCH.

The historical line-only run did not retain JSON-body readability attachments. A separate three-case execution retained real measurements with the same 17 sources: all viewport/opacity/animation/content-fit conditions and all 15 sampled points passed. Independent code and architect appendices accept that retention gap closure; these new measurements do not reconstruct the older run. Notice temporarily overlaps the modal heading, and live provider/actual roles/shared Base stacking remain open.
