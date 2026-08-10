---
title: Consume pantheon-base-v0.10.5 smoke runner contract
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-08
---

# Task Packet: 2026-08-08-foundation-upgrade-base-v0-10-5

Target repository: `pantheon-ops`  
Layer: `inheritance-sync (pantheon-base -> pantheon-ops)`  
Delivery tier: `L2`  
Mode: immutable release consumption and PR closeout

## Goal

Consume `pantheon-base-v0.10.5`, which distributes the shared smoke runner test and its three direct fixtures omitted by v0.10.4. Keep all `business/*` overlays Ops-owned and remove the stale local runner fixture.

## Landing-Side Decision

Base owns the release producer and shared smoke closure. Ops owns the exact consumer allowlist, lock, inheritance evidence, and business overlays. No shared behavior is hand-copied into Ops.

## Source Release

- Base commit: `b97c0f6d288e2c984fbd9215d6d3626929f68d85`
- Archive SHA-256: `a49457b96ec1ff18bd716c149708433798e5e99db8698fa686f6346c26f555f0`

## Verification Plan

- `npm run check:inheritance`
- `npm run test:foundation-release`
- `npm --prefix frontend run test:smoke:scripts`
- `npm --prefix frontend run lint`
- `npm --prefix frontend run type-check`
- `npm --prefix frontend run build`
- `$env:PATH='D:\\msys64\\mingw64\\bin;' + $env:PATH; $env:CGO_ENABLED='1'; go test -race ./...`

## Stop Points

- checksum or release identity mismatch
- consumer attempts to replace `business/*`
- hosted required check failure requiring a quality-gate decision

## Linkage

- Task manifest: `.harness/tasks/2026-08-08-foundation-upgrade-base-v0-10-5/manifest.json`
- Evidence: `.harness/evidence/2026-08-08-foundation-upgrade-base-v0-10-5/`
- PR: `https://github.com/duanxldragon/pantheon-ops/pull/75`
