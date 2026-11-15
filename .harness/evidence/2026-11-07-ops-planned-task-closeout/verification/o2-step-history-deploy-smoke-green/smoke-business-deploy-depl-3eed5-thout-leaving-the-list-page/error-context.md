# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy package detail action opens a modal without leaving the list page
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:207:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('row').filter({ hasText: 'deploy-package-modal-1794107352952' }).first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByRole('row').filter({ hasText: 'deploy-package-modal-1794107352952' }).first() with timeout 10000ms
  - waiting for getByRole('row').filter({ hasText: 'deploy-package-modal-1794107352952' }).first()

```

```yaml
- complementary:
    - text: P Pantheon Base
    - menu
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 软件组件
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
  132 |         headers,
  133 |         data: {
  134 |           hostname: `${token}-host`,
  135 |           ip: `10.248.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  136 |           os: 'linux',
  137 |           businessScopeId: scope.id,
  138 |           labels: [{ key: 'biz', val: token }],
  139 |         },
  140 |       }),
  141 |     );
  142 |
  143 |     const pkg = await expectSuccess<PackageRow>(
  144 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  145 |         headers,
  146 |         data: {
  147 |           name: token,
  148 |           version: '1.30.2',
  149 |           installCommand: 'echo install',
  150 |           status: 'enabled',
  151 |         },
  152 |       }),
  153 |     );
  154 |
  155 |     const task = await expectSuccess<TaskRow>(
  156 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  157 |         headers,
  158 |         data: {
  159 |           name: `${token}-task`,
  160 |           packageId: pkg.id,
  161 |           businessScopeId: scope.id,
  162 |           targetType: 'host',
  163 |           targetIds: [host.id],
  164 |           executorType: 'manual',
  165 |         },
  166 |       }),
  167 |     );
  168 |
  169 |     try {
  170 |       const started = await expectSuccess<TaskRow>(
  171 |         await startReviewedTask(request, task.id, { headers }),
  172 |       );
  173 |       expect(started.hosts).toHaveLength(1);
  174 |
  175 |       await expectSuccess(
  176 |         await request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
  177 |           headers,
  178 |           data: { status: 'success', stdout: 'installed via modal smoke' },
  179 |         }),
  180 |       );
  181 |
  182 |       await page.goto('/business/deploy/task', { waitUntil: 'networkidle' });
  183 |       await expect(page).toHaveURL(/\/business\/deploy\/task$/);
  184 |       const taskKeywordInput = page.locator('.search-toolbar input').first();
  185 |       await expect(taskKeywordInput).toBeVisible();
  186 |       await taskKeywordInput.fill(token);
  187 |
  188 |       const row = page.getByRole('row').filter({ hasText: `${token}-task` }).first();
  189 |       await expect(row).toBeVisible();
  190 |       await row.getByRole('button', { name: '详情' }).click();
  191 |
  192 |       const modal = page.locator('.arco-modal:visible').last();
  193 |       await expect(modal).toBeVisible();
  194 |       await expect(modal).toContainText(`${token}-task`);
  195 |       await expect(modal).toContainText(token);
  196 |       await expect(modal).toContainText(`${token}-host`);
  197 |       await expect(modal).toContainText('installed via modal smoke');
  198 |       await expect(page).toHaveURL(/\/business\/deploy\/task$/);
  199 |       await page.screenshot({ path: testInfo.outputPath('deploy-task-detail-modal.png'), fullPage: true });
  200 |     } finally {
  201 |       await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers }).catch(() => undefined);
  202 |       await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
  203 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  204 |     }
  205 |   });
  206 |
  207 |   test('deploy package detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
  208 |     const login = await loginByApi(request, adminCredentials);
  209 |     await installClientSession(page, login);
  210 |     const headers = apiRequestHeaders(login);
  211 |     const token = `deploy-package-modal-${Date.now()}`;
  212 |
  213 |     const pkg = await expectSuccess<PackageRow>(
  214 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  215 |         headers,
  216 |         data: {
  217 |           name: token,
  218 |           version: '1.30.2',
  219 |           executionMode: 'fixed',
  220 |           templateCode: 'nginx_systemd',
  221 |           sourceFileName: 'nginx-1.30.2.tar.gz',
  222 |           sourceUrl: `https://example.invalid/${token}.tar.gz`,
  223 |           status: 'enabled',
  224 |         },
  225 |       }),
  226 |     );
  227 |
  228 |     try {
  229 |       await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
  230 |       await expect(page).toHaveURL(/\/business\/deploy\/package$/);
  231 |       const row = page.getByRole('row').filter({ hasText: token }).first();
> 232 |       await expect(row).toBeVisible();
      |                         ^ Error: expect(locator).toBeVisible() failed
  233 |       await row.getByRole('button', { name: '详情' }).click();
  234 |
  235 |       const modal = page.locator('.arco-modal:visible').last();
  236 |       await expect(modal).toBeVisible();
  237 |       await expect(modal).toContainText(token);
  238 |       await expect(modal).toContainText('nginx-1.30.2.tar.gz');
  239 |       await expect(modal).toContainText('模板说明');
  240 |       await expect(page).toHaveURL(/\/business\/deploy\/package$/);
  241 |       await page.screenshot({ path: testInfo.outputPath('deploy-package-detail-modal.png'), fullPage: true });
  242 |     } finally {
  243 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  244 |     }
  245 |   });
  246 |
  247 |   test('deploy template detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
  248 |     const login = await loginByApi(request, adminCredentials);
  249 |     await installClientSession(page, login);
  250 |     const headers = apiRequestHeaders(login);
  251 |     const token = `deploy-template-modal-${Date.now()}`;
  252 |
  253 |     const pkg = await expectSuccess<PackageRow>(
  254 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  255 |         headers,
  256 |         data: {
  257 |           name: `${token}-pkg`,
  258 |           version: '1.30.2',
  259 |           executionMode: 'fixed',
  260 |           templateCode: 'nginx_systemd',
  261 |           sourceFileName: 'nginx-1.30.2.tar.gz',
  262 |           status: 'enabled',
  263 |         },
  264 |       }),
  265 |     );
  266 |
  267 |     const template = await expectSuccess<TemplateRow>(
  268 |       await request.post(`${apiBaseUrl}/business/deploy/templates`, {
  269 |         headers,
  270 |         data: {
  271 |           name: token,
  272 |           version: 'v1',
  273 |           packageId: pkg.id,
  274 |           executionMode: 'orchestrated',
  275 |           defaultAction: 'install',
  276 |           status: 'enabled',
  277 |           parameterSchema: {
  278 |             installRoot: '/data/nginx',
  279 |           },
  280 |           steps: [
  281 |             {
  282 |               stepCode: 'prepare',
  283 |               stepName: '准备目录',
  284 |               stepType: 'script',
  285 |               action: 'install',
  286 |               stepConfig: {
  287 |                 script: 'mkdir -p /data/nginx',
  288 |               },
  289 |               sort: 1,
  290 |             },
  291 |             {
  292 |               stepCode: 'install',
  293 |               stepName: '安装组件',
  294 |               stepType: 'package',
  295 |               action: 'install',
  296 |               packageId: pkg.id,
  297 |               sort: 2,
  298 |             },
  299 |           ],
  300 |         },
  301 |       }),
  302 |     );
  303 |
  304 |     try {
  305 |       await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  306 |       await expect(page).toHaveURL(/\/business\/deploy\/template$/);
  307 |       const templateKeywordInput = page.locator('.search-toolbar input').first();
  308 |       await expect(templateKeywordInput).toBeVisible();
  309 |       await templateKeywordInput.fill(token);
  310 |
  311 |       const row = page.getByRole('row').filter({ hasText: token }).first();
  312 |       await expect(row).toBeVisible();
  313 |       await row.getByRole('button', { name: '详情' }).click();
  314 |
  315 |       const modal = page.locator('.arco-modal:visible').last();
  316 |       await expect(modal).toBeVisible();
  317 |       await expect(modal).toContainText(token);
  318 |       await expect(modal).toContainText('准备目录');
  319 |       await expect(modal).toContainText('安装组件');
  320 |       await expect(modal).toContainText('/data/nginx');
  321 |       await expect(page).toHaveURL(/\/business\/deploy\/template$/);
  322 |       await page.screenshot({ path: testInfo.outputPath('deploy-template-detail-modal.png'), fullPage: true });
  323 |     } finally {
  324 |       await request.delete(`${apiBaseUrl}/business/deploy/templates/${template.id}`, { headers }).catch(() => undefined);
  325 |       await request.delete(`${apiBaseUrl}/business/deploy/packages/${pkg.id}`, { headers }).catch(() => undefined);
  326 |     }
  327 |   });
  328 |
  329 |   test('deploy detail page loads with summary and host table', async ({ page, request }, testInfo) => {
  330 |     const login = await loginByApi(request, adminCredentials);
  331 |     await installClientSession(page, login);
  332 |     const headers = apiRequestHeaders(login);
```
