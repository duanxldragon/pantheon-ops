# Summary — 2026-10-31-frontend-follow-redirects-audit

## What happened

`Security Gates` went red on `main` at 2026-10-31T04:36:50Z with no repository change involved.
`Dependency Vulnerabilities` reported `NPM_AUDIT_FRONTEND_RESULT=failure`: the frontend tree
resolved `follow-redirects@1.16.1` through `axios@1.20.0`, and that exact version entered the
advisory range of GHSA-mrwp-f3q8-3g9p (`>=1.16.0 <=1.16.1`, high, CWE-201). `--audit-level=high`
turns that into a non-zero exit, and the aggregate gate propagates it.

## What changed

One lock entry: `node_modules/follow-redirects` in `frontend/package-lock.json` moves from
`1.16.1` to `1.16.2` (version, resolved URL, integrity), three lines in and three lines out.
`frontend/package.json` is untouched, no `overrides` entry is added, and no source file changes.

`1.16.2` is the narrowest published version outside the advisory range and still satisfies
`axios`'s dependency range, so the resolver keeps owning the version.

Two wider tools were considered and rejected: `npm audit fix` would also re-resolve
`@typescript-eslint/*`, `i18next` and `react-i18next`, and `npm update` rewrites unrelated `libc`
fields across optional platform packages with the local npm. Both add review surface without
security value.

## Verification

- `npm ci` from the edited lock installs cleanly and `axios@1.20.0 -> follow-redirects@1.16.2`.
- Frontend audit: high 0, critical 0 (19 moderate remain, all below the gate threshold and all in
  dev-tooling chains that would need major upgrades).
- `lint`, `type-check` and `build` are clean.

## Scope note on the moderate advisories

The gate is `--audit-level=high`; the 19 moderate entries (`@typescript-eslint/*`, `eslint`,
`glob`, `minimatch`, `nyc`, `rimraf`, `vite-plugin-istanbul`) are not what made `main` red and
clearing them means jumping major versions of the lint and coverage toolchains. That is a separate,
deliberate decision and is not smuggled into an unblocking patch.
