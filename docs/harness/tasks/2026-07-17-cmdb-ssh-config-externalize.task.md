目标仓库：pantheon-ops
层级：business/cmdb
任务模式：implement

先读：
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md §11.2
- pantheon-base 的 system/config 配置读取契约（对应 base design 文档）

背景：
CMDB 设计 §11.2 声称 SSH 采集配置通过 system/config 管理，但实际
`cmdb.ssh.collect_timeout` 硬编码为 10s（host_service.go:303），
`cmdb.ssh.default_port` 只存在于模型默认值。无 config seed。

实现范围：
- 新增配置 seed：`cmdb.ssh.collect_timeout`（默认 10）、
  `cmdb.ssh.default_port`（默认 22），归属模块 business.cmdb
- host collect 链路读取配置值，读不到时回落当前默认值（10s / 22）
- 配置读取通过 base 提供的 config 能力接口，不直接 import
  system/config 内部 Service（遵守 CMDB 设计 §2 边界）
- 同步更新 BUSINESS_CMDB_MODULE_DESIGN.md §11.2：去掉"待办"标注

不处理：
- WinRM / Agent 采集
- 其他 cmdb 配置项
- system/config 模块本身的任何改动

同步要求：
- 仅本仓业务改动
- 若发现 base 未暴露业务模块可用的配置读取扩展点，属于共享层缺口，
  停下报告，回 base 补，不在 ops 本地绕过

验证方式：
- Backend: `go test ./backend/modules/business/cmdb/host`
- 手动/测试验证：改配置值后 collect 超时行为随之变化
- Smoke: none

停点：
- 见同步要求；另若配置读取需要新增跨模块依赖方向，先停下确认
