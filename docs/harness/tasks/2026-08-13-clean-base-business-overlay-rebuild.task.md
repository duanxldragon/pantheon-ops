---
title: Rebuild Ops from a clean Base snapshot
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-13
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - business-overlay.json
---

# Task Packet: 2026-08-13-clean-base-business-overlay-rebuild

## Goal

Replace the historical file-by-file foundation consumer with a deterministic clean rebuild: a complete Pantheon Base snapshot plus an explicitly declared Ops business overlay.

## Scope

### In

- Preserve `business/{bizscope,cmdb,deploy}`, business smoke, business design documents, and Ops repository governance.
- Keep the Base repository layout and Go module identity in the rebuilt product tree.
- Generate backend module, frontend module, component, menu component, and business fallback registries from one manifest.
- Prove the rebuild is deterministic and contains no undeclared generic product overlay.
- Validate in a temporary tree before replacing the current Ops baseline.

### Out

- New business behavior, schema, or UI design.
- Base product changes or another Base patch release.
- Retaining historical shared-source drift merely because it exists in the old Ops tree.

## Structural Scope

- Affected subgraph: Base snapshot installation -> business overlay copy -> generated registries -> backend/frontend runtime assembly.
- Boundary crossings: repository governance to product tree; `business/*` to Base public contracts; menu, permission, i18n, and smoke registration.
- Risk nodes: undeclared product files, incomplete business asset inventory, stale Go module imports, nondeterministic generation.
- Graph focus: `InitGeneratedBusinessModules`, `generatedBusinessModules`, `generatedComponentRegistry`, and `generatedMenuComponentKeys`.

## Minimum Viable Approach

Use the existing Base extension points and Node standard library. Add one machine-readable manifest and one deterministic rebuild tool; do not introduce a package, template engine, or parallel synchronization path.

## Success Criteria

- Two rebuilds from identical inputs produce identical reports and files.
- Generic product paths cannot be claimed by the Ops overlay.
- Business sources are preserved except for the intentional Go module import rewrite.
- Backend race, frontend lint/type/build, inheritance/overlay checks, and business smoke pass in the rebuilt tree.
- The current Ops tree is replaced only after staged evidence is green.

## Linkage

- Task ID: `2026-08-13-clean-base-business-overlay-rebuild`
- Change identity: `none`
- Evidence: `.harness/evidence/2026-08-13-clean-base-business-overlay-rebuild/summary.md`
- Review: `.harness/evidence/2026-08-13-clean-base-business-overlay-rebuild/review.md`

## Roles And Stop Points

- Implementer: build the manifest, generator, staged tree, and verification evidence.
- Reviewer: findings-first review of ownership completeness, determinism, and boundary enforcement.
- Human gate: final product/runtime acceptance after the atomic baseline replacement.
- Stop if staged business assets differ unexpectedly, a generic shared file requires an Ops override, or a required gate cannot be reproduced.
