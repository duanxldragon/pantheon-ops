# Generator任务：从CMDB开始的真实SSH业务验收

最新维护者范围：Pantheon Ops本地运行，115.190.251.143只是验证部署功能的SSH目标，从CMDB开始。root密码已提供，由Codex安全保管，你不需要/不得读取密码。Claude负责执行脚本/业务失败修复，Codex只验收实际结果。你是Generator而不是Planner，AGENTS的Planner不改backend/frontend约束不要误套为“业务代码交给Codex”。你不是唯一执行者，不撤销别人修改。

独占文件：.local/acceptance-20261121/cmdb-ssh-live.mjs、ssh-probe.py、必要的同prefix辅助/Playwright spec/config、以及本任务evidence下command-plan-cmdb.json、cmdb-generator.md。另一Claude写本地环境启动脚本/DPAPI env helper，你不能修改其runtime-env.ps1/start-local-runtime.ps1，不能改总矩阵/summary/manifest。最多6次聚焦读取后开始Write，禁止读Claude的result.jsonl/process.json/stderr.log或重复扫旧证据。

本机python已安装paramiko/cryptography。使用paramiko作为密码SSH transport（没有新依赖），SSH服务器端口22、root，已通过本机known_hosts核对ED25519 host key。身份密码由runner通过stdin JSON或PANTHEON_VERIFY_SSH_PASSWORD环境传入，脚本不把值输出/写日志/argv。Paramiko从当前Windows ~/.ssh/known_hosts加载，RejectPolicy，不AutoAdd，不读私钥。固定server授权115.190.251.143；timeout/auth_timeout/banner_timeout/命令输出有界，只输出去敏JSON。ssh-probe只读uname/id/容量与任务独立目录是否存在，后续校验marker/hash/进程仅本任务prefix资源，禁止遍历秘密。不要要求sshpass或免密；用户已授权root密码登录，不要让“必须免密”成为前置。

本地新隔离环境由另一Claude准备API http://127.0.0.1:8081/api/v1、Web http://127.0.0.1:5175，独立DB/Redis。admin初始密码等由runtime-env.ps1解密提供PANTHEON_SMOKE_PASSWORD或正确变量，按脚本协定使用env。测试SSH秘密独立由Codex注入。服务候选build由Codex从当前源码构建。

首批必须从正常HTTP业务入口验证：登录/CSRF -> 创建测试业务域/环境/组 -> 录入115.190.251.143的测试CMDB主机 -> SSH凭据创建(当前真正路由/字段)并绑定主机 -> 主机指纹/连通性/更新/标签/详情/分页 -> 部署包和固定模板 -> 如果时间允许关联服务目录/实例及SSH任务运行。请先读取当前module routes/DTO、frontend API和现有cmdb/deploy smoke用于路由与payload校准；只接受真实正常产品入口而非直接写DB给用例绕路。UI覆盖用真实浏览器，1440/390和主按钮关键状态按impeccable审查，不能mock API/provider。每个HTTP案例记录真实状态/业务code、必要非敏感ID、作用结果，不完整dump请求/响应secret。凭据响应需检查明文/密文字段不泄露，记录boolean/count而非值。

使用唯一prefix omx_accept_20261121_<runId>，远端目录只 /tmp/omx_accept_20261121_<runId>，只在此目录创建测试包/脚本/标记，不安装软件，不操作既有业务/系统服务，不重启服务器、不改防火墙/global配置。业务主机真实指纹要读并校验，失配负例必须执行前拒绝且无marker。命令模板注入以产品结构参数提供无害marker，若目标marker存在则FAIL并停止该用例，不继续扩大副作用。

首批脚本先完成CMDB和凭据的真实正反例，再具备部署任务创建/预览/start/poll终态/日志回读入口。受限角色正反例必须真实账号权限按独立授权设置；缺正确权限或发现当前产品缺陷不能改预期成PASS。全量V1-V8和旧十包按后续小片段推进，本片段只完成CMDB->SSH部署首批，别再次耗尽全部上下文整理矩阵。

以可重跑脚本+command-plan交付；支持--phase cmdb 或 deploy、--run-id；每条真实运行写本任务evidence唯一文件，失败不覆盖原结果。command-plan含caseId/cwd/executable/arguments数组/scriptPath/env变量名/timeout/预期/副作用/清理范围，密码不是参数值。清理只能当前fixture ID且核对prefix，保留失败记录；不DROP整库、不删其他任务目录。脚本未执行不写pass。你交付后Codex会执行并把结果回给你修复。
