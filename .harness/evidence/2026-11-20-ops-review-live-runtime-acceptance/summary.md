# 本地业务实机验收阶段结果

Task: 2026-11-20-ops-review-live-runtime-acceptance；Status: in-progress；Review: [review.md](review.md)。

## 部署实机（deploy lane，候选 k8s33）

2026-11-21 在 115.190.251.143（同时作为 SSH 部署目标）重跑部署验收，d00–d06 全部 PASS（7/7，run `omx_accept_20261121_cmdb02`）。修复后二进制（backendPid 26144，sha256 5cd98af…，见 [deploy-final-receipt.json](deploy-final-receipt.json)）：

- **d05 固定模板注入 P0 已实机关闭**：`serviceName: nginx$(touch …)` 注入任务在 create 即返回 HTTP 400 `business.deploy.task.templateParamsInvalid`，任务未启动、远端 115.190.251.143 无注入 marker、预览无 `$(…)` 语法。原始失败证据经 writeEvidence 命名策略保留在 `.retry-*.json`。
- d03 SSH 部署真实执行 + 日志持久化 + 远端 marker PASS；d04 指纹不符拒绝执行 + 无 marker PASS。
- 固定模板统一安全门禁（serviceName/path/port/hostname/version/卸载根等字段级校验，create 与 render 双防线）单测 72 PASS / 0 FAIL，含 7 个新安全用例。
- 修复内容留 dirty worktree（`deploy_fixed_templates.go`、`deploy_fixed_safety_test.go`），未提交/push，待维护者/Codex 复核。

## K8s 实机（k8s lane）

详见 [k8s-live-report.md](k8s-live-report.md)。**2026-11-23 更新：k8s lane 已在当前候选 `bad479fc` 上重跑通过 22 PASS / 0 FAIL / 0 BLOCKED（run `omx-accept-20261121-k8s35`，逐用例证据 `k8s/omx-accept-20261121-k8s35/`，回执 [k8s-lane-redrive-bad479fc-20261123.json](k8s-lane-redrive-bad479fc-20261123.json)），取代此前 binary `fba00055` 的 PARTIAL（Velero Pod Pending BLOCKED）。** Velero 陈旧 image metadata 阻塞已解决（image entity 重建，pod Running，k19 备份 Completed）。原 V1–V8 和 C01–C06/E01–E04 没有整体关闭。受限跨模块权限、双租户、部署恢复/取消、Argo/filedist/observability 等原范围仍待验收。服务器凭据不进入文档。

**Velero pod blocker 根因定位（2026-11-22，read-only 诊断）**：非镜像拉取失败。节点 containerd（v2.3.2-k3s2，115.190.251.143）上 `velero-plugin-for-aws@sha256:b6ecf1a…` 与 `velero@sha256:367a1d2…` 均已在场，但 init 容器持续 `Init:CreateContainerError`（76k+ 次重试/11d）：

```
failed to check if this is a checkpoint image: failed to get image from containerd
"sha256:1a2a8fe1…": image "docker.io/library/import-2026-11-10@sha256:af344e0c…": not found
```

即 containerd 2.x 的 checkpoint 预检沿 image ID `sha256:1a2a8fe1…` 追到陈旧 `import-2026-11-10@sha256:af344e / 39ac…` 两行 image metadata（`io.cri-containerd.image=managed`，来自过往 `ctr images import`），而该 metadata 指向已不可解析的引用。content store 五块 blob 全部在场（content ls 236 项），病灶是**元数据悬空而非内容丢失**。

**修复执行证据（维护者放行 2026-11-22，pinned SSH 115.190.251.143）**：
1. 放行一：`k3s ctr -n k8s.io images rm` 删除三行陈旧 metadata 已执行成功（rc 0，`after_imgrows` 确认三行消失），但 pod 报错**逐字节不变** → 陈旧引用不在 image-store 行内；
2. 放行二（扩需）：`kubectl delete pod velero-7865c6d85d-vqj6s` 已执行，Deployment 重建新 pod `velero-7865c6d85d-746qp`（13s，新 IP），报错仍**逐字节不变**，`ctr c ls`/`ctr s ls` 无 velero container 对象 → 陈旧引用不在 pod/sandbox 元数据（kubelet 缓存），而在 **image entity 自身 chain**（checkpoint 预检每次 CreateContainer 从实体 metadata 重建该引用）；
3. 放行三（扩需）：改为**重建 image entity** —— rm 两行 docker.io velero 镜像行 + 经国内镜像 `docker.m.daocloud.io`（该节点全部业务镜像经此拉取）按 digest 重拉 `velero@sha256:367a1d2…` / `velero-plugin-for-aws@sha256:b6ecf1a…`，必要时回标 docker.io 名，再 delete pod 验证 <font color="gray">（执行中，2026-11-22）</font>。
目标终态：`kubectl -n velero get pods` Running + `business/k8s/backup/velero/status` runningPods≥1。

## 权限安全探测（V-probe 19/19 PASS）

2026-11-21 14:13Z 候选 k8s33（sha256 5cd98af…，`runs/omx_accept_20261121_vprobe/v/probe.json`）对清单 V1/V2/V5 全通过，覆写此前「未执行」状态：

- **V1 非管理员角色全拒**：仅勾 task:view 的 viewer 角色对 `GET/POST /tasks`、`POST /tasks/1/start`、`GET /credentials` 全部 body-code 403（框架 envelope 为 HTTP 200），无默认放行路径。task/credential 路由未绑权限是已文档化的 SUP-08 default-deny 设计。
- **V2 包导出/详情**：package:view 角色 `GET /packages` 200、`GET /packages/export` 与 `GET /packages/1` 均 403，精确匹配预期 `[200,403,403]`；keyMatch2 有意缺口成立。
- **V5 凭据泄露面**：credential 列表响应无任何 secret-bearing 键；DB `secret_encrypted='v1:<base64>'` 密文；非创建者（有合法 deploy 授权）无法读取创建者凭据。**两个发现**：①无 `GET /credentials/:id` 详情路由（清单「单条详情」探针 404 本身即为证据）；②`business:deploy:credential:*` 权限键未 seed（`role.permission.invalid`），凭据当前为 admin-only，无授予型非管理员授权；残余 SEC-004 缺口：`ListCredentials` 无创建者过滤（潜伏，当前无暴露）。

CMDB lane 修正为 PASS：c01-c08 在 retry 序列（00:20Z）全部通过（含 ECDSA collection fingerprint 经 pinned session 鉴权）；首试 00:09Z c04-c08 FAIL 为瞬时环境抖动（ssh-probe 内部 TypeError / 瞬时 `ssh_connect_failed` / 残留 run dir），同一 live backend 下 deploy lane 同 run 7/7 PASS。错误指纹拒绝（c06 `cmdbhost.ssh_connect_failed`、host 状态不变）是预期安全行为，仅记录 UX 语义 gap。

## 业务能力综合探测（CAP-live run 3：31 PASS / 1 BLOCKED / 0 FAIL）

2026-11-22 Claude 独立执行（Codex 暂不可用），`runs/omx_accept_20261122_cap/manifest.json`（32 用例）对清单非 K8s 能力全量实机复核。两处此前 FAIL 经根因定位后确认为**探针载荷问题而非产品缺陷**：

- **observability datasource create `500 create_failed` 根因**：服务端 SSRF 门禁（`pkg/security/ssrf`，`AllowedPorts={80,443,8080,9090,3000,3100,16686}`，`BlockPrivateIPs`）按设计拒绝了我此前探针的 `example.com:8081`（端口不在白名单）与 `127.0.0.1`（私网），handler 统一包装为 `create_failed`。改为 `http://example.com`（公共 + 端口 80 允许）后 create/list/update 全部 200。**SSRF 防护因此获得正向实机证明**：`127.0.0.1:9090`（端口允许但私网）被拒 = BlockPrivateIPs 生效；`example.com:8081`（公共但端口不允许）被拒 = 端口白名单生效。
- **service instance create `500` 根因**：`validateVMTarget` 要求目标 CMDB host 的 `businessScopeId == 服务所属业务域`；旧探针取的是全局首个 host（属其他域）。改为**在同一业务域内先创建 CMDB host 再建实例**后，bizscope→application→service(vm)→host→instance 全链 200/200，实例 reconcile 亦 200——顺带把 CAP-CMDB 的写路径与 CMDBHostReader/BizScopeReader 装配也做实机证明。清理 7 步全部 200。
- **CAP-ARGOCD 非代码缺陷 BLOCKED**：`POST /cicd/argocd/servers` → 500 `argocd.encryption_key_missing`（`argocd_encryption.go loadEncryptionKey` 需 `PANTHEON_ENCRYPTION_KEY`，运行环境缺失）。安全属性成立：throwaway `authToken` 从未出现在响应，server 列表 200 无 token。见矩阵 `GAP-ARGOCD-ENVKEY`；同环境还缺 `PANTHEON_BUSINESS_SECRET_KEY`（`GAP-BUSINESS-SECRET-KEY`），非空业务密钥现不可落库，空串 Encrypt 为 no-op 故 datasource 无影响。
- **CAP-OPSINIT PASS**：fresh stdout/stderr 启动日志含 "Starting Pantheon Base"/"Velero reconciler started"/"starting server port=8081"/GIN 路由表，维护任务完成日志干净，全日志无 `panic`。
- **CAP-METRICS PASS**：`/metrics` 200 + `pantheon_*` 系列存在。
- **CAP-SECRETS PARTIAL**：V5 DB 密文交叉核对 + argocd token 不泄露均成立；完整「写→密文→读明文」因上述两个环境密钥缺失暂不可做实机闭环。

**路由矩阵扩展（dm2，30/30 PASS，`runs/omx_accept_20261122_dmatrix2/dm2/probe.json`）**：新建仅持 `business:deploy:package:view` 的非管理员角色，对 deploy 全部未绑定路由（templates CRUD/export/import、task preview/update/delete/pause/resume/retry/cancel/rollback/rollback-info/events/events-history/logs/export、targets/resolve、task-host result、credentials POST/PUT/DELETE）逐条探测，**全部 body-code 403**（default-deny），同一角色 `GET /packages` 200（grant 生效照旧）。未发现任何文档外授权路径；CAP-DEPLOY 因之从 PARTIAL 收敛为 PASS（残余 V6 前端 10s 计时并入 GAP-UI-BROWSER）。清理仍受 `auth.operation.verification_required` 拦截（GAP-CLEANUP-VERIFY）。

## 完整仓库 Go race 失败分类定位（`full-race-mysql-20261121-090845.log`）

`go test -race -count=1 -v ./...`（pantheon-ops 即 base 模块 + business overlay 的消费树）：**0 处 DATA RACE**；11 个失败标记中 10 个为**环境性**、0 个产品缺陷：

- **6 个 observability 测试 FAIL 根因 *未启动 Redis***：`query_permit_test.go:31` / `query_workspace_handler_test.go:110/163` 全部 `ping redis at 127.0.0.1:26316 (from PANTHEON_TEST_REDIS_ADDR): dial tcp … actively refused`（TestQueryPermitActualRedisActorAndTenantCaps、TestQueryResponseReleaseRevocationAndSourceRevision、TestQueryPrivacyDenialAndRestoreNeverExecutes、TestWorkspaceResponseRechecksDatasourceAction、TestExplicitProbeResponseRechecksRevokedActionAfterStatusWrite 及 TestDatasourceHTTPQueryCancellation 的 prometheus/victoriametrics/tempo 子用例）。非代码缺陷，测试依赖环境。
- **4 个包 10m 超时**（`pkg/database`、`system/i18n`、`auth/login`、`business/deploy`）：`test timed out after 10m0s`，期间日志显示共享 MySQL 每条 CREATE/ALTER 达 0.8–3s（SLOW SQL ≥200ms 大量）、`sql: database is closed`——race 插桩 × 慢 MySQL 导致超时，非导出 race。
- **39 skips / 未跑**：k8s 全子树模块 race 单独运行**全绿**（`k8s-race-mysql-20261121-163638.json`：0 fail / 0 skip / exit 0，`./modules/business/k8s/...`）。

结论：当前机器未启动 127.0.0.1:26316 的测试 Redis 时，该 race 门禁不能作为代码健康信号；补齐测试依赖后应能显著收敛。此分类不改变 K8s lane PARTIAL 的整体结论。

## 遗留测试主体清理（2026-11-24，GAP-CLEANUP-VERIFY 关闭）

在隔离验收库 `ops_acceptance_formal_20261121` 上以候选 `28cdcd00` 后端走真实二次验证流程：`POST /auth/operation-verify` 返回 code 200 并签发 operation token；随后带 `X-Operation-Token` 调用 `/system/user/batch-delete` 删除全部 16 个探针用户，`/system/role/batch-delete` 删除 226 个探针角色（首轮 220 个成功，6 个因仍持有用户返回 `role.delete.error.has_users`，用户删除后第二轮全部成功）。删除后经 API 复读：探针用户/角色残余 0，仅剩 7 个 seed 角色与 1 个 admin 用户。三个 `omx_accept_20261121_cmdb02` 包被 `deploypackage.in_use` 正确保护，作为历史任务 fixture 保留。证据 [cleanup-residue-20261124.json](cleanup-residue-20261124.json)；运行只动本任务自己的 8083 后端，未触碰并发 Codex 的 8080 后端与其数据库。

## 证据与回执

最新 K8s 原始结果：[k8s-acceptance-results.json](k8s-acceptance-results.json)；部署运行证：`runs/omx_accept_20261121_cmdb02/manifest.json`；原矩阵：[acceptance-matrix.json](acceptance-matrix.json)；构建身份：[k8s-final-receipt.json](k8s-final-receipt.json)、[deploy-final-receipt.json](deploy-final-receipt.json)。完整仓库 race 失败及 provider/UI gaps 明确保留。

不提交、不推送、不发布；最终视觉/功能验收由维护者签核。