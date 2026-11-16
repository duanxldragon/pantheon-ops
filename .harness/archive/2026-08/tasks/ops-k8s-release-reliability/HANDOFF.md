# Task Handoff: K8s Release Reliability

Read the parent handoff and completed capability contract first. This task must
use fake/disposable clients by default. Never mutate an external cluster without
an explicit maintainer gate.

## Start

1. Re-run K8s backend tests and frontend type-check; record exact baseline.
2. Freeze Release states, immutable snapshot, idempotency, rollout criteria,
   timeout, and reconciliation behavior in `STATUS.md`.
3. Confirm migration and BizScope provider contracts.
4. Implement intent-first persistence and fake-client tests before UI changes.

## Resume

Read the last rollout/failure evidence and `Next atomic action`. Reproduce with a
fake/disposable client. Never retry an ambiguous apply without reconciliation.

## Stop

Stop for external cluster mutation, credentials, base IAM/secret changes,
unapproved Namespace ownership, or unbounded rollout waits.
