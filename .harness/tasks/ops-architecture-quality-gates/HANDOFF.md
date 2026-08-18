# Task Handoff: Quality Gates

Read the parent handoff and all completed child summaries. Checker bootstrap may
start after capability contracts freeze, but final acceptance waits for every
dependency. This is a suitable task for cost-aware parallel lanes with a strong
final reviewer.

## Start

1. Record current Go, frontend, overlay, and architecture-scan baseline.
2. Freeze checker allowlist policy and negative fixtures before implementation.
3. Map every child acceptance invariant to one command/test/evidence artifact.
4. Assign product failures back to owning tasks; do not weaken predicates/gates.

## Resume

Read `STATUS.md`, failed command evidence, and allowlist review. Continue from
`Next atomic action`; rerun the smallest failing gate. Preserve unrelated changes.

## Stop

Stop for shared CI changes, credentials/external operations, broad exclusions, or
a product behavior mismatch that belongs to another task.
