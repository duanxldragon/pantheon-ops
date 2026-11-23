# 业务权限绑定与部署任务数据可见性

- Status: completed — code + local live verification passed; remote re-probe and maintainer final acceptance open
- Updated: 2026-11-23
- Baseline: a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc
- Owner: 业务授权负责人
- Priority: P1
- Activation: core
- Dependencies: T01

## Result

- deploy 权限绑定：package(6)/template(6)/task(10, 不含 export)/credential(4) = 26 键；详情与 `task:export` 刻意不绑定。
- credential 4 键已入 seed F 菜单（Sort 12–15）+ zh/en 后端 i18n；前端 fallback 与模块 locale 同步补齐。
- SEC-002 嵌套主机范围过滤修复；NEW-01 scoped 列表改为 id-only 候选 + 单次批量 hostReader + 精确 total + 按可见 ID 取页。
- 验证：全量 deploy 套件 PASS（1047.796s 真实 MySQL）；本地 live 角色矩阵 18/18 PASS；menu-contract / method-consumption / boundaries / overlay 全绿。

## Next Atomic Action

若需远端同候选证据：在 k8s33/k8s35 部署实例用 `0586a9e2…` 重跑角色矩阵；否则由维护者裁定本地 live 等价证据并做最终功能验收。

## Evidence

- `.harness/evidence/2026-11-20-ops-business-permissions-and-task-visibility/commands.json`
- `summary.md`（实施、验证、验收映射、七项安全检查）
- `review.md`（findings-first 独立审查；P1 已修复，P2 已登记）
- `t03-role-matrix-local.json`（18 探针 + 清理记录）
- 候选构建：[codex-build-t03-20261123-162526.json](../../../.harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/codex-build-t03-20261123-162526.json)
- 矩阵追加：`2026-11-20-ops-review-live-runtime-acceptance/acceptance-matrix.json` 的 V1/V5/deploy-lane `t03Recheck`

## Known Gaps

1. **REMOTE-REPROBE**：远端部署实例未用 T03 候选重跑；本地 live 为等价证据。
2. **NON-DEPLOY-BINDINGS**：bizscope/cicd/filedist/k8s 业务域仍 default-deny，未绑定。
3. **SEC-004**：credential 表无 owner/scope，CRUD 无创建者/范围过滤（密文不外泄；暴露元数据与轮换能力）。
4. **CREDENTIAL-DETAIL**：无 `GET /credentials/:id`。
5. **BASE-MATCHER**：`task:detail` 与详情路由需 Base 静态路由守卫正式 release。
6. **RACE**：`-race` 全量按 T02 维护者裁定接受 TSan 限制，未重跑。
