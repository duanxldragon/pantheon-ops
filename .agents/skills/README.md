# Repo-Local Skills

Chinese version: [README.zh.md](./README.zh.md)

Pantheon Ops keeps repository-local Codex skills here.

Shared template source:

- `harness-engineering/.agents/skills/README.md` at the workspace level

Available skills:

- `repo-verify`
- `repo-pr-gate`
- `repo-ci-triage`
- `gh-fix-ci`

Recommended order:

1. `repo-verify`
2. `repo-pr-gate`
3. `repo-ci-triage` when GitHub Actions is red
4. `gh-fix-ci` when the hosted run still needs GitHub-level investigation
