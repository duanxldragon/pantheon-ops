# Pantheon-Ops SRE Evolution Plan

Chinese version: [PLATFORM_SRE_EVOLUTION_PLAN.md](./PLATFORM_SRE_EVOLUTION_PLAN.md)

Updated: 2026-05-19

Type: Design (Roadmap)  
Layer: `platform`  
Status: Active

This roadmap defines how `pantheon-ops` can evolve from a web-based operations console into a Kubernetes-native SRE platform.

It is based on the actual code baseline captured on May 11, 2026 and extends the execution directions reserved in `BUSINESS_DEPLOY_MODULE_DESIGN.md`.

## Current Baseline

Already in place:

- CMDB host, group, and label management
- deployment task orchestration skeleton
- four-layer permission model and Casbin integration
- low-code generator and dynamic module lifecycle
- a working condition-expression engine

Main gaps:

- no real execution engine yet
- no agent communication loop
- no Kubernetes-native control plane
- no observability stack
- no production deployment packaging
- no CI/CD pipeline

## Five Stages

### Stage 1: Real Execution Engine

Introduce real SSH and agent-backed execution for deployment tasks.

Key outputs:

- `backend/pkg/executor/ssh_executor.go`
- `backend/pkg/executor/agent_executor.go`
- `deploy_service.go` orchestration changes

Primary learning areas:

- Go SSH clients
- concurrent execution
- context cancellation
- result writeback semantics

### Stage 2: Agent Sidecar

Build a separate Go agent service that can run on Kubernetes nodes, pull tasks, execute commands, report results, and expose Prometheus metrics.

Key outputs:

- `pantheon-agent/` standalone module
- long-poll or equivalent task retrieval
- execution reporting
- `/metrics` endpoint
- Docker image and DaemonSet deployment

### Stage 3: Kubernetes Operator

Build a `kubebuilder`-based operator so CMDB hosts and deploy tasks become Kubernetes-native resources.

Key outputs:

- `Host` and `DeployTask` CRDs
- controllers and webhooks
- Helm chart packaging
- sync bridge between CRDs and current Pantheon data models

### Stage 4: Observability

Add full-stack metrics, logging, dashboards, and alerting across `pantheon-ops`, agents, and the operator.

Key outputs:

- Prometheus metrics endpoints
- Grafana dashboards
- Loki/Promtail log collection
- Alertmanager rules

### Stage 5: Packaging and One-Click Deployment

Package the stack so a bare Kubernetes cluster can be turned into a runnable platform quickly.

Key outputs:

- `pantheon-ops` Dockerfile
- top-level Helm chart
- deployment script
- short operator handbook

## Skill Growth Target

The roadmap is also a learning path:

- stronger Go package design
- network and process execution
- Kubernetes controller patterns
- observability system design
- Docker and Helm packaging
- end-to-end delivery workflow

## Immediate Start

The first concrete milestone is small and practical: build `backend/pkg/executor/ssh_executor.go` and prove remote command execution end to end.

That step is the intended transition point from CRUD-only platform work into real SRE execution mechanics.

For the detailed stage tables, project skeletons, file paths, and week-by-week estimates, use the Chinese source as the authoritative roadmap.
