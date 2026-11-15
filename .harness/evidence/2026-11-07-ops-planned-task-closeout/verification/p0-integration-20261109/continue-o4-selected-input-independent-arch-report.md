## Summary

**REQUEST CHANGES；ARCH BLOCK。** 本次 delta 仍有实际输入采用阻塞，并存在可复现的 locale 派生一致性缺陷。[独立报告](/D:/workspace/go/pantheon-platform/pantheon-ops/.harness/evidence/2026-11-07-ops-planned-task-closeout/verification/p0-integration-20261109/continue-o4-selected-input-independent-arch-review-20261110-01.md)

## Analysis

- **P1：实际 Ops 输入仍无法冻结。** 两个指定忽略根已退出 Git 选择，但其余声明输入包含 **162 个嵌套 Git 仓库目录记录、13 个超长路径**；实际调用 selector 抛出 `Unsafe declared Ops input path`。解析器直接校验 Git 记录，未处理目录记录形态：[rebuild-from-base.mjs:27](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:27)。现有长度与路径门禁正确拒绝这些输入：[foundation-candidate.mjs:675](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/foundation-candidate.mjs:675)。这是采用阻塞，未发现安全绕过。[实际证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-selected-input-independent-arch-20261110-01/actual-selector-summary.json)

- **P2：assembly 与 canonical replay 的 locale 优先级不同。** Assembly 按原生目录顺序合并：[rebuild-from-base.mjs:358](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:358)；replay 使用排序后的 selected files：[rebuild-from-base.mjs:724](/D:/workspace/go/pantheon-platform/pantheon-ops/scripts/business-overlay/rebuild-from-base.mjs:724)。Windows 独立夹具中，`a/en-US.json` 与 `Z/en-US.json` 含同一 key，assembly 得到 `Z`，replay 得到 `a`。源码哈希、stamps、祖先身份和选择均未变，仍拒绝候选；无成功 receipt。[反例证据](/D:/workspace/go/pantheon-platform/pantheon-ops/.local/ops-acceptance/continue-o4-selected-input-independent-arch-20261110-01/adversarial.json)

## Root Cause

Git 枚举被假定为规则文件叶子列表，而实际输入含非叶记录；locale 派生同时采用了两套顺序合同。

## Recommendations

1. **中等工作量：** 明确剩余非叶 Git 记录与超长输入的处理策略，保留原资产及现有门禁，再提交独立审查。不得静默丢弃记录或扩大 evidence 忽略范围。
2. **小工作量：** Assembly 与 replay 使用同一份确定性 locale 叶子列表；补充 `a/Z` 重复 key 回归及未选 locale 排除验证。

## Architectural Status

`BLOCK`，仅针对本次 consumer 工具增量。

## Trade-offs

| 方案                 | 优点                | 代价                      |
| -------------------- | ------------------- | ------------------------- |
| 明确拒绝非叶记录     | 保留安全边界        | 当前采用继续阻塞          |
| 确定性展开许可输入   | 可保留合法 evidence | 需重审枚举与冻结一致性    |
| 统一 locale 合并顺序 | 最小修复            | 重复 key 的优先级可能改变 |

独立验证：聚焦 **29/29**、drift **27/27**，均零跳过；原始读取/复制期间 edit-revert 与隐式 go.sum 修改均被拒绝。请求及 8 个源码哈希审查前后匹配；原硬链接、缓存元数据及 sealed descriptor/artifact pin 保留。未运行真实装配；无 runtime、P0 或源码全域批准。停止条件已满足：诊断有实证，等待修正后重审。

```json
{
  "lane": "ARCH",
  "decision": "REQUEST CHANGES",
  "independent": true,
  "subjectSHA256": "8a866ac9c1d608558ee3d1e04805cd7f8a5e0a444b8fab2fc1dfe8ac7bd84a82",
  "requestSHA256": "8a866ac9c1d608558ee3d1e04805cd7f8a5e0a444b8fab2fc1dfe8ac7bd84a82",
  "architecturalStatus": "BLOCK",
  "sourceHashes": {
    ".gitignore": "04b7bb31311f121d90dbd72fdf8cff3ce6127ee73b6b77469bbcf147a7e7df12",
    "scripts/business-overlay/foundation-candidate.mjs": "02deaf187133b267622359e5756100262f99c48b031f76db47716157c8c974f1",
    "scripts/business-overlay/rebuild-from-candidate.mjs": "b06fa5a04595f9a245920814f3ba0e31b5f3a9009061c249c62ef69fd8030f06",
    "scripts/business-overlay/rebuild-from-base.mjs": "24e7e834622d7e61d240a25aedfd5c914cee86ea9780dc7280a31ce55ee06c76",
    "scripts/business-overlay/foundation-candidate.test.mjs": "6f2cb9a366de1fdb71838ccc89ccfc76af39d68570aafde06cbe660dfd9b8894",
    "scripts/business-overlay/candidate-ownership.test.mjs": "30f4c185051b232d414b4867c7a9b35601d335405d8477b26dcb236be09f6062",
    "scripts/business-overlay/check-shared-drift.mjs": "a6ee2b769c1d27167c7f426ad913981c28b99f462054342e826dc194a1c4302b",
    "tests/scripts/business-overlay/check-shared-drift.test.mjs": "b79af64de465f3b1e2efda7e14d595fb2ba24ee46bdb36b04f2e79824338be36"
  }
}
```
