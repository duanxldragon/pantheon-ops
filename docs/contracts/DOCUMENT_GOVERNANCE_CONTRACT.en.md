---
title: Document Governance Contract
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-04-30
---

# Document Governance Contract

Chinese version: [DOCUMENT_GOVERNANCE_CONTRACT.md](./DOCUMENT_GOVERNANCE_CONTRACT.md)

This document defines the document-governance model used by Pantheon Base.

Its purpose is not to invent another category of “nice-looking document names”. The goal is to converge project documentation from a pile of materials into an executable governance system, so that design, implementation, assessment, remediation, and acceptance all move around the same contract chain.

## 1. Conclusion

The idea of “contract documents” is reasonable and fits Pantheon’s current stage.

The issue is not a lack of documentation. The issue is that the current docs have three structural problems:

- mixed types: design docs, assessment docs, remediation records, and acceptance samples sit on the same level
- distorted priority: long-lived norms and phase-specific review drafts enter the same primary entry surface
- missing anchor: many assessments and remediations exist, but without an upper contract that defines boundary, goal, definition of done, and acceptance semantics

So “contract document” here is not a legal artifact. It is:

> the execution contract for a layer, a module, or a long-lived topic.

It answers:

- who owns this
- what level of completion counts as done
- what is explicitly out of scope
- where later design, implementation, assessment, remediation, and acceptance should attach

## 2. Document Layering Model

Pantheon should stabilize around five document types:

### 2.1 `Contract`

- highest constraint doc for a layer, module, or governance topic
- defines boundary, goals, non-goals, definition of done, and acceptance standards
- long lifecycle
- belongs in the main index
- later `Design / Assessment / Remediation / Acceptance` docs should reference it

### 2.2 `Design`

- explains how the contract should be realized
- expands structure, interaction, API, state, strategy, and design details
- long lifecycle
- belongs in the main index
- must belong under a `Contract`

### 2.3 `Assessment`

- explains how far the current implementation is from contract/design intent
- outputs gap review, risk judgment, and delta matrices
- medium lifecycle
- not a first-line main-index entry by default
- must point back to a contract

### 2.4 `Remediation`

- explains how to close the gaps found by an assessment
- used for remediation plans, governance plans, and convergence steps
- medium lifecycle
- may appear in the main index, but as a secondary entry
- must belong under a contract or a clear governance chain

### 2.5 `Acceptance`

- records how the contract or design should be accepted
- includes templates, examples, matrices, and stage acceptance records
- template and baseline docs may be retained long-term
- one-off samples may survive as historical baselines
- must point back to a contract

## 3. Relationship Model

Correct model:

```text
Contract
  -> Design
  -> Assessment
  -> Remediation
  -> Acceptance
```

Patterns that should stop:

```text
Assessment -> Assessment -> Assessment
Remediation -> New Remediation -> New Review Draft
README flattening every dated document
```

Assessments must not become the next primary docs. Each remediation cycle must not create fresh floating dated files without an upper anchor. Index pages must not devolve into raw timelines.

## 4. Recommended Granularity

Do not begin by writing a contract for every page.

The current best first-layer contracts are:

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`

Topic-level contracts should be added only when the topic spans multiple pages/modules and has lasting governance value, for example:

- platform shell
- dynamic menu governance
- i18n runtime governance
- dynamic module governance
- generator governance

Use these tests before creating a new contract:

- does it span multiple pages or modules
- will it be cited repeatedly
- does it need long-term acceptance discipline
- does it need an independent definition of done

## 5. Standard Contract Structure

Every contract should contain:

1. background
2. ownership layer
3. goals
4. non-goals
5. boundaries
6. dependencies
7. hard constraints
8. definition of done
9. acceptance standards
10. related documents

## 6. Naming and Directory Strategy

Do not try to rename every historical file in one migration wave.

Safer approach:

- keep current filenames first
- mark the type in content/frontmatter
- for new or rewritten docs, prefer YAML frontmatter

Long-term target shape:

```text
docs/
  README.md
  contracts/
  designs/
  acceptances/
```

But the first round should focus on:

- rewriting indexes
- introducing a contract template
- producing the first contract set
- removing overwritten review drafts
- tagging remaining docs with type and ownership

## 7. Immediate Next Steps

Recommended rollout:

1. establish governance skeleton
2. create the first platform-level contracts
3. map and clean up remaining docs

The first-round goal is simple:

> form a stable primary chain of `Contract -> Design / Assessment / Remediation / Acceptance`.

