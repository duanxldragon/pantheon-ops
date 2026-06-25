import { expect, type APIRequestContext, type Page } from '@playwright/test';
import { createDecipheriv, createHmac } from 'node:crypto';
import { spawn } from 'node:child_process';

const apiOrigin = process.env.PANTHEON_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';
export const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? `${apiOrigin}/api/v1`;
const defaultWebBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5173';
const totpPeriodSeconds = 30;
const totpDigits = 6;
const cachedTotpSecrets = new Map<string, string>();
const defaultMfaSecret = 'pantheon-mfa-dev-secret-key!';
const encryptedMfaSecretPrefix = 'mfa:v1:';

export type BrowserLoginResult = {
  accessToken: string;
  refreshToken: string;
  username: string;
  password: string;
  csrfToken: string;
};

export type LoginCredentials = {
  username: string;
  password: string;
};

type ApiEnvelope<T> = {
  code: number;
  data: T;
  message?: string;
};

type AuthPayload = {
  accessToken?: string;
  refreshToken?: string;
  mfaRequired?: boolean;
  challengeId?: string;
  setupRequired?: boolean;
  totpSecret?: string;
  totpProvisionUri?: string;
  user?: unknown;
};

function resolveCachedTotpSecret(username: string) {
  const normalized = username.trim().toUpperCase();
  if (!normalized) {
    return null;
  }
  return (
    cachedTotpSecrets.get(normalized) ??
    (normalized === adminCredentials.username.trim().toUpperCase()
      ? process.env.PANTHEON_SMOKE_ADMIN_TOTP_SECRET ?? null
      : null)
  );
}

export const adminCredentials: LoginCredentials = {
  username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
  password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
};

export async function primeChineseLocale(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('pantheon_lang', 'zh-CN');
    localStorage.setItem('pantheon_lang_explicit', '1');
  });
}

export function authHeaders(accessToken: string) {
  return {
    Authorization: `Bearer ${accessToken}`,
  };
}

function resolveRequestContext(requestLike: APIRequestContext | Page) {
  if ('post' in requestLike && typeof requestLike.post === 'function') {
    return requestLike as APIRequestContext;
  }
  return (requestLike as Page).request;
}

function extractCookieValue(setCookieHeader: string | undefined, name: string) {
  if (!setCookieHeader) {
    return null;
  }
  const match = setCookieHeader.match(new RegExp(`(?:^|,\\s*)${name}=([^;]+)`));
  return match?.[1] ?? null;
}

function collectSetCookieHeader(response: Awaited<ReturnType<APIRequestContext['post']>>) {
  const headerValues = response
    .headersArray()
    .filter((header) => header.name.toLowerCase() === 'set-cookie')
    .map((header) => header.value);
  if (headerValues.length > 0) {
    return headerValues.join(', ');
  }
  return response.headers()['set-cookie'];
}

function attachCookieTokens(
  response: Awaited<ReturnType<APIRequestContext['post']>>,
  payload: ApiEnvelope<AuthPayload>,
): ApiEnvelope<AuthPayload> {
  const setCookieHeader = collectSetCookieHeader(response);
  const accessToken = payload.data?.accessToken ?? extractCookieValue(setCookieHeader, 'pantheon_access_token');
  const refreshToken = payload.data?.refreshToken ?? extractCookieValue(setCookieHeader, 'pantheon_refresh_token');
  if (!accessToken || !refreshToken) {
    return payload;
  }
  return {
    ...payload,
    data: {
      ...payload.data,
      accessToken,
      refreshToken,
    },
  };
}

type MysqlConfig = {
  host: string;
  port: number;
  username: string;
  password: string;
  database: string;
  mysqlBin: string;
};

function parseDsn(dsn: string | undefined): MysqlConfig | null {
  const trimmed = String(dsn || '').trim();
  const marker = '@tcp(';
  const markerIndex = trimmed.indexOf(marker);
  if (markerIndex < 0) {
    return null;
  }
  const credentials = trimmed.slice(0, markerIndex);
  const separatorIndex = credentials.indexOf(':');
  if (separatorIndex < 0) {
    return null;
  }
  const username = credentials.slice(0, separatorIndex);
  const password = credentials.slice(separatorIndex + 1);
  const hostPortStart = markerIndex + marker.length;
  const hostPortEnd = trimmed.indexOf(')', hostPortStart);
  if (hostPortEnd < 0) {
    return null;
  }
  const hostPort = trimmed.slice(hostPortStart, hostPortEnd);
  const slashIndex = trimmed.indexOf('/', hostPortEnd);
  if (slashIndex < 0) {
    return null;
  }
  const queryIndex = trimmed.indexOf('?', slashIndex + 1);
  const database = queryIndex >= 0 ? trimmed.slice(slashIndex + 1, queryIndex) : trimmed.slice(slashIndex + 1);
  const [host, portText] = hostPort.split(':');
  return {
    host: process.env.PANTHEON_SMOKE_MYSQL_HOST || host || '127.0.0.1',
    port: Number(process.env.PANTHEON_SMOKE_MYSQL_PORT || portText || 3306),
    username: process.env.PANTHEON_SMOKE_MYSQL_USER || username || 'root',
    password: process.env.PANTHEON_SMOKE_MYSQL_PASSWORD || password || '',
    database: process.env.PANTHEON_SMOKE_MYSQL_DATABASE || database || '',
    mysqlBin: process.env.PANTHEON_SMOKE_MYSQL_BIN || 'mysql',
  };
}

function queryMysql(sql: string, config: MysqlConfig) {
  return new Promise<string>((resolve, reject) => {
    const child = spawn(
      config.mysqlBin,
      [
        `--host=${config.host}`,
        `--port=${String(config.port)}`,
        `--user=${config.username}`,
        '--default-character-set=utf8mb4',
        '--protocol=TCP',
        '--batch',
        '--skip-column-names',
        config.database,
        '-e',
        sql,
      ],
      {
        stdio: ['ignore', 'pipe', 'pipe'],
        shell: false,
        env: {
          ...process.env,
          MYSQL_PWD: config.password,
        },
      },
    );
    let stdout = '';
    let stderr = '';
    child.stdout?.setEncoding('utf8');
    child.stderr?.setEncoding('utf8');
    child.stdout?.on('data', (chunk) => {
      stdout += chunk;
    });
    child.stderr?.on('data', (chunk) => {
      stderr += chunk;
    });
    child.once('error', reject);
    child.once('exit', (code) => {
      if (code === 0) {
        resolve(stdout.trim());
        return;
      }
      reject(new Error(`mysql exited with code ${code ?? 'unknown'}: ${stderr.trim()}`));
    });
  });
}

function decryptMfaSecret(value: string, rawKey = process.env.PANTHEON_MFA_SECRET ?? defaultMfaSecret) {
  const trimmed = String(value || '').trim();
  if (!trimmed) {
    return '';
  }
  if (!trimmed.startsWith(encryptedMfaSecretPrefix)) {
    return trimmed;
  }
  const cipherText = Buffer.from(trimmed.slice(encryptedMfaSecretPrefix.length), 'base64');
  const key = Buffer.alloc(32);
  Buffer.from(String(rawKey || defaultMfaSecret)).copy(key, 0, 0, 32);
  const nonce = cipherText.subarray(0, 12);
  const authTag = cipherText.subarray(cipherText.length - 16);
  const encrypted = cipherText.subarray(12, cipherText.length - 16);
  const decipher = createDecipheriv('aes-256-gcm', key, nonce);
  decipher.setAuthTag(authTag);
  return Buffer.concat([decipher.update(encrypted), decipher.final()]).toString('utf8');
}

async function resolveTotpSecretForUser(username: string) {
  const normalizedUsername = username.trim();
  if (!normalizedUsername) {
    return null;
  }
  const cachedSecret = resolveCachedTotpSecret(normalizedUsername);
  if (cachedSecret) {
    return cachedSecret;
  }

  const mysqlConfig = parseDsn(process.env.PANTHEON_DSN);
  if (!mysqlConfig?.database) {
    return null;
  }

  const escapedUsername = normalizedUsername.replace(/'/g, "''");
  const encryptedSecret = await queryMysql(
    `
SELECT f.secret_encrypted
FROM system_auth_factor f
JOIN system_user u ON u.id = f.user_id
WHERE u.username = '${escapedUsername}' AND f.factor_type = 'totp' AND f.enabled = 1
ORDER BY f.id DESC
LIMIT 1;
`.trim(),
    mysqlConfig,
  ).catch(() => '');

  if (!encryptedSecret) {
    return null;
  }

  const secret = decryptMfaSecret(encryptedSecret);
  if (!secret) {
    return null;
  }
  cachedTotpSecrets.set(normalizedUsername.toUpperCase(), secret);
  return secret;
}

function decodeBase32NoPadding(secret: string) {
  const normalized = secret.trim().toUpperCase();
  if (!normalized) {
    throw new Error('auth.mfa.secret.required');
  }
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = 0;
  let value = 0;
  const bytes: number[] = [];
  for (const char of normalized) {
    if (char === '=') {
      break;
    }
    const index = alphabet.indexOf(char);
    if (index < 0) {
      throw new Error('auth.mfa.secret.invalid');
    }
    value = (value << 5) | index;
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      bytes.push((value >>> bits) & 0xff);
    }
  }
  return Buffer.from(bytes);
}

function generateTotpCode(secret: string, unixSeconds: number) {
  const key = decodeBase32NoPadding(secret);
  const counter = Math.floor(unixSeconds / totpPeriodSeconds);
  const payload = Buffer.alloc(8);
  payload.writeBigUInt64BE(BigInt(counter));
  const digest = createHmac('sha1', key).update(payload).digest();
  const offset = digest[digest.length - 1] & 0x0f;
  const binaryCode =
    ((digest[offset] & 0x7f) << 24) |
    ((digest[offset + 1] & 0xff) << 16) |
    ((digest[offset + 2] & 0xff) << 8) |
    (digest[offset + 3] & 0xff);
  return String(binaryCode % 10 ** totpDigits).padStart(totpDigits, '0');
}

async function resolveLoginResponse(
  request: APIRequestContext,
  response: Awaited<ReturnType<APIRequestContext['post']>>,
  credentials: LoginCredentials,
) {
  expect(response.ok()).toBeTruthy();
  const payload = attachCookieTokens(response, (await response.json()) as ApiEnvelope<AuthPayload>);
  expect(payload.code).toBe(200);
  if (payload.data?.accessToken && payload.data?.refreshToken) {
    return { response, payload };
  }
  if (!payload.data?.mfaRequired || !payload.data.challengeId) {
    throw new Error('auth.login.response_invalid');
  }
  const totpSecret = payload.data.totpSecret ?? (await resolveTotpSecretForUser(credentials.username));
  if (!totpSecret) {
    throw new Error('auth.mfa.secret.unavailable');
  }
  cachedTotpSecrets.set(credentials.username.trim().toUpperCase(), totpSecret);
  const verifyResponse = await request.post(`${apiBaseUrl}/auth/mfa/verify`, {
    data: {
      challengeId: payload.data.challengeId,
      code: generateTotpCode(totpSecret, Date.now() / 1000),
    },
  });
  expect(verifyResponse.ok()).toBeTruthy();
  const verifyPayload = attachCookieTokens(verifyResponse, (await verifyResponse.json()) as ApiEnvelope<AuthPayload>);
  expect(verifyPayload.code).toBe(200);
  expect(verifyPayload.data?.accessToken).toBeTruthy();
  expect(verifyPayload.data?.refreshToken).toBeTruthy();
  return { response: verifyResponse, payload: verifyPayload, password: credentials.password };
}

export async function loginByApi(
  requestLike: APIRequestContext | Page,
  credentials: LoginCredentials,
): Promise<BrowserLoginResult> {
  const request = resolveRequestContext(requestLike);
  const initialResponse = await request.post(`${apiBaseUrl}/auth/login`, {
    data: credentials,
  });
  const { response, payload } = await resolveLoginResponse(request, initialResponse, credentials);
  const csrfToken =
    extractCookieValue(collectSetCookieHeader(response), 'pantheon_csrf_token') ??
    `pantheon-smoke-csrf-${Date.now()}`;
  return {
    accessToken: payload.data.accessToken as string,
    refreshToken: payload.data.refreshToken as string,
    username: credentials.username,
    password: credentials.password,
    csrfToken,
  };
}

export async function signInWithUi(
  page: Page,
  credentials: LoginCredentials,
) {
  await primeChineseLocale(page);
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.getByPlaceholder(/请输入用户名|username/i).fill(credentials.username);
  await page.getByPlaceholder(/请输入密码|password/i).fill(credentials.password);
  const loginResponse = page.waitForResponse((response) =>
    response.url().includes('/api/v1/auth/login'),
  );
  await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();
  const response = await loginResponse;
  expect(response.ok()).toBeTruthy();
  const payload = await response.json() as ApiEnvelope<AuthPayload>;
  expect(payload.code).toBe(200);
  if (payload.data?.mfaRequired) {
    const totpSecret = payload.data.totpSecret ?? (await resolveTotpSecretForUser(credentials.username));
    expect(totpSecret).toBeTruthy();
    cachedTotpSecrets.set(credentials.username.trim().toUpperCase(), totpSecret as string);
    const mfaResponse = page.waitForResponse((nextResponse) =>
      nextResponse.url().includes('/api/v1/auth/mfa/verify'),
    );
    await page
      .getByPlaceholder(/6 位动态码|6-digit code|二次验证码/i)
      .fill(generateTotpCode(totpSecret as string, Date.now() / 1000));
    await page.getByRole('button', { name: /验证并登录|Verify and sign in/i }).click();
    const verifyResponse = await mfaResponse;
    expect(verifyResponse.ok()).toBeTruthy();
    const verifyPayload = await verifyResponse.json() as ApiEnvelope<AuthPayload>;
    expect(verifyPayload.code).toBe(200);
    expect(verifyPayload.data?.accessToken).toBeTruthy();
    await expect(page.locator('.app-shell__header')).toBeVisible();
    return verifyPayload.data.accessToken as string;
  }
  await expect(page.locator('.app-shell__header')).toBeVisible();
  return payload.data.accessToken as string;
}

export async function signInAsAdmin(page: Page) {
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  return login.accessToken;
}

export async function installClientSession(page: Page, login: BrowserLoginResult) {
  await primeChineseLocale(page);
  const appBaseUrl = new URL(
    page.url() === 'about:blank' ? '/' : page.url(),
    page.url() === 'about:blank' ? defaultWebBaseUrl : undefined,
  );
  const cookieUrl = appBaseUrl.origin;
  await page.context().addCookies([
    {
      name: 'pantheon_access_token',
      value: login.accessToken,
      url: cookieUrl,
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_refresh_token',
      value: login.refreshToken,
      url: cookieUrl,
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_csrf_token',
      value: login.csrfToken,
      url: cookieUrl,
      httpOnly: false,
      secure: false,
      sameSite: 'Strict',
    },
  ]);
}

export async function getCsrfToken(page: Page) {
  const cookies = await page.context().cookies();
  const csrfCookie = cookies.find((cookie) => cookie.name === 'pantheon_csrf_token');
  expect(csrfCookie?.value).toBeTruthy();
  return csrfCookie!.value;
}

export async function requestHeaders(page: Page, accessToken: string) {
  return {
    ...authHeaders(accessToken),
    'X-CSRF-Token': await getCsrfToken(page),
  };
}

export function apiRequestHeaders(login: BrowserLoginResult) {
  return {
    ...authHeaders(login.accessToken),
    'X-CSRF-Token': login.csrfToken,
    Cookie: `pantheon_csrf_token=${login.csrfToken}`,
  };
}

export async function getOperationToken(page: Page, accessToken: string) {
  const response = await page.request.post(`${apiBaseUrl}/auth/operation-verify`, {
    headers: await requestHeaders(page, accessToken),
    data: { password: adminCredentials.password },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data.operationToken as string;
}

export async function getApiOperationToken(requestLike: APIRequestContext | Page, login: BrowserLoginResult) {
  const request = resolveRequestContext(requestLike);
  const response = await request.post(`${apiBaseUrl}/auth/operation-verify`, {
    headers: apiRequestHeaders(login),
    data: { password: login.password },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data.operationToken as string;
}

export async function verifiedHeaders(page: Page, accessToken: string) {
  return {
    ...(await requestHeaders(page, accessToken)),
    'X-Operation-Token': await getOperationToken(page, accessToken),
  };
}

export async function verifiedApiHeaders(requestLike: APIRequestContext | Page, login: BrowserLoginResult) {
  return {
    ...apiRequestHeaders(login),
    'X-Operation-Token': await getApiOperationToken(requestLike, login),
  };
}

export async function installOperationToken(page: Page, accessToken: string) {
  const token = await getOperationToken(page, accessToken);
  await page.addInitScript((value) => {
    sessionStorage.setItem('pantheon_op_token', value);
  }, token);
  try {
    await page.evaluate((value) => {
      sessionStorage.setItem('pantheon_op_token', value);
    }, token);
  } catch {
    // The current document can still be about:blank in setup-heavy smoke tests.
    // The init script above installs the token on the next app navigation.
  }
}
