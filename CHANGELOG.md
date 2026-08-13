# Changelog

Pantheon Base 方法追踪记录。方法论本体位于 `pantheon-harness`。

格式灵感：[Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)

---

## [pantheon-base-v0.10.11] — 2026-08-11

Security maintenance release for the shared frontend toolchain.

### Changed
- Upgrade `js-yaml` to `3.15.1` to resolve GHSA-5p4m-2wfm-xmqj (CWE-407).
- Upgrade `nanoid` to `3.3.17` to remove the high-severity infinite-loop advisory in the Vite/PostCSS dependency tree.

### Verification
- Frontend `npm audit --audit-level=high` reports zero vulnerabilities.
- Foundation-release, documentation, sync-drift, and lockfile installation checks pass.

---

## [pantheon-base-v0.10.10] — 2026-08-10

Foundation release consumed by `pantheon-ops` for the shared CSRF and smoke contracts.

### Changed
- Distributed the shared request client so the CSRF cookie/header contract is present in foundation-release consumers.
- Corrected the shared smoke registry fixture used by hosted smoke validation.
- Exported the forbidden-request guard required by the shared auth boundary.

### Verification
- CI Summary, Quality, Security, CodeQL, and hosted smoke passed for the release candidate.
- Release manifest and archive checksum are bound to commit `a95e6e52eee8ae9aeb4fd115d18c7c37609290f6`.

---

## [pantheon-base-v0.10.1] — 2026-08-05

发布治理补丁，供 `pantheon-ops` 通过 foundation release 升级消费。

### Changed
- CI Summary 对 required job 采用 fail-closed 聚合；保留全仓 Go Lint 的真实 advisory 结果，并由 `quality.yml` 继续阻断 PR/merge queue 中的新代码 lint。
- Merge queue 使用其不可变 base SHA 限定新代码 lint；发布器在上传前复算并校验 foundation archive 的 SHA-256。
- 补齐历史 task/evidence/review 收口状态和 OpenSpec 仓库骨架。
- 发布资产、候选 SHA、PR 门禁和 consumer upgrade 重新绑定到 v0.10.1。

---

## [pantheon-base-v0.10.0] — 2026-08-04

首个采用 `pantheon-base-vX.Y.Z` 标准命名的 foundation release。

### Changed
- 发布工具统一校验 release tag、release line 和不可变候选 SHA。
- Release Gate、Security、SonarCloud 和 Full Smoke 改为失败关闭。
- 修复 Windows bundle、smoke cleanup、RangePicker 月边界和 Dashboard 嵌套按钮问题。

### Notes
- 完整 release notes、consumer impact、upgrade notes 和验证摘要位于
  [GitHub Release pantheon-base-v0.10.0](https://github.com/duanxldragon/pantheon-base/releases/tag/pantheon-base-v0.10.0)。

---

## [base-v0.9.0] — 2026-07-21

冻结版本，供 `pantheon-ops` 通过 foundation release 继承。

### Changed
- **Go module 重命名**: `pantheon-platform` → `pantheon-base`，覆盖 142 个 `.go` 文件 339 处 import，import 路径永不带 `backend/` 段（go.mod 在 `backend/` 内）
- **代码生成器模板同步**: `backendGenerator.ts` 生成的业务模块 Go import 改为 `pantheon-base/...`（消除 ops 新生成模块的隐形编译炸弹）
- **测试断言与脚本同步**: smoke 断言、`cleanup-generated-modules`、`triage-base-drift` 归一化逻辑同步新 module 名；`cleanup` 失效正则按当前真实形态规范化（去掉旧 `backend/` 段）

### Added
- **business/* 边界门禁（双层）**:
  - `golangci-lint depguard` 规则 `business-boundary`：禁止 `modules/business` import `modules/system|auth|platform` 内部，须走 `pkg/contracts`
  - `check-boundaries.mjs --strict --repo pantheon-base` 接入 CI `boundary-gate` job（新增 `--repo` 参数支持单仓扫描）
- **测试覆盖率门禁**: 新增 `scripts/harness/check-coverage.mjs` + CI `coverage-gate` job，初始阈值取现状实测（12.2%）的 90%（11%），后续版本 ratchet 抬升
- **MFA 生产强制安全基线**: `docs/DEPLOYMENT_GUIDE.md` 新增"强制安全基线（生产必选）"章节，`login.mfa_enabled=1` 等 5 项列为生产必选项

### Notes
- `VERSION` 文件维持 harness 方法论 shell 版本（1.4.0），不随产品线变更；base 产品线版本以 git tag 为载体
- 历史 CodeQL/Dependabot 分析临时文件未纳入本版本

---

## [1.3.0] — 2026-06-27

### Changed
- **Migrated to pantheon-harness**: 所有方法论引用从 `harness-engineering/agentic-method-kit` 迁移到 `pantheon-harness`
- **文档路径更新**: 更新 AGENTS.md、docs/README.md、docs/harness/*.md 中的路径引用
- **脚本路径更新**: 更新 scripts/harness/*.mjs 中的方法包文件路径

### Removed
- **清理 agentic-method-kit/**: 删除 67 个文件，已迁移到 pantheon-harness
- **清理 agentic-repo-shell/**: 删除 92 个文件，已迁移到 pantheon-harness

### Synchronized from pantheon-harness
- ERROR_RECOVERY_STRATEGY.md
- HANDOFF_PROTOCOL.md

---

## [1.2.0] — 2026-06-26

### Changed
- **System modularization**: 系统模块化重构
- **Infrastructure improvements**: 基础设施改进

---

## [1.0.0] — 2026-06-15

### Added
- **Initial harness adoption**: 初始采纳 Harness Engineering 方法论
- **Task packet templates**: 任务包模板
- **Verification evidence specs**: 验证证据规范
- **Multi-agent delivery workflow**: 多 Agent 交付流程
- **Agent execution checklist**: Agent 执行清单
