---
title: Repository Layout, Naming, and Layer Boundaries
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-09-23
---

# Repository Layout, Naming, and Layer Boundaries

Chinese version: [REPOSITORY_LAYOUT.md](./REPOSITORY_LAYOUT.md)

This document is the single authoritative standard for `pantheon-base` directory layout, file naming, controlled exceptions, and layer dependency directions. Any new directory, new file suffix, generated-artifact rule change, or cross-layer dependency must update this document first, then the gate whitelists, CI steps, documentation references, and tests.

`scripts/harness/check-structure-contract.mjs` takes its contract from sections 2 and 5 of this document; `scripts/harness/check-boundaries.mjs` enforces the import directions in section 8. The two are complementary: changing one requires syncing the other.

## 1. Root Groups

```text
backend/                  # Go backend: go.mod, entrypoint, domain modules, shared packages, migrations, performance tests
frontend/                 # React frontend: shell, page modules, smoke tests, fixtures, frontend scripts
docs/                     # Current docs, contracts, designs, acceptance docs, harness specs
scripts/                  # Root automation, GitHub collaboration, harness checks, release scripts
tests/                    # Root Node script tests, docs tests
.harness/                 # Runtime governance state (evidence/manifests created per task)
.agents/                  # repo-local agent notes, skills, schemas
.codex/                   # Codex repository config
.github/                  # GitHub workflows, templates, CODEOWNERS, Dependabot
.githooks/                # Local git hooks
config/                   # Method-chain config, currently config/method.config.json
database/                 # Docker Compose first-run SQL initialization
grafana/                  # Local Prometheus/Grafana observability config
openspec/                 # OpenSpec skeleton and entry notes
schema/generated/         # Generated cross-surface governance outputs
```

Root files fall into four groups:

- entry docs: `README.md`, `DESIGN.md`, `AGENTS.md`, `SECURITY.md`, `CHANGELOG.md`, `VERSION`
- build and dependency manifests: `package.json`, `package-lock.json`, `Dockerfile`, `docker-compose.yml` (Go's `go.mod`/`go.sum`/`.golangci.yml` live under `backend/`)
- security and quality config: `.gitleaksignore`, `.gitattributes`, `.gitmessage`
- local example/config entries: `.env.example`, `.mcp.json`, `SHELL_VERSION.json`

## 2. Placement Rules and Directory Whitelist

### 2.1 Placement Rules

1. Backend product code belongs under `backend/modules/` or `backend/pkg/`; do not create new business directories at the repository root. Backend performance/load test scripts belong under `backend/tests/performance/`.
2. Frontend runtime code belongs under `frontend/src/`; frontend scripts belong under `frontend/scripts/`; frontend smoke tests and test fixtures belong under `frontend/tests/`; frontend unit tests belong under `frontend/tests/unit/` (tests must not live in `frontend/src/`); `vitest.config.ts` is a legal frontend root config file (alongside the vite/playwright configs); `frontend/src/index.ts` is the public export entrypoint for the library build.
3. Root automation belongs under `scripts/`; matching tests belong under `tests/scripts/`.
4. Harness method checks belong under `scripts/harness/`; task manifests and evidence belong under `.harness/`.
5. Active architecture and governance docs belong under `docs/designs/`, `docs/contracts/`, and `docs/acceptances/`; phase audits, assessments, and process records are not committed — they stay with task evidence under `.harness/` and are cleaned up when a task closes.
6. Generated bundle output belongs under `dist/` and must not be committed.
7. `database/system_init.sql` remains in `database/` because `docker-compose.yml` references that stable path.
8. Local Grafana/Prometheus observability config remains in `grafana/` so it stays separate from application runtime code.

### 2.2 Directory Whitelist (the gate's decision basis)

| Scope | Allowed directories | Allowed files |
| --- | --- | --- |
| `backend/` top level | `cmd`, `internal`, `modules`, `pkg`, `tests` | `go.mod`, `go.sum`, `start-dev.sh`, `start-dev.bat`, `DEV_DB_INIT_GUIDE.md`, `.golangci.yml` |
| `backend/modules/` domains | `auth`, `business`, `lowcode`, `platform`, `system` | see 5.1 |
| `frontend/` top level | `config`, `public`, `scripts`, `src`, `tests` | `.eslintrc.json`/`.gitignore`/`.prettierignore`/`.prettierrc`, `AGENTS*.md`, `README*.md`, `eslint.config.js`, `index.html`, `package(-lock).json`, `playwright*.config.ts`, `tsconfig*.json`, `vite.config.ts`, `vitest.config.ts` |
| `frontend/src/` top level | `api`, `assets`, `components`, `core`, `hooks`, `i18n`, `modules`, `store` | `App.tsx`, `index.css`, `index.ts`, `main.tsx`, `vite-env.d.ts` |
| `frontend/src/modules/` domains | `auth`, `business`, `generated`, `lowcode`, `platform`, `system` | see 5.2 |

Any new directory outside the whitelist is a violation by default. When one is genuinely needed, follow the change procedure in section 10 and update this document first.

## 3. Local Noise Directories

These directories are not part of the repository structure and are ignored by `.gitignore`; they can be cleaned locally when a tidy root is needed, but they should not be committed:

```text
.claude/
.codegraph/
.husky/
.tmp/
.worktrees/
node_modules/
frontend/node_modules/
frontend/dist/
frontend/test-results/
dist/
uploads/
backend/uploads/
```

`.tmp/` holds temporary logs, downloaded CI artifacts, smoke executables, and local security scan output. `uploads/` and `backend/uploads/` hold local runtime upload data. `dist/` is generated foundation release bundle output.

## 4. Directories Not Moved

The following directories add root entries, but they are stable contracts and should not be moved just to reduce the top-level count:

- `config/`: harness sync and check scripts read `config/method.config.json`.
- `database/`: `docker-compose.yml` mounts `database/system_init.sql` directly.
- `.harness/`: task evidence and method execution records need a fixed automation location.
- `schema/generated/`: cross-surface capability ledgers are read by governance flows.

If these directories are consolidated later, scripts, CI, documentation references, and tests must be updated together. Moving files alone is not a valid cleanup.

## 5. File Naming Vocabulary

The vocabulary in this section is the canonical source for file suffixes. Gates enforce only part of it (see section 10), but new files should follow the full vocabulary so the tree stays predictable.

### 5.1 Backend Go

- File names are snake_case, matching `^[a-z0-9_]+\.go$`: no PascalCase, hyphens, or spaces.
- Test files `*_test.go` live in the same package and directory as the code under test; pure-function regressions use `_pure_test.go` to stay distinct from DB-backed tests.
- Responsibility suffix vocabulary:

| Suffix | Responsibility | Example |
| --- | --- | --- |
| `_handler.go` | HTTP / gin route handling | `user_handler.go` |
| `_service.go` | Domain service and orchestration | `role_service.go` |
| `_repository.go` | Persistence access | `session_repository.go` |
| `_model.go` | Persistence model / entity | `session_model.go` |
| `_dto.go` | Request / response DTOs | `dashboard_dto.go` |
| `_registry.go` | Registries (including generated) | `generated_registry.go` |
| `_seed.go` | Seed / fixture initialization | `setting_seed.go` |
| `_export.go` | Import/export and report writing | `i18n_export.go` |
| `_module.go` | Module wiring entrypoint | `module.go` |
| `_helpers.go` / `_utils.go` | Stateless helpers | `session_helpers.go` |
| `_test.go` | Tests | `health_test.go` |

- A package-level entry file may be named after its package (`system.go`, `lowcode.go`, `business.go`), but this must not be used to bypass responsibility suffixes.
- Register a new responsibility suffix in this table before using it; keep only one synonym per role (e.g. pick `_repository.go`, not both `_repo.go` and `_repository.go`).

### 5.2 Frontend TypeScript / TSX

| Category | Naming | Location | Example |
| --- | --- | --- | --- |
| Component | PascalCase `*.tsx` | `frontend/src/components/<group>/`, `frontend/src/modules/**/components/` | `PageEmpty.tsx` |
| Hook | `use<PascalCase>.ts(x)` | `frontend/src/hooks/` | `usePermission.ts` |
| Non-component module | camelCase `*.ts` | alongside its consumer | `tablePreferences.ts` |
| Test | `*.test.ts(x)` | `frontend/tests/` (not `frontend/src/`) | `router.test.ts` |
| API | camelCase `*.ts` | `frontend/src/api/` | `file.ts` |
| i18n resource | locale-named | `frontend/src/i18n/resources/` | `zh-CN.ts` |
| Generated registry | fixed names | `frontend/src/modules/generated/`, `frontend/src/core/router/` | `business.ts`, `generatedComponentRegistry.ts` |
| Generated module | business module name | `frontend/src/modules/business/<module>/` | generator output |
| Component style | same name as the component it styles (`ComponentName.css`), same directory | component dir | `TimeRangeFilter.css` |
| Group / module-level shared style | named after its **directory** (`<dir>.css`) or placed under an explicit `shared/` subdir | group dir | `auth.css`, `components/shared/list-page.css` |
| Cross-module / global shared style | centralized in `frontend/src/assets/` or global `index.css` | shared | `index.css` |

Styles have three tiers: a **component-specific** stylesheet's filename must equal the component
name (not the BEM block name, not the directory name); a **group/module-level** shared stylesheet
is named after its directory or lives under `shared/`; a **cross-module** shared stylesheet needs
an explicit shared location. BEM class names (`.time-range-filter__x`) do not change with the filename.

`index.ts` is for barrel re-exports only; it holds no implementation.

### 5.3 Documentation Naming

- Long-lived docs under `docs/` use `UPPER_SNAKE_CASE.md`, with an English companion `<NAME>.en.md`; the `.md` and `.en.md` pair must be maintained together.
- Phase materials are allowed only as `<YYYY-MM-DD>-<kebab-case>.md` under `.harness/` (task docs, evidence, reviews); do not promote them into the `docs/` first-tier index.
- Every Markdown file under `docs/`, `architecture/`, and `patterns/` must carry YAML frontmatter with the field set defined in section 6.

## 6. Document Taxonomy and Placement

Document types follow the five-type model in [Document Governance Contract](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md); frontmatter fields follow [DOCUMENT_FRONTMATTER_SCHEMA.md](../contracts/DOCUMENT_FRONTMATTER_SCHEMA.md).

| Type | Purpose | Directory | Lifecycle |
| --- | --- | --- | --- |
| `Contract` | Defines boundaries, goals, non-goals, completion | `docs/contracts/` | Long-lived, in the main index |
| `Design` | Describes how to build (must belong to a Contract) | `docs/designs/` | Long-lived, in the main index |
| `Assessment` | Gap between implementation and contract | `docs/reviews/` (if kept) or `.harness/evidence/` | Medium; delete when it has no reuse value |
| `Remediation` | How the gap will be closed | `.harness/tasks/` (as a task) or secondary `docs/` entry | Medium |
| `Acceptance` | Whether the contract is met | `docs/acceptances/` | Templates/baselines long-lived; one-off samples may be archived |

Rules:

- `Design / Assessment / Remediation / Acceptance` must carry a non-empty `linked_contracts` pointing at a real contract path.
- Retirement has only three destinations: absorbed by a formal Contract/Design/Acceptance and deleted, archived under `docs/archive/*` for sample/baseline/upgrade value, or kept because it is still explicitly referenced by the active governance chain. There is no "keep it and see" default.
- Main entries in `docs/README.md` may link only `Active` documents; per-directory indexes live in each directory's `README.md`.

## 7. Generated Artifact Rules

### 7.1 Source and Output

- Source of truth for generated capability: `system_module_registration` + `schema/generated/**.json`; the derived snapshot is `schema/generated/feature-ledger.json` (see [DESIGN.md §2.4](../../DESIGN.md)).
- Generated artifacts (**must not be hand-edited**):
  - backend: `backend/modules/business/generated_registry.go`, `backend/modules/system/iam/menu/generated_component_registry.go`, `backend/modules/business/<module>/**`
  - frontend: `frontend/src/modules/generated/*.ts`, `frontend/src/core/router/generatedComponentRegistry.ts`, `frontend/src/modules/business/<module>/**`, `frontend/src/i18n/resources/generated/**`
  - governance ledger: `schema/generated/feature-ledger.json`
- The single source of the reset templates for generated artifacts is `REGISTRY_TEMPLATES` in `frontend/scripts/cleanup-generated-modules.mjs`; do not copy an empty registry elsewhere.
- Files under `business/*` outside the generated paths above are the hand-maintained layer (module wiring, retired registrations, etc.); editing them does not require regeneration.

### 7.2 Identifiability and Drift Checks

- **Generated marker (enforced)**: the first line of every generated text artifact must be `// Code generated by pantheon low-code generator. DO NOT EDIT.`; JSON artifacts (`schema/generated/*.json`) cannot carry a comment and are identified by path and parseability. The marker string's single source is `GENERATED_MARKER` in `frontend/scripts/cleanup-generated-modules.mjs`.
- **Drift check (wired)**: `npm run check:generated` (`scripts/harness/check-generated.mjs`, **blocking** in quality.yml) verifies marker and existence for every generated artifact: the 11 stable artifacts by name, plus dynamic discovery of generated module files under `backend|frontend/**/modules/business/<module>/` (first-line marker on `.go/.ts/.tsx`) and `schema/generated/<scope>/<name>.json` (path + parseability); `npm run check:generated-modules` still detects leftover smoke-generated modules. The legacy `checkGeneratedRegistry` in `check-arch-boundaries.mjs` remains an overlapping implementation that is not wired into CI (see section 10).
- **Alias divergence**: `scripts/check-arch-boundaries.mjs` is a historical script that overlaps with `scripts/harness/check-boundaries.mjs`; until they converge, only `scripts/harness/check-boundaries.mjs` is the normative gate.

## 8. Layer Dependency Matrix

The mapping from logical layers to physical directories is in [DESIGN.md — layering and module boundaries](../../DESIGN.md); this section defines only the allowed dependency directions, which are the decision basis for `scripts/harness/check-boundaries.mjs` and `scripts/check-arch-boundaries.mjs`.

### 8.1 Layers and Physical Locations

| Logical layer | Backend | Frontend |
| --- | --- | --- |
| Platform shell `platform` | `backend/modules/platform`, `backend/internal`, `backend/pkg` | `frontend/src/modules/platform`, `frontend/src/core` |
| System foundation `system/auth` | `backend/modules/auth`, `backend/modules/system/{iam,org,config,audit,i18n}` | `frontend/src/modules/auth`, `frontend/src/modules/system/*` |
| Business domain `business/*` | `backend/modules/business/*` | `frontend/src/modules/business/*` |
| Shared contracts | `backend/pkg/contracts`, `backend/pkg/common`, `backend/pkg/database` | `frontend/src/api`, `frontend/src/components`, `frontend/src/hooks` |

`lowcode` is an independent work domain for generation/dynamic capability, callable by `platform` and `business` only through its public contract; `generated` is a frontend generated-registry directory and is never a source of the hand-written dependency graph.

### 8.2 Allowed Dependency Directions

| From ↓ / To → | platform | auth | system/* | business/* | pkg/contracts |
| --- | --- | --- | --- | --- | --- |
| `platform` | ✅ same layer | ⚠️ public contract only | ⚠️ public contract / read model only | ❌ | ✅ |
| `auth` | ❌ | ✅ same domain | ✅ system foundation, public contract preferred | ❌ | ✅ |
| `system/*` | ❌ | ❌ | ✅ same subdomain; cross-subdomain via public contract only | ❌ | ✅ |
| `business/*` | ❌ | ❌ | ❌ | ✅ same module; cross-module via public contract only | ✅ |

- ✅ allowed; ⚠️ allowed only through a public contract / adapter / read model, never by importing the other side's Service / Repository / Handler; ❌ forbidden.
- The frontend follows the same matrix: `business` must not import `modules/system|auth|platform` internals; `system` subdomains depend on each other only through `frontend/src/api` or a public component contract.

### 8.3 Violation Baseline and Closure

- The gate `scripts/harness/check-boundaries.mjs` now covers `business`, `platform`, and `auth` production code; `_test.go` and `*.test.*` / `*.spec.*` are explicitly exempt (tests may wire modules together, production code may not).
- `platform -> system/org/dept` is fixed per 8.2: the adapter moved to the composition root `backend/cmd/server/platform_org_governance.go`, and the platform module no longer imports the system implementation.
- The remaining known violations are frozen in `config/boundary-baseline.json` (each with a `reason` and `reviewBy=2026-12-31`). CI passes `--baseline` so known items are allowed and any new item is blocked:
  - a **type-only** dependency from `frontend/src/modules/platform/widgets.tsx` on `system/menu/api`;
  - direct `system/iam/user` dependencies in `auth/login`, `auth/login/login_runtime.go`, and `auth/security/security_service.go`;
  - four auth pages importing `system/components/shared/list-page.css`.
- The baseline is a **time-boxed controlled exception**; it must be re-reviewed or fixed at expiry, and a new violation must never be silenced by adding it to the baseline.
- Test-file and generated-file cross-domain dependencies are handled separately; an exemption needs an explicit comment and evidence and must not be conflated with production code.

## 9. Controlled Exceptions

The physical layouts below are existing exceptions. They are **allowed to exist, not to spread**. A new exception of the same kind must be registered here with an owner and a review condition first.

| Exception | Nature | Owner layer | Rationale | Review / expiry |
| --- | --- | --- | --- | --- |
| `backend/modules/auth`, `frontend/src/modules/auth` | top-level physical module | system/auth | `auth` is logically system foundation, but login/session/MFA/SSO lifecycles are independent; physical separation keeps `system` from becoming a junk drawer | review on every auth contract change |
| `backend/modules/lowcode`, `frontend/src/modules/lowcode` | top-level physical module | platform (lowcode work domain) | generator and dynamic-module governance form an independent work domain, not owned by any business domain | review on lowcode design changes |
| `frontend/src/modules/generated` | generated directory | platform | generated registries need a stable fixed path and are admitted by the domain whitelist | review by the generated-artifact governance task |
| `backend/tests/`, `frontend/tests/` | top-level test directories | platform | tests are physically separate from runtime code for whitelisting and gating | review on structure-gate changes |
| `config/`, `database/`, `grafana/`, `openspec/`, `schema/generated/` | stable root entries | platform | see section 4; referenced directly by scripts/compose/governance flows | review when the referencing party changes |

An exception is not a whitelist expansion. Shoving a violating directory into the exception table instead of fixing it is the same as hiding the problem.

## 10. Mechanical Gates and Update Points

### 10.1 Standard → Gate Mapping

| Rule area | Enforcer | Command | CI |
| --- | --- | --- | --- |
| §2 placement + §5 naming | `scripts/harness/check-structure-contract.mjs` | `npm run check:structure` | `quality.yml` (blocking) |
| §8.2 `business` / `platform` / `auth` cross-layer imports | `scripts/harness/check-boundaries.mjs` | `node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base --baseline config/boundary-baseline.json` | `ci.yml` (blocking; known debt in the 8.3 baseline) |
| §6 doc frontmatter / type / contract linkage | `scripts/frontmatter-check.mjs`, `scripts/harness/check-doc-frontmatter.mjs` | `npm run check:docs-frontmatter` | `quality.yml` |
| §6 internal doc links | `scripts/harness/check-doc-links.mjs` | `npm run check:harness-docs` | harness checks |
| §7 generated marker and artifact existence | `scripts/harness/check-generated.mjs` | `npm run check:generated` | `quality.yml` (blocking) |
| §7 generated module drift | `frontend/scripts/cleanup-generated-modules.mjs` | `npm run check:generated-modules` | `quality.yml` |
| §7 generated registry existence/non-emptiness (legacy) | `scripts/check-arch-boundaries.mjs` (`checkGeneratedRegistry`) | not wired into npm/CI | **gap** |

### 10.2 Change Procedure (standard first, gates second)

When adding or adjusting a directory, file suffix, exception, or dependency direction, use this order:

1. Update the relevant section of this document (§2 / §5 / §8 / §9).
2. Sync the gate whitelists or rules: `check-structure-contract.mjs`, `check-boundaries.mjs`, `check-arch-boundaries.mjs`.
3. Sync CI steps and npm scripts.
4. Sync indexes and all references such as `docs/designs/README.md` and `docs/README.md`.
5. Add or update the matching tests (`tests/scripts/**`) and run the verification commands in 10.1.

Changing only the document without the gate, or widening the whitelist without fixing the root cause, does not count as done.
