---
title: Business Development Workflow and AI Collaboration Guide
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-10
---

# Business Development Workflow and AI Collaboration Guide

Chinese version: [WORKFLOW.md](./WORKFLOW.md)

This document describes the Pantheon Base delivery workflow, with contract-first documentation governance and explicit AI collaboration rules.

## 0. Contract-Driven Documentation Flow

Since April 30, 2026, Pantheon defaults to a contract-first documentation model so that design, assessment, remediation, and acceptance all anchor back to one governing source.

Current contract entry points include:

- `DOCUMENT_GOVERNANCE_CONTRACT`
- `DOCUMENT_METADATA_AND_STATUS`
- `CONTRACT_TEMPLATE`
- `PLATFORM_CONTRACT`
- `SYSTEM_AUTH_CONTRACT`
- `SYSTEM_IAM_CONTRACT`
- `SYSTEM_ORG_CONTRACT`
- `SYSTEM_CONFIG_CONTRACT`

### 0.1 Basic Rules

Default progression:

`Contract -> Design -> Assessment -> Remediation -> Acceptance`

Before starting a new stream of work:

1. determine the owning layer
2. confirm whether a contract already exists
3. create or extend the contract anchor first if it does not
4. make every downstream document link back to the contract
5. keep untyped or unanchored draft notes out of the primary index

### 0.2 Document-Type Rules

Standard types are:

- `Contract`
- `Design`
- `Assessment`
- `Remediation`
- `Acceptance`
- `Archive`

Primary documents must at least declare type, layer, and status.

## 1. Full Lifecycle SOP for Business Features

### Phase 1: Data Model Design

- determine the owning layer and contract first
- add the contract or contract skeleton if missing
- write SQL DDL with business-module table prefixes such as `biz_order_`
- record DDL under `database/`

### Phase 2: Backend

- create the module under `modules/business/`
- generate `model`, `dto`, `repo`, and `service` slices
- keep core business flow in the service layer
- register routes through the explicit backend-module assembly contract

### Phase 3: Frontend UI

- create the module under `src/modules/business/`
- implement Arco Design pages and API bindings
- export a module manifest and register it through `src/core/router/modules.ts`

### Phase 4: Test

Verify:

- API responses including `200`, `401`, and `403`
- auth refresh and logout behavior
- multi-role Casbin behavior
- menu trees for `scope=nav` and `scope=manage`
- filtering, pagination, and sorting behavior
- UI triggers for audit logging
- validation failures such as uniqueness and admin-protection rules

### Phase 5: Quality Gates and Automated Review

This phase separates feature acceptance from code-quality protection.

- functional supervision confirms whether the change meets the requirement
- quality guardianship is handled by GitHub required checks, CodeQL, and optional Copilot review
- Copilot review comments do not replace required checks or approvals
- standard changes flow through `Quality Gates`, `Security Gates`, and squash auto-merge
- high-risk changes must document residual risk, rollback steps, and added verification

Default high-risk scope includes:

- `system/auth`
- `system/iam`
- `system/config`
- permission and audit flows
- shared `pkg/*`
- generator and dynamic-module flows
- CI, deploy, secrets, and credential handling

Required gate stack:

- local validation for the touched scope
- `Quality Gates` for docs governance, frontend contract, backend tests, the Duplication Gate result, and lightweight `smoke-sanity`
- `Security Gates` for secret scan, workflow posture, and CodeQL
- `Full Smoke Suite` only for manual, scheduled, or release-precheck regression runs
- dependency-vulnerability scans on `main/release`, scheduled, or manual workflows rather than every PR
- Copilot review status for architecture, intent, and maintainability context
- no Codacy dependency in the merge gate

Recommended GitHub protections:

- branch protection for `main` and release branches
- required status checks
- optional Copilot automatic review policy
- require conversation resolution
- enable automatic deletion of head branches so merged worktree branches do not keep accumulating

The delivery loop does not end at local validation; it ends only after the PR is merged into `main`, the head branch is deleted or manually deleted during closeout, the local worktree is cleaned up, and the closeout records the PR URL, merge commit, and branch cleanup status.

### Phase 6: Platform Smoke

For browser-path QA, screenshots, and interaction inspection on local Windows setups, the default manual tool is `gstack browse / gstack Browser`. Playwright is secondary for CI, API smoke, or explicit requests.

Core smoke rules:

- classify the owning layer before starting
- confirm frontend `:5173` and backend `:8080` are healthy first
- prefer API login to obtain tokens
- prefer one chained browser flow over fragmented manual steps
- keep a fixed minimum coverage set across login, dashboard, auth, IAM, org, config, and audit pages
- capture final URL, console errors, screenshot, and `snapshot -i` output for every page
- distinguish real product failures from Windows or tooling instability
- preserve JSON summaries, raw logs, and screenshots as acceptance evidence

## 2. AI Collaboration Prompt Guide

Before asking AI to generate plans or code, always provide:

- current owning layer
- current contract document path
- current document type
- current status

If these four items are unclear, do not jump straight into implementation generation.

### Backend Prompt Template

Ask AI to generate a vertical-slice business module with:

- `biz_` table prefixes
- DTO masking for sensitive fields
- unified `common.Success` and `common.Fail`
- service-owned business logic and repo-owned persistence only

### Frontend Prompt Template

Ask AI to generate a page under `src/modules/business` with:

- Arco Design layout
- `t()` for all user-visible text
- Pantheon Base indigo-theme compliance

## 3. Test and Deployment Flow

Recommended baseline:

1. `docker compose up -d`
2. initialize `database/system_init.sql`
3. set `PANTHEON_DSN` and optional Redis settings
4. set `PANTHEON_INITIAL_ADMIN_PASSWORD` in production
5. start backend with `go run ./backend/cmd/server`
6. start frontend with `npm run dev`
7. run `go test ./...`, frontend lint, build, and audit checks before submission

### 3.1 Documentation Sync Gate

Before submission, confirm:

1. the change still fits the owning contract
2. boundary changes are reflected in the contract first
3. new design documents link back to the correct contract
4. new assessment or remediation documents include type, status, and contract linkage
5. superseded old documents are removed, downgraded, or explicitly marked

### 3.2 Recommended GitHub Repository Controls

Recommended repository defaults:

1. protect `main` and `release/*`
2. make GitHub-native checks the only required status checks
3. enable auto-merge and prefer squash merge
4. require conversation resolution before merge
5. enable automatic branch deletion
6. enable secret scanning, dependency review, and code scanning when available
7. keep full-browser regression out of the default PR merge gate unless the repository is intentionally optimized for that cost
8. do not add Codacy as required checks

## 4. Smoke SOP for gstack on Windows

This SOP targets platform and system-domain pages rather than deep business-module specialty testing.

### 4.1 Pre-Run Checks

- confirm the target layer
- confirm frontend on `5173` and backend on `8080`
- confirm test-account availability

### 4.2 Recommended Order

1. get tokens through the login API
2. inject auth state through one chained browse flow
3. traverse `platform -> system/auth -> system/iam -> system/org -> system/config`
4. verify API behavior before blaming the page
5. export JSON, raw logs, and screenshots at the end

### 4.3 Recommended Command Pattern

Use one `browse chain` that opens login, waits for idle, injects tokens, jumps to dashboard, waits again, and collects URL, console, screenshot, and snapshot evidence.

### 4.4 Minimum Platform Coverage

- `platform`: `/dashboard`
- `system/auth`: `/login`, `/auth/security`, `/system/login-log`, `/system/session`
- `system/iam`: `/system/profile`, `/system/user`, `/system/user/1`, `/system/role`, `/system/menu`, `/system/permission`, `/system/operation-log`
- `system/org`: `/system/dept`, `/system/post`
- `system/config`: `/system/dict`, `/system/setting`

### 4.5 Required Outputs

- JSON summary with page, layer, status, log path, and screenshot path
- raw logs for URL, console errors, and snapshot output
- screenshot directory for review and regression comparison

### 4.6 Windows Notes

- `browse.exe` may occasionally need elevation
- fragmented calls are more likely to drift into blank pages than one chained flow
- `No active page`, screenshot timeouts, or closed contexts should be retried before they are treated as product defects

### 4.7 Document Destinations

Use the archive smoke example, acceptance checklist, and Windows-specific guidance documents as reference destinations for evidence.

## 5. Submission Rules for the `platform` Shell

This section applies to `platform` shell changes in particular.

### 5.1 Submission Gates

Platform-shell changes should not be submitted without shell-level smoke coverage, evidence capture, and contract-aligned documentation.

### 5.2 Acceptance Documentation Requirements

Acceptance notes must be able to point back to the platform contract and include evidence rather than only developer claims.

### 5.3 Fixed Summary Format

Use one stable summary structure so that acceptance and remediation records stay readable across iterations.

### 5.4 Forbidden Behaviors

- skipping contract anchoring
- shipping shell changes without smoke evidence
- treating undocumented drift as acceptable because the code already works
