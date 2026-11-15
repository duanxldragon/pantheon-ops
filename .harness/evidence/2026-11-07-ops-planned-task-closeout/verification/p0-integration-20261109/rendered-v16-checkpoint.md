# v16 rendered inspection

The browser matrix passed 18/18 with exitCode 0. Backend/frontend ran on the authorized test server 115.190.251.143; deterministic UI provider responses were intercepted. Real-provider acceptance is separate. Official startup migrations reached version36/dirty0 in the isolated v16 database. Owned backend/web processes stopped and forwarding closed.

The launcher subsequently exited1 while printing Unicode output (`UnicodeEncodeError`). Preserve `remote-p0-browser-v16-launch-failure.json`; it is a post-report runner failure, not a failing browser assertion. The v16 backend binary predates the final Argo HTTP-status classification repair, and none of these browser specs exercises Argo.

Impeccable inspection: 79 screenshots produced; 18 selected images inspected in `rendered-v16-{loki,tempo,deploy-k8s}-contact-sheet.png`. Exact image hashes/dimensions are in `rendered-v16-source-images.json`.

- Loki desktop1440/mobile390, light/dark: query/time/limit controls wrap; completed source and warning/follow notices remain visible. Partial and denied mobile states are distinct. Wide label rows stay inside intentionally scrollable tables, with omission notices. No full-page horizontal overflow in the assertions.
- Tempo desktop, light/dark: trace results and focusable detail region are visible; denied mobile state is readable and clears results. Both `tempo-*-mobile.png` captured sidebar/drawer transitions after resize; these are insufficient stable-frame mobile evidence and need replacement. Passing overflow assertions do not prove transitions have completed.
- Deploy history desktop/mobile, light/dark: current versus historical attempt context is present. The consumed step rail still shows a white background in dark mode; the Base repair must pass review and official consumption before this shared defect closes. Long full-page mobile contact thumbnails are insufficient for detailed text/contrast inspection.
- K8s dark mobile preview: source, bounded image change, rollback limitations and confirm/cancel controls are visible. This proves the image preview UI only.

No overall UI-polish or complete P0 acceptance claim. Stable Tempo mobile renders, repaired/released shared step rail and final functional/visual acceptance remain open. Older failing runs are retained.
