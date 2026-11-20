# C03 catalog-options regular-unit integration

Parent: `2026-11-16-ops-service-catalog-usability`; implementer `/root/e03_claude_partial_closeout`; business/service, L2 inherited from parent.

Reuse the seven existing catalog activation/abort/error regressions from the preserved evidence fixture, copy them into `frontend/tests/unit/business/service/useCatalogOptions.test.ts`, and use the regular Vitest configuration/unit glob. Change only the test import to its business-local relative source path for isolated TypeScript verification. Declare exactly this new test in `business-overlay.json` without removing or broadening existing declarations. Do not modify production ServiceCatalog/useCatalogOptions, backend, shared config/locale, old evidence fixture or runner.

Impeccable quality scope: form target options must show fresh loading/empty/error states and reject stale results after reopen/context changes. No visual layout changes or rendered evidence from this lane; desktop/mobile rendering remains a coordinator gate and no UI polish is claimed.

Validation: invoke installed native Node/Vitest entry with the existing hidden-child preload, standard config and narrow test path; require seven passing tests, zero skips. Run isolated strict TypeScript and scoped ESLint for the new test, JSON/manifest overlay checker, whitespace diff and source/fixture hashes. All Node descendants must inherit windowsHide; no shell shim or new dependency.

Stop: new regular-unit test and narrow overlay declaration verified, old evidence retained and new receipt handed off. This does not establish C03 backend target lifetime, NamespaceBinding/environment/action, provider/real roles, C02 dependency closure, full frontend/build/runtime or human acceptance.
