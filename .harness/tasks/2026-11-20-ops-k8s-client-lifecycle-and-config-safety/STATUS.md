# K8s 凭据导入、连接轮换与请求边界

- Status: in-progress
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

## Verification

- `go test -count=1 -p 1 -timeout 20m ./modules/business/k8s/...` → 11/11 包 ok。
- `npm run type-check` / `check:password-toggle` / `check:ui-contract` / 目标 eslint → PASS。
- 证据：`../evidence/2026-11-20-ops-k8s-client-lifecycle-and-config-safety/`（commands.json / summary.md / review.md + PNG/JSON）。

## Known Gaps

1. V8 真实集群凭据轮换未在本候选重跑。
2. `npm run test:smoke:business:k8s` 未执行（前端 dev server 被 harness 因低内存回收，按指示不自行重启）。
3. 断网/取消 provider 超时仅有单测级覆盖。

## Next Atomic Action

真实集群 V8 轮换 + k8s 浏览器 smoke 重跑后，补齐 V6/V8 验收并同步 review linkage。
