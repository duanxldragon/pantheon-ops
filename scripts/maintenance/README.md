# scripts/maintenance — 维护脚本

## rename-module.sh

批量将 Go module 从 `pantheon-platform` 重命名为 `pantheon-base`。

### 用法

```bash
# 1. 先 dry-run 预览影响面（不写文件）
bash scripts/maintenance/rename-module.sh --dry-run

# 2. 确认识别范围正确后真改
bash scripts/maintenance/rename-module.sh

# 3. 验证无残留
bash scripts/maintenance/verify-module-rename.sh
```

### 覆盖范围

| 步骤 | 目标 |
|---|---|
| 1 | `backend/**/*.go` 中带引号 import 字面量 + `backend/go.mod` L1 + `backend/DEV_DB_INIT_GUIDE.md` |
| 2 | `frontend/src/modules/lowcode/generator/backendGenerator.ts` 生成器模板 |
| 3 | `frontend/tests/smoke/business/generated/module-governance-{host-}real.spec.ts` 断言 |
| 4 | `frontend/scripts/cleanup-generated-modules.mjs` 正则（去 `backend/` 段） |
| 5 | `frontend/scripts/cleanup-generated-modules.test.mjs` fixture |
| 6 | `scripts/harness/triage-base-drift.mjs` drift 归一化 |

### 铁律

- import **永不带 `backend/` 段**（go.mod 在 `backend/` 内）。
- sed 仅匹配带前导引号的字符串字面量，不碰物理路径、注释、豁免清单。
- 豁免：`.harness/evidence/**`、`pantheon-ops/**`、`docs/**/*.md` 物理路径。

### 回退

每个任务单独 commit，回退用 `git revert <sha>`；或整支废弃：

```bash
git reset --hard origin/main
```

## verify-module-rename.sh

扫描 `backend/`、`frontend/src/`、`frontend/scripts/`、`frontend/tests/`、`scripts/harness/`，
命中非豁免的 `pantheon-platform/` 残留即 `exit 1`，否则 `exit 0`。
