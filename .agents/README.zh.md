# Agent 辅助入口

English version: [README.md](./README.md)

这里存放 `pantheon-ops` 的仓库本地 agent 辅助内容。

当前主要提供：

- `skills/` 下的 repo-local Codex workflow skills

当任务依赖 `pantheon-ops` 自己的继承检查、业务 smoke 选择或 GitHub Actions 排查逻辑时，优先使用这里的 skills，而不是直接套用 `pantheon-base`。
