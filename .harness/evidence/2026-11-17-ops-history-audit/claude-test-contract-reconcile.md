# Claude 测试契约 reconcile — 2026-11-17

任务范围：仅更新我拥有的三个脚本测试，使其与当前真实工作流/checker 对齐。**未运行任何验证**（本角色不执行命令）；由 Codex 负责运行与审查。

## 1. `tests/scripts/branch-hygiene-workflow.test.mjs`

历史断言「#142 删除了 push 触发」，但现行 `branch-hygiene.yml` 已恢复 `push: branches: [main]`。按当前真实独立入口（`push main` + `schedule` + `workflow_dispatch`）更新断言：

- 删除 `doesNotMatch push` 断言与 #142 注释，改为断言 `push` 到 `main`。
- 保留：最小权限（`contents: write` / `pull-requests: read`）、checkout/setup-node 全 sha pin、`persist-credentials: false`、专用 `node scripts/cleanup-github-branches.mjs`、不依赖 `pull_request.closed`。

## 2. `tests/scripts/security-audit-exit.test.mjs`

- step 名过期 → 恢复真实 shell 用例：`Report root npm vulnerabilities` / `Report frontend npm vulnerabilities`（`name.includes('frontend')` 判断，报告仍写 `npm-audit-{root,frontend}.json`）。
- 严格事件与 advisory 策略按真实工作流核实：`dependency-vulnerabilities` job 只在 `schedule / workflow_dispatch / push` 运行；advisory 事件（`pull_request` / `merge_group`）通过**整 job skip** 变得不阻塞，而非放宽 gate shell。原测试在 gate 内对 advisory 事件特判失败容忍，已删除。
  - 新增一个黑盒约束测试：读取真实 `security.yml` 的 job `if:`，断言仅含 schedule/workflow_dispatch/push 且不含 pull_request/merge_group。
  - 新增/保留真实 Bash 执行：`Enforce dependency vulnerability gate`（原名 `Enforce gate`，已纠正；env 增加 `OSV_CALL_ANALYSIS_RESULT`）任一 scanner 非 success → fail closed（status 1），all success → status 0。
- 保留 ci/audit 非零返回与报告保全的真实 Bash 测试（保留 `-e` banner，`npm ci` 失败→ abort→无报告，检查 `if(!ciCode)`）。
- 临时目录删除保留绝对路径位于本测试 tmp 目录的校验。

## 3. `tests/scripts/check-task-packet-template.test.mjs`

- 原测试 `import { validateTaskPacketTemplate }`（该 export 不存在，且 import 即运行 checker aborts），fixture 用 `TASK_PACKET_BASE_TEMPLATE.md`（Base 模板）。已重写为**黑盒 subprocess 测试**（`spawnSync(node, [checkerPath], {cwd})`），不改 checker。
- 使用真实 Ops 模板 markers：`docs/README.md/en`（`TASK_PACKET_OPS_TEMPLATE.md`、`` `base -> ops` ``/`` `base -> ops` sync ``）与 `docs/TASK_PACKET_OPS_TEMPLATE.md/en`（`目标仓库：pantheon-ops` / `Target repo: pantheon-ops`、同步/`写清 base commit` / `record the base commit`）。
- 覆盖：完整 pass（status 0 + stdout `OK`）、缺文件 fail、缺继承 marker fail。
- 未引入新依赖、未复制 Base 规范到 Ops；临时 fixture 用 `mkdtempSync(os.tmpdir(), …)` 并断言绝对路径直接位于 OS tmp 目录下再删除。

## 未运行验证

- 本角色不执行命令、不提交 → 所有改动未运行、未过 CI。潜在需 Codex 复核点：
  - security 测试 `-e` banner 语义与 step 无 `set -e` 的关系（ci 失败 abort → status=ciCode、无报告，维持 `if(!ciCode)` 断言）。
  - security compose job `if` 的提取正则（`^  [a-z][a-z-]*:$`）是否随工作流缩进/命名变化再接住。
  - task-template markers 常量与 checker `requiredMarkers` 的一致性（改动 checker 时需同步此测试）。
