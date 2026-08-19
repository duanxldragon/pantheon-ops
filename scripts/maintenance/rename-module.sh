#!/usr/bin/env bash
# rename-module.sh — 批量将 Go module pantheon-platform 重命名为 pantheon-base
#
# 用法:
#   bash scripts/maintenance/rename-module.sh            # 真改
#   bash scripts/maintenance/rename-module.sh --dry-run  # 仅预览命中，不写文件
#
# 铁律:
#   1. import 永不带 backend/ 段（go.mod 在 backend/ 内）。
#   2. sed 仅匹配带前导引号的字符串字面量 "pantheon-platform/ → "pantheon-base/。
#   3. 豁免 .harness/evidence/**、pantheon-ops/**、docs/**/*.md 物理路径。
set -euo pipefail

DRY_RUN=${1:-}
SED_INPLACE=(-i)
if [[ "$DRY_RUN" == "--dry-run" ]]; then
  SED_INPLACE=()   # 占位；dry-run 不真正执行 sed，仅列出影响文件
  echo "=== DRY RUN (no files will be modified) ==="
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

# dry-run：仅列出会命中的文件与行，不执行任何 sed 写操作
if [[ "$DRY_RUN" == "--dry-run" ]]; then
  echo "Affected .go files (import literals):"
  grep -rIl '"pantheon-platform/' backend --include='*.go' || true
  echo
  echo "Total import literal hits:"
  grep -rIn '"pantheon-platform/' backend --include='*.go' | wc -l
  echo
  echo "Other targeted files:"
  for f in \
    backend/go.mod \
    backend/DEV_DB_INIT_GUIDE.md \
    frontend/src/modules/lowcode/generator/backendGenerator.ts \
    frontend/tests/smoke/business/generated/module-governance-host-real.spec.ts \
    frontend/tests/smoke/business/generated/module-governance-real.spec.ts \
    frontend/scripts/cleanup-generated-modules.mjs \
    frontend/scripts/cleanup-generated-modules.test.mjs \
    scripts/harness/triage-base-drift.mjs; do
    n=$(grep -c 'pantheon-platform' "$f" 2>/dev/null || echo 0)
    echo "  $f  ($n hits)"
  done
  echo
  echo "=== DRY RUN complete. Re-run without --dry-run to apply. ==="
  exit 0
fi

step() { echo; echo "--- $* ---"; }

# 1. Go 源码 + go.mod（仅替换带前导引号字面量）
step "1. backend/**/*.go import literals"
find backend -type f -name '*.go' -print0 | xargs -0 sed "${SED_INPLACE[@]}" \
  -e 's|"pantheon-platform/|"pantheon-base/|g'

step "1b. backend/go.mod module declaration"
sed "${SED_INPLACE[@]}" -e '1s|^module pantheon-platform$|module pantheon-base|' backend/go.mod

step "1c. backend/DEV_DB_INIT_GUIDE.md (Q1: sync rename)"
sed "${SED_INPLACE[@]}" -e 's|pantheon-platform/|pantheon-base/|g' backend/DEV_DB_INIT_GUIDE.md

# 2. 生成器模板（5 处）
step "2. frontend generator template"
sed "${SED_INPLACE[@]}" -e 's|"pantheon-platform/|"pantheon-base/|g' \
  frontend/src/modules/lowcode/generator/backendGenerator.ts

# 3. smoke 断言（4 处，模板字符串带反引号，模式匹配 pantheon-platform/）
step "3. smoke spec assertions"
sed "${SED_INPLACE[@]}" -e 's|pantheon-platform/modules/business/|pantheon-base/modules/business/|g' \
  frontend/tests/smoke/business/generated/module-governance-host-real.spec.ts \
  frontend/tests/smoke/business/generated/module-governance-real.spec.ts

# 4. cleanup 正则（旧 pantheon-platform/backend/modules/business → 新 pantheon-base/modules/business，去 backend/）
step "4. cleanup script regex"
sed "${SED_INPLACE[@]}" \
  -e 's|pantheon-platform\\/backend\\/modules\\/business\\/|pantheon-base\\/modules\\/business\\/|g' \
  frontend/scripts/cleanup-generated-modules.mjs

# 5. cleanup 测试 fixture（带 backend/ 的旧形态，同步规范化）
step "5. cleanup test fixture"
sed "${SED_INPLACE[@]}" \
  -e 's|pantheon-platform/backend/modules/business/mdqaorder|pantheon-base/modules/business/mdqaorder|g' \
  frontend/scripts/cleanup-generated-modules.test.mjs

# 6. drift 脚本归一化
step "6. triage-base-drift normalization"
sed "${SED_INPLACE[@]}" \
  -e "s|replaceAll('pantheon-platform/backend', 'MODNAME/backend')|replaceAll('pantheon-base/backend', 'MODNAME/backend')|" \
  scripts/harness/triage-base-drift.mjs

if [[ "$DRY_RUN" == "--dry-run" ]]; then
  echo
  echo "=== DRY RUN complete. Above is the matched content preview. ==="
  echo "Affected files would be:"
  {
    grep -rIl '"pantheon-platform/' backend --include='*.go' || true
    echo backend/go.mod
    echo backend/DEV_DB_INIT_GUIDE.md
    echo frontend/src/modules/lowcode/generator/backendGenerator.ts
    echo frontend/tests/smoke/business/generated/module-governance-host-real.spec.ts
    echo frontend/tests/smoke/business/generated/module-governance-real.spec.ts
    echo frontend/scripts/cleanup-generated-modules.mjs
    echo frontend/scripts/cleanup-generated-modules.test.mjs
    echo scripts/harness/triage-base-drift.mjs
  } | sort -u
else
  echo
  echo "=== rename-module.sh complete. Run verify-module-rename.sh to confirm. ==="
fi
