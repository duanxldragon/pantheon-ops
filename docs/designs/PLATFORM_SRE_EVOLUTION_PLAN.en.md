# Pantheon-Ops SRE Evolution Plan

Chinese version: [PLATFORM_SRE_EVOLUTION_PLAN.md](./PLATFORM_SRE_EVOLUTION_PLAN.md)

Updated: 2026-07-17

Type: Design (Roadmap)
Layer: `platform`
Status: Active
Author: duanxldragon

This roadmap evolves `pantheon-ops` from a web operations console into a Kubernetes-native SRE platform. It reflects the current implementation truth and continues the Agent/SSH direction reserved by the Deploy design.

---

## 1. Current Code Baseline

### 1.1 Existing Capabilities

| Capability | Location | Maturity |
| :--- | :--- | :--- |
| CMDB hosts/groups/labels | `backend/modules/business/cmdb/` | production-ready |
| deployment orchestration and target expressions | `backend/modules/business/deploy/` | orchestration-ready |
| four-layer permissions and Casbin | `backend/pkg/database/casbin.go` | mature |
| low-code generator | `backend/modules/lowcode/generator/` | extensible |
| dynamic module lifecycle | `backend/modules/lowcode/dynamicmodule/` | extensible |
| AND/OR + eq/neq/in/notIn DSL | `deploy_service.go` | usable; later map to K8s label selectors |

### 1.2 Capability Gaps

| Gap | Current truth | Target |
| :--- | :--- | :--- |
| real execution engine | a real SSH loop is implemented inline in `business/deploy/deploy_service.go` using `ssh.Dial` with automatic result write-back | extract `backend/pkg/executor/`; add Agent and K8s Job modes |
| Agent communication | `ExecutorTypeAgent` exists without implementation | Agent Sidecar pulls tasks and reports results |
| Kubernetes integration | no `client-go` or kubeconfig | Operator, CRDs, and webhooks |
| observability | no metrics endpoint or structured logging baseline | Prometheus, Loki, tracing |
| packaging | docker-compose is development-only | Dockerfile, Helm chart, one-command deployment |
| CI/CD | absent | commit to image to Helm deployment |

---

## 2. Five-Stage Evolution Route

### Stage 1: Real Execution Engine (SSH + Agent Communication)

**Goal:** extract a clean executor boundary from the working inline SSH loop and add Agent communication.

| Step | Module | Current/remaining work |
| :--- | :--- | :--- |
| 1.1 | `backend/pkg/executor/ssh_executor.go` | implemented inline; extract connection, command execution, stdout/stderr, timeout, and host snapshot handling from `deploy_service.go` |
| 1.2 | `backend/pkg/executor/agent_executor.go` | pending: HTTP task delivery with retry and idempotency |
| 1.3 | `backend/modules/business/deploy/deploy_service.go` | SSH selection/execution/write-back is implemented inline; refactor to unified executor dispatch, then add concurrency/cancellation semantics |
| 1.4 | `backend/modules/business/deploy/deploy_service.go` | pending: distinguish manual result marking from executor reports |

Acceptance keeps the already working create/start/SSH/automatic-write-back loop, adds multi-host isolation, and preserves stdout/stderr/error recording.

### Stage 2: Agent Sidecar

Build a separate `pantheon-agent/` Go module that long-polls tasks, executes shell commands, streams or reports results, sends heartbeats, exposes Prometheus metrics, ships as a container, and runs as a DaemonSet.

Acceptance requires an end-to-end Agent task loop, metrics scraping, and offline detection.

### Stage 3: Kubernetes Operator

Build a kubebuilder-based `pantheon-operator/` with `Host` and `DeployTask` CRDs, reconcilers, validating/mutating webhooks, RBAC, and Helm packaging. Existing GORM models map to CRD spec/status, service validation maps to webhooks, and Deploy state transitions map to status conditions.

Acceptance requires CR-to-CMDB synchronization, task Job reconciliation, webhook rejection/defaulting, and Helm installation.

### Stage 4: Observability

Add Pantheon API/task metrics, Prometheus and Grafana, Agent metrics, Loki/Promtail logs, Alertmanager rules, and a notification webhook integration.

Core metrics cover API requests/latency, deployment task counts, Agent heartbeats, and Agent execution duration. Acceptance requires live dashboards, routed failure alerts, and searchable execution logs.

### Stage 5: Packaging and One-Click Deployment

Add a multi-stage `pantheon-ops` Dockerfile, a top-level Helm chart for the full stack, a deployment script, and a concise operations handbook. The target is a runnable stack on a blank Kubernetes cluster within 30 minutes.

---

## 3. Skill Matrix

| Area | Current | After Stage 1 | After Stage 3 | After Stage 5 |
| :--- | :--- | :--- | :--- | :--- |
| Go | framework usage | independent package design | complete Operator | independent Go services |
| Kubernetes | installation/operations | unchanged | CRD/Operator/Webhook development | native architecture design |
| Networking | basic HTTP | SSH and HTTP clients | K8s APIs | end-to-end integration |
| Observability | Prometheus usage | unchanged | exporter development | full-stack observability design |
| Docker/Helm | docker-compose | Dockerfile | Helm chart | one-command deployment |
| CI/CD | absent | unchanged | unchanged | commit to image to deploy |

---

## 4. Expected Timeline

| Stage | Estimate | Start condition |
| :--- | :--- | :--- |
| Stage 1: execution engine | 1-2 weeks | partially completed in the June 2026 Deploy closure; remaining focus is executor extraction and Agent work |
| Stage 2: Agent Sidecar | 4-5 weeks | Stage 1 complete |
| Stage 3: Kubernetes Operator | 5-6 weeks | Stage 2 complete |
| Stage 4: observability | 3-4 weeks | Stage 2 complete; may overlap Stage 3 |
| Stage 5: packaging | 2 weeks | Stages 3 and 4 complete |

Stage 1 was partially completed during the June 2026 Deploy closure; the remaining focus shifts to executor package extraction and Agent development.

---

## 5. Immediate Start (First Remaining Stage 1 Step)

Extract the proven inline SSH remote execution from `deploy_service.go` into a testable `backend/pkg/executor/ssh_executor.go`.

The extraction must preserve host fingerprint validation, request-scoped credentials, result write-back, and existing error-key semantics.

---

## 6. References

- Base architecture: `../../pantheon-base/DESIGN.md`
- [Deploy module design](./BUSINESS_DEPLOY_MODULE_DESIGN.md)
- [CMDB module design](./BUSINESS_CMDB_MODULE_DESIGN.md)
- Base permission model and module contract documents
