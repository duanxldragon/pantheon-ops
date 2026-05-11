# system/config 扩展设计

更新时间：2026-04-29

类型：Design
归属层：system/config
状态：Active

本文用于把 `system/config` 已经真实存在的扩展能力重新收口成一个统一设计锚点。

它不是为了替代：

- `docs/designs/DICT_AND_SETTING_DESIGN.md`
- `docs/designs/ERROR_CODE_AND_I18N.md`
- `docs/designs/LOWCODE_GENERATOR_GUIDE.md`

而是为了回答一个更上层的问题：

> `system/config` 现在到底包含哪些子域，它们各自负责什么，彼此怎么隔离，哪些属于高敏治理能力？

---

## 1. 设计目标

`system/config` 不再只是“字典 + 设置页”。

当前应把它理解为一个复合系统域，负责以下六类能力：

1. `dict`：运行时枚举与选项治理
2. `setting`：平台参数与策略配置
3. `i18n`：翻译资产与语言治理
4. `upload`：统一上传入口与存储配置
5. `dynamicmodule`：动态模块注册治理
6. `generator`：业务模块脚手架生成器

本文的目标：

- 把六块能力重新归到 `system/config`
- 明确哪些是普通配置能力，哪些是高敏治理能力
- 防止后续继续把 `system/config` 做成“大杂烩”
- 给验收清单、权限矩阵和后续专项设计文档提供总锚点

---

## 2. 总体边界

### 2.1 `system/config` 负责

- 平台级配置的读取、维护、缓存刷新与审计
- 字典和下拉选项的统一治理
- 翻译资产、语言包、缺失修复与生命周期治理
- 上传配置与统一上传入口
- 模块生成、模块注册、模块状态治理

### 2.2 `system/config` 不负责

- 用户、角色、菜单、权限授权本身，这属于 `system/iam`
- 登录、会话、密码、安全事件，这属于 `system/auth`
- 组织结构与组织治理，这属于 `system/org`
- 操作日志平台本身，这属于 `system/audit`
- 业务域运行时流程，这属于 `business/*`

### 2.3 关键约束

- `system/config` 可以沉淀“配置型公共能力”，但不能反向接管其他系统域职责
- `generator` 与 `dynamicmodule` 虽然放在 `system/config`，但不等于普通设置页能力
- 高敏能力必须和“可公开读配置”严格分开

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

## 3.5 dynamicmodule

职责：

- 动态模块清单查询
- 模块注册/卸载
- 生成后模块状态管理
- generated registry 更新与装配对齐

判断：

- 这不是普通配置能力
- 这是 **高敏平台治理能力**

原因：

- 它影响工作区源码与模块装配
- 它改变平台可用模块版图
- 错误操作会直接影响构建、路由和权限接入

必须坚持：

- 默认按开发/内部治理能力理解
- 写操作必须受更高权限和二次验证保护
- 文档中必须单独说明环境限制、审计、回滚和误操作防护

后续建议：

- 单独补 `docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`

## 3.6 generator

职责：

- 基于 schema 生成业务模块脚手架
- 生成前后端代码、菜单、权限、i18n 初始骨架
- 输出模块文件并交给动态模块治理链路注册

判断：

- `generator` 属于 `system/config` 的辅助开发子域
- 它的产品定位是“研发加速器”，不是“运行时低代码平台”

边界：

- 只能生成受约束的模块骨架
- 不负责运行时热插拔业务编排
- 不应反向侵入 `auth / iam / org` 等系统域边界

必须坚持：

- key-first i18n
- 生成内容遵守模块契约
- 生成器不绕过动态模块治理与权限检查

---

## 4. 风险分级

| 子域 | 风险级别 | 原因 |
| :--- | :--- | :--- |
| `dict` | 低 | 主要影响展示选项与校验一致性 |
| `setting` | 中 | 可影响平台运行策略与公开配置 |
| `i18n` | 中 | 可影响全局文案、导入导出与错误反馈 |
| `upload` | 中高 | 涉及文件访问路径、存储驱动和对象访问地址 |
| `dynamicmodule` | 高 | 涉及模块注册、源码写入、模块卸载 |
| `generator` | 中高 | 涉及生成代码质量与边界收口，通常与 `dynamicmodule` 联动 |

---

## 5. 前端页面归属

当前应统一按以下页面理解：

| 页面 | 子域 | 页面归属 |
| :--- | :--- | :--- |
| `/system/dict` | `dict` | `system/config` |
| `/system/setting` | `setting` | `system/config` |
| `/system/i18n` | `i18n` | `system/config` |
| `/system/modules` | `dynamicmodule` | `system/config` |
| `/system/generator` | `generator` | `system/config` |

约束：

- 这些页面不能再只按“系统设置附属页面”看待
- 它们都属于 `system/config`，但验收和权限强度不应完全一样

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
- `/system/modules`
- `/system/generator`

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
| `docs/designs/LOWCODE_GENERATOR_GUIDE.md` | 生成器使用与链路说明 | 是 `generator / dynamicmodule` 的操作型文档 |
| `docs/acceptances/ACCEPTANCE_CHECKLIST.md` | 统一验收门槛 | 应补齐 `system/config` 扩展能力验收 |

---

## 9. 当前结论

`system/config` 已经成长为一个真正的复合系统域。

接下来必须坚持两件事：

1. 继续按 `dict / setting / i18n / upload / dynamicmodule / generator` 六块能力做逻辑拆分
2. 对高敏能力单独提高文档、权限、验收和审计标准

否则它很容易再次退化成“什么都往里塞的 system 杂物间”。
