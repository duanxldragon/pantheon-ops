# 非 Velero 优先收口检查点

维护者要求：Velero 放最后，先完成其他 P0。O2/O3/O4/D1/D2/D3 父任务均开放；本节覆盖更早的可用性与整体通过描述，旧回执、失败和已消费请求保留。

## 已验证

- Base 生命周期两文件 CODE APPROVE / ARCH CLEAR、12 项 targeted CGO race通过；新完整 CGO race50测试包/2197测试子测试通过，0失败/race，1282.844秒。唯一实际skip为未配置端点的真实S3单项，保留显式runtime gap及旧真实S3局部证据。raw534多含一个Go忽略的既有`.tmp`SQL备份，normalized533精确匹配新candidate全部backend Go/SQL/mod/sum，前后稳定、无遗漏，旧533路径集仅两个批准hash变化。Root独立复核实际JSONL与pin通过。
- 新Base candidate74已正常native producer一次生成，1166当前源文件/批准map/descriptor/整个archive匹配；不是正式release，尚未正常Ops消费。
- 前一包装candidate6a正常首次Ops装配1290.515秒通过，Base1166/Ops5428/assembled6580完整读回，主选择输入前后稳定。drift808identical/0allowlisted、12迁移、19生成文件。其Base生命周期仍为前一版本，不能冒称已消费新74。
- 6a完整repo同对象短Junction解决Node长路径问题，无管理员/包锁源码修改。标准npm ci、lint、type-check、build、unit全部通过，53files/402tests/0skipfail；6580source/package-lock稳定。最终candidate将合并批准权限修复再装配，避免对即将变化的产品重复全量测试。
- R6行政会话撤销实际40requests/13asserts通过：旧session HTTP200/envelope401/no data、新session controls200，owned tenant8/user20/role28精确清理完成；不算重跑R4完整保存视图矩阵。

## 正在收口

- O2 R2第9请求role-create HTTP200/envelope400，仅tenant9创建，无远端动作；consumed不重放/复用/清tenant9。原生种子不含五个Deploy键package:list、task:list、credential:list/create/update，task:detail和bizscope:list确有种子。另有Deploy/BizScope API provider缺口。新设计复用Base共享扩展点、Ops既有keys；双评审发现`:id`还覆盖list/options/export，且role sync/workbench覆盖判断与新static guard不一致，正在统一最小方案。没有raw policy/admin业务正向替代。
- CMDB r2审计oracle已修复并双审：普通IAM实际数字URI、敏感键保留`***`、list/detail精确事件；26checks通过、原r1同check17RED、strictTS0。Rootfreshrequest和nativecheck后一次actual run44requests/26asserts停止于host正向控制。Root另两次只读HTTP证隔离库目标115.190.251.143记录为空。roles31–34/users22–25全部保留，未hostmutation/Unknown，不能擅自清理或重放。新r3在firstIAMpost前核RootexacthostID/snapshot/receipt，21hostshape+26retainedaudit检查和strictTS通过，正在新双审；单独Root native local-metadata fixture准备不算业务非admin正向覆盖。
- 旧启动8case范围纠正：pool2/0旧GREEN仅证明截断9表fixture退休清理已发生，实际MySQL panic/169routes；不能声明完整启动通过。ARCH认可6严格拒绝+2合法容量完整正向：前6项保留固定错误/9表schema无效果/routes0；后2项需完整native迁移、system/auth/providers准备，再调用真实standardbusinessentry正常返回/samepool容量不变/route-readiness和exactsentinel检查。不能合取互斥noEffects/cleanup。完整正向source/fixture正在准备。

## 证据哈希

下列原始证据在对应仓库忽略目录，主阅读面保留hash与范围；存在不等于审批。

| 证据                 | 仓库相对路径                                                                                                           | SHA256                                                             |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| Base full race       | `.local/o4-preflight-inheritance-repair-20261111/base-final-full-race-99e409429ce8/receipt.json`                       | `fe3c810664287f477d0d471ed67f96aafb9826bdb9fdd79b09987d0fbb8c12e5` |
| Base candidate seal  | 同目录`candidate-bound-full-race-seal.json`                                                                            | `2ace766b60d079da35df475942733aca1bdbb4cf07fe59fe2df9114ad5ded5cf` |
| Root full race验证   | 同目录`root-independent-full-race-verification.json`                                                                   | `55351267ba08017891dc68c47e1f8f2c6269e80691e2df5e349645015255ede6` |
| Base新producer       | `.local/o4-preflight-inheritance-repair-20261111/candidate-successor-20261111-r1/root-produced-candidate-receipt.json` | `a823ee52492cac294687ff82c96ca2ff5e7ff4fa7e2871a11d0595e63250cd68` |
| Base正式1192输入准备 | `.local/o4-formal-release-prepared-20261111/root-packaging-map-successor-r1-receipt.json`                              | `cebe838eb2656e40d5e810d93078eea16337c55fcc880db3c02bd0e1f8ae9a0b` |
| Ops 6a前端五门禁     | `.local/ops-acceptance/o4-fullassembly-20261111/actual-initial-6a-frontend-gates-summary-r10.json`                     | `973fcc76802033aa3f4d5b46e90cca5d2b63ebcecbfa88446effc28c5dd77369` |
| CMDB r2实际失败      | `.local/ops-acceptance/cmdb-public-key-only-adapter-20261111-r2/result.json`                                           | `ea4b5344fb2eabda47b85efb20ef49f3ffd9cf9f05179683b6c7312a50468687` |
| CMDB只读定位         | 同目录`root-readonly-host-diagnosis.json`                                                                              | `67a66fc1791f8f504b4d788521a9d59d3e96dbb1bdd4aa39a279ece4514c0bf9` |
| 启动判据独审         | `.local/post-config-reviews/o4-standard-entry-contract-arch-20261111-r1/review-r1.json`                                | `8da41745fbf2e62049b91bed22bedbffc4d7fd32aa582dff5c3a4e31b1d824bf` |

## 运行与剩余门禁

人工前端5173/PID22848、后端8080/PID31184及预览DB保留；验收18127/PID28876保留。三HTTP入口均200。预览DB不用于验收/清理。Ops主docs/.harness冻结在6a正常seal后解除；所有runningstage保持不可变。

剩余：CMDB公共grant/revoke/CSV/DataScope，O2真实非admin业务、O3/D2公共产品组合，完整启动正向，最终源码Ops CGO race与相关原生门禁，正式Base发行→四资产正式消费/兼容重建、独立整体评审和最终视觉功能验收。正式发布、基线替换和生产操作未执行；Velero仍后置，父任务无新增关闭。
