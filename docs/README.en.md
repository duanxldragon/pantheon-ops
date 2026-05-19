# pantheon-ops Documentation Index

Chinese version: [README.md](./README.md)

`pantheon-ops` is the operations-domain business repository built on top of `pantheon-base`. It inherits the base architecture, contracts, and acceptance standards from `pantheon-base`.

This directory contains only repository-owned business documents. Architecture, contracts, UI rules, and shared acceptance documents should come directly from `pantheon-base/docs/` and must not be duplicated into ops.

## Repository-Owned Documents

- [README.md](./README.md) / [README.en.md](./README.en.md): repo entry and reading order
- [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md) / [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md): inheritance relationship, version pin, local business scope, and override boundaries
- [designs/BUSINESS_CMDB_MODULE_DESIGN.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.md) / [designs/BUSINESS_CMDB_MODULE_DESIGN.en.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.en.md): CMDB business-module design
- [designs/BUSINESS_DEPLOY_MODULE_DESIGN.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.md) / [designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md): deploy business-module design
- [designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md) / [designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md](./designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md): canonical business error-key appendix
- [designs/PLATFORM_SRE_EVOLUTION_PLAN.md](./designs/PLATFORM_SRE_EVOLUTION_PLAN.md) / [designs/PLATFORM_SRE_EVOLUTION_PLAN.en.md](./designs/PLATFORM_SRE_EVOLUTION_PLAN.en.md): SRE evolution roadmap
- [../DESIGN.md](../DESIGN.md) / [../DESIGN.en.md](../DESIGN.en.md): repo-level design stance
- [../frontend/README.md](../frontend/README.md) / [../frontend/README.en.md](../frontend/README.en.md): frontend workspace notes
- [../frontend/tests/smoke/README.md](../frontend/tests/smoke/README.md) / [../frontend/tests/smoke/README.en.md](../frontend/tests/smoke/README.en.md): smoke-test layout and coverage boundaries
- [../CONTRIBUTING.md](../CONTRIBUTING.md) / [../CONTRIBUTING.en.md](../CONTRIBUTING.en.md): collaboration and verification rules
- [../SECURITY.md](../SECURITY.md) / [../SECURITY.en.md](../SECURITY.en.md): security reporting policy

Chinese `.md` files remain the primary reading surface for this repository, while `.en.md` companions support international collaboration and future expansion.
`PROJECT_INHERITANCE` now follows the same pattern as the rest of the docs: `PROJECT_INHERITANCE.md` is the Chinese primary entry and `PROJECT_INHERITANCE.en.md` is the English companion.

For actual onboarding, start with:

1. [../README.md](../README.md)
2. [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)
3. the `designs/BUSINESS_*` documents

## Recommended Reading Order

For Chinese-first onboarding, read:

1. [README.md](./README.md)
2. [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)
3. [designs/BUSINESS_CMDB_MODULE_DESIGN.md](./designs/BUSINESS_CMDB_MODULE_DESIGN.md)
4. [designs/BUSINESS_DEPLOY_MODULE_DESIGN.md](./designs/BUSINESS_DEPLOY_MODULE_DESIGN.md)

Then read foundation docs directly in `pantheon-base`:

1. `../../pantheon-base/DESIGN.md`
2. `../../pantheon-base/AGENTS.md`
3. `../../pantheon-base/docs/README.md`
4. matching base contracts, designs, and acceptance docs

## Governance Rule

- ops may add only business-owned documents
- base architecture, contract, UI, and shared acceptance docs must change in `pantheon-base`
- if a generic rule needs to change, open the change in base first and then upgrade ops
