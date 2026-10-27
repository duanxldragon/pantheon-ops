# pantheon-ops Documentation Index

Chinese version: [README.md](./README.md)

`pantheon-ops` is the operations-domain business repository built on top of `pantheon-base`. It inherits the base architecture, contracts, and acceptance standards from `pantheon-base`.

The default inheritance mode is now to consume `pantheon-base` foundation releases instead of tracking `main` directly.

This directory contains only repository-owned business documents. Architecture, contracts, UI rules, and shared acceptance documents should come directly from `pantheon-base/docs/` and must not be duplicated into ops.

## Repository-Owned Documents

- [README.md](./README.md) / [README.en.md](./README.en.md): repo entry and reading order
- [../DESIGN.md](../DESIGN.md) / [../DESIGN.en.md](../DESIGN.en.md): repository-level product, experience, boundary, and release contract
- [designs/BUSINESS_FUNCTIONAL_DESIGN.md](./designs/BUSINESS_FUNCTIONAL_DESIGN.md): global capability map, flows, state, security, UI, and V1 traceability (Chinese canonical)
- [V1_RELEASE_CHECKLIST.md](./V1_RELEASE_CHECKLIST.md): only source of truth for V1 release state, blockers, gates, and runtime evidence
- [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md) / [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md): inheritance relationship, version pin, local business scope, and override boundaries
- [../.agents/skills/README.zh.md](../.agents/skills/README.zh.md) / [../.agents/skills/README.md](../.agents/skills/README.md): repo-local agent skills for inheritance checks, PR closure, GitHub comment automation, and CI triage
- [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md) / [TASK_PACKET_OPS_TEMPLATE.en.md](./TASK_PACKET_OPS_TEMPLATE.en.md): minimum task-packet template for business work and `base -> ops` sync
- [designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md](./designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md) / [designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md](./designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md): business-scope module design for scope governance, host assignment boundaries, and deploy trust source
- [designs/BUSINESS_CMDB_MODULE_DESIGN.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.md) / [designs/BUSINESS_CMDB_MODULE_DESIGN.en.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.en.md): CMDB business-module design
- [designs/BUSINESS_DEPLOY_MODULE_DESIGN.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.md) / [designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md): deploy business-module design
- [designs/BUSINESS_SERVICE_MODULE_DESIGN.md](./designs/BUSINESS_SERVICE_MODULE_DESIGN.md) / [designs/BUSINESS_SERVICE_MODULE_DESIGN.en.md](./designs/BUSINESS_SERVICE_MODULE_DESIGN.en.md): application, service, instance, VM/K8s target, and reconciliation design
- [designs/BUSINESS_K8S_MODULE_DESIGN.md](./designs/BUSINESS_K8S_MODULE_DESIGN.md) / [designs/BUSINESS_K8S_MODULE_DESIGN.en.md](./designs/BUSINESS_K8S_MODULE_DESIGN.en.md): cluster, NamespaceBinding, runtime resources, Release, and V1 Service design
- [designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md) / [designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md): canonical business error-key appendix
- [designs/PLATFORM_SRE_EVOLUTION_PLAN.md](./designs/PLATFORM_SRE_EVOLUTION_PLAN.md) / [designs/PLATFORM_SRE_EVOLUTION_PLAN.en.md](./designs/PLATFORM_SRE_EVOLUTION_PLAN.en.md): SRE evolution roadmap
- [../frontend/README.md](../frontend/README.md) / [../frontend/README.en.md](../frontend/README.en.md): frontend workspace notes
- [../frontend/tests/smoke/README.md](../frontend/tests/smoke/README.md): smoke-test layout and coverage boundaries
- [../CONTRIBUTING.md](../CONTRIBUTING.md) / [../CONTRIBUTING.en.md](../CONTRIBUTING.en.md): collaboration and verification rules
- [../SECURITY.md](../SECURITY.md) / [../SECURITY.en.md](../SECURITY.en.md): security reporting policy

Chinese `.md` files remain the primary reading surface for this repository, while `.en.md` companions support international collaboration and future expansion.
`PROJECT_INHERITANCE` now follows the same pattern as the rest of the docs: `PROJECT_INHERITANCE.md` is the Chinese primary entry and `PROJECT_INHERITANCE.en.md` is the English companion.

For actual onboarding, start with:

1. [../README.md](../README.md)
2. [../DESIGN.md](../DESIGN.md)
3. [designs/BUSINESS_FUNCTIONAL_DESIGN.md](./designs/BUSINESS_FUNCTIONAL_DESIGN.md)
4. [V1_RELEASE_CHECKLIST.md](./V1_RELEASE_CHECKLIST.md)
5. [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)
6. the relevant `designs/BUSINESS_*_MODULE_DESIGN.md`
7. the linked task packet, evidence, and review

## Recommended Reading Order

For Chinese-first onboarding, read:

1. [README.md](./README.md)
2. [../DESIGN.md](../DESIGN.md)
3. [designs/BUSINESS_FUNCTIONAL_DESIGN.md](./designs/BUSINESS_FUNCTIONAL_DESIGN.md)
4. [V1_RELEASE_CHECKLIST.md](./V1_RELEASE_CHECKLIST.md)
5. [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)
6. the relevant module design

Then read foundation docs directly in `pantheon-base`:

1. `../../pantheon-base/DESIGN.md`
2. `../../pantheon-base/AGENTS.md`
3. `../../pantheon-base/docs/README.md`
4. matching base contracts, designs, and acceptance docs

## Governance Rule

- ops may add only business-owned documents
- base architecture, contract, UI, and shared acceptance docs must change in `pantheon-base`
- if a generic rule needs to change, open the change in base first and then upgrade ops
- for non-trivial business implementation or `base -> ops` sync work, prefer creating a minimal task packet first
- `docs/harness/tasks/` is reserved for task-packet instances and evidence linkage only; it is not part of the business-doc primary index

`docs/harness/` is the reviewable Harness contract and task-packet layer, while `.harness/` stores machine-readable state and evidence. They intentionally remain separate because repository checkers consume both paths. Superseded process documents are retained under `../archive/2026-08-process/` for traceability only.
