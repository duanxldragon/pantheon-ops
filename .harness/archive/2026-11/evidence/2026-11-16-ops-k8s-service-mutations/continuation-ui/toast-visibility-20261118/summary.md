# C04 toast readability and draft action correction

Status: C04 source correction; fresh browser/render and full build remain coordinator gates.

## Evidence and cause

The executor inspected these original screenshots under `.harness/evidence/2026-11-17-ops-history-audit/continuation-c04-ui-locale-refreshed-first-artifacts/`:

- `smoke-business-k8s-k8s-pag-636ae-ed-primary-states-at-1440px/c04-1440-conflict.png`
- `smoke-business-k8s-k8s-pag-f798f-red-primary-states-at-390px/c04-390-conflict.png`
- `smoke-business-k8s-k8s-pag-636ae-ed-primary-states-at-1440px/c04-1440-create.png`

Both conflict screenshots lack the visible toast; the later desktop create screenshot shows only its upper portion. Arco's global message wrapper is fixed at viewport top 40px with z-index 1003; the shared AppModal wrapper and dialog use 2500. The edit modal starts above 40px and blocks the toast. The shorter create modal starts below the toast top and exposes only the upper part. This is actual occlusion; waiting for the 100ms fade alone cannot solve it. `toBeVisible` did not prove viewport exposure.

The original frontend build red in `continuation-frontend-build-final-c04-ui-locale-refreshed.log` classifies `IconDelete` as a persistent destructive action requiring confirmation. Removing a draft port only changes the unsubmitted form; the persistent mutation remains the confirmed Save operation. A neutral minus control expresses this distinction accurately.

## Minimal corrections

- Only Service create/update calls from this C04 form opt into the existing `skipErrorMessage` API extension. A modal-owned Arco Message hook renders in the existing modal title layer, with one memoized owner for the modal lifetime. Shared success/load/delete feedback stays enabled. RequestError keys are translated with the shared fallback-key policy; non-RequestError feedback remains.
- Closing the modal unmounts its own toast. New create is checked after that natural lifecycle completion; the test does not clear global messages or fake a disappearance.
- Draft port removal uses the existing IconMinus with neutral text styling. Actual Service deletion still uses danger styling and Popconfirm. No checker exemption or shared checker change was introduced.
- The same 13 C04 cases now require canonical toast count 1 and total error-toast count 1, opacity and finite animations settled, all edges inside the viewport, content fitting its box, and hit testing at the center and four inset corners. Readability measurements are attached as JSON. Input/UID/resourceVersion, single request and not-loading assertions remain. After cancel, the owned toast must unmount within the modal close interval before opening create.

## Verification boundary

`verification.json` records exact scoped ESLint, production typecheck, shell visual contract, Playwright discovery and diff checks, along with 17 source hashes before/after. This executor does not run the browser or full frontend build; the coordinator reruns the 13 C04 cases, inspects the corrected desktop/mobile screenshots and runs the full build.

The generic global Message/AppModal stacking mismatch remains an explicit Base-first follow-up. This change uses C04-owned feedback extension points and does not modify shared CSS, shared request behavior, generated locales, seed, other business modules or historical red evidence.
