---
title: P2 规模化能力路线图
doc_type: Design
layer: platform / system/auth / system/iam / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-05
---

# P2 规模化能力路线图

English version: [P2_SCALE_ROADMAP.en.md](./P2_SCALE_ROADMAP.en.md)

本文用于收口 `pantheon-base` 迈向“标准企业后台产品底座”的第二阶段专题线。它不替代全局架构文档，而是承接第二阶段中真正进入 P2/规模化与产品化深化的方向：数据权限、真实多租户、SSO/OIDC、登录风控、业务模块自动化验收，以及平台级产品化控制台骨架。

---

## 1. 总原则

P2 不应做成一轮“大爆炸重构”。

第二阶段的前置条件是：Harness Engineering 已先完成默认 adoption 和关键门禁收口。否则后续跨 `platform / system/* / business/*` 的产品化补齐会重新发散。

第二阶段内部按四条专题线推进：

1. `auth-scale`
2. `platform-ops`
3. `governance-core`
4. `enterprise-backoffice`

正确顺序：

1. `auth-scale` 与 `governance-core` 先行，先补抽象边界与治理闭环。
2. `platform-ops` 跟进，补平台级消息、偏好和工作台控制台骨架。
3. `enterprise-backoffice` 最后推进，先补标准产品位和最小闭环样板。
4. 数据权限中间件和角色策略继续扩面。
5. 业务模块按 `DataScopeReq` 固定接入。
6. 租户就绪审查和租户字段策略继续深化，再决定真实多租户。
7. SSO/OIDC 在身份源明确后进入真实实现。
8. 登录风控继续从节流扩展到新设备 / 异地 / 高风险 MFA。
9. 业务 smoke 和报告归档常态化。

### 1.1 第二阶段完成定义

第二阶段不是“多几个页面”就算完成，而是每条专题线都要同时满足：

- 有合同或主设计锚点
- 有明确归属层
- 有数据模型或配置模型
- 有权限和审计语义
- 有自动化验证或固定 smoke
- 有未实现边界说明，避免伪能力上线

### 1.2 第二阶段不做

- 不在身份源未知时提前写真实 SSO provider 实现
- 不把通知中心直接做成完整 IM 系统
- 不把审批流、报表、监控一次做成独立大型产品
- 不在没有真实业务样板前过度抽象 workflow / report / alert 引擎

## 2. 四条专题线

### 2.1 `auth-scale`

目标：

- 让 `system/auth` 达到 `sso-ready`、`captcha-ready`、`risk-ready`

本线包含：

- provider 抽象
- 外部身份绑定模型
- 本地登录兜底与审计语义
- 登录告警与风险事件增强
- 安全策略页与配置页对齐

本线刻意不包含：

- 真实 OIDC / OAuth2 / LDAP / 企业微信 / 钉钉接入
- 单点登出协议细节

### 2.2 `platform-ops`

目标：

- 把平台壳层补成真实运营控制台骨架

本线包含：

- 通知中心消息模型
- dashboard widget 注册治理
- 用户偏好体系
- 平台级待办/告警摘要

### 2.3 `governance-core`

目标：

- 把已有治理能力从“发现型”推进到“闭环型”

本线包含：

- 权限工作台整改追踪
- 数据权限扩面
- 动态模块 / 生成器交付治理
- 高敏治理页验收基线增强

### 2.4 `enterprise-backoffice`

目标：

- 补齐标准后台产品位，但先从骨架和契约开始

本线包含：

- 审批流骨架
- 任务中心
- 调度中心
- 报表中心
- 监控告警中心

## 3. 数据权限

当前已有：

- `DataScopeReq`
- `WithDataScope`
- 生成器数据权限开关。
- CMDB Host 列表接入数据权限参数。
- `DataScopeMiddleware`。
- `system_role_data_scope` 角色数据范围策略表。
- CMDB Host 路由接入数据权限中间件。
- `/system/permission` 数据权限管理页，可按角色配置 `all / self / dept / dept_and_children / custom`。
- 多角色数据范围已按“授权叠加”合并：`all` 优先，多个 `custom` 合并部门集合，其他模式按固定优先级选择，避免依赖数据库返回顺序。
- `dept_and_children` 已通过 `system_dept.ancestors` 展开当前部门及下级部门，失败时记录日志并回退为当前部门。
- CMDB Host 已补 `dept_id` 数据范围字段，并用后端回归固定 `dept_and_children` 过滤行为。

后续要补：

- 更多业务模块 smoke 覆盖有权限/无权限数据集。

## 4. 多租户

当前策略：

- 单租户先行。
- 租户就绪。
- 不实现真实 tenant model。

进入真实多租户前必须先完成：

- `system_tenant` 设计。
- 租户识别方式。
- 用户与租户关系。
- 角色、菜单、权限、配置的作用域。
- 审计与导出的 tenant 过滤。
- 唯一键迁移策略。

## 5. SSO/OIDC

SSO 必须在身份源明确后实现。

在此之前，本线允许先做 `sso-ready` 预埋：

- provider 抽象接口
- 外部身份绑定模型
- 配置契约与审计字段
- 本地登录兜底策略
- 登录页与安全中心的非伪能力约束

进入实现前必须先完成：

- provider 类型。
- 回调域名。
- 本地用户绑定策略。
- 管理员本地登录兜底策略。
- 审计和注销语义。

## 6. 登录风控

当前已有：

- 来源级失败次数节流与临时锁定。
- 来源锁定、账号锁定安全事件落表和后台查询。
- 高敏动作二次验证。

后续增强：

- 新设备识别。
- 异地登录识别。
- 高频失败登录阻断。
- 高风险登录强制 MFA。

## 7. 业务模块自动化验收

目标：

- 每个业务域至少有一条固定 smoke。
- 报告归档到验收记录。
- 覆盖菜单、权限、i18n、审计和数据范围。

CMDB 应作为第一批样板。

当前 CMDB Host 已具备后端自动化样板：

- `go test ./backend/modules/business/cmdb/host` 覆盖 `dept_and_children` 只返回当前部门及下级部门数据。
- `go test ./backend/internal/middleware` 覆盖中间件从角色策略扩展部门树。
- `go test ./backend/pkg/database` 覆盖 `WithDataScope` 对展开后的 `DeptIDs` 生成过滤条件。

## 8. 平台级产品化控制台骨架

### 8.1 通知中心

目标：

- 从占位 inbox 演进到真实消息中心骨架

最低要求：

- 消息类别
- 已读/未读
- 来源域
- 跳转目标
- 审计或生成来源

### 8.2 审批流骨架

目标：

- 先建立后台通用“待审批 / 已处理 / 我的提交”产品位与接入契约

最低要求：

- 状态模型
- 指派/处理语义
- 审计与权限边界
- 至少一条真实样板流

### 8.3 调度中心 / 报表中心 / 监控告警中心

目标：

- 先建立模块位、接入契约和最小治理闭环，再决定具体深实现

最低要求：

- 模块归属
- 菜单与权限模型
- 列表 / 详情 / 执行动作边界
- 审计与告警留痕

## 9. P2 完成定义

P2 不能以“代码存在”为完成标准。

完成标准：

- 有设计合同。
- 有数据模型。
- 有权限和审计。
- 有迁移策略。
- 有自动化验证。
- 有回滚或禁用策略。
