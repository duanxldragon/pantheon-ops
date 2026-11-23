# 续跑只写片段：立刻交付本地环境三个文件

继续session cb461a8b-4839-4a48-b926-ee867df4d278。上轮24次读取达到上限，三个要求文件一个也未写。探索到此结束，本轮只给Write/Edit工具，直接生成文件，不尝试Read/Glob/Grep、不再要求Codex替你写脚本。

依然由Claude作为Generator负责 .local/acceptance-20261121/start-local-runtime.ps1、runtime-env.ps1 和本任务evidence/command-plan-runtime.json。独占三个文件，不修改CMDB执行者的文件，不读取凭据/Claude日志，不改业务源码。Codex验收和跑审核后命令。

确定环境：当前候选 .local/acceptance-20261121/pantheon-ops.exe 已由Codex go build -buildvcs=false -mod=readonly 成功，Go1.26.11，尚未启动。根目录D:/workspace/go/pantheon-platform/pantheon-ops。二进制server的实际变量为PANTHEON_DSN、PANTHEON_REDIS_ADDR/PASSWORD、PANTHEON_REDIS_REQUIRED、PANTHEON_ENV、PANTHEON_AUTO_MIGRATE、PANTHEON_PORT。secret实际变量PANTHEON_ACCESS_TOKEN_SECRET/REFRESH_TOKEN_SECRET/OP_TOKEN_SECRET/SETTING_SECRET/MFA_SECRET；初始admin PANTHEON_INITIAL_ADMIN_PASSWORD。frontend/vite.config.ts第39行代理变量PANTHEON_API_PROXY_TARGET。

按前次要求新建隔离MySQL33321/Redis26321/backend8081/vite5175，不停既有3306/6379/5173。DB ops_acceptance_20261121；运行目录 .local/acceptance-20261121/runtime 不已存在时才创建，有端口占用提前失败。工具D:/MySQL/mysql/bin/mysqld.exe及mysql.exe、D:/Redis/redis-server.exe、D:/nodejs/node.exe、frontend/node_modules/vite/bin/vite.js。只能当前用户DPAPI加密存生成随机password/各secret；helper设置PANTHEON_SMOKE_USERNAME=admin和PANTHEON_SMOKE_PASSWORD（等于随机初始admin），PANTHEON_API_BASE_URL=http://127.0.0.1:8081/api/v1、PANTHEON_WEB_BASE_URL=http://127.0.0.1:5175、PANTHEON_TEST_DSN_REQUIRED=1。helper一律不输出secret或原始解密结果，不存明文DSN日志。后台隐藏启动，Node使用既有hide-child-windows.mjs。

dev AutoMigrate=true沿用当前产品开发入口，同时记录迁移版本需后续正式空库升级验证gap，不假装这就关闭安装升级回滚。MySQL先--initialize-insecure后绑定127.0.0.1启动，并通过客户端stdin设置随机root密码、创建独立DB，连接凭据用ProcessStartInfo.Environment['MYSQL_PWD']内存传递不放argv；不把密码SQL输出到日志。Redis配置requirepass随机密码只可当前用户读的本地runtime目录文件，不写evidence，不公开监听。backend当前未提供bind host变量，记录服务8081可能监听all interfaces的本机边界，不编造有效PANTHEON_HOST；前端loopback。

所有启动等待/输入输出有界，启动ownership仅含PID/start/exe/ports/目录、候选binary hash/git HEAD/lock不含秘密。错误保留日志在runtime，不原样打印。只清理本次启动且PID/start/exe核实的进程，保留目录，零递归删除。不要为了追求复杂脚本陷入新一轮规划；复用已有范式写最小可靠入口。

SSH目标root密码已在Codex经过确认指纹验证成功（Linux6.8.0-55-generic），不是本地runtime前提。此续跑写完三份文件即交回，不等待任何认证。
