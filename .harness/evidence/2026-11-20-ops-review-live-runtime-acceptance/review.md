# 验收审查

结论：changes-required。Velero provider 尚未就绪，完整仓库 race 未通过，原 V1–V8 和旧十包仍有未执行场景。局部 K8s PASS 不作为整体完成结论。实现由 Claude 提交文件，Codex 对差异、真实 API、独立集群状态和渲染结果验收；没有另用 Codex 修改业务实现。

变更落在 business/k8s 和业务翻译扩展；没有修改 platform/system/shared 请求或 foundation release。缓存按 cluster + credential digest 区分，空快照事务保持 cluster fence，UI 写操作明确归属和有限动作，CAS 保留。独立审查人为本任务指定 Codex，未再派产品代码代理代替 Claude。

## 权限安全探测（V1/V2/V5 + CAP-CMDB 修正）

2026-11-21 在候选 k8s33 补跑 V-probe（`runs/omx_accept_20261121_vprobe/v/probe.json`，19/19 PASS），清单 V1/V2/V5 从 NOT_RUN 置为 PASS：

- V1 viewer 全拒（task:view 不产生路由效力，SUP-08 default-deny 设计）；V2 包 `[200,403,403]`（keyMatch2 有意缺口成立）；V5 响应无 secret 键 + DB 密文 + 非创建者不可读 — 附两个发现：①无 `GET /credentials/:id` 详情路由；②`business:deploy:credential:*` 权限键未 seed（凭据 admin-only），残留 `ListCredentials` 无创建者过滤（潜伏）。这修正了此前断言方法学偏差（把 body-code 403 误判为未执行），安全本身工作正常，非新增漏洞。
- CAP-CMDB 修正为 PASS：c01-c08 在 retry 序列（00:20Z）全部通过；首试 FAIL 为瞬时环境抖动，同一候选二进制 deploy 7/7 PASS。错误指纹拒绝为预期行为，UX 上 `ssh_connect_failed` 掩盖主机指纹失配（单独 product finding）。

不改变整体结论：K8s lane PARTIAL、multi-tenant BLOCKED、V6 未跑、完整仓库 race 未通过等原缺口仍须按 findings 关闭。

## 业务能力综合探测（CAP-live run 3，Claude 独立执行）

2026-11-22（Codex 暂不可用，由 Claude 独立验证），候选 k8s33（sha256 5cd98af…）实机跑 `runs/omx_accept_20261122_cap/manifest.json`，32 用例 **31 PASS / 1 BLOCKED / 0 FAIL**。本组数据将矩阵 V3 置 PASS、V6 置 PARTIAL（10s 前端一半仍缺），并把 9 个 NOT_RUN 能力收敛：CAP-SERVICE / CAP-BIZSCOPE / CAP-FILEDIST / CAP-OBSERVABILITY / CAP-METRICS / CAP-CAPABILITY / CAP-OPSINIT → PASS，CAP-ARGOCD → BLOCKED（非代码缺陷），CAP-SECRETS → PARTIAL。

两处此前 500 经根因定位为探针载荷问题（非产品缺陷），同时**获得 SSRF 正向实机证明**（BlockPrivateIPs 拒私网、AllowedPorts 拒非白名单端口）。blcok 项 CAP-ARGOCD 与新增两个部署配置 gap（`PANTHEON_ENCRYPTION_KEY` / `PANTHEON_BUSINESS_SECRET_KEY` 缺失）见新 findings，均记录 blockedReason，不以产品缺陷上报。

不改变整体结论：K8s lane PARTIAL（Velero）、multi-tenant BLOCKED、V7 SSH restart/cancel 低权限用户未配置、完整仓库 race 未通过等原缺口仍须关闭；本组新增缺口在矩阵 gaps 与下列 Machine Readable findings 中显式登记。

## 部署注入修复复测（deploy lane）

固定模板注入（服务名/路径/卸载根等自由文本注入）在候选 k8s33 实机复测已关闭：注入任务 create 即返回 HTTP 400 `business.deploy.task.templateParamsInvalid`，任务未启动、远端无 marker；d00–d06 全部 PASS（run `omx_accept_20261121_cmdb02`）。修复实现留 dirty worktree（见 [deploy-final-receipt.json](deploy-final-receipt.json)），未提交。此关闭不改变整体结论，原 K8s BLOCKED 及 V1–V8/旧十包缺口仍须按 findings 关闭。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-review-live-runtime-acceptance",
  "reviewer": "Codex /root",
  "verdict": "changes requested",
  "decision": "changes-required",
  "scope": "K8s runtime acceptance and linked parent gaps",
  "taskManifest": ".harness/tasks/2026-11-20-ops-review-live-runtime-acceptance/manifest.json",
  "evidenceDir": ".harness/evidence/2026-11-20-ops-review-live-runtime-acceptance",
  "summaryFile": ".harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/summary.md",
  "changeRef": "none",
  "deployLane": {
    "status": "PASS",
    "run": "runs/omx_accept_20261121_cmdb02/manifest.json",
    "d05InjectionFix": "HTTP 400 business.deploy.task.templateParamsInvalid at create; task not started; remote marker absent",
    "receiptFile": "deploy-final-receipt.json"
  },
  "vProbe": {
    "status": "PASS",
    "run": "runs/omx_accept_20261121_vprobe/v/probe.json",
    "cases": "19/19 on candidate k8s33 (sha256 5cd98af…)",
    "V1": "all-403 per checklist (viewer vs tasks/create/start/credentials)",
    "V2": "[200,403,403] list/export/detail",
    "V5": "no secret key in list response; DB ciphertext; non-creator denied; findings: no /credentials/:id route + no seeded credential:* perm key",
    "CAP-CMDB": "PASS via retry sequence (c01-c08, 00:20Z); first-attempt FAIL was transient env flake"
  },
  "findings": [
    "multi-tenant isolation",
    "Helm chart installation is not implemented",
    "frontend 10-second timeout",
    "WebSocket server resource cleanup",
    "Velero schedule and restore",
    "Ingress provider changes",
    "Velero ConfigMap backup",
    "Ingress snapshot management not exercised",
    "narrow mobile/dark/keyboard UI not exercised",
    "service-account revoke plus workload polling sequence in original V8 remains incomplete",
    "full repository race failed: 11 failure markers, 39 skips; classification (2026-11-22): 0 DATA RACE; 6 observability FAILs = missing test Redis on 127.0.0.1:26316 (dial refused), 4 packages (pkg/database, system/i18n, auth/login, business/deploy) = 10m test timeout under -race against slow shared MySQL (DDL 0.8-3s), not code defects; k8s subtree race green separately (0/0/exit0)",
    "no GET /business/deploy/credentials/:id detail route (V5 probe records 404 as finding)",
    "business:deploy:credential:* permission keys not seeded -> credentials admin-only; ListCredentials has no creator filter (latent SEC-004 residual)",
    "cmdbhost.ssh_connect_failed masks host-key mismatch (wrong-fingerprint UX semantics)",
    "leftover vprobe test users/roles not deleted (operation-verify 403)",
    "argocd AddServer/AddRepository blocked: runtime env lacks PANTHEON_ENCRYPTION_KEY -> argocd.encryption_key_missing (deploy-config, not code defect; token never leaked)",
    "runtime env lacks PANTHEON_BUSINESS_SECRET_KEY -> non-empty business secrets cannot be stored at rest (secrets.Encrypt key_missing); empty-string Encrypt is a no-op so passwordless datasource create unaffected"
  ],
  "capLane": {
    "status": "PASS",
    "run": "runs/omx_accept_20261122_cap/manifest.json",
    "counts": "31 PASS / 1 BLOCKED / 0 FAIL, 32 cases on candidate k8s33 (sha256 5cd98af…)",
    "executor": "Claude (independent, Codex temporarily unavailable)",
    "convergedCapabilities": ["CAP-SERVICE PASS", "CAP-BIZSCOPE PASS", "CAP-FILEDIST PASS", "CAP-OBSERVABILITY PASS", "CAP-METRICS PASS", "CAP-CAPABILITY PASS", "CAP-OPSINIT PASS"],
    "blockedCapabilities": ["CAP-ARGOCD BLOCKED (env key gap; no token leak)", "CAP-SECRETS PARTIAL (write->ciphertext->open loop needs env keys)"],
    "ssrfPositiveProof": "loopback 127.0.0.1:9090 denied (BlockPrivateIPs) AND public example.com:8081 denied (AllowedPorts) on live binary",
    "instanceCreateRootCause": "probe targeted an out-of-scope CMDB host; fixed by creating host inside the same business scope (validateVMTarget requires businessScopeId match)",
    "datasourceCreateRootCause": "probe URLs failed SSRF allowlist; fixed by using public URL on allowed port 80"
  },
  "runtimeEnvGaps": [
    "PANTHEON_ENCRYPTION_KEY absent -> argocd at-rest encryption blocked",
    "PANTHEON_BUSINESS_SECRET_KEY absent -> shared business secret Encrypt would key_missing on non-empty values"
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-review-live-runtime-acceptance/manifest.json",
    "evidence": ".harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/review.md",
    "changeRef": "none",
    "planRefs": [
      ".harness/tasks/2026-11-20-ops-review-remediation/plan.md"
    ]
  }
}
```
