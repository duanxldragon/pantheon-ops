# 只写：最小本地启动入口，立即交三个文件

延续cb461a8b-4839-4a48-b926-ee867df4d278。你已经完整读取上下文，前一片段长期思考而没有任何文件。现在降低effort，只给Write/Edit；你的任务是文件交付，不继续规划。最多3个Write调用：start-local-runtime.ps1、runtime-env.ps1、evidence/command-plan-runtime.json。每个PS1尽量不超过180行，无新框架/新C#类/复杂通用runner，直接ProcessStartInfo和固定路径。全部前提见上轮CLAUDE_RUNTIME_WRITE_NOW.md已在上下文，不要读文件。

用户最终范围保持：本地Ops，115.190.251.143是root密码SSH目标，从CMDB开始。root真实登录及指纹已成功。后端当前源码二进制已成功构建在.local/acceptance-20261121/pantheon-ops.exe，SHA256=1c51aa48b65670f123df76c2f6d78366b75ef4f201019a40de923c80bb7b0fd8；HEAD a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc，Go1.26.11。frontend lint0error1warning/build通过。

立即实现隔离MySQL33321/Redis26321/backend8081/vite5175和DB ops_acceptance_20261121，只在.local/acceptance-20261121/runtime写东西，不停止已有服务。runtime和端口必须预检。mysqld.exe/mysql.exe D:/MySQL/mysql/bin；Redis D:/Redis/redis-server.exe；Node D:/nodejs/node.exe；vite frontend/node_modules/vite/bin/vite.js。所有后台隐藏、有界wait，日志只本地runtime且不打印secret，记录真实ownership/hash/readiness。初始化MySQL insecure绑定loopback，然后SQL stdin随机设置root密码及新DB；连接MYSQL_PWD在子进程env而非argv。Redis随机密码配置文件本地受限ACL。随机admin及五个secret用当前用户DPAPI保存，runtime-env.ps1解密设置env但不输出值；此helper会由同一批准执行身份调用。secret不用root服务器密码，服务器密码由Codex单独注入。

真实后端env：PANTHEON_DSN、PORT8081、ENVdevelopment、AUTO_MIGRATEtrue、REDIS_ADDR127.0.0.1:26321、REDIS_PASSWORD随机、REDIS_REQUIREDtrue、INITIAL_ADMIN_PASSWORD随机、ACCESS_TOKEN_SECRET/REFRESH_TOKEN_SECRET/OP_TOKEN_SECRET/SETTING_SECRET/MFA_SECRET随机。不用不存在的PANTHEON_HOST；记录backend目前绑定all interfaces的本机限制。前端host127.0.0.1:5175，PANTHEON_API_PROXY_TARGET=http://127.0.0.1:8081。测试helper设置SMOKE_USERNAMEadmin/SMOKE_PASSWORD/INITIAL_ADMIN密码等于同一值，API_BASE_URL和WEB_BASE_URL，TEST_DSN_REQUIRED1独立DSN。dev自动建表仅本轮运行准备，不冒充正式迁移验收。

少写说明，多交文件。三个文件输出后final写文件名及NOT_RUN即结束。
