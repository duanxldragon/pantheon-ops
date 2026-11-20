# C05 CI 准备交接（2026-11-18）

本地 CI 接线与回归已通过；C05 整包运行态验收尚未完成。任务状态未修改。

## 改动与证据

- `.github/workflows/quality.yml` 在同一 `smoke-sanity` 候选 SHA、MySQL/Redis、健康检查与平台/系统 smoke 后，必过执行 CMDB、Deploy API、Deploy UI、Service、K8s 五个已有入口。`set -euo pipefail` 保留失败；沿用失败产物及 Quality Gates。
- `tests/scripts/quality-workflow.test.mjs` 检查步骤所属 job/顺序/脚本存在/禁止 bypass，并执行实际 workflow bash：成功时五个入口全序执行；分别让每个入口返回 23，证明原始退出码传播且后续不执行。假 npm 仅隔离 provider，不修改产品 fixture。
- 先 RED：`node --test tests/scripts/quality-workflow.test.mjs`，exit 1，12 项中 10 pass / 2 fail / 0 skip；两个新测试精确捕获缺失步骤。
- 再 GREEN：相同命令，exit 0，12 pass / 0 fail / 0 skip，8694.6568ms。
- `node --check tests/scripts/quality-workflow.test.mjs` 与 scoped `git diff --check` 均 exit 0。

Observed HEAD: `71c86cfe664ef9fde310cc68efbf2e8992e648f2`。共享工作区包含并发未提交改动，HEAD 不是本次完整候选认证；本次证据绑定具体 source hashes（见 JSON），不替代最终集成验证。

## Findings-first self-review

本次 scoped diff 未发现阻断 CI 接线的缺陷。新增门禁复用 Ops 已声明 overlay 所有权；没有改变现有 scope、候选 checkout、系统 suite、汇总 fail-closed 语义或 quality thresholds，没有引入新依赖。测试子进程沿用 `windowsHide: true`；临时日志目录 finally 清理。

此处是实施者自检，协调者独立 review 尚待完成；不将自检记录成独立评审通过。无产品源码/权限/seed/迁移/UI 修改，因此本子任务不运行 Go、frontend build、browser 或服务，完整候选仍由协调者验证。

Gate Outcomes: quality-workflow regression caught missing required core business smoke | none

## 显式 gap

- Hosted CI 尚未运行这次完整候选；45 分钟 job 时限是否容纳新增套件需 hosted 结果证明。
- 真实 MySQL/SSH/K8s、双部门/业务范围最小角色、并发/重复/故障/升级/回滚、当前 revision 桌面/窄屏渲染及维护者验收 `not-run`。
- Service/K8s 页面 smoke 含 mocked 数据，不能替代真实 provider 验收；五核心入口不覆盖 Observability/生成模块 umbrella。
- 既有 governance gate 红灯和历史 menuM3C runtime gap 未豁免；本次实现只关闭“业务 smoke CI 接线”子要求。

## Linkage

- Task: `.harness/tasks/2026-11-16-ops-business-runtime-acceptance/manifest.json`
- Plan: `.harness/tasks/2026-11-16-ops-business-runtime-acceptance/continuation-local-acceptance-plan-20261118.md`
- Commands and source hashes: `.harness/evidence/2026-11-16-ops-business-runtime-acceptance/ci-preparation-handoff-20261118.json`
