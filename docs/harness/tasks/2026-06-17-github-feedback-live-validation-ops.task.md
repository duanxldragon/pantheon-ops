# Task Packet: 2026-06-17-github-feedback-live-validation-ops

## Goal

Validate the GitHub feedback automation against live Pantheon Ops GitHub resources so PR review comments and linked issue comments can be closed without manual triage.

## Scope

### In

- Exercise the live GitHub feedback fetch, plan, and writeback flow for Pantheon Ops.
- Verify the automation derives context from repo-local task and evidence artifacts.
- Keep the repo-local PR automation gate aligned with the feedback closure workflow.

### Out

- GitHub Discussions live validation, because this repository does not have Discussions enabled.
- Any base foundation sync or business runtime behavior changes.

## Implementation Notes

- This patch stays in `pantheon-ops` because it is repository governance and local PR closure logic, not shared platform runtime behavior.
- The live validation uses temporary draft PRs, linked issues, and repository-local evidence instead of hand-written context payloads.
