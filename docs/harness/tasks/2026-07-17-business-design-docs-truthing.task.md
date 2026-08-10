目标仓库：pantheon-ops
层级：business/cmdb + business/deploy + business/bizscope + docs
任务模式：docs

先读：
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-ops/docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md
- pantheon-ops/docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md
- pantheon-ops/docs/designs/BUSINESS_BIZSCOPE_MODULE_DESIGN.md
- pantheon-ops/docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md
- pantheon-ops/docs/designs/PLATFORM_SRE_EVOLUTION_PLAN.md
- pantheon-ops/docs/designs/FOUNDATION_UPGRADE_PATH.md

背景：
2026-07-17 对 business 模块设计文档与代码基线（最后业务提交 2026-06-24
`feat: close out ops business workflows`）做了逐项审计。以下修订项全部经过
代码证据核实，本任务只做文档如实化（truthing），不改任何代码行为。

实现范围（逐文档逐项，全部为文档改动）：

一、BUSINESS_DEPLOY_MODULE_DESIGN.md
1. §7.2 接口清单补两行（代码证据 backend/modules/business/deploy/deploy_handler.go:23,35）：
   - `/packages/:id` GET 软件组件详情。权限口径如实写：路由已实现，但
     `business:deploy:package:list` 的 Casbin 映射
     （permission_policies.go:137-140）只覆盖 `GET /packages`，
     `/packages/:id` 无任何权限映射——文档标注该缺口，指向任务卡
     2026-07-17-business-permission-gaps，不得写成已授权事实。
   - `/tasks/:id` DELETE 删除未启动任务（draft/pending，同步清理 task_host 明细）
     权限 `business:deploy:task:delete`
2. §7.3 补 `DELETE /tasks/:id` 要点：仅 draft/pending 可删，
   关键错误 `business.deploy.task.invalidDeleteState`（附录已有该 key）。
3. §9 权限点清单补：`business:deploy:task:delete`（deploy_seed.go:36 已 seed）；
   `business:deploy:template:list/create/update/delete` 以如下口径补入：
   四项均已在 `backend/pkg/contracts/permission_policies.go:153-168` 定义
   API 路由映射，但 deploy_seed.go 仅 seed 了 template 列表菜单（C 项，
   PagePerm=template:list），create/update/delete 的 F 按钮 seed 缺失——
   文档写入权限清单的同时显式标注该 seed 缺口，指向任务卡
   2026-07-17-business-permission-gaps。
4. §9 审计动作补：删除部署任务。
5. §7.2 `/task-hosts/:id/report` 行加现状标注：当前与 `/result` 绑定同一
   handler（deploy_handler.go:39），"人工标记 vs 执行器上报"来源区分尚未实现，
   属于 Agent 阶段待办。
6. 头部"更新时间"改为 2026-07-17。

二、BUSINESS_CMDB_MODULE_DESIGN.md
1. §5.1 Host 表补三列（代码证据 host_model.go:23-25）：
   `business_scope_id` bigint INDEX、`business_scope_code` varchar(64)、
   `business_scope_name` varchar(128)，说明为 bizscope 绑定快照。
2. §6.3 Label API 补一行：`GET /labels/options` 标签规范下拉选项
   （label_handler.go:21）。权限口径如实写：路由已实现，但
   `business:cmdb:label:list` 的 Casbin 路由映射
   （permission_policies.go:96-99）只覆盖 `GET /labels`，未覆盖
   `/labels/options`——文档标注该权限映射缺口，指向任务卡
   2026-07-17-business-permission-gaps，不得将其写成已授权事实。
3. §11.2 依赖配置改为如实描述：`cmdb.ssh.collect_timeout` /
   `cmdb.ssh.default_port` 当前未接入 system/config，超时硬编码 10s
   （host_service.go:303）、端口默认 22 在模型层。保留原目标态描述，
   显式标注"配置化为待办，见任务卡 2026-07-17-cmdb-ssh-config-externalize"。
4. §13 seed 表菜单行补 `operations.cmdb.label`；组件键注册补
   `business/cmdb/host/CmdbHostDetail`、`business/cmdb/label/CmdbLabelSchemaList`。
5. §4.3 状态表 `online` 展示语义由"已上线"改为"可运维"，与 dict seed
   （cmdb/module.go:422）对齐；§4.3 前后文表述同步。
6. 头部"更新时间"改为 2026-07-17。

三、BUSINESS_ERROR_SEMANTICS_APPENDIX.md
1. 新增 bizscope 章节（放在 Deploy 清单之后）：
   - 现状清单：`bizscope.code_exists`、`bizscope.in_use`、`bizscope.not_found`、
     `param.invalid`（bizscope_service.go / bizscope_handler.go 实际返回值）
   - 显式标注：短 key 违反 §1.1 canonical 规则，属于历史债务
   - 给出目标态映射：`business.bizscope.codeExists` / `business.bizscope.inUse` /
     `business.bizscope.notFound`，迁移见任务卡
     2026-07-17-bizscope-error-key-canonicalization
2. 新增 Deploy Template 资源段占位：说明模板 CRUD 已上线但 canonical
   错误 key 清单未定义，列为待补项（不在本任务内发明新 key）。
3. 头部"更新时间"改为 2026-07-17。

四、BUSINESS_BIZSCOPE_MODULE_DESIGN.md
1. §5 错误语义表下方补一句：短 key 为历史现状，canonical 目标态及迁移
   计划见 BUSINESS_ERROR_SEMANTICS_APPENDIX.md bizscope 章节。
2. 头部"更新时间"改为 2026-07-17。

五、PLATFORM_SRE_EVOLUTION_PLAN.md
1. §1.1 代码位置修正：`backend/modules/system/generator/` →
   `backend/modules/lowcode/generator/`；`backend/modules/system/dynamicmodule/`
   → `backend/modules/lowcode/dynamicmodule/`；DSL 条件引擎行号引用改为
   不带行号的 `deploy_service.go`（行号已漂移）。
2. §1.2 "真实执行引擎"行改为如实状态：SSH 执行闭环已在
   `business/deploy/deploy_service.go` 内联实现（ssh.Dial 真实执行 + 结果回写），
   阶段一剩余工作为：抽出独立 `backend/pkg/executor/` 包 + Agent 执行器。
3. §2 阶段一表格同步：1.1/1.3 标注"已以内联方式实现，待重构抽包"，
   1.2/1.4 保持待办。
4. §4 时间线加一行说明：阶段一已部分完成（2026-06 deploy 闭环轮），
   剩余以重构与 Agent 为主，预估周数由 3-4 周调整为 1-2 周。
5. 头部"更新时间"改为 2026-07-17，状态保持 Active（基线已重打）。

六、FOUNDATION_UPGRADE_PATH.md
1. 改造为滚动文档：标题下加"当前基线"段落，写明 lock 已在
   `base-v0.8.11`（foundation-release.lock.json，commit a201a13）；
   原 0.8.4→0.8.5 内容整体降为"历史升级记录"章节保留。
2. 0.8.5→0.8.11 之间路径以一行说明收口：经 0.8.8/0.8.9/0.8.10/0.8.11
   多次 foundation-release-consumer 升级（.foundation/releases/ 有产物），
   逐版本明细未记录，后续升级必须在本文档追加记录。
3. 补齐头部规范字段（更新时间 2026-07-17 / 类型 Design / 归属层 platform /
   状态 Active）。
4. 同步 docs/README.md 第 25 行对该文档的描述（去掉写死的 0.8.4→0.8.5）。

七、英文镜像回流
以上六份文档改完后，同步各自 .en.md：
- BUSINESS_CMDB_MODULE_DESIGN.en.md（当前 117 行 vs 中文 705 行，需整体补齐结构）
- BUSINESS_DEPLOY_MODULE_DESIGN.en.md（200 vs 588，需整体补齐结构）
- BUSINESS_ERROR_SEMANTICS_APPENDIX.en.md（100 vs 197）
- BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md（仅增量）
- PLATFORM_SRE_EVOLUTION_PLAN.en.md（仅增量）
- FOUNDATION_UPGRADE_PATH.en.md（结构改造同步）
英文版允许为中文版的忠实精简翻译，但章节结构和事实陈述必须一致，
头部 Updated 与中文版同日。

不处理：
- 任何 backend/、frontend/src/ 代码改动
- bizscope 错误 key 的实际迁移（独立任务卡）
- cmdb.ssh 配置外部化实现（独立任务卡）
- base 共享路径与 foundation 升级本体
- 不发明附录中不存在的新错误 key

同步要求：
- 仅本仓文档改动
- 不在 ops 本地 override base 行为

验证方式：
- `node scripts/check-inheritance-contract.mjs`（如脚本存在）
- 逐项对照本任务卡"实现范围"自查：每一项都能在 git diff 中找到对应改动
- 中英文头部日期一致性 grep 抽查
- Smoke: none（纯文档）

停点：
- 如发现本任务卡所述代码证据与实际代码不符，停下报告差异，不要自行改口径
