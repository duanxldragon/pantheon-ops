# P2 规模化能力路线图

更新时间：2026-05-05

类型：Design
归属层：platform / system/auth / system/iam / system/config / business/*
状态：Active

本文用于收口 P2：数据权限、真实多租户、SSO/OIDC、登录风控和业务模块自动化验收。

---

## 1. 总原则

P2 不应做成一轮“大爆炸重构”。

正确顺序：

1. 数据权限中间件和角色策略。
2. 业务模块按 `DataScopeReq` 固定接入。
3. 租户就绪审查和租户字段策略。
4. 真正多租户模型。
5. SSO/OIDC。
6. 登录风控。
7. 业务 smoke 和报告归档常态化。

## 2. 数据权限

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

## 3. 多租户

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

## 4. SSO/OIDC

SSO 必须在身份源明确后实现。

进入实现前必须先完成：

- provider 类型。
- 回调域名。
- 本地用户绑定策略。
- 管理员本地登录兜底策略。
- 审计和注销语义。

## 5. 登录风控

当前已有：

- 来源级失败次数节流与临时锁定。
- 来源锁定、账号锁定安全事件落表和后台查询。
- 高敏动作二次验证。

后续增强：

- 新设备识别。
- 异地登录识别。
- 高频失败登录阻断。
- 高风险登录强制 MFA。

## 6. 业务模块自动化验收

目标：

- 每个业务域至少有一条固定 smoke。
- 报告归档到验收记录。
- 覆盖菜单、权限、i18n、审计和数据范围。

CMDB 应作为第一批样板。

当前 CMDB Host 已具备后端自动化样板：

- `go test ./backend/modules/business/cmdb/host` 覆盖 `dept_and_children` 只返回当前部门及下级部门数据。
- `go test ./backend/internal/middleware` 覆盖中间件从角色策略扩展部门树。
- `go test ./backend/pkg/database` 覆盖 `WithDataScope` 对展开后的 `DeptIDs` 生成过滤条件。

## 7. P2 完成定义

P2 不能以“代码存在”为完成标准。

完成标准：

- 有设计合同。
- 有数据模型。
- 有权限和审计。
- 有迁移策略。
- 有自动化验证。
- 有回滚或禁用策略。
