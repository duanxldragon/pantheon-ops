# Pantheon Base v0.15.0 — Release Notes

**Release date**: 2026-10-11
**Tag**: `pantheon-base-v0.15.0`
**Previous release**: `pantheon-base-v0.14.0`

## Summary

v0.15.0 is a security, quality and release-hygiene closeout on top of the v0.14.0
multi-tenant foundation. It carries no breaking changes and requires no migration
beyond the v0.14.0 upgrade path.

## Security

- **SSRF IPv4 denylist rebuilt from byte literals.** `backend/pkg/security/ssrf/validator.go`
  now constructs the reviewed RFC 1122/1918/3927/5771/8030 denylist with
  `netip.PrefixFrom(netip.AddrFrom4(...))` instead of CIDR string literals. The denylist
  behavior is unchanged (loopback, RFC1918, link-local/cloud-metadata, multicast and
  reserved ranges are still rejected) while clearing the 6 remaining `go:S1313`
  SonarCloud findings on `main`.

## Quality

- **SonarCloud back to zero open issues.** All 28 previously open issues were cleared;
  the `S1313` remainder is now resolved, so the Release Gate's "zero unresolved issues"
  rule is met without a waiver.
- **CodeQL and Dependabot clean** on the release candidate (0 open error/critical and
  0 high/critical alerts).

## CI / Reliability

- **Narrow-viewport smoke test stabilized.** `tests/smoke/platform/shell-top-panels.spec.ts`
  now waits for the shell popups to settle before measuring their boxes, removing a
  measurement race that intermittently failed the `390×844` containment assertions. The
  containment bound is unchanged — only the measurement timing was fixed.

## Documentation

- README (zh/en) milestone/maturity narrative removed; version tables re-pointed at
  `pantheon-base-v0.15.0`.
- Superseded delivery/review/planning documents archived under `docs/history/2026-10/`.
- `.harness` task status reconciled with the release closeout.

## Verification

- Backend: `go build ./...`, `go vet ./...`, `go test ./pkg/security/ssrf/...` green.
- Frontend: `npm run type-check` green.
- Hosted: required checks + Release Gate green on the release commit.

## Consumer impact (`pantheon-ops`)

The consumer lock, inheritance snapshot and business smoke remain a separate
`pantheon-ops` follow-up; no base-side interface changed in this release.
