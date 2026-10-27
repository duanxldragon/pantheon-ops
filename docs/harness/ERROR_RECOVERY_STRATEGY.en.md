---
title: Error Recovery Strategy
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-27
version: v1.0
---

# Error Recovery Strategy

Chinese version: [ERROR_RECOVERY_STRATEGY.md](./ERROR_RECOVERY_STRATEGY.md)

This document mirrors the Pantheon Base Harness error recovery policy. The Chinese version remains the detailed source for local execution rules.

## Purpose

Use this policy when an agent, CI gate, or review loop encounters a failure during delivery. The goal is to classify failures consistently and avoid uncontrolled retries.

## Recovery Classes

| Level | Meaning                                                  | Default Handling                                       |
| ----- | -------------------------------------------------------- | ------------------------------------------------------ |
| L1    | Transient or retryable failure                           | Retry with bounded backoff                             |
| L2    | Fixable implementation or configuration failure          | Fix, rerun the smallest proving check, record evidence |
| L3    | Ambiguous requirement or scope drift                     | Return to planning or human clarification              |
| L4    | Architecture, security, destructive, or human-gate issue | Stop and escalate to the human owner                   |

## Contract

- Record the failed command, environment, and recovery decision in `.harness/evidence/<task-id>/commands.json`.
- Do not hide failed attempts when they materially affected the final implementation.
- If a failure pattern repeats, promote it to `docs/harness/failure-registry.md` or a dedicated checker.
