---
title: Repository Layout
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-07-15
---

# Repository Layout

Chinese version: [REPOSITORY_LAYOUT.md](./REPOSITORY_LAYOUT.md)

This document defines the stable root layout for `pantheon-base`. The goal is to keep the repository root readable as an engineering entry point, not as a dumping ground for temporary artifacts.

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

## 2. Placement Rules

1. Backend product code belongs under `backend/modules/` or `backend/pkg/`; do not create new business directories at the repository root. Backend performance/load test scripts belong under `backend/tests/performance/`.
2. Frontend runtime code belongs under `frontend/src/`; frontend scripts belong under `frontend/scripts/`; frontend smoke tests and test fixtures belong under `frontend/tests/`; frontend unit tests belong under `frontend/tests/unit/` (tests must not live in `frontend/src/`); `vitest.config.ts` is a legal frontend root config file (alongside the vite/playwright configs).
3. Root automation belongs under `scripts/`; matching tests belong under `tests/scripts/`.
4. Harness method checks belong under `scripts/harness/`; task manifests and evidence belong under `.harness/`.
5. Active architecture and governance docs belong under `docs/designs/`, `docs/contracts/`, and `docs/acceptances/`; phase audits, assessments, and process records are not committed — they stay with task evidence under `.harness/` and are cleaned up when a task closes.
6. Generated bundle output belongs under `dist/` and must not be committed.
7. `database/system_init.sql` remains in `database/` because `docker-compose.yml` references that stable path.
8. Local Grafana/Prometheus observability config remains in `grafana/` so it stays separate from application runtime code.

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
