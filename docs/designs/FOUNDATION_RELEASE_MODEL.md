---
title: Foundation Release Model
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-04
---

# Foundation Release Model

English version: [FOUNDATION_RELEASE_MODEL.en.md](./FOUNDATION_RELEASE_MODEL.en.md)

本文定义 `pantheon-base` 如何作为共享底座持续迭代，同时让消费仓库只消费“标准 foundation release”，而不是直接追随 `main` 的中间状态。

## 1. 目标

`pantheon-base` 需要同时满足两件事：

1. 能持续做漏洞修复、质量治理、架构收敛和底座优化
2. 不把这些“正在变化的过程状态”直接泄漏给 `pantheon-ops` 这类消费仓库

因此默认模型应从：

```text
consumer repo -> track base/main
```

切换为：

```text
consumer repo -> consume base foundation release
```

## 2. 核心原则

### 2.1 `main` 不是消费接口

`pantheon-base/main` 是持续开发线，不是下游业务仓库的默认继承接口。

允许在 `main` 上发生的事情包括：

- 安全修复
- 重构
- 重复率治理
- 架构收敛
- 契约补齐
- 代码质量门禁增强

这些变更在稳定前都不应要求消费仓同步。

### 2.2 release 才是消费接口

`pantheon-ops` 和未来其他业务仓默认只消费以下之一：

- 显式 tag，例如 `pantheon-base-v0.10.0`
- 显式 release line，例如 `release/0.10`
- 紧急例外下的显式 commit，但必须带原因

默认不允许写成“跟随 `main`”。

### 2.3 base 负责发布，consumer 负责升级

职责分离：

- `pantheon-base` 负责发布一个经过门禁的 foundation release
- `pantheon-ops` 负责把本地业务 overlay 升级到某个 foundation release

这不是“同步代码”，而是“升级所消费的 foundation 版本”。

## 3. 资产分层

### 3.1 Foundation-Owned

以下内容默认由 `pantheon-base` 拥有：

- `platform`
- `system/*`
- 共享后台壳层
- 共享 frontend components / shell / table / upload / smoke helpers
- 共享 permission / i18n / audit / menu contracts
- 共享 generator / governance 底座约束

### 3.2 Consumer-Owned Overlay

以下内容默认由消费仓拥有：

- `business/*`
- 本地业务设计与验收
- 业务 smoke
- 业务 seed / 业务 i18n / 业务菜单接入

### 3.3 Integration Seams

允许下游挂接但仍受 base 契约约束的接缝包括：

- business module registry
- business component registry
- `business.*` i18n entries
- business menu mount points

`business component registry` 接缝由菜单契约门禁（`frontend/scripts/check-menu-contract.mjs`）按命名约定发现（前端 `*Registry.ts`、后端 `*registry.go`）。consumer 注入的 overlay 注册表因此无需修改 base 检查脚本即可被纳入一致性校验。

接缝必须显式、可审查、可升级；不能把它们演化成对共享底座的隐式 override。

## 4. Foundation Release 的最小定义

一个可被消费仓采用的 foundation release，至少要包含：

1. **版本标识**
   - 推荐 tag：`pantheon-base-v<major>.<minor>.<patch>`
2. **release notes**
   - 说明本次影响的共享层面
3. **consumer impact summary**
   - 说明哪些 consumer 可能受影响
4. **upgrade notes**
   - 说明 consumer 从前一 release 升级时要注意什么
5. **验证结论**
   - required checks、CodeQL、关键 smoke 或人工 evidence 结论

推荐把这些信息整理成统一的 release note / release manifest，而不是散落在聊天记录里。

### 4.1 Frontend shared-path ownership

`manifest.sharedPaths.frontend` 是可执行的所有权契约，不是一次性复制清单。基础壳层的 `App.tsx`、`main.tsx`、`vite-env.d.ts` 必须显式声明；共享传输和权限 helper 必须分别以 `frontend/src/api` 与 `frontend/src/hooks` 目录声明，避免新增叶子文件悄然落到 consumer。

打包会拒绝存在但未被该契约覆盖的通用前端根路径。`business/*`、已声明 overlay 和业务 i18n 接缝不由这个机制接管；它们仍按 consumer-owned overlay 规则升级。

## 5. Release Gate

一个 foundation release 在发布前，至少应满足：

- GitHub required checks 全绿
- CodeQL 没有未解释的可达高危问题
- Security Hotspots 已 review
- 与共享底座稳定性相关的高风险改动已完成独立 review
- 如果影响 consumer upgrade，需要补 consumer impact summary

当 release 包含以下类型变更时，必须把升级影响写清楚：

- 权限模型
- i18n key 语义
- 菜单/路由/registry 契约
- generator / dynamic-module 治理
- 共享 shell / shared table / upload / smoke helpers
- `.github/workflows/*`

## 6. Consumer Upgrade 模型

消费仓升级时，默认顺序是：

1. 选择目标 foundation release tag
2. 更新 `docs/PROJECT_INHERITANCE.md` 中的 base version / release line
3. 运行 inheritance / sync / drift checks
4. 只修复业务 overlay 与新 foundation release 的真实断点
5. 运行业务仓的最小验证集
6. 记录升级结果和残留差异

消费仓不应该做的事情：

- 直接跟 `main`
- 把共享层差异长期保留在本地 override
- 在没有 release 边界的情况下持续拷贝 base 文件

## 7. 版本建议

推荐使用语义化思路：

- `major`
  - 基础契约或消费方式发生破坏性变化
- `minor`
  - 向后兼容的新共享能力、治理增强、可消费优化
- `patch`
  - 安全修复、质量修复、兼容性补丁

如果 `pantheon-base` 还没准备好完整 package 化，也应至少先做到：

- 有稳定 tag
- 有 release notes
- 有 upgrade notes
- 有 consumer impact summary

## 8. 对 `pantheon-ops` 的直接要求

`pantheon-ops` 后续默认应写成：

- Base branch / release line：`release/<x.y>` 或相应稳定线
- Base version：`pantheon-base-v<x.y.z>`
- Inheritance mode：`foundation-release-consumer`

而不是：

- Base branch：`main`
- Base version：临时 commit

只有紧急例外才允许消费未发布 commit，并且必须留下原因、回滚边界和后续并入正式 release 的计划。
