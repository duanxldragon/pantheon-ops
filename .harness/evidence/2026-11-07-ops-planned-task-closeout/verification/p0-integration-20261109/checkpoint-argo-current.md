# P0 current continuation

## 最新 P0 检查点：2026-11-09（Argo 修复后）

本节覆盖下方历史状态。统一验收服务器仍为115.190.251.143；授权与现有加密凭证引用保持有效。真实迁移库v6处于36/dirty0。v10浏览器14/18通过；v13进一步使两种主题的部署选择/预览/日志工作台全部通过，整轮15/18通过。Tempo深色未进入数据源请求；Loki两个场景在登录处90秒超时。新的SSH会话/认证也连续超时，延长单次认证预算后的只读检查仍失败。不能将它们解释为provider功能通过或产品查询缺陷。

业务域Argo新增五项真实MySQL/HTTP回归：同步拒绝不能返回成功、受理不能提前完成、回滚按provider历史ID、请求取消、provider完成时间与错误脱敏。前三项已保存修复前失败，完整Argo模块CGO race通过。HTTP受理现在为Running/无finishedAt，不改共享网络/TLS策略、数据库schema或Base锁。真实Argo、资源差异、terminal history关联、请求结果不确定性与权限边界仍开放。新的完整源码CGO race正在运行，之前713文件通过只证明旧backend检查点。

当前frontend lint、正式v0.18.6重建、overlay与shared drift通过；ownership/document记录随后收口。已检查v13的dark移动目标编辑与历史工作台截图；共享ExecutionStepRail暗色白底/弱对比仍需Base正式修复消费。Argo v3.2.0官方固定版本安装脚本已启动（PID1231440/测试专属namespace），后续状态因SSH失败未核实；需先核实/清理已拥有资源，再继续真实provider验收。所有O2/O3/O4/D1/D2/D3仍开放，独立CODE/ARCH评审不可用，未发行/推送/替换基线。

证据：[当前渲染与远程结果](verification/p0-integration-20261109/rendered-v13-checkpoint.md)、[Argo修复边界](verification/p0-integration-20261109/argo-action-plan.md)、verification/p0-integration-20261109/{argo-action-before.txt,argo-action-after.txt,remote-p0-browser-v10-result.json,remote-p0-browser-v13-result.json,argo-install-current-result.json,argo-current-*-result.json}。
