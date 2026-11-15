## Code Review Summary

**REQUEST CHANGES** — 审查指定当前 diff 共 **3 个文件，2 项问题**：CRITICAL 0、HIGH 1、MEDIUM 1、LOW 0。业务域同步准入有所修复，但异步执行仍有授权窗口。

### Issues

**[HIGH] worker 实时读取之后缺少锁内重新授权**

位置：[backup_service.go:810](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:810)、[backup_service.go:816](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:816)

worker 先校验集群权限、凭据和本地备份，再等待实时 Backup GET，随后直接 POST Restore。新增 GET 最长可等待 30 秒；期间若部门归属、凭据或本地源状态被撤销，返回的实时对象仍为 Completed，就会继续提交。同步落库阶段的锁不覆盖这个窗口。

**修复建议：**实时读取后，在集群和源记录锁内重新校验当前授权、凭据与源状态，并保护最后校验到单次提交之间的边界。增加“第二次 GET 期间撤销”的回归覆盖，断言 Restore CREATE 为零；提交结果不确定时不得自动重提。

**[MEDIUM] 外部门拒绝测试被凭据失败掩盖**

位置：[restore_authorization_test.go:96](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/restore_authorization_test.go:96)、[restore_authorization_test.go:137](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/restore_authorization_test.go:137)

首次外部门测试运行时尚无凭据；后续部门撤销测试又沿用了上一子测试撤销的凭据。因此这些测试可能因缺失凭据而通过，不能证明部门权限检查有效。“另一个集群”测试也仅使用不存在的集群 ID。

**修复建议：**隔离各子测试状态；使用有效凭据、有效源和已缓存客户端验证外部门拒绝。跨集群测试应提供两个可访问且有效的集群，仅让源归属不匹配。保留零落库、零 provider 调用断言，并验证具体拒绝原因。

### 审查与验证状态

- 同步路径先授权，再查询匹配集群的备份；检查本地与实时 Completed 状态、过期及删除状态，并在落库时锁定集群和源记录。
- DataScope 的切片字段已复制；namespace mapping 转为 JSON 原生 map，属于边界修复，没有新增掩盖失败的 fallback。公开签名和默认 CRD 字段保持。
- 错误包装保留取消原因，HTTP 边界使用既有消息脱敏处理。
- **本次实际执行：**三个文件的 `gopls check` 完成，退出 0、无诊断；`git diff --check` 通过。专用 `lsp_diagnostics`、`ast_grep_search` 未提供，模式检查使用 `rg`。
- **仅核对凭据，未重跑测试：**red 日志记录拒绝前落库失败；green 日志记录测试通过、无 SKIP。三个当前文件哈希均匹配 closeout 和 green 前后哈希。未找到 `velero-resume-verification.json`。

| 文件                          | SHA256                                                             |
| ----------------------------- | ------------------------------------------------------------------ |
| backup_service.go             | `4320ceb9bfdfa85eb660ef8fcf83dc0b4de02ecddf3bce8b0babd3f91a88e934` |
| velero_client.go              | `b24de4061c69f37daa7b8bfcd13cf5a7132d5f2f0c4fbf286ef46da9aeb0552b` |
| restore_authorization_test.go | `16b228b1416bda79debb068fe0b00545c14e1c0921d35b3697a27e2e7df197ea` |

范围外依赖需上报 leader/architect：既有集群 helper 在引用凭据失效时可回退内联凭据，dynamic helper 的凭据查询也未使用 caller context；本次 diff 不能证明这些合同已修复。

**运行态限制：**本次未运行 MySQL 测试或远端操作。真实 backup/restore、S3、provider RBAC、审计、HTTP、完整仓库 race 与整个 P0 均未验证。本报告仅作独立代码/spec/security 裁决；未修改文件。
