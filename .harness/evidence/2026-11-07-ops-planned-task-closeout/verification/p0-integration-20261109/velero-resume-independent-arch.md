## Summary

**Architectural Status：`WATCH`。** 当前 diff 修复了“先持久化、后鉴权”的入口缺陷，namespace mapping 编码修复也有测试支持；可批准这两项局部源码修复。该结论不代表凭据撤销、异步终态或完整恢复授权已经闭环，更不代表真实 S3、HTTP、审计或最终 P0 验收通过。

本轮仅作只读独立审查，没有修改源码、计划、配置或 provider。

## Analysis

**1. 持久化前的信任边界修复成立。**

`CreateRestore` 先复制 DataScope，再调用 `restoreSource`；后者先获取经过集群授权和凭据检查的客户端，随后才查询源名称。提交时重新校验，并锁定集群和源记录；worker 执行时再次检查源。因此，外部门猜测源名称不能先创建恢复记录。证据：[backup_service.go:497](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:497)、[backup_service.go:541](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:541)、[backup_service.go:822](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:822)。

源资格同时要求本地 Completed、未过期，以及 live CRD Completed、未删除、有效 expiration；这证明源当前可用资格，不能证明 S3 对象内容或恢复结果。证据：[backup_service.go:836](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:836)。

**2. 凭据撤销测试的结论需要限定。**

实际 helper 在访问缓存前重新查询集群和凭据，所以缓存本身不会绕过这次检查。但是，引用凭据 revoked/missing 时，helper 仍回退到 `KubeconfigEncrypted`；若两种表示同时存在，撤销引用不能保证停止使用旧 inline 凭据。两类客户端均存在该路径：[cluster_service.go:466](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/cluster/cluster_service.go:466)、[cluster_service.go:505](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/cluster/cluster_service.go:505)。

新增撤销测试覆盖的是 **ref-only** fixture，没有覆盖上述双表示情况：[restore_authorization_test.go:127](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/restore_authorization_test.go:127)。这是依赖中的既有风险，需向 root 提交范围扩展，不能用本次绿色测试宣称“所有凭据撤销场景通过”。

**3. 锁保护本地提交，不能证明可靠 provider 终态。**

集群锁随数据库事务结束释放；worker 和 provider 调用发生在提交之后。worker 获取客户端、读取 live 源之后，也没有持续持有撤销锁。证据：[cluster_service.go:323](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/cluster/cluster_service.go:323)、[backup_service.go:557](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:557)、[backup_service.go:809](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:809)。

既有异步语义仍可能产生错误终态：provider 已接受 Create、但响应超时，worker 会写 Failed；reconciler 仅扫描 New/InProgress，之后无法纠正该记录。提交后、CRD 创建前重启，也没有持久化任务重放；NotFound 会写 Failed。证据：[velero_client.go:259](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/velero_client.go:259)、[backup_service.go:559](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:559)、[reconciler.go:158](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/reconciler.go:158)、[reconciler.go:178](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/reconciler.go:178)。这是源码推导的反例，本轮未执行故障注入。

**4. Namespace remap 的编码成立，授权证明仍缺失。**

转换保留了映射，测试验证实际提交对象中的 `owned-source → owned-target`：[velero_client.go:26](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/velero_client.go:26)、[restore_authorization_test.go:404](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/restore_authorization_test.go:404)。

但恢复路径没有检查目标 NamespaceBinding，且沿用 `includeClusterResources=true`：[backup_service.go:769](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:769)。这与 Active K8s 设计要求的 namespace mutation 授权存在张力：[BUSINESS_K8S_MODULE_DESIGN.md:67](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_K8S_MODULE_DESIGN.md:67)。该行为早于本 diff，属于完整恢复安全验收的待决边界。

响应转换未回显 live source Object 或凭据，公开构造器和 Service 签名保持不变，新增依赖限于业务模块、公共包及已有第三方库：[backup_service.go:11](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:11)、[backup_service.go:52](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:52)、[backup_service.go:645](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/k8s/backup/backup_service.go:645)。HTTP 错误呈现与审计落库未在本轮验证。

## Root Cause

入口缺陷源于将授权放在异步 worker 中，使数据库接受请求早于授权判定。当前修复解决了该顺序问题；剩余风险来自把**本地接受记录、远端提交结果、provider 执行终态**混在同一状态模型中，以及复用 helper 时未覆盖其凭据回退语义。

## Recommendations

1. **凭据双表示撤销回归——小工作量，高安全收益。** root 扩展到业务域 cluster helper：有引用 ID 时，revoked/missing 应拒绝；仅无引用 ID 时允许 legacy fallback，并补组合 fixture。
2. **异步状态修复——中工作量，高可靠性收益。** 区分确定拒绝与提交结果未知；未知结果保持可 reconcile，按稳定身份查询 provider，避免盲目重试。补“已接受但响应超时、延迟创建、提交后重启”测试。
3. **恢复权限边界——需业务裁决，中工作量。** 明确恢复是否为受限的集群级管理能力；否则校验有效目标 namespace 的 binding/action，并单独限制集群资源恢复。当前映射测试不能替代这些检查。

## Trade-offs

| 选择                         | 优点                       | 代价                                   |
| ---------------------------- | -------------------------- | -------------------------------------- |
| 批准局部入口修复，保留 WATCH | 尽快消除已复现的越权持久化 | 全面撤销、恢复授权和可靠终态仍开放     |
| 将上述残余一并闭环           | 获得更完整的恢复安全保证   | 扩展三个文件之外的范围及状态机验证成本 |

**最强反方：** 本 diff 小且 28 个新增子测试通过，不应以既有缺陷阻挡明确有效的入口修复。我的综合判断是批准局部修复，同时保留残余及最终验收门禁。

## References / Verification

**本轮执行：** CodeGraph context/status、源码与 HEAD 对照、SHA256 两次核对、指定 tracked diff 的 `git diff --check`。未重跑测试或执行远端操作。CodeGraph 有 pending changes，最终行号采用当前文件读取。

**供应方凭据：** scoped race、vet、build 均记录 exit 0；race 声明 CGO=1、MinGW，日志包含 28 个新增子测试通过。证据：[velero-resume-closeout.json:23](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/velero-resume-closeout.json:23)、[velero-resume-green-0.txt:43](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/velero-resume-green-0.txt:43)。

HEAD：`0a1ff0239a2b46e3a3ac6e563a84680960bc8db4`

当前 SHA256 与 closeout 一致：

| 文件                            | SHA256                                                             |
| ------------------------------- | ------------------------------------------------------------------ |
| `backup_service.go`             | `4320ceb9bfdfa85eb660ef8fcf83dc0b4de02ecddf3bce8b0babd3f91a88e934` |
| `velero_client.go`              | `b24de4061c69f37daa7b8bfcd13cf5a7132d5f2f0c4fbf286ef46da9aeb0552b` |
| `restore_authorization_test.go` | `16b228b1416bda79debb068fe0b00545c14e1c0921d35b3697a27e2e7df197ea` |

本轮诊断已完成。供应 closeout 明确 `remotePhasesExecuted=[]`、`liveAcceptancePassed=false`；真实 S3 持久化及恢复内容、HTTP/审计、清理、完整仓库门禁和最终 P0 批准仍由 root 收口：[velero-resume-closeout.json:122](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/velero-resume-closeout.json:122)。
