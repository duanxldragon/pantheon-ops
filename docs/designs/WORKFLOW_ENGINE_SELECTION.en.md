---
title: Workflow Engine Selection
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-26
---

# Workflow Engine Selection

Chinese version: [WORKFLOW_ENGINE_SELECTION.md](./WORKFLOW_ENGINE_SELECTION.md)

## 1. Background

Pantheon Base currently has no workflow engine. The `WORKFLOW.md` design document describes approval flow and RBAC integration requirements but has not been implemented.

This document evaluates major workflow engines for decision-making.

---

## 2. Engine Comparison

### 2.1 Camunda

| Dimension      | Score      | Notes                               |
| -------------- | ---------- | ----------------------------------- |
| Maturity       | ⭐⭐⭐⭐⭐ | BPMN 2.0 standard, industrial-grade |
| Learning Curve | ⭐⭐       | Steep                               |
| Integration    | ⭐⭐       | Requires standalone service         |
| Features       | ⭐⭐⭐⭐⭐ | All patterns supported              |
| Community      | ⭐⭐⭐⭐   | Mature                              |
| Go Integration | ⭐⭐       | REST API only                       |

### 2.2 Temporal

| Dimension      | Score      | Notes                                     |
| -------------- | ---------- | ----------------------------------------- |
| Maturity       | ⭐⭐⭐⭐   | Cloud-native, microservice-friendly       |
| Learning Curve | ⭐⭐⭐     | New concepts, good docs                   |
| Integration    | ⭐⭐⭐     | Requires service deployment               |
| Features       | ⭐⭐⭐⭐   | Long-running tasks, retries, compensation |
| Community      | ⭐⭐⭐⭐   | Active, Airbnb-backed                     |
| Go Integration | ⭐⭐⭐⭐⭐ | Native Go SDK                             |

### 2.3 Self-Built Lightweight Engine

| Dimension      | Score      | Notes                           |
| -------------- | ---------- | ------------------------------- |
| Maturity       | ⭐⭐       | Build from scratch              |
| Learning Curve | ⭐⭐⭐     | No external dependencies        |
| Integration    | ⭐⭐⭐⭐⭐ | In-process, no network overhead |
| Features       | ⭐⭐       | Basic approvals only            |

---

## 3. Recommendation

### Short-term: Self-built for basic approvals

- No external dependencies
- Fast implementation
- Sufficient for single-approver scenarios

### Long-term: Temporal for complex workflows

- Native Go SDK
- Complete error handling, retries
- Activity history

---

## 4. Decision Matrix

| Scenario                                  | Recommendation | Reason          |
| ----------------------------------------- | -------------- | --------------- |
| Simple approval (≤3 steps)                | Self-built     | No dependencies |
| Complex approval (signatures, conditions) | Camunda        | BPMN standard   |
| Long-running tasks                        | Temporal       | Native Go       |
