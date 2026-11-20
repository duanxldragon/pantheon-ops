# Review: 2026-11-16-ops-business-menu-m3c

## Verdict

findings addressed

## Findings

1. The service M+3C originally proposed retiring the `ServiceCatalog` 3-tab component. The committed authoritative i18n fallback bundle retains all `business.service.catalog.*` / `business.serviceinstance.*` component keys and has no per-page child keys, and the bundle is never regenerated. Resolution: keep `ServiceCatalog`, split the menu into parent + 3 children, and deep-link its active tab from the route path (`/business/service-catalog/{applications|services|instances}`). This stays faithful to the committed key set and passes the menu-contract gate. (Deviation from the earlier plan wording, justified by the authoritative bundle.)
2. The fresh `000040_biz_business_environment` source migration is a derived business asset; `check-business-overlay` and `check-shared-drift` resolve the runtime copy at `backend/pkg/database/migrations/`. Copies were added and verified byte-identical. `check:generated-modules` only flagged the new migration while it was untracked (a false positive), and passes once staged/committed.
3. `audit:i18n-locales` was failing before this PR on 5 pre-existing keys missing from the non-zh fallback bundles (`business.cmdb.host.matchedGroups*`, `business.deploy.package.latestDeployedAt/latestTask`). They are referenced in cmdb/deploy code and present in zh-CN; added correct English values to `en-US.ts`, which the other locales inherit via their en-US fallback import chain.

## Residual Risk

- Local rendered/playwright and end-to-end runtime smoke evidence is not captured in this session; rely on the GitHub `pull_request` workflows (smoke-sanity, quality) for clean-branch coverage.
- The `2026-11-13` menu-IA task packet planned in `pantheon-base` was lost during base release work; this packet (`2026-11-16-ops-business-menu-m3c`) is the ops-local replacement that satisfies PR governance.
