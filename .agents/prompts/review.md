# Harness Review Prompt

You are reviewing a Pantheon change using the tool-agnostic Harness process.

## Inputs

- Task packet: `docs/harness/tasks/<task-id>.task.md`
- Verification evidence: `.harness/evidence/<task-id>/`
- Harness contracts: `docs/harness/*`
- Visual quality protocol for UI work: `docs/harness/VISUAL_QUALITY_PROTOCOL.md`
- Tool adapter: `.agents/adapters/<tool>.md`

## Review Rules

1. Read the task packet before reviewing the diff.
2. Check whether the changed files match `Expected Files`.
3. Check whether `Do Not Touch` was respected.
4. Check layer ownership using `platform / system/auth / system/iam / system/org / system/config / business/*`.
5. Read the verification evidence before accepting implementation claims.
6. Treat missing required evidence as a review finding.
7. For UI work, verify that `impeccable` or the Visual Quality Protocol was applied.
8. Treat missing rendered UI evidence, or missing reason for no rendered evidence, as a finding.
9. Keep tool-specific behavior out of the review decision.

## Output Format

Start with findings, ordered by severity.

```md
## Findings

- P0/P1/P2: <file:line> <issue>

## Evidence Checked

- Task packet:
- Evidence directory:
- Commands reviewed:
- Visual evidence:

## Open Questions

- none

## Decision

approved | changes requested | blocked
```

P0/P1 unresolved findings cannot be approved.
