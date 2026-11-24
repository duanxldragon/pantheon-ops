# T06 K8s 凭据导入、连接轮换与请求边界 — 实施摘要

- Task: `2026-11-20-ops-k8s-client-lifecycle-and-config-safety`
- 日期: 2026-11-24
- 候选: pantheon-ops worktree 未提交改动；foundation `pantheon-base-v0.18.9`
- 状态: **implemented + 代码级验证 + 凭据输入渲染证据 + CI k8s 页面 smoke；真实集群轮换仍为显式 gap**
- 交付层级: L2（自审见 `review.md`，独立复核 gap 已登记）

## 交付内容

| 验收项 | 实现 | 验证 |
|---|---|---|
| 连接缓存绑定集群/凭据版本，更新/轮换/删除触发 Evict，并发不复活旧 client | `pkg/client_pool.go`：key=`{clusterID, sha256(kubeconfig)}`，atomic lastUsed + GC，`Evict(clusterID)` | `TestPool*` 6 项 PASS（含并发同凭据去重、异凭据隔离、GC 保留活跃） |
| 恶意 kubeconfig 在建立连接前拒绝，无本地执行/读取副作用 | `pkg/client_factory.go` `validateEmbeddedCredentials` 拒绝 Exec/AuthProvider/TokenFile/Client cert-key/CA file；SSRF 端口 443/6443/8443/16443 | `client_factory_test.go` 负例矩阵 PASS |
| 请求边界：明确 QPS/Burst，Pods 分页/有界，敏感输入默认掩码 | `restConfig.QPS=20/Burst=40`；`GetPods` 增加 `limit/continue` 分页并回传 continueToken；新增 `SecretTextArea` 掩码组件并接入 k8s 集群表单、filedist、CMDB 主机 | `TestRESTConfigPinsClientThrottle` PASS；`-webkit-text-security` 运行时探针 `disc`→`none` + 4 张截图 |
| CMDB 主机 SSH 私钥默认掩码 + 可控显隐 | `CmdbHostDetail.tsx` 改用 `SecretTextArea` | `t06-cmdb-mask-runtime.json` + 截图 |

## 验证结果

- `go test -count=1 -p 1 -timeout 20m ./modules/business/k8s/...` → **11/11 包 ok**
- `go test ./modules/business/k8s/pkg/`（含池与工厂全部）→ ok
- `npm run type-check` / `check:password-toggle` / `check:ui-contract` / 目标 eslint → 全 0 finding
- 渲染证据：`k8s-cluster-kubeconfig-{masked,revealed}.png`、`cmdb-host-ssh-key-{masked,revealed}.png`、`t06-*-mask-runtime.json`

## 显式 gap

1. V8 真实集群凭据轮换未在本候选重跑（缓存身份逻辑已由单测锁定）。
2. 断网/取消的 provider 超时仅有单测级覆盖，未做网络分区实机探针。

补充：`npm run test:smoke:business:k8s` 已由 PR #207 CI Smoke Sanity（run 45308078162，17m30s）在隔离后端执行并通过；该套件不覆盖凭据表单，掩码证据仍为定向 Playwright 探针。
