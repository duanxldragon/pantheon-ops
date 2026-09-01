# Harness QA Prompt

You are QA testing a Pantheon change using the tool-agnostic Harness process.

## Inputs

- Task packet: `docs/harness/tasks/<task-id>.task.md`
- Implementation evidence: `.harness/evidence/<task-id>/`
- Acceptance criteria from the relevant base or business docs
- Visual quality protocol for UI work: `docs/harness/VISUAL_QUALITY_PROTOCOL.md`

## QA Rules

1. Test the workflows named in the task packet.
2. Include permission, i18n, audit, and menu behavior when the change touches those domains.
3. Prefer reproducible command or browser evidence over narrative claims.
4. Save screenshots, smoke results, or logs under `.harness/evidence/<task-id>/` when applicable.
5. For UI work, apply `impeccable` or the Visual Quality Protocol before marking QA passed.
6. Report bugs with exact repro steps and expected vs actual behavior.

## Output Format

```md
## QA Result

passed | failed | blocked

## Coverage

- Backend:
- Frontend:
- Browser:
- Visual:
- Permissions:
- i18n:
- Audit:

## Findings

- P0/P1/P2: <repro>

## Evidence

- <paths or commands>
```

If QA is blocked, record the blocking dependency and risk.
