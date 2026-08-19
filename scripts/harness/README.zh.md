# Harness 脚本

English version: [README.md](./README.md)

这里存放 `pantheon-base` 的仓库本地 Harness 检查。

可复用方法事实源位于同级仓库 `../pantheon-harness/`。本目录只负责 `pantheon-base` 的落地执行：一部分脚本镜像自 portable checker，一部分脚本是 base 专属门禁。

默认上游解析顺序：

1. `--method-root <path>`
2. `PANTHEON_HARNESS_ROOT`
3. `config/method.config.json` -> `pantheonHarnessRoot`
4. `../pantheon-harness`

## 快速验证

```powershell
node scripts/harness/check-method-health.mjs --root . --strict
node scripts/harness/check-adoption.mjs --root . --strict
node scripts/harness/check-template-health.mjs --root . --strict
node scripts/harness/check-doc-frontmatter.mjs --root . --strict
node scripts/harness/check-doc-links.mjs --root . --strict
node scripts/harness/check-doc-inventory.mjs --root . --strict
node scripts/harness/check-sync-drift.mjs --root . --strict
node scripts/harness/check-encoding.mjs --root . --strict
```

## 共享工具

- `sort-utils.mjs` - 字符串排序工具。
- `upstream-root.mjs` - 解析当前 workspace 中的 `pantheon-harness` 方法源。

## Pantheon Base 专属检查

- `check-audit-coverage.mjs`
- `check-backend-dto-contract.mjs`
- `check-backend-response-contract.mjs`
- `check-coverage.mjs`（单测覆盖率门禁，支持 Go cover profile 与 vitest coverage-summary `--format json`）
- `check-inheritance-contract.mjs`
- `check-structure-contract.mjs`（目录放置 + 命名门禁，契约源 REPOSITORY_LAYOUT.md；与 check-boundaries 的 import 边界互补）
- `check-permission-contract.mjs`
- `triage-base-drift.mjs`
