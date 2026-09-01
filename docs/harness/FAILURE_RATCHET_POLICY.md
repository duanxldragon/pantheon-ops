---
title: Failure Ratchet Policy
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-08
---

# Failure Ratchet Policy

English version: [FAILURE_RATCHET_POLICY.en.md](./FAILURE_RATCHET_POLICY.en.md)

本文定义 repeated failure 应如何从一次性问题，升级为仓库级方法资产。

## 1. 目标

Harness 不应只记录“这里出过错”，还应定义“同类错误再次出现时，下一步要把控制面升级到哪里”。

ratchet loop 的目标是：

- 让重复错误不再只存在于聊天记录里
- 让经验从人工提醒升级为可复用规则
- 让高频失败最终被 deterministic sensor 或 gate 接住

## 2. 什么算 repeated failure

以下情况默认按 repeated failure 处理：

1. 同一类根因或控制缺口，在不同任务、不同 PR、不同 session 中再次出现
2. review 多次指出同一类问题，但仓库里还没有对应 guide、template、checker 或 smoke
3. 同一缺口在 `pantheon-base` 和业务仓重复出现
4. 同一类错误在 handoff 后又因为上下文丢失而复发

以下情况不必强行计为 repeated failure：

- 纯偶发 typo
- 明确无共享价值的一次性业务特例
- 已确认是 checker 误报，而非真实 failure pattern

## 3. Ratchet 梯度

### 3.1 Level 0：单次失败

第一次出现时，至少要把失败模式写进下列之一：

- closeout note
- review finding
- verification evidence 的 known gaps

目的不是定罪，而是保留可追踪描述。

### 3.2 Level 1：第二次出现

同类问题第二次出现时，必须至少升级一个前馈控制：

- `AGENTS.md`
- task packet 模板
- acceptance checklist
- contract / design / review guide
- repo-local how-to 或 playbook

如果没有升级动作，就不算完成 ratchet。

### 3.3 Level 2：第三次出现或跨仓复发

同类问题第三次出现，或已经跨仓、跨模块复发时，必须优先升级为反馈控制：

- test
- static check
- harness checker
- smoke case
- browser or runtime evidence gate
- failure registry entry

`failure registry` 是默认 landing file 之一，但不是唯一目标；如果能直接自动化，优先落成 sensor。

### 3.4 Level 3：高严重度或持续性失败

若 failure pattern 具备以下任一特征，可跳级升级：

- 安全、权限、审计、schema、数据损坏风险
- 会触发 human gate 的高影响变更
- 已经影响 release 质量或继承稳定性

这类问题可以直接升级为 blocking gate、required landing file 或 release review 必审项。

## 4. 记录要求

每次 ratchet 升级都应至少记录：

- failure pattern 是什么
- 这次升级到了哪个 artifact 或 sensor
- 为什么选这个层级，而不是更轻或更重的控制
- 下一次验证窗口是什么

推荐落点：

- `docs/harness/failure-registry.md`
- `HARNESS_OPEN_TASKS.md`
- release notes
- 对应 task packet / review / evidence

## 5. 与 failure registry 的关系

`failure registry` 负责记录“哪些问题已经被识别为值得持续记忆的 failure pattern”。

但它不代替：

- guide 更新
- template 更新
- checker 或 smoke 的自动化收口

是否应该把问题写入 registry，按以下规则判断：

- 如果只是一次性解释，不必进 registry
- 如果需要跨任务持续提醒，但暂时还无法自动化，优先进 registry
- 如果已经能自动化，registry 只保留摘要和来源链接

## 6. 回退与例外

若后续确认某次升级是基于误报、错误归因或临时工具缺陷，可回退到更轻一级控制。

但回退时必须同时记录：

- 为什么原判断不成立
- 哪个 artifact 被降级或删除
- 下一次复核时间
