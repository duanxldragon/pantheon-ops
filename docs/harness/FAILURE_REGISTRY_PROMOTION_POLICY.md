---
title: Failure Registry Promotion Policy
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-08
---

# Failure Registry Promotion Policy

English version: [FAILURE_REGISTRY_PROMOTION_POLICY.en.md](./FAILURE_REGISTRY_PROMOTION_POLICY.en.md)

本文定义 failure registry 何时从“建议存在”升级为“必需 landing file”。

failure registry 是 `FAILURE_RATCHET_POLICY.md` 定义的 ratchet loop 中一种 landing file，但不是唯一 landing 目标。

## 1. 当前模式

当前 `check-failure-registry.mjs` 对默认路径缺失只报 warning，允许新仓库渐进接入。

## 2. Promotion Threshold

当以下条件全部满足时，应把默认 registry presence 升级为 required landing file：

1. 至少 2 个下游 consumer repo 已播种 `docs/harness/failure-registry.md`
2. 这 2 个 consumer repo 连续 2 个发布周期保留该文件且格式合法
3. method maintainers 在最近一次 release review 中确认：
   - registry 不再只是试点资产
   - 缺失 registry 会削弱 ratchet loop

## 3. 升级动作

升级时必须同步：

- 修改 `check-failure-registry.mjs`，将默认缺失从 warning 升级为 finding
- 修改 `check-method-health.mjs` required landing files
- 更新 `HARNESS_OPEN_TASKS.md`
- 在 release note 中记录 cutover

## 4. 回退条件

若升级后发现 consumer bootstrap 仍广泛不带 registry，且导致明显 adoption friction，可临时回退到 warning，但必须记录原因和下一次评估日期。
