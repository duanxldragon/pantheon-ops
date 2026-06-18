# 仓库本地 Skills

English version: [README.md](./README.md)

这里存放 `pantheon-ops` 的 repo-local Codex skills。

共享模板源头：

- 工作区级的 `harness-engineering/.agents/skills/README.zh.md`

当前提供：

- `repo-verify`
- `repo-pr-gate`
- `gh-address-comments`
- `repo-ci-triage`
- `gh-fix-ci`

标准命令入口：

- `npm run check:github-feedback -- --repo <owner/repo> --pr <number>`：当 PR 或其关联 issue/discussion 仍有未收口反馈时直接失败

推荐顺序：

1. `repo-verify`
2. `repo-pr-gate`
3. 存在待处理 GitHub 评论时用 `gh-address-comments`
4. GitHub Actions 红灯时用 `repo-ci-triage`
5. 需要按 hosted run 继续排查时再用 `gh-fix-ci`
