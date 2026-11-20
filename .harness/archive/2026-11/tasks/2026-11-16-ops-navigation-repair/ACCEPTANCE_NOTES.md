# Codex 独立复现与验收范围

2026-11-16：真实本机旧数据库，后端8080/前端5173，正常UI登录，新独立Playwright context，不复用用户浏览器凭证。

## 已复现

- 服务目录两个root：旧id159 `/business/service` → 页面404；新id1818 `/business/service-catalog`。
- 子项1819/1820/1821点击改变URL，但页面内容一直为“新增应用/暂无应用”。`ServiceCatalog.tsx`使用`Tabs defaultActiveTab={activeTab}`而不是受控activeTab，故mounted组件只读初始值。页内Tabs也需要同步URL。
- 文件分发id1039 sort50，ArgoCD id1409 sort60，在系统菜单后；ArgoCD标签当前显示“应用发布”，与K8s child同名但没有清楚的GitOps含义。
- 原始证据：`.harness/evidence/2026-11-16-ops-navigation-repair/before/tree.json`、`probe.json`和`navigation-desktop.png`/`catalog-*.png`。

## 全量验收要求

1. 升级真实旧树：重复服务目录收敛为一个；保留旧id和角色绑定，重复seed不会再生；只处理业务拥有的旧菜单。
2. 服务目录父、应用/服务/实例侧栏入口与页内tab全部实际点击；正确URL、activeTab与内容三者一致；浏览器前进后退/刷新有效。
3. 所有业务父/叶菜单按实际旧树逐项点击，404/无反应/错误页面/加载异常明确记录。全量菜单顺序、父子关系、component注册、权限与旧数据迁移一起核查。
4. 文件分发跟随安装部署，ArgoCD紧邻K8s且位于系统菜单之前；business seed与frontend菜单排序同步，旧记录更新生效。
5. Desktop1440x900/mobile390x844实际截图、页面不溢出；服务目录无重复；后台权限不因修复被放宽。
6. 这是菜单用户流程修复的验收，不自动关闭SSH、历史工作台或C/E其它业务包。

执行者：Claude。Codex只写派发/复现/独立验收产物，不直接改业务代码。

## 全量复现补充

- 第一轮已逐项检查当前旧树的31个业务父/叶入口。结果在`before-all/all-business-clicks.json`；这轮按固定650ms等待，父项的旧工作台或短暂空白需用稳定内容等待复核，不直接定性产品缺陷。
- Kubernetes工作负载页显示`business.k8s.workload.*`裸key，业务i18n需要一起修复/验收；ArgoCD与K8s的应用发布同名需明确区分GitOps语义。
- K8s应用发布第一轮遇到验收locator同名，属于脚本定位问题，Codex已调整按菜单层级和可见性定位；不能将它算作产品点击失败。
- 修正验收脚本第二轮输出`before-confirmed/`，每项检查稳定非空内容/旧工作台残留/裸翻译key，后续`after/`使用同一核验规则。
- 第二轮稳定等待15秒后，服务目录父/K8s父/可观测父出现URL切到业务地址而正文仍为工作台；同路由子入口单独点击正常。请调查父项点击/页签缓存/业务父路由关系；如果是共享layout/tab机制问题，明确Base-first落点并提供具体补丁任务，不能在Ops复制修改共享层。`before-confirmed/all-business-clicks.json`保存实际URL和正文。

### 父菜单误报排除（优先于上一条）

`before-parent-stable/all-business-clicks.json`在每次进入工作台后先等待主内容就绪，再点击全部8个业务父/独立入口，全部正文正确且无滞留。上一条父入口现象属于验收脚本过早点击，不能据此修改Base或共享layout。Codex已同步向用户纠正。

### 追加交付要求

当前plan应补上工作负载业务裸key修复，及frontend business defineModule菜单排序与backend seed一致。ArgoCD展示标题应明确包含ArgoCD/GitOps，便于与K8s应用发布区分；仅将“应用发布”改为“应用交付”仍不够清晰。

frontend `ModuleMenuMeta`没有sort字段，不要扩展共享合同。业务`frontend/src/modules/business/index.ts`的模块注册顺序可以与业务菜单顺序同步，运行时排序仍由backend seed负责。

## 第一版实现的验收待处理项

- `mergeLegacyServiceMenu`两条Scan不应忽略数据库错误，否则可能在读失败时继续seed；应传播错误并让transaction回滚。
- 删除canonical父菜单前要保全其全部child引用：只重挂`module=business.service`会把用户自定义child留在已删除父节点下。应以最小迁移维护child可达性（不改child权限/业务内容），并测试自定义children不会悬空。
- 合法task packet需按本仓库模板/manifest schema校验；当前manifest的`business/navigation`、custom模板字符串/coverage维度等与门禁枚举不符，taskDoc不能指向缺少必填章节的ACCEPTANCE_NOTES。不能通过豁免新task逃过门禁。
- 初版`seed_migrate_test.go:insertLegacyMenu`的INSERT占位符与参数不一致（8个占位符，9个参数，module重复传入）。必须跑真实MySQL迁移回归，不能接受环境变量未配置导致skip的通过结果。Codex用现有testmysql helper创建/清理隔离测试库，不向当前业务库写fixture。

## 本机数据库恢复记录

首次真实迁移回归复现SQL参数错误。其后原33316 MySQL停止监听，8080健康变为degraded/database down；与3306系统MySQL无关，3306原凭证核对失败，没有修改业务连接配置。

Codex找到原`.tmp/ops-planned-cmdb-services/mysql-data`及现有`ops_manual_preview_20261111_v1`数据、原MySQL8.0.36二进制。原目录已有部分system目录/旧测试tablespace缺失（当前任务未删除）；通过补空的mysql/performance_schema/sys目录并使用`--upgrade=NONE`、原端口33316、原data目录启动恢复，不初始化/清空/切换数据库。19:21健康恢复HTTP200/status ok；其它历史测试库缺失警告记录在`.local/manual-preview-20261116/mysql-*.stderr.log`，不掩盖，也不将其定性为当前菜单修复代码导致。

## Codex独立门禁进度

- 19:21 `.local/manual-preview-20261116/check-go.ps1`：真实隔离MySQL、CGO_ENABLED=1、mingw64，`go test -race -v ./modules/business/service/...`全部15个用例PASS，无skip。`go-race-service.log`保留完整结果。执行者不需要读取用户设置/凭证来再次寻找DSN。
- menu-contract PASS（45菜单/55路由/250权限/54组件）、business-overlay PASS、business-boundaries PASS。frontend lint/type-check由Codex并行核验中。
- 待实现内容：工作负载裸key、明确ArgoCD名称、frontend模块注册顺序、合法task packet、完整UI导航回归（页内/侧栏/历史/刷新/全部叶入口）；截图与旧DB重启由Codex负责。

## Build门禁返修（19:36）

Codex `npm run build`在prebuild的smoke-coverage-contract失败：新增`frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts`未接入smoke命令，且未登记`frontend/tests/smoke/README.md`矩阵。请同步修改业务smoke脚本（复用现有test:smoke:business:service或新增明确business导航脚本）与README/overlay声明，保持rebuild后能执行同一test，不绕过门禁。原始输出`codex-frontend-build.log`。
