# Harness 脚本

English version: [README.md](./README.md)

这里存放 `pantheon-base` 的仓库本地 Harness 检查。

可复用方法事实源位于同级仓库 `../pantheon-harness/`。本目录只负责 `pantheon-base` 的落地执行：一部分脚本镜像自 portable checker，一部分脚本是 base 专属门禁。

默认上游解析顺序：

1. `--method-root <path>`
2. `PANTHEON_HARNESS_ROOT`
3. `config/method.config.json` -> `pantheonHarnessRoot`
4. `../pantheon-harness`

## 快速验证

```powershell
node scripts/harness/check-method-health.mjs --root . --strict
node scripts/harness/check-adoption.mjs --root . --strict
node scripts/harness/check-template-health.mjs --root . --strict
node scripts/harness/check-doc-frontmatter.mjs --root . --strict
node scripts/harness/check-doc-links.mjs --root . --strict
node scripts/harness/check-doc-inventory.mjs --root . --strict
node scripts/harness/check-sync-drift.mjs --root . --strict
node scripts/harness/check-encoding.mjs --root . --strict
node scripts/harness/check-ui-quality-gate.mjs --root . --strict
node scripts/harness/check-generated.mjs --root . --strict
```

## 检查器作用域（2026-11-04，FR-027）

归档是任务包的正常归宿，因此只扫活树的检查器会在不报红的情况下失去治理对象。2026-10-29 / 10-31 两轮归档后，三个 strict 证据检查器就是这样：只扫到 15 个（甚至 0 个）文件，而 196 份产物已在 `.harness/archive/<YYYY-MM>/` 下，却仍然 exit 0。

- `check-evidence.mjs`、`check-review.mjs`、`check-runtime-evidence.mjs` 现在同时发现 `.harness/evidence/**` 与 `.harness/archive/**` 下的证据；文本报告里的文件计数是作用域唯一可见证据——归档在涨而计数不涨，就是检查器又瞎了，与退出码无关。
- 用「按年代分代」而不是删除来处理历史。`check-evidence.mjs` 从 `CURRENT_SCHEMA_FROM = 2026-10-26` 起才强制；此前的结构性 finding 只统计不强制（当前 43 份归档产物共 205 条，会打印在报告里），这个数字必须往下走。这些历史事实可以补写但不可以编造：为没人观测过的轮次补 `commands[].status`、`cwd`、`agent` 只会制造无法证伪的历史。`File does not exist.` 与 `Invalid JSON` 任何年代都强制。
- 无法补写又不想伪造的产物进具名、具日期的豁免表：`config/review-loop-legacy-artifacts.json`（8 条）与 `config/runtime-evidence-legacy-artifacts.json`（2 条，`--legacy-config`）。豁免条目指向的文件一旦消失就报错，所以豁免不会比产物活得更久。
- **新证据必须写在 `.harness/evidence/<task-id>/` 并随任务包一起归档。** 写在别处的轮次对 `check-evidence` 的活树发现不可见；包与证据不同步则破坏 `check-task-packet` 的状态-证据一致性——两种情况的结局都是这一轮没人检查而门禁照常报绿。
- 登记表本身也受作用域影响：`check-failure-registry.mjs` 过去遇到空行就停止扫表，所以空行之后的行（`FR-026`）渲染成一张没有表头的孤立表，从未被校验。现在跨空行继续、遇到正文才停，`tests/scripts/harness-check-failure-registry.test.mjs` 把两半都钉住。

## 共享工具

- `sort-utils.mjs` - 字符串排序工具。
- `upstream-root.mjs` - 解析当前 workspace 中的 `pantheon-harness` 方法源。

## Pantheon Base 专属检查

- `check-audit-coverage.mjs`
- `check-backend-dto-contract.mjs`
- `check-backend-response-contract.mjs`
- `check-coverage.mjs`（单测覆盖率门禁，支持 Go cover profile 与 vitest coverage-summary `--format json`）
- `check-generated.mjs`（生成产物标记与漂移门禁，见 REPOSITORY_LAYOUT §7）
- `check-inheritance-contract.mjs`（必须在 workspace 根执行：`node scripts/harness/check-inheritance-contract.mjs --root .. --strict`；`--root .` 会把 workspace 与 ops 的文档报成"文件缺失"，那是误读不是 finding）
- `check-structure-contract.mjs`（目录放置 + 命名门禁，契约源 REPOSITORY_LAYOUT.md；与 check-boundaries 的 import 边界互补）
- `check-ui-quality-gate.mjs`（校验 UI 政策、CI 接线与新 UI 任务的证据声明）
- `check-permission-contract.mjs`
- `triage-base-drift.mjs`

## 调用角色与当前工作机制

- `check:checker-wiring` 识别命令中的直接 Node 入口和已声明 npm 别名，排除注释与 echo 数据；动态 wrapper 须另给证据。仅 package 配置的工具属于手动入口，CI 引用不等于运行成功或合并必过检查。
- 方法 policy CI：`quality.yml` 将消费仓库与 Harness 作为同级目录 checkout，由 receipt 的完整 `sourceCommit` 固定方法快照，调用已发布的消费检查器，不带更新参数。
- Ops 的跨层边界 CI 使用继承 checker 与现有 baseline；业务模块边界检查单独保留。
- Ops `check:coverage` 是手动继承工具，使用 `npm run check:coverage -- <artifact> --threshold <number>` 提供实际覆盖率文件与另行审查的阈值；不隐含默认阈值或覆盖率 CI 已验收。
- 当前任务索引使用尚未发布的 Harness `render-task-index.mjs`；显式生成不改变 manifest 状态。新 helper/共享 sensor 的发布与下游同步仍须后续完成。
