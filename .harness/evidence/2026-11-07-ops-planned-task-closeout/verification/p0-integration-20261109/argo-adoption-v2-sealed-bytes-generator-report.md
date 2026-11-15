R1 已完成本地修复并验证；root 独立审查和部署仍待执行。

## Changes Made

- 新增 [v2 runner](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/argo-adoption-v2-root-run.py:126)：seal/helper 各捕获一次，校验和消费同一缓冲区。
- 新增确定性回归测试、验证工具和独立新 seal；原始 31 文件封存保持完整。

## Verification

- `argo-adoption-v2-verify.py --label final2` → **exit 0**。
- **20/20** 新回归、**7/7** 原 helper 测试通过。
- AST、内存编译、差异检查通过；两个重读变异均被拦截。
- 67 个原资产哈希稳定，实际本地 preflight 通过。

## Assumptions / Notes

未读取凭据、执行远程操作或修改业务源码。未重跑完整 race，未声明服务器成功或独立批准。

## Summary

[详细交接及 root 精确命令](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/argo-adoption-v2-sealed-bytes-handoff.md) 已保存。

新候选 seal SHA256：`fb57123370e5c99d0149faa32237ff1b5462ad70422e4093f446938f0a875d03`。
