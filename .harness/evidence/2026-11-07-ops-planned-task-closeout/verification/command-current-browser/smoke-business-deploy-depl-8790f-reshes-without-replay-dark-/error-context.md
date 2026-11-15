# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-command.spec.ts >> viewed task command conflict refreshes without replay (dark)
- Location: tests\smoke\business\deploy\deploy-command.spec.ts:10:2

# Error details

```
Error: expect(received).toEqual(expected) // deep equality

- Expected  - 1
+ Received  + 7

- Array []
+ Array [
+   "Error: business.deploy.task.commandConflict
+
+ expect(received).toBeTruthy()
+
+ Received: false",
+ ]
```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
    - generic [ref=e3]:
        - complementary [ref=e4]:
            - generic [ref=e5]:
                - generic [ref=e6]: P
                - menu [ref=e10]:
                    - generic [ref=e11]:
                        - menuitem [ref=e12] [cursor=pointer]:
                            - button "工作台" [ref=e13]
                        - button "CMDB" [ref=e23] [cursor=pointer]
                        - menuitem [ref=e35] [cursor=pointer]:
                            - button "业务域" [ref=e36]
                        - menuitem [ref=e45] [cursor=pointer]:
                            - button "服务目录" [ref=e46]
                        - button "安装部署" [ref=e58] [cursor=pointer]
                        - button "Kubernetes" [ref=e70] [cursor=pointer]
                        - button "可观测中心" [ref=e82] [cursor=pointer]
                        - button "访问控制" [ref=e94] [cursor=pointer]
                        - button "组织架构" [ref=e108] [cursor=pointer]
                        - button "安全审计" [ref=e123] [cursor=pointer]
                        - button "低代码平台" [ref=e135] [cursor=pointer]
                        - button "平台配置" [ref=e147] [cursor=pointer]
                        - menuitem [ref=e156] [cursor=pointer]:
                            - button "文件分发" [ref=e157]
                        - menuitem [ref=e166] [cursor=pointer]:
                            - button "ArgoCD" [ref=e167]
        - generic [ref=e173]:
            - generic [ref=e174]:
                - generic [ref=e175]:
                    - button "切换导航" [ref=e176] [cursor=pointer]
                    - list [ref=e180]:
                        - listitem [ref=e181]: 首页
                        - listitem [ref=e185]: 任务详情
                - generic [ref=e186]:
                    - button "平台功能台" [ref=e188]:
                        - generic [ref=e191]: 搜索菜单、页面、功能…
                    - button "通知中心" [ref=e193] [cursor=pointer]
                    - button "平台偏好" [ref=e197] [cursor=pointer]
                    - 'button "用户: Administrator" [ref=e202] [cursor=pointer]':
                        - generic [ref=e203]: A
            - tablist "已打开页面" [ref=e205]:
                - tab "工作台" [ref=e206] [cursor=pointer]
                - tab "任务详情" [selected] [ref=e210] [cursor=pointer]
            - main [ref=e212]:
                - generic [ref=e214]:
                    - generic [ref=e215]:
                        - generic [ref=e216]:
                            - heading "command-dark-1794063615059" [level=5] [ref=e218]
                            - generic [ref=e219]: 任务详情
                        - generic [ref=e221]:
                            - generic [ref=e222]: 实时连接
                            - button "返回" [ref=e226] [cursor=pointer]
                    - generic [ref=e229]:
                        - generic [ref=e232]:
                            - generic [ref=e233]:
                                - generic [ref=e234]:
                                    - generic [ref=e235]: 运维平台 / 部署任务
                                    - heading "command-dark-1794063615059" [level=5] [ref=e236]
                                    - generic [ref=e237]: 查看任务摘要、状态流转和每台目标主机的执行结果，便于运维排障与人工收口。
                                - generic [ref=e238]: 执行中
                            - generic [ref=e240]:
                                - generic [ref=e241]:
                                    - generic [ref=e242]: 目标主机
                                    - generic [ref=e243]: "1"
                                - generic [ref=e244]:
                                    - generic [ref=e245]: 成功
                                    - generic [ref=e246]: "0"
                                - generic [ref=e247]:
                                    - generic [ref=e248]: 失败
                                    - generic [ref=e249]: "0"
                        - generic [ref=e253]:
                            - generic [ref=e254]: 任务状态流转
                            - list "任务状态流转" [ref=e256]:
                                - listitem [ref=e257]:
                                    - generic [ref=e259]: 草稿
                                - listitem [ref=e260]:
                                    - generic [ref=e262]: 待执行
                                - listitem [ref=e263]:
                                    - generic [ref=e265]: 执行中
                                - listitem [ref=e266]:
                                    - generic [ref=e268]: 成功
                                - listitem [ref=e269]:
                                    - generic [ref=e271]: 失败
                                - listitem [ref=e272]:
                                    - generic [ref=e274]: 已取消
                            - generic [ref=e275]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                        - generic [ref=e279]:
                            - generic [ref=e280]: 任务级操作
                            - generic [ref=e282]:
                                - generic [ref=e283]:
                                    - generic [ref=e284]: 仅允许在可操作状态执行启动或取消，任务结果会同步汇总到详情页。
                                    - generic [ref=e285]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                                - generic [ref=e286]:
                                    - button "暂停" [ref=e288] [cursor=pointer]
                                    - button "取消" [ref=e295] [cursor=pointer]
                        - table [ref=e301]:
                            - rowgroup [ref=e302]:
                                - row [ref=e303]:
                                    - cell "任务来源" [ref=e304]
                                    - cell "软件组件" [ref=e305]
                                    - cell "软件组件" [ref=e306]
                                    - cell "command-dark-1794063615059 1.0.0" [ref=e307]
                                - row [ref=e308]:
                                    - cell "任务动作" [ref=e309]
                                    - cell "安装" [ref=e310]
                                    - cell "执行模式" [ref=e311]
                                    - cell "固定任务" [ref=e312]
                                - row [ref=e313]:
                                    - cell "模板参数" [ref=e314]
                                    - cell "-" [ref=e315]
                                    - cell "状态" [ref=e316]
                                    - cell "执行中" [ref=e317]
                                - row [ref=e318]:
                                    - cell "目标来源" [ref=e319]
                                    - cell "主机" [ref=e320]
                                    - cell "业务域" [ref=e321]
                                    - cell "command-dark-1794063615059" [ref=e322]
                                - row [ref=e323]:
                                    - cell "执行方式" [ref=e324]
                                    - cell "手动执行" [ref=e325]
                                    - cell "外部任务 ID" [ref=e326]
                                    - cell "-" [ref=e327]
                                - row [ref=e328]:
                                    - cell "开始时间" [ref=e329]
                                    - cell "2026-11-07 23:00" [ref=e330]
                                    - cell "结束时间" [ref=e331]
                                    - cell "-" [ref=e332]
                                - row [ref=e333]:
                                    - cell "耗时" [ref=e334]
                                    - cell "-" [ref=e335]
                                    - cell "主机数" [ref=e336]
                                    - cell "1" [ref=e337]
                                - row [ref=e338]:
                                    - cell "成功数" [ref=e339]
                                    - cell "0" [ref=e340]
                                    - cell "失败数" [ref=e341]
                                    - cell "0" [ref=e342]
                                - row [ref=e343]:
                                    - cell "执行中" [ref=e344]
                                    - cell "1" [ref=e345]
                                    - cell "跳过数" [ref=e346]
                                    - cell "0" [ref=e347]
                                - row [ref=e348]:
                                    - cell "备注" [ref=e349]
                                    - cell "-" [ref=e350]
                        - generic [ref=e354]:
                            - generic [ref=e357]:
                                - generic [ref=e358]: 主机执行明细
                                - generic [ref=e359]: 先查看各状态分布，再按主机展开过程日志和标准输出。
                            - generic [ref=e361]:
                                - generic [ref=e362]:
                                    - generic [ref=e363]: 待执行
                                    - generic [ref=e364]: "0"
                                - generic [ref=e365]:
                                    - generic [ref=e366]: 执行中
                                    - generic [ref=e367]: "1"
                                - generic [ref=e368]:
                                    - generic [ref=e369]: 成功
                                    - generic [ref=e370]: "0"
                                - generic [ref=e371]:
                                    - generic [ref=e372]: 失败
                                    - generic [ref=e373]: "0"
                                - generic [ref=e374]:
                                    - generic [ref=e375]: 已跳过
                                    - generic [ref=e376]: "0"
                            - generic [ref=e378]:
                                - generic [ref=e379]:
                                    - generic [ref=e380]: 共 1 条
                                    - generic [ref=e381]: 左右滑动查看更多字段
                                - generic [ref=e384]:
                                    - table [ref=e388]:
                                        - rowgroup [ref=e398]:
                                            - row [ref=e399]:
                                                - columnheader "主机" [ref=e400]
                                                - columnheader "主机 IP" [ref=e402]
                                                - columnheader "状态" [ref=e404]
                                                - columnheader "开始时间" [ref=e406]
                                                - columnheader "结束时间" [ref=e408]
                                                - columnheader "耗时" [ref=e410]
                                                - columnheader "错误信息" [ref=e412]
                                                - columnheader "操作" [ref=e414]
                                        - rowgroup [ref=e416]:
                                            - row [ref=e417]:
                                                - cell "command-dark-1794063615059" [ref=e418]
                                                - cell "10.245.112.75" [ref=e420]
                                                - cell "执行中" [ref=e422]
                                                - cell "2026-11-07 23:00" [ref=e427]
                                                - cell "-" [ref=e429]
                                                - cell "-" [ref=e431]
                                                - cell "-" [ref=e433]
                                                - cell [ref=e435]:
                                                    - generic [ref=e439]:
                                                        - button "标记成功" [ref=e440] [cursor=pointer]
                                                        - button "标记失败" [ref=e441] [cursor=pointer]
                                    - list [ref=e446]:
                                        - listitem "上一页" [ref=e447]: .
                                        - listitem "第 1 页" [ref=e450] [cursor=pointer]: "1"
                                        - listitem "下一页" [ref=e451]: .
                        - region "日志来源" [ref=e455]:
                            - generic [ref=e456]:
                                - combobox "执行目标" [ref=e458] [cursor=pointer]:
                                    - generic "command-dark-1794063615059 / 10.245.112.75" [ref=e459]:
                                        - generic [ref=e460]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e461]: command-dark-1794063615059 / 10.245.112.75
                                - button "定位首个失败目标" [disabled] [ref=e467]
                                - button "重新连接" [ref=e469] [cursor=pointer]
                                - generic [ref=e470]: command-dark-1794063615059 / 10.245.112.75
                            - region "执行步骤" [ref=e473]:
                                - list [ref=e476]:
                                    - listitem [ref=e477]:
                                        - button "定位步骤 1" [ref=e478]:
                                            - generic [ref=e480]: package_default
                                            - generic [ref=e481]: pending
                            - generic [ref=e484]:
                                - generic [ref=e485]:
                                    - button "展开日志" [ref=e487] [cursor=pointer]
                                    - combobox "日志流" [ref=e489] [cursor=pointer]:
                                        - generic [ref=e491]:
                                            - textbox "日志流"
                                            - generic [ref=e492]: 日志流
                                - region "command-dark-1794063615059 / command-dark-1794063615059" [ref=e497]:
                                    - generic [ref=e498]:
                                        - generic [ref=e499]: command-dark-1794063615059 / command-dark-1794063615059
                                        - generic [ref=e500]:
                                            - textbox "搜索已保留日志" [ref=e504]
                                            - button "暂停追尾" [ref=e508] [cursor=pointer]
                                            - switch "自动换行" [checked] [ref=e509] [cursor=pointer]
                                    - generic [ref=e511]: 暂无日志
                            - generic [ref=e516]:
                                - generic [ref=e518]:
                                    - generic [ref=e519]:
                                        - generic [ref=e520]: command-dark-1794063615059
                                        - generic [ref=e521]: 10.245.112.75 / 2026-11-07 23:00
                                    - generic [ref=e523]: 执行中
                                - generic [ref=e526]:
                                    - generic [ref=e527]:
                                        - generic [ref=e528]: 共 3 条
                                        - generic [ref=e529]: 左右滑动查看更多字段
                                    - table [ref=e536]:
                                        - rowgroup [ref=e543]:
                                            - row [ref=e544]:
                                                - columnheader "步骤" [ref=e545]
                                                - columnheader "步骤类型" [ref=e547]
                                                - columnheader "阶段" [ref=e549]
                                                - columnheader "开始时间" [ref=e551]
                                                - columnheader "过程信息" [ref=e553]
                                        - rowgroup [ref=e555]:
                                            - row [ref=e556]:
                                                - cell "- -" [ref=e557]:
                                                    - generic [ref=e560]:
                                                        - generic [ref=e561]: "-"
                                                        - generic [ref=e562]: "-"
                                                - cell "-" [ref=e563]
                                                - cell "进入执行队列" [ref=e565]
                                                - cell "2026-11-07 23:00" [ref=e570]
                                                - cell "Task host queued" [ref=e572]
                                            - row [ref=e574]:
                                                - cell "- -" [ref=e575]:
                                                    - generic [ref=e578]:
                                                        - generic [ref=e579]: "-"
                                                        - generic [ref=e580]: "-"
                                                - cell "-" [ref=e581]
                                                - cell "记录执行结果" [ref=e583]
                                                - cell "2026-11-07 23:00" [ref=e588]
                                                - cell "status=failed" [ref=e590]
                                            - row [ref=e592]:
                                                - cell "- -" [ref=e593]:
                                                    - generic [ref=e596]:
                                                        - generic [ref=e597]: "-"
                                                        - generic [ref=e598]: "-"
                                                - cell "-" [ref=e599]
                                                - cell "错误信息" [ref=e601]
                                                - cell "2026-11-07 23:00" [ref=e606]
                                                - cell "command fixture failure" [ref=e608]
                                - generic [ref=e610]:
                                    - separator
                                - table [ref=e614]:
                                    - rowgroup [ref=e615]:
                                        - row [ref=e616]:
                                            - cell "标准输出" [ref=e617]
                                            - cell "-" [ref=e618]
                                        - row [ref=e620]:
                                            - cell "错误输出" [ref=e621]
                                            - cell "-" [ref=e622]
            - generic [ref=e624]: Pantheon Platform · Enterprise Admin Foundation
    - generic:
        - generic:
            - alert [ref=e625]: 任务状态已变化，请刷新并确认当前状态后重新操作。
            - alert [ref=e629]: 部署任务已进入重试
```

# Test source

```ts
  1  | import { expect, test, type APIResponse } from '@playwright/test';
  2  | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3  | import { startReviewedTask } from './reviewed-start';
  4  |
  5  | async function success<T>(response: APIResponse): Promise<T> {
  6  |  const payload = await response.json(); expect(response.ok(), payload.message).toBeTruthy(); expect(payload.code,payload.message).toBe(200); return payload.data as T;
  7  | }
  8  | type CommandTask = {id:number;status:string;stateVersion:string;hosts:{id:number}[]};
  9  | for(const mode of ['light','dark'] as const){
  10 |  test(`viewed task command conflict refreshes without replay (${mode})`,async({page},testInfo)=>{
  11 |   const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  12 |   await page.addInitScript(color=>localStorage.setItem('pantheon_color_mode',color),mode);
  13 |   const headers=apiRequestHeaders(login);const token=`command-${mode}-${Date.now()}`;
  14 |   let scopeId=0,hostId=0,packageId=0,taskId=0;let failed=false;
  15 |   try{
  16 |    scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  17 |    hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.245.${Math.floor(Math.random()*200)+1}.75`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  18 |    packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo fixture',status:'enabled'}}))).id;
  19 |    taskId=(await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:token,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}))).id;
  20 |    const running=await success<CommandTask>(await startReviewedTask(page.request,taskId,{headers,data:{}}));
  21 |    expect(running.stateVersion).toMatch(/^[a-f0-9]{64}$/);
  22 |    await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${running.hosts[0].id}/result`,{headers,data:{status:'failed',errorMessage:'command fixture failure'}}));
  23 |    const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}));expect(current.status).toBe('failed');
  24 |    // Real public API rejects a missing precondition before mutation.
  25 |    const missing=await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/retry`,{headers,data:{}});expect((await missing.json()).code).not.toBe(200);
  26 |    let submitted=false;let commands=0;let refreshes=0;
  27 |    await page.route(`**/business/deploy/tasks/${taskId}`,async route=>{
  28 |     const response=await route.fetch();const body=await response.json();if(!submitted)body.data.stateVersion=running.stateVersion;else refreshes++;
  29 |     await route.fulfill({response,json:body});
  30 |    });
  31 |    await page.route(`**/business/deploy/tasks/${taskId}/retry`,async route=>{
  32 |     submitted=true;commands++;expect(route.request().postDataJSON().stateVersion).toBe(commands===1?running.stateVersion:current.stateVersion);await route.continue();
  33 |    });
  34 |    await page.setViewportSize({width:1440,height:900});await page.goto(`/business/deploy/task/${taskId}`);
  35 |    const retry=page.getByRole('button',{name:'重试失败步骤',exact:true});await expect(retry).toBeVisible();
  36 |    const conflict=page.waitForResponse(response=>response.url().endsWith(`/tasks/${taskId}/retry`));await retry.click();expect((await (await conflict).json()).code).toBe(409);
  37 |    await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toBeVisible();await expect.poll(()=>refreshes).toBeGreaterThan(0);
  38 |    expect(commands).toBe(1);
  39 |    expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('failed');
  40 |    await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-desktop.png`),fullPage:true});
  41 |    await page.setViewportSize({width:390,height:844});if(await page.locator('.app-shell--mobile-open').count())await page.getByRole('button',{name:'切换导航',exact:true}).click();await expect(page.locator('.app-shell__sider')).not.toBeInViewport();await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-mobile.png`),fullPage:true});
  42 |    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  43 |    await retry.click();await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toHaveCount(0);
  44 |    await expect.poll(async()=> (await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('running');expect(commands).toBe(2);
  45 |   }catch(error){failed=true;throw error}finally{
  46 |    const cleanupErrors:string[]=[];
  47 |    try{
  48 |     const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  49 |     if(taskId){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));}
  50 |     for(const [id,endpoint] of [[taskId,'deploy/tasks'],[hostId,'cmdb/hosts'],[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(!id)continue;try{await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}))}catch(error){cleanupErrors.push(`${endpoint}/${id}: ${String(error)}`)}}
  51 |    }catch(error){cleanupErrors.push(String(error))}
> 52 |    await testInfo.attach('command-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskId,hostId,scopeId,packageId,cleanupErrors})});if(!failed)expect(cleanupErrors).toEqual([]);
     |                                                                                                                                                                                                  ^ Error: expect(received).toEqual(expected) // deep equality
  53 |   }
  54 |  });
  55 | }
  56 |
  57 | test('batch cancel retains failed selection and reports partial failure',async({page},testInfo)=>{
  58 |  const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  59 |  const headers=apiRequestHeaders(login);const token=`command-batch-${Date.now()}`;
  60 |  let scopeId=0,hostId=0,packageId=0;const taskIds:number[]=[];const hostIds:number[]=[];let failed=false;
  61 |  try{
  62 |   scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  63 |   hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.244.${Math.floor(Math.random()*200)+1}.77`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  64 |   hostIds.push(hostId);
  65 |   packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo batch fixture',status:'enabled'}}))).id;
  66 |   let staleVersion='';
  67 |   for(const suffix of ['a','b']){
  68 |    if(suffix==='b'){
  69 |     hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:`${token}-b-host`,ip:`10.243.${Math.floor(Math.random()*200)+1}.77`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;hostIds.push(hostId);
  70 |    }
  71 |    const created=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:`${token}-${suffix}`,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}));taskIds.push(created.id);staleVersion=created.stateVersion;
  72 |    await success(await startReviewedTask(page.request,created.id,{headers,data:{}}));
  73 |   }
  74 |   let submissions=0;
  75 |   await page.route('**/business/deploy/tasks?*',async route=>{const response=await route.fetch();const body=await response.json();if(submissions===0){const item=body.data.items.find((item:CommandTask)=>item.id===taskIds[1]);if(item)item.stateVersion=staleVersion;}await route.fulfill({response,json:body});});
  76 |   await page.route('**/business/deploy/tasks/*/cancel',async route=>{submissions++;await route.continue();});
  77 |   await page.goto('/business/deploy/task');
  78 |   const filtered=page.waitForResponse(response=>response.request().method()==='GET'&&response.url().includes('/business/deploy/tasks?')&&new URL(response.url()).searchParams.get('keyword')===token);
  79 |   await page.locator('.search-toolbar input').first().fill(token);await filtered;
  80 |   const rowA=page.getByRole('row').filter({hasText:`${token}-a`}).first();const rowB=page.getByRole('row').filter({hasText:`${token}-b`}).first();
  81 |   await expect(rowA).toBeVisible();await expect(rowB).toBeVisible();await rowA.locator('label.arco-checkbox').click();await expect(rowA.getByRole('checkbox')).toBeChecked();await rowB.locator('label.arco-checkbox').click();await expect(rowB.getByRole('checkbox')).toBeChecked();
  82 |   await page.getByRole('button',{name:'取消',exact:true}).first().click();await page.getByRole('button',{name:'确定',exact:true}).click();
  83 |   await expect(page.locator('.arco-alert-content').getByText('部分任务未能取消。列表已刷新，请确认剩余选择后重新操作。',{exact:true})).toBeVisible();
  84 |   await expect(rowA.getByRole('checkbox')).not.toBeChecked();await expect(rowB.getByRole('checkbox')).toBeChecked();expect(submissions).toBe(2);
  85 |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[0]}`,{headers}))).status).toBe('canceled');
  86 |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[1]}`,{headers}))).status).toBe('running');
  87 |   await page.screenshot({path:testInfo.outputPath('command-batch-partial.png'),fullPage:true});
  88 |  }catch(error){failed=true;throw error}finally{
  89 |   const errors:string[]=[];
  90 |   try{
  91 |    const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  92 |    for(const id of taskIds){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${id}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));await success(await page.request.delete(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));}
  93 |    for(const id of hostIds)await success(await page.request.delete(`${apiBaseUrl}/business/cmdb/hosts/${id}`,{headers:cleanupHeaders}));
  94 |    for(const [id,endpoint] of [[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(id)await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}));}
  95 |   }catch(error){errors.push(String(error))}
  96 |   await testInfo.attach('batch-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskIds,hostId,scopeId,packageId,errors})});if(!failed)expect(errors).toEqual([]);
  97 |  }
  98 | });
  99 |
```
