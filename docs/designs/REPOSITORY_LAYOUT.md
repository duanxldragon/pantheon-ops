---
title: 仓库目录布局
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-07-15
---

# 仓库目录布局

English version: [REPOSITORY_LAYOUT.en.md](./REPOSITORY_LAYOUT.en.md)

本文定义 `pantheon-base` 根目录的稳定分层。目标是让根目录看起来像工程入口，而不是临时文件堆放区。

## 1. 根目录分组

```text
backend/                  # Go 后端：go.mod、启动入口、领域模块、共享包、数据库迁移、性能压测
frontend/                 # React 前端：应用壳、页面模块、smoke、fixtures 与前端脚本
docs/                     # 当前文档、合同、设计、验收、harness 规范
scripts/                  # 根级自动化、GitHub 协作、harness 检查、release 脚本
tests/                    # 根级 Node 脚本测试、文档测试
.harness/                 # 方法执行运行时治理状态（证据与 manifest 按任务生成）
.agents/                  # repo-local agent 说明、skills、schemas
.codex/                   # Codex 本仓库配置
.github/                  # GitHub workflow、模板、CODEOWNERS、Dependabot
.githooks/                # 本地 git hooks
config/                   # 方法链路配置，当前保留 config/method.config.json
database/                 # Docker Compose 首次初始化 SQL
grafana/                  # Prometheus/Grafana 本地观测配置
openspec/                 # OpenSpec skeleton 与入口说明
schema/generated/         # 生成能力账本等跨端治理输出
```

根目录文件分为四类：

- 入口文档：`README.md`、`DESIGN.md`、`AGENTS.md`、`SECURITY.md`、`CHANGELOG.md`、`VERSION`。
- 构建与依赖清单：`package.json`、`package-lock.json`、`Dockerfile`、`docker-compose.yml`（Go 的 `go.mod`/`go.sum`/`.golangci.yml` 位于 `backend/`）。
- 安全与质量配置：`.gitleaksignore`、`.gitattributes`、`.gitmessage`。
- 本地示例配置：`.env.example`、`.mcp.json`、`SHELL_VERSION.json`。

## 2. 新文件放置规则

1. 业务或系统后端代码只能进入 `backend/modules/` 或 `backend/pkg/`，不要在根目录创建新的业务目录；后端性能压测脚本进入 `backend/tests/performance/`。
2. 前端运行时代码进入 `frontend/src/`；前端脚本进入 `frontend/scripts/`；前端 smoke 与测试 fixtures 进入 `frontend/tests/`；前端单元测试进入 `frontend/tests/unit/`（禁止在 `frontend/src/` 内放测试）；`vitest.config.ts` 是合法的 frontend 根级配置文件（与 vite/playwright 配置同级）。
3. 根级工程脚本进入 `scripts/`；对应测试进入 `tests/scripts/`。
4. harness 方法检查进入 `scripts/harness/`；任务 manifest 和证据进入 `.harness/`。
5. 当前仍有效的架构和治理文档进入 `docs/designs/`、`docs/contracts/`、`docs/acceptances/`；阶段性审计、评估、过程记录不入库，随任务证据留在 `.harness/` 并在任务关闭后清理。
6. 生成 bundle 输出进入 `dist/`，不要提交。
7. 数据库初始化脚本当前保留在 `database/system_init.sql`，因为 `docker-compose.yml` 直接引用该路径。
8. Grafana/Prometheus 本地观测配置当前保留在 `grafana/`，避免和应用运行代码混在一起。

本节规则由机械门禁强制执行：`scripts/harness/check-structure-contract.mjs`（`npm run check:structure`，quality.yml 阻塞步骤）。覆盖：backend/frontend 顶层白名单、`modules/` 域白名单（auth/business/lowcode/platform/system + 前端 generated）、Go 文件 snake_case、`frontend/src` 内禁放测试、共享组件 PascalCase、hooks `use*` 命名、禁止提交二进制。跨层 import 边界由 `check-boundaries.mjs` 负责，两者互补。新增合法目录时先改本文档，再同步门禁白名单。

## 3. 本地噪音目录

这些目录不是仓库结构的一部分，已由 `.gitignore` 排除；需要清爽根目录时可以按需清理，但不要把它们纳入提交：

```text
.claude/
.codegraph/
.husky/
.tmp/
.worktrees/
node_modules/
frontend/node_modules/
frontend/dist/
frontend/test-results/
dist/
uploads/
backend/uploads/
```

其中 `.tmp/` 用于临时日志、下载的 CI 产物、smoke 可执行文件和本地安全扫描输出；`uploads/` 与 `backend/uploads/` 是本地运行时上传数据；`dist/` 是生成的 foundation release bundle 输出。

## 4. 暂不移动的目录

以下目录虽然会增加根目录数量，但目前是稳定入口，不建议为了“少一层目录”而移动：

- `config/`：harness 同步和检查脚本固定读取 `config/method.config.json`。
- `database/`：`docker-compose.yml` 直接挂载 `database/system_init.sql`。
- `.harness/`：任务证据和方法执行记录需要固定位置，便于自动检查。
- `schema/generated/`：跨端能力账本由检查脚本和治理流程直接读取。

如果后续确实要收敛这些目录，必须同步更新脚本、CI、文档引用和相关测试，不能只移动文件。
