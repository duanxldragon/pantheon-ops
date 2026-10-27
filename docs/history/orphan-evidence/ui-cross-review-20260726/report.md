# UI 交叉审查报告（frontend-design × impeccable）

日期：2026-07-26 ｜ 范围：pantheon-base + pantheon-ops ｜ 性质：纯审查，未改任何代码

> **第二轮更新（同日晚，隔离实例已搭建）**：base 新鲜渲染证据已补齐——`fresh/` 目录 44+
> 截图（暗色全页 ×20、violet/slate/emerald 抽查、移动端 ×7、S4666 特写、交互/空态）+
> `probes.json` 计算样式取证；`full-page-audit.spec.ts` 20/20 全绿（零 console error、
> 零横向溢出、壳层布局守卫通过）。本文 findings 已按新证据逐条更新状态（标注【第二轮】）。
> 初轮渲染结论基于 07-16~07-23 档案截图。

## TL;DR

base 在"企业后台"品类里一致性和完成度高于平均：视觉定位（冷静/可信/工具化）成立且执行自洽，
token 纪律、交互态、响应式、对比度（AA 双模）均达标；无新 P0。主要问题集中在三类：
①文案层——hero 描述系统性 changelog 腔（内部重构叙事泄漏给用户）；②表格列宽策略缺失导致
1440 宽下多页关键列截断；③"契约三方不同步"——DESIGN.md §7.8、index.css 实现、检查器锚点
各说各话（S4666 是其中一例，实际受影响面更大）。ops 侧全部差异归入既定 base-sync 决策，
但同步范围清单需补五项（见跨仓节）。

## 一、双镜头总评

### frontend-design（美学/辨识度）

- **定位执行**：§7.5"冷静、可信、工具化"贯彻到位——中性表面+弱边框+克制圆角，语义色只用于
  状态；深色侧栏 + 四主题制是主要辨识度来源。
- **signature 元素**：hero 三段式（eyebrow 域徽标 + 议题式标题 + KPI 卡组 + 摘要按钮）贯穿全部
  系统页，是有意图的非模板结构，成立。但所有 hero 标题共用"围绕…完成统一…治理"句式，
  同构疲劳——每页第一行读起来一样，削弱页面身份。建议标题名词化，描述句留给副文案。
- **字体**：system-ui 栈是刻意的"零个性"选择（离线部署可控性优先，契约如此），`tnum` 表格
  等宽是加分细节；个性由结构与色彩承担，取舍正确。
- **动效**：200ms token 化 transition + hover translateY(-1px)，克制恰当。
- **文案**：治理术语密度对 ops 工程师受众可接受；changelog 腔 hero 与"欢迎回来"审计标题
  是明确扣分项（见 P1-1/P2-2）。
- **结论**：风格自洽、完成度高、辨识度中等（有 signature、无 memorable moment）——
  与工具化 brief 匹配，不建议为"独特性"加装饰。

### impeccable（质量门 Quality Bar）

| 检查项 | 结果（含第二轮更新） |
|---|---|
| 文本重叠/裁切 | **命中**：列截断新实证（角色列无省略号截断 ×3 主题、抽屉内断词换行）；安全中心卡裁切已排除 |
| 控件 hover/loading 改变布局尺寸 | 未发现（transition 仅 border/shadow/transform；hover/focus 特写已归档） |
| 主操作视觉清晰 | 桌面达标；移动端"新增"沉底（P1-4） |
| 空/加载/错误态 | 【更新】空态有设计（虚线框+图标+文案，security-event/筛选空态实证）；设置页"运行时风险"仍纯文本；error/permission 态未触发（gap） |
| icon/圆角/阴影/间距随机感 | 大体系统化；分页 9px/10px 魔法圆角、mix 比例三种值（P2） |
| 对比度/焦点态 | base `check:contrast` AA 双模通过；键盘焦点环实拍确认。ops 无对比度门禁 |
| 移动端横滚 | 表格为有意横滑+提示（合规模式）；头部头像换行为壳层级缺陷（P1-4，user/role 双页实证） |
| 【第二轮新增】暗色模式 | **通过**：20 页暗色全测，登录/工作台/列表/表单/会话五类表面无漏白、无不可读文本 |
| 【第二轮新增】四主题 | **通过**：indigo/emerald（档案）+ violet/slate（新摄）品牌色/语义色/侧栏均一致换肤 |
| 【第二轮新增】机械走查 | full-page-audit 20/20 绿；仅 role/menu 弹窗高 868px 逼近视口、permission 页"新增"按钮未找到（页面改为 tab 结构，走查 spec 需跟进） |

机械门禁：base 五项（ui-contract/shell-visual/search-toolbar/contrast/i18n-hardcode）全绿；
ops 三项全绿但少两个门禁且 ui-contract 带 2 文件豁免。

## 二、Findings

### P1（V1 前应修）

1. ~~**hero 描述系统性 changelog 腔**~~ →【第二轮：已关闭】`.system-page-hero__desc`
   元素在当前代码已整体移除（探针 null + 新截图 role/menu/operation-log 均无该段落），
   07-17 档案时代的问题已在后续迭代修复。
2. **表格关键列截断（列宽策略缺失）**——【第二轮：维持并有新实证】新库空表无法复现
   security-event/operation-log 大表截断（档案证据保留），但新截图新增两处实证：
   用户列表"角色"列 `系统管理员`→`系统管理` 无省略号硬截断（light/dark/emerald 三张
   均现）；角色成员抽屉内昵称 `Administrator` 断词换行。需统一 min-width/省略规则。
3. **会话管理信息设计**——【第二轮：部分改善、核心维持】表格已重构新增"最近活动/状态"
   列（好）；但"当前账号"标签仍每行重复（21 会话全同账号，纯噪音），hero 承诺"设备画像"
   仍无设备列。另暗色下"已下线"用红调标签表达中性历史状态，语义偏差（记 P2-13）。
4. **移动端（390）**——【第二轮：确认为壳层级问题】user/role 两页新截图一致：头部第三行
   孤立头像（flex 换行事故）；工具栏 4+ 全宽按钮纵向堆叠、主操作"新增"排最后。
5. ~~**安全中心右栏"账号安全概览"卡裁切**~~ →【第二轮：已关闭】clip 探针对右栏全部卡片
   scrollH==clientH，无裁切；7-18 档案疑点系数据态/滚动截图假象。
6. **DESIGN.md §7.8 与实现系统性错位**：圆角文档 sm6/md8/lg12 vs 实现 sm4/md6/lg8；
   间距文档 md16/lg24/xl32 vs 实现 --space-md12/lg16/xl24。实现跨仓自洽（base=ops），
   漂移在文档侧 → 文档重锚（与 S4666 同一决策类）。实现另有 xl/overlay/control/action
   四个未记载 token。
7. **S4666 级联漂移面扩大**——【第二轮：计算样式实证到手，见 §四】
   - `.filter-panel`（=SearchToolbar 卡）实际渲染：**纯白背景 rgb(255,255,255)** +
     `--panel-shadow-soft` + 6px 圆角——连 52% mix 都被更晚的 `.search-toolbar` 规则
     覆盖成 panel-bg-solid；检查器锚定的 82%/none 与现实差两代。
   - `.app-table .arco-table-container` 计算圆角 **0px**（锚定 radius-md）。
   - 弹窗体 `srgb(0.976,0.979,0.985)`（78% muted）vs 抽屉体 `rgb(255,255,255)`——
     两种覆盖层表面语义不一致，计算值实证。
   处置：维持"重锚检查器→修 CSS"既定决策，重锚范围按本清单扩大。
8. **operation-log 筛选弹层半透明穿透**（7-16 档案）——【第二轮：降级为待验证 P2】7-19
   起 toolbar 已改版为 SearchToolbar popover，新实例空表未触发弹层复核；留作补测项。

### P2（打磨）

1. 危险确认弹窗按钮文案"确定"→ 应复述动作（"清理"）；动作按钮全流程同名原则。
2. "欢迎回来"作为 /auth/login 的操作日志标题（toast 文案进审计数据）。
3. 安全中心第四张 KPI 卡标题"成功"语义不明（成功什么？）。
4. 筛选占位符风格不一：i18n 页"所属模块/语言"（裸名词）vs"请选择分组"（请选择前缀）。
5. 工具栏按钮排序跨页不一致（user: 导出/下载模板/导入+新增 vs i18n: 刷新/导出/导入/更多+新增）。
6. 行内操作 icon 不一致（role 页"角色成员"无 icon，编辑/删除有）。
7. dept 行内"组织根节点"标签同行出现两次（名称列 + 治理视角列）。
8. menu 页路由名称列"dashboard dashboard"两行重复渲染（疑双字段渲染）。
9. 死 token：--brand-gradient、--shell-brand-shadow（4 主题均定义、零消费）；ops 另有
   --login-glow/--login-metric-*。
10. 魔法值：分页控件 border-radius 9px/10px（不在 token 尺度）；dashboard.css 字面量
    8px/6px 圆角与 rgba 阴影。
11. 次级表面 mix 比例三种（52/78/82%）无 token 语义——建议收敛为 --surface-muted-* 阶梯。
12. 空态纯文本样式（如系统设置"当前未发现配置风险"大面积空白）→ 统一 page-empty 模式。

### 已修复确认（无需行动）

- 清理交互已统一：7-16 operation-log 行内双 select 清理条 → 7-19 已收敛（login-log 弹窗模式）。
- 登录日志 gorm default:1 吞失败状态（07-23 P0）：7-18 档案可见失败分类正常展示。

## 三、base ↔ ops 跨仓一致性

**【第二轮 ops 渲染实证（fresh-ops/ 26 文件 + probes-ops.json）】**：

- **系统页视觉与 base 高度一致**（工作台/用户/角色等布局同构）——foundation 继承有效，
  跨仓风格一致性在系统域成立。业务页（CMDB/部署）延续同一语言（运维平台 eyebrow + KPI hero）。
- **三项静态推断运行时实锤**：`--color-fill-bg*` 运行时为空（但现行工作台 DOM 已不引用
  这些类，**视觉无损**——坏规则属于旧版布局的死 CSS，降低紧迫性）；user 页
  `hasSearchToolbar:false` + 显式 搜索/重置 按钮（FilterPanel 在跑）；强设
  `data-color-mode=dark` 后 `--app-bg` 仍为亮色 `#f7f8fa`（无暗色 token）。
- **错误态证据（gap 关闭）**：cmdb 主机页空库下 list 失败——错误块设计良好（图标+标题+
  说明+重试按钮），但**单次加载弹出 5 个重复失败 toast 无去重**（ops 新 P2）。
- **ops 残留 base 已修问题**：hero 仍带 changelog 腔描述段（base 已整体移除）；
  侧栏品牌名显示 "Pantheon Base"（ops 种子 siteName 未覆盖为自身身份，新 P2）；
  用户列表角色列显示 "role." 疑似截断/未翻译（待 base-sync 复核）。

以下为静态审计结论（维持）：

全部按既定"ops 等 base 发行版后 base-sync"决策处理，不单独开修复任务；
但 **base-sync 验收清单需显式包含以下五项**：

1. **ops 无暗色模式**：全库无 `data-color-mode` 定义——同平台一明一暗断裂（最大体验差异）。
2. **ops 全线 FilterPanel**（15+ 文件含 business/deploy、cmdb）：SearchToolbar 未落地，
   §7.2 契约在 ops 不成立；business 页面迁移工作量应计入 sync 计划。
3. **ops dashboard.css 引用未定义变量**（--color-fill-bg 系列，Arco 也无此组）：工作台渐变
   declaration 整条失效、部分边框回退 currentColor。已在 ops ui-contract DRIFT_PENDING_SYNC
   豁免记录（连同 list-page.css），符合治理流程，但渲染影响此前未记录。
4. **ops token 层落后**：硬编码 #fff mix（base 已演进 --surface-lift）；缺 --space/--font-size/
   --motion/--z 阶梯；缺 --radius-xl。
5. **门禁与文档滞后**：ops 缺 check:contrast、check:search-toolbar-contract；
   ops CLAUDE.md 设计节仍写"Source Sans 3 为正文字体"，与 base DESIGN.md §7.6
   （system-ui 栈，SS3 可选增强）冲突。

## 四、S4666 重锚判定材料

**第二轮计算样式实测（probes.json，/system/user 实页）**：

| 选择器 | 检查器锚定 | 计算样式实测 |
|---|---|---|
| .filter-panel background | 82% muted mix | **rgb(255,255,255) 纯白**——比静态推定的 52% mix 更进一层，`.search-toolbar` 块最终覆盖为 panel-bg-solid |
| .filter-panel box-shadow | none | **panel-shadow-soft 在渲**（0 1px 2px + 0 8px 18px rgba(15,23,42,…)） |
| .filter-panel border-radius | — | 6px（=实现侧 radius-md） |
| .app-table .arco-table-container radius | var(--radius-md) | **0px**（外层 page-panel 6px 承担轮廓） |
| .app-dialog .arco-modal-content bg | — | srgb(0.976,0.979,0.985) ≈ 78% muted ✓ |
| .app-drawer .arco-drawer-content bg | （初始意图 82% muted） | **rgb(255,255,255)**——与弹窗体不一致，重锚时须一并决策 |

特写截图：`fresh/s4666-filter-panel-closeup.png`、`fresh/s4666-app-table-closeup.png`。
**判定：现渲染贯穿 7-16 至今全部截图并经多轮视觉验收——确认支持"重锚检查器到现渲染值"。**
注意 filter-panel 背景应锚 panel-bg-solid（纯白）而非 S4666 预研推定的 52% mix。

以下为初轮静态级联推定（保留供重锚对照）：

| 选择器 | 检查器锚定（早块） | 实际生效（晚块） | 行号 |
|---|---|---|---|
| .filter-panel background | 82% muted mix | 52% muted mix | 1741 → 2703 |
| .filter-panel box-shadow | none | var(--panel-shadow-soft) | 1742 → 2698 |
| .filter-panel border-color | 92% mix | 94% mix | 1740 → 2704 |
| .app-table .arco-table-container radius | var(--radius-md) | 0 | 2757 → 2817 |
| .app-table .arco-table-header radius | （同上族） | 0 | 2821 |
| .app-drawer .arco-drawer-content bg | 82% muted mix | var(--panel-bg-solid) | 1401 → 1423 |

档案截图旁证：7-17~7-23 所有列表页截图中筛选工具栏均呈现"浅色近白 + 微投影"（即 52%+
shadow-soft 生效值）、表格容器为直角——**现渲染即长期以来用户看到并验收过的样子**。
初步判断支持"重锚检查器到现渲染值"，最终确认待新鲜特写截图（gap #6）。

## 五、Completion Evidence（impeccable 协议，第二轮更新后）

- **Surfaces reviewed（20+）**：登录、工作台、用户+新增弹窗、用户详情、角色+成员抽屉、
  菜单、权限、部门、岗位、字典、系统设置、国际化、模块注册表、模块生成器、登录日志、
  操作日志、会话管理、安全事件、安全中心、个人中心、命令面板、清理危险弹窗。
- **Viewports**：1440×900（暗色 20 页新摄 + 明色档案全集 + audit sweep）；390×844
  移动（login/dashboard/user/role/session/setting 新摄 6 页 + 暗色 user）。
- **States（新摄实证）**：主按钮 hover/focus 特写、筛选空态（含清空筛选入口）、
  security-event 设计空态、新增弹窗、角色成员抽屉、禁用批量按钮、当前会话高亮。
- **Themes**：四主题齐——indigo 明/暗全套、emerald（档案+新摄 user）、violet/slate
  （login/dashboard/user 新摄）。
- **Known Gaps（收敛后）**：
  1. loading / error / permission-denied 态未触发（隔离实例无权限受限账号、后端无故障注入）；
  2. 大数据量下的表格列截断（operation-log 时间列、security-event 确认列）依赖档案证据，
     新库空表未复现；
  3. operation-log 筛选弹层穿透（7-16 档案）未在新版 SearchToolbar popover 上复核；
  4. ops 渲染证据见 fresh-ops/（若该目录缺失=ops 采集仍被会话工具故障阻塞，静态结论已完备）。

## 六、剩余步骤与 Teardown（第二轮更新）

**当前实例状态**：base 隔离实例运行中——后端 18080（$TEMP/pantheon-base-ui-test.exe，
库 pantheon_ui_test）+ vite 15173；ops 后端已构建（$TEMP/pantheon-ops-ui-test.exe）
未启动。base 证据已采齐。

**仅剩一步（ops 采集，一条命令）**：
```bash
node "D:/workspace/go/pantheon-platform/pantheon-ops/frontend/.tmp-capture/start-ops-instance.mjs" \
  && sleep 12 && cd "D:/workspace/go/pantheon-platform/pantheon-ops/frontend" \
  && npx playwright test --config .tmp-capture/capture.config.ts
```
产出：`fresh-ops/` 22 页桌面 + 3 移动截图 + probes-ops.json（dashboard 未定义变量取证、
FilterPanel vs SearchToolbar 探针、暗色可用性探针）。

**Teardown（全部完成后）**：
```bash
# 端口找 PID：netstat -ano | grep -E ":(18080|15173|28080|25173)" ；taskkill //PID <pid> //F
# ops PID 也记录在 %TEMP%/pantheon-ops-ui-test.pids.json
mysql -h 127.0.0.1 -u root -p'<redacted-mysql-password>' -e "DROP DATABASE pantheon_ui_test; DROP DATABASE pantheon_ops_ui_test"
rm "$TEMP/pantheon-base-ui-test.exe" "$TEMP/pantheon-ops-ui-test.exe"
rm -rf pantheon-base/frontend/.tmp-capture pantheon-ops/frontend/.tmp-capture
# 复核：8080/5173 用户实例仍在，pantheon/pantheon_base/pantheon_ops 库未动
```

## 附：初轮补测清单（已执行，留档）

```bash
# 1. 建库
mysql -h 127.0.0.1 -u root -p'<redacted-mysql-password>' -e "CREATE DATABASE IF NOT EXISTS pantheon_ui_test CHARACTER SET utf8mb4"
# 2. base 后端（不设 GIN_MODE=release）
cd pantheon-base/backend && go build -o /tmp/pantheon-ui-test.exe ./cmd/server
PANTHEON_DSN='root:<redacted-mysql-password>@tcp(127.0.0.1:3306)/pantheon_ui_test?charset=utf8mb4&parseTime=True&loc=Local' \
PANTHEON_PORT=18080 PANTHEON_REDIS_ADDR=127.0.0.1:6379 PANTHEON_REDIS_PASSWORD=<redacted-redis-password> /tmp/pantheon-ui-test.exe &
# 3. base 前端
cd pantheon-base/frontend && PANTHEON_API_PROXY_TARGET=http://127.0.0.1:18080 npx vite --port 15173 --strictPort &
# 4. 全页走查（18 页 + 布局守卫 + findings.json）
PANTHEON_EXTERNAL_WEB_SERVER=1 PANTHEON_WEB_BASE_URL=http://127.0.0.1:15173 \
PANTHEON_API_BASE_URL=http://127.0.0.1:15173/api/v1 \
npx playwright test tests/smoke/platform/full-page-audit.spec.ts --config playwright.config.ts --workers=1
# 5. 暗色/主题矩阵：走查脚本内 addInitScript 设 localStorage
#    pantheon_color_mode=dark / pantheon_theme=emerald|violet|slate
# 6. ops 镜像同法：pantheon_ops_ui_test / 28080 / 25173（env 名同 PANTHEON_*，见 ops README:75-77）
# 7. Teardown：taskkill 进程、DROP DATABASE、删 /tmp exe，netstat 复核 8080/5173 未动
```
