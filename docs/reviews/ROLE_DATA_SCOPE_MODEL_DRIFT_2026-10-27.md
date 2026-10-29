# Role data-scope model 漂移核查与决策（RC-022 / S3 波次）

- **任务包**: `.harness/tasks/2026-10-27-s3-role-data-scope-model/manifest.json`
- **决策日期**: 2026-10-28（maintainer-delegated）
- **结论**: 两个 model 逐字段一致 → 走任务包第一分支：**删除 role 侧镜像，role 直接引用
  `permission.PermissionRoleDataScopePolicy`**。零 DDL、零迁移、零权限行为变更。

## 1. 两个 model 的比对（漂移报告）

| 字段 | `role.roleDataScopePolicy`（已删除） | `permission.PermissionRoleDataScopePolicy` | 一致？ |
|------|------|------|------|
| TableName | `system_role_data_scope` | `system_role_data_scope` | ✅ |
| ID | `uint64` `primaryKey;autoIncrement` | 同 | ✅ |
| TenantID | `uint64` `not null;default:0;uniqueIndex:uk_role_data_scope_tenant_role,priority:1` | 同 | ✅ |
| RoleKey | `string` `size:64;not null;uniqueIndex:uk_role_data_scope_tenant_role,priority:2` | 同 | ✅ |
| Mode | `string` `size:32;not null;default:'all'` | 同 | ✅ |
| DeptIDs | `string` `type:text` | 同 | ✅ |

机械 diff（字段名 + 完整 gorm tag 逐字符比对）结果：**完全一致**。差异只有注释措辞。
`grep -o` 级别比对命令与输出存于 `.harness/evidence/2026-10-27-s3-role-data-scope-model/model-diff.md`。

## 2. 与 live schema 的比对

`system_role_data_scope` 的 schema 由三个迁移演进（本波次未改任何迁移）：

- **000001_init_schema**: `id, role_key(单列唯一 idx_system_role_data_scope_role_key), mode, scope_defs, created_at, updated_at` —— 无 tenant_id、无 dept_ids
- **000002_align_runtime_schema**: `ADD COLUMN dept_ids TEXT`，并从 scope_defs 回填
- **000020_tenant_phase1_core_tables**: guarded `ADD COLUMN tenant_id` + `ADD INDEX idx_role_data_scope_tenant_role (tenant_id, role_key)`（**非唯一**）

**记录的差异（非本波次修复项，迁移与 DDL 在 doNotTouch）**：

1. 索引形态：迁移侧是非唯一 `idx_role_data_scope_tenant_role`；两个 model 都声明
   `uniqueIndex:uk_role_data_scope_tenant_role`。迁移管理的库由 AutoMigrate 追加创建唯一索引；
   仅 bootstrap 的库直接从 AutoMigrate 得到唯一索引。**两个 model 之间在此项上仍然一致**
   （同 tag），不存在 model-vs-model 漂移；这是 model-vs-migration 的历史命名/唯一性错位，
   属于未来迁移治理包的范围。
2. 遗留列：`scope_defs`（000002 已回填到 dept_ids 但未删列）、`created_at/updated_at` 不被
   任何 model 映射 —— GORM 只映射声明字段，无行为影响。

## 3. 决策：删镜像，引用 permission 的导出 model

两个候选分支（任务包 scope）：

- ✅ **采用**：model 一致 → 共享 home + 删镜像。共享 home 选 **permission 包的导出 model**
  （data-scope 控制台的属主域）。关键事实：**role 包此前已经 import permission**
  （`role_policy_sync.go`、`role_data_scope_preserve_test.go`），import 方向早已存在且无环
  （permission 不依赖 role，`go list -deps` 验证）。原注释「避免跨目录 import」在
  role_policy_sync.go 引入 import 之后就失去了成立前提。
- ❌ 弃用：保留镜像 + 漂移 checker。既然 import 方向已存在，checker 只是把可结构性消除的
  重复变成一个需要长期维护的门禁，劣于删除。

落地：

- `role_service.go` 删除 `roleDataScopePolicy`，4 处引用（AutoMigrate / load / write / delete /
  HasTable）改指 `permission.PermissionRoleDataScopePolicy`；
- role 包 3 个测试文件的种子数据同步改指（role_data_scope_preserve_test 本就 import permission）；
- delete 路径仍然写同一张表（`system_role_data_scope`），行为零变化；
- `database/migrations/**`、`permission/**`、`database/**` 均未触碰（符合 doNotTouch）。

## 4. 验证

- `go build ./modules/system/iam/role/` 通过；`go test ./modules/system/iam/role/... ./modules/system/iam/permission/...` 全绿
- `grep -rn "roleDataScopePolicy" modules/system/iam/role/` → 0 命中
- data-scope 五种模式解析路径未触碰（permission_data_scope.go 零改动），全部既有
  permission data-scope 测试（含 2026-10-19/10-21 的两租户回归）通过

## 5. Runtime 证据 gap

「schema dump 对照」在本机不可执行（MySQL 凭据不可用，见 S3 主任务包 evidence），以
000001/000002/000020 三个迁移文件作为 schema 事实来源完成比对；运行态比对由 CI 的
MySQL 测试承担。
