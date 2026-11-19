你是本机 Claude 的低风险测试维护执行者，使用用户配置的默认模型，不启动子代理。你不是唯一执行者，不要撤销或覆盖他人改动。工作目录 D:/workspace/go/pantheon-platform/pantheon-ops。

仅拥有 tests/scripts/branch-hygiene-workflow.test.mjs、tests/scripts/security-audit-exit.test.mjs、tests/scripts/check-task-packet-template.test.mjs 和 .harness/evidence/2026-11-17-ops-history-audit/claude-test-contract-reconcile.md。先读 AGENTS.md；不要修改工作流、checker、其他代码、lock、任务状态或归档，不执行命令、不提交。

1. branch-hygiene 测试断言历史 PR142 删除 push，但现行 workflow 已恢复 push main。按当前真实独立 schedule/manual/push 入口更新断言，保留最小权限、pin、persist-credentials false、专用cleanup script和不依赖 pull_request.closed 的约束。
2. security-audit-exit 测试 step 名过期。读取 .github/workflows/security.yml，恢复真实 shell 用例，当前名称 Report root npm vulnerabilities / Report frontend npm vulnerabilities，严格事件与 advisory 事件策略按工作流核实。不能改工作流去满足旧测试，不能只做正则测试；保留 ci/audit 非零返回与报告保全、scanner fail-closed的实际Bash执行测试。临时目录删除必须验证绝对路径位于该测试的tmp目录。
3. task-template 测试 import 不存在的 export，fixture也是Base模板。改为 subprocess 调用当前Ops CLI的黑盒测试，不改checker；使用真实 docs/README.md/en 与 TASK_PACKET_OPS_TEMPLATE.md/en 的required markers，覆盖完整pass、缺文件、缺继承marker fail。临时fixture限定tmp子目录。不要引入新依赖、不要复制Base规范到Ops。

写简短报告，说明改动与未运行验证。Codex负责运行与审查。
