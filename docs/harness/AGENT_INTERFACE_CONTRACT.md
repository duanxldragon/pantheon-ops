---
title: Agent Interface Contract
doc_type: Contract
layer: method
status: Active
updated_at: 2026-06-08
---

# Agent Interface Contract

本文定义任何 agent 或人工工程师进入仓库时的输入、输出和行为协议。工具可以不同，协议必须一致。

## 1. Agent 必须提供的能力

合格执行者必须能做到：

- 读取仓库文档和源码。
- 判断任务归属层。
- 遵守当前仓库声明的层级、模块和继承边界。
- 修改文件时保持最小范围。
- 运行或明确说明无法运行的验证命令。
- 保存或摘要验证证据。
- 用 findings-first 格式做 review 或交付说明。

不能做到以上要求的工具，只能用于辅助阅读、草拟文案或局部建议，不能独立执行开发任务。

## 2. 任务开始输出

非 trivial 任务开始前，执行者必须给出：

```text
Primary layer:
Dependency layers:
Contract anchors:
Expected touched areas:
Verification plan:
Human gates:
```

示例：

```text
Primary layer: system/iam
Dependency layers: platform
Contract anchors:
- docs/contracts/<domain-contract>.md
- docs/designs/<feature-design>.md
Expected touched areas:
- backend/modules/system/iam/role
- frontend/src/modules/system/role
Verification plan:
- go test ./backend/modules/system/iam/...
- cd frontend && npm run check:menu-contract && npm run build
Human gates:
- none unless permission seed changes
```

## 3. 修改前检查

执行者修改代码或文档前必须判断是否影响：

- schema / migration / seed
- API contract
- permission
- menu
- i18n
- audit
- security
- shared/base inheritance
- generated files
- CI or deploy gates

如果有影响，必须把对应文档和验证命令加入任务范围。

## 4. 上下文读取规则

执行者不得把“没有读取相关合同”作为跳过约束的理由。

读取规则：

- 只读与任务有关的合同、设计、验收文档。
- 不复制大段文档到聊天上下文。
- 对长文档只摘出与当前任务相关的硬约束。
- 如果发现文档冲突，暂停并说明冲突，不自行选择有利版本。

## 5. 文件修改规则

执行者必须：

- 遵守已有目录结构和模块边界。
- 不改无关文件。
- 不重排大文件以制造噪声 diff。
- 不删除用户未要求删除的文件。
- 不覆盖用户已有未提交改动。
- 不把共享规则复制到派生仓库。

如果当前仓库有 foundation/overlay 关系，发现共享层 bug 时，默认先判断是否应在 foundation 侧修复。

## 6. 验证输出

任务完成前必须输出：

```text
Commands run:
- <command>: passed / failed / not run

Evidence:
- <path or summary>

Known gaps:
- <gap or none>
```

如果未运行验证，必须说明具体原因：

- 依赖未安装
- 环境缺少服务
- 需要网络或权限
- 当前任务纯文档
- 用户明确要求不运行

## 7. Review 输出

Review 必须 findings first：

```text
[P0|P1|P2] (confidence: N/10) file:line - issue
Impact:
Fix:
Verification:
```

没有发现问题时，必须说明：

```text
No P0/P1/P2 findings found.
Residual risk:
Verification checked:
```

## 8. 禁止行为

执行者不得：

- 因为某个工具不支持某能力而跳过仓库协议。
- 把聊天记录当作唯一需求来源。
- 用“看起来没问题”替代验证证据。
- 在派生仓库重写共享合同。
- 把权限、菜单、i18n、审计视为后续补丁。
- 为了让测试通过而降低测试覆盖或删除检查。
