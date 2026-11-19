# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\ServiceCatalog.nav.spec.ts >> Business navigation repair >> every business parent/leaf entry loads the expected main content
- Location: tests\smoke\business\service\ServiceCatalog.nav.spec.ts:237:3

# Error details

```
Error: Failed navigating to /business/deploy/task: expect(locator).toBeVisible() failed

Locator: locator('main').getByRole('heading', { name: '部署任务', exact: true })
Expected: visible
Timeout: 15000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('main').getByRole('heading', { name: '部署任务', exact: true }) with timeout 15000ms
  - waiting for locator('main').getByRole('heading', { name: '部署任务', exact: true })

```

# Page snapshot

```yaml
- generic [ref=f16e3]:
    - complementary [ref=f16e4]:
        - generic [ref=f16e5]:
            - generic [ref=f16e6]:
                - generic [ref=f16e7]: P
                - generic [ref=f16e8]: Pantheon Base
            - menu [ref=f16e12]:
                - generic [ref=f16e13]:
                    - menuitem [ref=f16e14] [cursor=pointer]:
                        - button "工作台" [ref=f16e15]
                    - button "CMDB" [ref=f16e25] [cursor=pointer]
                    - button "业务域" [ref=f16e40] [cursor=pointer]
                    - button "服务目录" [ref=f16e55] [cursor=pointer]
                    - generic [ref=f16e67]:
                        - button "安装部署" [ref=f16e70] [cursor=pointer]
                        - generic [ref=f16e79]:
                            - menuitem [ref=f16e80] [cursor=pointer]:
                                - button "软件组件" [ref=f16e83]
                            - menuitem [ref=f16e89] [cursor=pointer]:
                                - button "任务模板" [ref=f16e92]
                            - menuitem [ref=f16e98] [cursor=pointer]:
                                - button "部署任务" [ref=f16e101]
                    - menuitem [ref=f16e107] [cursor=pointer]:
                        - button "文件分发" [ref=f16e108]
                    - button "Kubernetes" [ref=f16e120] [cursor=pointer]
                    - menuitem [ref=f16e129] [cursor=pointer]:
                        - button "ArgoCD 应用交付" [ref=f16e130]
                    - button "可观测中心" [ref=f16e139] [cursor=pointer]
                    - button "访问控制" [ref=f16e151] [cursor=pointer]
                    - button "组织架构" [ref=f16e165] [cursor=pointer]
                    - button "安全审计" [ref=f16e180] [cursor=pointer]
                    - button "低代码平台" [ref=f16e192] [cursor=pointer]
                    - button "平台配置" [ref=f16e204] [cursor=pointer]
    - generic [ref=f16e213]:
        - generic [ref=f16e214]:
            - generic [ref=f16e215]:
                - button "切换导航" [ref=f16e216] [cursor=pointer]
                - list [ref=f16e220]:
                    - listitem [ref=f16e221]: 首页
                    - listitem [ref=f16e225]: 安装部署
                    - listitem [ref=f16e229]: 部署任务
            - generic [ref=f16e230]:
                - button "平台功能台" [ref=f16e232]:
                    - generic [ref=f16e235]: 搜索菜单、页面、功能…
                    - generic [ref=f16e236]: Ctrl K
                - button "通知中心" [ref=f16e238] [cursor=pointer]
                - button "平台偏好" [ref=f16e242] [cursor=pointer]
                - 'button "用户: Administrator" [ref=f16e247] [cursor=pointer]':
                    - generic [ref=f16e248]: A
                    - generic [ref=f16e250]:
                        - generic [ref=f16e251]: Administrator
                        - generic [ref=f16e252]: admin
        - tablist "已打开页面" [ref=f16e253]:
            - tab "工作台" [ref=f16e254] [cursor=pointer]
            - tab "服务目录" [ref=f16e258] [cursor=pointer]
            - tab "应用" [ref=f16e260] [cursor=pointer]
            - tab "服务" [ref=f16e262] [cursor=pointer]
            - tab "服务实例" [ref=f16e264] [cursor=pointer]
            - tab "软件组件" [ref=f16e266] [cursor=pointer]
            - tab "任务模板" [ref=f16e268] [cursor=pointer]
            - tab "部署任务" [selected] [ref=f16e270] [cursor=pointer]
        - main [ref=f16e272]:
            - generic [ref=f16e275]:
                - generic [ref=f16e277]:
                    - generic [ref=f16e278]:
                        - generic [ref=f16e279]: 运维平台 / 部署任务
                        - generic [ref=f16e280]: 部署任务
                    - generic [ref=f16e285]:
                        - generic [ref=f16e286]:
                            - generic [ref=f16e287]: 任务总数
                            - strong [ref=f16e288]: "41"
                        - generic [ref=f16e289]:
                            - generic [ref=f16e290]: 执行中
                            - strong [ref=f16e291]: "0"
                        - generic [ref=f16e292]:
                            - generic [ref=f16e293]: 成功
                            - strong [ref=f16e294]: "2"
                    - button "任务治理视角" [ref=f16e296] [cursor=pointer]
                - search "筛选" [ref=f16e303]:
                    - generic [ref=f16e305]:
                        - textbox "关键字" [ref=f16e312]
                        - combobox [ref=f16e314] [cursor=pointer]:
                            - generic [ref=f16e316]:
                                - textbox "状态"
                                - generic [ref=f16e317]: 状态
                - generic [ref=f16e324]:
                    - generic [ref=f16e325]:
                        - generic [ref=f16e327]:
                            - button "导出" [ref=f16e330] [cursor=pointer]
                            - button "新增" [ref=f16e335] [cursor=pointer]
                        - generic [ref=f16e338]: 已选 0 条
                        - button "清空选择" [disabled] [ref=f16e339]
                    - generic [ref=f16e341]:
                        - button "取消" [disabled]
                - generic [ref=f16e348]:
                    - table [ref=f16e352]:
                        - rowgroup [ref=f16e364]:
                            - row "任务名称 任务来源 任务动作 目标来源 业务域 执行方式 状态 主机数 操作" [ref=f16e365]:
                                - columnheader [ref=f16e366]:
                                    - generic [ref=f16e368] [cursor=pointer]:
                                        - checkbox
                                - columnheader "任务名称" [ref=f16e371]
                                - columnheader "任务来源" [ref=f16e373]
                                - columnheader "任务动作" [ref=f16e375]
                                - columnheader "目标来源" [ref=f16e377]
                                - columnheader "业务域" [ref=f16e379]
                                - columnheader "执行方式" [ref=f16e381]
                                - columnheader "状态" [ref=f16e383]
                                - columnheader "主机数" [ref=f16e385]
                                - columnheader "操作" [ref=f16e387]
                        - rowgroup [ref=f16e389]:
                            - row "deploy-smoke-1794895811602 软件组件 deploy-smoke-1794895811602 1.0.0 安装 主机 deploy-smoke-1794895811602 手动执行 草稿 0 详情 编辑 启动 删除" [ref=f16e390]:
                                - cell [ref=f16e391]:
                                    - generic [ref=f16e392] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-smoke-1794895811602" [ref=f16e395]
                                - cell "软件组件 deploy-smoke-1794895811602 1.0.0" [ref=f16e397]:
                                    - generic [ref=f16e400]:
                                        - generic [ref=f16e401]: 软件组件
                                        - generic [ref=f16e402]: deploy-smoke-1794895811602 1.0.0
                                - cell "安装" [ref=f16e403]
                                - cell "主机" [ref=f16e405]
                                - cell "deploy-smoke-1794895811602" [ref=f16e407]
                                - cell "手动执行" [ref=f16e409]
                                - cell "草稿" [ref=f16e411]
                                - cell "0" [ref=f16e416]
                                - cell [ref=f16e418]:
                                    - generic [ref=f16e421]:
                                        - button "详情" [ref=f16e423] [cursor=pointer]
                                        - button "编辑" [ref=f16e428] [cursor=pointer]
                                        - button "启动" [ref=f16e432] [cursor=pointer]
                                        - button "删除" [ref=f16e436] [cursor=pointer]
                            - row "deploy-api-trace-1790590233509-task 软件组件 deploy-api-trace-1790590233509-nginx 1.30.2 安装 主机 deploy-api-trace-1790590233509-scope SSH 执行 失败 1 详情 删除" [ref=f16e439]:
                                - cell [ref=f16e440]:
                                    - generic [ref=f16e441] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-api-trace-1790590233509-task" [ref=f16e444]
                                - cell "软件组件 deploy-api-trace-1790590233509-nginx 1.30.2" [ref=f16e446]:
                                    - generic [ref=f16e449]:
                                        - generic [ref=f16e450]: 软件组件
                                        - generic [ref=f16e451]: deploy-api-trace-1790590233509-nginx 1.30.2
                                - cell "安装" [ref=f16e452]
                                - cell "主机" [ref=f16e454]
                                - cell "deploy-api-trace-1790590233509-scope" [ref=f16e456]
                                - cell "SSH 执行" [ref=f16e458]
                                - cell "失败" [ref=f16e460]
                                - cell "1" [ref=f16e465]
                                - cell [ref=f16e467]:
                                    - generic [ref=f16e470]:
                                        - button "详情" [ref=f16e472] [cursor=pointer]
                                        - button "删除" [ref=f16e477] [cursor=pointer]
                            - row "deploy-smoke-1790590232575 软件组件 deploy-smoke-1790590232575 1.0.0 安装 主机 deploy-smoke-1790590232575 手动执行 成功 1 详情 删除" [ref=f16e480]:
                                - cell [ref=f16e481]:
                                    - generic [ref=f16e482] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-smoke-1790590232575" [ref=f16e485]
                                - cell "软件组件 deploy-smoke-1790590232575 1.0.0" [ref=f16e487]:
                                    - generic [ref=f16e490]:
                                        - generic [ref=f16e491]: 软件组件
                                        - generic [ref=f16e492]: deploy-smoke-1790590232575 1.0.0
                                - cell "安装" [ref=f16e493]
                                - cell "主机" [ref=f16e495]
                                - cell "deploy-smoke-1790590232575" [ref=f16e497]
                                - cell "手动执行" [ref=f16e499]
                                - cell "成功" [ref=f16e501]
                                - cell "1" [ref=f16e506]
                                - cell [ref=f16e508]:
                                    - generic [ref=f16e511]:
                                        - button "详情" [ref=f16e513] [cursor=pointer]
                                        - button "删除" [ref=f16e518] [cursor=pointer]
                            - row "deploy-closed-loop-1790567959382-task 软件组件 deploy-closed-loop-1790567959382-nginx 1.30.2 安装 主机 deploy-closed-loop-1790567959382-his-dev SSH 执行 失败 1 详情 删除" [ref=f16e521]:
                                - cell [ref=f16e522]:
                                    - generic [ref=f16e523] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-closed-loop-1790567959382-task" [ref=f16e526]
                                - cell "软件组件 deploy-closed-loop-1790567959382-nginx 1.30.2" [ref=f16e528]:
                                    - generic [ref=f16e531]:
                                        - generic [ref=f16e532]: 软件组件
                                        - generic [ref=f16e533]: deploy-closed-loop-1790567959382-nginx 1.30.2
                                - cell "安装" [ref=f16e534]
                                - cell "主机" [ref=f16e536]
                                - cell "deploy-closed-loop-1790567959382-his-dev" [ref=f16e538]
                                - cell "SSH 执行" [ref=f16e540]
                                - cell "失败" [ref=f16e542]
                                - cell "1" [ref=f16e547]
                                - cell [ref=f16e549]:
                                    - generic [ref=f16e552]:
                                        - button "详情" [ref=f16e554] [cursor=pointer]
                                        - button "删除" [ref=f16e559] [cursor=pointer]
                            - row "deploy-detail-edit-1790567953853-task 软件组件 deploy-detail-edit-1790567953853 1.0.0 安装 主机 deploy-detail-edit-1790567953853-scope 手动执行 草稿 0 详情 编辑 启动 删除" [ref=f16e562]:
                                - cell [ref=f16e563]:
                                    - generic [ref=f16e564] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-detail-edit-1790567953853-task" [ref=f16e567]
                                - cell "软件组件 deploy-detail-edit-1790567953853 1.0.0" [ref=f16e569]:
                                    - generic [ref=f16e572]:
                                        - generic [ref=f16e573]: 软件组件
                                        - generic [ref=f16e574]: deploy-detail-edit-1790567953853 1.0.0
                                - cell "安装" [ref=f16e575]
                                - cell "主机" [ref=f16e577]
                                - cell "deploy-detail-edit-1790567953853-scope" [ref=f16e579]
                                - cell "手动执行" [ref=f16e581]
                                - cell "草稿" [ref=f16e583]
                                - cell "0" [ref=f16e588]
                                - cell [ref=f16e590]:
                                    - generic [ref=f16e593]:
                                        - button "详情" [ref=f16e595] [cursor=pointer]
                                        - button "编辑" [ref=f16e600] [cursor=pointer]
                                        - button "启动" [ref=f16e604] [cursor=pointer]
                                        - button "删除" [ref=f16e608] [cursor=pointer]
                            - row "deploy-script-template-1790567948672-task deploy-script-template-1790567948672 v1 deploy-script-template-1790567948672 1.30.2 安装 主机 deploy-script-template-1790567948672-scope 手动执行 草稿 0 详情 编辑 启动 删除" [ref=f16e611]:
                                - cell [ref=f16e612]:
                                    - generic [ref=f16e613] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-script-template-1790567948672-task" [ref=f16e616]
                                - cell "deploy-script-template-1790567948672 v1 deploy-script-template-1790567948672 1.30.2" [ref=f16e618]:
                                    - generic [ref=f16e621]:
                                        - generic [ref=f16e622]: deploy-script-template-1790567948672 v1
                                        - generic [ref=f16e623]: deploy-script-template-1790567948672 1.30.2
                                - cell "安装" [ref=f16e624]
                                - cell "主机" [ref=f16e626]
                                - cell "deploy-script-template-1790567948672-scope" [ref=f16e628]
                                - cell "手动执行" [ref=f16e630]
                                - cell "草稿" [ref=f16e632]
                                - cell "0" [ref=f16e637]
                                - cell [ref=f16e639]:
                                    - generic [ref=f16e642]:
                                        - button "详情" [ref=f16e644] [cursor=pointer]
                                        - button "编辑" [ref=f16e649] [cursor=pointer]
                                        - button "启动" [ref=f16e653] [cursor=pointer]
                                        - button "删除" [ref=f16e657] [cursor=pointer]
                            - row "deploy-mark-failed-1790567944257-task 软件组件 deploy-mark-failed-1790567944257 1.0.0 安装 主机 deploy-mark-failed-1790567944257 手动执行 失败 1 详情 删除" [ref=f16e660]:
                                - cell [ref=f16e661]:
                                    - generic [ref=f16e662] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-mark-failed-1790567944257-task" [ref=f16e665]
                                - cell "软件组件 deploy-mark-failed-1790567944257 1.0.0" [ref=f16e667]:
                                    - generic [ref=f16e670]:
                                        - generic [ref=f16e671]: 软件组件
                                        - generic [ref=f16e672]: deploy-mark-failed-1790567944257 1.0.0
                                - cell "安装" [ref=f16e673]
                                - cell "主机" [ref=f16e675]
                                - cell "deploy-mark-failed-1790567944257" [ref=f16e677]
                                - cell "手动执行" [ref=f16e679]
                                - cell "失败" [ref=f16e681]
                                - cell "1" [ref=f16e686]
                                - cell [ref=f16e688]:
                                    - generic [ref=f16e691]:
                                        - button "详情" [ref=f16e693] [cursor=pointer]
                                        - button "删除" [ref=f16e698] [cursor=pointer]
                            - row "deploy-detail-1790567941127 软件组件 deploy-detail-1790567941127 1.0.0 安装 主机 deploy-detail-1790567941127 手动执行 草稿 0 详情 编辑 启动 删除" [ref=f16e701]:
                                - cell [ref=f16e702]:
                                    - generic [ref=f16e703] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-detail-1790567941127" [ref=f16e706]
                                - cell "软件组件 deploy-detail-1790567941127 1.0.0" [ref=f16e708]:
                                    - generic [ref=f16e711]:
                                        - generic [ref=f16e712]: 软件组件
                                        - generic [ref=f16e713]: deploy-detail-1790567941127 1.0.0
                                - cell "安装" [ref=f16e714]
                                - cell "主机" [ref=f16e716]
                                - cell "deploy-detail-1790567941127" [ref=f16e718]
                                - cell "手动执行" [ref=f16e720]
                                - cell "草稿" [ref=f16e722]
                                - cell "0" [ref=f16e727]
                                - cell [ref=f16e729]:
                                    - generic [ref=f16e732]:
                                        - button "详情" [ref=f16e734] [cursor=pointer]
                                        - button "编辑" [ref=f16e739] [cursor=pointer]
                                        - button "启动" [ref=f16e743] [cursor=pointer]
                                        - button "删除" [ref=f16e747] [cursor=pointer]
                            - row "deploy-detail-modal-1790567929573-task 软件组件 deploy-detail-modal-1790567929573 1.30.2 安装 主机 deploy-detail-modal-1790567929573 手动执行 成功 1 详情 删除" [ref=f16e750]:
                                - cell [ref=f16e751]:
                                    - generic [ref=f16e752] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-detail-modal-1790567929573-task" [ref=f16e755]
                                - cell "软件组件 deploy-detail-modal-1790567929573 1.30.2" [ref=f16e757]:
                                    - generic [ref=f16e760]:
                                        - generic [ref=f16e761]: 软件组件
                                        - generic [ref=f16e762]: deploy-detail-modal-1790567929573 1.30.2
                                - cell "安装" [ref=f16e763]
                                - cell "主机" [ref=f16e765]
                                - cell "deploy-detail-modal-1790567929573" [ref=f16e767]
                                - cell "手动执行" [ref=f16e769]
                                - cell "成功" [ref=f16e771]
                                - cell "1" [ref=f16e776]
                                - cell [ref=f16e778]:
                                    - generic [ref=f16e781]:
                                        - button "详情" [ref=f16e783] [cursor=pointer]
                                        - button "删除" [ref=f16e788] [cursor=pointer]
                            - row "deploy-api-trace-1790567915332-task 软件组件 deploy-api-trace-1790567915332-nginx 1.30.2 安装 主机 deploy-api-trace-1790567915332-scope SSH 执行 失败 1 详情 删除" [ref=f16e791]:
                                - cell [ref=f16e792]:
                                    - generic [ref=f16e793] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-api-trace-1790567915332-task" [ref=f16e796]
                                - cell "软件组件 deploy-api-trace-1790567915332-nginx 1.30.2" [ref=f16e798]:
                                    - generic [ref=f16e801]:
                                        - generic [ref=f16e802]: 软件组件
                                        - generic [ref=f16e803]: deploy-api-trace-1790567915332-nginx 1.30.2
                                - cell "安装" [ref=f16e804]
                                - cell "主机" [ref=f16e806]
                                - cell "deploy-api-trace-1790567915332-scope" [ref=f16e808]
                                - cell "SSH 执行" [ref=f16e810]
                                - cell "失败" [ref=f16e812]
                                - cell "1" [ref=f16e817]
                                - cell [ref=f16e819]:
                                    - generic [ref=f16e822]:
                                        - button "详情" [ref=f16e824] [cursor=pointer]
                                        - button "删除" [ref=f16e829] [cursor=pointer]
                    - generic [ref=f16e835]:
                        - generic [ref=f16e837]:
                            - generic [ref=f16e838]: 总数
                            - strong [ref=f16e839]: "41"
                        - list [ref=f16e840]:
                            - listitem "上一页" [ref=f16e841]:
                                - button "首页" [disabled] [ref=f16e843]:
                                    - generic [aria-hidden] [ref=f16e844]: 《
                                - text: .
                            - listitem "第 1 页" [ref=f16e848] [cursor=pointer]: "1"
                            - listitem "第 2 页" [ref=f16e849] [cursor=pointer]: "2"
                            - listitem "第 3 页" [ref=f16e850] [cursor=pointer]: "3"
                            - listitem "第 4 页" [ref=f16e851] [cursor=pointer]: "4"
                            - listitem "第 5 页" [ref=f16e852] [cursor=pointer]: "5"
                            - listitem "下一页" [ref=f16e853] [cursor=pointer]:
                                - button "末页" [ref=f16e858]:
                                    - generic [aria-hidden] [ref=f16e859]: 》
                                - text: .
                        - generic "页码" [ref=f16e860]:
                            - combobox [ref=f16e861] [cursor=pointer]:
                                - generic "10 条/页" [ref=f16e862]:
                                    - generic [ref=f16e863]:
                                        - textbox [aria-hidden]
                                        - generic [ref=f16e864]: 10 条/页
        - generic [ref=f16e869]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  177 |   if (node.path === '/business/deploy/task') {
  178 |     await expect(page.locator('main').getByRole('heading', { name: '部署任务', exact: true }))
  179 |       .toBeVisible({ timeout: 15000 });
  180 |   }
  181 |   await expect(page.locator('main'), `main content for ${node.path}`).toContainText(
  182 |     expectBody,
  183 |     { timeout: 15000 },
  184 |   );
  185 |   await expect(page.locator('main')).not.toContainText('高频管理入口');
  186 | }
  187 |
  188 | test.describe.serial('Business navigation repair', () => {
  189 |   test('sidebar business block is contiguous, one 服务目录, distinct ArgoCD label', async ({ page }) => {
  190 |     await signInWithUi(page, adminCredentials);
  191 |     await expect(siderLabel(page).first()).toBeVisible({ timeout: 15000 });
  192 |
  193 |     // Exactly one 服务目录 entry: the legacy id159 duplicate was merged into it.
  194 |     await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);
  195 |
  196 |     // ArgoCD root carries the full 应用交付 label (it is not bare 应用交付).
  197 |     await expect(sider(page).getByText('ArgoCD 应用交付', { exact: true })).toHaveCount(1);
  198 |
  199 |     // The only 应用发布 is the K8s release child, but the K8s group starts
  200 |     // collapsed — expand it first so the level-1 entry renders before asserting.
  201 |     await sider(page)
  202 |       .locator('.app-shell__menu-entry--level-0')
  203 |       .getByText('Kubernetes', { exact: true })
  204 |       .click({ timeout: 8000 });
  205 |     await expect(sider(page).getByText('应用发布', { exact: true })).toHaveCount(1);
  206 |
  207 |     // Top-level roots (level-0) appear in DOM order as the sidebar block: a platform
  208 |     // workbench root (e.g. 工作台) precedes the business run, which must sit as a
  209 |     // contiguous run in backend seed Sort (files=5, k8s=6, argocd=7, obs=8), ahead
  210 |     // of the system governance roots.
  211 |     const rootLabels = (await sider(page)
  212 |       .locator('.app-shell__menu-entry--level-0 .app-shell__menu-entry-label')
  213 |       .allTextContents()).map((text) => text.replace(/\s+/g, ' ').trim());
  214 |
  215 |     const tree = await businessTree(page);
  216 |     const roots = tree.filter((node) => node.path?.startsWith('/business/'));
  217 |     const ordinal = (p: string) => BACKEND_SEED_ORDER.indexOf(p);
  218 |     const ordered = [...roots].sort((a, b) => {
  219 |       const diff = ordinal(a.path ?? '') - ordinal(b.path ?? '');
  220 |       return diff !== 0 ? diff : a.id - b.id;
  221 |     });
  222 |     const expectedRootLabels = ordered.map((node) => menuLabel(node.titleKey));
  223 |
  224 |     // The business run starts at the CMDB root (a workbench root precedes it, so it
  225 |     // is not at index 0) and must be the exact, contiguous 8-label seed-order slice.
  226 |     const bizStart = rootLabels.indexOf(expectedRootLabels[0]);
  227 |     expect(bizStart, 'business block should begin at the CMDB root').toBeGreaterThanOrEqual(0);
  228 |     expect(rootLabels.slice(bizStart, bizStart + expectedRootLabels.length)).toEqual(expectedRootLabels);
  229 |
  230 |     // Nothing business-shaped recurs after the block, so system governance follows.
  231 |     const afterBiz = rootLabels.slice(bizStart + expectedRootLabels.length);
  232 |     for (const label of Array.from(new Set(expectedRootLabels))) {
  233 |       expect(afterBiz, `business root ${label} must not recur after the contiguous block`).not.toContain(label);
  234 |     }
  235 |   });
  236 |
  237 |   test('every business parent/leaf entry loads the expected main content', async ({ page }) => {
  238 |     test.setTimeout(180000);
  239 |     await signInWithUi(page, adminCredentials);
  240 |     const tree = await businessTree(page);
  241 |     const businessRoots = tree.filter((node) => node.path?.startsWith('/business/'));
  242 |     // Index live rows by path and record each node's parent path so the walk can
  243 |     // rebuild its ancestor chain top-down for a given route.
  244 |     const byPath = new Map<string, MenuNodeRow>();
  245 |     const parentPath = new Map<string, string | null>();
  246 |     const flatten = (nodes: MenuNodeRow[], parent: string | null) => {
  247 |       for (const node of nodes) {
  248 |         const route = node.path ?? '';
  249 |         byPath.set(route, node);
  250 |         parentPath.set(route, parent);
  251 |         flatten(node.children ?? [], route);
  252 |       }
  253 |     };
  254 |     flatten(businessRoots, null);
  255 |
  256 |     // Iterate the 30-stop plan and require every seed path to resolve to a live row
  257 |     // and be navigated — a missing seed is a hard failure, never a silent skip.
  258 |     for (const [route, body] of BUSINESS_ENTRIES) {
  259 |       const node = byPath.get(route);
  260 |       if (!node) {
  261 |         throw new Error(`seed menu row ${route} is missing from the live menu tree`);
  262 |       }
  263 |       if (!node.titleKey) {
  264 |         throw new Error(`seed menu row ${route} lacks a titleKey`);
  265 |       }
  266 |       const ancestors: MenuNodeRow[] = [];
  267 |       let cursor = parentPath.get(route);
  268 |       while (cursor) {
  269 |         const parent = byPath.get(cursor);
  270 |         if (parent) ancestors.unshift(parent);
  271 |         cursor = parentPath.get(cursor);
  272 |       }
  273 |       await test.step(`click ${route}`, async () => {
  274 |         try {
  275 |           await walkEntry(page, node, ancestors, businessRoots, body);
  276 |         } catch (error) {
> 277 |           throw new Error(`Failed navigating to ${route}: ${(error as Error).message}`, { cause: error });
      |                 ^ Error: Failed navigating to /business/deploy/task: expect(locator).toBeVisible() failed
  278 |         }
  279 |       });
  280 |     }
  281 |   });
  282 |
  283 |   test('ServiceCatalog sidebar 3 items, page tabs, back/forward, reload', async ({ page }) => {
  284 |     const badNav: string[] = [];
  285 |     page.on('response', (resp) => {
  286 |       if (resp.status() === 404 && resp.request().url().includes('/business/service')) badNav.push(resp.url());
  287 |     });
  288 |
  289 |     test.setTimeout(90000);
  290 |     await signInWithUi(page, adminCredentials);
  291 |     // Sidebar shows the service catalog group with its three direct children.
  292 |     await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);
  293 |     await sider(page).getByText('服务目录', { exact: true }).first().click({ timeout: 8000 });
  294 |
  295 |     const catalogCases = [
  296 |       { key: 'applications', label: '应用', button: '新增应用' },
  297 |       { key: 'services', label: '服务', button: '新增服务' },
  298 |       { key: 'instances', label: '服务实例', button: '新增服务实例' },
  299 |     ] as const;
  300 |
  301 |     // Locators scoped to MAIN so top-level header tabs with the same name can't match.
  302 |     const assertOn = async (item: (typeof catalogCases)[number]) => {
  303 |       await expect(page).toHaveURL(new RegExp(`/business/service-catalog/${item.key}$`));
  304 |       const main = page.locator('main');
  305 |       await expect(main.getByRole('button', { name: item.button, exact: true })).toBeVisible({ timeout: 15000 });
  306 |       await expect(main.getByRole('tab', { name: item.label, exact: true })).toHaveAttribute('aria-selected', 'true');
  307 |     };
  308 |
  309 |     // Sidebar: really clicking each of the three level-1 children loads its own page
  310 |     // and shows the matching 新增 button. Scoped to the 服务目录 ancestor so the
  311 |     // Kubernetes 服务 child cannot collide in strict mode.
  312 |     for (const item of catalogCases) {
  313 |       await test.step(`sidebar click ${item.key}`, async () => {
  314 |         await (await serviceCatalogChild(page, item.label)).click({ timeout: 8000 });
  315 |         await assertOn(item);
  316 |       });
  317 |     }
  318 |
  319 |     // In-page tabs drive the URL and write it back for all three.
  320 |     for (const item of catalogCases) {
  321 |       await test.step(`page tab ${item.label}`, async () => {
  322 |         await page.locator('main').getByRole('tab', { name: item.label, exact: true }).click({ timeout: 8000 });
  323 |         await assertOn(item);
  324 |       });
  325 |     }
  326 |
  327 |     // Browser back / forward stay consistent with the URL-driven active tab: the tab
  328 |     // loop above ends on 服务实例, so back lands on 服务 and forward re-enters 服务实例.
  329 |     await page.goBack();
  330 |     await assertOn(catalogCases[1]);
  331 |     await page.goForward();
  332 |     await assertOn(catalogCases[2]);
  333 |
  334 |     // Deep-link + real reload keeps the URL-driven tab for all three.
  335 |     for (const item of catalogCases) {
  336 |       await test.step(`deep link + reload ${item.key}`, async () => {
  337 |         await page.goto(`/business/service-catalog/${item.key}`, { waitUntil: 'domcontentloaded' });
  338 |         await assertOn(item);
  339 |         await page.reload({ waitUntil: 'domcontentloaded' });
  340 |         await assertOn(item);
  341 |       });
  342 |     }
  343 |
  344 |     expect(badNav).toEqual([]);
  345 |   });
  346 | });
  347 |
```
