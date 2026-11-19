# E01/E02 extension composition and error integration

L2, business-only; parent task packets are `2026-11-16-ops-filedist-product-closure` and `2026-11-16-ops-argocd-ownership-convergence`. Implementer: `/root/extension_startup_integration`. Other lanes own frozen execution/observation sources and FileDist UI; preserve their edits.

In: start FileDist after schema/provider wiring and before HTTP exposure; run ArgoCD on the same owned cancellation context; propagate startup failure through the existing panic adapter and stop owned components; translate nine existing canonical lifecycle keys in bilingual appendix, module runtime seed and declared frontend locale sources. Keep the current all/admin route boundary and nil authorization fail-closed.

Out: Base/system sources, frozen lifecycle/engine/service/observer implementations, new permission grants, credentials, publication and service restarts. No production FileDistributionAuthorizer adapter exists in the inspected composition/port graph; do not replace this with an allow-all implementation. This remains an explicit product gap.

Implementation: reuse `FileDistModule.Start/Stop` and `argocd.NewReconciler.Start/Stop`; the latter provides a joinable owned goroutine without changing the frozen `StartAll` API. Ops' void initialization adapter owns signal cancellation because the locked Base startup has no business lifecycle callback. Cancellation joins this observer and stops the distribution module. The unchanged Base process shutdown does not await this callback; process-level shutdown ordering remains an evidence gap.

Error mapping: the locked generic response resolver accepts lower-case key segments only. Business handlers explicitly preserve only the declared camel-case canonical keys, retain sentinel permission/validation handling and hide raw provider/database detail. Do not change Base's resolver.

Verification: meaningful startup failure/cancellation/idempotent cleanup unit regressions; exact canonical HTTP key and status regressions; scoped readonly Go compile/vet/gopls and CGO race where feasible; JSON parity/seed-key consistency. Root owns real MySQL, official rebuild/fallback regeneration, complete race, browser/provider and independent review. Do not declare either package complete from these local checks.
