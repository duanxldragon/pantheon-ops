---
title: Ship a full repository snapshot in the foundation release
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-13
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-13-foundation-release-repo-snapshot

## Goal

Publish a `git archive` full-repository `repo.tar` snapshot alongside the foundation release bundle, so a consumer rebuilds deterministically from a locked release without a live `../pantheon-base` working tree.

## Scope

### In

- Generate `repo.tar` from `git archive --format=tar <baseCommit>` in `build-release-bundle.mjs`.
- Emit a `repo.tar.sha256` sidecar and pack both into the release `.tgz`.
- Upload `repo.tar` + `repo.tar.sha256` as standalone GitHub release assets.
- Declare the `repoSnapshot` asset shape in the release manifest.
- Add bundle regression coverage: byte-identical to `git archive`, sha256 match, `config/`/`database/`/`schema/` present, packed into the `.tgz`.

### Out

- Ops business behavior or schema changes.
- Consumer-local overrides.
- Mutation of already-published `pantheon-base-v0.10.20` assets (this lands in v0.10.21+).

## Stop Conditions

- Stop if the snapshot is not byte-identical to `git archive` of the manifest baseCommit.
- Stop if `repo.tar.sha256` does not match the produced `repo.tar`.
- Stop if the release `.tgz` omits `repo.tar`/`repo.tar.sha256`.

## Linkage

- Task ID: `2026-08-13-foundation-release-repo-snapshot`
- Task Manifest: `.harness/tasks/2026-08-13-foundation-release-repo-snapshot/manifest.json`
- Evidence Directory: `.harness/evidence/2026-08-13-foundation-release-repo-snapshot/`
- Review File: `.harness/evidence/2026-08-13-foundation-release-repo-snapshot/review.md`
