---
title: 运行时模块引擎演进约束
doc_type: Design
layer: platform.lowcode
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-20
---

# 运行时模块引擎演进约束

本文只定义 Pantheon 未来从“受控生成链路”演进到“真热插拔运行时模块引擎”时必须遵守的架构约束。

它不是当前迭代的实现任务，也不要求当前代码立即重构。

当前主线仍然是：

- `system/generator`：生成 schema、契约和脚手架
- `system/dynamicmodule`：治理注册、激活状态、卸载与清理

本文的目的只有一个：

> 为未来真热插拔保留清晰演进方向，但不提前把当前主线改成半成品 runtime 方案。

---

## 1. 当前判断

Pantheon 当前低代码主线本质上仍是：

- 编译期后端模块装配
- 编译期前端页面装配
- 生成后受控激活

也就是说，当前是“受控生成器 / 模块接入加速器”，不是“运行时低代码平台”。

这条判断必须继续成立，直到运行时模块引擎真正落地。

禁止出现的中间态：

- 文档宣称支持热插拔，但实现仍依赖重启和重建
- 前端部分 runtime、后端仍是编译期注册
- 同一模块既走 runtime engine，又走 generated registry

---

## 2. 真热插拔的正式定义

Pantheon 未来只有同时满足以下条件，才允许对外宣称“真热插拔”：

1. 生成后无需重启后端进程
2. 生成后无需重建前端静态资源
3. 菜单、权限、路由、页面、API 在当前运行实例内即时生效
4. 模块卸载后在当前运行实例内即时失效
5. 生产环境不依赖 `go build`、`npm install`、`vite build`

如果还依赖：

- Go 源码生成并重新编译
- React/TSX 页面重新打包

那就仍然属于“编译期模块系统”，不属于真热插拔。

---

## 3. 架构分界线

### 3.1 当前方案

当前低代码模块是：

- 生成 Go/TS 源码
- 改写 generated registry
- 通过重启/重建进入运行态

### 3.2 未来 runtime 方案

未来真热插拔模块必须转成：

- 运行时 `schema + manifest` 模块
- 由统一 runtime engine 解释执行
- 由平台统一渲染列表、表单、详情、关系交互
- 由平台统一暴露 CRUD、lookup、relation API

结论：

- 未来真热插拔不能继续以“生成可编译 Go 模块”作为正式路径
- 未来真热插拔也不能继续以“生成 TSX 页面组件”作为正式路径

---

## 4. 当前阶段必须保留的约束

为了不把未来改造成本做高，当前实现只需要保留以下约束。

### 4.1 `schema/manifest` 必须继续作为事实源

无论当前是否仍生成源码，模块定义的真实来源都应继续收敛到：

- schema
- relation contract
- permission contract
- menu manifest
- lifecycle metadata

不得把关键语义只散落在生成后的前端页面代码中。

### 4.2 `dynamicmodule` 只负责治理，不提前承担 runtime engine

当前 `dynamicmodule` 应继续负责：

- 注册状态
- 激活诊断
- 卸载与清理
- autoRecycle 生命周期治理

但不应提前承担：

- 动态执行器
- 通用 CRUD runtime
- 前端 schema renderer

否则会把当前稳定治理链路拖进大重构。

### 4.3 前端模板继续向 contract 收敛，不转成半成品 DSL

当前可以继续生成：

- `List`
- `Form`
- `Detail`
- 主从表 / many-to-many 模板

但目标应是：

- 模式统一
- contract 明确
- 页面特例减少

而不是在当前阶段直接把页面体系改写成运行时 DSL 渲染器。

### 4.4 禁止引入第二套正式激活模型

在 runtime engine 真正落地前，Pantheon 只保留一条正式激活链路：

- 生成
- 注册
- 待激活
- 重启 / 重建
- 已接入

不得在当前阶段并行引入另一套“看似热插拔”的正式生产链路。

---

## 5. 未来 runtime engine 的正式落点

未来若启动真热插拔改造，推荐按以下职责拆分：

### 5.1 generator

只负责：

- 采集建模信息
- 生成 schema / manifest / contract
- 生成可选的开发脚手架导出物

不再承担正式生产模块代码生成职责。

### 5.2 dynamicmodule

只负责：

- 模块注册治理
- 生命周期状态流转
- 激活、停用、回滚、审计

不承担页面渲染和数据执行。

### 5.3 runtime module engine

新增独立能力，负责：

- schema 装载
- manifest 激活
- 运行时 CRUD 与 relation handler
- 前端页面 schema 渲染
- 路由与菜单运行时拼装
- 运行时缓存失效与回滚

---

## 6. 为什么当前不实现

当前不直接切 runtime engine，不是因为方向不对，而是因为它会同时改掉：

- 后端模块装配模型
- 前端路由与页面装配模型
- 权限与菜单生效路径
- 测试与交付链路

这已经超出当前主线收口范围。

当前最合理的策略是：

- 继续把现有生成链路收口到产品级
- 在设计上明确未来 runtime engine 的边界
- 等业务节奏和交付目标明确后，再单独立项切换

---

## 7. 当前结论

Pantheon 当前不做真热插拔实现，只保留真热插拔演进设计。

正式约束如下：

- 当前主线继续是“受控生成链路”
- 未来若做真热插拔，必须转向 `runtime module engine + manifest/schema activation`
- 在此之前，不提前引入第二套生产级模块系统
- 不为了未来可能的 runtime 改造，提前扩大当前收口范围

这份文档的意义，不是现在就开始重写底座，而是避免之后在错误方向上做半吊子改造。
