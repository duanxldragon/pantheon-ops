# 项目继承说明

English version: [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)

这份文件用于说明 `pantheon-ops` 如何继承 `pantheon-base`，以及本仓库本地允许扩展的范围。

默认规则已经从“跟随 base/main”调整为“消费 base foundation release”。`main` 可以继续承载优化和治理工作，但 ops 默认只升级到显式 release/tag。
`foundation-release.lock.json` 是机器可读的继承锚点；默认 `base-sync` 校验以这份 lock 为准，而不是直接以 `pantheon-base` 当前工作树为准。

## 1. 继承源

- Base repository：当前继承源是 `../pantheon-base`
- Base release line：当前跟随 `release/0.8`
- Base version：当前锁定到 `pantheon-base-v0.8.5`（`185e31f030108ad0e85cf8a6c87912222656ba3a`）
- Inheritance mode：`foundation-release-consumer`

### 1.1 版本号命名规则

自 2026-06-24 起，workspace 版本号统一采用 `pantheon-<product>-v<major>.<minor>.<patch>` 格式，权威定义见 [`../../docs/VERSIONING.md`](../../docs/VERSIONING.md)。

- 历史 tag `base-v0.8.1` / `base-v0.8.2` / `base-v0.8.3` / `base-v0.8.4` 保留为历史记录，不重命名
- 自 `pantheon-base-v0.8.5` 起，本仓库 lock、升级命令、release notes 中出现的版本号都必须用新格式
- 下次 ops 真正升级时，目标 `releaseVersion` 直接写新格式，`foundation-release.lock.json` 的 `releaseVersion` 字段在同一次升级中一次性切到新格式
- 不为切换命名格式单独发 patch；版本号变化必须伴随真实的底座差异

## 2. 继承的底座规则

本仓库继承自 `pantheon-base` 的核心规则包括：

- layer model：`platform / system/auth / system/iam / system/org / system/config / business/*`
- contract-first 文档流
- 共享 backend、frontend、permission、i18n、audit、acceptance 规则
- 共享 shell 和 system-domain UI constraints

## 3. 开工入口

开始编辑本仓库前，先读：

1. `../../docs/WORKSPACE_INHERITANCE.md`
2. `../pantheon-base/DESIGN.md`
3. `../pantheon-base/AGENTS.md`
4. `../pantheon-base/docs/README.md`
5. 对应的 base contracts、designs 和 acceptance docs
6. 本仓库对应 `docs/designs/BUSINESS_*` 和 `docs/acceptances/BUSINESS_*`

本仓库自己的 repo-local workflow skills 位于 `.agents/skills/`，主要用于：

- `repo-verify`：把业务改动或继承改动映射到最小验证矩阵
- `repo-pr-gate`：统一 PR 收口、落点说明和高风险门禁
- `gh-address-comments`：自动处理并收口 GitHub PR、Issue、Discussion 中的可执行评论
- `repo-ci-triage`：把 GitHub Actions 红灯映射回本地复现命令
- `gh-fix-ci`：在 hosted run 级别继续排查 CI 红灯

## 4. 本地业务范围

当前本仓库明确承载的业务域：

- `business/cmdb`
- `business/deploy`
- `business/bizscope`

## 5. 本地允许新增的文档范围

当前显式允许补充的本地业务文档包括：

- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`

未来可以继续增加新的 `business/*` 文档，但不应复制 base 的 platform 或 system-domain contracts。

## 6. Override Policy

- Allowed：业务域补充、业务验收说明、仓库本地执行细节
- Not allowed：重定义 base contracts、base layer ownership、key-first i18n rules、menu/permission split、shared UI hard constraints
- 如果 foundation 规则必须变更，先改 `pantheon-base`，再升级 `pantheon-ops`

## 6.1 Base First 同步机制

涉及平台底座、共享后台壳层、共享前端组件、上传能力、分页能力、通用表格行为时，先改 base，再同步 ops，最后叠加业务模块改动。

这里的“同步”后续默认理解为：

- 选择一个新的 `pantheon-base` foundation release
- 从 GitHub release 或本地 archive 安装该 release artifact
- 升级 ops 所消费的 release 版本
- 再修复 business overlay 与新底座 release 的真实断点

而不是长期直接跟随 `main` 做文件漂移同步。

已按此规则处理的共享项包括：

- `frontend/src/components/table/standardPagination.ts`
- `backend/pkg/upload/service.go`

Current business-domain overrides that stay local to `pantheon-ops`:

- `backend/modules/business/bizscope/*`
- `frontend/src/modules/business/bizscope/*`
- `backend/modules/business/cmdb/*`
- `frontend/src/modules/business/cmdb/*`
- `backend/modules/business/deploy/*`
- `frontend/src/modules/business/deploy/*`

## 6.2 推荐同步流程

同步时固定做四件事：在 `pantheon-base` 完成共享改动；记录共享路径；在 `pantheon-ops` 检查 backend、frontend、generator、i18n、menu 差异；只同步共享部分并重新执行业务模块校验。

当前推荐直接通过 release artifact + release consumer 执行同步，而不是人工整目录覆盖：

- `npm run foundation:install`：按 `foundation-release.lock.json` 下载并安装已锁定的 release artifact
- `npm run foundation:install -- --archive <foundation-release-version>.tgz`：从本地 archive 安装 release artifact
- `npm run upgrade:foundation:apply -- --manifest <bundle-root>\manifest.json --bundle <bundle-root>`
- 如果本地已有 `pantheon-base/releases/<version>/manifest.json`，可以直接让 ops 本地生成 bundle 并消费：`npm run upgrade:foundation:local-plan -- --release-version <version>` 或 `npm run upgrade:foundation:local-apply -- --release-version <version>`
- 该命令会同步共享 backend/frontend、保留 ops 本地 overlay（如 menu registry、generator workspace、frontend generated registry）、把共享 backend import 重写到 `pantheon-ops` 模块名，并补跑 frontend `base-sync` + `menu-contract`
- 日常开发时，`npm run check:base-sync` 只检查当前工作树是否仍符合 `foundation-release.lock.json` 锁定的 foundation release artifact；只有显式执行 `npm run check:base-sync:workspace` 时，才对比 `pantheon-base` 当前工作树，作为“是否需要发起新一轮 upgrade”的预演信号

## 6.3 不建议的同步方式

不要清空 ops 后整体覆盖、不要用 `git reset --hard` 抹平业务差异、不要用整仓拷贝覆盖 `business/*`。更稳妥的方式是共享路径按文件同步，业务路径按模块回流，每次同步后补 `go test` / `tsc` / smoke。

## 6.4 修复落点判定

遇到问题时，先判断改动应落在哪里：

- 属于 `platform`、`system/*`、共享后台壳层、共享分页、共享表格、共享上传、共享 i18n、共享 smoke helper 的，先改 `pantheon-base`
- 属于 `business/cmdb`、`business/deploy`、`business/bizscope` 的，留在 `pantheon-ops`
- 如果只是 ops 页面表现异常，但根因来自共享壳层或共享组件，也应回 base 修
- 如果不确定，先读 base 合同和本文件，再决定，不要凭文件位置直觉下手

## 6.5 同步收口清单

一次 `base -> ops` 同步至少回答清楚下面几件事：

- 这次共享改动对应的 base commit 是什么
- 共享路径哪些已同步，哪些故意未同步
- ops 本地 `business/*` 路径是否保持原样
- 菜单、权限、i18n、测试、smoke、文档是否与共享改动保持一致
- 是否分别验证了 base 和 ops 的最小启动、build 或 smoke
- 是否把剩余漂移显式记录，而不是留给下次会话猜

以下路径属于本地扩展点，不要求与 `pantheon-base` 字节级一致，但仍要保持语义清晰：

- `backend/modules/business/generated_registry.go`：允许注册 ops 本地 `business/*` 模块
- `backend/modules/system/iam/menu/generated_component_registry.go`：允许补充 ops 本地业务页面组件 key
- `backend/modules/system/i18n/builtin_locale_resources.json` 中的 `business.*` 词条：允许保留 ops 本地业务文案

## 6.6 可执行同步命令清单

推荐按下面顺序执行一次 `base -> ops` 同步：

1. 在 `pantheon-base` 完成共享修改并记录 base commit

```powershell
git -C D:\workspace\go\pantheon-platform\pantheon-base rev-parse --short HEAD
```

2. 在 `pantheon-ops` 安装当前锁定的 foundation release artifact

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
npm run foundation:install
```

3. 再跑一键继承校验，先把模板、继承契约和共享 backend/frontend 对齐状态一次性过掉

```powershell
npm run check:inheritance
```

4. 日常业务开发默认只检查“是否仍符合已锁定 release artifact”

```powershell
npm run check:base-sync:backend
npm run check:base-sync:frontend
```

5. 需要判断 base 最近演进是否已经值得同步时，再显式跑 workspace 对比

```powershell
npm run check:base-sync:workspace
```

6. 如果 workspace 对比确认需要升级，优先先在 `pantheon-base` 切新 release，再让 ops 消费该 release，而不是手搓文件覆盖

```powershell
npm run upgrade:foundation:local-plan -- --release-version pantheon-base-v0.8.5
npm run upgrade:foundation:local-apply -- --release-version pantheon-base-v0.8.5
```

7. 如需同步共享后端路径，按文件级方式同步，不覆盖 `business/*`

```powershell
git diff --name-only -- D:\workspace\go\pantheon-platform\pantheon-base\backend
```

8. 完成同步后分别执行最小验证

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-base
go test ./...

Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
go test ./...
npm run check:base-sync:backend
```

9. 如果本轮还涉及前端共享壳层、分页、共享表格或共享 i18n，再补最小前端验证或 smoke

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-ops\frontend
npm run build
```

记录结果时至少写清：

- base commit
- 共享路径哪些已同步
- 哪些路径故意未同步
- `business/*` 是否保持原样
- base/ops 的最小验证结果

常用本地命令：

- `npm run foundation:install`：安装 `foundation-release.lock.json` 锁定的 release artifact
- `npm run check:inheritance`：一键检查 task packet 模板、继承契约、foundation lock、共享 backend 和 frontend 对齐状态
- `npm run check:base-sync`：检查共享 backend + frontend 是否仍符合 `foundation-release.lock.json` 锁定的 release artifact
- `npm run check:base-sync:workspace`：显式检查当前 `pantheon-base` 工作树是否已经偏离 ops 当前锁定 release，作为是否发起 upgrade 的预演信号
- `npm run check:base-sync:backend`：仅检查共享 backend 文件级对齐
- `npm run check:base-sync:frontend`：仅检查共享 frontend 文件级对齐

## 6.7 自动化漂移检测

已配置两层自动化机制，确保 base 演进时不会被遗忘：

### 第一层：每 PR 强制继承契约

`.github/workflows/quality.yml` 中 `docs-governance` 作业现在**始终运行** `check:inheritance`，包含：

- task packet 模板完整性
- 继承契约文档关键标记
- `foundation-release.lock.json` 结构合法性
- 共享 backend 文件级对齐（本地有 base 仓库时）
- 共享 frontend 文件级对齐（本地有 base 仓库时）
- lock 文件新鲜度检查（base 仓库可用时自动计算落后 HEAD 的 commit 数）

CI 环境中 base 仓库不可用时，backend 检查自动跳过而非报错，由第二层兜底。

### 第二层：每周定时漂移检测

`.github/workflows/inheritance-drift-detection.yml`：

- 每周一 UTC 08:57 自动运行
- 同时 checkout `pantheon-base` 和 `pantheon-ops`
- 运行 `check:base-sync:backend:workspace` + `check:base-sync:frontend:workspace`
- 检测到漂移时自动创建带 `inheritance-drift` 标签的 Issue
- 已有未关闭漂移 Issue 时不重复创建
- 也支持手动触发：Actions → Inheritance Drift Detection → Run workflow

### 日常开发中的信号

本地开发时，`frontend` 的 `prebuild` 钩子已经包含 `check:base-sync`。
如果本地有 `pantheon-base` 仓库，文件级差异会在构建前失败并给出明确的 MISSING/DIFF 路径。

如果本地没有 base 仓库（常见于纯 ops 业务开发），建议定期运行：

```powershell
npm run check:base-sync:workspace
```

或关注每周自动创建的漂移 Issue。

## 7. 运行时隔离

- 运行时数据库必须和 `pantheon-base` 隔离
- 推荐默认 DSN 指向 `pantheon_ops`
- 可以共用一个 MySQL 实例，但不能共用同一套数据库 schema

## 8. 使用说明

- `PROJECT_INHERITANCE.md` 现在是中文主入口，和本仓库其他中文主文档保持一致
- 英文协作方请使用 [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)
- 两个版本语义应保持同步；若有差异，以当前仓库实际继承配置为准并及时回补另一语言版本
