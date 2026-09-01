# Harness Implementation Prompt

You are implementing a repository task using the tool-agnostic Harness process.

## Required Inputs

- Task manifest: `.harness/tasks/<task-id>/manifest.json`
- Task packet: `docs/harness/tasks/<task-id>.task.md`
- Harness contract: `docs/harness/HARNESS_ENGINEERING_CONTRACT.md`
- Agent interface contract: `docs/harness/AGENT_INTERFACE_CONTRACT.md`
- Visual quality protocol for UI work: `docs/harness/VISUAL_QUALITY_PROTOCOL.md`
- Relevant tool adapter under `.agents/adapters/`

## Execution Rules

1. Declare the primary layer before editing.
2. Read contract anchors listed in the task manifest or task packet.
3. Keep changes inside `Scope / In`.
4. Do not touch paths listed in `Expected Files / Do Not Touch`.
5. Prefer shared scripts under `scripts/harness/` over tool-specific checks.
6. For UI work, use `impeccable` first when available; otherwise follow `VISUAL_QUALITY_PROTOCOL.md`.
7. Record verification results under `.harness/evidence/<task-id>/`.
8. If verification is not run, record the reason, risk, and follow-up.

## Completion Output

```md
## Summary

- <changed behavior or artifact>

## Verification

- `<command>`: passed | failed | not-run

## Evidence

- `.harness/evidence/<task-id>/summary.md`
- `.harness/evidence/<task-id>/commands.json`
- UI work: screenshots or visual-review notes when applicable

## Known Gaps

- none
```

Do not claim completion without fresh verification evidence.
