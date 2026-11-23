# Claude执行派发 — V1–V8与旧十包原范围验收

维护者最新指令为“先完成V1–V8和旧十包未验收事项，派发给Claude完成，Codex只验收结果”。你是本次Generator执行者，Codex是Reviewer/Evaluator与验证命令的批准执行者。

目标仓库为D:/workspace/go/pantheon-platform/pantheon-ops。先读AGENTS.md、../AGENTS.md、docs/PROJECT_INHERITANCE.md、../pantheon-base/DESIGN.md、../pantheon-base/AGENTS.md、../pantheon-base/docs/README.md。
本轮L2，任务ID为2026-11-20-ops-review-live-runtime-acceptance，交付包收口任务为2026-11-20-ops-review-delivery-and-closeout。读父计划 .harness/tasks/2026-11-20-ops-review-remediation/plan.md 与 FINDINGS.json，以及 docs/reviews/pantheon-ops真实环境验证清单-2026-11-20.md。
使用C:/Users/xiaolong/.codex/skills/impeccable/SKILL.md作为UI前后质量门禁。

## 范围与推进方式

先验现有候选的V1-V8及C01-C06/E01-E04原标准。此指令覆盖父计划“先等所有T01-T15整改完成再运行T16”的排队方式。不要自行引入整批整改、全面租户迁移或新产品能力作为验收前提。
逐项读FINDINGS.json.priorTaskResiduals内10个归档manifest，原acceptanceCriteria、对应taskDoc、summary/review/historyCloseout.residuals和原provider矩阵是验收输入。历史证据需匹配当前候选、实际角色/provider与完整场景。
V1/V3全部403、V4可能跨租户可见和V8旧连接延迟失效是报告的核实预测。先记录真实结果和对结论影响；发现产品或安全缺陷时提出局部修复并复跑，不能修改预期隐藏失败。
旧10包status=completed是归档状态，未验收事项须逐项实证。Helm按当前登记产品能力及原实际开放范围验收。

你负责范围内业务代码、fixture、测试/验收脚本、证据生成和失败修复。你不是代码库唯一执行者，不得撤销他人改动。共享平台/系统域变更遵循Base-first，先报告具体依赖及最小上游修复，不在Ops override。Codex负责验收、执行已审查的具体命令和读截图。

## 本次CLI工具与执行合同

本次为Read/Edit/Write/Grep/Glob文件工具，空MCP且hooks关闭，模型/认证来自既有默认配置。
先产出可审核的具体执行脚本和command-plan.json，Codex审查后运行。你应继续完成脚本与局部修复，并读取后续返回的真实日志继续工作。
command-plan每项必须包含caseId、cwd、executable、arguments数组、scriptPath、所需环境变量名、预期结果、超时、持久化副作用和清理范围，不得用shell拼接命令。不在这个阶段把未执行标成pass。
后台进程必须隐藏，Node spawn/fork设置windowsHide并继承已有隐藏preload；使用既有 .local/manual-preview-20261116/hide-child-windows.mjs、process-environment.ps1 等辅助。需服务启动/重启时写可审核脚本，只操作本任务拥有的隔离候选进程。
不读取用户全局认证配置，不写全局设置，不使用生产凭据或报告中的远程IP作为自动授权目标。可以复用已有本地隔离DB和provider；缺环境时准备本地kind/容器方案与具体命令供Codex执行，不能以缺环境结束全部工作。
清理只针对本次fixture/任务专用目录，不做全盘或其他任务递归删除。每个解析后的绝对路径必须先检查在目标目录内。

## 现有可复用入口

.local/manual-preview-20261116/ 有preview/Go/进程环境/隐藏启动工具，.local/acceptance-20261119/ 有providers-status.ps1、c03-live/e03-live脚本。
本机8080/5173/3306/6379有监听，此信息仅readiness。严禁将旧binary或mock截图作为新候选/provider完整验收。
使用Glob限制这些目录的顶层文件，不递归扫描mysql-data或历史overlay树。
已有完整Go race入口需要D:/msys64/mingw64/bin与CGO_ENABLED=1，隔离PANTHEON_TEST_DSN必须required，凭据不得进入stdout或证据。
frontend/scripts与npm scripts按当前package.json核对，不能沿用不存在的命令。

## 第一轮交付

1. 建立 .harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/acceptance-matrix.json，V1-V8和10包每条原验收各有caseId、source、expected、currentEvidence、status=NOT_RUN/待核实、owner，完整原范围不得只写一个包级勾选。
2. 当前源码/lock与历史证据的候选核实，输出不含秘密的环境需求及实际可用入口。
3. 为能立即运行的V1-V5和核心C包提供 .local/acceptance-20261120/ 下局部验收脚本、具体command-plan.json。复用已有fixture和测试，确保走正常HTTP入口并记录真实状态码/副作用。
4. 输出V6-V8与扩展、安装升级失败恢复回滚的后续执行计划，继续写能准备的脚本。依赖缺失记录具体错误/缺项及可替代隔离方案。
5. summary.md清楚列出实际已做/待执行，executor-progress.json指向下一条atomic action。没有实际命令日志时不要生成成功receipt。全部代码改动列出文件和原因。

这是持续任务的第一个有时限执行片段。片段结束必须保存可续跑checkpoint，Codex会执行command-plan并把结果给你，再由你完成局部修复与后续验收。保持原失败，不提交/push/发布，不代签维护者最终视觉/功能验收。

