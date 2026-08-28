# Pantheon Ops - Operations Business Platform Design

Chinese source: [DESIGN.md](./DESIGN.md)

## Source of truth

- **Status:** Active; V1 remains `BLOCKED - NOT READY TO FREEZE`
- **Last refreshed:** 2026-08-24
- **Primary surfaces:** business scopes, service catalog, CMDB, deployment, Kubernetes
- **Functional design:** [Global Functional Design](./docs/designs/BUSINESS_FUNCTIONAL_DESIGN.en.md)
- **Freeze decision:** [V1 Release Freeze Checklist](./docs/V1_RELEASE_CHECKLIST.md)
- **Foundation:** [Pantheon Base Design](../pantheon-base/DESIGN.en.md)

This document is the repository-level product and experience contract for `pantheon-ops`. It owns only Ops `business/*` behavior. The platform shell, authentication, IAM, organization, configuration, audit, i18n, upload, and shared UI remain owned by `pantheon-base`.

Design intent is not implementation status. Use the status labels in the global functional design and the V1 freeze checklist to distinguish implemented behavior, freeze targets, exclusions, and post-V1 work.

## Brand

- **Personality:** calm, trustworthy, and tool-oriented for repeated operational work.
- **Trust signals:** truthful state, explicit ownership, traceable actions, recoverable tasks, redacted output, and reproducible evidence.
- **Avoid:** marketing-style composition, decorative gradients, nested card stacks, fake live state, controls without real behavior, and high-risk actions presented as ordinary CRUD.

## Product goals

- Connect responsibility to runtime targets through `BizScope -> Application -> Service -> ServiceInstance -> Resource`.
- Make VM and Kubernetes changes permissioned, scoped, snapshotted, idempotent, auditable, recoverable, and evidence-backed.
- Keep the V1 modular monolith while isolating business modules behind typed capabilities.
- Derive release state from reproducible gates and evidence rather than historical completion claims.

V1 does not deliver a complete observability/incident platform, split the product into microservices, introduce a general workflow engine, or duplicate Base-owned platform contracts.

## Personas and jobs

| Persona                | Primary job                                                           | Constraint                                                                   |
| ---------------------- | --------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Platform administrator | Configure permissions, menus, data scope, and release baseline        | Must preserve Base IAM/audit and the human freeze gate                       |
| SRE / operator         | Manage resources, deploy software, operate clusters, recover failures | Needs explicit targets, live feedback, idempotency, and traceable logs       |
| Business owner         | Maintain scope, application, service, and ownership context           | Sees only owned scope and never handles raw credentials                      |
| Security / auditor     | Review sensitive actions, access, secrets, and evidence               | Needs immutable events, correlation IDs, redaction, and export authorization |
| Release maintainer     | Review freeze gates and runtime cleanup                               | Only the maintainer may set `FROZEN`                                         |

## Information architecture

```text
Operations Platform
|-- Business Scopes
|-- Service Catalog
|-- CMDB
|   |-- Hosts
|   |-- Dynamic Groups
|   `-- Label Schema
|-- Deployment
|   |-- Packages
|   |-- Templates
|   `-- Tasks
`-- Kubernetes
    |-- Clusters
    |-- Workloads
    `-- Releases
```

Namespace, node, ConfigMap, Secret, pod-log, and related context belongs inside cluster/workload detail surfaces. Kubernetes Service CRUD is a V1 freeze target, not a completed feature. Observability is excluded from default V1 navigation and runtime.

## Design principles

1. Separate facts from intent: catalog owns responsibility, CMDB/Kubernetes own resource facts, and Deploy owns change intent and history.
2. Enforce permission before interaction: navigation, page, data, and action access are separate decisions, with the backend as the final gate.
3. Explain dangerous actions with target, impact, and preconditions.
4. Treat loading, empty, error, forbidden, conflict, partial failure, and recovery as first-class states.
5. Snapshot targets, templates, versions, and execution parameters to preserve history.
6. Use typed capabilities and stable references across modules; never share ORM models or raw table access.

## Visual language

- Reuse Base semantic color, typography, spacing, radius, elevation, icon, and motion tokens.
- Keep operational surfaces dense but calm; prioritize scanability, comparison, and repeated action.
- Use status text/icons in addition to color, stable layouts during loading, and no nested section cards.
- Use 150-300ms purposeful motion and never use animation to imply progress that the backend has not reported.

## Components

- Reuse Base page, filter, table, form, modal, submit, and page-state components.
- Ops-owned compositions include resource-reference selectors, target snapshot previews, state-transition controls, task event streams, redacted log viewers, and resource-version conflict guidance.
- Shared components and tokens return to Base; components that encode Ops business meaning stay under `frontend/src/modules/business/*`.

## Accessibility

- Follow Base accessibility requirements for keyboard access, visible focus, semantic labeling, contrast, and screen-reader announcements.
- Do not place primary actions behind hover-only controls.
- Keep task updates from stealing focus and respect reduced-motion preferences.

## Responsive behavior

- Desktop is the primary work surface; narrow desktop, tablet, and mobile must still support status inspection and critical recovery.
- Wide tables use controlled scrolling or column priority; detail views collapse to one column without fixed controls covering content.
- Touch layouts preserve impact summaries and confirmations for dangerous actions.

## Interaction states

- Loading preserves layout and prevents duplicate mutation.
- Empty distinguishes initial, filtered, and inaccessible states.
- Errors preserve entered data and offer recovery tied to stable error semantics.
- Accepted asynchronous requests are not described as successful execution.
- Disabled actions expose their state, permission, or dependency reason.
- Disconnected live streams are marked stale and fall back to documented polling.

## Content voice

- Use direct, precise, actionable language.
- Standardize terms around scope, application, service, service instance, resource, target snapshot, task, attempt, step, event, and release.
- Include object, count, and irreversible impact in destructive confirmations.
- Distinguish accepted, running, successful, partially failed, evidence-incomplete, ready for freeze, and frozen.

## Implementation constraints

- Inherit Go/Gin/GORM, React/TypeScript, Arco Design, and Pantheon tokens from Base; add no UI library.
- Keep the V1 modular monolith and assemble business modules through the overlay registry.
- Apply schema changes only through paired versioned migrations.
- Use `CredentialRef`; never expose secret values in API, logs, audit, export, or evidence.
- Bound external calls and concurrency; guard transitions with version/CAS; use `resourceVersion` for Kubernetes mutations.
- Rebuild from the pinned Base foundation and validate implementation against the freeze checklist and external runtime matrix.

## Release and roadmap

| Stage | Scope                                                                      | Decision                         |
| ----- | -------------------------------------------------------------------------- | -------------------------------- |
| V1    | Safe, recoverable BizScope, Service, CMDB, Deploy, and Kubernetes workflow | `BLOCKED`; see freeze checklist  |
| P1    | Generic resource fact center and authorization/audit/reliability ratchets  | Starts after V1 freeze           |
| P2    | Observability, GitOps, alerting, incidents, and SLOs                       | Excluded from default V1 runtime |

## Open questions

- [ ] The maintainer must authorize and record isolated VM and Kubernetes runtime targets.
- [ ] After V1, define the generic Resource migration window and compatibility projection lifetime.
- [ ] Observability requires a separate design review, permission matrix, migrations, and complete frontend flow before default enablement.

## Document reading order

1. `DESIGN.md`
2. `AGENTS.md`
3. `docs/README.md`
4. `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`
5. `docs/V1_RELEASE_CHECKLIST.md`
6. `docs/PROJECT_INHERITANCE.md`
7. Relevant Base contracts, designs, and acceptance documents
8. Relevant `docs/designs/BUSINESS_*` module source
9. Linked `.harness` task, status, evidence, and review

Ops does not maintain copies of Base contracts, acceptance documents, or platform designs. Historical process material under `archive/2026-08-process/` never overrides current freeze evidence.
