## Code Review Summary

**Recommendation: REQUEST CHANGES**

本报告保留 `locale-current-cache-approved` 候选的审查结论。导航就绪断言存在一项 MEDIUM 缺陷；工作树已补修复，但新验证和候选绑定尚未完成。其他受审范围未发现新增阻断。

**Files Reviewed:** 32 个范围内文件，另检查 3 个相关合同实现。  
**Total Issues:** 1。  
**Severity:** CRITICAL 0 / HIGH 0 / MEDIUM 1 / LOW 0。

### Issue

**[MEDIUM] 部署任务页的内容断言能提前命中工作台**

File: [候选 ServiceCatalog.nav.spec.ts](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/manual-preview-20261116/overlay-history-continuation-final-locale-current-cache-approved/frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts:174)

候选使用 `/部署任务/` 判断目标页加载完成，但工作台包含“最近部署任务”。URL 已切换而目标页面尚未渲染时，断言就能通过，因此 `3 passed` 不能充分证明全部页面加载正确。

原渲染证据明确记录 `/business/deploy/task` 的 `pathMatches=true`、`expectedContent=true`，同时 `wrongDashboard=true`；[menu-177.png](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-16-ops-navigation-repair/after-history-continuation-final-locale-current/menu-177.png)实际显示工作台，面包屑仍为“工作台”。该 probe 正确保留 **exit 1**，不能将它改写为通过，也不能删除工作台残留检查。

**Fix:** 等待准确目标 URL、目标页独有且可见的主标题/控件，并要求主区域不再包含工作台内容；Spec 和 probe 使用相同就绪合同。工作树现已加入 exact heading“部署任务”和工作台内容消失断言。须以独立新证据复审，并绑定新测试版本到新候选后才能关闭本问题。

### Spec 与根因检查

删除路径修复保留了明确声明路径缺失时的失败，只从 Git 枚举中排除已经删除的 tracked 文件。回归同时检查旧路径消失、归档字节保留及显式缺失声明报错；未引入静默跳过声明的分支。

locale 对账修复 canonical 数据缺口，没有绕过生成合同：718 个缺失键值补回，10 处值差异对齐；15 个 canonical 文件和五份生成资源已核对，全部原 fallback 值保留。跨文件重复键仍为 1199，其中既有值冲突仍为 20，未新增重复或冲突。

四份设计文档中的三项旧 MEDIUM 问题已修复：当前 `view` 权限、CMDB 配置接线与剩余 Base 依赖、Deploy 已 seed key 与尚未注册 API bindings 均已明确区分。

### 官方缓存 helper

[check-staged-official-cache-drift.mjs](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/manual-preview-20261116/check-staged-official-cache-drift.mjs:17)先检查 root/candidate lock 字节一致，再调用原 `resolveBaseFromLock(root)` 和原 `checkSharedDrift({root: target, snapshot})`。

官方 release/tag、pinned checksum、GitHub assets、archive commit 和重新解压校验均保留。该狭窄入口可接受的条件是继续限制在相同官方 lock/cache，保持原失败可见，并保留主 CLI 与此入口的独立证据；不能扩展为 local Base、替代 lock 或离线跳过官方身份校验。

实际缓存入口 **exit 0**：866 个 shared files，856 个与 Base 一致，10 个允许漂移，14 个派生 migration 已验证。原 candidate-root CLI 的 `spawnSync gh ETIMEDOUT` **exit 1** 独立保留。

七组内存诊断通过：范围外路径、非法名称、既有 receipt、lock mismatch、resolver error 均拒绝；drift 非零保持非零，成功保持零。诊断没有磁盘写入或网络调用，覆盖 helper 的分派和失败传播，不替代官方 resolver 验证。

### 验证状态与边界

- 完整 serial CGO race **exit 0**，使用 disposable MySQL、`-race -count=1` 和 `-p=1 -mod=readonly`。
- locale 迁移后的 frontend lint/type-check/build 均 **exit 0**；lint 保留一项既有 warning。
- 官方重建、staged overlay 和独立缓存 drift 比较通过；默认 drift CLI 超时失败保留。
- 独立核对 receipt 中 **1197** 个源文件的工作树/候选字节及 hashes 全部一致；businessOverlay.ts 仅补一个末尾 LF，原件和 hashes 相符。该清单限于 `backend`、`frontend/src`，不覆盖 smoke tests 或 `.local` helpers。
- 原导航 smoke **3/3 passed**；随后渲染 probe **exit 1**，14 项中 13 项通过，失败为上述部署任务就绪检查。已查看桌面和移动端截图；服务目录及移动导航检查通过，移动宽度为 390/390。完整视觉门禁仍未关闭。
- 六个 JavaScript 文件的 `node --check` 通过。`lsp_diagnostics`、`ast_grep_search` 不可用，未取得 LSP 诊断，不能给正式 APPROVE。

本报告不关闭 C/E、外部/生产验收或最终项目完成。保持旧 COMMENT 报告和本轮红证据，后续另写就绪断言复审报告。本 reviewer 未写入任何文件。
