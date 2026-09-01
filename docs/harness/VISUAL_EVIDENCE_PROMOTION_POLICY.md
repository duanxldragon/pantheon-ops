---
title: Visual Evidence Promotion Policy
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-08
---

# Visual Evidence Promotion Policy

English version: [VISUAL_EVIDENCE_PROMOTION_POLICY.en.md](./VISUAL_EVIDENCE_PROMOTION_POLICY.en.md)

本文定义 `check-visual-evidence.mjs --strict` 何时从观察性 gate 升级为阻断性 gate。

## 1. 当前模式

当前默认采用观察模式：

- `--strict` 仍然返回失败退出码
- CI 可继续使用 `continue-on-error: true` 观测结果

## 2. Promotion Threshold

满足以下全部条件时，应把 CI 中的 visual evidence job 升级为阻断性 gate：

1. 最近连续 3 个 UI-affecting non-trivial 任务都具备：
   - viewport 计划
   - state 计划
   - browser evidence 或 screenshot evidence，或显式 visual gap
2. 这 3 个任务中，`check-visual-evidence.mjs --strict` 均无 warnings
3. 同一时间窗口内，没有因为 checker 误报而需要人工回退规则

## 3. 回退条件

若升级后连续 2 个任务出现由 checker 定义不充分导致的误报，应：

- 暂时回退为观察模式
- 修正 checker 或 policy
- 重新开始计算 promotion window

## 4. 记录要求

promotion 或回退时，必须同步更新：

- `HARNESS_OPEN_TASKS.md`
- 对应 failure registry 或 review notes
- CI workflow 注释
