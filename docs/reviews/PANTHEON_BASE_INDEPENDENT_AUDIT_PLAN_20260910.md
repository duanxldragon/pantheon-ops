---
title: Pantheon Base 独立审计与租户补全计划
doc_type: Assessment
layer: platform / system/auth / system/iam / system/org / system/config / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-09-10
---

# Pantheon Base 独立审计与租户补全计划

## 1. 先给结论

当前 `pantheon-base` 可以继续作为**单租户企业后台底座**开发，但不能把它描述为已经具备多租户能力。真实租户实现目前是从零开始的横切工程，不是补一个 `Tenant` 表或增加一层中间件。

我的独立判断是：

1. **现在不建议直接开工全量多租户。** 现有代码、数据库和设计仍按单一平台空间运行，贸然把 `tenant_id` 扩散到所有表，会同时改变登录、会话、角色、菜单、权限、组织、设置、审计、导出、上传、统计和生成器。
2. **现在必须补租户就绪约束。** 这部分成本低，可以防止新业务继续写死全局唯一键、全局查询和无法切分的审计/导出逻辑。
3. **真正进入多租户前，先完成六项设计决策，再做一个可回退的垂直切片。** 六项决策是：租户主模型、租户识别、用户与租户关系、角色/菜单/权限/配置作用域、审计与导出过滤、唯一键迁移策略。
4. **原多租户设计中的 16 小时实现估算不可信。** 在未盘点全部读写路径、迁移策略和隔离测试前，合理估计是：前置设计与护栏 3--7 个工作日；单工程师完成可生产验证的 Phase 1 MVP 约 6--10 周。这个估算是中等置信度，业务表和 `pantheon-ops` 接入面增加时还会变大。

这份文档是计划和决策依据，不批准任何生产 schema、权限模型或认证链路变更。

## 2. 审计范围与证据

本次只读核对了：

- `DESIGN.md`、`docs/README.md`、`docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md`、`docs/designs/MULTI_TENANT_DESIGN.md`、`docs/designs/P2_SCALE_ROADMAP.md`；
- `docs/contracts/` 下 platform、auth、IAM、org、config 合同；
- `docs/reviews/` 下六方交叉评审、2026-09-10 功能补充审查和设计补充审计；
- 数据库初始化脚本、12 个 migration、认证/会话/token、数据权限、组织/IAM、设置、审计、上传、平台 dashboard、低代码生成器；
- 后端全量测试：`go test ./...` 通过；本次实测整体覆盖率约 **13.1%**，各高风险包差异较大。

证据与推断分开处理：下面标为“事实”的内容来自代码或文档直接核对；“判断”是基于这些事实对改造风险的推断。

## 3. 当前状态与需要补全的事项

### 3.1 P0：先补事实边界和伪能力

| 优先级 | 需要补全 | 当前事实 | 为什么先做 |
|---|---|---|---|
| P0 | 租户状态声明 | `MULTI_TENANT_DESIGN.md` 是设计稿；没有 `Tenant` 模型、租户表、migration、租户 middleware 或租户管理页。现有 `system_user`、`system_role`、`system_dept`、`system_post` 等仍是单空间模型。 | 防止团队把设计完成误当成实现完成，低估隔离工程。 |
| P0 | 生成器伪能力 | `backend/internal/scaffold/contract.go`、`frontend/src/modules/lowcode/generator/schema.ts` 和 `ModuleWizard.tsx` 暴露 `tenant` 数据范围，但运行时 `DataScopeMode` 没有该模式。 | 这是当前唯一直接面对开发者的静默失效选项，必须移除或禁用并明确标注未实现。 |
| P0 | 租户就绪检查进入模板和 DDL 评审 | 现有设计已要求判断 `tenant_id`、租户内唯一、查询/导出/统计过滤和审计维度，但业务落地仍需逐项留痕。 | 不改变运行态，却能阻止新业务继续扩大未来迁移面。 |
| P0 | 质量指标 | CI 后端覆盖率默认阈值为 11%，前端为 0%；本次实测整体约 13.1%。 | 在扩大表结构和权限分支前，必须先提高高风险路径的回归能力。 |

### 3.2 P1：进入租户前必须补齐的设计合同

| 领域 | 必须回答的问题 | 当前缺口 |
|---|---|---|
| Tenant master | `id` 类型、稳定 code、状态机、软删除、生命周期、平台管理员边界、是否支持暂停/归档。 | 设计稿给了字段示例，但没有与现有命名、错误码、审计和权限合同接合。 |
| User membership | 一个用户能否属于多个租户？登录后是否选择租户？平台管理员是否可跨租户？成员离开租户如何处理已有会话？ | 当前 token/session 只保存用户、角色和 session 信息，没有 `tenant_id` 或 membership 语义。见 `backend/pkg/authtoken/token.go`、`backend/modules/auth/session/session_model.go`。 |
| Tenant resolution | Phase 1 是否只信登录后 token？哪些请求允许平台级上下文？Header、子域名、Cookie、URL 是否全部禁用？ | 多租户设计同时列出 custom domain、subdomain、Header、JWT 四种方式。若 Header 在 JWT 前，客户端可伪造跨租户上下文；这是设计风险，必须先收敛。 |
| Resource scope | 哪些是 platform resource，哪些是 tenant resource，哪些支持 platform default + tenant override？ | 角色、菜单、权限、字典、设置、审计、上传、任务和动态模块的作用域尚未形成唯一矩阵。 |
| Authorization | 租户管理员能管理什么？平台管理员是否需要显式跨租户权限？Casbin subject/policy 如何包含 tenant？数据权限与租户过滤谁先执行？ | 现有权限策略和 `system_role_data_scope` 以全局 role key 为核心；`admin` 会绕过数据权限。直接增加列不能自动获得租户隔离。 |
| Query boundary | 查询、写入、批量操作、导出、统计、关联查询、后台任务和生成器如何强制 tenant filter？ | 现有 `database.WithDataScope` 只处理部门/用户数据范围；它不是租户隔离，也不会自动包住所有直接 GORM 查询。 |
| Migration | 旧数据如何归入默认租户？如何处理全局唯一键、外键、在线变更、失败回滚、双写/灰度和备份恢复？ | 设计稿只列了“建表、回填、加列、改代码、开关”，没有可执行 runbook 和回退条件。 |
| Cross-cutting data | session、MFA factor/challenge、security event、login/operation log、upload object key、Redis key、dashboard aggregate、i18n/cache 是否按租户隔离？ | 这些模型和路径当前没有租户维度，且部分是跨请求或异步数据。 |

### 3.3 P2：可以后置，不阻塞单租户交付

- 套餐、计费、订阅、配额和 License；
- 子域名/自定义域名发现；
- 独立 schema 或独立数据库；
- 租户级缓存管理、租户级任务调度、租户分析看板；
- 完整租户品牌中心和自助注册。

这些能力只有在租户核心隔离和真实业务场景稳定后才有投入价值。不要把它们混入第一阶段的“租户地基”。

## 4. 租户改造为什么会影响大

### 4.1 认证和会话不是局部字段变更

当前 access session 的 Redis payload 只有用户、用户名、角色、会话和客户端信息；数据库会话按 `user_id` 管理；MFA factor/challenge 也按 `user_id` 管理。要支持多租户，必须先决定“用户是平台主体还是租户成员”，再决定 token 的 tenant claim、session 归属、注销和切换语义。否则会出现同一用户在不同租户间串会话、串角色或错误继承权限。

### 4.2 IAM、Casbin 和数据权限必须一起设计

`system_role.role_key`、角色权限、角色菜单和 Casbin rule 当前按全局 role key 组织；`DataScopeReq`/`WithDataScope` 只实现 `all/self/dept/dept_and_children/custom`。因此“给现有表加 `tenant_id`”无法独立解决授权问题：角色作用域、策略 subject、部门树、管理员绕过逻辑和租户过滤必须形成同一条授权链。

### 4.3 迁移和唯一键是主要返工来源

当前用户、角色、岗位、设置、字典等表存在全局唯一索引，例如用户名、角色 key、岗位编码、设置 key 和字典编码。未来若这些对象需要租户内唯一，必须先做重复值盘点，再制定索引迁移和冲突处理规则。直接把索引改成 `(tenant_id, key)` 会在真实数据上失败，也可能改变现有登录和权限行为。

### 4.4 查询漏加过滤会形成高危泄漏

系统中存在大量直接 `db.Table/Where/Joins/Count/Scan` 的服务代码，平台 dashboard、审计导出、会话管理、上传对象路径、动态模块生成代码都不天然经过统一租户 scope。设计上依赖“每个 repository 记得 `WithTenant`”不够，必须建立可检查的入口、默认拒绝行为和跨租户测试。

## 5. 推荐执行顺序

### 阶段 0：决策包，不改代码（1--2 个工作日）

产出一页决策记录，冻结以下答案：

- 目标是内部多组织部署、SaaS，还是两者都要；
- 用户是否允许属于多个租户；
- 登录时如何确定租户，Phase 1 只采用哪一种可信来源；
- 平台管理员、租户管理员、普通成员的边界；
- 需要隔离的业务表清单和明确不隔离的 platform resource；
- 是否接受共享库/共享 schema 作为 Phase 1；
- 单租户兼容模式的保留期限和退出条件。

**停止条件**：以上任一项未定，不进入 schema 或认证实现。

### 阶段 1：租户就绪护栏（0.5--2 个工作日）

只做低风险、可回退工作：

1. 移除或禁用生成器的 `tenant` 选项，补回归测试和 UI/文档说明。
2. 在业务模块模板、DDL 评审和验收矩阵中加入租户字段、租户内唯一、查询/导出/统计、审计维度四项检查。
3. 建立“全局唯一键登记表”和“platform resource / tenant resource 矩阵”模板。
4. 将多租户设计稿的四种识别方式改为待决策项，避免实现者按示例直接支持 Header 或多路解析。
5. 把当前覆盖率基线和高风险包目标写入任务包；不在没有测试的情况下扩展隔离面。

**验收**：单租户行为不变；生成器不再提供静默失效模式；文档门禁、后端测试和前端单测通过。

### 阶段 2：租户合同和迁移 runbook（3--7 个工作日）

先写设计和测试夹具，不接入生产路由：

- `system_tenant`、membership、状态和管理员边界；
- `TenantContext` 的来源、缺失行为、平台上下文和可信度；
- token/session/MFA/security event 的租户语义；
- role/menu/permission/data-scope/Casbin 的作用域模型；
- 每张表的 scope 分类、字段、外键、索引和唯一键变化；
- login、list、detail、create/update/delete、batch、export、count、dashboard、async job 的统一过滤规则；
- 默认租户回填、冲突处理、在线迁移、备份、回滚、灰度和观测指标。

**验收**：两租户 fixture 能证明同名用户/角色在规则允许时共存；篡改 tenant claim、伪造 Header、越权 ID、批量请求、导出和统计均有预期拒绝结果；runbook 能在副本数据库完成演练。

### 阶段 3：单垂直切片试点（约 1--2 周）

选择一个真实业务模块或独立测试模块作为 canary，完成：

- token 到 context 的可信传递；
- tenant-aware 查询、写入、唯一键和导出；
- 业务权限 + 部门数据权限 + 租户过滤的组合；
- 审计、文件对象 key、缓存 key 的隔离；
- 两租户交叉访问、批量操作和并发测试；
- 单租户兼容模式下的回归。

试点期间不改全仓库系统表，不启用域名发现，不做套餐计费，不增加第二种租户识别方式。

**停止条件**：任一跨租户读取/写入、默认放行、导出泄漏、缓存串租户或迁移不可回滚，立即回到合同阶段；不得靠临时 if 分支继续扩大范围。

### 阶段 4：核心隔离实现（约 3--6 周）

只有阶段 3 通过后，才按表和调用链分批推进：

1. Tenant master 与 membership；
2. auth/session/token/MFA 语义；
3. user、role、dept、post、role policy、menu/permission 作用域；
4. 配置、字典、审计、导出、上传和缓存；
5. 业务表、动态模块和生成器模板；
6. dashboard、后台任务和所有跨域聚合；
7. 数据库索引、外键、回填和灰度迁移。

每个切片都必须有 migration、contract test、cross-tenant test、回滚说明和运行态证据。禁止一次性提交“全表加列 + 全部服务改写”的大 diff。

### 阶段 5：生产验证（1--2 周）

完成迁移副本演练、性能基线、并发/缓存验证、审计检索、备份恢复、故障回退、渗透测试和 focused smoke。此阶段结束前，系统只能宣称“受控试点”，不能宣称“多租户生产就绪”。

## 6. 目前不应开工的做法

- 不要按 `MULTI_TENANT_DESIGN.md` 的示例直接实现 custom domain、subdomain、Header、JWT 四路优先级。
- 不要只给 `system_user` 加 `tenant_id`，而把角色、Casbin、菜单、配置、审计和导出留在全局。
- 不要把部门 `DataScope` 当成租户隔离；它们是两个不同维度，必须明确组合顺序。
- 不要把 `Tenant Admin` 做成 `role_key == "admin"` 的别名；需要显式作用域和跨租户权限边界。
- 不要在没有迁移副本、冲突盘点和回滚方案时执行 NOT NULL、外键或唯一索引变更。
- 不要用“所有查询手工加 `tenant_id`”作为唯一安全措施；必须有统一上下文、默认拒绝和自动化泄漏测试。
- 不要把套餐、计费、域名、独立库和完整品牌中心塞进第一阶段。

## 7. 完成标准

租户阶段只有同时满足以下条件，才可以进入下一阶段或对外宣称完成：

- 设计合同、scope 矩阵、唯一键登记表和 migration runbook 已评审并互相链接；
- 单租户回归、双租户隔离、越权、批量、导出、统计、缓存、文件和异步路径均有测试；
- 所有租户资源的读写入口都有统一过滤，平台资源的跨租户行为有显式授权；
- auth/session/token/MFA/Casbin/数据权限的作用域语义一致；
- 迁移已在数据副本完成演练，含冲突、失败和回滚证据；
- 高风险包覆盖率达到任务包设定目标，CI 阈值不再低于真实基线；
- 有运行态 smoke、审计检索和性能证据；
- Reviewer 能指出受影响调用子图、边界穿越、默认放行点和剩余风险；
- 任一未实现能力都不会在 API、前端向导或文档中以“已支持”出现。

## 8. 最终建议

你担心“租户会把系统打乱”是有依据的，但风险来自**没有先冻结边界就全面改造**，不是来自租户这个目标本身。可控的做法是保留当前单租户运行态，先完成低成本 tenant-ready 护栏，再用一个真实垂直切片验证合同，最后分批迁移。这样即使在阶段 2 或 3 发现用户关系、权限作用域或唯一键模型不合适，也只需回退设计和试点，不会把整个底座拖进半完成状态。

在当前阶段，我建议的补全顺序是：

1. 先修生成器租户伪能力并补租户就绪检查；
2. 先把覆盖率和高风险路径回归能力提高到可承受水平；
3. 冻结六项租户合同和迁移 runbook；
4. 做单垂直切片和双租户隔离证据；
5. 通过试点后再决定是否投入 6--10 周做核心多租户 MVP。

如果业务仍主要是内网单组织，阶段 1 就足以支撑近期开发；只有当 SaaS、集团多组织或明确的租户隔离需求成为已确认目标时，才进入阶段 2 以后。
