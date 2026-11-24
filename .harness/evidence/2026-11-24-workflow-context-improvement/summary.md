# 工作机制改进验证记录

已保留三仓库所有权、Base-first、L0/L1/L2 和真实验收要求，并完成以下改进：

- AI 先读入口、当前任务和对应合同，按需读取设计与证据。
- 当前状态与历史检查点分离，manifest 生成任务索引，检查模式拒绝过期索引且不修改状态。
- CI 从 receipt 的完整方法 commit 获取固定快照，正常验证不更新 pin。
- 共享检查器排除注释、echo 和非入口参数；区分手动工具与 CI 配置引用。简单 folded YAML 正确折叠；复杂形式显式失败。
- Ops 工作流与命令只改仓库拥有的资产，真实锁定发行版重建后验证保留与共享漂移。

独立 code-reviewer 为 COMMENT（零剩余源码 finding；LSP/AST 不可用），architect 为 CLEAR。
Harness 行为测试 16/16；Base 最终行为/工作流测试 19/19；Ops 工作流/重建测试 29/29。
两仓库消费检查、索引检查、文档、编码、边界检查通过；正式 Base 快照暂存重建、overlay 和 shared drift 通过（866 个共享文件）。
44 个范围外文件和 25 个 backend 文件保持原 SHA256。Ops 开工备份与追加第八轮检查点后的快照分别保留。

记录真实失败和修复：缓存写权限/认证 sandbox、暂存资产重复下载网络失败、读取顺序 marker、task checklist 和 folded YAML。未把失败记 PASS。

产品功能开发、正式发行与 foundation lock 升级属于其他任务。用户随后授权本次方法改动的提交、PR、CI 和合并；最终 hosted 结果以对应 PR checks 为准。
没有产品 runtime/UI 改动，没有声称工作效率或 token/cost 的实测提升。

## 隔离提交与方法来源

Harness PR #13 已合并；PR 与 main CI 均通过。consumer 明确采用合并提交 `51fc783dc793b75ae91bdb936ba1418e2e95a354`，七份 policy 哈希不变。新的 helper/checker 修订仍未打正式发行 tag。

Ops 完整 CGO/race 使用指定 MinGW 和本任务独占 Redis 26394；此前 26316 的失效地址、沙箱缓存权限与原生路径执行失败分别保留，不计 PASS。原有全库 Harness historical artifact/schema 问题与 shell 1.4→1.5 同步不在此 PR：既有 CI 未启用这些全库 gate；本任务的 task/evidence/review 仍以严格模式单独校验，无豁免和门禁降级。历史 snapshot 通过路径级 Git -text 保留原字节。

rebase 到 `main` 的 T08 合并提交后重跑：Ops 工作流/重建 29/29、消费/边界/overlay/wiring/gorm/UI gate 全部 0 finding，完整 `go test -race ./...` exit 0（19.3s，本任务独占 Redis 26394/26395，T08 日志背压回归在内）。新证据见 `commands.json` 的 post-rebase 条目。

## 第二次 rebase（T09 收口合并后）

`main` 随后以 `697c0771`（PR #208，T09 全量套件收口与 smoke 证据）前进，本分支再次重放到该提交（新 head `74168cf9`）。该提交只改 `.harness/evidence/*` 与两个任务 `STATUS.md`，无语义源码差异，故既有 29/29 工作流/重建测试与完整 `go test -race ./...` 结论对代码树仍然成立；未伪造新的执行记录。`TASK_INDEX` 重渲染与提交内容一致，无 diff。

## Hosted 收口（2026-11-24）

- Harness PR #13 合并为 `51fc783dc793b75ae91bdb936ba1418e2e95a354`；main「Harness Method Health」success。
- Ops PR #209 合并为 `60bfdf98ce2a47a89873c9592a574417f8bdcfda`；必过 Docs Governance / Backend Tests SUCCESS（Frontend Contract 按条件 skipped）；非必需 Automate Solo PR（缺 RELEASE_GATE_TOKEN）与 Vercel preview 是既有外部红灯，未豁免、未降级、未用 solo-override。
- Base PR #526 合并为 `768c37088f705357779b0a0cfedab789aeaa5394`（Base-first：Base 先合）。
- 本任务 manifest 置 completed、task doc 置 Archived、`TASK_INDEX` 按 manifest 重渲染。正式发行、消费重建与产品验收仍是独立后续。
