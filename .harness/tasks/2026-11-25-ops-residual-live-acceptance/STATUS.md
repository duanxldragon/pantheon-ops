# 真实环境残余验收（多租户、受限角色、SA 轮换、Ingress、Velero、race Redis）

- Status: planned
- Updated: 2026-11-25
- Baseline: 444ce8e（报告）/ a2d69d4a（规划）；候选以开工时 main 为准
- Owner: 真实验收负责人
- Priority: P0
- Activation: core
- Dependencies: T05、T11、T12、T13、T14

## 来源

由 T16 `2026-11-20-ops-review-live-runtime-acceptance` 关闭时拆出的环境类残余。T16 已在同候选闭合 deploy 7/7、K8s 17/17（k8s38）、CAP 31/32、V-probe 19/19、T03 角色矩阵 18/18；以下项当时不可得：

- V4 双租户（`tenant_mode=multi` 未配置）
- V7 低权限 SSH restart/cancel 用户
- V8 SA 凭证撤销轮换
- Ingress provider 变更/快照
- Velero schedule/restore
- 完整 `-race` 所需测试 Redis `127.0.0.1:26316`
- `GAP-ARGOCD-ENVKEY`（运行环境缺 `PANTHEON_ENCRYPTION_KEY`）
- WebSocket server 侧资源释放观察

## Next Atomic Action

核对环境可用性：多租户配置、低权限 SSH 用户、测试 Redis、Ingress provider 与 Velero。具备一项执行一项，每项写候选绑定回执。

## Evidence

尚未执行。复用 T16 的 lane 脚本与证据格式。

## Known Gaps

- 环境未提供前全部项保持 gap，不做推断性关闭。
- 全新安装/升级/回滚与最终交付收口归 T17。
