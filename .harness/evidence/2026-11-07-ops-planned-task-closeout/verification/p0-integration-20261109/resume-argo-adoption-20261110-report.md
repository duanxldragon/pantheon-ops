Generator 范围已完成并验证：

- 已接入 capability 协商、原生 reviewed Sync/Rollback、私有不可变 history ID；保留授权、审计与 Unknown 恢复，无 fallback。
- MySQL-required CGO race：181 个非分组测试通过；vet/build/gopls/格式检查通过；部署 helper 离线测试 7/7。两项 live 测试未执行。
- 公共 API 不变，31 项源文件与产物哈希稳定。

[完整哈希、验证和 root 部署/rollback 说明](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/argo-adoption-20261110/summary.md)。

产物 seal：

```text
08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec
```

root 审查后部署：

```powershell
python -B .local/ops-acceptance/argo-adoption-root-run.py install --seal 08ba8f44013b10f92c533a1f96ed55a7346b0f6e17a968b8c313fdf1ab4d79ec
```

未执行远端操作。root 仍需完成独立审查、真实 IAM/controller/Service/rollback/丢失回复验收及完整门禁；旧 PATCH-only live relay 需更新。未授予 P0 approval。
