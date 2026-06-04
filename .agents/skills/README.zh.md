# 仓库本地 Skills

English version: [README.md](./README.md)

这里存放 `pantheon-ops` 的 repo-local Codex skills。

共享模板源头：

- 工作区级的 `harness-engineering/.agents/skills/README.zh.md`

当前提供：

- `repo-verify`
- `repo-pr-gate`
- `repo-ci-triage`
- `gh-fix-ci`

推荐顺序：

1. `repo-verify`
2. `repo-pr-gate`
3. GitHub Actions 红灯时用 `repo-ci-triage`
4. 需要按 hosted run 继续排查时再用 `gh-fix-ci`
