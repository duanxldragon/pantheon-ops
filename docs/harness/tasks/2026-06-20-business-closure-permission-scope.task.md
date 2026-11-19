目标仓库：pantheon-ops
层级：business/cmdb + business/deploy + business/bizscope
任务模式：implement

先读：

- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-base/DESIGN.md
- pantheon-base/AGENTS.md
- pantheon-base/docs/README.md
- pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md
- pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md
- pantheon-ops/docs/designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md

实现范围：

- 对齐业务模块页面进入权限与接口权限分层，避免 `pagePermission` 继续复用 `list`
- 修复 deploy 任务列表/详情/更新/取消/结果回写的数据范围闭环
- 修复 bizscope 详情、主机绑定相关接口的数据范围闭环
- 同步最小文档与权限映射测试

不处理：

- 新的 deploy 业务能力扩展
- 超出当前锁定 foundation release 的 base 工作树漂移
- 无关的 system/platform 共享逻辑

同步要求：

- 仅本仓业务改动
- 若发现根因属于共享壳层或系统域，回 base 修，不在 ops 本地 override

验证方式：

- Backend: 业务模块定向 `go test`
- Frontend: 仅做受影响配置与权限注册变更，不单独做视觉实现
- Smoke: none
- UI 证据：本轮无新增视觉实现；权限与路由改动以代码和测试为证据

停点：

- 不改 base version
- 不删除 inherited override
- 不改业务数据库结构
- 不回流共享逻辑到 base

## 2026-11-17 逐项复核（整包仍开放）

- BizScope 入口“不复用list”子要求由后继2026-07-18 Active设计明确替代：BUSINESS_BIZSCOPE_MODULE_DESIGN.md §5 API表与§6规定列表list/详情view；保留该授权边界，不机械改为view。独立裁决见 .harness/evidence/2026-11-17-ops-history-audit/june20-bizscope-intent-review.md。
- deploy/cmdb现有入口view和接口action分层继续核对；当前定向Windows CGO race已通过deploy8项、bizscope4项。新增合法可写请求的越权无副作用及范围内成功对照，最终复验与窄范围独立审查待完成。
- 原正文与历史日期保留。本次不标记整包completed，不替代当前运行态、完整race与最终角色验收。
