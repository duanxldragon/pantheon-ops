# Claude 新交付任务进度核对（2026-11-22）

结论：有实际新增交付与实机证据，整体仍为 **in-progress / changes-required**。本轮 Codex 核对源码、原始运行结果与 task packet，并独立执行固定模板/发布冲突回归；未重新操作服务器、未修改业务实现、未代签旧十包终验。此前 Claude 修改的 review.md 中 reviewer 字段不代表本轮独立签核；本文件为此次实际审查。

## 当前可确认的进展

| 范围 | 核对结果 | 边界 |
|---|---|---|
| K8s run12 / k8s33 | **22 PASS / 0 FAIL / 0 BLOCKED** | 最新原始全套场景，含发布/回滚和 ConfigMap Backup Completed；K8s全范围仍PARTIAL |
| K8s清理 | PASS | 独立namespace缺失确认，临时cluster/scope/user/role、release/storage/backup记录清理见原回执 |
| 部署 d00–d06 | 7 PASS | pinned SSH marker、错误指纹拒绝、注入serviceName在create返回400且未执行；不能据此关闭全部T02安全字段/卸载或V7恢复取消 |
| CMDB c01–c08 | retry有8项PASS | 首试失败保留，涉及瞬时环境与工具错误，不合并重复计数 |
| V1/V2/V5探测 | 19/19断言成立 | default-deny、包列表/导出/详情、脱敏/密文；不代表授权可用性和creator过滤已修 |
| V3 / 部署扩展路由矩阵 | 24/24 与30/30断言成立 | 包含已授全权限仍403的既有缺口，须分开记录现状核实与整改验收 |
| 综合CAP | 31 PASS / 1 BLOCKED | 业务域/目录/实例CRUD、文件上传下载和部分安全路径；observability连接/查询成功未证明，Argo缺环境key |
| 本轮独立回归 | **9顶层用例PASS、exit0、0skip、0fail** | CGO + MinGW + MySQL required + race；固定模板和发布typed409/持久failed/不重试 |

最新K8s：[原始summary](k8s/omx-accept-20261121-k8s12/summary-20261122-084758.json)；[部署注入拒绝](runs/omx_accept_20261121_cmdb02/deploy/d05.retry-1795266079971.json)；[本轮独立测试](codex-progress-tests-20261122-113249.json)。

## 需要纠正的结论

1. **observability未验收完整查询。** 连接测试返回500 `test_failed`，查询返回400 `param.invalid`。PASS断言仅为20秒内返回结构化响应；可确认参数/错误响应有界，不能关闭真实provider查询。filedist上传下载也不能替代E01远端分发、校验、取消和重启闭环。
2. **现状探测PASS与整改完成分开。** 父计划要求V1查看角色可读授权任务、V2安全详情、V3已授动作可用；实际仍有403，T03仍待修。V5通过全路由拒绝保护非管理员，不能证明按创建者过滤已存在。
3. **Velero阻塞结论已过时。** run12已在当前k8s33上备份Completed，原STATUS/报告仍指run11，22PASS+1BLOCKED还合并了旧UIfollowup。恢复和调度仍缺，不能把22场景PASS扩成全部K8s功能验收。
4. **完整race仍红。** 日志直接支持缺测试Redis；四包超时的性能/资源根因需补齐依赖重跑验证，“环境性”分类不足以排除代码问题。
5. **证据绑定/参数语义要补。** CAP manifest没有二进制/source回执；部署回执缺一个test文件，其余hash匹配，K8s74源hash均匹配。备份请求false/响应true需核实DB与实际CRD，零快照只能证明本次没有volume snapshot。

## 任务树及旧十包

本次新任务树18张卡（父任务+17子包）：**2 in-progress（父任务、T16）/16 planned/0 completed**。T02、T06、T07有实际局部代码和验收进展，卡片尚未对账；它们全部验收标准尚未独立完成。旧C01–C06/E01–E04归档不改变未验收residual，各criterion仍大量NOT_RUN/BLOCKED，旧“无可用K8s”原因需更新为已测部分及真实剩余gap。

## 剩余工作

- V4双租户与跨归属副作用
- V6浏览器10秒/长操作取消与可见终态
- V7 SSH重启、取消和租约恢复
- V8 ServiceAccount撤销与持续工作负载轮询
- Velero恢复/调度、Ingress快照与全新集群安装
- WebSocket服务端释放、移动/暗色/键盘UI
- Argo及业务密钥运行配置与真实provider
- 完整仓库race、正式重建/drift、旧十包逐条验收与T17交付安装升级回滚
- 遗留测试主体二次验证清理

本轮只新增进度审查和测试工具/日志，保留Claude证据原件及dirty worktree；未提交、push、发布或执行远端修复。下一个具体步骤是让Claude按本报告修正状态/矩阵与场景断言，优先补T03权限可用性、V4/V6–V8和真实provider闭环；业务实现仍由Claude负责。

详细机器回执：[codex-progress-review-20261122.json](codex-progress-review-20261122.json)。
