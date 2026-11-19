# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\ServiceCatalog.nav.spec.ts >> Business navigation repair >> every business parent/leaf entry loads the expected main content
- Location: tests\smoke\business\service\ServiceCatalog.nav.spec.ts:242:3

# Error details

```
Error: Failed navigating to /business/deploy/template: main content for /business/deploy/template

expect(received).toMatch(expected)

Expected pattern: /任务模板/
Received string:  ""
```

# Page snapshot

```yaml
- generic [ref=f15e3]:
    - complementary [ref=f15e4]:
        - generic [ref=f15e5]:
            - generic [ref=f15e6]:
                - generic [ref=f15e7]: P
                - generic [ref=f15e8]: Pantheon Base
            - menu [ref=f15e12]:
                - generic [ref=f15e13]:
                    - menuitem [ref=f15e14] [cursor=pointer]:
                        - button "工作台" [ref=f15e15]
                    - button "CMDB" [ref=f15e25] [cursor=pointer]
                    - button "业务域" [ref=f15e40] [cursor=pointer]
                    - button "服务目录" [ref=f15e55] [cursor=pointer]
                    - generic [ref=f15e67]:
                        - button "安装部署" [ref=f15e70] [cursor=pointer]
                        - generic [ref=f15e79]:
                            - menuitem [ref=f15e80] [cursor=pointer]:
                                - button "软件组件" [ref=f15e83]
                            - menuitem [ref=f15e89] [cursor=pointer]:
                                - button "任务模板" [active] [ref=f15e92]
                            - menuitem [ref=f15e98] [cursor=pointer]:
                                - button "部署任务" [ref=f15e101]
                    - menuitem [ref=f15e107] [cursor=pointer]:
                        - button "文件分发" [ref=f15e108]
                    - button "Kubernetes" [ref=f15e120] [cursor=pointer]
                    - menuitem [ref=f15e129] [cursor=pointer]:
                        - button "ArgoCD 应用交付" [ref=f15e130]
                    - button "可观测中心" [ref=f15e139] [cursor=pointer]
                    - button "访问控制" [ref=f15e151] [cursor=pointer]
                    - button "组织架构" [ref=f15e165] [cursor=pointer]
                    - button "安全审计" [ref=f15e180] [cursor=pointer]
                    - button "低代码平台" [ref=f15e192] [cursor=pointer]
                    - button "平台配置" [ref=f15e204] [cursor=pointer]
    - generic [ref=f15e213]:
        - generic [ref=f15e214]:
            - generic [ref=f15e215]:
                - button "切换导航" [ref=f15e216] [cursor=pointer]
                - list [ref=f15e220]:
                    - listitem [ref=f15e221]: 首页
                    - listitem [ref=f15e225]: 安装部署
                    - listitem [ref=f15e229]: 任务模板
            - generic [ref=f15e230]:
                - button "平台功能台" [ref=f15e232]:
                    - generic [ref=f15e235]: 搜索菜单、页面、功能…
                    - generic [ref=f15e236]: Ctrl K
                - button "通知中心" [ref=f15e238] [cursor=pointer]
                - button "平台偏好" [ref=f15e242] [cursor=pointer]
                - 'button "用户: Administrator" [ref=f15e247] [cursor=pointer]':
                    - generic [ref=f15e248]: A
                    - generic [ref=f15e250]:
                        - generic [ref=f15e251]: Administrator
                        - generic [ref=f15e252]: admin
        - tablist "已打开页面" [ref=f15e253]:
            - tab "工作台" [ref=f15e254] [cursor=pointer]
            - tab "环境管理" [ref=f15e258] [cursor=pointer]
            - tab "服务目录" [ref=f15e260] [cursor=pointer]
            - tab "应用" [ref=f15e262] [cursor=pointer]
            - tab "服务" [ref=f15e264] [cursor=pointer]
            - tab "服务实例" [ref=f15e266] [cursor=pointer]
            - tab "软件组件" [ref=f15e268] [cursor=pointer]
            - tab "任务模板" [selected] [ref=f15e270] [cursor=pointer]
        - main [ref=f15e272]:
            - generic [ref=f15e275]:
                - generic [ref=f15e277]:
                    - generic [ref=f15e278]:
                        - generic [ref=f15e279]: 运维平台 / 任务模板
                        - generic [ref=f15e280]: 任务模板
                    - generic [ref=f15e282]:
                        - generic [ref=f15e283]:
                            - generic [ref=f15e284]: 模板总数
                            - strong [ref=f15e285]: "19"
                        - generic [ref=f15e286]:
                            - generic [ref=f15e287]: 已启用
                            - strong [ref=f15e288]: "10"
                        - generic [ref=f15e289]:
                            - generic [ref=f15e290]: 步骤总数
                            - strong [ref=f15e291]: "13"
                    - button "模板治理视角" [ref=f15e293] [cursor=pointer]
                - search "筛选" [ref=f15e300]:
                    - generic [ref=f15e302]:
                        - textbox "关键字" [ref=f15e309]
                        - combobox [ref=f15e311] [cursor=pointer]:
                            - generic [ref=f15e313]:
                                - textbox "状态"
                                - generic [ref=f15e314]: 状态
                - generic [ref=f15e321]:
                    - generic [ref=f15e322]:
                        - generic [ref=f15e324]:
                            - generic [ref=f15e327]:
                                - button "导出" [ref=f15e329] [cursor=pointer]
                                - button "导入" [ref=f15e333] [cursor=pointer]
                            - button "新增" [ref=f15e338] [cursor=pointer]
                        - generic [ref=f15e341]: 已选 0 条
                        - button "清空选择" [disabled] [ref=f15e342]
                    - generic [ref=f15e344]:
                        - button "删除所选" [disabled]
                - generic [ref=f15e351]:
                    - table [ref=f15e355]:
                        - rowgroup [ref=f15e367]:
                            - row "模板名称 版本 分类 模板模式 模板步骤 默认组件 状态 说明 操作" [ref=f15e368]:
                                - columnheader [ref=f15e369]:
                                    - generic [ref=f15e371] [cursor=pointer]:
                                        - checkbox
                                - columnheader "模板名称" [ref=f15e374]
                                - columnheader "版本" [ref=f15e376]
                                - columnheader "分类" [ref=f15e378]
                                - columnheader "模板模式" [ref=f15e380]
                                - columnheader "模板步骤" [ref=f15e382]
                                - columnheader "默认组件" [ref=f15e384]
                                - columnheader "状态" [ref=f15e386]
                                - columnheader "说明" [ref=f15e388]
                                - columnheader "操作" [ref=f15e390]
                        - rowgroup [ref=f15e392]:
                            - row "deploy-script-template-1790567948672 v1 - 编排任务 安装 2 个步骤 准备目录 / 脚本步骤 安装组件 / 组件步骤 deploy-script-template-1790567948672 1.30.2 启用 详情 编辑 删除" [ref=f15e393]:
                                - cell [ref=f15e394]:
                                    - generic [ref=f15e395] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-script-template-1790567948672" [ref=f15e398]
                                - cell "v1" [ref=f15e400]
                                - cell "-" [ref=f15e402]
                                - cell "编排任务 安装" [ref=f15e404]:
                                    - generic [ref=f15e407]:
                                        - generic [ref=f15e408]: 编排任务
                                        - generic [ref=f15e411]: 安装
                                - cell "2 个步骤 准备目录 / 脚本步骤 安装组件 / 组件步骤" [ref=f15e414]:
                                    - generic [ref=f15e417]:
                                        - generic [ref=f15e418]: 2 个步骤
                                        - generic [ref=f15e420]:
                                            - generic [ref=f15e421]: 准备目录 / 脚本步骤
                                            - generic [ref=f15e424]: 安装组件 / 组件步骤
                                - cell "deploy-script-template-1790567948672 1.30.2" [ref=f15e427]
                                - cell "启用" [ref=f15e432]
                                - cell [ref=f15e437]
                                - cell [ref=f15e438]:
                                    - generic [ref=f15e441]:
                                        - button "详情" [ref=f15e443] [cursor=pointer]
                                        - button "编辑" [ref=f15e448] [cursor=pointer]
                                        - button "删除" [ref=f15e452] [cursor=pointer]
                            - row "deploy-script-template-1790567638470 v1 - 编排任务 安装 2 个步骤 准备目录 / 脚本步骤 安装组件 / 组件步骤 deploy-script-template-1790567638470 1.30.2 启用 详情 编辑 删除" [ref=f15e455]:
                                - cell [ref=f15e456]:
                                    - generic [ref=f15e457] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-script-template-1790567638470" [ref=f15e460]
                                - cell "v1" [ref=f15e462]
                                - cell "-" [ref=f15e464]
                                - cell "编排任务 安装" [ref=f15e466]:
                                    - generic [ref=f15e469]:
                                        - generic [ref=f15e470]: 编排任务
                                        - generic [ref=f15e473]: 安装
                                - cell "2 个步骤 准备目录 / 脚本步骤 安装组件 / 组件步骤" [ref=f15e476]:
                                    - generic [ref=f15e479]:
                                        - generic [ref=f15e480]: 2 个步骤
                                        - generic [ref=f15e482]:
                                            - generic [ref=f15e483]: 准备目录 / 脚本步骤
                                            - generic [ref=f15e486]: 安装组件 / 组件步骤
                                - cell "deploy-script-template-1790567638470 1.30.2" [ref=f15e489]
                                - cell "启用" [ref=f15e494]
                                - cell [ref=f15e499]
                                - cell [ref=f15e500]:
                                    - generic [ref=f15e503]:
                                        - button "详情" [ref=f15e505] [cursor=pointer]
                                        - button "编辑" [ref=f15e510] [cursor=pointer]
                                        - button "删除" [ref=f15e514] [cursor=pointer]
                            - row "deploy-script-template-1790567136614 v1 - 编排任务 安装 2 个步骤 准备目录 / 脚本步骤 安装组件 / 组件步骤 deploy-script-template-1790567136614 1.30.2 启用 详情 编辑 删除" [ref=f15e517]:
                                - cell [ref=f15e518]:
                                    - generic [ref=f15e519] [cursor=pointer]:
                                        - checkbox
                                - cell "deploy-script-template-1790567136614" [ref=f15e522]
                                - cell "v1" [ref=f15e524]
                                - cell "-" [ref=f15e526]
                                - cell "编排任务 安装" [ref=f15e528]:
                                    - generic [ref=f15e531]:
                                        - generic [ref=f15e532]: 编排任务
                                        - generic [ref=f15e535]: 安装
                                - cell "2 个步骤 准备目录 / 脚本步骤 安装组件 / 组件步骤" [ref=f15e538]:
                                    - generic [ref=f15e541]:
                                        - generic [ref=f15e542]: 2 个步骤
                                        - generic [ref=f15e544]:
                                            - generic [ref=f15e545]: 准备目录 / 脚本步骤
                                            - generic [ref=f15e548]: 安装组件 / 组件步骤
                                - cell "deploy-script-template-1790567136614 1.30.2" [ref=f15e551]
                                - cell "启用" [ref=f15e556]
                                - cell [ref=f15e561]
                                - cell [ref=f15e562]:
                                    - generic [ref=f15e565]:
                                        - button "详情" [ref=f15e567] [cursor=pointer]
                                        - button "编辑" [ref=f15e572] [cursor=pointer]
                                        - button "删除" [ref=f15e576] [cursor=pointer]
                            - row "redis-卸载（通用） v1 - 编排任务 卸载 1 个步骤 redis-卸载（通用） / 组件步骤 - 启用 版本可选组件模板 详情 编辑 删除" [ref=f15e579]:
                                - cell [ref=f15e580]:
                                    - generic [ref=f15e581] [cursor=pointer]:
                                        - checkbox
                                - cell "redis-卸载（通用）" [ref=f15e584]
                                - cell "v1" [ref=f15e586]
                                - cell "-" [ref=f15e588]
                                - cell "编排任务 卸载" [ref=f15e590]:
                                    - generic [ref=f15e593]:
                                        - generic [ref=f15e594]: 编排任务
                                        - generic [ref=f15e597]: 卸载
                                - cell "1 个步骤 redis-卸载（通用） / 组件步骤" [ref=f15e600]:
                                    - generic [ref=f15e603]:
                                        - generic [ref=f15e604]: 1 个步骤
                                        - generic [ref=f15e605]: redis-卸载（通用） / 组件步骤
                                - cell "-" [ref=f15e610]
                                - cell "启用" [ref=f15e615]
                                - cell "版本可选组件模板" [ref=f15e620]
                                - cell [ref=f15e622]:
                                    - generic [ref=f15e625]:
                                        - button "详情" [ref=f15e627] [cursor=pointer]
                                        - button "编辑" [ref=f15e632] [cursor=pointer]
                                        - button "删除" [ref=f15e636] [cursor=pointer]
                            - row "redis-安装（版本可选） v1 - 编排任务 安装 1 个步骤 redis-安装（版本可选） / 组件步骤 - 启用 版本可选组件模板 详情 编辑 删除" [ref=f15e639]:
                                - cell [ref=f15e640]:
                                    - generic [ref=f15e641] [cursor=pointer]:
                                        - checkbox
                                - cell "redis-安装（版本可选）" [ref=f15e644]
                                - cell "v1" [ref=f15e646]
                                - cell "-" [ref=f15e648]
                                - cell "编排任务 安装" [ref=f15e650]:
                                    - generic [ref=f15e653]:
                                        - generic [ref=f15e654]: 编排任务
                                        - generic [ref=f15e657]: 安装
                                - cell "1 个步骤 redis-安装（版本可选） / 组件步骤" [ref=f15e660]:
                                    - generic [ref=f15e663]:
                                        - generic [ref=f15e664]: 1 个步骤
                                        - generic [ref=f15e665]: redis-安装（版本可选） / 组件步骤
                                - cell "-" [ref=f15e670]
                                - cell "启用" [ref=f15e675]
                                - cell "版本可选组件模板" [ref=f15e680]
                                - cell [ref=f15e682]:
                                    - generic [ref=f15e685]:
                                        - button "详情" [ref=f15e687] [cursor=pointer]
                                        - button "编辑" [ref=f15e692] [cursor=pointer]
                                        - button "删除" [ref=f15e696] [cursor=pointer]
                            - row "nginx-卸载（通用） v1 - 编排任务 卸载 1 个步骤 nginx-卸载（通用） / 组件步骤 - 启用 版本可选组件模板 详情 编辑 删除" [ref=f15e699]:
                                - cell [ref=f15e700]:
                                    - generic [ref=f15e701] [cursor=pointer]:
                                        - checkbox
                                - cell "nginx-卸载（通用）" [ref=f15e704]
                                - cell "v1" [ref=f15e706]
                                - cell "-" [ref=f15e708]
                                - cell "编排任务 卸载" [ref=f15e710]:
                                    - generic [ref=f15e713]:
                                        - generic [ref=f15e714]: 编排任务
                                        - generic [ref=f15e717]: 卸载
                                - cell "1 个步骤 nginx-卸载（通用） / 组件步骤" [ref=f15e720]:
                                    - generic [ref=f15e723]:
                                        - generic [ref=f15e724]: 1 个步骤
                                        - generic [ref=f15e725]: nginx-卸载（通用） / 组件步骤
                                - cell "-" [ref=f15e730]
                                - cell "启用" [ref=f15e735]
                                - cell "版本可选组件模板" [ref=f15e740]
                                - cell [ref=f15e742]:
                                    - generic [ref=f15e745]:
                                        - button "详情" [ref=f15e747] [cursor=pointer]
                                        - button "编辑" [ref=f15e752] [cursor=pointer]
                                        - button "删除" [ref=f15e756] [cursor=pointer]
                            - row "nginx-安装（版本可选） v1 - 编排任务 安装 1 个步骤 nginx-安装（版本可选） / 组件步骤 - 启用 版本可选组件模板 详情 编辑 删除" [ref=f15e759]:
                                - cell [ref=f15e760]:
                                    - generic [ref=f15e761] [cursor=pointer]:
                                        - checkbox
                                - cell "nginx-安装（版本可选）" [ref=f15e764]
                                - cell "v1" [ref=f15e766]
                                - cell "-" [ref=f15e768]
                                - cell "编排任务 安装" [ref=f15e770]:
                                    - generic [ref=f15e773]:
                                        - generic [ref=f15e774]: 编排任务
                                        - generic [ref=f15e777]: 安装
                                - cell "1 个步骤 nginx-安装（版本可选） / 组件步骤" [ref=f15e780]:
                                    - generic [ref=f15e783]:
                                        - generic [ref=f15e784]: 1 个步骤
                                        - generic [ref=f15e785]: nginx-安装（版本可选） / 组件步骤
                                - cell "-" [ref=f15e790]
                                - cell "启用" [ref=f15e795]
                                - cell "版本可选组件模板" [ref=f15e800]
                                - cell [ref=f15e802]:
                                    - generic [ref=f15e805]:
                                        - button "详情" [ref=f15e807] [cursor=pointer]
                                        - button "编辑" [ref=f15e812] [cursor=pointer]
                                        - button "删除" [ref=f15e816] [cursor=pointer]
                            - row "minio-卸载 v1 - 编排任务 卸载 1 个步骤 minio-卸载 / 组件步骤 - 启用 已验证组件模板（安装/卸载分离） 详情 编辑 删除" [ref=f15e819]:
                                - cell [ref=f15e820]:
                                    - generic [ref=f15e821] [cursor=pointer]:
                                        - checkbox
                                - cell "minio-卸载" [ref=f15e824]
                                - cell "v1" [ref=f15e826]
                                - cell "-" [ref=f15e828]
                                - cell "编排任务 卸载" [ref=f15e830]:
                                    - generic [ref=f15e833]:
                                        - generic [ref=f15e834]: 编排任务
                                        - generic [ref=f15e837]: 卸载
                                - cell "1 个步骤 minio-卸载 / 组件步骤" [ref=f15e840]:
                                    - generic [ref=f15e843]:
                                        - generic [ref=f15e844]: 1 个步骤
                                        - generic [ref=f15e845]: minio-卸载 / 组件步骤
                                - cell "-" [ref=f15e850]
                                - cell "启用" [ref=f15e855]
                                - cell "已验证组件模板（安装/卸载分离）" [ref=f15e860]
                                - cell [ref=f15e862]:
                                    - generic [ref=f15e865]:
                                        - button "详情" [ref=f15e867] [cursor=pointer]
                                        - button "编辑" [ref=f15e872] [cursor=pointer]
                                        - button "删除" [ref=f15e876] [cursor=pointer]
                            - row "minio-安装 v1 - 编排任务 安装 1 个步骤 minio-安装 / 组件步骤 - 启用 已验证组件模板（安装/卸载分离） 详情 编辑 删除" [ref=f15e879]:
                                - cell [ref=f15e880]:
                                    - generic [ref=f15e881] [cursor=pointer]:
                                        - checkbox
                                - cell "minio-安装" [ref=f15e884]
                                - cell "v1" [ref=f15e886]
                                - cell "-" [ref=f15e888]
                                - cell "编排任务 安装" [ref=f15e890]:
                                    - generic [ref=f15e893]:
                                        - generic [ref=f15e894]: 编排任务
                                        - generic [ref=f15e897]: 安装
                                - cell "1 个步骤 minio-安装 / 组件步骤" [ref=f15e900]:
                                    - generic [ref=f15e903]:
                                        - generic [ref=f15e904]: 1 个步骤
                                        - generic [ref=f15e905]: minio-安装 / 组件步骤
                                - cell "-" [ref=f15e910]
                                - cell "启用" [ref=f15e915]
                                - cell "已验证组件模板（安装/卸载分离）" [ref=f15e920]
                                - cell [ref=f15e922]:
                                    - generic [ref=f15e925]:
                                        - button "详情" [ref=f15e927] [cursor=pointer]
                                        - button "编辑" [ref=f15e932] [cursor=pointer]
                                        - button "删除" [ref=f15e936] [cursor=pointer]
                            - row "redis-7.2.4-卸载 v1 - 编排任务 卸载 1 个步骤 redis-7.2.4-卸载 / 组件步骤 - 启用 已验证组件模板（安装/卸载分离） 详情 编辑 删除" [ref=f15e939]:
                                - cell [ref=f15e940]:
                                    - generic [ref=f15e941] [cursor=pointer]:
                                        - checkbox
                                - cell "redis-7.2.4-卸载" [ref=f15e944]
                                - cell "v1" [ref=f15e946]
                                - cell "-" [ref=f15e948]
                                - cell "编排任务 卸载" [ref=f15e950]:
                                    - generic [ref=f15e953]:
                                        - generic [ref=f15e954]: 编排任务
                                        - generic [ref=f15e957]: 卸载
                                - cell "1 个步骤 redis-7.2.4-卸载 / 组件步骤" [ref=f15e960]:
                                    - generic [ref=f15e963]:
                                        - generic [ref=f15e964]: 1 个步骤
                                        - generic [ref=f15e965]: redis-7.2.4-卸载 / 组件步骤
                                - cell "-" [ref=f15e970]
                                - cell "启用" [ref=f15e975]
                                - cell "已验证组件模板（安装/卸载分离）" [ref=f15e980]
                                - cell [ref=f15e982]:
                                    - generic [ref=f15e985]:
                                        - button "详情" [ref=f15e987] [cursor=pointer]
                                        - button "编辑" [ref=f15e992] [cursor=pointer]
                                        - button "删除" [ref=f15e996] [cursor=pointer]
                    - generic [ref=f15e1002]:
                        - generic [ref=f15e1004]:
                            - generic [ref=f15e1005]: 总数
                            - strong [ref=f15e1006]: "19"
                        - list [ref=f15e1007]:
                            - listitem "上一页" [ref=f15e1008]:
                                - button "首页" [disabled] [ref=f15e1010]:
                                    - generic [aria-hidden] [ref=f15e1011]: 《
                                - text: .
                            - listitem "第 1 页" [ref=f15e1015] [cursor=pointer]: "1"
                            - listitem "第 2 页" [ref=f15e1016] [cursor=pointer]: "2"
                            - listitem "下一页" [ref=f15e1017] [cursor=pointer]:
                                - button "末页" [ref=f15e1022]:
                                    - generic [aria-hidden] [ref=f15e1023]: 》
                                - text: .
                        - generic "页码" [ref=f15e1024]:
                            - combobox [ref=f15e1025] [cursor=pointer]:
                                - generic "10 条/页" [ref=f15e1026]:
                                    - generic [ref=f15e1027]:
                                        - textbox [aria-hidden]
                                        - generic [ref=f15e1028]: 10 条/页
        - generic [ref=f15e1033]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  182 |   if (node.parentId) {
  183 |     await (await childLocator(page, node)).click({ timeout: 8000 });
  184 |   } else {
  185 |     await (await rootLocator(page, node, roots)).click({ timeout: 8000 });
  186 |   }
  187 |   await settleBusinessPage(page);
  188 |
  189 |   const mainText = await page.locator('main').innerText();
  190 |   expect(mainText, `main content for ${node.path}`).toMatch(expectBody);
  191 | }
  192 |
  193 | test.describe.serial('Business navigation repair', () => {
  194 |   test('sidebar business block is contiguous, one 服务目录, distinct ArgoCD label', async ({ page }) => {
  195 |     await signInWithUi(page, adminCredentials);
  196 |     await expect(siderLabel(page).first()).toBeVisible({ timeout: 15000 });
  197 |
  198 |     // Exactly one 服务目录 entry: the legacy id159 duplicate was merged into it.
  199 |     await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);
  200 |
  201 |     // ArgoCD root carries the full 应用交付 label (it is not bare 应用交付).
  202 |     await expect(sider(page).getByText('ArgoCD 应用交付', { exact: true })).toHaveCount(1);
  203 |
  204 |     // The only 应用发布 is the K8s release child, but the K8s group starts
  205 |     // collapsed — expand it first so the level-1 entry renders before asserting.
  206 |     await sider(page)
  207 |       .locator('.app-shell__menu-entry--level-0')
  208 |       .getByText('Kubernetes', { exact: true })
  209 |       .click({ timeout: 8000 });
  210 |     await expect(sider(page).getByText('应用发布', { exact: true })).toHaveCount(1);
  211 |
  212 |     // Top-level roots (level-0) appear in DOM order as the sidebar block: a platform
  213 |     // workbench root (e.g. 工作台) precedes the business run, which must sit as a
  214 |     // contiguous run in backend seed Sort (files=5, k8s=6, argocd=7, obs=8), ahead
  215 |     // of the system governance roots.
  216 |     const rootLabels = (await sider(page)
  217 |       .locator('.app-shell__menu-entry--level-0 .app-shell__menu-entry-label')
  218 |       .allTextContents()).map((text) => text.replace(/\s+/g, ' ').trim());
  219 |
  220 |     const tree = await businessTree(page);
  221 |     const roots = tree.filter((node) => node.path?.startsWith('/business/'));
  222 |     const ordinal = (p: string) => BACKEND_SEED_ORDER.indexOf(p);
  223 |     const ordered = [...roots].sort((a, b) => {
  224 |       const diff = ordinal(a.path ?? '') - ordinal(b.path ?? '');
  225 |       return diff !== 0 ? diff : a.id - b.id;
  226 |     });
  227 |     const expectedRootLabels = ordered.map((node) => menuLabel(node.titleKey));
  228 |
  229 |     // The business run starts at the CMDB root (a workbench root precedes it, so it
  230 |     // is not at index 0) and must be the exact, contiguous 8-label seed-order slice.
  231 |     const bizStart = rootLabels.indexOf(expectedRootLabels[0]);
  232 |     expect(bizStart, 'business block should begin at the CMDB root').toBeGreaterThanOrEqual(0);
  233 |     expect(rootLabels.slice(bizStart, bizStart + expectedRootLabels.length)).toEqual(expectedRootLabels);
  234 |
  235 |     // Nothing business-shaped recurs after the block, so system governance follows.
  236 |     const afterBiz = rootLabels.slice(bizStart + expectedRootLabels.length);
  237 |     for (const label of Array.from(new Set(expectedRootLabels))) {
  238 |       expect(afterBiz, `business root ${label} must not recur after the contiguous block`).not.toContain(label);
  239 |     }
  240 |   });
  241 |
  242 |   test('every business parent/leaf entry loads the expected main content', async ({ page }) => {
  243 |     test.setTimeout(180000);
  244 |     await signInWithUi(page, adminCredentials);
  245 |     const tree = await businessTree(page);
  246 |     const businessRoots = tree.filter((node) => node.path?.startsWith('/business/'));
  247 |     // Index live rows by path and record each node's parent path so the walk can
  248 |     // rebuild its ancestor chain top-down for a given route.
  249 |     const byPath = new Map<string, MenuNodeRow>();
  250 |     const parentPath = new Map<string, string | null>();
  251 |     const flatten = (nodes: MenuNodeRow[], parent: string | null) => {
  252 |       for (const node of nodes) {
  253 |         const route = node.path ?? '';
  254 |         byPath.set(route, node);
  255 |         parentPath.set(route, parent);
  256 |         flatten(node.children ?? [], route);
  257 |       }
  258 |     };
  259 |     flatten(businessRoots, null);
  260 |
  261 |     // Iterate the 30-stop plan and require every seed path to resolve to a live row
  262 |     // and be navigated — a missing seed is a hard failure, never a silent skip.
  263 |     for (const [route, body] of BUSINESS_ENTRIES) {
  264 |       const node = byPath.get(route);
  265 |       if (!node) {
  266 |         throw new Error(`seed menu row ${route} is missing from the live menu tree`);
  267 |       }
  268 |       if (!node.titleKey) {
  269 |         throw new Error(`seed menu row ${route} lacks a titleKey`);
  270 |       }
  271 |       const ancestors: MenuNodeRow[] = [];
  272 |       let cursor = parentPath.get(route);
  273 |       while (cursor) {
  274 |         const parent = byPath.get(cursor);
  275 |         if (parent) ancestors.unshift(parent);
  276 |         cursor = parentPath.get(cursor);
  277 |       }
  278 |       await test.step(`click ${route}`, async () => {
  279 |         try {
  280 |           await walkEntry(page, node, ancestors, businessRoots, body);
  281 |         } catch (error) {
> 282 |           throw new Error(`Failed navigating to ${route}: ${(error as Error).message}`, { cause: error });
      |                 ^ Error: Failed navigating to /business/deploy/template: main content for /business/deploy/template
  283 |         }
  284 |       });
  285 |     }
  286 |   });
  287 |
  288 |   test('ServiceCatalog sidebar 3 items, page tabs, back/forward, reload', async ({ page }) => {
  289 |     const badNav: string[] = [];
  290 |     page.on('response', (resp) => {
  291 |       if (resp.status() === 404 && resp.request().url().includes('/business/service')) badNav.push(resp.url());
  292 |     });
  293 |
  294 |     test.setTimeout(90000);
  295 |     await signInWithUi(page, adminCredentials);
  296 |     // Sidebar shows the service catalog group with its three direct children.
  297 |     await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);
  298 |     await sider(page).getByText('服务目录', { exact: true }).first().click({ timeout: 8000 });
  299 |
  300 |     const catalogCases = [
  301 |       { key: 'applications', label: '应用', button: '新增应用' },
  302 |       { key: 'services', label: '服务', button: '新增服务' },
  303 |       { key: 'instances', label: '服务实例', button: '新增服务实例' },
  304 |     ] as const;
  305 |
  306 |     // Locators scoped to MAIN so top-level header tabs with the same name can't match.
  307 |     const assertOn = async (item: (typeof catalogCases)[number]) => {
  308 |       await expect(page).toHaveURL(new RegExp(`/business/service-catalog/${item.key}$`));
  309 |       const main = page.locator('main');
  310 |       await expect(main.getByRole('button', { name: item.button, exact: true })).toBeVisible({ timeout: 15000 });
  311 |       await expect(main.getByRole('tab', { name: item.label, exact: true })).toHaveAttribute('aria-selected', 'true');
  312 |     };
  313 |
  314 |     // Sidebar: really clicking each of the three level-1 children loads its own page
  315 |     // and shows the matching 新增 button. Scoped to the 服务目录 ancestor so the
  316 |     // Kubernetes 服务 child cannot collide in strict mode.
  317 |     for (const item of catalogCases) {
  318 |       await test.step(`sidebar click ${item.key}`, async () => {
  319 |         await (await serviceCatalogChild(page, item.label)).click({ timeout: 8000 });
  320 |         await assertOn(item);
  321 |       });
  322 |     }
  323 |
  324 |     // In-page tabs drive the URL and write it back for all three.
  325 |     for (const item of catalogCases) {
  326 |       await test.step(`page tab ${item.label}`, async () => {
  327 |         await page.locator('main').getByRole('tab', { name: item.label, exact: true }).click({ timeout: 8000 });
  328 |         await assertOn(item);
  329 |       });
  330 |     }
  331 |
  332 |     // Browser back / forward stay consistent with the URL-driven active tab: the tab
  333 |     // loop above ends on 服务实例, so back lands on 服务 and forward re-enters 服务实例.
  334 |     await page.goBack();
  335 |     await assertOn(catalogCases[1]);
  336 |     await page.goForward();
  337 |     await assertOn(catalogCases[2]);
  338 |
  339 |     // Deep-link + real reload keeps the URL-driven tab for all three.
  340 |     for (const item of catalogCases) {
  341 |       await test.step(`deep link + reload ${item.key}`, async () => {
  342 |         await page.goto(`/business/service-catalog/${item.key}`, { waitUntil: 'domcontentloaded' });
  343 |         await assertOn(item);
  344 |         await page.reload({ waitUntil: 'domcontentloaded' });
  345 |         await assertOn(item);
  346 |       });
  347 |     }
  348 |
  349 |     expect(badNav).toEqual([]);
  350 |   });
  351 | });
  352 |
```
