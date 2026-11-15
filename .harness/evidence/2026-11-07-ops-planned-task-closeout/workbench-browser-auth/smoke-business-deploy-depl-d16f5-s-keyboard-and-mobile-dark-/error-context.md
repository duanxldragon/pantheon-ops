# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (dark)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:14:3

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 200
Received: 403
```

# Test source

```ts
  136 |     }, [AUTH_USER_STORAGE_KEY, serializedUser] as const);
  137 |   }
  138 | }
  139 |
  140 | export async function installClientSession(page: Page, login: BrowserLoginResult) {
  141 |   await primeChineseLocale(page);
  142 |   await page.addInitScript((csrfToken) => {
  143 |     try {
  144 |       globalThis.localStorage?.setItem('pantheon_session_hint', '1');
  145 |       globalThis.localStorage?.setItem('pantheon_csrf_token', csrfToken);
  146 |     } catch {
  147 |       // about:blank and other opaque origins do not expose storage.
  148 |     }
  149 |   }, login.csrfToken);
  150 |   const appBaseUrl = new URL(
  151 |     page.url() === 'about:blank' ? '/' : page.url(),
  152 |     page.url() === 'about:blank' ? defaultWebBaseUrl : undefined,
  153 |   );
  154 |   const cookieUrl = appBaseUrl.origin;
  155 |   await page.context().addCookies([
  156 |     {
  157 |       name: 'pantheon_access_token',
  158 |       value: login.accessToken,
  159 |       url: cookieUrl,
  160 |       httpOnly: true,
  161 |       secure: false,
  162 |       sameSite: 'Strict',
  163 |     },
  164 |     {
  165 |       name: 'pantheon_refresh_token',
  166 |       value: login.refreshToken,
  167 |       url: cookieUrl,
  168 |       httpOnly: true,
  169 |       secure: false,
  170 |       sameSite: 'Strict',
  171 |     },
  172 |     {
  173 |       name: 'pantheon_csrf_token',
  174 |       value: login.csrfToken,
  175 |       url: cookieUrl,
  176 |       httpOnly: true,
  177 |       secure: false,
  178 |       sameSite: 'Strict',
  179 |     },
  180 |   ]);
  181 |   if (page.url() !== 'about:blank') {
  182 |     await page.evaluate((csrfToken) => {
  183 |       try {
  184 |         globalThis.localStorage?.setItem('pantheon_session_hint', '1');
  185 |         globalThis.localStorage?.setItem('pantheon_csrf_token', csrfToken);
  186 |       } catch {
  187 |         // Ignore opaque-origin storage failures for smoke setup.
  188 |       }
  189 |     }, login.csrfToken);
  190 |   }
  191 | }
  192 |
  193 | export async function getCsrfToken(page: Page) {
  194 |   const cookies = await page.context().cookies();
  195 |   const csrfCookie = cookies.find((cookie) => cookie.name === 'pantheon_csrf_token');
  196 |   expect(csrfCookie?.value).toBeTruthy();
  197 |   return csrfCookie!.value;
  198 | }
  199 |
  200 | export async function requestHeaders(page: Page, accessToken: string) {
  201 |   const csrfToken = await getCsrfToken(page);
  202 |   return {
  203 |     ...authHeaders(accessToken),
  204 |     'X-CSRF-Token': csrfToken,
  205 |     Cookie: `pantheon_csrf_token=${csrfToken}`,
  206 |   };
  207 | }
  208 |
  209 | export function apiRequestHeaders(login: BrowserLoginResult) {
  210 |   return {
  211 |     ...authHeaders(login.accessToken),
  212 |     'X-CSRF-Token': login.csrfToken,
  213 |     Cookie: `pantheon_csrf_token=${login.csrfToken}`,
  214 |   };
  215 | }
  216 |
  217 | export async function getOperationToken(page: Page, accessToken: string) {
  218 |   const response = await page.request.post(`${apiBaseUrl}/auth/operation-verify`, {
  219 |     headers: await requestHeaders(page, accessToken),
  220 |     data: { password: adminCredentials.password },
  221 |   });
  222 |   expect(response.ok()).toBeTruthy();
  223 |   const payload = await response.json();
  224 |   expect(payload.code).toBe(200);
  225 |   return payload.data.operationToken as string;
  226 | }
  227 |
  228 | export async function getApiOperationToken(requestLike: APIRequestContext | Page, login: BrowserLoginResult) {
  229 |   const request = resolveRequestContext(requestLike);
  230 |   const response = await request.post(`${apiBaseUrl}/auth/operation-verify`, {
  231 |     headers: apiRequestHeaders(login),
  232 |     data: { password: login.password },
  233 |   });
  234 |   expect(response.ok()).toBeTruthy();
  235 |   const payload = await response.json();
> 236 |   expect(payload.code).toBe(200);
      |                        ^ Error: expect(received).toBe(expected) // Object.is equality
  237 |   return payload.data.operationToken as string;
  238 | }
  239 |
  240 | export async function verifiedHeaders(page: Page, accessToken: string) {
  241 |   return {
  242 |     ...(await requestHeaders(page, accessToken)),
  243 |     'X-Operation-Token': await getOperationToken(page, accessToken),
  244 |   };
  245 | }
  246 |
  247 | export async function verifiedApiHeaders(requestLike: APIRequestContext | Page, login: BrowserLoginResult) {
  248 |   return {
  249 |     ...apiRequestHeaders(login),
  250 |     'X-Operation-Token': await getApiOperationToken(requestLike, login),
  251 |   };
  252 | }
  253 |
  254 | export async function installOperationToken(page: Page, accessToken: string) {
  255 |   const token = await getOperationToken(page, accessToken);
  256 |   await page.addInitScript((value) => {
  257 |     try {
  258 |       globalThis.sessionStorage?.setItem('pantheon_op_token', value);
  259 |     } catch {
  260 |       // about:blank and other opaque origins do not expose storage.
  261 |     }
  262 |   }, token);
  263 |   if (page.url() !== 'about:blank') {
  264 |     await page.evaluate((value) => {
  265 |       try {
  266 |         globalThis.sessionStorage?.setItem('pantheon_op_token', value);
  267 |       } catch {
  268 |         // Ignore opaque-origin storage failures for smoke setup.
  269 |       }
  270 |     }, token);
  271 |   }
  272 | }
  273 |
```
