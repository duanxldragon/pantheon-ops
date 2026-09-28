---
title: 仓库布局、命名与分层边界规范
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-09-23
---

# 仓库布局、命名与分层边界规范

English version: [REPOSITORY_LAYOUT.en.md](./REPOSITORY_LAYOUT.en.md)

本文是 `pantheon-base` 目录布局、文件命名、受控例外与分层依赖方向的**唯一 canonical 规范**。任何新增目录、新增文件后缀、生成规则调整或跨层依赖，都必须先改本文，再同步门禁白名单、CI 步骤、文档引用与测试。

`scripts/harness/check-structure-contract.mjs` 的契约源就是本文 §2 与 §5；`scripts/harness/check-boundaries.mjs` 负责 §8 的 import 方向。两者职责互补，改任一侧都要同步另一侧。

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

## 2. 新文件放置规则与目录白名单

### 2.1 放置规则

1. 业务或系统后端代码只能进入 `backend/modules/` 或 `backend/pkg/`，不要在根目录创建新的业务目录；后端性能压测脚本进入 `backend/tests/performance/`。
2. 前端运行时代码进入 `frontend/src/`；前端脚本进入 `frontend/scripts/`；前端 smoke 与测试 fixtures 进入 `frontend/tests/`；前端单元测试进入 `frontend/tests/unit/`（禁止在 `frontend/src/` 内放测试）；`vitest.config.ts` 是合法的 frontend 根级配置文件（与 vite/playwright 配置同级）；`frontend/src/index.ts` 是 library build 的公共导出入口。
3. 根级工程脚本进入 `scripts/`；对应测试进入 `tests/scripts/`。
4. harness 方法检查进入 `scripts/harness/`；任务 manifest 和证据进入 `.harness/`。
5. 当前仍有效的架构和治理文档进入 `docs/designs/`、`docs/contracts/`、`docs/acceptances/`；阶段性审计、评估、过程记录不入库，随任务证据留在 `.harness/` 并在任务关闭后清理。
6. 生成 bundle 输出进入 `dist/`，不要提交。
7. 数据库初始化脚本当前保留在 `database/system_init.sql`，因为 `docker-compose.yml` 直接引用该路径。
8. Grafana/Prometheus 本地观测配置当前保留在 `grafana/`，避免和应用运行代码混在一起。

### 2.2 目录白名单（机械门禁的判定依据）

| 范围 | 允许的目录 | 允许的文件 |
| --- | --- | --- |
| `backend/` 顶层 | `cmd`、`internal`、`modules`、`pkg`、`tests` | `go.mod`、`go.sum`、`start-dev.sh`、`start-dev.bat`、`DEV_DB_INIT_GUIDE.md`、`.golangci.yml` |
| `backend/modules/` 域 | `auth`、`business`、`lowcode`、`platform`、`system` | 见 §5.1 |
| `frontend/` 顶层 | `config`、`public`、`scripts`、`src`、`tests` | `.eslintrc.json`/`.gitignore`/`.prettierignore`/`.prettierrc`、`AGENTS*.md`、`README*.md`、`eslint.config.js`、`index.html`、`package(-lock).json`、`playwright*.config.ts`、`tsconfig*.json`、`vite.config.ts`、`vitest.config.ts` |
| `frontend/src/` 顶层 | `api`、`assets`、`components`、`core`、`hooks`、`i18n`、`modules`、`store` | `App.tsx`、`index.css`、`index.ts`、`main.tsx`、`vite-env.d.ts` |
| `frontend/src/modules/` 域 | `auth`、`business`、`generated`、`lowcode`、`platform`、`system` | 见 §5.2 |

白名单之外的新目录默认是违规。确实需要时按 §10 的流程先改本文档。

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

## 5. 文件命名词典

本节的词汇表是文件后缀的 canonical 来源。门禁只强制其中一部分（见 §10），但新增文件应遵循完整词汇表，以保持目录可预测。

### 5.1 后端 Go

- 文件名一律 snake_case，正则 `^[a-z0-9_]+\.go$`：禁止 PascalCase、连字符、空格。
- 测试文件 `*_test.go` 与被测文件同包同目录；纯函数回归测试用 `_pure_test.go` 与 DB-backed 测试区分。
- 职责后缀词汇表：

| 后缀 | 职责 | 示例 |
| --- | --- | --- |
| `_handler.go` | HTTP / gin 路由处理 | `user_handler.go` |
| `_service.go` | 领域服务与业务编排 | `role_service.go` |
| `_repository.go` | 持久化访问 | `session_repository.go` |
| `_model.go` | 持久化模型 / 实体 | `session_model.go` |
| `_dto.go` | 请求 / 响应数据传输对象 | `dashboard_dto.go` |
| `_registry.go` | 注册表（含生成注册表） | `generated_registry.go` |
| `_seed.go` | 种子 / fixture 初始化 | `setting_seed.go` |
| `_export.go` | 导入导出与报表写出 | `i18n_export.go` |
| `_module.go` | 模块装配入口 | `module.go` |
| `_helpers.go` / `_utils.go` | 无状态辅助函数 | `session_helpers.go` |
| `_test.go` | 测试 | `health_test.go` |

- 包级入口文件可用包名本身（`system.go`、`lowcode.go`、`business.go`），但不要借此绕开职责后缀。
- 新增职责后缀时，先在本表登记再使用；同义后缀（如 `_repo.go` 与 `_repository.go`）只保留一个，避免并行词汇。

### 5.2 前端 TypeScript / TSX

| 类别 | 命名 | 位置 | 示例 |
| --- | --- | --- | --- |
| 组件 | PascalCase `*.tsx` | `frontend/src/components/<group>/`、`frontend/src/modules/**/components/` | `PageEmpty.tsx` |
| Hook | `use<PascalCase>.ts(x)` | `frontend/src/hooks/` | `usePermission.ts` |
| 非组件模块 | camelCase `*.ts` | 与消费者同目录 | `tablePreferences.ts` |
| 测试 | `*.test.ts(x)` | `frontend/tests/`（禁止在 `frontend/src/`） | `router.test.ts` |
| API | camelCase `*.ts` | `frontend/src/api/` | `file.ts` |
| i18n 资源 | locale 命名 | `frontend/src/i18n/resources/` | `zh-CN.ts` |
| 生成注册表 | 固定名 | `frontend/src/modules/generated/`、`frontend/src/core/router/` | `business.ts`、`generatedComponentRegistry.ts` |
| 生成模块 | 业务模块名 | `frontend/src/modules/business/<module>/` | 生成器产物 |
| 组件样式 | 与被修饰组件同名（`ComponentName.css`），与组件同目录 | 组件目录 | `TimeRangeFilter.css` |
| 分组 / 模块级共享样式 | 以**所在目录名**命名（`<dir>.css`），或放在显式 `shared/` 子目录 | 分组目录 | `auth.css`、`components/shared/list-page.css` |
| 跨模块 / 全局共享样式 | 集中在 `frontend/src/assets/` 或全局 `index.css` | shared | `index.css` |

样式分三层：**组件专属**样式文件名必须等于组件名（不是 BEM 块名、也不是目录名）；
**分组/模块级**共享样式以目录名命名或在 `shared/` 下；**跨模块**共享样式必须有显式共享位置。
BEM 类名（`.time-range-filter__x`）不随文件名变化。

`index.ts` 仅用于 barrel 再导出，不放实现。

### 5.3 文档命名

- `docs/` 下的长期文档用 `UPPER_SNAKE_CASE.md`，英文 companion 为 `<NAME>.en.md`，且 `.md` 与 `.en.md` 必须成对维护。
- 阶段性材料只允许以 `<YYYY-MM-DD>-<kebab-case>.md` 留在 `.harness/`（任务文档、证据、review）；不要把它们提升到 `docs/` 一线入口。
- 所有 `docs/`、`architecture/`、`patterns/` 下的 Markdown 必须带 YAML frontmatter，字段集见 §6。

## 6. 文档分类与目录归属

文档类型遵循 [文档合同化治理方案](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md) 的五类模型；frontmatter 字段遵循 [DOCUMENT_FRONTMATTER_SCHEMA.md](../contracts/DOCUMENT_FRONTMATTER_SCHEMA.md)。

| 类型 | 作用 | 目录 | 生命周期 |
| --- | --- | --- | --- |
| `Contract` | 定义边界、目标、非目标、完成定义 | `docs/contracts/` | 长期，进主索引 |
| `Design` | 说明如何设计（须隶属某份 Contract） | `docs/designs/` | 长期，进主索引 |
| `Assessment` | 说明现状与合同的差距 | `docs/reviews/`（保留时）或 `.harness/evidence/` | 中等；无复用价值则删除 |
| `Remediation` | 说明整改方案 | `.harness/tasks/`（任务形式）或 `docs/` 次级入口 | 中等 |
| `Acceptance` | 说明是否达到合同要求 | `docs/acceptances/` | 模板/基线长期；一次性样例可归档 |

规则：

- `Design / Assessment / Remediation / Acceptance` 必须带非空 `linked_contracts`，指向真实存在的合同路径。
- 退场只有三种去向：被正式 Contract/Design/Acceptance 吸收后删除、因样例/基线/升级价值进入 `docs/archive/*`、或因仍在治理主链被明确引用而保留。不存在“先留着以后再说”的默认去向。
- `docs/README.md` 的主入口只能链接 `Active` 文档；目录级索引由各目录 `README.md` 负责。

## 7. 生成文件规则

### 7.1 生成源与生成产物

- 生成能力真相源：`system_module_registration` + `schema/generated/**.json`；派生快照为 `schema/generated/feature-ledger.json`（见 [DESIGN.md §2.4](../../DESIGN.md)）。
- 生成产物（**不得手工编辑**）：
  - 后端：`backend/modules/business/generated_registry.go`、`backend/modules/system/iam/menu/generated_component_registry.go`、`backend/modules/business/<module>/**`。
  - 前端：`frontend/src/modules/generated/*.ts`、`frontend/src/core/router/generatedComponentRegistry.ts`、`frontend/src/modules/business/<module>/**`、`frontend/src/i18n/resources/generated/**`。
  - 治理账本：`schema/generated/feature-ledger.json`。
- 生成产物的重置模板唯一来源是 `frontend/scripts/cleanup-generated-modules.mjs` 的 `REGISTRY_TEMPLATES`；不要在别处复制一份空注册表。
- `business/*` 目录内除上述生成路径外的文件属于手工维护层（模块装配、retired 登记等），改它们不需要重新生成。

### 7.2 可识别性与漂移检查

- **生成标记（已强制）**：生成文本产物首行必须为 `// Code generated by pantheon low-code generator. DO NOT EDIT.`；JSON 产物（`schema/generated/*.json`）无法注释，按路径与可解析性识别。标记字符串的唯一来源是 `frontend/scripts/cleanup-generated-modules.mjs` 的 `GENERATED_MARKER`。
- **漂移检查（已接入）**：`npm run check:generated`（`scripts/harness/check-generated.mjs`，quality.yml **阻塞**）校验生成产物的标记与存在性：11 个稳定产物按名单校验，此外动态发现 `backend|frontend/**/modules/business/<module>/` 下的生成模块文件（`.go/.ts/.tsx` 首行标记）与 `schema/generated/<scope>/<name>.json`（路径 + 可解析性）；`npm run check:generated-modules` 继续检测残留 smoke 生成模块。旧的 `check-arch-boundaries.mjs` 的 `checkGeneratedRegistry` 仍是未接入 CI 的重叠实现（见 §10）。
- **别名分歧**：`scripts/check-arch-boundaries.mjs` 是历史脚本，与 `scripts/harness/check-boundaries.mjs` 职责重叠；收敛前只以 `scripts/harness/check-boundaries.mjs` 为规范门禁。

## 8. 分层依赖矩阵

逻辑层与物理目录的映射见 [DESIGN.md 分层与模块边界](../../DESIGN.md)；本节只定义允许的依赖方向，这是 `scripts/harness/check-boundaries.mjs` 与 `scripts/check-arch-boundaries.mjs` 的判定依据。

### 8.1 层与物理位置

| 逻辑层 | 后端目录 | 前端目录 |
| --- | --- | --- |
| 平台壳层 `platform` | `backend/modules/platform`、`backend/internal`、`backend/pkg` | `frontend/src/modules/platform`、`frontend/src/core` |
| 系统底座 `system/auth` | `backend/modules/auth`、`backend/modules/system/{iam,org,config,audit,i18n}` | `frontend/src/modules/auth`、`frontend/src/modules/system/*` |
| 业务领域 `business/*` | `backend/modules/business/*` | `frontend/src/modules/business/*` |
| 公共契约 | `backend/pkg/contracts`、`backend/pkg/common`、`backend/pkg/database` | `frontend/src/api`、`frontend/src/components`、`frontend/src/hooks` |

`lowcode` 是生成/动态能力的独立工作域，可被 `platform` 与 `business` 通过其公开契约调用；`generated` 是前端生成注册表目录，不属于任何手写依赖图的源。

### 8.2 允许的依赖方向

| 依赖方 ↓ / 被依赖方 → | platform | auth | system/* | business/* | pkg/contracts |
| --- | --- | --- | --- | --- | --- |
| `platform` | ✅ 同层 | ⚠️ 仅 public contract | ⚠️ 仅 public contract / 读模型 | ❌ | ✅ |
| `auth` | ❌ | ✅ 同域 | ✅ 系统底座内部，优先 public contract | ❌ | ✅ |
| `system/*` | ❌ | ❌ | ✅ 同子域；跨子域仅 public contract | ❌ | ✅ |
| `business/*` | ❌ | ❌ | ❌ | ✅ 同模块；跨模块仅 public contract | ✅ |

- ✅ 允许；⚠️ 允许但只经 public contract / adapter / 读模型，不得 import 对方 Service / Repository / Handler；❌ 禁止。
- 前端遵循同一矩阵：`business` 不得 import `modules/system|auth|platform` 内部；`system` 子域之间只经 `frontend/src/api` 或公开组件契约。

### 8.3 违规基线与收口

- 门禁 `scripts/harness/check-boundaries.mjs` 现已覆盖 `business`、`platform`、`auth` 三类生产代码；`_test.go` 与 `*.test.*` / `*.spec.*` 显式豁免（测试可跨域装配，生产代码不可）。
- `platform -> system/org/dept` 已按 §8.2 修复：adapter 移入组合根 `backend/cmd/server/platform_org_governance.go`，platform 模块不再直接 import system 实现。
- 其余已知违规冻结在 `config/boundary-baseline.json`（逐条带 `reason` 与 `reviewBy=2026-12-31`）。CI 以 `--baseline` 放行已知项、阻断新增项：
  - `frontend/src/modules/platform/widgets.tsx` 对 `system/menu/api` 的**类型**依赖；
  - `auth/login`、`auth/login/login_runtime.go`、`auth/security/security_service.go` 对 `system/iam/user` 的直接依赖；
  - 4 个 auth 页面 import `system/components/shared/list-page.css`。
- 基线是**有期限的受控例外**，到期必须重审或修复；不得把新违规写进基线来消音。
- 测试文件与生成文件的跨域依赖单独处理，豁免必须有注释和证据，不能与生产代码混为一谈。

## 9. 受控例外

以下物理布局是既有例外，**只允许存在，不允许扩散**。新增同类例外必须先在本文登记并说明 owner 与复核条件。

| 例外 | 属性 | Owner 层 | 理由 | 复核 / 到期 |
| --- | --- | --- | --- | --- |
| `backend/modules/auth`、`frontend/src/modules/auth` | 顶层物理模块 | system/auth | `auth` 逻辑上属系统底座，但登录 / 会话 / MFA / SSO 生命周期独立，物理独立可避免 `system` 变成杂物间 | 每次 auth 合同变更时复核 |
| `backend/modules/lowcode`、`frontend/src/modules/lowcode` | 顶层物理模块 | platform（lowcode 工作域） | 生成器与动态模块治理是独立工作域，不被任何业务域私有 | lowcode 设计变更时复核 |
| `frontend/src/modules/generated` | 生成目录 | platform | 生成注册表需要稳定固定路径，按域白名单纳入 | 生成治理任务复核 |
| `backend/tests/`、`frontend/tests/` | 顶层测试目录 | platform | 测试与运行时代码物理分离，便于白名单与门禁 | 结构门禁变更时复核 |
| `config/`、`database/`、`grafana/`、`openspec/`、`schema/generated/` | 根级稳定入口 | platform | 见 §4，被脚本 / compose / 治理流程直接引用 | 引用方变更时复核 |

例外不是“白名单扩容”。把违规目录塞进例外表而不修复，等同掩盖问题。

## 10. 机械门禁与更新点

### 10.1 规范 → 门禁映射

| 规则区域 | 强制器 | 命令 | CI |
| --- | --- | --- | --- |
| §2 放置规则 + §5 命名 | `scripts/harness/check-structure-contract.mjs` | `npm run check:structure` | `quality.yml`（阻塞） |
| §8.2 `business` / `platform` / `auth` 跨层 import | `scripts/harness/check-boundaries.mjs` | `node scripts/harness/check-boundaries.mjs --strict --repo pantheon-base --baseline config/boundary-baseline.json` | `ci.yml`（阻塞；已知欠债见 §8.3 基线） |
| §6 文档 frontmatter / 类型 / 合同关联 | `scripts/frontmatter-check.mjs`、`scripts/harness/check-doc-frontmatter.mjs` | `npm run check:docs-frontmatter` | `quality.yml` |
| §6 文档内部链接 | `scripts/harness/check-doc-links.mjs` | `npm run check:harness-docs` | harness 检查 |
| §7 生成标记与产物存在性 | `scripts/harness/check-generated.mjs` | `npm run check:generated` | `quality.yml`（阻塞） |
| §7 生成模块漂移 | `frontend/scripts/cleanup-generated-modules.mjs` | `npm run check:generated-modules` | `quality.yml` |
| §7 生成注册表存在性与非空（旧） | `scripts/check-arch-boundaries.mjs`（`checkGeneratedRegistry`） | 未接入 npm/CI | **gap** |

### 10.2 变更流程（先规范，后门禁）

新增或调整目录、文件后缀、例外或依赖方向时，按此顺序：

1. 更新本文档对应章节（§2 / §5 / §8 / §9）。
2. 同步门禁白名单或规则：`check-structure-contract.mjs`、`check-boundaries.mjs`、`check-arch-boundaries.mjs`。
3. 同步 CI 步骤与 npm script。
4. 同步 `docs/designs/README.md`、`docs/README.md` 等索引和所有引用。
5. 补或更新对应测试（`tests/scripts/**`），并运行 §10.1 的验证命令。

只改文档不改门禁，或只扩白名单不修根因，都不算完成。
