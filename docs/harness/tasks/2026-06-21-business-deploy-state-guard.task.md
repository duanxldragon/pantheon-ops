目标仓库：pantheon-ops
层级：business/deploy
任务模式：implement

先读：
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-base/DESIGN.md
- pantheon-base/AGENTS.md
- pantheon-base/docs/README.md
- pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md

实现范围：
- 对齐 deploy 任务状态机的两个业务约束：
  - `POST /tasks/:id/cancel` 只允许 `pending / running`
  - `POST /task-hosts/:id/result` 在 `status=failed` 时必须填写 `errorMessage`
- 补充 deploy 服务层定向测试，锁住取消和失败回写规则

不处理：
- base 共享壳层或系统域编译/工具链问题
- deploy 新增批量能力、回滚能力或 agent 执行面扩展
- 业务数据库结构调整

同步要求：
- 仅本仓业务改动
- 不在 ops 本地 override base 行为

验证方式：
- Backend: `go test ./backend/modules/business/deploy`
- Frontend: `none`
- Smoke: `deploy-api.spec.ts` 已拆为 request-only spec，但当前环境因本地后端编译阻塞未取得运行证据
- UI 证据：本轮无新增视觉实现

停点：
- 不改 base version
- 不改业务数据库结构
- 不回流共享逻辑到 base
