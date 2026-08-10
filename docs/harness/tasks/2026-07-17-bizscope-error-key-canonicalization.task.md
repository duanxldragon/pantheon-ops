目标仓库：pantheon-ops
层级：business/bizscope
任务模式：implement

先读：
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-ops/docs/designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md
- pantheon-ops/docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md（bizscope 章节）

背景：
bizscope 后端当前返回非 canonical 短 key（bizscope_service.go /
bizscope_handler.go）：`bizscope.code_exists`、`bizscope.in_use`、
`bizscope.not_found`，以及 handler 侧 `bizscope.list.error` 等流水 key。
违反附录 §1.1 的 `business.<module>.<resource>.<reason>` 规则。

实现范围：
- 后端错误 key 迁移到 canonical 形态：
  - `bizscope.code_exists` → `business.bizscope.codeExists`
  - `bizscope.in_use` → `business.bizscope.inUse`
  - `bizscope.not_found` → `business.bizscope.notFound`
  - handler 侧 `bizscope.<action>.error` 系列统一评估：能归入业务语义的
    改 canonical，纯请求失败的回落 base 通用 key（`request.failed` 等）
- 前端 i18n 语言包同步新 key（zh-CN / en-US），旧 key 保留一个版本作为
  fallback 映射或直接清除（按前端错误翻译链路实际情况决定，二选一并记录）
- 同步更新 BUSINESS_ERROR_SEMANTICS_APPENDIX.md bizscope 章节：
  把"现状/目标态映射"改为"已迁移"清单
- 同步更新 BUSINESS_BIZSCOPE_MODULE_DESIGN.md §5 错误语义表

不处理：
- bizscope 之外模块的错误 key
- 权限点、路由、审计 action 变更
- base 通用错误规则调整

同步要求：
- 仅本仓业务改动

验证方式：
- Backend: `go test ./backend/modules/business/bizscope`
- Frontend: `npm run check:i18n-generated`、`npm run check:i18n-missing-keys`
- Smoke: 触发一次重复编码创建与占用删除，确认前端 toast 显示翻译文案
  而非裸 key（截图或 API 断言均可）

停点：
- 若发现前端存在对旧短 key 的硬编码分支判断（非 i18n 翻译），停下报告
