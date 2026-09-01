---
title: Agent Handoff Protocol
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-27
version: v1.0
---

# Agent Handoff Protocol

Chinese version: [HANDOFF_PROTOCOL.md](./HANDOFF_PROTOCOL.md)

This document mirrors the Pantheon Base Harness handoff protocol. The Chinese version remains the detailed source for local execution rules.

## Purpose

Use this protocol when work moves between planning, exploration, implementation, review, verification, or human decision stages.

## Required Handoff State

Every handoff must preserve:

- current goal and scope;
- task packet or parent task packet reference;
- changed files and ownership boundaries;
- verification already run and remaining gaps;
- review findings and unresolved risks;
- human gates or destructive decisions, when applicable.

## Contract

- Do not treat chat history as the only state transfer mechanism.
- Durable task state belongs in `docs/harness/tasks/`, `.harness/tasks/`, and `.harness/evidence/`.
- A receiver must be able to continue from repository artifacts without reconstructing hidden context.
