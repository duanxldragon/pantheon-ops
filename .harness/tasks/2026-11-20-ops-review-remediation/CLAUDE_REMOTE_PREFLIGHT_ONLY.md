# 独立文件任务：真实服务器只读盘点入口

维护者指定115.190.251.143验收pantheon-ops所有业务，且要求Claude完成执行内容、Codex只验收结果。本任务你是Generator；Codex是Reviewer。你不是唯一执行者，另一Claude正在编写全业务矩阵，保留其他人的编辑。

你的独占文件范围只有：
- .local/acceptance-20261120/remote-inventory.py
- .local/acceptance-20261120/remote-preflight.mjs
- .harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/command-plan-preflight.json
- .harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/preflight-generator.md

不要改matrix、summary、manifest、业务源码或任何其他文件。直接交付这四份文件，禁止读取其他Claude原始结果流。可读AGENTS.md及必读继承文档、现有.local/manual-preview-20261116/hide-child-windows.mjs作为约束，聚焦任务，不再次遍历10包或业务目录。服务访问/凭据未就绪不影响写脚本。

服务器SSH22可达，host key和当前known_hosts匹配，Ubuntu OpenSSH_9.6p1。root默认id_rsa公钥被拒绝，默认xiaolong也被拒绝；维护者的SSH连接信息已询问等待回复。不要猜密码，不读取私钥内容或用户全局认证。不降低StrictHostKeyChecking、不使用sshpass。历史HTTP端口80/443/3000拒绝连接、8080超时。

remote-inventory.py通过固定python3 -在SSH stdin执行，只读收集OS/CPU/RAM/磁盘、python/docker/podman/kubectl/k3s/go/node/nginx等可执行文件可用性、监听地址/端口、已运行容器的名称/镜像/状态和明确服务名称。精确检查常见Ops目录的存在，以及有确定产品来源目录时读取git HEAD、lock版本或二进制hash。禁止全磁盘扫描；禁止输出env值、容器全量inspect、ps完整参数、密码、密钥、kubeconfig、用户数据。固定argv子进程timeout，关闭stdin并有输出字节上限；所有失败以安全错误类型/退出码记录。最终stdout为单个JSON，不原样透传秘密或异常命令全文。

remote-preflight.mjs必须为本地Node入口，参数/环境变量只接受合法SSH target、port、identityPath、证据目录。远端IP固定授权115.190.251.143，用户名严格白名单字符，端口1-65535，不接受任意host/shell text。使用spawn('ssh', arguments数组, {windowsHide:true,...})，BatchMode=yes、StrictHostKeyChecking=yes、ConnectTimeout=10、ConnectionAttempts=1，可选指定已提供的identity path但不要读私钥内容。远端命令仅固定'python3 -'，将审核的remote-inventory.py字节写stdin；timeout/输入/输出/drain有界，记录真实exitCode/信号/耗时。证据目录解析后只能位于本任务 evidence 子树，写文件前验证目标且不覆盖已有记录。退出非0不能声称盘点成功，原始stderr不能未经脱敏写证据或打印；只保留允许的认证/连接错误类别。输出候选/provider gaps，不标业务功能pass。按现有隐藏preload兼容Node后代，不改全局设置。

command-plan-preflight.json写taskId、caseId、cwd、executable='node'、arguments数组、scriptPath、timeoutSeconds、所需环境变量名（不填值）、expected、sideEffects（只读远端/本地证据文件）、cleanupScope。preflight-generator.md写代码文件、适用边界和审查要点，明确脚本尚未执行。Codex会做语法与安全审查后运行。

这是小片段产出任务，完成四个文件即停止，不扩大范围，也不以“等待认证”结束而没有脚本。
