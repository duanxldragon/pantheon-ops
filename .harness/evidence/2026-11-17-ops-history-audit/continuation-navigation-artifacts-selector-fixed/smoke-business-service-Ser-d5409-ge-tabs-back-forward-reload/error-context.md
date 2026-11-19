# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\ServiceCatalog.nav.spec.ts >> Business navigation repair >> ServiceCatalog sidebar 3 items, page tabs, back/forward, reload
- Location: tests\smoke\business\service\ServiceCatalog.nav.spec.ts:275:3

# Error details

```
TimeoutError: locator.click: Timeout 8000ms exceeded.
Call log:
  - waiting for locator('.app-shell__sider').locator('.arco-menu-sub').filter({ has: locator('.app-shell__menu-entry--level-0').getByText('服务目录', { exact: true }) }).locator('.app-shell__menu-entry--level-1:visible').getByText('应用', { exact: true })

```

# Page snapshot

```yaml
- generic [ref=f1e3]:
    - complementary [ref=f1e4]:
        - generic [ref=f1e5]:
            - generic [ref=f1e6]:
                - generic [ref=f1e7]: P
                - generic [ref=f1e8]: Pantheon Base
            - menu [ref=f1e12]:
                - generic [ref=f1e13]:
                    - menuitem [ref=f1e14] [cursor=pointer]:
                        - button "工作台" [ref=f1e15]
                    - button "CMDB" [ref=f1e25] [cursor=pointer]
                    - button "业务域" [ref=f1e40] [cursor=pointer]
                    - generic [ref=f1e52]:
                        - button "服务目录" [active] [ref=f1e55] [cursor=pointer]
                        - generic [ref=f1e67]:
                            - menuitem [ref=f1e68] [cursor=pointer]:
                                - button "应用" [ref=f1e71]
                            - menuitem [ref=f1e80] [cursor=pointer]:
                                - button "服务" [ref=f1e83]
                            - menuitem [ref=f1e92] [cursor=pointer]:
                                - button "服务实例" [ref=f1e95]
                    - button "安装部署" [ref=f1e107] [cursor=pointer]
                    - menuitem [ref=f1e116] [cursor=pointer]:
                        - button "文件分发" [ref=f1e117]
                    - button "Kubernetes" [ref=f1e129] [cursor=pointer]
                    - menuitem [ref=f1e138] [cursor=pointer]:
                        - button "ArgoCD 应用交付" [ref=f1e139]
                    - button "可观测中心" [ref=f1e148] [cursor=pointer]
                    - button "访问控制" [ref=f1e160] [cursor=pointer]
                    - button "组织架构" [ref=f1e174] [cursor=pointer]
                    - button "安全审计" [ref=f1e189] [cursor=pointer]
                    - button "低代码平台" [ref=f1e201] [cursor=pointer]
                    - button "平台配置" [ref=f1e213] [cursor=pointer]
    - generic [ref=f1e222]:
        - generic [ref=f1e223]:
            - generic [ref=f1e224]:
                - button "切换导航" [ref=f1e225] [cursor=pointer]
                - list [ref=f1e229]:
                    - listitem [ref=f1e230]: 首页
                    - listitem [ref=f1e234]: 服务目录
            - generic [ref=f1e235]:
                - button "平台功能台" [ref=f1e237]:
                    - generic [ref=f1e240]: 搜索菜单、页面、功能…
                    - generic [ref=f1e241]: Ctrl K
                - button "通知中心" [ref=f1e243] [cursor=pointer]
                - button "平台偏好" [ref=f1e247] [cursor=pointer]
                - 'button "用户: Administrator" [ref=f1e252] [cursor=pointer]':
                    - generic [ref=f1e253]: A
                    - generic [ref=f1e255]:
                        - generic [ref=f1e256]: Administrator
                        - generic [ref=f1e257]: admin
        - tablist "已打开页面" [ref=f1e258]:
            - tab "工作台" [ref=f1e259] [cursor=pointer]
            - tab "服务目录" [selected] [ref=f1e263] [cursor=pointer]
        - main [ref=f1e265]:
            - generic [ref=f1e268]:
                - generic [ref=f1e270]:
                    - generic [ref=f1e271]:
                        - generic [ref=f1e272]: 运维平台 / 服务目录
                        - generic [ref=f1e273]: 服务目录
                    - generic [ref=f1e275]:
                        - generic [ref=f1e276]:
                            - generic [ref=f1e277]: 应用
                            - strong [ref=f1e278]: "0"
                        - generic [ref=f1e279]:
                            - generic [ref=f1e280]: 服务
                            - strong [ref=f1e281]: "0"
                        - generic [ref=f1e282]:
                            - generic [ref=f1e283]: 服务实例
                            - strong [ref=f1e284]: "0"
                        - generic [ref=f1e285]:
                            - generic [ref=f1e286]: 健康实例
                            - strong [ref=f1e287]: "0"
                - generic [ref=f1e289]:
                    - generic [ref=f1e293]:
                        - tab "应用" [selected] [ref=f1e294] [cursor=pointer]
                        - tab "服务" [ref=f1e296] [cursor=pointer]
                        - tab "服务实例" [ref=f1e298] [cursor=pointer]
                    - tabpanel "应用" [ref=f1e303]:
                        - generic [ref=f1e304]:
                            - search "筛选" [ref=f1e305]:
                                - button "新增应用" [ref=f1e309] [cursor=pointer]
                            - generic [ref=f1e312]: 暂无应用。
        - generic [ref=f1e321]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  206 |
  207 |     const tree = await businessTree(page);
  208 |     const roots = tree.filter((node) => node.path?.startsWith('/business/'));
  209 |     const ordinal = (p: string) => BACKEND_SEED_ORDER.indexOf(p);
  210 |     const ordered = [...roots].sort((a, b) => {
  211 |       const diff = ordinal(a.path ?? '') - ordinal(b.path ?? '');
  212 |       return diff !== 0 ? diff : a.id - b.id;
  213 |     });
  214 |     const expectedRootLabels = ordered.map((node) => menuLabel(node.titleKey));
  215 |
  216 |     // The business run starts at the CMDB root (a workbench root precedes it, so it
  217 |     // is not at index 0) and must be the exact, contiguous 8-label seed-order slice.
  218 |     const bizStart = rootLabels.indexOf(expectedRootLabels[0]);
  219 |     expect(bizStart, 'business block should begin at the CMDB root').toBeGreaterThanOrEqual(0);
  220 |     expect(rootLabels.slice(bizStart, bizStart + expectedRootLabels.length)).toEqual(expectedRootLabels);
  221 |
  222 |     // Nothing business-shaped recurs after the block, so system governance follows.
  223 |     const afterBiz = rootLabels.slice(bizStart + expectedRootLabels.length);
  224 |     for (const label of Array.from(new Set(expectedRootLabels))) {
  225 |       expect(afterBiz, `business root ${label} must not recur after the contiguous block`).not.toContain(label);
  226 |     }
  227 |   });
  228 |
  229 |   test('every business parent/leaf entry loads the expected main content', async ({ page }) => {
  230 |     test.setTimeout(180000);
  231 |     await signInWithUi(page, adminCredentials);
  232 |     const tree = await businessTree(page);
  233 |     const businessRoots = tree.filter((node) => node.path?.startsWith('/business/'));
  234 |     // Index live rows by path and record each node's parent path so the walk can
  235 |     // rebuild its ancestor chain top-down for a given route.
  236 |     const byPath = new Map<string, MenuNodeRow>();
  237 |     const parentPath = new Map<string, string | null>();
  238 |     const flatten = (nodes: MenuNodeRow[], parent: string | null) => {
  239 |       for (const node of nodes) {
  240 |         const route = node.path ?? '';
  241 |         byPath.set(route, node);
  242 |         parentPath.set(route, parent);
  243 |         flatten(node.children ?? [], route);
  244 |       }
  245 |     };
  246 |     flatten(businessRoots, null);
  247 |
  248 |     // Iterate the 30-stop plan and require every seed path to resolve to a live row
  249 |     // and be navigated — a missing seed is a hard failure, never a silent skip.
  250 |     for (const [route, body] of BUSINESS_ENTRIES) {
  251 |       const node = byPath.get(route);
  252 |       if (!node) {
  253 |         throw new Error(`seed menu row ${route} is missing from the live menu tree`);
  254 |       }
  255 |       if (!node.titleKey) {
  256 |         throw new Error(`seed menu row ${route} lacks a titleKey`);
  257 |       }
  258 |       const ancestors: MenuNodeRow[] = [];
  259 |       let cursor = parentPath.get(route);
  260 |       while (cursor) {
  261 |         const parent = byPath.get(cursor);
  262 |         if (parent) ancestors.unshift(parent);
  263 |         cursor = parentPath.get(cursor);
  264 |       }
  265 |       await test.step(`click ${route}`, async () => {
  266 |         try {
  267 |           await walkEntry(page, node, ancestors, businessRoots, body);
  268 |         } catch (error) {
  269 |           throw new Error(`Failed navigating to ${route}: ${(error as Error).message}`, { cause: error });
  270 |         }
  271 |       });
  272 |     }
  273 |   });
  274 |
  275 |   test('ServiceCatalog sidebar 3 items, page tabs, back/forward, reload', async ({ page }) => {
  276 |     const badNav: string[] = [];
  277 |     page.on('response', (resp) => {
  278 |       if (resp.status() === 404 && resp.request().url().includes('/business/service')) badNav.push(resp.url());
  279 |     });
  280 |
  281 |     test.setTimeout(90000);
  282 |     await signInWithUi(page, adminCredentials);
  283 |     // Sidebar shows the service catalog group with its three direct children.
  284 |     await expect(sider(page).getByText('服务目录', { exact: true })).toHaveCount(1);
  285 |     await sider(page).getByText('服务目录', { exact: true }).first().click({ timeout: 8000 });
  286 |
  287 |     const catalogCases = [
  288 |       { key: 'applications', label: '应用', button: '新增应用' },
  289 |       { key: 'services', label: '服务', button: '新增服务' },
  290 |       { key: 'instances', label: '服务实例', button: '新增服务实例' },
  291 |     ] as const;
  292 |
  293 |     // Locators scoped to MAIN so top-level header tabs with the same name can't match.
  294 |     const assertOn = async (item: (typeof catalogCases)[number]) => {
  295 |       await expect(page).toHaveURL(new RegExp(`/business/service-catalog/${item.key}$`));
  296 |       const main = page.locator('main');
  297 |       await expect(main.getByRole('button', { name: item.button, exact: true })).toBeVisible({ timeout: 15000 });
  298 |       await expect(main.getByRole('tab', { name: item.label, exact: true })).toHaveAttribute('aria-selected', 'true');
  299 |     };
  300 |
  301 |     // Sidebar: really clicking each of the three level-1 children loads its own page
  302 |     // and shows the matching 新增 button. Scoped to the 服务目录 ancestor so the
  303 |     // Kubernetes 服务 child cannot collide in strict mode.
  304 |     for (const item of catalogCases) {
  305 |       await test.step(`sidebar click ${item.key}`, async () => {
> 306 |         await (await serviceCatalogChild(page, item.label)).click({ timeout: 8000 });
      |                                                             ^ TimeoutError: locator.click: Timeout 8000ms exceeded.
  307 |         await assertOn(item);
  308 |       });
  309 |     }
  310 |
  311 |     // In-page tabs drive the URL and write it back for all three.
  312 |     for (const item of catalogCases) {
  313 |       await test.step(`page tab ${item.label}`, async () => {
  314 |         await page.locator('main').getByRole('tab', { name: item.label, exact: true }).click({ timeout: 8000 });
  315 |         await assertOn(item);
  316 |       });
  317 |     }
  318 |
  319 |     // Browser back / forward stay consistent with the URL-driven active tab: the tab
  320 |     // loop above ends on 服务实例, so back lands on 服务 and forward re-enters 服务实例.
  321 |     await page.goBack();
  322 |     await assertOn(catalogCases[1]);
  323 |     await page.goForward();
  324 |     await assertOn(catalogCases[2]);
  325 |
  326 |     // Deep-link + real reload keeps the URL-driven tab for all three.
  327 |     for (const item of catalogCases) {
  328 |       await test.step(`deep link + reload ${item.key}`, async () => {
  329 |         await page.goto(`/business/service-catalog/${item.key}`, { waitUntil: 'domcontentloaded' });
  330 |         await assertOn(item);
  331 |         await page.reload({ waitUntil: 'domcontentloaded' });
  332 |         await assertOn(item);
  333 |       });
  334 |     }
  335 |
  336 |     expect(badNav).toEqual([]);
  337 |   });
  338 | });
  339 |
```
