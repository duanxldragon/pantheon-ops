# 同候选真实环境与受限角色验收

- Status: in-progress
- Updated: 2026-11-23T08:40:00.000000+00:00
- Owner: Claude 业务修复 / Claude 独立验收（Codex 暂不可用）
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc，dirty worktree 保留

## 2026-11-24 同候选 k8s lane 17/17 PASS（main `697c0771`）

当前 `main` 后端（clean source，binary sha256 `d6a76a5f…`）在 `115.190.251.143` K3s 上执行 `k8s-live.py --run-id omx-accept-20261121-k8s38`：**17 PASS / 0 FAIL / 0 BLOCKED**。k12 真实 Pod 日志 WebSocket、k13 凭据轮换/恢复、k14 namespace ownership/写保护、k15 清理独立核验全部 PASS。证据 `k8s/omx-accept-20261121-k8s38/`（summary-20261124-112210.json + 17 个 k*.json）。注：该 harness 的 summary receipt 把 `sourceHead` 硬编码为规划基线 `a2d69d4a`，候选实际 source head 为 `697c0771`；运行使用独立 DB `ops_acceptance_formal_20261121`，跑后 `runtime-live/ownership.json` 已按字节复原（codex 的 8080 后端未受影响）。k8s37 为同一候选的预演，k12/k15 两项失败分别由 Origin 允许清单缺 `127.0.0.1:5175` 与 harness DB 指向 codex 库造成，均为环境配置，k8s38 修正后全绿。

## 当前结果（2026-11-23：k8s lane 同候选重跑 22/22 PASS）

本地 Ops 已启动；115.190.251.143 同时作为 SSH 部署与 K8s provider。**K8s lane 已在当前候选 `bad479fc`（`pantheon-ops-codex-t02b-20261122-172553.exe`）上重跑通过 22 PASS / 0 FAIL / 0 BLOCKED**，关闭此前 binary `fba00055` 的 PARTIAL 陈述（22 PASS / 1 BLOCKED Velero）。Velero 陈旧 containerd image metadata 阻塞已解决：image entity 重建后 pod Running（1 pod），k19 真实 ConfigMap 备份 `omx-accept-20261121-k8s35-backup` **Completed**、errors 0、warnings 0。k15 owned 资源清理独立核验通过。harrness 修复：`k8s-live.py` 读 `runtime-live/ownership.json` 的 dbname（不再硬编码 `ops_acceptance_formal_20261121`）；k8s34 的清理 fence FAIL 即该 harness DB 名 bug，修复后 k8s35 全绿。

运行证：`k8s-lane-redrive-bad479fc-20261123.json`（supersedes `k8s-final-receipt.json`）；逐用例 `.harness/evidence/.../k8s/omx-accept-20261121-k8s35/`（summary-20261123-093416.json + 22 个 k*.json，candidate receipt backendSha256=`bad479fc…`、sourceHead=a2d69d4a）。

部署 lane（d00–d06）在候选 k8s33 全部 PASS，固定模板注入 P0 已实机关闭：注入任务 create 即 HTTP 400 `business.deploy.task.templateParamsInvalid`，任务未启动、远端无 marker。修复实现留 dirty worktree，未提交/push，待维护者/Codex 复核。

权限安全探测（V-probe 19/19 PASS，`runs/omx_accept_20261121_vprobe/v/probe.json`）关闭清单 V1/V2/V5：V1 非管理员 viewer 全拒 403；V2 包 `[200,403,403]`；V5 凭据响应无 secret 键 + DB 密文 + 非创建者不可读（附两条发现：无 `/credentials/:id` 详情路由、`credential:*` 权限键未 seed，凭据 admin-only；`ListCredentials` 无创建者过滤为潜伏残余）。CAP-CMDB 修正为 PASS（retry 序列 c01-c08 全过，首试 FAIL 为瞬时环境抖动）。

**T03 联动补充（2026-11-23）**：T03 已补 deploy 权限绑定（26 键）与 credential seed/i18n。修复后角色矩阵在**本地**候选 `0586a9e2` 实跑 **18/18 PASS**（`t03-role-matrix-local.json`）：task:view 读列表 200、写/导出 403；credential:list 读 200 且响应无 secret 键、写 403；package:view 读 200、详情/导出 403。矩阵 V1/V5/deploy-lane 已追加 `t03Recheck`。远端 k8s33/k8s35 尚未用该候选重跑 → 本项远端证据保持 gap。

## Next Atomic Action

**K8s lane 主段落已闭环（当前候选 bad479fc 上 22/22 PASS）。** Velero 阻塞已解决（image entity 重建，pod Running，k19 备份 Completed），不再有待执行远端清理命令。**GAP-BUSINESS-SECRET-KEY 已由 T02 round6 关闭**（bad479fc 进程经 `restart-live-secret.ps1` 补齐随机 32 字节 hex 密钥，见 `2026-11-20-ops-deploy-fixed-template-security/round6-filedist-bootstrap-20261123.json`），argocd 的 `GAP-ARGOCD-ENVKEY` 仍缺失。K8s lane 收尾仍开放：V8 SA 凭证撤销轮换、Ingress provider 变更/快照、全新安装、WebSocket 释放（k12 已 PASS 但 server 侧资源回收需单独观察）、Velero schedule/restore——以上 shelf 项多数依赖 T06/T07/T08 的实施而非单方验收，留至实现队列。GAP-UI-BROWSER 前端浏览器、V4 多租户配置、V7 低权限用户为环境/放行依赖。完整仓库 race 已分类（见 Known Gaps）。修复留 dirty worktree，未提交/push，待维护者签核。

## CAP-live run 3（2026-11-22，Claude 独立执行）

32 用例 **31 PASS / 1 BLOCKED / 0 FAIL**（`runs/omx_accept_20261122_cap/manifest.json`），矩阵 V3→PASS、V6→PARTIAL，9 个 NOT_RUN 能力收敛：

- CAP-SERVICE / CAP-BIZSCOPE / CAP-FILEDIST / CAP-OBSERVABILITY / CAP-METRICS / CAP-CAPABILITY / CAP-OPSINIT → **PASS**
- CAP-ARGOCD → **BLOCKED**（`argocd.encryption_key_missing`，运行环境无 `PANTHEON_ENCRYPTION_KEY`；token 从未泄露）
- CAP-SECRETS → **PARTIAL**（V5 密文 + 无泄露成立；写→密文→读闭环需补两个环境密钥）
- CAP-DEPLOY → **PASS**（deploy lane 7/7 + dm2 路由矩阵 30/30：仅持 package:view 的非管理员对 templates/task actions/credentials 全部 403，正控 GET /packages 200，无文档外授权路径；V6 前端 10s 计时挂 GAP-UI-BROWSER）
- 两个 500 根因均为探针载荷问题非产品缺陷（datasource URL 撞 SSRF 白名单、instance host 跨域），反向获得 SSRF 正向实机证明（BlockPrivateIPs + 端口白名单均生效）。

## Evidence

见 [阶段汇总](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/summary.md)、[K8s 结果](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/k8s-live-report.md)、[审查](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/review.md)。**本轮 k8s lane 重跑回执**：[k8s-lane-redrive-bad479fc-20261123.json](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/k8s-lane-redrive-bad479fc-20261123.json) + [逐用例目录](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/k8s/omx-accept-20261121-k8s35/)。部署运行证与源码/二进制回执：`runs/omx_accept_20261121_cmdb02/manifest.json`、[deploy-final-receipt.json](../../evidence/2026-11-20-ops-review-live-runtime-acceptance/deploy-final-receipt.json)。CAP 运行证：`runs/omx_accept_20261122_cap/manifest.json`（cap/*.json 32 个用例）。专属测试资源清理回执保留；此前未执行状态已被真实证据取代。

## Known Gaps

**Velero 阻塞已关闭**：陈旧 containerd image metadata（`import-2026-11-10@sha256:…`）经 image entity 重建解决，pod Running，k19 备份 Completed（0 errors/0 warnings），见 k8s35 证据。剩余缺口不变：完整 Go race **已分类**：0 DATA RACE，6 个 observability FAIL=测试 Redis 127.0.0.1:26316 未启动（dial refused），4 包=慢 MySQL 下 -race 10m 超时（非代码缺陷），k8s 子树单独全绿（0/0）；V4 双租户 BLOCKED（tenant_mode multi 未配置）；V7 SSH restart/cancel 低权限用户未配置；V6 前端 10s 一半未跑（GAP-UI-BROWSER）；argocd 加密运行环境密钥缺失（`GAP-ARGOCD-ENVKEY`，业务 `GAP-BUSINESS-SECRET-KEY` 已由 T02 round6 关闭）；vprobe/v3 遗留测试主体清理停在 operation-verify 二次验证（GAP-CLEANUP-VERIFY）；K8s shelf 剩余：V8 SA 撤销轮换、Ingress provider 变更/快照、全新安装、WebSocket server 资源释放、Velero schedule/restore。部署 lane 全绿（含 d05 注入关闭）。不提交、不推送、不发布、不修改共享底座。
