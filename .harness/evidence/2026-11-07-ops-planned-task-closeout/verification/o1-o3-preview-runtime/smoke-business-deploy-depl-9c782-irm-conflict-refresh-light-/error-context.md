# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-selection-preview.spec.ts >> selection and preview confirm/conflict/refresh (light)
- Location: tests\smoke\business\deploy\deploy-selection-preview.spec.ts:11:2

# Error details

```
Error: apiRequestContext.post: connect ECONNREFUSED 127.0.0.1:24485
Call log:
  - → POST http://127.0.0.1:24485/api/v1/auth/login
    - user-agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.8078.4 Safari/537.36
    - accept: */*
    - accept-encoding: gzip,deflate,br
    - content-type: application/json
    - content-length: 40

```

# Test source

```ts
  1   | import { expect, type APIRequestContext, type Page } from '@playwright/test';
  2   | import { readAuthCookieSession } from '../../../scripts/lib/auth-cookie-session.mjs';
  3   | import { COOKIE_TOKEN_PLACEHOLDER } from '../../../src/core/auth/sessionSnapshot.ts';
  4   | import type { UserInfo } from '../../../src/modules/auth/security/types.ts';
  5   |
  6   | export const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8080/api/v1';
  7   | const defaultWebBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5173';
  8   | const AUTH_USER_STORAGE_KEY = 'pantheon_auth_user';
  9   |
  10  | export type BrowserLoginResult = {
  11  |   accessToken: string;
  12  |   refreshToken: string;
  13  |   username: string;
  14  |   password: string;
  15  |   csrfToken: string;
  16  | };
  17  |
  18  | export type LoginCredentials = {
  19  |   username: string;
  20  |   password: string;
  21  | };
  22  |
  23  | export const adminCredentials: LoginCredentials = {
  24  |   username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
  25  |   password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
  26  | };
  27  |
  28  | export async function primeChineseLocale(page: Page) {
  29  |   await page.addInitScript(() => {
  30  |     try {
  31  |       globalThis.localStorage?.setItem('pantheon_lang', 'zh-CN');
  32  |       globalThis.localStorage?.setItem('pantheon_lang_explicit', '1');
  33  |     } catch {
  34  |       // about:blank and other opaque origins do not expose storage.
  35  |     }
  36  |   });
  37  | }
  38  |
  39  | export function authHeaders(accessToken: string) {
  40  |   return {
  41  |     Authorization: `Bearer ${accessToken}`,
  42  |   };
  43  | }
  44  |
  45  | function resolveRequestContext(requestLike: APIRequestContext | Page) {
  46  |   if ('post' in requestLike && typeof requestLike.post === 'function') {
  47  |     return requestLike as APIRequestContext;
  48  |   }
  49  |   return (requestLike as Page).request;
  50  | }
  51  |
  52  | export async function loginByApi(
  53  |   requestLike: APIRequestContext | Page,
  54  |   credentials: LoginCredentials,
  55  | ): Promise<BrowserLoginResult> {
  56  |   const request = resolveRequestContext(requestLike);
> 57  |   const response = await request.post(`${apiBaseUrl}/auth/login`, {
      |                                  ^ Error: apiRequestContext.post: connect ECONNREFUSED 127.0.0.1:24485
  58  |     data: credentials,
  59  |   });
  60  |   expect(response.ok()).toBeTruthy();
  61  |   const payload = await response.json();
  62  |   expect(payload.code).toBe(200);
  63  |   const session = readAuthCookieSession(response.headers(), {
  64  |     csrfFallback: `pantheon-smoke-csrf-${Date.now()}`,
  65  |   });
  66  |   return {
  67  |     accessToken: session.accessToken,
  68  |     refreshToken: session.refreshToken,
  69  |     username: credentials.username,
  70  |     password: credentials.password,
  71  |     csrfToken: session.csrfToken,
  72  |   };
  73  | }
  74  |
  75  | export async function signInWithUi(
  76  |   page: Page,
  77  |   credentials: LoginCredentials,
  78  | ) {
  79  |   await primeChineseLocale(page);
  80  |   await page.goto('/login', { waitUntil: 'domcontentloaded' });
  81  |   await page.getByPlaceholder(/请输入用户名|username/i).fill(credentials.username);
  82  |   await page.getByPlaceholder(/请输入密码|password/i).fill(credentials.password);
  83  |   const loginResponse = page.waitForResponse((response) =>
  84  |     response.url().includes('/api/v1/auth/login'),
  85  |   );
  86  |   await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();
  87  |   const response = await loginResponse;
  88  |   expect(response.ok()).toBeTruthy();
  89  |   const payload = await response.json();
  90  |   expect(payload.code).toBe(200);
  91  |   await page.goto('/dashboard', { waitUntil: 'domcontentloaded' });
  92  |   await expect(page.locator('.app-shell__header')).toBeVisible();
  93  |   return COOKIE_TOKEN_PLACEHOLDER;
  94  | }
  95  |
  96  | export async function signInAsAdmin(page: Page) {
  97  |   const login = await loginByApi(page.request, adminCredentials);
  98  |   await installClientSession(page, login);
  99  |   await installAuthUserSnapshot(page, await getCurrentUserInfo(page.request, login.accessToken));
  100 |   return login.accessToken;
  101 | }
  102 |
  103 | async function getCurrentUserInfo(
  104 |   requestLike: APIRequestContext | Page,
  105 |   accessToken: string,
  106 | ): Promise<UserInfo> {
  107 |   const request = resolveRequestContext(requestLike);
  108 |   const response = await request.get(`${apiBaseUrl}/auth/me`, {
  109 |     headers: authHeaders(accessToken),
  110 |   });
  111 |   expect(response.ok()).toBeTruthy();
  112 |   const payload = await response.json();
  113 |   expect(payload.code).toBe(200);
  114 |   return payload.data as UserInfo;
  115 | }
  116 |
  117 | async function installAuthUserSnapshot(page: Page, userInfo: UserInfo) {
  118 |   const serializedUser = JSON.stringify({
  119 |     ...userInfo,
  120 |     preferences: undefined,
  121 |   } satisfies UserInfo);
  122 |   await page.addInitScript((value) => {
  123 |     try {
  124 |       globalThis.localStorage?.setItem('pantheon_auth_user', value);
  125 |     } catch {
  126 |       // about:blank and other opaque origins do not expose storage.
  127 |     }
  128 |   }, serializedUser);
  129 |   if (page.url() !== 'about:blank') {
  130 |     await page.evaluate(([storageKey, value]) => {
  131 |       try {
  132 |         globalThis.localStorage?.setItem(storageKey, value);
  133 |       } catch {
  134 |         // Ignore opaque-origin storage failures for smoke setup.
  135 |       }
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
```
