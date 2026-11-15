# 非 Velero P0 接续检查点 — 2026-11-11 23:53 CST

维护者排序继续生效：Velero 最后。O2、O3、O4、D1、D2、D3 六个父 P0 均开放；下面是局部事实，不是最终签署。

## 已有稳定门禁

Base 的 74d43c728ff9 候选已正常生成，1166 个来源文件核对通过。Base 完整 CGO race 已通过 50 个测试包、2197 个测试/子测试；1 个真实 S3 测试因端点未配置而跳过。前一 6a 候选已正常首次装配，6580 个完整文件回读；前端 npm ci/lint/type-check/build/unit 均通过，53 文件/402 单测，无失败/跳过。74 尚未消费，等待 O2 最终共享权限产品修复后合并新的正常候选，避免重复消费旧输入。

上一检查点：[23:20 事实与门禁](../continuation-20261111-non-velero-2320/checkpoint.md)。

## CMDB R3 已实际执行，撤权阶段失败

Root 创建了独立数据库中的 CMDB 元数据夹具 host1，公共回读、creator1、businessScope0/dept0 与唯一原生审计240均验证；未连接或修改远端主机。fixture ready SHA `0aa498e74b533a67be603e1906ae68858e85b8a3e9e7996ea1beafc778d7e3e8`。

新 R3 入口获得真实独立 native CODE APPROVE、ARCH APPROVE/CLEAR。具名 lsp_diagnostics 不可用的原 CODE COMMENT 保留；另一路未参与实现的 CODE 实际执行 TypeScript LanguageService、严格 TS、21 host/26 audit 检查，无诊断。Root 保存 ARCH 原消息的真实核心字段、pins、findings、限制，明确为摘录，不冒称完整原始 JSON。

Root 核对当前 PID28876、实际 exe SHA、端口所有者、来源与两个独立评审后，创建新的请求，先 --check 再仅一次 --run。结果 **failed**：69 个请求、57 个已通过断言，失败为 `exact current CMDB permission keys without wildcard`。

已通过：非管理员 view/export/import 正向与交叉拒绝、invalid import 零写入、同一 scope actor 原会话 all→self list/export 排除其他作者主机、主机快照保持。请求67公开 PUT role35 撤为空权限及68角色回读成功；69原 view 会话 /auth/me 仍与预期空 CMDB 权限集合不符。尚未判定是产品撤权缺陷还是 oracle 问题，交 native adapter 只读源码定位，禁止重放 mutation。

R3 新 roles35–38/users26–29 全部保留，未建立完整一致性，不清理；R2 roles31–34/users22–25 及旧失败也保留。host1保留供后续精确验收。

- [Root 运行前核验](../../../../../.local/ops-acceptance/cmdb-public-key-only-adapter-20261111-r3/root-before-run-verification.json)，SHA `d6254306a7fcc2fae6b94b7aa4d413a6d90daa70fc5f9857c671fc6120bbee46`
- [实际结果](../../../../../.local/ops-acceptance/cmdb-public-key-only-adapter-20261111-r3/result.json)，SHA `f17bda4067a7f7600d96b47e9d8c45d160dfa9ed31c9c5a1f5f94e98c3330921`
- attempt SHA `ee7f3fa0468c996a2949208ca5b3fada6c5dac85b70c084626baf52847f9aa75`；journal SHA `7ab97d7b87acff7f8f966e2be4440fdb715405ea53e293abf449b5d05a091099`
- [独立 native CODE](../../../../../.local/post-config-reviews/cmdb-r3-independent-native-code-20261111-r1/review-r1.json)，SHA `d35019a8275b2ce97194e632f172b3ff4f1eea9e7dd38555cc3f6c1696e4bb65`
- [独立 ARCH 核心回执](../../../../../.local/post-config-reviews/cmdb-r3-independent-arch-20261111-r1/review-r1.json)，SHA `752b3d601cacbea00b75d2e289d760aca429ef5a031184ddce77645a7c329c5e`

## O2 权限产品修复正在实施

design-r3 已获得 CODE design APPROVE + ARCH design APPROVE/CLEAR，仅批准设计与分配的隔离实现。Base Generator 负责唯一 registered-literal classifier、shared PolicyCoversRoute、StaticRouteMatcherFor、公共 guard、role sync/workbench adapter 和真实角色增授/撤权/同批替换/事务回滚测试；Ops Generator 负责16keys/27routes provider、init-first 和六个静态 GET guard。

Ops 隔离 draft 已写完 11 个 owned 文件，原生 AST/sourcecontract preimage RED27 → draft GREEN0，9 Go 文件 parse/gofmt 通过；完整 native compile/tests 待正常新的 Base candidate 装配，不用手抄共享代码或 Go overlay。Base 正在实现和准备 native tests，Root将仅子进程注入已有受保护测试fixture，不向代理展示凭证。

仅六 guarded routes 的运行与 coverage 等价在此范围；CMDB 动态/manual policy 兼容缺口、未 seed UI rollback key、credential metadata 治理边界继续显式保留。设计不证明 runtime、完整 P0、发行或 UI 完成。

- [CODE 设计核心回执](../../../../../.local/post-config-reviews/o2-design-r3-independent-code-20261111-r1/review-r1.json)，SHA `aff562a0e9bbcb20fc9874273d6354726bd5a9e07c7e3b2ea7ba20cde098131a`
- [ARCH 设计核心回执](../../../../../.local/post-config-reviews/o2-design-r3-independent-arch-20261111-r1/review-r1.json)，SHA `5157d88756c9317416d6968b2206b2770b880df723f93c686e2c71852ac75515`

## O4 启动探针与 D2

O4 complete-positive r1 独立 CODE 发现三项真实 runner 缺陷：一次执行 claim 仅绑 activation 而非 frozen successor；build/run 日志无限；超时/启动失败丢日志或结构化证据。Root明确每个 frozen successor 只可执行一次，r1不能运行；作者正在创建独立 r2、保留 failed-before witnesses，补有界采集/结构化失败和固定 freeze claim。原生完整启动 Go oracle/300 路由/54 表的静态核对不能升级为 runtime，通过六严格负向与两完整正向的同候选实际运行仍未完成。

D2 已启动独立 Generator 收口现有公共 adapter：已有严格 TS 与27纯 oracle 通过，正补独立 reviewer 身份、durable owned journal、原生审计 metadata/actor/tenant/status 和 admin provisioner 断言及新冻结回执。未执行 HTTP/SQL/SSH，不声明完整 D2。

## 预览、停止条件与下一步

人工预览5173/PID22848、8080/PID31184与专用DB保持；公共验收18127/PID28876也保持。全量 Base/Ops Go gate共用Redis必须串行；公共admin登录由Root串行。

下一步：定位CMDB撤权问题；完成O2源码双审/正常候选装配/新的隔离runtime；完成严格启动与公共O2/O3/O4/D2真实组合；最终稳定源码全量门禁、正式Base发行消费及独立整体/视觉功能验收。Velero留到这些之后，不豁免D1或最终P0。
