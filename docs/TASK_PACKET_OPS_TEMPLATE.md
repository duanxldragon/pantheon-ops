# Pantheon Ops Task Packet Template

English version: [TASK_PACKET_OPS_TEMPLATE.en.md](./TASK_PACKET_OPS_TEMPLATE.en.md)

这是一份给 `pantheon-ops` 使用的最小 task packet 样例。

适用范围：

- `business/cmdb`
- `business/deploy`
- `business/bizscope`
- `base -> ops` 继承同步

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
- 明确是业务功能闭环，还是共享后台同步
- 明确本轮不处理的业务域或平台域问题

同步要求：
- 仅本仓业务改动
- 或 `base -> ops` 同步
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

- 写清 base commit
- 写清共享路径哪些同步、哪些故意未同步
- 写清 `business/*` 路径是否保持原样
- 写清 base 和 ops 各自的最小验证结果
