# Generator任务：本地隔离Ops候选运行入口

最新维护者纠正：Pantheon Ops本地起，115.190.251.143仅作SSH部署目标，从CMDB开始业务验收。服务器使用root密码登录，秘密由Codex保管，绝不写入提示/日志。你是Claude Generator，Codex是Reviewer/验收命令执行者；AGENTS的Planner不改业务代码约束不适用于你的Generator角色。你不是唯一执行者，不撤销别人编辑。

独占范围 .local/acceptance-20261121/start-local-runtime.ps1、runtime-env.ps1、command-plan-runtime.json（后者放本任务evidence目录）。其他Claude负责CMDB真实SSH/API验收脚本，你不要改这些脚本/矩阵/summary。直接生成这三份可审核文件，不全树扫，不读取任何原始Claude结果流和私密文件。必读AGENTS/继承资料后聚焦运行态配置当前正确名称（可读后端配置代码、.env.example但不能读.env），复用既有工具，最多6次聚焦代码读取后Write。

已知本地工具：D:/Go/bin/go.exe、D:/nodejs/node.exe、D:/MySQL/mysql/bin/mysqld.exe和mysql.exe、D:/Redis/redis-server.exe。root Ops路径D:/workspace/go/pantheon-platform/pantheon-ops。backend/是Go cwd；frontend/node_modules/vite/bin/vite.js可用。当前MySQL3306/Redis6379/前端5173在跑；8080已无监听。不能用旧Go cache binary作当前候选，也不能停止别人的服务。

请准备独立运行环境：本任务目录 .local/acceptance-20261121/runtime/；新MySQL绑定127.0.0.1:33321、新Redis127.0.0.1:26321、backend8081、frontend5175。启动前四端口均必须空闲，runtime不存在或者校验自身ownership后拒绝重复；不能删除旧目录。数据DB名称ops_acceptance_20261121独立；MySQL初始化--no-defaults --initialize-insecure仅绑定loopback，随后设置随机root密码（SQL通过stdin或受限option file，不放argv/log）。密码、admin初始密码、Redis认证和PANTHEON各secret生成Crypto随机，DPAPI加密写只可当前用户解密的runtime-env.clixml/等价文件，私密文本不进入证据；runtime-env.ps1能安全解密并只设置本任务需要的env，输出仅成功/变量名不输出值。测试DSN required且此DB独立，最终race使用D:/msys64/mingw64/bin和CGO_ENABLED=1。

当前候选二进制由Codex在backend执行go build -buildvcs=false -mod=readonly -o ../.local/acceptance-20261121/pantheon-ops.exe ./cmd/server，你可提供该command-plan但不执行。启动脚本验证二进制存在记录SHA256/go version -m与git HEAD/源码身份/lock，若无法严格完整源码hash记录gap，不能虚构receipt。startup所有后台ProcessStartInfo/CreateNoWindow=true或Start-Process -WindowStyle Hidden；Node兼容已有.local/manual-preview-20261116/hide-child-windows.mjs后代隐藏preload。单个初始化/health等待有界并记录真实退出。后端使用当前真实变量名，development初始表/seed/正式业务迁移，后端仅loopback（如配置支持），前端host127.0.0.1，PANTHEON_API_PROXY_TARGET按当前vite配置核对8081。别复用现有生产/dev库、token、或切换旧库multi模式。

服务启动ownership含taskId、每个PID/StartTime/executable/port/目录/DB、build SHA/HEAD/lock，不含任何配置秘密。日志文件仅runtime目录，禁止打印原始.env或启动环境。初始化错误只打印安全类型/exitCode，保持原本日志仅本地隔离目录而非.harness。脚本不得任意kill既有8080/5173服务，失败时只停止此次真实启动且PID/StartTime匹配的任务进程，无递归删除。已有目录保留以供验收。

command-plan-runtime.json含taskId/caseId/cwd/executable/arguments数组/scriptPath/timeoutSeconds/environmentVariableNames/expected/sideEffects/cleanupScope。三个文件写出后停止。脚本未执行，状态NOT_RUN。不要等待SSH认证，因为本地启动可独立推进。
