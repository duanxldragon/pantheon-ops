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

## 3. 开工前必须补的阅读顺序

开始编辑本仓库前，先按这条顺序阅读：

1. `../../docs/WORKSPACE_INHERITANCE.md`
2. `../pantheon-base/DESIGN.md`
3. `../pantheon-base/AGENTS.md`
4. `../pantheon-base/docs/README.md`
5. 对应的 base contracts、designs 和 acceptance docs

## 4. 本地业务范围

当前本仓库明确承载的业务域：

- `business/cmdb`
- `business/deploy`

## 5. 本地允许新增的文档范围

当前显式允许补充的本地业务文档包括：

- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`

未来可以继续增加新的 `business/*` 文档，但不应复制 base 的 platform 或 system-domain contracts。

## 6. Override Policy

- Allowed：业务域补充、业务验收说明、仓库本地执行细节
- Not allowed：重定义 base contracts、base layer ownership、key-first i18n rules、menu/permission split、shared UI hard constraints
- 如果 foundation 规则必须变更，先改 `pantheon-base`，再升级 `pantheon-ops`

## 7. 运行时隔离

- 运行时数据库必须和 `pantheon-base` 隔离
- 推荐默认 DSN 指向 `pantheon_ops`
- 可以共用一个 MySQL 实例，但不能共用同一套数据库 schema

## 8. 使用说明

- `PROJECT_INHERITANCE.md` 现在是中文主入口，和本仓库其他中文主文档保持一致
- 英文协作方请使用 [PROJECT_INHERITANCE.en.md](./PROJECT_INHERITANCE.en.md)
- 两个版本语义应保持同步；若有差异，以当前仓库实际继承配置为准并及时回补另一语言版本
