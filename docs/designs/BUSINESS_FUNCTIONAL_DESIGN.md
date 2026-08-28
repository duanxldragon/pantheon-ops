# Pantheon-Ops 全局功能设计

更新时间：2026-08-28
类型：Design
归属层：business（全局）
状态：Active（V1 冻结基线；各模块实现状态以 `../V1_RELEASE_CHECKLIST.md` 为准）

本文是 pantheon-ops 的全局功能视图：能力地图、跨模块流程、状态机、安全模型与 V1 能力追溯。此前本文档在 `docs/README.md` 索引中存在悬空引用，本文为 2026-08-28 评审后补建版本，结论与《pantheon-ops v1.0 冻结前综合评审报告》一致。

## 1. 能力地图（含 V1 实现状态）

状态图例：✅ 已实现且验证 / ⚠️ 已实现有缺陷 / ❌ 壳级未实现（V1 下架） / 📐 规划中

| 域 | 能力 | 模块 | V1 状态 |
|---|---|---|---|
| 底座 | 认证会话（opaque token、轮换、注销失效） | system/auth | ✅ |
| 底座 | 四层权限（菜单/页面/操作/Casbin API） | system/iam | ✅ |
| 底座 | 操作审计（非 GET 全覆盖、脱敏） | system/audit | ⚠️ 缺 before/after |
| 底座 | 统一加密（`v1:`/`enc:v1:` AES-256-GCM） | pkg | ✅ |
| 资源 | 主机台账（生命周期状态机+乐观锁） | business/cmdb | ⚠️ UpdateStatus 绕过 retire（P0） |
| 资源 | 分组（树形+条件）、标签规范 | business/cmdb | ✅ |
| 资源 | 业务域（绑定/解绑事务+锁） | business/bizscope | ⚠️ 删除引用检查不全（P0） |
| 资源 | 应用/服务/实例目录（VM/K8s target） | business/service | ⚠️ 前端仅只读 |
| 交付 | 部署包/模板/任务三层模型 | business/deploy | ✅ |
| 交付 | SSH 执行（幂等键/checkpoint 续跑/租约/重试） | business/deploy | ✅ |
| 交付 | 组件固定模板（nginx/redis/jdk，systemd） | business/deploy | ✅ |
| 交付 | K8s 集群纳管（kubeconfig 加密+轮换+探测） | business/k8s | ✅ |
| 交付 | K8s 资源（NS/CM/Secret/Workload 只读+部分操作） | business/k8s | ⚠️ 缺 Update |
| 交付 | 镜像发布 Release（幂等指纹+回滚） | business/k8s | ✅ |
| GitOps | ArgoCD 应用纳管/同步/回滚 | business/cicd | ❌ 伪实现，V1 下架，v1.2 重做 |
| 备份 | Velero 备份/恢复/定时 | business/k8s/backup | ❌ 伪实现，V1 下架，v1.2 重做 |
| 观测 | 主机/K8s 纳管监控 | observability | 📐 v1.1/v1.2（见专项设计） |
| 观测 | 平台自监控 | grafana/ + pkg/metrics | ✅（仅自监控） |
| 商业 | 模块开关/License | system | 📐 v1.1（见专项设计） |

## 2. 跨模块流程（V1 主链路）

### 2.1 资源管理闭环

```
CMDB 主机（pending→assigned） → 业务域绑定（事务+锁）
→ 应用/服务/实例登记（capability 校验 target 与业务域一致）
→ 主机生命周期（retire 前查活跃服务实例引用）
```

### 2.2 部署交付闭环

```
部署包（fixed/template 模板） → 创建任务（业务域圈定目标主机）
→ 启动（幂等键防重放 / 主机租约防并发 / Attempt worker 认领）
→ SSH 执行（组件健康探测：已达标则跳过 = 幂等）
→ 步骤 checkpoint（失败重试从失败步续跑）
→ 结果回写（事件流+日志块，错误信息脱敏）
→ ServiceInstance 状态联动（begin/finish）
```

### 2.3 K8s 交付闭环（V1 无 GitOps 段）

```
集群纳管（kubeconfig 加密入库+连通探测）
→ Namespace 绑定（AllowedActions 白名单）
→ Release 镜像发布（幂等指纹+rollout 轮询+回滚审计）
```

### 2.4 规划闭环（v1.1+）

```
主机监控：CMDB → node_exporter 下发（deploy 引擎） → file_sd → Prometheus
GitOps：Release/镜像 → ArgoCD Application → 集群状态回传
备份：业务域 → 集群 → namespace → Velero CRD → 状态 reconcile
```

## 3. 状态机汇总

| 对象 | 状态集 | 关键规则 |
|---|---|---|
| CMDB 主机生命周期 | pending→assigned→maintenance→retired | 乐观锁 state_version；retired 需引用检查；**唯一合法写通道为状态机 API** |
| 部署任务 | draft→pending→running→success/failed/canceled（含 pause_requested/paused/cancel_requested） | 事件流记录每次流转；reconciler 收敛悬挂态 |
| 部署步骤 | pending→running→success/failed/skipped/canceled | checkpoint 续跑依据 |
| 服务实例 | desired/observed/health + lifecycle_version | CorrelationID 幂等；Reconcile 超时收敛 |
| K8s 集群 | active/unreachable | 30s 探测 |
| 凭据 | 版本化（credential_ref_version） | 轮换不失效执行中任务 |

## 4. 安全模型（V1 事实）

1. 认证：opaque token（Redis 会话），HttpOnly+Secure+SameSite=Strict Cookie，CSRF 头，15min/7d。
2. 授权：Casbin RBAC + DataScope 数据范围 + SecureAction 危险操作二次确认。
3. 凭据：SSH 密码不落库（凭据引用 `v1:` 加密）；kubeconfig 独立表加密+版本轮换。
4. 审计：全局操作日志（含失败分类、敏感掩码）；**已知缺口**：ArgoCD/Velero 凭据明文（随伪实现一并 v1.2 修复）、before/after 不全（P1）。

## 5. UI 信息架构

```
工作台
├── 运维平台（业务域）
│   ├── CMDB（主机/分组/标签）
│   ├── 部署（包/模板/任务：列表-详情-日志三视图）
│   ├── 应用服务（应用/服务/实例）
│   └── K8s（集群/资源/发布）
└── 系统管理（用户/角色/菜单/审计/设置）
```

前端规范沿用 base：组件注册表 + 菜单 seed + 构建期契约检查；业务模块通过 `businessOverlayComponentRegistry.ts` 注入。

## 6. V1 追溯

- 评审基线：《pantheon-ops v1.0 冻结前综合评审报告》（2026-08-28）。
- 执行计划：`V1_FREEZE_REMEDIATION_DEV_PLAN.md`（P0/P1/P2 按优先级排序）。
- 发布状态：`../V1_RELEASE_CHECKLIST.md`（唯一事实源）。
- 已知遗留：P1 清单见评审报告第 6 节与开发计划 Phase 2。

## 7. 商业版图（v1.1 起）

免费版：底座 + CMDB + 业务域 + 部署引擎 + 主机监控。
商业版：K8s 全量 + GitOps + Velero 备份 + 流程编排。
技术基础：模块开关三层裁剪 + ed25519 离线 License（详见 `BUSINESS_MODULE_LICENSING_DESIGN.md`）。
