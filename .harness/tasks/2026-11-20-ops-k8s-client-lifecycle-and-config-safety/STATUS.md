# K8s 凭据导入、连接轮换与请求边界

- Status: completed（证据与 review 完成；维护者最终视觉/功能验收 Human Gate 仍开放）
- Updated: 2026-11-24
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: K8s 客户端负责人
- Priority: P1
- Activation: core
- Dependencies: T01, T04

## Implemented

- 连接池 key=`{clusterID, sha256(kubeconfig)}`、atomic lastUsed + GC、`Evict(clusterID)`；`TestPool*` 6 项 PASS。
- kubeconfig 只接受嵌入凭据；拒绝 Exec/AuthProvider/TokenFile/Client cert-key/CA file；SSRF 端口 443/6443/8443/16443。
- `restConfig.QPS=20/Burst=40`（`TestRESTConfigPinsClientThrottle`）。
- Pods `limit/continue` 分页 + `continueToken` 回传。
- 敏感输入默认掩码：新增 `SecretTextArea`，接入 k8s 集群表单 / filedist / CMDB 主机详情；`-webkit-text-security` 探针 `disc`→`none` + 4 张截图。
- 节点列表读取使用请求上下文并保留 30s 上界：客户端断开即取消 provider 调用，不再等到超时。

## Verification

- `go test -count=1 -p 1 -timeout 20m ./modules/business/k8s/...` → 11/11 包 ok。
- `npm run type-check` / `check:password-toggle` / `check:ui-contract` / 目标 eslint → PASS。
- `test:smoke:business:k8s` 由 PR #207 CI Smoke Sanity（run 45308078162，17m30s）执行并绿（该套件覆盖 K8s 页面路由，不覆盖凭据/kubeconfig 表单）。
- **真实集群轮换 PASS**：main `697c0771` 候选后端在 `115.190.251.143` K3s 执行 k8s-live `omx-accept-20261121-k8s38`，17/17 PASS（k13 version 1→坏 token→恢复）。
- **断网/取消 runtime PASS**（2026-11-24）：候选后端（本包 `GetNodes` 上下文传播修复）在真实 K3s 上先建立节点列表基线；随后用来源限定的临时 `INPUT` DROP 规则在 provider 侧制造网络分区，请求挂起；客户端 1s 后断开，后端 handler 于 `1010ms` 结束（基线 78.5ms、恢复 47.92ms），规则删除后 provider 流量恢复（`t06-cancel-20261124-135304.json`）。
- 证据：`../evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/`（commands.json / summary.md / review.md + PNG/JSON + live run）。

## Known Gaps

- 维护者最终视觉/功能验收（Human Gate）。
- 无独立 review：Codex 不可用，本候选 review 由 primary agent 完成并在 review.md 记录。

## Next Atomic Action

维护者最终验收；Base `keyMatch2` 静态路由守卫按 T04 上游依赖跟踪。
