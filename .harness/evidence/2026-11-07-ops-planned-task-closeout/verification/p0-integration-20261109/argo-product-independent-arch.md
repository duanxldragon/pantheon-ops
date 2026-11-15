Architectural Status: **`BLOCK`**，仅针对本轮 Argo CD diff/new files；不代表整个 O3/P0 或发行裁决。

动作关联与持久化“不重发”设计有源码依据，应保留。但新授权边界存在两处缺口，旧 webhook 调用已发生兼容性回归；默认构造仍缺少生产 authority，不能宣称产品功能完成。

1. **P1：应用 PUT 只授权旧目标。**  
   请求中的 destination server/namespace 先进入临时对象，随后被数据库对象覆盖；回调检查的是旧目标。Handler 随后调用 Service 保存新目标，因此“允许修改旧 namespace”可变成“配置未经授权的新 namespace”。这是源码路径判断，未作 live exploit 复现。  
   证据：[授权对象覆盖](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:52)、[回调调用](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:96)、[新目标写入](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_service.go:399)。  
   **最小修复：**按 DTO 的实际更新语义合成 proposed application，分别授权当前对象和 proposed 目标，再写入。

2. **P1：namespace 约束忽略 API group。**  
   Diff 保存了 Group，但许可检查只检查 Kind；默认 namespace 的补全也只依据 Kind。自定义 group 下名为 `Role`、`Deployment` 的资源因此能通过当前谓词，无法证明它属于受支持的 namespace 资源。实际越界 mutation 还取决于集群 CRD 和 provider policy，本轮未验证这些条件。  
   证据：[执行许可检查](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:203)、[namespace 补全](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:250)、[Kind-only allowlist](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:341)。  
   **最小修复：**复用一个明确的 Group/Kind allowlist，同时用于补全和许可检查；当前无需新增 discovery 框架。

3. **P2：旧 webhook 入口保持签名，却不再保持行为。**  
   `SyncApplication(id, revision)` 不携带 actor/ticket，而新提交入口首先要求有效 ticket。现有 webhook 仍调用这个入口；其测试使用 `recordingSyncer`，不能发现真实 Service 的拒绝。  
   证据：[无 actor 的 wrapper](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:19)、[ticket 检查](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:43)、[webhook 调用](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_webhook.go:372)、[测试替身](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_webhook_test.go:78)。  
   **最小 handoff：**明确自动同步是否完全交给 Argo 原生能力；若保留 Ops 自动触发，需要独立的、明确授权的机器主体契约，不能伪造 admin actor 或绕过 review。

4. **P2：持久化终态恢复成立，调用者的准确结果恢复尚未闭环。**  
   reservation 保存了 action nonce，但没有保存 ticket/review 的关联标识；公开 history 仅按 application 返回最近 50 条，隐藏 actor/marker。丢失初始响应的调用者无法通过 ticket 准确找回自己的 history ID；重试则先经过过期与漂移检查。  
   证据：[reservation 元数据](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:60)、[重试前检查](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:43)、[公开历史投影](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_service.go:489)。  
   **最小修复：**在现有 history metadata 中保存 review 关联标识，允许经实时授权的同一 actor 找回既有结果；恢复路径绝不重新 dispatch。

已确认的设计支撑：

- reservation 在 dispatch 前提交；未结束的 Unknown/Running 阻止新提交，outcome 写失败仍保留可恢复记录。[action.go:64](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:64)、[action.go:150](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:150)
- reconciler 要求 UID、revision、唯一 nonce 匹配，并校验终态时间和成功 revision，旧或无关终态不能完成该记录。[reconciler.go:158](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_reconciler.go:158)
- ticket 绑定 actor、provider UID/resourceVersion、本地配置/凭据指纹及私有 diff；provider PATCH 再检查 UID/version/spec。[preview.go:81](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:81)、[client.go:269](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_client.go:269)
- 回滚只支持历史 source 与当前 source 除 revision 外一致的单源版本恢复；历史配置漂移被拒绝。Diff 不公开 manifest 值，旧资源保持 `retained`，符合 `prune=false`。[preview.go:151](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:151)、[preview.go:295](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:295)

根因是**授权尚未绑定完整的有效请求目标，而安全提交协议也尚未贯通所有调用方与结果恢复路径**。无需引入新的执行引擎、队列或 Ops 系统域依赖。

Base/协调者仍需接手以下边界：

- 默认构造不设置 authority，全部注册 HTTP 路由都会拒绝；这是明确的生产依赖。[service.go:30](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_service.go:30)、[handler_authorization.go:27](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:27)
- 公共 port 要求实时重读权限/DataScope，但 actor 没有显式 session provenance；不能据当前接口和 fixture 证明真实会话吊销闭环。[authorization.go:14](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_authorization.go:14)
- 审计清洗发生在 route handler 内，现有测试没有 global rejection middleware，无法证明全局拒绝之前已清洗。[handler_authorization.go:23](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:23)、[product_test.go:202](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_product_test.go:202)
- 测试使用 AutoMigrate；官方迁移链、现有前端 caller、permission/i18n 装配未在本轮源码范围内检查。[action_test.go:26](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action_test.go:26)
- Live driver 使用 fixture authority 和替代 client factory，不能替代真实产品 auth/TLS 验收。[product_live_test.go:46](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_product_live_test.go:46)

| 选择                                  | 优点                     | 代价                                                    |
| ------------------------------------- | ------------------------ | ------------------------------------------------------- |
| 保留当前 reservation/native operation | 持久化安全、复杂度有限   | Unknown 可能长期阻塞；dispatch 持锁最多受 30 秒超时约束 |
| 在现有 metadata 补结果关联            | 支持准确恢复，无需新队列 | 必须校验 actor、当前授权和匹配边界                      |
| Ops 自行实现系统授权                  | 可快速接线               | 违反系统域隔离，不接受                                  |

**最强反论：**nil authority 默认拒绝、provider policy 也可能拦截非法资源，当前缺口未必能在生产触发。  
**权衡：**这些保护适合保留未启用状态，不能作为未来安全接线的证明。  
**综合建议：**保留当前安全骨架，先做两处小授权修复和调用契约收口；实时身份能力通过 Base 公共契约接入。

独立执行了 `git diff --check`、`gofmt -l` 和哈希核对，19/19 文件匹配供应回执；未执行 race、VM、迁移或外部 mutation。供应 race 日志有 91 个 passing test events，`TestArgoProductLive` skipped，不能记作本轮独立执行。[verification-result.json:21](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-product-verification-result.json:21)

关键源码 SHA-256，与[完整清单](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-product-verification-result.json:100)一致：

```text
argocd_handler_authorization.go fb80bda56347af0dfd39ada46ebe7e79aee86193dd3880cbb1cc4524063c8fb0
argocd_preview.go               81688b73bb5f21ad8aebf559f5363caf8e3f20a95849133ac9cd61178cb6f264
argocd_action.go                cd8b29367659f5ea446db007c64aad4629c2a7b1f4b20e5b7dcbaec04e386fda
```

本轮诊断已有源码依据，停止于只读报告。修复后需重新独立复核；完整 O3/P0 和最终 release approval 仍由协调者负责。
