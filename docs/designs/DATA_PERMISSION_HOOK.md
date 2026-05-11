# 数据权限架构钩子 (Data Permission Hook)

更新时间：2026-05-05

类型：Design
归属层：system/iam
状态：Active

本文定义 Pantheon 平台的数据权限（行级过滤）架构占位规范，用于支持未来“部门级”、“本人级”或“自定义维度”的数据隔离需求。

## 1. 核心契约

### 1.1 数据结构 (`backend/pkg/common/data_scope.go`)
- `DataScopeReq`: 封装了当前用户的身份标识（UserID, DeptID, RoleKeys）以及资源标识。

### 1.2 GORM 钩子 (`backend/pkg/database/scope.go`)
- `WithDataScope(req *common.DataScopeReq)`: 这是一个标准的 GORM Scope。
- **规范**：所有涉及“多行数据读取”的业务方法（如 `ListXXX`, `ExportXXX`）**必须**接受该参数并在查询时注入。

## 2. 使用范例

### 后端 Service 层
```go
func (s *OrderService) ListOrders(query *OrderQuery, dataScope *common.DataScopeReq) (*OrderPageResp, error) {
    db := s.db.Model(&Order{}).Scopes(database.WithDataScope(dataScope))
    // ... 其他查询逻辑
}
```

### 后端 Handler 层
```go
func (h *OrderHandler) List(c *gin.Context) {
    dataScope := common.GetDataScope(c)
    resp, err := h.svc.ListOrders(&query, dataScope)
    // ... 
}
```

## 3. 未来扩展路径 (Roadmap)

`WithDataScope` 已具备最小安全语义：未设置 `Mode` 或 `Mode=all` 时保持历史行为；`dept / dept_and_children / custom` 模式如果缺少部门上下文，会返回空结果，避免 `dept_id=0` 用户在组织可选场景下被误解释为“拥有全部部门数据”。

当前已补齐的 P2 基线：

1. **中间件注入**：`DataScopeMiddleware` 已在 `JWTAuthMiddleware` 之后构造 `DataScopeReq`，并从 `system_role_data_scope` 读取角色数据范围策略。
2. **策略管理页**：`/system/permission` 的“数据权限”页已支持按角色配置 `all / self / dept / dept_and_children / custom`。
3. **业务接入样板**：`business/cmdb/host` 列表已接入 `common.GetDataScope(c)` 与 `database.WithDataScope(dataScope)`。
4. **脚手架集成**：业务模块生成器已提供 `enableDataScope` / `dataScopeMode` 契约，新生成模块可默认包含数据权限注入点。
5. **部门树展开**：`dept_and_children` 已基于 `system_dept.ancestors` 展开当前部门及下级部门，并写入 `DataScopeReq.DeptIDs`。
6. **失败兜底**：用户部门、角色策略或部门树加载失败时会记录日志；部门树不可用时回退为当前部门，避免误放大数据范围。
7. **业务回归样板**：`business/cmdb/host` 已新增 `dept_id` 数据范围字段，并用后端测试固定有权限/无权限数据集过滤行为。

### 3.1 多角色合并规则

当前数据权限按“角色授权叠加”处理，不引入 deny 语义：

- `admin` 角色绕过数据权限过滤。
- 任一角色为 `all` 时，最终范围为 `all`。
- 多个 `custom` 角色合并部门 ID 集合。
- 无 `all/custom` 时，按 `dept_and_children -> dept -> self` 选择最宽可见范围。
- 数据库返回顺序不得影响最终结果，后端必须用测试固定该行为。

如果未来需要“拒绝优先”或“最小权限交集”模型，应新增独立策略类型，而不是复用当前角色范围字段隐式表达。

后续扩展仍按以下顺序推进：

1. **租户支持**：未来进入真实多租户后，在统一 scope 中注入 `tenant_id = ?` 条件。
2. **高级策略表达式**：如确有需要，再评估 Casbin 条件表达式到 SQL 过滤片段的映射。
3. **业务 smoke**：继续为更多业务域补有权限/无权限的数据集验证。

## 4. 为什么现在做占位？

- **预防大规模重构**：如果在系统上线后再加数据权限，需要修改成百上千个 Service 方法签名。
- **确立开发文化**：让业务开发者从第一天起就意识到“数据是有归属和边界的”。
