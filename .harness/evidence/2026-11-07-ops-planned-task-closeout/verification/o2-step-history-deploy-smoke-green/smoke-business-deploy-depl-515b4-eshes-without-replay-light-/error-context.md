# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-command.spec.ts >> viewed task command conflict refreshes without replay (light)
- Location: tests\smoke\business\deploy\deploy-command.spec.ts:10:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('button', { name: '重试失败步骤', exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByRole('button', { name: '重试失败步骤', exact: true }) with timeout 10000ms
  - waiting for getByRole('button', { name: '重试失败步骤', exact: true })

```

```yaml
- complementary:
    - text: P Pantheon Base
    - menu
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 任务详情
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "平台偏好"
- 'button "用户: 用户"': 用 用户
- tablist "已打开页面"
- main:
    - text: 加载失败 当前区块暂时无法返回数据，请重试或稍后再查看。
    - button "重试"
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3   | import { startReviewedTask } from './reviewed-start';
  4   |
  5   | async function success<T>(response: APIResponse): Promise<T> {
  6   |  const payload = await response.json(); expect(response.ok(), payload.message).toBeTruthy(); expect(payload.code,payload.message).toBe(200); return payload.data as T;
  7   | }
  8   | type CommandTask = {id:number;status:string;stateVersion:string;hosts:{id:number}[]};
  9   | for(const mode of ['light','dark'] as const){
  10  |  test(`viewed task command conflict refreshes without replay (${mode})`,async({page},testInfo)=>{
  11  |   const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  12  |   await page.addInitScript(color=>localStorage.setItem('pantheon_color_mode',color),mode);
  13  |   const headers=apiRequestHeaders(login);const token=`command-${mode}-${Date.now()}`;
  14  |   let scopeId=0,hostId=0,packageId=0,taskId=0;let failed=false;
  15  |   try{
  16  |    scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  17  |    hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.245.${Math.floor(Math.random()*200)+1}.${Math.floor(Math.random()*200)+1}`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  18  |    packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo fixture',status:'enabled'}}))).id;
  19  |    taskId=(await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:token,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}))).id;
  20  |    let running=await success<CommandTask>(await startReviewedTask(page.request,taskId,{headers,data:{}}));
  21  |    expect(running.stateVersion).toMatch(/^[a-f0-9]{64}$/);
  22  |    const paused=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/pause`,{headers,data:{stateVersion:running.stateVersion}}));expect(paused.status).toBe('paused');
  23  |    running=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/resume`,{headers,data:{stateVersion:paused.stateVersion}}));expect(running.status).toBe('running');
  24  |    await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${running.hosts[0].id}/result`,{headers,data:{status:'failed',errorMessage:'command fixture failure'}}));
  25  |    const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}));expect(current.status).toBe('failed');
  26  |    // Real public API rejects a missing precondition before mutation.
  27  |    const missing=await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/retry`,{headers,data:{}});expect((await missing.json()).code).not.toBe(200);
  28  |    let submitted=false;let commands=0;let refreshes=0;
  29  |    await page.route(`**/business/deploy/tasks/${taskId}`,async route=>{
  30  |     const response=await route.fetch();const body=await response.json();if(!submitted)body.data.stateVersion=running.stateVersion;else refreshes++;
  31  |     await route.fulfill({response,json:body});
  32  |    });
  33  |    await page.route(`**/business/deploy/tasks/${taskId}/retry`,async route=>{
  34  |     submitted=true;commands++;expect(route.request().postDataJSON().stateVersion).toBe(commands===1?running.stateVersion:current.stateVersion);await route.continue();
  35  |    });
  36  |    await page.setViewportSize({width:1440,height:900});await page.goto(`/business/deploy/task/${taskId}`);
> 37  |    const retry=page.getByRole('button',{name:'重试失败步骤',exact:true});await expect(retry).toBeVisible();
      |                                                                                        ^ Error: expect(locator).toBeVisible() failed
  38  |    const conflict=page.waitForResponse(response=>response.url().endsWith(`/tasks/${taskId}/retry`));await retry.click();expect((await (await conflict).json()).code).toBe(409);
  39  |    await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toBeVisible();await expect.poll(()=>refreshes).toBeGreaterThan(0);
  40  |    expect(commands).toBe(1);
  41  |    expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('failed');
  42  |    await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-desktop.png`),fullPage:true});
  43  |    await page.setViewportSize({width:390,height:844});if(await page.locator('.app-shell--mobile-open').count())await page.getByRole('button',{name:'切换导航',exact:true}).click();await expect(page.locator('.app-shell__sider')).not.toBeInViewport();await page.evaluate(()=>window.scrollTo(0,0));await page.screenshot({path:testInfo.outputPath(`command-conflict-${mode}-mobile.png`),fullPage:true});
  44  |    expect(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth)).toBe(true);
  45  |    await retry.click();await expect(page.locator('.arco-alert-content').getByText('任务状态已变化，请刷新并确认当前状态后重新操作。',{exact:true})).toHaveCount(0);
  46  |    await expect.poll(async()=> (await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers}))).status).toBe('running');expect(commands).toBe(2);
  47  |   }catch(error){failed=true;throw error}finally{
  48  |    const cleanupErrors:string[]=[];
  49  |    try{
  50  |     const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  51  |     if(taskId){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));}
  52  |     for(const [id,endpoint] of [[taskId,'deploy/tasks'],[hostId,'cmdb/hosts'],[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(!id)continue;try{await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}))}catch(error){cleanupErrors.push(`${endpoint}/${id}: ${String(error)}`)}}
  53  |    }catch(error){cleanupErrors.push(String(error))}
  54  |    await testInfo.attach('command-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskId,hostId,scopeId,packageId,cleanupErrors})});if(!failed)expect(cleanupErrors).toEqual([]);
  55  |   }
  56  |  });
  57  | }
  58  |
  59  | test('batch cancel retains failed selection and reports partial failure',async({page},testInfo)=>{
  60  |  const login=await loginByApi(page.request,adminCredentials);await installClientSession(page,login);await primeChineseLocale(page);
  61  |  const headers=apiRequestHeaders(login);const token=`command-batch-${Date.now()}`;
  62  |  let scopeId=0,hostId=0,packageId=0;const taskIds:number[]=[];const hostIds:number[]=[];let failed=false;
  63  |  try{
  64  |   scopeId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`,{headers,data:{code:token,name:token,environment:'dev',status:'active'}}))).id;
  65  |   hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:token,ip:`10.244.${Math.floor(Math.random()*200)+1}.${Math.floor(Math.random()*200)+1}`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;
  66  |   hostIds.push(hostId);
  67  |   packageId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`,{headers,data:{name:token,version:'1.0.0',installCommand:'echo batch fixture',status:'enabled'}}))).id;
  68  |   let staleVersion='';
  69  |   for(const suffix of ['a','b']){
  70  |    if(suffix==='b'){
  71  |     hostId=(await success<{id:number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`,{headers,data:{hostname:`${token}-b-host`,ip:`10.243.${Math.floor(Math.random()*200)+1}.${Math.floor(Math.random()*200)+1}`,os:'linux',status:'assigned',businessScopeId:scopeId}}))).id;hostIds.push(hostId);
  72  |    }
  73  |    const created=await success<CommandTask>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`,{headers,data:{name:`${token}-${suffix}`,packageId,businessScopeId:scopeId,targetType:'host',targetIds:[hostId],executorType:'manual'}}));taskIds.push(created.id);staleVersion=created.stateVersion;
  74  |    await success(await startReviewedTask(page.request,created.id,{headers,data:{}}));
  75  |   }
  76  |   let submissions=0;
  77  |   await page.route('**/business/deploy/tasks?*',async route=>{const response=await route.fetch();const body=await response.json();if(submissions===0){const item=body.data.items.find((item:CommandTask)=>item.id===taskIds[1]);if(item)item.stateVersion=staleVersion;}await route.fulfill({response,json:body});});
  78  |   await page.route('**/business/deploy/tasks/*/cancel',async route=>{submissions++;await route.continue();});
  79  |   await page.goto('/business/deploy/task');
  80  |   const filtered=page.waitForResponse(response=>response.request().method()==='GET'&&response.url().includes('/business/deploy/tasks?')&&new URL(response.url()).searchParams.get('keyword')===token);
  81  |   await page.locator('.search-toolbar input').first().fill(token);await filtered;
  82  |   const rowA=page.getByRole('row').filter({hasText:`${token}-a`}).first();const rowB=page.getByRole('row').filter({hasText:`${token}-b`}).first();
  83  |   await expect(rowA).toBeVisible();await expect(rowB).toBeVisible();await rowA.locator('label.arco-checkbox').click();await expect(rowA.getByRole('checkbox')).toBeChecked();await rowB.locator('label.arco-checkbox').click();await expect(rowB.getByRole('checkbox')).toBeChecked();
  84  |   await page.getByRole('button',{name:'取消',exact:true}).first().click();await page.getByRole('button',{name:'确定',exact:true}).click();
  85  |   await expect(page.locator('.arco-alert-content').getByText('部分任务未能取消。列表已刷新，请确认剩余选择后重新操作。',{exact:true})).toBeVisible();
  86  |   await expect(rowA.getByRole('checkbox')).not.toBeChecked();await expect(rowB.getByRole('checkbox')).toBeChecked();expect(submissions).toBe(2);
  87  |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[0]}`,{headers}))).status).toBe('canceled');
  88  |   expect((await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskIds[1]}`,{headers}))).status).toBe('running');
  89  |   await page.screenshot({path:testInfo.outputPath('command-batch-partial.png'),fullPage:true});
  90  |  }catch(error){failed=true;throw error}finally{
  91  |   const errors:string[]=[];
  92  |   try{
  93  |    const cleanupHeaders=await verifiedApiHeaders(page.request,login);
  94  |    for(const id of taskIds){const current=await success<CommandTask>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));if(current.status==='running')await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${id}/cancel`,{headers:cleanupHeaders,data:{stateVersion:current.stateVersion}}));await success(await page.request.delete(`${apiBaseUrl}/business/deploy/tasks/${id}`,{headers:cleanupHeaders}));}
  95  |    for(const id of hostIds)await success(await page.request.delete(`${apiBaseUrl}/business/cmdb/hosts/${id}`,{headers:cleanupHeaders}));
  96  |    for(const [id,endpoint] of [[scopeId,'bizscope'],[packageId,'deploy/packages']] as const){if(id)await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`,{headers:cleanupHeaders}));}
  97  |   }catch(error){errors.push(String(error))}
  98  |   await testInfo.attach('batch-fixture-cleanup',{contentType:'application/json',body:JSON.stringify({token,taskIds,hostId,scopeId,packageId,errors})});if(!failed)expect(errors).toEqual([]);
  99  |  }
  100 | });
  101 |
```
