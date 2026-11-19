# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\k8s\k8s-pages.spec.ts >> C04 Kubernetes Service UI contract >> stale edit shows one translated error and keeps draft and original version
- Location: tests\smoke\business\k8s\k8s-pages.spec.ts:336:3

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  locator('.arco-message').filter({ hasText: '服务已变化，请刷新列表并重新核对后操作。' })
Expected: 1
Received: 0
Timeout:  10000ms

Call log:
  - Expect "toHaveCount" locator('.arco-message').filter({ hasText: '服务已变化，请刷新列表并重新核对后操作。' }) with timeout 10000ms
  - waiting for locator('.arco-message').filter({ hasText: '服务已变化，请刷新列表并重新核对后操作。' })
    10 × locator resolved to 2 elements
       - unexpected value "2"
    13 × locator resolved to 0 elements
       - unexpected value "0"

```

# Page snapshot

```yaml
- generic [ref=e1]:
    - generic [ref=e3]:
        - complementary [ref=e4]:
            - generic [ref=e5]:
                - generic [ref=e6]:
                    - generic [ref=e7]: P
                    - generic [ref=e8]: Pantheon Base
                - menu [ref=e12]:
                    - generic [ref=e13]:
                        - menuitem [ref=e14] [cursor=pointer]:
                            - button "工作台" [ref=e15]
                        - button "CMDB" [ref=e25] [cursor=pointer]
                        - button "业务域" [ref=e40] [cursor=pointer]
                        - button "服务目录" [ref=e55] [cursor=pointer]
                        - button "安装部署" [ref=e70] [cursor=pointer]
                        - menuitem [ref=e79] [cursor=pointer]:
                            - button "文件分发" [ref=e80]
                        - generic [ref=e89]:
                            - button "Kubernetes" [ref=e92] [cursor=pointer]
                            - generic [ref=e101]:
                                - menuitem [ref=e102] [cursor=pointer]:
                                    - button "K8s 集群" [ref=e105]
                                - menuitem [ref=e114] [cursor=pointer]:
                                    - button "工作负载" [ref=e117]
                                - menuitem [ref=e126] [cursor=pointer]:
                                    - button "服务" [ref=e129]
                                - menuitem [ref=e135] [cursor=pointer]:
                                    - button "应用发布" [ref=e138]
                                - menuitem [ref=e144] [cursor=pointer]:
                                    - button "Ingress" [ref=e147]
                                - menuitem [ref=e153] [cursor=pointer]:
                                    - button "存储管理" [ref=e156]
                                - menuitem [ref=e165] [cursor=pointer]:
                                    - button "Velero 备份" [ref=e168]
                        - menuitem [ref=e174] [cursor=pointer]:
                            - button "ArgoCD 应用交付" [ref=e175]
                        - button "可观测中心" [ref=e184] [cursor=pointer]
                        - button "访问控制" [ref=e196] [cursor=pointer]
                        - button "组织架构" [ref=e210] [cursor=pointer]
                        - button "安全审计" [ref=e225] [cursor=pointer]
                        - button "低代码平台" [ref=e237] [cursor=pointer]
                        - button "平台配置" [ref=e249] [cursor=pointer]
        - generic [ref=e258]:
            - generic [ref=e259]:
                - generic [ref=e260]:
                    - button "切换导航" [ref=e261] [cursor=pointer]
                    - list [ref=e265]:
                        - listitem [ref=e266]: 首页
                        - listitem [ref=e270]: Kubernetes
                        - listitem [ref=e274]: 服务
                - generic [ref=e275]:
                    - button "平台功能台" [ref=e277]:
                        - generic [ref=e280]: 搜索菜单、页面、功能…
                        - generic [ref=e281]: Ctrl K
                    - button "通知中心" [ref=e283] [cursor=pointer]
                    - button "平台偏好" [ref=e287] [cursor=pointer]
                    - 'button "用户: Administrator" [ref=e292] [cursor=pointer]':
                        - generic [ref=e293]: A
                        - generic [ref=e295]:
                            - generic [ref=e296]: Administrator
                            - generic [ref=e297]: admin
            - tablist "已打开页面" [ref=e298]:
                - tab "工作台" [ref=e299] [cursor=pointer]
                - tab "服务" [selected] [ref=e303] [cursor=pointer]
            - main [ref=e305]:
                - generic [ref=e308]:
                    - generic [ref=e310]:
                        - generic [ref=e311]:
                            - generic [ref=e312]: Kubernetes / 服务
                            - generic [ref=e313]: 服务
                        - generic [ref=e315]:
                            - generic [ref=e316]:
                                - generic [ref=e317]: 服务总数
                                - strong [ref=e318]: "1"
                            - generic [ref=e319]:
                                - generic [ref=e320]: 负载均衡服务
                                - strong [ref=e321]: "0"
                    - search "筛选" [ref=e323]:
                        - generic [ref=e325]:
                            - textbox "名称" [ref=e332]
                            - generic [ref=e333]:
                                - combobox [disabled] [ref=e334]:
                                    - generic "c04-cluster" [ref=e335]:
                                        - generic [ref=e336]:
                                            - textbox [disabled] [aria-hidden]:
                                                - /placeholder: 集群
                                            - generic [ref=e337]: c04-cluster
                                - textbox "命名空间" [ref=e343]
                    - generic [ref=e345]:
                        - button "刷新" [disabled] [ref=e348]
                        - button "新增服务" [ref=e353] [cursor=pointer]
                    - table [ref=e366]:
                        - rowgroup [ref=e375]:
                            - row [ref=e376]:
                                - columnheader "名称" [ref=e377]
                                - columnheader "命名空间" [ref=e379]
                                - columnheader "类型" [ref=e381]
                                - columnheader "Cluster IP" [ref=e383]
                                - columnheader "端口" [ref=e385]
                                - columnheader "选择器" [ref=e387]
                                - columnheader "操作" [ref=e389]
                        - rowgroup [ref=e391]:
                            - row [ref=e392]:
                                - cell "c04-routing" [ref=e393]
                                - cell "prod" [ref=e395]
                                - cell "NodePort" [ref=e397]
                                - cell "10.96.0.42" [ref=e402]
                                - cell "80:http-web/TCP, 9090:9091/TCP" [ref=e404]
                                - cell "app=web, tier=" [ref=e406]
                                - cell [ref=e408]:
                                    - generic [ref=e411]:
                                        - button "编辑" [ref=e413] [cursor=pointer]
                                        - button "删除" [ref=e417] [cursor=pointer]
            - generic [ref=e420]: Pantheon Platform · Enterprise Admin Foundation
    - dialog [ref=e423]:
        - generic [ref=e424]:
            - generic [ref=e425]: 编辑服务路由
            - generic [ref=e428]:
                - generic [ref=e429]: 修改类型、端口和选择器；其他配置保持原样。若服务已变化，请关闭后刷新列表并重新编辑。
                - generic [ref=e430]:
                    - generic [ref=e431]:
                        - generic [ref=e432]: 名称
                        - textbox "名称" [disabled] [ref=e433]: c04-routing
                    - generic [ref=e434]:
                        - generic [ref=e435]: 命名空间
                        - textbox "命名空间" [disabled] [ref=e436]: prod
                - generic [ref=e437]:
                    - generic [ref=e438]: 类型
                    - combobox "类型" [ref=e439] [cursor=pointer]:
                        - generic "NodePort" [ref=e440]:
                            - generic [ref=e441]:
                                - textbox [aria-hidden]
                                - generic [ref=e442]: NodePort
                - generic [ref=e447]:
                    - generic [ref=e448]: 选择器
                    - textbox "选择器" [ref=e449]:
                        - /placeholder: app=example, component=api
                        - text: app=user-draft
                    - generic [ref=e450]: 使用逗号分隔 key=value；留空表示不使用选择器。
                - generic [ref=e451]:
                    - generic [ref=e452]: 端口
                    - generic [ref=e453]: 目标端口可填写数字或容器端口名称，留空则使用服务端口。多个端口必须命名；节点端口留空可由集群分配。
                    - group "端口 1" [ref=e454]:
                        - generic [ref=e456]:
                            - generic [ref=e457]:
                                - generic [ref=e458]: 端口名称
                                - textbox "端口名称" [ref=e459]: http
                            - generic [ref=e460]:
                                - generic [ref=e461]: 服务端口
                                - generic [ref=e464]:
                                    - spinbutton "服务端口" [ref=e465]: "80"
                                    - generic [ref=e466]:
                                        - generic [ref=e467] [cursor=pointer]
                                        - generic [ref=e470] [cursor=pointer]
                            - generic [ref=e473]:
                                - generic [ref=e474]: 目标端口
                                - textbox "目标端口" [ref=e475]: http-next
                            - generic [ref=e476]:
                                - generic [ref=e477]: 协议
                                - combobox "协议" [ref=e478] [cursor=pointer]:
                                    - generic "TCP" [ref=e479]:
                                        - generic [ref=e480]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e481]: TCP
                            - generic [ref=e486]:
                                - generic [ref=e487]: 节点端口
                                - generic [ref=e490]:
                                    - spinbutton "节点端口" [ref=e491]: "30080"
                                    - generic [ref=e492]:
                                        - generic [ref=e493] [cursor=pointer]
                                        - generic [ref=e496] [cursor=pointer]
                        - button "移除端口 1" [ref=e499] [cursor=pointer]: 移除端口
                    - group "端口 2" [ref=e501]:
                        - generic [ref=e503]:
                            - generic [ref=e504]:
                                - generic [ref=e505]: 端口名称
                                - textbox "端口名称" [ref=e506]: metrics
                            - generic [ref=e507]:
                                - generic [ref=e508]: 服务端口
                                - generic [ref=e511]:
                                    - spinbutton "服务端口" [ref=e512]: "9090"
                                    - generic [ref=e513]:
                                        - generic [ref=e514] [cursor=pointer]
                                        - generic [ref=e517] [cursor=pointer]
                            - generic [ref=e520]:
                                - generic [ref=e521]: 目标端口
                                - textbox "目标端口" [ref=e522]: "9091"
                            - generic [ref=e523]:
                                - generic [ref=e524]: 协议
                                - combobox "协议" [ref=e525] [cursor=pointer]:
                                    - generic "TCP" [ref=e526]:
                                        - generic [ref=e527]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e528]: TCP
                            - generic [ref=e533]:
                                - generic [ref=e534]: 节点端口
                                - generic [ref=e537]:
                                    - spinbutton "节点端口" [ref=e538]: "30090"
                                    - generic [ref=e539]:
                                        - generic [ref=e540] [cursor=pointer]
                                        - generic [ref=e543] [cursor=pointer]
                        - button "移除端口 2" [ref=e546] [cursor=pointer]: 移除端口
                    - button "添加端口" [ref=e548] [cursor=pointer]
                - generic [ref=e551]:
                    - generic [ref=e552]: 资源版本
                    - code [ref=e555]: "27"
            - generic [ref=e556]:
                - button "取消" [ref=e557] [cursor=pointer]
                - button "确定" [active] [ref=e558] [cursor=pointer]
            - button "Close" [ref=e559] [cursor=pointer]
```

# Test source

```ts
  63  |       .toBe(true);
  64  |     await page.screenshot({ path: testInfo.outputPath('k8s-service-mobile.png') });
  65  |   });
  66  | });
  67  |
  68  | // Business-data fixtures exercise the UI/HTTP contract. Real login remains in
  69  | // use; provider admission and live role acceptance belong to the runtime lane.
  70  | async function installServiceFixture(
  71  |   page: Page,
  72  |   options: {
  73  |     stale?: boolean;
  74  |     empty?: boolean;
  75  |     missingIdentity?: boolean;
  76  |     selectorless?: boolean;
  77  |     listGate?: Promise<void>;
  78  |     updateGate?: Promise<void>;
  79  |   } = {},
  80  | ) {
  81  |   const service: K8sServiceRow = {
  82  |     name: 'c04-routing',
  83  |     namespace: 'prod',
  84  |     type: 'NodePort',
  85  |     clusterIP: '10.96.0.42',
  86  |     uid: options.missingIdentity ? '' : 'c04-uid-original',
  87  |     resourceVersion: '27',
  88  |     selector: options.selectorless ? null : { app: 'web', tier: '' },
  89  |     ports: [
  90  |       { name: 'http', port: 80, targetPort: 'http-web', nodePort: 30080, protocol: 'TCP' },
  91  |       { name: 'metrics', port: 9090, targetPort: 9091, nodePort: 30090, protocol: 'TCP' },
  92  |     ],
  93  |   };
  94  |   let rows = options.empty ? [] : [service];
  95  |   const writes: Array<{ method: string; url: URL; body?: Record<string, unknown> }> = [];
  96  |   const reads: URL[] = [];
  97  |   await page.route(/\/api\/v1\/business\/k8s\/clusters(?:\?.*)?$/, (route) =>
  98  |     route.fulfill({
  99  |       json: { code: 200, data: { items: [{ id: 41, name: 'c04-cluster' }], total: 1 } },
  100 |     }),
  101 |   );
  102 |   await page.route(
  103 |     /\/api\/v1\/business\/k8s\/clusters\/41\/services(?:\/[^?]+)?(?:\?.*)?$/,
  104 |     async (route) => {
  105 |       const request = route.request();
  106 |       const url = new URL(request.url());
  107 |       if (request.method() === 'GET') {
  108 |         reads.push(url);
  109 |         if (options.listGate) await options.listGate;
  110 |         await route.fulfill({ json: { code: 200, data: { items: rows, total: rows.length } } });
  111 |         return;
  112 |       }
  113 |       const body =
  114 |         request.method() === 'DELETE'
  115 |           ? undefined
  116 |           : (request.postDataJSON() as Record<string, unknown>);
  117 |       writes.push({ method: request.method(), url, body });
  118 |       if (request.method() === 'PUT' && options.updateGate) await options.updateGate;
  119 |       if (options.stale) {
  120 |         await route.fulfill({
  121 |           status: 409,
  122 |           json: { code: 409, message: 'k8s.service.resource_version_conflict' },
  123 |         });
  124 |         return;
  125 |       }
  126 |       if (request.method() === 'DELETE') rows = [];
  127 |       else if (request.method() === 'PUT') Object.assign(service, body, { resourceVersion: '28' });
  128 |       else if (request.method() === 'POST') rows.push({ ...service, ...body } as K8sServiceRow);
  129 |       await route.fulfill({
  130 |         json: { code: 200, data: request.method() === 'DELETE' ? null : service },
  131 |       });
  132 |     },
  133 |   );
  134 |   return { service, reads, writes };
  135 | }
  136 |
  137 | async function openServiceEditor(page: Page) {
  138 |   await page.goto('/business/k8s/service', { waitUntil: 'domcontentloaded' });
  139 |   const row = page.locator('tbody tr').filter({ hasText: 'c04-routing' });
  140 |   await row.getByRole('button', { name: '编辑', exact: true }).click();
  141 |   const dialog = page.getByRole('dialog');
  142 |   await expect(dialog).toBeVisible();
  143 |   return dialog;
  144 | }
  145 |
  146 | async function captureServiceState(page: Page, testInfo: TestInfo, name: string) {
  147 |   await expect
  148 |     .poll(() => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
  149 |     .toBe(true);
  150 |   await expectNoVisibleK8sI18nKeys(page);
  151 |   expect(await page.locator('body').innerText()).not.toMatch(
  152 |     /\bk8s\.(service|namespace)\.[A-Za-z0-9_.-]+\b/,
  153 |   );
  154 |   const path = testInfo.outputPath(`${name}.png`);
  155 |   await page.screenshot({ path, animations: 'disabled' });
  156 |   await testInfo.attach(name, { path, contentType: 'image/png' });
  157 | }
  158 |
  159 | async function expectReadableServiceNotice(page: Page, testInfo: TestInfo, name: string) {
  160 |   const notice = page
  161 |     .locator('.arco-message')
  162 |     .filter({ hasText: '服务已变化，请刷新列表并重新核对后操作。' });
> 163 |   await expect(notice).toHaveCount(1);
      |                        ^ Error: expect(locator).toHaveCount(expected) failed
  164 |   await expect(page.locator('.arco-message-error')).toHaveCount(1);
  165 |   await expect(notice).toBeVisible();
  166 |   const measure = () =>
  167 |     notice.evaluate((node) => {
  168 |       const rect = node.getBoundingClientRect();
  169 |       const inset = Math.min(4, rect.width / 4, rect.height / 4);
  170 |       const points = [
  171 |         [rect.left + rect.width / 2, rect.top + rect.height / 2],
  172 |         [rect.left + inset, rect.top + inset],
  173 |         [rect.right - inset, rect.top + inset],
  174 |         [rect.left + inset, rect.bottom - inset],
  175 |         [rect.right - inset, rect.bottom - inset],
  176 |       ];
  177 |       const ancestors: Element[] = [];
  178 |       for (let current: Element | null = node; current; current = current.parentElement)
  179 |         ancestors.push(current);
  180 |       return {
  181 |         rect: {
  182 |           left: rect.left,
  183 |           top: rect.top,
  184 |           right: rect.right,
  185 |           bottom: rect.bottom,
  186 |           width: rect.width,
  187 |           height: rect.height,
  188 |         },
  189 |         insideViewport:
  190 |           rect.width > 0 &&
  191 |           rect.height > 0 &&
  192 |           rect.left >= 0 &&
  193 |           rect.top >= 0 &&
  194 |           rect.right <= globalThis.innerWidth &&
  195 |           rect.bottom <= globalThis.innerHeight,
  196 |         opaque: ancestors.every((element) => Number(getComputedStyle(element).opacity) === 1),
  197 |         animationSettled: [
  198 |           ...node.getAnimations({ subtree: true }),
  199 |           ...ancestors.flatMap((element) => element.getAnimations()),
  200 |         ].every((animation) => animation.playState !== 'running' && !animation.pending),
  201 |         contentFits:
  202 |           node.scrollWidth <= node.clientWidth + 1 && node.scrollHeight <= node.clientHeight + 1,
  203 |         points: points.map(([x, y]) => ({
  204 |           x,
  205 |           y,
  206 |           uncovered: document.elementFromPoint(x, y)?.closest('.arco-message') === node,
  207 |         })),
  208 |       };
  209 |     });
  210 |   await expect
  211 |     .poll(async () => {
  212 |       const result = await measure();
  213 |       return (
  214 |         result.insideViewport &&
  215 |         result.opaque &&
  216 |         result.animationSettled &&
  217 |         result.contentFits &&
  218 |         result.points.every((point) => point.uncovered)
  219 |       );
  220 |     })
  221 |     .toBe(true);
  222 |   await testInfo.attach(`${name}-notice-readability`, {
  223 |     body: JSON.stringify(await measure(), null, 2),
  224 |     contentType: 'application/json',
  225 |   });
  226 | }
  227 |
  228 | test.describe('C04 Kubernetes Service UI contract', () => {
  229 |   test.beforeEach(async ({ page }) => {
  230 |     const login = await loginByApi(page.request, adminCredentials);
  231 |     await installClientSession(page, login);
  232 |     await primeChineseLocale(page);
  233 |   });
  234 |
  235 |   test('editing preserves named target ports, allocated node ports and read identity', async ({
  236 |     page,
  237 |   }) => {
  238 |     const fixture = await installServiceFixture(page);
  239 |     const dialog = await openServiceEditor(page);
  240 |     await expect(dialog.getByLabel('名称', { exact: true })).toBeDisabled();
  241 |     await expect(dialog.getByLabel('命名空间', { exact: true })).toBeDisabled();
  242 |     await expect(dialog.getByLabel('目标端口', { exact: true }).first()).toHaveValue('http-web');
  243 |     await dialog.getByLabel('选择器', { exact: true }).fill('app=api, tier=');
  244 |     await dialog.getByRole('button', { name: /确定|确认/, exact: true }).click();
  245 |     await expect(dialog).toBeHidden();
  246 |     expect(fixture.writes).toHaveLength(1);
  247 |     const write = fixture.writes[0];
  248 |     expect(write.method).toBe('PUT');
  249 |     expect(write.url.pathname).toBe('/api/v1/business/k8s/clusters/41/services/c04-routing');
  250 |     expect(write.url.searchParams.get('namespace')).toBe('prod');
  251 |     expect(write.body).toEqual({
  252 |       type: 'NodePort',
  253 |       selector: { app: 'api', tier: '' },
  254 |       uid: 'c04-uid-original',
  255 |       resourceVersion: '27',
  256 |       ports: [
  257 |         { name: 'http', port: 80, targetPort: 'http-web', nodePort: 30080, protocol: 'TCP' },
  258 |         { name: 'metrics', port: 9090, targetPort: 9091, nodePort: 30090, protocol: 'TCP' },
  259 |       ],
  260 |     });
  261 |     expect(fixture.reads.every((url) => url.pathname.endsWith('/services'))).toBe(true);
  262 |   });
  263 |
```
