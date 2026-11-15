## Code Review Summary

**REQUEST CHANGES**

**Files Reviewed:** 13（仅 ArgoCD 当前变更及新增文件）  
**Total Issues:** 6 — CRITICAL 0、HIGH 4、MEDIUM 2、LOW 0。

Stage 1 的公共入口与授权合规未通过；集合接口的静默失败也未通过根因检查。

### Issues

1. **[HIGH] 脱敏未覆盖上游授权拒绝。**  
   [argocd_handler_authorization.go:24](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:24) 的审计覆盖在 TokenAuth/Casbin 之后执行。上游拒绝会跳过它；共享审计默认规则未遮蔽 `sshKey`、`caCert`、`intentVersion`，可能保存私钥、受限配置或有效 ticket。  
   **修复：** 通过 Base 公共契约在首次拒绝之前应用业务审计白名单；补真实认证/权限拒绝链路回归。现有 fixture 测试不足以证明此边界安全。

2. **[HIGH] 更新授权检查遗漏新目标。**  
   [argocd_handler_authorization.go:57](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler_authorization.go:57) 读取旧应用后覆盖请求中的 destination/namespace，随后只补充新 source URL。用户可能通过旧目标授权，写入无权修改的新目标配置。  
   **修复：** 按实际更新语义构造 proposed application，同时授权当前和拟议目标；失败时保持数据库不变。

3. **[HIGH，集成缺口] Base 实时授权及会话接线仍缺失。**  
   [argocd_service.go:30](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_service.go:30) 默认留下 nil authority，当前 composition root 未注入；[ActionActor:14](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_authorization.go:14) 也未传递发起会话标识。受保护的生产入口会 fail closed，实时角色/DataScope/session 撤销尚未完成。  
   **修复：** 保留 failclosed，接入 Base 公共实时读端口，并验证发起会话撤销后 provider 零调用。fixture callback 只能证明端口调用。

4. **[HIGH] 必填 ticket 与现有调用契约断开。**  
   [argocd_dto.go:134](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_dto.go:134) 新增必填 `intentVersion`，现有前端 sync/rollback 仍只发送 `{revision}`。另有 [SyncApplication:20](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:20) 未传 actor/ticket，无法通过新校验。  
   **修复：** 协调完成 preview→confirm→execute 客户端及调用方契约迁移，保留强制审阅；补权限、409 刷新及错误翻译验收。

5. **[MEDIUM，根因检查阻断] 数据库失败被吞成成功列表。**  
   [argocd_handler.go:87](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_handler.go:87)，同类代码位于 196、291 行，遇到任何读取错误直接 `continue`。数据库故障或请求取消可能返回成功空列表/部分列表。  
   **修复：** 区分明确拒绝、记录消失与基础设施失败；后者返回脱敏失败并保留安全诊断，补故障和取消回归。

6. **[MEDIUM] 普通轮询使 ticket 无故失效。**  
   [argocd_authorization.go:72](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_authorization.go:72) 将完整 app/server 纳入 fingerprint；reconciler 每轮更新 `LastChecked`、`UpdatedAt`。即使配置和 provider 均未变化，下一轮轮询仍会导致 409。  
   **修复：** fingerprint 仅包含执行配置、身份和凭据版本；保留 provider UID/RV/spec 条件检查，补轮询不变与真实配置漂移回归。

### Evidence and limits

- 13 文件均通过本地 `gopls check`；`git diff --check` 通过。指定的 `lsp_diagnostics`、`ast_grep_search` 未暴露，使用本地诊断及源码检查，未声称运行这些工具。
- 审查结束时，13 文件 SHA-256 全部匹配[验证回执及完整哈希清单](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-product-verification-result.json)。该回执记录 scoped race/vet/build 通过，**live 测试被跳过**。
- 精确 action marker、UID/revision 关联、条件 patch、持久化 Unknown/no-repeat，以及生产 TLS/SSRF 限制已有源码支持。
- 新状态复用现有 history 字段；本轮未证明正式迁移启动路径。历史 source/config 漂移、完整 O3 preview/UI 和真实 Base 公共入口验收仍开放。

未修改源码或证据文件。本结论仅为独立 CODE/spec/security 审查；Base 契约和接线问题需提交协调者/architect，不能作为完整 O3/D1/P0 批准。
