# Review — 2026-10-31-frontend-follow-redirects-audit

## Posture readings

1. **Boundary** — the change is a single entry in the consumer-owned `frontend/package-lock.json`.
   `package.json`, the security workflow and every source path are untouched.
2. **Failure attribution** — `main` went red without a commit. The advisory range changed under a
   pinned dependency, so the attribution is time, not code.
3. **Fix minimality** — `1.16.0` and `1.16.1` are the only vulnerable releases; `1.16.2` is the
   smallest step out. The diff is exactly three lines in one lock entry.
4. **Tool choice** — `npm audit fix` and `npm update` both drag unrelated resolutions into the
   diff; the direct entry edit plus `npm ci` proves the lock is self-consistent.
5. **Gate integrity** — nothing is suppressed. No `overrides`, no audit exemptions, no workflow
   threshold change, no `--omit=dev` trick. The gate keeps catching this class.
6. **Residual advisories** — the 19 moderate entries stay; they are below the enforced severity and
   clearing them requires major toolchain upgrades that belong in their own change.
7. **Base parity** — `pantheon-base` carries the same `follow-redirects@1.16.1` pin and gets the
   identical lock entry edit in the same round, so the fix is not consumer-only while the base goes
   red on its own schedule.

## Verdict

Approve. The change is a lockfile refresh with no behavioural surface.

## Machine Readable

```json
{
  "taskId": "2026-10-31-frontend-follow-redirects-audit",
  "verdict": "approve",
  "risk": "low",
  "boundary": "frontend/package-lock.json (single entry)",
  "behaviourChange": false,
  "gateSuppression": false,
  "verification": ["npm ci", "npm ls follow-redirects", "npm audit (high=0)", "lint", "type-check", "build"]
}
```
