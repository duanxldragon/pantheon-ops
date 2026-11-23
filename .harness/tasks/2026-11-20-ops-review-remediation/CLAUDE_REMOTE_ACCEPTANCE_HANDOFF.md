# Claude续跑：115.190.251.143 全业务真实验收

维护者现明确要求：在其服务器 115.190.251.143 验证 pantheon-ops 所有业务功能。继续遵守先前“Claude负责执行内容、Codex只验收结果”的角色划分。本提示与最新用户指令优先于上轮“该IP未授权”的限定，该IP现在是授权验收目标；这不授权生产破坏、改全局配置、公开敏感端口或撤销他人改动。

继续任务 2026-11-20-ops-review-live-runtime-acceptance，关联旧十包 C01-C06/E01-E04、V1-V8 和交付收口任务。上一轮 session 18caaf71-f384-405f-aabe-494cd82b9d74 达到30轮调用上限，未产出矩阵/脚本，不是产品验证失败。本轮必须尽早交付文件，禁止重新读取原始Claude结果流，也不要重读已进入上下文的全套AGENTS/设计/父计划。最多六次聚焦读取后开始Write；按小批次交付可审核脚本，不以再次遍历全部仓库结束。

## 已核实状态

- SSH端口22响应；默认xiaolong及历史记录中的root均用BatchMode=yes/StrictHostKeyChecking=yes被拒绝(publickey,password)。Codex已向维护者询问SSH用户名、端口、可用私钥路径/SSH别名以及现有部署访问地址与测试账号配置位置，等待补充；不得猜密码或读取用户全局认证文件。
- 历史Web地址 http://115.190.251.143:3000/ 当前连接失败；历史API地址8080仍在探测，尚无成功结果。
- 现有本地8080监听进程是Go cache下server.exe，不能据此认定当前候选验收；不要绑定旧二进制成功。
- Docker/Kubernetes此前读取本地用户配置受沙箱限制，批准请求被用户中断；现在主目标是服务器，不要把本地Docker作为所有服务器验收的前提。

## 本輪交付（现在开始写）

1. 在 .harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/acceptance-matrix.json 建立逐项矩阵。必须含 V1-V8、旧十包原 acceptanceCriteria/historyCloseout残余/provider行，以及当前 business-overlay.json 和运行时 capability/模块注册表中的全部业务功能。每项有 caseId、module、source、expected、status、evidence、provider、role、cleanup。没有真实执行日志的行只能NOT_RUN/BLOCKED，原风险预测先核实而非当pass。退役/禁用能力也必须核对API、菜单与实际登记一致，不能静默漏掉。
2. 先交付 .local/acceptance-20261120/remote-inventory.py（远程只读）与remote-preflight.mjs（本地审核执行入口）和 command-plan.json。preflight接收SSH target/port/identity path参数或环境变量，不硬编码秘密。使用Node原生spawn/execFile、arguments数组、windowsHide:true、有界timeout、BatchMode=yes/StrictHostKeyChecking=yes；不得sshpass、跳过认证、StrictHostKeyChecking=no、拼接来自用户的shell文本。可将固定已审查Python程序通过ssh stdin以python3 -执行，远端只读收集OS/CPU/RAM/disk、可执行文件可用性、监听端口、运行容器的名称/镜像/状态、服务名称与源码commit/二进制hash；不得输出环境值、完整启动命令、容器inspect全量、密码、密钥或kubeconfig。优先精确目录，不遍历用户数据。收集不足保留明确gap。
3. 写后续HTTP/API真实验证脚本，覆盖业务域、CMDB、凭据、部署包/模板/任务/实例、服务目录、K8s集群/workload/Service/Ingress/storage/logs、文件分发、ArgoCD、observability及其他在册能力。核对正常HTTP入口、真实provider、最小角色正反例、双业务域/租户、分页101+、终态日志/审计和UI主流程。复用当前smoke与旧入口，编写远端隔离fixture方案；仅测试本任务prefix资源，不任意重启/升级既有产品或触碰现存业务资源。
4. SSH未就绪时仍可准备HTTP、provider、候选/构建、UI/安装升级失败恢复回滚脚本。环境计划明确独立目录/数据库/Redis prefix或实例/namespace/低权限SSH用户/fixture与清理范围；不要自行安装软件或执行远端代码，先交具体command-plan由Codex审核运行。
5. 写 summary.md、executor-progress.json（下一条atomic action）、环境缺项与续跑checkpoint。每个脚本交付后记录scriptPath、caseId、cwd、executable、arguments数组、timeout、环境变量名、预期、副作用和清理范围。代码内容由你完成；Codex负责审查、运行具体验收命令和读渲染截图，再将失败结果回给你修复。

## 严格保留的合同

业务实现留Ops；共享platform/system域问题Base-first，不在Ops override，也不发布或手抄本地Base快照。保留当前他人改动，不提交/push/发布。Windows后台隐藏启动，使用已有hide-child-windows.mjs和process-environment.ps1。UI用impeccable + Base DESIGN.md，真实桌面1440/窄屏390及关键交互状态。V7注入必须在执行前拒绝、无标记；远端测试用专用低权限用户，root仅有需要的只读环境盘点或明确隔离准备入口，不以root跑危险注入。Helm依据当前登记功能而非新增完整Chart要求。不能代签维护者最终验收。

本轮先交付可运行preflight和完整矩阵，再增量准备业务验证。任何未跑/缺认证/缺provider的项不标通过；保留真实退出码、时间、候选身份和清理证据。
