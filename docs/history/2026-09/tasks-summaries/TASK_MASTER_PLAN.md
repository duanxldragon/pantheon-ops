# Pantheon Base - Post Cross-Review Task Master Plan

**Generated**: 2026-09-08  
**Source**: PANTHEON_BASE_CROSS_REVIEW_REPORT.md  
**Status**: Ready for execution

---

## Executive Summary

Based on the 6-AI cross-review report, **15 actionable tasks** have been generated and organized by priority. This master plan provides the roadmap to elevate Pantheon-Base from **7.8/10 (准企业级)** to **8.5/10+ (成熟企业级)**.

**Critical Path**: P0 tasks (3) → P1 tasks (6) → P2 tasks (6)

**Total Estimated Effort**: ~160 hours over 6 months

---

## Priority Overview

| Priority | Count | Total Effort | Blocking Status |
|----------|-------|--------------|-----------------|
| **P0 - Critical** | 3 | 4.5 hours | Blocks production deployment |
| **P1 - High** | 6 | 80 hours | Blocks enterprise readiness |
| **P2 - Medium** | 6 | 75 hours | Enhances competitiveness |

---

## P0 Tasks (Critical - Complete in 1 week)

### 1. License Declaration
- **ID**: `2026-09-08-p0-license-declaration`
- **Effort**: 20 minutes
- **Files**: `LICENSE`, `README.md`, `package.json`
- **Blocker**: Enterprise legal compliance red line
- **Status**: Ready to start

### 2. Security Gates Enforcement
- **ID**: `2026-09-08-p0-security-gates-enforcement`
- **Effort**: 1.75 hours
- **Files**: GitHub Branch Protection Rules, `QUALITY_AND_SECURITY_STRATEGY.md`
- **Blocker**: Security vulnerability window period
- **Status**: Ready to start

### 3. CSP/HSTS Implementation
- **ID**: `2026-09-08-p0-csp-hsts-implementation`
- **Effort**: 2.5 hours
- **Files**: `backend/internal/middleware/security_headers.go`, `SECURITY.md`
- **Blocker**: XSS, clickjacking, MITM attack surface
- **Status**: Ready to start

**P0 Total**: 4.45 hours (~1 work day)

---

## P1 Tasks (High - Complete in 3 months)

### 1. Test Coverage Phase 1 - Core Security Modules
- **ID**: `2026-09-08-p1-test-coverage-phase1`
- **Effort**: 44 hours (1 month)
- **Target**: Auth 60%, IAM 50%, Audit 50%, Overall 30%
- **Blocked By**: P0 tasks
- **Status**: Waiting for P0 completion

### 2. SSO/OIDC Design & Implementation
- **ID**: `2026-09-08-p1-sso-oidc-design`
- **Effort**: 12 hours
- **Files**: `backend/modules/auth/sso/`, OIDC provider integration
- **Blocker**: Enterprise identity source integration
- **Status**: Design can start in parallel

### 3. K8s Production Manifests
- **ID**: `2026-09-08-p1-k8s-manifests`
- **Effort**: 8 hours
- **Files**: `k8s/`, Deployment/Service/Ingress/HPA configs
- **Blocker**: Cloud deployment readiness
- **Status**: Can start in parallel

### 4. Data Permission Business Integration
- **ID**: `2026-09-08-p1-data-permission-integration`
- **Effort**: 6 hours
- **Files**: Department scope examples, user data filtering guides
- **Blocker**: Enterprise data isolation requirements
- **Status**: Waiting for coverage phase 1

### 5. Performance Baseline Testing
- **ID**: `2026-09-08-p1-performance-baseline`
- **Effort**: 8 hours
- **Files**: Performance test scripts, benchmark reports
- **Blocker**: Production capacity planning
- **Status**: Can start in parallel

### 6. Security Header Hardening (CSP Nonce)
- **ID**: `2026-09-08-p1-csp-nonce-hardening`
- **Effort**: 2 hours
- **Files**: CSP nonce generation, inline script refactoring
- **Blocked By**: P0-3 (CSP/HSTS)
- **Status**: Waiting for P0-3

**P1 Total**: 80 hours (~2 months with 40% allocation)

---

## P2 Tasks (Medium - Complete in 6 months)

### 1. Test Coverage Phase 2 - System Domain
- **ID**: `2026-09-08-p2-test-coverage-phase2`
- **Effort**: 24 hours
- **Target**: Org/I18n/Dict/Setting modules 40%, Overall 40%
- **Blocked By**: P1-1 (Coverage Phase 1)
- **Status**: ✅ Completed (2026-09-24) — packet `2026-09-08-p2-test-coverage-phase2`. Measured: org 62.2%, i18n 68.7%, dict 45.3%, setting 62.8%; overall 56.0% — all ≥ 40% target.

### 2. Multi-Tenant Design
- **ID**: `2026-09-08-p2-multi-tenant-design`
- **Effort**: 16 hours
- **Files**: Tenant isolation design, tenant DB routing
- **Status**: ✅ Completed (2026-09-24, design scope) — packet `2026-09-08-p2-multi-tenant-design`. Design delivered as `docs/designs/MULTI_TENANT_DESIGN.md`, superseded by frozen `docs/contracts/TENANT_CONTRACT_V1.md`; deployment manifests under `k8s/`.

### 3. Login Risk Control
- **ID**: `2026-09-08-p2-login-risk-control`
- **Effort**: 12 hours
- **Files**: Device fingerprinting, geo-location detection
- **Blocked By**: P1-2 (SSO/OIDC)

### 4. Community Building (CONTRIBUTING.md + Issue Templates)
- **ID**: `2026-09-08-p2-community-building`
- **Effort**: 4 hours
- **Files**: `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/`
- **Status**: ✅ Completed (2026-09-24) — packet `2026-09-08-p2-community-building`. `CONTRIBUTING.md` + `.github/ISSUE_TEMPLATE/` (3 templates) in place.

### 5. Test Coverage Phase 3 - Platform & Lowcode
- **ID**: `2026-09-08-p2-test-coverage-phase3`
- **Effort**: 16 hours
- **Target**: Platform/Lowcode 30%, Overall 50%
- **Blocked By**: P2-1 (Coverage Phase 2)
- **Status**: ✅ Completed (2026-09-24) — packet `2026-09-08-p2-test-coverage-phase3`. Measured: platform aggregate 75.2%, lowcode aggregate 56.3%, overall 56.0% — all ≥ target (30%/30%/50%).

### 6. Production Case Study Documentation
- **ID**: `2026-09-08-p2-production-case-study`
- **Effort**: 8 hours (ongoing)
- **Files**: Case study templates, deployment evidence
- **Status**: Requires real production deployments

**P2 Total**: 80 hours (~3 months with 25% allocation)

---

## Execution Strategy

### Week 1: P0 Sprint (Immediate)
```
Day 1: License Declaration (20 min)
Day 1: Security Gates Enforcement (2 hours)
Day 2: CSP/HSTS Implementation (2.5 hours)
Day 3: Verification & PR merge
```

### Month 1-2: P1 Core Quality (Parallel Tracks)
```
Track 1 (Primary): Test Coverage Phase 1 (44 hours)
  Week 1: Analysis
  Week 2-3: Core modules
  Week 4: Verification

Track 2 (Secondary): K8s Manifests (8 hours)
  Week 2-3: Implementation

Track 3 (Design): SSO/OIDC Design (12 hours)
  Week 1-2: Architecture design
  Week 3-4: Implementation
```

### Month 3: P1 Completion & P2 Start
```
- Data Permission Integration (6 hours)
- Performance Baseline (8 hours)
- CSP Nonce Hardening (2 hours)
- Start Test Coverage Phase 2 (24 hours)
```

### Month 4-6: P2 Long-tail
```
- Multi-tenant Design
- Login Risk Control
- Test Coverage Phase 3
- Community Building
- Production Cases (ongoing)
```

---

## Risk Mitigation

### Risk 1: Test Coverage Effort Underestimated
- **Probability**: Medium
- **Impact**: High
- **Mitigation**: Start with Phase 1 only, re-evaluate effort after 2 weeks

### Risk 2: SSO/OIDC Complexity
- **Probability**: Medium
- **Impact**: Medium
- **Mitigation**: Design-first approach, consider third-party libraries

### Risk 3: P0 Tasks Block Everything
- **Probability**: Low
- **Impact**: Critical
- **Mitigation**: P0 tasks are simple (4.5 hours total), minimal risk

---

## Success Metrics

### Short-term (1 month)
- [ ] P0 tasks 100% complete
- [ ] Test coverage: 12.2% → 30%
- [ ] Security headers: CSP/HSTS live
- [ ] License declared

### Mid-term (3 months)
- [ ] P1 tasks 80% complete
- [ ] Core modules: 60%+ coverage
- [ ] SSO/OIDC implemented
- [ ] K8s manifests available

### Long-term (6 months)
- [ ] Overall coverage: 40-50%
- [ ] Multi-tenant design complete
- [ ] Enterprise readiness score: 8.5/10+
- [ ] At least 2 production case studies

---

## Task Dependency Graph

```
P0-1 (License) ────────────────┐
P0-2 (Security Gates) ─────────┼──→ Production Deployment OK
P0-3 (CSP/HSTS) ───────────────┘
        │
        ├──→ P1-1 (Test Coverage Phase 1) ──→ P1-4 (Data Permission)
        │                                  └──→ P2-1 (Coverage Phase 2) ──→ P2-5 (Coverage Phase 3)
        │
        ├──→ P1-2 (SSO/OIDC) ──→ P2-3 (Login Risk Control)
        │
        ├──→ P1-3 (K8s Manifests) ──→ Production Cloud Deployment OK
        │
        ├──→ P1-5 (Performance Baseline) ──→ Capacity Planning OK
        │
        └──→ P1-6 (CSP Nonce) ──→ Security Hardening Complete

P2-2 (Multi-tenant Design) ─── Independent track
P2-4 (Community Building) ───── Independent track
P2-6 (Case Studies) ──────────── Ongoing
```

---

## Approval & Kickoff

**Prepared by**: Claude (Cross-Review Synthesizer)  
**Reviewed by**: _[Pending]_  
**Approved by**: _[Project Owner]_  
**Kickoff Date**: _[To be scheduled]_

**Next Action**: Review this master plan → Approve P0 tasks → Execute Week 1 sprint

---

## Task Directory Structure

All task packets follow the harness methodology and are located at:

```
pantheon-base/.harness/tasks/
├── 2026-09-08-p0-license-declaration/
│   ├── task.md
│   └── manifest.json
├── 2026-09-08-p0-security-gates-enforcement/
│   ├── task.md
│   └── manifest.json
├── 2026-09-08-p0-csp-hsts-implementation/
│   ├── task.md
│   └── manifest.json
├── 2026-09-08-p1-test-coverage-phase1/
│   ├── task.md
│   └── manifest.json
└── [... 11 more tasks to be generated]
```

Evidence will be captured at:
```
pantheon-base/.harness/evidence/
└── [task-id]/
    ├── commands.json
    ├── summary.md
    └── review.md
```

---

**End of Master Plan**

Refer to individual task packets for detailed implementation guidance.
