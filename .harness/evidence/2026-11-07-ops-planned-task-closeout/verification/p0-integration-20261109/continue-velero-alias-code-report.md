## Code Review Summary

**CODE: REQUEST CHANGES**，仅限 Root 的 staged alias recovery。  
**Files Reviewed:** 5 · **Total Issues:** 2  
CRITICAL: 0 · HIGH: 1 · MEDIUM: 1 · LOW: 0

### Issues

**[HIGH] 重编译 inspector 丢失 captured-byte 读取保护**  
File: [continue-velero-alias-remote.py:317](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-remote.py:317)

直接编译原始函数文本丢失已有 `CapturedSyntax` 转换。函数随后重开 provider 源码并 `eval` 其中的表达式，执行的内容没有绑定到此前捕获的字节。

**复现：**独立测试运行当前 inspector 的实际重读/eval 语句块；替换临时源码后，未校验表达式产生文件副作用且 argv 检查通过。保留原转换时，先以 `captured-sealed-bytes-drift` 拒绝。

**Fix:** 保留 captured-read 转换，或直接使用已加载模块的封存源码缓冲区。补充校验后源码变化的回归覆盖；事后重新哈希不能阻止已经发生的副作用。

**[MEDIUM] 前五对象没有完整 spec 基线约束**  
File: [continue-velero-alias-remote.py:161](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-velero-alias-remote.py:161)

完整 policy 比较仅覆盖 `entries[5:]`。前五对象沿用部分断言，允许未审查的 Service spec 被纳入新的 diagnostic 基线。

**复现：**改变 Service 的 `sessionAffinity` 或加入 `internalTrafficPolicy=Local`，原五对象 adapter 仍返回与原基线完全相同的证明。这不满足 exact22 spec 漂移必须在 tag 前拒绝的要求。

**Fix:** 将前五对象的完整规格和明确允许的默认值绑定到不可变审查证据，在任一 tag 前拒绝漂移；保留原 canonical inventory 字节。

### Validation

- 原 27 项测试通过；独立 counterexample 测试 2 项通过，共 **29 项，零 skips**。
- 五个语法编译通过；重新运行真实 **Pyright 1.1.414 LSP：零错误、零警告**。
- 五份源码的审查前后 SHA 与 sealed request 一致；request SHA 为 `f053fb6dda5d44af557eabdee4ad04b676da47591d7add0e210a442425a81d70`。
- 原 v8 失败 `f27d06fcc7160e534d3a2d8dde304367620bc7bf1f4e66148476d3b405de2601` 保留。

执行命令：

```powershell
python -B .local/ops-acceptance/continue-velero-alias-test.py
python -B .local/ops-acceptance/continue-velero-alias-root.py check-inputs --request-sha256 f053fb6dda5d44af557eabdee4ad04b676da47591d7add0e210a442425a81d70
python -B .local/ops-acceptance/python-lsp-check.py independent-code-review-alias continue-velero-alias-contract.py continue-velero-alias-remote.py continue-velero-alias-root.py continue-velero-alias-generate.py continue-velero-alias-test.py
python -B .harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-independent-code-review-test.py
```

[审查凭证与源码 pins](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-velero-alias-independent-code-review-verification.json)。`lsp_diagnostics`/`ast_grep_search` API 未提供，使用真实 LSP 与 Python AST 检查替代。

Stage 1 和 captured-source 根因门禁未通过，停止正常质量批准。未执行远端阶段、读取凭据、修改作者源码或重跑历史 158 项测试；未形成 ARCH、P0、public HTTP 或 audit composition 批准。
