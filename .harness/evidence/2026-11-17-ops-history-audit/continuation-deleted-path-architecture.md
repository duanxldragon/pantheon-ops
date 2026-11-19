## Summary

**Architectural Status: `WATCH`.** 未发现 deleted-path 修复和导航定位变更的架构阻断；缺失显式声明仍然失败，历史证据迁移保留原始字节。正式重建已通过，但最终候选文档已变化、共享漂移检查和导航运行时验证尚未绿灯，不能宣称 merge-ready 或项目完成。

本报告为独立、只读架构评审内容，供主线程追加保存为 `continuation-deleted-path-architecture.md`。

## Analysis

1. **删除路径修复针对实际根因，没有把所有缺失文件静默忽略。**  
   `scripts/business-overlay/rebuild-from-base.mjs:58` 用 `git ls-files --deleted` 排除已从工作树删除的索引条目，再消费 cached/others 清单。`copyDeclaredPath` 仍在显式声明不存在时抛错（`:89`），实际复制仍使用 `copyFileSync`（`:82`），迁移源仍必须是普通文件（`:170`）。  
   回归 fixture 先提交 live evidence（`tests/scripts/business-overlay/rebuild-from-base.test.mjs:36`），再归档；确认旧路径不存在、归档字节保留（`:134`）、重复 report 相同（`:188`），以及再次显式声明旧路径失败（`:199`）。  
   红日志明确是删除旧路径后的 `ENOENT copyfile`：`.harness/evidence/2026-11-17-ops-history-audit/continuation-overlay-deleted-path-red.log:15`；绿日志为 1/1 passed、0 skipped（`continuation-overlay-deleted-path-green.log:2`）。

2. **导航定位与实际 DOM 一致，运行时通过证据仍缺失。**  
   `frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts:140` 在 sider 的 `.arco-menu-inline` 内使用相对 `has` locator，随后限定 level-1 子项。DOM 证据确实显示服务目录父项及三子项位于该 group：`continuation-catalog-dom.json:34`。  
   内容等待改为 `main.toContainText(expectBody)`（测试文件 `:174`），避免空内容第一次读取直接失败。URL、按钮及选中 tab 的验证仍保留（`:294`）。  
   最新实际运行仍为 **1 passed / 1 failed / 1 did not run**，全导航测试超过 180000ms：`continuation-navigation-smoke-final-relative-scope.log:10`、`:18`。第三项服务目录测试未执行，因此 DOM 结构证据不能替代其功能通过证据。

3. **当前设计文档保持了真实权限边界。**  
   CMDB 的 labels/options 与 labels 使用 `label:view`，与 `backend/modules/business/cmdb/permission_routes.go:56` 一致；文档对应 `BUSINESS_CMDB_MODULE_DESIGN.md:356`、`.en.md:166`。  
   Deploy 文档现在区分模板 key/menu seed 与 API bindings：`.md:700`、`.en.md:248`。模板 create/update/delete seed 确实存在于 `backend/modules/business/deploy/deploy_seed.go:73`，当前注册函数只包含 package bindings（`permission_routes.go:12`）。  
   Package detail gap 仍开放（设计 `.md:389`、`.en.md:129`），符合只注册 package list/export 等分离授权的代码（`permission_routes.go:14`、`:26`）。官方 v0.18.9 snapshot 的 Casbin matcher 仍为 `keyMatch2`：`.foundation/releases/pantheon-base-v0.18.9/repo/backend/pkg/database/casbin.go:47`。未将该共享问题在 Ops 本地绕过或假关闭。

4. **已验证重建身份，但它不等于当前最终文档候选。**  
   官方锁为 v0.18.9 / `6316f9b7189c86542424145fa49344ad737e952b`：`foundation-release.lock.json:5`。重建收据为 exit 0（`continuation-foundation-deleted-path-cache-approved-verification.json:37`）；日志记录 1804 个 owned/generated 文件（`continuation-official-rebuild-final-deleted-path-cache-approved.log:1`）。  
   本 lane 独立读取并校验 target report：**1804/1804 SHA256 匹配，0 缺失/不匹配**；report SHA256：
   `930903010b4ac74cfe9a36db26b64f8e446c3138760c1780516f622cc57a0ec9`。  
   Target：
   `.local/manual-preview-20261116/overlay-history-continuation-final-deleted-path-cache-approved`。

   锁、重建脚本、回归测试和导航测试与当前 source 字节相同；四份设计文档均已在重建后变化，source/target 哈希不同。后续完整候选验收必须重新绑定当前文档字节。

5. **Foundation 检查不是全绿。**  
   Overlay 59/59、root scripts 238/238，均 0 skipped：`continuation-overlay-tests-final-deleted-path-cache-approved.log:70`、`continuation-scripts-final-deleted-path-cache-approved.log:472`。原 sandbox 57/59 的两项失败明确是 Go cache access denied（`continuation-overlay-tests-final-deleted-path-fixed.log:84`、`:96`）。  
   但 staged drift 收据 exit 1（`continuation-foundation-deleted-path-cache-approved-verification.json:71`），日志只有 HTTPS `wsarecv` 超时（`continuation-staged-drift-final-deleted-path-cache-approved.log:1`）。没有完成比较结果，不能记为 drift passed。CLI 需先通过官方 resolver，再进入比较（`scripts/business-overlay/check-shared-drift.mjs:158`）。

## Root Cause

Git cached 清单保留被归档移动后已不存在的旧工作树路径，导致 overlay 尝试复制旧 live evidence。修复使 inventory 表示实际候选工作树，同时保留显式路径声明和迁移源的失败边界。

## Recommendations

1. **低工作量：** 文档修改稳定后重新生成候选并记录 report/lock/hash 身份；当前 target 只能证明其冻结内容。
2. **中工作量：** 重跑正式 shared-drift 检查，保留当前网络失败；只有得到完整比较结果才能关闭该门禁。
3. **中工作量：** 在绑定身份的候选上，以低并发执行剩余 race/runtime 和导航三项测试；保持既有超时和覆盖范围，不跳过失败项。此前 auth/deploy/i18n 红灯本 lane 未重验，继续开放。
4. **低工作量：** 保留现有 missing-declaration 和迁移 regular-file 回归；以后不要改为通用 `existsSync` 静默过滤。

## Trade-offs

| 选择                            | 收益                                       | 代价                                                       |
| ------------------------------- | ------------------------------------------ | ---------------------------------------------------------- |
| 排除 Git worktree deleted 条目  | 支持历史证据迁移，保留声明错误检查         | 广泛声明根目录下的删除成为候选内容，需要固定候选清单和哈希 |
| 继续相对 DOM group 定位         | 避免 Kubernetes 同名子项冲突，符合实测 DOM | 依赖当前菜单结构；实际三项功能测试仍需通过                 |
| 保持 Base-first detail 授权 gap | 避免 viewer 权限扩大为 export              | 受限角色 detail 授权继续未完成                             |

**停止条件：** 本 lane 的窄范围诊断与建议已具备文件和收据证据；主线程完成最终候选绑定及剩余门禁前，维持 `WATCH` / `COMMENT`，不提升为完成或可合并结论。
