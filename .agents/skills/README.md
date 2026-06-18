# Repo-Local Skills

Chinese version: [README.zh.md](./README.zh.md)

Pantheon Ops keeps repository-local Codex skills here.

Shared template source:

- `harness-engineering/.agents/skills/README.md` at the workspace level

Available skills:

- `repo-verify`
- `repo-pr-gate`
- `gh-address-comments`
- `repo-ci-triage`
- `gh-fix-ci`

Standard command entrypoints:

- `npm run check:github-feedback -- --repo <owner/repo> --pr <number>`: fail fast when the PR or linked issue/discussion feedback is not fully closed yet

Recommended order:

1. `repo-verify`
2. `repo-pr-gate`
3. `gh-address-comments` when open GitHub feedback needs action
4. `repo-ci-triage` when GitHub Actions is red
5. `gh-fix-ci` when the hosted run still needs GitHub-level investigation
