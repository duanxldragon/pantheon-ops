v4 诊断材料已准备并验证，真实终态问题尚未关闭。

## Changes Made

- live 测试增加清理前的独立状态观察，保留原终态判据。
- 准备哈希绑定的 v4 Application/Service 诊断；v3 证据全部保留。

## Verification

- required MySQL、MinGW/CGO race：159 个测试事件通过，零失败；两个 live 测试明确跳过。
- vet、build、gopls、格式及脱敏检查通过。[验证证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v4-evidence/resume-argo-v4-verify-1-result.json)

## Assumptions / Notes

v3 的 PATCH 200 后，GET 均缺少 operation/state；现有证据无法确定 mutation 未持久化还是 provider 未处理，因此未臆造产品修复。cleanup 0 不代表终态成功；完整 Base 当前会话、审计和 HTTP wiring 仍开放。

## Summary

按限定未执行远程。root 下一步执行：

`python .local/ops-acceptance/resume-argo-v4-live.py --phase run`

[精确请求与哈希](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v4-evidence/resume-argo-v4-phase-request.json) · [诊断与交接说明](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/resume-argo-v4-evidence/resume-argo-v4-summary.md)
