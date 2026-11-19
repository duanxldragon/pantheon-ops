# C04 Service UI continuation

Tier: L2. Parent packet: `.harness/tasks/2026-11-16-ops-k8s-service-mutations/continuation-plan.md`.

Owner: C04 UI executor; no backend, shared-system, generated fallback, runtime, archive or goal ownership. Other executors share this working tree; their changes are preserved.

Surface: operational table and responsive edit/create form. Intended feel: compact, calm routing controls using existing Pantheon components and tokens.

1. Add UID/resourceVersion and numeric-or-named targetPort to the Service API; add PUT and preconditioned DELETE.
2. Reuse the current modal and table, add a labelled multi-port editor, preserve read identity, selectorless Services and allocated node ports. Reject malformed/duplicate selector entries; remove nodePort explicitly when switching to ClusterIP.
3. Gate create/update/delete separately; keep draft on provider failure and rely on shared request feedback once. Provide explicit list refresh, never silently refresh a stale version.
4. Synchronize only C04 canonical locale keys and error appendix. Backend owner synchronizes seed; coordinator rebuilds generated fallback.
5. Verify scoped ESLint, production TypeScript, Playwright discovery, JSON values and source diff. Coordinator runs actual browser/runtime and inspects screenshots at 1440x900 and 390x844.

Acceptance still requiring coordinator evidence: browser execution, rendered primary/list/permission states, live provider and actual role/environment admission, full CGO race, official-lock rebuild and final review. Local static checks and mocked scenarios do not close these gates.
