# SSH验收脚本首轮审查

- Task: 2026-11-20-ops-review-live-runtime-acceptance
- Reviewer: Codex /root
- Reviewed file: .local/acceptance-20261121/ssh-probe.py
- Verdict: changes-required；尚未批准执行其远端写入/清理入口
- Review date: 2026-11-21

## Findings

1. 路径约束可被 `..` 越过。调用 `_confine_path('/tmp/omx_accept_20261121_review/../../etc/passwd')` 返回原路径，终端结果 `path_escape_accepted=True`。必须先规范化绝对路径、按当前runId限定根目录，并核实远端realpath及符号链接边界；不能只检查字符串prefix。任何清理/读取应在命令执行前拒绝越界。
2. 主机指纹比较错误。用本任务known_hosts同一ED25519公钥调用 `_presented_matches_entry` 得到False。PKey序列化本已包含SSH wire key字段，当前重复包装且使用不正确方法。需用Paramiko支持的公钥bytes直接比较，并通过同公钥与不同公钥两种反例验证。False不得同时返回ok=true或退出0。
3. `_parse_known_hosts_entry` 对hash hostname未核实目标便返回首个hashed条目，且不处理逗号分隔hostnames；应复用Paramiko HostKeys.lookup，不另写不可靠解析器。
4. CLI仍接受任意host/port/user，未固定授权115.190.251.143:22/root；密码验证不能被参数改为其他目标。必须在连接之前校验。
5. known_hosts仅从用户目录读取，不能指定已经核实的任务专用known_hosts。需支持PANTHEON_VERIFY_SSH_KNOWN_HOSTS（本任务文件），保持RejectPolicy，不更新用户全局信任。
6. main在ok=false/knownHostsVerified=false时仍返回0；各子命令必须同时通过预期断言及退出码。输出读取/等待exit-status需总deadline，避免截断后channel阻塞；错误只输出安全元数据。

## Actual verification

本地只读Python导入执行了两个函数级反例，真实结果为：

```text
path_escape_accepted=True
same_host_key_matches=False
trusted_sha256=SHA256:FhLFRIfRIGotV2tS+sIZWxniie2qG5Xu8H3igYJ8RNU
```

没有用上述脚本执行远端写入或删除。独立密码SSH只读连接已经通过，服务器为Linux6.8.0-55-generic，root，主机指纹与原known_hosts匹配。

## Handoff

Claude修正脚本与调用方后，Codex重新执行本地安全反例、真实指纹与只读probe，再验CMDB/SSH业务链路。不得降低验证标准或忽略knownHostsVerified字段。
