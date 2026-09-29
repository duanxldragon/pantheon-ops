---
title: 模块开关与 License 设计（商业化技术基础）
doc_type: Design
layer: system/config
status: Draft
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-28
---

# 模块开关与 License 设计（商业化技术基础）

> 状态说明：Draft（v1.1 目标；当前**不存在**任何模块启停与 License 机制，见 `.harness/v1.2-docs/V1_FREEZE_REMEDIATION_DEV_PLAN.md` Phase 3）

2026-08-28 评审确认：`capability.go` 是跨模块接口契约（端口/适配器），不是商业开关；模块集合由编译期 overlay（business-overlay.json + 注册表）决定，运行时不可裁剪。本文设计运行时模块启停与 License 校验。

## 1. 商业拆分目标

| 模块 | 免费版 | 商业版 |
|---|---|---|
| 底座（认证/RBAC/审计/组织/配置） | ✓ | ✓ |
| CMDB（主机/分组/标签） | ✓ | ✓ |
| 业务域 | ✓ | ✓ |
| 安装部署（SSH 组件生命周期） | ✓ | ✓ |
| 主机监控 | ✓ | ✓ |
| K8s（集群/资源/Release） | ✗ | ✓ |
| ArgoCD GitOps | ✗ | ✓ |
| Velero 备份恢复 | ✗ | ✓ |
| 流程编排 | ✗ | ✓ |

拆分逻辑：免费版锚定"内网资产+自动化运维"（解决装中间件靠人肉的真实痛点，获客钩子），商业版锚定"云原生治理"（K8s 多集群+GitOps+备份恢复，最清晰的付费理由）。

## 2. 模块开关机制（Module Registry）

### 数据模型

```sql
system_module (
  module_key     VARCHAR(64) PRIMARY KEY,  -- cmdb / deploy / k8s / cicd / backup / observability
  module_name    VARCHAR(128),
  tier           VARCHAR(16),              -- free | commercial
  enabled        TINYINT,                  -- 运行时开关
  builtin        TINYINT,                  -- 免费版内置模块不可关闭
  license_bound  TINYINT                   -- 是否受 License 控制
)
```

### 三层裁剪

1. **菜单层**：菜单 seed 按模块标记 module_key，装配动态菜单时 join system_module 过滤 disabled 模块（复用现有 retired_modules 清理机制的反向逻辑——retired 是"删"，这里是"藏"）。
2. **API 层**：路由装配处（contracts 层包一个 `ModuleGate(moduleKey)` 中间件），disabled 模块的 API 返回统一错误 `module.not_enabled`（403），而非 404——前端可提示"需升级商业版"。
3. **前端层**：`/user/info` 下发 enabled_modules，路由守卫与按钮按模块过滤；被隐藏模块的直接 URL 访问落到"功能未启用"页。

### 关键约束

- 编译期 overlay 不变（代码全量编译，运行时裁剪），避免维护两套构建。
- disabled 模块的后台任务（reconciler/轮询）一律不启动。
- License 失效时：商业模块 API 拒绝新操作，**已有数据可读**（只读降级），为升级留台阶。

## 3. License 机制

### 格式（ed25519 离线签名）

```json
{
  "licensee": "客户名称",
  "modules": ["k8s", "cicd", "backup"],
  "max_hosts": 200,
  "max_clusters": 5,
  "expires_at": "2027-08-28T00:00:00Z",
  "issued_at": "2026-08-28T00:00:00Z"
}
```

签名：私钥（厂商侧保管）ed25519 签名上述 payload，License 文件 = payload + signature。平台内置公钥验签。

### 设计决策

1. **离线优先**：私有化环境无外网，License 为文件导入（系统管理→License 管理），不联网激活、不心跳。过期靠本机时间（接受本机时间被改的风险，这是私有化 License 的固有取舍）。
2. **量级限制**：max_hosts 对齐 CMDB 主机数、max_clusters 对齐 K8s 集群数，超限时新增被拒绝、存量可用。
3. **到期行为**：商业模块只读降级（见上），数据永不锁死——私有化客户最恨数据劫持。
4. **审计**：License 导入/过期/降级事件入操作日志。

## 4. 实施边界

- 本设计不改变 Casbin 权限模型——模块开关在权限**之上**（先过模块门，再过权限门）。
- 不做：在线更新 License、多租户 License、按用量计费。
- 依赖顺序：模块开关（本设计 §2）先行，License（§3）后行；开关机制本身对免费版也有效（用户可自行关闭不用的模块）。

## 5. 验收标准（v1.1）

1. 关闭 k8s 模块后：菜单消失、API 返回 module.not_enabled、reconciler 停转、已有集群数据只读可见。
2. 导入只含 cmdb/deploy 的 License：行为同上；导入全模块 License：恢复。
3. License 过期：商业模块只读，免费模块不受影响。
4. 伪造 License 文件：验签失败拒绝导入。
