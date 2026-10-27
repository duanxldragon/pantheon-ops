# O1 Unified Context And Target Selector

- Priority: `P1`
- Layer: `business/bizscope + business/cmdb`
- Status: `blocked`
- Base Dependency: B3 `ContextSelector` in immutable foundation release
- Consumers: Deploy, K8s release, file distribution, bulk operations

## Outcome

提供统一的业务上下文与目标选择体验，支持拓扑、分组、主机、集群资源、手工输入和排除集合，而不绕过 DataScope。

## In Scope

- BizScope/environment context header and selector adapter。
- CMDB topology/group/host adapters，remote search、分页/虚拟化和状态提示。
- 手工 IP/ID 输入的解析、去重、匹配、未知/越权项反馈。
- selected/excluded/invalid summary、目标快照和提交前 revalidation。

## Out Of Scope

- 不在 Ops 复制 `ContextSelector` 外壳。
- 不把客户端选择结果当作授权事实。
- 不扩大现有 BizScope/CMDB ownership 或直接跨模块读表。

## Contract Anchors

- `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md` §5。
- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`。
- BizScope/CMDB provider-owned typed capability contracts。

## Expected Files

- `frontend/src/modules/business/shared/` 下业务 adapter/composition。
- 各消费模块的最小接入和业务 i18n。
- provider-owned backend read/resolve endpoint 仅在现有合同不足时增加。

## Acceptance

1. 切换 BizScope 会清理或标记不再合法的选择，不保留越权幽灵目标。
2. 手工输入报告有效、重复、未知、越权和状态不符数量。
3. 服务端在创建/执行前按稳定 ID、scope version 和状态重新解析。
4. 审计保存来源分布、排除规则、解析时间和最终数量，不保存秘密。
5. desktop 多栏与 mobile 分步视图可完成全键盘选择；10k 候选不失控。

## Evidence

- unit/integration：parser、dedupe、DataScope、scope change、stale ID、TOCTOU revalidation。
- permission matrix：view/search/select/resolve 与消费动作。
- Playwright：topology/group/manual/exclusion、mobile、error/forbidden/partial invalid。
- screenshots：desktop/mobile x populated/invalid/forbidden x light/dark。

## Gates

- Base lock、跨模块 typed capability、新 API/schema 和共享组件 drift。
- 外部生产资源不在本包内创建或修改。
