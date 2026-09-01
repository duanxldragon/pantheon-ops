---
title: system/config 扩展设计
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-04-29
---

# system/config 扩展设计

English version: [SYSTEM_CONFIG_EXTENDED_DESIGN.en.md](./SYSTEM_CONFIG_EXTENDED_DESIGN.en.md)

本文用于把 `system/config` 已经真实存在的配置型公共能力重新收口成一个统一设计锚点。

它不是为了替代：

- `docs/designs/DICT_AND_SETTING_DESIGN.md`
- `docs/designs/ERROR_CODE_AND_I18N.md`
- `docs/designs/LOWCODE_GENERATOR_GUIDE.md`

而是为了回答一个更上层的问题：

> `system/config` 现在到底包含哪些子域，它们各自负责什么，彼此怎么隔离，以及它与低代码工作域如何协作？

---

## 1. 设计目标

`system/config` 不再只是“字典 + 设置页”。

当前应把它理解为一个复合系统域，负责以下四类配置型公共能力：

1. `dict`：运行时枚举与选项治理
2. `setting`：平台参数与策略配置
3. `i18n`：翻译资产与语言治理
4. `upload`：统一上传入口与存储配置
本文的目标：

- 把四块配置能力重新归到 `system/config`
- 明确 `system/config` 与低代码工作域的协作边界
- 防止后续继续把 `system/config` 做成“大杂烩”
- 给验收清单、权限矩阵和后续专项设计文档提供总锚点

---

## 2. 总体边界

### 2.1 `system/config` 负责

- 平台级配置的读取、维护、缓存刷新与审计
- 字典和下拉选项的统一治理
- 翻译资产、语言包、缺失修复与生命周期治理
- 上传配置与统一上传入口

### 2.2 `system/config` 不负责

- 用户、角色、菜单、权限授权本身，这属于 `system/iam`
- 登录、会话、密码、安全事件，这属于 `system/auth`
- 组织结构与组织治理，这属于 `system/org`
- 操作日志平台本身，这属于 `system/audit`
- 业务域运行时流程，这属于 `business/*`
- `/system/modules` 与 `/system/generator` 的工作域导航，这属于 `platform.lowcode`
- 模块注册与接入状态治理，这属于 `system/dynamicmodule`
- 模块脚手架生成，这属于 `system/generator`

### 2.3 关键约束

- `system/config` 可以沉淀“配置型公共能力”，但不能反向接管其他系统域职责
- `platform.lowcode` 可以聚合低代码工作域，但不等于能力归属
- `system/generator` 与 `system/dynamicmodule` 虽然在产品上同属低代码工作域，但实现上必须分离

---

## 3. 子域拆分

## 3.1 dict

职责：

- 字典类型
- 字典项
- 字典缓存刷新
- 前端 options 下发

判断：

- `dict` 是标准配置子域
- 它属于低风险治理能力，重点在一致性和复用，不属于高敏运维能力

现有设计落点：

- [DICT_AND_SETTING_DESIGN.md](./DICT_AND_SETTING_DESIGN.md)

## 3.2 setting

职责：

- `basic / security / login / audit / upload / i18n / ui` 分组配置
- 平台公开配置
- 敏感设置加密存储
- 配置审计与缓存刷新

判断：

- `setting` 是 `system/config` 的中枢子域
- 它负责“配置值”，但不直接负责其所有运行时消费语义

例如：

- `login.session_idle_minutes` 由 `platform` 壳层和 `system/auth` 消费
- `audit.operation_log_retention_days` 由 `system/audit` 消费
- `upload.*` 由 `upload` 子域消费

## 3.3 i18n

职责：

- 运行时语言包
- 翻译记录 CRUD
- 导入、导出、模板下载
- 缺失 locale 检测
- key 重命名预览与迁移
- builtin locale 回填
- 未使用 key 生命周期治理

判断：

- `i18n` 已经是独立子域，不应再被视为设置页附属功能
- 它既有“内容治理”属性，也有“运行时资源发布”属性

边界：

- `i18n` 负责翻译资产
- `frontend` 负责消费和 fallback
- 业务模块负责声明 namespace、补 key 和验收覆盖

后续建议：

- 单独补 `docs/designs/I18N_MODULE_DESIGN.md`

## 3.4 upload

职责：

- 统一上传入口
- 存储驱动切换
- 大小、类型、访问路径限制
- 本地文件访问入口
- S3-compatible 对象存储支持

判断：

- `upload` 是公共基础能力，配置归属 `system/config`
- 运行时文件处理属于平台公共能力，不应散落到业务模块各自实现

边界：

- 配置归 `system/config`
- 文件物理读写属于平台公共包
- 业务模块只能复用统一入口，不能各写一套上传协议

后续建议：

- 单独补 `docs/designs/UPLOAD_AND_STORAGE_DESIGN.md`

## 3.5 低代码相邻能力

`system/config` 需要与以下能力协作，但不再拥有它们：

- `platform.lowcode`：负责低代码工作域的菜单聚合、入口编排与阅读路径
- `system/dynamicmodule`：负责模块接入、卸载、清理、注册表一致性与待激活状态治理
- `system/generator`：负责模块 schema 校验、受控生成与治理摘要输出

约束：

- `generator` 不能跳过 `dynamicmodule` 直接宣告模块生效
- `dynamicmodule` 不替代 `generator` 做 schema 设计
- `system/config` 只为低代码链路提供相邻配置能力，例如 i18n、上传与受管数据源元数据，不再吞并其治理归属

---

## 4. 风险分级

| 子域 | 风险级别 | 原因 |
| :--- | :--- | :--- |
| `dict` | 低 | 主要影响展示选项与校验一致性 |
| `setting` | 中 | 可影响平台运行策略与公开配置 |
| `i18n` | 中 | 可影响全局文案、导入导出与错误反馈 |
| `upload` | 中高 | 涉及文件访问路径、存储驱动和对象访问地址 |
---

## 5. 前端页面归属

当前 `system/config` 只直接承载以下页面：

| 页面 | 子域 | 页面归属 |
| :--- | :--- | :--- |
| `/system/dict` | `dict` | `system/config` |
| `/system/setting` | `setting` | `system/config` |
| `/system/i18n` | `i18n` | `system/config` |

低代码工作域页面另行归属为：

| 页面 | 能力 | 页面归属 |
| :--- | :--- | :--- |
| `/system/modules` | `system/dynamicmodule` | `platform.lowcode` |
| `/system/generator` | `system/generator` | `platform.lowcode` |

---

## 6. 权限与安全约束

## 6.1 普通治理能力

包括：

- 字典 CRUD
- 设置查看与保存
- i18n 查看、编辑、导入导出、缓存刷新

要求：

- 页面权限
- 动作权限
- Casbin 接口权限
- 审计记录

## 6.2 高敏治理能力

包括：

- 与 `system/config` 相邻的低代码治理写操作
- 动态模块注册
- 动态模块卸载
- 生成器触发代码生成
- 影响平台模块装配的写操作

要求：

- 更高动作权限
- 二次验证
- 环境限制
- 清晰的审计归因
- 必要时要求显式回滚说明

---

## 7. 验收要求

`system/config` 后续验收不得只覆盖 `/system/dict` 和 `/system/setting`。

至少应固定覆盖：

- `/system/dict`
- `/system/setting`
- `/system/i18n`

每页至少检查：

- 页面可打开
- `pagePermission` 生效
- 主要动作权限生效
- console 无阻断错误
- 审计链路完整

高敏页额外检查：

- 二次验证是否生效
- 是否受环境限制
- 失败时是否给出明确阻断原因

---

## 8. 与其他文档的关系

| 文档 | 负责什么 | 与本文关系 |
| :--- | :--- | :--- |
| `docs/designs/DICT_AND_SETTING_DESIGN.md` | 字典与设置细节 | 是 `dict / setting` 子域细化文档 |
| `docs/designs/ERROR_CODE_AND_I18N.md` | 错误 key 与 i18n 责任边界 | 是 `i18n` 子域的重要配套文档 |
| `docs/designs/LOWCODE_GENERATOR_GUIDE.md` | 生成器使用与链路说明 | 是 `platform.lowcode` 工作域下的操作型文档 |
| `docs/acceptances/ACCEPTANCE_CHECKLIST.md` | 统一验收门槛 | 应补齐 `system/config` 扩展能力验收 |

---

## 9. 当前结论

`system/config` 已经成长为一个清晰的配置复合系统域。

接下来必须坚持两件事：

1. 继续按 `dict / setting / i18n / upload` 四块配置能力做逻辑拆分
2. 把低代码工作域稳定收敛在 `platform.lowcode`，并保持 `system/generator` 与 `system/dynamicmodule` 能力分离

否则它很容易再次退化成“什么都往里塞的 system 杂物间”。
