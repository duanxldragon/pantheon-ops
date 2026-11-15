## Code Review Summary

**REQUEST CHANGES：发现 1 项 HIGH，v11 工具输入闭包尚不完整。**

**Files Reviewed:** 11 个源码文件，另核验 2 个二进制及关联证据。  
**Severity:** CRITICAL 0 / HIGH 1 / MEDIUM 0 / LOW 0。

### Issue

**[HIGH] 所选 MinGW 编译器的 DLL 依赖未固定，漂移可通过请求校验。**

File: [contract.py:246](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v11-contract.py:246)

工具发现仅将所选 `cc1.exe` 等程序加入清单。独立 GCC 选择探针确认实际 C 编译使用该程序；模块枚举确认它加载了 **8 个未出现在 `files` 或 `toolFiles` 中的 MinGW DLL**，包括 `D:/msys64/mingw64/bin/libgmp-10.dll`。

**Repro：**复制该 DLL 并改变首字节，仅将哈希读取重定向到这个本地副本。重新运行实际依赖发现后：

- 清单保持相等，该 DLL 的哈希函数调用次数为 **0**。
- [validate_request:342](D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o2-live-v11-contract.py:342)仍接受请求。
- 外部原件保持不变。

因此，现有前后稳定检查无法证明声明的完整工具输入绑定。已封存二进制的哈希仍有效，但不能补足遗漏的构建输入证据。

**Fix：**为实际选中的编译器及链接器发现并固定其解析到的 MinGW DLL 依赖，纳入前后字节校验；加入该真实依赖漂移必须被拒绝的回归。保留 v10/v11 原请求和证据，通过新继任候选完成修复。

### Scoped verification

- 指定 v11 请求 SHA256 匹配。
- 12 个候选、88 个证据、232 个历史保护项、65 个 owner pins 匹配；**6,256 个相关文件前后哈希无变化**。
- **5,532 个归档成员**全部匹配。
- 重跑五组实际 selector；独立缓存迁移的三个生成 Cgo 路径经明确归一化后，字节及其余清单一致。
- `runtime/race.go`、`race_windows.syso` 的本地副本漂移均被拒绝，v10 race selector 缺口已修复。
- 三个 Go 文件独立原生 `gopls check` 无诊断；八个 Python 文件 AST 检查通过。
- 原矩阵及关键 runner 函数体保持等价；无授权执行提前拒绝，旧 owner receipts 不能授权 v11。未发现新增失败掩盖分支。

[独立复核证据](D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-o2-v11-independent-code-20261110-02-evidence.json)

CODE 审查已完成，修复上述闭包缺口后需重新提交候选。未写入授权或真实 attempt，未访问凭据或远端。真实 VM/cold-restart、public Base identity/IAM/audit、HTTP/browser/SSE 和整体 P0 验收仍开放。

```json
{
  "lane": "CODE",
  "decision": "REQUEST CHANGES",
  "independent": true,
  "subjectSHA256": "ffbc6666c896de2b00ec2eb799aa96e41e9b4d3bb0e5a9417d049db361649e51",
  "requestSHA256": "ffbc6666c896de2b00ec2eb799aa96e41e9b4d3bb0e5a9417d049db361649e51"
}
```
