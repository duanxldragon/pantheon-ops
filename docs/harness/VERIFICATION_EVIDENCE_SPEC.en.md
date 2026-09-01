# Verification Evidence Spec

Chinese version: [VERIFICATION_EVIDENCE_SPEC.md](./VERIFICATION_EVIDENCE_SPEC.md)

Type: Contract
Layer: platform
Status: Active

This document defines the format for Pantheon task verification evidence. The evidence format must be tool-agnostic.

## 1. Evidence Directory

Default directory:

```text
.harness/evidence/<task-id>/
  summary.md
  commands.json
  screenshots/
  smoke-results/
  logs/
  review.md
```

Whether `.harness/evidence/` is committed can be chosen by project policy. CI artifacts may use the same structure.

## 2. `summary.md`

```md
# Verification Summary: <task-id>

## Scope

- Primary layer:
- Changed files:

## Commands

| Command | CWD | Result | Notes |
|---|---|---|---|
| `go test ./...` | `pantheon-base` | passed |  |

## Graph Checks

- Used CodeGraph: yes | no
- Affected subgraph:
- Structural checks: `cycle` / `hub` / `call-depth` / `sensitive-flow`
- Findings: none | `<finding>`

## Session Economics

- Response mode: `terse | standard | detailed | none`
- Cost sensitivity: `low | medium | high | none`
- Tokens / cost / retries / delegations: `none | concise summary`
- Notes: `none | provenance or caveat`

## Browser Evidence

- none

## Known Gaps

- none

## Completion Status

complete | blocked | partial
```

## 3. `commands.json`

```json
{
  "taskId": "YYYY-MM-DD-task-name",
  "repo": "pantheon-platform",
  "agent": {
    "tool": "agent",
    "adapter": "repository-defined adapter"
  },
  "commands": [
    {
      "command": "go test ./backend/...",
      "cwd": "pantheon-base",
      "status": "passed",
      "durationMs": 0,
      "notes": ""
    }
  ],
  "graphChecks": {
    "usedCodeGraph": true,
    "affectedSubgraph": [
      "route -> handler -> service -> repo"
    ],
    "checks": ["cycle", "hub", "call-depth", "sensitive-flow"],
    "findings": [],
    "notes": ""
  },
  "sessionEconomics": {
    "responseMode": "terse",
    "costSensitivity": "medium",
    "inputTokens": 12000,
    "outputTokens": 3400,
    "cacheReadTokens": 9000,
    "cacheWriteTokens": 800,
    "estimatedCostUsd": 1.42,
    "retryCount": 1,
    "delegationCount": 0,
    "notes": "derived from agent session record"
  },
  "browserEvidence": [],
  "runtimeSensitive": true,
  "runtimeLogs": ["logs/auth-smoke.log"],
  "runtimeMetrics": ["p95=120ms"],
  "runtimeGap": "",
  "linkage": {
    "taskManifest": ".harness/tasks/YYYY-MM-DD-task-name/manifest.json",
    "evidenceDir": ".harness/evidence/YYYY-MM-DD-task-name/",
    "reviewFile": ".harness/evidence/YYYY-MM-DD-task-name/review.md",
    "changeRef": "openspec/changes/<name>/",
    "planRefs": ["docs/superpowers/plans/<file>.md"]
  },
  "knownGaps": [],
  "completedAt": "YYYY-MM-DDTHH:mm:ssZ"
}
```

`agent.tool` may be:

- `agent`
- `human`
- `other`

Specific tool examples include `codex`, `claude-code`, `cursor`, `github-copilot`, `openhands`, `aider`.

When a task includes browser verification for UI, routes, or permission states, `browserEvidence` should include at least:

```json
{
  "viewport": "desktop",
  "url": "/system/user",
  "screenshot": "screenshots/user-desktop.png",
  "consoleErrors": [],
  "checkedStates": ["loading", "empty", "error", "permission"]
}
```

If the current environment cannot produce screenshots, a single record may use `visualGap` instead of `screenshot`, or the global reason may be added to `knownGaps`.

For runtime-sensitive work, `commands.json` should also prefer recording:

```json
{
  "runtimeSensitive": true,
  "runtimeLogs": ["logs/auth-smoke.log"],
  "runtimeMetrics": ["p95=120ms"],
  "runtimeTraces": ["trace/auth-login-01"],
  "runtimePerformance": ["login p95 < 150ms"],
  "runtimeGap": "staging trace export unavailable"
}
```

Where:

- `runtimeSensitive`: explicitly marks runtime risk
- `runtimeLogs` / `runtimeMetrics` / `runtimeTraces` / `runtimePerformance`: runtime signals
- `runtimeGap`: an explicit explanation when signals cannot be collected in the current environment

For long-running, delegated, or cost-sensitive work, `commands.json` should also prefer recording:

```json
{
  "sessionEconomics": {
    "responseMode": "terse",
    "costSensitivity": "high",
    "inputTokens": 12000,
    "outputTokens": 3400,
    "cacheReadTokens": 9000,
    "cacheWriteTokens": 800,
    "estimatedCostUsd": 1.42,
    "retryCount": 1,
    "delegationCount": 2,
    "notes": "derived from agent session record"
  }
}
```

Where:

- `responseMode`: the default narration budget for the session
- `costSensitivity`: whether token/cost should be treated as an explicit throughput constraint
- `inputTokens` / `outputTokens` / `cacheReadTokens` / `cacheWriteTokens` / `estimatedCostUsd`: session-economics signals to record when available
- `retryCount` / `delegationCount`: lightweight indicators that retries or subagent usage may be driving cost
- `notes`: provenance, estimation method, or an explicit caveat

## 4.2 Artifact Linkage

`commands.json` should record artifact linkage explicitly:

- `linkage.taskManifest`
- `linkage.evidenceDir`
- `linkage.reviewFile`
- `linkage.changeRef`
- `linkage.planRefs`

For structural, high-risk, or cross-layer changes, `commands.json` should also record `graphChecks` when practical. The goal is to show which affected subgraph was reviewed and what structural conclusions were reached, not to emit a full-repo graph report.

Rules:

- `taskId` must match the task id encoded in `linkage.taskManifest` and the `linkage.evidenceDir` directory name
- if `reviewFile` exists, it should live under the matching evidence directory
- if there is no OpenSpec change, `changeRef` must be `none`
- if there is no superpowers plan, `planRefs` may be an empty array

If the repository also keeps a human-readable task packet, keep it paired to the same task id; the machine-readable evidence closure keys off `taskManifest`.

## 4. UI Evidence

When UI, routes, permission states, i18n, or browser interaction are affected, the task must save or reference:

- final URL
- console errors
- screenshot
- smoke result
- viewport information

Local workflows may use gstack browse, Playwright, browser extensions, or manual screenshots, but the evidence landing structure must remain consistent.

## 4.1 Minimum UI Evidence Fields

For UI-affecting tasks, evidence must include:

- viewport label (`desktop`, `mobile`, or equivalent)
- screenshot path or explicit visual gap record
- final URL or route
- console error result
- checked states: `loading`, `empty`, `error`, `permission`, if relevant

## 5. Verification Not Run

If verification was not run, the reason must be recorded:

```md
## Not Run

| Command | Reason | Risk |
|---|---|---|
| `npm run build` | dependencies not installed | frontend build regressions not ruled out |
```

“Not enough time” or “should be fine” are not valid verification exemptions.

For runtime-sensitive work, “tests passed” is also insufficient if there is neither a runtime signal nor an explicit runtime gap.

For long-running, delegated, or cost-sensitive work, “done” is also insufficient if there is neither `sessionEconomics` nor an explicit note that the current tool cannot expose those signals.

This sync is docs-only. `pantheon-base` has not yet upgraded its local evidence checker for these new fields, so `sessionEconomics` is a method recommendation today rather than a mechanical hard requirement.
