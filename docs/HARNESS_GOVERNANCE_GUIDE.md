# Pantheon-Base Harness Governance 指南（指针）

本文件原是一份与 `docs/harness-pr-generator-guide.md` 内容重叠的 PR/manifest 指南，
两处并存导致同一件事有两个说法，且都引用了已归档的 `sonarcloud-s4666` 模板（DOC-02/GOV-07）。

2026-11-03 起只保留一份：

- **PR body 生成、manifest 结构、evidence 三件套、本地预检** → [harness-pr-generator-guide.md](harness-pr-generator-guide.md)
- **结构权威**：`.github/pull_request_template.md`（PR body section）、`scripts/task-manifest.mjs`（manifest 校验）、`scripts/check-task-packet-template.mjs`（packet section）
- **门禁**：`.github/workflows/quality.yml` 的 `Docs Governance` job

本文件保留仅为不让旧链接失效，不再承载内容。
