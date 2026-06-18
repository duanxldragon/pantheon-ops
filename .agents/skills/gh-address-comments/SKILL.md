---
name: gh-address-comments
description: Use when Pantheon Ops has actionable GitHub PR review comments, issue comments, or discussion comments and the agent should fix, verify, and respond automatically with gh CLI
---

# GH Address Comments

Use this skill when GitHub feedback exists on the current branch PR, or on issues/discussions explicitly linked to the change, and the agent should handle it end to end instead of waiting for per-comment approval.

## Scope Resolution

Prefer targets in this order:

1. the open PR for the current branch
2. issues linked from that PR, task packet, or user-provided URLs
3. discussions linked from that PR or user-provided URLs

Do not sweep unrelated repository comments.

## Auth And Access

- Run `gh auth status` first.
- Run `gh` commands with elevated network access.
- If auth fails or required scopes are missing, stop and ask for `gh auth login`.

## Minimal Loop

1. Gather and classify GitHub feedback with `node scripts/fetch-github-feedback.mjs --repo <owner/repo> --pr <number> --json`.
   - If `--repo` or `--pr` is omitted, the script resolves the current repo and current-branch PR through `gh`.
   - For offline debugging or fixture replay, use `node scripts/fetch-github-feedback.mjs --input <snapshot.json> --json`.
   - Use `node scripts/fetch-github-feedback.mjs --repo <owner/repo> --pr <number> --check` or `npm run check:github-feedback -- --repo <owner/repo> --pr <number>` to block PR closure while non-closed feedback remains.
2. For unattended execution, derive the writeback plan with `node scripts/run-github-feedback-loop.mjs --snapshot <snapshot.json> --context <context.json> --dry-run --json`.
   - `context.json` supplies the verified change summary, verification lines, ownership explanation, and out-of-scope rationale that the loop may quote back.
3. Materialize the writeback plan in JSON, then apply it with `node scripts/address-github-feedback.mjs --snapshot <snapshot.json> --plan <plan.json>`.
   - Use `--dry-run --json` first when you need to inspect the exact gh operations before execution.
2. Classify each item:
   - actionable code, docs, config, or test change
   - explanation only
   - already satisfied by the current diff
   - out of scope or conflicting
3. Apply the smallest real fix or produce the smallest defensible no-change explanation.
4. Run the local proof commands.
5. Only after verification, reply and close:
   - reply to the comment with the fix summary or rationale
   - resolve PR review threads
   - mark discussion replies as answered when the thread is actually complete
   - close issues only when the patch fully handles the issue and no further tracking value remains
6. Re-run `repo-pr-gate` before final PR closure.

## Automation Rules

- Default to acting without comment-by-comment approval.
- Escalate only when feedback would:
  - change product scope or acceptance criteria
  - require security-risk acceptance
  - conflict with repository contracts or landing-side ownership
  - depend on runtime or production evidence that is unavailable
- If two comments request incompatible outcomes, stop and surface the conflict.

## Pantheon Ops Rules

- Run the landing-side check first when comments touch shared platform, `system/*`, shared shell, menu, permission, i18n, upload, audit, or other inherited paths. If the real fix belongs in `pantheon-base`, do not hide it with an ops-only workaround.
- Run `npm run check:inheritance` when feedback may reflect shared-path drift, release-upgrade fallout, or inherited workflow behavior.
- If a comment is really about hosted CI behavior, use `repo-ci-triage` first and `gh-fix-ci` only when local reproduction is green or inconclusive.
- For business-only comments, keep the fix local to `business/*` and verify the smallest affected business surface.

## Final Report

- target PR, issue, and discussion identifiers
- comments handled
- commands run
- comments or threads closed automatically
- remaining escalations or nonlocal risk
