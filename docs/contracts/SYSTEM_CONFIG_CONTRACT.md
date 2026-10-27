---
title: system/config 合同文档
doc_type: Contract
layer: system/config
status: Active
related_designs:
  - docs/designs/DICT_AND_SETTING_DESIGN.md
  - docs/designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md
  - docs/designs/I18N_MODULE_DESIGN.md
  - docs/designs/UPLOAD_AND_STORAGE_DESIGN.md
  - docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md
  - docs/designs/GENERATOR_MODULE_DESIGN.md
  - docs/designs/BUSINESS_DICT_INTEGRATION_GUIDE.md
related_acceptances:
  - docs/acceptances/ACCEPTANCE_CHECKLIST.md
  - docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md
  - docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md
updated_at: 2026-05-12
---

# system/config 合同文档

English version: [SYSTEM_CONFIG_CONTRACT.en.md](./SYSTEM_CONFIG_CONTRACT.en.md)

关联设计：
- `DICT_AND_SETTING_DESIGN.md`
- `SYSTEM_CONFIG_EXTENDED_DESIGN.md`
- `I18N_MODULE_DESIGN.md`
- `UPLOAD_AND_STORAGE_DESIGN.md`
- `DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
- `GENERATOR_MODULE_DESIGN.md`
- `BUSINESS_DICT_INTEGRATION_GUIDE.md`

关联验收：
- `ACCEPTANCE_CHECKLIST.md`
- `SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
- `PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`

---

本文用于定义 Pantheon `system/config` 能力域的执行契约。

它锁定的是配置型公共能力的总边界，避免后续把 `dict / setting / i18n / upload` 再次混成一个没有风险分级、没有职责区分的“大配置杂物间”。

---

## 1. 背景

Pantheon 当前的 `system/config` 已经不再只是“字典 + 设置页”。

它当前承载 4 类配置型公共能力：

1. `dict`
2. `setting`
3. `i18n`
4. `upload`
如果没有 `system/config` 合同，后续最容易继续发生：

- 普通配置能力继续被混成一个大页、大表单、大杂烩
- `i18n / upload` 继续被误当成设置页附属功能
- 设计文档越来越多，但没有统一总锚点
- `system/config` 重新变成最典型的大杂烩系统域

## 2. 归属层

本合同归属 `system/config`。

它覆盖：

- 字典与运行时选项治理
- 平台参数与策略配置
- 翻译资产与语言治理
- 统一上传入口与存储配置

它不等于：

- `system/auth` 登录、会话、密码、安全策略消费语义
- `system/iam` 用户、角色、菜单、权限治理
- `system/org` 组织结构治理
- `platform` 壳层导航与工作台聚合
- `platform.lowcode` 低代码工作域聚合
- `system/dynamicmodule` 模块注册与接入治理
- `system/generator` 受控代码生成能力

## 3. 目标

`system/config` 合同的目标是锁定以下 6 件事：

1. 明确 `config` 是配置型公共能力域，而不是其他系统域的剩余项容器
2. 明确 `dict / setting / i18n / upload` 的总边界
3. 明确 `config` 与低代码工作域、模块生成、动态接入治理之间的协作边界
4. 明确 `config` 只定义配置与治理，不直接吞掉其他系统域运行时职责
5. 明确 `system/config` 的完成定义和验收口径
6. 为后续专题子合同预留稳定扩展位

## 4. 非目标

本合同明确不负责：

- 登录、会话、密码、登录失败锁定等认证主链路
- 角色授权、菜单元数据语义、权限工作台
- 部门、岗位和组织结构治理
- 业务域自己的配置语义和生命周期
- `platform.lowcode` 的工作域导航
- `system/generator` 的模块骨架生成
- `system/dynamicmodule` 的模块接入治理

换句话说：

- `system/config` 可以提供公共配置能力；
- 但不能反向接管 `auth / iam / org / platform` 的核心职责。

## 5. 边界

### 5.1 覆盖对象

- `/system/dict`
- `/system/setting`
- `/system/i18n`
- 统一上传接口与上传配置
- 配置缓存刷新
- 配置变更审计

### 5.2 子域边界

#### `dict`

- 负责字典类型、字典项、状态、排序、options 下发

#### `setting`

- 负责平台参数、分组配置、公开配置、敏感配置、缓存刷新与审计
- 设置前端必须保持“总览入口 + 单分组编辑页”结构，不得重新退化为承载全部配置与审计的大杂烩单页
- 设置审计属于 `setting` 与 `audit` 的交叉能力，只允许收敛在 `audit` 分组页，不得重新扩散为所有分组的常驻区块

#### `i18n`

- 负责翻译资产、语言包、导入导出、缺失检测、key 生命周期治理

#### `upload`

- 负责上传配置、统一上传入口、存储驱动切换、访问 URL 生成

#### 低代码相邻能力

- `platform.lowcode`：负责 `/system/lowcode`、`/system/modules`、`/system/generator` 的导航聚合与工作域编排
- `system/dynamicmodule`：负责模块接入治理、注册状态与 generated registry 对齐
- `system/generator`：负责业务模块骨架生成、schema 校验、生成链路与受控注册入口
- 上述三者与 `system/config` 相邻，但不再归 `config` 直管

### 5.3 不覆盖对象

- `/login`
- `/auth/security`
- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/system/dept`
- `/system/post`
- 平台壳层导航和工作台聚合
- `/system/modules`
- `/system/generator`

## 6. 依赖

`system/config` 合同依赖以下文档与约束：

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [DICT_AND_SETTING_DESIGN.md](../designs/DICT_AND_SETTING_DESIGN.md)
- [SYSTEM_CONFIG_EXTENDED_DESIGN.md](../designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [I18N_MODULE_DESIGN.md](../designs/I18N_MODULE_DESIGN.md)
- [UPLOAD_AND_STORAGE_DESIGN.md](../designs/UPLOAD_AND_STORAGE_DESIGN.md)
- [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](../designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [GENERATOR_MODULE_DESIGN.md](../designs/GENERATOR_MODULE_DESIGN.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 域边界约束

- `system/config` 只承载配置型公共能力
- 不能反向定义 `auth / iam / org / platform` 的核心职责
- 子域之间可以共享配置来源，但不能吞并彼此语义

### 7.2 风险分级约束

- `dict / setting / i18n` 属于普通配置或内容治理能力
- `upload` 属于公共基础能力，配置在 `config`，运行时在平台公共包
- `dynamicmodule / generator` 属于独立高敏系统能力，不按 `system/config` 子域处理

### 7.3 高敏能力约束

- `dynamicmodule` 默认按开发/内部治理能力理解
- 写操作必须受更高权限、环境限制和二次验证保护
- `generator` 的高敏动作是“生成并注册”，而不是“查看页面”
- 真实写操作不得绕过后端 Casbin、环境守卫和二次验证
- `/system/i18n`、`/system/setting`、`/system/setting/:groupKey` 必须遵守统一页面准入规则
- 页头只表达当前页主任务，不重复右侧定位菜单
- 首屏只能保留一个治理摘要容器，禁止再堆“说明卡片墙”
- 高敏动作只能收束到页头动作区、表头动作区或治理抽屉，不得散在首屏多个位置
- 页面不得同时承担“概览中心 + 深度编辑器 + 长说明文”三种职责

### 7.4 运行时消费约束

- `setting` 可以输出配置值，但不直接接管所有消费语义
- `i18n.default_language`、`login.*`、`audit.*` 等配置由其他系统域或平台消费时，必须保持边界清晰
- `i18n.default_language` 只表示平台默认语言，运行时仅在“当前会话无显式语言覆盖且当前用户无语言偏好”时生效
- `system/config` 提供默认值来源，但不拥有“登录页语言选择 / 用户偏好 / 当前会话语言”的最终优先级解释权
- `upload` 统一入口必须被业务模块复用，不允许各自发明上传协议

### 7.5 文档约束

- `system/config` 的设计、评估、整改、验收文档都必须回指本合同
- 后续允许在本合同之下继续拆专题子合同，但不得绕过本总合同

## 8. 完成定义

`system/config` 达到“当前已完成”至少应满足：

### 8.0 批量删除能力约束

- 字典类型、字典项支持受控批量删除，归属 `system/config/dict`，不得扩散成跨系统域通用删除器。
- 批量删除必须使用独立权限点 `system:dict:batch-delete`；批量启停使用 `system:dict:batch-update`，二者不得混用。
- 批量删除接口必须复用单条删除服务校验，保留字典类型被字典项占用、唯一键软删释放、字典缓存失效等保护逻辑。
- 批量删除属于高风险写操作，必须经过二次验证，并返回部分成功结果：`deletedCount`、`failedCount`、`failures[]`。

### 8.1 职责完成

- 四类配置子域边界清晰
- `system/config` 与低代码相邻能力的归属边界清晰

### 8.2 运行时完成

- 字典、设置、i18n、上传主链路稳定
- 配置值与运行时消费语义不再混乱

### 8.3 风险控制完成

- 上传能力具备统一配置与统一入口
- `i18n` 已被视为独立子域，而不是设置页附属功能
- 高敏页面首屏职责清晰：`modules` 负责已接入模块治理，`generator` 负责新模块接入，`i18n` 负责运行时语言资产治理，`setting` 保持总览 / 单组拆分

### 8.4 文档与验收完成

- `config` 的主设计、整改、验收文档都能回链本合同
- 后续专题子合同可以在本合同下继续拆分

## 9. 验收标准

`system/config` 相关改动至少应通过以下验收：

### 9.1 文档验收

- 符合 [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 后端与配置验收

- 相关模块改动应补对应 `go test`
- 如果影响上传、i18n、动态模块或生成器，应补各自链路验证

### 9.3 前端与构建验收

- `cd frontend && npm run build`
- `cd frontend && npm run check:i18n-hardcode`
- 如果影响配置页或高敏治理页主链路，补页面级冒烟或验收记录

### 9.4 页面与主链路验收

- `/system/dict`
- `/system/setting`
- `/system/i18n`
- `/system/modules`
- `/system/generator`

### 9.5 高敏治理验收

- `dynamicmodule` 和 `generator` 不能按普通设置页口径验收
- 必须显式检查权限、环境限制、二次验证和审计链路
- 必须显式检查页面准入：无冗余右侧定位菜单、无 hero 墙、首屏只有一个治理摘要容器

## 10. 关联文档

### 10.1 Design

- [DICT_AND_SETTING_DESIGN.md](../designs/DICT_AND_SETTING_DESIGN.md)
- [SYSTEM_CONFIG_EXTENDED_DESIGN.md](../designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [I18N_MODULE_DESIGN.md](../designs/I18N_MODULE_DESIGN.md)
- [UPLOAD_AND_STORAGE_DESIGN.md](../designs/UPLOAD_AND_STORAGE_DESIGN.md)
- [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](../designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [GENERATOR_MODULE_DESIGN.md](../designs/GENERATOR_MODULE_DESIGN.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)

## 11. 后续专题子合同预留

本合同是 `system/config` 的总合同。

后续如继续深化，允许在其下补专题级合同，例如：

- `system/config -> i18n contract`
- `system/config -> upload contract`
- `platform.lowcode -> dynamicmodule governance contract`
- `platform.lowcode -> generator governance contract`

规则：

- 子合同不能改写本总合同边界
- 子合同只做更细的完成定义与验收补充
- 在没有专题子合同前，仍以本总合同为主依据

