# Task Handoff: State Machines

Read the parent handoff and completed Deploy/ServiceInstance summaries first. This
task has one state-contract owner. Lower-cost tools may add matrix tests or UI only
after the owner freezes transitions.

## Start

1. Sample current Host status data and record the legacy mapping decision.
2. Freeze Host lifecycle/connectivity and Service desired/observed/health states.
3. Write the full action transition matrix, CAS/version, callback, timeout, and
   rollback rules in `STATUS.md` before editing code.
4. Implement owner services and tests before wiring UI actions.

## Resume

Read the state matrix and failing transition evidence first. Reproduce a stale or
concurrent transition before editing predicates. Never merge Host and service state
to avoid schema work.

## Stop

Stop for unsafe legacy mapping, unpublished dependency contracts, base workflow
changes, or a transition requirement not covered by the approved matrix.
