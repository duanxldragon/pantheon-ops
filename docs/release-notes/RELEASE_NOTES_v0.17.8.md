# Pantheon Base v0.17.8 — Release Notes

Patch release on the `release/0.17` line. Supersedes `pantheon-base-v0.17.7`.

Two dependency facts move, and both are base-owned: the Go directive in `backend/go.mod` and the
`zod` entry in `frontend/package-lock.json`. They are in base because the consumer cannot keep them:
`pantheon-ops` rebuilds its tree from a locked base snapshot, and neither file appears in its
`repositoryOverlayPaths` or `replaceRoots`, so a consumer-side bump is overwritten by the next
`rebuild:from-base`.

## Summary

- **No schema change.** The migration chain still ends at `000030_tenant_plan_definitions` (schema 30).
- **No route, permission code, menu seed, i18n copy, config key or exported symbol changes.**
- **No product source changes.** No file under `backend/` (other than `go.mod`) or `frontend/src/` is
  touched; the runtime code is byte-identical.
- What changes is which toolchain and which dev-only dependency version the tree actually resolves,
  and therefore what the security gates say about it.

## What changed

### `backend/go.mod` — Go 1.26.9 → 1.26.10

`GO-2026-6809` ("Distrust compromised intermediate certificates in `crypto/x509`") is fixed in Go
1.26.10. `pantheon-ops` main's `Security Gates` run `40457897311`, job `Dependency Vulnerabilities`
`122035403107`, reports its four inputs as `govulncheck: success`, `npm audit (root): success`,
`npm audit (frontend): failure`, `osv call analysis: failure`. The OSV step fails because its scan of
the consumer tree reports `stdlib 1.26.9` with
`{"GO-2026-6809": {"called": true, "unimportant": false}}` — the advisory is reachable through the
certificate-verification path, so the repository's `Enforce OSV call analysis` gate refuses it.

Raising the `go` directive is the fix, not a label: CI installs the toolchain from it via
`go-version-file: backend/go.mod`, and OSV derives the reported stdlib version from the same line.
This is the same instrument this repository used on 2026-09-22 to clear seven reachable stdlib CVEs
by moving 1.26.5 to 1.26.6.

### `frontend/package-lock.json` — `zod` 4.3.6 → 4.6.7

`GHSA-mrhx-c7xc-m3w5` (high) covers `zod < 4.6.6`: memory exhaustion while validating oversized
constrained collections. It is what makes `npm audit --audit-level=high` fail for the frontend
project.

`zod` is not a direct dependency and nothing in `src/` imports it. It enters through
`eslint-plugin-react-hooks` and `zod-validation-error`, both of which accept
`^3.25.0 || ^4.0.0`, so the reachable surface is the dev toolchain — ESLint's own config validation.
The diff is three fields (`version`, `resolved`, `integrity`) on the `node_modules/zod` entry.
`npm ci` and `npm run lint` are the compatibility readings.

### `Dockerfile` — `golang:1.26.5-alpine` → `golang:1.26.10-alpine`

The image tag is a second, independent read of the Go version: workflows take it from `go.mod`, the
Dockerfile does not. Leaving it at 1.26.5 while the module requires 1.26.10 would push container
builds onto the toolchain-download path inside `go mod download`, so both move together and the
reason is written into the file. `docs/DEPLOYMENT_GUIDE.md`, `README.md` and `README.en.md` state the
same baseline, so their text moves with it.

## Compatibility

Consumers advance `foundation-release.lock.json` from `pantheon-base-v0.17.7` to
`pantheon-base-v0.17.8` and re-run `npm run rebuild:from-base`. Nothing else is required:

- No migration to run (schema unchanged at 30).
- No permission grant or menu re-seed.
- No API or DTO surface moved.
- After the rebuild, `backend/go.mod` carries `go 1.26.10` and `frontend/package-lock.json` carries
  `zod 4.6.7` in the consumer tree, because both files come from the base snapshot. Consumers that
  ran dependabot against either file will see the bump reverted by design; this release is the
  supported path.
- Consumers building containers need `golang:1.26.10-alpine` reachable; the tag exists on Docker Hub.

## Verification

- `go version` under the new directive resolves to `go1.26.10` (toolchain download succeeded).
- `go build ./...`, `go vet ./...` and `gofmt -l .` clean; `go test -count=1 ./...` reports 37
  packages `ok` and no `FAIL`.
- `go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...` → **`No vulnerabilities found`**,
  `Your code is affected by 0 vulnerabilities`. The one remaining required-module advisory
  (`golang.org/x/crypto` `GO-2026-5932`) is reported as not called.
- `osv-scanner` v2.3.3 lockfile differential: the pre-fix pair reports `stdlib 1.26.9`
  `GO-2026-6809` and `zod 4.3.6` `GHSA-mrhx-c7xc-m3w5`; the post-fix pair reports neither, leaving
  only `golang.org/x/crypto 0.57.0` `GO-2026-5932`.
- `npm audit --package-lock-only --registry=https://registry.npmjs.org --audit-level=high`:
  **1 high** on the pre-fix lock, **`found 0 vulnerabilities`** on the post-fix lock.
- Frontend `npm ci` (453 packages), `tsc -b`, `npm run build` and `npm run lint` all clean on the
  updated lock.
- Governance chain green locally: docs frontmatter, harness encoding, structure contract, task
  packet, evidence, review, adoption, doc links, doc inventory, generated artifacts, duplication and
  PR governance.
- CI: `CodeQL`, `Go Lint`, `Backend Tests` (`go test -race ./...` on Linux), `Docs Governance`,
  `Frontend Contract`, `Duplication Gate` and `Release Gate` as recorded in the publication facts.

## Known residual

- **`SonarCloud Scan` is red on `pantheon-ops`; it is not a base failure.** The step that fails is
  ops' own `.github/workflows/sonarcloud.yml` — ops' `.github` root is a consumer overlay, so that
  workflow belongs to the consumer — and its run `40457897338` on ops `main` ends with
  `Failed to query JRE metadata: GET https://api.sonarcloud.io/analysis/jres → HTTP 403 Forbidden`;
  ops also still carries 249 unresolved Sonar issues on `main`. Base has no scanner workflow and never
  had one (`git log --diff-filter=D -- .github/workflows/sonarcloud.yml` on base history is empty), so
  base's Sonar evidence is the SonarCloud App analysis plus this repository's own `SonarCloud Gate`
  release-gate job. Both were `success` on the release target `e256d1fb`, and the Sonar API reports
  quality gate `OK` with `0` unresolved issues on `main`. An earlier draft of these notes attributed
  the 403 to base as well; that attribution was carried forward from previous rounds without being
  re-measured and is corrected here instead of being left standing. Repairing the consumer's token is
  a maintainer credential action; per the standing rule this round changed no Sonar rule, project
  setting or workflow file.
- **`golang.org/x/crypto` `GO-2026-5932`** (openpgp "unsafe by design", unmaintained, no fixed
  version) remains in the required graph. Both scanners report it as not called, so it blocks no gate
  today. Whether it must be replaced by `github.com/ProtonMail/go-crypto` is a dependency-risk policy
  decision for the maintainer; it is registered in the task packet, not closed by this release.
- `go test -race` cannot run on this Windows host (cgo disabled), so the race reading belongs to the
  Linux CI job.
- `osv-scanner` honours `.gitignore` in recursive mode, and `pantheon-base/.gitignore` ignores `.tmp/`,
  which is where this round's linked worktree lived; the first scan therefore reported
  `No package sources found` and the initial diagnosis (dot-prefixed path segments being skipped) was
  later falsified by a control run. With `--no-ignore`, the consumer's own command
  (`--recursive --call-analysis=go`) runs over base's real tree and exits 0 with a single
  not-called residual, so the base side carries its own gate reading; ops main remains the
  confirmation of the aggregate gate step.
