# Ops Source Ownership Classification

## Method

- Compare each candidate path with `foundation-release.lock.json` at `pantheon-base-v0.10.12` (`16918771e2650f8c045b0e086144eb290e774704`).
- Apply `docs/PROJECT_INHERITANCE.md`: only `frontend/src/modules/business/**` is operations business-owned; generic platform and shared UX code must return to `pantheon-base`.
- Do not classify SonarCloud findings solely by which repository currently contains their source.

## Legitimate Operations Business Source

- `frontend/src/modules/business/cmdb/**`
- `frontend/src/modules/business/deploy/**`
- `frontend/src/modules/business/bizscope/**`

The baseline contains 156 findings in this category. These are genuine operations-domain remediation candidates, although most are historical complexity and duplication debt rather than Bugs or Vulnerabilities.

## Locked Foundation Source

- 23 baseline findings are in paths consumed from the locked foundation release.
- The source is inherited, while SonarCloud issue dispositions are project-scoped and do not transfer from `pantheon-base` to `pantheon-ops`.
- Do not patch those files locally to suppress an ops-only SonarCloud result.

## Missed Foundation Consumption Candidates

The following paths are generic frontend application or transport concerns, are not within `modules/business/**`, and are not covered by the current frontend shared-path lock:

| Path | Classification | Required disposition |
|---|---|---|
| `frontend/src/App.tsx` | Application shell, authentication bootstrap, menu redirect, and route permission integration | Move ownership to `pantheon-base`; include in a future foundation release. |
| `frontend/src/main.tsx` | Application bootstrap, i18n, theme, and public-settings initialization | Move ownership to `pantheon-base`; include in a future foundation release. |
| `frontend/src/hooks/usePermission.ts` | Shared authorization helper | Move ownership to `pantheon-base`; include in a future foundation release. |
| `frontend/src/api/file.ts` | Generic file download and CSV export helper | Reconcile with the base helper, including CSV formula neutralization, then release from `pantheon-base`. |
| `frontend/src/api/upload.ts` | Generic multipart upload helper | Reconcile timeout/error-policy behavior in `pantheon-base`, then release. |
| `frontend/src/api/importExport.ts` | Generic import/export presentation and file helper | Reconcile with the base helper, then release. |

These paths are historical foundation-consumption omissions, not operations business code. Their local presence explains part of the apparently larger ops frontend tree, but it does not authorize copying or repairing generic shared behavior only in ops.

## Scan-Mechanism Contribution

| Baseline category | Findings | Interpretation |
|---|---:|---|
| Tooling and workflows | 195 | Automatic-analysis scope difference plus separately triaged CI security. |
| SQL and i18n resource maps | 91 | Known base-excluded analysis noise. |
| Locked foundation source | 23 | Separate SonarCloud-project disposition mismatch. |
| Unlocked generic/common source | 21 | Base-consumption review candidates. |
| Business source | 156 | Real operations ownership. |

The additional findings are therefore not attributable solely to business feature volume. Scope configuration, independent project dispositions, and missed foundation-consumption paths are material contributors.
