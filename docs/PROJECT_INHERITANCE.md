# 项目继承说明

English version: [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)

这份文件用于说明 `pantheon-ops` 如何继承 `pantheon-base`，以及本仓库本地允许扩展的范围。

## 1. 继承源

- Base repository：当前继承源是 `../pantheon-base`
- Base branch：当前跟随 `main`
- Base version：当前锁定到 `0b06ee4`（`0b06ee40ae2a281bf2a0004343368599a326bc67`）
- Inheritance mode：`foundation-only`，表示 ops 继承底座规则，本地只扩展业务域

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

2. 在 `pantheon-ops` 先跑一键继承校验，先把模板、继承契约和共享 backend 对齐状态一次性过掉

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
npm run check:inheritance
```

3. 如果上一步失败，再单独检查共享 backend 是否仍与 base 对齐，并根据输出决定同步范围

```powershell
npm run check:base-sync:backend
```

4. 如需同步共享后端路径，按文件级方式同步，不覆盖 `business/*`

```powershell
git diff --name-only -- D:\workspace\go\pantheon-platform\pantheon-base\backend
```

5. 完成同步后分别执行最小验证

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-base
go test ./...

Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
go test ./...
npm run check:base-sync:backend
```

6. 如果本轮还涉及前端共享壳层、分页、共享表格或共享 i18n，再补最小前端验证或 smoke

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

- `npm run check:inheritance`：一键检查 task packet 模板、继承契约、共享 backend 对齐状态

## 7. 运行时隔离

- 运行时数据库必须和 `pantheon-base` 隔离
- 推荐默认 DSN 指向 `pantheon_ops`
- 可以共用一个 MySQL 实例，但不能共用同一套数据库 schema

## 8. 使用说明

- `PROJECT_INHERITANCE.md` 现在是中文主入口，和本仓库其他中文主文档保持一致
- 英文协作方请使用 [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)
- 两个版本语义应保持同步；若有差异，以当前仓库实际继承配置为准并及时回补另一语言版本
