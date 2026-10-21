---
title: Repository Layout, Naming, and Layer Boundaries
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-10-11
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
- **The package name must equal its directory name** (`package <dirname>`). This is what makes `go doc`, compiler errors, IDE navigation, and import paths line up with the directory tree; when they disagree every caller is forced to write an import alias, which amounts to admitting the directory name is the real identity. The only exception is `backend/cmd/*`: an executable entrypoint must be `package main` and is not compared against the directory name. An external test package `foo_test` (in `foo/`) is likewise compliant. Gate rule: `go-package-dir-match`.
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

The judgement is a single question: **does this file declare a component?** Declares a component → PascalCase `*.tsx`; does not → camelCase (suffix `.ts`, or `.tsx` when it contains JSX). The `frontend-file-naming` gate applies this to `frontend/src/modules/**` and `frontend/src/core/**`.

| Category | Naming | Location | Example |
| --- | --- | --- | --- |
| Component | PascalCase `*.tsx` | `frontend/src/components/<group>/`, `frontend/src/modules/**` (see "Component placement") | `PageEmpty.tsx`, `UserList.tsx` |
| Hook | `use<PascalCase>.ts(x)` | `frontend/src/hooks/`, `frontend/src/core/**`, module dirs | `usePermission.ts`, `useLayoutMenu.tsx` |
| Non-component module | camelCase `*.ts` | alongside its consumer | `tablePreferences.ts` |
| Non-component JSX module | camelCase `*.tsx` | alongside its consumer | `icon.tsx`, `clientInfo.tsx`, `widgets.tsx` |
| Test | `*.test.ts(x)` | `frontend/tests/` (not `frontend/src/`) | `router.test.ts` |
| API | camelCase `*.ts` | `frontend/src/api/` | `file.ts` |
| i18n resource | locale-named | `frontend/src/i18n/resources/` | `zh-CN.ts` |
| Generated registry | fixed names | `frontend/src/modules/generated/`, `frontend/src/core/router/` | `business.ts`, `generatedComponentRegistry.ts` |
| Generated module | business module name | `frontend/src/modules/business/<module>/` | generator output |
| Component style | same name as the component it styles (`ComponentName.css`), same directory | component dir | `TimeRangeFilter.css` |
| Group / module-level shared style | named after its **directory** (`<dir>.css`) or placed under an explicit `shared/` subdir | group dir | `auth.css` |
| Cross-module / global shared style | centralized in `frontend/src/assets/` or global `index.css` | shared | `assets/list-page.css`, `index.css` |

Styles have three tiers: a **component-specific** stylesheet's filename must equal the component
name (not the BEM block name, not the directory name); a **group/module-level** shared stylesheet
is named after its directory or lives under `shared/`; a **cross-module** shared stylesheet needs
an explicit shared location — a stylesheet imported by two or more `modules/<domain>/` trees is
cross-module by definition and must live in `frontend/src/assets/`, not inside one module (otherwise
every consumer has to import across a domain boundary, which conflicts with 8.2).
BEM class names (`.time-range-filter__x`) do not change with the filename.

**Non-component JSX module**: the file contains JSX so its suffix is `.tsx`, but it exports render
functions, data, or types rather than a component (`renderMenuIcon()`, `renderClientInfo()`, a widget
definition table). Such files use camelCase; **as soon as one declares a component it must be renamed
to PascalCase** — the gate detects components through component type markers (`React.FC`, `React.memo`,
`React.forwardRef`, `ComponentType`) and PascalCase `const`/`function`/`class` declarations.

**Component placement**: pick one per module and do not mix — with few components (1–3) put them
directly at the module root (`modules/system/iam/user/UserList.tsx`); a module with a group of
components keeps them all under a `components/` subdirectory (`modules/auth/login/components/Login.tsx`).
This is a team convention rather than a gate rule: the gate checks naming only, because relocating
component directories does not improve readability but does generate a large, meaningless diff.

**`index.ts` / `index.tsx` is for barrel re-exports only; it holds no implementation** (gate `barrel-only`).
The enforced scope is `frontend/src/{core,components,hooks}/**/index.ts(x)` — those layers use barrels by
convention. Two legal exceptions: `frontend/src/modules/**/index.ts` is a module descriptor
(`defineModule({...})`) and `frontend/src/i18n/index.ts` is the locale table; neither is a barrel and
neither is subject to this rule.

**Path alias**: use `@/` for cross-module imports (`@/core/...`, `@/components/...`), configured in
`paths` in `frontend/tsconfig.app.json` and `resolve.alias` in both `frontend/vite.config.ts` and
`frontend/vitest.config.ts` — all three must stay in sync. Existing relative imports are grandfathered
and may be migrated opportunistically; new cross-module imports should use the alias, because a relative
path has to be rewritten wholesale the moment a directory moves.

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
- **For frontend `system/*` subdomains the "public contract" is the sibling's `api` module**
  (`modules/system/<sub>/<leaf>/api.ts`). Subdomain A may import B's `api` but must not import B's
  component, hook, or utility files directly; when A needs B's renderer, B re-exports it from its own
  `api` module (as `iam/user/api.ts` re-exports `UserDetailContent`), which is what makes it a
  "public component contract" — the caller still depends only on `api`.
  The `system/*` ts rules are matched against the **resolved target path**, not the import literal,
  because `../../../../i18n` (the top-level `frontend/src/i18n`) and `../../i18n` (the `system/i18n`
  subdomain) share a final segment and cannot be told apart from the literal alone.

### 8.3 Violation Baseline and Closure

- The gate `scripts/harness/check-boundaries.mjs` now covers `business`, `platform`, and `auth` production code **plus every `system/*` subdomain**; `_test.go` and `*.test.*` / `*.spec.*` are explicitly exempt (tests may wire modules together, production code may not).
- **Repository root resolution (hardened 2026-10-11)**: `scanRepository` used to trust any directory whose name matched the repository, so a leftover local shell
  `pantheon-base/pantheon-base/` (holding a single empty `docs/`) was mistaken for the checkout — **every layer scan read zero files and the gate reported "no findings" while scanning nothing**. Root resolution now requires the candidate to actually contain `backend/` or `frontend/`, and an unresolvable root fails `--strict` instead of passing quietly.
- `platform -> system/org/dept` is fixed per 8.2: the adapter moved to the composition root `backend/cmd/server/platform_org_governance.go`, and the platform module no longer imports the system implementation.
- `system/*` subdomain ts rules (added 2026-10-11): `systemDomainRules()` previously returned `ts: []`, so even once the frontend directories existed the scan applied no rules. Cross-subdomain imports are now checked per 8.2 — a sibling is reachable only through its `api`.
- The remaining known violations are frozen in `config/boundary-baseline.json` (each with a `reason` and `reviewBy=2026-12-31`). CI passes `--baseline` so known items are allowed and any new item is blocked:
  - a **type-only** dependency from `frontend/src/modules/platform/widgets.tsx` on `system/iam/menu/api` (the only entry left).
- **Closed on schedule (2026-10-11)**: the four baselined auth entries importing `system/components/shared/list-page.css` were removed — their own ratchet was executed by promoting the stylesheet to `frontend/src/assets/list-page.css`, so auth no longer imports across into the system module, and the misleading `modules/system/components/` directory is gone. The baseline shrank from 5 entries to 1. (The earlier three `backend/modules/auth/* -> modules/system/iam/user` entries were removed on 2026-09-23 when the `pkg/contracts/authuser` port landed.)
- The baseline is a **time-boxed controlled exception**; it must be re-reviewed or fixed at expiry, and a new violation must never be silenced by adding it to the baseline. An entry's ratchet text *is* its fix plan: when the conditions are met, **execute** the ratchet and delete the entry rather than renewing it.
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
| `frontend/src/modules/system/profile` | one-sided (frontend only) | system/auth | the profile page has no backend package of the same name; its logic is split across `auth/security/profile_dto.go` and `system/iam/user/user_preferences.go`. It is a page composition, not a domain package | review when the backend gains a `profile` package or the frontend merges it into `iam/user` |
| `backend/modules/system/iam/tenant` | one-sided (backend only) | system/iam | the tenant module has no frontend counterpart: tenant switching happens in `auth/login` (chosen at sign-in), and there is no standalone runtime tenant page | review when the frontend gains a tenant-switching entry |
| frontend has no `auth/oidc` counterpart | one-sided (backend only) | system/auth | SSO/OIDC uses a browser redirect on the frontend and has no standalone module; `auth/index.ts` carries a comment saying so | review on auth contract changes |
| the empty `frontend/src/modules/business/` | generation container | platform | corresponds to `backend/modules/business/*`; having no artifacts at HEAD is normal (they appear with a generation task) | review by the generated-artifact governance task |

The first three are also registered in `CORRESPONDENCE_EXCEPTIONS` in
`scripts/harness/check-structure-contract.mjs`: a one-sided subdomain is only legal once registered
there — an unregistered one-sided directory fails the `domain-correspondence` gate outright.

An exception is not a whitelist expansion. Shoving a violating directory into the exception table instead of fixing it is the same as hiding the problem.

## 10. Mechanical Gates and Update Points

### 10.1 Standard → Gate Mapping

| Rule area | Enforcer | Command | CI |
| --- | --- | --- | --- |
| §2 placement + §5.1 Go naming + §5.2 frontend naming/barrels + §11 domain correspondence | `scripts/harness/check-structure-contract.mjs` | `npm run check:structure` | `quality.yml` (blocking) |
| §8.2 `business` / `platform` / `auth` / each `system/*` subdomain cross-layer imports | `scripts/harness/check-boundaries.mjs` | `node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base --baseline config/boundary-baseline.json` | `ci.yml` (blocking; known debt in the 8.3 baseline) |
| §6 doc frontmatter / type / contract linkage | `scripts/frontmatter-check.mjs`, `scripts/harness/check-doc-frontmatter.mjs` | `npm run check:docs-frontmatter` | `quality.yml` |
| §6 internal doc links | `scripts/harness/check-doc-links.mjs` | `npm run check:harness-docs` | harness checks |
| §7 generated marker and artifact existence | `scripts/harness/check-generated.mjs` | `npm run check:generated` | `quality.yml` (blocking) |
| §7 generated module drift | `frontend/scripts/cleanup-generated-modules.mjs` | `npm run check:generated-modules` | `quality.yml` |
| §7 generated registry existence/non-emptiness (legacy) | `scripts/check-arch-boundaries.mjs` (`checkGeneratedRegistry`) | not wired into npm/CI | **gap** |

Rule inventory of `check-structure-contract.mjs` (all at zero findings, and a new rule must land at zero too):

| Rule | Covers |
| --- | --- |
| `go-placement` / `go-file-naming` | Go file placement and snake_case file names |
| `go-package-dir-match` | §5.1 package name == directory name (`cmd/*` exempt) |
| `frontend-file-naming` | §5.2 `.ts`/`.tsx` naming under `modules/**` and `core/**` |
| `barrel-only` | §5.2 `core|components|hooks/**/index.ts(x)` re-export only |
| `component-naming` / `hook-naming` | shared `components/**` and `hooks/**` |
| `domain-correspondence` | §11 frontend/backend subdomain correspondence (with §9 exceptions) |
| `frontend-root` / `frontend-src-root` / `frontend-module-domain` / `backend-root` / `backend-module-domain` | §2 placement whitelist |
| `no-tests-in-src` / `no-tracked-binaries` | §2.2 test placement, no committed binaries |

### 10.2 Change Procedure (standard first, gates second)

When adding or adjusting a directory, file suffix, exception, or dependency direction, use this order:

1. Update the relevant section of this document (§2 / §5 / §8 / §9).
2. Sync the gate whitelists or rules: `check-structure-contract.mjs`, `check-boundaries.mjs`, `check-arch-boundaries.mjs`.
3. Sync CI steps and npm scripts.
4. Sync indexes and all references such as `docs/designs/README.md` and `docs/README.md`.
5. Add or update the matching tests (`tests/scripts/**`) and run the verification commands in 10.1.

Changing only the document without the gate, or widening the whitelist without fixing the root cause, does not count as done.

## 11. Frontend/Backend Domain Correspondence

This section is the **single source of truth** for "frontend leaf ↔ backend path ↔ `module` value ↔
componentKey ↔ API prefix". Before it existed, correspondence could only be guessed by comparing
directory depth, and every review had to re-explain the same mapping.

- **Physical paths**: frontend `frontend/src/modules/<domain>/…`, backend `backend/modules/<domain>/…`,
  with the same `<domain>` on both sides (`auth`/`business`/`lowcode`/`platform`/`system`).
- **Leaf names**: business leaves match on both sides (`user`/`role`/`menu`/`permission`/`dept`/`post`/`dict`/`setting`/`audit`/`i18n`).
- **Aggregation layer**: the backend has an extra `iam`/`org`/`config` level under `system`, and the
  frontend **physically keeps that same level** (`frontend/src/modules/system/iam/user/`), so both trees
  have equal depth and can be read side by side.
- **`module` value**: the logical identifier used for menu/permission grouping (e.g. `system.iam`), which
  is **not** the physical path. It is defined by the `Module` field in `backend/modules/system/seed.go`,
  and the frontend module descriptor must agree with it; `frontend/scripts/check-menu-contract.mjs` verifies this.
- **componentKey**: the frontend component a menu points at, as a **literal string** (e.g. `system/user/UserList`),
  not derived from any path. The backend seed's `Component`, the frontend `componentRegistry.ts`, and the
  route's `componentKey` must match character for character; `check:menu-contract` verifies this.
- **API prefix**: the REST resource name, which may differ from the directory name (see the audit row).

### 11.1 The `system` foundation

| Concept | Frontend path | Backend path | `module` value | componentKey | API prefix |
| --- | --- | --- | --- | --- | --- |
| User | `system/iam/user` | `system/iam/user` | `system.iam` | `system/user/UserList`, `system/user/UserDetail` | `/system/user` |
| Role | `system/iam/role` | `system/iam/role` | `system.iam` | `system/role/RoleList` | `/system/role` |
| Menu | `system/iam/menu` | `system/iam/menu` | `system.iam` | `system/menu/MenuList` | `/system/menu` |
| Permission | `system/iam/permission` | `system/iam/permission` | `system.iam` | `system/permission/PermissionList` | `/system/permission` |
| Tenant | **none** (§9 exception) | `system/iam/tenant` | — | — | (backend only) |
| Dept | `system/org/dept` | `system/org/dept` | `system.org` | `system/dept/DeptList` | `/system/dept` |
| Post | `system/org/post` | `system/org/post` | `system.org` | `system/post/PostList` | `/system/post` |
| Dictionary | `system/config/dict` | `system/config/dict` | `system.config` | `system/dict/DictPage` | `/system/dict` |
| Settings | `system/config/setting` | `system/config/setting` | `system.config` | `system/setting/SettingOverviewPage`, `system/setting/SettingGroupPage` | `/system/setting` |
| i18n | `system/i18n` | `system/i18n` | `system.config` | `system/i18n/I18nList` | `/system/i18n` |
| Audit | `system/audit` | `system/audit` | `system.audit` | `system/audit/OperationLogList` | `/system/operation-log` |
| Profile | `system/profile` | **none** (§9 exception) | (no `module` value) | `system/profile/ProfileCenter` | `/system/profile` |

Two cases that look inconsistent but are settled layering — do not "fix" them by directory name:

- **i18n**'s physical package is `system/i18n` (a sibling of `config`), but it hangs under the **config
  menu group**, so its `module` value is `system.config` and the seed uses `ParentKey: "config"`. The
  backend seed and the frontend descriptor agree; this is menu information architecture, not drift.
- **Audit** is two layers of vocabulary: directory / package / `module` value use `audit`, while route /
  API / permission codes use `operation-log` (`system:operation-log:*`). **Both sides take identical
  values for both layers**, so this is not "one feature with four names". Unifying the vocabulary would
  change the API route, the permission codes, and existing menu data, violating the explicit non-goal of
  `.harness/NAMING_AND_BOUNDARIES_REMEDIATION_PLAN_2026-09-22.md` ("do not change API, database,
  permissions, or runtime behavior"), so it is kept as layering and pinned down here.
- **Profile** has no `module` value and no menu entry (`menus: []`, `permissions: []`): it is a route-level
  page rather than a menu module, reachable from the user avatar dropdown — which is exactly why it is a
  registered controlled exception.

### 11.2 `auth` and the other domains

| Concept | Frontend path | Backend path | `module` value | Note |
| --- | --- | --- | --- | --- |
| Login | `auth/login` | `auth/login` | `system.auth` | tenant selection also lives here |
| MFA | `auth/mfa` | `auth/mfa` | `system.auth` | |
| SSO / OIDC | **none** (§9 exception) | `auth/oidc` | — | browser redirect on the frontend, no standalone module |
| Security center | `auth/security` | `auth/security` | `system.auth` | componentKeys `auth/SecurityCenter` etc. |
| Session | `auth/session` | `auth/session` | `system.auth` | componentKey `auth/SessionList` |
| Platform dashboard | `platform/{Dashboard,widgets,api}` | `platform/{dashboard_*,health,routes}` | `platform` | `health`/`routes` having no frontend counterpart is normal |
| Dynamic module | `lowcode/dynamicmodule` | `lowcode/dynamicmodule` | `system.lowcode` | UI mounted at `/system/modules` |
| Generator | `lowcode/generator` | `lowcode/generator` | `system.lowcode` | frontend has an extra `pages/` level |
| Business generated module | `business/<module>` (generated) | `business/<module>` (generated) | business module name | generated artifact; empty at HEAD is normal |

The `domain-correspondence` gate checks only the first two levels of the `system` foundation (see 10.1);
for `auth`/`platform`/`lowcode`/`business` this section is the documentary source of truth. When adding a
one-sided directory, update this section and state whether the other side will be added or a §9 exception
is being registered.
