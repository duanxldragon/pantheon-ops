v3 时效补丁已实现并封存：每次 S3 PUT/DELETE 前检查 predecessor；过期后拒绝写入或补偿删除，失败收据保留准确探针路径、成功 PUT 和不确定结果，只允许检查。

仅新增指定五个源码文件和 v3 父任务证据。原有保护项全部保持一致。

| `continue-velero-alias-v3-*.py` | SHA256                                                             |
| ------------------------------- | ------------------------------------------------------------------ |
| contract                        | `bfac98f9c72ae3cbe3ff4bc3a9d97fe3d3c377f471e1ab851586c945772a4ca6` |
| remote                          | `5e443950e04855771f6a36ad67041439a34902be9eb66dc0a960891e4dcb951b` |
| root                            | `d3604678f345f258157a572add4fa8d8d367f30f27d66082e0e06589a1285897` |
| generate                        | `a1c3738d2ebff357e18f20866f5ca9d84131d1f178332e321ef81ac2588ad88f` |
| test                            | `da68651673b5a7281a152ecd0dce5d6138f1ccd6a9cd5efa70eb513585a8f176` |

Request SHA256：`96fef023020ad1f8e880e01f04612acfa0480bd3009c009063fa8756703e3b07`。实际只生成一次。

验证结果：

- 两项 freshness 断言在原 v2 上实际 RED，在 v3 上 GREEN。
- 原 230 项通过；新增 8 项通过；v3 当前 53 项、完整唯一回归 238 项，零跳过。四个原 v1 反例保留并重现。
- 五文件编译通过；真实 Pyright 1.1.414 LSP：零错误、零警告，无抑制。
- 237 个保护项未变，包含继承 223 项及指定 v2 源码、请求、证据和独立审查。测试、LSP、请求和封存源码哈希一致。

[548 字完整交接及证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-v3-handoff.md) 包含哈希校验及 `Inspect-Receipt` 定义。Dispatcher 在两份独立最终审批通过后，分阶段执行：

```powershell
$aTool = '.local/ops-acceptance/continue-velero-alias-v3-root.py'
$aSHA = '96fef023020ad1f8e880e01f04612acfa0480bd3009c009063fa8756703e3b07'
python -B $aTool check-inputs --request-sha256 $aSHA

$d = python -B $aTool diagnostic --request-sha256 $aSHA | ConvertFrom-Json
Inspect-Receipt $d

$r = python -B $aTool repair --request-sha256 $aSHA --receipt $d.artifact --sha256 $d.sha256 | ConvertFrom-Json
Inspect-Receipt $r

$v = python -B $aTool validate --request-sha256 $aSHA --receipt $r.artifact --sha256 $r.sha256 | ConvertFrom-Json
Inspect-Receipt $v

$c = python -B $aTool cleanup --request-sha256 $aSHA --receipt $v.artifact --sha256 $v.sha256 | ConvertFrom-Json
Inspect-Receipt $c
```

每一步必须检查完整成功收据，才能使用其明确名称和哈希继续。失败时将 `$failed` 绑定到准确返回结果：

```powershell
python -B $aTool inspect-failed --request-sha256 $aSHA --receipt $failed.artifact --sha256 $failed.sha256
```

**停止点已达到：本地实现及验证完成。独立 v3 CODE/ARCH 和实际运行验收仍待 Dispatcher；执行门禁保持阻断。未执行远程操作，未自批源码或关闭 P0。**
