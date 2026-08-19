#!/usr/bin/env bash
# verify-module-rename.sh — 验证 module 重命名无残留
#
# 扫描所有应已迁移的位置；命中非豁免即 exit 1。
# 豁免: pantheon-platform/backend 物理路径形态（仅 docs/ 中允许，且本脚本不扫 docs/）。
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

hits=$(grep -rIn --include='*.go' --include='*.ts' --include='*.tsx' --include='*.mjs' \
       'pantheon-platform/' backend/ frontend/src/ frontend/scripts/ frontend/tests/ scripts/harness/ 2>/dev/null \
       | grep -v 'pantheon-platform/backend' || true)

# 同时检查带引号的 import 字面量残留（更严格）
quoted=$(grep -rIn --include='*.go' '"pantheon-platform/' backend/ 2>/dev/null || true)

if [[ -n "$hits" || -n "$quoted" ]]; then
  echo "FAIL: residual pantheon-platform/ references:"
  [[ -n "$hits" ]] && echo "$hits"
  [[ -n "$quoted" ]] && echo "$quoted"
  exit 1
fi

echo "OK: no residual pantheon-platform/ imports outside allowlist"
