# 项目继承说明

English version: [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)

`pantheon-ops` 采用“完整 Pantheon Base 快照 + 声明式 business overlay”的继承模式。升级时从选定的 Base commit 重建完整产品树，再注入 Ops 业务资产；不在旧树上逐文件同步，也不维护共享源码白名单。

## 1. 所有权

- `pantheon-base`：拥有 `platform`、`system/*`、共享前后端、生成器和产品级工程配置。
- `pantheon-ops`：拥有 `business/bizscope`、`business/cmdb`、`business/deploy`、业务 smoke、业务设计和仓库身份。
- `business-overlay.json`：Ops 唯一机器可读资产清单和装配清单。
- `.business-overlay-report.json`：每次重建输出的 Base commit、Ops 文件哈希、生成文件和 import 重写记录。

重建结果保留 Base 的目录布局和 Go module `pantheon-base`。业务源码在注入时将历史 import `pantheon-ops/backend/...` 改写为 `pantheon-base/...`，共享源码不再改写 module identity。

## 2. 装配扩展点

业务模块通过 Ops 重建器生成的静态 overlay 注册表接入：

- `backend/modules/business/business_overlay_registry.go`
- `frontend/src/modules/businessOverlay.ts`
- `frontend/src/core/router/businessOverlayComponentRegistry.ts`
- `backend/modules/system/iam/menu/business_overlay_component_registry.go`

重建器只在 Base 的聚合入口接入这些静态注册表。Base 的 `generated_*` 注册表仍由低代码生成器独占，可以反复生成和 purge，不得存放 Ops 内置业务注册。菜单、权限、数据库 migration、seed、运行态 i18n 和 API 路由由各业务模块负责。

## 3. 升级流程

```powershell
# 1. 在临时目录重建，不覆盖当前 Ops
npm run rebuild:from-base -- --base ..\pantheon-base --target .tmp\clean-base-overlay

# 2. 验证声明和生成结果
node scripts/business-overlay/check-business-overlay.mjs --root .tmp\clean-base-overlay

# 3. 在暂存树运行 backend、frontend、业务 smoke 和 SonarCloud 门禁

# 4. 只有全部通过后，才以暂存树替换 Ops 工作树
```

相同 Base commit 和相同 overlay 必须得到相同报告与文件。每周 GitHub workflow 直接在 Base HEAD 上执行兼容性重建和编译；失败表示真实接口断点，不表示需要逐文件追平代码差异。

## 4. 禁止路径

- 不发布 Base patch 只为补消费白名单。
- 不把 Base release producer 脚本复制到 Ops。
- 不把 Ops 全树 import 改写成 `pantheon-ops`。
- 不保留未在 `business-overlay.json` 声明的通用产品文件。
- 不在未完成暂存验证前覆盖当前 Ops。

## 5. 变更落点

- 通用平台、系统域、共享 UI、共享生成器问题回 `pantheon-base` 修复。
- 业务模型、业务页面、业务权限和业务验收留在 `pantheon-ops`。
- Base 升级只处理真实 contract break；代码行数差异和共享文件差异不再作为人工同步任务。

## 6. 验证门禁

- `npm run test:business-overlay`
- `npm run check:business-overlay`
- Windows `go test -race ./...`（在 `backend/` 执行并启用 CGO）
- `frontend`: lint、type-check、build
- business smoke
- SonarCloud revision、代码量和重复率复核

当前仓库替换属于 L2 human gate：暂存验证通过后才能执行最终基线替换。
