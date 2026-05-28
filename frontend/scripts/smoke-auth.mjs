import { spawn } from 'node:child_process';
import { createDecipheriv, createHmac } from 'node:crypto';
import process from 'node:process';

export const defaultMfaSecret = 'pantheon-mfa-dev-secret-key!';
const encryptedMfaSecretPrefix = 'mfa:v1:';
const totpPeriodSeconds = 30;
const totpDigits = 6;

export function extractCookieValue(setCookieHeader, name) {
  if (!setCookieHeader) {
    return null;
  }
  const match = String(setCookieHeader).match(new RegExp(String.raw`(?:^|,\s*)${name}=([^;]+)`));
  return match?.[1] ?? null;
}

export function parseDsn(dsn) {
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
  const database =
    queryIndex >= 0
      ? trimmed.slice(slashIndex + 1, queryIndex)
      : trimmed.slice(slashIndex + 1);
  const [host, portText] = hostPort.split(':');
  return {
    host: host || '127.0.0.1',
    port: Number(portText || '3306'),
    username,
    password,
    database,
  };
}

export function resolveMysqlConfig() {
  const parsedDsn = parseDsn(process.env.PANTHEON_DSN);
  if (!parsedDsn) {
    return null;
  }
  return {
    host: process.env.PANTHEON_SMOKE_MYSQL_HOST || parsedDsn.host || '127.0.0.1',
    port: Number(process.env.PANTHEON_SMOKE_MYSQL_PORT || parsedDsn.port || 3306),
    username: process.env.PANTHEON_SMOKE_MYSQL_USER || parsedDsn.username || 'root',
    password: process.env.PANTHEON_SMOKE_MYSQL_PASSWORD || parsedDsn.password || '',
    database: process.env.PANTHEON_SMOKE_MYSQL_DATABASE || parsedDsn.database || '',
    mysqlBin: process.env.PANTHEON_SMOKE_MYSQL_BIN || 'mysql',
  };
}

function queryMysql(sql, config) {
  return new Promise((resolve, reject) => {
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

export async function executeMysql(sql, config = resolveMysqlConfig()) {
  if (!config?.database) {
    throw new Error('mysql.config.unavailable');
  }
  return queryMysql(sql, config);
}

export function decryptMfaSecret(value, rawKey = process.env.PANTHEON_MFA_SECRET ?? defaultMfaSecret) {
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

function decodeBase32NoPadding(secret) {
  const normalized = String(secret || '').trim().toUpperCase();
  if (!normalized) {
    throw new Error('auth.mfa.secret.required');
  }
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = 0;
  let value = 0;
  const bytes = [];
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

export function generateTotpCode(secret, unixSeconds = Date.now() / 1000) {
  const key = decodeBase32NoPadding(secret);
  const counter = Math.floor(Number(unixSeconds) / totpPeriodSeconds);
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

export async function resolveTotpSecretForUser(username) {
  const normalizedUsername = String(username || '').trim();
  if (!normalizedUsername) {
    return null;
  }
  if (
    normalizedUsername === (process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin') &&
    process.env.PANTHEON_SMOKE_ADMIN_TOTP_SECRET
  ) {
    return process.env.PANTHEON_SMOKE_ADMIN_TOTP_SECRET;
  }
  const mysqlConfig = resolveMysqlConfig();
  if (!mysqlConfig?.database) {
    return null;
  }
  const escapedUsername = normalizedUsername.replace(/'/g, "''");
  const sql = `
SELECT f.secret_encrypted
FROM system_auth_factor f
JOIN system_user u ON u.id = f.user_id
WHERE u.username = '${escapedUsername}' AND f.factor_type = 'totp' AND f.enabled = 1
ORDER BY f.id DESC
LIMIT 1;
`;
  const encrypted = await queryMysql(sql, mysqlConfig).catch(() => '');
  if (!encrypted) {
    return null;
  }
  return decryptMfaSecret(encrypted);
}

export function buildAuthHeaders(session) {
  return {
    Authorization: `Bearer ${session.accessToken}`,
    'X-CSRF-Token': session.csrfToken,
    Cookie: `pantheon_csrf_token=${session.csrfToken}`,
  };
}

export function buildVerifiedHeaders(session, operationToken) {
  return {
    ...buildAuthHeaders(session),
    'X-Operation-Token': operationToken,
  };
}

export async function loginWithOptionalMfa(apiBaseUrl, { username, password }) {
  const loginResponse = await fetch(`${apiBaseUrl}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });
  if (!loginResponse.ok) {
    throw new Error(`Login failed: HTTP ${loginResponse.status}`);
  }
  const loginPayload = await loginResponse.json();
  if (loginPayload.code !== 200) {
    throw new Error(`Login failed: code ${loginPayload.code}`);
  }
  const loginData = loginPayload.data || {};
  if (loginData.accessToken && loginData.refreshToken) {
    return {
      accessToken: loginData.accessToken,
      refreshToken: loginData.refreshToken,
      csrfToken:
        extractCookieValue(loginResponse.headers.get('set-cookie'), 'pantheon_csrf_token') ??
        `pantheon-smoke-csrf-${Date.now()}`,
    };
  }
  if (!loginData.mfaRequired || !loginData.challengeId) {
    throw new Error('auth.login.response_invalid');
  }
  const totpSecret = loginData.totpSecret ?? (await resolveTotpSecretForUser(username));
  if (!totpSecret) {
    throw new Error('auth.mfa.secret.unavailable');
  }
  const verifyResponse = await fetch(`${apiBaseUrl}/auth/mfa/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      challengeId: loginData.challengeId,
      code: generateTotpCode(totpSecret),
    }),
  });
  if (!verifyResponse.ok) {
    throw new Error(`MFA verify failed: HTTP ${verifyResponse.status}`);
  }
  const verifyPayload = await verifyResponse.json();
  if (verifyPayload.code !== 200) {
    throw new Error(`MFA verify failed: code ${verifyPayload.code}`);
  }
  return {
    accessToken: verifyPayload.data.accessToken,
    refreshToken: verifyPayload.data.refreshToken,
    csrfToken:
      extractCookieValue(verifyResponse.headers.get('set-cookie'), 'pantheon_csrf_token') ??
      extractCookieValue(loginResponse.headers.get('set-cookie'), 'pantheon_csrf_token') ??
      `pantheon-smoke-csrf-${Date.now()}`,
  };
}

export async function getOperationToken(apiBaseUrl, session, password) {
  const response = await fetch(`${apiBaseUrl}/auth/operation-verify`, {
    method: 'POST',
    headers: {
      ...buildAuthHeaders(session),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ password }),
  });
  if (!response.ok) {
    throw new Error(`Operation verify failed: HTTP ${response.status}`);
  }
  const payload = await response.json();
  if (payload.code !== 200) {
    throw new Error(`Operation verify failed: code ${payload.code}`);
  }
  return payload.data.operationToken;
}
