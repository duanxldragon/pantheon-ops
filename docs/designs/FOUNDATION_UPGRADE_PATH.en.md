# Foundation Release Upgrade Path

Chinese version: [FOUNDATION_UPGRADE_PATH.md](./FOUNDATION_UPGRADE_PATH.md)

Updated: 2026-07-17

Type: Design
Layer: `platform`
Status: Active

This is the rolling upgrade record for the `pantheon-base` foundation releases consumed by `pantheon-ops`. Every real upgrade must update the current baseline and append a historical record.

## Current Baseline

- `foundation-release.lock.json` is locked to `base-v0.8.11`
- Base commit: `a201a13f05615903f2d57df4809e2a923a87c668`
- Consumer mode: `foundation-release-consumer`
- Local release artifact: `.foundation/releases/base-v0.8.11`

Between `base-v0.8.5` and `base-v0.8.11`, the repository completed multiple foundation-release-consumer upgrades through `base-v0.8.8`, `base-v0.8.9`, `base-v0.8.10`, and `base-v0.8.11`. Matching artifacts remain under `.foundation/releases/`, but per-version details were not recorded. Every future upgrade must append its version, commit, scope, verification, and rollback information here.

## Historical Upgrade Records

### base-v0.8.4 -> base-v0.8.5

#### Basic Information

| Field | Value |
|---|---|
| From | `base-v0.8.4`（`412a465e7c3a2620b81e81edf3d5c9a16fb33952`）|
| To | `base-v0.8.5`（`185e31f030108ad0e85cf8a6c87912222656ba3a`）|
| Published | 2026-06-24 |
| Consumer Mode | `foundation-release-consumer` |
| Change Summary | Auth foundation hardening, secure-action runtime updates, upload safety fixes, release artifact packaging |

#### Change Scope

##### Backend Changes

| Path | Change Type | Description |
|---|---|---|
| `backend/internal/middleware/secure_action_middleware.go` | MODIFY | Secure-action middleware hardening |
| `backend/internal/middleware/operation_log_middleware.go` | MODIFY | Operation log middleware changes |
| `backend/modules/auth/login/login_handler.go` | MODIFY | Login handler updates |
| `backend/modules/auth/login/login_runtime.go` | MODIFY | Login runtime updates |
| `backend/modules/auth/security/security_service.go` | MODIFY | Security service hardening |
| `backend/modules/auth/mfa/mfa_service.go` | MODIFY | MFA service updates |
| `backend/modules/auth/session/session_service.go` | MODIFY | Session service updates |
| `backend/modules/system/config/setting/setting_handler.go` | MODIFY | Setting handler updates |
| `backend/pkg/authtoken/token.go` | MODIFY | Auth token handling updates |
| `backend/pkg/upload/service.go` | MODIFY | Upload service security fixes |

##### Frontend Changes

| Path | Change Type | Description |
|---|---|---|
| `frontend/src/store/authTypes.ts` | MODIFY | Auth type definitions updates |
| `frontend/src/modules/system/user/*` | MODIFY | User management page updates |
| `frontend/src/modules/system/setting/*` | MODIFY | System setting page updates |
| `frontend/src/modules/system/role/*` | MODIFY | Role management page updates |

##### New Files

- `frontend/src/modules/system/post/PostList.tsx`（post list）
- `frontend/src/modules/system/permission/PermissionWorkbenchTab.tsx`（permission workbench tab）

#### ops Local Overlays Are Unaffected

The following paths are ops-local extension points and will not be overwritten by the upgrade:

- `backend/modules/business/generated_registry.go`
- `backend/modules/system/iam/menu/generated_component_registry.go`
- `frontend/src/core/router/componentRegistry.ts`
- `frontend/src/core/router/generatedComponentRegistry.ts`
- `frontend/src/core/router/modules.ts`
- `frontend/src/modules/system/generator/backend-generator.ts`
- `backend/modules/system/i18n/builtin_locale_resources.json`（`business.*` keys will be preserved and merged）

#### Upgrade Steps

##### Step 1: Record current base version

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-base
git rev-parse --short HEAD
```

##### Step 2: Preview upgrade diff (recommended)

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-ops
npm run upgrade:foundation:local-plan -- --release-version base-v0.8.5
```

This runs in dry-run mode and shows all shared files that would change, without modifying anything.

##### Step 3: Apply the upgrade

```powershell
npm run upgrade:foundation:local-apply -- --release-version base-v0.8.5
```

This will:

1. Stash any uncommitted ops business changes
2. Apply `base-v0.8.5` shared-backend and shared-frontend files (with module reference rewriting)
3. Preserve ops-local overlay files
4. Update inheritance docs and `foundation-release.lock.json`
5. Merge `business.*` keys into `builtin_locale_resources.json`
6. Run `go build ./...` to validate backend module rewrites
7. Run `check:inheritance` for full inheritance validation

##### Step 4: Verify

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-ops
go build ./backend/...

cd frontend
npm run build

npm run check:inheritance
```

##### Step 5: Rollback if needed

```powershell
cd D:\workspace\go\pantheon-platform\pantheon-ops
git checkout -- .
git stash pop
```

Then manually restore `foundation-release.lock.json`, `docs/PROJECT_INHERITANCE.md`, and `docs/PROJECT_INHERITANCE.en.md`.

#### Expected Outcome

After a successful upgrade:

- `foundation-release.lock.json` shows `releaseVersion: "base-v0.8.5"` and `baseCommit: "185e31f030108ad0e85cf8a6c87912222656ba3a"`
- All shared backend/frontend files are byte-identical to `base-v0.8.5` (overlay files excluded)
- Both `go build ./...` and `check:inheritance` pass

#### Rollback Notes

- `upgrade:foundation:local-apply` automatically stashes local changes before writing files
- If `go build` fails, the script reports the failure without auto-rollback
- Manual rollback: `git checkout -- .` + `git stash pop`
- `foundation-release.lock.json` version numbers must be manually restored
