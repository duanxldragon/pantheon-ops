# 只写续跑：CMDB首批HTTP脚本及SSH helper修正

继续2d1fb07d-e5a0-4258-9918-a5053fe9a515。已经读到准确routes/DTO/auth/collect/deploy live test，上轮30次调用只写ssh-probe.py，首轮Reviewer已拒绝。只给Write/Edit工具，不要再探索，不去读不存在的Read/Glob工具。直接Write覆盖你自己写的ssh-probe.py，再写cmdb-ssh-live.mjs、command-plan-cmdb.json、cmdb-generator.md。文件越小越好，不造框架。只先完成--phase cmdb；部署阶段下个片段复用本次fixtures增量做。

Reviewer实测发现：_confine_path('/tmp/omx_accept_20261121_review/../../etc/passwd')被接受；同known_hosts公钥_presented_matches_entry为False。你必须修：先posixpath规范化路径并限定给定runId的根目录，远端操作前realpath验证symlink不能逃逸；main任意okFalse需nonzero；总deadline覆盖stdout/stderr和exit-status以免输出截断后阻塞。CLI只允许115.190.251.143/22/root，连接前拒绝其他目标。删除自写known_hosts parsing及重复wire包装，直接Paramiko HostKeys.lookup和key.asbytes比较；支持PANTHEON_VERIFY_SSH_KNOWN_HOSTS（默认为当前任务.local/acceptance-20261121/known_hosts，已经保存经既有known_hosts核实的公钥），RejectPolicy不更新全局信任。

trusted host SHA256:FhLFRIfRIGotV2tS+sIZWxniie2qG5Xu8H3igYJ8RNU，真实root密码登录成功，Linux6.8.0-55-generic。秘密由Codex从机器绑定DPAPI保管后通过PANTHEON_VERIFY_SSH_PASSWORD env注入，不要读private*.dpapi文件内容。Python paramiko已安装，可只给fingerprint/probe/assert-absent/expect-file四个入口，先删除cleanup命令避免没必要的远端root递归清理；后续由严格fixture边界的独立脚本处理。测试目标只/tmp/omx_accept_20261121_<id>目录。

本地API8081/Web5175由另一Claude启动脚本准备；从env取PANTHEON_SMOKE_USERNAME/PANTHEON_SMOKE_PASSWORD/PANTHEON_INITIAL_ADMIN_PASSWORD，API http://127.0.0.1:8081/api/v1。首批只正常HTTP登录CSRF -> 测试组/业务域/环境 -> 测试主机(115.190.251.143)、密码SSH凭据 -> 主机指纹录入、真实采集/连接 -> 列表/详情/编辑/标签/分页 -> 凭据响应脱敏/失配拒绝/无远端marker。保持当前准确DTO，不直接写DB。created ID/prefix记录成可续用checkpoint，别立即删主机/凭据/组，因为下轮部署会复用。凭据private字段只记录boolean，不dump任何secret请求/响应。

HTTP/auth复用frontend/scripts/lib/auth-cookie-session.mjs，但先记你已读的真实导出和调用方式；不从旧foundation版本import。Node subprocess arguments数组/windowsHide:true继承既有hide-child-windows.mjs，Python输入密码不在argv。脚本不能读取私密原始文件，测试密码以env注入。每个case记录实际HTTP状态+business code+断言+真实远端结果，只本任务evidence唯一输出不覆盖历史；FAILED立即非0并保留失败fixture，NOT_RUN不当PASS。

CMDB首批可运行即可，别在本片段写全面矩阵/UI/所有V项目。command-plan精确argv/env名/timeout/cwd/expected/副作用及cleanup scope。脚本未执行时只写NOT_RUN。
