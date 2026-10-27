# Upgrading to pantheon-base v0.11.1

**Target Version**: v0.11.1  
**From Version**: v0.10.x (any)  
**Compatibility**: 100% backward compatible  
**Estimated Duration**: < 1 hour (excluding full smoke suite)

---

## Overview

pantheon-base v0.11.1 is a **documentation and tooling release** with zero breaking changes. All modifications are additive:

- ✅ No API endpoint changes
- ✅ No database schema changes  
- ✅ No configuration changes
- ✅ No authentication flow changes

Upgrade is safe and straightforward for all consumers.

---

## Prerequisites

- Clean git worktree in consuming repository
- Go 1.21+ installed
- Node.js 20+ and npm 10+ installed
- MySQL 8.0+ running (for smoke tests)
- Redis 7.0+ running (for smoke tests)

---

## Backend Upgrade (Go Module)

### Step 1: Update go.mod

**Option A: Manual edit**

Edit `backend/go.mod`:
```go
require github.com/duanxldragon/pantheon-base/backend v0.11.1
```

Keep local replace directive for development:
```go
replace github.com/duanxldragon/pantheon-base/backend => ../../pantheon-base/backend
```

**Option B: go get command**

```bash
cd backend
go get github.com/duanxldragon/pantheon-base/backend@v0.11.1
```

### Step 2: Resolve dependencies

```bash
go mod tidy
```

### Step 3: Verify compilation

```bash
go build ./cmd/server
```

**Expected**: Build succeeds with no errors.

### Step 4: Run tests

```bash
go test ./...
```

**Expected**: All tests pass (no code changes required).

---

## Frontend Upgrade (NPM Package)

### Step 1: Build base UI package

Navigate to pantheon-base frontend:

```bash
cd pantheon-base/frontend
```

Build library distribution:

```bash
npm run build:lib
```

**Expected output**:
```
vite v5.x building for production...
✓ 249 modules transformed.
dist/index.js          X KB
dist/index.d.ts        Y KB
✓ built in Xs
```

Create NPM package tarball:

```bash
npm pack
```

**Expected output**:
```
duanxldragon-pantheon-base-ui-0.11.1.tgz
```

Verify tarball integrity:

```bash
ls -lh duanxldragon-pantheon-base-ui-0.11.1.tgz
sha256sum duanxldragon-pantheon-base-ui-0.11.1.tgz
```

### Step 2: Update consumer dependency

Navigate to consumer frontend (e.g., pantheon-ops):

```bash
cd ../../pantheon-ops/frontend
```

Edit `package.json`:

```json
{
  "dependencies": {
    "@duanxldragon/pantheon-base-ui": "file:../../pantheon-base/frontend/duanxldragon-pantheon-base-ui-0.11.1.tgz"
  }
}
```

**Note**: Path must be correct relative to consumer's package.json location.

### Step 3: Reinstall dependencies

```bash
npm install
```

Verify installed version:

```bash
npm list @duanxldragon/pantheon-base-ui
```

**Expected output**:
```
└── @duanxldragon/pantheon-base-ui@0.11.1
```

### Step 4: Type check

```bash
npm run type-check
```

**Expected**: Zero TypeScript errors (100% compatible).

---

## Contract Validation

Validate that business modules still comply with base contracts:

### Menu Contract

```bash
npm run check:menu-contract
```

**Expected**: PASS (no menu structure changes in v0.11.1).

### UI Contract

```bash
npm run check:ui-contract
```

**Expected**: PASS (validates design token usage).

### Business Boundaries

```bash
npm run check:business-boundaries
```

**Expected**: PASS (validates business/base separation).

### Troubleshooting Contract Failures

**If menu contract fails**:
- Base added new required menu fields → review error message
- Update business menu definitions to include new fields

**If UI contract fails**:
- Business modules use hardcoded values instead of tokens → migrate to semantic tokens
- See `docs/frontend/TOKEN_MIGRATION_GUIDE.md`

**If boundaries check fails**:
- Business modules accidentally override shared paths → rename conflicting files
- Review `.harness/contracts/boundaries.json`

---

## Smoke Testing

### Quick Smoke (5-10 minutes)

Platform-level contracts only:

```bash
cd frontend
npm run test:smoke:platform:contracts
```

System pages validation:

```bash
npm run test:smoke:system:pages
```

### Business Module Smoke (Optional)

If consumer has business-specific smoke tests:

```bash
npm run test:smoke:business:cmdb   # If exists
npm run test:smoke:business:k8s    # If exists
# ... other business modules
```

### Full Regression Suite (15-30 minutes)

**Warning**: This runs 386 Playwright specs. Only run if time permits.

```bash
npm run test:smoke:all
```

**Expected**: All specs pass (100% compatible).

**If tests fail**:
1. Check base release notes for unexpected behavior changes
2. Review test failure logs in `.harness/evidence/`
3. Verify backend and Redis are running
4. Rollback if failures indicate real incompatibility (unlikely)

---

## Optional: Adopt New Features

v0.11.1 introduces optional capabilities that require no immediate action:

### 1. Design Token System (67 tokens available)

**Before** (v0.10.x, 32 tokens):
```css
.my-component {
  padding: 16px;  /* Hardcoded */
}
```

**After** (v0.11.1, 67 tokens):
```css
.my-component {
  padding: var(--space-lg);  /* Semantic token */
}
```

**Benefits**:
- Consistent spacing across all components
- Theme-aware (light/dark mode support)
- Mechanical validation via `check-ui-contract`

**Guide**: `docs/frontend/TOKEN_MIGRATION_GUIDE.md`

### 2. Container Semantics (9 new tokens)

**Interactive containers** (inputs, buttons, clickable cards):
```css
.my-input {
  border-color: var(--container-interactive-border);
  background: var(--container-interactive-bg);
}

.my-input:hover {
  border-color: var(--container-interactive-border-hover);
}
```

**Display containers** (stats, descriptions, info cards):
```css
.my-stat-card {
  border-color: var(--container-display-border);
  background: var(--container-display-bg);
}
```

**Guide**: `docs/frontend/COMPONENT_STYLING_GUIDE.md`

### 3. Engineering Documentation (3,325 lines)

Five comprehensive guides now available:

- **COMPONENT_STYLING_GUIDE.md**: Atomic design patterns, styling rules
- **UI_PATTERN_LIBRARY.md**: 18 operational primitives with examples
- **DESIGN_ENGINEERING_GUIDE.md**: 4-layer engineering methodology
- **TOKEN_MIGRATION_GUIDE.md**: Migration tooling, gradual adoption
- **VERSION_MANAGEMENT_GUIDE.md**: Semantic versioning, release procedures

**Location**: `docs/frontend/`

---

## Foundation Release Lock (For pantheon-ops)

If consumer uses `foundation-release.lock.json`:

Update lock file:

```json
{
  "schemaVersion": 1,
  "baseVersion": "pantheon-base-v0.11.1",
  "releaseLine": "release/0.11",
  "releaseDate": "2026-09-04",
  "mode": "npm-package-consumer",
  "dependencies": {
    "go": {
      "module": "github.com/duanxldragon/pantheon-base/backend",
      "version": "v0.11.1"
    },
    "npm": {
      "package": "@duanxldragon/pantheon-base-ui",
      "version": "0.11.1",
      "tarball": "file:../../pantheon-base/frontend/duanxldragon-pantheon-base-ui-0.11.1.tgz"
    }
  }
}
```

Commit lock file with dependency changes:

```bash
git add foundation-release.lock.json backend/go.mod backend/go.sum frontend/package.json frontend/package-lock.json
git commit -m "chore: upgrade foundation to pantheon-base-v0.11.1"
```

---

## Rollback Procedure

If unexpected issues arise, rollback is straightforward:

### Step 1: Revert commit

```bash
git revert HEAD  # Undo upgrade commit
```

### Step 2: Reinstall dependencies

Backend:
```bash
cd backend
go mod tidy
```

Frontend:
```bash
cd ../frontend
npm install
```

### Step 3: Verify rollback

Backend:
```bash
cd ../backend
go list -m github.com/duanxldragon/pantheon-base/backend
# Should show: v0.10.x (previous version)
```

Frontend:
```bash
cd ../frontend
npm list @duanxldragon/pantheon-base-ui
# Should show: @duanxldragon/pantheon-base-ui@0.10.x
```

### Step 4: Restart services

```bash
# Backend
cd ../backend
go run ./cmd/server

# Frontend (new terminal)
cd ../frontend
npm run dev
```

---

## Troubleshooting

### Issue: npm install fails "tarball not found"

**Cause**: Base UI package tarball not built.

**Fix**:
```bash
cd pantheon-base/frontend
npm run build:lib
npm pack
ls -lh duanxldragon-pantheon-base-ui-0.11.1.tgz  # Verify exists
```

### Issue: Type errors after upgrade

**Cause**: TypeScript cannot find new types (should not happen in v0.11.1).

**Diagnosis**:
```bash
cat node_modules/@duanxldragon/pantheon-base-ui/dist/index.d.ts | head -50
```

**Expected**: All exported types from v0.10.x still present (100% compatible).

**Fix**: If types genuinely missing (regression), report issue and rollback.

### Issue: Contract check failures

**Cause**: Business modules use patterns that violate new validations.

**Diagnosis**: Read contract check error messages carefully.

**Fix**: Update business code to comply with contracts (typically trivial).

### Issue: Smoke tests fail

**Cause 1**: Backend or Redis not running.

**Fix**: Start services:
```bash
# Terminal 1
cd backend && go run ./cmd/server

# Terminal 2
redis-server
```

**Cause 2**: Database migrations not applied.

**Fix**: Migrations run on backend startup. Check logs for errors.

**Cause 3**: Genuine regression (unlikely in v0.11.1).

**Fix**: Review test logs, report issue, rollback.

### Issue: Build performance degradation

**Cause**: Not specific to v0.11.1.

**Diagnosis**: Compare build times:
```bash
time npm run build  # Before vs after upgrade
```

**Expected**: No significant difference (documentation-only release).

---

## Post-Upgrade Checklist

After successful upgrade, verify:

- [ ] Backend compiles: `go build ./cmd/server`
- [ ] Backend tests pass: `go test ./...`
- [ ] Frontend type-checks: `npm run type-check`
- [ ] Menu contract passes: `npm run check:menu-contract`
- [ ] UI contract passes: `npm run check:ui-contract`
- [ ] Boundaries check passes: `npm run check:business-boundaries`
- [ ] Quick smoke passes: `npm run test:smoke:platform:contracts`
- [ ] Services start: Backend + Frontend dev servers
- [ ] Foundation lock updated (if applicable)
- [ ] Commit created: "chore: upgrade foundation to pantheon-base-v0.11.1"

---

## Support

**Issues during upgrade?**

1. Review this guide's Troubleshooting section
2. Check `releases/pantheon-base-v0.11.1/consumer-impact.md`
3. Check GitHub issues: https://github.com/duanxldragon/pantheon-base/issues
4. Report new issues with:
   - Your platform-ops version
   - Error messages (full logs)
   - Steps to reproduce

**Questions about new features?**

- Design tokens: `docs/frontend/TOKEN_MIGRATION_GUIDE.md`
- Engineering guides: `docs/frontend/DESIGN_ENGINEERING_GUIDE.md`
- Release notes: `releases/pantheon-base-v0.11.1/release-notes.md`

---

**Upgrade Guide Version**: 1.0  
**Last Updated**: 2026-09-04  
**Compatibility**: pantheon-base v0.11.1  
**Author**: duanxiaolong <435000465@qq.com>
