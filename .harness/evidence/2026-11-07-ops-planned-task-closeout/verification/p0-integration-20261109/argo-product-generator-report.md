模块内修复已完成并通过本地验证；产品闭环仍有明确依赖。

## Changes Made

- [动作与恢复](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_action.go:39)：精确关联终态、持久化未知结果、防止重发、提交前重读凭证配置。
- [预览与授权](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/argocd_preview.go:73)：绑定 actor/版本/配置，隐藏清单值，缺少当前授权接口时拒绝执行。

## Verification

- MinGW、`CGO_ENABLED=1`： scoped race **91 个测试事件通过**。
- scoped `go vet`、`go build`、格式及 diff 检查通过。
- [命令、结果与稳定源码哈希](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-product-verification-result.json)。

## Assumptions / Notes

Base 当前 actor/DataScope 授权能力及接线仍缺失，受保护的生产操作会拒绝执行。真实 provider 新验收尚未运行；原 28 项仍是 provider-only。

Root 执行请求及精确脚本哈希已保存于 [phase request](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-product-phase-request.json)：

```powershell
python .local/ops-acceptance/argo-product-live.py --phase run
```

## Summary

[完整交接报告](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-product-generator-report.md) 列明授权、运行态、UI 和独立评审缺口；未作 O3/D1/P0 整体验收声明。
