# Foundation Upgrade Runbook — base-v0.8.11 → base-v0.9.0

日期：2026-07-26
决策：先消费 base 已切好的 v0.9.0（tag `base-v0.9.0` @ `0f9a8030`，dist 产物 2026-07-22 构建），ops 立即开启业务开发；待 base-v0.9.1 切出后再做 0.9.0→0.9.1 滚动升级，实证升级通道。

## 关键约束

- **不要用 `npm run upgrade:foundation:local-apply`**：该包装脚本会在 base 当前工作区（main@71f0bed，已领先 tag 4 天）重新打 bundle，产物会与 v0.9.0 tag 不符。
- 改用底层 `consume-foundation-release.mjs` 直接指向 7-22 已构建的 dist 产物（忠实于 tag 提交）。
- apply 前 ops 工作区最好干净；若有未提交改动，脚本会自动 `git stash`，**成功后不会自动 pop**，需手动 `git stash pop` 恢复。

## 执行步骤（ops 仓根目录）

```bash
cd D:\workspace\go\pantheon-platform\pantheon-ops

# 0. 确认工作区状态
git status -sb

# 1. 预览（不改任何文件）
node scripts/foundation-release/consume-foundation-release.mjs \
  --manifest ../pantheon-base/dist/foundation-releases/base-v0.9.0/manifest.json \
  --bundle ../pantheon-base/dist/foundation-releases/base-v0.9.0 \
  --dry-run

# 2. 应用（含 lock/继承文档更新 + 4 项检查 + 失败自动回滚）
node scripts/foundation-release/consume-foundation-release.mjs \
  --manifest ../pantheon-base/dist/foundation-releases/base-v0.9.0/manifest.json \
  --bundle ../pantheon-base/dist/foundation-releases/base-v0.9.0 \
  --apply-shared-backend --apply-shared-frontend \
  --update-inheritance-docs --rollback-on-error --check

# 3. 验证
cd backend && go build ./... && cd ..
cd frontend && npm run build && cd ..

# 4. 若步骤 0 有本地改动被 stash
git stash pop
```

## 预期结果

- `foundation-release.lock.json`：releaseVersion → `base-v0.9.0`，baseCommit → `0f9a8030b577488965fbaefadb4b4c973b6de718`，releaseLine → `0.9`，localPath → `.foundation/releases/base-v0.9.0`
- `.foundation/releases/base-v0.9.0/` 产物就位
- `docs/PROJECT_INHERITANCE.md`（zh/en）基线行自动更新
- overlay 文件不被触碰（generated_registry.go、componentRegistry.ts 等）
- 四项检查通过：check-inheritance-contract / check-base-backend-sync / sync-base-shared --check / check-menu-contract --check

## 待补文档（apply 成功后）

`docs/designs/FOUNDATION_UPGRADE_PATH.md` 需追加 base-v0.8.11 → base-v0.9.0 历史记录（当前基线、commit、变更范围、验证与回滚信息）——该文件不在 Claude 直接可写清单内，由维护者或 codex 落笔，内容可参考本 runbook。

## 已知后续

- base-v0.9.1 切出后（含 7-25 Sonar 清零 + 安全告警归零成果），ops 做 0.9.0→0.9.1 滚动升级实证；届时 base 需检出到 v0.9.1 对应提交或直接消费其 dist 产物。
- v0.9.0 发行包三份 notes 为空（治理瑕疵，FOUNDATION_RELEASE_MODEL §4），v0.9.1 起已备好写实模板（pantheon-base/.harness/evidence/release-base-v0.9.1/）。
