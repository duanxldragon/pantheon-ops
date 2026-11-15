**HIGH — 已选文件的复制读仍可绕过 capture 校验。ARCH 决定：`REQUESTCHANGES`，架构状态：`BLOCK`。**

## Analysis

[candidateAssemblyFileSystem:1083](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:1083) 将 `readStableFile(source)` 返回的 Buffer 直接写入目标。[readStableFile:639](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:639) 校验路径、fd 和时间戳，但没有将返回字节与 capture 哈希绑定；已有的绑定检查位于 [frozenInputReader:769](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:769)，目前用于 replay。

独立反例仅替换一次选定源文件 fd 的同长度读取结果，保留真实源字节和元数据：

- 业务夹具源仍为 `package cmdb`，装配结果变为 `package fake`。
- 元数据源仍为 `ORIGINAL evidence`，装配结果变为 `INJECTED evidence`。
- 两者均发布 `sourceStable: true` receipt，且 capture 哈希仍等于真实源哈希。

[可运行反例:24](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-fix-independent-arch-20261110-req9b6a0bec-r1/confirm-copy.test.mjs:24) · [实际字节及哈希证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-fix-independent-arch-20261110-req9b6a0bec-r1/confirmed-copy-mismatch.json)

## Root Cause

源稳定性与复制结果分别记录，缺少二者的字节关联；[receipt:226](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-candidate.mjs:226) 仍标记稳定。前后重新捕获未变化的源文件，无法认证此前返回的替换 Buffer。

## Recommendations

1. **中等工作量：**在复制写入前，将实际 Buffer 的长度及 SHA256 与冻结 Ops inventory／Base descriptor 对照；复用现有绑定逻辑，保留归档边界和目录守卫。
2. **低工作量：**纳入两个反例，要求拒绝 receipt，再独立复审。

| 方案           | 收益                         | 代价                       |
| -------------- | ---------------------------- | -------------------------- |
| 复制入口校验   | 在 import 重写前认证原始字节 | 增加一次摘要计算           |
| 装配后关联清单 | 能检测最终差异               | 需要额外处理重写及生成关系 |

## Validation / Stop Condition

修复回归 **11/11**；新增独立测试 **5/7**；新夹具确认 **0/2**，两个拒绝断言再次失败，零 skips。目录选择及五种 locale 的排序、Base 保留和双 replay 检查通过。详见 [完整报告](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-fix-independent-arch-20261110-req9b6a0bec-r1/review.md)。

8 个 AFTER 实际哈希均匹配且审查期间稳定；BEFORE 值与历史 request、preservation 记录一致，未独立重哈希历史 BEFORE 源字节。作者 198/198 日志哈希已核验。`checkJs=false` 不证明完整类型安全。

已达到诊断停止条件。真实装配门禁继续阻断；完整 O4 C–F、P0、运行态及正式消费保持开放。

```json
{
  "lane": "ARCH",
  "decision": "REQUESTCHANGES",
  "architecturalStatus": "BLOCK",
  "independent": true,
  "subjectSHA256": "9b6a0bec4daea400f8c99144934bbfcd87f0659db65e0993e8c91df085dd7786",
  "requestSHA256": "9b6a0bec4daea400f8c99144934bbfcd87f0659db65e0993e8c91df085dd7786",
  "sourceHashes": {
    ".gitignore": "053c2f12d5dfb20b171a6f0b17ec9c2694b67fdf2bc67cda13aecf3544cffd78",
    "scripts/business-overlay/foundation-candidate.mjs": "17687364e7cc0420164707a9d92b33971f83d755efe57d61ace7e166a7105033",
    "scripts/business-overlay/rebuild-from-candidate.mjs": "d92eb077c1b67cb3a0b291be24958db0533ecfac8c3c586930f0d9ce2468f104",
    "scripts/business-overlay/rebuild-from-base.mjs": "94ce67fc97fd358200835dc81df769d9dcbe7865e32b74981dc84ecdb5004e4f",
    "scripts/business-overlay/foundation-candidate.test.mjs": "c9be950b627e86c92a6e0e42d1ef1886119c4bf050fe62aa3f1b894dd8d76ff2",
    "scripts/business-overlay/candidate-ownership.test.mjs": "30f4c185051b232d414b4867c7a9b35601d335405d8477b26dcb236be09f6062",
    "scripts/business-overlay/check-shared-drift.mjs": "a6ee2b769c1d27167c7f426ad913981c28b99f462054342e826dc194a1c4302b",
    "tests/scripts/business-overlay/check-shared-drift.test.mjs": "b79af64de465f3b1e2efda7e14d595fb2ba24ee46bdb36b04f2e79824338be36"
  }
}
```
