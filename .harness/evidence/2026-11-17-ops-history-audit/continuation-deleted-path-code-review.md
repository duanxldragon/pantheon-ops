**REQUEST CHANGES。** 本轮审查 7 个目标文件；发现 **MEDIUM 3**，无 CRITICAL/HIGH。删除路径修复本身未发现阻塞。

1. **[MEDIUM] 现行权限表仍使用失效的 `list` key。**  
   [CMDB 中文:356](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md:356)、[英文:166](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.en.md:166) 与 [Deploy 中文:388](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md:388)、[英文:128](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md:128) 仍将列表 API 标为 `label:list` / `package:list`，但当前 provider 绑定使用 `view`，旧别名未找到。**修复：**同步 API 表及权限粒度说明，区分历史目标与当前绑定。

2. **[MEDIUM] CMDB 英文 SSH 状态与实现、中文版矛盾。**  
   [英文:256](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.en.md:256) 仍称超时/端口硬编码。当前已有配置解析、`Collect` 接线及 `Create` 默认端口消费。**修复：**同步中文版“已实现 reader 消费、供应 seam/seed 尚未完成”的准确边界，保留原任务开放。

3. **[MEDIUM] 模板 API 映射完整性的断言缺乏当前证据。**  
   [Deploy 中文:700](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md:700)、[英文:248](/D:/workspace/go/pantheon-platform/pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.en.md:248) 引用的 `permission_policies.go` 已不存在；当前 provider 未找到四项模板绑定，seed 测试只证明按钮/权限 key。**修复：**分别记录已实现 seed 与未证明的 API 授权，移除失效锚点并保留对应 gap。

验证：红回归重现 ENOENT；删除路径过滤后显式缺失声明仍抛错。最新 receipt 已完成：**overlay 59/59、脚本 238/238、正式重建及暂存 overlay 均通过**。生产子进程新增 `windowsHide` 合理。

导航相对定位器及 `toContainText` 保留真实断言，但运行仍为 **1 passed / 1 failed / 1 not run**；不能确认超时由并发负载导致。没有新增渲染截图，不声称视觉或导航验收完成。

LSP/AST 不可用；完整 race、runtime 与最终批准仍独立开放。只读角色未写报告，请主代理将此结论保存为指定 additive report。
