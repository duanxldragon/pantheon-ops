# system/config 合同文档

更新时间：2026-04-30

类型：Contract
归属层：system/config
状态：Active

关联设计：
- `DICT_AND_SETTING_DESIGN.md`
- `SYSTEM_CONFIG_EXTENDED_DESIGN.md`
- `I18N_MODULE_DESIGN.md`
- `UPLOAD_AND_STORAGE_DESIGN.md`
- `DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
- `GENERATOR_MODULE_DESIGN.md`
- `BUSINESS_DICT_INTEGRATION_GUIDE.md`

关联评估：
- `SYSTEM_MODULE_AUDIT.md`
- `PLATFORM_GAP_AUDIT_20260429.md`

关联整改：
- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`

关联验收：
- `ACCEPTANCE_CHECKLIST.md`
- `SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
- `PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md`
- `QA_SMOKE_REPORT_20260420.md`

---

本文用于定义 Pantheon `system/config` 能力域的执行契约。

它锁定的是配置型公共能力的总边界，避免后续把 `dict / setting / i18n / upload / dynamicmodule / generator` 再次混成一个没有风险分级、没有职责区分的“大配置杂物间”。

---

## 1. 背景

Pantheon 当前的 `system/config` 已经明显超过“字典 + 设置页”的规模。

它现在至少承载 6 类能力：

1. `dict`
2. `setting`
3. `i18n`
4. `upload`
5. `dynamicmodule`
6. `generator`

如果没有 `system/config` 合同，后续最容易继续发生：

- 普通配置能力和高敏治理能力放在一个口径里讨论
- `i18n / upload / dynamicmodule / generator` 继续被误当成设置页附属功能
- 设计文档越来越多，但没有统一总锚点
- `system/config` 重新变成最典型的大杂烩系统域

## 2. 归属层

本合同归属 `system/config`。

它覆盖：

- 字典与运行时选项治理
- 平台参数与策略配置
- 翻译资产与语言治理
- 统一上传入口与存储配置
- 动态模块治理
- 模块生成器与辅助开发链路

它不等于：

- `system/auth` 登录、会话、密码、安全策略消费语义
- `system/iam` 用户、角色、菜单、权限治理
- `system/org` 组织结构治理
- `platform` 壳层导航与工作台聚合

## 3. 目标

`system/config` 合同的目标是锁定以下 6 件事：

1. 明确 `config` 是配置型公共能力域，而不是其他系统域的剩余项容器
2. 明确普通配置能力和高敏治理能力的风险分级
3. 明确 `dict / setting / i18n / upload / dynamicmodule / generator` 的总边界
4. 明确 `config` 只定义配置与治理，不直接吞掉其他系统域运行时职责
5. 明确 `system/config` 的完成定义和验收口径
6. 为后续专题子合同预留稳定扩展位

## 4. 非目标

本合同明确不负责：

- 登录、会话、密码、登录失败锁定等认证主链路
- 角色授权、菜单元数据语义、权限工作台
- 部门、岗位和组织结构治理
- 业务域自己的配置语义和生命周期
- 直接把 `generator` 或 `dynamicmodule` 定义成成熟运行时低代码平台

换句话说：

- `system/config` 可以提供公共配置和辅助开发能力；
- 但不能反向接管 `auth / iam / org / platform` 的核心职责。

## 5. 边界

### 5.1 覆盖对象

- `/system/dict`
- `/system/setting`
- `/system/i18n`
- `/system/modules`
- `/system/generator`
- 统一上传接口与上传配置
- 配置缓存刷新
- 配置变更审计

### 5.2 子域边界

#### `dict`

- 负责字典类型、字典项、状态、排序、options 下发

#### `setting`

- 负责平台参数、分组配置、公开配置、敏感配置、缓存刷新与审计

#### `i18n`

- 负责翻译资产、语言包、导入导出、缺失检测、key 生命周期治理

#### `upload`

- 负责上传配置、统一上传入口、存储驱动切换、访问 URL 生成

#### `dynamicmodule`

- 负责模块接入治理、注册状态与 generated registry 对齐

#### `generator`

- 负责业务模块骨架生成、schema 校验、生成链路与受控注册入口

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

## 6. 依赖

`system/config` 合同依赖以下文档与约束：

- [DESIGN.md](D:/workspace/go/pantheon-ops/DESIGN.md)
- [AGENTS.md](D:/workspace/go/pantheon-ops/AGENTS.md)
- [BACKEND.md](D:/workspace/go/pantheon-ops/docs/designs/BACKEND.md)
- [FRONTEND.md](D:/workspace/go/pantheon-ops/docs/designs/FRONTEND.md)
- [DICT_AND_SETTING_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/DICT_AND_SETTING_DESIGN.md)
- [SYSTEM_CONFIG_EXTENDED_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [I18N_MODULE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/I18N_MODULE_DESIGN.md)
- [UPLOAD_AND_STORAGE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/UPLOAD_AND_STORAGE_DESIGN.md)
- [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [GENERATOR_MODULE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/GENERATOR_MODULE_DESIGN.md)
- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. 强约束

### 7.1 域边界约束

- `system/config` 只承载配置型公共能力和受控辅助开发能力
- 不能反向定义 `auth / iam / org / platform` 的核心职责
- 子域之间可以共享配置来源，但不能吞并彼此语义

### 7.2 风险分级约束

- `dict / setting / i18n` 属于普通配置或内容治理能力
- `upload` 属于公共基础能力，配置在 `config`，运行时在平台公共包
- `dynamicmodule / generator` 属于高敏治理能力，不按普通设置页口径处理

### 7.3 高敏能力约束

- `dynamicmodule` 默认按开发/内部治理能力理解
- 写操作必须受更高权限、环境限制和二次验证保护
- `generator` 的高敏动作是“生成并注册”，而不是“查看页面”
- 真实写操作不得绕过后端 Casbin、环境守卫和二次验证

### 7.4 运行时消费约束

- `setting` 可以输出配置值，但不直接接管所有消费语义
- `i18n.default_language`、`login.*`、`audit.*` 等配置由其他系统域或平台消费时，必须保持边界清晰
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

- 六类子域边界清晰
- 普通配置能力和高敏治理能力分级清晰

### 8.2 运行时完成

- 字典、设置、i18n、上传、动态模块、生成器主链路稳定
- 配置值与运行时消费语义不再混乱

### 8.3 风险控制完成

- `dynamicmodule / generator` 高敏动作具备受控边界
- 上传能力具备统一配置与统一入口
- `i18n` 已被视为独立子域，而不是设置页附属功能

### 8.4 文档与验收完成

- `config` 的主设计、整改、验收文档都能回链本合同
- 后续专题子合同可以在本合同下继续拆分

## 9. 验收标准

`system/config` 相关改动至少应通过以下验收：

### 9.1 文档验收

- 符合 [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- 符合 [DOCUMENT_GOVERNANCE_CONTRACT.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- 符合 [DOCUMENT_METADATA_AND_STATUS.md](D:/workspace/go/pantheon-ops/docs/contracts/DOCUMENT_METADATA_AND_STATUS.md)

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

## 10. 关联文档

### 10.1 Design

- [DICT_AND_SETTING_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/DICT_AND_SETTING_DESIGN.md)
- [SYSTEM_CONFIG_EXTENDED_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [I18N_MODULE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/I18N_MODULE_DESIGN.md)
- [UPLOAD_AND_STORAGE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/UPLOAD_AND_STORAGE_DESIGN.md)
- [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [GENERATOR_MODULE_DESIGN.md](D:/workspace/go/pantheon-ops/docs/designs/GENERATOR_MODULE_DESIGN.md)

### 10.2 Assessment

- [SYSTEM_MODULE_AUDIT.md](D:/workspace/go/pantheon-ops/docs/assessments/SYSTEM_MODULE_AUDIT.md)
- [PLATFORM_GAP_AUDIT_20260429.md](D:/workspace/go/pantheon-ops/docs/assessments/PLATFORM_GAP_AUDIT_20260429.md)

### 10.3 Remediation

- [BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md](D:/workspace/go/pantheon-ops/docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md)

### 10.4 Acceptance

- [ACCEPTANCE_CHECKLIST.md](D:/workspace/go/pantheon-ops/docs/acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](D:/workspace/go/pantheon-ops/docs/acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- [QA_SMOKE_REPORT_20260420.md](D:/workspace/go/pantheon-ops/docs/archive/QA_SMOKE_REPORT_20260420.md)

## 11. 后续专题子合同预留

本合同是 `system/config` 的总合同。

后续如继续深化，允许在其下补专题级合同，例如：

- `system/config -> i18n contract`
- `system/config -> upload contract`
- `system/config -> dynamicmodule contract`
- `system/config -> generator contract`

规则：

- 子合同不能改写本总合同边界
- 子合同只做更细的完成定义与验收补充
- 在没有专题子合同前，仍以本总合同为主依据
