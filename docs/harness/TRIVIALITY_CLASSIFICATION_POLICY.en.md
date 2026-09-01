# Triviality Classification Policy

Chinese version: [TRIVIALITY_CLASSIFICATION_POLICY.md](./TRIVIALITY_CLASSIFICATION_POLICY.md)

Type: Policy
Layer: platform
Status: Active

This policy defines how work is classified as `trivial` or `non-trivial` so the repository does not rely on personal judgment alone when deciding whether task packets, evidence, and review artifacts are required.

## 1. Default Rule

Default to `non-trivial`.

Work may be marked `trivial` only when this policy explicitly allows it.

## 2. Decision Tree

Answer the following in order. If any answer is “yes”, the task is `non-trivial`:

1. Does it change code behavior, interfaces, schema, permissions, menus, audit, i18n, routes, UI states, or user flow?
2. Does it cross layers, or affect contracts, designs, acceptances, or governed docs?
3. Does it add, modify, or remove runtime dependencies, external services, CI gates, release rules, or security boundaries?
4. Does completion require dedicated verification commands, browser evidence, review artifacts, or residual-risk notes?
5. Does it involve any human-gated high-impact action?

Only if all answers are “no” should the task be checked against the `trivial` allowlist.

## 3. Trivial Allowlist

Only the following work may be marked `trivial`:

- a single typo fix
- a small clarification to comments or README text that does not change policy meaning
- read-only queries, log inspection, or status reporting
- formatting-only changes already covered by an existing formatter and not changing behavior

## 4. Output Requirements

### 4.1 Trivial

Even `trivial` work must still:

- state why it qualifies as trivial
- record the minimum verification performed, or explain why none was needed
- never bypass a human gate

### 4.2 Non-trivial

`non-trivial` work must ship with:

- a task packet, or an explicit link to an approved parent task packet
- verification evidence
- a review artifact
- known gaps and residual risk

## 5. Priority

If this policy conflicts with repository contracts or direct user instructions, direct user instructions and repository contracts win.
