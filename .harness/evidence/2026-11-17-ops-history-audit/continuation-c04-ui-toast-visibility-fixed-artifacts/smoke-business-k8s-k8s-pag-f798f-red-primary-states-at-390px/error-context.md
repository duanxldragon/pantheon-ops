# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\k8s\k8s-pages.spec.ts >> C04 Kubernetes Service UI contract >> rendered primary states at 390px
- Location: tests\smoke\business\k8s\k8s-pages.spec.ts:451:5

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
                - generic [ref=e6]: P
                - menu [ref=e10]:
                    - generic [ref=e11]:
                        - menuitem [ref=e12] [cursor=pointer]:
                            - button "工作台" [ref=e13]
                        - button "CMDB" [ref=e23] [cursor=pointer]
                        - button "业务域" [ref=e38] [cursor=pointer]
                        - button "服务目录" [ref=e53] [cursor=pointer]
                        - button "安装部署" [ref=e68] [cursor=pointer]
                        - menuitem [ref=e77] [cursor=pointer]:
                            - button "文件分发" [ref=e78]
                        - generic [ref=e87]:
                            - button "Kubernetes" [ref=e90] [cursor=pointer]
                            - generic [ref=e99]:
                                - menuitem [ref=e100] [cursor=pointer]:
                                    - button "K8s 集群" [ref=e103]
                                - menuitem [ref=e112] [cursor=pointer]:
                                    - button "工作负载" [ref=e115]
                                - menuitem [ref=e124] [cursor=pointer]:
                                    - button "服务" [ref=e127]
                                - menuitem [ref=e133] [cursor=pointer]:
                                    - button "应用发布" [ref=e136]
                                - menuitem [ref=e142] [cursor=pointer]:
                                    - button "Ingress" [ref=e145]
                                - menuitem [ref=e151] [cursor=pointer]:
                                    - button "存储管理" [ref=e154]
                                - menuitem [ref=e163] [cursor=pointer]:
                                    - button "Velero 备份" [ref=e166]
                        - menuitem [ref=e172] [cursor=pointer]:
                            - button "ArgoCD 应用交付" [ref=e173]
                        - button "可观测中心" [ref=e182] [cursor=pointer]
                        - button "访问控制" [ref=e194] [cursor=pointer]
                        - button "组织架构" [ref=e208] [cursor=pointer]
                        - button "安全审计" [ref=e223] [cursor=pointer]
                        - button "低代码平台" [ref=e235] [cursor=pointer]
                        - button "平台配置" [ref=e247] [cursor=pointer]
        - generic [ref=e256]:
            - generic [ref=e257]:
                - generic [ref=e258]:
                    - button "切换导航" [ref=e259] [cursor=pointer]
                    - list [ref=e263]:
                        - listitem [ref=e264]: 首页
                        - listitem [ref=e268]: Kubernetes
                        - listitem [ref=e272]: 服务
                - generic [ref=e273]:
                    - button "平台功能台" [ref=e275]:
                        - generic [ref=e278]: 搜索菜单、页面、功能…
                    - button "通知中心" [ref=e280] [cursor=pointer]
                    - button "平台偏好" [ref=e284] [cursor=pointer]
                    - 'button "用户: Administrator" [ref=e289] [cursor=pointer]':
                        - generic [ref=e290]: A
            - tablist "已打开页面" [ref=e292]:
                - tab "工作台" [ref=e293] [cursor=pointer]
                - tab "服务" [selected] [ref=e297] [cursor=pointer]
            - main [ref=e299]:
                - generic [ref=e302]:
                    - generic [ref=e304]:
                        - generic [ref=e305]:
                            - generic [ref=e306]: Kubernetes / 服务
                            - generic [ref=e307]: 服务
                        - generic [ref=e309]:
                            - generic [ref=e310]:
                                - generic [ref=e311]: 服务总数
                                - strong [ref=e312]: "1"
                            - generic [ref=e313]:
                                - generic [ref=e314]: 负载均衡服务
                                - strong [ref=e315]: "0"
                    - search "筛选" [ref=e317]:
                        - generic [ref=e319]:
                            - textbox "名称" [ref=e326]
                            - button "筛选" [ref=e327] [cursor=pointer]
                    - generic [ref=e332]:
                        - button "刷新" [disabled] [ref=e335]
                        - button "新增服务" [ref=e340] [cursor=pointer]
                    - generic [ref=e346]:
                        - generic [ref=e347]:
                            - generic [ref=e348]: 共 1 条
                            - generic [ref=e349]: 左右滑动查看更多字段
                        - table [ref=e356]:
                            - rowgroup [ref=e365]:
                                - row [ref=e366]:
                                    - columnheader "名称" [ref=e367]
                                    - columnheader "命名空间" [ref=e369]
                                    - columnheader "类型" [ref=e371]
                                    - columnheader "Cluster IP" [ref=e373]
                                    - columnheader "端口" [ref=e375]
                                    - columnheader "选择器" [ref=e377]
                                    - columnheader "操作" [ref=e379]
                            - rowgroup [ref=e381]:
                                - row [ref=e382]:
                                    - cell "c04-routing" [ref=e383]
                                    - cell "prod" [ref=e385]
                                    - cell "NodePort" [ref=e387]
                                    - cell "10.96.0.42" [ref=e392]
                                    - cell "80:http-web/TCP, 9090:9091/TCP" [ref=e394]
                                    - cell "app=web, tier=" [ref=e396]
                                    - cell [ref=e398]:
                                        - generic [ref=e401]:
                                            - button "编辑" [ref=e403] [cursor=pointer]
                                            - button "删除" [ref=e407] [cursor=pointer]
            - generic [ref=e410]: Pantheon Platform · Enterprise Admin Foundation
    - dialog [ref=e413]:
        - generic [ref=e414]:
            - generic [ref=e415]: 编辑服务路由
            - generic [ref=e418]:
                - generic [ref=e419]: 修改类型、端口和选择器；其他配置保持原样。若服务已变化，请关闭后刷新列表并重新编辑。
                - generic [ref=e420]:
                    - generic [ref=e421]:
                        - generic [ref=e422]: 名称
                        - textbox "名称" [disabled] [ref=e423]: c04-routing
                    - generic [ref=e424]:
                        - generic [ref=e425]: 命名空间
                        - textbox "命名空间" [disabled] [ref=e426]: prod
                - generic [ref=e427]:
                    - generic [ref=e428]: 类型
                    - combobox "类型" [ref=e429] [cursor=pointer]:
                        - generic "NodePort" [ref=e430]:
                            - generic [ref=e431]:
                                - textbox [aria-hidden]
                                - generic [ref=e432]: NodePort
                - generic [ref=e437]:
                    - generic [ref=e438]: 选择器
                    - textbox "选择器" [ref=e439]:
                        - /placeholder: app=example, component=api
                        - text: app=web, tier=
                    - generic [ref=e440]: 使用逗号分隔 key=value；留空表示不使用选择器。
                - generic [ref=e441]:
                    - generic [ref=e442]: 端口
                    - generic [ref=e443]: 目标端口可填写数字或容器端口名称，留空则使用服务端口。多个端口必须命名；节点端口留空可由集群分配。
                    - group "端口 1" [ref=e444]:
                        - generic [ref=e446]:
                            - generic [ref=e447]:
                                - generic [ref=e448]: 端口名称
                                - textbox "端口名称" [ref=e449]: http
                            - generic [ref=e450]:
                                - generic [ref=e451]: 服务端口
                                - generic [ref=e454]:
                                    - spinbutton "服务端口" [ref=e455]: "80"
                                    - generic [ref=e456]:
                                        - generic [ref=e457] [cursor=pointer]
                                        - generic [ref=e460] [cursor=pointer]
                            - generic [ref=e463]:
                                - generic [ref=e464]: 目标端口
                                - textbox "目标端口" [ref=e465]: http-next
                            - generic [ref=e466]:
                                - generic [ref=e467]: 协议
                                - combobox "协议" [ref=e468] [cursor=pointer]:
                                    - generic "TCP" [ref=e469]:
                                        - generic [ref=e470]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e471]: TCP
                            - generic [ref=e476]:
                                - generic [ref=e477]: 节点端口
                                - generic [ref=e480]:
                                    - spinbutton "节点端口" [ref=e481]: "30080"
                                    - generic [ref=e482]:
                                        - generic [ref=e483] [cursor=pointer]
                                        - generic [ref=e486] [cursor=pointer]
                        - button "移除端口 1" [ref=e489] [cursor=pointer]: 移除端口
                    - group "端口 2" [ref=e491]:
                        - generic [ref=e493]:
                            - generic [ref=e494]:
                                - generic [ref=e495]: 端口名称
                                - textbox "端口名称" [ref=e496]: metrics
                            - generic [ref=e497]:
                                - generic [ref=e498]: 服务端口
                                - generic [ref=e501]:
                                    - spinbutton "服务端口" [ref=e502]: "9090"
                                    - generic [ref=e503]:
                                        - generic [ref=e504] [cursor=pointer]
                                        - generic [ref=e507] [cursor=pointer]
                            - generic [ref=e510]:
                                - generic [ref=e511]: 目标端口
                                - textbox "目标端口" [ref=e512]: "9091"
                            - generic [ref=e513]:
                                - generic [ref=e514]: 协议
                                - combobox "协议" [ref=e515] [cursor=pointer]:
                                    - generic "TCP" [ref=e516]:
                                        - generic [ref=e517]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e518]: TCP
                            - generic [ref=e523]:
                                - generic [ref=e524]: 节点端口
                                - generic [ref=e527]:
                                    - spinbutton "节点端口" [ref=e528]: "30090"
                                    - generic [ref=e529]:
                                        - generic [ref=e530] [cursor=pointer]
                                        - generic [ref=e533] [cursor=pointer]
                        - button "移除端口 2" [ref=e536] [cursor=pointer]: 移除端口
                    - button "添加端口" [ref=e538] [cursor=pointer]
                - generic [ref=e541]:
                    - generic [ref=e542]: 资源版本
                    - code [ref=e545]: "27"
            - generic [ref=e546]:
                - button "取消" [ref=e547] [cursor=pointer]
                - button "确定" [active] [ref=e548] [cursor=pointer]
            - button "Close" [ref=e549] [cursor=pointer]
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
