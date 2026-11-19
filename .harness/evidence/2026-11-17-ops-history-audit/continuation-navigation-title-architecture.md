# Continuation navigation-title architecture review

## Summary

**Architectural status: WATCH. Recommended synthesis: COMMENT.**

The latest candidate closes the deployment-task navigation readiness gap and completes the official-cache shared-drift comparison. This is an additive review: earlier failures and WATCH reports remain valid historical evidence, and the unchanged governance failures prevent an overall completion or deployment approval.

## Analysis

### Navigation readiness now matches the rendered component

The deployment-task page supplies `operations.deploy.task.menu` to `GovernanceSummaryBar` (`frontend/src/modules/business/deploy/task/DeployTaskList.tsx:649`); that key resolves to **部署任务** (`frontend/src/modules/business/deploy/locales/zh-CN.json:5`). The component renders the title as `Typography.Text`, explaining why the intermediate semantic-heading locator failed (`frontend/src/components/governance/GovernanceSummaryBar.tsx:48`).

The current smoke walker waits for the correct URL, exact title text, title visibility, expected content, and absence of the dashboard marker (`frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts:174`). The render probe applies the same checks before capturing page state (`.local/manual-preview-20261116/probe-navigation.mjs:115`). It retains rejection of failure pages, wrong dashboard content, empty pages, and untranslated keys (`.local/manual-preview-20261116/probe-navigation.mjs:126`).

These assertions ground readiness in the actual destination page without removing the existing rejection checks.

### Latest navigation evidence passes; earlier failures remain preserved

The latest smoke reports **3 passed** (`.harness/evidence/2026-11-17-ops-history-audit/continuation-navigation-smoke-final-locale-current-title.log:9`). Both smoke and render execution receipts record exit 0 (`.harness/evidence/2026-11-17-ops-history-audit/continuation-navigation-locale-current-title-verification.json:17`, `:35`).

All 14 render acceptance rows pass, including business-page navigation and mobile navigation (`.harness/evidence/2026-11-16-ops-navigation-repair/after-history-continuation-final-locale-current-title/acceptance.json:7`, `:55`). Independent probe inspection confirmed 30 entries, no rejected rows, and `errors=[]`; those requirements remain explicit in the probe (`.local/manual-preview-20261116/probe-navigation.mjs:182`).

The reviewed deployment-task desktop and mobile-navigation screenshots provide bounded rendered evidence. They do not establish independent visual acceptance of every page.

The binding retains the earlier render failure and intermediate heading-locator failure as separate attempts (`.harness/evidence/2026-11-17-ops-history-audit/continuation-navigation-title-binding.json:50`, `:94`). It also explicitly records that the original local helper byte snapshot is unavailable (`:43`).

### Candidate identity remains coherent

The candidate is:

`.local/manual-preview-20261116/overlay-history-continuation-final-locale-current-title-cache-approved`

Independent hashing confirmed all **1932 owned files** match the candidate report, and all **1197 source rows** match their recorded hashes in both source and candidate. The report identifies Base commit `6316f9b7189c86542424145fa49344ad737e952b` (`.local/manual-preview-20261116/overlay-history-continuation-final-locale-current-title-cache-approved/.business-overlay-report.json:3`).

The 1197-source set retains SHA256 `75285994b4249a9fd92f0f30beb9a4b0182fe1b9c7635266794d629a986cf29c` (`.harness/evidence/2026-11-17-ops-history-audit/continuation-locale-current-title-cache-approved-source-equivalence.json:6`). The binding records the new Spec hash `4c9ed10eb811183c1ec727ddbd8d8259989f15c931db35023e1b4b9285c999cb` and candidate-report hash (`.harness/evidence/2026-11-17-ops-history-audit/continuation-navigation-title-binding.json:7`, `:27`).

### Official-cache drift retry preserves the inheritance contract

Official rebuild and staged overlay passed; the candidate-root CLI drift attempt remains exit 1 (`.harness/evidence/2026-11-17-ops-history-audit/continuation-rebuild-locale-current-title-cache-approved-verification.json:14`, `:31`, `:48`). The first official-cache attempt also retains its GitHub TLS-handshake timeout (`.harness/evidence/2026-11-17-ops-history-audit/continuation-staged-drift-official-cache-final-locale-current-title-cache-approved.log:5`).

The network-approved retry passed: **866 shared files compared, 856 identical, 10 allowlisted differences, 253 business-overlay files excluded under the existing contract, and 14 derived migrations verified** (`.harness/evidence/2026-11-17-ops-history-audit/continuation-staged-drift-official-cache-final-locale-current-title-network-approved.log:11`).

The helper requires byte-identical candidate/root locks and calls the original resolver and checker (`.local/manual-preview-20261116/check-staged-official-cache-drift.mjs:17`, `:23`). The resolver retains official release, checksum, tag-commit, fresh-extraction, and shared-path validation (`scripts/business-overlay/foundation-release.mjs:111`, `:128`). The resulting identity receipt records exit 0 and explicitly preserves the candidate-root timeout boundary (`.harness/evidence/2026-11-17-ops-history-audit/continuation-overlay-history-continuation-final-locale-current-title-cache-approved-official-cache-drift-identity.json:11`).

## Root Cause

The original generic deployment-task text check could observe dashboard content before destination content completed rendering. The intermediate replacement assumed a semantic heading, while the actual title is `Typography.Text`. The current exact visible title, URL, and dashboard-exclusion checks resolve these synchronization mistakes (`frontend/src/components/governance/GovernanceSummaryBar.tsx:48`; `frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts:174`).

## Remaining WATCH concerns

- The generator produces five business fallback files but wires only Chinese and English into root resources; runtime loading uses base, overrides, and generated resources (`scripts/business-overlay/rebuild-from-base.mjs:365`, `:390`; `frontend/src/i18n/index.ts:51`). This remains a multilingual wiring concern.
- Duplicate locale ownership remains unresolved: `stepConditionKey` is “参数名” in the deploy locale and “条件变量” in the legacy locale. Recursive `Object.assign` permits later files to overwrite earlier values (`frontend/src/modules/business/deploy/locales/zh-CN.json:126`; `frontend/src/modules/business/v12locales/zh-CN.json:762`; `scripts/business-overlay/rebuild-from-base.mjs:371`).
- Governance failure lists remain unchanged at **547 task / 129 evidence / 91 review / 6 links**. Independent comparison confirmed identical findings, not merely matching counts. The receipt explicitly states that zero delta does not waive existing failures (`.harness/evidence/2026-11-17-ops-history-audit/continuation-title-before-progress-gate-delta.json:6`, `:15`, `:24`, `:33`, `:42`).

## Recommendations

1. **Low effort:** Accept this additive evidence as closing the navigation readiness and official-cache drift verification gaps; preserve all earlier attempts and reports.
2. **Medium effort:** Resolve multilingual fallback wiring and duplicate-key ownership in their appropriate inheritance scope, with runtime language verification.
3. **Separate governance work:** Close the recorded historical failures through their authorized adjudication process. This review supplies no waiver.

## Trade-offs

| Choice                                        | Benefit                                                            | Cost                                            |
| --------------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------------- |
| Exact existing title selector                 | Matches current rendering and avoids a production-component change | Couples the test to the class and Chinese title |
| Shared semantic readiness contract            | Could support broader reuse                                        | Requires a separately scoped shared UI change   |
| Official resolver with task-local cache reuse | Preserves pinned release validation                                | Remains dependent on GitHub availability        |

This architecture lane is complete for the latest title candidate. **WATCH / COMMENT remains the disposition; production/D4, merge readiness, and overall historical-task completion are outside this approval.**
