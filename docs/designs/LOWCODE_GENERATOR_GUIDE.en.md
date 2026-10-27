---
title: Low-Code Module Code Generation Platform - Integration and Usage Guide
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-07-12
---

# Low-Code Module Code Generation Platform - Integration and Usage Guide

Chinese version: [LOWCODE_GENERATOR_GUIDE.md](./LOWCODE_GENERATOR_GUIDE.md)

> **Positioning: P2+ business module code generation platform (not a skeleton, not a runtime low-code).**
>
> The current `lowcode/generator` inside `pantheon-base` goes well beyond a
> "code skeleton" target — it provides a full stack: schema, i18n, datasource,
> relation, menu, permission, audit, registry rewrite, and dashboard widget
> injection, and closes the loop of "source generation + workspace write +
> generated registry rewrite + pending activation registration".
>
> Within this document, "skeleton" means: **a wizard run with the default
> `enterprise` template produces code that is sufficient to ship a CRUD business
> module** — not that the generator itself is a skeleton.

## 1. Overview

This guide describes the Pantheon low-code module code generation platform as a development-acceleration capability that helps engineers ship modules conforming to the P2+ enterprise contract, rather than redefining the core platform architecture.

The current closed loop already covers:

1. module description rules and templates
2. backend and frontend code generation
3. generated route, component, and backend-module registry rewriting
4. visual configuration UI
5. pending-activation registration state and smoke baseline
6. `i18n-first` structured key generation
7. automatic platform-dashboard quick-action widget generation

## 1. Positioning Boundary

- Pantheon’s main goal remains a standard enterprise backoffice platform
- the generator belongs to `system/config -> generator`
- the platform layer owns module contracts, review rules, and acceptance gates
- the generator is for developer acceleration, scaffold generation, controlled registration, and governance
- it is not a runtime low-code platform and must not redefine core `auth`, `iam`, `org`, or `config` boundaries

## 2. `i18n-first` Constraints

The generator emits structured i18n keys rather than hardcoded natural-language strings in source code.

Key patterns include:

- module title
- field labels
- placeholders
- help text
- enum options
- action names
- audit titles

The wizard may still accept natural-language initial values, but generated frontend code, backend seeds, menu titles, permission titles, and audit titles must all consume keys.

### Pre-Generation Translation Preview and CSV Flow

The wizard shows a translation preview before generation, including:

- menu titles
- page titles
- field labels, placeholders, and help text
- enum options
- permission titles
- audit titles

Recommended flow:

1. let the generator produce initial Chinese and English values
2. edit them in the preview table
3. export CSV if business or localization staff need to adjust them
4. import the CSV back
5. write the final values into generated schema, backend generated i18n seeds, and frontend generated fallback resources

CSV focuses on initial Chinese and English values. Other locales should remain governed through the runtime `system/i18n` workflow. English may serve as fallback for the currently supported non-Chinese locales to avoid raw-key exposure.

## 3. Menu Information-Architecture Rules

Business modules may express hierarchy through path segments:

- `cmdb`
- `cmdb/host`
- `cmdb/vendor`

Rules:

1. use a single segment for a top-level business module
2. use nested names when a page belongs under an existing business context
3. leave `parentMenu` empty unless you really need a nondefault existing parent
4. let the generator create missing parent menu seeds from the path

Generated results should include:

- parent menu path and title key
- page menu path and title key
- permission prefix
- module namespace
- optional platform dashboard quick-action widget metadata

### Platform Dashboard Registration

The generator currently adds a platform quick-action widget for navigable `business/*` modules unless explicitly disabled.

Rules:

- only generate `quick-action` widgets by default
- do not generate widgets for relation tables
- do not generate widgets for `system/*`
- widget copy must also be key-driven

## 4. Multi-Table Business Modeling Guidance

The generator is better suited to “one business entity, one standard management page” than to one giant multi-table mega-module.

For example, a CMDB context should usually split into separate modules such as:

- `cmdb/host`
- `cmdb/vendor`
- `cmdb/group`
- `cmdb/host_group`

Relation tables should not automatically become visible CRUD navigation pages. They are usually better consumed through detail pages, bind dialogs, or relation selectors.

## 5. P2+ Enterprise Contracts

P2+ is not about online runtime low-code orchestration. It is about enriching generation-time governance metadata.

### Template Version

Template versions should stay explicit so future upgrades can identify migration needs.

### Module Dependencies

Dependencies should express business prerequisites only. They must not cause the generator to create direct cross-module service coupling.

### Relation Contracts

Relation contracts help preview lookup and many-to-many relationships in schema and validation, even when they do not yet auto-generate full cross-table transaction behavior.

### Data-Permission Hooks

Enterprise templates should enable data-scope hooks by default where appropriate, using existing shared hooks rather than module-local authorization inventions.

## 6. Quick Start

### 1. Install Dependencies

Install the frontend dependencies required by the wizard and its archive or export features.

### 2. Current One-Click Registration Boundary

The current service object is only `business/*` module generation and registration through `lowcode/generator` plus `lowcode/dynamicmodule`.

### 3. Add Internationalization Translations

Keep translations key-first and route generated assets into both backend seeds and frontend fallback resources.

### 4. Productized Feedback After Generation

Generation should produce meaningful summaries for what was written, what remains pending activation, and what next verification steps are required.

## 7. Usage Examples

Typical examples include:

- generating an order module from the visual wizard
- using the code-generation API
- dynamically registering a generated module

The Chinese primary guide remains the authoritative detailed example surface.

## 8. Test Checklist

### Backend Tests

Verify:

- compile health
- runtime tests
- registry and module-assembly integrity

### Frontend Tests

Verify:

- type checks
- generator contract smoke
- menu contract checks
- development-server startup

### End-to-End Scenarios

Verify the generated module can be created, registered, discovered, and browsed through the expected platform flows.

## 9. Generated Code Structure

Generated output should include:

- schema artifacts
- backend module scaffold
- frontend module scaffold
- generated registry updates
- i18n resources
- menu and permission metadata

## 10. Notes and Constraints

- keep generation inside `business/*`
- keep i18n key-first
- keep module contracts explicit
- do not let the generator bypass dynamic-module governance
- relation tables should not be promoted to first-class navigation pages without design review

## 11. Recommended Next Optimizations

Future optimization should focus on stronger governance metadata, safer relation modeling, better acceptance evidence, and higher-quality generation templates rather than expanding into uncontrolled runtime low-code behavior.

## 12. Related Files

### Generation Architecture

**Current architecture: Frontend TypeScript generators execute inside Go backend via Node.js delegation.**

Full call chain:

1. Frontend wizard (`ModuleWizard.tsx`) collects user schema, calls `/api/lowcode/generator/preview`
2. Backend handler (`generator_handler.go`) passes schema to `GeneratorService.PreviewGeneratedFiles`
3. `scaffold/workspace.go`'s `GenerateModuleFilesFromSchema` writes schema to temp JSON file
4. Via `exec.Command(node, script, schemaPath)` calls `frontend/scripts/export-generated-module.mjs`
5. Script temporarily compiles TypeScript → Node.js, executes `ModuleExporter.generateAll()`
6. Result JSON streams back to Go, returns `[]GeneratedFile` to frontend

Key files:

- `frontend/scripts/export-generated-module.mjs` — Node.js entry, delegates to `exporter.ts`
- `frontend/src/modules/lowcode/generator/exporter.ts` — Orchestrates backend + frontend generators
- `frontend/src/modules/lowcode/generator/backendGenerator.ts` — Go code generator
- `frontend/src/modules/lowcode/generator/frontendGenerator.ts` — TypeScript/React code generator
- `frontend/src/modules/lowcode/generator/schema.ts` — Shared schema type definition
- `backend/internal/scaffold/workspace.go` — Go-side entry, `text/template` generates registries
- `backend/internal/scaffold/engine.go` — Template engine stub (planned: replace Node delegation)
- `backend/internal/scaffold/types.go` — Shared type definitions

### Known Technical Debt

- `frontendGenerator.ts` (2439 LOC) and `backendGenerator.ts` (~800+ LOC) are monolithic template generators
- Detail page generator (`generateDetailPage`) contains ~1200 lines of inline template string
- `templates/` directory extracted: `detailTemplate_clean.txt`, `indexTemplate.ts`, `apiTemplate.ts`, `listTemplate.ts`, `formTemplate.ts`
- Plan: migrate backend Go code generation to `scaffold/engine.go`, eliminate Node delegation
- `frontendGenerator.ts`'s `generateDetailPage` (~1810 lines) needs further extraction into separate template segments

### Other Core Files
- `frontend/src/modules/lowcode/generator/pages/ModuleWizard.tsx` — Configuration wizard
- `frontend/src/modules/lowcode/generator/pages/components/FieldEditor.tsx` — Field editor
- `frontend/src/modules/lowcode/generator/pages/components/CodePreview.tsx` — Code preview
- `backend/modules/lowcode/dynamicmodule/dynamic_module_service.go` — Service layer
- `backend/modules/lowcode/dynamicmodule/dynamic_module_handler.go` — Handler
- `backend/modules/lowcode/dynamicmodule/module.go` — Module registration
- `frontend/src/modules/lowcode/dynamicmodule/api.ts` — API
- `frontend/src/modules/lowcode/dynamicmodule/ModuleManager.tsx` — Manager page

## 13. Acceptance Standard

Acceptance should verify:

- generated code matches module contracts
- i18n keys and seeds are complete
- menu and permission registration are coherent
- pending-activation and registry states are correct
- generated modules do not cross boundary lines into core system domains

## 14. Summary

The lightweight generator is a controlled developer-assistance tool for business-module onboarding. Its value is acceleration with governance, not replacing the platform architecture with a runtime low-code system.
