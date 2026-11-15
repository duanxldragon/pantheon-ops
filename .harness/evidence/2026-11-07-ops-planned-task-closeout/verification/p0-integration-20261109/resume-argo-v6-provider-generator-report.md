## Changes Made

- [Provider 补丁](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/provider/reviewed-actions-v1.patch)：新增独立认证路径，复用原生操作构造，以 UID/resourceVersion 单次条件更新提交，冲突不重试。
- [README](/D:/workspace/go/pantheon-platform/pantheon-ops/backend/modules/business/cicd/argocd/provider/README.md)：记录协议、兼容限制和升级成本。

## Verification

- 定向 CGO race：44 个非分组用例通过，零跳过。
- 相关包 vet/build 通过，源码哈希稳定。
- 8 个变更路径；4,820 个其他上游文件字节一致；补丁正向应用、反向校验通过。

## Assumptions / Notes

[详细交接](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/resume-argo-v6-provider-handoff.md)已落盘。独立评审、服务端镜像、部署、Ops 接入和真实终态验收仍由 root 收口。

## Summary

限定的 provider 源码与本地验证交付完成，可进入独立评审；完整 P0 尚未关闭。
