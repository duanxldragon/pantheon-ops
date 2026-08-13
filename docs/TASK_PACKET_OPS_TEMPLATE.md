# Pantheon Ops Task Packet Template

English version: [TASK_PACKET_OPS_TEMPLATE.en.md](./TASK_PACKET_OPS_TEMPLATE.en.md)

这是一份给 `pantheon-ops` 使用的最小 task packet 样例。

适用范围：

- `business/cmdb`
- `business/deploy`
- `business/bizscope`
- Base 快照 + business overlay 重建

直接复制后补全即可：

```text
目标仓库：pantheon-ops
层级：business/cmdb / business/deploy / business/bizscope / inheritance-sync
任务模式：review / implement / ui / inheritance-sync / smoke / docs
先读：
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-base/DESIGN.md
- pantheon-base/AGENTS.md
- pantheon-base/docs/README.md
- 对应 base contract / design / acceptance
- 对应 BUSINESS_* 文档

实现范围：
- 明确是业务功能闭环，还是完整 Base 快照重建
- 明确本轮不处理的业务域或平台域问题
- 先应用最小复杂度阶梯：不做 / 复用 base 或 ops 现有能力 / 标准库 / 平台原生 / 已安装依赖 / 一条局部表达式 / 最小新增代码

同步要求：
- 仅本仓业务改动
- 或 Base 快照 + business overlay 重建
- 如果发现根因属于共享壳层或系统域，回 base 修，不在 ops 本地 override

验证方式：
- Backend: `go test ...` / `go test ./...`
- Frontend: `npm run build`
- Smoke: 业务 smoke / 全链路 smoke / `none`
- UI 任务补 rendered evidence，或明确说明无法渲染原因

停点：
- 如果要修改 base version、删除 inherited override、改业务数据库结构、或回流共享逻辑，先停下确认
```

同步类任务额外要求：

- 写清 Base commit
- 写清 `business-overlay.json` 是否变化
- 写清业务资产比对、生成注册表和幂等重建结果
- 写清暂存树的 backend、frontend、business smoke 和 SonarCloud 结果
