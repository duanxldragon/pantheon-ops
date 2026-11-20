# O3 Change Preview And Risk

- Priority: `P0`
- Layer: `business/deploy + business/k8s + business/cicd`
- Status: `blocked`
- Base Dependency: B3 `ChangeDiff`
- Business Dependency: immutable intent/current-target version contracts

## Outcome

在 Deploy、K8s Release、Argo CD sync 和 rollback 执行前展示可追踪的差异、影响对象、风险和回滚条件。

## In Scope

- 统一 preview envelope：intent ID/version、current version、target version、objects、diff、risk、rollback。
- Deploy package/template/parameter preview。
- K8s manifest/image/config/resource preview。
- Argo CD sync/rollback version tree and resource diff。
- conflict-on-change：预览后对象变化必须重新预览。

## Out Of Scope

- 不在浏览器执行可信 diff 或安全判断。
- 不保证所有外部系统都能自动回滚。
- 不展示秘密、完整 credential 或受限配置值。

## Expected Files

- 各业务域 provider-owned preview endpoint、DTO、tests。
- `frontend/src/modules/business/**` 中基于 Base `ChangeDiff` 的业务组合。
- permission/audit/i18n/smoke 和风险 fixture。

## Acceptance

1. preview 与 execute 绑定同一 intent/version，版本漂移返回 409/conflict。
2. 新增/修改/删除/冲突和未变化项可区分，风险不只靠颜色表达。
3. 风险摘要指出目标数量、环境、不可逆项、健康检查与回滚可用性。
4. 秘密字段在进入 UI 前遮蔽，下载/复制同样遵守权限和审计。
5. 大 diff 可折叠/搜索/跳过未变化项，mobile 至少可完成审阅和取消。

## Evidence

- backend：intent binding、TOCTOU conflict、permission/DataScope、redaction、audit。
- domain fixtures：Deploy、K8s、Argo CD、rollback 的 safe/high-risk/irreversible cases。
- Playwright：preview -> confirm -> conflict -> refresh，keyboard and mobile。
- external evidence：可用测试集群/Argo CD 的实际 diff，或 endpoint/credential gap。
- screenshots：desktop/mobile x low/high-risk/conflict/forbidden x light/dark。

## Gates

- 新 schema/public API、外部 production mutation、风险策略影响发布权限、Base lock。
