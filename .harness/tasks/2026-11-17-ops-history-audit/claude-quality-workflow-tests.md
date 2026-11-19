你是本机 Claude 的低风险测试维护执行者，使用用户现有默认模型，不启动子代理。你不是唯一执行者，不要撤销其他修改。

仅拥有 tests/scripts/quality-workflow.test.mjs 和 .harness/evidence/2026-11-17-ops-history-audit/claude-quality-workflow-tests.md。先读 AGENTS.md 与当前 .github/workflows/quality.yml、tests/scripts/security-audit-exit.test.mjs。不要改工作流、生产源码、其他测试、overlay、状态或归档，不运行命令。Codex统一验证。

该测试大量断言已废弃的 docs_only/governance_only scope 和旧 step 名称；现行策略更严格，不能为旧测试降低实际门禁。更新为Ops现行workflow契约：backend-tests 对所有事件必跑；frontend-contract/smoke-sanity 在PR根据真实scope筛选，在push/manual/merge-group仍必跑；go-lint 对PR/merge-group使用 new-from-rev 并fail-closed，full baseline push/manual report-only；duplication对PR/merge-group report-only但其他严格；Quality Gates汇总必需结果并拒绝scope任务失败。PR body edited触发已恢复，docs治理/template/真实body检查存在且UI gate无continue-on-error。

除结构断言外，为汇总shell和lint enforce shell写有价值的实际Bash矩阵测试，使用当前shell、env mock结果，不能复制implementation为expected。包括scope任务failure时不绿色（如果发现真实工作流没有判断该状态，在报告指出，不要弱化测试）、backend fail永远拒绝、PR scope允许合法skip而其他事件不允许skip、PR新代码lint失败必须拒绝、manual全仓lintbaseline失败保持当前report-only。涉及子进程，必须 windowsHide:true、shell:false（直接git-bash绝对路径）；不得打开cmd窗口。fixture目录在tmp明确验证后清理。无新依赖。

只写事实报告和改动，不自称验证通过。SHA shell测试由Codex负责 tests/scripts/ci-workflow-dispatch.test.mjs，你不要修改。
