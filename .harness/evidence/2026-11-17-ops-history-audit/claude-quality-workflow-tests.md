# Claude → quality-workflow.test.mjs 契约对账 — 2026-11-17

任务范围：Claude 执行者仅更新 `tests/scripts/quality-workflow.test.mjs`，使其与当前真实
`.github/workflows/quality.yml` 契约对齐；**不改动工作流**（`quality.yml` 为只读对齐源），
不触碰 `tests/scripts/ci-workflow-dispatch.test.mjs`（Codex 拥有）。统一运行/审查交由 Codex。

## 对齐基线（写本文档时的真实工作流）

- 工作流 `quality.yml` worktree blob SHA：`b70f7a4e79d6f1f114cd2e00c859177a9dbde65f`
- 测试 `quality-workflow.test.mjs` worktree blob SHA：`13652385ebb31a44a97a352b99b8ef3c54a6d953`
- 基线 commit：`71c86cfe664ef9fde310cc68efbf2e8992e648f2`（`chore/tasks: ops task-progress closeout`）

## 旧契约为何失败（本测试先前断言的是一个已被移除的契约）

原测试断言 `change-scope` 存在 `docs_only` / `governance_only` / `all_count` 输出与
`all_count == docs_only_count` 分类、`predicate-quantifier: every`（配合 doesNotMatch）、
`quality-gates` 使用 `GATE_OK` / `GOVERNANCE_ONLY` / `RUNTIME_GATES_REQUIRED=false`、
`frontend-contract`/`backend-tests`/`smoke-sanity`/`go-lint` 的
`if: event == merge_group || governance_only != 'true'` 跳过条件、`Enforce UI quality gate` /
`Full-repo lint is report-only on push/workflow_dispatch` 等旧 step/文案。

现行 `quality.yml` 已重构成经 scope 输出的真实门禁契约（见下），上述断言全部失配，运行即红：

- `change-scope` 仅输出 `candidate_sha`/`frontend_contract`/`smoke_sanity`/`overlay_sensitive`，
  `predicate-quantifier: some`，无 docs/governance 缓冲区。
- 运行时 job 按 PR 可检测真实 scope 筛选：`frontend-contract`/`smoke-sanity` 用
  `event_name != 'pull_request' || needs.change-scope.outputs.<x> == 'true'`；`backend-tests` 无 `if:` 恒跑。
- go-lint 非 push 一律 fail-closed；push/manual（workflow_dispatch）全仓库 lint 保持 report-only。
- duplication 在 PR/merge_group 为 report-only，其余事件 fail-closed。
- quality-gates 用 `EXPECTED_RESULT: success` 全量聚合，docs/backend/duplication 恒 needed。

## 重写后测试内容（10 条，全部真实断言 + 真实 Bash 执行）

1. **change scope 分类真实门禁而非已移除的 docs/governance 缓冲** — paths-filter SHA pin、
   `predicate-quantifier: some`、`Detect changed scope` 仅限 PR、三个 real 输出及其 glob 白名单；
   并 `doesNotMatch` 认定 `docs_only`/`governance_only`/`all_count` 已移除。
2. **每个 checkout 都钉到已解析的 candidate commit** — workflow_dispatch `sha` 输入、
   `Resolve and verify candidate commit` 6–40 hex 校验与 `HEAD != requested → fail`，
   `candidate_sha` 输出，消费 job 的 `ref: candidate_sha`。真实 Bash：非 hex → status 1；
   合法 HEAD → status 0 且写 `sha=`。
3. **frontend-contract / smoke-sanity 仅按 PR scope 门控** — `event_name != pull_request || scope == true`。
4. **backend-tests 恒跑并 fail-closed** — needs change-scope、无 `if:`、`go test -race ./...`。
5. **go-lint 仅 lint 新增代码、真实 Bash 执行回报/拒绝分支** — PR/merge_group 用
   `--new-from-rev`；`Enforce Go lint gate`：PR/merge_group failure→1，push/manual failure→0，success→0。
6. **duplication 在 PR/merge_group report-only、真实 Bash 执行** — 两事件 failure→0，
   push/manual failure→1，success→0。
7. **quality-gates 全量聚合并在真实 Bash 中强制** — dependencies 含全部 7 job、`if: always()`、
   `EXPECTED_RESULT`、恒强制执行 docs/backend/duplication；frontend/smoke 按 scope 需要才拒；
   go-lint 非 push 拒、push 放行。覆盖 all-success→0 与各单点 failure→1/0。
8. **docs governance 以严格非 advisory step 封堵 UI 质量漂移** — `Check UI quality gate` +
   `npm run check:ui-quality-gate`、无 continue-on-error、含 `Run tests/scripts catch-all`。
9. **docs governance 校验 PR governance 模板与 body，并在 body 编辑时重跑** — PR types
   flow 数组含 `edited`、`Check PR governance template`、`Validate PR governance body`
   （排除 dependabot、`--event "$GITHUB_EVENT_PATH"`）。
10. **不做无用的 pantheon-base foundation 二次 checkout**（doesNotMatch）。

运行真实 Bash 复用了 `security-audit-exit.test.mjs` 的既有习惯：从工作流按 step 提取 `run: |`
shell 体，经 git-bash 子进程（`spawnSync(..., {cwd, env, windowsHide:true})`，非 `shell:false`/cmd）
执行并断言退出码；bash 路径在本机解析为 `<git>/bin/bash.exe`（fallback `usr/bin`），跨平台回退 `bash`。

## 本执行者已实际验证（不虚构）

在仓库根运行：

```
node --test tests/scripts/quality-workflow.test.mjs
```

结果：**10 pass / 0 fail**，退出码 0（`tests 10, pass 10, fail 0`）。真实 Bash 探针
（go-lint / duplication / quality-gates / candidate 校验）全部在本机 git-bash 上执行并通过。

## 留给 Codex 统一验证的复核点

- `npm run test:scripts` 全体（含本文件与其它执行者/Codex 拥有的测试文件）一键通过。
- 本文件提取 shell 的缩进约定（`        run: |` ↔ 10 空格 body）若工作流后续改缩进/命名需同步
  此测试（`shellFor` 依赖 step name 精确匹配）。
- `docs_only`/`governance_only` 已移除的 doesNotMatch 是本契约基线；若未来恢复该分类需同步测试。
