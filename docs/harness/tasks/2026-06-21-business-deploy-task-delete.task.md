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
- 补齐部署任务删除能力的后端接口、权限映射、seed 与前端交互
- 删除规则限定为未启动任务，具体状态为 `draft` / `pending`
- 删除时同步清理 `biz_deploy_task_host` 明细，避免业务残留

不处理：
- base 共享壳层或系统域继承漂移
- 运行中、已完成任务的历史归档/回收站能力
- 新的 deploy 批量删除能力

同步要求：
- 仅本仓业务改动
- 不在 ops 本地 override base 行为

验证方式：
- Backend: `go test ./backend/modules/business/deploy ./backend/pkg/contracts`
- Frontend: `npm run check:i18n-generated`、`npm run check:i18n-missing-keys`、`npm run check:menu-contract`
- Smoke: `playwright test tests/smoke/business/deploy/deploy-api.spec.ts --config=playwright.api.config.ts`
- UI 证据：当前环境缺少 Playwright 浏览器二进制时，明确记录无渲染证据

停点：
- 不改 base version
- 不改业务数据库结构
- 不回流共享逻辑到 base
