## Summary

**Architectural Status: `WATCH`；建议 `COMMENT`。** 新候选身份、locale/doc 等价性及官方缓存比较已独立验证；未发现本轮 deleted-path、locale 回迁或缓存 helper 的架构阻断。导航 smoke 已 3/3 通过，但新增渲染 probe 实际为 **exit 1、13/14 通过**，仍不能给出整体视觉验收通过结论。

以下为追加报告 `continuation-current-candidate-architecture.md` 正文；不覆盖旧 WATCH 报告。

## Analysis

### 官方缓存 helper 保留了锁定比较契约

`.local/manual-preview-20261116/check-staged-official-cache-drift.mjs:8` 将工作目录和候选解析为真实路径，并限制候选位于本任务 `.local` 目录。`:17` 要求工作仓库锁与候选锁逐字节相同；`:23` 直接调用原 `resolveBaseFromLock(root)`，随后原样调用 `checkSharedDrift({ root: target, snapshot })`。

原 resolver 仍检查：

- pinned bundle/repo.tar checksum、官方 release、asset checksum、tag commit：`scripts/business-overlay/foundation-release.mjs:138`。
- git-archive commit 和重新提取 snapshot：`:111`。
- release manifest 与锁的 sharedPaths 相同：`:148`。

原比较器继续读取 **target** 的 manifest/allowlist，而不是工作仓库的比较结果：`scripts/business-overlay/check-shared-drift.mjs:24`、`:40`。没有新增本地 Base checkout、替代锁或跳过官方验证路径。

实际 helper receipt 为 exit 0：`continuation-drift-cache-locale-current-cache-approved-verification.json:13`。日志完成 **866 shared files / 856 identical / 10 allowlisted drift / 14 derived migrations**：`continuation-staged-drift-official-cache-final-locale-current-cache-approved.log:11`。

原 candidate-root CLI 仍是独立失败尝试，`spawnSync gh ETIMEDOUT` 已保留：`continuation-staged-drift-final-locale-current-cache-approved.log:1`。helper 成功没有将旧失败改写为成功。

### 新候选身份已独立核对

候选：

`.local/manual-preview-20261116/overlay-history-continuation-final-locale-current-cache-approved`

本 lane 只读重算结果：

- **1833/1833 report owned entries** 与候选实际 SHA256 匹配。
- report SHA256：`b8ba6850419427a973d2f29dde4f831fc7b650b58bbc1c1fdebe850c65f8e12f`。
- `continuation-locale-current-cache-approved-source-equivalence.json` 的 **1197 source rows**，当前 source、target 均与记录哈希匹配，0 不匹配。
- 15 份 canonical JSON、5 fallback、4 份 CMDB/Deploy 文档、rebuild/regression/nav、foundation lock 均 source/target 字节相同。
- Base commit 仍为 `6316f9b7189c86542424145fa49344ad737e952b`；官方 helper 身份收据见 `continuation-overlay-history-continuation-final-locale-current-cache-approved-official-cache-drift-identity.json:8`。

`businessOverlay.ts` 原件为 660 字节，当前为 661 字节；独立确认 **唯一差异是末尾追加一个 LF**，与 `continuation-business-overlay-newline.json:4` 的 before/after/candidate SHA256 一致。

前轮“旧 target 文档不一致”的 gap，在这个新候选中已关闭。

### 原修复与边界保持一致

Deleted-path inventory 仍只排除 Git 明确报告的 worktree deletion（`scripts/business-overlay/rebuild-from-base.mjs:58`），显式不存在声明仍抛错（`:89`）。回归文件与此前已验证版本哈希相同，没有改为任意缺失文件静默过滤。

四份设计文档当前与候选一致，仍正确区分：

- CMDB labels/options 的现行 `label:view` 绑定：`BUSINESS_CMDB_MODULE_DESIGN.md:357`、`.en.md:167`。
- Deploy template key/menu seed 与尚未注册的 API bindings：`BUSINESS_DEPLOY_MODULE_DESIGN.md:700`、`.en.md:248`。
- Package detail/export matcher gap 仍需 Base 修复及正式消费：`.md:389`、`.en.md:129`。

未新增 template API 绑定或本地绕过该 Base gap。

### 新增验证通过，但渲染 probe 保持红灯

完整 serial CGO race 实际 exit 0；收据记录 `TEST_DSN_REQUIRED=1`、CGO=1、`-race -count=1 -timeout=10m`、`-p=1 -mod=readonly`：`continuation-full-race-disposable-serial.log.receipt.json:2`。这是 disposable MySQL 的功能/race 证据，不是生产耐久性或 D4 证据。

迁移后 lint/typecheck/build 均 exit 0：`continuation-frontend-locale-current-verification.json:14`、`:31`、`:48`。Lint 仍有一条已记录 warning，不能称为无警告：`continuation-frontend-lint-final-locale-current.log:7`。

导航 smoke **3 passed**：`continuation-navigation-smoke-final-locale-current.log:9`。但渲染 probe receipt 实际 exit 1：`continuation-navigation-locale-current-verification.json:35`；`after-history-continuation-final-locale-current/acceptance.json:7` 唯一失败是 `/business/deploy/task`。

该 probe 先等待 `/部署任务/`（`.local/manual-preview-20261116/probe-navigation.mjs:113`），而旧 dashboard 已包含“最近部署任务”；URL 已改变时，这个通用文本可提前满足等待。随后 `wrongDashboard` 检查仍正确拒绝旧内容（`:119`）。

独立查看 `menu-177.png`，确实仍显示工作台、工作台 breadcrumb 和 dashboard。现有证据证明捕获到了未切换完成的页面，**不能证明等待稳定后该路由仍错误**。应补充独有页面 readiness 后复验，保留旧 red 和 `wrongDashboard` 断言。

移动目录/导航断言已通过（acceptance `:56`），记录 viewport/page 均为 390；已查看实际 `catalog-mobile.png`。这支持该次移动截图与无溢出结果，不代表整体视觉 gate 通过。

## Root Cause

已修复的问题是 Git 索引遗留删除路径，以及手工 generated fallback 没有回写 canonical JSON。当前新发现的渲染检查失败则是 **页面 readiness 文本不够独特，旧 dashboard 可提前满足**；生产路由持续错误尚未被证明。

## Recommendations

1. **低工作量：** 为 deploy/task 使用独有 heading/主操作按钮，并确认 dashboard 内容退出后再捕获；保留旧失败和错误页面断言，重新执行 probe。
2. **继续保留 WATCH：** 三语言 business fallback 的运行时接线尚未因本轮生成而改变；生成器仅注入 zh/en（`rebuild-from-base.mjs:388`），runtime loader 合并路径见 `frontend/src/i18n/index.ts:51`。
3. **继续保留 WATCH：** canonical 异值重复 key 的既有覆盖顺序债务未关闭，例如 `deploy/locales/zh-CN.json:126` 与 `v12locales/zh-CN.json:762`；生成器仍依赖递归 `Object.assign`（`rebuild-from-base.mjs:370`）。

## Trade-offs

| 方案                  | 收益                               | 边界                                   |
| --------------------- | ---------------------------------- | -------------------------------------- |
| 复用工作仓库官方缓存  | 避免候选重新下载；保留完整官方验证 | 原 candidate-root CLI 超时仍是失败记录 |
| 保持本轮窄范围修复    | 候选可重建且文案行为保持           | 三语言 loader、重复 key 所有权另行治理 |
| 独有 readiness 后复验 | 区分异步切换与稳定错误页面         | 当前渲染 red 不能预先关闭              |

**停止条件：** 本 lane 已完成当前候选和 helper 的独立架构核对；渲染 probe 仍有真实失败待复验。`WATCH` 不构成项目完成、生产/D4 或整体可合并批准。
